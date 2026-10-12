// Package deliverability builds the Deliverability screen (spec 7.8): health
// per sending domain and dedicated IP (shared IP pools labelled shared and
// checked at domain level), account health per sending provider against
// that provider's own thresholds, blocklists (84 sources via HostRepute when
// connected, a built-in set of 8 public DNSBLs checked directly otherwise),
// Google Postmaster Tools, Microsoft SNDS, the DMARC report summary and the
// incident timeline. It also runs the local checks (internal/dnscheck),
// exposes the reputation verdict Send Readiness uses and records incidents
// for other domains.
package deliverability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/hostrepute"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service holds the deliverability dependencies.
type Service struct {
	Pool       *pgxpool.Pool
	Q          *store.Queries
	Audit      *audit.Logger
	Log        *slog.Logger
	HostRepute *hostrepute.Service
	Jobs       hostrepute.JobInserter
	// DNS answers the local checks; Lists is the built-in DNSBL set.
	DNS   dnscheck.Resolver
	Lists []dnscheck.Blocklist
	Now   func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) lists() []dnscheck.Blocklist {
	if s.Lists != nil {
		return s.Lists
	}
	return dnscheck.Builtin
}

// state is everything the screen is built from, read once per request.
type state struct {
	ws  uuid.UUID
	now time.Time
	// today is the workspace's current date (at 00:00 UTC, like DATE columns).
	today       time.Time
	zone        *time.Location
	conn        *store.HostreputeConnection
	cands       []hostrepute.Candidate
	providers   []store.ListActiveProvidersRow
	streams     []store.ListProviderStreamsRow
	warmups     []store.ListActiveWarmupPlansRow
	blocklists  []store.BlocklistResult
	checks      map[string]store.DeliverabilityAssetCheck
	postmaster  []store.PostmasterDaily
	snds        []store.SndsDaily
	dmarc       []store.DmarcReport
	domains     []store.ListAssetSourcesRow
	monitored   map[string]monitored
	running     *store.DeliverabilityCheckRun
	lastBuiltin *store.DeliverabilityCheckRun
}

type monitored struct {
	Asset   string `json:"asset"`
	Score   *int   `json:"score"`
	Verdict string `json:"verdict"`
}

// connected reports whether HostRepute data is current: connected and the
// last sync worked.
func (st *state) connected() bool {
	return st.conn != nil && (st.conn.Status == "connected" || st.conn.Status == "limit_reached") && st.conn.LastSyncError == ""
}

// hostReputeStale reports whether there is HostRepute data that can't be
// refreshed (token rejected or HostRepute unreachable).
func (st *state) hostReputeStale() bool {
	return st.conn != nil && !st.connected()
}

func (s *Service) load(ctx context.Context, ws uuid.UUID, tz string) (*state, error) {
	now := s.now()
	st := &state{ws: ws, now: now, today: dateIn(now, tz), zone: zoneOf(tz), checks: map[string]store.DeliverabilityAssetCheck{}, monitored: map[string]monitored{}}
	q := s.Q
	if conn, err := q.GetHostReputeConnection(ctx, ws); err == nil {
		st.conn = &conn
		var ms []monitored
		_ = json.Unmarshal(conn.Monitored, &ms)
		for _, m := range ms {
			st.monitored[strings.ToLower(m.Asset)] = m
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	var err error
	if st.cands, err = hostrepute.Candidates(ctx, q, ws); err != nil {
		return nil, err
	}
	if st.domains, err = q.ListAssetSources(ctx, ws); err != nil {
		return nil, err
	}
	if st.providers, err = q.ListActiveProviders(ctx, ws); err != nil {
		return nil, err
	}
	if st.streams, err = q.ListProviderStreams(ctx, ws); err != nil {
		return nil, err
	}
	if st.warmups, err = q.ListActiveWarmupPlans(ctx, ws); err != nil {
		return nil, err
	}
	if st.blocklists, err = q.ListBlocklistResults(ctx, ws); err != nil {
		return nil, err
	}
	checks, err := q.ListAssetChecks(ctx, ws)
	if err != nil {
		return nil, err
	}
	for _, c := range checks {
		st.checks[strings.ToLower(c.Asset)] = c
	}
	since := st.today.AddDate(0, 0, -hostrepute.PostmasterDays)
	if st.postmaster, err = q.ListPostmasterDays(ctx, store.ListPostmasterDaysParams{WorkspaceID: ws, Since: since}); err != nil {
		return nil, err
	}
	if st.snds, err = q.ListSNDSDays(ctx, store.ListSNDSDaysParams{WorkspaceID: ws, Since: since}); err != nil {
		return nil, err
	}
	if st.dmarc, err = q.ListDMARCReports(ctx, store.ListDMARCReportsParams{WorkspaceID: ws, Since: st.now.AddDate(0, 0, -hostrepute.DMARCDays)}); err != nil {
		return nil, err
	}
	if run, err := q.RunningCheckRun(ctx, store.RunningCheckRunParams{WorkspaceID: ws, AliveAfter: st.now.Add(-runTTL)}); err == nil {
		st.running = &run
	}
	if run, err := q.LatestFinishedCheckRun(ctx, store.LatestFinishedCheckRunParams{WorkspaceID: ws, Via: "builtin"}); err == nil {
		st.lastBuiltin = &run
	}
	return st, nil
}

// runTTL is how long a check run may stay "running" before it counts as dead.
const runTTL = 10 * time.Minute

// Overview builds the Deliverability screen.
func (s *Service) Overview(ctx context.Context, ws *store.Workspace) (DeliverabilityOverview, error) {
	st, err := s.load(ctx, ws.ID, ws.Timezone)
	if err != nil {
		return DeliverabilityOverview{}, err
	}
	out := DeliverabilityOverview{
		HostRepute: st.hostReputeHeader(),
		Checks:     st.checkSchedule(len(s.lists())),
		Assets:     st.assetHealth(len(s.lists())),
		Pools:      st.pools(),
		Providers:  st.providerHealth(),
		Blocklists: st.blocklistStatus(len(s.lists())),
		Postmaster: st.postmasterPanel(),
		SNDS:       st.sndsPanel(),
		DMARC:      st.dmarcSummary(),
	}
	if out.Incidents, err = s.timeline(ctx, st); err != nil {
		return out, err
	}
	if out.Postmaster != nil && out.Postmaster.Peak != nil {
		out.Postmaster.Peak.Note = peakNote(out.Incidents.Items, out.Postmaster.Peak.Day)
	}
	return out, nil
}

func (st *state) hostReputeHeader() DeliverabilityHostRepute {
	if st.conn == nil {
		return DeliverabilityHostRepute{Status: "not_connected"}
	}
	c := st.conn
	out := DeliverabilityHostRepute{Status: c.Status, TeamName: c.TeamName, Sources: c.Sources, IntervalMinutes: c.CheckIntervalMinutes,
		LastSyncAt: c.LastSyncAt, TokenRejectedAt: c.TokenRejectedAt, DMARCRua: c.DmarcRua}
	if out.Status == "limit_reached" {
		out.Status = "connected"
	}
	if c.LastSyncError != "" && out.Status == "connected" {
		out.Unreachable = &HostReputeOutage{Message: c.LastSyncError, Since: c.LastSyncAttemptAt}
	}
	return out
}

func (st *state) checkSchedule(builtin int) DeliverabilityChecks {
	out := DeliverabilityChecks{Via: "builtin", Interval: "daily", BuiltinSources: builtin}
	if st.conn != nil && (st.conn.Status == "connected" || st.conn.Status == "limit_reached") {
		out.Via, out.Interval = "hostrepute", "hourly"
		if st.conn.LastCheckAt != nil {
			last := *st.conn.LastCheckAt
			out.LastAt = &last
			next := last.Add(time.Duration(max(st.conn.CheckIntervalMinutes, 1)) * time.Minute)
			out.NextAt = &next
		}
	} else if st.lastBuiltin != nil && st.lastBuiltin.FinishedAt != nil {
		last := *st.lastBuiltin.FinishedAt
		next := last.Add(24 * time.Hour)
		out.LastAt, out.NextAt = &last, &next
	}
	if st.running != nil {
		r := runView(*st.running)
		out.Running = &r
	}
	return out
}

func runView(r store.DeliverabilityCheckRun) CheckRun {
	assets := r.Assets
	if assets == nil {
		assets = []string{}
	}
	return CheckRun{ID: r.ID.String(), Via: r.Via, Trigger: r.Trigger, Status: r.Status, Assets: assets, Total: r.Total, Answered: r.Answered,
		Listed: r.Listed, Error: r.Error, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt}
}

// blocklistVia picks the source shown as current: HostRepute when it has
// results (even stale ones), else the built-in check.
func (st *state) blocklistVia() string {
	if st.conn != nil && slices.ContainsFunc(st.blocklists, func(b store.BlocklistResult) bool { return b.Via == "hostrepute" }) {
		return "hostrepute"
	}
	if slices.ContainsFunc(st.blocklists, func(b store.BlocklistResult) bool { return b.Via == "builtin" }) {
		return "builtin"
	}
	if st.conn != nil {
		return "hostrepute"
	}
	return "none"
}

func impactRank(s string) int {
	switch s {
	case "high":
		return 0
	case "medium":
		return 1
	}
	return 2
}

// assetBlocklists summarizes one asset's results from one source.
func (st *state) assetBlocklists(asset, via string, sources int) AssetBlocklists {
	out := AssetBlocklists{Via: via, Listings: []string{}}
	if via == "none" {
		return out
	}
	n := 0
	for _, b := range st.blocklists {
		if b.Via != via || !strings.EqualFold(b.Asset, asset) {
			continue
		}
		n++
		if out.CheckedAt == nil || b.CheckedAt.After(*out.CheckedAt) {
			t := b.CheckedAt
			out.CheckedAt = &t
		}
		if b.Listed {
			out.Listed++
			out.Listings = append(out.Listings, b.Source)
			if b.Impact == "high" {
				out.HighImpact = true
			}
		}
	}
	// HostRepute and the built-in set check every source on every asset of
	// the matching kind; show the set size, not the per-kind count.
	out.Sources = sources
	if n == 0 {
		out.CheckedAt = nil
	}
	return out
}

func (st *state) sourcesFor(via string, builtin int) int {
	if via == "hostrepute" && st.conn != nil && st.conn.Sources > 0 {
		return int(st.conn.Sources)
	}
	if via == "hostrepute" {
		set := map[string]bool{}
		for _, b := range st.blocklists {
			if b.Via == "hostrepute" {
				set[b.Source] = true
			}
		}
		return len(set)
	}
	return builtin
}

func (st *state) providerShort(id uuid.UUID) string {
	for _, p := range st.providers {
		if p.ID == id {
			return campaigns.ShortName(p.Name, p.Adapter)
		}
	}
	return ""
}

// assetHealth builds the cards for sending domains and dedicated IPs.
func (st *state) assetHealth(builtin int) []AssetHealth {
	out := []AssetHealth{}
	via := st.blocklistVia()
	sources := st.sourcesFor(via, builtin)
	stale := st.hostReputeStale()
	var staleAt *time.Time
	if stale && st.conn != nil {
		staleAt = st.conn.LastCheckAt
		if staleAt == nil {
			staleAt = st.conn.LastSyncAt
		}
	}
	for _, c := range st.cands {
		if c.Role != hostrepute.RoleSendingDomain && c.Role != hostrepute.RoleDedicatedIP {
			continue
		}
		a := AssetHealth{Asset: c.Asset, Type: c.Type, Role: c.Role, UsedBy: []string{}, Tone: "ok"}
		a.Blocklists = st.assetBlocklists(c.Asset, via, sources)
		if via == "hostrepute" && stale {
			a.Stale = staleAt
			fb := st.assetBlocklists(c.Asset, "builtin", builtin)
			if fb.CheckedAt != nil {
				a.Fallback = &fb
			}
		}
		if m, ok := st.monitored[strings.ToLower(c.Asset)]; ok && via == "hostrepute" {
			a.Score, a.Verdict = m.Score, m.Verdict
		}
		switch c.Type {
		case hostrepute.AssetDomain:
			a.UsedBy = st.usedBy(c.Asset)
			if chk, ok := st.checks[strings.ToLower(c.Asset)]; ok {
				a.Auth = authSummary(chk)
			}
			a.DMARCAligned = st.dmarcAligned(c.Asset)
			a.GmailReputation = st.latestPostmaster(c.Asset).DomainReputation
		case hostrepute.AssetIP:
			if c.ProviderID != nil {
				a.ProviderID = c.ProviderID.String()
				a.Provider = c.Provider
				for _, p := range st.providers {
					if p.ID == *c.ProviderID {
						a.ProviderStatus = p.Status
					}
				}
				a.Warmup = st.warmup(*c.ProviderID, c.Asset)
			}
			if chk, ok := st.checks[strings.ToLower(c.Asset)]; ok {
				var r struct {
					RDNS dnscheck.ReverseDNS `json:"rdns"`
				}
				if json.Unmarshal(chk.Result, &r) == nil && r.RDNS.IP != "" {
					a.ReverseDNS = &ReverseDNS{PTR: r.RDNS.PTR, Matches: r.RDNS.Matches, Status: r.RDNS.Status}
				}
			}
			if sn := st.latestSNDS(c.Asset); sn != nil {
				a.SNDS = &SNDSBrief{FilterResult: sn.FilterResult, TrapHits: st.sndsTraps(c.Asset, 7)}
			}
		}
		a.Tone = assetTone(a)
		out = append(out, a)
	}
	return out
}

func assetTone(a AssetHealth) string {
	switch {
	case a.Stale != nil:
		return "bad"
	case a.Blocklists.Listed > 0 && a.Blocklists.HighImpact:
		return "bad"
	case a.Blocklists.Listed > 0:
		return "warn"
	case a.Auth != nil && (a.Auth.SPF != dnscheck.StatusPass || (a.Auth.DKIM != dnscheck.StatusPass && a.Auth.DKIM != "unknown") || a.Auth.DMARC == dnscheck.StatusMissing || a.Auth.DMARC == dnscheck.StatusInvalid):
		return "warn"
	case a.Score != nil && *a.Score < 70:
		return "bad"
	case a.Score != nil && *a.Score < 90:
		return "warn"
	}
	return "ok"
}

func authSummary(c store.DeliverabilityAssetCheck) *DomainAuthSummary {
	var r struct {
		Auth *dnscheck.DomainAuth `json:"auth"`
	}
	if json.Unmarshal(c.Result, &r) != nil || r.Auth == nil {
		return nil
	}
	t := c.CheckedAt
	s := r.Auth.Summary
	return &DomainAuthSummary{SPF: s.SPF, DKIM: s.DKIM, DMARC: s.DMARC, BIMI: s.BIMI, ReturnPath: s.ReturnPath, CheckedAt: &t}
}

// usedBy lists the providers that send for a domain: those with records on
// it, or every connected provider when the domain has no per-provider records.
func (st *state) usedBy(domain string) []string {
	var ids []uuid.UUID
	for _, d := range st.domains {
		if !strings.EqualFold(d.Domain, domain) {
			continue
		}
		var recs []struct {
			ProviderID string `json:"provider_id"`
		}
		_ = json.Unmarshal(d.DnsRecords, &recs)
		for _, r := range recs {
			if id, err := uuid.Parse(r.ProviderID); err == nil && !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}
	out := []string{}
	for _, p := range st.providers {
		if len(ids) == 0 || slices.Contains(ids, p.ID) {
			out = append(out, campaigns.ShortName(p.Name, p.Adapter))
		}
	}
	return out
}

func (st *state) warmup(provider uuid.UUID, ip string) *WarmupProgress {
	for _, w := range st.warmups {
		if w.ProviderID != provider || (w.Ip != "" && w.Ip != ip) {
			continue
		}
		start := time.Date(w.StartDate.Year(), w.StartDate.Month(), w.StartDate.Day(), 0, 0, 0, 0, time.UTC)
		day := int(st.today.Sub(start).Hours()/24) + 1
		return &WarmupProgress{Day: min(max(day, 1), int(w.Days)), Days: int(w.Days)}
	}
	// Providers without a stored plan describe it in their status ("Warming up · day 9 of 21").
	for _, p := range st.providers {
		if p.ID != provider || p.Status != "warming_up" {
			continue
		}
		var d, n int
		if i := strings.Index(p.StatusDetail, "day "); i >= 0 {
			if _, err := fmt.Sscanf(p.StatusDetail[i:], "day %d of %d", &d, &n); err == nil && d > 0 && n > 0 {
				return &WarmupProgress{Day: d, Days: n}
			}
		}
	}
	return nil
}

type postmasterLatest struct {
	DomainReputation string
	IPReputation     string
}

func (st *state) latestPostmaster(domain string) postmasterLatest {
	var out postmasterLatest
	for _, d := range st.postmaster {
		if strings.EqualFold(d.Domain, domain) {
			if d.DomainReputation != "" {
				out.DomainReputation = d.DomainReputation
			}
			if d.IpReputation != "" {
				out.IPReputation = d.IpReputation
			}
		}
	}
	return out
}

func (st *state) latestSNDS(ip string) *store.SndsDaily {
	var out *store.SndsDaily
	for i := range st.snds {
		if st.snds[i].Ip == ip {
			out = &st.snds[i]
		}
	}
	return out
}

func (st *state) sndsTraps(ip string, days int) int {
	since := st.today.AddDate(0, 0, -days)
	n := 0
	for _, d := range st.snds {
		if d.Ip == ip && !d.Day.Before(since) {
			n += int(d.TrapHits)
		}
	}
	return n
}

// mainDomain is the first sending domain (the one Postmaster and DMARC
// panels describe).
func (st *state) mainDomain() string {
	for _, c := range st.cands {
		if c.Role == hostrepute.RoleSendingDomain {
			return c.Asset
		}
	}
	return ""
}

func round(v float64, digits int) float64 {
	p := math.Pow(10, float64(digits))
	return math.Round(v*p) / p
}

func pctOf(f *float64) *float64 {
	if f == nil {
		return nil
	}
	v := round(*f*100, 2)
	return &v
}

func (st *state) postmasterPanel() *PostmasterPanel {
	domain := st.mainDomain()
	var days []store.PostmasterDaily
	for _, d := range st.postmaster {
		if strings.EqualFold(d.Domain, domain) {
			days = append(days, d)
		}
	}
	if len(days) == 0 {
		return nil
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Day.Before(days[j].Day) })
	last := days[len(days)-1]
	out := &PostmasterPanel{Domain: domain, Via: last.Via, Series: []PostmasterPoint{}}
	out.Stale = last.Via == "hostrepute" && st.hostReputeStale()
	latest := st.latestPostmaster(domain)
	out.DomainReputation, out.IPReputation = latest.DomainReputation, latest.IPReputation
	for _, p := range st.pools() {
		if p.Adapter == "ses" || out.IPReputationPool == "" {
			out.IPReputationPool = p.Short
		}
	}
	out.SPF, out.DKIM, out.DMARC, out.TLS = pctOf(last.SpfRate), pctOf(last.DkimRate), pctOf(last.DmarcRate), pctOf(last.TlsRate)
	var sum float64
	var n int
	fbl, fblSeen := 0, false
	var peak *store.PostmasterDaily
	for i, d := range days {
		if d.SpamRate != nil {
			sum += *d.SpamRate
			n++
			if peak == nil || *d.SpamRate > *peak.SpamRate {
				peak = &days[i]
			}
		}
		if d.FblFlagged != nil {
			fbl += int(*d.FblFlagged)
			fblSeen = true
		}
	}
	if n > 0 {
		avg := round(sum/float64(n)*100, 2)
		out.SpamRate = &avg
	}
	if fblSeen {
		out.FeedbackLoopFlagged = &fbl
	}
	if peak != nil {
		out.Peak = &PostmasterPeak{Day: peak.Day, SpamRate: round(*peak.SpamRate*100, 2)}
	}
	// Two-day buckets from 30 days ago to yesterday.
	out.From = st.today.AddDate(0, 0, -hostrepute.PostmasterDays)
	out.To = st.today
	start := out.From
	for b := 0; b < hostrepute.PostmasterDays/2; b++ {
		from := start.AddDate(0, 0, b*2)
		to := from.AddDate(0, 0, 2)
		var s float64
		var k int
		for _, d := range days {
			if !d.Day.Before(from) && d.Day.Before(to) && d.SpamRate != nil {
				s += *d.SpamRate
				k++
			}
		}
		pt := PostmasterPoint{From: from, To: to.AddDate(0, 0, -1)}
		if k > 0 {
			v := round(s/float64(k)*100, 3)
			pt.SpamRate = &v
		}
		out.Series = append(out.Series, pt)
	}
	return out
}

// peakNote explains the spam-rate peak with an incident of that day.
func peakNote(items []IncidentItem, day time.Time) string {
	for _, it := range items {
		y1, m1, d1 := it.OccurredAt.Date()
		y2, m2, d2 := day.Date()
		if y1 == y2 && m1 == m2 && d1 == d2 && (it.Kind == "reputation_change" || it.Kind == "migration") {
			return "first send after migration"
		}
	}
	return ""
}

func (st *state) sndsPanel() *SNDSPanel {
	for _, c := range st.cands {
		if c.Role != hostrepute.RoleDedicatedIP {
			continue
		}
		last := st.latestSNDS(c.Asset)
		if last == nil {
			continue
		}
		out := &SNDSPanel{IP: c.Asset, Via: last.Via, FilterResult: last.FilterResult, ComplaintRate: pctOf(last.ComplaintRate), TrapHits: st.sndsTraps(c.Asset, 7)}
		out.Stale = last.Via == "hostrepute" && st.hostReputeStale()
		since := st.today.AddDate(0, 0, -7)
		for _, d := range st.snds {
			if d.Ip == c.Asset && !d.Day.Before(since) {
				out.Messages7d += d.Messages
				out.JMRPReports += int(d.JmrpReports)
			}
		}
		return out
	}
	return nil
}

// dateIn returns t's date in the timezone, at 00:00 UTC (how DATE columns scan).
func dateIn(t time.Time, tz string) time.Time {
	y, m, d := t.In(zoneOf(tz)).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func zoneOf(tz string) *time.Location {
	if loc, err := time.LoadLocation(tz); err == nil && tz != "" {
		return loc
	}
	return time.UTC
}
