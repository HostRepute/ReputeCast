package engine

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// The estimate plays a send out with the send loop's own allocation and
// throttles (SR-5, C4): these cases check its arithmetic against the
// rules; estimate_test.go checks it against real sends on a virtual clock.

var estStart = time.Date(2026, 10, 14, 13, 0, 0, 0, time.UTC) // 09:00 in New York

func route(adapter string, rate float64, cells map[string]int64) *simRoute {
	return &simRoute{provider: store.Provider{ID: uuid.New(), Adapter: adapter, Status: "healthy"}, rate: rate, rateFrom: "published", cells: cells, sentOn: map[string]int64{}}
}

func plan(routes ...*simRoute) simPlan {
	ny, _ := time.LoadLocation("America/New_York")
	return simPlan{start: estStart, loc: ny, routes: routes, throttles: map[string]float64{"outlook": 5, "yahoo": 4, "icloud": 3}}
}

func near(t *testing.T, what string, got, want time.Duration, tolerance time.Duration) {
	t.Helper()
	if d := got - want; d < -tolerance || d > tolerance {
		t.Errorf("%s: %s, want about %s", what, got, want)
	}
}

// The board's send: SES 28,073 at 14/s and Postal 3,119 at its 2/s ramp,
// in parallel: the slower route sets the time (33 minutes).
func TestEstimateParallelRoutes(t *testing.T) {
	ses := route("ses", 14, map[string]int64{"gmail": 12900, "outlook": 5900, "yahoo": 2500, "icloud": 2200, "other": 4573})
	postal := route("postal", 2, map[string]int64{"gmail": 1448, "outlook": 651, "yahoo": 307, "icloud": 295, "other": 418})
	est := (&Engine{}).simulate(plan(ses, postal))
	near(t, "SES", est.Providers[0].Done.Sub(estStart), 2005*time.Second, 15*time.Second)
	near(t, "Postal", est.Providers[1].Done.Sub(estStart), 1560*time.Second, 15*time.Second)
	near(t, "the send", est.End.Sub(estStart), 2005*time.Second, 15*time.Second)
	if est.Providers[0].Days != 1 || est.Open {
		t.Fatalf("estimate: %+v", est)
	}
}

// A mailbox provider's throttle is shared by every route: 10,000 Outlook
// recipients take 2,000 seconds at 5/s, whatever the providers' rates.
func TestEstimateMailboxThrottle(t *testing.T) {
	ses := route("ses", 14, map[string]int64{"outlook": 6000})
	pm := route("postmark", 50, map[string]int64{"outlook": 4000})
	est := (&Engine{}).simulate(plan(ses, pm))
	near(t, "Outlook at 5/s", est.End.Sub(estStart), 2000*time.Second, 10*time.Second)
}

// The A/B test: the sample goes first, the rest when the winner is picked
// (2 hours), at the routes' rates.
func TestEstimateABWait(t *testing.T) {
	ses := route("ses", 14, map[string]int64{"gmail": 28073})
	postal := route("postal", 2, map[string]int64{"gmail": 3119})
	p := plan(ses, postal)
	decide := estStart.Add(2 * time.Hour)
	p.ab = &EstimateAB{Kind: "subject", Variants: 2, TestPercent: 20, Hours: 2, DecideAt: &decide}
	est := (&Engine{}).simulate(p)
	if est.AB.Sample != 6239 && est.AB.Sample != 6238 {
		t.Errorf("sample: %d", est.AB.Sample)
	}
	near(t, "sample", est.AB.SampleEnd.Sub(estStart), 401*time.Second, 15*time.Second) // SES's 5,615 at 14/s
	near(t, "everyone", est.End.Sub(estStart), 2*time.Hour+1604*time.Second, 20*time.Second)
	// A person picks the winner: the rest waits for them.
	p = plan(route("ses", 14, map[string]int64{"gmail": 10000}))
	p.ab = &EstimateAB{Kind: "subject", Variants: 2, TestPercent: 20, Hours: 2}
	est = (&Engine{}).simulate(p)
	if !est.Open || est.End.Sub(estStart) > 200*time.Second {
		t.Fatalf("manual pick: open %v, end %s", est.Open, est.End.Sub(estStart))
	}
}

// A warm-up plan's caps spread a share over days (the review's "83
// minutes" for a send that took about 20 days).
func TestEstimateWarmupCapsOverDays(t *testing.T) {
	postal := route("postal", 2, map[string]int64{"gmail": 10000})
	postal.rateFrom = "warmup"
	ny, _ := time.LoadLocation("America/New_York")
	caps := map[string]int64{"2026-10-14": 3000, "2026-10-15": 3500, "2026-10-16": 4000, "2026-10-17": 4500}
	postal.allowance = func(t time.Time) int64 { return caps[t.In(ny).Format("2006-01-02")] }
	est := (&Engine{}).simulate(plan(postal))
	if est.Providers[0].Days != 3 {
		t.Fatalf("days: %d", est.Providers[0].Days)
	}
	// Day 3 starts at midnight in New York and sends the last 3,500.
	thirdDay := time.Date(2026, 10, 16, 0, 0, 0, 0, ny)
	near(t, "day 3", est.End.Sub(thirdDay), 1750*time.Second, 10*time.Second)
}

// A provider's 24-hour quota: what's over it waits for the window to free.
func TestEstimateQuota(t *testing.T) {
	ses := route("ses", 14, map[string]int64{"gmail": 5000})
	left := int64(1000)
	ses.quotaLeft, ses.quotaMax = &left, 50000
	est := (&Engine{}).simulate(plan(ses))
	if est.Providers[0].OverQuota != 4000 {
		t.Fatalf("over quota: %d", est.Providers[0].OverQuota)
	}
	near(t, "after the quota frees", est.End.Sub(estStart), 24*time.Hour+286*time.Second, 10*time.Second)
}

// Best time: each hour's part goes at its hour, within 24 hours.
func TestEstimateBestTime(t *testing.T) {
	p := plan(route("ses", 14, map[string]int64{"gmail": 24000}))
	p.bestTime = true
	est := (&Engine{}).simulate(p)
	if got := est.End.Sub(estStart); got < 23*time.Hour || got > 24*time.Hour+time.Minute {
		t.Fatalf("best time: %s", got)
	}
}

// A held share (its provider is down) doesn't go out in the estimate.
func TestEstimateHeldShare(t *testing.T) {
	ses := route("ses", 14, map[string]int64{"gmail": 1400})
	postal := route("postal", 2, map[string]int64{"gmail": 1000})
	postal.held = true
	est := (&Engine{}).simulate(plan(ses, postal))
	near(t, "SES only", est.End.Sub(estStart), 100*time.Second, 3*time.Second)
	if !est.Providers[1].Held || est.Providers[1].Seconds != 0 {
		t.Fatalf("held: %+v", est.Providers[1])
	}
}

// A rate under one per second still sends (the loop carries the rest).
func TestEstimateSlowRate(t *testing.T) {
	est := (&Engine{}).simulate(plan(route("smtp", 0.5, map[string]int64{"gmail": 100})))
	if got := est.End.Sub(estStart).Seconds(); math.Abs(got-200) > 3 {
		t.Fatalf("0.5/s: %v s", got)
	}
}

// Send Readiness's "Send via SES only" stores a split; the router reads it
// (it read only a top-level provider_id before, and sent by the rules).
func TestRouteOverrideReadsReadinessSplits(t *testing.T) {
	ses, postal := uuid.New(), uuid.New()
	raw := []byte(`{"splits":[{"provider_id":"` + postal.String() + `","weight":0},{"provider_id":"` + ses.String() + `","weight":100}]}`)
	if got := routeOverride(raw); got == nil || *got != ses {
		t.Fatalf("readiness override: %v", got)
	}
	if got := routeOverride([]byte(`{"provider_id":"` + postal.String() + `"}`)); got == nil || *got != postal {
		t.Fatalf("top-level provider: %v", got)
	}
	if got := routeOverride([]byte(`{"splits":[]}`)); got != nil {
		t.Fatalf("no split: %v", got)
	}
}
