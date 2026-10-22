package push

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Push's own words follow their reader's language, from
// messages/<locale>.json: what the app shows a member (problems, labels,
// the default prompt a member creates) in the request's language
// (i18n.LangFrom), the worker-missing notification in each member's
// (kept with notify's TitleText…), and what a site's readers see (the
// account page, the Home Screen steps, the preference center, an expired
// link) in the reader's. Audit entries, logs, the push services' answers
// and the Events tab's raw column keep their English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "push")

var (
	mProblemAlreadyOn      = msgs.M("problem.alreadyOn")
	mProblemContact        = msgs.M("problem.contact")
	mProblemContactNoKeys  = msgs.M("problem.contactNoKeys")
	mProblemNoKeysToRotate = msgs.M("problem.noKeysToRotate")
	mProblemRotateConfirm  = msgs.M("problem.rotateConfirm")

	mProblemSiteExists   = msgs.M("problem.siteExists")
	mProblemSiteAddress  = msgs.M("problem.siteAddress")
	mProblemSiteNoKeys   = msgs.M("problem.siteNoKeys")
	mProblemWorkerPath   = msgs.M("problem.workerPath")
	mProblemPickImage    = msgs.M("problem.pickImage")
	mProblemPickIDs      = msgs.M("problem.pickIds")
	mProblemPickSites    = msgs.M("problem.pickSites")
	mProblemCampaignName = msgs.M("problem.campaignName")
	mProblemLink         = msgs.M("problem.link")
	mProblemNotEditable  = msgs.M("problem.notEditable")
	mProblemStaleVersion = msgs.M("problem.staleVersion")
	mProblemButton       = msgs.M("problem.button")

	mProblemScheduleLocked = msgs.M("problem.scheduleLocked")
	mProblemFutureTime     = msgs.M("problem.futureTime")
	mProblemAlreadySent    = msgs.M("problem.alreadySent")
	mProblemChanged        = msgs.M("problem.changedSinceReview")
	mProblemBlocked        = msgs.M("problem.blockingChecks")
	mProblemUnschedule     = msgs.M("problem.unschedule")
	mProblemCancel         = msgs.M("problem.cancel")

	mProblemTestNoKeys       = msgs.M("problem.testNoKeys")
	mProblemTestSubscription = msgs.M("problem.testSubscription")
	mProblemTestTitle        = msgs.M("problem.testTitle")

	mProblemPromptName      = msgs.M("problem.promptName")
	mProblemPromptSite      = msgs.M("problem.promptSite")
	mProblemPromptPushOff   = msgs.M("problem.promptPushOff")
	mProblemPromptNoSite    = msgs.M("problem.promptNotPushSite")
	mProblemPromptUnchanged = msgs.M("problem.promptUnchanged")
	mProblemPromptDraft     = msgs.M("problem.promptDraft")
	mProblemPromptStyle     = msgs.M("problem.promptStyle")
	mProblemPromptTitle     = msgs.M("problem.promptTitle")
	mProblemPromptTitleLen  = msgs.M("problem.promptTitleLength")
	mProblemPromptMsgLen    = msgs.M("problem.promptMessageLength")
	mProblemPromptButtons   = msgs.M("problem.promptButtons")
	mProblemPromptWhen      = msgs.M("problem.promptWhen")
	mProblemPromptScroll    = msgs.M("problem.promptScroll")
	mProblemPromptPath      = msgs.M("problem.promptPath")
	mProblemPromptPages     = msgs.M("problem.promptPages")
	mProblemPromptAskAgain  = msgs.M("problem.promptAskAgain")

	mLabelPublicationSite = msgs.M("label.publicationSite")
	mLabelCopyOf          = msgs.M("label.copyOf")

	mPromptTitle   = msgs.M("prompt.title")
	mPromptMessage = msgs.M("prompt.message")
	mPromptAllow   = msgs.M("prompt.allow")
	mPromptNotNow  = msgs.M("prompt.notNow")

	mNotifyWorkerTitle   = msgs.M("notify.workerMissing.title")
	mNotifyWorkerSummary = msgs.M("notify.workerMissing.summary")
	mNotifyWorkerBody    = msgs.M("notify.workerMissing.body")

	mPublicLinkGone = msgs.M("public.linkGone")
)
