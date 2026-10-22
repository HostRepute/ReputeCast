package approvals

import (
	"encoding/json"

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
