package automations

import (
	"slices"
	"strings"
	"testing"
)

func TestDiffWelcomeSeriesMatchesThePublishBoard(t *testing.T) {
	items := Diff(welcomeV3(), welcomeV4(), DiffContext{
		FromRefs: welcomeRefs(), ToVersion: 4,
		Failures: map[string]WebhookFailure{"w1": {URL: "https://hooks.slack.com/services/T000/B000/XXXX", Status: 404, Since: sep29, Failures: 1284}},
		Check:    func(string, int) string { return "" },
	})
	want := []DiffItem{
		{Kind: "added", Title: `A/B split after "Step 2", 50 / 50`, Detail: `Two new email steps: 3A "Set up your first flag" and 3B "A 5-minute walkthrough". Both pass Send Readiness content checks.`},
		{Kind: "changed", Title: `Delay before "Step 2": 1 day → 2 days`, Detail: `Contacts already waiting keep their original timer unless you move them to v4 below.`},
		{Kind: "removed", Title: `Webhook "notify-slack" after the trigger`, Detail: `Endpoint https://hooks.slack.com/… was returning 404 since 29 Sep. 1,284 failed calls are in the webhook log.`},
		{Kind: "unchanged", Title: `Trigger, step 1, branch, "No" path, goal and exit`},
	}
	if len(items) != len(want) {
		t.Fatalf("got %d items: %+v", len(items), items)
	}
	for i, w := range want {
		if items[i].Kind != w.Kind || items[i].Title != w.Title || items[i].Detail != w.Detail {
			t.Errorf("item %d:\n got %s | %s | %s\nwant %s | %s | %s", i, items[i].Kind, items[i].Title, items[i].Detail, w.Kind, w.Title, w.Detail)
		}
	}
	if n := Changes(items); n != 3 {
		t.Fatalf("changes = %d, want 3", n)
	}
	// Moving nodes around is not a change.
	moved := welcomeV4()
	for i := range moved.Nodes {
		moved.Nodes[i].Position.X += 40
	}
	if n := Changes(Diff(welcomeV4(), moved, DiffContext{FromRefs: welcomeRefs()})); n != 0 {
		t.Fatalf("moving steps counted as %d changes", n)
	}
}

func TestDiffRewiredBranch(t *testing.T) {
	from := welcomeV4()
	to := welcomeV4()
	for i, e := range to.Edges {
		if e.Source == "n4" && e.Data.Path == PathNo {
			to.Edges[i].Target = "n13"
		}
	}
	items := Diff(from, to, DiffContext{FromRefs: welcomeRefs(), ToVersion: 5})
	var titles []string
	for _, it := range items {
		titles = append(titles, it.Kind+": "+it.Title)
	}
	if !slices.Contains(titles, `changed: After "Opened step 1 within 2 days?", the "No" path goes to "Leave the flow · stays on the list" and the "Yes" path goes to "Wait 2 days"`) &&
		!slices.ContainsFunc(titles, func(s string) bool { return strings.Contains(s, `"No" path goes to "Leave the flow`) }) {
		t.Fatalf("rewire not described: %v", titles)
	}
}

func TestValidateWelcome(t *testing.T) {
	if issues := Errors(Validate(welcomeV4(), welcomeRefs())); len(issues) != 0 {
		t.Fatalf("welcome v4 should be valid: %+v", issues)
	}
	g := welcomeV4()
	// Disconnect the "No" path and orphan its steps.
	g.Edges = slices.DeleteFunc(g.Edges, func(e FlowEdge) bool { return e.Source == "n4" && e.Data.Path == PathNo })
	codes := map[string]bool{}
	for _, i := range Validate(g, welcomeRefs()) {
		codes[i.Code] = true
	}
	for _, c := range []string{"missing_path", "orphan"} {
		if !codes[c] {
			t.Errorf("expected %s in %v", c, codes)
		}
	}
	// A loop back up is refused.
	g = welcomeV4()
	g.Edges = append(g.Edges, edge("n12", "n3", ""))
	codes = map[string]bool{}
	for _, i := range Validate(g, welcomeRefs()) {
		codes[i.Code] = true
	}
	if !codes["cycle"] {
		t.Fatalf("cycle not detected: %v", codes)
	}
	// References to deleted lists and private webhook URLs.
	g = welcomeV4()
	g.Nodes[0].Data.Trigger.ListID = "gone"
	g.Nodes = append(g.Nodes, node("w", NodeWebhook, 0, 0, FlowNodeData{Webhook: &WebhookConfig{URL: "http://169.254.169.254/latest/meta-data"}}))
	g.Edges = append(g.Edges, edge("n13", "w", ""))
	var msgs []string
	for _, i := range Validate(g, welcomeRefs()) {
		msgs = append(msgs, i.Code)
	}
	for _, c := range []string{"missing_ref", "invalid_url", "terminal_output"} {
		if !slices.Contains(msgs, c) {
			t.Errorf("expected %s in %v", c, msgs)
		}
	}
}

func TestTitles(t *testing.T) {
	g := welcomeV4()
	refs := welcomeRefs()
	cases := map[string]string{
		"n1":  "Weekly Digest",
		"n2":  "Wait 10 minutes",
		"n3":  `Welcome · step 1 "You're in"`,
		"n4":  "Opened step 1 within 2 days?",
		"n7":  "50 / 50 on step 3",
		"n10": "Clicked a docs link",
		"n13": "Leave the flow · stays on the list",
	}
	for id, want := range cases {
		if got := NodeTitle(g, *g.Node(id), refs); got != want {
			t.Errorf("%s: %q, want %q", id, got, want)
		}
	}
	if got := TriggerLabel(g.Node("n1").Data.Trigger, refs); got != "Joined list · Weekly Digest" {
		t.Errorf("trigger label %q", got)
	}
	if got := GoalLabel(*g.Node("n10"), refs); got != "Clicked docs" {
		t.Errorf("goal label %q", got)
	}
}

func TestBlockedHosts(t *testing.T) {
	for _, h := range []string{"localhost", "127.0.0.1", "10.1.2.3", "172.20.0.1", "192.168.1.1", "169.254.169.254", "100.64.0.1",
		"::1", "fd00::1", "fe80::1", "[::ffff:127.0.0.1]", "0.0.0.0", "metadata.google.internal", "intranet", "printer.local"} {
		if !blockedHost(h) {
			t.Errorf("%s should be blocked", h)
		}
	}
	for _, h := range []string{"hooks.slack.com", "8.8.8.8", "api.hubspot.com", "2606:4700:4700::1111"} {
		if blockedHost(h) {
			t.Errorf("%s should be allowed", h)
		}
	}
}
