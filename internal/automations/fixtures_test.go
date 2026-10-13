package automations

import "time"

// The Welcome series from the design: v3 is live, v4 is the draft on the
// AutomationFlow board, and AutomationPublish shows the diff between them.

func welcomeRefs() *Refs {
	return &Refs{
		Lists: map[string]string{"list-digest": "Weekly Digest"},
		Forms: map[string]string{},
		Emails: map[string]int{
			"em-1": 5, "em-2": 2, "em-3a": 1, "em-3b": 1, "em-pt": 1,
		},
		EmailNames: map[string]string{
			"em-1":  `Welcome · step 1 "You're in"`,
			"em-2":  `Step 2 "Three things to try first"`,
			"em-3a": `Step 3 "Set up your first flag"`,
			"em-3b": `Step 3 "A 5-minute walkthrough"`,
			"em-pt": `"Did you get our welcome?" · plain text`,
		},
		Fields:    map[string]string{},
		Providers: map[string]string{"prov-postmark": "Postmark"},
	}
}

func dur(n int, unit string) *AutomationDuration { return &AutomationDuration{Amount: n, Unit: unit} }

func emailStep(id, subject string, version int) *AutomationEmailStep {
	return &AutomationEmailStep{EmailID: id, EmailVersion: version, Subject: subject,
		SendWindow: &AutomationSendWindow{Enabled: true, Start: "07:00", End: "21:00"}, SkipRecentHours: 4}
}

func node(id, typ string, x, y float64, d AutomationNodeData) AutomationNode {
	return AutomationNode{ID: id, Type: typ, Position: AutomationNodePosition{X: x, Y: y}, Data: d}
}

func edge(src, dst, path string) AutomationEdge {
	id := src + "-" + dst
	return AutomationEdge{ID: id, Source: src, Target: dst, Data: AutomationEdgeData{Path: path}}
}

func welcomeV4() AutomationGraph {
	return AutomationGraph{
		Nodes: []AutomationNode{
			node("n1", NodeTrigger, 370, 24, AutomationNodeData{Trigger: &AutomationTrigger{Kind: TriggerJoinedList, ListID: "list-digest", ConfirmedOnly: true}}),
			node("n2", NodeDelay, 370, 144, AutomationNodeData{Delay: dur(10, "minutes")}),
			node("n3", NodeEmail, 370, 215, AutomationNodeData{Email: emailStep("em-1", "You're in. Here's what to expect from Acme", 5)}),
			node("n4", NodeBranch, 370, 399, AutomationNodeData{Condition: &AutomationCondition{Kind: "opened", Step: "n3", Within: dur(2, "days"), HumanOnly: true}}),
			node("n5", NodeDelay, 150, 531, AutomationNodeData{Delay: dur(2, "days")}),
			node("n6", NodeEmail, 150, 623, AutomationNodeData{Email: emailStep("em-2", "Three things to try first", 2)}),
			node("n7", NodeSplit, 150, 719, AutomationNodeData{Variants: []AutomationSplitVariant{{Key: "A", Weight: 50}, {Key: "B", Weight: 50}}}),
			node("n8", NodeEmail, 30, 816, AutomationNodeData{Email: emailStep("em-3a", "Set up your first flag", 1)}),
			node("n9", NodeEmail, 270, 816, AutomationNodeData{Email: emailStep("em-3b", "A 5-minute walkthrough", 1)}),
			node("n10", NodeGoal, 150, 932, AutomationNodeData{Label: "Clicked docs", Condition: &AutomationCondition{Kind: "clicked", URLContains: "docs"}}),
			node("n11", NodeDelay, 590, 531, AutomationNodeData{Delay: dur(3, "days")}),
			node("n12", NodeEmail, 590, 623, AutomationNodeData{Email: emailStep("em-pt", "Did you get our welcome?", 1)}),
			node("n13", NodeExit, 590, 735, AutomationNodeData{}),
		},
		Edges: []AutomationEdge{
			edge("n1", "n2", ""), edge("n2", "n3", ""), edge("n3", "n4", ""),
			edge("n4", "n5", PathYes), edge("n4", "n11", PathNo),
			edge("n5", "n6", ""), edge("n6", "n7", ""),
			edge("n7", "n8", "A"), edge("n7", "n9", "B"),
			edge("n8", "n10", ""), edge("n9", "n10", ""),
			edge("n11", "n12", ""), edge("n12", "n13", ""),
		},
	}
}

func welcomeV3() AutomationGraph {
	g := welcomeV4()
	var nodes []AutomationNode
	for _, n := range g.Nodes {
		switch n.ID {
		case "n7", "n8", "n9":
			continue
		case "n5":
			n.Data.Delay = dur(1, "days")
		}
		nodes = append(nodes, n)
	}
	nodes = append(nodes, node("w1", NodeWebhook, 370, 144, AutomationNodeData{Webhook: &AutomationWebhook{Name: "notify-slack", URL: "https://hooks.slack.com/services/T000/B000/XXXX", Method: "POST"}}))
	g.Nodes = nodes
	g.Edges = []AutomationEdge{
		edge("n1", "w1", ""), edge("w1", "n2", ""), edge("n2", "n3", ""), edge("n3", "n4", ""),
		edge("n4", "n5", PathYes), edge("n4", "n11", PathNo),
		edge("n5", "n6", ""), edge("n6", "n10", ""),
		edge("n11", "n12", ""), edge("n12", "n13", ""),
	}
	return g
}

var sep29 = time.Date(2026, 9, 29, 14, 0, 0, 0, time.UTC)
