// Package reports builds the campaign report (spec 7.7, honest analytics
// 5.5) from campaign_counters, its exports and read-only report links, whose
// public page shows the same aggregate numbers and nothing else.
package reports

import (
	"context"
	"encoding/json"
	"errors"

	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/linkrule"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mailbox"
	provstate "github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// FinalAfter is how long after a send ends opens and clicks keep counting
// before the report calls its results final.
const FinalAfter = 72 * time.Hour

// LinkThreshold is the share of clicks (percent) under which a link is
// grouped as "Other" on the click map.
const LinkThreshold = 5.0

// HotShare is the share of clicks (percent) from which a link's bubble is red.
const HotShare = 20.0

// BacklogAfter is how old the oldest unprocessed provider event must be
// before the report says part of it is behind.
const BacklogAfter = 5 * time.Minute

// Suppressor adds addresses to the shared suppression list (the audience
// domain's suppression service), writing its own audit entry. The note is
// stored in English with its message kept, so the list reads it in each
// reader's language.
type Suppressor interface {
	AddSuppressions(ctx context.Context, ws *store.Workspace, emails []string, note string, text i18n.Text) (added, already int, err error)
}

type Service struct {
	Pool     *pgxpool.Pool
	Q        *store.Queries
	Audit    *audit.Logger
	Box      *secrets.Box
	URLs     publicurl.Builder
	Suppress Suppressor
	// Limits protects the public report endpoint; nil uses defaults.
	Limits *Limiter
	Now    func() time.Time
	// Languages adds results per language; nil leaves them out.
	Languages *languages.Service
	// Links reads the view-online pages new subscribers came from (an
	// issue's forwards); nil leaves those out.
	Links *linktoken.Signer
	// RetryAudience counts who an audience goes to, as Review & send counts
	// them (Send Readiness): what a retry of blocked addresses would reach.
	// Nil counts the blocked addresses alone.
	RetryAudience func(ctx context.Context, ws *store.Workspace, audience json.RawMessage) (int64, error)
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// unsent statuses have no results yet.
var unsent = map[string]bool{"draft": true, "approval": true, "approved": true, "scheduled": true}

func (s *Service) Report(ctx context.Context, ws *store.Workspace, id uuid.UUID) (CampaignReport, error) {
	return s.ReportIn(ctx, ws, id, "")
}

// ReportIn is the report with the click map of one language version ("" is
// the source).
func (s *Service) ReportIn(ctx context.Context, ws *store.Workspace, id uuid.UUID, lang string) (CampaignReport, error) {
	c, err := s.Q.ReportCampaign(ctx, store.ReportCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return CampaignReport{}, api.MapError(err, "Campaign")
	}
	return s.buildIn(ctx, ws, c, lang)
}

func (s *Service) build(ctx context.Context, ws *store.Workspace, c store.ReportCampaignRow) (CampaignReport, error) {
	return s.buildIn(ctx, ws, c, "")
}

func (s *Service) buildIn(ctx context.Context, ws *store.Workspace, c store.ReportCampaignRow, lang string) (CampaignReport, error) {
	now := s.now()
	loc := provstate.Location(ws.Timezone)
	l := i18n.LangFrom(ctx)
	out := CampaignReport{Mailboxes: []ReportMailboxRow{}}
	label, err := s.audienceLabel(ctx, ws.ID, c.Audience)
	if err != nil {
		return out, err
	}
	ab := parseAB(c.AbTest)
	info := ReportCampaignInfo{ID: c.ID.String(), Name: c.Name, Status: c.Status, Audience: label, ScheduledAt: c.ScheduledAt,
		StartedAt: c.StartedAt, CompletedAt: c.CompletedAt, Origin: c.Origin, Notes: []string{}, Timezone: scheduleTZ(c.Schedule)}
	if ab != nil && ab.Kind != "" {
		args := i18n.Args{"kind": ab.Kind, "kindName": strings.ReplaceAll(ab.Kind, "_", " "), "winner": ab.Winner}
		n := mNoteAB.Text(l, args)
		if ab.Winner != "" {
			n = mNoteABWinner.Text(l, args)
		}
		info.Notes = append(info.Notes, n)
	}
	if c.Note != "" {
		info.Notes = append(info.Notes, c.Note)
	}
	// Messages left out while in doubt (review GS-1): the live monitor's
	// "Leave them out" says the report shows them as not sent, in doubt.
	leftOut, err := s.Q.ReportLeftOutInDoubt(ctx, &c.ID)
	if err != nil {
		return out, err
	}
	if leftOut > 0 {
		info.Notes = append(info.Notes, mNoteInDoubt.Text(l, i18n.Args{"count": leftOut}))
	}
	if c.CompletedAt != nil {
		fa := c.CompletedAt.Add(FinalAfter)
		info.FinalAt = &fa
		info.Final = !now.Before(fa)
	}
	if c.Origin != "" {
		// Imported history arrives with its final numbers.
		info.Final = true
	}
	out.Campaign = info

	cells, err := s.Q.ReportCounterCells(ctx, c.ID)
	if err != nil {
		return out, err
	}
	var sent int64
	for _, cell := range cells {
		sent += cell.Sent
	}
	if sent == 0 && unsent[c.Status] {
		ns := &ReportNotSent{Status: c.Status, ScheduledAt: c.ScheduledAt, Timezone: info.Timezone}
		if r, err := s.Q.ReportLatestReadiness(ctx, c.ID); err == nil {
			ns.Readiness = r.Verdict
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return out, err
		}
		if last, err := s.Q.ReportLatestSent(ctx, store.ReportLatestSentParams{WorkspaceID: ws.ID, ExceptID: c.ID}); err == nil {
			ns.LastReport = &ReportCampaignRef{ID: last.ID.String(), Name: last.Name}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return out, err
		}
		out.NotSent = ns
		out.Headline.SentBy = []ReportProviderCount{}
		out.Headline.NotReportedBy = []string{}
		out.Comparisons = ReportComparisons{Tiles: []ReportComparison{}, Unavailable: "not_final"}
		out.Engagement.Points = []ReportEngagementPoint{}
		out.ClickMap.Links = []ReportLink{}
		out.ClickMap.Variants = []ReportVariantClickMap{}
		out.Insights = []ReportInsight{}
		out.Providers.Columns = []ReportProviderColumn{}
		out.Bounces.Rows = []ReportBounceRow{}
		out.Unsubscribes.Reasons = []ReportUnsubscribeReason{}
		return out, nil
	}

	routes, err := s.Q.ReportRoutes(ctx, c.ID)
	if err != nil {
		return out, err
	}
	provs, err := s.Q.ProvListActive(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Headline, out.Mailboxes = headline(cells, c.Origin)
	if out.Comparisons, err = s.compare(ctx, ws, c, cells, info.Final); err != nil {
		return out, err
	}
	if out.Growth, err = s.growth(ctx, ws, c); err != nil {
		return out, err
	}
	if out.Providers, err = s.providerTable(ctx, c, cells, routes, loc); err != nil {
		return out, err
	}
	if err := s.trackingSources(ctx, c, &out.Providers, provs); err != nil {
		return out, err
	}
	hourly, err := s.Q.ReportHourly(ctx, store.ReportHourlyParams{CampaignID: c.ID, MaxHour: 71})
	if err != nil {
		return out, err
	}
	out.Engagement = engagement(hourly, out.Headline.Delivered, hoursSoFar(c, info, now))
	links, err := s.Q.ReportLinks(ctx, c.ID)
	if err != nil {
		return out, err
	}
	if out.ClickMap, err = s.clickMaps(ctx, l, c, ab, links); err != nil {
		return out, err
	}
	// The send's bot clicks: on every link, whichever variant's or language
	// version's email the map shows.
	for _, lk := range links {
		out.Headline.BotClicks += lk.BotClicks
	}
	if s.Languages != nil {
		if out.Languages, err = s.Languages.CampaignResults(ctx, ws, c.ID, c.Audience); err != nil {
			return out, err
		}
		if out.Languages != nil {
			// The map shows one language version: its HTML and its links.
			if html, from, to, ok := s.Languages.ClickMapVersion(ctx, c.ID, lang); ok {
				mine := make([]store.ReportLinksRow, 0, len(links))
				for _, l := range links {
					if int(l.Position) >= from && int(l.Position) < to {
						l.Position -= int32(from)
						mine = append(mine, l)
					}
				}
				out.ClickMap = clickMap(l, linkrule.Mark(html), mine, linkrule.Rows(html))
				out.ClickMap.Language = lang
				if lang == "" && len(out.Languages.ClickMaps) > 0 {
					out.ClickMap.Language = out.Languages.ClickMaps[0].Language
				}
			}
		}
	}
	cid := c.ID
	codes, err := s.Q.ReportBounceCodes(ctx, store.ReportBounceCodesParams{CampaignID: &cid, WorkspaceID: ws.ID})
	if err != nil {
		return out, err
	}
	out.Bounces = bounceTable(l, codes)
	reasons, err := s.Q.ReportUnsubscribeReasons(ctx, store.ReportUnsubscribeReasonsParams{CampaignID: &cid, WorkspaceID: ws.ID})
	if err != nil {
		return out, err
	}
	if c.Origin != "" && len(reasons) == 0 {
		// History imported from another platform has its unsubscribe total
		// but nobody was asked why: no reasons, not "No reason given".
		out.Unsubscribes = ReportUnsubscribes{Total: out.Headline.Unsubscribes, Reasons: []ReportUnsubscribeReason{}}
	} else {
		out.Unsubscribes = unsubscribes(l, out.Headline.Unsubscribes, reasons)
	}
	if out.Insights, err = s.insights(ctx, l, ws, c, &out); err != nil {
		return out, err
	}
	if ab != nil && len(ab.Variants) >= 2 {
		t := abResult(l, ab, parseSendAB(c.SendAb), cells, c, now)
		out.ABTest = &t
		if t.State != "testing" && t.RolloutAt != nil && c.StartedAt != nil && t.Remaining > 0 {
			out.Engagement.Rollout = &ReportRollout{Variant: t.Winner, At: *t.RolloutAt, Hour: round(t.RolloutAt.Sub(*c.StartedAt).Hours(), 2)}
		}
	}
	if !info.Final {
		if out.Backlog, err = s.backlog(ctx, ws, routes, provs, now); err != nil {
			return out, err
		}
	}
	return out, nil
}

// scheduleTZ reads the timezone of a campaign's schedule.
func scheduleTZ(raw json.RawMessage) string {
	var sc struct {
		Timezone string `json:"timezone"`
	}
	_ = json.Unmarshal(raw, &sc)
	return sc.Timezone
}

// audienceLabel turns an audience definition into "Weekly Digest" or
// "All subscribers" (every public list of the workspace), as the campaigns
// list does.
func (s *Service) audienceLabel(ctx context.Context, wsID uuid.UUID, raw json.RawMessage) (string, error) {
	var a struct {
		Lists    []uuid.UUID `json:"lists"`
		Segments []uuid.UUID `json:"segments"`
	}
	// An audience that is missing or doesn't parse has no label; the header
	// leaves it out.
	if err := json.Unmarshal(raw, &a); err != nil || (len(a.Lists) == 0 && len(a.Segments) == 0) {
		return "", nil //nolint:nilerr // no audience, no label
	}
	lists, err := s.Q.ListWorkspaceLists(ctx, wsID)
	if err != nil {
		return "", err
	}
	var public []uuid.UUID
	for _, l := range lists {
		// "All subscribers" is every public list but the ones a store fills (X20).
		if l.Public && l.Origin != "store" {
			public = append(public, l.ID)
		}
	}
	lang := i18n.LangFrom(ctx)
	if len(a.Segments) == 0 && len(public) > 1 && !slices.ContainsFunc(public, func(id uuid.UUID) bool { return !slices.Contains(a.Lists, id) }) {
		return mAudienceAll.Text(lang, nil), nil
	}
	names := make(map[uuid.UUID]string, len(lists))
	for _, l := range lists {
		names[l.ID] = l.Name
	}
	parts := []string{}
	for _, id := range a.Lists {
		if n, ok := names[id]; ok {
			parts = append(parts, n)
		}
	}
	if n := len(a.Segments); n > 0 {
		parts = append(parts, mAudienceSegments.Text(lang, i18n.Args{"count": n}))
	}
	return strings.Join(parts, " + "), nil
}

// counts sums a slice of counter cells.
type counts struct {
	sent, delivered, hard, soft, complained, rejected, human, privacy, machine, clicked, unsub int64
}

func (a *counts) add(c store.ReportCounterCellsRow) {
	a.sent += c.Sent
	a.delivered += c.Delivered
	a.hard += c.HardBounced
	a.soft += c.SoftBounced
	a.complained += c.Complained
	a.rejected += c.Rejected
	a.human += c.OpenedHuman
	a.privacy += c.OpenedPrivacy
	a.machine += c.OpenedMachine
	a.clicked += c.Clicked
	a.unsub += c.Unsubscribed
}

// measurable are the delivered recipients whose opens can be measured:
// all but the privacy-protected (Apple Mail opened them on delivery).
func (a counts) measurable() int64 { return max(a.delivered-a.privacy, 0) }

func (a counts) rates() ReportRates {
	return ReportRates{
		Sent: a.sent, Delivered: a.delivered,
		DeliveredRate: rate(a.delivered, a.sent), HumanOpenRate: rate(a.human, a.delivered), MachineOpenRate: rate(a.machine, a.delivered),
		Measurable: a.measurable(), HumanOpenRateMeasurable: rate(a.human, a.measurable()), PrivacyOpenRate: rate(a.privacy, a.delivered),
		ClickRate: rate(a.clicked, a.delivered), BounceRate: rate(a.hard+a.soft, a.sent), ComplaintRate: rate(a.complained, a.delivered),
		UnsubscribeRate: rate(a.unsub, a.delivered),
	}
}

// Mailbox providers in display order, with their names.
var mailboxes = []struct{ key, name string }{
	{"gmail", "Gmail"}, {"outlook", "Outlook"}, {"yahoo", "Yahoo"}, {"icloud", "iCloud"}, {"other", "Other"},
}

func mailboxKey(k string) string {
	for _, m := range mailboxes {
		if m.key == k {
			return k
		}
	}
	return "other"
}

// headline sums the counters into the tiles and the mailbox-provider rows.
func headline(cells []store.ReportCounterCellsRow, origin string) (ReportHeadline, []ReportMailboxRow) {
	var all counts
	byMailbox := map[string]*counts{}
	byProvider := map[uuid.UUID]*ReportProviderCount{}
	order := []uuid.UUID{}
	for _, c := range cells {
		all.add(c)
		k := mailboxKey(c.MailboxProvider)
		if byMailbox[k] == nil {
			byMailbox[k] = &counts{}
		}
		byMailbox[k].add(c)
		pc := byProvider[c.ProviderID]
		if pc == nil {
			pc = &ReportProviderCount{ProviderID: c.ProviderID.String(), Short: providerShort(c.ProviderName, c.Adapter, origin)}
			byProvider[c.ProviderID] = pc
			order = append(order, c.ProviderID)
		}
		pc.Count += c.Sent
	}
	h := ReportHeadline{
		Sent: all.sent, SentBy: []ReportProviderCount{}, Delivered: all.delivered, DeliveredRate: rate(all.delivered, all.sent),
		HumanOpens: all.human, HumanOpenRate: rate(all.human, all.delivered), MachineOpens: all.machine, MachineOpenRate: rate(all.machine, all.delivered),
		Measurable: all.measurable(), HumanOpenRateMeasurable: rate(all.human, all.measurable()), PrivacyOpens: all.privacy, PrivacyOpenRate: rate(all.privacy, all.delivered),
		Clicks: all.clicked, ClickRate: rate(all.clicked, all.delivered), ClickToOpen: rate(all.clicked, all.human),
		Unsubscribes: all.unsub, UnsubscribeRate: rate(all.unsub, all.delivered), Complaints: all.complained, ComplaintRate: rate(all.complained, all.delivered),
		Bounces: all.hard + all.soft, HardBounces: all.hard, SoftBounces: all.soft, BounceRate: rate(all.hard+all.soft, all.sent), Rejected: all.rejected,
	}
	// Undelivered: sent, and neither delivered nor bounced (yet). Rejected
	// messages were never sent: a provider that drops a message it took
	// takes it back out of sent (internal/counters).
	h.Undelivered = max(all.sent-all.delivered-all.hard-all.soft, 0)
	for _, id := range order {
		h.SentBy = append(h.SentBy, *byProvider[id])
	}
	sort.SliceStable(h.SentBy, func(i, j int) bool { return h.SentBy[i].Count > h.SentBy[j].Count })
	// Complaints only where they are reported (review F-2): Gmail and
	// iCloud never tell senders, so their rate is n/a with the reason, and
	// the headline's rate where reported leaves their deliveries out.
	rows := []ReportMailboxRow{}
	h.NotReportedBy = []string{}
	var reported counts
	for _, m := range mailboxes {
		c := byMailbox[m.key]
		if c == nil || c.sent == 0 {
			continue
		}
		r := ReportMailboxRow{Key: m.key, Name: m.name, ReportRates: c.rates()}
		if mailbox.ReportsComplaints(m.key) {
			reported.delivered += c.delivered
			reported.complained += c.complained
		} else {
			r.ComplaintRate, r.ComplaintRateNA = nil, mailbox.NotReported
			h.NotReportedBy = append(h.NotReportedBy, m.name)
		}
		rows = append(rows, r)
	}
	h.ComplaintRateReported = rate(reported.complained, reported.delivered)
	return h, rows
}

// providerShort names a provider in badges; history imported from another
// platform is labelled with that platform.
func providerShort(name, adapter, origin string) string {
	if name == "" && origin != "" {
		return strings.ToUpper(origin[:1]) + origin[1:]
	}
	if name == "" {
		return "Unknown"
	}
	return provstate.Short(name, adapter)
}

func (s *Service) providerTable(ctx context.Context, c store.ReportCampaignRow, cells []store.ReportCounterCellsRow, routes []store.ReportRoutesRow, loc *time.Location) (ReportProviderTable, error) {
	out := ReportProviderTable{Columns: []ReportProviderColumn{}}
	byProv := map[uuid.UUID]*counts{}
	names := map[uuid.UUID][2]string{}
	order := []uuid.UUID{}
	var total int64
	for _, cell := range cells {
		if byProv[cell.ProviderID] == nil {
			byProv[cell.ProviderID] = &counts{}
			names[cell.ProviderID] = [2]string{cell.ProviderName, cell.Adapter}
			order = append(order, cell.ProviderID)
		}
		byProv[cell.ProviderID].add(cell)
		total += cell.Sent
	}
	route := map[uuid.UUID]store.ReportRoutesRow{}
	var planned int64
	for _, r := range routes {
		route[r.ProviderID] = r
		planned += r.Planned
		if byProv[r.ProviderID] == nil {
			byProv[r.ProviderID] = &counts{}
			names[r.ProviderID] = [2]string{r.ProviderName, r.Adapter}
			order = append(order, r.ProviderID)
		}
	}
	ids := make([]uuid.UUID, 0, len(order))
	for _, id := range order {
		if id != uuid.Nil {
			ids = append(ids, id)
		}
	}
	warm := map[uuid.UUID]store.ReportWarmupAtRow{}
	if len(ids) > 0 && c.StartedAt != nil {
		plans, err := s.Q.ReportWarmupAt(ctx, ids)
		if err != nil {
			return out, err
		}
		for _, p := range plans {
			if _, seen := warm[p.ProviderID]; !seen {
				warm[p.ProviderID] = p
			}
		}
	}
	for _, id := range order {
		n := names[id]
		col := ReportProviderColumn{ProviderID: id.String(), Name: n[0], Short: providerShort(n[0], n[1], c.Origin), Adapter: n[1], Color: provstate.Color(n[1]), ReportRates: byProv[id].rates()}
		if col.Name == "" {
			col.Name = col.Short
		}
		if r, ok := route[id]; ok {
			cost := round(float64(r.CostMicros)/1e6, 2)
			col.Cost = &cost
			if r.DeliveryP50Ms > 0 {
				v := int(r.DeliveryP50Ms)
				col.MedianDeliveryMs = &v
			}
			if planned > 0 {
				col.Percent = round(float64(r.Planned)/float64(planned)*100, 1)
			}
		} else if total > 0 {
			col.Percent = round(float64(byProv[id].sent)/float64(total)*100, 1)
		}
		if w, ok := warm[id]; ok && c.StartedAt != nil {
			day := provstate.WarmupDay(w.StartDate, *c.StartedAt, loc)
			if day >= 1 && day <= int(w.Days) {
				col.WarmupDay = &day
				col.EngagedFirst = w.EngagedFirst
			}
		}
		out.Columns = append(out.Columns, col)
	}
	sort.SliceStable(out.Columns, func(i, j int) bool { return out.Columns[i].Sent > out.Columns[j].Sent })
	sent := 0
	for _, col := range out.Columns {
		if col.Sent > 0 {
			sent++
		}
	}
	out.Split = sent > 1
	return out, nil
}

// trackingSources says, for each sending provider of the send, whose
// tracking counted its opens and clicks and how its deliveries and bounces
// are known (ReportStates' tracking source notes). A provider whose own
// tracking is kept counted the send when it reported its opens or clicks.
func (s *Service) trackingSources(ctx context.Context, c store.ReportCampaignRow, t *ReportProviderTable, provs []store.Provider) error {
	byID := make(map[string]store.Provider, len(provs))
	for _, p := range provs {
		byID[p.ID.String()] = p
	}
	for i := range t.Columns {
		col := &t.Columns[i]
		col.TrackedBy, col.Events = "reputecast", "none"
		col.DeliveryInferred = col.Adapter == "smtp"
		p, ok := byID[col.ProviderID]
		if !ok {
			continue
		}
		switch provstate.Caps(p).Webhooks {
		case "SNS":
			col.Events = "sns"
		case "Bounce mailbox":
			col.Events = "bounce_mailbox"
		case "":
		default:
			col.Events = "webhook"
		}
		if provstate.TrackingOf(p).Mode != "provider" {
			continue
		}
		reported, err := s.Q.ReportProviderTracked(ctx, store.ReportProviderTrackedParams{CampaignID: c.ID, ProviderID: p.ID})
		if err != nil {
			return err
		}
		if reported {
			col.TrackedBy = "provider"
		}
	}
	return nil
}

// hoursSoFar is how many hours of the 72-hour window have passed.
func hoursSoFar(c store.ReportCampaignRow, info ReportCampaignInfo, now time.Time) int {
	if info.Final || c.StartedAt == nil {
		return 72
	}
	h := int(now.Sub(*c.StartedAt).Hours())
	return min(max(h, 0), 72)
}

// engagement turns hourly counts into cumulative points (hour 0 is the send).
func engagement(rows []store.ReportHourlyRow, delivered int64, hours int) ReportEngagement {
	out := ReportEngagement{Points: []ReportEngagementPoint{}, Delivered: delivered, Hours: hours}
	if len(rows) == 0 {
		return out
	}
	out.Available = true
	byHour := map[int]store.ReportHourlyRow{}
	last := 0
	for _, r := range rows {
		byHour[int(r.Hour)] = r
		last = max(last, int(r.Hour)+1)
	}
	end := max(min(hours, 72), min(last, 72))
	var h, p, m, k int64
	out.Points = append(out.Points, ReportEngagementPoint{Hour: 0})
	var at6 int64
	for i := 0; i < end; i++ {
		r := byHour[i]
		h += r.OpenedHuman
		p += r.OpenedPrivacy
		m += r.OpenedMachine
		k += r.Clicked
		out.Points = append(out.Points, ReportEngagementPoint{Hour: i + 1, HumanOpens: h, PrivacyOpens: p, MachineOpens: m, Clicks: k})
		if i+1 == 6 {
			at6 = h
		}
	}
	if end < 6 {
		at6 = h
	}
	if h > 0 {
		v := round(float64(at6)/float64(h)*100, 1)
		out.HumanWithin6h = &v
	}
	return out
}

// clickMap sizes each link against all link clicks and groups the small
// ones. A link's human clicks count each person once (a repeat click on the
// same link doesn't add one), so the map's total is people per link: 2,996
// people can make 3,412 human clicks across the links (Report board).
//
// lines are the email's rows of links (linkrule.Rows): a row whose links
// are all under the threshold is one entry instead of joining "Other".
func clickMap(l i18n.Lang, html string, rows []store.ReportLinksRow, lines [][]int) ReportClickMap {
	rows = groupPersonal(l, rows)
	out := ReportClickMap{HTML: html, Available: strings.TrimSpace(html) != "", Links: []ReportLink{}, Threshold: LinkThreshold, Rows: []ReportLinkRow{},
		Variants: []ReportVariantClickMap{}}
	for _, r := range rows {
		out.Total += r.UniqueClicks
		out.BotTotal += r.BotClicks
	}
	var other ReportLinkGroup
	type labelled struct {
		label  string
		clicks int64
	}
	var grouped []labelled
	for _, r := range rows {
		l := ReportLink{ID: r.ID.String(), URL: r.Url, Label: linkLabel(r.Label, r.Url), Position: int(r.Position), Clicks: r.Clicks, UniqueClicks: r.UniqueClicks,
			BotClicks: r.BotClicks}
		if out.Total > 0 {
			l.Share = round(float64(r.UniqueClicks)/float64(out.Total)*100, 1)
		}
		l.Grouped = l.Share < LinkThreshold
		l.Hot = l.Share >= HotShare
		out.Links = append(out.Links, l)
	}
	// Rows of small links: one entry each.
	inRow := map[int]bool{}
	byPos := map[int]int{}
	last := -1
	for i, lk := range out.Links {
		byPos[lk.Position] = i
		last = max(last, lk.Position)
	}
	for _, line := range lines {
		row := ReportLinkRow{Positions: []int{}, Labels: []string{}}
		small := true
		for _, p := range line {
			i, ok := byPos[p]
			if !ok || !out.Links[i].Grouped {
				small = false
				break
			}
			lk := out.Links[i]
			row.Positions = append(row.Positions, p)
			row.Labels = append(row.Labels, lk.Label)
			row.Clicks += lk.UniqueClicks
			row.BotClicks += lk.BotClicks
		}
		if !small || len(row.Positions) < 2 {
			continue
		}
		switch {
		case row.Positions[0] <= 1:
			row.Place = "header"
		case row.Positions[len(row.Positions)-1] == last:
			row.Place = "footer"
		default:
			row.Place = "body"
		}
		if out.Total > 0 {
			row.Share = round(float64(row.Clicks)/float64(out.Total)*100, 1)
		}
		for _, p := range row.Positions {
			inRow[p] = true
			out.Links[byPos[p]].Grouped = false
		}
		out.Rows = append(out.Rows, row)
	}
	for _, lk := range out.Links {
		if lk.Grouped && !inRow[lk.Position] && lk.UniqueClicks > 0 {
			other.Links++
			other.Clicks += lk.UniqueClicks
			grouped = append(grouped, labelled{lk.Label, lk.UniqueClicks})
		}
	}
	if other.Links > 0 && out.Total > 0 {
		other.Share = round(float64(other.Clicks)/float64(out.Total)*100, 1)
		sort.SliceStable(grouped, func(i, j int) bool { return grouped[i].clicks > grouped[j].clicks })
		other.Labels = []string{}
		for i := 0; i < len(grouped) && i < 2; i++ {
			other.Labels = append(other.Labels, grouped[i].label)
		}
		out.Other = &other
	}
	return out
}

// referralLinks are the Referral block's personal links: each reader's own
// link, share links and rewards page (Grow, brief §6).
var referralLinks = regexp.MustCompile(`\{\{-?\s*referral\.`)

// groupPersonal makes the Referral block's links one entry of the click
// map, "Referral links (personal)", at the first one's place.
func groupPersonal(l i18n.Lang, rows []store.ReportLinksRow) []store.ReportLinksRow {
	out := make([]store.ReportLinksRow, 0, len(rows))
	at := -1
	for _, r := range rows {
		if !referralLinks.MatchString(r.Url) {
			out = append(out, r)
			continue
		}
		if at < 0 {
			at = len(out)
			r.Label = mReferralLinks.Text(l, nil)
			out = append(out, r)
			continue
		}
		out[at].Clicks += r.Clicks
		out[at].UniqueClicks += r.UniqueClicks
		out[at].BotClicks += r.BotClicks
	}
	return out
}

// linkLabel is the link's text, or its address without the scheme.
func linkLabel(label, url string) string {
	if l := strings.TrimSpace(label); l != "" {
		return l
	}
	u := strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
	return strings.TrimSuffix(u, "/")
}

// Unsubscribe reasons in display order with their bar tones; reason.<key>
// names them.
var reasonDefs = []struct{ key, tone string }{
	{"too_frequent", "ink"},
	{"not_relevant", "ink"},
	{"none", "muted"},
	{"never_signed_up", "bad"},
	{"spam", "bad"},
	{"other", "muted"},
}

// unsubscribes reads the answers on the unsubscribe page. People who left
// without answering (or with one click from their mail app) are "No reason
// given"; people who chose fewer emails didn't leave and are counted apart.
func unsubscribes(l i18n.Lang, total int64, rows []store.ReportUnsubscribeReasonsRow) ReportUnsubscribes {
	out := ReportUnsubscribes{Total: total, Reasons: []ReportUnsubscribeReason{}}
	by := map[string]int64{}
	var answered int64
	for _, r := range rows {
		if r.ChoseFewer {
			out.ChoseFewer += r.N
			continue
		}
		if r.Reason == "one_click" {
			continue
		}
		by[r.Reason] += r.N
		answered += r.N
	}
	if total < answered {
		total = answered
		out.Total = answered
	}
	by["none"] = total - answered
	for _, d := range reasonDefs {
		n := by[d.key]
		if n <= 0 {
			continue
		}
		r := ReportUnsubscribeReason{Key: d.key, Label: reasonLabel(l, d.key), Count: n, Tone: d.tone}
		if total > 0 {
			r.Share = round(float64(n)/float64(total)*100, 1)
		}
		out.Reasons = append(out.Reasons, r)
	}
	// Most common first; "Other" stays last.
	sort.SliceStable(out.Reasons, func(i, j int) bool {
		a, b := out.Reasons[i], out.Reasons[j]
		if (a.Key == "other") != (b.Key == "other") {
			return b.Key == "other"
		}
		return a.Count > b.Count
	})
	return out
}

// backlog reports provider events that wait to be processed for the
// campaign's providers, when the oldest is older than BacklogAfter.
func (s *Service) backlog(ctx context.Context, ws *store.Workspace, routes []store.ReportRoutesRow, provs []store.Provider, now time.Time) (*ReportIngestBacklog, error) {
	if len(routes) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, 0, len(provs))
	for _, p := range provs {
		ids = append(ids, p.ID)
	}
	rows, err := s.Q.ReportIngestBacklog(ctx, store.ReportIngestBacklogParams{WorkspaceID: ws.ID, ProviderIds: ids})
	if err != nil {
		return nil, err
	}
	inRoute := map[uuid.UUID]bool{}
	for _, r := range routes {
		inRoute[r.ProviderID] = true
	}
	byID := map[uuid.UUID]store.Provider{}
	for _, p := range provs {
		byID[p.ID] = p
	}
	out := &ReportIngestBacklog{Providers: []ReportBacklogProvider{}, Current: []string{}}
	behind := map[uuid.UUID]bool{}
	for _, r := range rows {
		if !inRoute[r.ProviderID] || now.Sub(r.Oldest) < BacklogAfter {
			continue
		}
		p := byID[r.ProviderID]
		feed := provstate.Caps(p).Webhooks
		if feed == "" {
			feed = "webhook"
		}
		out.Providers = append(out.Providers, ReportBacklogProvider{ProviderID: r.ProviderID.String(), Name: p.Name, Short: provstate.Short(p.Name, p.Adapter),
			Feed: feed, Waiting: r.Waiting, Since: r.Oldest})
		behind[r.ProviderID] = true
		if out.Since.IsZero() || r.Oldest.Before(out.Since) {
			out.Since = r.Oldest
		}
	}
	if len(out.Providers) == 0 {
		return nil, nil
	}
	out.Minutes = int(now.Sub(out.Since).Minutes())
	for _, p := range provs {
		if !behind[p.ID] {
			out.Current = append(out.Current, provstate.Short(p.Name, p.Adapter))
		}
	}
	// "Postal and Postmark events are current": by name, as the board reads.
	sort.Strings(out.Current)
	return out, nil
}
