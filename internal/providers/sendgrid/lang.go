package sendgrid

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What the adapter tells people (its connection fields, terms and pricing,
// the account's standing, tracking, the problems it explains) reads in each
// reader's language: the messages are in messages/<locale>.json
// (internal/providers, text.go). SendGrid's own names (Event Webhook, its
// menus) stay as SendGrid writes them.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "providers.sendgrid")

var (
	mAccount           = msgs.M("account")
	mAccountRestricted = msgs.M("account.restricted")
	mTrackingScope     = msgs.M("tracking.scope")
	mTrackingOff       = msgs.M("tracking.off")
	mTrackingOn        = msgs.M("tracking.on")
	mNoWebhookID       = msgs.M("error.noWebhookId")
	mDomainMissing     = msgs.M("error.domainMissing")
	mIPsRestricted     = msgs.M("error.ipsRestricted")
)

// say returns a message's English and the message, kept for its readers.
func say(m i18n.Msg, args i18n.Args) (string, i18n.Text) {
	return m.Text(i18n.Default, args), m.Keep(args)
}
