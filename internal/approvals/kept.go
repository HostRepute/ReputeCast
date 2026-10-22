package approvals

import (
	"encoding/json"

	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// textJSON stores a kept message.
func textJSON(t i18n.Text) json.RawMessage {
	b, _ := json.Marshal(t)
	return b
}

// keptValue is a value a change or a summary shows: a message kept for its
// readers (a schedule, a version, a count), or data shown as it is (a
// subject, a link, names).
type keptValue struct {
	Text i18n.Text `json:"text,omitzero"`
	Raw  string    `json:"raw,omitempty"`
}

// text writes the value in a reader's language.
func (v keptValue) text(l i18n.Lang) string {
	if !v.Text.IsZero() {
		return render(l, v.Text)
	}
	return v.Raw
}

// automationItem is a change in the automation version a request
// publishes: added, changed or removed, and the step as automations names
// it (its kept message).
type automationItem struct {
	Kind  string    `json:"kind"`
	Title i18n.Text `json:"title,omitzero"`
}

// localAutomation is what an automation publish request covers, its
// changes in the reader's language.
func localAutomation(l i18n.Lang, snap Snapshot) *ApprovalAutomationSnapshot {
	a := snap.Automation
	if a == nil || l.Locale == i18n.English || len(snap.AutomationItems) != len(a.Changes) {
		return a
	}
	out := *a
	out.Changes = make([]string, len(a.Changes))
	for i, it := range snap.AutomationItems {
		out.Changes[i] = a.Changes[i]
		if it.Title.ID != "" || len(it.Title.Args) > 0 {
			if title := automations.TextIn(l, it.Title); title != "" {
				out.Changes[i] = mChangeKind.Text(l, i18n.Args{"kind": it.Kind, "title": title})
			}
		}
	}
	return &out
}

// withRuleTexts gives stored rules the messages kept beside them.
func withRuleTexts(rules []ApprovalRuleMatch, texts map[string]ruleTexts) []ApprovalRuleMatch {
	for i := range rules {
		if t, ok := texts[rules[i].Key]; ok {
			rules[i].texts = &t
		}
	}
	return rules
}

var (
	mScheduleNow  = msgs.M("schedule.now")
	mScheduleAt   = msgs.M("schedule.at")
	mScheduleBest = msgs.M("schedule.best")
)
