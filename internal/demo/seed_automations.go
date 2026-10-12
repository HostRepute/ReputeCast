package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/automations"
)

func init() { seedAutomations = seedAutomationsImpl }

// The automations of the design (AutomationsList, AutomationFlow,
// AutomationPublish, AutomationsStates): the Welcome series live on v3 with
// the v4 draft whose diff the publish dialog shows, Onboarding and
// Re-engagement live, the Re-permission draft and the paused Beta invite,
// with 30 days of per-step stats and the contacts in each flow right now.

type demoEmail struct {
	key, name, subject, preheader, heading, cta string
	versions                                    int
	body                                        []string
}

var demoEmails = []demoEmail{
	{key: "w1", name: `Welcome · step 1 "You're in"`, subject: "You're in. Here's what to expect from Acme", preheader: "One email a week, and how to make the most of it.", heading: "You're in", cta: "Read the getting-started guide", versions: 5,
		body: []string{"Thanks for subscribing to the Weekly Digest. Every Tuesday: product news, a guide worth your time and the changelog.", "Start with the guide most people find useful first."}},
	{key: "w2", name: `Step 2 "Three things to try first"`, subject: "Three things to try first", preheader: "Small wins for your first week.", heading: "Three things to try first", cta: "Try feature flags", versions: 2,
		body: []string{"Feature flags, audit log exports and the new environments view: here's where to start."}},
	{key: "w3a", name: `Step 3 "Set up your first flag"`, subject: "Set up your first flag in 3 minutes", preheader: "A short walkthrough.", heading: "Set up your first flag", cta: "Open the docs", versions: 1,
		body: []string{"Create a flag, target a segment and roll it out to 10% of traffic."}},
	{key: "w3b", name: `Step 3 "A 5-minute walkthrough"`, subject: "A 5-minute walkthrough", preheader: "Watch it, then try it.", heading: "A 5-minute walkthrough", cta: "Watch the walkthrough", versions: 1,
		body: []string{"Five minutes, one flag, zero deploys."}},
	{key: "wpt", name: `"Did you get our welcome?" · plain text`, subject: "Did you get our welcome?", preheader: "", heading: "Did you get our welcome?", cta: "Read the guide", versions: 1,
		body: []string{"Our first email may have landed in another tab. Here's the short version, in plain text."}},
	{key: "o1", name: `Onboarding · step 1 "Welcome to Acme"`, subject: "Welcome to Acme", preheader: "Your workspace is ready.", heading: "Welcome to Acme", cta: "Open your workspace", versions: 3, body: []string{"Your workspace is ready. Here's the one thing to do first."}},
	{key: "o2", name: `Step 2 "Create your first flag"`, subject: "Create your first flag", preheader: "It takes about two minutes.", heading: "Create your first flag", cta: "Create a flag", versions: 2, body: []string{"Flags hold everything else. Create one to see how it fits together."}},
	{key: "o3", name: `Step 3 "Next: environments"`, subject: "Nice work. Next: environments", preheader: "Separate staging from production.", heading: "Next: environments", cta: "Set up environments", versions: 1, body: []string{"Environments keep staging and production apart."}},
	{key: "o4", name: `Step 3 "Need a hand with your first flag?"`, subject: "Need a hand with your first flag?", preheader: "Reply and a human answers.", heading: "Need a hand?", cta: "Book 15 minutes", versions: 1, body: []string{"Stuck on your first flag? Reply to this email; a person on our team answers."}},
	{key: "o5", name: `Step 4 "Invite your team"`, subject: "Invite your team", preheader: "Flags work better together.", heading: "Invite your team", cta: "Invite a teammate", versions: 2, body: []string{"Teams that invite a second person in the first week stay three times longer."}},
	{key: "o6", name: `Step 5 "Your first week"`, subject: "Your first week with Acme", preheader: "What teams like yours do next.", heading: "Your first week", cta: "Read the guide", versions: 1, body: []string{"A week in, here's what teams like yours do next."}},
	{key: "r1", name: `Re-engagement "Still want these?"`, subject: "Still want these?", preheader: "One click keeps you on the list.", heading: "Still want these?", cta: "Yes, keep me subscribed", versions: 2,
		body: []string{"We haven't seen you open our emails in a while. If you'd like to keep getting them, tell us with one click."}},
	{key: "p1", name: `Re-permission "Still want to hear from Acme?"`, subject: "Still want to hear from Acme?", preheader: "Confirm and we'll keep you on the list.", heading: "Still want to hear from us?", cta: "Yes, confirm", versions: 1,
		body: []string{"You moved with us from Mailchimp. Confirm you still want our emails; if you don't, we'll stop sending in 14 days."}},
	{key: "b1", name: `Beta invite "You're in the beta"`, subject: "You're in the beta", preheader: "Accept within 3 days to keep your spot.", heading: "You're in the beta", cta: "Accept the invite", versions: 1,
		body: []string{"Thanks for applying. Your spot is held for 3 days."}},
	{key: "b2", name: `Beta reminder "Your invite expires tomorrow"`, subject: "Your beta invite expires tomorrow", preheader: "Accept to keep your spot.", heading: "Your invite expires tomorrow", cta: "Accept the invite", versions: 1,
		body: []string{"Your beta spot is held until tomorrow."}},
}

type flowNode = automations.FlowNode
type flowData = automations.FlowNodeData

func fnode(id, typ string, x, y float64, d flowData) flowNode {
	return flowNode{ID: id, Type: typ, Position: automations.FlowPosition{X: x, Y: y}, Data: d}
}

func fedge(src, dst, path string) automations.FlowEdge {
	return automations.FlowEdge{ID: "e-" + src + "-" + dst, Source: src, Target: dst, Data: automations.FlowEdgeData{Path: path}}
}

func fdur(n int, unit string) *automations.FlowDuration {
	return &automations.FlowDuration{Amount: n, Unit: unit}
}

func seedAutomationsImpl(ctx context.Context, s *Seeder) error {
	ws := s.WS()
	linh, marcus := s.Users["linh@acme.io"], s.Users["marcus@acme.io"]
	digest := s.Lists["Weekly Digest"]

	// Emails used by the send steps, each with its version history.
	emails := map[string]uuid.UUID{}
	emailVersion := map[string]int{}
	for i, e := range demoEmails {
		id := ID()
		emails[e.key] = id
		emailVersion[e.key] = e.versions
		created := s.Ago(time.Duration(60-i) * 24 * time.Hour)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO automation_emails (id, workspace_id, name, latest_version, created_by, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$6)`,
			id, ws.ID, e.name, e.versions, linh.ID, created); err != nil {
			return err
		}
		for v := 1; v <= e.versions; v++ {
			c := automations.DemoEmail(e.subject, e.preheader, e.heading, e.cta, e.body...)
			if _, err := s.Tx.Exec(ctx, `INSERT INTO automation_email_versions (email_id, version, subject, preheader, mjml, html, plain_text, created_by, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
				id, v, c.Subject, c.Preheader, c.MJML, c.HTML, c.Text, linh.ID, created.Add(time.Duration(v)*24*time.Hour)); err != nil {
				return err
			}
		}
	}
	email := func(key, subject string) *automations.EmailStepConfig {
		return &automations.EmailStepConfig{EmailID: emails[key].String(), EmailVersion: emailVersion[key], Subject: subject,
			SendWindow: &automations.SendWindow{Enabled: true, Start: "07:00", End: "21:00"}, SkipRecentHours: 4}
	}

	// Welcome series: v4 is the draft on the AutomationFlow board.
	welcomeV4 := automations.FlowGraph{
		Nodes: []flowNode{
			fnode("n1", automations.NodeTrigger, 370, 24, flowData{Trigger: &automations.TriggerConfig{Kind: automations.TriggerJoinedList, ListID: digest.String(), ConfirmedOnly: true, Reentry: "once"}}),
			fnode("n2", automations.NodeDelay, 370, 144, flowData{Delay: fdur(10, "minutes")}),
			fnode("n3", automations.NodeEmail, 370, 215, flowData{Email: email("w1", "You're in. Here's what to expect from Acme")}),
			fnode("n4", automations.NodeBranch, 370, 399, flowData{Condition: &automations.FlowCondition{Kind: "opened", Step: "n3", Within: fdur(2, "days"), HumanOnly: true}}),
			fnode("n5", automations.NodeDelay, 150, 531, flowData{Delay: fdur(2, "days")}),
			fnode("n6", automations.NodeEmail, 150, 623, flowData{Email: email("w2", "Three things to try first")}),
			fnode("n7", automations.NodeSplit, 150, 719, flowData{Variants: []automations.SplitVariant{{Key: "A", Weight: 50}, {Key: "B", Weight: 50}}}),
			fnode("n8", automations.NodeEmail, 30, 816, flowData{Email: email("w3a", "Set up your first flag in 3 minutes")}),
			fnode("n9", automations.NodeEmail, 270, 816, flowData{Email: email("w3b", "A 5-minute walkthrough")}),
			fnode("n10", automations.NodeGoal, 150, 932, flowData{Label: "Clicked docs", Condition: &automations.FlowCondition{Kind: "clicked", URLContains: "docs"}}),
			fnode("n11", automations.NodeDelay, 590, 531, flowData{Delay: fdur(3, "days")}),
			fnode("n12", automations.NodeEmail, 590, 623, flowData{Email: email("wpt", "Did you get our welcome?")}),
			fnode("n13", automations.NodeExit, 590, 735, flowData{}),
		},
		Edges: []automations.FlowEdge{
			fedge("n1", "n2", ""), fedge("n2", "n3", ""), fedge("n3", "n4", ""),
			fedge("n4", "n5", automations.PathYes), fedge("n4", "n11", automations.PathNo),
			fedge("n5", "n6", ""), fedge("n6", "n7", ""),
			fedge("n7", "n8", "A"), fedge("n7", "n9", "B"),
			fedge("n8", "n10", ""), fedge("n9", "n10", ""),
			fedge("n11", "n12", ""), fedge("n12", "n13", ""),
		},
	}
	webhook := fnode("w1", automations.NodeWebhook, 370, 144, flowData{Webhook: &automations.WebhookConfig{Name: "notify-slack", URL: "https://hooks.slack.com/services/T024BE7LD/B05ACME01/notify-signups", Method: "POST"}})
	welcomeV3 := withoutNodes(welcomeV4, "n7", "n8", "n9")
	for i := range welcomeV3.Nodes {
		if welcomeV3.Nodes[i].ID == "n5" {
			welcomeV3.Nodes[i].Data.Delay = fdur(1, "days")
		}
	}
	welcomeV3.Nodes = append(welcomeV3.Nodes, webhook)
	welcomeV3.Edges = []automations.FlowEdge{
		fedge("n1", "w1", ""), fedge("w1", "n2", ""), fedge("n2", "n3", ""), fedge("n3", "n4", ""),
		fedge("n4", "n5", automations.PathYes), fedge("n4", "n11", automations.PathNo),
		fedge("n5", "n6", ""), fedge("n6", "n10", ""), fedge("n11", "n12", ""), fedge("n12", "n13", ""),
	}
	welcomeV3 = automations.Layout(welcomeV3)
	welcomeV2 := withoutNodes(welcomeV3, "n11", "n12")
	welcomeV2.Edges = []automations.FlowEdge{
		fedge("n1", "w1", ""), fedge("w1", "n2", ""), fedge("n2", "n3", ""), fedge("n3", "n4", ""),
		fedge("n4", "n5", automations.PathYes), fedge("n4", "n13", automations.PathNo),
		fedge("n5", "n6", ""), fedge("n6", "n10", ""),
	}
	welcomeV2 = automations.Layout(welcomeV2)
	welcomeV1 := withoutNodes(welcomeV2, "n2")
	welcomeV1.Edges = []automations.FlowEdge{
		fedge("n1", "w1", ""), fedge("w1", "n3", ""), fedge("n3", "n4", ""),
		fedge("n4", "n5", automations.PathYes), fedge("n4", "n13", automations.PathNo),
		fedge("n5", "n6", ""), fedge("n6", "n10", ""),
	}
	welcomeV1 = automations.Layout(welcomeV1)

	welcome := ID()
	if err := insertFlow(ctx, s, flowSeed{
		id: welcome, name: "Welcome series", description: "4 emails over 9 days", status: "live", template: "welcome-series",
		live: 3, created: s.designDay(24, 10, 4), createdBy: linh.ID, updated: s.Ago(47 * time.Minute),
		versions: []versionSeed{
			{graph: welcomeV1, by: linh.ID, created: s.designDay(24, 10, 4), published: ptrTime(s.designDay(24, 10, 20)), retired: ptrTime(s.designDay(24, 16, 2)), summary: "created from the Welcome series template"},
			{graph: welcomeV2, by: linh.ID, created: s.designDay(24, 15, 40), published: ptrTime(s.designDay(24, 16, 2)), retired: ptrTime(s.designDay(1, 9, 30)), summary: "delay 10 min before step 1", migration: "finish"},
			{graph: welcomeV3, by: marcus.ID, created: s.designDay(10, 11, 5), published: ptrTime(s.designDay(10, 11, 32)), summary: `added "No" path with plain-text follow-up`, migration: "finish"},
			{graph: welcomeV4, by: linh.ID, created: s.Ago(3 * time.Hour)},
		},
	}); err != nil {
		return err
	}

	// Onboarding · new workspace: v7 live since 28 Sep.
	onboarding := automations.FlowGraph{
		Nodes: []flowNode{
			fnode("n1", automations.NodeTrigger, 0, 0, flowData{Trigger: &automations.TriggerConfig{Kind: automations.TriggerAPIEvent, Event: "workspace.created", Reentry: "once"}}),
			fnode("n2", automations.NodeEmail, 0, 0, flowData{Email: email("o1", "Welcome to Acme")}),
			fnode("n3", automations.NodeDelay, 0, 0, flowData{Delay: fdur(1, "days")}),
			fnode("n4", automations.NodeEmail, 0, 0, flowData{Email: email("o2", "Create your first flag")}),
			fnode("n5", automations.NodeWebhook, 0, 0, flowData{Webhook: &automations.WebhookConfig{Name: "Sync to HubSpot", URL: "https://api.hubapi.com/automation/v4/webhook-triggers/2463510/acme-signup", Method: "POST"}}),
			fnode("n6", automations.NodeBranch, 0, 0, flowData{Condition: &automations.FlowCondition{Kind: "opened", Step: "n4", Within: fdur(3, "days"), HumanOnly: true}}),
			fnode("n7", automations.NodeEmail, 0, 0, flowData{Email: email("o3", "Nice work. Next: environments")}),
			fnode("n8", automations.NodeEmail, 0, 0, flowData{Email: email("o4", "Need a hand with your first flag?")}),
			fnode("n9", automations.NodeDelay, 0, 0, flowData{Delay: fdur(2, "days")}),
			fnode("n10", automations.NodeEmail, 0, 0, flowData{Email: email("o5", "Invite your team")}),
			fnode("n11", automations.NodeWebhook, 0, 0, flowData{Webhook: &automations.WebhookConfig{Name: "Update CRM stage", URL: "https://api.hubapi.com/crm/v3/objects/deals/acme-stage", Method: "POST"}}),
			fnode("n12", automations.NodeDelay, 0, 0, flowData{Delay: fdur(3, "days")}),
			fnode("n13", automations.NodeEmail, 0, 0, flowData{Email: email("o6", "Your first week with Acme")}),
			fnode("n14", automations.NodeGoal, 0, 0, flowData{Label: "Invited a teammate", Condition: &automations.FlowCondition{Kind: "event", Event: "teammate.invited"}}),
		},
		Edges: []automations.FlowEdge{
			fedge("n1", "n2", ""), fedge("n2", "n3", ""), fedge("n3", "n4", ""), fedge("n4", "n5", ""), fedge("n5", "n6", ""),
			fedge("n6", "n7", automations.PathYes), fedge("n6", "n8", automations.PathNo),
			fedge("n7", "n9", ""), fedge("n8", "n9", ""), fedge("n9", "n10", ""), fedge("n10", "n11", ""),
			fedge("n11", "n12", ""), fedge("n12", "n13", ""), fedge("n13", "n14", ""),
		},
	}
	onboarding = automations.Layout(onboarding)
	onboardingID := ID()
	var obVersions []versionSeed
	for v := 1; v <= 7; v++ {
		created := s.designDay(14+(7-v)*6, 10, 0)
		vs := versionSeed{graph: onboarding, by: marcus.ID, created: created, published: ptrTime(created.Add(40 * time.Minute)), summary: onboardingNotes[v-1], migration: "finish"}
		if v < 7 {
			vs.retired = ptrTime(s.designDay(14+(6-v)*6, 12, 0))
		}
		obVersions = append(obVersions, vs)
	}
	if err := insertFlow(ctx, s, flowSeed{
		id: onboardingID, name: "Onboarding · new workspace", description: "6 emails, 2 webhooks", status: "live", template: "onboarding",
		live: 7, created: s.designDay(50, 10, 0), createdBy: marcus.ID, updated: s.designDay(14, 10, 40), versions: obVersions,
	}); err != nil {
		return err
	}

	// Re-engagement: v2 live since 1 Sep.
	reengagement := automations.Layout(automations.FlowGraph{
		Nodes: []flowNode{
			fnode("n1", automations.NodeTrigger, 0, 0, flowData{Trigger: &automations.TriggerConfig{Kind: automations.TriggerTagAdded, Tag: "sunset-entered", Reentry: "once"}}),
			fnode("n2", automations.NodeEmail, 0, 0, flowData{Email: email("r1", "Still want these?")}),
			fnode("n3", automations.NodeBranch, 0, 0, flowData{Condition: &automations.FlowCondition{Kind: "opened", Step: "n2", Within: fdur(14, "days"), HumanOnly: true}}),
			fnode("n4", automations.NodeGoal, 0, 0, flowData{Label: "Opened within 14 d", Condition: &automations.FlowCondition{Kind: "opened", HumanOnly: true}}),
			fnode("n5", automations.NodeList, 0, 0, flowData{Label: "Stop broadcasts", Action: &automations.ListActionConfig{Kind: "unsubscribe_all"}}),
		},
		Edges: []automations.FlowEdge{
			fedge("n1", "n2", ""), fedge("n2", "n3", ""),
			fedge("n3", "n4", automations.PathYes), fedge("n3", "n5", automations.PathNo),
		},
	})
	reengagementID := ID()
	if err := insertFlow(ctx, s, flowSeed{
		id: reengagementID, name: "Re-engagement", description: "1 email, then sunset", status: "live", template: "re-engagement",
		live: 2, created: s.designDay(80, 9, 0), createdBy: linh.ID, updated: s.designDay(41, 9, 30),
		versions: []versionSeed{
			{graph: reengagement, by: linh.ID, created: s.designDay(80, 9, 0), published: ptrTime(s.designDay(80, 9, 20)), retired: ptrTime(s.designDay(26, 8, 0)), summary: "first published version"},
			{graph: reengagement, by: linh.ID, created: s.designDay(41, 9, 0), published: ptrTime(s.designDay(41, 9, 30)), summary: `window before "Stop broadcasts" 7 days → 14 days`, migration: "finish"},
		},
	}); err != nil {
		return err
	}

	// Re-permission · Mailchimp import: a draft, never published.
	repermission := automations.Layout(automations.FlowGraph{
		Nodes: []flowNode{
			fnode("n1", automations.NodeTrigger, 0, 0, flowData{Trigger: &automations.TriggerConfig{Kind: automations.TriggerTagAdded, Tag: "mailchimp-pending", Reentry: "once"}}),
			fnode("n2", automations.NodeEmail, 0, 0, flowData{Email: email("p1", "Still want to hear from Acme?")}),
			fnode("n3", automations.NodeBranch, 0, 0, flowData{Condition: &automations.FlowCondition{Kind: "clicked", Step: "n2", URLContains: "confirm", Within: fdur(14, "days")}}),
			fnode("n4", automations.NodeGoal, 0, 0, flowData{Label: "Confirmed opt-in", Condition: &automations.FlowCondition{Kind: "clicked", URLContains: "confirm"}}),
			fnode("n5", automations.NodeList, 0, 0, flowData{Label: "Suppress non-responders", Action: &automations.ListActionConfig{Kind: "unsubscribe_all"}}),
		},
		Edges: []automations.FlowEdge{
			fedge("n1", "n2", ""), fedge("n2", "n3", ""),
			fedge("n3", "n4", automations.PathYes), fedge("n3", "n5", automations.PathNo),
		},
	})
	if err := insertFlow(ctx, s, flowSeed{
		id: ID(), name: "Re-permission · Mailchimp import", description: "1 email, suppress non-responders after 14 d", status: "draft",
		created: s.designDay(5, 14, 0), createdBy: linh.ID, updated: s.designDay(2, 16, 10),
		versions: []versionSeed{{graph: repermission, by: linh.ID, created: s.designDay(5, 14, 0), summary: "created from a blank canvas"}},
	}); err != nil {
		return err
	}

	// Beta invite: paused 5 Oct by Marcus Chen; the form is linked once the
	// forms seeder has run.
	beta := automations.Layout(automations.FlowGraph{
		Nodes: []flowNode{
			fnode("n1", automations.NodeTrigger, 0, 0, flowData{Trigger: &automations.TriggerConfig{Kind: automations.TriggerFormSubmitted, Reentry: "once"}}),
			fnode("n2", automations.NodeEmail, 0, 0, flowData{Email: email("b1", "You're in the beta")}),
			fnode("n3", automations.NodeBranch, 0, 0, flowData{Condition: &automations.FlowCondition{Kind: "clicked", Step: "n2", URLContains: "accept", Within: fdur(3, "days")}}),
			fnode("n4", automations.NodeGoal, 0, 0, flowData{Label: "Accepted invite", Condition: &automations.FlowCondition{Kind: "clicked", URLContains: "accept"}}),
			fnode("n5", automations.NodeEmail, 0, 0, flowData{Email: email("b2", "Your beta invite expires tomorrow")}),
			fnode("n6", automations.NodeExit, 0, 0, flowData{}),
		},
		Edges: []automations.FlowEdge{
			fedge("n1", "n2", ""), fedge("n2", "n3", ""),
			fedge("n3", "n4", automations.PathYes), fedge("n3", "n5", automations.PathNo), fedge("n5", "n6", ""),
		},
	})
	betaID := ID()
	if err := insertFlow(ctx, s, flowSeed{
		id: betaID, name: "Beta invite", description: "2 emails, goal on accept", status: "paused",
		live: 1, created: s.designDay(34, 13, 0), createdBy: marcus.ID, updated: s.designDay(7, 17, 12),
		paused: ptrTime(s.designDay(7, 17, 12)), pausedBy: &marcus.ID, pauseReason: "beta is full",
		versions: []versionSeed{{graph: beta, by: marcus.ID, created: s.designDay(34, 13, 0), published: ptrTime(s.designDay(34, 13, 30)), summary: "first published version"}},
	}); err != nil {
		return err
	}
	s.After(func(ctx context.Context, s *Seeder) error { return linkBetaForm(ctx, s, betaID, beta) })

	// Thirty days of per-step stats.
	ses, postmark := s.Providers["ses"].String(), s.Providers["postmark"].String()
	stats := []nodeStats{
		// Welcome series (entries all since 29 Sep, when the Slack webhook started failing).
		{a: welcome, v: 3, node: "n1", from: 12, c: counters{"entered": 1284, "completed": 1284}},
		{a: welcome, v: 3, node: "w1", from: 12, c: counters{"entered": 1284, "completed": 1284, "failed": 1284}},
		{a: welcome, v: 3, node: "n2", from: 12, c: counters{"entered": 1284, "completed": 1284, "waiting": 1284}},
		{a: welcome, v: 3, node: "n3", from: 12, c: counters{"entered": 1284, "completed": 1284, "sent": 1276, "skipped": 8, "delivered": 1267, "bounced": 6, "opened": 958, "opened_human": 775, "clicked": 177},
			providers: map[string]int64{ses: 1266}, extraProviders: []dayProvider{{provider: postmark, n: 10, day: 10}}},
		{a: welcome, v: 3, node: "n4", from: 12, c: counters{"entered": 1284, "completed": 1276, "waiting": 1284, "branch_yes": 778, "branch_no": 498}, paths: map[string]int64{"yes": 778, "no": 498}},
		{a: welcome, v: 3, node: "n5", from: 11, c: counters{"entered": 778, "completed": 778, "waiting": 778}},
		{a: welcome, v: 3, node: "n6", from: 10, c: counters{"entered": 778, "completed": 778, "sent": 778, "delivered": 773, "bounced": 3, "opened": 560, "opened_human": 441, "clicked": 147},
			providers: map[string]int64{ses: 778}},
		{a: welcome, v: 3, node: "n10", from: 6, c: counters{"entered": 360, "goal_reached": 312, "exited": 48}},
		{a: welcome, v: 3, node: "n11", from: 10, c: counters{"entered": 498, "completed": 498, "waiting": 498}},
		{a: welcome, v: 3, node: "n12", from: 9, c: counters{"entered": 498, "completed": 498, "sent": 498, "delivered": 495, "bounced": 2, "opened": 240, "opened_human": 188, "clicked": 30},
			providers: map[string]int64{ses: 498}},
		{a: welcome, v: 3, node: "n13", from: 9, c: counters{"entered": 436, "exited": 436}},
	}
	stats = append(stats, onboardingStats(onboardingID, ses)...)
	stats = append(stats, reengagementStats(reengagementID, ses)...)
	stats = append(stats, betaStats(betaID, ses)...)
	if err := insertStats(ctx, s, stats); err != nil {
		return err
	}

	// The Slack webhook removed in v4 has been failing since 29 Sep.
	failSince := s.designDay(13, 8, 12)
	detail, _ := json.Marshal(map[string]any{"url": webhook.Data.Webhook.URL, "status": 404, "error": "", "name": "notify-slack"})
	if _, err := s.Tx.Exec(ctx, `INSERT INTO automation_alerts (id, automation_id, kind, node_id, detail, failures, first_at, last_at) VALUES ($1,$2,'step_failing','w1',$3,1284,$4,$5)`,
		ID(), welcome, detail, failSince, s.Ago(52*time.Minute)); err != nil {
		return err
	}
	for i := 0; i < 12; i++ {
		at := s.Ago(time.Duration(52+i*97) * time.Minute)
		if _, err := s.Tx.Exec(ctx, `INSERT INTO automation_webhook_calls (id, automation_id, version, node_id, url, status_code, duration_ms, attempt, called_at) VALUES ($1,$2,3,'w1',$3,404,$4,1,$5)`,
			ID(), welcome, webhook.Data.Webhook.URL, 120+i*7%40, at); err != nil {
			return err
		}
	}

	// Contacts in each flow right now.
	contacts, err := demoContacts(ctx, s, 900)
	if err != nil {
		return err
	}
	return seedRuns(ctx, s, contacts, []flowRuns{
		{a: welcome, v: 3, at: []runsAt{
			{node: "n2", n: 4, wait: "delay", span: 10 * time.Minute, delay: 10 * time.Minute},
			{node: "n4", n: 168, wait: "branch", span: 2 * 24 * time.Hour, delay: 2 * 24 * time.Hour},
			{node: "n5", n: 203, wait: "delay", span: 24 * time.Hour, delay: 24 * time.Hour},
			{node: "n11", n: 237, wait: "delay", span: 3 * 24 * time.Hour, delay: 3 * 24 * time.Hour},
		}},
		{a: onboardingID, v: 7, at: []runsAt{
			{node: "n3", n: 96, wait: "delay", span: 24 * time.Hour, delay: 24 * time.Hour},
			{node: "n6", n: 143, wait: "branch", span: 3 * 24 * time.Hour, delay: 3 * 24 * time.Hour},
			{node: "n9", n: 121, wait: "delay", span: 2 * 24 * time.Hour, delay: 2 * 24 * time.Hour},
			{node: "n12", n: 128, wait: "delay", span: 3 * 24 * time.Hour, delay: 3 * 24 * time.Hour},
		}},
		{a: reengagementID, v: 2, at: []runsAt{
			{node: "n3", n: 832, wait: "branch", span: 14 * 24 * time.Hour, delay: 14 * 24 * time.Hour},
		}},
	})
}

var onboardingNotes = []string{
	"first published version",
	`added webhook "Sync to HubSpot"`,
	`delay before "Step 2": 2 days → 1 day`,
	`added branch "Opened step 2 within 3 days?"`,
	`new content for "Onboarding · step 1": v2 → v3`,
	`added webhook "Update CRM stage"`,
	`goal: "Created a project" → "Invited a teammate"`,
}

func ptrTime(t time.Time) *time.Time { return &t }

func withoutNodes(g automations.FlowGraph, ids ...string) automations.FlowGraph {
	drop := map[string]bool{}
	for _, id := range ids {
		drop[id] = true
	}
	out := automations.FlowGraph{Nodes: []flowNode{}, Edges: []automations.FlowEdge{}}
	for _, n := range g.Nodes {
		if !drop[n.ID] {
			out.Nodes = append(out.Nodes, n)
		}
	}
	for _, e := range g.Edges {
		if !drop[e.Source] && !drop[e.Target] {
			out.Edges = append(out.Edges, e)
		}
	}
	return out
}

type versionSeed struct {
	graph              automations.FlowGraph
	by                 uuid.UUID
	created            time.Time
	published, retired *time.Time
	summary, migration string
}

type flowSeed struct {
	id                                  uuid.UUID
	name, description, status, template string
	live                                int
	created, updated                    time.Time
	createdBy                           uuid.UUID
	paused                              *time.Time
	pausedBy                            *uuid.UUID
	pauseReason                         string
	versions                            []versionSeed
}

func insertFlow(ctx context.Context, s *Seeder, f flowSeed) error {
	draft := len(f.versions)
	var live any
	shown := f.versions[draft-1].graph
	if f.live > 0 {
		live = f.live
		shown = f.versions[f.live-1].graph
	}
	trigger := json.RawMessage(`{}`)
	if t := shown.Trigger(); t != nil && t.Data.Trigger != nil {
		trigger, _ = json.Marshal(t.Data.Trigger)
	}
	if _, err := s.Tx.Exec(ctx, `INSERT INTO automations (id, workspace_id, name, description, status, template_key, live_version, draft_version, trigger,
		created_by, updated_by, created_at, updated_at, paused_at, paused_by, pause_reason)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10,$11,$12,$13,$14,$15)`,
		f.id, s.WS().ID, f.name, f.description, f.status, f.template, live, draft, trigger, f.createdBy, f.created, f.updated, f.paused, f.pausedBy, f.pauseReason); err != nil {
		return err
	}
	for i, v := range f.versions {
		var publishedBy any
		if v.published != nil {
			publishedBy = v.by
		}
		if _, err := s.Tx.Exec(ctx, `INSERT INTO automation_versions (automation_id, version, graph, created_by, created_at, published_by, published_at, retired_at, summary, migration)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			f.id, i+1, v.graph.MustJSON(), v.by, v.created, publishedBy, v.published, v.retired, v.summary, v.migration); err != nil {
			return err
		}
	}
	return nil
}

// linkBetaForm points the Beta invite trigger at the beta-apply form,
// creating the form when the forms seeder didn't.
func linkBetaForm(ctx context.Context, s *Seeder, flow uuid.UUID, g automations.FlowGraph) error {
	ws := s.WS()
	var form uuid.UUID
	err := s.Tx.QueryRow(ctx, `SELECT id FROM forms WHERE workspace_id = $1 AND slug = 'beta-apply'`, ws.ID).Scan(&form)
	if err == pgx.ErrNoRows {
		form = ID()
		if _, err := s.Tx.Exec(ctx, `INSERT INTO forms (id, workspace_id, name, kind, slug, list_ids, double_opt_in, status, created_by, created_at)
			VALUES ($1,$2,'Beta application','hosted','beta-apply',$3,true,'live',$4,$5)`,
			form, ws.ID, []uuid.UUID{s.Lists["Beta Program"]}, s.Users["marcus@acme.io"].ID, s.Ago(40*24*time.Hour)); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	for i := range g.Nodes {
		if g.Nodes[i].Type == automations.NodeTrigger {
			g.Nodes[i].Data.Trigger.FormID = form.String()
		}
	}
	trigger, _ := json.Marshal(g.Trigger().Data.Trigger)
	if _, err := s.Tx.Exec(ctx, `UPDATE automation_versions SET graph = $2 WHERE automation_id = $1`, flow, g.MustJSON()); err != nil {
		return err
	}
	_, err = s.Tx.Exec(ctx, `UPDATE automations SET trigger = $2 WHERE id = $1`, flow, trigger)
	return err
}

type counters map[string]int64

type dayProvider struct {
	provider string
	n        int64
	day      int
}

// nodeStats spreads one step's counters over days [from … to] before today
// (to defaults to 0, today). goal_reached is kept to the last 7 days when
// goalFrom is set.
type nodeStats struct {
	a              uuid.UUID
	v              int32
	node           string
	from, to       int
	c              counters
	paths          map[string]int64
	providers      map[string]int64
	extraProviders []dayProvider
	split          map[string][2]int // counter → own day range
}

var statColumns = []string{"entered", "completed", "waiting", "exited", "dropped", "goal_reached", "skipped", "failed", "sent",
	"delivered", "bounced", "complained", "opened", "opened_human", "clicked", "unsubscribed", "branch_yes", "branch_no"}

// dayWeights gives each day of a range a slightly different volume so the
// numbers look like real traffic; the parts always add up (Split).
func dayWeights(n int) []float64 {
	w := make([]float64, n)
	for i := range w {
		w[i] = 1 + 0.35*math.Sin(float64(i)*1.7) + 0.15*math.Cos(float64(i)*0.9)
	}
	return w
}

func insertStats(ctx context.Context, s *Seeder, all []nodeStats) error {
	type row struct {
		vals      map[string]int64
		paths     map[string]int64
		providers map[string]int64
	}
	today := time.Date(s.Now.UTC().Year(), s.Now.UTC().Month(), s.Now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	type key struct {
		a    uuid.UUID
		v    int32
		node string
		day  int
	}
	acc := map[key]*row{}
	var order []key
	for _, ns := range all {
		get := func(d int) *row {
			k := key{ns.a, ns.v, ns.node, d}
			r, ok := acc[k]
			if !ok {
				r = &row{vals: map[string]int64{}, paths: map[string]int64{}, providers: map[string]int64{}}
				acc[k] = r
				order = append(order, k)
			}
			return r
		}
		spread := func(total int64, from, to int, put func(r *row, n int64)) {
			days := from - to + 1
			if days <= 0 {
				days = 1
			}
			parts := Split(total, dayWeights(days))
			for i, n := range parts {
				put(get(to+i), n)
			}
		}
		for k, total := range ns.c {
			from, to := ns.from, ns.to
			if r, ok := ns.split[k]; ok {
				from, to = r[0], r[1]
			}
			key := k
			spread(total, from, to, func(r *row, n int64) { r.vals[key] += n })
		}
		for k, total := range ns.paths {
			key := k
			spread(total, ns.from, ns.to, func(r *row, n int64) { r.paths[key] += n })
		}
		for k, total := range ns.providers {
			key := k
			spread(total, ns.from, ns.to, func(r *row, n int64) { r.providers[key] += n })
		}
		for _, x := range ns.extraProviders {
			get(x.day).providers[x.provider] += x.n
		}
	}
	rows := make([][]any, 0, len(order))
	for _, k := range order {
		r := acc[k]
		paths, _ := json.Marshal(r.paths)
		provs, _ := json.Marshal(r.providers)
		vals := []any{k.a, k.v, k.node, today.AddDate(0, 0, -k.day)}
		for _, c := range statColumns {
			vals = append(vals, r.vals[c])
		}
		vals = append(vals, json.RawMessage(paths), json.RawMessage(provs))
		rows = append(rows, vals)
	}
	cols := append(append([]string{"automation_id", "version", "node_id", "day"}, statColumns...), "paths", "providers")
	_, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"automation_step_stats"}, cols, pgx.CopyFromRows(rows))
	return err
}

func emailCounters(entered, sent int64, openRate, clickRate float64) counters {
	delivered := int64(math.Round(float64(sent) * 0.993))
	human := int64(math.Round(float64(delivered) * openRate))
	return counters{"entered": entered, "completed": entered, "sent": sent, "skipped": entered - sent, "delivered": delivered,
		"bounced": sent - delivered, "opened": human + human/4, "opened_human": human, "clicked": int64(math.Round(float64(delivered) * clickRate))}
}

func onboardingStats(a uuid.UUID, ses string) []nodeStats {
	const v = 7
	mk := func(node string, from int, c counters, sent int64) nodeStats {
		ns := nodeStats{a: a, v: v, node: node, from: from, c: c}
		if sent > 0 {
			ns.providers = map[string]int64{ses: sent}
		}
		return ns
	}
	return []nodeStats{
		mk("n1", 29, counters{"entered": 906, "completed": 906}, 0),
		mk("n2", 29, emailCounters(906, 902, 0.62, 0.21), 902),
		mk("n3", 28, counters{"entered": 902, "completed": 806, "waiting": 902}, 0),
		mk("n4", 28, emailCounters(806, 801, 0.58, 0.19), 801),
		mk("n5", 28, counters{"entered": 806, "completed": 806, "failed": 4}, 0),
		{a: a, v: v, node: "n6", from: 27, c: counters{"entered": 806, "completed": 663, "waiting": 806, "branch_yes": 452, "branch_no": 211}, paths: map[string]int64{"yes": 452, "no": 211}},
		mk("n7", 26, emailCounters(452, 449, 0.57, 0.18), 449),
		mk("n8", 25, emailCounters(211, 210, 0.41, 0.09), 210),
		mk("n9", 25, counters{"entered": 663, "completed": 542, "waiting": 663}, 0),
		mk("n10", 24, emailCounters(542, 539, 0.52, 0.31), 539),
		mk("n11", 24, counters{"entered": 542, "completed": 542, "failed": 2}, 0),
		mk("n12", 24, counters{"entered": 542, "completed": 461, "waiting": 542}, 0),
		mk("n13", 22, emailCounters(461, 458, 0.49, 0.12), 458),
		{a: a, v: v, node: "n14", from: 29, c: counters{"entered": 643, "goal_reached": 123, "exited": 299},
			split: map[string][2]int{"goal_reached": {29, 7}}},
		// The last week's goals, on top of the earlier ones (KPI "onboarding 221").
		{a: a, v: v, node: "n14", from: 6, c: counters{"goal_reached": 221}},
	}
}

func reengagementStats(a uuid.UUID, ses string) []nodeStats {
	const v = 2
	return []nodeStats{
		{a: a, v: v, node: "n1", from: 29, c: counters{"entered": 4811, "completed": 4811}},
		{a: a, v: v, node: "n2", from: 29, c: emailCounters(4811, 4797, 0.142, 0.031), providers: map[string]int64{ses: 4797}},
		{a: a, v: v, node: "n3", from: 29, c: counters{"entered": 4811, "completed": 3993, "waiting": 4811, "branch_yes": 885, "branch_no": 3108}, paths: map[string]int64{"yes": 885, "no": 3108}},
		{a: a, v: v, node: "n4", from: 29, c: counters{"entered": 885, "goal_reached": 814}, split: map[string][2]int{"goal_reached": {29, 7}}},
		{a: a, v: v, node: "n4", from: 6, c: counters{"goal_reached": 71}},
		{a: a, v: v, node: "n5", from: 29, c: counters{"entered": 3108, "completed": 3108, "exited": 3108}},
	}
}

func betaStats(a uuid.UUID, ses string) []nodeStats {
	const v = 1
	return []nodeStats{
		{a: a, v: v, node: "n1", from: 29, to: 8, c: counters{"entered": 214, "completed": 214}},
		{a: a, v: v, node: "n2", from: 29, to: 8, c: counters{"entered": 214, "completed": 214, "sent": 213, "skipped": 1, "delivered": 213, "opened": 160, "opened_human": 134, "clicked": 117}, providers: map[string]int64{ses: 213}},
		{a: a, v: v, node: "n3", from: 28, to: 8, c: counters{"entered": 214, "completed": 188, "waiting": 214, "branch_yes": 111, "branch_no": 77}, paths: map[string]int64{"yes": 111, "no": 77}},
		{a: a, v: v, node: "n4", from: 28, to: 8, c: counters{"entered": 111, "goal_reached": 111}},
		{a: a, v: v, node: "n5", from: 26, to: 7, c: counters{"entered": 77, "completed": 77, "sent": 77, "delivered": 77, "opened": 58, "opened_human": 49, "clicked": 17}, providers: map[string]int64{ses: 77}},
		{a: a, v: v, node: "n6", from: 26, to: 7, c: counters{"entered": 77, "exited": 77}},
	}
}

var demoFirst = []string{"Maya", "Leo", "Priya", "Jonas", "Amara", "Ethan", "Sofia", "Kenji", "Lucia", "Omar", "Hana", "Mateo", "Zoe", "Ravi", "Ines", "Felix", "Nora", "Tariq", "Elena", "Sam"}
var demoLast = []string{"Chen", "Garcia", "Patel", "Müller", "Okafor", "Brooks", "Rossi", "Tanaka", "Silva", "Haddad", "Kim", "Lopez", "Novak", "Iyer", "Costa", "Weber", "Berg", "Aziz", "Petrova", "Reed"}

// demoContacts returns n contacts of the workspace for runs, creating
// placeholder contacts only when the audience seeder hasn't made enough.
func demoContacts(ctx context.Context, s *Seeder, n int) ([]uuid.UUID, error) {
	ws := s.WS()
	rows, err := s.Tx.Query(ctx, `SELECT id FROM contacts WHERE workspace_id = $1 AND status = 'active' AND erased_at IS NULL ORDER BY created_at, id LIMIT $2`, ws.ID, n)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, err
	}
	if len(ids) >= n {
		return ids, nil
	}
	providers := []string{"gmail", "gmail", "outlook", "yahoo", "icloud", "other"}
	var copyRows [][]any
	for i := len(ids); i < n; i++ {
		id := ID()
		first, last := demoFirst[i%len(demoFirst)], demoLast[(i/len(demoFirst))%len(demoLast)]
		email := fmt.Sprintf("%s.%s%d@example.com", toLowerASCII(first), toLowerASCII(last), i)
		created := s.Ago(time.Duration(30+i%300) * 24 * time.Hour)
		copyRows = append(copyRows, []any{id, ws.ID, email, first, last, "active", "demo", "America/New_York", providers[i%len(providers)], created, created})
		ids = append(ids, id)
	}
	_, err = s.Tx.CopyFrom(ctx, pgx.Identifier{"contacts"}, []string{"id", "workspace_id", "email", "first_name", "last_name", "status", "source", "timezone", "mailbox_provider", "created_at", "updated_at"}, pgx.CopyFromRows(copyRows))
	return ids, err
}

func toLowerASCII(s string) string {
	out := []rune{}
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			out = append(out, r+'a'-'A')
		case r == 'ü':
			out = append(out, 'u')
		default:
			out = append(out, r)
		}
	}
	return string(out)
}

type runsAt struct {
	node  string
	n     int
	wait  string
	span  time.Duration
	delay time.Duration
}

type flowRuns struct {
	a  uuid.UUID
	v  int32
	at []runsAt
}

// seedRuns places contacts in flows: each waits at a step, with its timer
// somewhere in the step's window so they move on over the coming days.
func seedRuns(ctx context.Context, s *Seeder, contacts []uuid.UUID, flows []flowRuns) error {
	var rows [][]any
	for _, f := range flows {
		i := 0
		for _, at := range f.at {
			for k := 0; k < at.n; k++ {
				c := contacts[i%len(contacts)]
				i++
				// Arrived somewhere in the last span; the timer ends delay later.
				frac := (float64(k) + 0.5) / float64(at.n)
				arrived := s.Now.Add(-time.Duration(frac * float64(at.span) * 0.96))
				until := arrived.Add(at.delay)
				entered := arrived.Add(-time.Duration(1+k%5) * time.Hour)
				state := map[string]any{"wait": at.wait, "until": until.UTC().Format(time.RFC3339Nano)}
				if at.wait == "branch" {
					state["want"] = "opened"
					state["human"] = true
				}
				st, _ := json.Marshal(state)
				rows = append(rows, []any{ID(), f.a, f.v, c, at.node, "waiting", json.RawMessage(st), entered, arrived, until})
			}
		}
	}
	_, err := s.Tx.CopyFrom(ctx, pgx.Identifier{"automation_runs"}, []string{"id", "automation_id", "version", "contact_id", "node_id", "status", "state", "entered_at", "node_entered_at", "next_at"}, pgx.CopyFromRows(rows))
	return err
}
