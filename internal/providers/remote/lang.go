package remote

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// A community adapter writes English: its name, terms note, pricing and
// connection fields read as its author wrote them. What ReputeCast writes
// about it reads in each reader's language: the capability labels of the
// Adapters tab, the terms note when the manifest has none, the descriptor's
// words, the problems of the runtime and the errors of a call to the
// adapter process (kept with the call, providers.CallError.Text). The
// messages are in messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "providers.remote")

var (
	mLabelWebhooks        = msgs.M("label.webhooks")
	mLabelSuppressionPull = msgs.M("label.suppressionPull")
	mLabelSuppressionPush = msgs.M("label.suppressionPush")
	mLabelSuppressionSync = msgs.M("label.suppressionSync")
	mLabelBatch           = msgs.M("label.batch")
	mLabelDomainAPI       = msgs.M("label.domainAPI")
	mLabelDomainDKIM      = msgs.M("label.domainDKIM")
	mLabelQuota           = msgs.M("label.quota")
	mLabelQuotaRate       = msgs.M("label.quotaRate")

	mTermsAllowed    = msgs.M("terms.allowed")
	mTermsForbidden  = msgs.M("terms.forbidden")
	mTermsYourPolicy = msgs.M("terms.yourPolicy")

	mEventsWebhooks    = msgs.M("events.webhooks")
	mEventsNone        = msgs.M("events.none")
	mDomainAPI         = msgs.M("domain.api")
	mDomainDKIM        = msgs.M("domain.dkim")
	mDomainManual      = msgs.M("domain.manual")
	mSuppressionPull   = msgs.M("suppression.pull")
	mSuppressionPush   = msgs.M("suppression.push")
	mSuppressionTwoWay = msgs.M("suppression.twoWay")
	mSuppressionLocal  = msgs.M("suppression.local")
	mCallEndpoint      = msgs.M("call.endpointRefused")
	mCallNoAddress     = msgs.M("call.noAddress")
	mCallPaused        = msgs.M("call.paused")
	mCallBusy          = msgs.M("call.busy")
	mCallUnreachable   = msgs.M("call.unreachable")
	mCallCutOff        = msgs.M("call.cutOff")
	mCallInvalid       = msgs.M("call.invalid")
	mCallSignature     = msgs.M("call.signature")
	mCallRefused       = msgs.M("call.refused")
	mCallFailed        = msgs.M("call.failed")
	mProblemManifest   = msgs.M("problem.manifestUnusable")
	mProblemKeyTaken   = msgs.M("problem.keyTaken")
	mProblemAddress    = msgs.M("problem.address")
	mProblemInUse      = msgs.M("problem.inUse")
	mProblemAdminOnly  = msgs.M("problem.adminOnly")
)

// say returns a message's English and the message, kept for its readers.
func say(m i18n.Msg, args i18n.Args) (string, i18n.Text) {
	return m.Text(i18n.Default, args), m.Keep(args)
}
