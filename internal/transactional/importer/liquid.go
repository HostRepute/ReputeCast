package importer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hostrepute/reputecast/internal/liquid"
)

// Liquid sources (Notifuse, files written in Liquid) mostly carry over:
// ReputeCast renders Liquid with the same engine Notifuse uses. What
// changes is what ReputeCast doesn't offer: includes and renders, filters
// and tags it doesn't have, and (for transactional mail) contact fields.

var (
	liquidSpan = regexp.MustCompile(`\{\{-?[\s\S]*?-?\}\}|\{%-?[\s\S]*?-?%\}`)
	tagName    = regexp.MustCompile(`^\{%-?\s*([A-Za-z_]\w*)`)
	pathRe     = regexp.MustCompile(`(^|[^\w.'"\]])([A-Za-z_][\w-]*(?:\.[A-Za-z_][\w-]*|\[\d+\])*)`)
	wholePath  = regexp.MustCompile(`^[A-Za-z_][\w-]*(?:\.[A-Za-z_][\w-]*|\[\d+\])*$`)
	stringLit  = regexp.MustCompile(`"[^"]*"|'[^']*'`)
	numericCmp = regexp.MustCompile(`[<>]=?\s*-?\d|-?\d+\s*[<>]`)
	indexRe    = regexp.MustCompile(`\[\d+\]`)
)

// knownTags are the tags ReputeCast's Liquid has (include and render are
// switched off: templates can't read files).
var knownTags = map[string]bool{
	"if": true, "elsif": true, "else": true, "endif": true, "unless": true, "endunless": true,
	"case": true, "when": true, "endcase": true, "for": true, "endfor": true, "break": true, "continue": true,
	"cycle": true, "assign": true, "capture": true, "endcapture": true, "comment": true, "endcomment": true,
	"raw": true, "endraw": true, "increment": true, "decrement": true, "tablerow": true, "endtablerow": true,
}

// knownFilters are Liquid's standard filters.
var knownFilters = map[string]bool{
	"abs": true, "append": true, "at_least": true, "at_most": true, "capitalize": true, "ceil": true, "compact": true,
	"concat": true, "date": true, "default": true, "divided_by": true, "downcase": true, "escape": true, "escape_once": true,
	"first": true, "floor": true, "join": true, "last": true, "lstrip": true, "map": true, "minus": true, "modulo": true,
	"newline_to_br": true, "plus": true, "prepend": true, "remove": true, "remove_first": true, "replace": true,
	"replace_first": true, "reverse": true, "round": true, "rstrip": true, "size": true, "slice": true, "sort": true,
	"sort_natural": true, "split": true, "strip": true, "strip_html": true, "strip_newlines": true, "times": true,
	"truncate": true, "truncatewords": true, "uniq": true, "upcase": true, "url_decode": true, "url_encode": true,
	"where": true, "json": true, "inspect": true, "sum": true, "find": true, "reject": true, "has": true, "base64_encode": true,
	"base64_decode": true, "cgi_escape": true, "uri_escape": true, "slugify": true, "to_integer": true, "squish": true,
	"pluralize": true, "number_with_delimiter": true, "xml_escape": true, "where_exp": true, "group_by": true,
	"sample": true, "array_to_sentence_string": true, "normalize_whitespace": true, "date_to_xmlschema": true,
	"date_to_rfc822": true, "date_to_string": true, "date_to_long_string": true,
}

// keywords that look like paths inside tags.
var liquidWords = map[string]bool{
	"and": true, "or": true, "not": true, "contains": true, "in": true, "true": true, "false": true, "nil": true,
	"null": true, "empty": true, "blank": true, "with": true, "as": true, "for": true, "reversed": true, "limit": true,
	"offset": true, "if": true, "elsif": true, "unless": true, "case": true, "when": true, "assign": true,
	"capture": true, "cycle": true, "increment": true, "decrement": true, "tablerow": true, "cols": true,
	"else": true, "forloop": true, "tablerowloop": true,
}

func convertLiquidFields(fields []fieldSrc, opt Options, u *usage) ([]string, []Note) {
	outs := make([]string, len(fields))
	notes := []Note{}
	// Locals first: names a template assigns are never variables.
	for _, f := range fields {
		for _, span := range liquidSpan.FindAllString(f.src, -1) {
			m := tagName.FindStringSubmatch(span)
			if m == nil {
				continue
			}
			switch m[1] {
			case "assign", "capture", "increment", "decrement":
				rest := strings.TrimSpace(strings.TrimPrefix(strings.Trim(span, "{%-} \t\n"), m[1]))
				if name := pathRe.FindStringSubmatch(" " + rest); name != nil {
					u.locals[rootOf(name[2])] = true
				}
			}
		}
	}
	for i, f := range fields {
		out, ns := convertLiquidField(f, opt, u)
		outs[i] = out
		notes = append(notes, ns...)
	}
	return outs, notes
}

type liquidLoop struct {
	name string
	norm string
}

func convertLiquidField(f fieldSrc, opt Options, u *usage) (string, []Note) {
	src := f.src
	notes := []Note{}
	if src == "" {
		return "", notes
	}
	note := func(kind NoteKind, line int, construct, msg string) {
		notes = append(notes, Note{Kind: kind, Field: f.name, Line: line, Construct: construct, Message: msg})
	}
	var b strings.Builder
	last := 0
	guard := 0
	loops := []liquidLoop{}
	inRaw, inComment := false, false
	for _, loc := range liquidSpan.FindAllStringIndex(src, -1) {
		span := src[loc[0]:loc[1]]
		line := 1 + strings.Count(src[:loc[0]], "\n")
		b.WriteString(src[last:loc[0]])
		last = loc[1]
		name := ""
		if m := tagName.FindStringSubmatch(span); m != nil {
			name = m[1]
		}
		// Raw and comment blocks are copied as they are.
		switch {
		case inRaw:
			if name == "endraw" {
				inRaw = false
			}
			b.WriteString(span)
			continue
		case inComment:
			if name == "endcomment" {
				inComment = false
			}
			b.WriteString(span)
			continue
		case name == "raw":
			inRaw = true
			b.WriteString(span)
			continue
		case name == "comment":
			inComment = true
			b.WriteString(span)
			continue
		}
		inURL := urlAttrTail.MatchString(tail(b.String(), 300))
		if strings.HasPrefix(span, "{%") {
			switch {
			case name == "include" || name == "render":
				b.WriteString(span)
				note(NoteManual, line, span, fmt.Sprintf("`{%% %s %%}` has no ReputeCast equivalent: templates can't include other files. Use a saved section.", name))
				continue
			case name != "" && !knownTags[name]:
				b.WriteString(span)
				note(NoteManual, line, span, fmt.Sprintf("the Liquid tag `%s` isn't available in ReputeCast.", name))
				continue
			}
		}
		out, changes := rewriteLiquidSpan(span, opt)
		for _, c := range changes {
			note(c.kind, line, c.construct, c.message)
		}
		// Filters ReputeCast doesn't have.
		for _, fl := range filtersOf(out) {
			if !knownFilters[fl] {
				note(NoteManual, line, span, fmt.Sprintf("the Liquid filter `%s` isn't available in ReputeCast.", fl))
			}
		}
		b.WriteString(out)
		// Usage.
		inner := strings.Trim(out, "{}%- \t\r\n")
		switch {
		case strings.HasPrefix(out, "{{"):
			expr, filters, _ := strings.Cut(inner, "|")
			for _, p := range liquidPaths(expr) {
				norm := normLiquid(p, loops)
				isDate := strings.Contains(filters, "date")
				bare := guard == 0
				u.touch(norm, line, func(pu *pathUse) {
					pu.printed = true
					pu.inURL = pu.inURL || inURL
					pu.date = pu.date || isDate
					if d, ok := defaultOf(filters); ok {
						pu.hasDef, pu.def = true, d
					} else {
						pu.bare = pu.bare || bare
					}
				})
			}
		case name == "for" || name == "tablerow":
			rest := strings.TrimSpace(strings.TrimPrefix(inner, name))
			v, coll, ok := strings.Cut(rest, " in ")
			norm := ""
			if ok {
				coll = strings.Fields(coll + " ")[0]
				if wholePath.MatchString(coll) {
					norm = normLiquid(coll, loops)
					bare := guard == 0
					u.touch(norm, line, func(pu *pathUse) { pu.iterated = true; pu.bare = pu.bare || bare })
				}
			}
			// Every for pushes a loop (a range loops too), so endfor pops the
			// right one.
			loopNorm := ""
			if norm != "" {
				loopNorm = norm + "[]"
			}
			loops = append(loops, liquidLoop{name: strings.TrimSpace(v), norm: loopNorm})
			guard++
		case name == "endfor" || name == "endtablerow":
			if len(loops) > 0 {
				loops = loops[:len(loops)-1]
			}
			guard--
		case name == "if" || name == "unless" || name == "elsif" || name == "case" || name == "when":
			rest := strings.TrimPrefix(inner, name)
			numeric := numericCmp.MatchString(rest)
			for _, p := range liquidPaths(rest) {
				norm := normLiquid(p, loops)
				u.touch(norm, line, func(pu *pathUse) { pu.cond = true; pu.number = pu.number || numeric })
			}
			if name == "if" || name == "unless" || name == "case" {
				guard++
			}
		case name == "endif" || name == "endunless" || name == "endcase":
			guard--
		case name == "assign":
			if _, rhs, ok := strings.Cut(inner, "="); ok {
				expr, _, _ := strings.Cut(rhs, "|")
				for _, p := range liquidPaths(expr) {
					u.touch(normLiquid(p, loops), line, func(pu *pathUse) { pu.printed = true })
				}
			}
		}
	}
	b.WriteString(src[last:])
	out := b.String()
	// The whole field must parse; a line that already needs a fix isn't
	// reported twice.
	flagged := map[int]bool{}
	for _, n := range notes {
		if n.Kind == NoteManual {
			flagged[n.Line] = true
		}
	}
	for _, p := range liquid.Validate(out) {
		if flagged[p.Line] {
			continue
		}
		note(NoteManual, p.Line, "", LiquidMessage(p))
	}
	return out, notes
}

var codeTail = regexp.MustCompile(` in (\{%[\s\S]*?%\}|\{\{[\s\S]*?\}\})\.?$`)

// LiquidMessage turns a Liquid problem into a report sentence without its
// "Line N:" (the report shows the line) and with the tag as code.
func LiquidMessage(p liquid.LiquidProblem) string {
	msg := strings.TrimPrefix(p.Message, fmt.Sprintf("Line %d: ", p.Line))
	msg = codeTail.ReplaceAllString(msg, " in `$1`.")
	return lowerFirst(msg)
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func tail(s string, n int) string {
	if len(s) > n {
		return s[len(s)-n:]
	}
	return s
}

// liquidPaths lists the variable paths of an expression: strings,
// keywords and numbers are left out.
func liquidPaths(expr string) []string {
	expr = stringLit.ReplaceAllString(expr, `""`)
	out := []string{}
	for _, m := range pathRe.FindAllStringSubmatch(" "+expr, -1) {
		p := m[2]
		if liquidWords[rootOf(p)] || liquidWords[p] {
			continue
		}
		out = append(out, p)
	}
	return out
}

// normLiquid maps a Liquid path to a data path, through the loops around it:
// "item.price" inside "for item in items" is "items[].price".
func normLiquid(p string, loops []liquidLoop) string {
	p = indexRe.ReplaceAllString(p, "[]")
	p = strings.TrimSuffix(strings.TrimSuffix(p, ".size"), ".first")
	for k := len(loops) - 1; k >= 0; k-- {
		l := loops[k]
		if p == l.name || strings.HasPrefix(p, l.name+".") {
			if l.norm == "" {
				return ""
			}
			return l.norm + p[len(l.name):]
		}
	}
	if strings.HasPrefix(p, "forloop") {
		return ""
	}
	return p
}

var defaultRe = regexp.MustCompile(`default:\s*("([^"]*)"|'([^']*)'|[\w.]+)`)

func defaultOf(filters string) (string, bool) {
	m := defaultRe.FindStringSubmatch(filters)
	if m == nil {
		return "", false
	}
	switch {
	case m[2] != "" || strings.HasPrefix(m[1], `"`):
		return m[2], true
	case m[3] != "" || strings.HasPrefix(m[1], `'`):
		return m[3], true
	}
	return m[1], true
}

var filterRe = regexp.MustCompile(`\|\s*([A-Za-z_]\w*)`)

func filtersOf(span string) []string {
	s := stringLit.ReplaceAllString(span, `""`)
	out := []string{}
	for _, m := range filterRe.FindAllStringSubmatch(s, -1) {
		out = append(out, m[1])
	}
	return out
}

type liquidChange struct {
	kind      NoteKind
	construct string
	message   string
}

var (
	contactEmail = regexp.MustCompile(`\bcontact\.email\b`)
	contactField = regexp.MustCompile(`\bcontact\.([A-Za-z_]\w*)`)
	listField    = regexp.MustCompile(`\blist\.(name|id)\b`)
)

// rewriteLiquidSpan maps Notifuse's names to ReputeCast's in one tag.
func rewriteLiquidSpan(span string, opt Options) (string, []liquidChange) {
	changes := []liquidChange{}
	if opt.Dialect != DialectNotifuse {
		return span, changes
	}
	out := span
	if strings.Contains(out, "notification_center_url") {
		out = strings.ReplaceAll(out, "notification_center_url", "preferences_url")
		changes = append(changes, liquidChange{NoteChange, "notification_center_url", "Notifuse's `notification_center_url` became `preferences_url`."})
	}
	if opt.Transactional {
		// The contact's address is the recipient's.
		if contactEmail.MatchString(stringLit.ReplaceAllString(out, `""`)) {
			out = contactEmail.ReplaceAllString(out, "to.email")
			changes = append(changes, liquidChange{NoteChange, "contact.email", "`contact.email` became `to.email`, the recipient's address."})
		}
		if contactField.MatchString(stringLit.ReplaceAllString(out, `""`)) {
			for _, m := range contactField.FindAllStringSubmatch(out, -1) {
				changes = append(changes, liquidChange{NoteChange, m[0], fmt.Sprintf("`%s` became `%s`: transactional mail has no contact profile, so the call sends it in its data.", m[0], m[1])})
			}
			out = contactField.ReplaceAllString(out, "$1")
		}
		if strings.Contains(out, "confirm_subscription_url") {
			changes = append(changes, liquidChange{NoteManual, "confirm_subscription_url", "`confirm_subscription_url` belongs to Notifuse's double opt-in emails. ReputeCast sends its own confirmations."})
		}
		if m := listField.FindString(out); m != "" {
			changes = append(changes, liquidChange{NoteManual, m, fmt.Sprintf("`%s` has no value in transactional mail: it isn't sent to a list.", m)})
		}
	}
	return out, changes
}
