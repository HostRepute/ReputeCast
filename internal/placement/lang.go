package placement

import (
	"embed"
	"strings"

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
	mSeedHumansOnly       = msgs.M("seed.humansOnly")
	mOAuthState           = msgs.M("oauth.stateUnreadable")
	mOAuthApp             = msgs.M("oauth.appNotSetUp")
	mOAuthNoEmail         = msgs.M("oauth.noEmail")
	mCopyProviderRemoved  = msgs.M("copy.providerRemoved")

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

// credentials name what each mailbox provider calls the password a seed
// signs in with (seedbox.Provider.Password), as a sentence writes it.
var credentials = map[string]i18n.Msg{
	"App password":          msgs.M("credential.appPassword"),
	"App-specific password": msgs.M("credential.appSpecific"),
	"Password":              msgs.M("credential.password"),
}

// credentialIn is a provider's password, named in a reader's language in
// the middle of a sentence: "app password", "mật khẩu ứng dụng".
func credentialIn(l i18n.Lang, label string) string {
	if m, ok := credentials[label]; ok {
		return m.Text(l, nil)
	}
	return strings.ToLower(label)
}

// errProviderRemoved is a copy's error when its provider was removed
// before it went out; the copy view writes it in the reader's language.
const errProviderRemoved = "the provider was removed"

// copyError is why a copy failed, for a reader: the provider's own words
// stay as they are.
func copyError(l i18n.Lang, e string) string {
	if e == errProviderRemoved {
		return mCopyProviderRemoved.Text(l, nil)
	}
	return e
}
