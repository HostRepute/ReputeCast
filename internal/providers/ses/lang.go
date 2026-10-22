package ses

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What the adapter tells people (its connection fields, terms and pricing,
// the account's standing, tracking, the provisioning rows, an AWS incident)
// reads in each reader's language: the English stays in the code and the
// messages it comes from are in messages/<locale>.json (internal/providers,
// text.go). Names AWS gives things (Access key ID, configuration set, SNS
// topic, region names) stay as AWS writes them.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "providers.ses")

var (
	mAccount        = msgs.M("account")
	mScopeSet       = msgs.M("tracking.scope")
	mScopeNone      = msgs.M("tracking.scopeNone")
	mTrackingNone   = msgs.M("tracking.none")
	mTrackingBoth   = msgs.M("tracking.openClick")
	mTrackingOpen   = msgs.M("tracking.open")
	mTrackingClick  = msgs.M("tracking.click")
	mTrackingOff    = msgs.M("tracking.off")
	mStepTopic      = msgs.M("provision.topic")
	mStepConfirm    = msgs.M("provision.confirm")
	mStepConfigSet  = msgs.M("provision.configSet")
	mDoneTopic      = msgs.M("provision.topicDone")
	mDoneConfirm    = msgs.M("provision.confirmDone")
	mDoneConfigSet  = msgs.M("provision.configSetDone")
	mStatusIncident = msgs.M("status.incident")
)

// say returns a message's English and the message, kept for its readers.
func say(m i18n.Msg, args i18n.Args) (string, i18n.Text) {
	return m.Text(i18n.Default, args), m.Keep(args)
}
