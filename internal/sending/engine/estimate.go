package engine

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/warmup"
)

// Estimates (review SR-5, contract C4): how long a send takes and when each
// part goes out, played out with the engine's own rules instead of a copy
// of them: each provider's rate (its published maximum, the adapter's
// default when it publishes none, or the warm-up plan's ramp), the
// warm-up plan's cap day after day, the provider's 24-hour quota, the
// per-mailbox-provider throttles shared by every route, the A/B test's
// sample and wait, best-time spreading and routes sending in parallel. The
// send loop's allocation and throttle refill are the same functions.

// EstimateOptions describe the send an estimate is for. Send Readiness
// counts the recipients; the engine knows how it sends them.
type EstimateOptions struct {
	// Start is when the send starts (now when zero).
	Start time.Time
	// Recipients counts the recipients per provider and mailbox provider.
	Recipients map[uuid.UUID]map[string]int64
	// Throttles are Send Readiness's per-mailbox-provider throttles for the
	// send; nil keeps the workspace's and the defaults (sendThrottles).
	Throttles map[string]float64
}

// Estimate is how the engine would send a campaign.
type Estimate struct {
	Start time.Time `json:"start"`
	// End is when the last message goes out. Open is true when it isn't
	// known: a person picks the A/B winner, so the rest waits for them.
	End  time.Time `json:"end"`
	Open bool      `json:"open,omitempty"`
	// Providers are the routes, in the order of the recipients given.
	Providers []EstimateProvider `json:"providers"`
	// Throttles are the per-mailbox-provider throttles the send uses
	// (messages per second; absent means the provider's full rate).
	Throttles map[string]float64 `json:"throttles"`
	// AB is the A/B test's timeline, nil without a test.
	AB *EstimateAB `json:"ab,omitempty"`
	// BestTime is true when each recipient gets it at their usual open
	// hour, within 24 hours of the start.
	BestTime bool `json:"bestTime,omitempty"`
}

// EstimateProvider is one route of the send.
type EstimateProvider struct {
	ProviderID uuid.UUID `json:"providerId"`
	Recipients int64     `json:"recipients"`
	// Rate is the messages per second the engine sends at; RateFrom is
	// where it comes from: published (the provider's own maximum),
	// default (the adapter's, when the provider publishes none) or warmup
	// (the warm-up plan's ramp).
	Rate     float64 `json:"rate"`
	RateFrom string  `json:"rateFrom" enum:"published,default,warmup"`
	// Seconds is how long the route sends (waits left out); Done is when
	// its last message goes out.
	Seconds float64   `json:"seconds"`
	Done    time.Time `json:"done"`
	// Days is how many days the warm-up plan's caps spread the share over
	// (1: it all goes on the start day); CapLeft is what the plan allows
	// on the start day (nil without a plan).
	Days    int    `json:"days"`
	CapLeft *int64 `json:"capLeft,omitempty"`
	// QuotaLeft is the provider's 24-hour quota left (nil without one);
	// OverQuota is how many messages wait for the quota to free up.
	QuotaLeft *int64 `json:"quotaLeft,omitempty"`
	OverQuota int64  `json:"overQuota,omitempty"`
	// Held is true when the provider can't take mail now: its share waits
	// (Send Readiness offers moving it).
	Held bool `json:"held,omitempty"`
}

// EstimateAB is an A/B test's timeline: the sample goes first, the rest
// when the winner is picked.
type EstimateAB struct {
	Kind        string `json:"kind"`
	Variants    int    `json:"variants"`
	TestPercent int    `json:"testPercent"`
	Sample      int64  `json:"sample"`
	// SampleEnd is when the sample's last message goes out; DecideAt when
	// the winner is picked (nil when a person picks it).
	SampleEnd time.Time  `json:"sampleEnd"`
	DecideAt  *time.Time `json:"decideAt,omitempty"`
	Hours     int        `json:"hours"`
}

// estimateHorizon bounds an estimate: past it, a send is "more than 60
// days" (a warm-up plan far too small for the audience).
const estimateHorizon = 60 * 24 * time.Hour

// Estimate plays a campaign's send out with the engine's rules.
func (e *Engine) Estimate(ctx context.Context, campaignID uuid.UUID, o EstimateOptions) (Estimate, error) {
	cc, err := e.Q.SendingCampaignContext(ctx, campaignID)
	if err != nil {
		return Estimate{}, err
	}
	ws, err := e.Q.GetWorkspace(ctx, cc.WorkspaceID)
	if err != nil {
		return Estimate{}, err
	}
	start := o.Start
	if start.IsZero() {
		start = e.now()
	}
	loc := state.Location(cc.WorkspaceTimezone)
	plan := simPlan{start: start, loc: loc, throttles: sendThrottles(ws.SendingDefaults, o.Throttles)}
	var sc schedule
	_ = json.Unmarshal(cc.Schedule, &sc)
	plan.bestTime = sc.Mode == "best_time"
	ids := make([]uuid.UUID, 0, len(o.Recipients))
	for id := range o.Recipients {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return cellTotal(o.Recipients[ids[i]]) > cellTotal(o.Recipients[ids[j]]) })
	for _, id := range ids {
		p, err := e.Q.ProvGet(ctx, store.ProvGetParams{WorkspaceID: cc.WorkspaceID, ID: id})
		if err != nil {
			return Estimate{}, err
		}
		plan.routes = append(plan.routes, e.simRouteOf(ctx, p, o.Recipients[id], start, loc))
	}
	if ab := abTestOf(cc.AbTest); ab != nil && ab.Winner == "" && ab.TestPercent > 0 && ab.TestPercent < 100 && len(ab.Variants) >= 2 {
		plan.ab = &EstimateAB{Kind: ab.Kind, Variants: len(ab.Variants), TestPercent: ab.TestPercent, Hours: max(ab.DurationHours, 1)}
		if ab.Metric != "manual" {
			at := start.Add(time.Duration(plan.ab.Hours) * time.Hour)
			plan.ab.DecideAt = &at
		}
	}
	return e.simulate(plan), nil
}

func cellTotal(m map[string]int64) int64 {
	var n int64
	for _, v := range m {
		n += v
	}
	return n
}

// simPlan is a send as an estimate plays it out.
type simPlan struct {
	start     time.Time
	loc       *time.Location
	routes    []*simRoute
	throttles map[string]float64
	ab        *EstimateAB
	bestTime  bool
}

// simRoute is one provider's share in an estimate.
type simRoute struct {
	provider store.Provider
	rate     float64
	rateFrom string
	held     bool
	cells    map[string]int64
	// allowance is what the warm-up plan allows on the local day of t
	// (nil without a plan).
	allowance func(t time.Time) int64
	sentOn    map[string]int64
	quotaLeft *int64
	quotaMax  int64
	carry     float64
	out       EstimateProvider
}

// simRouteOf reads a provider's rate, warm-up plan and quota as the send
// loop does (rateOf, warmup.AllowanceFor, the provider's health).
func (e *Engine) simRouteOf(ctx context.Context, p store.Provider, cells map[string]int64, start time.Time, loc *time.Location) *simRoute {
	r := &simRoute{provider: p, cells: cells, sentOn: map[string]int64{}}
	rate := e.rateOf(ctx, e.Q, p, loc, start)
	r.rate, r.rateFrom = rate.PerSecond, "default"
	h := state.HealthOf(p)
	if h.Rate != nil && h.Rate.Max > 0 {
		r.rateFrom = "published"
	}
	if rate.WarmingUp {
		if a, ok, err := warmup.AllowanceFor(ctx, e.Q, p.ID, start, loc); err == nil && ok && a.Rate > 0 && a.Rate <= r.rate {
			r.rateFrom = "warmup"
		}
		r.allowance = func(t time.Time) int64 {
			a, ok, err := warmup.AllowanceFor(ctx, e.Q, p.ID, t, loc)
			if err != nil || !ok || a.Paused {
				return 0
			}
			if dayKey(t, loc) == dayKey(start, loc) {
				return max(a.Cap-a.Sent, 0)
			}
			return a.Cap
		}
		capLeft := r.allowance(start)
		r.out.CapLeft = &capLeft
	}
	if q := h.Quota; q != nil && q.Max > 0 && q.Kind != "warmup_daily" {
		left := max(q.Max-q.Used, 0)
		r.quotaLeft, r.quotaMax = &left, q.Max
		r.out.QuotaLeft = &left
	}
	r.held = !state.Available(p) && p.Status != state.Degraded
	return r
}

func dayKey(t time.Time, loc *time.Location) string { return t.In(loc).Format("2006-01-02") }

// simulate plays the send out a second at a time, as the send loop ticks:
// each route earns its rate, the allowance is split across mailbox
// providers by the loop's own allocate within the throttles' tokens (the
// loop's refill), a warm-up cap stops a route until the next day, the
// quota until it frees up 24 hours later, the A/B rest waits for the
// winner and best time spreads the audience over 24 hours. Stretches in
// which nothing can move are skipped to the next thing that frees mail.
func (e *Engine) simulate(p simPlan) Estimate {
	out := Estimate{Start: p.start, End: p.start, Throttles: p.throttles, BestTime: p.bestTime, Providers: []EstimateProvider{}}
	rs := &runState{queue: map[queueKey]queueCell{}, throttles: p.throttles, tokens: map[string]float64{}, holds: map[string]hold{}}
	// A cell's A/B rest waits for the winner.
	rest := map[queueKey]int64{}
	totals := map[queueKey]int64{}
	sample := 1.0
	if p.ab != nil {
		sample = float64(p.ab.TestPercent) / 100
	}
	routes := map[uuid.UUID]*runRoute{}
	for _, r := range p.routes {
		r.out.ProviderID, r.out.Rate, r.out.RateFrom, r.out.Held = r.provider.ID, r.rate, r.rateFrom, r.held
		routes[r.provider.ID] = &runRoute{provider: r.provider}
		for mbp, n := range r.cells {
			r.out.Recipients += n
			if r.held || n <= 0 {
				// A held share waits for a person (Send Readiness offers
				// moving it); it doesn't go out in this estimate.
				continue
			}
			k := queueKey{r.provider.ID, mbp}
			totals[k] = n
			if p.ab != nil {
				rest[k] = n - int64(math.Round(float64(n)*sample))
				p.ab.Sample += n - rest[k]
			}
			rs.queue[k] = queueCell{queued: n, waiting: n}
		}
	}
	decided := p.ab == nil
	quotaFree := p.start.Add(24 * time.Hour)
	end, sampleEnd := p.start, p.start
	limit := p.start.Add(estimateHorizon)
	for t := p.start; t.Before(limit); {
		if !decided && p.ab.DecideAt != nil && !t.Before(*p.ab.DecideAt) {
			decided = true
		}
		if !t.Before(quotaFree) {
			// The quota's 24-hour window rolled past what it counted.
			for _, r := range p.routes {
				if r.quotaLeft != nil && *r.quotaLeft <= 0 {
					full := r.quotaMax
					r.quotaLeft = &full
				}
			}
			quotaFree = quotaFree.Add(24 * time.Hour)
		}
		// What is due: the sample (everyone once the winner is picked), and
		// with best time, each hour's part of it.
		due := 1.0
		if p.bestTime {
			due = math.Min(1, (t.Sub(p.start).Hours()+1)/24)
		}
		var left, ready int64
		for k, c := range rs.queue {
			eligible := totals[k]
			if !decided {
				eligible -= rest[k]
			}
			sent := totals[k] - c.queued
			c.waiting = c.queued - max(min(int64(math.Ceil(float64(eligible)*due))-sent, c.queued), 0)
			rs.queue[k] = c
			left += c.queued
			if r := p.route(k.provider); r.canSend(t, p.loc) {
				ready += c.queued - c.waiting
			}
		}
		if left == 0 {
			break
		}
		if ready == 0 {
			next, ok := p.nextEvent(t, decided, quotaFree)
			if !ok {
				break // the rest waits for a person to pick the winner
			}
			t = next
			continue
		}
		e.refill(rs, t)
		sent := 0
		for _, r := range p.routes {
			if !r.canSend(t, p.loc) {
				continue
			}
			r.carry += r.rate
			want := int(math.Floor(r.carry))
			if want <= 0 {
				continue
			}
			if r.quotaLeft != nil {
				want = int(min(int64(want), *r.quotaLeft))
			}
			n := 0
			for mbp, k := range e.allocate(rs, routes[r.provider.ID], want) {
				if k <= 0 {
					continue
				}
				key := queueKey{r.provider.ID, mbp}
				c := rs.queue[key]
				c.queued -= int64(k)
				rs.queue[key] = c
				if th, ok := rs.throttles[mbp]; ok && th > 0 {
					rs.tokens[mbp] -= float64(k)
				}
				n += k
			}
			if n == 0 {
				r.carry = math.Min(r.carry, r.rate+1)
				continue
			}
			r.carry = math.Max(r.carry-float64(n), 0)
			r.sentOn[dayKey(t, p.loc)] += int64(n)
			if r.quotaLeft != nil {
				l := *r.quotaLeft - int64(n)
				r.quotaLeft = &l
				if l <= 0 && !t.After(quotaFree) {
					r.out.OverQuota = r.queued(rs)
				}
			}
			r.out.Seconds++
			r.out.Done = t.Add(time.Second)
			end = t.Add(time.Second)
			sent += n
		}
		if !decided && sent > 0 {
			sampleEnd = end
		}
		t = t.Add(time.Second)
	}
	out.End = end
	for _, r := range p.routes {
		r.out.Days = max(len(r.sentOn), 1)
		if r.out.Done.IsZero() {
			r.out.Done = p.start
		}
		out.Providers = append(out.Providers, r.out)
	}
	if p.ab != nil {
		p.ab.SampleEnd = sampleEnd
		out.Open = p.ab.DecideAt == nil
		out.AB = p.ab
	}
	return out
}

// route is the plan's route of a provider.
func (p simPlan) route(id uuid.UUID) *simRoute {
	for _, r := range p.routes {
		if r.provider.ID == id {
			return r
		}
	}
	return nil
}

// canSend reports whether the route may send at t: not held, its warm-up
// allowance and its quota not used up.
func (r *simRoute) canSend(t time.Time, loc *time.Location) bool {
	if r == nil || r.held {
		return false
	}
	if r.allowance != nil && r.allowance(t)-r.sentOn[dayKey(t, loc)] <= 0 {
		return false
	}
	return r.quotaLeft == nil || *r.quotaLeft > 0
}

// queued is what is left of the route's share.
func (r *simRoute) queued(rs *runState) int64 {
	var n int64
	for k, c := range rs.queue {
		if k.provider == r.provider.ID {
			n += c.queued
		}
	}
	return n
}

// nextEvent is the next time mail can move when none can now: the winner
// is picked, the next best-time hour, the next local day (a warm-up cap)
// or the quota freeing. ok is false when nothing will free it (a person
// picks the A/B winner).
func (p simPlan) nextEvent(t time.Time, decided bool, quotaFree time.Time) (time.Time, bool) {
	var best time.Time
	consider := func(c time.Time) {
		if c.After(t) && (best.IsZero() || c.Before(best)) {
			best = c
		}
	}
	if !decided && p.ab.DecideAt != nil {
		consider(*p.ab.DecideAt)
	}
	if p.bestTime && t.Before(p.start.Add(24*time.Hour)) {
		consider(p.start.Add(t.Sub(p.start).Truncate(time.Hour) + time.Hour))
	}
	for _, r := range p.routes {
		if r.held {
			continue
		}
		if r.allowance != nil {
			local := t.In(p.loc)
			consider(time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, p.loc))
		}
		if r.quotaLeft != nil && *r.quotaLeft <= 0 {
			consider(quotaFree)
		}
	}
	return best, !best.IsZero()
}
