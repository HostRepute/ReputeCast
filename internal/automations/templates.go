package automations

import (
	"html"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
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

// starterEmail is a starter's email: its messages are
// template.<starter>.<key>.{name,subject,preheader,heading,cta,body1…}.
type starterEmailDef struct {
	key   string
	paras int
}

// templateDef is a starter template. What it says and what it creates
// (the flow's labels, its emails) are in messages/<locale>.json under
// template.<msg>.*: the cards read in the reader's language, and a flow is
// created in its creator's, for its emails' readers.
type templateDef struct {
	key     string
	msg     string
	steps   int
	preview []string
	emails  []starterEmailDef
	// build makes the graph on a list, or on a store when store is set
	// (templates_store.go): its id is what build gets. say writes the
	// template's own words (template.<msg>.<part>).
	build func(ref string, say func(part string) string) AutomationGraph
	store bool
}

// text is one of the template's own words in a language.
func (t templateDef) text(l i18n.Lang, part string) string {
	return msg("template."+t.msg+"."+part).Text(l, nil)
}

// card is the template as the list offers it, in a language.
func (t templateDef) card(l i18n.Lang) AutomationStarterTemplate {
	return AutomationStarterTemplate{Key: t.key, Name: t.text(l, "name"), Summary: t.text(l, "summary"), Outline: t.text(l, "outline"),
		Steps: t.steps, Preview: t.preview}
}

// graph is the flow the template creates, in its creator's language.
func (t templateDef) graph(ref string, l i18n.Lang) AutomationGraph {
	return t.build(ref, func(part string) string { return t.text(l, part) })
}

// email is one of the template's emails, in its creator's language.
func (t templateDef) email(l i18n.Lang, e starterEmailDef) templateEmail {
	p := e.key + "."
	out := templateEmail{key: e.key, name: t.text(l, p+"name"), subject: t.text(l, p+"subject"), preheader: t.text(l, p+"preheader"),
		heading: t.text(l, p+"heading"), cta: t.text(l, p+"cta")}
	for i := 1; i <= e.paras; i++ {
		out.body = append(out.body, t.text(l, p+"body"+strconv.Itoa(i)))
	}
	return out
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
		key: "welcome-series", msg: "welcome", steps: 7, preview: []string{NodeTrigger, NodeEmail, NodeEmail, NodeGoal},
		emails: []starterEmailDef{{"w1", 2}, {"w2", 1}, {"w3", 1}},
		build: func(listID string, say func(string) string) AutomationGraph {
			return AutomationGraph{
				Nodes: []AutomationNode{
					fork("n1", NodeTrigger, AutomationNodeData{Trigger: &AutomationTrigger{Kind: TriggerJoinedList, ListID: listID, ConfirmedOnly: true, Reentry: "once"}}),
					fork("n2", NodeDelay, AutomationNodeData{Delay: &AutomationDuration{Amount: 10, Unit: "minutes"}}),
					emailNode("n3", "w1", say("w1.subject")),
					fork("n4", NodeBranch, AutomationNodeData{Condition: &AutomationCondition{Kind: "opened", Step: "n3", Within: &AutomationDuration{Amount: 2, Unit: "days"}, HumanOnly: true}}),
					emailNode("n5", "w2", say("w2.subject")),
					emailNode("n6", "w3", say("w3.subject")),
					fork("n7", NodeGoal, AutomationNodeData{Label: say("goal"), Condition: &AutomationCondition{Kind: "clicked"}}),
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
		key: "re-engagement", msg: "reengagement", steps: 5, preview: []string{NodeTrigger, NodeEmail, NodeBranch, NodeExit},
		emails: []starterEmailDef{{"r1", 2}},
		build: func(_ string, say func(string) string) AutomationGraph {
			return AutomationGraph{
				Nodes: []AutomationNode{
					fork("n1", NodeTrigger, AutomationNodeData{Trigger: &AutomationTrigger{Kind: TriggerTagAdded, Tag: "sunset-entered", Reentry: "once"}}),
					emailNode("n2", "r1", say("r1.subject")),
					fork("n3", NodeBranch, AutomationNodeData{Condition: &AutomationCondition{Kind: "opened", Step: "n2", Within: &AutomationDuration{Amount: 14, Unit: "days"}, HumanOnly: true}}),
					fork("n4", NodeExit, AutomationNodeData{Label: say("exit")}),
					fork("n5", NodeList, AutomationNodeData{Label: say("stop"), Action: &AutomationListAction{Kind: "unsubscribe_all"}}),
				},
				Edges: []AutomationEdge{
					link("n1", "n2", ""), link("n2", "n3", ""),
					link("n3", "n4", PathYes), link("n3", "n5", PathNo),
				},
			}
		},
	},
	{
		key: "onboarding", msg: "onboarding", steps: 12, preview: []string{NodeTrigger, NodeEmail, NodeWebhook, NodeEmail, NodeGoal},
		emails: []starterEmailDef{{"o1", 1}, {"o2", 1}, {"o3", 1}, {"o4", 1}, {"o5", 1}},
		build: func(_ string, say func(string) string) AutomationGraph {
			return AutomationGraph{
				Nodes: []AutomationNode{
					fork("n1", NodeTrigger, AutomationNodeData{Trigger: &AutomationTrigger{Kind: TriggerAPIEvent, Event: "user.signed_up", Reentry: "once"}}),
					emailNode("n2", "o1", say("o1.subject")),
					fork("n3", NodeDelay, AutomationNodeData{Delay: &AutomationDuration{Amount: 1, Unit: "days"}}),
					emailNode("n4", "o2", say("o2.subject")),
					fork("n5", NodeDelay, AutomationNodeData{Delay: &AutomationDuration{Amount: 2, Unit: "days"}}),
					emailNode("n6", "o3", say("o3.subject")),
					fork("n7", NodeWebhook, AutomationNodeData{Webhook: &AutomationWebhook{Name: say("webhook"), Method: "POST"}}),
					fork("n8", NodeDelay, AutomationNodeData{Delay: &AutomationDuration{Amount: 3, Unit: "days"}}),
					emailNode("n9", "o4", say("o4.subject")),
					fork("n10", NodeDelay, AutomationNodeData{Delay: &AutomationDuration{Amount: 7, Unit: "days"}}),
					emailNode("n11", "o5", say("o5.subject")),
					fork("n12", NodeGoal, AutomationNodeData{Label: say("goal"), Condition: &AutomationCondition{Kind: "event", Event: "account.activated"}}),
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

// Templates lists the starter templates, in English.
func Templates() []AutomationStarterTemplate {
	out := make([]AutomationStarterTemplate, len(templates))
	for i, t := range templates {
		out[i] = t.card(english)
	}
	return out
}

// templatesFor lists the starter templates a workspace can use, in a
// language: the ones that start on a store's events need a connected store.
func templatesFor(l i18n.Lang, hasStore bool) []AutomationStarterTemplate {
	out := make([]AutomationStarterTemplate, 0, len(templates))
	for _, t := range templates {
		if t.store && !hasStore {
			continue
		}
		out = append(out, t.card(l))
	}
	return out
}

func templateByKey(key string) (templateDef, bool) {
	for _, t := range templates {
		if t.key == key {
			return t, true
		}
	}
	return templateDef{}, false
}

// EmailContent is the content of one email version.
type EmailContent struct {
	Subject, Preheader, MJML, HTML, Text string
}

var (
	mEmailFooter     = msg("email.footer")
	mEmailFooterText = msg("email.footerText")
	mEmailBlankBody  = msg("email.blankBody")
)

// starterEmail renders a template email as MJML, matching HTML and plain
// text, with the footer in l. Liquid tags ({{ unsubscribe_url }}) are filled
// in at send time.
func starterEmail(l i18n.Lang, e templateEmail) EmailContent {
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
	footer := mEmailFooter.HTML(l, i18n.Args{"workspace": i18n.HTML("{{ workspace_name }}"), "address": i18n.HTML("{{ footer_address }}")},
		i18n.Tags{"link": func(inner string) string { return `<a href="{{ unsubscribe_url }}">` + inner + `</a>` }})
	mj.WriteString(`<mj-text font-size="12px" color="#5B6472">` + footer + `</mj-text></mj-column></mj-section></mj-body></mjml>`)
	hb.WriteString(`<p style="margin:24px 0 0;font-size:12px;color:#5B6472">` + footer + `</p></td></tr></table></td></tr></table></body></html>`)
	tx.WriteString(mEmailFooterText.Text(l, i18n.Args{"url": "{{ unsubscribe_url }}"}) + "\n{{ footer_address }}\n")
	return EmailContent{Subject: e.subject, Preheader: e.preheader, MJML: mj.String(), HTML: hb.String(), Text: tx.String()}
}

// BlankEmail is the content of a new email made from the step panel.
func BlankEmail(name, subject string) EmailContent { return blankEmail(english, name, subject) }

// blankEmail is a new email in its creator's language.
func blankEmail(l i18n.Lang, name, subject string) EmailContent {
	heading := emailQuoted(name)
	if heading == "" {
		heading = name
	}
	return starterEmail(l, templateEmail{name: name, subject: subject, heading: heading, body: []string{mEmailBlankBody.Text(l, nil)}})
}

// DemoEmail renders seeded demo emails with the starter layout.
func DemoEmail(subject, preheader, heading, cta string, body ...string) EmailContent {
	return starterEmail(english, templateEmail{subject: subject, preheader: preheader, heading: heading, cta: cta, body: body})
}
