package smtp

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What the adapter tells people (its connection fields and terms, the
// server's standing, what the connection and bounce-mailbox tests found)
// reads in each reader's language: the messages are in
// messages/<locale>.json (internal/providers, text.go). SMTP's own words
// (STARTTLS, AUTH PLAIN, ECONNREFUSED) and the server's replies stay as
// they are.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "providers.smtp")

var (
	mAccount        = msgs.M("account")
	mSummary        = msgs.M("test.summary")
	mNoHost         = msgs.M("test.noHost")
	mNoStartTLS     = msgs.M("test.noStartTLS")
	mNoLogin        = msgs.M("test.noLoginMethod")
	mRefused        = msgs.M("test.refused")
	mNotFound       = msgs.M("test.notFound")
	mTimedOut       = msgs.M("test.timedOut")
	mMailboxAuth    = msgs.M("mailbox.rejected")
	mMailboxMissing = msgs.M("mailbox.missing")
)

// say returns a message's English and the message, kept for its readers.
func say(m i18n.Msg, args i18n.Args) (string, i18n.Text) {
	return m.Text(i18n.Default, args), m.Keep(args)
}
