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
	prepared := protect.Replace(src)
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
	return Result{HTML: unprotect.Replace(html), Errors: []MJMLProblem{}, Duration: time.Since(start)}
}
