package mailjet

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What the adapter tells people (its connection fields, terms and pricing,
// the API key's standing, tracking, the problems it explains) reads in each
// reader's language: the messages are in messages/<locale>.json
// (internal/providers, text.go). Mailjet's own names (its menus, the
// TrackOpens and TrackClicks settings) stay as Mailjet writes them.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "providers.mailjet")

var (
	mAccount        = msgs.M("account")
	mTrackingScope  = msgs.M("tracking.scope")
	mTrackingDetail = msgs.M("tracking.detail")
	mDomainMissing  = msgs.M("error.domainMissing")
	mCallbackTaken  = msgs.M("error.callbackTaken")
)

// say returns a message's English and the message, kept for its readers.
func say(m i18n.Msg, args i18n.Args) (string, i18n.Text) {
	return m.Text(i18n.Default, args), m.Keep(args)
}
