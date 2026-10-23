package texting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/approvals"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/channels"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/store"
)

// Approvals is the approval flow (internal/approvals): the workspace's
// rules and requests apply to SMS campaigns as to email ones.
type Approvals interface {
	ChannelGate(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID) (*store.ApprovalRequest, error)
	ChannelReview(ctx context.Context, ws *store.Workspace, p *authz.Principal, campaignID uuid.UUID) (approvals.ReviewApproval, error)
}

// LargeRecipients and LargeCost: sends above these ask for the recipient
// count to be typed.
const (
	LargeRecipients = 1000
	LargeCost       = 100.0
)

// Check verdicts and statuses.
const (
	VerdictReady    = "ready"
	VerdictWarnings = "warnings"
	VerdictBlocked  = "blocked"
	Pass            = "pass"
	Warn            = "warn"
	Block           = "block"
)

// SmsCheck is one check of Review & send for SMS. The app words it from
// its facts.
type SmsCheck struct {
	Key    string `json:"key" enum:"audience,senders,registration,routes,consent,suppressions,quiet_hours,time_zones,opt_out,sender_named,links,length,budget"`
	Status string `json:"status" enum:"pass,warn,block"`
	// Count is how many recipients it concerns; Total the people it was
	// counted from.
	Count int64 `json:"count"`
	Total int64 `json:"total"`
	// Countries and Sender: the recipients' countries and the sender
	// concerned (registration and routes).
	Countries []string `json:"countries"`
	Sender    string   `json:"sender,omitempty"`
	Provider  string   `json:"provider,omitempty"`
	// Registration is the sender's status there; Since its date.
	Registration string     `json:"registration,omitempty"`
	Since        *time.Time `json:"since,omitempty"`
	// Senders are the registered senders, for the passing check.
	Senders []SmsCheckSender `json:"senders"`
	Lists   []string         `json:"lists"`
	// Reasons counts suppressions by reason (landline, invalid…).
	Reasons map[string]int64 `json:"reasons"`
	// Text is the opt-out text or the brand prefix it found.
	Text string `json:"text,omitempty"`
	// Characters, Segments and Encoding: the length check's.
	Characters int    `json:"characters"`
	Segments   int    `json:"segments"`
	Encoding   string `json:"encoding,omitempty"`
	// Links counts the links; Domain is the short-link domain; Shorteners
	// the public shorteners found.
	Links      int        `json:"links"`
	Domain     string     `json:"domain,omitempty"`
	VerifiedOn *time.Time `json:"verifiedOn,omitempty"`
	Shorteners []string   `json:"shorteners"`
	// Window is the quiet-hours window; Local the local send time.
	Window string `json:"window,omitempty"`
	Local  string `json:"local,omitempty"`
	// Cost, Remaining, Spent, After and Budget are the budget check's.
	Cost      float64 `json:"cost"`
	Remaining float64 `json:"remaining"`
	Spent     float64 `json:"spent"`
	After     float64 `json:"after"`
	Budget    float64 `json:"budget"`
	AlertPct  int     `json:"alertPct"`
	// ByArea and ByCountry split the assumed time zones (area code, the
	// country's main zone).
	ByArea    int64 `json:"byArea"`
	ByCountry int64 `json:"byCountry"`
	// Choice is the registration warning's fix in force: skip or wait.
	Choice string `json:"choice,omitempty" enum:"skip,wait,"`
	// ClearedBy and ClearedAt: who emptied the opt-out text, and when.
	ClearedBy string     `json:"clearedBy,omitempty"`
	ClearedAt *time.Time `json:"clearedAt,omitempty"`
}

// SmsCheckSender is a sender in the passing registration check.
type SmsCheckSender struct {
	Sender       string     `json:"sender"`
	Registration string     `json:"registration"`
	Since        *time.Time `json:"since,omitempty"`
	Countries    []string   `json:"countries"`
}

// planned is one recipient's text as the review plans it.
type planned struct {
	r        Recipient
	decision Decision
	segments int
	cost     float64
	// skip is why the recipient gets nothing ("pending", "no_route"…).
	skip string
}

// review is everything Review & send, the composer's cost and the send
// work from.
type review struct {
	c        store.SmsCampaignRow
	ws       *store.Workspace
	graph    *graph
	text     Text
	measured Measured
	funnel   Funnel
	plan     []planned
	checks   []SmsCheck
	verdict  string
	cost     float64
	sent     int64
	spent    float64
	lists    []string
	at       time.Time
}

func (rv *review) counts() (pass, warn, block int) {
	for _, ch := range rv.checks {
		switch ch.Status {
		case Pass:
			pass++
		case Warn:
			warn++
		case Block:
			block++
		}
	}
	return
}

// textOf assembles a campaign's text with the rules' locked parts.
func textOf(body string, r Rules) Text {
	return Text{Prefix: r.BrandPrefix, Body: body, OptOut: r.OptOutText}
}

func (s *Service) shortDomain(r Rules) string {
	if r.ShortStatus == "verified" {
		return r.ShortDomain
	}
	return ""
}

// buildReview plans a campaign: audience, routes, costs and checks.
func (s *Service) buildReview(ctx context.Context, ws *store.Workspace, c store.SmsCampaignRow) (*review, error) {
	g, err := s.graph(ctx, s.Q, ws)
	if err != nil {
		return nil, err
	}
	rv := &review{c: c, ws: ws, graph: g, at: s.now()}
	rv.text = textOf(c.Body, g.rules)
	domain := s.shortDomain(g.rules)
	rv.measured = measure(rv.text, domain)
	aud := parseAudience(c.Audience)
	if rv.funnel, err = s.funnel(ctx, ws, aud); err != nil {
		return nil, err
	}
	if rv.lists, err = s.audienceNames(ctx, ws, aud); err != nil {
		return nil, err
	}
	personalized := len(rv.measured.Fields) > 0
	decisions := map[string]Decision{}
	for _, r := range rv.funnel.Recipients {
		d, ok := decisions[r.Country]
		if !ok {
			d = g.decide(r.Country)
			decisions[r.Country] = d
		}
		p := planned{r: r, decision: d, segments: rv.measured.Count.Segments}
		if personalized {
			body := render(withShortLinks(rv.text.Body, domain, placeholder), r)
			p.segments = sms.CountText(Text{Prefix: rv.text.Prefix, Body: body, OptOut: rv.text.OptOut}.Full()).Segments
		}
		if d.Sender == nil {
			p.skip = d.Hold
		} else {
			p.cost = d.Price * float64(p.segments)
			rv.cost += p.cost
			rv.sent++
		}
		rv.plan = append(rv.plan, p)
	}
	rv.cost = round4(rv.cost)
	if rv.spent, err = s.spentThisMonth(ctx, ws); err != nil {
		return nil, err
	}
	rv.checks = s.checks(ctx, rv)
	_, w, b := rv.counts()
	switch {
	case b > 0:
		rv.verdict = VerdictBlocked
	case w > 0:
		rv.verdict = VerdictWarnings
	default:
		rv.verdict = VerdictReady
	}
	return rv, nil
}

// checks runs the SMS checks: blocks first, then warnings, then passes.
func (s *Service) checks(ctx context.Context, rv *review) []SmsCheck {
	g, f, m := rv.graph, rv.funnel, rv.measured
	out := []SmsCheck{}
	add := func(c SmsCheck) {
		if c.Countries == nil {
			c.Countries = []string{}
		}
		if c.Senders == nil {
			c.Senders = []SmsCheckSender{}
		}
		if c.Lists == nil {
			c.Lists = []string{}
		}
		if c.Reasons == nil {
			c.Reasons = map[string]int64{}
		}
		if c.Shorteners == nil {
			c.Shorteners = []string{}
		}
		out = append(out, c)
	}
	if f.CanReceive == 0 {
		add(SmsCheck{Key: "audience", Status: Block, Total: f.Base, Lists: rv.lists})
	}
	if len(g.senders) == 0 {
		add(SmsCheck{Key: "senders", Status: Block})
	}
	// Registration and routes, by sender and country.
	type gap struct {
		countries []string
		sender    string
		provider  string
		status    string
		since     *time.Time
		count     int64
	}
	gaps := map[string]*gap{}
	noRoute := map[string]int64{}
	for _, p := range rv.plan {
		if p.skip == "" {
			continue
		}
		if p.skip == "no_route" {
			noRoute[p.r.Country]++
			continue
		}
		key := p.skip
		sn := ""
		prov := ""
		if p.decision.Route != nil {
			if x, ok := g.senders[p.decision.Route.SenderID]; ok {
				sn, prov, key = senderLabel(x), x.ProviderName, key+"|"+x.ID.String()
			}
		}
		gp, ok := gaps[key]
		if !ok {
			gp = &gap{sender: sn, provider: prov, status: p.skip, since: p.decision.Reg.Since}
			gaps[key] = gp
		}
		if !slices.Contains(gp.countries, p.r.Country) {
			gp.countries = append(gp.countries, p.r.Country)
		}
		gp.count++
	}
	keys := make([]string, 0, len(gaps))
	for k := range gaps {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		gp := gaps[k]
		sort.Strings(gp.countries)
		add(SmsCheck{Key: "registration", Status: Warn, Count: gp.count, Countries: gp.countries, Sender: gp.sender, Provider: gp.provider,
			Registration: gp.status, Since: gp.since, Choice: rv.c.PendingPolicy})
	}
	if len(noRoute) > 0 {
		cs := make([]string, 0, len(noRoute))
		var n int64
		for c, k := range noRoute {
			cs = append(cs, c)
			n += k
		}
		sort.Strings(cs)
		add(SmsCheck{Key: "routes", Status: Warn, Count: n, Countries: cs})
	}
	if f.Assumed > 0 && f.CanReceive > 0 {
		add(SmsCheck{Key: "time_zones", Status: Warn, Count: f.Assumed, ByArea: f.AssumedByArea, ByCountry: f.Assumed - f.AssumedByArea})
	}
	// The text.
	if strings.TrimSpace(rv.text.OptOut) == "" {
		c := SmsCheck{Key: "opt_out", Status: Block}
		if g.rules.UpdatedBy != nil {
			if u, err := s.Q.GetUser(ctx, *g.rules.UpdatedBy); err == nil {
				c.ClearedBy, c.ClearedAt = u.Name, &g.rules.UpdatedAt
			}
		}
		add(c)
	}
	shorteners := []string{}
	for _, l := range m.Links {
		if l.Shortener && !slices.Contains(shorteners, l.Host) {
			shorteners = append(shorteners, l.Host)
		}
	}
	if len(shorteners) > 0 {
		add(SmsCheck{Key: "links", Status: Block, Links: len(m.Links), Shorteners: shorteners, Domain: g.rules.ShortDomain})
	} else if len(m.Links) > 0 && g.rules.ShortStatus != "verified" {
		add(SmsCheck{Key: "links", Status: Block, Links: len(m.Links), Domain: g.rules.ShortDomain, Registration: g.rules.ShortStatus})
	}
	budget := g.rules.Budget
	if budget > 0 && rv.cost > 0 && rv.spent+rv.cost > budget+1e-9 {
		add(SmsCheck{Key: "budget", Status: Block, Cost: rv.cost, Remaining: round2(math.Max(budget-rv.spent, 0)), Spent: round2(rv.spent),
			After: round2(rv.spent + rv.cost), Budget: budget, AlertPct: g.rules.BudgetAlertPct})
	}
	if strings.TrimSpace(rv.text.Prefix) == "" {
		add(SmsCheck{Key: "sender_named", Status: Warn})
	}
	if m.Count.Segments > 3 {
		add(SmsCheck{Key: "length", Status: Warn, Characters: m.Count.Characters, Segments: m.Count.Segments, Encoding: string(m.Count.Encoding), Cost: rv.cost})
	}
	// Passes.
	if f.CanReceive > 0 {
		add(SmsCheck{Key: "consent", Status: Pass, Count: f.CanReceive + f.Suppressed + f.Shared, Total: f.Base, Lists: rv.lists})
		add(SmsCheck{Key: "suppressions", Status: Pass, Count: f.Suppressed, Reasons: f.SuppressedByReason})
	}
	if len(g.senders) > 0 {
		pass := SmsCheck{Key: "registration", Status: Pass}
		if len(gaps) > 0 || len(noRoute) > 0 {
			pass.Total = 1 // "every other recipient"
		}
		seen := map[uuid.UUID]*SmsCheckSender{}
		for _, p := range rv.plan {
			if p.decision.Sender == nil {
				continue
			}
			sn := p.decision.Sender
			cs, ok := seen[sn.ID]
			if !ok {
				cs = &SmsCheckSender{Sender: senderLabel(*sn), Registration: p.decision.Reg.Status, Since: p.decision.Reg.Since, Countries: []string{}}
				seen[sn.ID] = cs
			}
			if !slices.Contains(cs.Countries, p.r.Country) {
				cs.Countries = append(cs.Countries, p.r.Country)
			}
		}
		for _, id := range g.order {
			if cs, ok := seen[id]; ok {
				pass.Senders = append(pass.Senders, *cs)
			}
		}
		if len(pass.Senders) > 0 {
			add(pass)
		}
	}
	window := g.rules.Window.String()
	switch rv.c.ScheduleMode {
	case "local":
		if rv.c.LocalMinute != nil {
			add(SmsCheck{Key: "quiet_hours", Status: Pass, Window: window, Local: channels.Clock(int(*rv.c.LocalMinute))})
		}
	default:
		add(SmsCheck{Key: "quiet_hours", Status: Pass, Window: window})
	}
	if strings.TrimSpace(rv.text.OptOut) != "" {
		add(SmsCheck{Key: "opt_out", Status: Pass, Text: rv.text.OptOut})
	}
	if strings.TrimSpace(rv.text.Prefix) != "" {
		add(SmsCheck{Key: "sender_named", Status: Pass, Text: rv.text.Prefix})
	}
	if len(shorteners) == 0 && (len(m.Links) == 0 || g.rules.ShortStatus == "verified") {
		add(SmsCheck{Key: "links", Status: Pass, Links: len(m.Links), Domain: g.rules.ShortDomain, VerifiedOn: g.rules.ShortVerifiedAt})
	}
	if m.Count.Segments <= 3 {
		add(SmsCheck{Key: "length", Status: Pass, Characters: m.Count.Characters, Segments: m.Count.Segments, Encoding: string(m.Count.Encoding)})
	}
	if budget <= 0 || rv.spent+rv.cost <= budget+1e-9 {
		add(SmsCheck{Key: "budget", Status: Pass, Cost: rv.cost, Remaining: round2(math.Max(budget-rv.spent, 0)), Spent: round2(rv.spent),
			After: round2(rv.spent + rv.cost), Budget: budget, AlertPct: g.rules.BudgetAlertPct})
	}
	order := map[string]int{Block: 0, Warn: 1, Pass: 2}
	sort.SliceStable(out, func(i, j int) bool { return order[out[i].Status] < order[out[j].Status] })
	return out
}

// fingerprint sums up what a person reviewed: the send refuses a review
// that no longer describes the campaign.
func (rv *review) fingerprint() string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%d|%s|%s|%d|%.4f", rv.text.Full(), rv.c.Audience, rv.c.ScheduleMode, rv.sent, rv.c.PendingPolicy, rv.c.Thresholds, rv.c.Version, rv.cost)
	if rv.c.LocalDate != nil {
		fmt.Fprintf(h, "|%s", rv.c.LocalDate.Format("2006-01-02"))
	}
	if rv.c.LocalMinute != nil {
		fmt.Fprintf(h, "|%d", *rv.c.LocalMinute)
	}
	if rv.c.SendAt != nil {
		fmt.Fprintf(h, "|%s", rv.c.SendAt.UTC().Format(time.RFC3339))
	}
	return hex.EncodeToString(h.Sum(nil))[:24]
}

// contentHash is what approvals compare to know the text changed.
func contentHash(full string) string {
	sum := sha256.Sum256([]byte(full))
	return hex.EncodeToString(sum[:])[:32]
}

// scheduleLabel writes the schedule for approvals and the audit log.
func scheduleLabel(c store.SmsCampaignRow, loc *time.Location) string {
	switch c.ScheduleMode {
	case "now":
		return "Now, holding whoever is outside their window"
	case "at":
		if c.SendAt != nil {
			return c.SendAt.In(loc).Format("2 Jan 2006 15:04") + " for everyone"
		}
	case "local":
		if c.LocalDate != nil && c.LocalMinute != nil {
			return c.LocalDate.Format("2 Jan 2006") + " at " + channels.Clock(int(*c.LocalMinute)) + " in each recipient's time zone"
		}
	}
	return "Not scheduled"
}

// scheduleText is scheduleLabel's message, for approvers who read another
// language.
func scheduleText(c store.SmsCampaignRow, loc *time.Location) i18n.Text {
	switch c.ScheduleMode {
	case "now":
		return msgs.M("approval.scheduleNow").Keep(nil)
	case "at":
		if c.SendAt != nil {
			return msgs.M("approval.scheduleAt").Keep(i18n.Args{"at": c.SendAt.In(loc)})
		}
	case "local":
		if c.LocalDate != nil && c.LocalMinute != nil {
			m := int(*c.LocalMinute)
			at := time.Date(c.LocalDate.Year(), c.LocalDate.Month(), c.LocalDate.Day(), m/60, m%60, 0, 0, time.UTC)
			return msgs.M("approval.scheduleLocal").Keep(i18n.Args{"at": at})
		}
	}
	return msgs.M("approval.scheduleNone").Keep(nil)
}

// DescribeForApproval implements approvals.ChannelCampaigns.
func (s *Service) DescribeForApproval(ctx context.Context, ws *store.Workspace, campaignID uuid.UUID) (approvals.ChannelCampaign, error) {
	c, err := s.Q.SmsCampaign(ctx, store.SmsCampaignParams{ID: campaignID, WorkspaceID: ws.ID})
	if err != nil {
		return approvals.ChannelCampaign{}, err
	}
	rv, err := s.buildReview(ctx, ws, c)
	if err != nil {
		return approvals.ChannelCampaign{}, err
	}
	_, warns, blocks := rv.counts()
	full := rv.text.Full()
	loc := location(ws.Timezone)
	label, labelText := scheduleLabel(c, loc), scheduleText(c, loc)
	snap := approvals.Snapshot{Version: int(c.Version), ContentHash: contentHash(full), Audience: c.Audience, AudienceLabels: rv.lists, AudienceCount: rv.sent,
		AudienceBaseline: rv.sent, Suppressed: rv.funnel.Suppressed, Route: []approvals.SnapshotShare{}, Cost: rv.cost, Channel: "sms", Text: full, ScheduleLabel: label, ScheduleText: labelText,
		Readiness: approvals.SnapshotReadiness{Verdict: rv.verdict, Passed: len(rv.checks) - warns - blocks, Warnings: warns, Blocks: blocks, Total: len(rv.checks), RanAt: rv.at}}
	smsSnap := &approvals.ApprovalSMSSnapshot{Text: full, Prefix: rv.text.Prefix, OptOut: rv.text.OptOut, Body: rv.text.Body, Chars: rv.measured.Count.Characters,
		Segments: rv.measured.Count.Segments, Encoding: string(rv.measured.Count.Encoding), Recipients: rv.sent, Cost: rv.cost, Schedule: label,
		Countries: []approvals.ApprovalSMSCountry{}}
	for _, line := range rv.byCountry() {
		smsSnap.Countries = append(smsSnap.Countries, approvals.ApprovalSMSCountry{Country: line.Country, Sender: line.Sender, Provider: line.Provider,
			Recipients: line.Recipients, Price: line.Price, Cost: line.Cost, Skipped: line.Skipped > 0 && line.Recipients == 0})
	}
	byProvider := map[uuid.UUID]*approvals.SnapshotShare{}
	provOrder := []uuid.UUID{}
	for _, p := range rv.plan {
		if p.decision.Sender == nil {
			continue
		}
		pid := p.decision.Sender.ProviderID
		sh, ok := byProvider[pid]
		if !ok {
			sh = &approvals.SnapshotShare{ProviderID: pid.String(), Name: p.decision.Sender.ProviderName, Short: p.decision.Sender.ProviderName,
				Adapter: p.decision.Sender.ProviderAdapter, Label: senderLabel(*p.decision.Sender)}
			byProvider[pid] = sh
			provOrder = append(provOrder, pid)
		}
		sh.Recipients++
		sh.Cost += p.cost
	}
	for _, pid := range provOrder {
		sh := byProvider[pid]
		sh.Cost = round2(sh.Cost)
		if rv.sent > 0 {
			sh.Percent = math.Round(float64(sh.Recipients)*1000/float64(rv.sent)) / 10
		}
		snap.Route = append(snap.Route, *sh)
	}
	snap.SMS = smsSnap
	cur := approvals.Current{Version: int(c.Version), ContentHash: snap.ContentHash, Audience: c.Audience, AudienceNames: rv.lists, AudienceCount: rv.sent,
		Text: full, ScheduleLabel: label, ScheduleText: labelText}
	return approvals.ChannelCampaign{Name: c.Name, Status: c.Status, Version: int(c.Version), Snapshot: snap, Current: cur, Blocks: blocks,
		SendPermission: authz.SMSSend, Facts: approvals.RuleFacts{Recipients: rv.sent, Verdict: verdictForRules(rv.verdict), Warnings: warns, Channel: "sms", Cost: rv.cost}}, nil
}

// verdictForRules maps SMS verdicts onto the approval rules' words.
func verdictForRules(v string) string {
	switch v {
	case VerdictWarnings:
		return "warnings"
	case VerdictBlocked:
		return "blocked"
	}
	return "ready"
}

// CountryLine is one country of a campaign's cost table.
type CountryLine struct {
	Country    string  `json:"country"`
	Recipients int64   `json:"recipients"`
	Skipped    int64   `json:"skipped"`
	SkipReason string  `json:"skipReason,omitempty"`
	Sender     string  `json:"sender"`
	SenderKind string  `json:"senderKind"`
	Provider   string  `json:"provider"`
	Adapter    string  `json:"adapter"`
	Segments   int     `json:"segments"`
	Price      float64 `json:"price"`
	// PendingPrice is the price once a pending sender is approved.
	PendingPrice float64 `json:"pendingPrice"`
	Cost         float64 `json:"cost"`
}

// byCountry is the cost table: countries with recipients first, by count,
// then skipped ones.
func (rv *review) byCountry() []CountryLine {
	lines := map[string]*CountryLine{}
	for _, p := range rv.plan {
		l, ok := lines[p.r.Country]
		if !ok {
			l = &CountryLine{Country: p.r.Country, Segments: p.segments}
			if p.decision.Route != nil {
				if sn, ok := rv.graph.senders[p.decision.Route.SenderID]; ok {
					l.Sender, l.SenderKind, l.Provider, l.Adapter = senderLabel(sn), sn.Kind, sn.ProviderName, sn.ProviderAdapter
				}
			}
			l.Price = p.decision.Price
			lines[p.r.Country] = l
		}
		if p.skip != "" {
			l.Skipped++
			l.SkipReason = p.skip
			l.PendingPrice = p.decision.Price
			continue
		}
		l.Recipients++
		l.Cost += p.cost
		if p.segments > l.Segments {
			l.Segments = p.segments
		}
	}
	out := make([]CountryLine, 0, len(lines))
	for _, l := range lines {
		l.Cost = round2(l.Cost)
		if l.Recipients == 0 {
			l.Price = 0
		}
		out = append(out, *l)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if (a.Recipients == 0) != (b.Recipients == 0) {
			return a.Recipients > 0
		}
		if a.Provider != b.Provider {
			return providerRank(rv.graph, a.Adapter) < providerRank(rv.graph, b.Adapter)
		}
		if a.Recipients != b.Recipients {
			return a.Recipients > b.Recipients
		}
		return a.Country < b.Country
	})
	return out
}

// providerRank orders providers as they were connected.
func providerRank(g *graph, adapter string) int {
	best := math.MaxInt
	for _, p := range g.providers {
		if p.Adapter == adapter {
			best = min(best, int(p.ConnectedAt.Unix()))
		}
	}
	return best
}

func jsonOf(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
