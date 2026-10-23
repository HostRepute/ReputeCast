package texting

import (
	"context"
	"embed"
	"regexp"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/segments"
)

// What people read about texts follows their app language
// (messages/<locale>.json): the problems with their fixes, the Send SMS
// step's publish checks, the short link's expiry page, the labels the
// screens show (an email subscription's source, a suppression's context, a
// duplicate's name), and the words of the public text message terms page
// (headings, the closing box, the footer); the terms themselves are the
// workspace's own text.
//
// Text kept for later readers keeps its English and reads in each reader's
// language: notifications keep their messages (notify's TitleText…), audit
// entries their reason and summary (audit.KeepReason, audit.KeepText), and
// what ReputeCast stored in English before it had messages (a consent
// change's evidence, a suppression's context and code, a provider's answer,
// the short-link domain's check) is matched back to its message when it's
// read (storedIn); rows from before read the same way.
//
// Stays as it is: what providers, carriers and people wrote (a provider's
// answer to a call, a carrier's rejection reason, the HTTP status words),
// SMS bodies and the workspace's replies, codes, names, logs, the CSV
// export, the MCP agent texts (agents.go), the approval snapshot's
// schedule (approvals keep and compare it as English, scheduleLabel), and
// the Send SMS step's canvas issues (automations.ChannelStep.Validate has
// no language).

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "texting")

// say writes a message in the request's language: API clients that don't
// ask read English.
func say(ctx context.Context, m i18n.Msg, args i18n.Args) string {
	return m.Text(i18n.LangFrom(ctx), args)
}

// The evidence ReputeCast records for a consent change made by text ("Opted
// out by text", "Text message") or in the preference center is stored in
// English, as every consent record is, and read in each reader's language
// (evidenceIn); evidence people or forms gave stays as it was given.
var (
	evOptedOut         = msgs.M("evidence.optedOut")
	evOptedIn          = msgs.M("evidence.optedIn")
	evConfirmed        = msgs.M("evidence.confirmed")
	evTextMessage      = msgs.M("evidence.textMessage")
	evConfirmation     = msgs.M("evidence.confirmation")
	evPreferenceCenter = msgs.M("evidence.preferenceCenter")
	evAccountPage      = msgs.M("evidence.accountPage")
)

// evidenceWords are those messages by their English.
var evidenceWords = map[string]i18n.Msg{}

func init() {
	for _, k := range msgs.Keys() {
		if strings.HasPrefix(k, "evidence.") {
			m := msgs.M(k)
			evidenceWords[m.Source()] = m
		}
	}
}

// evidenceIn writes a consent record's page or note in a language when
// ReputeCast wrote it; any other stays as it was given.
func evidenceIn(l i18n.Lang, s string) string {
	if m, ok := evidenceWords[s]; ok {
		return m.Text(l, nil)
	}
	return s
}

// Problems: what went wrong, and what to do.
var (
	mNameCampaign          = msgs.M("problem.nameCampaign")
	mPickList              = msgs.M("problem.pickList")
	mPickLists             = msgs.M("problem.pickLists")
	mPickSegments          = msgs.M("problem.pickSegments")
	mPickContact           = msgs.M("problem.pickContact")
	mPickMember            = msgs.M("problem.pickMember")
	mPickProvider          = msgs.M("problem.pickProvider")
	mPickCatalog           = msgs.M("problem.pickCatalog")
	mPickSender            = msgs.M("problem.pickSender")
	mPickSenderOfWorkspace = msgs.M("problem.pickSenderOfWorkspace")
	mPickTestNumbers       = msgs.M("problem.pickTestNumbers")
	mCountryCodes          = msgs.M("problem.countryCodes")
	mWentOutDuplicate      = msgs.M("problem.wentOutDuplicate")
	mVersionConflict       = msgs.M("problem.versionConflict")
	mNoAnswer              = msgs.M("problem.noAnswer")
	mNoAnswerRetry         = msgs.M("problem.noAnswerRetry")
	mPhoneOrCountry        = msgs.M("problem.phoneOrCountry")
	mPhoneFormat           = msgs.M("problem.phoneFormat")
	mPhoneFormatUK         = msgs.M("problem.phoneFormatUK")
	mNotPhoneNumber        = msgs.M("problem.notPhoneNumber")
	mNotTextable           = msgs.M("problem.notTextable")
	mPhoneFirst            = msgs.M("problem.phoneFirst")
	mEvidence              = msgs.M("problem.evidence")
	mFutureConsent         = msgs.M("problem.futureConsent")
	mOptedOutByText        = msgs.M("problem.optedOutByText")
	mRemoveReason          = msgs.M("problem.removeReason")
	mFieldRequired         = msgs.M("problem.fieldRequired")
	mCredentialsRejected   = msgs.M("problem.credentialsRejected")
	mCredentialsAgain      = msgs.M("problem.credentialsRejectedToo")
	mNoNumberList          = msgs.M("problem.noNumberList")
	mNotInAccount          = msgs.M("problem.notInAccount")
	mSenderIDFormat        = msgs.M("problem.senderIdFormat")
	mSenderExists          = msgs.M("problem.senderExists")
	mSenderInRoute         = msgs.M("problem.senderInRoute")
	mSenderIDNanp          = msgs.M("problem.senderIdNanp")
	mRejectReason          = msgs.M("problem.rejectReason")
	mFutureDate            = msgs.M("problem.futureDate")
	mNoAdapter             = msgs.M("problem.noAdapter")
	mNoWebhookAPI          = msgs.M("problem.noWebhookApi")
	mWebhooksFailed        = msgs.M("problem.webhooksFailed")
	mProviderInRoute       = msgs.M("problem.providerInRoute")
	mNoProviderSender      = msgs.M("problem.noProviderSender")
	mNoSender              = msgs.M("problem.noSender")
	mTestNumberFirst       = msgs.M("problem.testNumberFirst")
	mTestNumberExists      = msgs.M("problem.testNumberExists")
	mSendSettingsLocked    = msgs.M("problem.sendSettingsLocked")
	mFutureTime            = msgs.M("problem.futureTime")
	mPickDate              = msgs.M("problem.pickDate")
	mInsideWindow          = msgs.M("problem.insideWindow")
	mDateAndTime           = msgs.M("problem.dateAndTime")
	mAlreadyScheduled      = msgs.M("problem.alreadyScheduled")
	mAlreadyWentOut        = msgs.M("problem.alreadyWentOut")
	mReviewChanged         = msgs.M("problem.reviewChanged")
	mBlocked               = msgs.M("problem.blocked")
	mConfirmWarnings       = msgs.M("problem.confirmWarnings")
	mTypeCount             = msgs.M("problem.typeCount")
	mChangedWhileSending   = msgs.M("problem.changedWhileSending")
	mUnscheduleOnly        = msgs.M("problem.unscheduleOnly")
	mTextsWentOut          = msgs.M("problem.textsWentOut")
	mNotSent               = msgs.M("problem.notSent")
	mPauseOnly             = msgs.M("problem.pauseOnly")
	mResumeOnly            = msgs.M("problem.resumeOnly")
	mSendOver              = msgs.M("problem.sendOver")
	mSendAction            = msgs.M("problem.sendAction")
	mTooEarly              = msgs.M("problem.tooEarly")
	mTooLate               = msgs.M("problem.tooLate")
	mEmptyWindow           = msgs.M("problem.emptyWindow")
	mKeywordEmpty          = msgs.M("problem.keywordEmpty")
	mKeywordFormat         = msgs.M("problem.keywordFormat")
	mKeywordDuplicate      = msgs.M("problem.keywordDuplicate")
	mKeywordRequired       = msgs.M("problem.keywordRequired")
	mReplyEmpty            = msgs.M("problem.replyEmpty")
	mConfirmationText      = msgs.M("problem.confirmationText")
	mThreshold             = msgs.M("problem.threshold")
	mNoProviderKeywords    = msgs.M("problem.noProviderKeywords")
	mRequireYes            = msgs.M("problem.requireYes")
	mCountryTwice          = msgs.M("problem.countryTwice")
	mPickFallback          = msgs.M("problem.pickFallback")
	mFallbackOther         = msgs.M("problem.fallbackOther")
	mFallbackRegistered    = msgs.M("problem.fallbackRegistered")
	mOtherRoute            = msgs.M("problem.otherRoute")
	mDomain                = msgs.M("problem.domain")
	mCarriersAsk           = msgs.M("problem.carriersAsk")
)

// The Send SMS step's badges in the publish dialog.
var (
	mCheckQuietWait     = msgs.M("check.quietWait")
	mCheckQuietSkip     = msgs.M("check.quietSkip")
	mCheckQuietSome     = msgs.M("check.quietSome")
	mCheckNoSender      = msgs.M("check.noSender")
	mCheckNoRoute       = msgs.M("check.noRoute")
	mCheckPending       = msgs.M("check.pending")
	mCheckRejected      = msgs.M("check.rejected")
	mCheckNotSupported  = msgs.M("check.notSupported")
	mCheckNotRegistered = msgs.M("check.notRegistered")
	mCheckAllSenders    = msgs.M("check.allSenders")
)

// Notifications, kept as messages so each member reads them in their own
// language, and the audit entries' lines ReputeCast writes itself.
var (
	mNotifyPausedTitle      = msgs.M("notify.pausedTitle")
	mNotifyPausedSummary    = msgs.M("notify.pausedSummary")
	mNotifyBudgetUsedTitle  = msgs.M("notify.budgetUsedTitle")
	mNotifyBudgetUsedBody   = msgs.M("notify.budgetUsedBody")
	mNotifyBudgetAlertTitle = msgs.M("notify.budgetAlertTitle")
	mNotifyBudgetAlertBody  = msgs.M("notify.budgetAlertBody")
	mNotifyCredentialsTitle = msgs.M("notify.credentialsTitle")
	mNotifyCredentialsBody  = msgs.M("notify.credentialsBody")
	mNotifyDownTitle        = msgs.M("notify.downTitle")
	mNotifyDownBody         = msgs.M("notify.downBody")
	mNotifyBackTitle        = msgs.M("notify.backTitle")
	mNotifyBackBody         = msgs.M("notify.backBody")
	mNotifySilentTitle      = msgs.M("notify.silentTitle")
	mNotifySilentBody       = msgs.M("notify.silentBody")
	mNotifyApprovedTitle    = msgs.M("notify.registrationApprovedTitle")
	mNotifyApprovedBody     = msgs.M("notify.registrationApprovedBody")
	mNotifyRejectedTitle    = msgs.M("notify.registrationRejectedTitle")
	mNotifyRejectedBody     = msgs.M("notify.registrationRejectedBody")
	mAuditPausedReason      = msgs.M("audit.pausedReason")
	mAuditPausedSummary     = msgs.M("audit.pausedSummary")
	mAuditBudgetReason      = msgs.M("audit.budgetReason")
	mLabelCopyOf            = msgs.M("label.copyOf")
	mLinkExpired            = msgs.M("link.expired")
	mSourcePreferenceCenter = msgs.M("source.preferenceCenter")
	mSourceOneClick         = msgs.M("source.oneClick")
	mSourceRepermission     = msgs.M("source.repermission")
	mSourceTeam             = msgs.M("source.team")
	mSourceAPI              = msgs.M("source.api")
	mContextConfirmation    = msgs.M("context.confirmation")
	mContextTest            = msgs.M("context.test")
	mAnswerNone             = msgs.M("answer.none")
	mAnswerUnauthorized     = msgs.M("answer.unauthorized")
	mDomainUnavailable      = msgs.M("domain.unavailable")
	mDomainPoints           = msgs.M("domain.points")
	mDomainOther            = msgs.M("domain.other")
	mDomainNoAnswer         = msgs.M("domain.noAnswer")
	mDomainNone             = msgs.M("domain.none")
	mCodeProviderList       = msgs.M("code.providerList")
	contextTexts            = []i18n.Msg{mContextConfirmation, mContextTest}
	answerTexts             = []i18n.Msg{mAnswerNone, mAnswerUnauthorized}
	domainTexts             = []i18n.Msg{mDomainUnavailable, mDomainPoints, mDomainOther, mDomainNoAnswer, mDomainNone}
	providerListTexts       = []i18n.Msg{mCodeProviderList}
	storedPatterns          = map[string]*regexp.Regexp{}
	storedPatternArgs       = map[string][]string{}
	messageArgument         = regexp.MustCompile(`\{(\w+)\}`)
)

func init() {
	for _, group := range [][]i18n.Msg{contextTexts, answerTexts, domainTexts, providerListTexts} {
		for _, m := range group {
			storedPatterns[m.ID()], storedPatternArgs[m.ID()] = storedPattern(m.Source())
		}
	}
}

// storedPattern reads English a message wrote back into its arguments: its
// simple arguments ({target}) match any text, the rest as it is.
func storedPattern(src string) (*regexp.Regexp, []string) {
	var b strings.Builder
	names := []string{}
	b.WriteString("^")
	last := 0
	for _, at := range messageArgument.FindAllStringSubmatchIndex(src, -1) {
		b.WriteString(regexp.QuoteMeta(src[last:at[0]]))
		b.WriteString("(.+?)")
		names = append(names, src[at[2]:at[3]])
		last = at[1]
	}
	b.WriteString(regexp.QuoteMeta(src[last:]))
	b.WriteString("$")
	return regexp.MustCompile(b.String()), names
}

// storedIn writes text ReputeCast stored in English, as one of ms wrote
// it, in a reader's language; anything else (what a provider or a person
// wrote, text from before ReputeCast had words for it) stays as stored.
func storedIn(l i18n.Lang, s string, ms []i18n.Msg) string {
	if s == "" {
		return s
	}
	for _, m := range ms {
		sub := storedPatterns[m.ID()].FindStringSubmatch(s)
		if sub == nil {
			continue
		}
		args := i18n.Args{}
		for i, name := range storedPatternArgs[m.ID()] {
			args[name] = sub[i+1]
		}
		return m.Text(l, args)
	}
	return s
}

// contextOf names what a number was sent when something happened to it, in
// English for what's stored (a suppression's context); contextIn writes
// it in a reader's language.
func contextOf(kind string) string {
	switch kind {
	case "confirmation":
		return mContextConfirmation.Source()
	case "test":
		return mContextTest.Source()
	}
	return ""
}

func contextIn(l i18n.Lang, stored string) string { return storedIn(l, stored, contextTexts) }

// answerIn is a provider's answer to a failed call in a reader's language:
// ReputeCast's own words for it ("no answer", "invalid credentials"); the
// status and the provider's words stay as they came.
func answerIn(l i18n.Lang, stored string) string { return storedIn(l, stored, answerTexts) }

// shortDetail is the short-link domain's last check, as stored: English.
func shortDetail(m i18n.Msg, args i18n.Args) string { return m.Text(i18n.Default, args) }

// shortDetailIn writes the short-link domain's last check in a reader's
// language.
func shortDetailIn(l i18n.Lang, stored string) string { return storedIn(l, stored, domainTexts) }

// providerListCode is the code a number pulled from a provider's opt-out
// list carries without a keyword ("Twilio opt-out list").
func providerListCode(provider string) string {
	return mCodeProviderList.Text(i18n.Default, i18n.Args{"provider": provider})
}

// suppressionCodeIn writes a suppression's code in a reader's language:
// ReputeCast's words for a provider's opt-out list; a provider's code or a
// keyword stays as it came.
func suppressionCodeIn(l i18n.Lang, reason, code string) string {
	if reason != "provider_list" {
		return code
	}
	return storedIn(l, code, providerListTexts)
}

// emailSourceLabel names where an email subscription came from, as the
// profile's consent card does ("Brevo import", "preference center"), in a
// reader's language; the words are internal/audience's for the same card.
func emailSourceLabel(l i18n.Lang, src string) string {
	switch src {
	case "preference_center":
		return mSourcePreferenceCenter.Text(l, nil)
	case "one_click":
		return mSourceOneClick.Text(l, nil)
	case "repermission":
		return mSourceRepermission.Text(l, nil)
	case "team", "manual":
		return mSourceTeam.Text(l, nil)
	case "api":
		return mSourceAPI.Text(l, nil)
	}
	if strings.HasPrefix(src, "import:") || strings.HasPrefix(src, "form:") {
		return segments.SourceLabelIn(l, src)
	}
	return src
}

// localNames reports whether countries are named in a reader's own
// language: an app language other than English (a page in a language
// ReputeCast isn't translated into reads English names, as its labels).
func localNames(l i18n.Lang) bool {
	return l.Locale != "" && l.Locale != i18n.English && l.Locale != i18n.Pseudo && l.Locale.Tag() != language.English
}

// countryNameIn is a country's name in a reader's language, as the app
// names it (Intl.DisplayNames): "Germany", "Đức"; English keeps
// countryName's words.
func countryNameIn(l i18n.Lang, code string) string {
	english := countryName(code)
	if !localNames(l) || english == code {
		return english
	}
	r, err := language.ParseRegion(code)
	if err != nil {
		return english
	}
	if n := display.Regions(l.Locale.Tag()).Name(r); n != "" {
		return n
	}
	return english
}

// countriesIn names countries the way the language writes a list:
// "Ireland, Germany and Spain".
func countriesIn(l i18n.Lang, codes []string) string {
	names := make([]string, 0, len(codes))
	for _, c := range codes {
		names = append(names, countryNameIn(l, c))
	}
	return i18n.AndList(l.Locale, names)
}
