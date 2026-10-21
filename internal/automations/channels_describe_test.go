package automations

import "testing"

// The publish diff and approvals name Send SMS steps and the conditions on
// texts as the canvas does (web/src/features/automations/graph.ts).
func TestSMSTitles(t *testing.T) {
	refs := &Refs{Lists: map[string]string{"l1": "Acme Summit 2026"}}
	cond := func(c AutomationCondition) AutomationNodeData { return AutomationNodeData{Condition: &c} }
	g := AutomationGraph{Nodes: []AutomationNode{
		{ID: "s1", Type: NodeSMS, Data: AutomationNodeData{Label: "Day-before reminder", SMS: &AutomationSMSStep{Body: "See you tomorrow!"}}},
		{ID: "s2", Type: NodeSMS, Data: AutomationNodeData{SMS: &AutomationSMSStep{Body: "Hi"}}},
		{ID: "s3", Type: NodeSMS},
		{ID: "b1", Type: NodeBranch, Data: cond(AutomationCondition{Kind: CondSMSConsent, ListID: "l1"})},
		{ID: "b2", Type: NodeBranch, Data: cond(AutomationCondition{Kind: CondSMSClicked, Step: "s1", Within: &AutomationDuration{Amount: 2, Unit: "days"}})},
		{ID: "f1", Type: NodeFilter, Data: cond(AutomationCondition{Kind: CondSMSReplied})},
		{ID: "g1", Type: NodeGoal, Data: cond(AutomationCondition{Kind: CondSMSConsent})},
	}}
	cases := map[string]string{
		"s1": "Day-before reminder",
		"s2": "Send SMS",
		"s3": "Write the text",
		"b1": "Has SMS consent for Acme Summit 2026?",
		"b2": "Clicked a link in day-before reminder within 2 days?",
		"f1": "Only contacts replied to any text",
		"g1": "Has SMS consent",
	}
	for id, want := range cases {
		if got := NodeTitle(g, *g.Node(id), refs); got != want {
			t.Errorf("%s: %q, want %q", id, got, want)
		}
	}
	if got := TypeName(NodeSMS); got != "Send SMS" {
		t.Errorf("type name %q", got)
	}
}
