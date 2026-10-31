package automations

import "testing"

// AT-13: the sequence lists the emails top to bottom with how many reached
// the next one, across a branch.
func TestSequenceFollowsTheEmails(t *testing.T) {
	g := AutomationGraph{
		Nodes: []AutomationNode{
			node("t", NodeTrigger, 0, 0, AutomationNodeData{}),
			node("m1", NodeEmail, 0, 0, AutomationNodeData{}),
			node("b", NodeBranch, 0, 0, AutomationNodeData{}),
			node("m2", NodeEmail, 0, 0, AutomationNodeData{}),
			node("d", NodeDelay, 0, 0, AutomationNodeData{}),
			node("m3", NodeEmail, 0, 0, AutomationNodeData{}),
		},
		Edges: []AutomationEdge{edge("t", "m1", ""), edge("m1", "b", ""), edge("b", "m2", "yes"), edge("b", "d", "no"), edge("d", "m3", "")},
	}
	stats := map[string]*AutomationStepStats{
		"m1": {Sent: 100, Delivered: 98, OpenedHuman: 60, Clicked: 14, Unsubscribed: 1},
		"m2": {Entered: 55, Sent: 55},
		"m3": {Entered: 37, Sent: 36},
	}
	seq := sequence(g, stats)
	if len(seq) != 3 || seq[0].NodeID != "m1" || seq[0].Delivered != 98 || seq[0].Next == nil || *seq[0].Next != 92 {
		t.Fatalf("sequence: %+v", seq)
	}
	if seq[1].Next != nil || seq[2].Next != nil {
		t.Fatalf("the last emails of each path have no next: %+v %+v", seq[1], seq[2])
	}
}
