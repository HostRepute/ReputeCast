package automations

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

func TestSendWindow(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	at := func(h, m int) time.Time { return time.Date(2026, 10, 12, h, m, 0, 0, ny) }
	cases := []struct {
		now        time.Time
		start, end string
		inside     bool
		opens      time.Time
	}{
		{at(9, 0), "07:00", "21:00", true, time.Time{}},
		{at(22, 30), "07:00", "21:00", false, at(7, 0).AddDate(0, 0, 1)},
		{at(5, 0), "07:00", "21:00", false, at(7, 0)},
		{at(23, 0), "21:00", "07:00", true, time.Time{}}, // overnight window
		{at(12, 0), "21:00", "07:00", false, at(21, 0)},
	}
	for _, c := range cases {
		opens, inside := nextWindow(c.now, ny, c.start, c.end)
		if inside != c.inside || (!inside && !opens.Equal(c.opens)) {
			t.Errorf("%s in %s–%s: inside %v opens %s", c.now.Format("15:04"), c.start, c.end, inside, opens.In(ny))
		}
	}
}

func TestSplitAssignmentIsStableAndWeighted(t *testing.T) {
	n := FlowNode{ID: "s", Type: NodeSplit, Data: FlowNodeData{Variants: []SplitVariant{{Key: "A", Weight: 80}, {Key: "B", Weight: 20}}}}
	counts := map[string]int{}
	for i := 0; i < 5000; i++ {
		id := uuid.Must(uuid.NewV7())
		v := pickVariant(n, id)
		if pickVariant(n, id) != v {
			t.Fatal("assignment changed for the same run")
		}
		counts[v]++
	}
	if share := float64(counts["A"]) / 5000; math.Abs(share-0.8) > 0.03 {
		t.Fatalf("A got %.1f%%", share*100)
	}
}

func TestFieldConditions(t *testing.T) {
	c := &store.ContactForRunRow{Email: "a@b.co", FirstName: "Ana", Attributes: []byte(`{"plan":"Pro","seats":12}`)}
	for _, tc := range []struct {
		cond FlowCondition
		want bool
	}{
		{FlowCondition{Field: "plan", Op: "equals", Value: "pro"}, true},
		{FlowCondition{Field: "plan", Op: "not_equals", Value: "pro"}, false},
		{FlowCondition{Field: "seats", Op: "equals", Value: "12"}, true},
		{FlowCondition{Field: "first_name", Op: "contains", Value: "an"}, true},
		{FlowCondition{Field: "company", Op: "is_set"}, false},
		{FlowCondition{Field: "company", Op: "is_not_set"}, true},
	} {
		if got := fieldMatches(c, &tc.cond); got != tc.want {
			t.Errorf("%+v: %v", tc.cond, got)
		}
	}
}

func TestFinishEstimateAndMoveImpact(t *testing.T) {
	v3 := welcomeV3()
	inflow := []store.CountInFlowByNodeRow{{Version: 3, NodeID: "n4", N: 168}, {Version: 3, NodeID: "n5", N: 203}, {Version: 3, NodeID: "n11", N: 237}}
	if got := finishEstimate(v3, inflow); got != "about 5 days" {
		t.Fatalf("estimate %q", got)
	}
	items := Diff(v3, welcomeV4(), DiffContext{FromRefs: welcomeRefs(), ToVersion: 4})
	if got := moveImpact(v3, welcomeV4(), items, inflow, welcomeRefs()); got != "203 contacts waiting before step 2 would get the 2-day delay; 0 contacts are affected by the removed webhook." {
		t.Fatalf("impact %q", got)
	}
}

func TestLayoutKeepsBranchesApart(t *testing.T) {
	g := Layout(welcomeV4())
	pos := map[string]FlowPosition{}
	for _, n := range g.Nodes {
		pos[n.ID] = n.Position
	}
	if pos["n5"].X >= pos["n11"].X || pos["n8"].X >= pos["n9"].X || pos["n10"].Y <= pos["n8"].Y || pos["n2"].Y <= pos["n1"].Y {
		t.Fatalf("layout %+v", pos)
	}
}
