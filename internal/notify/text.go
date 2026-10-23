package notify

import (
	"encoding/json"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// A notification is written once, when its event happens, and read later by
// every member it reaches: in the bell, the notification center and email.
// So that each reads it in their own language, the event passes its title,
// summary and body as catalog messages kept with their arguments
// (msgs.M(key).Keep(args), Notification.TitleText…). ToUser stores the
// English in the title, summary and body columns, as before, and the kept
// messages beside it in the notification's data, under dataKey. Reading,
// Kept.In writes each part in the reader's language; a part without a
// message (an event its package hasn't converted yet, rows from before)
// reads in the English it stored.

// dataKey is the key of a notification's data that holds its kept text.
const dataKey = "i18n"

// Kept is a notification's title, summary and body kept as messages.
type Kept struct {
	Title   i18n.Text `json:"title,omitzero"`
	Summary i18n.Text `json:"summary,omitzero"`
	Body    i18n.Text `json:"body,omitzero"`
}

// IsZero reports whether no part is kept as a message.
func (k Kept) IsZero() bool { return k.Title.IsZero() && k.Summary.IsZero() && k.Body.IsZero() }

// KeptIn reads the kept text of a stored notification's data. Data without
// it gives a zero Kept.
func KeptIn(data []byte) Kept {
	var d struct {
		Kept Kept `json:"i18n"`
	}
	if len(data) == 0 || json.Unmarshal(data, &d) != nil {
		return Kept{}
	}
	return d.Kept
}

// Localized is a stored notification's title, summary and body in a
// reader's language: each kept part in that language, and the English that
// was stored for the others.
func (k Kept) Localized(l i18n.Lang, title, summary, body string) (string, string, string) {
	return in(k.Title, l, title), in(k.Summary, l, summary), in(k.Body, l, body)
}

func in(t i18n.Text, l i18n.Lang, english string) string {
	if s := t.In(l); s != "" {
		return s
	}
	return english
}

// Localize writes a stored notification in a reader's language, from its
// columns and its data: for readers outside this package, such as the
// Agency area's bell.
func Localize(l i18n.Lang, title, summary, body string, data []byte) (string, string, string) {
	return KeptIn(data).Localized(l, title, summary, body)
}
