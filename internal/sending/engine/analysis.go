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

	"github.com/hostrepute/reputecast/internal/i18n"
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
	Reason    string            `json:"reason,omitempty"`
	ErrorKind string            `json:"errorKind,omitempty"`
	From      *time.Time        `json:"from,omitempty"`
	To        *time.Time        `json:"to,omitempty"`
	Attempts  int               `json:"attempts,omitempty"`
	Succeeded int               `json:"succeeded"`
	Host      string            `json:"host,omitempty"`
	Policy    string            `json:"policy,omitempty"`
	Log       []IncidentLogLine `json:"log,omitempty"`
	Probed    bool              `json:"probed,omitempty"`
	// Retried: the share was retried for 15 minutes and still failed, so a
	// person is asked again (outages.go).
	Retried     bool          `json:"retried,omitempty"`
	ErrorRate   float64       `json:"errorRate,omitempty"`
	SentOnRoute int64         `json:"sentOnRoute,omitempty"`
	Remaining   int64         `json:"remaining,omitempty"`
	Move        *MoveEstimate `json:"move,omitempty"`
	WarmupNote  string        `json:"warmupNote,omitempty"`

	// Messages in doubt (doubt.go), per sending provider.
	InDoubt []InDoubtShare `json:"inDoubt,omitempty"`

	Causes          []IncidentCause  `json:"causes,omitempty"`
	Recommendations []string         `json:"recommendations,omitempty"`
	Options         []IncidentOption `json:"options,omitempty"`
	Changes         *ResumeChanges   `json:"changes,omitempty"`

	// texts are the messages the sentences above come from (live_lang.go).
	texts *detailTexts
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
	Mailbox string `json:"mailbox,omitempty"`
	// Group is who Exclude takes out of the rest of the send: "imported"
	// (never-engaged imports, from Source when set), "lapsed" (at risk or
	// sunset) or "never" (never opened). ImportedOnly is the older spelling
	// of "imported".
	Group        string  `json:"group,omitempty"`
	Source       string  `json:"source,omitempty"`
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
	// Checks are what ReputeCast checked on the target before offering
	// it (review GS-5): the sender's domain, its quota, its warm-up, and
	// what happens to the recipients held for the A/B test or the canary.
	Checks []MoveCheck `json:"checks,omitempty"`
}

// MoveCheck is one check a move's target passed ("pass") or that couldn't
// be told ("unknown"), as a sentence.
type MoveCheck struct {
	Key    string `json:"key"`
	Result string `json:"result" enum:"pass,warn,unknown"`
	Text   string `json:"text"`
}

// analyzeThreshold explains a threshold trip with the send's own data.
func (e *Engine) analyzeThreshold(ctx context.Context, rs *runState, ts tallies, tr trip, held int64, now time.Time) IncidentDetail {
	d := IncidentDetail{Events: tr.events, Base: tr.base}
	scope := ts.total
	scopeName := "the campaign"
	// sc names the scope in the sentences: a mailbox provider by its key
	// ("Other" is a word), a route by its provider's badge name.
	sc := i18n.Args{"scope": tr.scope, "who": ""}
	switch tr.scope {
	case "mailbox_provider":
		if m := ts.mailbox[tr.mailbox]; m != nil {
			scope = *m
		}
		scopeName = mailboxName(tr.mailbox)
		sc["mailbox"] = tr.mailbox
	case "route":
		if p := ts.provider[tr.provider]; p != nil {
			scope = *p
		}
		scopeName = rs.routes[tr.provider].shortName()
		sc["who"] = scopeName
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
	if note := thresholdNote(tr); note != "" {
		d.setNote(keep(note, nil))
	}
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
	// who names the scope's recipients in sentences: "Outlook" for a
	// mailbox provider, nothing for a route or the whole campaign (whose
	// cohorts are counted campaign-wide).
	who := ""
	if tr.scope == "mailbox_provider" {
		who = scopeName
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
	if tr.metric == "complaint" {
		e.complaintCohort(ctx, rs, tr, scope, held, who, &d, changes)
	}
	if tr.metric == "bounce" {
		d.addCause(keep("cause.staleLead", nil), keep("cause.stale", with(sc, "events", tr.events, "sent", scope.sent)))
	}
	if tr.metric == "deferral" {
		d.addCause(keep("cause.slowingLead", sc), keep("cause.slowing", with(sc, "events", tr.events, "sent", scope.sent)))
	}
	defs, _ := e.Q.SendingDeferralsSince(ctx, store.SendingDeferralsSinceParams{CampaignID: &cid, MailboxProvider: mbpFilter})
	if tr.metric != "deferral" {
		if defs == 0 {
			body := keep("cause.fullRate", sc)
			if d.Throttle > 0 {
				body = keep("cause.throttled", with(sc, "throttleRate", d.Throttle))
			}
			if tr.scope == "campaign" {
				body = keep("cause.noDeferrals", nil)
			}
			d.addCause(keep("cause.notRateLead", nil), body)
		} else {
			d.addCause(keep("cause.deferralsLead", nil), keep("cause.deferrals", with(sc, "deferred", defs)))
		}
	}
	if tr.scope != "campaign" {
		d.addRecommendation(keep("rec.lower", with(sc, "lowerRate", lower)))
	}
	if changes.Exclude > 0 {
		// Lapsed readers go to the re-engagement flow; imports and people
		// who never opened are asked for permission again.
		pattern := "%permission%"
		if changes.Group == groupLapsed {
			pattern = "%engagement%"
		}
		// Only a live flow enrolls the contacts it gets; a draft would keep them waiting.
		if a, err := e.Q.SendingAutomationByName(ctx, store.SendingAutomationByNameParams{WorkspaceID: rs.campaign.WorkspaceID, Pattern: pattern}); err == nil && a.Status == "live" {
			changes.Automation, changes.AutomationID = flowName(a.Name), a.ID.String()
			d.addRecommendation(keep("rec.moveToFlow", i18n.Args{"exclude": changes.Exclude, "automation": changes.Automation}))
		}
	}
	d.Changes = changes
	d.setOptions(thresholdOptions(tr, sc, held, changes))
	return d
}

func (rs *runState) routesRate() float64 {
	total := 0.0
	for _, r := range rs.routes {
		total += r.rate.PerSecond
	}
	return total
}

// thresholdOptions are the ways to continue after a threshold trip. sc
// names the scope (analyzeThreshold).
func thresholdOptions(tr trip, sc i18n.Args, held int64, ch *ResumeChanges) []option {
	metric := tr.metric
	if metric != "complaint" && metric != "bounce" && metric != "deferral" {
		metric = "none"
	}
	// Resuming as is re-arms the threshold from the current counts (review
	// GS-2): what today's rate would bring in the rest.
	expect := int64(math.Round(float64(held) * tr.value / 100))
	resume := i18n.Args{"held": held, "metric": metric, "ratePct": round2(tr.value), "expect": expect, "more": yesNo(expect > 0)}
	body := keep("opt.resumeScoped", with(sc, "held", held, "metric", metric, "ratePct", resume["ratePct"], "expect", expect, "more", resume["more"]))
	if tr.scope == "campaign" {
		body = keep("opt.resumeAll", resume)
	}
	out := []option{{key: "resume", title: keep("opt.resumeTitle", i18n.Args{"recommended": "no"}), body: body, action: keep("opt.resumeAction", nil)}}
	// A campaign-wide resume takes a group out; a mailbox provider's or a
	// route's also slows down.
	rate := 0.0
	if ch != nil && tr.scope != "campaign" {
		rate = ch.Rate
	}
	if ch != nil && (ch.Exclude > 0 || rate > 0) {
		// What resuming with changes does: exclude a group, slow down, hand
		// the excluded contacts to a flow.
		combo := "rate"
		switch {
		case ch.Exclude > 0 && rate > 0 && ch.Automation != "":
			combo = "all"
		case ch.Exclude > 0 && rate > 0:
			combo = "excludeRate"
		case ch.Exclude > 0 && ch.Automation != "":
			combo = "excludeFlow"
		case ch.Exclude > 0:
			combo = "exclude"
		}
		args := i18n.Args{"combo": combo, "exclude": ch.Exclude, "group": ch.group(), "rest": max(held-ch.Exclude, 0), "changeRate": rate, "automation": ch.Automation}
		out = append(out, option{key: "resume_with_changes", recommended: true, title: keep("opt.changesTitle", nil), body: keep("opt.changes", args),
			action: keep("opt.changesAction", nil)})
	}
	if tr.scope == "campaign" {
		out = append(out, option{key: "cancel", title: keep("opt.cancelCampaign", nil), body: keep("opt.cancelCampaignBody", i18n.Args{"held": held}), action: keep("opt.cancelCampaign", nil)})
	} else {
		out = append(out, option{key: "cancel_share", title: keep("opt.cancelShare", sc), body: keep("opt.cancelShareBody", with(sc, "held", held)),
			action: keep("opt.cancelShare", sc)})
	}
	if len(out) == 2 {
		out[0].recommended = true
		out[0].title = keep("opt.resumeTitle", i18n.Args{"recommended": "yes"})
	}
	return out
}

// thresholdNote is the note under "What tripped": why the threshold
// matters, by metric and mailbox provider (a message key).
func thresholdNote(tr trip) string {
	switch tr.metric {
	case "complaint":
		switch tr.mailbox {
		case "outlook":
			return "note.complaintOutlook"
		case "gmail":
			return "note.complaintGmail"
		case "yahoo":
			return "note.complaintYahoo"
		}
		return "note.complaint"
	case "bounce":
		return "note.bounce"
	case "deferral":
		return "note.deferral"
	}
	return ""
}

// Cohorts an auto-pause can exclude (ResumeChanges.Group).
const (
	groupImported = "imported"
	groupLapsed   = "lapsed"
	groupNever    = "never"
)

// group is the cohort Exclude takes out ("never" by default).
func (c *ResumeChanges) group() string {
	switch {
	case c.Group != "":
		return c.Group
	case c.ImportedOnly:
		return groupImported
	}
	return groupNever
}

// flowName drops a flow name's qualifier: "Re-permission · Mailchimp
// import" is "Re-permission" in a sentence.
func flowName(name string) string {
	if i := strings.Index(name, " · "); i > 0 {
		return name[:i]
	}
	return name
}

// complaintCohort looks for the group of recipients most complaints come
// from and recommends taking it out of the rest of the send: never-engaged
// contacts of one import (with unclear consent, or new to the sending
// domain), contacts who stopped opening (at risk or sunset), or anyone who
// never opened. When no group stands out it says so.
func (e *Engine) complaintCohort(ctx context.Context, rs *runState, tr trip, scope tally, held int64, who string, d *IncidentDetail, changes *ResumeChanges) {
	cid := rs.campaign.ID
	cohorts, _ := e.Q.SendingComplaintCohorts(ctx, store.SendingComplaintCohortsParams{CampaignID: &cid, MailboxProvider: tr.mailbox})
	var imported *store.SendingComplaintCohortsRow
	var importEvents, never, lapsed int64
	for i := range cohorts {
		c := cohorts[i]
		if c.NeverEngaged {
			never += c.N
		}
		if c.Tier == "at_risk" || c.Tier == "sunset" {
			lapsed += c.N
		}
		if strings.HasPrefix(c.Source, "import:") && c.NeverEngaged && (imported == nil || c.Source == imported.Source) {
			importEvents += c.N
			if imported == nil {
				imported = &cohorts[i]
			}
		}
	}
	stands := func(n int64) bool { return n >= 2 && float64(n) >= 0.5*float64(tr.events) }
	// sc names the scope's recipients in the sentences: a mailbox
	// provider's by its key, nothing for a route or the whole campaign.
	sc := i18n.Args{"scope": "campaign", "who": "", "complaints": tr.events, "held": held}
	if who != "" {
		sc["scope"], sc["mailbox"] = "mailbox_provider", tr.mailbox
	}
	// expected is the rate on what is left once the group is out.
	expected := func(size store.SendingGroupSizeRow, events int64) float64 {
		groupSent := size.Total - size.Remaining
		groupDelivered := int64(float64(groupSent) * float64(scope.delivered) / math.Max(float64(scope.sent), 1))
		return round2(math.Max(rate(tr.events-events, max(scope.delivered-groupDelivered, 1)), 0.01))
	}
	size := func(group, source string) store.SendingGroupSizeRow {
		s, _ := e.Q.SendingGroupSize(ctx, store.SendingGroupSizeParams{CampaignID: &cid, MailboxProvider: tr.mailbox, Grp: group, Source: source})
		return s
	}
	switch {
	case imported != nil && stands(importEvents):
		src := strings.TrimPrefix(imported.Source, "import:")
		sz := size(groupImported, imported.Source)
		// The import's day, as "2 Jan" in the workspace's time zone.
		day := "none"
		if imp, err := e.Q.SendingImportOf(ctx, store.SendingImportOfParams{WorkspaceID: rs.campaign.WorkspaceID, Source: src}); err == nil {
			day = imp.CreatedAt.In(rs.loc).Format(time.RFC3339)
		}
		source := src
		if source == "" {
			source = "none"
		}
		args := with(sc, "total", sz.Total, "events", importEvents, "source", source, "sourceName", sourceName(src), "importDay", day)
		lead, body := keep("cause.importedLead", sc), keep("cause.imported", args)
		if imported.Consent == "pending" {
			lead, body = keep("cause.importedPendingLead", sc), keep("cause.importedPending", args)
		}
		d.addCause(lead, body)
		ids, _ := e.Q.SendingCohortContacts(ctx, store.SendingCohortContactsParams{CampaignID: &cid, MailboxProvider: tr.mailbox, Source: imported.Source})
		if len(ids) > 0 {
			prior, err := e.Q.SendingPriorSends(ctx, store.SendingPriorSendsParams{WorkspaceID: rs.campaign.WorkspaceID, ContactIds: derefIDs(ids), CampaignID: &cid})
			if err == nil && prior == 0 {
				d.addCause(keep("cause.firstSendLead", i18n.Args{"domain": domainOf(rs.content.FromEmail)}),
					keep("cause.firstSend", i18n.Args{"source": source, "sourceName": sourceName(src)}))
			}
		}
		changes.Group, changes.Source, changes.ImportedOnly, changes.Exclude = groupImported, imported.Source, true, sz.Remaining
		if sz.Remaining > 0 {
			changes.Expected = expected(sz, importEvents)
			d.addRecommendation(keep("rec.excludeImported", with(sc, "exclude", sz.Remaining, "expectedPct", changes.Expected)))
		}
	case stands(never):
		sz := size(groupNever, "")
		d.addCause(keep("cause.neverLead", sc), keep("cause.never", with(sc, "never", never)))
		changes.Group, changes.Exclude = groupNever, sz.Remaining
		if sz.Remaining > 0 {
			changes.Expected = expected(sz, never)
			d.addRecommendation(keep("rec.excludeNever", with(sc, "exclude", sz.Remaining, "expectedPct", changes.Expected)))
		}
	case stands(lapsed):
		sz := size(groupLapsed, "")
		d.addCause(keep("cause.lapsedLead", sc), keep("cause.lapsed", with(sc, "total", sz.Total, "lapsed", lapsed)))
		changes.Group, changes.Exclude = groupLapsed, sz.Remaining
		if sz.Remaining > 0 {
			changes.Expected = expected(sz, lapsed)
			d.addRecommendation(keep("rec.excludeLapsed", with(sc, "exclude", sz.Remaining, "expectedPct", changes.Expected)))
		}
	default:
		d.addCause(keep("cause.noGroupLead", nil), keep("cause.noGroup", sc))
	}
}

// analyzeOutage explains a held route.
func (e *Engine) analyzeOutage(ctx context.Context, rs *runState, r *runRoute, reason string, remaining int64, now time.Time) IncidentDetail {
	key := routeKey{rs.campaign.ID, r.provider.ID}
	l := e.callLogFor(key)
	e.mu.Lock()
	first, last, failures := l.firstFailure, l.lastFailure, l.failures
	lines := append([]IncidentLogLine(nil), l.lines...)
	e.mu.Unlock()
	if len(lines) == 0 {
		// The provider health job found the outage, or another process
		// sent: what the provider answered comes from its health record.
		lines, first, last = healthLines(r.provider, now)
	}
	d := IncidentDetail{Reason: reason, Attempts: failures, Remaining: remaining, Log: lines}
	if !first.IsZero() {
		f, t := first, last
		d.From, d.To = &f, &t
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
	p := i18n.Args{"name": name}
	proxy := strings.Contains(joined, "502") || strings.Contains(joined, "503")
	switch {
	case reason == "auth":
		d.ErrorKind = "rejected credentials"
		d.addCause(keep("cause.authLead", p), keep("cause.auth", nil))
	case reason == "identity":
		// The provider refuses the sender or its domain: nothing sends
		// until it is verified there (review PR-7).
		d.ErrorKind = "unverified sender"
		domain := domainOf(rs.content.FromEmail)
		d.addCause(keep("cause.identityLead", i18n.Args{"name": name, "domain": firstNonEmpty(domain, "none")}),
			keep("cause.identity", i18n.Args{"name": name, "sender": firstNonEmpty(rs.content.FromEmail, "none")}))
	case reason == "quota":
		d.ErrorKind = "quota"
		args := i18n.Args{"name": name, "known": "no", "used": int64(0), "max": int64(0)}
		if h := state.HealthOf(r.provider); h.Quota != nil && h.Quota.Max > 0 {
			args["known"], args["used"], args["max"] = "yes", h.Quota.Used, h.Quota.Max
		}
		d.addCause(keep("cause.quotaLead", p), keep("cause.quotaUsedUp", args))
	case reason == "api_errors":
		d.ErrorKind = "API errors"
		d.addCause(keep("cause.apiErrorsLead", p), keep("cause.apiErrors", i18n.Args{"name": name, "errorPct": d.ErrorRate, "limitPct": rs.thresholds.APIErrors}))
	case reason == "throttled":
		d.ErrorKind = "throttling errors"
		d.addCause(keep("cause.throttlingLead", p), keep("cause.throttledLong", p))
	case strings.Contains(joined, "econnrefused") || strings.Contains(joined, "connection refused") || strings.Contains(joined, "timeout") || strings.Contains(joined, "dial "):
		d.ErrorKind = "connection errors"
		if isSelfHosted(r.provider) {
			d.addCause(keep("cause.crashedLead", p), keep("cause.crashed", i18n.Args{"host": d.Host}))
		} else {
			d.addCause(keep("cause.networkLead", p), keep("cause.network", p))
		}
		if proxy {
			d.addCause(keep("cause.proxyLead", nil), keep("cause.proxy", p))
		}
	case strings.Contains(joined, "429") || strings.Contains(joined, "throttl"):
		d.ErrorKind = "throttling errors"
		d.addCause(keep("cause.throttlingLead", p), keep("cause.throttling", nil))
	default:
		d.ErrorKind = "API errors"
		if reason == "provider_down" {
			d.addCause(keep("cause.downLead", p), keep("cause.down", nil))
		} else {
			d.addCause(keep("cause.outageLead", p), keep("cause.outage", p))
		}
		if proxy {
			d.addCause(keep("cause.proxyLead", nil), keep("cause.proxy", p))
		}
	}
	if r.rate.WarmingUp && r.rate.Cap > 0 {
		// Today's count already has what this send handed to the provider.
		d.addCause(i18n.Text{}, keep("cause.warmupCap", i18n.Args{"sentToday": r.rate.SentToday, "cap": r.rate.Cap, "perSecondRate": r.rate.PerSecond}))
		d.setWarmupNote(keep("note.warmupShift", i18n.Args{"who": short}))
	} else if h := state.HealthOf(r.provider); h.Quota != nil && h.Quota.Max > 0 && reason != "quota" {
		d.addCause(i18n.Text{}, keep("cause.quota", i18n.Args{"used": h.Quota.Used, "max": h.Quota.Max}))
	}
	set, _ := routing.Load(ctx, e.Q, rs.campaign.WorkspaceID)
	target := e.moveTarget(ctx, rs, r, set)
	if target != nil {
		var warming bool
		d.Move, warming = e.moveEstimate(ctx, rs, *target, remaining, now)
		risk := "low"
		if warming {
			risk = "high"
		}
		d.setRisk(risk, keep("risk."+risk, i18n.Args{"target": d.Move.TargetShort}))
		sender := senderVerification(ctx, e.Q, rs.campaign.WorkspaceID, rs.campaign.SenderID)
		d.setMoveChecks(moveChecks(rs, e.vetTarget(ctx, e.Q, sender, *target, remaining, rs.loc), d.Move, remaining))
	}
	d.setOptions(outageOptions(short, remaining, d.SentOnRoute, d.Move, e.seedInbox(ctx, rs.campaign.WorkspaceID) != ""))
	return d
}

// healthLines are a provider's recent failed calls as an outage's log
// (oldest first, within the last 15 minutes, at most 6), with the first
// and last one's time.
func healthLines(p store.Provider, now time.Time) (lines []IncidentLogLine, first, last time.Time) {
	h := state.HealthOf(p)
	for i := len(h.RecentErrors) - 1; i >= 0; i-- {
		rec := h.RecentErrors[i]
		if now.Sub(rec.At) > 15*time.Minute || rec.At.After(now) {
			continue
		}
		text := callText(&providers.CallError{Op: rec.Op, Status: rec.Status, Code: rec.Code, Message: rec.Message})
		if n := len(lines); n > 0 && lines[n-1].Text == text && rec.At.Sub(lines[n-1].At) < time.Minute {
			continue
		}
		lines = append(lines, IncidentLogLine{At: rec.At, Text: text, Bad: true})
		if first.IsZero() {
			first = rec.At
		}
		last = rec.At
	}
	if len(lines) > 6 {
		lines = lines[len(lines)-6:]
	}
	return lines, first, last
}

// moveEstimate is what moving n recipients to target costs, and whether
// the target is warming up (the reputation risk).
func (e *Engine) moveEstimate(ctx context.Context, rs *runState, target store.Provider, n int64, now time.Time) (*MoveEstimate, bool) {
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
	return m, tr.WarmingUp
}

// moveChecks are the checks a move's target passed, as the outage card
// lists them (the target passed every check that could fail: it was
// chosen by moveTarget), and what happens to the recipients the A/B test
// or the canary holds.
func moveChecks(rs *runState, v state.Vetting, m *MoveEstimate, n int64) []moveCheck {
	var out []moveCheck
	domain := domainOf(rs.content.FromEmail)
	for _, c := range v.Checks {
		args := i18n.Args{"target": m.TargetShort, "domain": firstNonEmpty(domain, "none")}
		switch {
		case c.Key == state.VetDomain && c.Result == state.VetPass:
			out = append(out, moveCheck{c.Key, "pass", keep("check.domain", args)})
		case c.Key == state.VetDomain:
			out = append(out, moveCheck{c.Key, "unknown", keep("check.domainUnknown", args)})
		case c.Key == state.VetQuota && c.Result == state.VetPass && c.Left != nil && m.QuotaMax != nil:
			args["free"], args["max"], args["kind"] = max(*c.Left-n, 0), *m.QuotaMax, c.Detail
			out = append(out, moveCheck{c.Key, "pass", keep("check.quota", args)})
		case c.Key == state.VetQuota && c.Result == state.VetUnknown:
			out = append(out, moveCheck{c.Key, "unknown", keep("check.quotaUnknown", args)})
		case c.Key == state.VetWarm && c.Result == state.VetPass:
			out = append(out, moveCheck{c.Key, "pass", keep("check.warm", args)})
		case c.Key == state.VetWarm:
			out = append(out, moveCheck{c.Key, "warn", keep("check.warming", args)})
		}
	}
	switch {
	case rs.ab.Phase == "testing":
		out = append(out, moveCheck{"holdouts", "pass", keep("check.abHoldouts", nil)})
	case rs.canaryHolds():
		out = append(out, moveCheck{"holdouts", "pass", keep("check.canaryHoldouts", nil)})
	}
	return out
}

type moveCheck struct {
	key, result string
	text        i18n.Text
}

// outageOptions are the ways to continue after an outage held a share;
// seed says Retry's test messages go to a seed inbox (outages.go).
func outageOptions(short string, remaining, sent int64, move *MoveEstimate, seed bool) []option {
	p := i18n.Args{"who": short, "remaining": remaining, "seed": yesNo(seed)}
	out := []option{{key: "retry", title: keep("opt.retryTitle", with(p, "recommended", "no")), body: keep("opt.retryBody", p), action: keep("opt.retry", p)}}
	if move != nil {
		mv := with(p, "target", move.TargetShort, "sent", sent)
		out = append(out, option{key: "move", recommended: true, title: keep("opt.moveTitle", mv), body: keep("opt.moveBody", mv), action: keep("opt.move", mv)})
	} else {
		out[0].recommended = true
		out[0].title = keep("opt.retryTitle", with(p, "recommended", "yes"))
	}
	out = append(out, option{key: "cancel_share", title: keep("opt.cancelShare", p), body: keep("opt.cancelShareLater", p), action: keep("opt.cancelShare", p)})
	return out
}

// outageArgs are an outage's headline for the messages: the provider,
// whether it rejected the credentials, the kind of errors and for how long.
func outageArgs(d IncidentDetail, name string) i18n.Args {
	mins := 3
	if d.From != nil && d.To != nil {
		mins = max(int(math.Round(d.To.Sub(*d.From).Minutes())), 1)
	}
	kind := map[string]string{"connection errors": "connection", "throttling errors": "throttling"}[d.ErrorKind]
	if kind == "" {
		kind = "api"
	}
	reason := "errors"
	switch d.Reason {
	case "auth", "identity", "quota", "throttled", "api_errors":
		reason = d.Reason
	}
	return i18n.Args{"provider": name, "reason": reason, "errorKind": kind, "mins": mins, "errorPct": d.ErrorRate}
}

func outageHeadline(d IncidentDetail, name string) string {
	if d.Retried {
		return name + " is still down after 15 minutes of retrying"
	}
	switch d.Reason {
	case "auth":
		return name + " rejected the credentials"
	case "identity":
		return name + " refused the sender: it isn't verified there"
	case "quota":
		return name + "'s sending quota is used up"
	case "api_errors":
		return fmt.Sprintf("%s failed %s of its calls over 3 minutes", name, fmtRate(d.ErrorRate))
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

// durationArgs puts a send's duration the way durationText writes it into
// args: under a minute, minutes, or hours and minutes.
func durationArgs(args i18n.Args, d time.Duration) {
	m := int(math.Round(d.Minutes()))
	switch {
	case m < 1:
		args["took"] = "under"
	case m < 120:
		args["took"], args["mins"] = "minutes", m
	default:
		args["took"], args["hours"], args["rest"] = "hours", m/60, m%60
	}
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
