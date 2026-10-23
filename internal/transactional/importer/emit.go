package importer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// scope is the context a Mustache block sets: the root, a loop item or an
// object ("with").
type scope struct {
	// expr is the Liquid expression for "this" in the scope ("" at the root).
	expr string
	// norm is the normalized data path of "this" ("items[]", "user").
	norm string
	loop bool
	// param and index are the block params naming the item and the loop
	// index ("as |item i|").
	param, index string
}

// emitter writes a tree as Liquid. It runs twice: once to collect how each
// value is used (so "if" knows a value is a list), once to write.
type emitter struct {
	d       Dialect
	u       *usage
	collect bool
	field   string
	b       strings.Builder
	notes   []Note
	scopes  []scope
	// guard counts the conditions and loops around the current position.
	guard int
	// loopNames are loop variable names chosen per block, so both passes
	// agree.
	loopNames map[*mBlock]string
	taken     map[string]bool
}

func (e *emitter) note(kind NoteKind, line int, construct string, t i18n.Text) {
	if e.collect {
		return
	}
	e.notes = append(e.notes, noteOf(kind, e.field, line, construct, t))
}

func (e *emitter) dialectName() string {
	if e.d == DialectPostmark {
		return "Mustachio"
	}
	return "Handlebars"
}

// sendgrid and mailgun say whether a provider's helpers apply. Handlebars
// files may come from either, so they get both.
func (e *emitter) sendgrid() bool { return e.d == DialectSendGrid || e.d == DialectHandlebarsFile }
func (e *emitter) mailgun() bool  { return e.d == DialectMailgun || e.d == DialectHandlebarsFile }

// convertMustache converts one field of a Mustachio or Handlebars template.
// Every field of a template is collected first (fields share variables),
// then written.
func convertMustacheFields(fields []fieldSrc, d Dialect, u *usage) ([]string, []Note) {
	type parsed struct {
		nodes []mnode
		probs []parseProblem
		pre   []Note
	}
	ps := make([]parsed, len(fields))
	loopNames := map[*mBlock]string{}
	for i, f := range fields {
		src, pre := preprocess(f.src, d, f.name)
		nodes, probs := parseTree(lex(src))
		ps[i] = parsed{nodes, probs, pre}
	}
	// Pass 1: usage.
	for i, f := range fields {
		e := &emitter{d: d, u: u, collect: true, field: f.name, scopes: []scope{{}}, loopNames: loopNames, taken: map[string]bool{}}
		e.nodes(ps[i].nodes)
	}
	// Loop names must not shadow a variable of the template.
	taken := map[string]bool{}
	for p := range u.paths {
		taken[rootOf(p)] = true
	}
	for b := range loopNames {
		delete(loopNames, b)
	}
	outs := make([]string, len(fields))
	notes := []Note{}
	for i, f := range fields {
		e := &emitter{d: d, u: u, field: f.name, scopes: []scope{{}}, loopNames: loopNames, taken: taken}
		e.notes = append(e.notes, ps[i].pre...)
		for _, p := range ps[i].probs {
			e.note(NoteManual, p.line, p.construct, p.text)
		}
		e.nodes(ps[i].nodes)
		outs[i] = e.b.String()
		notes = append(notes, e.notes...)
	}
	return outs, notes
}

type fieldSrc struct{ name, src string }

func (e *emitter) nodes(ns []mnode) {
	for _, n := range ns {
		switch n := n.(type) {
		case mText:
			e.b.WriteString(n.s)
		case mVar:
			e.variable(n.t)
		case mComment:
			e.comment(n.t)
		case mVerbatim:
			e.verbatim(n.t)
		case *mBlock:
			e.block(n)
		}
	}
}

// lines keeps the line count of a tag that spans lines, so the converted
// template's lines match the source's.
func lines(src string) string { return strings.Repeat("\n", strings.Count(src, "\n")) }

func outTag(t token, inner string) string {
	open, close := "{{ ", " }}"
	if t.trimL {
		open = "{{- "
	}
	if t.trimR {
		close = " -}}"
	}
	return open + inner + lines(t.src) + close
}

func stmtTag(t token, inner string) string {
	open, close := "{% ", " %}"
	if t.trimL {
		open = "{%- "
	}
	if t.trimR {
		close = " -%}"
	}
	return open + inner + lines(t.src) + close
}

func (e *emitter) comment(t token) {
	if t.body == "" {
		e.b.WriteString(lines(t.src))
		return
	}
	open, close := stmtTag(token{trimL: t.trimL}, "comment"), stmtTag(token{trimR: t.trimR}, "endcomment")
	body := strings.ReplaceAll(t.body, "{%", "{ %")
	// The body was trimmed: put back the newlines it lost.
	extra := strings.Count(t.src, "\n") - strings.Count(body, "\n")
	e.b.WriteString(open + body + strings.Repeat("\n", max(extra, 0)) + close)
}

func (e *emitter) verbatim(t token) {
	e.b.WriteString(t.src)
	switch t.kind {
	case tPartial:
		e.note(NoteManual, t.line, t.src, say("note.partial", i18n.Args{"dialect": e.dialectName(), "src": strings.TrimSpace(t.src)}))
	case tDelims:
		e.note(NoteManual, t.line, t.src, say("note.delimiters", i18n.Args{"src": t.src}))
	case tRawBlock:
		e.note(NoteManual, t.line, t.src, say("note.rawBlock", i18n.Args{"src": t.src}))
	case tText:
		if t.unclosed {
			e.note(NoteManual, t.line, firstLine(t.src), say("note.unclosed", i18n.Args{"src": firstLine(t.src)}))
		}
	}
}

func firstLine(s string) string {
	s, _, _ = strings.Cut(s, "\n")
	if len(s) > 40 {
		s = s[:40] + "…"
	}
	return s
}

// resolve turns a Mustache path into a Liquid expression and its
// normalized data path ("" for values with no data path, like forloop).
func (e *emitter) resolve(path string) (liq, norm string, ok bool, why i18n.Text) {
	base := len(e.scopes) - 1
	p := path
	switch {
	case strings.HasPrefix(p, "@root."):
		base, p = 0, strings.TrimPrefix(p, "@root.")
	case p == "@root":
		return "", "", false, say("why.root", nil)
	case strings.HasPrefix(p, "@../"):
		return "", "", false, say("why.outerLoop", i18n.Args{"path": path})
	}
	// A block param names a loop's item ("as |row|"): "row.v" is the item's v.
	if first, rest, _ := strings.Cut(p, "."); !strings.HasPrefix(p, ".") {
		for k := len(e.scopes) - 1; k > 0; k-- {
			if s := e.scopes[k]; s.param != "" && s.param == first {
				if rest == "" {
					return s.expr, s.norm, true, i18n.Text{}
				}
				base, p = k, rest
				break
			}
		}
	}
	for strings.HasPrefix(p, "../") {
		p = strings.TrimPrefix(p, "../")
		if base > 0 {
			base--
		}
	}
	if p == ".." {
		p = "."
		if base > 0 {
			base--
		}
	}
	sc := e.scopes[base]
	switch p {
	case "this", ".":
		p = ""
	default:
		p = strings.TrimPrefix(strings.TrimPrefix(p, "this."), "./")
	}
	// Data variables of loops. Liquid's forloop is always the innermost
	// loop, so @index of an outer loop has no twin.
	if strings.HasPrefix(p, "@") {
		target, innermost := -1, -1
		for k := base; k >= 0 && target < 0; k-- {
			if e.scopes[k].loop {
				target = k
			}
		}
		for k := len(e.scopes) - 1; k >= 0 && innermost < 0; k-- {
			if e.scopes[k].loop {
				innermost = k
			}
		}
		if target < 0 {
			return "", "", false, say("why.outsideLoop", i18n.Args{"path": path})
		}
		if target != innermost {
			return "", "", false, say("why.outerLoop", i18n.Args{"path": path})
		}
		switch p {
		case "@index":
			return "forloop.index0", "", true, i18n.Text{}
		case "@first":
			return "forloop.first", "", true, i18n.Text{}
		case "@last":
			return "forloop.last", "", true, i18n.Text{}
		}
		return "", "", false, say("why.noEquivalent", i18n.Args{"path": path})
	}
	if sc.index != "" && p == sc.index {
		return "forloop.index0", "", true, i18n.Text{}
	}
	if p == "" {
		if base == 0 {
			return "", "", false, say("why.topLevel", i18n.Args{"path": path})
		}
		return sc.expr, sc.norm, true, i18n.Text{}
	}
	segs := strings.Split(p, ".")
	liqSegs := make([]string, 0, len(segs))
	normSegs := make([]string, 0, len(segs))
	for _, s := range segs {
		s = strings.TrimSuffix(strings.TrimPrefix(s, "["), "]")
		switch {
		case s == "":
			return "", "", false, say("why.notPath", i18n.Args{"path": path})
		case isNumber(s) && !strings.ContainsAny(s, ".-"):
			if len(liqSegs) == 0 {
				return "", "", false, say("why.notPath", i18n.Args{"path": path})
			}
			liqSegs[len(liqSegs)-1] += "[" + s + "]"
			normSegs = append(normSegs, "[]")
		case !validName(s):
			return "", "", false, say("why.notName", i18n.Args{"path": path})
		default:
			liqSegs = append(liqSegs, s)
			normSegs = append(normSegs, s)
		}
	}
	liq = strings.Join(liqSegs, ".")
	if sc.expr != "" {
		liq = sc.expr + "." + liq
	}
	norm = sc.norm
	for _, s := range normSegs {
		switch {
		case s == "[]":
			norm += "[]"
		case norm == "":
			norm = s
		default:
			norm += "." + s
		}
	}
	return liq, norm, true, i18n.Text{}
}

var nameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

func validName(s string) bool { return nameRe.MatchString(s) }

// useAt records a use of a normalized path.
func (e *emitter) useAt(norm string, line int, f func(*pathUse)) {
	if !e.collect || norm == "" {
		return
	}
	e.u.touch(norm, line, f)
}

var urlAttrTail = regexp.MustCompile(`(?i)\b(href|src|action|background)\s*=\s*["']?[^"'<>]*$`)

func (e *emitter) inURLAttr() bool {
	s := e.b.String()
	if len(s) > 300 {
		s = s[len(s)-300:]
	}
	return urlAttrTail.MatchString(s)
}

// variable writes {{x}}, {{{x}}} and the inline helpers.
func (e *emitter) variable(t token) {
	if t.body == "" {
		e.b.WriteString(t.src)
		e.note(NoteManual, t.line, t.src, say("note.empty", i18n.Args{"src": t.src}))
		return
	}
	if mapped, ok := e.placeholder(t); ok {
		e.b.WriteString(outTag(t, mapped))
		return
	}
	head, args, _, ok := parseCall(t.body)
	if !ok {
		e.keep(t, say("note.unreadable", i18n.Args{"src": t.src}))
		return
	}
	if len(args) > 0 {
		e.inlineHelper(t, head, args)
		return
	}
	switch hx, _ := parseArg(head); hx.kind {
	case eString:
		e.b.WriteString(hx.s + lines(t.src))
		return
	case eNumber, eBool:
		e.b.WriteString(hx.s + lines(t.src))
		return
	case eNull:
		e.b.WriteString(lines(t.src))
		return
	}
	liq, norm, ok, why := e.resolve(head)
	if !ok {
		e.keep(t, why)
		return
	}
	inURL := e.inURLAttr()
	bare := e.guard == 0
	e.useAt(norm, t.line, func(p *pathUse) {
		p.printed = true
		p.bare = p.bare || bare
		p.inURL = p.inURL || inURL
	})
	e.b.WriteString(outTag(t, liq))
}

// keep writes a construct as it was and asks for a manual fix: the draft
// shows a Liquid error on that line until someone rewrites it.
func (e *emitter) keep(t token, msg i18n.Text) {
	e.b.WriteString(t.src)
	e.note(NoteManual, t.line, t.src, msg)
}

// placeholder maps the providers' own placeholders: Postmark's
// pm:unsubscribe, SendGrid's unsubscribe groups.
func (e *emitter) placeholder(t token) (string, bool) {
	b := strings.TrimSpace(t.body)
	switch {
	case e.d == DialectPostmark && strings.EqualFold(b, "pm:unsubscribe"):
		e.note(NoteChange, t.line, t.src, say("note.unsubscribe", i18n.Args{"provider": "Postmark", "src": t.src}))
		return "unsubscribe_url", true
	case e.sendgrid() && b == "unsubscribe" && len(e.scopes) == 1:
		e.note(NoteChange, t.line, t.src, say("note.unsubscribe", i18n.Args{"provider": "SendGrid", "src": t.src}))
		return "unsubscribe_url", true
	case e.sendgrid() && b == "unsubscribe_preferences" && len(e.scopes) == 1:
		e.note(NoteChange, t.line, t.src, say("note.preferences", i18n.Args{"src": t.src}))
		return "preferences_url", true
	case e.d == DialectPostmark && b == "@content":
		return "", false
	}
	return "", false
}

// inlineHelper writes {{helper args}}: SendGrid's formatDate, insert and
// length convert; anything else needs a manual fix.
func (e *emitter) inlineHelper(t token, head string, args []expr) {
	switch {
	case e.sendgrid() && head == "formatDate" && (len(args) == 2 || len(args) == 3) && args[0].kind == ePath && args[1].kind == eString:
		liq, norm, ok, why := e.resolve(args[0].s)
		if !ok {
			e.keep(t, why)
			return
		}
		f, unknown := momentToStrftime(args[1].s)
		bare := e.guard == 0
		e.useAt(norm, t.line, func(p *pathUse) { p.printed, p.date, p.bare = true, true, p.bare || bare })
		e.b.WriteString(outTag(t, fmt.Sprintf("%s | date: %s", liq, quote(f))))
		na := i18n.Args{"src": t.src, "out": outTag(token{}, fmt.Sprintf("%s | date: %s", liq, quote(f))), "offset": "none", "unknown": "none"}
		if len(args) == 3 {
			na["offset"] = quote(args[2].s)
		}
		if len(unknown) > 0 {
			na["unknown"] = strings.Join(unknown, ", ")
		}
		e.note(NoteChange, t.line, t.src, say("note.date", na))
	case e.sendgrid() && head == "insert" && len(args) >= 1 && args[0].kind == ePath:
		liq, norm, ok, why := e.resolve(args[0].s)
		if !ok {
			e.keep(t, why)
			return
		}
		def, hasDef := "", false
		for _, a := range args[1:] {
			if a.kind == eString && strings.HasPrefix(a.s, "default=") {
				def, hasDef = strings.TrimPrefix(a.s, "default="), true
			}
		}
		bare := e.guard == 0
		e.useAt(norm, t.line, func(p *pathUse) {
			p.printed = true
			if hasDef {
				p.hasDef, p.def = true, def
			} else {
				p.bare = p.bare || bare
			}
		})
		out := liq
		if hasDef {
			out = fmt.Sprintf("%s | default: %s", liq, quote(def))
		}
		e.b.WriteString(outTag(t, out))
		if hasDef {
			e.note(NoteChange, t.line, t.src, say("note.became", i18n.Args{"src": t.src, "out": outTag(token{}, out)}))
		}
	case e.sendgrid() && head == "length" && len(args) == 1 && args[0].kind == ePath:
		liq, norm, ok, why := e.resolve(args[0].s)
		if !ok {
			e.keep(t, why)
			return
		}
		e.useAt(norm, t.line, func(p *pathUse) { p.iterated = true })
		e.b.WriteString(outTag(t, liq+" | size"))
	default:
		e.keep(t, say("note.helper", i18n.Args{"dialect": e.dialectName(), "name": head}))
	}
}

func quote(s string) string {
	if !strings.Contains(s, `"`) {
		return `"` + s + `"`
	}
	if !strings.Contains(s, `'`) {
		return `'` + s + `'`
	}
	return `"` + strings.ReplaceAll(s, `"`, `”`) + `"`
}

// Block helpers.

var comparisons = map[string]string{
	"equals":      "==",
	"equal":       "==",
	"notEquals":   "!=",
	"greaterThan": ">",
	"lessThan":    "<",
}

func (e *emitter) block(b *mBlock) {
	head, args, params, ok := parseCall(b.open.body)
	if !ok {
		e.keepBlock(b, say("note.unreadable", i18n.Args{"src": b.open.src}))
		return
	}
	if b.inverted {
		e.section(b, head, args, true)
		return
	}
	switch {
	case head == "if" || head == "unless":
		e.ifBlock(b, head, args)
	case head == "each":
		e.eachBlock(b, args, params)
	case head == "with":
		if len(args) != 1 || args[0].kind != ePath {
			e.keepBlock(b, say("note.oneValue", i18n.Args{"src": b.open.src}))
			return
		}
		e.withBlock(b, args[0].s, false)
	case comparisons[head] != "" && (head != "equal" || e.mailgun()) && (head == "equal" || e.sendgrid()):
		e.compareBlock(b, head, args)
	case (head == "and" || head == "or") && e.sendgrid():
		e.logicBlock(b, head, args)
	case len(args) == 0 && validPath(head):
		e.section(b, head, nil, false)
	default:
		e.keepBlock(b, say("note.helper", i18n.Args{"dialect": e.dialectName(), "name": head}))
	}
}

func validPath(s string) bool {
	if s == "" || strings.ContainsAny(s, "\"'()=") {
		return false
	}
	return true
}

// keepBlock writes a block's own tags as they were (a manual fix) and still
// converts what is inside.
func (e *emitter) keepBlock(b *mBlock, msg i18n.Text) {
	e.note(NoteManual, b.open.line, b.open.src, msg)
	e.b.WriteString(b.open.src)
	e.guard++
	for i, br := range b.branches {
		if i > 0 && br.elseTok != nil {
			e.b.WriteString(br.elseTok.src)
		}
		e.nodes(br.nodes)
	}
	e.guard--
	if b.close != nil {
		e.b.WriteString(b.close.src)
	}
}

// truthy writes a Mustache condition in Liquid: Mustache and Handlebars
// skip empty text and empty lists, Liquid only skips missing values and
// false.
func (e *emitter) truthy(liq, norm string) string {
	if norm == "" {
		return liq
	}
	switch e.u.typeOf(norm) {
	case TypeList:
		return liq + ".size > 0"
	case TypeBool, TypeObject:
		return liq
	case TypeNumber:
		return liq + " and " + liq + " != 0"
	}
	return liq + ` and ` + liq + ` != ""`
}

// condition converts one condition operand.
func (e *emitter) condition(a expr, line int) (string, bool, i18n.Text) {
	switch a.kind {
	case eString, eNumber, eBool:
		if a.kind == eString {
			return quote(a.s), true, i18n.Text{}
		}
		return a.s, true, i18n.Text{}
	case eNull:
		return "nil", true, i18n.Text{}
	case eSub:
		// (length items) is the one subexpression with a Liquid twin.
		if len(a.sub) == 2 && a.sub[0].kind == ePath && a.sub[0].s == "length" && a.sub[1].kind == ePath && e.sendgrid() {
			liq, norm, ok, why := e.resolve(a.sub[1].s)
			if !ok {
				return "", false, why
			}
			e.useAt(norm, line, func(p *pathUse) { p.iterated = true; p.cond = true })
			return liq + ".size", true, i18n.Text{}
		}
		return "", false, say("why.subexpression", nil)
	case ePath:
		liq, norm, ok, why := e.resolve(a.s)
		if !ok {
			return "", false, why
		}
		e.useAt(norm, line, func(p *pathUse) { p.cond = true })
		return liq, true, i18n.Text{}
	}
	return "", false, say("why.hash", nil)
}

func (e *emitter) operandPath(a expr) string {
	if a.kind != ePath {
		return ""
	}
	_, norm, ok, _ := e.resolve(a.s)
	if !ok {
		return ""
	}
	return norm
}

func (e *emitter) ifBlock(b *mBlock, head string, args []expr) {
	if len(args) != 1 {
		e.keepBlock(b, say("note.oneValue", i18n.Args{"src": b.open.src}))
		return
	}
	cond, ok, why := e.condition(args[0], b.open.line)
	if !ok {
		e.keepBlock(b, why)
		return
	}
	cond = e.truthy(cond, e.operandPath(args[0]))
	tag := head
	e.b.WriteString(stmtTag(b.open, tag+" "+cond))
	e.branches(b, "end"+tag, head)
}

// branches writes a block's branches with {% else %} or {% elsif %}.
func (e *emitter) branches(b *mBlock, end, kind string) {
	e.guard++
	for i, br := range b.branches {
		if i > 0 {
			t := *br.elseTok
			rest := strings.TrimSpace(t.body)
			switch {
			case rest == "":
				e.b.WriteString(stmtTag(t, "else"))
			case kind == "if" && strings.HasPrefix(rest, "if "):
				_, args, _, ok := parseCall(rest)
				if !ok || len(args) != 1 {
					e.keep(t, say("note.oneValue", i18n.Args{"src": t.src}))
					break
				}
				cond, ok, why := e.condition(args[0], t.line)
				if !ok {
					e.keep(t, why)
					break
				}
				e.b.WriteString(stmtTag(t, "elsif "+e.truthy(cond, e.operandPath(args[0]))))
			default:
				e.keep(t, say("note.noEquivalentHere", i18n.Args{"src": t.src}))
			}
		}
		e.nodes(br.nodes)
	}
	e.guard--
	if b.close != nil {
		e.b.WriteString(stmtTag(*b.close, end))
	}
}

func (e *emitter) compareBlock(b *mBlock, head string, args []expr) {
	if len(args) != 2 {
		e.keepBlock(b, say("note.twoValues", i18n.Args{"src": b.open.src}))
		return
	}
	l, ok, why := e.condition(args[0], b.open.line)
	if !ok {
		e.keepBlock(b, why)
		return
	}
	r, ok, why := e.condition(args[1], b.open.line)
	if !ok {
		e.keepBlock(b, why)
		return
	}
	op := comparisons[head]
	if op == ">" || op == "<" {
		for _, a := range args {
			if p := e.operandPath(a); p != "" {
				e.useAt(p, b.open.line, func(p *pathUse) { p.number = true })
			}
		}
	}
	e.b.WriteString(stmtTag(b.open, fmt.Sprintf("if %s %s %s", l, op, r)))
	if head == "equal" {
		e.note(NoteChange, b.open.line, b.open.src, say("note.mailgunEqual", i18n.Args{"src": b.open.src, "left": l, "right": r}))
	}
	e.branches(b, "endif", "compare")
}

func (e *emitter) logicBlock(b *mBlock, head string, args []expr) {
	if len(args) < 2 {
		e.keepBlock(b, say("note.atLeastTwo", i18n.Args{"src": b.open.src}))
		return
	}
	parts := make([]string, 0, len(args))
	plain := true
	for _, a := range args {
		c, ok, why := e.condition(a, b.open.line)
		if !ok {
			e.keepBlock(b, why)
			return
		}
		t := e.truthy(c, e.operandPath(a))
		if t != c {
			plain = false
		}
		parts = append(parts, t)
	}
	if head == "or" && !plain {
		// Liquid has no parentheses: "a and a != '' or b" wouldn't group as
		// written, so "or" checks presence only.
		raw := make([]string, 0, len(args))
		for _, a := range args {
			c, _, _ := e.condition(a, b.open.line)
			raw = append(raw, c)
		}
		cond := strings.Join(raw, " or ")
		e.b.WriteString(stmtTag(b.open, "if "+cond))
		e.note(NoteChange, b.open.line, b.open.src, say("note.emptyPasses", i18n.Args{"src": b.open.src, "cond": cond}))
		e.branches(b, "endif", "logic")
		return
	}
	e.b.WriteString(stmtTag(b.open, "if "+strings.Join(parts, " "+head+" ")))
	e.branches(b, "endif", "logic")
}

// singular names a loop's item after its collection: items → item.
func singular(name string) string {
	irregular := map[string]string{"people": "person", "children": "child", "men": "man", "women": "woman", "data": "entry", "news": "story", "media": "medium"}
	if s, ok := irregular[name]; ok {
		return s
	}
	switch {
	case strings.HasSuffix(name, "ies") && len(name) > 4:
		return strings.TrimSuffix(name, "ies") + "y"
	case strings.HasSuffix(name, "sses"), strings.HasSuffix(name, "shes"), strings.HasSuffix(name, "ches"), strings.HasSuffix(name, "xes"):
		return strings.TrimSuffix(name, "es")
	case strings.HasSuffix(name, "ss"):
		return ""
	case strings.HasSuffix(name, "s") && len(name) > 2:
		return strings.TrimSuffix(name, "s")
	}
	return ""
}

// loopName picks the item name for a loop over coll: the block param, else
// the singular of the collection, else "item", never a name the template
// already uses.
func (e *emitter) loopName(b *mBlock, coll string, params []string) string {
	if n, ok := e.loopNames[b]; ok {
		return n
	}
	name := ""
	if len(params) > 0 && validName(params[0]) {
		name = params[0]
	} else {
		name = singular(lastSegment(coll))
		if name == "" || !validName(name) {
			name = "item"
		}
	}
	if !e.collect {
		// Sibling loops may share a name; a loop inside another, or a
		// variable of the template, may not.
		base := name
		for n := 2; e.taken[name] || e.inScope(name); n++ {
			name = fmt.Sprintf("%s%d", base, n)
		}
	}
	e.loopNames[b] = name
	return name
}

func (e *emitter) inScope(name string) bool {
	for _, s := range e.scopes {
		if s.expr == name {
			return true
		}
	}
	return false
}

func (e *emitter) eachBlock(b *mBlock, args []expr, params []string) {
	if len(args) != 1 || args[0].kind != ePath {
		e.keepBlock(b, say("note.oneList", i18n.Args{"src": b.open.src}))
		return
	}
	coll, norm, ok, why := e.resolve(args[0].s)
	if !ok {
		e.keepBlock(b, why)
		return
	}
	bare := e.guard == 0
	e.useAt(norm, b.open.line, func(p *pathUse) { p.iterated = true; p.bare = p.bare || bare })
	name := e.loopName(b, args[0].s, params)
	open := stmtTag(b.open, fmt.Sprintf("for %s in %s", name, coll))
	e.b.WriteString(open)
	e.note(NoteChange, b.open.line, b.open.src, say("note.became", i18n.Args{"src": b.open.src, "out": stmtTag(token{}, fmt.Sprintf("for %s in %s", name, coll))}))
	sc := scope{expr: name, norm: norm + "[]", loop: true}
	if len(params) > 0 {
		sc.param = params[0]
	}
	if len(params) > 1 {
		sc.index = params[1]
	}
	e.scopes = append(e.scopes, sc)
	e.guard++
	e.nodes(b.branches[0].nodes)
	e.scopes = e.scopes[:len(e.scopes)-1]
	for _, br := range b.branches[1:] {
		t := *br.elseTok
		if strings.TrimSpace(t.body) != "" {
			e.keep(t, say("note.noEquivalentInLoop", i18n.Args{"src": t.src}))
		} else {
			e.b.WriteString(stmtTag(t, "else"))
		}
		e.nodes(br.nodes)
	}
	e.guard--
	if b.close != nil {
		e.b.WriteString(stmtTag(*b.close, "endfor"))
	}
}

// withBlock: {{#with user}} and Mustache sections on an object render when
// the value is there, with names inside read from it.
func (e *emitter) withBlock(b *mBlock, path string, section bool) {
	liq, norm, ok, why := e.resolve(path)
	if !ok {
		e.keepBlock(b, why)
		return
	}
	e.useAt(norm, b.open.line, func(p *pathUse) { p.cond = true })
	cond := e.truthy(liq, norm)
	e.b.WriteString(stmtTag(b.open, "if "+cond))
	if section {
		e.note(NoteChange, b.open.line, b.open.src, say("note.section", i18n.Args{"src": b.open.src, "cond": cond, "liq": liq}))
	} else {
		e.note(NoteChange, b.open.line, b.open.src, say("note.with", i18n.Args{"src": b.open.src, "cond": cond, "liq": liq}))
	}
	e.scopes = append(e.scopes, scope{expr: liq, norm: norm})
	e.guard++
	e.nodes(b.branches[0].nodes)
	e.scopes = e.scopes[:len(e.scopes)-1]
	for _, br := range b.branches[1:] {
		t := *br.elseTok
		if strings.TrimSpace(t.body) != "" {
			e.keep(t, say("note.noEquivalentHere", i18n.Args{"src": t.src}))
		} else {
			e.b.WriteString(stmtTag(t, "else"))
		}
		e.nodes(br.nodes)
	}
	e.guard--
	if b.close != nil {
		e.b.WriteString(stmtTag(*b.close, "endif"))
	}
}

// section is a Mustache section {{#name}} (a loop for a list, otherwise a
// "with" that also checks the value) or an inverted section {{^name}}.
func (e *emitter) section(b *mBlock, head string, args []expr, inverted bool) {
	if len(args) > 0 || !validPath(head) {
		e.keepBlock(b, say("note.helper", i18n.Args{"dialect": e.dialectName(), "name": head}))
		return
	}
	liq, norm, ok, why := e.resolve(head)
	if !ok {
		e.keepBlock(b, why)
		return
	}
	if inverted {
		e.useAt(norm, b.open.line, func(p *pathUse) { p.cond = true })
		e.b.WriteString(stmtTag(b.open, "unless "+e.truthy(liq, norm)))
		e.branches(b, "endunless", "unless")
		return
	}
	if norm != "" && e.u.typeOf(norm) == TypeList && !e.collect || e.collect && e.u.has(norm) && e.u.paths[norm].iterated {
		e.eachBlock(b, []expr{{kind: ePath, s: head}}, nil)
		return
	}
	e.withBlock(b, head, true)
}

// preprocess maps placeholders written outside {{ }}: Mailgun's
// %recipient.x% and unsubscribe variables, SendGrid's unsubscribe tags. The
// replacements never span lines, so line numbers stay the same.
func preprocess(src string, d Dialect, field string) (string, []Note) {
	notes := []Note{}
	type rule struct {
		re   *regexp.Regexp
		repl func(m []string) string
		msg  func(m []string, repl string) (NoteKind, i18n.Text)
	}
	changed := func(provider string) func(m []string, repl string) (NoteKind, i18n.Text) {
		return func(m []string, repl string) (NoteKind, i18n.Text) {
			return NoteChange, say("note.providerBecame", i18n.Args{"provider": provider, "src": m[0], "out": strings.ReplaceAll(strings.ReplaceAll(repl, "{{", "{{ "), "}}", " }}")})
		}
	}
	var rules []rule
	if d == DialectMailgun || d == DialectHandlebarsFile {
		rules = append(rules,
			// The recipient's address and name are to.email and to.name.
			rule{mailgunTo, func(m []string) string {
				if m[1] == "_name" {
					return "{{to.name}}"
				}
				return "{{to.email}}"
			}, changed("Mailgun")},
			rule{mailgunRecipient, func(m []string) string { return "{{" + m[1] + "}}" }, changed("Mailgun")},
			rule{mailgunUnsub, func([]string) string { return "{{unsubscribe_url}}" }, changed("Mailgun")},
			rule{mailgunOther, func(m []string) string { return m[0] }, func(m []string, _ string) (NoteKind, i18n.Text) {
				return NoteManual, say("note.mailgunOther", i18n.Args{"src": m[0]})
			}},
		)
	}
	if d == DialectSendGrid || d == DialectHandlebarsFile {
		rules = append(rules, rule{sendgridUnsub, func(m []string) string {
			if strings.Contains(m[0], "preferences") {
				return "{{preferences_url}}"
			}
			return "{{unsubscribe_url}}"
		}, changed("SendGrid")})
	}
	for _, r := range rules {
		var b strings.Builder
		last := 0
		for _, loc := range r.re.FindAllStringSubmatchIndex(src, -1) {
			m := make([]string, len(loc)/2)
			for k := range m {
				if loc[2*k] >= 0 {
					m[k] = src[loc[2*k]:loc[2*k+1]]
				}
			}
			repl := r.repl(m)
			kind, msg := r.msg(m, repl)
			notes = append(notes, noteOf(kind, field, 1+strings.Count(src[:loc[0]], "\n"), m[0], msg))
			b.WriteString(src[last:loc[0]])
			b.WriteString(repl)
			last = loc[1]
		}
		b.WriteString(src[last:])
		src = b.String()
	}
	return src, notes
}

var (
	mailgunTo        = regexp.MustCompile(`%recipient(_email|_name)?%`)
	mailgunRecipient = regexp.MustCompile(`%recipient\.([A-Za-z_][A-Za-z0-9_]*)%`)
	mailgunUnsub     = regexp.MustCompile(`%(unsubscribe_url|tag_unsubscribe_url|mailing_list_unsubscribe_url)%`)
	mailgunOther     = regexp.MustCompile(`%recipient(_[a-z]+)?%`)
	sendgridUnsub    = regexp.MustCompile(`<%asm_(group_unsubscribe|global_unsubscribe|preferences)_raw_url%>`)
)
