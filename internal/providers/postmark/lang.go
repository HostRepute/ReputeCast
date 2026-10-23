package postmark

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What the adapter tells people (its connection fields, terms and pricing,
// the account's standing, tracking) reads in each reader's language: the
// messages are in messages/<locale>.json (internal/providers, text.go).
// Postmark's own names (Server API token, the Broadcast stream, its
// settings TrackOpens and TrackLinks) stay as Postmark writes them.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "providers.postmark")

var (
	mPricingPlan    = msgs.M("pricingPlan")
	mAccount        = msgs.M("account")
	mTrackingScope  = msgs.M("tracking.scope")
	mTrackingDetail = msgs.M("tracking.detail")
	mStreamsTaken   = msgs.M("error.streamsTaken")
	mDomainMissing  = msgs.M("error.domainMissing")
)

// say returns a message's English and the message, kept for its readers.
func say(m i18n.Msg, args i18n.Args) (string, i18n.Text) {
	return m.Text(i18n.Default, args), m.Keep(args)
}
