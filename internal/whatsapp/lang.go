package whatsapp

import (
	"context"
	"embed"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What a person reads about WhatsApp follows their app language
// (messages/<locale>.json): the problems with their fixes, the composer's
// schedule and field labels, a template's language, a rejected template's
// reason in ReputeCast's words, the consent changes a keyword made on the
// contact's timeline, and the notifications, which keep their messages so
// every member reads them in their own language (notify's TitleText…).
//
// Stays as it is: Meta's own words (its errors, a rejection reason it
// gave), the webhook's field names in the Events tab, codes (statuses,
// reasons, template and language codes), names, the automation step's
// issues (automations.ChannelStep.Validate has no language), the webhook
// event descriptions (developer documentation), logs and the CSV export.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "whatsapp")

var (
	mTokenRejected           = msgs.M("problem.tokenRejected")
	mUnknownID               = msgs.M("problem.unknownId")
	mMetaUnavailable         = msgs.M("problem.metaUnavailable")
	mMetaAnswered            = msgs.M("problem.metaAnswered")
	mAlreadyConnectedSwitch  = msgs.M("problem.alreadyConnectedDisconnect")
	mAlreadyConnected        = msgs.M("problem.alreadyConnected")
	mBusinessIDNumber        = msgs.M("problem.businessIdNumber")
	mPhoneIDNumber           = msgs.M("problem.phoneIdNumber")
	mConnectFirst            = msgs.M("problem.connectFirst")
	mReplaceWhat             = msgs.M("problem.replaceWhat")
	mPhoneFormat             = msgs.M("problem.phoneFormat")
	mPickApprovedTemplate    = msgs.M("problem.pickApprovedTemplate")
	mNoApprovedTemplate      = msgs.M("problem.noApprovedTemplate")
	mSyncTokenRejected       = msgs.M("problem.syncTokenRejected")
	mPickList                = msgs.M("problem.pickList")
	mPickLists               = msgs.M("problem.pickLists")
	mPickSegments            = msgs.M("problem.pickSegments")
	mOnlyApproved            = msgs.M("problem.onlyApproved")
	mNameCampaign            = msgs.M("problem.nameCampaign")
	mNotEditable             = msgs.M("problem.notEditable")
	mStaleSaved              = msgs.M("problem.staleSaved")
	mFutureTime              = msgs.M("problem.futureTime")
	mDayAndTime              = msgs.M("problem.dayAndTime")
	mInsideWindow            = msgs.M("problem.insideWindow")
	mUnknownField            = msgs.M("problem.unknownField")
	mPickTemplateFirst       = msgs.M("problem.pickTemplateFirst")
	mTemplateNotApproved     = msgs.M("problem.templateNotApproved")
	mPhoneFirst              = msgs.M("problem.phoneFirst")
	mEvidence                = msgs.M("problem.evidence")
	mFutureConsent           = msgs.M("problem.futureConsent")
	mOptedOutStart           = msgs.M("problem.optedOutStart")
	mTokenReplaceFirst       = msgs.M("problem.tokenReplaceFirst")
	mWriteReply              = msgs.M("problem.writeReply")
	mWindowClosed            = msgs.M("problem.windowClosed")
	mWindowClosedAt          = msgs.M("problem.windowClosedAt")
	mOptedOut                = msgs.M("problem.optedOut")
	mAlreadyStatus           = msgs.M("problem.alreadyStatus")
	mStaleReviewed           = msgs.M("problem.staleReviewed")
	mBlocking                = msgs.M("problem.blocking")
	mUnscheduleOnly          = msgs.M("problem.unscheduleOnly")
	mCancelOnly              = msgs.M("problem.cancelOnly")
	mRatesDate               = msgs.M("problem.ratesDate")
	mRatesDuplicate          = msgs.M("problem.ratesDuplicate")
	mRatesDefault            = msgs.M("problem.ratesDefault")
	mScheduleNow             = msgs.M("label.now")
	mScheduleAt              = msgs.M("label.at")
	mScheduleLocal           = msgs.M("label.local")
	mCopyOf                  = msgs.M("label.copyOf")
	mNotifyTokenTitle        = msgs.M("notify.tokenTitle")
	mNotifyTokenSummary      = msgs.M("notify.tokenSummary")
	mNotifyTokenSummaryHeld  = msgs.M("notify.tokenSummaryHeld")
	mNotifyTokenBody         = msgs.M("notify.tokenBody")
	mNotifyQualityTitle      = msgs.M("notify.qualityTitle")
	mNotifyQualitySummary    = msgs.M("notify.qualitySummary")
	mNotifyQualityLimit      = msgs.M("notify.qualitySummaryLimit")
	mNotifyQualityBody       = msgs.M("notify.qualityBody")
	mNotifyRejectedTitle     = msgs.M("notify.rejectedTitle")
	mNotifyRejectedSummary   = msgs.M("notify.rejectedSummary")
	mNotifyRejectedMeta      = msgs.M("notify.rejectedSummaryMeta")
	mNotifyRejectedCode      = msgs.M("notify.rejectedSummaryCode")
	mNotifyRejectedBody      = msgs.M("notify.rejectedBody")
	mStoredPage              = msgs.M("stored.page")
	mStoredOptedOut          = msgs.M("stored.optedOut")
	mStoredOptedIn           = msgs.M("stored.optedIn")
	fieldKeys                = []string{"first_name", "last_name", "full_name", "email", "phone"}
	rejectionCodes           = []string{"INVALID_FORMAT", "ABUSIVE_CONTENT", "INCORRECT_CATEGORY", "TAG_CONTENT_MISMATCH", "SCAM", "PROMOTIONAL"}
	storedTexts              = map[string]i18n.Msg{}
	englishForRejectionCodes = map[string]string{}
)

func init() {
	// The texts written into consent records are the catalog's English;
	// readers get them in their own language (storedIn).
	for _, m := range []i18n.Msg{mStoredPage, mStoredOptedOut, mStoredOptedIn} {
		storedTexts[m.Source()] = m
	}
	for _, code := range rejectionCodes {
		englishForRejectionCodes[code] = rejectionMsg(code).Source()
	}
}

// langOf is how the request reads: English for API clients that don't ask.
func langOf(ctx context.Context) i18n.Lang { return i18n.LangFrom(ctx) }

// storedIn is text ReputeCast wrote into a record in English (a consent
// change's page and note), in a reader's language; anything else (what a
// person or Meta wrote) as it was stored.
func storedIn(l i18n.Lang, s string) string {
	if m, ok := storedTexts[s]; ok {
		return m.Text(l, nil)
	}
	return s
}

// rejectionMsg is a Meta rejection code in ReputeCast's words.
func rejectionMsg(code string) i18n.Msg {
	return msgs.M("reason." + strings.ToLower(code))
}

// reasonWords is a rejection code in words, in English for what's stored:
// "" for a code ReputeCast has no words for.
func reasonWords(code string) string { return englishForRejectionCodes[strings.ToUpper(code)] }

// reasonIn is a rejected template's reason in a reader's language: Meta's
// own words as Meta gave them, else ReputeCast's words for its code.
func reasonIn(l i18n.Lang, code, stored string) string {
	words, ok := englishForRejectionCodes[strings.ToUpper(code)]
	if ok && (stored == "" || stored == words) {
		return rejectionMsg(code).Text(l, nil)
	}
	return stored
}

// fieldLabel is a contact field a template variable can take: "First name".
func fieldLabel(l i18n.Lang, key string) string { return msgs.M("label.field."+key).Text(l, nil) }

// LanguageLabel names a template's language in English: "English (US)".
func LanguageLabel(code string) string { return LanguageLabelIn(i18n.Default, code) }

// LanguageLabelIn names a template's language in a reader's language; a
// code ReputeCast has no name for stays the code.
func LanguageLabelIn(l i18n.Lang, code string) string {
	if m, err := msgs.Lookup("language." + code); err == nil {
		return m.Text(l, nil)
	}
	return code
}

// dayMonth and clock write a moment as the reader's region does: 21 Oct,
// 09:00.
func dayMonth(l i18n.Lang, t time.Time) string { return l.Region.DayMonth(t) }
func clock(l i18n.Lang, t time.Time) string    { return l.Region.Time(t, "") }
