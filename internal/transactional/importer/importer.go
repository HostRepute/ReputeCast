// Package importer brings transactional templates from other providers into
// ReputeCast (add-on spec §3.8, §5.9): Postmark (Mustachio), SendGrid dynamic
// templates and Mailgun (Handlebars), Notifuse (Liquid), and MJML or HTML
// files. Each template is converted to ReputeCast Liquid, its variables are
// declared from what it uses, and a report says, per template, whether it
// converted cleanly, converted with notes, or needs a manual fix, naming the
// line and the construct. Imported templates only ever become drafts.
//
// The converters are pure functions (Convert); the API clients that read the
// providers (Postmark, SendGrid, Mailgun, Notifuse) dial through the SSRF
// guard like the contact importers.
package importer

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/hostrepute/reputecast/internal/mjml"
)

// Syntax is the template language of a source.
type Syntax string

// Template languages the converter reads.
const (
	Mustachio  Syntax = "mustachio"  // Postmark
	Handlebars Syntax = "handlebars" // SendGrid dynamic templates, Mailgun
	Liquid     Syntax = "liquid"     // Notifuse, and files written for ReputeCast
)

// Dialect refines a syntax with the helpers and placeholders of one provider.
type Dialect string

// Dialects, one per source.
const (
	DialectPostmark Dialect = "postmark"
	DialectSendGrid Dialect = "sendgrid"
	DialectMailgun  Dialect = "mailgun"
	DialectNotifuse Dialect = "notifuse"
	// DialectFile is an MJML or HTML file written in Liquid (or with plain
	// {{ name }} tokens, which read the same in every language).
	DialectFile Dialect = "file"
	// DialectHandlebarsFile is a file written in Handlebars, exported from
	// SendGrid or Mailgun: both providers' helpers apply.
	DialectHandlebarsFile Dialect = "handlebars-file"
)

// SyntaxOf returns the language a dialect is written in.
func SyntaxOf(d Dialect) Syntax {
	switch d {
	case DialectPostmark:
		return Mustachio
	case DialectSendGrid, DialectMailgun, DialectHandlebarsFile:
		return Handlebars
	}
	return Liquid
}

// Format is how the template's body is stored.
type Format string

// Body formats.
const (
	FormatHTML Format = "html"
	FormatMJML Format = "mjml"
)

// Source is one template as the provider (or a file) has it.
type Source struct {
	// Ref is the template's id or alias at the source ("password-reset",
	// "d-8f2c…", "welcome_email", "receipt.mjml").
	Ref string
	// Alias is the provider's human key when it has one (Postmark alias,
	// Mailgun name, Notifuse id); it seeds the ReputeCast ID.
	Alias     string
	Name      string
	Subject   string
	Preheader string
	Format    Format
	Body      string
	Text      string
	// Sample is test data from the source (SendGrid's and Notifuse's
	// test_data), used for samples and the first test data set.
	Sample map[string]any
	// Notes found while reading the source (a layout merged in, a visual
	// editor tree that couldn't be read).
	Notes []Note
	// Category is the source's own category (Notifuse), for the picker.
	Category string
}

// Status of a converted template.
type Status string

// Report statuses, as the conversion report shows them.
const (
	StatusConverted Status = "converted"
	StatusNotes     Status = "notes"
	StatusManual    Status = "manual"
)

// NoteKind says how much attention a note needs.
type NoteKind string

// Note kinds.
const (
	// NoteInfo is shown in the report but doesn't change the status
	// (variables declared, a renamed ID).
	NoteInfo NoteKind = "info"
	// NoteChange is a construct that converted with a visible change:
	// "{{#each items}} became {% for item in items %}".
	NoteChange NoteKind = "note"
	// NoteManual needs a person: the construct was kept as it was and the
	// draft shows a Liquid error until someone replaces it.
	NoteManual NoteKind = "manual"
)

// Note is one finding of the conversion. Text in backticks is code.
type Note struct {
	Kind NoteKind `json:"kind" enum:"info,note,manual"`
	// Field is where it is: subject, preheader, body or text.
	Field string `json:"field" enum:"subject,preheader,body,text,template"`
	// Line in the converted field (1-based), 0 when it isn't about a line.
	Line int `json:"line"`
	// Construct is the source construct, as written ("{{> footer}}").
	Construct string `json:"construct,omitempty"`
	// Message is a sentence; code is in backticks.
	Message string `json:"message"`
}

// Variable types, as the editor's Variables panel offers them.
const (
	TypeText   = "text"
	TypeNumber = "number"
	TypeBool   = "boolean"
	TypeDate   = "date"
	TypeURL    = "url"
	TypeList   = "list"
	TypeObject = "object"
)

// Variable is a variable the template uses, declared for ReputeCast.
type Variable struct {
	Name     string `json:"name"`
	Type     string `json:"type" enum:"text,number,boolean,date,url,list,object"`
	Required bool   `json:"required"`
	// Sample is a value that renders (from the source's test data when it
	// has one); Default is the fallback the template itself writes.
	Sample    any    `json:"sample,omitempty"`
	Default   string `json:"default,omitempty"`
	Sensitive bool   `json:"sensitive"`
}

// Converted is a template converted to ReputeCast Liquid.
type Converted struct {
	Ref       string `json:"ref"`
	Name      string `json:"name"`
	Subject   string `json:"subject"`
	Preheader string `json:"preheader"`
	Format    Format `json:"format"`
	// Body is the MJML or HTML source; HTML is what it compiles to (the
	// body itself for HTML templates).
	Body      string     `json:"body"`
	HTML      string     `json:"html"`
	Text      string     `json:"text"`
	Variables []Variable `json:"variables"`
	// TestData is the first named test data set ("Imported sample").
	TestData map[string]any `json:"testData"`
	Status   Status         `json:"status"`
	Notes    []Note         `json:"notes"`
}

// Options tune a conversion.
type Options struct {
	Dialect Dialect
	// Transactional is true for transactional drafts: Notifuse's contact
	// fields become plain variables, since transactional mail doesn't read
	// contact profiles. False keeps them for campaign templates.
	Transactional bool
}

// Convert turns a source template into ReputeCast Liquid with its variables
// and report.
func Convert(src Source, opt Options) Converted {
	out := Converted{Ref: src.Ref, Name: strings.TrimSpace(src.Name), Format: src.Format, Notes: []Note{}, Variables: []Variable{}}
	if out.Format == "" {
		out.Format = FormatHTML
	}
	out.Notes = append(out.Notes, src.Notes...)
	uses := newUsage()
	fields := []fieldSrc{{"subject", src.Subject}, {"preheader", src.Preheader}, {"body", src.Body}, {"text", src.Text}}
	var outs []string
	var notes []Note
	switch SyntaxOf(opt.Dialect) {
	case Mustachio, Handlebars:
		outs, notes = convertMustacheFields(fields, opt.Dialect, uses)
	default:
		outs, notes = convertLiquidFields(fields, opt, uses)
	}
	out.Notes = append(out.Notes, notes...)
	out.Subject, out.Preheader, out.Body, out.Text = outs[0], outs[1], outs[2], outs[3]
	out.HTML = out.Body
	if out.Format == FormatMJML {
		res := mjml.Compile(out.Body)
		out.HTML = res.HTML
		for _, p := range res.Errors {
			msg := strings.TrimPrefix(p.Message, fmt.Sprintf("Line %d: ", p.Line))
			out.Notes = append(out.Notes, Note{Kind: NoteManual, Field: "body", Line: p.Line, Message: "the MJML doesn't compile: " + lowerFirst(strings.TrimSuffix(msg, ".")) + "."})
		}
	}
	out.Variables = uses.declare(src.Sample)
	out.TestData = sampleData(out.Variables, src.Sample)
	if uses.has("view_in_browser_url") {
		out.Notes = append(out.Notes, Note{Kind: NoteManual, Field: "body", Line: uses.get("view_in_browser_url").firstLine, Construct: "view_in_browser_url",
			Message: "transactional mail has no web version, so `view_in_browser_url` prints nothing. Remove the link."})
	}
	out.Notes = append(out.Notes, out.renameInvalid()...)
	out.Notes = dedupeNotes(out.Notes)
	out.Status = statusOf(out.Notes)
	return out
}

// apiName is a variable name the API accepts in data.
var apiName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// renameInvalid gives variables whose names the API can't take (first-name,
// 2fa_code) a name it can, everywhere the template uses them.
func (c *Converted) renameInvalid() []Note {
	notes := []Note{}
	taken := map[string]bool{}
	for _, v := range c.Variables {
		taken[v.Name] = true
	}
	for i, v := range c.Variables {
		if apiName.MatchString(v.Name) {
			continue
		}
		name := sanitizeName(v.Name)
		for n := 2; taken[name]; n++ {
			name = fmt.Sprintf("%s_%d", sanitizeName(v.Name), n)
		}
		taken[name] = true
		c.Subject = renameVar(c.Subject, v.Name, name)
		c.Preheader = renameVar(c.Preheader, v.Name, name)
		c.Body = renameVar(c.Body, v.Name, name)
		c.HTML = renameVar(c.HTML, v.Name, name)
		c.Text = renameVar(c.Text, v.Name, name)
		if val, ok := c.TestData[v.Name]; ok {
			delete(c.TestData, v.Name)
			c.TestData[name] = val
		}
		c.Variables[i].Name = name
		notes = append(notes, Note{Kind: NoteChange, Field: "template", Construct: v.Name,
			Message: fmt.Sprintf("`%s` became `%s`: variable names use letters, digits and underscores, so your app sends `%s` in data.", v.Name, name, name)})
	}
	return notes
}

var notName = regexp.MustCompile(`[^A-Za-z0-9_]+`)

func sanitizeName(s string) string {
	out := strings.Trim(notName.ReplaceAllString(s, "_"), "_")
	if out == "" {
		return "value"
	}
	if out[0] >= '0' && out[0] <= '9' {
		out = "v_" + out
	}
	return out
}

var liquidBlock = regexp.MustCompile(`(?s)\{\{.*?\}\}|\{%.*?%\}`)

// renameVar renames a variable in the Liquid tags and outputs of src, as a
// whole name (not inside another name, a path or a quoted string).
func renameVar(src, from, to string) string {
	if !strings.Contains(src, from) {
		return src
	}
	ident := func(b byte) bool {
		return b == '_' || b == '-' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
	}
	return liquidBlock.ReplaceAllStringFunc(src, func(b string) string {
		var out strings.Builder
		i := 0
		for i < len(b) {
			j := strings.Index(b[i:], from)
			if j < 0 {
				out.WriteString(b[i:])
				break
			}
			j += i
			end := j + len(from)
			quotes := strings.Count(b[:j], `"`) + strings.Count(b[:j], `'`)
			if (j > 0 && (ident(b[j-1]) || b[j-1] == '.')) || (end < len(b) && ident(b[end])) || quotes%2 == 1 {
				out.WriteString(b[i : j+1])
				i = j + 1
				continue
			}
			out.WriteString(b[i:j])
			out.WriteString(to)
			i = end
		}
		return out.String()
	})
}

func statusOf(notes []Note) Status {
	st := StatusConverted
	for _, n := range notes {
		switch n.Kind {
		case NoteManual:
			return StatusManual
		case NoteChange:
			st = StatusNotes
		}
	}
	return st
}

// dedupeNotes keeps one note per construct and message, the first line it
// appears on (a helper used ten times is one fix).
func dedupeNotes(in []Note) []Note {
	out := make([]Note, 0, len(in))
	seen := map[string]bool{}
	for _, n := range in {
		k := string(n.Kind) + "\x00" + n.Construct + "\x00" + n.Message
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, n)
	}
	// Manual fixes first, then changes, then information; by field and line.
	rank := map[NoteKind]int{NoteManual: 0, NoteChange: 1, NoteInfo: 2}
	field := map[string]int{"template": 0, "subject": 1, "preheader": 2, "body": 3, "text": 4}
	slices.SortStableFunc(out, func(a, b Note) int {
		if rank[a.Kind] != rank[b.Kind] {
			return rank[a.Kind] - rank[b.Kind]
		}
		if field[a.Field] != field[b.Field] {
			return field[a.Field] - field[b.Field]
		}
		return a.Line - b.Line
	})
	return out
}

// Summary is the report's Notes cell: the first manual fix or note, or the
// declared variables ("2 variables declared: name, action_url").
func (c Converted) Summary() string {
	for _, n := range c.Notes {
		if n.Kind == NoteManual || n.Kind == NoteChange {
			return n.Message
		}
	}
	for _, n := range c.Notes {
		if n.Kind == NoteInfo && n.Field == "template" {
			return n.Message
		}
	}
	return VariablesLine(c.Variables)
}

// VariablesLine is "2 variables declared: name, action_url", or
// "4 variables declared" when listing them would be long.
func VariablesLine(vs []Variable) string {
	switch {
	case len(vs) == 0:
		return "No variables"
	case len(vs) == 1:
		return fmt.Sprintf("1 variable declared: %s", vs[0].Name)
	case len(vs) == 2:
		return fmt.Sprintf("2 variables declared: %s, %s", vs[0].Name, vs[1].Name)
	}
	return fmt.Sprintf("%d variables declared", len(vs))
}

var (
	idInvalid = regexp.MustCompile(`[^a-z0-9]+`)
	idTrim    = regexp.MustCompile(`^-+|-+$`)
)

// MaxIDLength is the longest template ID an import suggests.
const MaxIDLength = 64

// SuggestID turns a source alias or name into a ReputeCast template ID:
// lowercase letters, digits and hyphens.
func SuggestID(alias, name string) string {
	for _, s := range []string{alias, name} {
		id := idTrim.ReplaceAllString(idInvalid.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), "-"), "")
		if len(id) > MaxIDLength {
			id = strings.TrimRight(id[:MaxIDLength], "-")
		}
		if id != "" {
			return id
		}
	}
	return "template"
}

// UniqueID returns id, or id-2, id-3… when taken already holds it. The
// chosen ID is added to taken.
func UniqueID(id string, taken map[string]bool) string {
	out := id
	for n := 2; taken[out]; n++ {
		suffix := fmt.Sprintf("-%d", n)
		base := id
		if len(base)+len(suffix) > MaxIDLength {
			base = strings.TrimRight(base[:MaxIDLength-len(suffix)], "-")
		}
		out = base + suffix
	}
	taken[out] = true
	return out
}
