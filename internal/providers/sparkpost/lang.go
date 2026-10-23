package sparkpost

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What the adapter tells people (its connection fields, terms and pricing,
// the account's standing, tracking, the problems it explains) reads in each
// reader's language: the messages are in messages/<locale>.json
// (internal/providers, text.go). SparkPost's own names (its menus,
// permissions and plans) stay as SparkPost writes them.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "providers.sparkpost")

var (
	mAccount        = msgs.M("account")
	mTrackingScope  = msgs.M("tracking.scope")
	mTrackingOn     = msgs.M("tracking.on")
	mTrackingOff    = msgs.M("tracking.off")
	mNotRegistered  = msgs.M("error.notRegistered")
	mEndpointStatus = msgs.M("error.endpointStatus")
)

// say returns a message's English and the message, kept for its readers.
func say(m i18n.Msg, args i18n.Args) (string, i18n.Text) {
	return m.Text(i18n.Default, args), m.Keep(args)
}
