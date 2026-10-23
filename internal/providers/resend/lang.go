package resend

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What the adapter tells people (its connection fields, terms and pricing,
// the account's standing, tracking, the problems it explains) reads in each
// reader's language: the messages are in messages/<locale>.json
// (internal/providers, text.go). Resend's own names (Full access and
// Sending access keys, its menus) stay as Resend writes them.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "providers.resend")

var (
	mPricingPlan    = msgs.M("pricingPlan")
	mAccount        = msgs.M("account")
	mAccountSending = msgs.M("account.sendingKey")
	mTrackingScope  = msgs.M("tracking.scope")
	mTrackingBoth   = msgs.M("tracking.openClick")
	mTrackingOpen   = msgs.M("tracking.open")
	mTrackingClick  = msgs.M("tracking.click")
	mTrackingOff    = msgs.M("tracking.off")
	mDomainMissing  = msgs.M("error.domainMissing")
)

// say returns a message's English and the message, kept for its readers.
func say(m i18n.Msg, args i18n.Args) (string, i18n.Text) {
	return m.Text(i18n.Default, args), m.Keep(args)
}
