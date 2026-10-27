package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/providers/bounce"
	"github.com/hostrepute/reputecast/internal/store"
)

// Thresholds by bounce type and deferral levels (review GS-11; Traffic
// rules › Thresholds by bounce type). Besides the overall bounce and
// complaint limits the guard watches:
//   - block/policy bounces (rejections for reputation, policy or
//     content) at each mailbox provider;
//   - unknown users (the other hard bounces: addresses or domains that
//     don't exist) over the whole send, at two levels;
//   - soft bounces and deferrals at each mailbox provider over a recent
//     window, at two levels: slow it to half rate, then pause it.
//
// Each level acts once on its scope until a person resumes the scope, and
// waits for the same evidence as the other thresholds: at least 5 events
// and 500 sent to the scope (within the window, for deferrals). A level
// that pauses goes through the auto-pause (an incident to decide on); one
// that slows down or warns leaves a notice to acknowledge.

// Actions a level takes.
const (
	actPauseMailbox = "pause_mailbox"
	actSlow         = "slow"
	actPauseSend    = "pause_send"
	actWarn         = "warn"
)

// validAction reports whether a stored action is one the guard knows.
func validAction(a string) bool {
	return a == actPauseMailbox || a == actSlow || a == actPauseSend || a == actWarn
}

// typedEvery is how often the guard reads bounce types and recent
// deferrals from the event log (the counters don't keep them).
const typedEvery = 10 * time.Second

// blockClass reports whether a hard bounce's class is a block or policy
// rejection.
func blockClass(class string) bool {
	return class == bounce.Reputation || class == bounce.SpamBlock || class == bounce.PolicyBlock
}

// typed are a send's hard bounces by type, and its deferrals and sends
// within the deferral windows, per mailbox provider.
type typed struct {
	at             time.Time
	block, unknown map[string]int64
	// deferred and sent within each window (minutes).
	deferred, sent map[int]map[string]int64
}

// typedTotals are a send's block/policy and other hard bounces per
// mailbox provider.
func typedTotals(ctx context.Context, q *store.Queries, campaignID uuid.UUID) (block, unknown map[string]int64, err error) {
	block, unknown = map[string]int64{}, map[string]int64{}
	rows, err := q.SendingBounceTypes(ctx, &campaignID)
	if err != nil {
		return nil, nil, err
	}
	for _, r := range rows {
		if blockClass(r.BounceClass) {
			block[r.MailboxProvider] += r.N
		} else {
			unknown[r.MailboxProvider] += r.N
		}
	}
	return block, unknown, nil
}

// readTyped reads what the typed thresholds need, at most every
// typedEvery.
func (e *Engine) readTyped(ctx context.Context, rs *runState, now time.Time) error {
	if !rs.typed.at.IsZero() && now.Sub(rs.typed.at) < typedEvery && !now.Before(rs.typed.at) {
		return nil
	}
	t, err := readTypedAt(ctx, e.Q, rs.campaign.ID, rs.thresholds, now)
	if err != nil {
		return err
	}
	rs.typed = t
	return nil
}

// readTypedAt reads a send's typed counts as of now.
func readTypedAt(ctx context.Context, q *store.Queries, campaignID uuid.UUID, th Thresholds, now time.Time) (typed, error) {
	t := typed{at: now, deferred: map[int]map[string]int64{}, sent: map[int]map[string]int64{}}
	var err error
	if t.block, t.unknown, err = typedTotals(ctx, q, campaignID); err != nil {
		return t, err
	}
	for _, w := range []int{th.DeferralSlowWindow, th.DeferralWindow} {
		if w <= 0 || t.sent[w] != nil {
			continue
		}
		since := now.Add(-time.Duration(w) * time.Minute)
		t.deferred[w], t.sent[w] = map[string]int64{}, map[string]int64{}
		drows, err := q.SendingDeferredSince(ctx, store.SendingDeferredSinceParams{CampaignID: &campaignID, Since: since})
		if err != nil {
			return t, err
		}
		for _, r := range drows {
			t.deferred[w][r.MailboxProvider] = r.N
		}
		srows, err := q.SendingSentSinceByMailbox(ctx, store.SendingSentSinceByMailboxParams{CampaignID: &campaignID, Since: &since})
		if err != nil {
			return t, err
		}
		for _, r := range srows {
			t.sent[w][r.MailboxProvider] = r.N
		}
	}
	return t, nil
}

// typedTrip is a typed level over its limit.
type typedTrip struct {
	trip
	level  string // the level's key in the guard state: block, unknown1, unknown2, deferral1, deferral2, complaint
	action string
}

// typedLevel is one level of a windowed deferral threshold.
type typedLevel struct {
	key    string
	limit  float64
	window int
	action string
}

// deferralLevels are the deferral levels, the second (pausing) first.
func deferralLevels(th Thresholds) []typedLevel {
	return []typedLevel{{"deferral2", th.Deferral, th.DeferralWindow, th.DeferralAction}, {"deferral1", th.DeferralSlow, th.DeferralSlowWindow, th.DeferralSlowAction}}
}

// checkTyped returns the first typed level over its limit that hasn't
// acted yet.
func (e *Engine) checkTyped(rs *runState, ts tallies, now time.Time) *typedTrip {
	th, ty := rs.thresholds, rs.typed
	for _, mbp := range sortedKeys(ts.mailbox) {
		if h, on := rs.holds[mbp]; on && h.whole() {
			continue
		}
		key := rearmMailbox(mbp)
		m := rs.guard.judged(key, *ts.mailbox[mbp])
		base := rs.guard.Rearm[key]
		block := max(ty.block[mbp]-base.Block, 0)
		if th.BlockBounce > 0 && !rs.guard.acted("block", mbp) && m.sent >= minSampleScoped && over(block, m.sent, th.BlockBounce, minEvents) {
			return &typedTrip{trip{scope: "mailbox_provider", mailbox: mbp, metric: "block_bounce", value: rate(block, m.sent), limit: th.BlockBounce, events: block, base: m.sent},
				"block", th.BlockAction}
		}
		for _, lv := range deferralLevels(th) {
			if lv.limit <= 0 || lv.window <= 0 || rs.guard.acted(lv.key, mbp) {
				continue
			}
			// A mailbox provider a person resumed is judged again once a
			// whole window passed since.
			if !base.At.IsZero() && now.Sub(base.At) < time.Duration(lv.window)*time.Minute {
				continue
			}
			d, s := ty.deferred[lv.window][mbp], ty.sent[lv.window][mbp]
			if s >= minSampleScoped && over(d, s, lv.limit, minDeferrals) {
				return &typedTrip{trip{scope: "mailbox_provider", mailbox: mbp, metric: "deferral", value: rate(d, s), limit: lv.limit, events: d, base: s, window: lv.window},
					lv.key, lv.action}
			}
		}
	}
	// Unknown users over the whole send, the second level first.
	var unknown int64
	for _, n := range ty.unknown {
		unknown += n
	}
	t := rs.guard.judged(rearmCampaign, ts.total)
	unknown = max(unknown-rs.guard.Rearm[rearmCampaign].Unknown, 0)
	for _, lv := range []typedLevel{{"unknown2", th.UnknownPause, 0, th.UnknownPauseAction}, {"unknown1", th.UnknownWarn, 0, th.UnknownWarnAction}} {
		if lv.limit <= 0 || rs.guard.acted(lv.key, "") {
			continue
		}
		if t.sent >= minSampleCampaign && over(unknown, t.sent, lv.limit, minEvents) {
			return &typedTrip{trip{scope: "campaign", metric: "unknown_user", value: rate(unknown, t.sent), limit: lv.limit, events: unknown, base: t.sent}, lv.key, lv.action}
		}
	}
	return nil
}

// acted reports whether a typed level acted on a scope already.
func (g guardState) acted(level, mbp string) bool {
	_, ok := g.Acted[level+":"+mbp]
	return ok
}

// markActed records that a typed level acted.
func (g *guardState) markActed(level, mbp string, now time.Time) {
	if g.Acted == nil {
		g.Acted = map[string]time.Time{}
	}
	g.Acted[level+":"+mbp] = now
}

// typedAct does what a typed level says: pause (the mailbox provider, or
// the send), slow down, or warn. It returns true when the whole send
// paused.
func (e *Engine) typedAct(ctx context.Context, rs *runState, ts tallies, tt typedTrip, now time.Time) (bool, error) {
	rs.guard.markActed(tt.level, tt.mailbox, now)
	if err := saveGuard(ctx, e.Q, rs.campaign.ID, rs.guard, now); err != nil {
		return false, err
	}
	tr := tt.trip
	switch tt.action {
	case actPauseMailbox, actPauseSend:
		if tt.action == actPauseSend || rs.thresholds.Scope == "campaign" {
			tr.scope = "campaign"
		}
		return tr.scope == "campaign", e.autoPause(ctx, rs, ts, tr, now)
	case actSlow:
		return false, e.slowDown(ctx, rs, ts, tt, now)
	}
	return false, e.notice(ctx, rs, ts, tt, "warned", 0, now)
}

// slowRate is half a mailbox provider's rate: its throttle, or its share
// of the open routes' rates.
func (rs *runState) slowRate(mbp string) float64 {
	if t := rs.throttles[mbp]; t > 0 {
		return math.Max(math.Round(t*5)/10, 0.1)
	}
	var v float64
	for _, pid := range rs.order {
		r := rs.routes[pid]
		if r == nil || !r.open() {
			continue
		}
		var all, mine int64
		for k, c := range rs.queue {
			if k.provider == pid {
				all += c.queued
				if k.mailbox == mbp {
					mine += c.queued
				}
			}
		}
		if all > 0 {
			v += r.rate.PerSecond * float64(mine) / float64(all)
		}
	}
	return math.Max(math.Round(v*5)/10, 0.1)
}

// slowDown slows the tripped scope to half rate for the rest of the send:
// a mailbox provider through its throttle, the whole send through its
// routes' rates.
func (e *Engine) slowDown(ctx context.Context, rs *runState, ts tallies, tt typedTrip, now time.Time) error {
	var to float64
	if tt.scope == "mailbox_provider" {
		to = rs.slowRate(tt.mailbox)
		th := map[string]float64{}
		for k, x := range rs.throttles {
			th[k] = x
		}
		th[tt.mailbox] = to
		b, _ := json.Marshal(th)
		if err := e.Q.SendingSetThrottles(ctx, store.SendingSetThrottlesParams{CampaignID: rs.campaign.ID, Throttles: b, At: now}); err != nil {
			return err
		}
		rs.throttles = th
	} else {
		if rs.guard.RouteRates == nil {
			rs.guard.RouteRates = map[string]float64{}
		}
		for _, pid := range rs.order {
			if r := rs.routes[pid]; r != nil && r.rate.PerSecond > 0 {
				half := math.Max(math.Round(r.rate.PerSecond*5)/10, 0.1)
				rs.guard.RouteRates[pid.String()] = half
				r.rate.PerSecond = half
				to += half
			}
		}
		if err := saveGuard(ctx, e.Q, rs.campaign.ID, rs.guard, now); err != nil {
			return err
		}
	}
	return e.notice(ctx, rs, ts, tt, "slowed", to, now)
}

// notice records what a typed level did when it didn't pause (slowed the
// scope down or warned): an incident to acknowledge, an audit entry and,
// for a warning, a notification.
func (e *Engine) notice(ctx context.Context, rs *runState, ts tallies, tt typedTrip, action string, to float64, now time.Time) error {
	c := rs.campaign
	scopeName := "the campaign"
	if tt.scope == "mailbox_provider" {
		scopeName = mailboxName(tt.mailbox)
	}
	words := headlineArgs(tt.trip, scopeName)
	words["name"], words["action"], words["mailbox"] = quoted(c.Name), action, tt.mailbox
	words["windowed"], words["windowMin"] = yesNo(tt.window > 0), int64(tt.window)
	words["events"], words["base"], words["slowRate"] = tt.events, tt.base, to
	words["valuePct"], words["limitPct"] = round2(tt.value), tt.limit
	d := e.analyzeThreshold(ctx, rs, ts, tt.trip, 0, now)
	d.Action, d.SlowedTo, d.Window = action, to, tt.window
	d.Causes, d.Recommendations, d.Options = nil, nil, nil
	if d.texts != nil {
		d.texts.Causes, d.texts.Recommendations, d.texts.Options = nil, nil, nil
	}
	d.addCause(keep("cause.typedLead", words), keep("cause.typed", words))
	d.addRecommendation(keep("rec."+tt.metric, words))
	d.setOptions([]option{{key: "acknowledge", recommended: true, title: keep("opt.acknowledgeTitle", nil), body: keep("opt.acknowledgeNotice", words), action: keep("opt.acknowledge", nil)}})
	value, limit := round4(tt.value), round4(tt.limit)
	return db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		id := newID()
		if err := q.SendingInsertIncident(ctx, store.SendingInsertIncidentParams{ID: id, CampaignID: c.ID, Kind: "threshold", Scope: tt.scope, MailboxProvider: tt.mailbox,
			Metric: tt.metric, Value: &value, Threshold: &limit, Held: 0, Detail: d.stored(), TrippedAt: now}); err != nil {
			return err
		}
		if action == "warned" {
			if err := notify.ToMembersWith(ctx, tx, c.WorkspaceID, authz.CampaignsSend, notify.Notification{
				Kind: notify.KindSendPaused, Severity: "warn", TitleText: keep("notify.typedTitle", words), BodyText: keep("notify.typedBody", words),
				Link: liveLink(c.WorkspaceSlug, c.ID), Data: map[string]any{"campaignId": c.ID.String(), "incidentId": id.String()},
			}); err != nil {
				return err
			}
		}
		summary := fmt.Sprintf("%s %s over limit %s · %s of %s · %s", metricName(tt.metric), fmtRate(tt.value), fmtRate(tt.limit), countText(tt.events), countText(tt.base),
			map[string]string{"slowed": "slowed to half rate", "warned": "warning only"}[action])
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.threshold_" + action, TargetType: "campaign", TargetID: c.ID.String(),
			TargetName: c.Name, Reason: thresholdHeadline(tt.trip, scopeName), Actor: systemActor, Metadata: map[string]any{"scope": tt.scope, "mailbox_provider": tt.mailbox,
				"metric": tt.metric, "value": value, "limit": limit, "level": tt.level, "slowed_to": to, "summary": summary, "source": "guard", "via": "threshold"}})
	})
}

// notices are the incidents a person acknowledges rather than decides on:
// complaints after the send, typed levels that slowed down or warned,
// reply rules that only alert, template errors (CO-4).
func isNotice(in store.CampaignIncident) bool {
	switch in.Kind {
	case "complaint_watch", "render_error":
		return true
	case "threshold", "reply_rule":
		a := parseDetail(in.Detail).Action
		return a == "slowed" || a == "warned" || a == "alerted"
	}
	return false
}
