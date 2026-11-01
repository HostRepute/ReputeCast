package automations_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
)

// The review 2026-11 M44 tests: automations, complete.

func joinList(t *testing.T, env *apptest.Env, list string, contact uuid.UUID) {
	t.Helper()
	if _, err := env.App.Pool.Exec(context.Background(), `INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at) VALUES ($1,$2,'subscribed','form:test',now())
		ON CONFLICT (list_id, contact_id) DO UPDATE SET status = 'subscribed'`, list, contact); err != nil {
		t.Fatal(err)
	}
}

func trig(t automations.AutomationTrigger) automations.AutomationNodeData {
	return automations.AutomationNodeData{Trigger: &t}
}

func delayStep(n int, unit string) automations.AutomationNodeData {
	return automations.AutomationNodeData{Delay: &automations.AutomationDuration{Amount: n, Unit: unit}}
}

func emailData(id string) automations.AutomationNodeData {
	return automations.AutomationNodeData{Email: &automations.AutomationEmailStep{EmailID: id, EmailVersion: 1, Subject: "Hello"}}
}

func tagStep(tag string) automations.AutomationNodeData {
	return automations.AutomationNodeData{Action: &automations.AutomationListAction{Kind: "add_tag", Tag: tag}}
}

func countWhere(t *testing.T, env *apptest.Env, sql string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// AT-2: the contacts at a step page by cursor; one can be taken out of
// the flow, with a reason, logged; the profile lists their flows.
func TestContactsAtAStepAndRemovingOne(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	svc := engine(env, &clock{t: time.Now()})
	emailID := newEmail(t, c, "Test · at a step")
	product := listID(t, env, "Product Updates")
	flow := publishFlow(t, c, "At a step", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: product})),
			node("d", automations.NodeDelay, delayStep(2, "days")),
			node("m", automations.NodeEmail, emailData(emailID)),
		},
		Edges: []automations.AutomationEdge{link("t", "d", ""), link("d", "m", "")},
	})
	var people []uuid.UUID
	for _, e := range []string{"step1@example.com", "step2@example.com", "step3@example.com"} {
		id := addContact(t, env, e, nil)
		people = append(people, id)
		joinList(t, env, product, id)
	}
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	type page struct {
		Total int64
		Items []struct {
			RunID, ContactID, Email, Wait string
			Until                         *time.Time
		}
		Next string
	}
	var p1, p2 page
	c.Do("GET", wsPath+"/"+flow+"/steps/d/contacts?limit=2", nil).MustStatus(t, 200).JSON(t, &p1)
	if p1.Total != 3 || len(p1.Items) != 2 || p1.Next == "" || p1.Items[0].Wait != "delay" || p1.Items[0].Until == nil {
		t.Fatalf("first page: %+v", p1)
	}
	c.Do("GET", wsPath+"/"+flow+"/steps/d/contacts?limit=2&cursor="+p1.Next, nil).MustStatus(t, 200).JSON(t, &p2)
	if len(p2.Items) != 1 || p2.Next != "" || p2.Items[0].RunID == p1.Items[0].RunID || p2.Items[0].RunID == p1.Items[1].RunID {
		t.Fatalf("second page: %+v", p2)
	}
	c.Do("GET", wsPath+"/"+flow+"/steps/d/contacts?cursor=nonsense", nil).MustStatus(t, 422)

	target := p1.Items[0]
	c.Do("POST", wsPath+"/"+flow+"/runs/"+target.RunID+"/remove", map[string]any{"reason": "  "}).MustStatus(t, 422)
	c.Do("POST", wsPath+"/"+flow+"/runs/"+target.RunID+"/remove", map[string]any{"reason": "Asked to stop the onboarding emails"}).MustStatus(t, 204)
	c.Do("POST", wsPath+"/"+flow+"/runs/"+target.RunID+"/remove", map[string]any{"reason": "Again"}).MustStatus(t, 409)
	var status, reason string
	if err := env.App.Pool.QueryRow(ctx, `SELECT status, exit_reason FROM automation_runs WHERE id = $1`, target.RunID).Scan(&status, &reason); err != nil || status != "exited" || reason != "removed" {
		t.Fatalf("removed run: %s %s %v", status, reason, err)
	}
	if n := countWhere(t, env, `SELECT count(*) FROM audit_events WHERE action = 'automation.contact_remove' AND reason = 'Asked to stop the onboarding emails' AND target_id = $1`, target.ContactID); n != 1 {
		t.Fatalf("%d audit entries", n)
	}
	if n := countWhere(t, env, `SELECT coalesce(sum(dropped), 0)::bigint FROM automation_step_stats WHERE automation_id = $1 AND node_id = 'd'`, flow); n != 1 {
		t.Fatalf("dropped at the step: %d", n)
	}
	var mine struct {
		Items []struct{ AutomationID, Name, Status, Step, ExitReason string }
	}
	c.Do("GET", "/workspaces/acme-newsletter/contacts/"+target.ContactID+"/automations", nil).MustStatus(t, 200).JSON(t, &mine)
	if len(mine.Items) != 1 || mine.Items[0].AutomationID != flow || mine.Items[0].Status != "exited" || mine.Items[0].ExitReason != "removed" || mine.Items[0].Step == "" {
		t.Fatalf("the contact's flows: %+v", mine.Items)
	}
	// A viewer without the audience can't list who is in a flow.
	_, viewer := signedIn(t, "aisha@acme.io")
	viewer.Do("POST", wsPath+"/"+flow+"/runs/"+p1.Items[1].RunID+"/remove", map[string]any{"reason": "Not mine to do"}).MustStatus(t, 403)
}

// AT-4: resuming says what happens to contacts whose wait ended during
// the pause: skip the email they missed, or spread them over hours.
func TestResumeWithOverdueContacts(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	svc := engine(env, &clock{t: time.Now()})
	emailID := newEmail(t, c, "Test · overdue")
	product := listID(t, env, "Product Updates")
	graph := automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: product})),
			node("d", automations.NodeDelay, delayStep(1, "hours")),
			node("m", automations.NodeEmail, emailData(emailID)),
			node("tag", automations.NodeList, tagStep("after-overdue")),
		},
		Edges: []automations.AutomationEdge{link("t", "d", ""), link("d", "m", ""), link("m", "tag", "")},
	}
	flow := publishFlow(t, c, "Overdue", graph)
	var people []uuid.UUID
	for i := range 4 {
		id := addContact(t, env, "overdue"+string(rune('a'+i))+"@example.com", nil)
		people = append(people, id)
		joinList(t, env, product, id)
	}
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	c.Do("POST", wsPath+"/"+flow+"/pause", map[string]any{"reason": "Fixing the email"}).MustStatus(t, 200)
	// Their delays end during the pause.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE automation_runs SET next_at = now() - interval '5 minutes',
		state = jsonb_set(state, '{until}', to_jsonb(now() - interval '5 minutes')) WHERE automation_id = $1`, flow); err != nil {
		t.Fatal(err)
	}
	var detail struct {
		Overdue int64
		Status  string
	}
	c.Do("GET", wsPath+"/"+flow, nil).MustStatus(t, 200).JSON(t, &detail)
	if detail.Overdue != 4 || detail.Status != "paused" {
		t.Fatalf("detail: %+v", detail)
	}
	c.Do("POST", wsPath+"/"+flow+"/resume", map[string]any{"overdue": "spread"}).MustStatus(t, 422)
	c.Do("POST", wsPath+"/"+flow+"/resume", map[string]any{"overdue": "skip"}).MustStatus(t, 200)
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if n := countWhere(t, env, `SELECT count(*) FROM messages WHERE automation_id = $1`, flow); n != 0 {
		t.Fatalf("%d emails went out; the overdue ones were to be skipped", n)
	}
	for _, p := range people {
		if nodeID, status, _ := run(t, env, flow, p); nodeID != "tag" || status != "completed" {
			t.Fatalf("run at %s %s, want past the skipped email", nodeID, status)
		}
	}
	if n := countWhere(t, env, `SELECT count(*) FROM audit_events WHERE action = 'automation.resume' AND metadata ->> 'choice' = 'skip' AND (metadata ->> 'overdue')::int = 4`); n != 1 {
		t.Fatalf("resume audit entries: %d", n)
	}

	// Spread: the next ones move on evenly over six hours.
	flow2 := publishFlow(t, c, "Overdue spread", graph)
	for i := range 4 {
		id := addContact(t, env, "spread"+string(rune('a'+i))+"@example.com", nil)
		joinList(t, env, listID(t, env, "Product Updates"), id)
	}
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	c.Do("POST", wsPath+"/"+flow2+"/pause", map[string]any{}).MustStatus(t, 200)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE automation_runs SET next_at = now() - interval '5 minutes' WHERE automation_id = $1`, flow2); err != nil {
		t.Fatal(err)
	}
	c.Do("POST", wsPath+"/"+flow2+"/resume", map[string]any{"overdue": "spread", "spreadHours": 6}).MustStatus(t, 200)
	var first, last time.Time
	var distinct int
	if err := env.App.Pool.QueryRow(ctx, `SELECT min(next_at), max(next_at), count(DISTINCT next_at) FROM automation_runs WHERE automation_id = $1`, flow2).Scan(&first, &last, &distinct); err != nil {
		t.Fatal(err)
	}
	if distinct != 4 || last.Sub(first) < 4*time.Hour || last.Sub(first) > 6*time.Hour {
		t.Fatalf("spread from %v to %v (%d distinct)", first, last, distinct)
	}
}

// AT-11 and AT-6: a flow starts on any of its triggers, re-entry can wait
// N days, and only event triggers can be added to the first.
func TestMoreTriggersAndReentryAfterDays(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	svc := engine(env, &clock{t: time.Now()})
	product := listID(t, env, "Product Updates")
	// A date trigger can't be the second one.
	var created struct{ ID string }
	c.Do("POST", wsPath, map[string]any{"name": "Bad"}).MustStatus(t, 201).JSON(t, &created)
	var saved struct{ Issues []struct{ Code string } }
	c.Do("PUT", wsPath+"/"+created.ID+"/draft", map[string]any{"revision": 0, "graph": automations.AutomationGraph{
		Nodes: []automations.AutomationNode{node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: product,
			Also: []automations.AutomationTrigger{{Kind: automations.TriggerDateField, Field: "created_at"}}}))},
		Edges: []automations.AutomationEdge{},
	}}).MustStatus(t, 200).JSON(t, &saved)
	if !strings.Contains(func() string { b, _ := json.Marshal(saved.Issues); return string(b) }(), "also_kind") {
		t.Fatalf("issues: %+v", saved.Issues)
	}

	flow := publishFlow(t, c, "Two triggers", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: product, Reentry: "days", ReentryDays: 30,
				Also: []automations.AutomationTrigger{{Kind: automations.TriggerTagAdded, Tag: "vip"}}})),
			node("tag", automations.NodeList, tagStep("welcomed")),
		},
		Edges: []automations.AutomationEdge{link("t", "tag", "")},
	})
	var list struct {
		Items []struct{ ID, Trigger string }
	}
	c.Do("GET", wsPath, nil).MustStatus(t, 200).JSON(t, &list)
	for _, it := range list.Items {
		if it.ID == flow && !strings.Contains(it.Trigger, ", or Tag added") {
			t.Fatalf("the list's trigger: %q", it.Trigger)
		}
	}
	joiner := addContact(t, env, "joiner@example.com", nil)
	joinList(t, env, product, joiner)
	vip := addContact(t, env, "vip@example.com", nil)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET tags = array_append(tags, 'vip') WHERE id = $1`, vip); err != nil {
		t.Fatal(err)
	}
	if started, err := svc.Dispatch(ctx); err != nil || started != 2 {
		t.Fatalf("started %d: %v", started, err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	retag := func() {
		if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET tags = array_remove(tags, 'vip') WHERE id = $1`, vip); err != nil {
			t.Fatal(err)
		}
		if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET tags = array_append(tags, 'vip') WHERE id = $1`, vip); err != nil {
			t.Fatal(err)
		}
	}
	retag()
	if started, err := svc.Dispatch(ctx); err != nil || started != 0 {
		t.Fatalf("within 30 days the contact entered again (%d): %v", started, err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE automation_runs SET entered_at = now() - interval '31 days' WHERE automation_id = $1 AND contact_id = $2`, flow, vip); err != nil {
		t.Fatal(err)
	}
	retag()
	if started, err := svc.Dispatch(ctx); err != nil || started != 1 {
		t.Fatalf("after 30 days the contact should enter again (%d): %v", started, err)
	}
}

// AT-8: a Joined list flow can let contacts go when they leave the list;
// a finished run's trigger data goes after 90 days.
func TestLeavingTheListEndsTheRun(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	clk := &clock{t: time.Now()}
	svc := engine(env, clk)
	emailID := newEmail(t, c, "Test · leave")
	product := listID(t, env, "Product Updates")
	flow := publishFlow(t, c, "Leave with the list", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: product, ExitOnLeave: true})),
			node("d", automations.NodeDelay, delayStep(2, "days")),
			node("m", automations.NodeEmail, emailData(emailID)),
		},
		Edges: []automations.AutomationEdge{link("t", "d", ""), link("d", "m", "")},
	})
	leaver := addContact(t, env, "leaver@example.com", nil)
	stayer := addContact(t, env, "stayer@example.com", nil)
	joinList(t, env, product, leaver)
	joinList(t, env, product, stayer)
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE list_memberships SET status = 'unsubscribed' WHERE contact_id = $1 AND list_id = $2`, leaver, product); err != nil {
		t.Fatal(err)
	}
	clk.advance(time.Minute) // the database woke the run at its now()
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	var status, reason string
	if err := env.App.Pool.QueryRow(ctx, `SELECT status, exit_reason FROM automation_runs WHERE automation_id = $1 AND contact_id = $2`, flow, leaver).Scan(&status, &reason); err != nil || status != "exited" || reason != "left_list" {
		t.Fatalf("leaver: %s %s %v", status, reason, err)
	}
	if _, st, _ := run(t, env, flow, stayer); st != "waiting" {
		t.Fatalf("stayer: %s", st)
	}
	// 90 days on, the finished run keeps no trigger data.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE automation_runs SET state = '{"ctx": {"order": "1001"}}', finished_at = now() - interval '91 days' WHERE automation_id = $1 AND contact_id = $2`, flow, leaver); err != nil {
		t.Fatal(err)
	}
	if err := svc.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if _, _, state := run(t, env, flow, leaver); len(state) != 0 {
		t.Fatalf("state kept: %v", state)
	}
}

// AT-9: the workspace's marketing cap holds an automation email back, or
// skips it, and leaves the contact out of a campaign's audience.
func TestMarketingCap(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	svc := engine(env, &clock{t: time.Now()})
	c.Do("PUT", "/workspaces/acme-newsletter/marketing-cap", map[string]any{"count": 51, "days": 7}).MustStatus(t, 422)
	var rules struct {
		MarketingCap struct{ Count, Days int }
	}
	c.Do("PUT", "/workspaces/acme-newsletter/marketing-cap", map[string]any{"count": 1, "days": 7}).MustStatus(t, 200).JSON(t, &rules)
	if rules.MarketingCap.Count != 1 || rules.MarketingCap.Days != 7 {
		t.Fatalf("cap: %+v", rules.MarketingCap)
	}
	if n := countWhere(t, env, `SELECT count(*) FROM audit_events WHERE action = 'marketing_cap.updated'`); n != 1 {
		t.Fatalf("audit: %d", n)
	}
	emailID := newEmail(t, c, "Test · cap")
	product := listID(t, env, "Product Updates")
	mk := func(name, overCap string) string {
		return publishFlow(t, c, name, automations.AutomationGraph{
			Nodes: []automations.AutomationNode{
				node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: product})),
				node("m", automations.NodeEmail, automations.AutomationNodeData{Email: &automations.AutomationEmailStep{EmailID: emailID, EmailVersion: 1, Subject: "Hi", OverCap: overCap}}),
				node("tag", automations.NodeList, tagStep("after-cap")),
			},
			Edges: []automations.AutomationEdge{link("t", "m", ""), link("m", "tag", "")},
		})
	}
	waits := mk("Cap waits", "")
	skips := mk("Cap skips", "skip")
	capped := addContact(t, env, "capped@example.com", nil)
	// A campaign email yesterday: the cap of one is reached.
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO messages (id, workspace_id, contact_id, email, stream, status, queued_at)
		SELECT $1, workspace_id, id, email, 'broadcast', 'delivered', now() - interval '1 day' FROM contacts WHERE id = $2`, uuid.Must(uuid.NewV7()), capped); err != nil {
		t.Fatal(err)
	}
	joinList(t, env, product, capped)
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if nodeID, status, state := run(t, env, waits, capped); nodeID != "m" || status != "waiting" || state["wait"] != "cap" {
		t.Fatalf("waiting flow: %s %s %v", nodeID, status, state)
	}
	if nodeID, status, _ := run(t, env, skips, capped); nodeID != "tag" || status != "completed" {
		t.Fatalf("skipping flow: %s %s", nodeID, status)
	}
	if n := countWhere(t, env, `SELECT count(*) FROM messages WHERE contact_id = $1 AND stream = 'automation'`, capped); n != 0 {
		t.Fatalf("%d automation emails went over the cap", n)
	}
	// Send Readiness counts the contact out.
	ws := uuid.MustParse(func() string {
		var id string
		_ = env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&id)
		return id
	}())
	rows, err := env.App.Q.ReadinessRecipients(ctx, readinessParams(ws, uuid.MustParse(product)))
	if err != nil {
		t.Fatal(err)
	}
	over := 0
	for _, r := range rows {
		if r.ID == capped && r.OverCap {
			over++
		}
	}
	if over != 1 {
		t.Fatal("the capped contact isn't over the cap in Send Readiness")
	}
}

// AT-10: "Enters a segment" starts on contacts who come into a watched
// segment after its first evaluation; "Clicked a link" on a person's
// click whose address matches.
func TestSegmentAndLinkTriggers(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	svc := engine(env, &clock{t: time.Now()})
	seg := &segments.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, SQL: &segments.SQLRunner{Pool: env.App.Pool}}
	svc.SegmentMembers = seg.ContactIDs
	var segment struct{ ID string }
	c.Do("POST", "/workspaces/acme-newsletter/segments", map[string]any{"name": "Beta testers", "mode": "builder", "refresh": "dynamic",
		"definition": map[string]any{"groups": []map[string]any{{"match": "all", "conditions": []map[string]any{{"field": "tag", "op": "has", "value": "beta-tester"}}}}}}).
		MustStatus(t, 201).JSON(t, &segment)
	already := addContact(t, env, "already-beta@example.com", nil)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET tags = ARRAY['beta-tester'] WHERE id = $1`, already); err != nil {
		t.Fatal(err)
	}
	flow := publishFlow(t, c, "Entered beta", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerSegmentEntered, SegmentID: segment.ID})),
			node("tag", automations.NodeList, tagStep("beta-welcomed")),
		},
		Edges: []automations.AutomationEdge{link("t", "tag", "")},
	})
	if err := svc.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if n := countWhere(t, env, `SELECT count(*) FROM automation_events WHERE kind = 'segment_entered'`); n != 0 {
		t.Fatalf("the first evaluation queued %d entries; members already in don't enter", n)
	}
	newcomer := addContact(t, env, "new-beta@example.com", nil)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET tags = ARRAY['beta-tester'] WHERE id = $1`, newcomer); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE segments SET watched_at = now() - interval '11 minutes' WHERE id = $1`, segment.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if started, err := svc.Dispatch(ctx); err != nil || started != 1 {
		t.Fatalf("started %d: %v", started, err)
	}
	if n := countWhere(t, env, `SELECT count(*) FROM automation_runs WHERE automation_id = $1 AND contact_id = $2`, flow, newcomer); n != 1 {
		t.Fatal("the newcomer didn't enter")
	}

	clicks := publishFlow(t, c, "Pricing clickers", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerLinkClicked, URLContains: "/pricing"})),
			node("tag", automations.NodeList, tagStep("pricing-curious")),
		},
		Edges: []automations.AutomationEdge{link("t", "tag", "")},
	})
	ws := func() uuid.UUID {
		var id uuid.UUID
		_ = env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&id)
		return id
	}()
	pricing := addContact(t, env, "pricing@example.com", nil)
	blog := addContact(t, env, "blog@example.com", nil)
	campaign := "campaign:" + uuid.Must(uuid.NewV7()).String()
	if err := automations.EmitLinkClicked(ctx, env.App.Q, ws, pricing, campaign, "https://acme.io/Pricing?plan=team", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := automations.EmitLinkClicked(ctx, env.App.Q, ws, blog, campaign, "https://acme.io/blog", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if n := countWhere(t, env, `SELECT count(*) FROM automation_runs WHERE automation_id = $1`, clicks); n != 1 {
		t.Fatalf("%d entered on a click", n)
	}
	if n := countWhere(t, env, `SELECT count(*) FROM automation_runs WHERE automation_id = $1 AND contact_id = $2`, clicks, pricing); n != 1 {
		t.Fatal("the pricing clicker didn't enter")
	}
}

// AT-7: a date trigger at each contact's own 09:00.
func TestDateTriggersAtTheContactsTime(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO custom_fields (id, workspace_id, key, label, type) SELECT $1, id, 'renewal', 'Renewal', 'date' FROM workspaces WHERE slug = 'acme-newsletter'`, uuid.Must(uuid.NewV7())); err != nil {
		t.Fatal(err)
	}
	// 01:00 UTC: 10:00 in Tokyo, 21:00 the evening before in New York.
	clk := &clock{t: time.Date(2026, 6, 1, 1, 0, 0, 0, time.UTC)}
	svc := engine(env, clk)
	emailID := newEmail(t, c, "Test · renewal")
	publishFlow(t, c, "Renewal", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerDateField, Field: "renewal", At: "09:00", ContactTime: true})),
			node("m", automations.NodeEmail, emailData(emailID)),
		},
		Edges: []automations.AutomationEdge{link("t", "m", "")},
	})
	tokyo := addContact(t, env, "tokyo@example.com", map[string]any{"renewal": "2026-06-01"})
	ny := addContact(t, env, "ny@example.com", map[string]any{"renewal": "2026-06-01"})
	if _, err := env.App.Pool.Exec(ctx, `UPDATE contacts SET timezone = 'Asia/Tokyo' WHERE id = $1`, tokyo); err != nil {
		t.Fatal(err)
	}
	events := func(id uuid.UUID) int64 {
		return countWhere(t, env, `SELECT count(*) FROM automation_events WHERE kind = 'date_field' AND contact_id = $1`, id)
	}
	if err := svc.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if events(tokyo) != 1 || events(ny) != 0 {
		t.Fatalf("at 01:00 UTC: Tokyo %d, New York %d", events(tokyo), events(ny))
	}
	clk.advance(13 * time.Hour) // 10:00 in New York, 1 June
	if err := svc.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if events(tokyo) != 1 || events(ny) != 1 {
		t.Fatalf("at 14:00 UTC: Tokyo %d, New York %d", events(tokyo), events(ny))
	}
}

// AT-5: moving 20,000 waiting contacts to a new version is a few
// statements, not one per contact.
func TestPublishMovesManyContactsQuickly(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	emailID := newEmail(t, c, "Test · many")
	product := listID(t, env, "Product Updates")
	g := automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: product})),
			node("d", automations.NodeDelay, delayStep(3, "days")),
			node("m", automations.NodeEmail, emailData(emailID)),
		},
		Edges: []automations.AutomationEdge{link("t", "d", ""), link("d", "m", "")},
	}
	flow := publishFlow(t, c, "Many", g)
	if _, err := env.App.Pool.Exec(ctx, `
		WITH ws AS (SELECT id FROM workspaces WHERE slug = 'acme-newsletter'),
		ins AS (
			INSERT INTO contacts (id, workspace_id, email)
			SELECT gen_random_uuid(), ws.id, 'many' || n || '@example.com' FROM ws, generate_series(1, 20000) n
			RETURNING id
		)
		INSERT INTO automation_runs (id, automation_id, version, contact_id, node_id, status, state, entered_at, node_entered_at, next_at)
		SELECT gen_random_uuid(), $1, 1, ins.id, 'd', 'waiting', jsonb_build_object('wait', 'delay', 'until', now() + interval '3 days'),
		       now() - interval '1 day', now() - interval '1 day', now() + interval '2 days'
		FROM ins`, flow); err != nil {
		t.Fatal(err)
	}
	g.Nodes[1] = node("d", automations.NodeDelay, delayStep(2, "days"))
	var saved struct{ DraftVersion int }
	var detail struct{ Revision int64 }
	c.Do("GET", wsPath+"/"+flow, nil).MustStatus(t, 200).JSON(t, &detail)
	c.Do("PUT", wsPath+"/"+flow+"/draft", map[string]any{"graph": g, "revision": detail.Revision}).MustStatus(t, 200).JSON(t, &saved)
	start := time.Now()
	c.Do("POST", wsPath+"/"+flow+"/publish", map[string]any{"version": saved.DraftVersion, "migration": "move"}).MustStatus(t, 200)
	if took := time.Since(start); took > 5*time.Second {
		t.Fatalf("publishing moved 20,000 contacts in %v", took)
	}
	if n := countWhere(t, env, `SELECT count(*) FROM automation_runs WHERE automation_id = $1 AND version = $2
		AND abs(extract(epoch FROM next_at - (node_entered_at + interval '2 days'))) < 1`, flow, saved.DraftVersion); n != 20000 {
		t.Fatalf("%d of 20,000 runs moved with the new delay", n)
	}
}

// AT-3: an automation email can be previewed as a contact and sent as a
// test through the System route.
func TestPreviewAndTestSendAnAutomationEmail(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	mail := &systemMail{}
	env.App.Mail = mail
	emailID := newEmail(t, c, "Test · preview")
	c.Do("PUT", "/workspaces/acme-newsletter/automation-emails/"+emailID, map[string]any{"subject": "Hi {{ contact.first_name | default: \"there\" }}", "preheader": "",
		"html": "<p>Hello {{ contact.first_name | default: \"there\" }}</p><a href=\"{{ unsubscribe_url }}\">Unsubscribe</a>"}).MustStatus(t, 200)
	maya := addContact(t, env, "maya@example.com", nil)
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE contacts SET first_name = 'Maya' WHERE id = $1`, maya); err != nil {
		t.Fatal(err)
	}
	var pv struct{ Subject, HTML, Contact string }
	c.Do("POST", "/workspaces/acme-newsletter/automation-emails/"+emailID+"/preview", map[string]any{"contactId": maya}).MustStatus(t, 200).JSON(t, &pv)
	if pv.Subject != "Hi Maya" || !strings.Contains(pv.HTML, "Hello Maya") || pv.Contact != "Maya Test" && pv.Contact != "Maya" {
		t.Fatalf("preview: %+v", pv)
	}
	var res struct {
		Sent     []string
		Provider string
	}
	c.Do("POST", "/workspaces/acme-newsletter/automation-emails/"+emailID+"/test", map[string]any{"to": []string{"linh@acme.io"}, "contactId": maya}).MustStatus(t, 200).JSON(t, &res)
	if len(res.Sent) != 1 || len(mail.sent) != 1 || mail.sent[0].Subject != "[Test] Hi Maya" || strings.Contains(mail.sent[0].HTML, "{{") {
		t.Fatalf("test: %+v %+v", res, mail.sent)
	}
	c.Do("POST", "/workspaces/acme-newsletter/automation-emails/"+emailID+"/test", map[string]any{"to": []string{"not-an-address"}}).MustStatus(t, 422)
	if n := countWhere(t, env, `SELECT count(*) FROM audit_events WHERE action = 'automation.test_send'`); n != 1 {
		t.Fatalf("audit: %d", n)
	}
	_ = http.StatusOK
}

// systemMail is a System-route sender that keeps what it sends.
type systemMail struct {
	mu   sync.Mutex
	sent []mailer.Message
}

func (f *systemMail) SendSystem(_ context.Context, m mailer.Message) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return "Postmark", nil
}

func (f *systemMail) SystemAvailable(context.Context, *uuid.UUID) bool { return true }

func readinessParams(ws, list uuid.UUID) store.ReadinessRecipientsParams {
	now := time.Now()
	return store.ReadinessRecipientsParams{BouncedSince: now.AddDate(0, 0, -30), CapCount: 1, CapSince: now.AddDate(0, 0, -7), WorkspaceID: ws,
		Lists: []uuid.UUID{list}, Contacts: []uuid.UUID{}, ExcludeLists: []uuid.UUID{}, ExcludeContacts: []uuid.UUID{}}
}

// Review 2026-11 AT-15: a watched segment that takes long is checked less
// often and after the quick ones, one that fails backs off, and the
// trigger's options say how often each is checked.
func TestSlowWatchedSegmentsBackOff(t *testing.T) {
	ms := func(v int32) *int32 { return &v }
	for _, c := range []struct {
		ms   *int32
		want time.Duration
	}{{nil, 10 * time.Minute}, {ms(2_000), 10 * time.Minute}, {ms(90_000), 30 * time.Minute}, {ms(600_000), 2 * time.Hour}} {
		if got := automations.WatchInterval(c.ms); got != c.want {
			t.Errorf("WatchInterval(%v) = %v, want %v", c.ms, got, c.want)
		}
	}

	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	clk := &clock{t: time.Now()}
	svc := engine(env, clk)
	var order []string
	failing := ""
	svc.SegmentMembers = func(_ context.Context, _ uuid.UUID, id uuid.UUID) ([]uuid.UUID, error) {
		order = append(order, id.String())
		if id.String() == failing {
			return nil, errors.New("canceling statement due to statement timeout")
		}
		return nil, nil
	}
	mk := func(name string) string {
		var s struct{ ID string }
		c.Do("POST", "/workspaces/acme-newsletter/segments", map[string]any{"name": name, "mode": "builder", "refresh": "dynamic",
			"definition": map[string]any{"groups": []map[string]any{{"match": "all", "conditions": []map[string]any{{"field": "tag", "op": "has", "value": name}}}}}}).
			MustStatus(t, 201).JSON(t, &s)
		publishFlow(t, c, "On "+name, automations.AutomationGraph{
			Nodes: []automations.AutomationNode{
				node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerSegmentEntered, SegmentID: s.ID})),
				node("tag", automations.NodeList, tagStep(name+"-in")),
			},
			Edges: []automations.AutomationEdge{link("t", "tag", "")},
		})
		return s.ID
	}
	slow, quick := mk("slow-sql"), mk("quick")
	// The slow one took 90 seconds last time; both were checked 15 minutes ago.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE segments SET watched_at = $3, watch_ms = CASE WHEN id = $1 THEN 90000 ELSE 40 END WHERE id IN ($1, $2)`,
		slow, quick, clk.now().Add(-15*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := svc.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(order, quick) || slices.Contains(order, slow) {
		t.Fatalf("after 15 minutes: evaluated %v; the quick one is due, the slow one waits 30", order)
	}
	// 31 minutes on, both are due: the quick one goes first.
	order = nil
	if _, err := env.App.Pool.Exec(ctx, `UPDATE segments SET watched_at = $2 WHERE id IN ($1, $3)`, slow, clk.now().Add(-31*time.Minute), quick); err != nil {
		t.Fatal(err)
	}
	if err := svc.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if len(order) < 2 || order[0] != quick || order[1] != slow {
		t.Fatalf("order: %v, want the quick one first", order)
	}
	// A segment that fails is marked checked, and backs off by the time it took.
	order, failing = nil, quick
	if _, err := env.App.Pool.Exec(ctx, `UPDATE segments SET watched_at = $2 WHERE id = $1`, quick, clk.now().Add(-11*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := svc.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	var watched time.Time
	if err := env.App.Pool.QueryRow(ctx, `SELECT watched_at FROM segments WHERE id = $1`, quick).Scan(&watched); err != nil || watched.Before(clk.now().Add(-time.Minute)) {
		t.Fatalf("a failed evaluation isn't marked: %v %v", watched, err)
	}
	var opts struct {
		Segments []struct {
			ID                        string
			CheckMinutes, EvalSeconds int
		}
	}
	c.Do("GET", wsPath+"/options", nil).MustStatus(t, 200).JSON(t, &opts)
	for _, s := range opts.Segments {
		if s.ID == slow && (s.CheckMinutes != 10 || s.EvalSeconds > 1) {
			// Its last evaluation (just now, in the stub) was quick again.
			t.Fatalf("slow segment's option: %+v", s)
		}
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE segments SET watch_ms = 90000 WHERE id = $1`, slow); err != nil {
		t.Fatal(err)
	}
	c.Do("GET", wsPath+"/options", nil).MustStatus(t, 200).JSON(t, &opts)
	for _, s := range opts.Segments {
		if s.ID == slow && (s.CheckMinutes != 30 || s.EvalSeconds != 90) {
			t.Fatalf("slow segment's option: %+v", s)
		}
	}
}
