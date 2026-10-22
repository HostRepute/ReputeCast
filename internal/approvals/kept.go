package approvals

import "github.com/hostrepute/reputecast/internal/i18n"

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

var (
	mScheduleNow  = msgs.M("schedule.now")
	mScheduleAt   = msgs.M("schedule.at")
	mScheduleBest = msgs.M("schedule.best")
)
