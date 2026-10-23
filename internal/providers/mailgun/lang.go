package mailgun

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What the adapter tells people (its connection fields, terms and pricing,
// the account's standing, tracking, the problems it explains) reads in each
// reader's language: the messages are in messages/<locale>.json
// (internal/providers, text.go). Mailgun's own names (Private API key, HTTP
// webhook signing key, its menus) stay as Mailgun writes them.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "providers.mailgun")

var (
	mAccount       = msgs.M("account")
	mTrackingScope = msgs.M("tracking.scope")
	mTrackingUnsub = msgs.M("tracking.unsubscribe")
	mTrackingOn    = msgs.M("tracking.on")
	mTrackingOff   = msgs.M("tracking.off")
	mDomainUnknown = msgs.M("error.domainUnknown")
	mTooManyURLs   = msgs.M("error.tooManyURLs")
)

// say returns a message's English and the message, kept for its readers.
func say(m i18n.Msg, args i18n.Args) (string, i18n.Text) {
	return m.Text(i18n.Default, args), m.Keep(args)
}
