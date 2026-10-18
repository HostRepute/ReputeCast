package verification

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Problems and notifications a member reads follow their language
// (messages/<locale>.json). Audit entries keep their English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "verification")

var (
	mUnknownService   = msgs.M("problem.unknownService")
	mAlreadyConnected = msgs.M("problem.alreadyConnected")
	mNotConnected     = msgs.M("problem.notConnected")
	mKeyRejected      = msgs.M("problem.keyRejected")
	mNoCreditsKey     = msgs.M("problem.noCreditsKey")
	mUnreachable      = msgs.M("problem.unreachable")
	mNoService        = msgs.M("problem.noService")
	mNoServiceAsk     = msgs.M("problem.noServiceAsk")
	mNothingToCheck   = msgs.M("problem.nothingToCheck")
	mJobFinished      = msgs.M("problem.jobFinished")
	mJobNotStopped    = msgs.M("problem.jobNotStopped")
	mFormsEdit        = msgs.M("problem.formsEdit")
	mFormsEditPlain   = msgs.M("problem.formsEditPlain")
	mNoFormsService   = msgs.M("problem.noFormsService")
	mContactErased    = msgs.M("problem.contactErased")

	mTrayRunning       = msgs.M("tray.running")
	mTrayDone          = msgs.M("tray.done")
	mTrayStopped       = msgs.M("tray.stopped")
	mTrayCancelled     = msgs.M("tray.cancelled")
	mTrayDoneDetail    = msgs.M("tray.doneDetail")
	mTrayStoppedDetail = msgs.M("tray.stoppedDetail")
	mTrayPaused        = msgs.M("tray.paused")
	mTrayEta           = msgs.M("tray.eta")
	mTrayEtaSoon       = msgs.M("tray.etaSoon")

	// MJobFinishedTitle and the rest word the notifications (notify.go in
	// internal/app formats them per recipient).
	MJobFinishedTitle   = msgs.M("notify.jobFinishedTitle")
	MJobFinishedSummary = msgs.M("notify.jobFinishedSummary")
	MJobStoppedTitle    = msgs.M("notify.jobStoppedTitle")
	MJobStoppedCredits  = msgs.M("notify.jobStoppedCredits")
	MJobStoppedDown     = msgs.M("notify.jobStoppedDown")
	MJobStoppedKey      = msgs.M("notify.jobStoppedKey")
	MCreditsLowTitle    = msgs.M("notify.creditsLowTitle")
	MCreditsLowSummary  = msgs.M("notify.creditsLowSummary")
	MCreditsLowNeeded   = msgs.M("notify.creditsLowNeeded")
)
