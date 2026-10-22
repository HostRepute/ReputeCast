package automations

import (
	"fmt"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The canvas (web/src/features/automations/graph.ts) renders the same titles;
// keep the two in step. Each name here is a text (lang.go): the exported
// functions write it in English, for agents, webhook payloads and the audit
// log; screens write it in the reader's language with say.

var (
	mTypeTrigger = msg("type.trigger")
	mTypeDelay   = msg("type.delay")
	mTypeEmail   = msg("type.email")
	mTypeBranch  = msg("type.branch")
	mTypeFilter  = msg("type.filter")
	mTypeSplit   = msg("type.split")
	mTypeWebhook = msg("type.webhook")
	mTypeList    = msg("type.list")
	mTypeGoal    = msg("type.goal")
	mTypeExit    = msg("type.exit")
	mTypeSMS     = msg("type.sms")
	mTypeStep    = msg("type.step")

	mTriggerJoinedList        = msg("trigger.joinedList")
	mTriggerFormSubmitted     = msg("trigger.formSubmitted")
	mTriggerTagAdded          = msg("trigger.tagAdded")
	mTriggerAPIEvent          = msg("trigger.apiEvent")
	mTriggerDateField         = msg("trigger.dateField")
	mTriggerScoreCrosses      = msg("trigger.scoreCrosses")
	mTriggerPollAnswered      = msg("trigger.pollAnswered")
	mTriggerBoughtProduct     = msg("trigger.boughtProduct")
	mTriggerMilestoneReached  = msg("trigger.milestoneReached")
	mTriggerOrderPlaced       = msg("trigger.orderPlaced")
	mTriggerFirstOrder        = msg("trigger.firstOrder")
	mTriggerAbandonedCheckout = msg("trigger.abandonedCheckout")
	mTriggerOther             = msg("trigger.other")
	mTriggerLabel             = msg("trigger.label")
	mTriggerNone              = msg("trigger.none")

	mRefAList          = msg("ref.aList")
	mRefAForm          = msg("ref.aForm")
	mRefAPoll          = msg("ref.aPoll")
	mRefAMilestone     = msg("ref.aMilestone")
	mRefAnyMilestone   = msg("ref.anyMilestone")
	mRefAStore         = msg("ref.aStore")
	mRefAnyProduct     = msg("ref.anyProduct")
	mRefDeletedProduct = msg("ref.deletedProduct")
	mRefRemovedStep    = msg("ref.removedStep")
	mRefTrigger        = msg("ref.trigger")
	mRefSendEmail      = msg("ref.sendEmail")
	mRefAnyEmail       = msg("ref.anyEmail")
	mRefAnyText        = msg("ref.anyText")
	mRefMilestone      = msg("ref.milestone")

	mDuration          = msg("duration")
	mDurationAdjective = msg("durationAdjective")

	mCondOpened           = msg("cond.opened")
	mCondOpenedWithin     = msg("cond.openedWithin")
	mCondClickedURL       = msg("cond.clickedUrl")
	mCondClickedURLWithin = msg("cond.clickedUrlWithin")
	mCondClickedIn        = msg("cond.clickedIn")
	mCondClickedInWithin  = msg("cond.clickedInWithin")
	mCondTagged           = msg("cond.tagged")
	mCondInList           = msg("cond.inList")
	mCondSentEvent        = msg("cond.sentEvent")
	mCondBought           = msg("cond.bought")
	mCondPlacedOrder      = msg("cond.placedOrder")
	mCondFieldSet         = msg("cond.fieldSet")
	mCondFieldEmpty       = msg("cond.fieldEmpty")
	mCondFieldNot         = msg("cond.fieldNot")
	mCondFieldContains    = msg("cond.fieldContains")
	mCondFieldIs          = msg("cond.fieldIs")
	mCondFieldOneOf       = msg("cond.fieldOneOf")
	mCondSMSConsent       = msg("cond.smsConsent")
	mCondSMSConsentAny    = msg("cond.smsConsentAny")
	mCondSMSClicked       = msg("cond.smsClicked")
	mCondSMSClickedWithin = msg("cond.smsClickedWithin")
	mCondSMSReplied       = msg("cond.smsReplied")
	mCondSMSRepliedWithin = msg("cond.smsRepliedWithin")

	mTitlePickTrigger    = msg("title.pickTrigger")
	mTitleWait           = msg("title.wait")
	mTitleWaitFor        = msg("title.waitFor")
	mTitlePickEmail      = msg("title.pickEmail")
	mTitleBranch         = msg("title.branch")
	mTitleSetCondition   = msg("title.setCondition")
	mTitleFilter         = msg("title.filter")
	mTitleSplitOn        = msg("title.splitOn")
	mTitleSetURL         = msg("title.setUrl")
	mTitlePickAction     = msg("title.pickAction")
	mTitleAddToList      = msg("title.addToList")
	mTitleRemoveFromList = msg("title.removeFromList")
	mTitleAddTag         = msg("title.addTag")
	mTitleRemoveTag      = msg("title.removeTag")
	mTitleUnsubscribeAll = msg("title.unsubscribeAll")
	mTitleExit           = msg("title.exit")
	mTitleSMS            = msg("title.sms")
	mTitleWriteText      = msg("title.writeText")

	mGoalEnd         = msg("goal.end")
	mGoalClickedURL  = msg("goal.clickedUrl")
	mGoalClickedAny  = msg("goal.clickedAny")
	mGoalOpened      = msg("goal.opened")
	mGoalTagged      = msg("goal.tagged")
	mGoalJoined      = msg("goal.joined")
	mGoalJoinedAny   = msg("goal.joinedAny")
	mGoalSent        = msg("goal.sent")
	mGoalBought      = msg("goal.bought")
	mGoalBoughtAny   = msg("goal.boughtAny")
	mGoalPlacedOrder = msg("goal.placedOrder")
	mGoalOther       = msg("goal.other")
)

func (r *Refs) emailName(id string) string {
	if r == nil || r.EmailNames == nil {
		return ""
	}
	return r.EmailNames[id]
}

// TypeName is the step type as the palette names it.
func TypeName(t string) string { return say(english, typeName(t)) }

func typeName(t string) i18n.Text {
	switch t {
	case NodeTrigger:
		return txt(mTypeTrigger)
	case NodeDelay:
		return txt(mTypeDelay)
	case NodeEmail:
		return txt(mTypeEmail)
	case NodeBranch:
		return txt(mTypeBranch)
	case NodeFilter:
		return txt(mTypeFilter)
	case NodeSplit:
		return txt(mTypeSplit)
	case NodeWebhook:
		return txt(mTypeWebhook)
	case NodeList:
		return txt(mTypeList)
	case NodeGoal:
		return txt(mTypeGoal)
	case NodeExit:
		return txt(mTypeExit)
	case NodeSMS:
		return txt(mTypeSMS)
	}
	return txt(mTypeStep)
}

// TriggerKindLabel names a trigger kind ("Joined list").
func TriggerKindLabel(kind string) string { return say(english, triggerKindLabel(kind)) }

func triggerKindLabel(kind string) i18n.Text {
	switch kind {
	case TriggerJoinedList:
		return txt(mTriggerJoinedList)
	case TriggerFormSubmitted:
		return txt(mTriggerFormSubmitted)
	case TriggerTagAdded:
		return txt(mTriggerTagAdded)
	case TriggerAPIEvent:
		return txt(mTriggerAPIEvent)
	case TriggerDateField:
		return txt(mTriggerDateField)
	case TriggerScoreCrosses:
		return txt(mTriggerScoreCrosses)
	case TriggerPollAnswered:
		return txt(mTriggerPollAnswered)
	case TriggerBoughtProduct:
		return txt(mTriggerBoughtProduct)
	case TriggerMilestoneReached:
		return txt(mTriggerMilestoneReached)
	case TriggerOrderPlaced:
		return txt(mTriggerOrderPlaced)
	case TriggerFirstOrder:
		return txt(mTriggerFirstOrder)
	case TriggerAbandonedCheckout:
		return txt(mTriggerAbandonedCheckout)
	}
	return txt(mTriggerOther)
}

// TriggerRef is what a trigger points at, for "Joined list · Weekly Digest".
func TriggerRef(t *AutomationTrigger, refs *Refs) string {
	return say(english, triggerRef(t, refs))
}

// triggerRef is empty (no ID, no arguments) when the trigger points at nothing.
func triggerRef(t *AutomationTrigger, refs *Refs) i18n.Text {
	if t == nil {
		return i18n.Text{}
	}
	switch t.Kind {
	case TriggerJoinedList:
		if refs != nil {
			if n, ok := refs.Lists[t.ListID]; ok {
				return nameOrNothing(n)
			}
		}
		return txt(mRefAList)
	case TriggerFormSubmitted:
		if refs != nil {
			if n, ok := refs.Forms[t.FormID]; ok {
				return nameOrNothing(n)
			}
		}
		return txt(mRefAForm)
	case TriggerTagAdded:
		return nameOrNothing(t.Tag)
	case TriggerAPIEvent:
		return nameOrNothing(t.Event)
	case TriggerDateField:
		return nameOrNothing(t.Field)
	case TriggerScoreCrosses:
		return scoreTriggerRef(t)
	case TriggerPollAnswered:
		if refs != nil {
			if n, ok := refs.Polls[t.Poll]; ok {
				return nameOrNothing(n)
			}
		}
		return txt(mRefAPoll)
	case TriggerBoughtProduct:
		return productRef(t.ProductID, refs)
	case TriggerMilestoneReached:
		if t.MilestoneID == "" {
			return txt(mRefAnyMilestone)
		}
		if refs != nil {
			if m, ok := refs.milestones[t.MilestoneID]; ok {
				return milestoneText(m.count, m.reward)
			}
			if n, ok := refs.Milestones[t.MilestoneID]; ok {
				return nameOrNothing(n)
			}
		}
		return txt(mRefAMilestone)
	case TriggerOrderPlaced, TriggerFirstOrder, TriggerAbandonedCheckout:
		if refs != nil {
			if n, ok := refs.Stores[t.Store]; ok {
				return nameOrNothing(n)
			}
		}
		return txt(mRefAStore)
	}
	return i18n.Text{}
}

// nameOrNothing is a name, or nothing when it's empty.
func nameOrNothing(s string) i18n.Text {
	if s == "" {
		return i18n.Text{}
	}
	return name(s)
}

// milestone is a referral milestone as a flow names it.
type milestone struct {
	count  int32
	reward string
}

// milestoneText is how a flow names a referral milestone: "3 referrals ·
// The Acme Field Guide".
func milestoneText(count int32, reward string) i18n.Text {
	return txt(mRefMilestone, "count", count, "reward", reward)
}

// productRef names a "Bought a product" trigger's or goal's product: any
// product, or one that's gone.
func productRef(id string, refs *Refs) i18n.Text {
	if id == "" {
		return txt(mRefAnyProduct)
	}
	if refs != nil {
		if n, ok := refs.Products[id]; ok {
			return name(n)
		}
	}
	return txt(mRefDeletedProduct)
}

// TriggerLabel is the list's trigger column: "Joined list · Weekly Digest".
func TriggerLabel(t *AutomationTrigger, refs *Refs) string {
	return say(english, triggerLabel(t, refs))
}

func triggerLabel(t *AutomationTrigger, refs *Refs) i18n.Text {
	if t == nil || t.Kind == "" {
		return txt(mTriggerNone)
	}
	ref := triggerRef(t, refs)
	if emptyText(ref) {
		return triggerKindLabel(t.Kind)
	}
	return txt(mTriggerLabel, "kind", triggerKindLabel(t.Kind), "ref", ref)
}

// DurationText reads "10 minutes", "1 day", "2 weeks".
func DurationText(d *AutomationDuration) string { return say(english, durationText(d)) }

func durationText(d *AutomationDuration) i18n.Text {
	if d == nil {
		return i18n.Text{}
	}
	return txt(mDuration, "unit", d.Unit, "n", d.Amount)
}

// durationAdjective reads "2-day", "10-minute".
func durationAdjective(d *AutomationDuration) i18n.Text {
	if d == nil {
		return i18n.Text{}
	}
	return txt(mDurationAdjective, "unit", d.Unit, "n", d.Amount)
}

// emailPrefix is the part of an email name before its quoted subject:
// `Welcome · step 1 "You're in"` → "Welcome · step 1".
func emailPrefix(name string) string {
	if i := strings.IndexAny(name, "\"“"); i >= 0 {
		name = name[:i]
	}
	return strings.TrimRight(strings.TrimSpace(name), " ·")
}

// emailQuoted is the quoted part of an email name, without quotes.
func emailQuoted(name string) string {
	i := strings.IndexAny(name, "\"“")
	if i < 0 {
		return ""
	}
	rest := name[i+1:]
	if j := strings.IndexAny(rest, "\"”"); j >= 0 {
		return rest[:j]
	}
	return rest
}

// EmailShortRef is the last segment of the prefix: "Welcome · step 1" → "step 1".
func EmailShortRef(name string) string {
	p := emailPrefix(name)
	if i := strings.LastIndex(p, " · "); i >= 0 {
		p = p[i+len(" · "):]
	}
	if p == "" {
		return strings.Trim(name, "\"“” ")
	}
	return p
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	// Only ordinary words: "Step 2" → "step 2", but "A/B split" and "SES" stay.
	if r[0] >= 'A' && r[0] <= 'Z' && len(r) > 1 && r[1] >= 'a' && r[1] <= 'z' {
		r[0] += 'a' - 'A'
	}
	return string(r)
}

func emailNodeName(n AutomationNode, refs *Refs) string {
	if n.Data.Email == nil {
		return ""
	}
	return refs.emailName(n.Data.Email.EmailID)
}

// NodeRef is how other text refers to a step: an email by its prefix
// ("Step 2"), anything else by its label or title.
func NodeRef(g AutomationGraph, n AutomationNode, refs *Refs) string {
	return say(english, nodeRef(g, n, refs))
}

func nodeRef(g AutomationGraph, n AutomationNode, refs *Refs) i18n.Text {
	if n.Data.Label != "" {
		return name(n.Data.Label)
	}
	if n.Type == NodeEmail {
		nm := emailNodeName(n, refs)
		if p := emailPrefix(nm); p != "" {
			return name(p)
		}
		if nm != "" {
			return name(nm)
		}
		return txt(mRefSendEmail)
	}
	return nodeTitle(g, n, refs)
}

// inlineRef is a lower-case reference for running text ("step 1").
func inlineRef(g AutomationGraph, id string, refs *Refs) i18n.Text {
	n := g.Node(id)
	if n == nil {
		return txt(mRefRemovedStep)
	}
	if n.Type == NodeEmail && n.Data.Label == "" {
		if nm := emailNodeName(*n, refs); nm != "" {
			return lower(name(EmailShortRef(nm)))
		}
	}
	return lower(nodeRef(g, *n, refs))
}

// stepName names a step in a sentence: `branch "Opened step 1 within 2 days?"`.
func stepName(g AutomationGraph, n AutomationNode, refs *Refs) i18n.Text {
	if n.Type == NodeTrigger {
		return txt(mRefTrigger)
	}
	return txt(mStepNamed, "type", lowerType(typeName(n.Type)), "step", nodeRef(g, n, refs))
}

// conditionPhrase reads a condition after "Only contacts …" or as a
// question; it's empty for a condition it can't read.
func conditionPhrase(g AutomationGraph, c *AutomationCondition, refs *Refs) i18n.Text {
	if c == nil {
		return i18n.Text{}
	}
	var within i18n.Text
	if c.Within != nil && c.Within.Amount > 0 {
		within = durationText(c.Within)
	}
	withWindow := func(plain, windowed i18n.Msg, kv ...any) i18n.Text {
		if emptyText(within) {
			return txt(plain, kv...)
		}
		return txt(windowed, append(kv, "duration", within)...)
	}
	target := txt(mRefAnyEmail)
	if c.Step != "" {
		target = inlineRef(g, c.Step, refs)
	}
	switch c.Kind {
	case CondSMSConsent, CondSMSClicked, CondSMSReplied:
		return smsPhrase(g, c, refs, within)
	case "opened":
		return withWindow(mCondOpened, mCondOpenedWithin, "step", target)
	case "clicked":
		if c.URLContains != "" {
			return withWindow(mCondClickedURL, mCondClickedURLWithin, "url", c.URLContains)
		}
		return withWindow(mCondClickedIn, mCondClickedInWithin, "step", target)
	case "tag":
		return txt(mCondTagged, "tag", c.Tag)
	case "list":
		list := txt(mRefAList)
		if refs != nil && refs.Lists[c.ListID] != "" {
			list = name(refs.Lists[c.ListID])
		}
		return txt(mCondInList, "list", list)
	case "event":
		return txt(mCondSentEvent, "event", c.Event)
	case TriggerBoughtProduct:
		return txt(mCondBought, "product", productRef(c.ProductID, refs))
	case ConditionPlacedOrder:
		return txt(mCondPlacedOrder)
	case "field":
		switch c.Op {
		case "is_set":
			return txt(mCondFieldSet, "field", c.Field)
		case "is_not_set":
			return txt(mCondFieldEmpty, "field", c.Field)
		case "not_equals":
			return txt(mCondFieldNot, "field", c.Field, "value", c.Value)
		case "contains":
			return txt(mCondFieldContains, "field", c.Field, "value", c.Value)
		case "is_one_of":
			return txt(mCondFieldOneOf, "field", c.Field, "values", orValuesText(c.Value))
		}
		return txt(mCondFieldIs, "field", c.Field, "value", c.Value)
	}
	return i18n.Text{}
}

// NodeTitle is the bold line on the canvas node.
func NodeTitle(g AutomationGraph, n AutomationNode, refs *Refs) string {
	return say(english, nodeTitle(g, n, refs))
}

func nodeTitle(g AutomationGraph, n AutomationNode, refs *Refs) i18n.Text {
	d := n.Data
	switch n.Type {
	case NodeTrigger:
		if d.Trigger == nil || d.Trigger.Kind == "" {
			return txt(mTitlePickTrigger)
		}
		if ref := triggerRef(d.Trigger, refs); !emptyText(ref) {
			return ref
		}
		return triggerKindLabel(d.Trigger.Kind)
	case NodeDelay:
		if d.Delay == nil {
			return txt(mTitleWait)
		}
		return txt(mTitleWaitFor, "duration", durationText(d.Delay))
	case NodeEmail:
		if nm := emailNodeName(n, refs); nm != "" {
			return name(nm)
		}
		return txt(mTitlePickEmail)
	case NodeBranch:
		if d.Label != "" {
			return name(d.Label)
		}
		if p := conditionPhrase(g, d.Condition, refs); !emptyText(p) {
			return txt(mTitleBranch, "condition", upper(p))
		}
		return txt(mTitleSetCondition)
	case NodeFilter:
		if p := conditionPhrase(g, d.Condition, refs); !emptyText(p) {
			return txt(mTitleFilter, "condition", p)
		}
		return txt(mTitleSetCondition)
	case NodeSplit:
		weights := name(splitWeights(n))
		if len(d.Variants) > 0 {
			if next := g.NextOnPath(n.ID, d.Variants[0].Key); next != "" {
				return txt(mTitleSplitOn, "weights", weights, "step", inlineRef(g, next, refs))
			}
		}
		return weights
	case NodeWebhook:
		if d.Webhook != nil && d.Webhook.Name != "" {
			return name(d.Webhook.Name)
		}
		if d.Webhook != nil && d.Webhook.URL != "" {
			return name(hostOf(d.Webhook.URL))
		}
		return txt(mTitleSetURL)
	case NodeList:
		a := d.Action
		if a == nil {
			return txt(mTitlePickAction)
		}
		list := txt(mRefAList)
		if refs != nil && refs.Lists[a.ListID] != "" {
			list = name(refs.Lists[a.ListID])
		}
		switch a.Kind {
		case "add_list":
			return txt(mTitleAddToList, "list", list)
		case "remove_list":
			return txt(mTitleRemoveFromList, "list", list)
		case "add_tag":
			return txt(mTitleAddTag, "tag", a.Tag)
		case "remove_tag":
			return txt(mTitleRemoveTag, "tag", a.Tag)
		case "unsubscribe_all":
			return txt(mTitleUnsubscribeAll)
		}
		return txt(mTitlePickAction)
	case NodeGoal:
		return goalTitle(d.Condition, refs)
	case NodeExit:
		if d.Label != "" {
			return name(d.Label)
		}
		return txt(mTitleExit)
	case NodeSMS:
		if d.Label != "" {
			return name(d.Label)
		}
		if d.SMS != nil && strings.TrimSpace(d.SMS.Body) != "" {
			return txt(mTitleSMS)
		}
		return txt(mTitleWriteText)
	}
	return typeName(n.Type)
}

// splitWeights reads an A/B split's weights: "50 / 50".
func splitWeights(n AutomationNode) string {
	parts := make([]string, 0, len(n.Data.Variants))
	for _, v := range n.Data.Variants {
		parts = append(parts, fmt.Sprint(v.Weight))
	}
	return strings.Join(parts, " / ")
}

func goalTitle(c *AutomationCondition, refs *Refs) i18n.Text {
	if c == nil {
		return txt(mGoalEnd)
	}
	switch c.Kind {
	case "clicked":
		if c.URLContains != "" {
			return txt(mGoalClickedURL, "url", c.URLContains)
		}
		return txt(mGoalClickedAny)
	case "opened":
		return txt(mGoalOpened)
	case "tag":
		return txt(mGoalTagged, "tag", c.Tag)
	case "list":
		if refs != nil && refs.Lists[c.ListID] != "" {
			return txt(mGoalJoined, "list", refs.Lists[c.ListID])
		}
		return txt(mGoalJoinedAny)
	case "event":
		return txt(mGoalSent, "event", c.Event)
	case TriggerBoughtProduct:
		if c.ProductID == "" {
			return txt(mGoalBoughtAny)
		}
		return txt(mGoalBought, "product", productRef(c.ProductID, refs))
	case ConditionPlacedOrder:
		return txt(mGoalPlacedOrder)
	case "field", CondSMSConsent, CondSMSClicked, CondSMSReplied:
		return upper(conditionPhrase(AutomationGraph{}, c, refs))
	}
	return txt(mGoalOther)
}

// smsPhrase is a condition on texts: "has SMS consent for Acme Summit
// 2026", "clicked a link in Day-before reminder".
func smsPhrase(g AutomationGraph, c *AutomationCondition, refs *Refs, within i18n.Text) i18n.Text {
	text := txt(mRefAnyText)
	if c.Step != "" {
		if g.Node(c.Step) != nil {
			text = inlineRef(g, c.Step, refs)
		}
	}
	switch c.Kind {
	case CondSMSConsent:
		if c.ListID == "" {
			return txt(mCondSMSConsentAny)
		}
		list := txt(mRefAList)
		if refs != nil && refs.Lists[c.ListID] != "" {
			list = name(refs.Lists[c.ListID])
		}
		return txt(mCondSMSConsent, "list", list)
	case CondSMSClicked:
		if emptyText(within) {
			return txt(mCondSMSClicked, "text", text)
		}
		return txt(mCondSMSClickedWithin, "text", text, "duration", within)
	}
	if emptyText(within) {
		return txt(mCondSMSReplied, "text", text)
	}
	return txt(mCondSMSRepliedWithin, "text", text, "duration", within)
}

// GoalLabel is the short goal name for the automations list.
func GoalLabel(n AutomationNode, refs *Refs) string { return say(english, goalLabel(n, refs)) }

func goalLabel(n AutomationNode, refs *Refs) i18n.Text {
	if n.Data.Label != "" {
		return name(n.Data.Label)
	}
	return goalTitle(n.Data.Condition, refs)
}

func hostOf(raw string) string {
	s := strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	return s
}

// shortURL keeps the host and hides the path: "https://hooks.slack.com/…".
func shortURL(raw string) string {
	scheme := "https://"
	if strings.HasPrefix(raw, "http://") {
		scheme = "http://"
	}
	host := hostOf(raw)
	if strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(raw, scheme), host), "/") == "" {
		return scheme + host
	}
	return scheme + host + "/…"
}
