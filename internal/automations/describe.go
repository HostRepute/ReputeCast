package automations

import (
	"fmt"
	"strings"
)

// The canvas (web/src/features/automations/graph.ts) renders the same titles;
// keep the two in step.

func (r *Refs) emailName(id string) string {
	if r == nil || r.EmailNames == nil {
		return ""
	}
	return r.EmailNames[id]
}

// TypeName is the step type as the palette names it.
func TypeName(t string) string {
	switch t {
	case NodeTrigger:
		return "Trigger"
	case NodeDelay:
		return "Delay"
	case NodeEmail:
		return "Send email"
	case NodeBranch:
		return "Branch"
	case NodeFilter:
		return "Filter"
	case NodeSplit:
		return "A/B split"
	case NodeWebhook:
		return "Webhook"
	case NodeList:
		return "List action"
	case NodeGoal:
		return "Goal"
	case NodeExit:
		return "Exit"
	}
	return "Step"
}

// TriggerKindLabel names a trigger kind ("Joined list").
func TriggerKindLabel(kind string) string {
	switch kind {
	case TriggerJoinedList:
		return "Joined list"
	case TriggerFormSubmitted:
		return "Form submitted"
	case TriggerTagAdded:
		return "Tag added"
	case TriggerAPIEvent:
		return "API event"
	case TriggerDateField:
		return "Date field"
	}
	return "Trigger"
}

// TriggerRef is what a trigger points at, for "Joined list · Weekly Digest".
func TriggerRef(t *TriggerConfig, refs *Refs) string {
	if t == nil {
		return ""
	}
	switch t.Kind {
	case TriggerJoinedList:
		if refs != nil {
			if n, ok := refs.Lists[t.ListID]; ok {
				return n
			}
		}
		return "a list"
	case TriggerFormSubmitted:
		if refs != nil {
			if n, ok := refs.Forms[t.FormID]; ok {
				return n
			}
		}
		return "a form"
	case TriggerTagAdded:
		return t.Tag
	case TriggerAPIEvent:
		return t.Event
	case TriggerDateField:
		return t.Field
	}
	return ""
}

// TriggerLabel is the list's trigger column: "Joined list · Weekly Digest".
func TriggerLabel(t *TriggerConfig, refs *Refs) string {
	if t == nil || t.Kind == "" {
		return "No trigger yet"
	}
	ref := TriggerRef(t, refs)
	if ref == "" {
		return TriggerKindLabel(t.Kind)
	}
	return TriggerKindLabel(t.Kind) + " · " + ref
}

// DurationText reads "10 minutes", "1 day", "2 weeks".
func DurationText(d *FlowDuration) string {
	if d == nil {
		return ""
	}
	unit := strings.TrimSuffix(d.Unit, "s")
	if unit == "" {
		unit = "day"
	}
	if d.Amount == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", d.Amount, unit)
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

// emailNodeName is the email's name for a send step.
func emailNodeName(n FlowNode, refs *Refs) string {
	if n.Data.Email == nil {
		return ""
	}
	return refs.emailName(n.Data.Email.EmailID)
}

// NodeRef is how other text refers to a step: an email by its prefix
// ("Step 2"), anything else by its label or title.
func NodeRef(g FlowGraph, n FlowNode, refs *Refs) string {
	if n.Data.Label != "" {
		return n.Data.Label
	}
	if n.Type == NodeEmail {
		name := emailNodeName(n, refs)
		if p := emailPrefix(name); p != "" {
			return p
		}
		if name != "" {
			return name
		}
		return "Send email"
	}
	return NodeTitle(g, n, refs)
}

// inlineRef is a lower-case reference for running text ("step 1").
func inlineRef(g FlowGraph, id string, refs *Refs) string {
	n := g.Node(id)
	if n == nil {
		return "a removed step"
	}
	if n.Type == NodeEmail && n.Data.Label == "" {
		if name := emailNodeName(*n, refs); name != "" {
			return lowerFirst(EmailShortRef(name))
		}
	}
	return lowerFirst(NodeRef(g, *n, refs))
}

// stepName names a step in a sentence: `Branch "Opened step 1 within 2 days?"`.
func stepName(g FlowGraph, n FlowNode, refs *Refs) string {
	if n.Type == NodeTrigger {
		return "the trigger"
	}
	return fmt.Sprintf("%s %s", strings.ToLower(TypeName(n.Type)[:1])+TypeName(n.Type)[1:], quote(NodeRef(g, n, refs)))
}

// conditionPhrase reads a condition after "Only contacts …" or as a question.
func conditionPhrase(g FlowGraph, c *FlowCondition, refs *Refs) string {
	if c == nil {
		return ""
	}
	within := ""
	if c.Within != nil && c.Within.Amount > 0 {
		within = " within " + DurationText(c.Within)
	}
	target := "any email"
	if c.Step != "" {
		target = inlineRef(g, c.Step, refs)
	}
	switch c.Kind {
	case "opened":
		return "opened " + target + within
	case "clicked":
		if c.URLContains != "" {
			return fmt.Sprintf("clicked a %s link%s", c.URLContains, within)
		}
		return "clicked a link in " + target + within
	case "tag":
		return "tagged " + c.Tag
	case "list":
		name := "a list"
		if refs != nil && refs.Lists[c.ListID] != "" {
			name = refs.Lists[c.ListID]
		}
		return "in " + name
	case "event":
		return "sent " + c.Event
	case "field":
		switch c.Op {
		case "is_set":
			return c.Field + " is set"
		case "is_not_set":
			return c.Field + " is empty"
		case "not_equals":
			return fmt.Sprintf("%s isn't %s", c.Field, c.Value)
		case "contains":
			return fmt.Sprintf("%s contains %s", c.Field, c.Value)
		}
		return fmt.Sprintf("%s is %s", c.Field, c.Value)
	}
	return ""
}

// NodeTitle is the bold line on the canvas node.
func NodeTitle(g FlowGraph, n FlowNode, refs *Refs) string {
	d := n.Data
	switch n.Type {
	case NodeTrigger:
		if d.Trigger == nil || d.Trigger.Kind == "" {
			return "Pick a trigger"
		}
		if ref := TriggerRef(d.Trigger, refs); ref != "" {
			return ref
		}
		return TriggerKindLabel(d.Trigger.Kind)
	case NodeDelay:
		if d.Delay == nil {
			return "Wait"
		}
		return "Wait " + DurationText(d.Delay)
	case NodeEmail:
		if name := emailNodeName(n, refs); name != "" {
			return name
		}
		return "Pick an email"
	case NodeBranch:
		if d.Label != "" {
			return d.Label
		}
		if p := conditionPhrase(g, d.Condition, refs); p != "" {
			return capFirst(p) + "?"
		}
		return "Set a condition"
	case NodeFilter:
		if p := conditionPhrase(g, d.Condition, refs); p != "" {
			return "Only contacts " + strings.Replace(p, "is ", "where ", 0)
		}
		return "Set a condition"
	case NodeSplit:
		parts := make([]string, 0, len(d.Variants))
		for _, v := range d.Variants {
			parts = append(parts, fmt.Sprint(v.Weight))
		}
		title := strings.Join(parts, " / ")
		if len(d.Variants) > 0 {
			if next := g.NextOnPath(n.ID, d.Variants[0].Key); next != "" {
				title += " on " + inlineRef(g, next, refs)
			}
		}
		return title
	case NodeWebhook:
		if d.Webhook != nil && d.Webhook.Name != "" {
			return d.Webhook.Name
		}
		if d.Webhook != nil && d.Webhook.URL != "" {
			return hostOf(d.Webhook.URL)
		}
		return "Set a URL"
	case NodeList:
		a := d.Action
		if a == nil {
			return "Pick an action"
		}
		list := "a list"
		if refs != nil && refs.Lists[a.ListID] != "" {
			list = refs.Lists[a.ListID]
		}
		switch a.Kind {
		case "add_list":
			return "Add to " + list
		case "remove_list":
			return "Remove from " + list
		case "add_tag":
			return "Add tag " + a.Tag
		case "remove_tag":
			return "Remove tag " + a.Tag
		case "unsubscribe_all":
			return "Unsubscribe from all lists"
		}
		return "Pick an action"
	case NodeGoal:
		return goalTitle(d.Condition, refs)
	case NodeExit:
		if d.Label != "" {
			return d.Label
		}
		return "Leave the flow · stays on the list"
	}
	return TypeName(n.Type)
}

func goalTitle(c *FlowCondition, refs *Refs) string {
	if c == nil {
		return "Reached the end of the flow"
	}
	switch c.Kind {
	case "clicked":
		if c.URLContains != "" {
			return fmt.Sprintf("Clicked a %s link", c.URLContains)
		}
		return "Clicked any link"
	case "opened":
		return "Opened an email"
	case "tag":
		return "Tagged " + c.Tag
	case "list":
		if refs != nil && refs.Lists[c.ListID] != "" {
			return "Joined " + refs.Lists[c.ListID]
		}
		return "Joined a list"
	case "event":
		return "Sent " + c.Event
	case "field":
		return capFirst(conditionPhrase(FlowGraph{}, c, refs))
	}
	return "Goal"
}

// GoalLabel is the short goal name for the automations list.
func GoalLabel(n FlowNode, refs *Refs) string {
	if n.Data.Label != "" {
		return n.Data.Label
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

// quote wraps a name in straight double quotes, as the product copy does
// ("Step 2", "No" path); unlike %q it never escapes inner characters.
func quote(s string) string { return `"` + s + `"` }
