package engine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// IncidentDetail is what the engine stores with an incident
// (campaign_incidents.detail): the evidence at the moment it tripped,
// likely causes, recommendations and the choices offered.
type IncidentDetail struct {
	// Threshold trips: the scope's numbers when it tripped.
	Sent       int64         `json:"sent,omitempty"`
	Total      int64         `json:"total,omitempty"`
	Delivered  int64         `json:"delivered,omitempty"`
	Events     int64         `json:"events,omitempty"`
	Base       int64         `json:"base,omitempty"`
	Campaign   *IncidentRate `json:"campaign,omitempty"`
	Secondary  *IncidentRate `json:"secondary,omitempty"`
	Note       string        `json:"note,omitempty"`
	Continuing []string      `json:"continuing,omitempty"`
	Throttle   float64       `json:"throttle,omitempty"`

	// Outages.
	Reason      string            `json:"reason,omitempty"`
	ErrorKind   string            `json:"errorKind,omitempty"`
	From        *time.Time        `json:"from,omitempty"`
	To          *time.Time        `json:"to,omitempty"`
	Attempts    int               `json:"attempts,omitempty"`
	Succeeded   int               `json:"succeeded"`
	Host        string            `json:"host,omitempty"`
	Policy      string            `json:"policy,omitempty"`
	Log         []IncidentLogLine `json:"log,omitempty"`
	Probed      bool              `json:"probed,omitempty"`
	ErrorRate   float64           `json:"errorRate,omitempty"`
	SentOnRoute int64             `json:"sentOnRoute,omitempty"`
	Remaining   int64             `json:"remaining,omitempty"`
	Move        *MoveEstimate     `json:"move,omitempty"`
	WarmupNote  string            `json:"warmupNote,omitempty"`

	Causes          []IncidentCause  `json:"causes,omitempty"`
	Recommendations []string         `json:"recommendations,omitempty"`
	Options         []IncidentOption `json:"options,omitempty"`
	Changes         *ResumeChanges   `json:"changes,omitempty"`
}

// IncidentRate is a rate against its limit.
type IncidentRate struct {
	Metric string  `json:"metric"`
	Events int64   `json:"events"`
	Base   int64   `json:"base"`
	Value  float64 `json:"value"`
	Limit  float64 `json:"limit"`
}

// IncidentCause is one likely cause: a bold lead and the evidence.
type IncidentCause struct {
	Lead string `json:"lead"`
	Body string `json:"body"`
}

// IncidentOption is one way to continue.
type IncidentOption struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	Recommended bool   `json:"recommended"`
	// Action is the primary button's label when this option is chosen.
	Action string `json:"action"`
}

// ResumeChanges is what "Resume with changes" does.
type ResumeChanges struct {
	Mailbox      string  `json:"mailbox,omitempty"`
	ImportedOnly bool    `json:"importedOnly"`
	Exclude      int64   `json:"exclude"`
	Rate         float64 `json:"rate,omitempty"`
	Automation   string  `json:"automation,omitempty"`
	AutomationID string  `json:"automationId,omitempty"`
	Expected     float64 `json:"expected,omitempty"`
}

// MoveEstimate is what moving a held share to another provider costs.
type MoveEstimate struct {
	TargetID     string  `json:"targetId"`
	Target       string  `json:"target"`
	TargetShort  string  `json:"targetShort"`
	ExtraCost    float64 `json:"extraCost"`
	ExtraSeconds int64   `json:"extraSeconds"`
	// Rate is the target's rate in messages per second ("Extra time at 14 /s").
	Rate      float64 `json:"rate"`
	QuotaFree *int64  `json:"quotaFree,omitempty"`
	QuotaMax  *int64  `json:"quotaMax,omitempty"`
	Risk      string  `json:"risk"` // low, high
	RiskText  string  `json:"riskText"`
}

// analyzeThreshold explains a threshold trip with the send's own data.
func (e *Engine) analyzeThreshold(ctx context.Context, rs *runState, ts tallies, tr trip, held int64, now time.Time) IncidentDetail {
	d := IncidentDetail{Events: tr.events, Base: tr.base}
	scope := ts.total
	scopeName := "the campaign"
	switch tr.scope {
	case "mailbox_provider":
		if m := ts.mailbox[tr.mailbox]; m != nil {
			scope = *m
		}
		scopeName = mailboxName(tr.mailbox)
	case "route":
		if p := ts.provider[tr.provider]; p != nil {
			scope = *p
		}
		scopeName = rs.routes[tr.provider].shortName()
	}
	d.Sent, d.Delivered, d.Total = scope.sent, scope.delivered, scope.sent+held
	all := ts.total
	switch tr.metric {
	case "complaint":
		d.Campaign = &IncidentRate{Metric: "complaint", Events: all.complained, Base: all.delivered, Value: round2(rate(all.complained, all.delivered)), Limit: rs.thresholds.Complaint}
		d.Secondary = &IncidentRate{Metric: "bounce", Events: scope.bounced(), Base: scope.sent, Value: round2(rate(scope.bounced(), scope.sent)), Limit: rs.thresholds.Bounce}
	case "bounce":
		d.Campaign = &IncidentRate{Metric: "bounce", Events: all.bounced(), Base: all.sent, Value: round2(rate(all.bounced(), all.sent)), Limit: rs.thresholds.Bounce}
		d.Secondary = &IncidentRate{Metric: "complaint", Events: scope.complained, Base: scope.delivered, Value: round2(rate(scope.complained, scope.delivered)), Limit: rs.thresholds.MailboxComplaint}
	case "deferral":
		d.Campaign = &IncidentRate{Metric: "deferral", Events: all.deferred, Base: all.sent, Value: round2(rate(all.deferred, all.sent)), Limit: rs.thresholds.Deferral}
		d.Secondary = &IncidentRate{Metric: "bounce", Events: scope.bounced(), Base: scope.sent, Value: round2(rate(scope.bounced(), scope.sent)), Limit: rs.thresholds.Bounce}
	}
	d.Note = thresholdNote(tr)
	if tr.scope == "mailbox_provider" {
		for _, mbp := range routing.MailboxProviders {
			if mbp == tr.mailbox {
				continue
			}
			if _, onHold := rs.holds[mbp]; onHold {
				continue
			}
			for k, c := range rs.queue {
				if k.mailbox == mbp && c.queued+c.held > 0 {
					d.Continuing = append(d.Continuing, mailboxName(mbp))
					break
				}
			}
		}
		d.Throttle = rs.throttles[tr.mailbox]
	}
	cid := rs.campaign.ID
	mbpFilter := tr.mailbox
	cohorts, _ := e.Q.SendingComplaintCohorts(ctx, store.SendingComplaintCohortsParams{CampaignID: &cid, MailboxProvider: mbpFilter})
	if tr.metric != "complaint" {
		cohorts = nil
	}
	var cohort *store.SendingComplaintCohortsRow
	var cohortEvents int64
	for i := range cohorts {
		c := cohorts[i]
		if strings.HasPrefix(c.Source, "import:") && c.NeverEngaged {
			cohortEvents += c.N
			if cohort == nil {
				cohort = &cohorts[i]
			}
		}
	}
	changes := &ResumeChanges{Mailbox: tr.mailbox}
	throttle := d.Throttle
	if throttle <= 0 {
		throttle = rs.routesRate()
	}
	lower := math.Max(1, math.Floor(throttle/2))
	if throttle >= 5 && lower < 2 {
		lower = 2
	}
	changes.Rate = lower
	if cohort != nil && cohortEvents >= 2 && float64(cohortEvents) >= 0.5*float64(tr.events) {
		size, _ := e.Q.SendingCohortSize(ctx, store.SendingCohortSizeParams{CampaignID: &cid, MailboxProvider: mbpFilter, Source: cohort.Source})
		src := strings.TrimPrefix(cohort.Source, "import:")
		when := ""
		if imp, err := e.Q.SendingImportOf(ctx, store.SendingImportOfParams{WorkspaceID: rs.campaign.WorkspaceID, Source: src}); err == nil {
			when = " on " + imp.CreatedAt.In(rs.loc).Format("2 Jan")
		}
		lead := fmt.Sprintf("Imported %s contacts that never engaged.", scopeName)
		body := fmt.Sprintf("%s %s recipients came from the %s import%s and have never opened. %s of the %s complaints are from this group.",
			countText(size.Total), scopeName, sourceName(src), when, countText(cohortEvents), countText(tr.events))
		if cohort.Consent == "pending" {
			lead = fmt.Sprintf("Imported %s contacts with unclear consent.", scopeName)
			body = fmt.Sprintf("%s %s recipients came from the %s import%s with \"pending\" consent and have never opened. %s of the %s complaints are from this group.",
				countText(size.Total), scopeName, sourceName(src), when, countText(cohortEvents), countText(tr.events))
		}
		d.Causes = append(d.Causes, IncidentCause{Lead: lead, Body: body})
		ids, _ := e.Q.SendingCohortContacts(ctx, store.SendingCohortContactsParams{CampaignID: &cid, MailboxProvider: mbpFilter, Source: cohort.Source})
		if len(ids) > 0 {
			prior, err := e.Q.SendingPriorSends(ctx, store.SendingPriorSendsParams{WorkspaceID: rs.campaign.WorkspaceID, ContactIds: derefIDs(ids), CampaignID: &cid})
			if err == nil && prior == 0 {
				domain := domainOf(rs.content.FromEmail)
				d.Causes = append(d.Causes, IncidentCause{Lead: "First send to these contacts from " + domain + ".",
					Body: fmt.Sprintf("They last heard from you through %s; the new sender domain looks unfamiliar.", sourceName(src))})
			}
		}
		changes.ImportedOnly = true
		changes.Exclude = size.Remaining
		if size.Remaining > 0 {
			cohortSent := size.Total - size.Remaining
			cohortDelivered := int64(float64(cohortSent) * float64(scope.delivered) / math.Max(float64(scope.sent), 1))
			expected := rate(tr.events-cohortEvents, max(scope.delivered-cohortDelivered, 1))
			changes.Expected = round2(math.Max(expected, 0.01))
			d.Recommendations = append(d.Recommendations, fmt.Sprintf("Exclude the %s never-engaged %s imports from the remaining %s. Expected complaint rate after the change: about %s.",
				countText(size.Remaining), scopeName, countText(held), fmtRate(changes.Expected)))
		}
	} else if tr.metric == "complaint" {
		n, _ := e.Q.SendingGroupCount(ctx, store.SendingGroupCountParams{CampaignID: &cid, MailboxProvider: mbpFilter, ImportedOnly: false})
		var never int64
		for _, c := range cohorts {
			if c.NeverEngaged {
				never += c.N
			}
		}
		if never >= 2 && float64(never) >= 0.5*float64(tr.events) {
			d.Causes = append(d.Causes, IncidentCause{Lead: fmt.Sprintf("%s contacts who never engaged.", scopeName),
				Body: fmt.Sprintf("%s of the %s complaints came from recipients who have never opened an email from you.", countText(never), countText(tr.events))})
			changes.Exclude = n
			if n > 0 {
				d.Recommendations = append(d.Recommendations, fmt.Sprintf("Exclude the %s never-engaged %s recipients from the remaining %s.", countText(n), scopeName, countText(held)))
			}
		} else {
			d.Causes = append(d.Causes, IncidentCause{Lead: "No single group stands out.",
				Body: fmt.Sprintf("The %s complaints come from across the %s audience, not from one import or signup source.", countText(tr.events), scopeName)})
		}
	}
	if tr.metric == "bounce" {
		d.Causes = append(d.Causes, IncidentCause{Lead: "Stale or mistyped addresses.",
			Body: fmt.Sprintf("%s of %s %s recipients bounced; most hard bounces are unknown mailboxes, which points at an old or unverified list.", countText(tr.events), countText(scope.sent), scopeName)})
	}
	if tr.metric == "deferral" {
		d.Causes = append(d.Causes, IncidentCause{Lead: scopeName + " is slowing the send down.",
			Body: fmt.Sprintf("%s of %s messages were deferred with temporary errors (such as 421 4.7.0); %s accepts mail again at a lower rate.", countText(tr.events), countText(scope.sent), scopeName)})
	}
	defs, _ := e.Q.SendingDeferralsSince(ctx, store.SendingDeferralsSinceParams{CampaignID: &cid, MailboxProvider: mbpFilter})
	if tr.metric != "deferral" {
		if defs == 0 {
			body := fmt.Sprintf("%s sent at the provider's full rate with 0 deferrals.", scopeName)
			if d.Throttle > 0 {
				body = fmt.Sprintf("%s was throttled to %s /s with 0 deferrals.", scopeName, rateText(d.Throttle))
			}
			if tr.scope == "campaign" {
				body = "There were 0 deferrals."
			}
			d.Causes = append(d.Causes, IncidentCause{Lead: "Not a rate issue.", Body: body})
		} else {
			d.Causes = append(d.Causes, IncidentCause{Lead: "Deferrals too.", Body: fmt.Sprintf("%s messages were deferred, so %s may also be slowing the send down.", countText(defs), scopeName)})
		}
	}
	if tr.scope != "campaign" {
		d.Recommendations = append(d.Recommendations, fmt.Sprintf("Lower the %s rate to %s /s for the rest of this send.", scopeName, rateText(lower)))
	}
	if changes.Exclude > 0 {
		if a, err := e.Q.SendingAutomationByName(ctx, store.SendingAutomationByNameParams{WorkspaceID: rs.campaign.WorkspaceID, Pattern: "%permission%"}); err == nil {
			changes.Automation, changes.AutomationID = a.Name, a.ID.String()
			d.Recommendations = append(d.Recommendations, fmt.Sprintf("Move those %s contacts into the \"%s\" automation instead of this broadcast.", countText(changes.Exclude), a.Name))
		}
	}
	d.Changes = changes
	d.Options = thresholdOptions(tr, scopeName, held, changes)
	return d
}

func (rs *runState) routesRate() float64 {
	total := 0.0
	for _, r := range rs.routes {
		total += r.rate.PerSecond
	}
	return total
}

func thresholdOptions(tr trip, scopeName string, held int64, ch *ResumeChanges) []IncidentOption {
	metric := map[string]string{"complaint": "complaint", "bounce": "bounce", "deferral": "deferral"}[tr.metric]
	if tr.scope == "campaign" {
		return []IncidentOption{
			{Key: "resume", Title: "Resume as is", Body: fmt.Sprintf("Send to the %s remaining recipients. The threshold stays armed and will pause again at the next %s.", countText(held), metric), Action: "Resume as is"},
			{Key: "cancel", Title: "Cancel the campaign", Body: fmt.Sprintf("Skip the %s remaining recipients. They're marked \"not sent\" in the report.", countText(held)), Action: "Cancel the campaign"},
		}
	}
	out := []IncidentOption{
		{Key: "resume", Title: "Resume as is", Body: fmt.Sprintf("Send to all %s remaining %s recipients. The threshold stays armed and will pause again at the next %s.", countText(held), scopeName, metric), Action: "Resume as is"},
	}
	if ch != nil && (ch.Exclude > 0 || ch.Rate > 0) {
		parts := []string{}
		if ch.Exclude > 0 {
			what := "never-engaged contacts"
			if ch.ImportedOnly {
				what = "never-engaged imports"
			}
			parts = append(parts, fmt.Sprintf("Exclude %s %s", countText(ch.Exclude), what))
		}
		rest := held - ch.Exclude
		if ch.Rate > 0 {
			parts = append(parts, fmt.Sprintf("send to %s at %s /s", countText(max(rest, 0)), rateText(ch.Rate)))
		}
		if ch.Automation != "" && ch.Exclude > 0 {
			parts = append(parts, "add the excluded contacts to "+ch.Automation)
		}
		body := strings.Join(parts, ", ") + "."
		out = append(out, IncidentOption{Key: "resume_with_changes", Title: "Resume with changes (recommended)", Body: capitalize(body), Recommended: true, Action: "Resume with changes"})
	}
	out = append(out, IncidentOption{Key: "cancel_share", Title: "Cancel the " + scopeName + " share", Body: fmt.Sprintf("Skip the remaining %s %s recipients. The rest of the campaign finishes normally.", countText(held), scopeName), Action: "Cancel the " + scopeName + " share"})
	if len(out) == 2 {
		out[0].Recommended = true
		out[0].Title += " (recommended)"
	}
	return out
}

func thresholdNote(tr trip) string {
	switch tr.metric {
	case "complaint":
		switch tr.mailbox {
		case "outlook":
			return "Microsoft's SmartScreen uses complaint rate directly; staying under 0.1% protects placement for every later send."
		case "gmail":
			return "Gmail filters bulk mail on the spam rate its users report; staying under 0.1% protects placement for every later send."
		case "yahoo":
			return "Yahoo filters on the complaints its feedback loop reports; staying under 0.1% protects placement for every later send."
		}
		return "Mailbox providers filter on complaint rates; staying under 0.1% protects placement for every later send."
	case "bounce":
		return "Bounce rates above 2% tell mailbox providers the list is old; providers such as SES review accounts above 5%."
	case "deferral":
		return "Mailbox providers defer mail they want to slow down; sending slower usually clears it within the hour."
	}
	return ""
}

// analyzeOutage explains a held route.
func (e *Engine) analyzeOutage(ctx context.Context, rs *runState, r *runRoute, reason string, remaining int64, now time.Time) IncidentDetail {
	key := routeKey{rs.campaign.ID, r.provider.ID}
	l := e.callLogFor(key)
	e.mu.Lock()
	first, failures := l.firstFailure, l.failures
	lines := append([]IncidentLogLine(nil), l.lines...)
	e.mu.Unlock()
	d := IncidentDetail{Reason: reason, Attempts: failures, Remaining: remaining, Log: lines}
	if !first.IsZero() {
		f := first
		d.From = &f
	}
	if n := len(lines); n > 0 {
		t := lines[n-1].At
		d.To = &t
	}
	d.Host = hostOf(r.provider)
	if ts, err := e.tallies(ctx, rs.campaign.ID); err == nil {
		if p := ts.provider[r.provider.ID]; p != nil {
			d.SentOnRoute = p.sent
		}
	}
	if wins, err := e.Q.ProvCallWindowTotals(ctx, store.ProvCallWindowTotalsParams{ProviderIds: []uuid.UUID{r.provider.ID}, Since: now.Add(-3 * time.Minute)}); err == nil && len(wins) == 1 && wins[0].Calls > 0 {
		d.ErrorRate = round2(rate(wins[0].Errors, wins[0].Calls))
	}
	if set, err := routing.Load(ctx, e.Q, rs.campaign.WorkspaceID); err == nil {
		d.Policy = string(set.Policy(providers.StreamBroadcast))
	}
	joined := strings.ToLower(strings.Join(func() []string {
		out := make([]string, len(lines))
		for i, l := range lines {
			out[i] = l.Text
		}
		return out
	}(), " "))
	name, short := r.provider.Name, r.shortName()
	switch {
	case reason == "auth":
		d.ErrorKind = "rejected credentials"
		d.Causes = append(d.Causes, IncidentCause{Lead: name + " rejected the credentials.", Body: "They may have been rotated or revoked. Replace them in Sending providers; nothing else will make this share send."})
	case strings.Contains(joined, "econnrefused") || strings.Contains(joined, "connection refused") || strings.Contains(joined, "timeout") || strings.Contains(joined, "dial "):
		d.ErrorKind = "connection errors"
		if isSelfHosted(r.provider) {
			d.Causes = append(d.Causes, IncidentCause{Lead: name + "'s web process restarted or crashed.", Body: "Connections to " + d.Host + " are refused, which points at the process or the host rather than the API or your account."})
		} else {
			d.Causes = append(d.Causes, IncidentCause{Lead: "The network path to " + name + " is down.", Body: "Connections fail before " + name + " answers; check the server's outbound network and DNS."})
		}
		if strings.Contains(joined, "502") || strings.Contains(joined, "503") {
			d.Causes = append(d.Causes, IncidentCause{Lead: "Host or reverse-proxy outage.", Body: "502 from the proxy in front of " + name + " matches an upstream that is down."})
		}
	case strings.Contains(joined, "429") || strings.Contains(joined, "throttl"):
		d.ErrorKind = "throttling errors"
		d.Causes = append(d.Causes, IncidentCause{Lead: name + " is throttling the account.", Body: "It answered 429 Too Many Requests; the account's rate or daily quota may be lower than ReputeCast assumed."})
	default:
		d.ErrorKind = "API errors"
		if reason == "provider_down" {
			d.Causes = append(d.Causes, IncidentCause{Lead: name + " is down.", Body: "Its health checks failed; ReputeCast's provider monitor marked it down."})
		} else {
			d.Causes = append(d.Causes, IncidentCause{Lead: name + " is having an outage.", Body: "Its API failed every call; check " + name + "'s status page."})
		}
		if strings.Contains(joined, "502") || strings.Contains(joined, "503") {
			d.Causes = append(d.Causes, IncidentCause{Lead: "Host or reverse-proxy outage.", Body: "502 from the proxy in front of " + name + " matches an upstream that is down."})
		}
	}
	if r.rate.WarmingUp && r.rate.Cap > 0 {
		d.Causes = append(d.Causes, IncidentCause{Body: fmt.Sprintf("Not a quota or rate issue: %s of today's %s warm-up cap used, %s /s.", countText(r.rate.SentToday+d.SentOnRoute), countText(r.rate.Cap), rateText(r.rate.PerSecond))})
		d.WarmupNote = "The " + short + " warm-up schedule shifts by a day so the IP doesn't miss its ramp."
	} else if h := state.HealthOf(r.provider); h.Quota != nil && h.Quota.Max > 0 {
		d.Causes = append(d.Causes, IncidentCause{Body: fmt.Sprintf("Not a quota issue: %s of %s used.", countText(h.Quota.Used), countText(h.Quota.Max))})
	}
	set, _ := routing.Load(ctx, e.Q, rs.campaign.WorkspaceID)
	target := e.moveTarget(ctx, rs, r, set)
	if target != nil {
		d.Move = e.moveEstimate(ctx, rs, *target, remaining, now)
	}
	d.Options = outageOptions(short, remaining, d.SentOnRoute, d.Move)
	return d
}

func (e *Engine) moveEstimate(ctx context.Context, rs *runState, target store.Provider, n int64, now time.Time) *MoveEstimate {
	m := &MoveEstimate{TargetID: target.ID.String(), Target: target.Name, TargetShort: state.Short(target.Name, target.Adapter)}
	if pr := state.PricingOf(target); pr.Model == "metered" {
		m.ExtraCost = math.Round(float64(n)*pr.Per1000/1000*100) / 100
	}
	tr := e.rateOf(ctx, e.Q, target, rs.loc, now)
	if tr.PerSecond > 0 {
		m.ExtraSeconds = int64(math.Round(float64(n) / tr.PerSecond))
		m.Rate = tr.PerSecond
	}
	if h := state.HealthOf(target); h.Quota != nil && h.Quota.Max > 0 {
		cid := rs.campaign.ID
		left, _ := e.Q.SendingRemainingOn(ctx, store.SendingRemainingOnParams{CampaignID: &cid, ProviderID: &target.ID})
		free := max(h.Quota.Max-h.Quota.Used-left-n, 0)
		mx := h.Quota.Max
		m.QuotaFree, m.QuotaMax = &free, &mx
	}
	if tr.WarmingUp {
		m.Risk, m.RiskText = "high", "High · "+m.TargetShort+" is warming up"
	} else {
		m.Risk, m.RiskText = "low", "Low · "+m.TargetShort+" is warm"
	}
	return m
}

func outageOptions(short string, remaining, sent int64, move *MoveEstimate) []IncidentOption {
	out := []IncidentOption{{Key: "retry", Title: "Retry " + short,
		Body:   fmt.Sprintf("Keep the %s recipients on %s and retry every 60 s. If it stays down for 15 more minutes, you'll be asked again.", countText(remaining), short),
		Action: "Retry " + short}}
	if move != nil {
		out = append(out, IncidentOption{Key: "move", Title: fmt.Sprintf("Move the remaining %s share to %s (recommended)", short, move.TargetShort),
			Body:        fmt.Sprintf("%s recipients join the %s queue now. The %s already sent via %s keep their tracking.", countText(remaining), move.TargetShort, countText(sent), short),
			Recommended: true, Action: fmt.Sprintf("Move %s to %s", countText(remaining), move.TargetShort)})
	} else {
		out[0].Recommended = true
		out[0].Title += " (recommended)"
	}
	out = append(out, IncidentOption{Key: "cancel_share", Title: "Cancel the " + short + " share",
		Body:   fmt.Sprintf("Skip the %s remaining recipients. They're marked \"not sent\" in the report and can be re-sent later.", countText(remaining)),
		Action: "Cancel the " + short + " share"})
	return out
}

func outageHeadline(d IncidentDetail, name string) string {
	if d.Reason == "auth" {
		return name + " rejected the credentials"
	}
	mins := 3
	if d.From != nil && d.To != nil {
		mins = max(int(math.Round(d.To.Sub(*d.From).Minutes())), 1)
	}
	return fmt.Sprintf("%s returned %s for %d %s", name, d.ErrorKind, mins, plural(int64(mins), "minute", "minutes"))
}

func hostOf(p store.Provider) string {
	var cfg struct {
		ServerURL string `json:"server_url"`
		Host      string `json:"host"`
		Region    string `json:"region"`
	}
	_ = jsonUnmarshal(p.Config, &cfg)
	switch {
	case cfg.ServerURL != "":
		return strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(cfg.ServerURL, "https://"), "http://"), "/")
	case cfg.Host != "":
		return cfg.Host
	case cfg.Region != "":
		return "email." + cfg.Region + ".amazonaws.com"
	}
	return p.Name
}

func isSelfHosted(p store.Provider) bool {
	if d, ok := providers.Lookup(p.Adapter); ok {
		return d.Kind == "self_hosted" || d.Kind == "smtp"
	}
	return p.Adapter == "postal" || p.Adapter == "smtp"
}

func sourceName(src string) string {
	switch src {
	case "mailchimp":
		return "Mailchimp"
	case "brevo":
		return "Brevo"
	case "mailerlite":
		return "MailerLite"
	case "listmonk":
		return "listmonk"
	case "keila":
		return "Keila"
	case "csv":
		return "CSV"
	case "xlsx":
		return "spreadsheet"
	}
	if src == "" {
		return "an"
	}
	return src
}

func derefIDs(in []*uuid.UUID) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(in))
	for _, id := range in {
		if id != nil {
			out = append(out, *id)
		}
	}
	return out
}

// Text helpers in the product's number style.

func countText(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// fmtRate is a small rate with two decimals (0.18%), larger ones with one.
func fmtRate(v float64) string {
	if v < 1 {
		return strconv.FormatFloat(v, 'f', 2, 64) + "%"
	}
	return strconv.FormatFloat(v, 'f', 1, 64) + "%"
}

func fmtPct(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) + "%" }

func rateText(v float64) string {
	if v == math.Trunc(v) {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

func durationText(d time.Duration) string {
	m := int(math.Round(d.Minutes()))
	switch {
	case m < 1:
		return "under a minute"
	case m < 120:
		return strconv.Itoa(m) + " " + plural(int64(m), "minute", "minutes")
	}
	h := m / 60
	return strconv.Itoa(h) + " h " + strconv.Itoa(m%60) + " min"
}

func plural(n int64, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func itoa(n int) string { return strconv.Itoa(n) }

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func asCallError(err error, target **providers.CallError) bool { return errors.As(err, target) }

var _ = pgx.ErrNoRows
