package audience

import (
	"context"
	"embed"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/hostrepute/reputecast/internal/emailaddr"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
)

// What members read from the Audience screens follows their app language
// (i18n.LangFrom): problems, the timeline's labels, source, reason and
// mailbox names, List health's insight and the suppression list's notes.
// Text stored for later readers (a consent event's detail and note, a
// suppression's context) keeps its message beside the English (kept.go).
// Public pages and the subscriber's own emails follow the page's language
// instead (internal/pagelabels): FieldError carries a Code for them. The
// words are in messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "audience")

// say writes one of the package's messages in the request's language.
func say(ctx context.Context, m i18n.Msg, args i18n.Args) string {
	return m.Text(i18n.LangFrom(ctx), args)
}

// keep is English stored for later readers with the message it came from.
type keep struct {
	English string
	Text    i18n.Text
}

// kept writes a message in English to store, and keeps it for readers in
// other languages.
func kept(m i18n.Msg, args i18n.Args) keep {
	return keep{English: m.Text(i18n.Default, args), Text: m.Keep(args)}
}

var (
	mCursorExpired     = msgs.M("problem.cursorExpired")
	mSelectContacts    = msgs.M("problem.selectContacts")
	mBadIDs            = msgs.M("problem.badIds")
	mPickList          = msgs.M("problem.pickList")
	mListGone          = msgs.M("problem.listGone")
	mScopeLists        = msgs.M("problem.scopeLists")
	mEnterTag          = msgs.M("problem.enterTag")
	mSuppressReason    = msgs.M("problem.suppressReason")
	mNoSystemProvider  = msgs.M("problem.noSystemProvider")
	mPickAction        = msgs.M("problem.pickAction")
	mEraseConfirm      = msgs.M("problem.eraseConfirm")
	mSMSConsentFilter  = msgs.M("problem.smsConsentFilter")
	mViewName          = msgs.M("problem.viewName")
	mFilterList        = msgs.M("problem.filterList")
	mJoinedAfter       = msgs.M("problem.joinedAfter")
	mFilterSegment     = msgs.M("problem.filterSegment")
	mNotLanguageNear   = msgs.M("problem.notLanguageClosest")
	mNotLanguage       = msgs.M("problem.notLanguage")
	mLanguageFilter    = msgs.M("problem.languageFilter")
	mNotDeleted        = msgs.M("problem.notDeleted")
	mRestoreExpired    = msgs.M("problem.restoreExpired")
	mRestoreNameTaken  = msgs.M("problem.restoreNameTaken")
	mListName          = msgs.M("problem.listName")
	mListExists        = msgs.M("problem.listExists")
	mFieldKey          = msgs.M("problem.fieldKey")
	mFieldBuiltin      = msgs.M("problem.fieldBuiltin")
	mFieldMoney        = msgs.M("problem.fieldMoney")
	mFieldExists       = msgs.M("problem.fieldExists")
	mPaidNeedsEarn     = msgs.M("problem.paidNeedsEarn")
	mPickPaid          = msgs.M("problem.pickPaid")
	mPolicyActive      = msgs.M("problem.policyActive")
	mPolicyAtRisk      = msgs.M("problem.policyAtRisk")
	mPolicySunset      = msgs.M("problem.policySunset")
	mPolicyStop        = msgs.M("problem.policyStop")
	mPolicyDelete      = msgs.M("problem.policyDelete")
	mFieldNumber       = msgs.M("problem.fieldNumber")
	mFieldDate         = msgs.M("problem.fieldDate")
	mFieldRealDate     = msgs.M("problem.fieldRealDate")
	mFieldBool         = msgs.M("problem.fieldBoolean")
	mFieldSelect       = msgs.M("problem.fieldSelect")
	mFieldTooLong      = msgs.M("problem.fieldTooLong")
	mTimezone          = msgs.M("problem.timezone")
	mNotCustomField    = msgs.M("problem.notCustomField")
	mEmailInvalid      = msgs.M("problem.emailInvalid")
	mPickLists         = msgs.M("problem.pickLists")
	mContactSuppressed = msgs.M("problem.contactSuppressed")
	mContactExists     = msgs.M("problem.contactExists")
	mWorkspaceLists    = msgs.M("problem.workspaceLists")
	mOptinSuppressed   = msgs.M("problem.optinSuppressed")
	mAlreadySuppressed = msgs.M("problem.alreadySuppressed")
	mScore             = msgs.M("problem.score")
	mAmount            = msgs.M("problem.amount")
	mFilterStore       = msgs.M("problem.filterStore")
	mSuppressionSource = msgs.M("problem.suppressionSource")
	mImportGroup       = msgs.M("problem.importGroup")
	mCSVUnreadable     = msgs.M("problem.csvUnreadable")
	mUploadTooMany     = msgs.M("problem.uploadTooMany")
	mUploadEmpty       = msgs.M("problem.uploadEmpty")

	mAnEmail         = msgs.M("label.anEmail")
	mMailboxOther    = msgs.M("label.mailboxOther")
	mImporterOther   = msgs.M("label.importerOther")
	mFailSuppressed  = msgs.M("failure.suppressed")
	mFailRefused     = msgs.M("failure.refused")
	mAlreadyNote     = msgs.M("note.alreadySuppressed")
	mSourceCheckout  = msgs.M("source.checkout")
	mSupImport       = msgs.M("suppressionSource.import")
	mSupExport       = msgs.M("suppressionSource.export")
	mSkipNoAdapter   = msgs.M("skipped.noAdapter")
	mSkipCredentials = msgs.M("skipped.credentials")
	mSkipNoAPI       = msgs.M("skipped.noApi")
	mSkipNoAnswer    = msgs.M("skipped.noAnswer")
	mInsightRepermit = msgs.M("insight.repermission")
	mInsightRepAll   = msgs.M("insight.repermissionAll")
	mInsightAttend   = msgs.M("insight.attention")
	mGrowReferral    = msgs.M("source.referralNamed")
	mGrowPartner     = msgs.M("source.partnerNamed")

	// Stored for later readers (kept texts).
	mNoteAddedByTeam   = msgs.M("note.addedByTeam")
	mNoteConfirmSent   = msgs.M("note.confirmationSent")
	mNoteWaiting       = msgs.M("note.waitingConfirmation")
	mDetailEveryIssue  = msgs.M("detail.frequencyAll")
	mDetailWeekly      = msgs.M("detail.frequencyWeekly")
	mDetailMonthly     = msgs.M("detail.frequencyMonthly")
	mDetailPaused      = msgs.M("detail.paused")
	mDetailPauseLifted = msgs.M("detail.pauseLifted")
	mContextUnsubPage  = msgs.M("context.unsubscribePage")
	mContextOneClick   = msgs.M("context.oneClick")
	mContextCSVUpload  = msgs.M("context.csvUpload")
)

// lookup writes a message chosen by a key known at run time, or english
// when the catalog has none.
func lookup(l i18n.Lang, key, english string) string {
	if m, err := msgs.Lookup(key); err == nil {
		return m.Text(l, nil)
	}
	return english
}

// reasonNameIn is a suppression reason in a reader's language: "Hard bounce".
func reasonNameIn(l i18n.Lang, r string) string { return lookup(l, "reason."+r, reasonName(r)) }

// emailProblem is emailaddr.Check's problem in a reader's language: "no domain".
func emailProblem(l i18n.Lang, problem string) string {
	switch problem {
	case emailaddr.ProblemNoDomain:
		return lookup(l, "email.noDomain", problem)
	case emailaddr.ProblemDoubleDot:
		return lookup(l, "email.doubleDot", problem)
	case emailaddr.ProblemNotAnAddress:
		return lookup(l, "email.notAnAddress", problem)
	case emailaddr.ProblemTooLong:
		return lookup(l, "email.tooLong", problem)
	}
	return problem
}

// EmailProblemIn is emailaddr.Check's problem in a reader's language, for
// other packages that show it (the import preview's invalid rows).
func EmailProblemIn(l i18n.Lang, problem string) string { return emailProblem(l, problem) }

// mailboxLabelIn is MailboxLabel in a reader's language: "Other · Google
// Workspace" is "Khác · Google Workspace"; names stay.
func mailboxLabelIn(l i18n.Lang, provider, detail string) string {
	if provider != emailaddr.Other {
		return MailboxLabel(provider, detail)
	}
	name := mMailboxOther.Text(l, nil)
	if detail != "" {
		return name + " · " + detail
	}
	return name
}

// languageNameIn is a content language's name in a reader's language, as
// Intl.DisplayNames names it on the web: "Spanish", "Tiếng Tây Ban Nha";
// "" when the code isn't a language.
func languageNameIn(l i18n.Lang, code string) string {
	english := languageName(code)
	if english == "" || l.Locale == "" || l.Locale == i18n.English || l.Locale == i18n.Pseudo {
		return english
	}
	tag, err := language.Parse(languages.Base(code))
	if err != nil {
		return english
	}
	if n := display.Languages(l.Locale.Tag()).Name(tag); n != "" {
		return n
	}
	return english
}

// CountryNameIn is a country's name in a reader's language, as
// Intl.DisplayNames names it on the web ("United States", "Hoa Kỳ"); the
// code itself when it isn't a country.
func CountryNameIn(l i18n.Lang, code string) string {
	english := CountryName(code)
	if l.Locale == "" || l.Locale == i18n.English || l.Locale == i18n.Pseudo || english == code {
		return english
	}
	r, err := language.ParseRegion(strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		return english
	}
	if n := display.Regions(l.Locale.Tag()).Name(r); n != "" {
		return n
	}
	return english
}

// ReferralSource names a contact who joined through a referral, in a
// reader's language: "Referral · Sam Rivera". name is the referrer's.
func ReferralSource(l i18n.Lang, name string) string {
	return mGrowReferral.Text(l, i18n.Args{"name": name})
}

// PartnerSource names a contact who joined on a partner's page, in a
// reader's language: "Partner · Ops Weekly".
func PartnerSource(l i18n.Lang, name string) string {
	return mGrowPartner.Text(l, i18n.Args{"name": name})
}
