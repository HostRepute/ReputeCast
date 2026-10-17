package mjml

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	gomjml "github.com/preslavrachev/gomjml/mjml"
)

// MJMLProblem is a compile error or warning with its line (0 when unknown).
type MJMLProblem struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
	Tag     string `json:"tag,omitempty"`
}

// Result is the outcome of a server-side compile.
type Result struct {
	HTML     string
	Errors   []MJMLProblem
	Duration time.Duration
}

// gomjml decodes &quot;, &lt; and &gt; before parsing and writes attribute
// values back unescaped, so a quote in alt text would end the attribute. We
// swap those entities for private-use characters before compiling and put
// the entities back afterwards; the private-use range never appears in
// ordinary email content.
var protect = strings.NewReplacer(
	"&quot;", "", "&#34;", "", "&#x22;", "",
	"&lt;", "", "&#60;", "", "&#x3c;", "", "&#x3C;", "",
	"&gt;", "", "&#62;", "", "&#x3e;", "", "&#x3E;", "",
)

var unprotect = strings.NewReplacer("", "&quot;", "", "&lt;", "", "&gt;")

// voidHTML closes HTML void elements so the XML-based compiler accepts
// content written as HTML (<br>, <img …>).
var voidHTML = regexp.MustCompile(`(?i)<(br|hr|img|wbr|input|meta|link|col|source)((?:\s+[^<>]*?)?)\s*(/?)>`)

var xmlLine = regexp.MustCompile(`line (\d+): (.*)$`)

// Compile renders MJML to HTML with gomjml, the server-side twin of the
// composer's mjml-browser. It is used when content arrives as MJML only
// (the API, automations); the composer stores the HTML it compiled itself.
func Compile(src string) Result {
	start := time.Now()
	if _, err := Parse(src); err != nil {
		var pe *ParseError
		if errors.As(err, &pe) {
			return Result{Errors: []MJMLProblem{{Line: pe.Line, Message: pe.Error()}}, Duration: time.Since(start)}
		}
		return Result{Errors: []MJMLProblem{{Message: err.Error()}}, Duration: time.Since(start)}
	}
	// ReputeCast's rc-* attributes come out and conditional blocks get their
	// markers (conditional.go).
	prepared := protect.Replace(markBodyRaw(PrepareCompile(src)))
	prepared = voidHTML.ReplaceAllStringFunc(prepared, func(m string) string {
		if strings.HasSuffix(m, "/>") {
			return m
		}
		return strings.TrimSuffix(m, ">") + "/>"
	})
	html, err := gomjml.Render(prepared)
	if err != nil {
		p := MJMLProblem{Message: err.Error()}
		if m := xmlLine.FindStringSubmatch(err.Error()); m != nil {
			p.Line, _ = strconv.Atoi(m[1])
			p.Message = "Line " + m[1] + ": " + m[2]
		}
		var me *gomjml.Error
		if errors.As(err, &me) && len(me.Details) > 0 {
			d := me.Details[0]
			p = MJMLProblem{Line: d.Line, Message: d.Message, Tag: d.TagName}
		}
		return Result{Errors: []MJMLProblem{p}, Duration: time.Since(start)}
	}
	return Result{HTML: repairRaw(unprotect.Replace(html)), Errors: []MJMLProblem{}, Duration: time.Since(start)}
}

// Markers around the content of Raw HTML blocks placed directly in mj-body.
const (
	rawStart = "<!--rc-raw-start-->"
	rawEnd   = "<!--rc-raw-end-->"
)

// markBodyRaw marks the content of every Raw HTML block in mj-body. gomjml
// puts such a block between two sections inside Outlook's conditional
// comment, which hides it from every other client; repairRaw uses the
// markers to close the comment around it, as MJML does.
func markBodyRaw(src string) string {
	root, err := Parse(src)
	if err != nil {
		return src
	}
	body := Body(root)
	if body == nil {
		return src
	}
	type insert struct {
		at   int
		text string
	}
	var ins []insert
	for _, c := range body.Children {
		if c.Tag != "mj-raw" || c.End <= c.Start {
			continue
		}
		open := strings.IndexByte(src[c.Start:c.End], '>')
		closeTag := strings.LastIndex(strings.ToLower(src[c.Start:c.End]), "</mj-raw")
		if open < 0 || closeTag < 0 || c.Start+open+1 > c.Start+closeTag || src[c.Start+open-1] == '/' {
			continue
		}
		ins = append(ins, insert{c.Start + open + 1, rawStart}, insert{c.Start + closeTag, rawEnd})
	}
	if len(ins) == 0 {
		return src
	}
	var b strings.Builder
	last := 0
	for _, in := range ins {
		b.WriteString(src[last:in.at])
		b.WriteString(in.text)
		last = in.at
	}
	b.WriteString(src[last:])
	return b.String()
}

// repairRaw closes Outlook's conditional comment before each marked Raw
// HTML block it was left open around, reopens it after, and removes the
// markers.
func repairRaw(h string) string {
	if !strings.Contains(h, rawStart) {
		return h
	}
	var b strings.Builder
	inCond := false
	// track updates inCond from the conditional comments in s, in order.
	track := func(s string) {
		for {
			o, c := strings.Index(s, "<!--[if"), strings.Index(s, "<![endif]-->")
			switch {
			case o < 0 && c < 0:
				return
			case c < 0 || (o >= 0 && o < c):
				inCond = true
				s = s[o+len("<!--[if"):]
			default:
				inCond = false
				s = s[c+len("<![endif]-->"):]
			}
		}
	}
	rest := h
	for {
		i := strings.Index(rest, rawStart)
		if i < 0 {
			b.WriteString(rest)
			return b.String()
		}
		before := rest[:i]
		b.WriteString(before)
		track(before)
		if inCond {
			b.WriteString("<![endif]-->")
			inCond = false
		}
		rest = rest[i+len(rawStart):]
		j := strings.Index(rest, rawEnd)
		if j < 0 {
			b.WriteString(rest)
			return b.String()
		}
		b.WriteString(rest[:j])
		rest = rest[j+len(rawEnd):]
		if strings.HasPrefix(strings.TrimSpace(rest), rawStart) {
			continue
		}
		// Reopen when what follows is the rest of the conditional comment.
		c, o := strings.Index(rest, "<![endif]-->"), strings.Index(rest, "<!--[if")
		if c >= 0 && (o < 0 || c < o) {
			b.WriteString("<!--[if mso | IE]>")
			inCond = true
		}
	}
}
