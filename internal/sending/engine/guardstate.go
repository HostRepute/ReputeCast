package engine

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// guardState is campaign_sends.guard: what the guard keeps about a send
// besides its thresholds and holds.
type guardState struct {
	// Rearm is where each resumed scope's thresholds start again (review
	// GS-2): the scope's counts when a person resumed it. Keys are
	// "campaign", "mailbox:<key>" and "route:<provider id>". A resumed scope
	// is judged on what happened since, with the evidence a fresh one
	// needs, so old events never pause it again.
	Rearm map[string]counts `json:"rearm,omitempty"`
	// RouteRates are sending providers slowed for this send when a person
	// resumed them with changes, in messages per second.
	RouteRates map[string]float64 `json:"routeRates,omitempty"`
}

// counts are a scope's counters at a moment.
type counts struct {
	Sent       int64 `json:"sent"`
	Delivered  int64 `json:"delivered"`
	Deferred   int64 `json:"deferred"`
	Hard       int64 `json:"hard"`
	Soft       int64 `json:"soft"`
	Complained int64 `json:"complained"`
}

func countsOf(t tally) counts {
	return counts{Sent: t.sent, Delivered: t.delivered, Deferred: t.deferred, Hard: t.hard, Soft: t.soft, Complained: t.complained}
}

// plus adds a scope's counts (a scope that joins the campaign-wide check
// again: its events so far are left out of it).
func (c counts) plus(t tally) counts {
	c.Sent += t.sent
	c.Delivered += t.delivered
	c.Deferred += t.deferred
	c.Hard += t.hard
	c.Soft += t.soft
	c.Complained += t.complained
	return c
}

// since is the tally without what the base had counted (never below zero).
func (t tally) since(b counts) tally {
	sub := func(a, b int64) int64 { return max(a-b, 0) }
	t.sent, t.delivered, t.deferred = sub(t.sent, b.Sent), sub(t.delivered, b.Delivered), sub(t.deferred, b.Deferred)
	t.hard, t.soft, t.complained = sub(t.hard, b.Hard), sub(t.soft, b.Soft), sub(t.complained, b.Complained)
	return t
}

func parseGuard(raw []byte) guardState {
	var g guardState
	_ = jsonUnmarshal(raw, &g)
	return g
}

func (g *guardState) rearm(key string, c counts) {
	if g.Rearm == nil {
		g.Rearm = map[string]counts{}
	}
	g.Rearm[key] = c
}

// rejoin leaves a scope's events so far out of the campaign-wide check,
// which judges it again from now on.
func (g *guardState) rejoin(t tally) {
	g.rearm(rearmCampaign, g.Rearm[rearmCampaign].plus(t))
}

const rearmCampaign = "campaign"

func rearmMailbox(mbp string) string      { return "mailbox:" + mbp }
func rearmRoute(pid uuid.UUID) string     { return "route:" + pid.String() }
func (g guardState) raw() json.RawMessage { b, _ := json.Marshal(g); return b }

// routeRate is a route's rate for this send when a person lowered it.
func (g guardState) routeRate(pid uuid.UUID) float64 { return g.RouteRates[pid.String()] }

// saveGuard writes a send's guard state.
func saveGuard(ctx context.Context, q *store.Queries, campaignID uuid.UUID, g guardState, now time.Time) error {
	return q.SendingSetGuard(ctx, store.SendingSetGuardParams{CampaignID: campaignID, Guard: g.raw(), At: now})
}

// judged is a scope's tally as its thresholds judge it: since a person
// resumed it, when they did.
func (g guardState) judged(key string, t tally) tally {
	if b, ok := g.Rearm[key]; ok {
		return t.since(b)
	}
	return t
}
