package i18n

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// The parsed form of an ICU MessageFormat string.
type node interface{ isNode() }

type textNode string

// argNode is {name} or {name, number|date|time[, style]}.
type argNode struct {
	name  string
	kind  string // "", "number", "date", "time"
	style string
	// inSentence is a plain argument that follows other words of its
	// sentence, in a Vietnamese translation (names.go).
	inSentence bool
}

// pluralNode is {name, plural|selectordinal, [offset:n] =0 {…} one {…} other {…}}.
type pluralNode struct {
	name    string
	ordinal bool
	offset  int
	options map[string][]node
}

// selectNode is {name, select, a {…} other {…}}.
type selectNode struct {
	name    string
	options map[string][]node
}

// poundNode is # inside a plural: the number, minus the offset.
type poundNode struct{}

// tagNode is <name>…</name>.
type tagNode struct {
	name     string
	children []node
}

func (textNode) isNode()   {}
func (argNode) isNode()    {}
func (pluralNode) isNode() {}
func (selectNode) isNode() {}
func (poundNode) isNode()  {}
func (tagNode) isNode()    {}

// ParseError points at what's wrong in a message.
type ParseError struct {
	Message string
	Offset  int
	Reason  string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("i18n: %s at offset %d in %q", e.Reason, e.Offset, e.Message)
}

type parser struct {
	src []rune
	pos int
	raw string
}

// scope is where the parser is: inside an option (a '}' ends the text),
// inside a plural (# is the number), inside a tag (</name> ends it).
type scope struct {
	option bool
	plural bool
	tag    string
}

// parse reads an ICU MessageFormat string.
func parse(s string) ([]node, error) {
	p := &parser{src: []rune(s), raw: s}
	nodes, err := p.message(scope{})
	if err != nil {
		return nil, err
	}
	if p.pos < len(p.src) {
		return nil, p.fail("unexpected " + strconv.QuoteRune(p.src[p.pos]))
	}
	return nodes, nil
}

func (p *parser) fail(reason string) error {
	return &ParseError{Message: p.raw, Offset: p.pos, Reason: reason}
}

func (p *parser) peek(off int) (rune, bool) {
	if p.pos+off < len(p.src) {
		return p.src[p.pos+off], true
	}
	return 0, false
}

// message reads text, arguments and tags until what ends its scope.
func (p *parser) message(sc scope) ([]node, error) {
	var out []node
	var text strings.Builder
	flush := func() {
		if text.Len() > 0 {
			out = append(out, textNode(text.String()))
			text.Reset()
		}
	}
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		switch {
		case c == '{':
			flush()
			n, err := p.argument(sc)
			if err != nil {
				return nil, err
			}
			out = append(out, n)
		case c == '}':
			if !sc.option {
				return nil, p.fail("unmatched '}'")
			}
			if sc.tag != "" {
				return nil, p.fail("missing </" + sc.tag + ">")
			}
			flush()
			return out, nil
		case c == '#' && sc.plural:
			flush()
			out = append(out, poundNode{})
			p.pos++
		case c == '<' && p.isTagStart():
			if next, _ := p.peek(1); next == '/' {
				if sc.tag == "" {
					return nil, p.fail("closing tag without an opening tag")
				}
				flush()
				return out, nil
			}
			flush()
			t, err := p.tag(sc)
			if err != nil {
				return nil, err
			}
			out = append(out, t)
		case c == '\'':
			p.apostrophe(&text, sc.plural)
		default:
			text.WriteRune(c)
			p.pos++
		}
	}
	if sc.tag != "" {
		return nil, p.fail("missing </" + sc.tag + ">")
	}
	if sc.option {
		return nil, p.fail("missing '}'")
	}
	flush()
	return out, nil
}

// apostrophe follows ICU's DOUBLE_OPTIONAL mode, as intl-messageformat
// does: two apostrophes in a row are one apostrophe; an apostrophe before
// a syntax character quotes text up to the next single apostrophe; any
// other apostrophe is literal.
func (p *parser) apostrophe(text *strings.Builder, inPlural bool) {
	next, ok := p.peek(1)
	switch {
	case ok && next == '\'':
		text.WriteRune('\'')
		p.pos += 2
	case ok && (next == '{' || next == '}' || next == '<' || next == '>' || next == '|' || (next == '#' && inPlural)):
		p.pos++
		for p.pos < len(p.src) {
			c := p.src[p.pos]
			if c == '\'' {
				if n, ok := p.peek(1); ok && n == '\'' {
					text.WriteRune('\'')
					p.pos += 2
					continue
				}
				p.pos++
				return
			}
			text.WriteRune(c)
			p.pos++
		}
	default:
		text.WriteRune('\'')
		p.pos++
	}
}

func isNameRune(c rune) bool {
	return c == '_' || c == '-' || c == '.' || unicode.IsLetter(c) || unicode.IsDigit(c)
}

// isTagStart reports whether '<' opens or closes a tag (<b>, </b>), as
// opposed to a literal less-than sign.
func (p *parser) isTagStart() bool {
	i := p.pos + 1
	if i < len(p.src) && p.src[i] == '/' {
		i++
	}
	return i < len(p.src) && (unicode.IsLetter(p.src[i]) || p.src[i] == '_')
}

func (p *parser) tag(sc scope) (node, error) {
	p.pos++ // <
	start := p.pos
	for p.pos < len(p.src) && isNameRune(p.src[p.pos]) {
		p.pos++
	}
	name := string(p.src[start:p.pos])
	if p.pos >= len(p.src) || p.src[p.pos] != '>' {
		return nil, p.fail("tag <" + name + "> isn't closed with '>'")
	}
	p.pos++
	inner := sc
	inner.tag = name
	inner.option = false
	children, err := p.message(inner)
	if err != nil {
		return nil, err
	}
	closing := "</" + name + ">"
	if !strings.HasPrefix(string(p.src[p.pos:]), closing) {
		return nil, p.fail("expected " + closing)
	}
	p.pos += len([]rune(closing))
	return tagNode{name: name, children: children}, nil
}

func (p *parser) space() {
	for p.pos < len(p.src) && unicode.IsSpace(p.src[p.pos]) {
		p.pos++
	}
}

func (p *parser) word() string {
	start := p.pos
	for p.pos < len(p.src) && isNameRune(p.src[p.pos]) {
		p.pos++
	}
	return string(p.src[start:p.pos])
}

func (p *parser) expect(c rune) error {
	if p.pos >= len(p.src) || p.src[p.pos] != c {
		return p.fail("expected " + strconv.QuoteRune(c))
	}
	p.pos++
	return nil
}

func (p *parser) argument(sc scope) (node, error) {
	p.pos++ // {
	p.space()
	name := p.word()
	if name == "" {
		return nil, p.fail("an argument needs a name")
	}
	p.space()
	if p.pos < len(p.src) && p.src[p.pos] == '}' {
		p.pos++
		return argNode{name: name}, nil
	}
	if err := p.expect(','); err != nil {
		return nil, err
	}
	p.space()
	kind := p.word()
	p.space()
	switch kind {
	case "plural", "selectordinal":
		if err := p.expect(','); err != nil {
			return nil, err
		}
		return p.plural(name, kind == "selectordinal", sc)
	case "select":
		if err := p.expect(','); err != nil {
			return nil, err
		}
		inner := scope{option: true, plural: sc.plural}
		opts, err := p.options(inner, false)
		if err != nil {
			return nil, err
		}
		if _, ok := opts["other"]; !ok {
			return nil, p.fail("select {" + name + "} needs an 'other' option")
		}
		return selectNode{name: name, options: opts}, nil
	case "number", "date", "time":
		style := ""
		if p.pos < len(p.src) && p.src[p.pos] == ',' {
			p.pos++
			start := p.pos
			level := 0
			for p.pos < len(p.src) {
				c := p.src[p.pos]
				if c == '{' {
					level++
				} else if c == '}' {
					if level == 0 {
						break
					}
					level--
				}
				p.pos++
			}
			style = strings.TrimSpace(string(p.src[start:p.pos]))
		}
		if err := p.expect('}'); err != nil {
			return nil, err
		}
		return argNode{name: name, kind: kind, style: style}, nil
	}
	return nil, p.fail("unknown argument type " + strconv.Quote(kind))
}

func (p *parser) plural(name string, ordinal bool, _ scope) (node, error) {
	p.space()
	offset := 0
	if strings.HasPrefix(string(p.src[p.pos:]), "offset:") {
		p.pos += len("offset:")
		p.space()
		start := p.pos
		for p.pos < len(p.src) && unicode.IsDigit(p.src[p.pos]) {
			p.pos++
		}
		n, err := strconv.Atoi(string(p.src[start:p.pos]))
		if err != nil {
			return nil, p.fail("offset needs a number")
		}
		offset = n
	}
	opts, err := p.options(scope{option: true, plural: true}, true)
	if err != nil {
		return nil, err
	}
	if _, ok := opts["other"]; !ok {
		return nil, p.fail("plural {" + name + "} needs an 'other' option")
	}
	return pluralNode{name: name, ordinal: ordinal, offset: offset, options: opts}, nil
}

// options reads `key {message}` pairs up to the argument's closing '}'.
func (p *parser) options(inner scope, plural bool) (map[string][]node, error) {
	opts := map[string][]node{}
	for {
		p.space()
		if p.pos >= len(p.src) {
			return nil, p.fail("missing '}'")
		}
		if p.src[p.pos] == '}' {
			p.pos++
			if len(opts) == 0 {
				return nil, p.fail("no options")
			}
			return opts, nil
		}
		var key string
		if plural && p.src[p.pos] == '=' {
			p.pos++
			key = "=" + p.word()
		} else {
			key = p.word()
		}
		if key == "" || key == "=" {
			return nil, p.fail("expected an option name")
		}
		p.space()
		if err := p.expect('{'); err != nil {
			return nil, err
		}
		msg, err := p.message(inner)
		if err != nil {
			return nil, err
		}
		if err := p.expect('}'); err != nil {
			return nil, err
		}
		if _, dup := opts[key]; dup {
			return nil, p.fail("option " + key + " appears twice")
		}
		opts[key] = msg
	}
}

// argNames lists the argument and tag names a message uses.
func argNames(nodes []node, into map[string]bool) {
	for _, n := range nodes {
		switch v := n.(type) {
		case argNode:
			into[v.name] = true
		case pluralNode:
			into[v.name] = true
			for _, o := range v.options {
				argNames(o, into)
			}
		case selectNode:
			into[v.name] = true
			for _, o := range v.options {
				argNames(o, into)
			}
		case tagNode:
			into["<"+v.name+">"] = true
			argNames(v.children, into)
		}
	}
}
