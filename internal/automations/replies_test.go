package automations_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/db"
)

// receiveReply stores what the inbound job stores for a reply to message
// (nil: a reply it couldn't tie to one email) from contact, about flow.
func receiveReply(t *testing.T, env *apptest.Env, message any, contact uuid.UUID, flow string, auto bool, at time.Time) {
	t.Helper()
	if _, err := env.App.Pool.Exec(context.Background(), `INSERT INTO inbound_messages (id, workspace_id, kind, message_id, contact_id, automation_id, auto_reply, subject, received_at)
		SELECT $1, workspace_id, 'reply', $2, id, $3, $4, 'Re: Hello', $5 FROM contacts WHERE id = $6`, uuid.Must(uuid.NewV7()), message, flow, auto, at, contact); err != nil {
		t.Fatal(err)
	}
}

// M58 G-18: a person's reply ends the contact's run in a flow that stops on
// reply, with the reason "replied"; a flow without the switch carries on.
func TestRepliesEndRunsOfFlowsThatStopOnReply(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	svc := engine(env, &clock{t: time.Now()})
	var runID, flow, contact, ws uuid.UUID
	var version int32
	if err := env.App.Pool.QueryRow(ctx, `SELECT r.id, r.automation_id, r.contact_id, a.workspace_id, r.version FROM automation_runs r
		JOIN automations a ON a.id = r.automation_id WHERE a.name = 'Welcome series' AND r.status = 'waiting' ORDER BY r.id LIMIT 1`).
		Scan(&runID, &flow, &contact, &ws, &version); err != nil {
		t.Fatal(err)
	}
	active := countWhere(t, env, `SELECT count(*) FROM automation_runs WHERE automation_id = $1 AND contact_id = $2 AND status IN ('active', 'waiting')`, flow, contact)
	end := func() int {
		t.Helper()
		var n int
		if err := db.Tx(ctx, env.App.Pool, func(tx pgx.Tx) error {
			var err error
			n, err = svc.EndRunsOnReply(ctx, tx, ws, flow, contact, time.Now())
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return n
	}
	// The demo's Welcome series doesn't stop on reply.
	if n := end(); n != 0 {
		t.Fatalf("a flow without the switch ended %d runs", n)
	}
	var status string
	if err := env.App.Pool.QueryRow(ctx, `SELECT status FROM automation_runs WHERE id = $1`, runID).Scan(&status); err != nil || status != "waiting" {
		t.Fatalf("run without the switch: %s %v", status, err)
	}

	// Its version turns "Stop when the contact replies" on.
	var raw []byte
	if err := env.App.Pool.QueryRow(ctx, `SELECT graph FROM automation_versions WHERE automation_id = $1 AND version = $2`, flow, version).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	g, err := automations.ParseGraph(raw)
	if err != nil {
		t.Fatal(err)
	}
	g.Trigger().Data.Trigger.ExitOnReply = true
	if _, err := env.App.Pool.Exec(ctx, `UPDATE automation_versions SET graph = $3 WHERE automation_id = $1 AND version = $2`, flow, version, g.MustJSON()); err != nil {
		t.Fatal(err)
	}
	var node string
	if err := env.App.Pool.QueryRow(ctx, `SELECT node_id FROM automation_runs WHERE id = $1`, runID).Scan(&node); err != nil {
		t.Fatal(err)
	}
	dropped := countWhere(t, env, `SELECT coalesce(sum(dropped), 0)::bigint FROM automation_step_stats WHERE automation_id = $1 AND node_id = $2`, flow, node)
	if n := end(); int64(n) != active || n == 0 {
		t.Fatalf("ended %d runs, the contact had %d", n, active)
	}
	var reason string
	var finished *time.Time
	if err := env.App.Pool.QueryRow(ctx, `SELECT status, exit_reason, finished_at FROM automation_runs WHERE id = $1`, runID).Scan(&status, &reason, &finished); err != nil ||
		status != "exited" || reason != automations.ExitReplied || finished == nil {
		t.Fatalf("run after the reply: %s %s %v %v", status, reason, finished, err)
	}
	if n := countWhere(t, env, `SELECT coalesce(sum(dropped), 0)::bigint FROM automation_step_stats WHERE automation_id = $1 AND node_id = $2`, flow, node); n < dropped+1 {
		t.Fatalf("dropped at %s: %d, was %d", node, n, dropped)
	}
	// Nothing left to end.
	if n := end(); n != 0 {
		t.Fatalf("a second reply ended %d runs", n)
	}
}

// M58 G-18.3: a "replied" branch waits for a person's reply to its email
// (an auto-reply doesn't count) and a reply wakes it; a "replied" goal is
// met by a reply to any email of the flow.
func TestRepliedBranchAndGoal(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	clk := &clock{t: time.Now()}
	svc := engine(env, clk)
	emailID := newEmail(t, c, "Test · replied")
	product := listID(t, env, "Product Updates")
	flow := publishFlow(t, c, "Replied branch", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: product})),
			node("m", automations.NodeEmail, emailData(emailID)),
			node("b", automations.NodeBranch, automations.AutomationNodeData{Condition: &automations.AutomationCondition{Kind: automations.CondReplied, Step: "m",
				Within: &automations.AutomationDuration{Amount: 1, Unit: "days"}}}),
			node("yes", automations.NodeList, tagStep("replied-yes")),
			node("no", automations.NodeList, tagStep("replied-no")),
		},
		Edges: []automations.AutomationEdge{link("t", "m", ""), link("m", "b", ""), link("b", "yes", "yes"), link("b", "no", "no")},
	})
	goalFlow := publishFlow(t, c, "Replied goal", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: product})),
			node("m", automations.NodeEmail, emailData(emailID)),
			node("d", automations.NodeDelay, delayStep(3, "days")),
			node("g", automations.NodeGoal, automations.AutomationNodeData{Label: "Replied", Condition: &automations.AutomationCondition{Kind: automations.CondReplied}}),
		},
		Edges: []automations.AutomationEdge{link("t", "m", ""), link("m", "d", ""), link("d", "g", "")},
	})
	who := addContact(t, env, "replier@example.com", nil)
	joinList(t, env, product, who)
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	nodeID, status, state := run(t, env, flow, who)
	if nodeID != "b" || status != "waiting" || state["want"] != automations.CondReplied || state["message"] == nil {
		t.Fatalf("the contact should wait at the branch: %s %s %v", nodeID, status, state)
	}
	// An auto-reply wakes nobody.
	receiveReply(t, env, state["message"], who, flow, true, clk.now().Add(time.Minute))
	receiveReply(t, env, nil, who, goalFlow, true, clk.now().Add(time.Minute))
	if err := svc.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if nodeID, status, _ = run(t, env, flow, who); nodeID != "b" || status != "waiting" {
		t.Fatalf("after an auto-reply the run is at %s %s", nodeID, status)
	}
	if _, st, _ := run(t, env, goalFlow, who); st != "waiting" {
		t.Fatalf("an auto-reply met the goal: %s", st)
	}
	// The person replies: the branch wakes and takes "Yes"; the goal flow
	// meets its goal on a reply to any of its emails.
	receiveReply(t, env, state["message"], who, flow, false, clk.now().Add(time.Minute))
	receiveReply(t, env, nil, who, goalFlow, false, clk.now().Add(time.Minute))
	if err := svc.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if nodeID, status, _ = run(t, env, flow, who); nodeID != "yes" || status != "completed" {
		t.Fatalf("after the reply the run is at %s %s", nodeID, status)
	}
	if nodeID, st, _ := run(t, env, goalFlow, who); nodeID != "g" || st != "goal" {
		t.Fatalf("goal flow: %s %s", nodeID, st)
	}
}
