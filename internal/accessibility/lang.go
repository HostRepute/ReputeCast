package accessibility

import (
	"context"
	"embed"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
)

// The words people read (issues' titles, details and tags, the gallery's
// and the policy's summaries, the publish dialog's badges, fixes and their
// problems) are messages in messages/<locale>.json. Check writes a report
// in English, as Send Readiness, API clients and agents read it, and keeps
// the messages with each issue (AccessibilityIssue.Texts); a report reads
// in a member's language through AccessibilityReport.In, and a kept text
// through TextIn.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "accessibility")

var (
	mNameImage       = msgs.M("name.image")
	mNameLinkedImage = msgs.M("name.linkedImage")
	mNameNoText      = msgs.M("name.noText")

	mAltTitle        = msgs.M("alt.title")
	mAltDetail       = msgs.M("alt.detail")
	mAltTag          = msgs.M("alt.tag")
	mLinkedAltTitle  = msgs.M("linkedAlt.title")
	mLinkedAltDetail = msgs.M("linkedAlt.detail")
	mNoTextTitle     = msgs.M("noText.title")
	mNoTextDetail    = msgs.M("noText.detail")
	mNoTextLinkTag   = msgs.M("noText.linkTag")
	mNoTextButtonTag = msgs.M("noText.buttonTag")

	mBareTitle       = msgs.M("bare.title")
	mBareTag         = msgs.M("bare.tag")
	mVagueTitle      = msgs.M("vague.title")
	mVagueTag        = msgs.M("vague.tag")
	mLinkTextAlso    = msgs.M("linkText.also")
	mLinkTextSplit   = msgs.M("linkText.split")
	mLinkTextWrite   = msgs.M("linkText.write")
	mContrastTitle   = msgs.M("contrast.title")
	mContrastDetail  = msgs.M("contrast.detail")
	mContrastTag     = msgs.M("contrast.tag")
	mSizeTitle       = msgs.M("size.title")
	mSizeTitleIn     = msgs.M("size.titleIn")
	mSizeTag         = msgs.M("size.tag")
	mSizeSubject     = msgs.M("size.subject")
	mSizeSubjectIn   = msgs.M("size.subjectIn")
	mHeadingTitle    = msgs.M("heading.title")
	mHeadingDetail   = msgs.M("heading.detail")
	mHeadingTag      = msgs.M("heading.tag")
	mHeadingSubject  = msgs.M("heading.subject")
	mHeadingSubject1 = msgs.M("heading.subjectOne")

	mPhraseSerious     = msgs.M("phrase.serious")
	mPhraseSeriousBare = msgs.M("phrase.seriousBare")
	mPhraseBareURLs    = msgs.M("phrase.bareUrls")
	mPhraseVagueLinks  = msgs.M("phrase.vagueLinks")
	mPhraseSkipped     = msgs.M("phrase.skippedHeading")
	mPhraseTextSize    = msgs.M("phrase.textSize")
	mPhraseTextSizeIn  = msgs.M("phrase.textSizeIn")

	mSummaryNone   = msgs.M("summary.none")
	mSummaryIssues = msgs.M("summary.issues")

	mGallerySerious     = msgs.M("gallery.serious")
	mGallerySeriousAlso = msgs.M("gallery.seriousAlso")
	mGalleryModerate    = msgs.M("gallery.moderate")
	mGalleryBare        = msgs.M("gallery.bare")
	mGalleryVague       = msgs.M("gallery.vague")
	mGalleryHeading     = msgs.M("gallery.heading")
	mGallerySize        = msgs.M("gallery.size")

	mDraftBadge       = msgs.M("draft.badge")
	mDraftNone        = msgs.M("draft.none")
	mDraftFixIn       = msgs.M("draft.fixIn")
	mDraftBlocker     = msgs.M("draft.blocker")
	mDraftBlockerLive = msgs.M("draft.blockerLive")

	mFindingsLanguages     = msgs.M("findings.languages")
	mFindingsBlocks        = msgs.M("findings.blocks")
	mFindingsOkTitle       = msgs.M("findings.okTitle")
	mFindingsOkDetail      = msgs.M("findings.okDetail")
	mFindingsLangDirTitle  = msgs.M("findings.langDirTitle")
	mFindingsLangDirDetail = msgs.M("findings.langDirDetail")
	mFindingsTablesTitle   = msgs.M("findings.tablesTitle")
	mFindingsTablesNone    = msgs.M("findings.tablesNone")
	mFindingsTablesAdded   = msgs.M("findings.tablesAdded")

	mProblemOnSerious   = msgs.M("problem.onSerious")
	mProblemMinText     = msgs.M("problem.minText")
	mProblemUncheckable = msgs.M("problem.uncheckable")

	mReadinessNoContent = msgs.M("readiness.noContent")
	mReadinessWaits     = msgs.M("readiness.waits")
	mReadinessNoCompile = msgs.M("readiness.noCompile")

	mFixDecorativeLabel   = msgs.M("fix.decorativeLabel")
	mFixDecorativeApplied = msgs.M("fix.decorativeApplied")
	mFixDecorativeSummary = msgs.M("fix.decorativeSummary")
	mFixContrastLabel     = msgs.M("fix.contrastLabel")
	mFixColorApplied      = msgs.M("fix.colorApplied")
	mFixColorAppliedLabel = msgs.M("fix.colorAppliedLabel")
	mFixOneClickLabel     = msgs.M("fix.oneClickLabel")
	mFixHeadingsApplied   = msgs.M("fix.headingsApplied")
	mFixSizeApplied       = msgs.M("fix.sizeApplied")
	mFixSizeAppliedIn     = msgs.M("fix.sizeAppliedIn")
	mFixHeadingOneSummary = msgs.M("fix.headingOneSummary")
	mFixHeadingsSummary   = msgs.M("fix.headingsSummary")
	mFixSizeSummary       = msgs.M("fix.sizeSummary")
	mFixSizeSummaryIn     = msgs.M("fix.sizeSummaryIn")

	mErrTarget         = msgs.M("err.target")
	mErrChanged        = msgs.M("err.changed")
	mErrWriteAlt       = msgs.M("err.writeAlt")
	mErrWriteLink      = msgs.M("err.writeLink")
	mErrNotColor       = msgs.M("err.notColor")
	mErrTextSize       = msgs.M("err.textSize")
	mErrNoHeading      = msgs.M("err.noHeading")
	mErrHeadingLevels  = msgs.M("err.headingLevels")
	mErrMixedText      = msgs.M("err.mixedText")
	mErrChooseRole     = msgs.M("err.chooseRole")
	mErrEmptyText      = msgs.M("err.emptyText")
	mErrHTMLCampaign   = msgs.M("err.htmlCampaign")
	mErrNothingToClick = msgs.M("err.nothingOneClick")
)

// IssueTexts are an issue's words as messages, kept so they can be written
// in each reader's language (TextIn); the issue's English fields are
// written from them.
type IssueTexts struct {
	Title i18n.Text
	// Detail is one message a sentence.
	Detail []i18n.Text
	Tag    i18n.Text
	Short  i18n.Text
	// Name is what an image, link or button issue is about, as a sentence
	// starts ("The hero image"); zero for the other kinds.
	Name i18n.Text
	// Subject is what a one-click fix changes ("Card titles").
	Subject i18n.Text
}

// TextIn writes a text this package kept (an issue's title, a fix's
// summary) in a reader's language. Arguments name sections and languages
// in that language too: "section" holds a section label
// (mjml.SectionLabel), an argument ending in "Langs" language codes and
// one ending in "List" a list of words, both joined the way the language
// joins a list; "ratio" and "need" are contrast ratios (2.1:1). It returns
// "" for a message no catalog has.
func TextIn(l i18n.Lang, t i18n.Text) string {
	if t.IsZero() {
		return ""
	}
	key, ok := strings.CutPrefix(t.ID, msgs.Namespace+".")
	if !ok {
		return t.In(l)
	}
	m, err := msgs.Lookup(key)
	if err != nil {
		return ""
	}
	args := make(i18n.Args, len(t.Args))
	for k, v := range t.Args {
		switch {
		case k == "section":
			if s, ok := v.(string); ok {
				v = mjml.SectionWord(l, s)
			}
		case strings.HasSuffix(k, "Langs"):
			codes := stringsOf(v)
			names := make([]string, 0, len(codes))
			for _, c := range codes {
				names = append(names, languageInSentence(l, c))
			}
			v = i18n.AndList(l.Locale, names)
		case strings.HasSuffix(k, "List"):
			v = i18n.AndList(l.Locale, stringsOf(v))
		case k == "ratio":
			if f, ok := v.(float64); ok {
				v = l.Region.Decimal(RoundDown(f), 1) + ":1"
			}
		case k == "need":
			if f, ok := v.(float64); ok {
				v = needIn(l, f)
			}
		}
		args[k] = v
	}
	return m.Text(l, args)
}

// needIn writes a required contrast: 4.5:1, 3:1.
func needIn(l i18n.Lang, n float64) string {
	if n == float64(int64(n)) {
		return l.Region.Int(int64(n)) + ":1"
	}
	return l.Region.Decimal(n, 1) + ":1"
}

// stringsOf reads a list argument: []string as kept, []any once it went
// through JSON.
func stringsOf(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, e := range x {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		return []string{x}
	}
	return nil
}

// english writes a kept text in English.
func english(t i18n.Text) string { return TextIn(i18n.Default, t) }

// sentencesIn writes sentences side by side; false when one isn't known.
func sentencesIn(l i18n.Lang, ts []i18n.Text) (string, bool) {
	parts := make([]string, 0, len(ts))
	for _, t := range ts {
		s := TextIn(l, t)
		if s == "" {
			return "", false
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " "), true
}

// titleIn writes a text that starts a sentence, with a capital: messages
// may start with an argument written in lower case ("footer text is 11 px").
func titleIn(l i18n.Lang, t i18n.Text) string { return capitalize(TextIn(l, t)) }

// worded sets an issue's English title, tag and detail from their messages
// and keeps the messages.
func worded(is AccessibilityIssue, title, tag i18n.Text, detail ...i18n.Text) AccessibilityIssue {
	is.Texts.Title, is.Texts.Tag, is.Texts.Detail = title, tag, detail
	is.Title, is.Tag = titleIn(i18n.Default, title), english(tag)
	is.Detail, _ = sentencesIn(i18n.Default, detail)
	return is
}

// languageNames' messages, by base code.
func languageMsg(code string) (i18n.Msg, bool) {
	m, err := msgs.Lookup("lang." + BaseLanguage(code))
	return m, err == nil
}

// LanguageNameIn names a content language in a reader's language: "es" is
// "Spanish" in English and "Tiếng Tây Ban Nha" in Vietnamese, as the app
// names it. A language without a name keeps its code.
func LanguageNameIn(l i18n.Lang, code string) string {
	if m, ok := languageMsg(code); ok {
		return m.Text(l, nil)
	}
	return code
}

// LanguageListIn is "English, Spanish and German" in a reader's language,
// as a list reads inside a sentence.
func LanguageListIn(l i18n.Lang, codes []string) string {
	names := make([]string, 0, len(codes))
	for _, c := range codes {
		names = append(names, languageInSentence(l, c))
	}
	return i18n.AndList(l.Locale, names)
}

// languageInSentence is a language's name inside a sentence. Vietnamese
// names start with the common noun "tiếng", in lower case after the first
// word of a sentence ("bằng tiếng Anh").
func languageInSentence(l i18n.Lang, code string) string {
	name := LanguageNameIn(l, code)
	if l.Locale == i18n.Vietnamese {
		return lowerFirstRune(name)
	}
	return name
}

func lowerFirstRune(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToLower(r)) + s[n:]
}

// In writes a report in a reader's language: its issues' titles, details,
// tags, short names, names, fix subjects and locations, and its languages'
// names. Check writes English, and English readers get the report as it
// is.
func (r AccessibilityReport) In(l i18n.Lang) AccessibilityReport {
	if l.Locale == i18n.English {
		return r
	}
	langs := make([]AccessibilityLanguage, len(r.Languages))
	for i, x := range r.Languages {
		x.Name = LanguageNameIn(l, x.Code)
		langs[i] = x
	}
	r.Languages = langs
	issues := make([]AccessibilityIssue, len(r.Issues))
	for i, is := range r.Issues {
		issues[i] = is.In(l)
	}
	r.Issues = issues
	return r
}

// In writes one issue's words in a reader's language.
func (is AccessibilityIssue) In(l i18n.Lang) AccessibilityIssue {
	if l.Locale == i18n.English {
		return is
	}
	t := is.Texts
	if s := titleIn(l, t.Title); s != "" {
		is.Title = s
	}
	if s := TextIn(l, t.Tag); s != "" {
		is.Tag = s
	}
	if s := TextIn(l, t.Short); s != "" {
		is.Short = s
	}
	if s := titleIn(l, t.Name); s != "" {
		is.Name = s
	}
	if s := titleIn(l, t.Subject); s != "" {
		is.Fix.Subject = s
	}
	if len(t.Detail) > 0 {
		if s, ok := sentencesIn(l, t.Detail); ok {
			is.Detail = s
		}
	}
	if len(is.Location) > 0 {
		loc := make([]string, len(is.Location))
		for i, part := range is.Location {
			if tag, label, ok := strings.Cut(part, " · "); ok {
				part = tag + " · " + mjml.SectionWord(l, label)
			}
			loc[i] = part
		}
		is.Location = loc
	}
	return is
}

// textError is a problem a person reads: its message is English in Error
// and in their language through ErrorIn.
type textError struct{ t i18n.Text }

func (e *textError) Error() string { return english(e.t) }

func errText(m i18n.Msg, args i18n.Args) error { return &textError{t: m.Keep(args)} }

// ErrorIn writes an error from this package (a fix that can't be applied)
// in a reader's language, in lower case as Error writes it; other errors
// keep their words.
func ErrorIn(l i18n.Lang, err error) string {
	var te *textError
	if errors.As(err, &te) {
		if s := TextIn(l, te.t); s != "" {
			return s
		}
	}
	var pe *mjml.ParseError
	if errors.As(err, &pe) && l.Locale != i18n.English {
		return pe.In(l)
	}
	return err.Error()
}

// langError is an error written in a request's language, wrapping the one
// it explains (errors.Is and errors.As still see it).
type langError struct {
	text string
	err  error
}

func (e *langError) Error() string { return e.text }
func (e *langError) Unwrap() error { return e.err }

// inLanguage makes err read in the request's language: callers that show
// err.Error() to the person (Review & send's fixes) show it in theirs.
func inLanguage(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	l := i18n.LangFrom(ctx)
	if l.Locale == i18n.English {
		return err
	}
	return &langError{text: ErrorIn(l, err), err: err}
}
