package automations_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/automations"
)

// M63 G-33: a contact in the holdout group passes an automation's email
// step without an email (the flow goes on, so goals and branches stay
// comparable) and the step counts it as held out and skipped; a contact
// outside the group gets the email.
func TestHoldoutPassesTheEmailStep(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	svc := engine(env, &clock{t: time.Now()})
	// 20% of contacts, out of automations only.
	c.Do("PUT", "/workspaces/acme-newsletter/holdout", map[string]any{"on": true, "basisPoints": 2000, "campaigns": false, "automations": true}).MustStatus(t, 200)
	var seed int64
	if err := env.App.Pool.QueryRow(ctx, `SELECT p.seed FROM holdout_periods p JOIN workspaces w ON w.id = p.workspace_id
		WHERE w.slug = 'acme-newsletter' AND p.ended_at IS NULL`).Scan(&seed); err != nil {
		t.Fatal(err)
	}
	emailID := newEmail(t, c, "Test · holdout")
	product := listID(t, env, "Product Updates")
	flow := publishFlow(t, c, "Holdout passes", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: product})),
			node("m", automations.NodeEmail, automations.AutomationNodeData{Email: &automations.AutomationEmailStep{EmailID: emailID, EmailVersion: 1, Subject: "Hi"}}),
			node("tag", automations.NodeList, tagStep("after-holdout")),
		},
		Edges: []automations.AutomationEdge{link("t", "m", ""), link("m", "tag", "")},
	})
	// One contact in the group, one outside it.
	var held, reached uuid.UUID
	for i := 0; held == uuid.Nil || reached == uuid.Nil; i++ {
		if i > 200 {
			t.Fatal("no contact fell in or out of a 20% holdout group in 200 tries")
		}
		id := addContact(t, env, fmt.Sprintf("holdout-%d@example.com", i), nil)
		in := countWhere(t, env, `SELECT count(*) FROM (SELECT 1 WHERE rc_holdout_bucket($1, $2) < 2000) x`, id, seed) == 1
		switch {
		case in && held == uuid.Nil:
			held = id
		case !in && reached == uuid.Nil:
			reached = id
		default:
			continue
		}
		joinList(t, env, product, id)
	}
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if nodeID, status, _ := run(t, env, flow, held); nodeID != "tag" || status != "completed" {
		t.Fatalf("held-out contact: %s %s, want the flow to go on past the email", nodeID, status)
	}
	if n := countWhere(t, env, `SELECT count(*) FROM messages WHERE contact_id = $1 AND stream = 'automation'`, held); n != 0 {
		t.Fatalf("%d automation emails to a contact in the holdout group", n)
	}
	if n := countWhere(t, env, `SELECT count(*) FROM messages WHERE contact_id = $1 AND stream = 'automation'`, reached); n != 1 {
		t.Fatalf("%d automation emails to a contact outside the holdout group, want 1", n)
	}
	if n := countWhere(t, env, `SELECT coalesce(sum(held_out), 0) FROM automation_step_stats WHERE automation_id = $1 AND node_id = 'm'`, flow); n != 1 {
		t.Fatalf("held out %d at the email step, want 1", n)
	}
	if n := countWhere(t, env, `SELECT coalesce(sum(skipped), 0) FROM automation_step_stats WHERE automation_id = $1 AND node_id = 'm'`, flow); n != 1 {
		t.Fatalf("skipped %d at the email step, want 1", n)
	}
	// The step's stats say so.
	var stats struct {
		Steps []struct {
			NodeID  string `json:"nodeId"`
			HeldOut int64  `json:"heldOut"`
			Skipped int64  `json:"skipped"`
		} `json:"steps"`
	}
	c.Do("GET", wsPath+"/"+flow+"/stats", nil).MustStatus(t, 200).JSON(t, &stats)
	found := false
	for _, s := range stats.Steps {
		if s.NodeID == "m" {
			found = s.HeldOut == 1 && s.Skipped == 1
		}
	}
	if !found {
		t.Fatalf("stats: %+v", stats.Steps)
	}
}
