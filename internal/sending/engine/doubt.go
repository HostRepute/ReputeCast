package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

// No message is sent twice (review GS-1).
//
// A provider call can end without an answer ReputeCast recorded: the worker
// stops in the middle of it (a deploy, a crash), the provider took the batch
// but the answer came too late, the result couldn't be written. Its messages
// stay claimed, and a claim is never taken over (SendingClaim). Once a claim
// is older than staleClaim the guard settles it: a provider that dedupes by
// message id (providers.IdempotentSender) gets the message again inside its
// idempotency window, with the same key; any other message is in doubt.
// Nothing sends a message in doubt again on a guess: an event the provider
// reports for it settles it (InDoubtHook), or a person sends it again or
// leaves it out (ResolveInDoubt). While messages are in doubt and nothing
// else is left, the send waits for that decision.

const (
	// settleEvery is how often a send's stale claims are looked for.
	settleEvery = 10 * time.Second
	// idempotencyMargin keeps a message sent again clear of the end of the
	// provider's idempotency window.
	idempotencyMargin = 5 * time.Minute
)

// inDoubtReason is messages.error of a message in doubt.
const inDoubtReason = "In doubt: the call that handed it to the provider ended without a recorded answer, so it isn't sent again unless someone decides to."

// idempotency is a route's idempotency window, 0 when its provider doesn't
// dedupe (the demo transport stands in for providers that don't).
func idempotency(t providers.Provider) time.Duration {
	if t == nil {
		return 0
	}
	return providers.IdempotencyWindow(t)
}

// adapterWindows caches each adapter's idempotency window (an adapter's
// window doesn't depend on its account).
var adapterWindows sync.Map // adapter → time.Duration

// dedupes reports whether a connected provider recognizes a message it
// already took (its adapter is a providers.IdempotentSender); the demo
// transport doesn't.
func dedupes(p store.Provider) bool { return window(p) > 0 }

// window is how long a connected provider recognizes a message it already
// took, 0 when it doesn't (the demo transport doesn't).
func window(p store.Provider) time.Duration {
	if manage.IsDemo(p) {
		return 0
	}
	if w, ok := adapterWindows.Load(p.Adapter); ok {
		return w.(time.Duration)
	}
	var w time.Duration
	if d, ok := providers.Lookup(p.Adapter); ok && d.New != nil {
		if t, err := d.New(providers.Config{ProviderID: p.ID, WorkspaceID: p.WorkspaceID}); err == nil {
			w = providers.IdempotencyWindow(t)
		}
	}
	adapterWindows.Store(p.Adapter, w)
	return w
}

// resendable is how far back a stale claim of a provider that dedupes may
// be sent again, false when it may not.
func resendable(t providers.Provider, now time.Time) (time.Time, bool) {
	w := idempotency(t)
	if w <= idempotencyMargin+staleClaim {
		return time.Time{}, false
	}
	return now.Add(-(w - idempotencyMargin)), true
}

// settleStale settles a send's stale claims (see above) and keeps the
// in-doubt incident in step with what is still in doubt.
func (e *Engine) settleStale(ctx context.Context, rs *runState, now time.Time) error {
	if !rs.settledAt.IsZero() && now.Sub(rs.settledAt) < settleEvery {
		return nil
	}
	rs.settledAt = now
	cid := rs.campaign.ID
	stale := now.Add(-staleClaim)
	var ids []uuid.UUID
	var since []time.Time
	for _, pid := range rs.order {
		if at, ok := resendable(rs.routes[pid].transport, now); ok {
			ids, since = append(ids, pid), append(since, at)
		}
	}
	if len(ids) > 0 {
		if _, err := e.Q.SendingRequeueStale(ctx, store.SendingRequeueStaleParams{ProviderIds: ids, Since: since, Stale: stale, CampaignID: &cid}); err != nil {
			return err
		}
	}
	marked, err := e.Q.SendingMarkInDoubt(ctx, store.SendingMarkInDoubtParams{Reason: inDoubtReason, Stale: stale, CampaignID: &cid})
	if err != nil {
		return err
	}
	if len(marked) > 0 {
		rs.queueAt = time.Time{}
	}
	return e.syncInDoubt(ctx, rs.campaign, now)
}

// InDoubtShare is one sending provider's messages in doubt.
type InDoubtShare struct {
	ProviderID string `json:"providerId"`
	Provider   string `json:"provider"`
	Short      string `json:"short"`
	Count      int64  `json:"count"`
	// Since is when the oldest of them was handed over (the call whose
	// answer was lost).
	Since time.Time `json:"since"`
	// Idempotent: the provider recognizes a message it already took for
	// WindowSeconds after the first try (its idempotency key).
	Idempotent    bool  `json:"idempotent"`
	WindowSeconds int64 `json:"windowSeconds,omitempty"`
	// SafeUntil is until when sending them again can't deliver any twice:
	// the end of the provider's window for the oldest of them, less a
	// margin. Absent when the provider doesn't recognize them at all.
	SafeUntil *time.Time `json:"safeUntil,omitempty"`
	// Safe: sending them again now can't deliver any twice.
	Safe bool `json:"safe"`
}

// inDoubtShares are a campaign's messages in doubt per sending provider, as
// of now: whether sending them again is still inside each provider's
// idempotency window.
func (e *Engine) inDoubtShares(ctx context.Context, q *store.Queries, campaign uuid.UUID, now time.Time) ([]InDoubtShare, int64, error) {
	rows, err := q.SendingInDoubtByProvider(ctx, &campaign)
	if err != nil {
		return nil, 0, err
	}
	out := []InDoubtShare{}
	var total int64
	for _, r := range rows {
		sh := InDoubtShare{Count: r.N, Since: r.Since}
		if r.ProviderID != nil {
			sh.ProviderID = r.ProviderID.String()
			if p, err := q.ProvGetAny(ctx, *r.ProviderID); err == nil {
				sh.Provider, sh.Short = p.Name, state.Short(p.Name, p.Adapter)
				if w := window(p); w > 0 {
					until := r.Since.Add(w - idempotencyMargin)
					sh.Idempotent, sh.WindowSeconds, sh.SafeUntil = true, int64(w/time.Second), &until
					sh.Safe = now.Before(until)
				}
			}
		}
		total += r.N
		out = append(out, sh)
	}
	return out, total, nil
}

// syncInDoubt opens the in-doubt incident when messages are in doubt (the
// team is told once), keeps its count current, and resolves it once events
// settled every one of them.
func (e *Engine) syncInDoubt(ctx context.Context, c store.SendingCampaignContextRow, now time.Time) error {
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		shares, total, err := e.inDoubtShares(ctx, q, c.ID, now)
		if err != nil {
			return err
		}
		open, err := q.SendingOpenIncidents(ctx, c.ID)
		if err != nil {
			return err
		}
		var inc *store.CampaignIncident
		for i := range open {
			if open[i].Kind == "in_doubt" {
				inc = &open[i]
				break
			}
		}
		switch {
		case total == 0 && inc == nil:
			return nil
		case total == 0:
			return q.SendingResolveIncident(ctx, store.SendingResolveIncidentParams{ID: inc.ID, At: &now, Resolution: "auto_recovered"})
		case inc != nil:
			if inc.Held == total {
				return nil
			}
			return q.SendingUpdateIncidentDetail(ctx, store.SendingUpdateIncidentDetailParams{ID: inc.ID, Detail: inDoubtDetail(shares, total).stored(), Held: total})
		}
		id := newID()
		d := inDoubtDetail(shares, total)
		if err := q.SendingInsertIncident(ctx, store.SendingInsertIncidentParams{ID: id, CampaignID: c.ID, Kind: "in_doubt", Scope: "campaign",
			Metric: "in_doubt", Held: total, Detail: d.stored(), TrippedAt: now}); err != nil {
			return err
		}
		words := i18n.Args{"name": quoted(c.Name), "count": total}
		if err := notify.ToMembersWith(ctx, tx, c.WorkspaceID, authz.CampaignsSend, notify.Notification{
			Kind: notify.KindSendPaused, Severity: "warn", TitleText: keep("notify.inDoubtTitle", words), BodyText: keep("notify.inDoubtBody", words),
			Link: liveLink(c.WorkspaceSlug, c.ID), Data: map[string]any{"campaignId": c.ID.String(), "incidentId": id.String()},
		}); err != nil {
			return err
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.messages_in_doubt", TargetType: "campaign", TargetID: c.ID.String(),
			TargetName: c.Name, Actor: systemActor, Reason: english("notify.inDoubtBody", words),
			Metadata: map[string]any{"in_doubt": total, "summary": english("audit.inDoubt", words), "source": "send worker", "via": "stale claim"}})
	})
}

// inDoubtDetail is the in-doubt incident's detail: the shares and the two
// ways to continue. Whether sending them again is safe changes with time
// (a provider recognizes a message only inside its idempotency window), so
// the live monitor works it out again for every snapshot.
func inDoubtDetail(shares []InDoubtShare, total int64) IncidentDetail {
	d := IncidentDetail{InDoubt: shares, Remaining: total}
	safe := inDoubtSafety(shares)
	args := i18n.Args{"count": total, "safe": safe}
	d.setOptions([]option{
		{key: "resend", recommended: safe == "yes", title: keep("opt.resendTitle", args), body: keep("opt.resendBody", args), action: keep("opt.resendAction", args)},
		{key: "leave_out", recommended: safe != "yes", title: keep("opt.leaveOutTitle", args), body: keep("opt.leaveOutBody", args), action: keep("opt.leaveOutAction", args)},
	})
	return d
}

// inDoubtSafety is whether sending the shares again now can deliver any
// twice: "yes" when every provider still recognizes its messages, "no"
// when none does, "some" otherwise.
func inDoubtSafety(shares []InDoubtShare) string {
	safe, unsafe := 0, 0
	for _, sh := range shares {
		if sh.Safe {
			safe++
		} else {
			unsafe++
		}
	}
	switch {
	case safe > 0 && unsafe == 0:
		return "yes"
	case safe > 0:
		return "some"
	}
	return "no"
}

// ResolveInDoubt is a person's decision on a campaign's messages in doubt:
// send them again (resend) or leave them out. It returns how many it
// changed.
func (e *Engine) ResolveInDoubt(ctx context.Context, campaignID, by uuid.UUID, resend bool, provider *uuid.UUID) (int64, error) {
	now := e.now()
	var n int64
	err := db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := q.SendingCampaignForUpdate(ctx, campaignID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if c.Status != "sending" && c.Status != "paused" {
			return stateErr("state.notSending", campaignArgs(c.Name, c.Status))
		}
		ref, name := e.actorRef(ctx, q, by)
		reason := fmt.Sprintf("Left out by %s: in doubt after its provider call ended without a recorded answer.", firstNonEmpty(name, "a member"))
		if n, err = q.SendingResolveInDoubt(ctx, store.SendingResolveInDoubtParams{Resend: resend, Reason: reason, CampaignID: &campaignID, ProviderID: provider}); err != nil {
			return err
		}
		if n == 0 {
			return stateErr("state.nothingInDoubt", campaignArgs(c.Name, c.Status))
		}
		_, left, err := e.inDoubtShares(ctx, q, campaignID, now)
		if err != nil {
			return err
		}
		open, err := q.SendingOpenIncidents(ctx, campaignID)
		if err != nil {
			return err
		}
		resolution := "cancelled"
		if resend {
			resolution = "resumed"
		}
		for _, in := range open {
			if in.Kind != "in_doubt" {
				continue
			}
			if left == 0 {
				if err := q.SendingResolveIncident(ctx, store.SendingResolveIncidentParams{ID: in.ID, At: &now, Resolution: resolution, ResolvedBy: ref}); err != nil {
					return err
				}
			} else if err := q.SendingUpdateIncidentDetail(ctx, store.SendingUpdateIncidentDetailParams{ID: in.ID, Detail: in.Detail, Held: left}); err != nil {
				return err
			}
		}
		if err := e.unpause(ctx, q, c, now); err != nil {
			return err
		}
		if !records(ctx) {
			return nil
		}
		action, summary := "campaign.in_doubt_left_out", english("audit.inDoubtLeftOut", i18n.Args{"count": n})
		if resend {
			action, summary = "campaign.in_doubt_resent", english("audit.inDoubtResent", i18n.Args{"count": n})
		}
		meta := map[string]any{"count": n, "summary": summary}
		if provider != nil {
			meta["provider_id"] = provider.String()
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: action, TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: e.actorFor(ctx, ref, name), After: map[string]any{"in_doubt": left}, Metadata: meta})
	})
	if err == nil {
		e.kick(ctx, campaignID)
	}
	return n, err
}

// unpause sends a paused campaign again after a decision on a share,
// unless something else keeps it paused: a person's pause, the canary
// waiting for a decision, or another campaign-wide incident. A decision on
// one share never undoes a person's pause (review GS-5).
func (e *Engine) unpause(ctx context.Context, q *store.Queries, c store.Campaign, now time.Time) error {
	if c.Status != "paused" {
		return nil
	}
	snd, err := q.SendingGetSend(ctx, c.ID)
	if err != nil {
		return err
	}
	if snd.PausedAt != nil {
		return nil
	}
	open, err := q.SendingOpenIncidents(ctx, c.ID)
	if err != nil {
		return err
	}
	for _, in := range open {
		if in.Scope == "campaign" && in.Kind != "in_doubt" && in.Kind != "render_error" {
			return nil
		}
	}
	return q.SendingSetCampaignStatus(ctx, store.SendingSetCampaignStatusParams{ID: c.ID, Status: "sending", At: now})
}

// settleOutboxStale settles the automation outbox's stale claims as
// settleStale does a campaign's. Automation emails have no monitor to
// decide on: an email in doubt stays in doubt (its flow's log shows it)
// unless its provider reports an event for it.
func (e *Engine) settleOutboxStale(ctx context.Context, now time.Time) error {
	stale := now.Add(-staleClaim)
	pids, err := e.Q.SendingOutboxStaleProviders(ctx, stale)
	if err != nil || len(pids) == 0 {
		return err
	}
	var ids []uuid.UUID
	var since []time.Time
	for _, pid := range pids {
		p, err := e.Q.ProvGetAny(ctx, pid)
		if err != nil || manage.IsDemo(p) || e.Factory == nil {
			continue
		}
		t, err := e.Factory.Open(ctx, p)
		if err != nil {
			continue
		}
		if at, ok := resendable(t, now); ok {
			ids, since = append(ids, pid), append(since, at)
		}
	}
	if len(ids) > 0 {
		if _, err := e.Q.SendingOutboxRequeueStale(ctx, store.SendingOutboxRequeueStaleParams{ProviderIds: ids, Since: since, Stale: stale}); err != nil {
			return err
		}
	}
	_, err = e.Q.SendingOutboxMarkInDoubt(ctx, store.SendingOutboxMarkInDoubtParams{Reason: inDoubtReason, Stale: stale})
	return err
}

// releaseClaims puts claimed messages that were never handed to a provider
// back in the queue.
func (e *Engine) releaseClaims(ctx context.Context, rows []store.SendingClaimRow) {
	if len(rows) == 0 {
		return
	}
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	if err := e.Q.SendingRelease(context.WithoutCancel(ctx), ids); err != nil {
		e.log().WarnContext(ctx, "release claimed messages", "err", err)
	}
}

// recordAttempts is how many times a batch's outcome is written before the
// send gives up on it (its messages then end in doubt).
const recordAttempts = 4

// recordOutcome writes what happened to a batch the provider answered for.
// It runs even when the job is told to stop, because the provider has the
// mail, and tries again after a database error.
func (e *Engine) recordOutcome(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	ctx = context.WithoutCancel(ctx)
	var err error
	for attempt := range recordAttempts {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 250 * time.Millisecond)
		}
		if err = sending.Tx(ctx, e.Pool, func(tx pgx.Tx) error { return fn(ctx, tx) }); err == nil {
			return nil
		}
		e.log().WarnContext(ctx, "record a batch's outcome", "attempt", attempt+1, "err", err)
	}
	return err
}

// InDoubtHook settles a message in doubt, or one left out while in doubt,
// when its provider reports an event for it: the provider took it after
// all. It counts as sent (rejected, when that is what the provider says;
// the event itself counts as usual before the hook); the in-doubt incident
// follows at the guard's next pass. Wired into the normalizer
// (internal/ingest).
func (e *Engine) InDoubtHook() ingest.MessageHook {
	return func(ctx context.Context, tx pgx.Tx, _ store.Provider, messageID uuid.UUID, _ string, ev providers.Event) error {
		status := map[providers.EventType]string{
			providers.EventDelivered: "delivered", providers.EventOpen: "delivered", providers.EventClick: "delivered", providers.EventUnsubscribe: "delivered",
			providers.EventDeferred: "deferred", providers.EventSoftBounce: "deferred", providers.EventHardBounce: "bounced",
			providers.EventComplaint: "complained", providers.EventRejected: "rejected",
		}[ev.Type]
		if status == "" {
			return nil
		}
		at := ev.OccurredAt
		if at.IsZero() {
			at = e.now()
		}
		q := store.New(tx)
		m, err := q.SendingSettleInDoubt(ctx, store.SendingSettleInDoubtParams{Status: status, At: at, ID: messageID})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // not in doubt: the usual case
		}
		if err != nil || m.CampaignID == nil || m.ProviderID == nil {
			return err
		}
		// The counters follow the message's state (internal/counters): a
		// rejected message was never sent.
		sent, rejected := int64(1), int64(0)
		if m.Status == "rejected" {
			sent, rejected = 0, 1
		}
		return q.SendingBumpCounters(ctx, store.SendingBumpCountersParams{CampaignID: *m.CampaignID, ProviderIds: []uuid.UUID{*m.ProviderID},
			Mailboxes: []string{m.MailboxProvider}, Variants: []string{m.Variant}, Sent: []int64{sent}, Rejected: []int64{rejected}})
	}
}
