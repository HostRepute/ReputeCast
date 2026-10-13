package httpgeneric

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Templates use Liquid-style output tags only: {{ name }}, {{ secrets.api_key }},
// {{ settings.domain | default: "eu" }}, {{ "api:" | append: secrets.api_key | base64_encode }}.
// There are no control-flow tags ({% %}): a mapping is a fixed request shape.
//
// Values are escaped for where they appear, so a template can't be broken by
// the data it carries:
//   - body (JSON): inside a JSON string a value is JSON-escaped without
//     quotes ("subject": "{{ subject }}"); outside a string it is JSON-encoded
//     ("headers": {{ headers_json }} becomes an object, a missing value null);
//   - URL: values are path-escaped, or query-escaped after "?";
//   - headers: values are inserted as text with line breaks removed.

// mode is the escaping context of a template.
type mode int

const (
	modeJSON mode = iota
	modeURL
	modeText
)

// segment is literal text or a placeholder.
type segment struct {
	lit string
	ph  *placeholder
	// inString is true when the placeholder sits inside a JSON string
	// (body) or in the query string (URL).
	inString bool
}

type placeholder struct {
	src     string
	base    operand
	filters []filter
}

type operand struct {
	path    []string // variable path ("secrets", "api_key")
	literal *string  // string literal
}

type filter struct {
	name string
	args []operand
}

// template is a parsed template.
type template struct {
	mode mode
	segs []segment
}

// filters are the Liquid filters a template may use, with their arity.
var filters = map[string]int{
	"default": 1, "prepend": 1, "append": 1, "base64_encode": 0, "url_encode": 0,
	"downcase": 0, "upcase": 0, "strip": 0, "json": 0,
}

// parseTemplate splits src into literal text and placeholders, tracking the
// escaping context of each placeholder.
func parseTemplate(src string, m mode) (*template, error) {
	t := &template{mode: m}
	var lit []byte
	inString, escaped, inQuery := false, false, false
	for i := 0; i < len(src); i++ {
		if strings.HasPrefix(src[i:], "{{") {
			if m == modeJSON && inString && escaped {
				// "\{{" inside a JSON string is a literal "{{" (for an API's
				// own placeholders): the backslash only marks it and is
				// dropped, as "\{" is no JSON escape.
				lit = append(lit[:len(lit)-1], '{', '{')
				escaped = false
				i++
				continue
			}
			end := strings.Index(src[i+2:], "}}")
			if end < 0 {
				return nil, fmt.Errorf("a {{ placeholder at position %d is not closed with }}", i)
			}
			raw := src[i+2 : i+2+end]
			ph, err := parsePlaceholder(raw)
			if err != nil {
				return nil, err
			}
			if len(lit) > 0 {
				t.segs = append(t.segs, segment{lit: string(lit)})
				lit = lit[:0]
			}
			ctx := false
			switch m {
			case modeJSON:
				ctx = inString
			case modeURL:
				ctx = inQuery
			}
			t.segs = append(t.segs, segment{ph: ph, inString: ctx})
			i += 2 + end + 1
			continue
		}
		c := src[i]
		lit = append(lit, c)
		switch m {
		case modeJSON:
			switch {
			case escaped:
				escaped = false
			case c == '\\' && inString:
				escaped = true
			case c == '"':
				inString = !inString
			}
		case modeURL:
			if c == '?' {
				inQuery = true
			}
		}
	}
	if len(lit) > 0 {
		t.segs = append(t.segs, segment{lit: string(lit)})
	}
	return t, nil
}

// parsePlaceholder parses `base | filter: arg, arg | filter`.
func parsePlaceholder(raw string) (*placeholder, error) {
	ph := &placeholder{src: strings.TrimSpace(raw)}
	parts, err := splitTop(raw, '|')
	if err != nil {
		return nil, err
	}
	base, err := parseOperand(parts[0])
	if err != nil {
		return nil, fmt.Errorf("{{ %s }}: %w", ph.src, err)
	}
	ph.base = base
	for _, p := range parts[1:] {
		name, rest, _ := strings.Cut(p, ":")
		name = strings.TrimSpace(name)
		arity, ok := filters[name]
		if !ok {
			return nil, fmt.Errorf("{{ %s }}: unknown filter %q (use default, prepend, append, base64_encode, url_encode, downcase, upcase, strip or json)", ph.src, name)
		}
		f := filter{name: name}
		if strings.TrimSpace(rest) != "" {
			args, err := splitTop(rest, ',')
			if err != nil {
				return nil, err
			}
			for _, a := range args {
				op, err := parseOperand(a)
				if err != nil {
					return nil, fmt.Errorf("{{ %s }}: %w", ph.src, err)
				}
				f.args = append(f.args, op)
			}
		}
		if len(f.args) != arity {
			return nil, fmt.Errorf("{{ %s }}: filter %s takes %d argument(s)", ph.src, name, arity)
		}
		ph.filters = append(ph.filters, f)
	}
	return ph, nil
}

// splitTop splits s on sep outside quoted strings.
func splitTop(s string, sep byte) ([]string, error) {
	var out []string
	var quote byte
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == sep:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote in %q", strings.TrimSpace(s))
	}
	return append(out, s[start:]), nil
}

func parseOperand(s string) (operand, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return operand{}, fmt.Errorf("empty value")
	}
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		v := s[1 : len(s)-1]
		return operand{literal: &v}, nil
	}
	path := strings.Split(s, ".")
	for _, p := range path {
		if p == "" {
			return operand{}, fmt.Errorf("%q is not a valid name", s)
		}
		for _, r := range p {
			if !nameChar(r) {
				return operand{}, fmt.Errorf("%q is not a valid name", s)
			}
		}
	}
	return operand{path: path}, nil
}

func nameChar(r rune) bool {
	return r == '_' || r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}

// render fills the template from data.
func (t *template) render(data map[string]any) (string, error) {
	var b strings.Builder
	for _, s := range t.segs {
		if s.ph == nil {
			b.WriteString(s.lit)
			continue
		}
		v, err := s.ph.eval(data)
		if err != nil {
			return "", err
		}
		switch t.mode {
		case modeJSON:
			if s.inString {
				enc, err := jsonString(stringify(v))
				if err != nil {
					return "", err
				}
				b.WriteString(enc[1 : len(enc)-1])
				continue
			}
			enc, err := jsonValue(v)
			if err != nil {
				return "", fmt.Errorf("{{ %s }}: %w", s.ph.src, err)
			}
			b.WriteString(enc)
		case modeURL:
			if s.inString {
				b.WriteString(url.QueryEscape(stringify(v)))
			} else {
				b.WriteString(url.PathEscape(stringify(v)))
			}
		default:
			b.WriteString(strings.NewReplacer("\r", " ", "\n", " ").Replace(stringify(v)))
		}
	}
	return b.String(), nil
}

func (ph *placeholder) eval(data map[string]any) (any, error) {
	v := ph.base.value(data)
	for _, f := range ph.filters {
		var arg any
		if len(f.args) > 0 {
			arg = f.args[0].value(data)
		}
		switch f.name {
		case "default":
			if isEmpty(v) {
				v = arg
			}
		case "prepend":
			v = stringify(arg) + stringify(v)
		case "append":
			v = stringify(v) + stringify(arg)
		case "base64_encode":
			v = base64.StdEncoding.EncodeToString([]byte(stringify(v)))
		case "url_encode":
			v = url.QueryEscape(stringify(v))
		case "downcase":
			v = strings.ToLower(stringify(v))
		case "upcase":
			v = strings.ToUpper(stringify(v))
		case "strip":
			v = strings.TrimSpace(stringify(v))
		case "json":
			enc, err := jsonValue(v)
			if err != nil {
				return nil, err
			}
			v = enc
		}
	}
	return v, nil
}

func (o operand) value(data map[string]any) any {
	if o.literal != nil {
		return *o.literal
	}
	var cur any = data
	for _, p := range o.path {
		switch m := cur.(type) {
		case map[string]any:
			cur = m[p]
		case map[string]string:
			v, ok := m[p]
			if !ok {
				return nil
			}
			cur = v
		default:
			return nil
		}
	}
	return cur
}

func isEmpty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case map[string]string:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	case []any:
		return len(x) == 0
	}
	return false
}

// stringify renders a value as text: strings as they are, numbers and
// booleans in JSON form, objects and arrays as JSON, nothing as "".
func stringify(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case json.Number:
		return x.String()
	case bool:
		return strconv.FormatBool(x)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func jsonString(s string) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

func jsonValue(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}
