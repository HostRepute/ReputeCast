package manage

import (
	"embed"
	"errors"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

// What Sending providers tells people (capability chips, problems, the
// outage timeline, the webhooks-silent checks, the Adapters tab) reads in
// the reader's language (i18n.LangFrom): the messages are in
// messages/<locale>.json. What an adapter wrote (its account standing,
// tracking details, terms, field labels) carries its own messages and reads
// through providers.TextIn (internal/providers, text.go). Texts the health
// job stores for later readers (incident titles, check details) keep their
// English and the message beside it.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "providers.manage")

var (
	mBadgeBulkAllowed      = msgs.M("badge.bulkAllowed")
	mBadgeBulkStream       = msgs.M("badge.bulkStream")
	mBadgeBulkStreamsOnly  = msgs.M("badge.bulkStreamsOnly")
	mBadgeBulkYourPolicy   = msgs.M("badge.bulkYourPolicy")
	mBadgeBulk             = msgs.M("badge.bulk")
	mBadgeWebhooks         = msgs.M("badge.webhooks")
	mBadgeBounceMailbox    = msgs.M("badge.bounceMailbox")
	mBadgeDomainAPI        = msgs.M("badge.domainAPI")
	mBadgeDomainDKIM       = msgs.M("badge.domainDKIM")
	mBadgeSuppressionSync  = msgs.M("badge.suppressionSync")
	mBadgeBatchPer         = msgs.M("badge.batchPer")
	mBadgeBatch            = msgs.M("badge.batch")
	mBadgeDedicatedIP      = msgs.M("badge.dedicatedIP")
	mBadgeDedicatedOption  = msgs.M("badge.dedicatedIPsOption")
	mBadgeDedicatedIPs     = msgs.M("badge.dedicatedIPs")
	mBadgeSandboxDetection = msgs.M("badge.sandboxDetection")
	mBadgeSandboxToken     = msgs.M("badge.sandboxToken")
	mBadgeSandboxMode      = msgs.M("badge.sandboxMode")
	mBadgeSandboxAddresses = msgs.M("badge.sandboxAddresses")
	mBadgeQuotaRate        = msgs.M("badge.quotaRate")
	mBadgeQuota            = msgs.M("badge.quota")
	mBadgeQuotaManual      = msgs.M("badge.quotaManual")
	mBadgePricing          = msgs.M("badge.pricing")

	mStreamName = msgs.M("stream.name")
	mStatusPage = msgs.M("label.statusPage")
	mKindLabel  = msgs.M("label.kind")
	mSource     = msgs.M("label.source")

	mCheckResponds     = msgs.M("check.responds")
	mCheckAuthAccepted = msgs.M("check.authAccepted")
	mCheckSelfTestOK   = msgs.M("check.selfTestOK")
	mCheckSelfTest     = msgs.M("check.selfTestStatus")
	mTestUnreachable   = msgs.M("test.unreachable")

	mIncidentDown          = msgs.M("incident.down")
	mIncidentAuthDown      = msgs.M("incident.authDown")
	mIncidentFailover      = msgs.M("incident.failover")
	mIncidentHeld          = msgs.M("incident.held")
	mIncidentFailoverEnded = msgs.M("incident.failoverEnded")
	mIncidentRecovered     = msgs.M("incident.recovered")
	mIncidentSilent        = msgs.M("incident.silent")
	mIncidentResumed       = msgs.M("incident.resumed")
	mIncidentVetted        = msgs.M("incident.failoverVetted")
	mIncidentFallbackFail  = msgs.M("incident.fallbackFailed")
	mIncidentRejecting     = msgs.M("incident.rejecting")
	mIncidentAccepting     = msgs.M("incident.accepting")

	mNotifyDownTitle   = msgs.M("notify.downTitle")
	mNotifyAuthTitle   = msgs.M("notify.authTitle")
	mNotifyDownBody    = msgs.M("notify.downBody")
	mNotifyHeldTitle   = msgs.M("notify.heldTitle")
	mNotifyHeldBody    = msgs.M("notify.heldBody")
	mNotifySilentTitle = msgs.M("notify.silentTitle")
	mNotifySilentBody  = msgs.M("notify.silentBody")

	mNotifyRejectingTitle = msgs.M("notify.rejectingTitle")
	mNotifyRejectingBody  = msgs.M("notify.rejectingBody")

	mErrorSettings = msgs.M("error.settings")

	mAdapterMissing      = msgs.M("problem.adapterMissing")
	mEnterValue          = msgs.M("problem.enterValue")
	mNotCredential       = msgs.M("problem.notCredential")
	mEnterNewValue       = msgs.M("problem.enterNewValue")
	mCredentialsUnusable = msgs.M("problem.credentialsUnusable")
	mCredentialsRefused  = msgs.M("problem.credentialsRefused")
	mCredentialsRejected = msgs.M("problem.credentialsRejected")
	mPrice               = msgs.M("problem.price")
	mStillCarries        = msgs.M("problem.stillCarries")
	mTrackingNoAPI       = msgs.M("problem.trackingNoAPI")
	mTrackingRefused     = msgs.M("problem.trackingRefused")
	mWebhookManual       = msgs.M("problem.webhookManual")
	mWebhookRefused      = msgs.M("problem.webhookRefused")
	mNoBackfill          = msgs.M("problem.noBackfill")
	mNoBroadcastStream   = msgs.M("problem.noBroadcastStream")
	mStreamRefused       = msgs.M("problem.streamRefused")
	mNothingHeld         = msgs.M("problem.nothingHeld")
	mPickTarget          = msgs.M("problem.pickTarget")
	mTargetMissing       = msgs.M("problem.targetMissing")
	mTargetUnavailable   = msgs.M("problem.targetUnavailable")

	mAdminOnly          = msgs.M("adapters.adminOnly")
	mNoRemote           = msgs.M("adapters.noRemote")
	mManifestUnusable   = msgs.M("adapters.manifestUnusable")
	mMappingUnusable    = msgs.M("adapters.mappingUnusable")
	mKeyTaken           = msgs.M("adapters.keyTaken")
	mInstalled          = msgs.M("adapters.installed")
	mUpToDate           = msgs.M("adapters.upToDate")
	mUnreadable         = msgs.M("adapters.unreadable")
	mRenamed            = msgs.M("adapters.renamed")
	mKindChanged        = msgs.M("adapters.kindChanged")
	mNewUnusable        = msgs.M("adapters.newUnusable")
	mKeyChanged         = msgs.M("adapters.keyChanged")
	mNewMappingUnusable = msgs.M("adapters.newMappingUnusable")
	mInUse              = msgs.M("adapters.inUse")
	mHeaderTwice        = msgs.M("adapters.headerTwice")
	mBodyNotJSON        = msgs.M("adapters.bodyNotJson")
	mNameTaken          = msgs.M("adapters.nameTaken")
	mNameInstalled      = msgs.M("adapters.nameInstalled")
	mNoHTTP             = msgs.M("adapters.noHTTP")
	mEndpointMethod     = msgs.M("adapters.endpointMethod")
	mEndpointURL        = msgs.M("adapters.endpointURL")
	mManifestURL        = msgs.M("adapters.manifestURL")
	mDownload           = msgs.M("adapters.download")
	mStatus             = msgs.M("adapters.status")
	mNotJSON            = msgs.M("adapters.notJson")
	mIncomplete         = msgs.M("adapters.incomplete")
)

// keep returns a message's English and the message, kept for readers who
// come later.
func keep(m i18n.Msg, args i18n.Args) (string, i18n.Text) {
	return m.Text(i18n.Default, args), m.Keep(args)
}

// streamName is a stream's name in the reader's language ("Broadcasts").
func streamName(l i18n.Lang, stream string) string {
	return mStreamName.Text(l, i18n.Args{"stream": stream})
}

// problem is an error whose text is shown to the person as it is, written
// in their language when it reaches them (problemIn).
type problem struct {
	m    i18n.Msg
	args i18n.Args
}

func (p problem) Error() string { return p.m.Text(i18n.Default, p.args) }

// problemIn writes err for the reader: a problem in their language, an
// adapter's error with its message (providers.ErrorIn), anything else as it
// is.
func problemIn(l i18n.Lang, err error) string {
	var p problem
	if errors.As(err, &p) {
		return p.m.Text(l, p.args)
	}
	return providers.ErrorIn(err, l)
}
