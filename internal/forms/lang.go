package forms

import (
	"context"
	"embed"
	"encoding/json"
	"maps"

	"golang.org/x/text/language"
	names "golang.org/x/text/language/display"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/pagelabels"
)

// Two languages meet in this package. What a member reads in the app
// follows their app language (i18n.LangFrom): the problems, the branding's
// contrast notes, and the custom domain's and Turnstile's findings, which
// the settings store in English beside the messages they came from
// (public_page_settings.texts). What a visitor reads follows the page's
// language: the built-in labels (internal/pagelabels), the publisher's own
// text, and ReputeCast's defaults for that text while the publisher keeps
// them (the Subscribe page's footnote, the unsubscribe page's offer). The
// words are in messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "forms")

var (
	mUploadChoose        = msgs.M("problem.uploadChoose")
	mUploadInterrupted   = msgs.M("problem.uploadInterrupted")
	mArchiveList         = msgs.M("problem.archiveList")
	mArchiveFile         = msgs.M("problem.archiveFile")
	mNotYourLanguage     = msgs.M("problem.notYourLanguage")
	mPagesWrittenIn      = msgs.M("problem.pagesWrittenIn")
	mNoPageString        = msgs.M("problem.noPageString")
	mUnder600            = msgs.M("problem.under600")
	mPrivacyTranslation  = msgs.M("problem.privacyTranslation")
	mLabelsShipped       = msgs.M("problem.labelsShipped")
	mNotLabel            = msgs.M("problem.notLabel")
	mLabelNoPlaceholders = msgs.M("problem.labelNoPlaceholders")
	mLabelPlaceholder    = msgs.M("problem.labelPlaceholder")
	mLabelUnclosed       = msgs.M("problem.labelUnclosed")
	mNoLabelField        = msgs.M("problem.noLabelField")
	mFieldLabel80        = msgs.M("problem.fieldLabel80")
	mNameButton          = msgs.M("problem.nameButton")
	mConsentUpToDate     = msgs.M("problem.consentUpToDate")
	mPickFixed           = msgs.M("problem.pickFixed")
	mPickMode            = msgs.M("problem.pickMode")
	mDomainName          = msgs.M("problem.domainName")
	mDomainOwn           = msgs.M("problem.domainOwn")
	mDomainTaken         = msgs.M("problem.domainTaken")
	mNameForm            = msgs.M("problem.nameForm")
	mNotContactField     = msgs.M("problem.notContactField")
	mNotFieldKey         = msgs.M("problem.notFieldKey")
	mOfferList           = msgs.M("problem.offerList")
	mRedirectURL         = msgs.M("problem.redirectURL")
	mWorkspaceLists      = msgs.M("problem.workspaceLists")
	mNotDomainQuoted     = msgs.M("problem.notDomainQuoted")
	mNotDomain           = msgs.M("problem.notDomain")
	mNotDomainShort      = msgs.M("problem.notDomainShort")
	mPublisherName       = msgs.M("problem.publisherName")
	mFontFamily          = msgs.M("problem.fontFamily")
	mFontURL             = msgs.M("problem.fontURL")
	mPrivacyURL          = msgs.M("problem.privacyURL")
	mPickForm            = msgs.M("problem.pickForm")
	mListLine300         = msgs.M("problem.listLine300")
	mFewerOffer          = msgs.M("problem.fewerOffer")
	mPreviewEmail        = msgs.M("problem.previewEmail")
	mLogoEmpty           = msgs.M("problem.logoEmpty")
	mLogoTooBig          = msgs.M("problem.logoTooBig")
	mLogoScripts         = msgs.M("problem.logoScripts")
	mLogoType            = msgs.M("problem.logoType")
	mSmsNeedsPhone       = msgs.M("problem.smsNeedsPhone")
	mTurnstileSiteKey    = msgs.M("problem.turnstileSiteKey")
	mTurnstileSecret     = msgs.M("problem.turnstileSecret")
	mTurnstileUnreadable = msgs.M("problem.turnstileUnreadable")
	mConfirmedAlready    = msgs.M("problem.confirmedAlready")

	mNotColour   = msgs.M("palette.notColour")
	mTooDark     = msgs.M("palette.tooDark")
	mWhiteText   = msgs.M("palette.whiteText")
	mDarkerLinks = msgs.M("palette.darkerLinks")

	mDomainUnavailable = msgs.M("domain.unavailable")
	mDomainCNAME       = msgs.M("domain.cname")
	mDomainTXT         = msgs.M("domain.txt")
	mDomainResolves    = msgs.M("domain.resolves")
	mDomainWrongCNAME  = msgs.M("domain.wrongCname")
	mDomainNoAnswer    = msgs.M("domain.noAnswer")
	mDomainNoCNAME     = msgs.M("domain.noCname")
	mDomainLastFailed  = msgs.M("domain.lastFailed")

	mTurnstileSiteRejected   = msgs.M("turnstile.siteRejected")
	mTurnstileSiteUnknown    = msgs.M("turnstile.siteUnknown")
	mTurnstileWidget         = msgs.M("turnstile.widget")
	mTurnstileReported       = msgs.M("turnstile.reported")
	mTurnstileSecretRejected = msgs.M("turnstile.secretRejected")

	mDefaultFootnote     = msgs.M("default.footnote")
	mDefaultFewerTitle   = msgs.M("default.fewerTitle")
	mDefaultFewerText    = msgs.M("default.fewerText")
	mDefaultFewerButton  = msgs.M("default.fewerButton")
	mDefaultFormFootnote = msgs.M("default.formFootnote")
	mDefaultConsent      = msgs.M("default.consent")
	mDefaultBuiltin      = msgs.M("default.builtinHeading")
	mAddressInvalid      = msgs.M("default.addressInvalid")
)

// say is a message in the request's language, for problems.
func say(ctx context.Context, m i18n.Msg, args i18n.Args) string {
	return m.Text(i18n.LangFrom(ctx), args)
}

// english is a message as the settings store it for later readers.
func english(t i18n.Text) string { return textIn(i18n.Default, t) }

// languageNameIn is a language's name in a reader's language, as the web
// names it (Intl.DisplayNames): "Spanish", "Tiếng Tây Ban Nha"; raw when
// the code isn't a language.
func languageNameIn(l i18n.Lang, code, raw string) string {
	english := languages.NameOf(code)
	if english == "" {
		return raw
	}
	if l.Locale == i18n.English || l.Locale == i18n.Pseudo {
		return english
	}
	tag, err := language.Parse(languages.Base(code))
	if err != nil {
		return english
	}
	if n := names.Languages(l.Locale.Tag()).Name(tag); n != "" {
		return n
	}
	return english
}

// textIn writes a kept message for a reader. An argument that is itself a
// kept message (the domain check's finding inside "Last check failed: …")
// is written in the reader's language too; "" when the message is gone.
func textIn(l i18n.Lang, t i18n.Text) string {
	var args i18n.Args
	for k, v := range t.Args {
		nested, ok := keptArg(v)
		if !ok {
			continue
		}
		s := textIn(l, nested)
		if s == "" {
			return ""
		}
		if args == nil {
			args = maps.Clone(t.Args)
		}
		args[k] = s
	}
	if args == nil {
		args = t.Args
	}
	return i18n.Text{ID: t.ID, Args: args}.In(l)
}

// keptArg reads a message argument that is a kept message: an i18n.Text
// when it's written, a JSON object with an id once it went through JSON.
func keptArg(v any) (i18n.Text, bool) {
	switch x := v.(type) {
	case i18n.Text:
		return x, !x.IsZero()
	case map[string]any:
		id, _ := x["id"].(string)
		if id == "" {
			return i18n.Text{}, false
		}
		args, _ := x["args"].(map[string]any)
		return i18n.Text{ID: id, Args: args}, true
	}
	return i18n.Text{}, false
}

// keptTexts is a texts column holding one kept message under column; nil
// for a zero message, which removes the column's message.
func keptTexts(column string, t i18n.Text) json.RawMessage {
	if t.IsZero() {
		return nil
	}
	b, err := json.Marshal(map[string]i18n.Text{column: t})
	if err != nil {
		return nil
	}
	return b
}

// keptIn writes a stored English text for a reader: the message kept for
// its column in their language, or the English when the reader reads
// English, the row kept no message or its catalog no longer has it.
func keptIn(l i18n.Lang, texts json.RawMessage, column, stored string) string {
	if stored == "" || len(texts) == 0 || l.Locale == i18n.English {
		return stored
	}
	var kept map[string]i18n.Text
	if json.Unmarshal(texts, &kept) != nil {
		return stored
	}
	t, ok := kept[column]
	if !ok || t.IsZero() {
		return stored
	}
	if s := textIn(l, t); s != "" {
		return s
	}
	return stored
}

// pageDefault is a text of the pages that has a default, on a page in a
// language: the default in that language while the publisher keeps
// ReputeCast's English default and ReputeCast ships the language's words,
// else the text as stored.
func pageDefault(code string, m i18n.Msg, stored string) string {
	if stored != m.Text(i18n.Default, nil) || !pagelabels.Shipped(code) {
		return stored
	}
	return m.Text(pagelabels.Lang(code), nil)
}
