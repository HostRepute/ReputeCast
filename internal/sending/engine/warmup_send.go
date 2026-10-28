package engine

import (
	"context"
	"encoding/json"
	"math"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/routing"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/warmup"
)

// Warm-up in the send loop (review PR-8, C5). A warming provider sends no
// more than today's cap (the workspace's day, every stream counting), per
// mailbox provider when its plan caps them separately, nothing while its
// plan is paused, and its share goes out most engaged first. When the cap
// stops mail that is still queued, the rest goes through a warm provider
// of the same route when the plan allows it (overflow), or waits for
// tomorrow's cap as a hold with choices (wait, move, cancel).

// CapHoldReason is a route's status detail reason while today's warm-up
// cap holds its share.
const CapHoldReason = "warmup_cap"

// capHoldDetail is a route's status detail while the warm-up holds its
// share: until tomorrow's cap starts, or while the plan is paused.
type capHoldDetail struct {
	Reason string     `json:"reason"`
	Until  *time.Time `json:"until,omitempty"`
	Paused bool       `json:"paused,omitempty"`
}

// warmRoute is a warming route's allowance today, as the loop spends it.
type warmRoute struct {
	a warmup.Allowance
	// spent is what this process claimed since the allowance was read,
	// per mailbox bucket: messages count in the database only once sent.
	spent    map[string]int64
	spentAll int64
}

// warmRouteFor reads a provider's warm-up allowance; nil when it isn't
// warming up.
func (e *Engine) warmRouteFor(ctx context.Context, p store.Provider, now time.Time) *warmRoute {
	a, ok, err := warmup.AllowanceFor(ctx, e.Q, p.ID, now, nil)
	if err != nil || !ok {
		return nil
	}
	return &warmRoute{a: a, spent: map[string]int64{}}
}

// left is what the route may still send today, overall.
func (w *warmRoute) left() int64 {
	if w == nil {
		return math.MaxInt64
	}
	return max(w.a.Left()-w.spentAll, 0)
}

// mailboxLeft is what the route may still send today to one mailbox
// provider.
func (w *warmRoute) mailboxLeft(mbp string) int64 {
	if w == nil {
		return math.MaxInt64
	}
	return max(min(w.a.MailboxLeft(mbp)-w.spent[warmup.MailboxOf(mbp)], w.left()), 0)
}

// clamp limits one tick's allocation to what is left of today's cap, per
// mailbox provider. A nil route (not warming up) changes nothing.
func (w *warmRoute) clamp(alloc map[string]int) map[string]int {
	if w == nil {
		return alloc
	}
	total := w.left()
	out := map[string]int{}
	for _, mbp := range sortedKeys(alloc) {
		n := min(int64(alloc[mbp]), w.mailboxLeft(mbp), total)
		if n > 0 {
			out[mbp] = int(n)
			total -= n
		}
	}
	return out
}

// allocateWarm is allocate for a route that may be warming up: mailbox
// providers already at their part of today's cap are left out (so a tick
// never goes to one that can't take it while others can) and none gets
// more than what is left of its part.
func (e *Engine) allocateWarm(rs *runState, r *runRoute, want int) map[string]int {
	if r.warm == nil {
		return e.allocate(rs, r, want)
	}
	open := make(map[queueKey]queueCell, len(rs.queue))
	for k, c := range rs.queue {
		if k.provider == r.provider.ID && r.warm.mailboxLeft(k.mailbox) <= 0 {
			continue
		}
		open[k] = c
	}
	view := *rs
	view.queue = open
	return r.warm.clamp(e.allocate(&view, r, want))
}

// sent counts what a tick claimed toward today's cap.
func (w *warmRoute) sent(rows []store.SendingClaimRow) {
	if w == nil {
		return
	}
	for _, r := range rows {
		w.spent[warmup.MailboxOf(r.MailboxProvider)]++
		w.spentAll++
	}
}

// warmStopped reports whether today's cap (or a pause) stops mail of the
// route that is still queued: nothing is left overall, or every mailbox
// provider with mail queued is at its own cap.
func (e *Engine) warmStopped(rs *runState, r *runRoute) bool {
	w := r.warm
	if w == nil {
		return false
	}
	queued := int64(0)
	open := false
	for k, c := range rs.queue {
		if k.provider != r.provider.ID {
			continue
		}
		n := c.queued - c.waiting
		if n <= 0 {
			continue
		}
		queued += n
		if w.mailboxLeft(k.mailbox) > 0 {
			open = true
		}
	}
	if queued == 0 {
		return false
	}
	return w.a.Paused || w.left() <= 0 || !open
}

// warmCapReached handles a route stopped by today's warm-up cap or a
// paused plan: the rest goes through a warm provider of the same route when
// the plan allows it and one can take it (state.Vet), as the audience build
// does, else the share waits for tomorrow's cap (or the plan's resume) as a
// hold with choices.
func (e *Engine) warmCapReached(ctx context.Context, rs *runState, r *runRoute, now time.Time) error {
	if r.warm == nil || r.status == "held" {
		return nil
	}
	cid, pid := rs.campaign.ID, r.provider.ID
	if r.warm.a.Overflow {
		if target := e.overflowTarget(ctx, rs, r, now); target != nil {
			return e.overflow(ctx, rs, r, *target, now)
		}
	}
	return sending.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var held int64
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM messages WHERE campaign_id = $1 AND provider_id = $2 AND status = 'queued'`, cid, pid).Scan(&held); err != nil {
			return err
		}
		// The cap holds the share until tomorrow's cap; a paused plan holds
		// it until someone resumes the plan (warmup.ResumeCapHolds).
		reason := "cap"
		var until *time.Time
		if r.warm.a.Paused {
			reason = "paused"
		} else {
			next := r.warm.a.NextDay
			until = &next
		}
		detail, _ := json.Marshal(capHoldDetail{Reason: CapHoldReason, Until: until, Paused: r.warm.a.Paused})
		if _, err := q.ProvSetRouteStatus(ctx, store.ProvSetRouteStatusParams{CampaignID: cid, ProviderID: pid, Status: "held", StatusDetail: string(detail)}); err != nil {
			return err
		}
		inc, _ := json.Marshal(map[string]any{"reason": reason, "cap": r.warm.a.Cap, "sent": r.warm.a.Sent + r.warm.spentAll, "day": r.warm.a.Day, "resume_at": until})
		if _, err := tx.Exec(ctx, `INSERT INTO campaign_incidents (id, campaign_id, kind, scope, provider_id, metric, held, detail, tripped_at)
			VALUES ($1, $2, 'throttled', 'route', $3, $4, $5, $6, $7)`, newID(), cid, pid, CapHoldReason, held, inc, now); err != nil {
			return err
		}
		short := state.Short(r.provider.Name, r.provider.Adapter)
		resume := r.warm.a.NextDay
		title, body := warmup.CapHoldNotice(rs.campaign.Name, short, held, reason, resume.In(rs.loc))
		if err := notify.ToMembersWith(ctx, tx, rs.campaign.WorkspaceID, authz.CampaignsSend, notify.Notification{
			Kind: notify.KindSendPaused, Severity: "warn", Title: title.In(i18n.Default), TitleText: title, Body: body.In(i18n.Default), BodyText: body,
			Link: "/w/" + rs.campaign.WorkspaceSlug + "/providers/warmup", Data: map[string]any{"campaignId": cid.String(), "providerId": pid.String(), "reason": CapHoldReason},
		}); err != nil {
			return err
		}
		r.status, r.detail = "held", routeDetail{Reason: CapHoldReason, Until: until}
		return nil
	})
}

// warmWaiting is how much of the queue waits on routes a warm-up cap (or a
// paused plan) holds: it goes on by itself (warmup.ResumeCapHolds), so the
// campaign keeps sending instead of waiting for a decision.
func (e *Engine) warmWaiting(rs *runState) int64 {
	var n int64
	for k, c := range rs.queue {
		if r := rs.routes[k.provider]; r != nil && r.status == "held" && r.detail.Reason == CapHoldReason && r.detail.Decision != "move" && r.detail.Decision != "cancel" {
			n += c.queued
		}
	}
	return n
}

// overflowTarget is the warm provider over-cap mail goes through: the
// largest share of the same Broadcasts split that isn't warming up and can
// take what's queued (state.Vet with the campaign's sender).
func (e *Engine) overflowTarget(ctx context.Context, rs *runState, r *runRoute, now time.Time) *store.Provider {
	set, err := routing.Load(ctx, e.Q, rs.campaign.WorkspaceID)
	if err != nil {
		return nil
	}
	def, ok := set.Default(providers.StreamBroadcast)
	if !ok {
		return nil
	}
	var queued int64
	for k, c := range rs.queue {
		if k.provider == r.provider.ID {
			queued += c.queued
		}
	}
	sender := senderVerification(ctx, e.Q, rs.campaign.WorkspaceID, rs.campaign.SenderID)
	var best *store.Provider
	bestW := -1
	for _, sp := range def.Splits {
		if sp.ProviderID == r.provider.ID || sp.Weight <= bestW {
			continue
		}
		p, err := e.Q.ProvGetAny(ctx, sp.ProviderID)
		if err != nil || p.WorkspaceID != rs.campaign.WorkspaceID {
			continue
		}
		v := e.vetTarget(ctx, e.Q, sender, p, queued, rs.loc)
		if !v.Automatic(false) {
			continue
		}
		best, bestW = &p, sp.Weight
	}
	return best
}

// overflow moves the route's queued mail to a warm provider: its messages,
// the planned counts of both routes, and today's overflow count of the
// plan ("Over the cap so far today").
func (e *Engine) overflow(ctx context.Context, rs *runState, r *runRoute, target store.Provider, now time.Time) error {
	cid, pid := rs.campaign.ID, r.provider.ID
	note := routing.RouteNote("instead", i18n.Args{"alt": state.Short(target.Name, target.Adapter), "provider": state.Short(r.provider.Name, r.provider.Adapter),
		"why": "cap", "tier": "", "day": r.warm.a.Day})
	return sending.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE messages SET provider_id = $3, route_note = $4, claimed_at = NULL
			WHERE campaign_id = $1 AND provider_id = $2 AND status = 'queued' AND (claimed_at IS NULL OR claimed_at < $5)`, cid, pid, target.ID, note, now.Add(-staleClaim))
		if err != nil {
			return err
		}
		moved := tag.RowsAffected()
		if moved == 0 {
			return nil
		}
		if _, err := tx.Exec(ctx, `INSERT INTO campaign_routes (campaign_id, provider_id, planned, status) VALUES ($1, $2, $3, 'pending')
			ON CONFLICT (campaign_id, provider_id) DO UPDATE SET planned = campaign_routes.planned + EXCLUDED.planned,
			status = CASE WHEN campaign_routes.status = 'done' THEN 'pending' ELSE campaign_routes.status END`, cid, target.ID, moved); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE campaign_routes SET planned = greatest(planned - $3, 0) WHERE campaign_id = $1 AND provider_id = $2`, cid, pid, moved); err != nil {
			return err
		}
		day := time.Date(now.In(rs.loc).Year(), now.In(rs.loc).Month(), now.In(rs.loc).Day(), 0, 0, 0, 0, time.UTC)
		return store.New(tx).WarmupAddOverflow(ctx, store.WarmupAddOverflowParams{ID: r.warm.a.PlanID, Day: day, N: moved})
	})
}

// sendOrder is a message's place in the send: random, so the first
// minutes are a fair sample of the audience, except on a warming provider
// whose plan sends to the most engaged first: Active, then Cooling,
// contacts without a tier, At risk and Sunset, random within each tier.
func (r *router) sendOrder(ctx context.Context, pid uuid.UUID, tier string) int32 {
	p, ok := r.byID[pid]
	if !ok {
		return rand.Int32() //nolint:gosec // an order, not a secret
	}
	if pr := r.rate(ctx, p); !pr.WarmingUp || !pr.EngagedFirst {
		return rand.Int32() //nolint:gosec // an order, not a secret
	}
	return int32(tierRank(tier))<<28 | rand.Int32N(1<<28) //nolint:gosec // an order, not a secret
}

// tierRank orders engagement tiers, most engaged first.
func tierRank(tier string) int {
	switch tier {
	case "active":
		return 0
	case "cooling":
		return 1
	case "at_risk":
		return 3
	case "sunset":
		return 4
	}
	return 2
}

// outboxWarm is a warming provider's allowance for automation emails (nil
// when it isn't warming up) and the mailbox providers whose part of today's
// cap is used up (their mail waits, SendingOutboxClaim). Automation mail
// counts toward the cap and respects a paused plan: what the cap or the
// pause stops stays queued, never dropped, until tomorrow's cap or the
// plan's resume, unless the plan sends what's over the cap through a warm
// provider and the Automations fallback is warm and can take it (PR-8,
// review R3.L6b). Over a mailbox provider's part of the cap, only that
// mailbox provider's mail moves.
func (e *Engine) outboxWarm(ctx context.Context, p store.Provider, now time.Time) (*warmRoute, []string, error) {
	full := []string{}
	w := e.warmRouteFor(ctx, p, now)
	if w == nil {
		return nil, full, nil
	}
	for _, sh := range warmup.MailboxShares {
		if w.mailboxLeft(sh.Mailbox) <= 0 {
			full = append(full, sh.Mailbox)
		}
	}
	if len(full) == 0 || !w.a.Overflow {
		return w, full, nil
	}
	set, err := routing.Load(ctx, e.Q, p.WorkspaceID)
	if err != nil {
		return nil, nil, err
	}
	def, ok := set.Default(providers.StreamAutomation)
	if !ok || def.Fallback == nil || *def.Fallback == p.ID {
		return w, full, nil
	}
	fb, err := e.Q.ProvGetAny(ctx, *def.Fallback)
	if err != nil {
		return w, full, nil //nolint:nilerr // no fallback any more: the mail waits for tomorrow's cap
	}
	var queued int64
	if err := e.Pool.QueryRow(ctx, `SELECT count(*) FROM messages WHERE provider_id = $1 AND status = 'queued' AND campaign_id IS NULL AND stream = 'automation'
		AND `+mailboxBucket+` = ANY($2)`, p.ID, full).Scan(&queued); err != nil {
		return nil, nil, err
	}
	if queued == 0 {
		return w, full, nil
	}
	// Vetted with the stream's senders, as a failover is: without them the
	// sender's domain is unknown there and nothing moves by itself.
	v := state.Vet(fb, state.Load{Messages: max(queued, 1), Sender: manage.StreamSenders(ctx, e.Q, p.WorkspaceID, providers.StreamAutomation),
		Warmup: warmup.Today(ctx, e.Q, fb.ID, now, nil)})
	if !v.Automatic(false) {
		return w, full, nil
	}
	why := "cap"
	if w.a.Paused {
		why = "paused"
	}
	note := routing.RouteNote("instead", i18n.Args{"alt": state.Short(fb.Name, fb.Adapter), "provider": state.Short(p.Name, p.Adapter), "why": why, "tier": "", "day": w.a.Day})
	err = sending.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE messages SET provider_id = $2, claimed_at = NULL, route_note = $3
			WHERE provider_id = $1 AND status = 'queued' AND campaign_id IS NULL AND stream = 'automation' AND claimed_at IS NULL
			AND `+mailboxBucket+` = ANY($4)`, p.ID, fb.ID, note, full)
		if err != nil {
			return err
		}
		if n := tag.RowsAffected(); n > 0 {
			return store.New(tx).WarmupAddOverflow(ctx, store.WarmupAddOverflowParams{ID: w.a.PlanID, Day: w.today(), N: n})
		}
		return nil
	})
	return w, full, err
}

// mailboxBucket is a message's part of a warming provider's cap in SQL:
// its mailbox provider when the plan caps it separately, else "other"
// (warmup.MailboxOf; SendingOutboxClaim reads it the same way).
const mailboxBucket = `(CASE WHEN mailbox_provider IN ('gmail', 'outlook', 'yahoo', 'icloud') THEN mailbox_provider ELSE 'other' END)`

// today is the workspace's day the allowance counts, as a date.
func (w *warmRoute) today() time.Time {
	d := w.a.NextDay.AddDate(0, 0, -1)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

// take counts a message toward today's cap when its mailbox provider's part
// (or the cap) has room, and reports whether it had.
func (w *warmRoute) take(mbp string) bool {
	if w == nil {
		return true
	}
	if w.mailboxLeft(mbp) <= 0 {
		return false
	}
	w.spent[warmup.MailboxOf(mbp)]++
	w.spentAll++
	return true
}

// Transactional mail and the warm-up (review R3.L6b). A password reset never
// waits for a warm-up. A warming provider sends transactional mail within
// today's cap (per mailbox provider when its plan caps them separately),
// counting toward it like every stream; over the cap, or while its plan is
// paused, a message goes through a warm provider of its route that can take
// it, else through the warming provider anyway, counted as over today's cap
// (WarmupAddOverCap) for the warm-up page and the provider card to show.

// txWarm is a warming provider's transactional mail in one tick.
type txWarm struct {
	w *warmRoute
	// over are the claimed messages that go out over today's cap.
	over map[uuid.UUID]bool
}

// overSent counts the messages sent over today's cap.
func (t *txWarm) overSent(sent []uuid.UUID) int64 {
	if t == nil {
		return 0
	}
	var n int64
	for _, id := range sent {
		if t.over[id] {
			n++
		}
	}
	return n
}

// warmTransactional sorts a tick's claimed transactional messages on a
// warming provider: within today's cap they stay; over it a warm provider
// of their route takes them (moved now, their claim released); with none,
// they stay and go out over the cap. Not warming up: every message stays.
func (o *TransactionalOutbox) warmTransactional(ctx context.Context, p store.Provider, rows []store.TxnOutboxClaimRow, now time.Time) ([]store.TxnOutboxClaimRow, *txWarm, error) {
	w := o.E.warmRouteFor(ctx, p, now)
	if w == nil {
		return rows, nil, nil
	}
	tw := &txWarm{w: w, over: map[uuid.UUID]bool{}}
	keep := make([]store.TxnOutboxClaimRow, 0, len(rows))
	type move struct {
		id uuid.UUID
		to store.Provider
	}
	var moves []move
	for _, row := range rows {
		if w.take(row.MailboxProvider) {
			keep = append(keep, row)
			continue
		}
		if to := o.warmAlternative(ctx, p, row.Route, row.Tried, now); to != nil {
			moves = append(moves, move{row.ID, *to})
			continue
		}
		tw.over[row.ID] = true
		keep = append(keep, row)
	}
	if len(moves) == 0 {
		return keep, tw, nil
	}
	why := "cap"
	if w.a.Paused {
		why = "paused"
	}
	err := sending.Tx(ctx, o.E.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for _, m := range moves {
			to := m.to.ID
			note := routing.RouteNote("instead", i18n.Args{"alt": state.Short(m.to.Name, m.to.Adapter), "provider": state.Short(p.Name, p.Adapter), "why": why, "tier": "", "day": w.a.Day})
			if err := q.TxnMoveForWarmup(ctx, store.TxnMoveForWarmupParams{ID: m.id, ProviderID: &to, RouteNote: note}); err != nil {
				return err
			}
			if err := q.TxnSetProviderSince(ctx, store.TxnSetProviderSinceParams{MessageID: m.id, At: now}); err != nil {
				return err
			}
		}
		return nil
	})
	return keep, tw, err
}

// warmAlternative is the warm provider of a transactional message's route
// that takes it while its warming provider can't today: the route's other
// provider when its failover policy lets mail move by itself, the message
// didn't give up on it, it isn't warming up itself and it can take the
// message (state.Vet, as a failover vets it).
func (o *TransactionalOutbox) warmAlternative(ctx context.Context, current store.Provider, routeRaw, triedRaw json.RawMessage, now time.Time) *store.Provider {
	var rs txRouteSnap
	_ = json.Unmarshal(routeRaw, &rs)
	if rs.Policy == string(routing.PauseAndAsk) {
		return nil
	}
	var tried []attempt
	_ = json.Unmarshal(triedRaw, &tried)
	gaveUp := map[string]bool{}
	for _, a := range tried {
		if a.GaveUp {
			gaveUp[a.ProviderID] = true
		}
	}
	cands := []uuid.UUID{rs.Primary}
	if rs.Fallback != nil {
		cands = append(cands, *rs.Fallback)
	}
	for _, id := range cands {
		if id == uuid.Nil || id == current.ID || gaveUp[id.String()] {
			continue
		}
		p, err := o.E.Q.ProvGetAny(ctx, id)
		if err != nil || p.WorkspaceID != current.WorkspaceID || p.DisconnectedAt != nil || !state.Available(p) {
			continue
		}
		if o.E.warmRouteFor(ctx, p, now) != nil {
			continue // warming up too
		}
		if !state.Vet(p, state.Load{Messages: 1}).Allowed() {
			continue
		}
		return &p
	}
	return nil
}
