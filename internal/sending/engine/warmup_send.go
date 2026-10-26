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

// outboxWarmLeft is how many automation emails a warming provider may
// still send today (math.MaxInt64 when it isn't warming up). When its cap
// or a pause stops them, they go through the Automations fallback if it is
// warm and can take them, else they wait for tomorrow's cap.
func (e *Engine) outboxWarmLeft(ctx context.Context, p store.Provider, now time.Time) (int64, error) {
	w := e.warmRouteFor(ctx, p, now)
	if w == nil {
		return math.MaxInt64, nil
	}
	left := w.left()
	if left > 0 || !w.a.Overflow {
		return left, nil
	}
	set, err := routing.Load(ctx, e.Q, p.WorkspaceID)
	if err != nil {
		return 0, err
	}
	def, ok := set.Default(providers.StreamAutomation)
	if !ok || def.Fallback == nil || *def.Fallback == p.ID {
		return 0, nil
	}
	fb, err := e.Q.ProvGetAny(ctx, *def.Fallback)
	if err != nil {
		return 0, nil //nolint:nilerr // no fallback any more: the mail waits for tomorrow's cap
	}
	var queued int64
	if err := e.Pool.QueryRow(ctx, `SELECT count(*) FROM messages WHERE provider_id = $1 AND status = 'queued' AND campaign_id IS NULL AND stream = 'automation'`, p.ID).Scan(&queued); err != nil {
		return 0, err
	}
	v := state.Vet(fb, state.Load{Messages: max(queued, 1), Warmup: warmup.Today(ctx, e.Q, fb.ID, now, nil)})
	if !v.Automatic(false) {
		return 0, nil
	}
	tag, err := e.Pool.Exec(ctx, `UPDATE messages SET provider_id = $2, claimed_at = NULL
		WHERE provider_id = $1 AND status = 'queued' AND campaign_id IS NULL AND stream = 'automation'`, p.ID, fb.ID)
	if err != nil {
		return 0, err
	}
	if n := tag.RowsAffected(); n > 0 {
		loc := state.Location("")
		if ws, err := e.Q.GetWorkspace(ctx, p.WorkspaceID); err == nil {
			loc = state.Location(ws.Timezone)
		}
		day := time.Date(now.In(loc).Year(), now.In(loc).Month(), now.In(loc).Day(), 0, 0, 0, 0, time.UTC)
		if err := e.Q.WarmupAddOverflow(ctx, store.WarmupAddOverflowParams{ID: w.a.PlanID, Day: day, N: n}); err != nil {
			return 0, err
		}
	}
	return 0, nil
}

// warmFull reports whether a warming provider's cap (or its paused plan)
// leaves no room today: transactional mail then goes to its fallback
// first. Product mail never waits for a warm-up, so with no fallback it
// still goes out (and counts toward the cap).
func (e *Engine) warmFull(ctx context.Context, p store.Provider, now time.Time) bool {
	w := e.warmRouteFor(ctx, p, now)
	return w != nil && w.left() <= 0
}
