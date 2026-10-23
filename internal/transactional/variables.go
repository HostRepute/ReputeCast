package transactional

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Variable types (brief §3.1).
const (
	TypeText    = "text"
	TypeNumber  = "number"
	TypeBoolean = "boolean"
	TypeDate    = "date"
	TypeURL     = "url"
	TypeList    = "list"
	TypeObject  = "object"
)

// TransactionalVariable is a variable a template declares. Sample is what
// the editor and the code samples fill in; Default is printed when an
// optional variable isn't sent. For list and object variables both are
// JSON text.
type TransactionalVariable struct {
	Name        string `json:"name" minLength:"1" maxLength:"64" pattern:"^[A-Za-z_][A-Za-z0-9_]*$" example:"reset_url"`
	Type        string `json:"type" enum:"text,number,boolean,date,url,list,object"`
	Required    bool   `json:"required"`
	Sample      string `json:"sample,omitempty" maxLength:"4000"`
	Default     string `json:"default,omitempty" maxLength:"4000"`
	Sensitive   bool   `json:"sensitive" doc:"Never stored: logs and stored content show ••••••"`
	Description string `json:"description,omitempty" maxLength:"300"`
}

// TransactionalTestData is a named JSON sample of a template's variables,
// used by the preview, the checks and test sends.
type TransactionalTestData struct {
	Name string         `json:"name" minLength:"1" maxLength:"80" example:"New user"`
	Data map[string]any `json:"data"`
}

// Reserved names: what ReputeCast fills in itself. to.email and to.name are
// the recipient; Optional categories get the category's unsubscribe link
// and the preference center.
var reserved = []string{"to", "unsubscribe_url", "preferences_url", "forloop", "tablerowloop"}

var nameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func parseVariables(raw json.RawMessage) []TransactionalVariable {
	out := []TransactionalVariable{}
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = []TransactionalVariable{}
	}
	return out
}

func parseTestData(raw json.RawMessage) []TransactionalTestData {
	out := []TransactionalTestData{}
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		out = []TransactionalTestData{}
	}
	for i := range out {
		if out[i].Data == nil {
			out[i].Data = map[string]any{}
		}
	}
	return out
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`null`)
	}
	return b
}

// validateVariables checks declarations: names, types, unique names, and
// sample and default values that parse for their type. Its problems are in
// the reader's language.
func validateVariables(ctx context.Context, vars []TransactionalVariable) error {
	seen := map[string]bool{}
	for i, v := range vars {
		loc := fmt.Sprintf("body.variables[%d]", i)
		if !nameRe.MatchString(v.Name) {
			return problemInvalid(loc+".name", say(ctx, "variable.badName", i18n.Args{"name": v.Name}))
		}
		if slices.Contains(reserved, v.Name) {
			return problemInvalid(loc+".name", say(ctx, "variable.reserved", i18n.Args{"name": v.Name}))
		}
		if seen[v.Name] {
			return problemInvalid(loc+".name", say(ctx, "variable.twice", i18n.Args{"name": v.Name}))
		}
		seen[v.Name] = true
		switch v.Type {
		case TypeText, TypeNumber, TypeBoolean, TypeDate, TypeURL, TypeList, TypeObject:
		default:
			return problemInvalid(loc+".type", say(ctx, "variable.unknownType", i18n.Args{"name": v.Name, "type": strconv.Quote(v.Type)}))
		}
		for _, f := range []struct{ field, value string }{{"sample", v.Sample}, {"default", v.Default}} {
			if f.value == "" {
				continue
			}
			if _, err := typedValue(v.Type, f.value); err != nil {
				return problemInvalid(loc+"."+f.field, say(ctx, "variable.badValue",
					i18n.Args{"field": f.field, "name": v.Name, "type": v.Type, "problem": problemText(lang(ctx), err)}))
			}
		}
	}
	return nil
}

// typedValue turns a sample or default (always text in the editor) into
// the value the template sees.
func typedValue(typ, s string) (any, error) {
	switch typ {
	case TypeNumber:
		var n json.Number
		if err := json.Unmarshal([]byte(strings.TrimSpace(s)), &n); err != nil {
			return nil, &valueError{key: "value.number", value: strconv.Quote(s)}
		}
		if f, err := n.Float64(); err == nil {
			return f, nil
		}
		return s, nil
	case TypeBoolean:
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "true", "yes", "1":
			return true, nil
		case "false", "no", "0":
			return false, nil
		}
		return nil, &valueError{key: "value.boolean", value: strconv.Quote(s)}
	case TypeList:
		var v []any
		if err := json.Unmarshal([]byte(s), &v); err != nil {
			return nil, &valueError{key: "value.list"}
		}
		return v, nil
	case TypeObject:
		var v map[string]any
		if err := json.Unmarshal([]byte(s), &v); err != nil {
			return nil, &valueError{key: "value.object"}
		}
		return v, nil
	}
	return s, nil
}

// sampleData builds data from the declared samples.
func sampleData(vars []TransactionalVariable) map[string]any {
	out := map[string]any{}
	for _, v := range vars {
		if v.Sample == "" {
			continue
		}
		if val, err := typedValue(v.Type, v.Sample); err == nil {
			out[v.Name] = val
		}
	}
	return out
}

// missingRequired lists the required variables data doesn't carry. A null
// counts as missing; an empty string is the app's choice and counts as
// sent.
func missingRequired(vars []TransactionalVariable, data map[string]any) []string {
	out := []string{}
	for _, v := range vars {
		if !v.Required {
			continue
		}
		if val, ok := data[v.Name]; !ok || val == nil {
			out = append(out, v.Name)
		}
	}
	return out
}

// sensitiveNames lists the variables marked sensitive.
func sensitiveNames(vars []TransactionalVariable) []string {
	out := []string{}
	for _, v := range vars {
		if v.Sensitive {
			out = append(out, v.Name)
		}
	}
	return out
}

// maskData replaces sensitive values with ••••••. Undeclared keys are kept
// as sent (they render nothing).
func maskData(vars []TransactionalVariable, data map[string]any) map[string]any {
	out := make(map[string]any, len(data))
	for k, v := range data {
		out[k] = v
	}
	for _, v := range vars {
		if !v.Sensitive {
			continue
		}
		if _, ok := out[v.Name]; ok {
			out[v.Name] = Mask
		}
	}
	return out
}

// dataKeys are the variable names a call sent, sorted.
func dataKeys(data map[string]any) []string {
	out := make([]string, 0, len(data))
	for k, v := range data {
		if v != nil {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// VariableUse is a variable a template prints or tests, with the first line
// it appears on (1-based, in the source it was found in).
type VariableUse struct {
	Name   string `json:"name"`
	Where  string `json:"where" enum:"subject,preheader,body,text"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

var (
	spanRe     = regexp.MustCompile(`\{\{-?[\s\S]*?-?\}\}|\{%-?[\s\S]*?-?%\}`)
	tokenRe    = regexp.MustCompile(`"[^"]*"|'[^']*'|[A-Za-z_][\w-]*(?:\.[A-Za-z_][\w-]*|\[[^\]]*\])*|\|\s*[A-Za-z_]\w*|\S`)
	keywordSet = map[string]bool{
		"and": true, "or": true, "not": true, "contains": true, "true": true, "false": true, "nil": true, "null": true,
		"empty": true, "blank": true, "in": true, "with": true, "limit": true, "offset": true, "reversed": true, "else": true,
		"cols": true, "by": true, "as": true, "first": true, "last": true, "size": true,
	}
	// Tags whose words are not variables.
	inertTags = map[string]bool{"else": true, "endif": true, "endfor": true, "endunless": true, "endcase": true, "endcapture": true,
		"comment": true, "endcomment": true, "raw": true, "endraw": true, "break": true, "continue": true, "endtablerow": true, "liquid": true}
)

// usedVariables lists the variables a Liquid source reads, by root name,
// in order of first use. Loop variables, captured and assigned names are
// local and left out.
func usedVariables(src, where string) []VariableUse {
	out := []VariableUse{}
	seen := map[string]bool{}
	local := map[string]bool{}
	for _, m := range spanRe.FindAllStringIndex(src, -1) {
		span := src[m[0]:m[1]]
		tag := strings.HasPrefix(span, "{%")
		inner := strings.Trim(span[2:len(span)-2], "-")
		inner = strings.TrimSpace(htmlUnescape(inner))
		toks := tokenRe.FindAllString(inner, -1)
		if len(toks) == 0 {
			continue
		}
		start := 0
		if tag {
			name := toks[0]
			if inertTags[name] {
				continue
			}
			start = 1
			switch name {
			case "for", "tablerow":
				// {% for item in items %}: item is local.
				if len(toks) > 1 {
					local[rootOf(toks[1])] = true
					start = 2
				}
			case "assign":
				if len(toks) > 1 {
					local[rootOf(toks[1])] = true
					start = 2
				}
			case "capture":
				if len(toks) > 1 {
					local[rootOf(toks[1])] = true
				}
				continue
			case "cycle", "include", "render":
				continue
			}
		}
		afterPipe := false
		for _, tok := range toks[start:] {
			switch {
			case strings.HasPrefix(tok, "|"):
				afterPipe = true // the filter name follows; its arguments may be variables
				continue
			case tok == ":" || tok == ",":
				afterPipe = false
				continue
			case afterPipe:
				afterPipe = false
				continue
			case strings.HasPrefix(tok, `"`) || strings.HasPrefix(tok, `'`):
				continue
			}
			if !nameStart(tok) {
				continue
			}
			root := rootOf(tok)
			if keywordSet[root] || local[root] || seen[root] {
				continue
			}
			seen[root] = true
			line, col := lineCol(src, m[0])
			out = append(out, VariableUse{Name: root, Where: where, Line: line, Column: col})
		}
	}
	return out
}

func nameStart(tok string) bool {
	c := tok[0]
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func rootOf(tok string) string {
	if i := strings.IndexAny(tok, ".["); i > 0 {
		return tok[:i]
	}
	return tok
}

func lineCol(src string, at int) (int, int) {
	line := strings.Count(src[:at], "\n") + 1
	col := at - strings.LastIndexByte(src[:at], '\n')
	return line, col
}

var htmlEntities = strings.NewReplacer("&quot;", `"`, "&#34;", `"`, "&#39;", "'", "&apos;", "'", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&nbsp;", " ")

func htmlUnescape(s string) string { return htmlEntities.Replace(s) }

// templateUses lists the variables of every part of a template, merged by
// name (first appearance wins).
func templateUses(subject, preheader, body, text string) []VariableUse {
	out := []VariableUse{}
	seen := map[string]bool{}
	for _, part := range []struct{ src, where string }{{subject, "subject"}, {preheader, "preheader"}, {body, "body"}, {text, "text"}} {
		for _, u := range usedVariables(part.src, part.where) {
			if seen[u.Name] {
				continue
			}
			seen[u.Name] = true
			out = append(out, u)
		}
	}
	return out
}

func problemInvalid(location, message string) *Problem {
	p := problem(422, "validation_failed", message)
	p.Errors = append(p.Errors, invalidDetail(location, message))
	return p
}
