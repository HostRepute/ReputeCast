package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// Provider outages (review GS-3). Whoever finds an outage, the send loop
// (checkOutage) or the provider health job (HoldShares), the share goes
// through the same path: moved when the Broadcast failover policy allows it
// and a vetted provider can take it, otherwise held with the engine's
// analysis and its ways to continue. A person's Retry keeps the share on
// hold and sends one test message every minute; three that go through in a
// row resume the share by itself, and one that fails after 15 minutes asks
// a person again.

// probesToResume is how many test messages in a row a retried share's
// provider must take before the share resumes by itself.
const probesToResume = 3

// probeHeader marks the test message a retried share sends to a seed inbox.
const probeHeader = "X-ReputeCast-Probe"

// HoldShares implements sending.Outages: every running send's share on a
// provider the health job marked down goes through the send's outage path.
// A send whose loop is running right now is left to it: the loop rereads
// the provider within seconds and takes the same path.
func (e *Engine) HoldShares(ctx context.Context, providerID uuid.UUID, reason string) error {
	if reason != "auth" {
		reason = "provider_down"
	}
	ids, err := e.Q.SendingSharesOn(ctx, providerID)
	if err != nil {
		return err
	}
	var errs []error
	for _, id := range ids {
		if err := e.holdShare(ctx, id, providerID, reason); err != nil {
			errs = append(errs, fmt.Errorf("campaign %s: %w", id, err))
		}
	}
	return errors.Join(errs...)
}

func (e *Engine) holdShare(ctx context.Context, campaignID, providerID uuid.UUID, reason string) error {
	unlock, err := e.lockSend(ctx, campaignID)
	if errors.Is(err, errBusy) {
		return nil
	}
	if err != nil {
		return err
	}
	defer unlock()
	now := e.now()
	rs, err := e.loadState(ctx, campaignID, now, true)
	if errors.Is(err, errStopped) {
		return nil
	}
	if err != nil {
		return err
	}
	r := rs.routes[providerID]
	switch {
	case r == nil:
		return nil
	case r.open():
		return e.outage(ctx, rs, r, reason, now, false)
	case r.status == "held" && r.detail.Decision != "retry":
		in, err := e.routeIncident(ctx, e.Q, campaignID, providerID)
		if err != nil {
			return err
		}
		return e.adoptHold(ctx, rs, r, in, now)
	}
	return nil
}

// routeIncident is a route's open outage incident (nil when it has none).
func (e *Engine) routeIncident(ctx context.Context, q *store.Queries, campaignID, providerID uuid.UUID) (*store.CampaignIncident, error) {
	open, err := q.SendingOpenIncidents(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	for i := range open {
		if open[i].Kind == "provider_outage" && open[i].ProviderID != nil && *open[i].ProviderID == providerID {
			return &open[i], nil
		}
	}
	return nil, nil
}

// adoptHold makes sure a held share has a decision to take: one nobody was
// asked about gets its incident, and an incident that offers no way to
// continue (a hold from before the health job used this path) gets the
// engine's analysis and options. A person's earlier choice to wait for
// the provider is kept.
func (e *Engine) adoptHold(ctx context.Context, rs *runState, r *runRoute, in *store.CampaignIncident, now time.Time) error {
	if in == nil {
		if r.detail.Decision != "" {
			return nil
		}
		return e.holdOutage(ctx, rs, r, firstNonEmpty(r.detail.Reason, "outage"), false, now)
	}
	old := parseDetail(in.Detail)
	if len(old.Options) > 0 {
		return nil
	}
	cid := rs.campaign.ID
	remaining, err := e.Q.SendingRemainingOn(ctx, store.SendingRemainingOnParams{CampaignID: &cid, ProviderID: &r.provider.ID})
	if err != nil {
		return err
	}
	d := e.analyzeOutage(ctx, rs, r, firstNonEmpty(old.Reason, r.detail.Reason, "outage"), remaining, now)
	if err := e.Q.SendingUpdateIncidentDetail(ctx, store.SendingUpdateIncidentDetailParams{ID: in.ID, Detail: d.stored(), Held: remaining}); err != nil {
		return err
	}
	if r.detail.Incident == "" {
		det := r.detail
		det.Incident = in.ID.String()
		if det.Reason == "" {
			det.Reason = d.Reason
		}
		return e.setRouteDetail(ctx, rs, r, det, "held")
	}
	return nil
}

// probeHeld looks after a held route: a retried share's test messages,
// otherwise its incident (options for one that has none, and one health
// check after the hold: "health check · still failing").
func (e *Engine) probeHeld(ctx context.Context, rs *runState, r *runRoute, now time.Time) error {
	if r.detail.Decision == "retry" {
		return e.retryProbe(ctx, rs, r, now)
	}
	if now.Sub(r.lastProbe) < 30*time.Second {
		return nil
	}
	r.lastProbe = now
	in, err := e.routeIncident(ctx, e.Q, rs.campaign.ID, r.provider.ID)
	if err != nil {
		return err
	}
	if in == nil || len(parseDetail(in.Detail).Options) == 0 {
		return e.adoptHold(ctx, rs, r, in, now)
	}
	if now.Sub(in.TrippedAt) < 30*time.Second {
		return nil
	}
	d := parseDetail(in.Detail)
	if d.Probed {
		return nil
	}
	line := english("probe.failing", nil)
	if e.probeAPI(ctx, r, now) {
		line = english("probe.answering", nil)
	}
	d.Log = append(d.Log, IncidentLogLine{At: now, Text: line})
	d.Probed = true
	return e.Q.SendingUpdateIncidentDetail(ctx, store.SendingUpdateIncidentDetailParams{ID: in.ID, Detail: d.stored(), Held: in.Held})
}

// retryProbe sends a retried share's test message when it is due. Three
// in a row that go through resume the share; a failure once the 15 minutes
// are over asks a person again (or moves the share when the failover
// policy allows it).
func (e *Engine) retryProbe(ctx context.Context, rs *runState, r *runRoute, now time.Time) error {
	d := r.detail
	if d.NextProbe != nil && now.Before(*d.NextProbe) {
		return nil
	}
	ok, err := e.probe(ctx, rs, r, d.Seed, now)
	if err != nil {
		return err
	}
	if ok {
		d.Probes++
		if d.Probes >= probesToResume {
			return e.resumeProbed(ctx, rs, r, now)
		}
	} else {
		d.Probes = 0
		if d.Until != nil && !now.Before(*d.Until) {
			return e.outage(ctx, rs, r, firstNonEmpty(d.Reason, "outage"), now, true)
		}
	}
	next := now.Add(retryEvery)
	d.NextProbe = &next
	n, err := e.Q.SendingSetProbe(ctx, store.SendingSetProbeParams{CampaignID: rs.campaign.ID, ProviderID: r.provider.ID, Old: rawRouteDetail(r.detail), New: rawRouteDetail(d)})
	if err != nil {
		return err
	}
	if n == 1 {
		r.detail = d
	} else {
		// Someone decided in the meantime: read the route again.
		rs.loadedAt = time.Time{}
	}
	return nil
}

// probe sends one test message through a retried share's provider: to a
// connected seed inbox when the workspace has one, else one of the share's
// own messages (it counts as sent when the provider takes it). When
// nothing of the share can go yet, the provider's API answers instead.
func (e *Engine) probe(ctx context.Context, rs *runState, r *runRoute, seed bool, now time.Time) (bool, error) {
	if r.transport == nil {
		// The adapter can't be opened (its settings): the test fails.
		return false, nil
	}
	if seed {
		if to := e.seedInbox(ctx, rs.campaign.WorkspaceID); to != "" {
			return e.probeSeed(ctx, rs, r, to, now), nil
		}
	}
	ok, tried, err := e.probeMessage(ctx, rs, r, now)
	if err != nil || tried {
		return ok, err
	}
	return e.probeAPI(ctx, r, now), nil
}

// seedInbox is a connected seed inbox of the workspace ("" without one).
func (e *Engine) seedInbox(ctx context.Context, ws uuid.UUID) string {
	rows, err := e.Q.SendingProbeSeed(ctx, ws)
	if err != nil || len(rows) == 0 {
		return ""
	}
	return rows[0]
}

// probeSeed sends the test message to a seed inbox.
func (e *Engine) probeSeed(ctx context.Context, rs *runState, r *runRoute, to string, now time.Time) bool {
	c := rs.campaign
	msg := providers.Message{ID: newID(), Stream: providers.StreamBroadcast,
		From: mail.Address{Name: rs.content.FromName, Address: rs.content.FromEmail}, To: mail.Address{Address: to},
		Subject: "ReputeCast test message through " + r.provider.Name,
		Text: fmt.Sprintf("ReputeCast sent this message through %s to check that it takes mail again before the rest of %q goes out. You can delete it.",
			r.provider.Name, c.Name),
		Headers: map[string]string{probeHeader: "outage"}, Tags: map[string]string{"probe": "outage", "campaign": c.ID.String()}}
	if rs.content.ReplyTo != "" {
		msg.ReplyTo = &mail.Address{Address: rs.content.ReplyTo}
	}
	cctx, cancel := e.callContext(ctx, now)
	res, err := r.transport.Send(cctx, []providers.Message{msg})
	cancel()
	if err == nil && len(res) != 1 {
		err = &providers.CallError{Op: "Send", Message: fmt.Sprintf("the provider answered for %d of 1 messages", len(res)), Temporary: true}
	}
	if err == nil && res[0].Err != nil {
		err = res[0].Err
	}
	e.recordCall(routeKey{c.ID, r.provider.ID}, now, err)
	if rerr := manage.RecordSend(context.WithoutCancel(ctx), e.Pool, r.provider, res, err, now); rerr != nil {
		e.log().WarnContext(ctx, "record provider call", "provider", r.provider.ID, "err", rerr)
	}
	return err == nil
}

// probeMessage sends one of the share's own messages that can go now. It
// reports whether one was tried and whether the provider took it.
func (e *Engine) probeMessage(ctx context.Context, rs *runState, r *runRoute, now time.Time) (ok, tried bool, err error) {
	cid, pid := rs.campaign.ID, r.provider.ID
	var keys []string
	for k, c := range rs.queue {
		if k.provider != pid || c.queued <= c.waiting {
			continue
		}
		if h, on := rs.holds[k.mailbox]; on && !h.Resumed {
			continue
		}
		keys = append(keys, k.mailbox)
	}
	sort.Strings(keys)
	for _, mbp := range keys {
		rows, err := e.Q.SendingClaim(ctx, store.SendingClaimParams{CampaignID: &cid, ProviderID: &pid, MailboxProvider: mbp, Now: now, Lim: 1})
		if err != nil {
			return false, false, err
		}
		if len(rows) == 0 {
			continue
		}
		rs.queueAt = time.Time{}
		res, err := e.send(ctx, rs, r, rows, now)
		return res.sent > 0, true, err
	}
	return false, false, nil
}

// probeAPI asks the provider's API whether it answers (its quota, or its
// credentials check); the demo's simulated provider answers from its
// scenario.
func (e *Engine) probeAPI(ctx context.Context, r *runRoute, now time.Time) bool {
	switch {
	case r.sim != nil:
		return !r.sim.Outage(now)
	case r.transport == nil:
		return r.provider.Status != state.Down
	}
	cctx, cancel := e.callContext(ctx, now)
	defer cancel()
	switch a := r.transport.(type) {
	case providers.QuotaReporter:
		_, err := a.Quota(cctx)
		return err == nil
	case providers.Verifier:
		_, err := a.Verify(cctx)
		return err == nil
	}
	return false
}

// resumeProbed sends a retried share again after its provider took three
// test messages in a row.
func (e *Engine) resumeProbed(ctx context.Context, rs *runState, r *runRoute, now time.Time) error {
	c := rs.campaign
	pid := r.provider.ID
	resumed := false
	err := db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		cur, err := q.SendingCampaignForUpdate(ctx, c.ID)
		if err != nil {
			return err
		}
		if cur.Status != "sending" && cur.Status != "paused" {
			return nil
		}
		rt, err := routeOf(ctx, q, c.ID, pid)
		if err != nil || rt == nil || rt.Status != "held" || parseRouteDetail(rt.StatusDetail).Decision != "retry" {
			return err
		}
		if err := q.SendingSetRouteStatus(ctx, store.SendingSetRouteStatusParams{CampaignID: c.ID, ProviderID: pid, Status: "sending", StatusDetail: `{"decision":"resume"}`}); err != nil {
			return err
		}
		resumed = true
		if err := emit(ctx, tx, c.WorkspaceID, "campaign.resumed", campaignData(c.ID, c.Name, "resumed_by", "System", "scope", "route",
			"provider_id", pid.String(), "provider", r.provider.Name, "decision", "retry")); err != nil {
			return err
		}
		summary := english("audit.shareProbed", i18n.Args{"who": r.shortName(), "count": probesToResume})
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.share_resumed", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: systemActor, Before: map[string]any{"route": "held"}, After: map[string]any{"route": "sending", "probes": probesToResume},
			Metadata: map[string]any{"summary": summary, "provider": r.provider.Name, "source": "guard", "via": "retry"}})
	})
	if err != nil || !resumed {
		return err
	}
	r.status, r.detail = "sending", routeDetail{Decision: "resume"}
	r.judgedFrom = now
	e.resetCalls(routeKey{c.ID, pid})
	return nil
}

// routeOf is one route of a campaign, read in a transaction (nil when the
// campaign has no share on the provider).
func routeOf(ctx context.Context, q *store.Queries, campaignID, providerID uuid.UUID) (*store.SendingRoutesRow, error) {
	routes, err := q.SendingRoutes(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	for i := range routes {
		if routes[i].ProviderID == providerID {
			return &routes[i], nil
		}
	}
	return nil, nil
}

// rawRouteDetail is a route's detail as campaign_routes.status_detail keeps
// it.
func rawRouteDetail(d routeDetail) string {
	b, _ := json.Marshal(d)
	if s := string(b); s != "{}" {
		return s
	}
	return ""
}

// retried marks an outage asked about again after a retry: still failing
// after 15 minutes of test messages. It is the first likely cause.
func (d *IncidentDetail) retried(who string) {
	d.Retried = true
	d.addCause(keep("cause.retriedLead", i18n.Args{"mins": int64(retryFor / time.Minute)}), keep("cause.retried", i18n.Args{"who": who}))
	n := len(d.Causes) - 1
	d.Causes = append([]IncidentCause{d.Causes[n]}, d.Causes[:n]...)
	k := d.kept()
	k.Causes = append([]causeTexts{k.Causes[n]}, k.Causes[:n]...)
}

// canaryHolds reports whether recipients wait for the canary to pass.
func (rs *runState) canaryHolds() bool {
	for _, c := range rs.queue {
		if c.canary > 0 {
			return true
		}
	}
	return false
}
