package webhooks

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What members read about webhooks follows their app language
// (i18n.LangFrom): the events' descriptions in Settings and the problems.
// What goes to an endpoint (payloads, headers, event names) stays English.
// The words are in messages/<locale>.json; an event another package adds
// without a message there shows its own description.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "webhooks")

var (
	mManaged         = msgs.M("problem.managed")
	mUnknownEndpoint = msgs.M("problem.unknownEndpoint")
	mUnknownEvent    = msgs.M("problem.unknownEvent")
	mOnlyFailed      = msgs.M("problem.onlyFailed")
)

// eventsIn is the event catalogue with the descriptions in a reader's
// language.
func eventsIn(l i18n.Lang) []WebhookEventInfo {
	out := make([]WebhookEventInfo, len(Events))
	for i, e := range Events {
		out[i] = e
		if m, err := msgs.Lookup("event." + e.Name); err == nil {
			out[i].Description = m.Text(l, nil)
		}
	}
	return out
}
