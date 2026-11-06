package automations_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
)

// draftFlow creates a flow as c and saves g as its draft; it returns the
// flow and the draft's version.
func draftFlow(t *testing.T, c *apptest.Client, name string, g automations.AutomationGraph) (string, int) {
	t.Helper()
	var created struct{ ID string }
	c.Do("POST", wsPath, map[string]any{"name": name}).MustStatus(t, 201).JSON(t, &created)
	return created.ID, saveDraft(t, c, created.ID, g)
}

// saveDraft saves g as a flow's draft (whatever was saved since) and
// returns its version.
func saveDraft(t *testing.T, c *apptest.Client, id string, g automations.AutomationGraph) int {
	t.Helper()
	var saved struct {
		DraftVersion int
		Issues       []struct{ Code, Message string }
	}
	c.Do("PUT", wsPath+"/"+id+"/draft", map[string]any{"graph": g, "revision": 0}).MustStatus(t, 200).JSON(t, &saved)
	if len(saved.Issues) > 0 {
		t.Fatalf("flow %s has issues: %+v", id, saved.Issues)
	}
	return saved.DraftVersion
}

func publishAs(c *apptest.Client, id string, version int) *apptest.Response {
	return c.Do("POST", wsPath+"/"+id+"/publish", map[string]any{"version": version, "migration": "finish"})
}

// refusedFor checks a 403 whose problem says want.
func refusedFor(t *testing.T, what, want string, r *apptest.Response) {
	t.Helper()
	b, _ := json.Marshal(r.MustStatus(t, 403).Problem(t))
	if !strings.Contains(string(b), want) {
		t.Fatalf("%s: %s", what, b)
	}
}

// blockedFor checks that the publish dialog shows want as what stops Publish.
func blockedFor(t *testing.T, c *apptest.Client, id, want string) {
	t.Helper()
	var pv struct {
		CanPublish bool
		Blocker    string
	}
	c.Do("GET", wsPath+"/"+id+"/publish-preview", nil).MustStatus(t, 200).JSON(t, &pv)
	if pv.CanPublish || !strings.Contains(pv.Blocker, want) {
		t.Fatalf("publish dialog of %s: %+v", id, pv)
	}
}

// A member limited to some lists publishes and resumes only flows that
// start when a contact joins one of them; such a flow sends on its own.
func TestListScopedMemberPublishesFlowsOnTheirLists(t *testing.T) {
	env, linh := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	svc := engine(env, &clock{t: time.Now()})
	product, beta := listID(t, env, "Product Updates"), listID(t, env, "Beta Program")
	var segment uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT s.id FROM segments s JOIN workspaces w ON w.id = s.workspace_id WHERE w.slug = 'acme-newsletter' LIMIT 1`).Scan(&segment); err != nil {
		t.Fatal(err)
	}
	b, _ := uuid.Parse(beta)
	kim := scopedMember(t, env, authz.RoleEditor, authz.Scope{Lists: []uuid.UUID{b}})
	email := newEmail(t, linh, "Test · scoped flows")
	joined := func(list string) automations.AutomationTrigger {
		return automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: list}
	}
	withEmail := func(tr automations.AutomationTrigger) automations.AutomationGraph {
		return automations.AutomationGraph{
			Nodes: []automations.AutomationNode{node("t", automations.NodeTrigger, trig(tr)), node("d", automations.NodeDelay, delayStep(2, "days")),
				node("m", automations.NodeEmail, emailData(email))},
			Edges: []automations.AutomationEdge{link("t", "d", ""), link("d", "m", "")},
		}
	}
	tagging := func(tr automations.AutomationTrigger) automations.AutomationGraph {
		return automations.AutomationGraph{
			Nodes: []automations.AutomationNode{node("t", automations.NodeTrigger, trig(tr)), node("x", automations.NodeList, tagStep("scoped"))},
			Edges: []automations.AutomationEdge{link("t", "x", "")},
		}
	}
	const listsOnly = "Your access is limited to some lists: a flow you publish or resume can start only when a contact joins one of them."

	// Another list, a tag or a segment let in contacts of other lists:
	// refused, and the publish dialog says so.
	other, v := draftFlow(t, kim, "Scoped · Product Updates", tagging(joined(product)))
	blockedFor(t, kim, other, listsOnly)
	refusedFor(t, "Product Updates", listsOnly, publishAs(kim, other, v))
	tagged, tv := draftFlow(t, kim, "Scoped · tag", tagging(automations.AutomationTrigger{Kind: automations.TriggerTagAdded, Tag: "vip"}))
	refusedFor(t, "a tag", listsOnly, publishAs(kim, tagged, tv))
	either := joined(beta)
	either.Also = []automations.AutomationTrigger{{Kind: automations.TriggerSegmentEntered, SegmentID: segment.String()}}
	segmented, sv := draftFlow(t, kim, "Scoped · Beta Program or a segment", tagging(either))
	refusedFor(t, "Beta Program or a segment", listsOnly, publishAs(kim, segmented, sv))
	refusedIDs := []uuid.UUID{uuid.MustParse(other), uuid.MustParse(tagged), uuid.MustParse(segmented)}
	if n := countWhere(t, env, `SELECT count(*) FROM automations WHERE id = ANY($1::uuid[]) AND live_version IS NOT NULL`, refusedIDs); n != 0 {
		t.Fatalf("%d refused flows went live", n)
	}
	// Linh's access isn't limited: she publishes the first.
	publishAs(linh, other, v).MustStatus(t, 200)

	// A paused flow's waiting contacts go on when it resumes: Kim can't
	// resume Linh's flow while a contact who came in by Product Updates
	// waits in it, though its live version starts on Beta Program.
	shared := publishFlow(t, linh, "Scoped · moved to Beta Program", withEmail(joined(product)))
	waiting := addContact(t, env, "scoped-waiting@example.com", nil)
	joinList(t, env, product, waiting)
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if _, status, _ := run(t, env, shared, waiting); status != "waiting" {
		t.Fatalf("the contact from Product Updates should wait: %s", status)
	}
	publishAs(linh, shared, saveDraft(t, linh, shared, withEmail(joined(beta)))).MustStatus(t, 200)
	linh.Do("POST", wsPath+"/"+shared+"/pause", map[string]any{"reason": "Checking the copy"}).MustStatus(t, 200)
	refusedFor(t, "resume with a contact from Product Updates", listsOnly, kim.Do("POST", wsPath+"/"+shared+"/resume", map[string]any{}))
	var runID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM automation_runs WHERE automation_id = $1 AND contact_id = $2`, shared, waiting).Scan(&runID); err != nil {
		t.Fatal(err)
	}
	linh.Do("POST", wsPath+"/"+shared+"/runs/"+runID.String()+"/remove", map[string]any{"reason": "Not on the beta"}).MustStatus(t, 204)
	kim.Do("POST", wsPath+"/"+shared+"/resume", map[string]any{}).MustStatus(t, 200)

	// Linh's Product Updates flow, paused: Kim can't resume it; Linh can.
	linh.Do("POST", wsPath+"/"+other+"/pause", map[string]any{}).MustStatus(t, 200)
	refusedFor(t, "resume Product Updates", listsOnly, kim.Do("POST", wsPath+"/"+other+"/resume", map[string]any{}))
	linh.Do("POST", wsPath+"/"+other+"/resume", map[string]any{}).MustStatus(t, 200)

	// On Beta Program: Kim publishes it, and it sends with nobody behind it.
	mine, mv := draftFlow(t, kim, "Scoped · Beta Program", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{node("t", automations.NodeTrigger, trig(joined(beta))), node("m", automations.NodeEmail, emailData(email))},
		Edges: []automations.AutomationEdge{link("t", "m", "")},
	})
	publishAs(kim, mine, mv).MustStatus(t, 200)
	tester := addContact(t, env, "scoped-beta@example.com", nil)
	joinList(t, env, beta, tester)
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if n := countWhere(t, env, `SELECT count(*) FROM messages WHERE automation_id = $1 AND contact_id = $2 AND stream = 'automation'`, mine, tester); n != 1 {
		t.Fatalf("Kim's flow queued %d emails", n)
	}
}

// A member limited to some sender domains publishes and resumes only flows
// whose emails go out from one of them: the workspace's default sender's.
// A flow without an email sends from no domain.
func TestSenderScopedMemberPublishesFlowsFromTheirDomains(t *testing.T) {
	env, linh := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	svc := engine(env, &clock{t: time.Now()})
	product := listID(t, env, "Product Updates")
	kim := scopedMember(t, env, authz.RoleEditor, authz.Scope{SenderDomains: []string{"acme.io"}})
	email := newEmail(t, linh, "Test · scoped sender")
	trigger := node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: product}))
	withEmail := automations.AutomationGraph{
		Nodes: []automations.AutomationNode{trigger, node("m", automations.NodeEmail, emailData(email))},
		Edges: []automations.AutomationEdge{link("t", "m", "")},
	}
	const fromNews = "Your access is limited to some sender domains, and this flow's emails go out from news.acme.io, the workspace's default sender."

	sends, v := draftFlow(t, kim, "Scoped · from news.acme.io", withEmail)
	blockedFor(t, kim, sends, fromNews)
	refusedFor(t, "publish", fromNews, publishAs(kim, sends, v))
	quiet, qv := draftFlow(t, kim, "Scoped · no email", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{trigger, node("x", automations.NodeList, tagStep("scoped"))},
		Edges: []automations.AutomationEdge{link("t", "x", "")},
	})
	publishAs(kim, quiet, qv).MustStatus(t, 200)

	// Linh publishes it; paused, Kim can't resume it, Linh can.
	publishAs(linh, sends, v).MustStatus(t, 200)
	linh.Do("POST", wsPath+"/"+sends+"/pause", map[string]any{}).MustStatus(t, 200)
	refusedFor(t, "resume", fromNews, kim.Do("POST", wsPath+"/"+sends+"/resume", map[string]any{}))
	linh.Do("POST", wsPath+"/"+sends+"/resume", map[string]any{}).MustStatus(t, 200)
	if n := countWhere(t, env, `SELECT count(*) FROM audit_events WHERE action = 'automation.resume' AND target_id = $1`, sends); n != 1 {
		t.Fatalf("resumes audited: %d", n)
	}

	// Once the workspace sends from acme.io by default, Kim's flow is hers
	// to publish, and it sends on its own.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE workspaces w SET default_sender_id = s.id FROM senders s
		WHERE s.workspace_id = w.id AND s.email = 'marcus@acme.io' AND w.slug = 'acme-newsletter'`); err != nil {
		t.Fatal(err)
	}
	mine, mv := draftFlow(t, kim, "Scoped · from acme.io", withEmail)
	publishAs(kim, mine, mv).MustStatus(t, 200)
	contact := addContact(t, env, "scoped-sender@example.com", nil)
	joinList(t, env, product, contact)
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if n := countWhere(t, env, `SELECT count(*) FROM messages WHERE automation_id = $1 AND contact_id = $2 AND stream = 'automation'`, mine, contact); n != 1 {
		t.Fatalf("Kim's flow queued %d emails", n)
	}
}
