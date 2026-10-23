package mjml

import (
	"embed"
	"regexp"
	"strconv"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What people read from this package is in messages/<locale>.json: the
// layout warnings (LintIn), the parse errors (ParseError.In,
// MJMLProblem.In, CompileIn) and the names of an email's sections
// (SectionWord). Lint, Compile and ParseError.Error write English, as API
// clients and logs read them. The section labels Outline and SectionLabel
// return ("hero", "2 columns", "footer") stay English: code compares them.

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "mjml")

var (
	mLine = msgs.M("line")

	mUnclosedComment      = msgs.M("parse.unclosedComment")
	mUnfinishedClosingTag = msgs.M("parse.unfinishedClosingTag")
	mNoOpeningTag         = msgs.M("parse.noOpeningTag")
	mUnclosedTag          = msgs.M("parse.unclosedTag")
	mUnfinishedTag        = msgs.M("parse.unfinishedTag")
	mUnclosedQuote        = msgs.M("parse.unclosedQuote")
	mEmpty                = msgs.M("parse.empty")
	mNoRoot               = msgs.M("parse.noRoot")

	mLintColumns    = msgs.M("lint.columns")
	mLintImageWidth = msgs.M("lint.imageWidth")

	mSectionColumns = msgs.M("section.columns")
)

// sectionWords are the section labels people read, by label.
var sectionWords = map[string]i18n.Msg{
	"header":               msgs.M("section.header"),
	"footer":               msgs.M("section.footer"),
	"hero":                 msgs.M("section.hero"),
	"raw HTML":             msgs.M("section.rawHtml"),
	"wrapper":              msgs.M("section.wrapper"),
	"divider":              msgs.M("section.divider"),
	"image":                msgs.M("section.image"),
	"button":               msgs.M("section.button"),
	"text":                 msgs.M("section.text"),
	"article cards":        msgs.M("section.articleCards"),
	"product cards":        msgs.M("section.productCards"),
	"poll":                 msgs.M("section.poll"),
	"sponsor":              msgs.M("section.sponsor"),
	"upgrade":              msgs.M("section.upgrade"),
	"feed items":           msgs.M("section.feedItems"),
	"store items":          msgs.M("section.storeItems"),
	"transactional footer": msgs.M("section.transactionalFooter"),
	"receipt lines":        msgs.M("section.receiptLines"),
	"preview end":          msgs.M("section.previewEnd"),
	"section":              msgs.M("section.section"),
	"column":               msgs.M("section.column"),
	"group":                msgs.M("section.group"),
	"spacer":               msgs.M("section.spacer"),
	"social":               msgs.M("section.social"),
	"navbar":               msgs.M("section.navbar"),
	"table":                msgs.M("section.table"),
}

var columnsLabel = regexp.MustCompile(`^(\d+) columns$`)

// SectionWord names a section label (Outline, SectionLabel) in a reader's
// language, in lower case as it reads inside a sentence: "chân thư" for
// "footer", "2 cột" for "2 columns". Labels it doesn't know, such as a
// block's own name, stay as they are.
func SectionWord(l i18n.Lang, label string) string {
	if m := columnsLabel.FindStringSubmatch(label); m != nil {
		n, _ := strconv.Atoi(m[1])
		return mSectionColumns.Text(l, i18n.Args{"count": n})
	}
	if m, ok := sectionWords[label]; ok {
		return m.Text(l, nil)
	}
	return label
}

// parseError is a syntax problem written from its message.
func parseError(line int, m i18n.Msg, args i18n.Args) *ParseError {
	return &ParseError{Line: line, Message: m.Text(i18n.Default, args), text: m.Keep(args)}
}

// atLine writes "Line 3: …" in a language.
func atLine(l i18n.Lang, line int, problem string) string {
	return mLine.Text(l, i18n.Args{"line": strconv.Itoa(line), "problem": problem})
}

// In writes the problem in a reader's language: "Line 3: unclosed
// <mj-section>", "Dòng 3: <mj-section> chưa được đóng".
func (e *ParseError) In(l i18n.Lang) string {
	problem := e.Message
	if s := e.text.In(l); s != "" {
		problem = s
	}
	return atLine(l, e.Line, problem)
}

// lineProblem is a problem of ours at a line: the message after "Line N: ".
func lineProblem(line int, tag string, m i18n.Msg, args i18n.Args) MJMLProblem {
	return MJMLProblem{Line: line, Tag: tag, Message: atLine(i18n.Default, line, m.Text(i18n.Default, args)), text: m.Keep(args)}
}

// In writes a compile problem or a layout warning in a reader's language.
// What the compiler reports in its own words stays in English after the
// line.
func (p MJMLProblem) In(l i18n.Lang) string {
	switch {
	case !p.text.IsZero():
		if s := p.text.In(l); s != "" {
			return atLine(l, p.Line, s)
		}
	case p.detail != "":
		return atLine(l, p.Line, p.detail)
	}
	return p.Message
}

// CompileIn is Compile with its problems written in a reader's language.
func CompileIn(l i18n.Lang, src string) Result {
	res := Compile(src)
	for i := range res.Errors {
		res.Errors[i].Message = res.Errors[i].In(l)
	}
	return res
}
