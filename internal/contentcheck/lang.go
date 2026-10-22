package contentcheck

import (
	"embed"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
)

// An issue's words (title, detail, canvas tag, fix labels) are messages in
// messages/<locale>.json. Check writes them in English, as Send Readiness,
// API clients and agents read them, and keeps the messages with each issue;
// ContentCheckReport.In writes a report in a reader's language.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "contentcheck")

var (
	mAltTitle  = msgs.M("alt.title")
	mAltTag    = msgs.M("alt.tag")
	mAltDetail = msgs.M("alt.detail")
	mAltFix    = msgs.M("alt.fix")

	mSizeNear          = msgs.M("size.near")
	mSizeOver          = msgs.M("size.over")
	mSizeDetail        = msgs.M("size.detail")
	mSizeMinify        = msgs.M("size.minify")
	mSizeBulkOne       = msgs.M("size.bulkOne")
	mSizeBulkMany      = msgs.M("size.bulkMany")
	mSizeBulkCards     = msgs.M("size.bulkCards")
	mSizeRemoveSection = msgs.M("size.removeSection")
	mSizeRemoveRow     = msgs.M("size.removeRow")

	mRatioTitle  = msgs.M("ratio.title")
	mRatioDetail = msgs.M("ratio.detail")

	mFooterBoth           = msgs.M("footer.both")
	mFooterBothTag        = msgs.M("footer.bothTag")
	mFooterAddress        = msgs.M("footer.address")
	mFooterAddressTag     = msgs.M("footer.addressTag")
	mFooterUnsubscribe    = msgs.M("footer.unsubscribe")
	mFooterUnsubscribeTag = msgs.M("footer.unsubscribeTag")
	mFooterDetail         = msgs.M("footer.detail")
	mFooterFix            = msgs.M("footer.fix")

	mLiquidTitle         = msgs.M("liquid.title")
	mLiquidDetail        = msgs.M("liquid.detail")
	mUnknownFieldTitle   = msgs.M("unknownField.title")
	mUnknownFieldDetail  = msgs.M("unknownField.detail")
	mSubjectMissing      = msgs.M("subject.missing")
	mSubjectMissingDet   = msgs.M("subject.missingDetail")
	mSubjectSpam         = msgs.M("subject.spam")
	mSubjectSuggested    = msgs.M("subject.suggested")
	mSubjectUseSuggested = msgs.M("subject.useSuggestion")
	mPlainTextTitle      = msgs.M("plainText.title")
	mPlainTextDetail     = msgs.M("plainText.detail")

	mLinkEmptyTitle       = msgs.M("link.emptyTitle")
	mLinkEmptyTag         = msgs.M("link.emptyTag")
	mLinkEmptyDetail      = msgs.M("link.emptyDetail")
	mLinkEdit             = msgs.M("link.edit")
	mLinkBlocklistTitle   = msgs.M("link.blocklistTitle")
	mLinkBlocklistTag     = msgs.M("link.blocklistTag")
	mLinkBlocklistDetail  = msgs.M("link.blocklistDetail")
	mLinkReplaceWith      = msgs.M("link.replaceWith")
	mLinkShortenerTitle   = msgs.M("link.shortenerTitle")
	mLinkShortenerTag     = msgs.M("link.shortenerTag")
	mLinkShortenerDetail  = msgs.M("link.shortenerDetail")
	mLinkBrokenTitle      = msgs.M("link.brokenTitle")
	mLinkBrokenTag        = msgs.M("link.brokenTag")
	mLinkBrokenDetail     = msgs.M("link.brokenDetail")
)

// TextIn writes a message this package kept in a reader's language. Its
// "section" argument is a section label (mjml.SectionLabel), named in that
// language too, and "lists" a list of names, joined the way the language
// joins them. Messages of other packages are written as i18n.Text.In
// writes them. It returns "" for a message no catalog has.
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
		switch k {
		case "section":
			if s, ok := v.(string); ok {
				v = mjml.SectionWord(l, s)
			}
		case "lists":
			v = i18n.AndList(l.Locale, stringsOf(v))
		}
		args[k] = v
	}
	return m.Text(l, args)
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

// english writes a kept message in English.
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

// worded fills an issue's English words from their messages and keeps the
// messages: its title, canvas tag (none when zero) and detail, one message
// a sentence.
func worded(is ContentCheckIssue, title, tag i18n.Text, detail ...i18n.Text) ContentCheckIssue {
	is.TitleText, is.TagText, is.DetailText = title, tag, detail
	is.Title = english(title)
	if !tag.IsZero() {
		is.Tag = english(tag)
	}
	is.Detail, _ = sentencesIn(i18n.Default, detail)
	return is
}

// percentArg is a whole percentage (92) as a {share, number, percent}
// argument, which the reader's region writes: 92%.
func percentArg(pct float64) float64 { return pct / 100 }

// fixOf is a fix whose label is a message.
func fixOf(kind string, label i18n.Text, path []int) ContentCheckFix {
	return ContentCheckFix{Kind: kind, Label: english(label), LabelText: label, Path: path}
}

// In writes a report's words in a reader's language: its issues' titles,
// details, canvas tags and fix labels. Check writes English, and so does
// In for English readers; issues another domain added without their
// messages keep their English.
func (r ContentCheckReport) In(l i18n.Lang) ContentCheckReport {
	if l.Locale == i18n.English || len(r.Issues) == 0 {
		return r
	}
	issues := make([]ContentCheckIssue, len(r.Issues))
	for i, is := range r.Issues {
		issues[i] = is.In(l)
	}
	r.Issues = issues
	return r
}

// In writes one issue's words in a reader's language.
func (is ContentCheckIssue) In(l i18n.Lang) ContentCheckIssue {
	if l.Locale == i18n.English {
		return is
	}
	if s := TextIn(l, is.TitleText); s != "" {
		is.Title = s
	}
	if s := TextIn(l, is.TagText); s != "" {
		is.Tag = s
	}
	if len(is.DetailText) > 0 {
		if s, ok := sentencesIn(l, is.DetailText); ok {
			is.Detail = s
		}
	}
	if len(is.Fixes) > 0 {
		fixes := make([]ContentCheckFix, len(is.Fixes))
		for i, f := range is.Fixes {
			if s := TextIn(l, f.LabelText); s != "" {
				f.Label = s
			}
			fixes[i] = f
		}
		is.Fixes = fixes
	}
	return is
}
