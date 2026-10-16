package placement

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Problems and notifications a member reads follow their language
// (messages/<locale>.json). Audit entries keep their English.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "placement")

var (
	mSeedRefused          = msgs.M("seed.refused")
	mSeedRefusedIMAP      = msgs.M("seed.refusedIMAP")
	mSeedRefusedMicrosoft = msgs.M("seed.refusedMicrosoft")
	mSeedUnreachable      = msgs.M("seed.unreachable")
	mSeedBlocked          = msgs.M("seed.blocked")
	mSeedDuplicate        = msgs.M("seed.duplicate")
	mSeedUnknownProvider  = msgs.M("seed.unknownProvider")
	mSeedPasswordNeeded   = msgs.M("seed.passwordNeeded")
	mSeedOAuthOnly        = msgs.M("seed.oauthOnly")
	mSeedOAuthNotSetUp    = msgs.M("seed.oauthNotSetUp")
	mSeedOAuthFailed      = msgs.M("seed.oauthFailed")
	mSeedOAuthWrong       = msgs.M("seed.oauthWrongAccount")

	mTestNoSeeds          = msgs.M("test.noSeeds")
	mTestNoConnectedSeeds = msgs.M("test.noConnectedSeeds")
	mTestNoContent        = msgs.M("test.noContent")
	mTestContentEmpty     = msgs.M("test.contentEmpty")
	mTestNotPublished     = msgs.M("test.notPublished")
	mTestNoProviders      = msgs.M("test.noProviders")
	mTestNoRoute          = msgs.M("test.noRoute")
	mTestProviderDown     = msgs.M("test.providerDown")
	mTestWrongStream      = msgs.M("test.wrongStream")
	mTestCompareNeedsTwo  = msgs.M("test.compareNeedsTwo")
	mTestNoSender         = msgs.M("test.noSender")
	mTestWarmupCap        = msgs.M("test.warmupCap")
	mTestNotWaiting       = msgs.M("test.notWaiting")
	mTestCopiesGone       = msgs.M("test.copiesGone")

	mSummaryProvider  = msgs.M("summary.provider")
	mResultInbox      = msgs.M("result.inbox")
	mResultTab        = msgs.M("result.tab")
	mResultSpam       = msgs.M("result.spam")
	mResultNotArrived = msgs.M("result.notArrived")
	mResultNotChecked = msgs.M("result.notChecked")
	mResultFailed     = msgs.M("result.failed")

	// MFinishedTitle and the rest word the notifications (internal/app).
	MFinishedTitle   = msgs.M("notify.finishedTitle")
	MSeedFailedTitle = msgs.M("notify.seedFailedTitle")
	MSeedFailedBody  = msgs.M("notify.seedFailedBody")
	MSeedExpiredBody = msgs.M("notify.seedExpiredBody")
)
