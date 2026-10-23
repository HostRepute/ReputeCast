package connect

import (
	"embed"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

// What the connect flow and Settings › Providers tell people reads in the
// reader's language (i18n.LangFrom): the catalog's wording, the stepper,
// the steps, the test's timeline, the summary and every problem. The
// messages are in messages/<locale>.json; the wording of the providers the
// design names is there too, under adapter.<key>. What an adapter writes
// itself (its account standing, field labels, provisioning rows, errors)
// carries its own messages and reads through providers.TextIn.
//
// The setup's state (provider_setups.state) keeps what a step found for
// whoever opens the setup next: the English as it was, and the message
// beside it (Connection.AccountText, StepError.Texts, ProvisionState.Texts,
// ProbeState.SummaryText and ErrorText, SelfTestState.ReasonText,
// Registration.DetailText, SuppressionPreview.ErrorText). Rows written
// before the messages were kept read in English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "connect")

var (
	mInfoTitle                   = msgs.M("info.title")
	mInfoHeadingConnect          = msgs.M("info.headingConnect")
	mInfoHeadingAPIKey           = msgs.M("info.headingAPIKey")
	mInfoLeadDefault             = msgs.M("info.leadDefault")
	mInfoLeadRegisters           = msgs.M("info.leadRegisters")
	mInfoConnectionHintDefault   = msgs.M("info.connectionHintDefault")
	mInfoSectionDefault          = msgs.M("info.sectionDefault")
	mInfoSectionAPIKey           = msgs.M("info.sectionAPIKey")
	mInfoWebhookEventsDefault    = msgs.M("info.webhookEventsDefault")
	mTermsRecommended            = msgs.M("terms.recommended")
	mTermsBulkAllowed            = msgs.M("terms.bulkAllowed")
	mTermsBroadcastStreams       = msgs.M("terms.broadcastStreams")
	mTermsTransactionalOnly      = msgs.M("terms.transactionalOnly")
	mTermsDependsOnServer        = msgs.M("terms.dependsOnServer")
	mTermsOwnPolicy              = msgs.M("terms.ownPolicy")
	mChipWebhooks                = msgs.M("chip.webhooks")
	mChipWebhooksVia             = msgs.M("chip.webhooksVia")
	mChipEventWebhook            = msgs.M("chip.eventWebhook")
	mChipEventGrid               = msgs.M("chip.eventGrid")
	mChipBounceMailbox           = msgs.M("chip.bounceMailbox")
	mChipDomainAPI               = msgs.M("chip.domainAPI")
	mChipDKIMAPI                 = msgs.M("chip.dkimAPI")
	mChipDomainManual            = msgs.M("chip.domainManual")
	mChipSuppressionPull         = msgs.M("chip.suppressionPull")
	mChipSuppressionSync         = msgs.M("chip.suppressionSync")
	mChipSuppressionLocal        = msgs.M("chip.suppressionLocal")
	mChipBatchN                  = msgs.M("chip.batchN")
	mChipBatch                   = msgs.M("chip.batch")
	mChipDedicatedIP             = msgs.M("chip.dedicatedIP")
	mChipDedicatedDepends        = msgs.M("chip.dedicatedDepends")
	mChipDedicatedPaid           = msgs.M("chip.dedicatedPaid")
	mChipDedicatedIPs            = msgs.M("chip.dedicatedIPs")
	mChipDedicated300k           = msgs.M("chip.dedicated300k")
	mChipDedicatedPro            = msgs.M("chip.dedicatedPro")
	mChipDedicatedScale          = msgs.M("chip.dedicatedScale")
	mChipSandboxDetection        = msgs.M("chip.sandboxDetection")
	mChipSandboxToken            = msgs.M("chip.sandboxToken")
	mChipSandboxMode             = msgs.M("chip.sandboxMode")
	mChipSandboxDomain           = msgs.M("chip.sandboxDomain")
	mChipSandboxAddresses        = msgs.M("chip.sandboxAddresses")
	mChipSandbox                 = msgs.M("chip.sandbox")
	mChipSandboxKind             = msgs.M("chip.sandboxKind")
	mChipQuotaDiscovery          = msgs.M("chip.quotaDiscovery")
	mChipQuotaManual             = msgs.M("chip.quotaManual")
	mChipQuotaPlan               = msgs.M("chip.quotaPlan")
	mCommunityCantRun            = msgs.M("community.cantRun")
	mCommunityHTTPName           = msgs.M("community.httpName")
	mCommunityBuiltIn            = msgs.M("community.builtIn")
	mCommunityHTTPSummary        = msgs.M("community.httpSummary")
	mCapabilityBounceMailbox     = msgs.M("capability.bounceMailbox")
	mCapabilityWebhooks          = msgs.M("capability.webhooks")
	mCapabilitySuppressionPull   = msgs.M("capability.suppressionPull")
	mCapabilitySuppressionPush   = msgs.M("capability.suppressionPush")
	mCapabilitySuppressionSync   = msgs.M("capability.suppressionSync")
	mCapabilityDomainAPI         = msgs.M("capability.domainAPI")
	mCapabilityDKIMAPI           = msgs.M("capability.dkimAPI")
	mSummarySES                  = msgs.M("summary.ses")
	mSummaryConnected            = msgs.M("summary.connected")
	mSummaryDomain               = msgs.M("summary.domain")
	mSummarySkipped              = msgs.M("summary.skipped")
	mSummaryDelivered            = msgs.M("summary.delivered")
	mSummarySentTo               = msgs.M("summary.sentTo")
	mRegionLabel                 = msgs.M("region.label")
	mStreamName                  = msgs.M("stream.name")
	mStreamDescription           = msgs.M("stream.description")
	mStreamsNoBulk               = msgs.M("streams.noBulk")
	mStreamsBroadcastOnly        = msgs.M("streams.broadcastOnly")
	mRoleLabel                   = msgs.M("role.label")
	mFailoverPolicy              = msgs.M("failover.policy")
	mFailoverTakesOver           = msgs.M("failover.takesOver")
	mFailoverFallsBack           = msgs.M("failover.fallsBack")
	mFailoverNoFallback          = msgs.M("failover.noFallback")
	mStreamsTrackingOff          = msgs.M("streams.trackingOff")
	mStreamsTrackingOn           = msgs.M("streams.trackingOn")
	mTipsSecond                  = msgs.M("tips.second")
	mTipsSplit                   = msgs.M("tips.split")
	mStreamsDefaults             = msgs.M("streams.defaults")
	mThrottleItem                = msgs.M("throttle.item")
	mErrorSettings               = msgs.M("error.settings")
	mErrorNoHostIdentity         = msgs.M("error.noHostIdentity")
	mErrorRoleRejected           = msgs.M("error.roleRejected")
	mErrorProviderAddressBlocked = msgs.M("error.providerAddressBlocked")
	mErrorKeyRejected            = msgs.M("error.keyRejected")
	mErrorUnavailable            = msgs.M("error.unavailable")
	mErrorMailAddressBlocked     = msgs.M("error.mailAddressBlocked")
	mReasonInvalidURL            = msgs.M("reason.invalidURL")
	mReasonLocalHost             = msgs.M("reason.localHost")
	mReasonPrivateIP             = msgs.M("reason.privateIP")
	mReasonNoDNS                 = msgs.M("reason.noDNS")
	mReasonPrivateDNS            = msgs.M("reason.privateDNS")
	mReasonRefused               = msgs.M("reason.refused")
	mReasonTimeout               = msgs.M("reason.timeout")
	mReasonCertificate           = msgs.M("reason.certificate")
	mReasonProxy                 = msgs.M("reason.proxy")
	mRegistrationAfterReachable  = msgs.M("registration.afterReachable")
	mRegistrationManual          = msgs.M("registration.manual")
	mMailboxSummary              = msgs.M("mailbox.summary")
	mMailboxCantTest             = msgs.M("mailbox.cantTest")
	mProvisionVerify             = msgs.M("provision.verify")
	mProvisionVerified           = msgs.M("provision.verified")
	mProvisionAccount            = msgs.M("provision.account")
	mProvisionAccountRead        = msgs.M("provision.accountRead")
	mProvisionAccountReadAs      = msgs.M("provision.accountReadAs")
	mProvisionHintPublicURL      = msgs.M("provision.hintPublicURL")
	mProvisionNoConfirmation     = msgs.M("provision.noConfirmation")
	mProvisionHintSNS            = msgs.M("provision.hintSNS")
	mProvisionHintIAMUser        = msgs.M("provision.hintIAMUser")
	mProvisionHintRolePolicy     = msgs.M("provision.hintRolePolicy")
	mTrackingConfigSets          = msgs.M("tracking.configSets")
	mProblemSetupNotFound        = msgs.M("problem.setupNotFound")
	mProblemNoAdapter            = msgs.M("problem.noAdapter")
	mProblemCancelled            = msgs.M("problem.cancelled")
	mProblemCancelledShort       = msgs.M("problem.cancelledShort")
	mProblemConnectFirstSteps    = msgs.M("problem.connectFirstSteps")
	mProblemOnlyOpen             = msgs.M("problem.onlyOpen")
	mProblemConnectFirst         = msgs.M("problem.connectFirst")
	mProblemDisconnected         = msgs.M("problem.disconnected")
	mProblemChooseAccess         = msgs.M("problem.chooseAccess")
	mProblemEnterSMTP            = msgs.M("problem.enterSMTP")
	mProblemEnterField           = msgs.M("problem.enterField")
	mProblemWholeNumber          = msgs.M("problem.wholeNumber")
	mProblemPickListed           = msgs.M("problem.pickListed")
	mProblemRoleARN              = msgs.M("problem.roleARN")
	mProblemAccessKeyID          = msgs.M("problem.accessKeyID")
	mProblemSecretKey            = msgs.M("problem.secretKey")
	mProblemSMTPHost             = msgs.M("problem.smtpHost")
	mProblemNoSecret             = msgs.M("problem.noSecret")
	mProblemTunnel               = msgs.M("problem.tunnel")
	mProblemNoPolling            = msgs.M("problem.noPolling")
	mProblemOutgoingFirst        = msgs.M("problem.outgoingFirst")
	mProblemMailboxPassword      = msgs.M("problem.mailboxPassword")
	mProblemNothingToRetry       = msgs.M("problem.nothingToRetry")
	mProblemPickRegion           = msgs.M("problem.pickRegion")
	mProblemSESNoAnswer          = msgs.M("problem.sesNoAnswer")
	mProblemNoProductionAccess   = msgs.M("problem.noProductionAccess")
	mProblemAWSRejected          = msgs.M("problem.awsRejected")
	mProblemDomain               = msgs.M("problem.domain")
	mProblemPickFrom             = msgs.M("problem.pickFrom")
	mProblemPickSender           = msgs.M("problem.pickSender")
	mProblemSeedAddress          = msgs.M("problem.seedAddress")
	mProblemSeedOne              = msgs.M("problem.seedOne")
	mProblemTestTo               = msgs.M("problem.testTo")
	mProblemTestRejected         = msgs.M("problem.testRejected")
	mProblemNoBroadcasts         = msgs.M("problem.noBroadcasts")
	mProblemSplitBroadcastsOnly  = msgs.M("problem.splitBroadcastsOnly")
	mProblemAlreadyCarries       = msgs.M("problem.alreadyCarries")
	mProblemThrottleKeys         = msgs.M("problem.throttleKeys")
	mProblemThrottleValue        = msgs.M("problem.throttleValue")
	mTestSubject                 = msgs.M("test.subject")
	mEmailFooter                 = msgs.M("email.footer")
	mEmailIntro                  = msgs.M("email.intro")
	mEmailWorks                  = msgs.M("email.works")
	mEmailWorksText              = msgs.M("email.worksText")
	mTimelineQueued              = msgs.M("timeline.queued")
	mTimelineQueuedDetail        = msgs.M("timeline.queuedDetail")
	mTimelineAccepted            = msgs.M("timeline.accepted")
	mTimelineWaitingFor          = msgs.M("timeline.waitingFor")
	mTimelineAcceptedDetail      = msgs.M("timeline.acceptedDetail")
	mTimelineSent                = msgs.M("timeline.sent")
	mTimelineWaitingFirst        = msgs.M("timeline.waitingFirst")
	mTimelineSentDetail          = msgs.M("timeline.sentDetail")
	mTimelineSentInferred        = msgs.M("timeline.sentInferred")
	mTimelineSentEvent           = msgs.M("timeline.sentEvent")
	mTimelinePolling             = msgs.M("timeline.polling")
	mTimelineDelivered           = msgs.M("timeline.delivered")
	mTimelineWaitingDelivery     = msgs.M("timeline.waitingDelivery")
	mTimelineDeliveredDetail     = msgs.M("timeline.deliveredDetail")
	mTimelineDeliveredMailpit    = msgs.M("timeline.deliveredMailpit")
	mTimelineSMTPReply           = msgs.M("timeline.smtpReply")
	mTimelineFailed              = msgs.M("timeline.failed")
	mTimelineOpened              = msgs.M("timeline.opened")
	mTimelineWaitingOpen         = msgs.M("timeline.waitingOpen")
	mTimelineOpenPixel           = msgs.M("timeline.openPixel")
	mTimelineOpenEvent           = msgs.M("timeline.openEvent")
	mTestRoundTrip               = msgs.M("test.roundTrip")
	mMailboxOther                = msgs.M("mailbox.other")
	mSourceCSVName               = msgs.M("source.csvName")
	mSourceKeepsConsent          = msgs.M("source.keepsConsent")
	mSourceYouConfirm            = msgs.M("source.youConfirm")
	mListsWeeklyDigest           = msgs.M("lists.weeklyDigest")
	mListsNewsletter             = msgs.M("lists.newsletter")
	mListsProductUpdates         = msgs.M("lists.productUpdates")
	mListsBeta                   = msgs.M("lists.beta")
	mDoneTitle                   = msgs.M("done.title")
	mDoneAWSAccess               = msgs.M("done.awsAccess")
	mDoneAWSAccessText           = msgs.M("done.awsAccessText")
	mDoneEvents                  = msgs.M("done.events")
	mDoneEventsSNS               = msgs.M("done.eventsSNS")
	mDoneEventsUnfinished        = msgs.M("done.eventsUnfinished")
	mDoneConfigSets              = msgs.M("done.configSets")
	mDoneConfigSetsText          = msgs.M("done.configSetsText")
	mDoneSMTP                    = msgs.M("done.smtp")
	mDoneSMTPText                = msgs.M("done.smtpText")
	mDoneBounces                 = msgs.M("done.bounces")
	mDoneBouncesText             = msgs.M("done.bouncesText")
	mDoneNoBounceMailbox         = msgs.M("done.noBounceMailbox")
	mDoneAccess                  = msgs.M("done.access")
	mDoneAccessText              = msgs.M("done.accessText")
	mDoneEventsPolling           = msgs.M("done.eventsPolling")
	mDoneEventsRegistered        = msgs.M("done.eventsRegistered")
	mDoneEventsManual            = msgs.M("done.eventsManual")
	mDoneEventsUnreachable       = msgs.M("done.eventsUnreachable")
	mDoneDomain                  = msgs.M("done.domain")
	mDoneDomainSkipped           = msgs.M("done.domainSkipped")
	mDoneDomainText              = msgs.M("done.domainText")
	mDoneStreams                 = msgs.M("done.streams")
	mDoneStreamsNone             = msgs.M("done.streamsNone")
	mDoneStreamsText             = msgs.M("done.streamsText")
	mDoneStreamsRole             = msgs.M("done.streamsRole")
	mDoneSuppressions            = msgs.M("done.suppressions")
	mDoneSuppressionsImported    = msgs.M("done.suppressionsImported")
	mDoneSuppressionsPushed      = msgs.M("done.suppressionsPushed")
	mDoneContacts                = msgs.M("done.contacts")
	mDoneContactsImport          = msgs.M("done.contactsImport")
	mDoneContactsEmpty           = msgs.M("done.contactsEmpty")
	mDoneContactsKept            = msgs.M("done.contactsKept")
	mStatusLabel                 = msgs.M("status.label")
	mSettingsStreamFallback      = msgs.M("settings.streamFallback")
	mSettingsStreamRules         = msgs.M("settings.streamRules")
	mSettingsNotRouted           = msgs.M("settings.notRouted")
	mIdentityLabel               = msgs.M("identity.label")
	mIdentityVia                 = msgs.M("identity.via")
	mPricingSource               = msgs.M("pricing.source")
)

// say returns a message's English and the message, kept for whoever reads
// it later.
func say(m i18n.Msg, args i18n.Args) (string, i18n.Text) {
	return m.Text(i18n.Default, args), m.Keep(args)
}

// english reports whether l reads what was stored, as it was stored.
func english(l i18n.Lang) bool { return l.Locale == i18n.English || l.Locale == "" }

// textIn is stored English for a reader: the kept message in their
// language, else the English.
func textIn(english string, t i18n.Text, l i18n.Lang) string {
	return providers.TextIn(english, t, l)
}

// textsIn writes stored sentences for a reader: each kept message in their
// language, joined with spaces, or the English when one isn't known.
func textsIn(english string, ts []i18n.Text, l i18n.Lang) string {
	if len(ts) == 0 || english == "" || l.Locale == i18n.English || l.Locale == "" {
		return english
	}
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		s := t.In(l)
		if s == "" {
			return english
		}
		out = append(out, s)
	}
	return strings.Join(out, " ")
}

// callText is the kept message of a failed call, when the adapter worded it
// itself (providers.CallError.Text).
func callText(err error) i18n.Text {
	var ce *providers.CallError
	if errors.As(err, &ce) && ce.Message != "" {
		return ce.Text
	}
	return i18n.Text{}
}

// lowerFirst writes a label's first letter in lower case, to use it inside
// a sentence ("Khóa API" → "khóa API"), unless it starts an acronym
// ("SMTP host").
func lowerFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError || !unicode.IsUpper(r) {
		return s
	}
	if next, _ := utf8.DecodeRuneInString(s[n:]); unicode.IsUpper(next) {
		return s
	}
	return string(unicode.ToLower(r)) + s[n:]
}

// streamName is a stream's name ("Broadcasts").
func streamName(l i18n.Lang, stream string) string {
	return mStreamName.Text(l, i18n.Args{"stream": stream})
}

// yesNo is a select argument.
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
