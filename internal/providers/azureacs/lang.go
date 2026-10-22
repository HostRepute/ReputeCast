package azureacs

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What the adapter tells people (its connection fields, terms and pricing,
// the resource's standing, tracking, the problems it explains) reads in each
// reader's language: the messages are in messages/<locale>.json
// (internal/providers, text.go). Azure's own names (Entra ID, Event Grid,
// Communication Services, its settings) stay as Microsoft writes them.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "providers.azureacs")

var (
	mAccount        = msgs.M("account")
	mTrackingScope  = msgs.M("tracking.scope")
	mTrackingDetail = msgs.M("tracking.detail")
	mNoEndpoint     = msgs.M("error.noEndpoint")
	mNoCredentials  = msgs.M("error.noCredentials")
	mNoToken        = msgs.M("error.noToken")
)

// say returns a message's English and the message, kept for its readers.
func say(m i18n.Msg, args i18n.Args) (string, i18n.Text) {
	return m.Text(i18n.Default, args), m.Keep(args)
}
