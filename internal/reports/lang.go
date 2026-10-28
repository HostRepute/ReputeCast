package reports

import (
	"embed"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What members read in a campaign report follows their app language
// (i18n.LangFrom): bounce types, unsubscribe reasons, the A/B test, the
// audience and notes in the header, and problems. A shared report reads in
// the language its viewer's browser asks for. The CSV export's columns and
// values stay English: they are data. The words are in
// messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "reports")

var (
	mAudienceAll      = msgs.M("audience.all")
	mAudienceSegments = msgs.M("audience.segments")
	mNoteAB           = msgs.M("note.ab")
	mNoteABWinner     = msgs.M("note.abWinner")
	mReferralLinks    = msgs.M("label.referralLinks")
	mBounceNote       = msgs.M("note.bounceSuppressed")

	mNothingToShare = msgs.M("problem.nothingToShare")
	mTooManyLinks   = msgs.M("problem.tooManyLinks")
	mSharedLimited  = msgs.M("problem.sharedRateLimited")
	mSharedGone     = msgs.M("problem.sharedGone")
	mUnknownBounce  = msgs.M("problem.unknownBounceType")
	mSoftBounces    = msgs.M("problem.softBounces")
	mSenderBlocks   = msgs.M("problem.senderBlocks")

	mInsightBlocked              = msgs.M("insight.blocked")
	mInsightBlockedDetail        = msgs.M("insight.blockedDetail")
	mInsightCheckDelisting       = msgs.M("insight.checkDelisting")
	mInsightMailboxBounces       = msgs.M("insight.mailboxBounces")
	mInsightMailboxBouncesDetail = msgs.M("insight.mailboxBouncesDetail")
	mInsightOpenSNDS             = msgs.M("insight.openSnds")
	mInsightSNDSDetail           = msgs.M("insight.sndsDetail")
	mInsightOpenPostmaster       = msgs.M("insight.openPostmaster")
	mInsightTopLink              = msgs.M("insight.topLink")
	mInsightTopLinkDetail        = msgs.M("insight.topLinkDetail")
	mInsightTopLinkVariant       = msgs.M("insight.topLinkVariant")
	mInsightTopLinkVariantDetail = msgs.M("insight.topLinkVariantDetail")
	mInsightShowLink             = msgs.M("insight.showLink")
	mAllSuppressed               = msgs.M("problem.allSuppressed")

	mFinalAlertPeople  = msgs.M("problem.finalAlertPeople")
	mFinalAlertNotSent = msgs.M("problem.finalAlertNotSent")
	mFinalAlertFinal   = msgs.M("problem.finalAlertFinal")
	mFinalAlertTitle   = msgs.M("notify.finalTitle")
	mFinalAlertBody    = msgs.M("notify.finalBody")
)

// bounceLabel is a bounce type in a reader's language: "Hard · unknown user".
func bounceLabel(l i18n.Lang, key string) string { return msgs.M("bounce."+key).Text(l, nil) }

// reasonLabel is an unsubscribe reason in a reader's language: "Too many emails".
func reasonLabel(l i18n.Lang, key string) string { return msgs.M("reason."+key).Text(l, nil) }

// abLabel is what an A/B test compares, in a reader's language: "subject
// line"; "" for a kind it doesn't know.
func abLabel(l i18n.Lang, kind string) string {
	if m, err := msgs.Lookup("ab." + kind); err == nil {
		return m.Text(l, nil)
	}
	return ""
}
