package automations

import (
	"math"
	"strings"
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
	n := AutomationNode{ID: "s", Type: NodeSplit, Data: AutomationNodeData{Variants: []AutomationSplitVariant{{Key: "A", Weight: 80}, {Key: "B", Weight: 20}}}}
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
	c := &store.AutomationContactRow{Email: "a@b.co", FirstName: "Ana", Attributes: []byte(`{"plan":"Pro","seats":12}`)}
	for _, tc := range []struct {
		cond AutomationCondition
		want bool
	}{
		{AutomationCondition{Field: "plan", Op: "equals", Value: "pro"}, true},
		{AutomationCondition{Field: "plan", Op: "not_equals", Value: "pro"}, false},
		{AutomationCondition{Field: "seats", Op: "equals", Value: "12"}, true},
		{AutomationCondition{Field: "first_name", Op: "contains", Value: "an"}, true},
		{AutomationCondition{Field: "company", Op: "is_set"}, false},
		{AutomationCondition{Field: "company", Op: "is_not_set"}, true},
	} {
		if got := fieldMatches(c, &tc.cond); got != tc.want {
			t.Errorf("%+v: %v", tc.cond, got)
		}
	}
}

func TestHeldDetailNamesEachProvidersProblem(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	now := time.Date(2026, 10, 12, 11, 0, 0, 0, ny)
	today := time.Date(2026, 10, 12, 9, 2, 0, 0, ny)
	route := AutomationRoute{
		Primary:  []AutomationRouteProvider{{Short: "SES", Status: "down", DownSince: &today, DownReason: "outage"}},
		Fallback: &AutomationRouteProvider{Short: "Postmark", Status: "down", DownSince: &today, DownReason: "auth", LastErrorStatus: 401},
	}
	want := "SES has been Down since 09:02 and Postmark, the fallback, returned 401: its API key or token was rejected. 23 emails are queued and will send the moment a provider is healthy; delays in the flow keep counting."
	if got := heldDetail(route, 23, ny, now); got != want {
		t.Fatalf("held detail\n got %q\nwant %q", got, want)
	}
	earlier := today.AddDate(0, 0, -2)
	route = AutomationRoute{Primary: []AutomationRouteProvider{{Short: "SES", Status: "down", DownSince: &earlier}, {Short: "Postal", Status: "paused", Detail: "Disconnected"}}}
	want = "SES has been Down since 10 Oct and Postal is paused (Disconnected). 1 email is queued and will send the moment a provider is healthy; delays in the flow keep counting."
	if got := heldDetail(route, 1, ny, now); got != want {
		t.Fatalf("held detail\n got %q\nwant %q", got, want)
	}
}

func TestFinishEstimateAndMoveImpact(t *testing.T) {
	v3 := welcomeV3()
	inflow := []store.CountAutomationInFlowByNodeRow{{Version: 3, NodeID: "n2", N: 4}, {Version: 3, NodeID: "n4", N: 168}, {Version: 3, NodeID: "n5", N: 440}}
	// From the 10-minute delay: the 2-day branch window, then the 3-day "No" delay.
	if got := finishEstimate(v3, inflow); got != "about 6 days" {
		t.Fatalf("estimate %q", got)
	}
	items := Diff(v3, welcomeV4(), DiffContext{FromRefs: welcomeRefs(), ToVersion: 4})
	// Everyone in the delay or on the way to it (the AutomationPublish board).
	if got := moveImpact(v3, welcomeV4(), items, inflow, welcomeRefs()); got != "612 contacts waiting before step 2 would get the 2-day delay; 0 contacts are affected by the removed webhook." {
		t.Fatalf("impact %q", got)
	}
	inflow = append(inflow, store.CountAutomationInFlowByNodeRow{Version: 3, NodeID: "n11", N: 9})
	if got := moveImpact(v3, welcomeV4(), items, inflow, welcomeRefs()); !strings.HasPrefix(got, "612 contacts") {
		t.Fatalf("contacts on the other path count: %q", got)
	}
}

func TestLayoutKeepsBranchesApart(t *testing.T) {
	g := Layout(welcomeV4())
	pos := map[string]AutomationNodePosition{}
	for _, n := range g.Nodes {
		pos[n.ID] = n.Position
	}
	if pos["n5"].X >= pos["n11"].X || pos["n8"].X >= pos["n9"].X || pos["n10"].Y <= pos["n8"].Y || pos["n2"].Y <= pos["n1"].Y {
		t.Fatalf("layout %+v", pos)
	}
}

// A step stays demo data only while its address is unchanged, whatever the
// client sends.
func TestKeepDemo(t *testing.T) {
	before := AutomationGraph{Nodes: []AutomationNode{
		{ID: "a", Type: NodeWebhook, Data: AutomationNodeData{Webhook: &AutomationWebhook{URL: "https://api.hubapi.com/x", Demo: true}}},
		{ID: "b", Type: NodeWebhook, Data: AutomationNodeData{Webhook: &AutomationWebhook{URL: "https://api.hubapi.com/y", Demo: true}}},
		{ID: "c", Type: NodeWebhook, Data: AutomationNodeData{Webhook: &AutomationWebhook{URL: "https://mine.example/z"}}},
	}}
	g := keepDemo(AutomationGraph{Nodes: []AutomationNode{
		{ID: "a", Type: NodeWebhook, Data: AutomationNodeData{Webhook: &AutomationWebhook{URL: "https://api.hubapi.com/x"}}},
		{ID: "b", Type: NodeWebhook, Data: AutomationNodeData{Webhook: &AutomationWebhook{URL: "https://mine.example/y"}}},
		{ID: "c", Type: NodeWebhook, Data: AutomationNodeData{Webhook: &AutomationWebhook{URL: "https://mine.example/z", Demo: true}}},
	}}, before)
	if !g.Nodes[0].Data.Webhook.Demo || g.Nodes[1].Data.Webhook.Demo || g.Nodes[2].Data.Webhook.Demo {
		t.Fatalf("demo marks: %v %v %v", g.Nodes[0].Data.Webhook.Demo, g.Nodes[1].Data.Webhook.Demo, g.Nodes[2].Data.Webhook.Demo)
	}
}
