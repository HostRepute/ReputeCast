package automations

import (
	"html"
	"strings"
)

// AutomationStarterTemplate is one of the starter flows offered on the automations
// list and in the empty state. Each opens as a draft; every step is editable.
type AutomationStarterTemplate struct {
	Key  string `json:"key" enum:"welcome-series,re-engagement,onboarding,abandoned-cart"`
	Name string `json:"name"`
	// Summary is the card text in the list's "Starter templates" section.
	Summary string `json:"summary"`
	// Outline is the card text in the empty state ("Joined list → wait 10 min → …").
	Outline string `json:"outline"`
	// Steps counts the steps the template creates.
	Steps int `json:"steps"`
	// Preview lists the step types along the main path, for the dot strip.
	Preview []string `json:"preview"`
}

// templateEmail is an email a template creates along with the flow.
type templateEmail struct {
	key, name, subject, preheader, heading, cta string
	body                                        []string
}

type templateDef struct {
	AutomationStarterTemplate
	description string
	emails      []templateEmail
	// build makes the graph on a list, or on a store when store is set
	// (templates_store.go): its id is what build gets.
	build func(ref string) AutomationGraph
	store bool
}

func fork(id, typ string, d AutomationNodeData) AutomationNode {
	return AutomationNode{ID: id, Type: typ, Data: d}
}

func link(src, dst, path string) AutomationEdge {
	return AutomationEdge{ID: "e-" + src + "-" + dst, Source: src, Target: dst, Data: AutomationEdgeData{Path: path}}
}

func emailNode(id, emailKey, subject string) AutomationNode {
	return fork(id, NodeEmail, AutomationNodeData{Email: &AutomationEmailStep{
		EmailID: "tpl:" + emailKey, EmailVersion: 1, Subject: subject,
		SendWindow: &AutomationSendWindow{Enabled: true, Start: "07:00", End: "21:00"}, SkipRecentHours: 4,
	}})
}

var templates = []templateDef{
	{
		AutomationStarterTemplate: AutomationStarterTemplate{
			Key: "welcome-series", Name: "Welcome series",
			Summary: "Trigger on list join, 3 emails over a week, branch on first open, goal on first click.",
			Outline: "Joined list → wait 10 min → email 1 → branch on open → emails 2 and 3 → goal: first click",
			Steps:   7, Preview: []string{NodeTrigger, NodeEmail, NodeEmail, NodeGoal},
		},
		description: "3 emails over a week",
		emails: []templateEmail{
			{key: "w1", name: `Welcome · step 1 "You're in"`, subject: "You're in. Here's what to expect", preheader: "One email a week, and how to make the most of it.",
				heading: "You're in", cta: "Read the getting-started guide", body: []string{"Thanks for subscribing. Every week we send one email with what's new, a guide worth your time and the changelog.", "Start with the guide most people find useful first."}},
			{key: "w2", name: `Step 2 "Three things to try first"`, subject: "Three things to try first", preheader: "Small wins for your first week.",
				heading: "Three things to try first", cta: "Try the first one", body: []string{"Here are three things people get the most out of in their first week."}},
			{key: "w3", name: `Step 3 "Did you get our welcome?"`, subject: "Did you get our welcome?", preheader: "A plain-text note in case the first one got buried.",
				heading: "Did you get our welcome?", cta: "Open the guide", body: []string{"Our first email may have landed in another tab. Here's the short version."}},
		},
		build: func(listID string) AutomationGraph {
			return AutomationGraph{
				Nodes: []AutomationNode{
					fork("n1", NodeTrigger, AutomationNodeData{Trigger: &AutomationTrigger{Kind: TriggerJoinedList, ListID: listID, ConfirmedOnly: true, Reentry: "once"}}),
					fork("n2", NodeDelay, AutomationNodeData{Delay: &AutomationDuration{Amount: 10, Unit: "minutes"}}),
					emailNode("n3", "w1", "You're in. Here's what to expect"),
					fork("n4", NodeBranch, AutomationNodeData{Condition: &AutomationCondition{Kind: "opened", Step: "n3", Within: &AutomationDuration{Amount: 2, Unit: "days"}, HumanOnly: true}}),
					emailNode("n5", "w2", "Three things to try first"),
					emailNode("n6", "w3", "Did you get our welcome?"),
					fork("n7", NodeGoal, AutomationNodeData{Label: "First click", Condition: &AutomationCondition{Kind: "clicked"}}),
				},
				Edges: []AutomationEdge{
					link("n1", "n2", ""), link("n2", "n3", ""), link("n3", "n4", ""),
					link("n4", "n5", PathYes), link("n4", "n6", PathNo),
					link("n5", "n7", ""), link("n6", "n7", ""),
				},
			}
		},
	},
	{
		AutomationStarterTemplate: AutomationStarterTemplate{
			Key: "re-engagement", Name: "Re-engagement",
			Summary: `Trigger when a contact enters the Sunset tier, one "still want these?" email, exit or stop broadcasts.`,
			Outline: `Tag "sunset-entered" → "still want these?" email → wait 14 d → branch on open → exit or stop broadcasts`,
			Steps:   5, Preview: []string{NodeTrigger, NodeEmail, NodeBranch, NodeExit},
		},
		description: "1 email, then sunset",
		emails: []templateEmail{
			{key: "r1", name: `Re-engagement "Still want these?"`, subject: "Still want these?", preheader: "One click keeps you on the list.",
				heading: "Still want these?", cta: "Yes, keep me subscribed", body: []string{"We haven't seen you open our emails in a while. If you'd like to keep getting them, tell us with one click.", "If not, we'll stop sending; you can come back any time."}},
		},
		build: func(string) AutomationGraph {
			return AutomationGraph{
				Nodes: []AutomationNode{
					fork("n1", NodeTrigger, AutomationNodeData{Trigger: &AutomationTrigger{Kind: TriggerTagAdded, Tag: "sunset-entered", Reentry: "once"}}),
					emailNode("n2", "r1", "Still want these?"),
					fork("n3", NodeBranch, AutomationNodeData{Condition: &AutomationCondition{Kind: "opened", Step: "n2", Within: &AutomationDuration{Amount: 14, Unit: "days"}, HumanOnly: true}}),
					fork("n4", NodeExit, AutomationNodeData{Label: "Stays subscribed"}),
					fork("n5", NodeList, AutomationNodeData{Label: "Stop broadcasts", Action: &AutomationListAction{Kind: "unsubscribe_all"}}),
				},
				Edges: []AutomationEdge{
					link("n1", "n2", ""), link("n2", "n3", ""),
					link("n3", "n4", PathYes), link("n3", "n5", PathNo),
				},
			}
		},
	},
	{
		AutomationStarterTemplate: AutomationStarterTemplate{
			Key: "onboarding", Name: "Onboarding",
			Summary: "Trigger on an API event, 5 emails tied to product milestones, webhook to your CRM, goal on activation.",
			Outline: "API event → 5 milestone emails → webhook to your CRM → goal: activated",
			Steps:   12, Preview: []string{NodeTrigger, NodeEmail, NodeWebhook, NodeEmail, NodeGoal},
		},
		description: "5 emails, 1 webhook",
		emails: []templateEmail{
			{key: "o1", name: `Onboarding · step 1 "Welcome aboard"`, subject: "Welcome aboard", preheader: "Your account is ready.", heading: "Welcome aboard", cta: "Open your account", body: []string{"Your account is ready. Here's the one thing to do first."}},
			{key: "o2", name: `Step 2 "Create your first project"`, subject: "Create your first project", preheader: "It takes about two minutes.", heading: "Create your first project", cta: "Create a project", body: []string{"Projects hold everything else. Create one to see how it fits together."}},
			{key: "o3", name: `Step 3 "Invite your team"`, subject: "Invite your team", preheader: "Work is better together.", heading: "Invite your team", cta: "Invite a teammate", body: []string{"Teams that invite a second person in the first week stay three times longer."}},
			{key: "o4", name: `Step 4 "Connect your tools"`, subject: "Connect your tools", preheader: "Bring in the data you already have.", heading: "Connect your tools", cta: "See integrations", body: []string{"Connect the tools you already use so nothing lives in two places."}},
			{key: "o5", name: `Step 5 "Your first month"`, subject: "Your first month", preheader: "What people do next.", heading: "Your first month", cta: "Read the guide", body: []string{"A month in, here's what teams like yours do next."}},
		},
		build: func(string) AutomationGraph {
			return AutomationGraph{
				Nodes: []AutomationNode{
					fork("n1", NodeTrigger, AutomationNodeData{Trigger: &AutomationTrigger{Kind: TriggerAPIEvent, Event: "user.signed_up", Reentry: "once"}}),
					emailNode("n2", "o1", "Welcome aboard"),
					fork("n3", NodeDelay, AutomationNodeData{Delay: &AutomationDuration{Amount: 1, Unit: "days"}}),
					emailNode("n4", "o2", "Create your first project"),
					fork("n5", NodeDelay, AutomationNodeData{Delay: &AutomationDuration{Amount: 2, Unit: "days"}}),
					emailNode("n6", "o3", "Invite your team"),
					fork("n7", NodeWebhook, AutomationNodeData{Webhook: &AutomationWebhook{Name: "Sync to CRM", Method: "POST"}}),
					fork("n8", NodeDelay, AutomationNodeData{Delay: &AutomationDuration{Amount: 3, Unit: "days"}}),
					emailNode("n9", "o4", "Connect your tools"),
					fork("n10", NodeDelay, AutomationNodeData{Delay: &AutomationDuration{Amount: 7, Unit: "days"}}),
					emailNode("n11", "o5", "Your first month"),
					fork("n12", NodeGoal, AutomationNodeData{Label: "Activated", Condition: &AutomationCondition{Kind: "event", Event: "account.activated"}}),
				},
				Edges: []AutomationEdge{
					link("n1", "n2", ""), link("n2", "n3", ""), link("n3", "n4", ""), link("n4", "n5", ""),
					link("n5", "n6", ""), link("n6", "n7", ""), link("n7", "n8", ""), link("n8", "n9", ""),
					link("n9", "n10", ""), link("n10", "n11", ""), link("n11", "n12", ""),
				},
			}
		},
	},
}

// Templates lists the starter templates.
func Templates() []AutomationStarterTemplate {
	out := make([]AutomationStarterTemplate, len(templates))
	for i, t := range templates {
		out[i] = t.AutomationStarterTemplate
	}
	return out
}

// templatesFor lists the starter templates a workspace can use: the ones
// that start on a store's events need a connected store.
func templatesFor(hasStore bool) []AutomationStarterTemplate {
	out := make([]AutomationStarterTemplate, 0, len(templates))
	for _, t := range templates {
		if t.store && !hasStore {
			continue
		}
		out = append(out, t.AutomationStarterTemplate)
	}
	return out
}

func templateByKey(key string) (templateDef, bool) {
	for _, t := range templates {
		if t.Key == key {
			return t, true
		}
	}
	return templateDef{}, false
}

// EmailContent is the content of one email version.
type EmailContent struct {
	Subject, Preheader, MJML, HTML, Text string
}

// starterEmail renders a template email as MJML, matching HTML and plain text.
// Liquid tags ({{ unsubscribe_url }}) are filled in at send time.
func starterEmail(e templateEmail) EmailContent {
	var mj, hb, tx strings.Builder
	mj.WriteString(`<mjml><mj-head><mj-attributes><mj-all font-family="IBM Plex Sans, Arial, sans-serif" /></mj-attributes><mj-preview>` + html.EscapeString(e.preheader) + `</mj-preview></mj-head>`)
	mj.WriteString(`<mj-body background-color="#F3F4F6"><mj-section background-color="#FFFFFF" padding="32px 24px"><mj-column>`)
	mj.WriteString(`<mj-text font-size="22px" font-weight="700" color="#0B1220">` + html.EscapeString(e.heading) + `</mj-text>`)
	hb.WriteString(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>` + html.EscapeString(e.subject) + `</title></head>`)
	hb.WriteString(`<body style="margin:0;background:#F3F4F6;font-family:'IBM Plex Sans',Arial,sans-serif;color:#0B1220">`)
	hb.WriteString(`<div style="display:none;max-height:0;overflow:hidden">` + html.EscapeString(e.preheader) + `</div>`)
	hb.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding:24px 12px">`)
	hb.WriteString(`<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%;background:#FFFFFF"><tr><td style="padding:32px 24px">`)
	hb.WriteString(`<h1 style="margin:0 0 16px;font-size:22px">` + html.EscapeString(e.heading) + `</h1>`)
	tx.WriteString(e.heading + "\n\n")
	for _, p := range e.body {
		mj.WriteString(`<mj-text font-size="15px" line-height="1.6" color="#374151">` + html.EscapeString(p) + `</mj-text>`)
		hb.WriteString(`<p style="margin:0 0 14px;font-size:15px;line-height:1.6;color:#374151">` + html.EscapeString(p) + `</p>`)
		tx.WriteString(p + "\n\n")
	}
	if e.cta != "" {
		mj.WriteString(`<mj-button href="{{ cta_url }}" background-color="#0F766E" border-radius="8px">` + html.EscapeString(e.cta) + `</mj-button>`)
		hb.WriteString(`<p style="margin:20px 0"><a href="{{ cta_url }}" style="display:inline-block;padding:12px 18px;background:#0F766E;color:#FFFFFF;border-radius:8px;text-decoration:none;font-weight:600">` + html.EscapeString(e.cta) + `</a></p>`)
		tx.WriteString(e.cta + ": {{ cta_url }}\n\n")
	}
	footer := `You get this because you subscribed to {{ workspace_name }}. <a href="{{ unsubscribe_url }}">Unsubscribe</a> · {{ footer_address }}`
	mj.WriteString(`<mj-text font-size="12px" color="#5B6472">` + footer + `</mj-text></mj-column></mj-section></mj-body></mjml>`)
	hb.WriteString(`<p style="margin:24px 0 0;font-size:12px;color:#5B6472">` + footer + `</p></td></tr></table></td></tr></table></body></html>`)
	tx.WriteString("Unsubscribe: {{ unsubscribe_url }}\n{{ footer_address }}\n")
	return EmailContent{Subject: e.subject, Preheader: e.preheader, MJML: mj.String(), HTML: hb.String(), Text: tx.String()}
}

// BlankEmail is the content of a new email made from the step panel.
func BlankEmail(name, subject string) EmailContent {
	heading := emailQuoted(name)
	if heading == "" {
		heading = name
	}
	return starterEmail(templateEmail{name: name, subject: subject, heading: heading, body: []string{"Write your email here."}})
}

// DemoEmail renders seeded demo emails with the starter layout.
func DemoEmail(subject, preheader, heading, cta string, body ...string) EmailContent {
	return starterEmail(templateEmail{subject: subject, preheader: preheader, heading: heading, cta: cta, body: body})
}
