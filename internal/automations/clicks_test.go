package automations_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// TestClickConditionsCountPeopleOnly: an "If clicked" branch and a
// "clicked" goal only follow a person's click (review AN-2). A security
// scanner that follows every link of the email clicks first; its clicks
// wait to settle and then settle as a bot's, so they neither wake the
// branch nor meet the goal, and the contact takes the "No" path when the
// window closes. A person's click counts once it settles.
func TestClickConditionsCountPeopleOnly(t *testing.T) {
	env, c := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	clk := &clock{t: time.Now()}
	svc := engine(env, clk)
	tr, ok := env.App.SendingEngine().Recorder.(*tracking.Service)
	if !ok {
		t.Fatalf("recorder %T", env.App.SendingEngine().Recorder)
	}
	product := listID(t, env, "Product Updates")
	trigger := &automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: product, ConfirmedOnly: true}
	branchFlow := publishFlow(t, c, "Clicked branch", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, automations.AutomationNodeData{Trigger: trigger}),
			node("m", automations.NodeEmail, automations.AutomationNodeData{Email: &automations.AutomationEmailStep{EmailID: newEmail(t, c, "Clicks · branch"), EmailVersion: 1,
				Subject: "Clicks · branch"}}),
			node("b", automations.NodeBranch, automations.AutomationNodeData{Condition: &automations.AutomationCondition{Kind: "clicked", Step: "m",
				Within: &automations.AutomationDuration{Amount: 1, Unit: "hours"}}}),
			node("yes", automations.NodeList, automations.AutomationNodeData{Action: &automations.AutomationListAction{Kind: "add_tag", Tag: "clicked-yes"}}),
			node("no", automations.NodeList, automations.AutomationNodeData{Action: &automations.AutomationListAction{Kind: "add_tag", Tag: "clicked-no"}}),
		},
		Edges: []automations.AutomationEdge{link("t", "m", ""), link("m", "b", ""), link("b", "yes", "yes"), link("b", "no", "no")},
	})
	goalFlow := publishFlow(t, c, "Clicked goal", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, automations.AutomationNodeData{Trigger: trigger}),
			node("m", automations.NodeEmail, automations.AutomationNodeData{Email: &automations.AutomationEmailStep{EmailID: newEmail(t, c, "Clicks · goal"), EmailVersion: 1,
				Subject: "Clicks · goal"}}),
			node("d", automations.NodeDelay, automations.AutomationNodeData{Delay: &automations.AutomationDuration{Amount: 3, Unit: "days"}}),
			node("g", automations.NodeGoal, automations.AutomationNodeData{Label: "Clicked", Condition: &automations.AutomationCondition{Kind: "clicked"}}),
		},
		Edges: []automations.AutomationEdge{link("t", "m", ""), link("m", "d", ""), link("d", "g", "")},
	})
	scanned := addContact(t, env, "scanned@example.com", nil)
	person := addContact(t, env, "person@example.com", nil)
	for _, id := range []uuid.UUID{scanned, person} {
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO list_memberships (list_id, contact_id, status, source, subscribed_at) VALUES ($1,$2,'subscribed','form:test',now())`, product, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	message := func(flow string, contact uuid.UUID) uuid.UUID {
		t.Helper()
		var id uuid.UUID
		if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM messages WHERE automation_id = $1 AND contact_id = $2`, flow, contact).Scan(&id); err != nil {
			t.Fatalf("the flow's email to %s: %v", contact, err)
		}
		return id
	}
	if node, status, _ := run(t, env, branchFlow, scanned); node != "b" || status != "waiting" {
		t.Fatalf("the scanned contact waits at the branch, is at %s %s", node, status)
	}
	// A scanner follows three links of each email within two seconds: each
	// click looks like a person's on its own, so it waits to settle.
	at := clk.now().Add(5 * time.Minute)
	const chrome = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"
	var burst []sending.Engagement
	for _, flow := range []string{branchFlow, goalFlow} {
		id := message(flow, scanned)
		for i := range 3 {
			burst = append(burst, sending.Engagement{MessageID: id, Click: true, Link: i, UserAgent: chrome, IP: "198.51.100.40", Method: "GET",
				At: at.Add(time.Duration(i) * time.Second)})
		}
	}
	if err := tr.RecordEngagement(ctx, burst); err != nil {
		t.Fatal(err)
	}
	var pending int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM delivery_events WHERE contact_id = $1 AND type = 'click' AND class = 'pending' AND NOT machine`, scanned).Scan(&pending); err != nil || pending != 6 {
		t.Fatalf("%d of the scanner's clicks wait to settle: %v", pending, err)
	}
	scanAndTick := func() {
		t.Helper()
		if err := svc.Scan(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Tick(ctx); err != nil {
			t.Fatal(err)
		}
	}
	// Clicks still waiting to settle wake no branch and meet no goal.
	scanAndTick()
	if node, status, _ := run(t, env, branchFlow, scanned); node != "b" || status != "waiting" {
		t.Fatalf("an unsettled click moved the branch: %s %s", node, status)
	}
	if node, status, _ := run(t, env, goalFlow, scanned); node == "g" || status == "goal" {
		t.Fatalf("an unsettled click met the goal: %s %s", node, status)
	}
	// Settled: the burst is the scanner's, filed with the bot clicks.
	if _, err := tr.SettleEngagement(ctx, at.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	var bots int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM delivery_events WHERE contact_id = $1 AND type = 'click' AND class = 'bot' AND machine`, scanned).Scan(&bots); err != nil || bots != 6 {
		t.Fatalf("%d of the scanner's clicks settled as a bot's: %v", bots, err)
	}
	scanAndTick()
	if node, status, _ := run(t, env, branchFlow, scanned); node != "b" || status != "waiting" {
		t.Fatalf("a bot's click moved the branch: %s %s", node, status)
	}
	if node, status, _ := run(t, env, goalFlow, scanned); node == "g" || status == "goal" {
		t.Fatalf("a bot's click met the goal: %s %s", node, status)
	}
	// A person's click counts once it settles: the branch says yes and the
	// goal is met.
	var person1 []sending.Engagement
	for _, flow := range []string{branchFlow, goalFlow} {
		person1 = append(person1, sending.Engagement{MessageID: message(flow, person), Click: true, Link: 0, UserAgent: chrome, IP: "198.51.100.41", Method: "GET", At: at})
	}
	if err := tr.RecordEngagement(ctx, person1); err != nil {
		t.Fatal(err)
	}
	scanAndTick()
	if node, _, _ := run(t, env, branchFlow, person); node != "b" {
		t.Fatalf("a person's click counted before it settled: at %s", node)
	}
	if _, err := tr.SettleEngagement(ctx, at.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	scanAndTick()
	if node, status, _ := run(t, env, branchFlow, person); node != "yes" || status != "completed" {
		t.Fatalf("a person's click: the branch went to %s %s, want yes", node, status)
	}
	if node, status, _ := run(t, env, goalFlow, person); node != "g" || status != "goal" {
		t.Fatalf("a person's click: the goal flow is at %s %s", node, status)
	}
	// The window closes: the scanned contact never clicked.
	clk.advance(61 * time.Minute)
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if node, status, _ := run(t, env, branchFlow, scanned); node != "no" || status != "completed" {
		t.Fatalf("the scanned contact: %s %s, want the No path", node, status)
	}
	var tags []string
	if err := env.App.Pool.QueryRow(ctx, `SELECT tags FROM contacts WHERE id = $1`, scanned).Scan(&tags); err != nil || slices.Contains(tags, "clicked-yes") || !slices.Contains(tags, "clicked-no") {
		t.Fatalf("the scanned contact's tags: %v %v", tags, err)
	}
	// The contact's profile counts the person's click only.
	var profile struct {
		Stats struct{ Clicks int64 }
	}
	for _, x := range []struct {
		id   uuid.UUID
		want int64
	}{{scanned, 0}, {person, 2}} {
		c.Do("GET", "/workspaces/acme-newsletter/contacts/"+x.id.String(), nil).MustStatus(t, 200).JSON(t, &profile)
		if profile.Stats.Clicks != x.want {
			t.Fatalf("contact %s: the profile counts %d clicked emails, want %d", x.id, profile.Stats.Clicks, x.want)
		}
	}
}
