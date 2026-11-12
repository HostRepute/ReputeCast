package automations

import "testing"

// M58: any trigger can stop the flow when the contact replies; a "replied"
// branch needs a window like opened and clicked; the canvas and the diff
// name both.
func TestStopOnReplyAndRepliedCondition(t *testing.T) {
	g := welcomeV4()
	g.Node("n1").Data.Trigger.ExitOnReply = true
	if issues := Errors(Validate(g, welcomeRefs())); len(issues) != 0 {
		t.Fatalf("a flow that stops on reply should be valid: %+v", issues)
	}
	if !g.StopsOnReply() || welcomeV4().StopsOnReply() {
		t.Fatal("StopsOnReply follows the trigger's switch")
	}
	api := AutomationGraph{
		Nodes: []AutomationNode{
			node("t", NodeTrigger, 0, 0, AutomationNodeData{Trigger: &AutomationTrigger{Kind: TriggerAPIEvent, Event: "trial.started", ExitOnReply: true}}),
			node("m", NodeEmail, 0, 100, AutomationNodeData{Email: emailStep("em-1", "You're in. Here's what to expect from Acme", 5)}),
			node("b", NodeBranch, 0, 200, AutomationNodeData{Condition: &AutomationCondition{Kind: CondReplied, Step: "m", Within: dur(2, "days")}}),
			node("f", NodeFilter, 0, 300, AutomationNodeData{Condition: &AutomationCondition{Kind: CondReplied}}),
			node("g", NodeGoal, 0, 400, AutomationNodeData{Condition: &AutomationCondition{Kind: CondReplied}}),
			node("x", NodeExit, 200, 300, AutomationNodeData{}),
		},
		Edges: []AutomationEdge{edge("t", "m", ""), edge("m", "b", ""), edge("b", "f", PathYes), edge("b", "x", PathNo), edge("f", "g", "")},
	}
	if issues := Errors(Validate(api, welcomeRefs())); len(issues) != 0 {
		t.Fatalf("an API event flow that stops on reply should be valid: %+v", issues)
	}
	refs := welcomeRefs()
	for id, want := range map[string]string{
		"b": "Replied to step 1 within 2 days?",
		"f": "Only contacts replied to any email",
		"g": "Replied to an email",
	} {
		if got := NodeTitle(api, *api.Node(id), refs); got != want {
			t.Errorf("%s: %q, want %q", id, got, want)
		}
	}
	if got := say(vi, nodeTitle(api, *api.Node("b"), refs)); got != "Đã trả lời step 1 trong vòng 2 ngày?" {
		t.Errorf("Vietnamese branch title %q", got)
	}

	// A replied branch without a window can't wait for the reply.
	api.Node("b").Data.Condition.Within = nil
	issues := Errors(Validate(api, welcomeRefs()))
	if len(issues) != 1 || issues[0].Code != "invalid_window" || issues[0].Message != "Set how long to wait for the reply." {
		t.Fatalf("branch without a window: %+v", issues)
	}

	// The publish diff says what changed when only the switch did.
	items := Diff(welcomeV4(), g, DiffContext{FromRefs: welcomeRefs(), ToVersion: 5})
	var got []string
	for _, it := range items {
		if it.Kind == "changed" {
			got = append(got, it.Title+" | "+it.Detail)
		}
	}
	want := "Trigger: stops when the contact replies to an email from this flow | It applies to the contacts on v5 once it's live; contacts who stay on an earlier version keep that version's setting."
	if len(got) != 1 || got[0] != want {
		t.Fatalf("diff: %q", got)
	}
	items = Diff(g, welcomeV4(), DiffContext{FromRefs: welcomeRefs(), ToVersion: 6})
	for _, it := range items {
		if it.Kind == "changed" && it.Title != "Trigger: doesn't stop when the contact replies to an email from this flow" {
			t.Fatalf("switching it off: %q", it.Title)
		}
	}
}
