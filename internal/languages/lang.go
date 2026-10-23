package languages

import (
	"context"
	"embed"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
	"golang.org/x/text/unicode/norm"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// What the server writes about content languages follows the reader's app
// language (i18n.LangFrom): problems, language names (CLDR's, as browsers
// name them), the compare view's block names, what changed in a version
// after an approval, and the labels of saved versions. A saved version
// keeps its English label for API clients and older rows, with its
// message beside it (label_text) to read in each reader's language. The
// messages are in messages/<locale>.json.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "languages")

var (
	mSending               = msgs.M("problem.sending")
	mSent                  = msgs.M("problem.sent")
	mPickWorkspaceLanguage = msgs.M("problem.pickWorkspaceLanguage")
	mRTL                   = msgs.M("problem.rtl")
	mNotWorkspaceLanguage  = msgs.M("problem.notWorkspaceLanguage")
	mWrittenIn             = msgs.M("problem.writtenIn")
	mHasVersion            = msgs.M("problem.hasVersion")
	mApplyDoesntCompile    = msgs.M("problem.applyDoesntCompile")
	mSourceOneVersion      = msgs.M("problem.sourceOneVersion")
	mNoMJML                = msgs.M("problem.noMJML")
	mWhichBlock            = msgs.M("problem.whichBlock")
	mDoesntCompile         = msgs.M("problem.doesntCompile")
	mPickFromList          = msgs.M("problem.pickFromList")
	mAlreadyAdded          = msgs.M("problem.alreadyAdded")
	mIsDefault             = msgs.M("problem.isDefault")
	mIsFallback            = msgs.M("problem.isFallback")
	mPickYourLanguage      = msgs.M("problem.pickYourLanguage")
	mPickDefault           = msgs.M("problem.pickDefault")
	mPickFallback          = msgs.M("problem.pickFallback")
	mFlowSourceLanguage    = msgs.M("problem.flowSourceLanguage")
	mArchived              = msgs.M("problem.archived")
	mSubjectOneLine        = msgs.M("problem.subjectOneLine")
	mConflict              = msgs.M("problem.conflict")
	mConflictSomeone       = msgs.M("problem.conflictSomeone")
	mMJML                  = msgs.M("problem.mjml")
	mTooManyVersions       = msgs.M("problem.tooManyVersions")
	mVersionNotFound       = msgs.M("problem.versionNotFound")
	mLanguageNotFound      = msgs.M("problem.languageNotFound")
	mBlockNotFound         = msgs.M("problem.blockNotFound")

	mLabelCopiedFrom     = msgs.M("label.copiedFrom")
	mLabelLayoutOf       = msgs.M("label.layoutOf")
	mLabelAppliedFrom    = msgs.M("label.appliedFrom")
	mLabelBlockCopied    = msgs.M("label.blockCopied")
	mLabelBlockRemoved   = msgs.M("label.blockRemoved")
	mLabelAgent          = msgs.M("label.agent")
	mLabelFlowEditor     = msgs.M("label.flowEditor")
	mLabelTranslatedFrom = msgs.M("label.translatedFrom")

	mBlockHeading = msgs.M("block.heading")
	mBlockShort   = msgs.M("block.short")

	mSummaryLogo      = msgs.M("summary.logo")
	mSummaryImage     = msgs.M("summary.image")
	mSummaryNoAlt     = msgs.M("summary.noAlt")
	mSummaryNoAltFile = msgs.M("summary.noAltFile")

	mDiffNone    = msgs.M("diff.none")
	mDiffAdded   = msgs.M("diff.added")
	mDiffChanged = msgs.M("diff.changed")
	mDiffStatus  = msgs.M("diff.status")
	mDiffHeld    = msgs.M("diff.held")
	mDiffRemoved = msgs.M("diff.removed")

	mChangeVersionAdded        = msgs.M("change.versionAdded")
	mChangeVersionRemoved      = msgs.M("change.versionRemoved")
	mChangeMarkedTranslated    = msgs.M("change.markedTranslated")
	mChangeBackToDraft         = msgs.M("change.backToDraft")
	mChangeSentAgain           = msgs.M("change.sentAgain")
	mChangeHeld                = msgs.M("change.held")
	mChangeEdited              = msgs.M("change.edited")
	mChangeSubject             = msgs.M("change.subject")
	mChangeBlock               = msgs.M("change.block")
	mChangeBlocks              = msgs.M("change.blocks")
	mChangeSubjectBlock        = msgs.M("change.subjectBlock")
	mChangeSubjectBlocks       = msgs.M("change.subjectBlocks")
	mChangeBlockRemoved        = msgs.M("change.blockRemoved")
	mChangeSubjectBlockRemoved = msgs.M("change.subjectBlockRemoved")

	mReadersNoLanguage     = msgs.M("readers.noLanguage")
	mReadersOtherLanguages = msgs.M("readers.otherLanguages")
)

// langOf is the language the request reads in.
func langOf(ctx context.Context) i18n.Lang { return i18n.LangFrom(ctx) }

// named is a message whose {language} names a language code in l, as a
// name reads inside a sentence.
func named(l i18n.Lang, m i18n.Msg, code string, args i18n.Args) string {
	a := i18n.Args{"language": nameInSentence(l, code)}
	for k, v := range args {
		a[k] = v
	}
	return m.Text(l, a)
}

var (
	namersMu sync.Mutex
	namers   = map[i18n.Locale]display.Namer{}
)

// NameIn names a language in a reader's language, as a label: "Spanish",
// "Tiếng Tây Ban Nha". English readers (and the pseudo-locale) get the
// catalog's English name, NameOf. Other readers get the names the app
// uses elsewhere (accessibility.LanguageNameIn), else CLDR's, as browsers
// name languages; a code the catalog doesn't have is named by its code.
func NameIn(l i18n.Lang, code string) string {
	if l.Locale == "" || l.Locale == i18n.English || l.Locale == i18n.Pseudo {
		return NameOf(code)
	}
	lang, ok := Lookup(code)
	if !ok {
		return code
	}
	if n := accessibility.LanguageNameIn(l, lang.Code); n != "" && n != lang.Code {
		return n
	}
	tag, err := language.Parse(lang.Code)
	if err != nil {
		return lang.Name
	}
	namersMu.Lock()
	n, cached := namers[l.Locale]
	if !cached {
		n = display.Languages(l.Locale.Tag())
		namers[l.Locale] = n
	}
	namersMu.Unlock()
	if n != nil {
		if s := n.Name(tag); s != "" {
			return s
		}
	}
	return lang.Name
}

// NameInSentence is a language's name inside a sentence: Vietnamese names
// start with the common noun "tiếng", in lower case there ("bằng tiếng
// Anh"); English names read as NameIn.
func NameInSentence(l i18n.Lang, code string) string { return nameInSentence(l, code) }

func nameInSentence(l i18n.Lang, code string) string {
	name := NameIn(l, code)
	if l.Locale != i18n.Vietnamese {
		return name
	}
	r, n := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError {
		return name
	}
	return string(unicode.ToLower(r)) + name[n:]
}

// plainKey lowercases a text and drops its diacritics, to search and sort
// language names in any language: "Tiếng Ả Rập" → "tieng a rap".
func plainKey(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(strings.TrimSpace(s))) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case r == 'đ':
			b.WriteRune('d')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// notFound answers 404 with a sentence in the request's language.
func notFound(ctx context.Context, m i18n.Msg) *api.Problem {
	return api.Err(http.StatusNotFound, api.CodeNotFound, m.Text(langOf(ctx), nil))
}

// versionNotFound is the 404 of a missing language version: "Spanish
// version not found."
func versionNotFound(l i18n.Lang, lang string) *api.Problem {
	return api.Err(http.StatusNotFound, api.CodeNotFound, named(l, mVersionNotFound, lang, nil))
}

// zoneOf is a workspace's time zone, for times in sentences.
func zoneOf(ctx context.Context, q *store.Queries, wsID uuid.UUID) *time.Location {
	tz := ""
	if ws := api.WorkspaceFrom(ctx); ws != nil && ws.ID == wsID {
		tz = ws.Timezone
	} else if ws, err := q.GetWorkspace(ctx, wsID); err == nil {
		tz = ws.Timezone
	}
	if loc, err := time.LoadLocation(tz); err == nil && tz != "" {
		return loc
	}
	return time.UTC
}

// --- saved versions' labels ---------------------------------------------------

// label is a saved version's label: its English, stored as it always was,
// and its message kept for readers of other languages. A language in it is
// kept as its code ("code") and named in each reader's language.
type label struct {
	English string
	Text    i18n.Text
}

func newLabel(m i18n.Msg, code string, args i18n.Args) label {
	a := i18n.Args{}
	for k, v := range args {
		a[k] = v
	}
	if code != "" {
		a["code"] = code
	}
	return label{English: m.Text(i18n.Default, withLanguage(i18n.Default, a)), Text: m.Keep(a)}
}

// raw is the kept message as stored in label_text.
func (lb label) raw() []byte {
	if lb.Text.IsZero() {
		return nil
	}
	b, err := json.Marshal(lb.Text)
	if err != nil {
		return nil
	}
	return b
}

// withLanguage names a kept language code ("code") as {language}, in l.
func withLanguage(l i18n.Lang, args i18n.Args) i18n.Args {
	code, ok := args["code"].(string)
	if !ok {
		return args
	}
	out := i18n.Args{"language": nameInSentence(l, code)}
	for k, v := range args {
		out[k] = v
	}
	return out
}

// labelIn is a saved version's label for a reader: the English stored with
// it for English readers and for saves without a kept message (older
// rows, labels the composer sent), the message in the reader's language
// otherwise.
func labelIn(l i18n.Lang, english string, raw []byte) string {
	if l.Locale == i18n.English || len(raw) == 0 {
		return english
	}
	var t i18n.Text
	if json.Unmarshal(raw, &t) != nil || t.IsZero() {
		return english
	}
	t.Args = withLanguage(l, t.Args)
	if s := t.In(l); s != "" {
		return s
	}
	return english
}

// --- block names ---------------------------------------------------------------

// BlockRef says what a block is, so it can be named in any language: its
// kind (header, headerLinks, footer, raw, social, cards, columns, hero,
// image, heroImage, button, headline, intro, text, email, other) and what
// its name needs: a column count, the first words of its text, the MJML
// label of an unusual block, its highest heading.
type BlockRef struct {
	Kind    string `json:"kind"`
	Count   int    `json:"count,omitempty"`
	Words   string `json:"words,omitempty"`
	Label   string `json:"label,omitempty"`
	Heading int    `json:"heading,omitempty"`
}

func (r BlockRef) args() i18n.Args {
	name := r.Label
	if name != "" {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	return i18n.Args{"kind": r.Kind, "count": r.Count, "words": r.Words, "label": r.Label, "name": name}
}

// Name is the block's name in the compare view, in l: "Header · logo and
// links", "Đầu thư · logo và liên kết".
func (r BlockRef) Name(l i18n.Lang) string {
	m, err := msgs.Lookup("block." + r.Kind)
	if err != nil {
		m = msgs.M("block.other")
	}
	name := m.Text(l, r.args())
	if r.Heading > 0 {
		name = mBlockHeading.Text(l, i18n.Args{"name": name, "level": r.Heading})
	}
	return name
}

// Short is the block in a sentence, in l: "intro", "two cards".
func (r BlockRef) Short(l i18n.Lang) string { return mBlockShort.Text(l, r.args()) }

// nameIn is a block's name for a reader.
func (b Block) nameIn(l i18n.Lang) string {
	if l.Locale == i18n.English || b.ref.Kind == "" {
		return b.Name
	}
	return b.ref.Name(l)
}

// Ref says what the block is (BlockRef).
func (b Block) Ref() BlockRef { return b.ref }

var (
	altRe   = regexp.MustCompile(`(Logo|Image), alt "([^"\n]*)"`)
	noAltRe = regexp.MustCompile(`(\S+) · no alt text`)
)

// summaryIn writes the compare view's text of a block for a reader: the
// email's own words stay as they are, ReputeCast's description of its
// images reads in the reader's language. Stored texts (what a version was
// translated from) are written the same way, so the diff compares like
// with like.
func summaryIn(l i18n.Lang, s string) string {
	if l.Locale == i18n.English || s == "" {
		return s
	}
	s = altRe.ReplaceAllStringFunc(s, func(m string) string {
		p := altRe.FindStringSubmatch(m)
		msg := mSummaryImage
		if p[1] == "Logo" {
			msg = mSummaryLogo
		}
		return msg.Text(l, i18n.Args{"alt": p[2]})
	})
	return noAltRe.ReplaceAllStringFunc(s, func(m string) string {
		p := noAltRe.FindStringSubmatch(m)
		if p[1] == "Image" {
			return mSummaryNoAlt.Text(l, nil)
		}
		return mSummaryNoAltFile.Text(l, i18n.Args{"file": p[1]})
	})
}
