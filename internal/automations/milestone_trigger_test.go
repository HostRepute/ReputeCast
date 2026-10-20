package automations

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/store"
)

// "Reached a milestone" (Grow): a flow starts on its milestone, or on any
// milestone when none is picked; the list reads the milestone's name.
func TestMilestoneTrigger(t *testing.T) {
	ev := store.ClaimAutomationEventsRow{Kind: TriggerMilestoneReached, Ref: "m3", Payload: []byte(`{"program":"p1","milestone":"m3","count":3}`)}
	if !triggerMatches(AutomationTrigger{Kind: TriggerMilestoneReached, ProgramID: "p1", MilestoneID: "m3"}, ev) {
		t.Fatal("its milestone doesn't start the flow")
	}
	if triggerMatches(AutomationTrigger{Kind: TriggerMilestoneReached, ProgramID: "p1", MilestoneID: "m10"}, ev) {
		t.Fatal("another milestone starts the flow")
	}
	if !triggerMatches(AutomationTrigger{Kind: TriggerMilestoneReached, ProgramID: "p1"}, ev) {
		t.Fatal("any milestone of its program doesn't start the flow")
	}
	if triggerMatches(AutomationTrigger{Kind: TriggerMilestoneReached, ProgramID: "p2"}, ev) {
		t.Fatal("another program's milestone starts the flow")
	}
	if !triggerMatches(AutomationTrigger{Kind: TriggerMilestoneReached}, ev) {
		t.Fatal("any milestone doesn't start the flow")
	}
	refs := &Refs{Milestones: map[string]string{"m3": "3 referrals · The Acme Field Guide"}}
	if got := TriggerLabel(&AutomationTrigger{Kind: TriggerMilestoneReached, MilestoneID: "m3"}, refs); got != "Reached a milestone · 3 referrals · The Acme Field Guide" {
		t.Fatalf("label: %q", got)
	}
	if got := TriggerLabel(&AutomationTrigger{Kind: TriggerMilestoneReached}, refs); got != "Reached a milestone · any milestone" {
		t.Fatalf("label: %q", got)
	}
	g := AutomationGraph{Nodes: []AutomationNode{{ID: "t", Type: NodeTrigger, Data: AutomationNodeData{Trigger: &AutomationTrigger{Kind: TriggerMilestoneReached, MilestoneID: "gone"}}},
		{ID: "x", Type: NodeExit}}, Edges: []AutomationEdge{{ID: "e", Source: "t", Target: "x"}}}
	found := false
	for _, is := range Validate(g, refs) {
		if is.Code == "missing_ref" {
			found = true
		}
	}
	if !found {
		t.Fatal("a deleted milestone isn't reported")
	}
}
