package importer

import (
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The Mustache family (Postmark's Mustachio, Handlebars as SendGrid and
// Mailgun run it) is read into a small tree, then written out as Liquid.

type tokKind int

const (
	tText tokKind = iota
	tVar
	tOpen     // {{#name …}}
	tInverted // {{^name}}
	tClose    // {{/name}}
	tElse     // {{else}}, {{else if x}}, {{^}}
	tComment
	tPartial  // {{> name}}
	tDelims   // {{=<% %>=}}
	tRawBlock // {{{{raw}}}}
)

type token struct {
	kind tokKind
	// src is the exact source text: the text, or the whole tag.
	src string
	// body is the tag's content without delimiters, sigil and ~.
	body string
	// raw: {{{x}}} or {{& x}} (no HTML escaping at the source).
	raw          bool
	line         int
	trimL, trimR bool
	// unclosed: "{{" without "}}".
	unclosed bool
}

// lex splits a template into text and tags.
func lex(src string) []token {
	out := []token{}
	line := 1
	i := 0
	textStart := 0
	flushText := func(end int) {
		if end > textStart {
			out = append(out, token{kind: tText, src: src[textStart:end], line: line - strings.Count(src[textStart:end], "\n")})
		}
	}
	for i < len(src) {
		j := strings.Index(src[i:], "{{")
		if j < 0 {
			break
		}
		start := i + j
		// Count lines up to the tag.
		line += strings.Count(src[i:start], "\n")
		flushText(start)
		tok, end := lexTag(src, start)
		tok.line = line
		out = append(out, tok)
		line += strings.Count(src[start:end], "\n")
		i = end
		textStart = end
	}
	line += strings.Count(src[i:], "\n")
	if textStart < len(src) {
		out = append(out, token{kind: tText, src: src[textStart:], line: line - strings.Count(src[textStart:], "\n")})
	}
	return out
}

// lexTag reads the tag starting at src[start] ("{{") and returns it with the
// offset after it.
func lexTag(src string, start int) (token, int) {
	if strings.HasPrefix(src[start:], "{{{{") {
		end := strings.Index(src[start:], "}}}}")
		if end < 0 {
			return token{kind: tRawBlock, src: src[start:], unclosed: true}, len(src)
		}
		end += start + 4
		return token{kind: tRawBlock, src: src[start:end], body: strings.TrimSpace(src[start+4 : end-4])}, end
	}
	p := start + 2
	t := token{}
	if p < len(src) && src[p] == '~' {
		t.trimL = true
		p++
	}
	// Triple braces: {{{x}}} or {{~{x}~}}.
	if p < len(src) && src[p] == '{' {
		end := strings.Index(src[p:], "}}}")
		if end < 0 {
			return token{kind: tText, src: src[start:], unclosed: true}, len(src)
		}
		inner := src[p+1 : p+end]
		stop := p + end + 3
		if strings.HasSuffix(inner, "}~") {
			// {{~{x}~}}: the "~" sits before the last two braces.
			inner = strings.TrimSuffix(inner, "}~")
			t.trimR = true
		}
		t.kind, t.raw, t.body, t.src = tVar, true, strings.TrimSpace(inner), src[start:stop]
		return t, stop
	}
	// Comments may contain "}}": {{!-- … --}}.
	if strings.HasPrefix(src[p:], "!--") {
		end := strings.Index(src[p:], "--}}")
		endTrim := strings.Index(src[p:], "--~}}")
		switch {
		case end < 0 && endTrim < 0:
			return token{kind: tText, src: src[start:], unclosed: true}, len(src)
		case endTrim >= 0 && (end < 0 || endTrim < end):
			stop := p + endTrim + 5
			return token{kind: tComment, src: src[start:stop], body: strings.TrimSpace(src[p+3 : p+endTrim]), trimL: t.trimL, trimR: true}, stop
		}
		stop := p + end + 4
		return token{kind: tComment, src: src[start:stop], body: strings.TrimSpace(src[p+3 : p+end]), trimL: t.trimL}, stop
	}
	end := strings.Index(src[p:], "}}")
	if end < 0 {
		return token{kind: tText, src: src[start:], unclosed: true}, len(src)
	}
	inner := src[p : p+end]
	stop := p + end + 2
	if strings.HasSuffix(inner, "~") {
		t.trimR = true
		inner = strings.TrimSuffix(inner, "~")
	}
	t.src = src[start:stop]
	body := strings.TrimSpace(inner)
	switch {
	case body == "":
		t.kind = tVar
	case body[0] == '#':
		t.kind, t.body = tOpen, strings.TrimSpace(body[1:])
		// Handlebars decorators and inline partials ({{#*inline}}) aren't
		// helpers: keep them for a manual fix.
	case body == "^":
		t.kind = tElse
	case body[0] == '^':
		t.kind, t.body = tInverted, strings.TrimSpace(body[1:])
	case body[0] == '/':
		t.kind, t.body = tClose, strings.TrimSpace(body[1:])
	case body[0] == '!':
		t.kind, t.body = tComment, strings.TrimSpace(body[1:])
	case body[0] == '>':
		t.kind, t.body = tPartial, strings.TrimSpace(body[1:])
	case body[0] == '=' && strings.HasSuffix(body, "="):
		t.kind, t.body = tDelims, body
	case body[0] == '&':
		t.kind, t.raw, t.body = tVar, true, strings.TrimSpace(body[1:])
	case body == "else" || strings.HasPrefix(body, "else "):
		t.kind, t.body = tElse, strings.TrimSpace(strings.TrimPrefix(body, "else"))
	default:
		t.kind, t.body = tVar, body
	}
	return t, stop
}

// Tree nodes.
type mnode interface{ isNode() }

type mText struct{ s string }
type mVar struct{ t token }
type mComment struct{ t token }
type mVerbatim struct{ t token } // partials, delimiters, raw blocks, strays

type mBranch struct {
	// elseTok is nil for the first branch; "else if x" has a condition.
	elseTok *token
	nodes   []mnode
}

type mBlock struct {
	open     token
	inverted bool
	branches []mBranch
	close    *token
}

func (mText) isNode()     {}
func (mVar) isNode()      {}
func (mComment) isNode()  {}
func (mVerbatim) isNode() {}
func (*mBlock) isNode()   {}

// parseProblem is a structure error with the line it is on.
type parseProblem struct {
	line      int
	construct string
	text      i18n.Text
}

// parseTree builds the tree. Structure problems (a close that doesn't match,
// a block never closed) are returned with their line; the tree keeps what
// could be read so the rest still converts.
func parseTree(toks []token) ([]mnode, []parseProblem) {
	root := &mBlock{branches: []mBranch{{}}}
	stack := []*mBlock{root}
	probs := []parseProblem{}
	cur := func() *mBranch {
		b := stack[len(stack)-1]
		return &b.branches[len(b.branches)-1]
	}
	for _, t := range toks {
		switch t.kind {
		case tText:
			if t.unclosed {
				cur().nodes = append(cur().nodes, mVerbatim{t: t})
				continue
			}
			cur().nodes = append(cur().nodes, mText{s: t.src})
		case tVar:
			cur().nodes = append(cur().nodes, mVar{t: t})
		case tComment:
			cur().nodes = append(cur().nodes, mComment{t: t})
		case tOpen, tInverted:
			b := &mBlock{open: t, inverted: t.kind == tInverted, branches: []mBranch{{}}}
			cur().nodes = append(cur().nodes, b)
			stack = append(stack, b)
		case tElse:
			if len(stack) == 1 {
				probs = append(probs, parseProblem{line: t.line, construct: t.src, text: say("note.elseOutside", i18n.Args{"src": t.src})})
				cur().nodes = append(cur().nodes, mVerbatim{t: t})
				continue
			}
			tt := t
			b := stack[len(stack)-1]
			b.branches = append(b.branches, mBranch{elseTok: &tt})
		case tClose:
			name := blockName(t.body)
			// Find the block it closes.
			at := -1
			for k := len(stack) - 1; k >= 1; k-- {
				if blockName(stack[k].open.body) == name {
					at = k
					break
				}
			}
			if at < 0 {
				probs = append(probs, parseProblem{line: t.line, construct: t.src, text: say("note.closesUnopened", i18n.Args{"src": t.src})})
				cur().nodes = append(cur().nodes, mVerbatim{t: t})
				continue
			}
			for k := len(stack) - 1; k > at; k-- {
				o := stack[k].open
				probs = append(probs, parseProblem{line: o.line, construct: o.src, text: say("note.neverClosed", i18n.Args{"src": o.src})})
			}
			tt := t
			stack[at].close = &tt
			stack = stack[:at]
		default:
			cur().nodes = append(cur().nodes, mVerbatim{t: t})
		}
	}
	for k := len(stack) - 1; k >= 1; k-- {
		o := stack[k].open
		probs = append(probs, parseProblem{line: o.line, construct: o.src, text: say("note.neverClosed", i18n.Args{"src": o.src})})
	}
	return root.branches[0].nodes, probs
}

// blockName is the helper or path that opens a block ("each" in
// "each items", "user" in "user").
func blockName(body string) string {
	if f := strings.Fields(body); len(f) > 0 {
		return f[0]
	}
	return ""
}

// Expressions inside tags.

type exprKind int

const (
	ePath exprKind = iota
	eString
	eNumber
	eBool
	eNull
	eSub  // (helper args…)
	eHash // key=value
)

type expr struct {
	kind exprKind
	s    string // the path, literal text or hash key
	sub  []expr // a subexpression's parts, or a hash value
}

// splitArgs splits a tag body on spaces outside quotes and parentheses.
func splitArgs(s string) ([]string, bool) {
	out := []string{}
	var q byte
	depth := 0
	start := -1
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case q != 0:
			if c == '\\' && i+1 < len(s) {
				i++
				continue
			}
			if c == q {
				q = 0
			}
		case c == '"' || c == '\'':
			if start < 0 {
				start = i
			}
			q = c
		case c == '(':
			if start < 0 {
				start = i
			}
			depth++
		case c == ')':
			depth--
			if depth < 0 {
				return nil, false
			}
		case (c == ' ' || c == '\t' || c == '\n' || c == '\r') && depth == 0:
			if start >= 0 {
				out = append(out, s[start:i])
				start = -1
			}
			continue
		default:
			if start < 0 {
				start = i
			}
		}
	}
	if q != 0 || depth != 0 {
		return nil, false
	}
	if start >= 0 {
		out = append(out, s[start:])
	}
	return out, true
}

func parseArg(a string) (expr, bool) {
	switch {
	case a == "":
		return expr{}, false
	case a[0] == '"' || a[0] == '\'':
		if len(a) < 2 || a[len(a)-1] != a[0] {
			return expr{}, false
		}
		return expr{kind: eString, s: a[1 : len(a)-1]}, true
	case a[0] == '(':
		if a[len(a)-1] != ')' {
			return expr{}, false
		}
		parts, ok := splitArgs(a[1 : len(a)-1])
		if !ok || len(parts) == 0 {
			return expr{}, false
		}
		sub := make([]expr, 0, len(parts))
		for _, p := range parts {
			e, ok := parseArg(p)
			if !ok {
				return expr{}, false
			}
			sub = append(sub, e)
		}
		return expr{kind: eSub, sub: sub}, true
	case a == "true" || a == "false":
		return expr{kind: eBool, s: a}, true
	case a == "null" || a == "undefined":
		return expr{kind: eNull, s: a}, true
	case isNumber(a):
		return expr{kind: eNumber, s: a}, true
	}
	if k, v, ok := strings.Cut(a, "="); ok && k != "" && !strings.ContainsAny(k, ".[/") {
		val, ok := parseArg(v)
		if !ok {
			return expr{}, false
		}
		return expr{kind: eHash, s: k, sub: []expr{val}}, true
	}
	return expr{kind: ePath, s: a}, true
}

func isNumber(s string) bool {
	if s == "" {
		return false
	}
	dot := false
	for i, c := range s {
		switch {
		case c == '-' && i == 0 && len(s) > 1:
		case c == '.' && !dot && i > 0:
			dot = true
		case c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return true
}

// parseCall reads "helper arg arg key=value as |a b|".
func parseCall(body string) (head string, args []expr, params []string, ok bool) {
	// Block params: "items as |item idx|".
	if i := strings.Index(body, " as |"); i >= 0 && strings.HasSuffix(strings.TrimSpace(body), "|") {
		inner := strings.TrimSuffix(strings.TrimSpace(body[i+len(" as |"):]), "|")
		params = strings.Fields(inner)
		body = body[:i]
	}
	parts, ok := splitArgs(body)
	if !ok || len(parts) == 0 {
		return "", nil, params, false
	}
	head = parts[0]
	for _, p := range parts[1:] {
		e, ok := parseArg(p)
		if !ok {
			return "", nil, params, false
		}
		args = append(args, e)
	}
	return head, args, params, true
}
