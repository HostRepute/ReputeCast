// Package liquid renders personalization in subjects, preheaders and email
// bodies with Liquid (osteele/liquid): {{ first_name | default: "there" }}.
//
// Email HTML escapes quotes and angle brackets, and editors write them as
// entities, so Prepare decodes entities inside {{ … }} and {% … %} before
// parsing. The composer's short form {{ first_name | "there" }} is accepted
// and read as default: "there". Rendering is bounded: no includes, ranges of
// at most 1,000 items and at most 4 MB of output.
package liquid

import (
	"errors"
	"fmt"
	"html"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	osteele "github.com/osteele/liquid"
)

// MaxOutput is the largest rendered result, in bytes.
const MaxOutput = 4 << 20

// MaxRange is the longest literal range a for loop may iterate.
const MaxRange = 1000

// PersonalizationToken is a personalization token offered by the composer.
type PersonalizationToken struct {
	Key   string `json:"key" example:"first_name"`
	Label string `json:"label" example:"First name"`
	Kind  string `json:"kind" enum:"contact,custom,system"`
	// Fallback is the suggested default for contacts without a value.
	Fallback string `json:"fallback,omitempty"`
}

// CustomField is a workspace contact field (custom_fields).
type CustomField struct {
	Key, Label, Type string
}

// ContactFields are on every contact.
var ContactFields = []PersonalizationToken{
	{Key: "first_name", Label: "First name", Kind: "contact", Fallback: "there"},
	{Key: "last_name", Label: "Last name", Kind: "contact"},
	{Key: "email", Label: "Email address", Kind: "contact"},
}

// SystemFields are links ReputeCast fills in per recipient at send time.
var SystemFields = []PersonalizationToken{
	{Key: "unsubscribe_url", Label: "Unsubscribe link", Kind: "system"},
	{Key: "preferences_url", Label: "Email preferences link", Kind: "system"},
	{Key: "view_in_browser_url", Label: "View in browser link", Kind: "system"},
}

// Catalog lists the tokens of a workspace: contact fields, its custom
// fields, then the system links.
func Catalog(custom []CustomField) []PersonalizationToken {
	out := make([]PersonalizationToken, 0, len(ContactFields)+len(custom)+len(SystemFields))
	out = append(out, ContactFields...)
	for _, c := range custom {
		f := PersonalizationToken{Key: c.Key, Label: c.Label, Kind: "custom"}
		if c.Key == "plan" {
			f.Fallback = "free"
		}
		out = append(out, f)
	}
	return append(out, SystemFields...)
}

var (
	spanRe  = regexp.MustCompile(`\{\{-?[\s\S]*?-?\}\}|\{%-?[\s\S]*?-?%\}`)
	rangeRe = regexp.MustCompile(`\(\s*(-?\d+)\s*\.\.\s*(-?\d+)\s*\)`)
	identRe = regexp.MustCompile(`^[A-Za-z_][\w.-]*`)
)

// Prepare decodes HTML entities inside Liquid delimiters and rewrites the
// short fallback form, so Liquid embedded in compiled HTML parses.
func Prepare(src string) string {
	if !strings.Contains(src, "{{") && !strings.Contains(src, "{%") {
		return src
	}
	return spanRe.ReplaceAllStringFunc(src, func(span string) string {
		s := html.UnescapeString(span)
		s = strings.ReplaceAll(s, " ", " ")
		if strings.HasPrefix(s, "{{") {
			s = shortDefault(s)
		}
		return s
	})
}

// shortDefault turns {{ x | "y" }} into {{ x | default: "y" }}.
func shortDefault(span string) string {
	open, close := "{{", "}}"
	inner := span[2 : len(span)-2]
	if strings.HasPrefix(inner, "-") {
		open += "-"
		inner = inner[1:]
	}
	if strings.HasSuffix(inner, "-") {
		close = "-" + close
		inner = inner[:len(inner)-1]
	}
	parts := splitFilters(inner)
	changed := false
	for i := 1; i < len(parts); i++ {
		p := strings.TrimSpace(parts[i])
		if isStringLiteral(p) {
			parts[i] = " default: " + p + " "
			changed = true
		}
	}
	if !changed {
		return span
	}
	return open + strings.Join(parts, "|") + close
}

// splitFilters splits an expression on | outside quotes.
func splitFilters(s string) []string {
	var out []string
	var q byte
	last := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case q != 0:
			if c == q {
				q = 0
			}
		case c == '"' || c == '\'':
			q = c
		case c == '|':
			out = append(out, s[last:i])
			last = i + 1
		}
	}
	return append(out, s[last:])
}

func isStringLiteral(s string) bool {
	if len(s) < 2 {
		return false
	}
	q := s[0]
	if (q != '"' && q != '\'') || s[len(s)-1] != q {
		return false
	}
	return !strings.ContainsRune(s[1:len(s)-1], rune(q))
}

// LiquidProblem is a template error with its line.
type LiquidProblem struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

var engine = func() *osteele.Engine {
	e := osteele.NewEngine()
	e.UnregisterTag("include")
	return e
}()

func checkRanges(src string) error {
	for _, m := range rangeRe.FindAllStringSubmatch(src, -1) {
		a, _ := strconv.Atoi(m[1])
		b, _ := strconv.Atoi(m[2])
		if b-a+1 > MaxRange {
			return fmt.Errorf("loops are limited to %d items; (%s..%s) has %d", MaxRange, m[1], m[2], b-a+1)
		}
	}
	return nil
}

// Validate parses a template and returns its problems (empty when it parses).
func Validate(src string) []LiquidProblem {
	out := []LiquidProblem{}
	p := Prepare(src)
	if err := checkRanges(p); err != nil {
		out = append(out, LiquidProblem{Message: sentence(err.Error())})
	}
	if _, err := engine.ParseString(p); err != nil {
		out = append(out, problem(err, p))
	}
	return out
}

var snippetRe = regexp.MustCompile(`in (\{%[\s\S]*?%\}|\{\{[\s\S]*?\}\})\.?$`)

func problem(err osteele.SourceError, src string) LiquidProblem {
	msg := err.Error()
	// "Liquid error (line 3): unterminated …" → "Line 3: unterminated …"
	if i := strings.Index(msg, "): "); i > 0 && strings.HasPrefix(msg, "Liquid") {
		msg = msg[i+3:]
	}
	line := err.LineNumber()
	// The parser reports the line of the template start for unclosed blocks;
	// locate the quoted tag instead.
	if m := snippetRe.FindStringSubmatch(msg); m != nil {
		if i := strings.Index(src, m[1]); i >= 0 {
			line = strings.Count(src[:i], "\n") + 1
		}
	}
	if line > 0 {
		return LiquidProblem{Line: line, Message: fmt.Sprintf("Line %d: %s", line, sentence(msg))}
	}
	return LiquidProblem{Message: sentence(msg)}
}

func sentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	s = strings.ToUpper(s[:1]) + s[1:]
	if !strings.HasSuffix(s, ".") {
		s += "."
	}
	return s
}

// ErrTooLarge is returned when rendering exceeds MaxOutput.
var ErrTooLarge = errors.New("the rendered email is larger than 4 MB")

type limitWriter struct {
	b strings.Builder
}

func (w *limitWriter) Write(p []byte) (int, error) {
	if w.b.Len()+len(p) > MaxOutput {
		return 0, ErrTooLarge
	}
	return w.b.Write(p)
}

func Render(src string, vars map[string]any) (string, error) {
	if !strings.Contains(src, "{{") && !strings.Contains(src, "{%") {
		return src, nil
	}
	p := Prepare(src)
	if err := checkRanges(p); err != nil {
		return "", err
	}
	tpl, err := engine.ParseString(p)
	if err != nil {
		return "", errors.New(problem(err, p).Message)
	}
	w := &limitWriter{}
	if err := tpl.FRender(io.Writer(w), osteele.Bindings(vars)); err != nil {
		if errors.Is(err, ErrTooLarge) || strings.Contains(err.Error(), ErrTooLarge.Error()) {
			return "", ErrTooLarge
		}
		return "", errors.New(problem(err, p).Message)
	}
	return w.b.String(), nil
}

// Use is a variable a template prints, with its fallback.
type Use struct {
	Key         string `json:"key"`
	Fallback    string `json:"fallback,omitempty"`
	HasFallback bool   `json:"hasFallback"`
	Count       int    `json:"count"`
}

// Uses lists the variables printed with {{ … }}, in order of first use.
func Uses(src string) []Use {
	out := []Use{}
	idx := map[string]int{}
	for _, span := range spanRe.FindAllString(Prepare(src), -1) {
		if !strings.HasPrefix(span, "{{") {
			continue
		}
		inner := strings.Trim(span[2:len(span)-2], "- \t\n")
		parts := splitFilters(inner)
		key := identRe.FindString(strings.TrimSpace(parts[0]))
		if key == "" {
			continue
		}
		u := Use{Key: key}
		for _, f := range parts[1:] {
			f = strings.TrimSpace(f)
			if strings.HasPrefix(f, "default:") {
				u.HasFallback = true
				lit := strings.TrimSpace(strings.TrimPrefix(f, "default:"))
				if isStringLiteral(lit) {
					u.Fallback = lit[1 : len(lit)-1]
				} else {
					u.Fallback = lit
				}
			}
		}
		if i, ok := idx[key]; ok {
			out[i].Count++
			if !out[i].HasFallback && u.HasFallback {
				out[i].HasFallback, out[i].Fallback = true, u.Fallback
			}
			continue
		}
		u.Count = 1
		idx[key] = len(out)
		out = append(out, u)
	}
	return out
}

// Vars builds the variables of one recipient: contact fields, custom fields
// (attributes) and the per-recipient links. Contact fields are also
// available as contact.first_name.
func Vars(email, firstName, lastName string, attributes map[string]any, links map[string]string) map[string]any {
	contact := map[string]any{"email": email, "first_name": firstName, "last_name": lastName}
	vars := map[string]any{}
	keys := make([]string, 0, len(attributes))
	for k := range attributes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		contact[k] = attributes[k]
		vars[k] = attributes[k]
	}
	vars["email"], vars["first_name"], vars["last_name"] = email, firstName, lastName
	vars["contact"] = contact
	for k, v := range links {
		vars[k] = v
	}
	return vars
}
