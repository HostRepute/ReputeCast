package warmup_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/warmup"
)

const ws = "/workspaces/acme-newsletter"

func TestRampGrowsToTheTarget(t *testing.T) {
	days := warmup.Ramp(21, 200, 48000)
	if len(days) != 21 || days[0].Volume != 200 || days[20].Volume != 48000 {
		t.Fatalf("ramp ends %d … %d", days[0].Volume, days[20].Volume)
	}
	for i := 1; i < len(days); i++ {
		if days[i].Volume < days[i-1].Volume {
			t.Fatalf("day %d shrinks: %d < %d", i+1, days[i].Volume, days[i-1].Volume)
		}
		if days[i].Share < days[i-1].Share {
			t.Fatalf("share drops on day %d", i+1)
		}
	}
	if days[0].Share != 10 || days[20].Share != 40 || len(days[20].Tiers) != len(warmup.Tiers) || len(days[0].Tiers) != 1 {
		t.Fatalf("first %+v, last %+v", days[0], days[20])
	}
	if warmup.RateFor(200) != 1 || warmup.RateFor(6000) != 2 {
		t.Fatalf("rates %v %v", warmup.RateFor(200), warmup.RateFor(6000))
	}
}

func TestHeldDayKeepsYesterdaysCap(t *testing.T) {
	sched, _ := json.Marshal(warmup.Ramp(21, 200, 48000))
	progress, _ := json.Marshal([]warmup.Result{{Day: 6, Cap: 2600, Sent: 2600, Bounce: 2.3, Slowed: true}, {Day: 7, Cap: 2600, Sent: 2600, Held: true}})
	p := warmup.ParsePlan(sched, progress)
	if p.Cap(7) != 2600 {
		t.Fatalf("held day cap = %d", p.Cap(7))
	}
	if p.PlannedCap(7) == 2600 {
		t.Fatal("planned cap should keep the schedule's volume")
	}
	if r, ok := p.Result(6); !ok || !r.Slowed {
		t.Fatalf("day 6 result %+v", r)
	}
}

func TestWarmupEndpoints(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	var o struct {
		Plans []struct {
			ID, Status string
			Day, Days  int
			TodayCap   int64
			Schedule   []struct {
				Day    int
				Status string
				Uses   []struct{ Name string }
			}
			Log      []struct{ Kind string }
			Settings warmup.WarmupSettings
		}
		Candidates []struct{ ProviderID string }
	}
	owner.Do("GET", ws+"/warmup", nil).MustStatus(t, 200).JSON(t, &o)
	if len(o.Plans) != 1 {
		t.Fatalf("plans %+v", o.Plans)
	}
	plan := o.Plans[0]
	if plan.Day != 9 || plan.Days != 21 || plan.TodayCap != 6000 || len(plan.Log) != 4 || len(plan.Schedule) != 21 {
		t.Fatalf("plan day %d/%d cap %d log %d", plan.Day, plan.Days, plan.TodayCap, len(plan.Log))
	}
	if plan.Schedule[5].Status != "slowdown" || plan.Schedule[8].Status != "today" {
		t.Fatalf("day 6 %s, day 9 %s", plan.Schedule[5].Status, plan.Schedule[8].Status)
	}
	path := ws + "/warmup/" + plan.ID

	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("GET", ws+"/warmup", nil).MustStatus(t, 200)
	settings := warmup.WarmupSettings{Days: 21, EngagedFirst: true, GuardBounce: 1.5, GuardComplaint: 0.08, GuardAction: "step_back", SpreadMailbox: true}
	viewer.Do("PUT", path, settings).MustStatus(t, 403)
	bad := settings
	bad.GuardBounce = 50
	owner.Do("PUT", path, bad).MustStatus(t, 422)
	owner.Do("PUT", path, settings).MustStatus(t, 200)
	owner.Do("GET", ws+"/warmup", nil).MustStatus(t, 200).JSON(t, &o)
	if o.Plans[0].Settings.GuardBounce != 1.5 || o.Plans[0].Settings.GuardAction != "step_back" {
		t.Fatalf("settings %+v", o.Plans[0].Settings)
	}

	viewer.Do("POST", path+"/pause", nil).MustStatus(t, 403)
	owner.Do("POST", path+"/pause", nil).MustStatus(t, 200)
	owner.Do("GET", ws+"/warmup", nil).MustStatus(t, 200).JSON(t, &o)
	if o.Plans[0].Status != "paused" {
		t.Fatalf("status after pause %s", o.Plans[0].Status)
	}
	owner.Do("POST", path+"/resume", nil).MustStatus(t, 200)

	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action IN ('warmup.update','warmup.pause','warmup.resume')`).Scan(&n); err != nil || n != 3 {
		t.Fatalf("warm-up audit = %d %v", n, err)
	}

	// The other providers can start a plan; Postal already has one.
	if len(o.Candidates) == 0 {
		t.Fatal("no warm-up candidates")
	}
	var created struct {
		Day, Days int
		Status    string
	}
	owner.Do("POST", ws+"/warmup", map[string]any{"providerId": o.Candidates[0].ProviderID, "days": 14, "engagedFirst": true, "guardBounce": 2, "guardComplaint": 0.1, "guardAction": "hold", "spreadMailbox": true}).
		MustStatus(t, 201).JSON(t, &created)
	if created.Days != 14 || created.Day != 1 {
		t.Fatalf("created %+v", created)
	}
}
