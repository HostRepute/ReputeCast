package approvals_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/readiness/readinesstest"
	"github.com/hostrepute/reputecast/internal/sending"
)

// canaryStarter is a sending engine that tells its canary plan (C1:
// CanaryPlanFor) and records the canary choice each start carries.
type canaryStarter struct {
	fakeStarter
	plan    sending.CanaryPlan
	choices []string
}

func (c *canaryStarter) Start(ctx context.Context, id, by uuid.UUID) error {
	c.mu.Lock()
	c.choices = append(c.choices, sending.CanaryFrom(ctx, id))
	c.mu.Unlock()
	return c.fakeStarter.Start(ctx, id, by)
}

func (c *canaryStarter) CanaryPlanFor(context.Context, uuid.UUID) (sending.CanaryPlan, error) {
	return c.plan, nil
}

type canaryReview struct {
	Canary struct {
		Choice  string
		Planned bool
		Auto    bool
		Reasons []string
		Size    int64
	}
}

// C1: Review & send shows the engine's canary plan and the choice; the
// choice goes with the start of a send now, and is kept with what was
// confirmed for a scheduled one.
func TestCanaryChoice(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	digest := campaignID(t, linh, "Weekly Digest #142")
	id := uuid.MustParse(digest)
	canary := func() canaryReview {
		t.Helper()
		var r canaryReview
		linh.Do("GET", ws+"/campaigns/"+digest+"/review", nil).MustStatus(t, 200).JSON(t, &r)
		return r
	}

	// The engine doesn't tell: automatic, plan unknown.
	env.App.Sending = &fakeStarter{}
	if c := canary().Canary; c.Choice != "auto" || c.Planned || len(c.Reasons) != 0 {
		t.Fatalf("no plan: %+v", c)
	}
	starter := &canaryStarter{plan: sending.CanaryPlan{Auto: true, Reasons: []string{"the first send from news.acme.io"}, Size: 1560}}
	env.App.Sending = starter
	if c := canary().Canary; c.Choice != "auto" || !c.Planned || !c.Auto || c.Size != 1560 || len(c.Reasons) != 1 {
		t.Fatalf("the engine's plan: %+v", c)
	}

	// No approval in the way.
	var policy struct{ Policy map[string]any }
	linh.Do("GET", ws+"/approval-policy", nil).MustStatus(t, 200).JSON(t, &policy)
	for _, k := range []string{"warmingProvider", "readyWithWarnings", "authorLacksSend", "approvalLists"} {
		policy.Policy[k] = false
	}
	policy.Policy["audienceMin"] = map[string]any{"enabled": false, "n": 10000}
	linh.Do("PUT", ws+"/approval-policy", policy.Policy).MustStatus(t, 200)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE approval_requests SET status = 'cancelled' WHERE campaign_id = $1 AND status IN ('pending', 'approved')`, id); err != nil {
		t.Fatal(err)
	}
	linh.Do("POST", ws+"/campaigns/"+digest+"/unschedule", nil).MustStatus(t, 204)
	send := func(choice string) *apptest.Response {
		t.Helper()
		readinesstest.Post(t, env, linh, ws+"/campaigns/"+digest+"/readiness").MustStatus(t, 200)
		r := getReview(t, linh, digest)
		return linh.Do("POST", ws+"/campaigns/"+digest+"/send", map[string]any{"runId": r.Run.ID, "acknowledged": warnings(r),
			"confirm": fmt.Sprint(r.Run.Estimates.Audience.Final), "canary": choice})
	}
	send("sometimes").MustStatus(t, 422)

	// Scheduled with "No canary": kept with what was confirmed, and audited.
	send("off").MustStatus(t, 200)
	var kept string
	if err := env.App.Pool.QueryRow(ctx, `SELECT canary FROM campaign_send_confirmations WHERE campaign_id = $1`, id).Scan(&kept); err != nil || kept != "off" {
		t.Fatalf("kept: %q (%v)", kept, err)
	}
	if c := canary().Canary; c.Choice != "off" {
		t.Fatalf("the review shows the choice: %+v", c)
	}
	var audited string
	if err := env.App.Pool.QueryRow(ctx, `SELECT metadata->>'canary' FROM audit_events WHERE action = 'campaign.scheduled' AND target_id = $1 ORDER BY occurred_at DESC LIMIT 1`, digest).Scan(&audited); err != nil || audited != "off" {
		t.Fatalf("audited: %q (%v)", audited, err)
	}

	// Sent now with a canary: the choice goes with the start.
	linh.Do("POST", ws+"/campaigns/"+digest+"/unschedule", nil).MustStatus(t, 204)
	linh.Do("PATCH", ws+"/campaigns/"+digest+"/send-settings", map[string]any{"schedule": map[string]any{"mode": "now"}}).MustStatus(t, 204)
	send("on").MustStatus(t, 200)
	// The engine heard each choice: when it was scheduled, and now.
	if len(starter.choices) != 2 || starter.choices[0] != "off" || starter.choices[1] != "on" {
		t.Fatalf("started with %v", starter.choices)
	}
}
