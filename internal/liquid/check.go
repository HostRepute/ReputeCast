package liquid

import (
	"regexp"
	"strings"
)

// Issue kinds (Issue.Kind).
const (
	// IssueSyntax: the template doesn't parse, so the email can't be
	// rendered for anyone.
	IssueSyntax = "syntax"
	// IssueUnknownField: the field doesn't exist, so every recipient gets
	// its fallback, or nothing.
	IssueUnknownField = "unknown_field"
)

// Issue is a problem with an email's personalization (Check).
type Issue struct {
	Kind string `json:"kind" enum:"syntax,unknown_field"`
	// Line and Message are a syntax error's line and the parser's words
	// ("Line 3: unterminated tag…"), in English.
	Line    int    `json:"line,omitempty"`
	Message string `json:"message,omitempty"`
	// Field is an unknown field ("frist_name"); HasFallback and Fallback
	// say what it prints instead ("there").
	Field       string `json:"field,omitempty"`
	HasFallback bool   `json:"hasFallback,omitempty"`
	Fallback    string `json:"fallback,omitempty"`
}

var (
	assignRe = regexp.MustCompile(`\{%-?\s*(?:assign|capture)\s+([A-Za-z_][\w-]*)`)
	forRe    = regexp.MustCompile(`\{%-?\s*(?:for|tablerow)\s+([A-Za-z_][\w-]*)\s+in\b`)
)

// Check finds what goes wrong when src is rendered (C8): syntax errors,
// which stop the email from rendering at all, and fields that don't exist,
// flagged even when they have a fallback, since {{ frist_name | default:
// "there" }} greets everyone with "there". fields are the fields that exist
// (Catalog keys); contact.*, the system links, loop and assigned variables
// and the per-recipient poll and survey links are always known. With no
// fields, only the syntax is checked.
func Check(src string, fields []string) []Issue {
	out := []Issue{}
	if !strings.Contains(src, "{{") && !strings.Contains(src, "{%") {
		return out
	}
	for _, p := range Validate(src) {
		out = append(out, Issue{Kind: IssueSyntax, Line: p.Line, Message: p.Message})
	}
	if len(out) > 0 || len(fields) == 0 {
		// A template that doesn't parse has no reliable variables.
		return out
	}
	known := map[string]bool{"contact": true, "forloop": true, "tablerowloop": true}
	for _, f := range fields {
		known[f] = true
	}
	for _, f := range SystemFields {
		known[f.Key] = true
	}
	prepared := Prepare(src)
	for _, re := range []*regexp.Regexp{assignRe, forRe} {
		for _, m := range re.FindAllStringSubmatch(prepared, -1) {
			known[m[1]] = true
		}
	}
	for _, u := range Uses(src) {
		root := strings.SplitN(u.Key, ".", 2)[0]
		if known[u.Key] || known[root] || strings.HasPrefix(root, "rc_poll_") || strings.HasPrefix(root, "rc_survey_") {
			continue
		}
		out = append(out, Issue{Kind: IssueUnknownField, Field: u.Key, HasFallback: u.HasFallback, Fallback: u.Fallback})
	}
	return out
}
