package postal

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What the adapter tells people (its connection fields, terms and pricing,
// the server's standing, the problems it explains) reads in each reader's
// language: the messages are in messages/<locale>.json (internal/providers,
// text.go). Postal's own names (its menus) stay as Postal writes them.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "providers.postal")

var (
	mAccount   = msgs.M("account")
	mBadKey    = msgs.M("error.badKey")
	mSuspended = msgs.M("error.suspended")
)

// say returns a message's English and the message, kept for its readers.
func say(m i18n.Msg, args i18n.Args) (string, i18n.Text) {
	return m.Text(i18n.Default, args), m.Keep(args)
}
