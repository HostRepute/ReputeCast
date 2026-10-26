package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailbox"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/store"
)

// tally sums counters for one scope.
type tally struct {
	sent, delivered, deferred, hard, soft, complained, rejected, human, machine, clicked, unsubscribed int64
	// reported is what was delivered at mailbox providers that report
	// complaints: the base of a complaint rate (review F-2, GS-6).
	reported int64
}

// complaintRate is the complaint rate where complaints are reported.
func (t tally) complaintRate() float64 { return rate(t.complained, t.reported) }

func (t tally) bounced() int64 { return t.hard + t.soft }

func (t *tally) add(r store.SendingCountersRow) {
	t.sent += r.Sent
	t.delivered += r.Delivered
	t.deferred += r.Deferred
	t.hard += r.HardBounced
	t.soft += r.SoftBounced
	t.complained += r.Complained
	t.rejected += r.Rejected
	t.human += r.OpenedHuman
	t.machine += r.OpenedMachine
	t.clicked += r.Clicked
	t.unsubscribed += r.Unsubscribed
	if mailbox.ReportsComplaints(r.MailboxProvider) {
		t.reported += r.Delivered
	}
}

// tallies are a campaign's counters by scope.
type tallies struct {
	total    tally
	mailbox  map[string]*tally
	provider map[uuid.UUID]*tally
	variant  map[string]*tally
}

func (e *Engine) tallies(ctx context.Context, campaign uuid.UUID) (tallies, error) {
	rows, err := e.Q.SendingCounters(ctx, campaign)
	if err != nil {
		return tallies{}, err
	}
	return tallyRows(rows), nil
}

func tallyRows(rows []store.SendingCountersRow) tallies {
	t := tallies{mailbox: map[string]*tally{}, provider: map[uuid.UUID]*tally{}, variant: map[string]*tally{}}
	for _, r := range rows {
		t.total.add(r)
		m := t.mailbox[r.MailboxProvider]
		if m == nil {
			m = &tally{}
			t.mailbox[r.MailboxProvider] = m
		}
		m.add(r)
		p := t.provider[r.ProviderID]
		if p == nil {
			p = &tally{}
			t.provider[r.ProviderID] = p
		}
		p.add(r)
		v := t.variant[r.Variant]
		if v == nil {
			v = &tally{}
			t.variant[r.Variant] = v
		}
		v.add(r)
	}
	return t
}

func rate(n, d int64) float64 {
	if d <= 0 {
		return 0
	}
	return float64(n) / float64(d) * 100
}

// guard checks thresholds, outages, the A/B test and completion. It returns
// how long nothing can be sent, or errStopped when the loop should end.
func (e *Engine) guard(ctx context.Context, rs *runState, now time.Time) (time.Duration, error) {
	if err := e.settleStale(ctx, rs, now); err != nil {
		return 0, err
	}
	if now.Sub(rs.queueAt) >= time.Second {
		if err := e.refreshQueue(ctx, rs, now); err != nil {
			return 0, err
		}
	}
	ts, err := e.tallies(ctx, rs.campaign.ID)
	if err != nil {
		return 0, err
	}
	if stop, err := e.checkThresholds(ctx, rs, ts, now); err != nil || stop {
		if err == nil {
			err = errStopped
		}
		return 0, err
	}
	for _, pid := range rs.order {
		if err := e.checkOutage(ctx, rs, rs.routes[pid], now); err != nil {
			return 0, err
		}
	}
	if err := e.checkAB(ctx, rs, ts, now); err != nil {
		return 0, err
	}
	var queued, held, waiting, blocked, doubt, probing int64
	var next *time.Time
	for k, c := range rs.queue {
		queued += c.queued
		held += c.held
		waiting += c.waiting
		doubt += c.doubt
		r := rs.routes[k.provider]
		h, onHold := rs.holds[k.mailbox]
		if r == nil || (onHold && !h.Resumed) || !r.open() {
			blocked += c.queued
			if r != nil && (!onHold || h.Resumed) && r.retrying() {
				probing += c.queued
			}
		}
		if c.nextAt != nil && (next == nil || c.nextAt.Before(*next)) {
			t := *c.nextAt
			next = &t
		}
	}
	switch {
	case queued == 0 && held == 0 && doubt == 0:
		return 0, e.finish(ctx, rs, ts, now)
	case queued == 0 && held == 0:
		// Only messages in doubt are left: the send waits for someone to
		// send them again or leave them out (doubt.go).
		return 0, e.waitForDecision(ctx, rs, now)
	case held == 0 && queued > 0 && queued == blocked && probing > 0:
		// Only retried shares are left: the loop stays to send their test
		// messages (outages.go).
		return rs.untilProbe(now), nil
	case held == 0 && queued > 0 && queued == blocked:
		// Only held shares are left: nothing sends until someone decides.
		return 0, e.waitForDecision(ctx, rs, now)
	case queued-blocked > 0 && queued-blocked == waiting && next != nil:
		return max(next.Sub(now), time.Second), nil
	case queued-blocked <= 0 && held > 0 && rs.ab.DecideAt != nil:
		return min(max(rs.ab.DecideAt.Sub(now), time.Second), rs.untilProbe(now)), nil
	case queued-blocked <= 0 && held > 0:
		return min(30*time.Second, rs.untilProbe(now)), nil
	}
	return 0, nil
}

// retrying reports whether a route is a retried share sending test
// messages (outages.go).
func (r *runRoute) retrying() bool { return r.status == "held" && r.detail.Decision == "retry" }

// untilProbe is how long until a retried share's next test message (a
// minute when none is retried).
func (rs *runState) untilProbe(now time.Time) time.Duration {
	wait := retryEvery
	for _, r := range rs.routes {
		if !r.retrying() {
			continue
		}
		if r.detail.NextProbe == nil || !now.Before(*r.detail.NextProbe) {
			return time.Second
		}
		wait = min(wait, r.detail.NextProbe.Sub(now))
	}
	return max(wait, time.Second)
}

// trip is one threshold that tripped.
type trip struct {
	scope    string // campaign, mailbox_provider, route
	mailbox  string
	provider uuid.UUID
	metric   string // bounce, complaint, deferral
	value    float64
	limit    float64
	events   int64
	base     int64
}

// checkThresholds pauses a mailbox provider, a route or the whole campaign
// when a rate goes over its limit. With the "route" scope (the default) the
// narrowest scope that explains a trip is paused: a mailbox provider, then a
// sending provider; the campaign-wide rates are checked on what isn't
// already held, so one mailbox provider's complaints never pause Gmail. With
// the "campaign" scope any trip pauses everything. It returns true when the
// whole campaign paused.
func (e *Engine) checkThresholds(ctx context.Context, rs *runState, ts tallies, now time.Time) (bool, error) {
	th := rs.thresholds
	var trips []trip
	for _, mbp := range sortedKeys(ts.mailbox) {
		h, onHold := rs.holds[mbp]
		if onHold && !h.Resumed {
			continue
		}
		// A scope a person resumed is judged on what happened since, with
		// the evidence a fresh one needs: old events never pause it again
		// (review GS-2).
		m := rs.guard.judged(rearmMailbox(mbp), *ts.mailbox[mbp])
		if _, rearmed := rs.guard.Rearm[rearmMailbox(mbp)]; !rearmed && onHold && h.Rearm != nil {
			// A resume recorded before the guard kept its counts.
			m = m.since(counts{Complained: h.Rearm["complaint"], Hard: h.Rearm["bounce"], Deferred: h.Rearm["deferral"]})
		}
		switch {
		case m.reported >= minSampleScoped && over(m.complained, m.reported, th.MailboxComplaint, minEvents):
			trips = append(trips, trip{scope: "mailbox_provider", mailbox: mbp, metric: "complaint", value: m.complaintRate(), limit: th.MailboxComplaint, events: m.complained, base: m.reported})
		case m.sent >= minSampleScoped && over(m.bounced(), m.sent, th.Bounce, minEvents):
			trips = append(trips, trip{scope: "mailbox_provider", mailbox: mbp, metric: "bounce", value: rate(m.bounced(), m.sent), limit: th.Bounce, events: m.bounced(), base: m.sent})
		case m.sent >= minSampleScoped && over(m.deferred, m.sent, th.Deferral, minDeferrals):
			trips = append(trips, trip{scope: "mailbox_provider", mailbox: mbp, metric: "deferral", value: rate(m.deferred, m.sent), limit: th.Deferral, events: m.deferred, base: m.sent})
		}
		if len(trips) > 0 {
			break
		}
	}
	if len(trips) == 0 {
		// A sending provider is held for its bounces (its IP or domain
		// reputation, addresses it rejects). Complaints come from a mailbox
		// provider's readers, whoever delivered the mail: they are watched
		// per mailbox provider and campaign-wide, never per route, which
		// would hold a whole provider for one mailbox provider's readers.
		for _, pid := range rs.order {
			r := rs.routes[pid]
			pt := ts.provider[pid]
			if pt == nil || !r.open() {
				continue
			}
			p := rs.guard.judged(rearmRoute(pid), *pt)
			if p.sent >= minSampleScoped && over(p.bounced(), p.sent, th.Bounce, minEvents) {
				trips = append(trips, trip{scope: "route", provider: pid, metric: "bounce", value: rate(p.bounced(), p.sent), limit: th.Bounce, events: p.bounced(), base: p.sent})
				break
			}
		}
	}
	if len(trips) == 0 {
		// Campaign-wide, on the mailbox providers and routes still sending.
		t := tally{}
		for mbp, m := range ts.mailbox {
			if h, onHold := rs.holds[mbp]; onHold && !h.Resumed {
				continue
			}
			t.add(store.SendingCountersRow{MailboxProvider: mbp, Sent: m.sent, Delivered: m.delivered, Deferred: m.deferred, HardBounced: m.hard, SoftBounced: m.soft, Complained: m.complained})
		}
		for pid, p := range ts.provider {
			if r := rs.routes[pid]; r != nil && (r.status == "held" || r.status == "paused") {
				t.sent -= p.sent
				t.delivered -= p.delivered
				t.hard -= p.hard
				t.soft -= p.soft
				t.complained -= p.complained
				t.reported -= p.reported
			}
		}
		t = rs.guard.judged(rearmCampaign, t)
		if t.sent >= minSampleCampaign && over(t.bounced(), t.sent, th.Bounce, minEvents) {
			trips = append(trips, trip{scope: "campaign", metric: "bounce", value: rate(t.bounced(), t.sent), limit: th.Bounce, events: t.bounced(), base: t.sent})
		}
		// Complaints count only where mailbox providers report them: Gmail's
		// and iCloud's delivered mail would dilute the rate (review F-2).
		if t.reported >= minSampleCampaign && over(t.complained, t.reported, th.Complaint, minEvents) {
			trips = append(trips, trip{scope: "campaign", metric: "complaint", value: t.complaintRate(), limit: th.Complaint, events: t.complained, base: t.reported})
		}
	}
	if len(trips) == 0 {
		return false, nil
	}
	tr := trips[0]
	if th.Scope == "campaign" {
		tr.scope = "campaign"
	}
	return tr.scope == "campaign", e.autoPause(ctx, rs, ts, tr, now)
}

// autoPause holds the tripped scope, records the incident with what
// tripped, likely causes and recommended actions, and tells the team.
func (e *Engine) autoPause(ctx context.Context, rs *runState, ts tallies, tr trip, now time.Time) error {
	c := rs.campaign
	var held int64
	for k, cell := range rs.queue {
		switch {
		case tr.scope == "campaign",
			tr.scope == "mailbox_provider" && k.mailbox == tr.mailbox,
			tr.scope == "route" && k.provider == tr.provider:
			held += cell.queued + cell.held
		}
	}
	detail := e.analyzeThreshold(ctx, rs, ts, tr, held, now)
	db_detail := detail.stored()
	id := newID()
	var pid *uuid.UUID
	if tr.scope == "route" {
		p := tr.provider
		pid = &p
	}
	value, limit := round4(tr.value), round4(tr.limit)
	scopeName := "the campaign"
	switch tr.scope {
	case "mailbox_provider":
		scopeName = mailboxName(tr.mailbox)
	case "route":
		scopeName = rs.routes[tr.provider].shortName()
	}
	headline := thresholdHeadline(tr, scopeName)
	// The headline's words for the notification and the incident title,
	// kept for each reader's language.
	words := headlineArgs(tr, scopeName)
	words["name"], words["held"] = quoted(c.Name), held
	err := db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.SendingInsertIncident(ctx, store.SendingInsertIncidentParams{ID: id, CampaignID: c.ID, Kind: "threshold", Scope: tr.scope, ProviderID: pid,
			MailboxProvider: tr.mailbox, Metric: tr.metric, Value: &value, Threshold: &limit, Held: held, Detail: db_detail, TrippedAt: now}); err != nil {
			return err
		}
		switch tr.scope {
		case "campaign":
			if err := q.SendingSetCampaignStatus(ctx, store.SendingSetCampaignStatusParams{ID: c.ID, Status: "paused", At: now}); err != nil {
				return err
			}
		case "mailbox_provider":
			s, err := q.SendingGetSendForUpdate(ctx, c.ID)
			if err != nil {
				return err
			}
			holds := parseHolds(s.Holds)
			holds[tr.mailbox] = hold{Incident: id.String(), Since: now}
			b, _ := json.Marshal(holds)
			if err := q.SendingSetHolds(ctx, store.SendingSetHoldsParams{CampaignID: c.ID, Holds: b, At: now}); err != nil {
				return err
			}
			rs.holds = holds
		case "route":
			d, _ := json.Marshal(routeDetail{Reason: "threshold", Incident: id.String()})
			if err := q.SendingSetRouteStatus(ctx, store.SendingSetRouteStatusParams{CampaignID: c.ID, ProviderID: tr.provider, Status: "paused", StatusDetail: string(d)}); err != nil {
				return err
			}
			rs.routes[tr.provider].status = "paused"
		}
		cid := c.ID
		title := keep("incident.autoPaused", words)
		if err := q.ProvInsertIncident(ctx, store.ProvInsertIncidentParams{ID: newID(), WorkspaceID: c.WorkspaceID, OccurredAt: now, Kind: "auto_pause", Severity: "bad",
			Title: title.In(i18n.Default), CampaignID: &cid, ProviderID: pid, Texts: titleTexts(title)}); err != nil {
			return err
		}
		if err := notify.ToMembersWith(ctx, tx, c.WorkspaceID, authz.CampaignsSend, notify.Notification{
			Kind: notify.KindSendPaused, Severity: "bad", TitleText: keep("notify.autoPausedTitle", words), BodyText: keep("notify.autoPausedBody", words),
			Link: liveLink(c.WorkspaceSlug, c.ID), Data: map[string]any{"campaignId": c.ID.String(), "incidentId": id.String()},
		}); err != nil {
			return err
		}
		data := campaignData(c.ID, c.Name, "paused_by", "auto_pause", "reason", headline, "scope", tr.scope, "metric", tr.metric, "value", value, "limit", limit, "held", held)
		switch tr.scope {
		case "mailbox_provider":
			data["mailbox_provider"] = tr.mailbox
		case "route":
			data["provider_id"], data["provider"] = tr.provider.String(), rs.routes[tr.provider].shortName()
		}
		if err := emit(ctx, tx, c.WorkspaceID, "campaign.paused", data); err != nil {
			return err
		}
		// As the audit log shows it: "complaint rate 0.18% over limit 0.10%
		// · 7 of 3,980 · 2,522 recipients held" on "October Product Update ·
		// Outlook route".
		target := c.Name
		if tr.scope != "campaign" {
			target += " · " + scopeName + " route"
		}
		summary := fmt.Sprintf("%s %s over limit %s · %s of %s · %s recipients held", metricName(tr.metric), fmtRate(tr.value), fmtRate(tr.limit),
			countText(tr.events), countText(tr.base), countText(held))
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.auto_paused", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Reason: headline, Actor: systemActor, Metadata: map[string]any{"scope": tr.scope, "mailbox_provider": tr.mailbox, "metric": tr.metric, "value": value, "limit": limit, "held": held,
				"summary": summary, "target": target, "source": "guard", "via": "threshold"}})
	})
	return err
}

// headlineArgs are a threshold trip's headline for the messages: whose
// rate, which one, its value and its limit.
func headlineArgs(tr trip, scopeName string) i18n.Args {
	args := i18n.Args{"scope": tr.scope, "who": scopeName, "metric": tr.metric}
	if tr.scope == "campaign" {
		args["who"] = ""
	}
	pctArgs(args, "value", tr.value)
	pctArgs(args, "limit", tr.limit)
	return args
}

// pctArgs puts a percentage the way fmtRate writes it ("0.18%", "2.4%")
// into args: name as a fraction for the messages' percent format, and
// name+"2" whether it takes two decimals.
func pctArgs(args i18n.Args, name string, v float64) {
	d, two := 1, "no"
	if v < 1 {
		d, two = 2, "yes"
	}
	r, _ := strconv.ParseFloat(strconv.FormatFloat(v, 'f', d, 64), 64)
	args[name], args[name+"2"] = r/100, two
}

// titleTexts is an incidents.texts value keeping a title's message.
func titleTexts(t i18n.Text) []byte {
	b, _ := json.Marshal(map[string]i18n.Text{"title": t})
	return b
}

func liveLink(slug string, id uuid.UUID) string {
	return "/w/" + slug + "/campaigns/" + id.String() + "/live"
}

// metricName names a threshold's metric ("complaint rate").
func metricName(metric string) string {
	return map[string]string{"bounce": "bounce rate", "complaint": "complaint rate", "deferral": "deferral rate", "api_errors": "API error rate"}[metric]
}

func thresholdHeadline(tr trip, scopeName string) string {
	metric := metricName(tr.metric)
	subject := scopeName + " " + metric
	if tr.scope == "campaign" {
		subject = strings.ToUpper(metric[:1]) + metric[1:]
	}
	return fmt.Sprintf("%s reached %s (limit %s)", subject, fmtRate(tr.value), fmtRate(tr.limit))
}

// open reports whether the route still sends (not held, paused, moved,
// cancelled or done).
func (r *runRoute) open() bool { return r.status == "pending" || r.status == "sending" }

// shortName is the provider's badge label ("SES", "Postal").
func (r *runRoute) shortName() string {
	if r == nil {
		return "the provider"
	}
	return state.Short(r.provider.Name, r.provider.Adapter)
}

// checkOutage holds a route whose provider has been failing, or that the
// provider health job marked down. Broadcast failover asks a person by
// default and moves the share automatically only to a warmed-up provider
// when the stream's policy allows it.
func (e *Engine) checkOutage(ctx context.Context, rs *runState, r *runRoute, now time.Time) error {
	if r == nil {
		return nil
	}
	key := routeKey{rs.campaign.ID, r.provider.ID}
	l := e.callLogFor(key)
	e.mu.Lock()
	first, failures, lastOK := l.firstFailure, l.failures, l.lastSuccess
	throttledSince, throttles := l.firstThrottle, l.throttles
	e.mu.Unlock()
	if (!first.IsZero() && !rs.started.IsZero() && first.Before(rs.started)) || (!throttledSince.IsZero() && !rs.started.IsZero() && throttledSince.Before(rs.started)) {
		// Failures of an earlier send of this campaign (started again).
		e.resetCalls(key)
		first, failures, lastOK, throttledSince, throttles = time.Time{}, 0, time.Time{}, time.Time{}, 0
	}
	if r.status == "held" {
		return e.probeHeld(ctx, rs, r, now)
	}
	if r.status != "pending" && r.status != "sending" {
		return nil
	}
	// The provider kept throttling the account for three minutes, or the
	// share of its failed calls passed the send's limit (review GS-4).
	if !throttledSince.IsZero() && now.Sub(throttledSince) >= outageAfter && throttles >= outageMinCalls && (lastOK.IsZero() || lastOK.Before(throttledSince)) {
		return e.outage(ctx, rs, r, "throttled", now, false)
	}
	// The API error threshold judges a provider that fails part of its
	// calls. One that fails every call in a row is an outage, judged below
	// after three minutes; one that only throttles is judged above.
	onlyThrottled := !throttledSince.IsZero() && first.IsZero()
	streak := !first.IsZero() && failures >= outageMinCalls && (lastOK.IsZero() || lastOK.Before(first))
	if !onlyThrottled && !streak && now.Sub(r.apiCheckedAt) >= settleEvery {
		r.apiCheckedAt = now
		trip, _, err := e.apiErrorTrip(ctx, rs, r, now)
		if err != nil {
			return err
		}
		if trip {
			return e.outage(ctx, rs, r, "api_errors", now, false)
		}
	}
	down := r.provider.Status == state.Down
	failing := !first.IsZero() && now.Sub(first) >= outageAfter && failures >= outageMinCalls && (lastOK.IsZero() || lastOK.Before(first))
	if !down && !failing {
		return nil
	}
	reason := "outage"
	if down && !failing {
		reason = "provider_down"
		if h := state.HealthOf(r.provider); h.Down != nil && h.Down.Reason == "auth" {
			reason = "auth"
		}
	}
	return e.outage(ctx, rs, r, reason, now, false)
}

func (e *Engine) setRouteDetail(ctx context.Context, rs *runState, r *runRoute, d routeDetail, status string) error {
	b, _ := json.Marshal(d)
	s := string(b)
	if s == "{}" {
		s = ""
	}
	if err := e.Q.SendingSetRouteStatus(ctx, store.SendingSetRouteStatusParams{CampaignID: rs.campaign.ID, ProviderID: r.provider.ID, Status: status, StatusDetail: s}); err != nil {
		return err
	}
	r.status, r.detail = status, d
	return nil
}

// outage handles a failing route: an automatic move to a warmed-up
// provider when the Broadcast policy allows it, otherwise a hold that waits
// for a person. again is a retried share still failing after 15 minutes
// (outages.go): the person is asked again.
func (e *Engine) outage(ctx context.Context, rs *runState, r *runRoute, reason string, now time.Time, again bool) error {
	set, err := routing.Load(ctx, e.Q, rs.campaign.WorkspaceID)
	if err != nil {
		return err
	}
	policy := set.Policy(providers.StreamBroadcast)
	target := e.moveTarget(ctx, rs, r, set)
	if target != nil && policy != routing.PauseAndAsk && reason != "auth" {
		return e.autoMove(ctx, rs, r, target, reason, policy, now)
	}
	return e.holdOutage(ctx, rs, r, reason, again, now)
}

// moveTarget is where a held share would go: the stream's fallback, else
// the largest share of the Broadcast split, among the providers whose
// vetting passes for what is left of the share (never one warming up).
func (e *Engine) moveTarget(ctx context.Context, rs *runState, r *runRoute, set routing.Set) *store.Provider {
	provs, err := e.Q.SendingWorkspaceProviders(ctx, rs.campaign.WorkspaceID)
	if err != nil {
		return nil
	}
	cid := rs.campaign.ID
	left, err := e.Q.SendingRemainingOn(ctx, store.SendingRemainingOnParams{CampaignID: &cid, ProviderID: &r.provider.ID})
	if err != nil {
		return nil
	}
	sender := senderVerification(ctx, e.Q, rs.campaign.WorkspaceID, rs.campaign.SenderID)
	byID := map[uuid.UUID]store.Provider{}
	for _, p := range provs {
		byID[p.ID] = p
	}
	ok := func(id uuid.UUID) bool {
		p, found := byID[id]
		return found && id != r.provider.ID && e.vetTarget(ctx, e.Q, sender, p, left, rs.loc).Automatic(false)
	}
	def, found := set.Default(providers.StreamBroadcast)
	if found && def.Fallback != nil && ok(*def.Fallback) {
		p := byID[*def.Fallback]
		return &p
	}
	best, bestW := uuid.Nil, -1
	if found {
		for _, s := range def.Splits {
			if ok(s.ProviderID) && s.Weight > bestW {
				best, bestW = s.ProviderID, s.Weight
			}
		}
	}
	for _, pid := range rs.order {
		if bestW < 0 && ok(pid) {
			best, bestW = pid, 0
		}
	}
	if bestW < 0 {
		return nil
	}
	p := byID[best]
	return &p
}

// holdRoute puts a route on hold and asks a person what to do.
func (e *Engine) holdRoute(ctx context.Context, rs *runState, r *runRoute, reason string, now time.Time) error {
	return e.holdOutage(ctx, rs, r, reason, false, now)
}

// holdOutage puts a route on hold with the engine's analysis and options and
// asks a person what to do; again says the share was retried for 15
// minutes and still fails. The campaign's row is locked and the route read
// again first, so a person's decision taken meanwhile (a move, a cancel, a
// retry) is never overwritten.
func (e *Engine) holdOutage(ctx context.Context, rs *runState, r *runRoute, reason string, again bool, now time.Time) error {
	c := rs.campaign
	cid := c.ID
	remaining, err := e.Q.SendingRemainingOn(ctx, store.SendingRemainingOnParams{CampaignID: &cid, ProviderID: &r.provider.ID})
	if err != nil {
		return err
	}
	detail := e.analyzeOutage(ctx, rs, r, reason, remaining, now)
	if again {
		detail.retried(r.shortName())
	}
	b := detail.stored()
	id := newID()
	short := r.shortName()
	headline := outageHeadline(detail, r.provider.Name)
	words := outageArgs(detail, r.provider.Name)
	words["name"], words["who"], words["remaining"] = quoted(c.Name), short, remaining
	skipped := false
	err = db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.SendingCampaignForUpdate(ctx, c.ID)
		if err != nil {
			return err
		}
		rt, err := routeOf(ctx, q, c.ID, r.provider.ID)
		if err != nil {
			return err
		}
		if cur.Status != "sending" && cur.Status != "paused" || rt == nil || !holdable(*rt, again) {
			skipped = true
			return nil
		}
		// The provider health job may have held it already: one open incident per route.
		open, err := q.SendingOpenIncidents(ctx, c.ID)
		if err != nil {
			return err
		}
		for _, in := range open {
			if in.Kind == "provider_outage" && in.ProviderID != nil && *in.ProviderID == r.provider.ID {
				id = in.ID
				if err := q.SendingUpdateIncidentDetail(ctx, store.SendingUpdateIncidentDetailParams{ID: in.ID, Detail: b, Held: remaining}); err != nil {
					return err
				}
				return e.markHeld(ctx, q, rs, r, reason, id)
			}
		}
		pid := r.provider.ID
		apiErr := detail.ErrorRate
		limit := rs.thresholds.APIErrors
		if err := q.SendingInsertIncident(ctx, store.SendingInsertIncidentParams{ID: id, CampaignID: c.ID, Kind: "provider_outage", Scope: "route", ProviderID: &pid,
			Metric: "api_errors", Value: &apiErr, Threshold: &limit, Held: remaining, Detail: b, TrippedAt: now}); err != nil {
			return err
		}
		if err := e.markHeld(ctx, q, rs, r, reason, id); err != nil {
			return err
		}
		title := keep("incident.broadcastHeld", words)
		if err := q.ProvInsertIncident(ctx, store.ProvInsertIncidentParams{ID: newID(), WorkspaceID: c.WorkspaceID, OccurredAt: now, Kind: "broadcast_held", Severity: "warn",
			Title: title.In(i18n.Default), CampaignID: &cid, ProviderID: &pid, Texts: titleTexts(title)}); err != nil {
			return err
		}
		titleKey, bodyKey := "notify.decisionTitle", "notify.decisionBody"
		if again {
			titleKey, bodyKey = "notify.askAgainTitle", "notify.askAgainBody"
		}
		if err := notify.ToMembersWith(ctx, tx, c.WorkspaceID, authz.CampaignsSend, notify.Notification{
			Kind: notify.KindSendPaused, Severity: "bad", TitleText: keep(titleKey, words), BodyText: keep(bodyKey, words),
			Link: liveLink(c.WorkspaceSlug, c.ID), Data: map[string]any{"campaignId": c.ID.String(), "providerId": pid.String(), "incidentId": id.String()},
		}); err != nil {
			return err
		}
		if err := emit(ctx, tx, c.WorkspaceID, "campaign.paused", campaignData(c.ID, c.Name, "paused_by", "provider_outage", "reason", headline, "scope", "route",
			"provider_id", pid.String(), "provider", r.provider.Name, "held", remaining)); err != nil {
			return err
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.share_held", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Reason: headline, Actor: systemActor, Metadata: map[string]any{"provider": r.provider.Name, "reason": reason, "held": remaining, "policy": detail.Policy,
				"summary": fmt.Sprintf("%s · %s recipients held, decision requested", headline, countText(remaining)), "target": c.Name + " · " + short + " share",
				"source": "guard", "via": "provider outage", "asked_again": again}})
	})
	if err == nil && skipped {
		// Someone decided in the meantime: read the routes again.
		rs.loadedAt = time.Time{}
	}
	return err
}

// holdable reports whether a route may be put on hold now: it still sends,
// or it is held and either waits for a decision or (again) was retried.
func holdable(rt store.SendingRoutesRow, again bool) bool {
	switch rt.Status {
	case "pending", "sending":
		return true
	case "held":
		return again || parseRouteDetail(rt.StatusDetail).Decision != "retry"
	}
	return false
}

func (e *Engine) markHeld(ctx context.Context, q *store.Queries, rs *runState, r *runRoute, reason string, incident uuid.UUID) error {
	d, _ := json.Marshal(routeDetail{Reason: reason, Incident: incident.String()})
	if err := q.SendingSetRouteStatus(ctx, store.SendingSetRouteStatusParams{CampaignID: rs.campaign.ID, ProviderID: r.provider.ID, Status: "held", StatusDetail: string(d)}); err != nil {
		return err
	}
	r.status, r.detail = "held", routeDetail{Reason: reason, Incident: incident.String()}
	return nil
}

// autoMove fails a route over to a warmed-up provider on its own (the
// Broadcast policy allows it) and records it.
func (e *Engine) autoMove(ctx context.Context, rs *runState, r *runRoute, target *store.Provider, reason string, policy routing.Policy, now time.Time) error {
	c := rs.campaign
	cid := c.ID
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.SendingCampaignForUpdate(ctx, cid)
		if err != nil {
			return err
		}
		rt, err := routeOf(ctx, q, cid, r.provider.ID)
		if err != nil {
			return err
		}
		if cur.Status != "sending" && cur.Status != "paused" || rt == nil || (rt.Status != "pending" && rt.Status != "sending" && rt.Status != "held") {
			// Someone decided in the meantime.
			rs.loadedAt = time.Time{}
			return nil
		}
		moved, err := e.moveShare(ctx, q, cid, r.provider.ID, target.ID)
		if err != nil {
			return err
		}
		pid := r.provider.ID
		detail, _ := json.Marshal(map[string]any{"reason": reason, "moved": moved, "to": target.Name, "policy": string(policy), "log": e.callLines(routeKey{cid, pid})})
		if err := q.SendingInsertIncident(ctx, store.SendingInsertIncidentParams{ID: newID(), CampaignID: cid, Kind: "provider_outage", Scope: "route", ProviderID: &pid,
			Metric: "api_errors", Held: 0, Detail: detail, TrippedAt: now}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE campaign_incidents SET resolved_at = $2, resolution = 'moved' WHERE campaign_id = $1 AND provider_id = $3 AND resolved_at IS NULL`, cid, now, pid); err != nil {
			return err
		}
		r.status = "done"
		title := keep("incident.failover", i18n.Args{"name": quoted(c.Name), "who": r.shortName(), "target": state.Short(target.Name, target.Adapter), "moved": moved})
		if err := q.ProvInsertIncident(ctx, store.ProvInsertIncidentParams{ID: newID(), WorkspaceID: c.WorkspaceID, OccurredAt: now, Kind: "failover", Severity: "ok",
			Title: title.In(i18n.Default), CampaignID: &cid, ProviderID: &pid, Texts: titleTexts(title)}); err != nil {
			return err
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.share_moved", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: systemActor, Metadata: map[string]any{"from": r.provider.Name, "to": target.Name, "moved": moved, "automatic": true, "policy": string(policy),
				"source": "guard", "via": "failover",
				"summary": fmt.Sprintf("%s recipients moved from %s to %s automatically (failover policy)", countText(moved), r.provider.Name, target.Name)}})
	})
}

// moveShare moves what is left of one provider's share to another.
func (e *Engine) moveShare(ctx context.Context, q *store.Queries, campaign, from, to uuid.UUID) (int64, error) {
	cid := campaign
	moved, err := q.SendingMoveRemaining(ctx, store.SendingMoveRemainingParams{CampaignID: &cid, FromProvider: &from, ToProvider: &to})
	if err != nil {
		return 0, err
	}
	if err := q.SendingAddPlanned(ctx, store.SendingAddPlannedParams{CampaignID: campaign, ProviderID: to, Planned: moved}); err != nil {
		return 0, err
	}
	routes, err := q.SendingRoutes(ctx, campaign)
	if err != nil {
		return 0, err
	}
	for _, rt := range routes {
		switch rt.ProviderID {
		case from:
			d, _ := json.Marshal(map[string]any{"decision": "move", "to": to.String(), "moved": moved})
			if err := q.SendingSetRouteStatus(ctx, store.SendingSetRouteStatusParams{CampaignID: campaign, ProviderID: from, Status: "done", StatusDetail: string(d)}); err != nil {
				return 0, err
			}
			if err := e.adjustPlanned(ctx, q, campaign, from, -moved); err != nil {
				return 0, err
			}
		case to:
			if rt.Status == "done" || rt.Status == "pending" {
				if err := q.SendingSetRouteStatus(ctx, store.SendingSetRouteStatusParams{CampaignID: campaign, ProviderID: to, Status: "sending", StatusDetail: rt.StatusDetail}); err != nil {
					return 0, err
				}
			}
		}
	}
	return moved, nil
}

func (e *Engine) adjustPlanned(ctx context.Context, q *store.Queries, campaign, provider uuid.UUID, by int64) error {
	return q.SendingAddPlanned(ctx, store.SendingAddPlannedParams{CampaignID: campaign, ProviderID: provider, Planned: by})
}

func (e *Engine) callLines(k routeKey) []IncidentLogLine {
	l := e.callLogFor(k)
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]IncidentLogLine(nil), l.lines...)
}

// checkAB decides an A/B test when its window ends and releases the winner
// to everyone who waited.
func (e *Engine) checkAB(ctx context.Context, rs *runState, ts tallies, now time.Time) error {
	if rs.ab.Phase != "testing" || rs.ab.DecideAt == nil || now.Before(*rs.ab.DecideAt) {
		return nil
	}
	winner, rates := abWinner(rs.ab, ts)
	return e.decideAB(ctx, rs.campaign, winner, rates, nil, "", now)
}

// abWinner picks the variant with the best rate on the test's metric.
func abWinner(ab abState, ts tallies) (string, map[string]float64) {
	rates := map[string]float64{}
	best, bestRate := "", -1.0
	for _, v := range ab.Variants {
		t := ts.variant[v]
		r := 0.0
		if t != nil {
			if ab.Metric == "clicks" {
				r = rate(t.clicked, t.delivered)
			} else {
				r = rate(t.human, t.delivered)
			}
		}
		rates[v] = round2(r)
		if r > bestRate {
			best, bestRate = v, r
		}
	}
	return best, rates
}

func (e *Engine) decideAB(ctx context.Context, c store.SendingCampaignContextRow, winner string, rates map[string]float64, by *uuid.UUID, byName string, now time.Time) error {
	err := db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		s, err := q.SendingGetSendForUpdate(ctx, c.ID)
		if err != nil {
			return err
		}
		ab := parseAB(s.Ab)
		if ab.Phase != "testing" {
			return nil
		}
		cid := c.ID
		released, err := q.SendingReleaseHeld(ctx, store.SendingReleaseHeldParams{CampaignID: &cid, Variant: winner})
		if err != nil {
			return err
		}
		ab.Phase, ab.Winner, ab.DecidedAt, ab.DecidedBy, ab.Rates = "decided", winner, &now, byName, rates
		b, _ := json.Marshal(ab)
		if err := q.SendingSetAB(ctx, store.SendingSetABParams{CampaignID: c.ID, Ab: b, At: now}); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE campaigns SET ab_test = jsonb_set(coalesce(ab_test, '{}'::jsonb), '{winner}', to_jsonb($2::text)) WHERE id = $1 AND ab_test IS NOT NULL`, c.ID, winner); err != nil {
			return err
		}
		actor := systemActor
		if by != nil {
			actor = &audit.Actor{Type: authz.ActorUser, ID: by, Name: byName}
		}
		if audit.PrincipalFrom(ctx) != nil {
			actor = nil
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.ab_winner_picked", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: actor, After: map[string]any{"winner": winner, "rates": rates, "released": released}})
	})
	if err == nil {
		e.kick(ctx, c.ID)
	}
	return err
}

// waitForDecision marks a campaign whose only remaining recipients are on
// hold as paused, so lists and the monitor say nothing is sending.
func (e *Engine) waitForDecision(ctx context.Context, rs *runState, now time.Time) error {
	if err := e.Q.SendingSetCampaignStatus(ctx, store.SendingSetCampaignStatusParams{ID: rs.campaign.ID, Status: "paused", At: now}); err != nil {
		return err
	}
	return errStopped
}

// finish completes a send: nothing is left to send.
func (e *Engine) finish(ctx context.Context, rs *runState, ts tallies, now time.Time) error {
	c := rs.campaign
	status := "sent"
	if ts.total.sent == 0 {
		status = "cancelled"
	}
	err := db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.SendingCampaignForUpdate(ctx, c.ID)
		if err != nil {
			return err
		}
		if cur.Status != "sending" {
			return nil
		}
		if err := q.SendingMarkCompleted(ctx, store.SendingMarkCompletedParams{ID: c.ID, Status: status, At: now}); err != nil {
			return err
		}
		if err := q.SendingFinishRoutes(ctx, store.SendingFinishRoutesParams{CampaignID: c.ID, Status: "done"}); err != nil {
			return err
		}
		if err := q.SendingSetDeliveryP50(ctx, c.ID); err != nil {
			return err
		}
		if err := q.SendingSetPhase(ctx, store.SendingSetPhaseParams{CampaignID: c.ID, Phase: "finished", FinishedAt: &now, At: now}); err != nil {
			return err
		}
		// Complaints keep arriving for days: they're watched (review GS-6).
		if err := startWatch(ctx, q, c, now); err != nil {
			return err
		}
		t := ts.total
		dur := time.Duration(0)
		if cur.StartedAt != nil {
			dur = now.Sub(*cur.StartedAt)
		}
		// "47,980 sent in 1 h 12 min: 47,512 delivered (99.0%), 312
		// bounced, 7 complaints", kept for each reader's language.
		delivered, _ := strconv.ParseFloat(strconv.FormatFloat(rate(t.delivered, t.sent), 'f', 1, 64), 64)
		words := i18n.Args{"name": quoted(c.Name), "sent": t.sent, "delivered": t.delivered, "rate": delivered / 100, "bounced": t.bounced(), "complaints": t.complained}
		durationArgs(words, dur)
		seen := map[uuid.UUID]bool{}
		for _, u := range []*uuid.UUID{rs.send.StartedBy, rs.send.ScheduledBy} {
			if u == nil || seen[*u] {
				continue
			}
			seen[*u] = true
			wsID := c.WorkspaceID
			if err := notify.ToUser(ctx, tx, *u, notify.Notification{WorkspaceID: &wsID, Kind: notify.KindSendFinished, Severity: "ok", TitleText: keep("notify.finishedTitle", words),
				BodyText: keep("notify.finishedBody", words), Link: liveLink(c.WorkspaceSlug, c.ID), Data: map[string]any{"campaignId": c.ID.String()}}); err != nil {
				return err
			}
		}
		// As the audit log shows it: "47,980 sent · SES 43,182 · Postal 4,798
		// · 99.1% delivered · $4.32".
		parts := []string{countText(t.sent) + " sent"}
		type share struct {
			name string
			sent int64
		}
		shares := []share{}
		for pid, pt := range ts.provider {
			if pt.sent > 0 {
				shares = append(shares, share{rs.routes[pid].shortName(), pt.sent})
			}
		}
		sort.Slice(shares, func(i, j int) bool {
			if shares[i].sent != shares[j].sent {
				return shares[i].sent > shares[j].sent
			}
			return shares[i].name < shares[j].name
		})
		for _, sh := range shares {
			parts = append(parts, sh.name+" "+countText(sh.sent))
		}
		parts = append(parts, fmtPct(rate(t.delivered, t.sent))+" delivered")
		var micros int64
		routes, err := q.SendingRoutes(ctx, c.ID)
		if err != nil {
			return err
		}
		for _, r := range routes {
			micros += r.CostMicros
		}
		cost := round2(float64(micros) / 1e6)
		if micros > 0 {
			parts = append(parts, fmt.Sprintf("$%.2f", cost))
		}
		if err := emit(ctx, tx, c.WorkspaceID, "campaign.completed", campaignData(c.ID, c.Name, "status", status, "sent", t.sent, "delivered", t.delivered,
			"bounced", t.bounced(), "complained", t.complained, "cost", cost, "duration_seconds", int64(dur/time.Second))); err != nil {
			return err
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.completed", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: e.systemFor(ctx, c.ID), Before: map[string]any{"status": "sending"}, After: map[string]any{"status": status, "sent": t.sent, "delivered": t.delivered, "bounced": t.bounced(), "complained": t.complained},
			Metadata: map[string]any{"summary": strings.Join(parts, " · "), "source": "send worker"}})
	})
	if err != nil {
		return err
	}
	return errStopped
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }
