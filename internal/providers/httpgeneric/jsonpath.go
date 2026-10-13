package httpgeneric

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// jsonPath is a parsed JSONPath of the subset mappings use: the root "$",
// member access (.name, ['name'], ["name"]), array indexes ([0]) and
// wildcards ([*], .*). Filters and recursive descent are not supported.
type jsonPath struct {
	src   string
	steps []step
}

type step struct {
	key      string
	index    int
	isIndex  bool
	wildcard bool
}

func parsePath(src string) (*jsonPath, error) {
	s := strings.TrimSpace(src)
	if s == "" {
		return nil, nil
	}
	if !strings.HasPrefix(s, "$") {
		return nil, fmt.Errorf("JSONPath %q must start with $", src)
	}
	p := &jsonPath{src: s}
	i := 1
	for i < len(s) {
		switch s[i] {
		case '.':
			i++
			if i < len(s) && s[i] == '*' {
				p.steps = append(p.steps, step{wildcard: true})
				i++
				continue
			}
			j := i
			for j < len(s) && s[j] != '.' && s[j] != '[' {
				j++
			}
			if j == i {
				return nil, fmt.Errorf("JSONPath %q has an empty name", src)
			}
			p.steps = append(p.steps, step{key: s[i:j]})
			i = j
		case '[':
			end := strings.IndexByte(s[i:], ']')
			if end < 0 {
				return nil, fmt.Errorf("JSONPath %q has an unclosed [", src)
			}
			inner := strings.TrimSpace(s[i+1 : i+end])
			switch {
			case inner == "*":
				p.steps = append(p.steps, step{wildcard: true})
			case len(inner) >= 2 && (inner[0] == '\'' || inner[0] == '"') && inner[len(inner)-1] == inner[0]:
				p.steps = append(p.steps, step{key: inner[1 : len(inner)-1]})
			default:
				n, err := strconv.Atoi(inner)
				if err != nil || n < 0 {
					return nil, fmt.Errorf("JSONPath %q: [%s] is not an index, a quoted name or *", src, inner)
				}
				p.steps = append(p.steps, step{index: n, isIndex: true})
			}
			i += end + 1
		default:
			return nil, fmt.Errorf("JSONPath %q: unexpected %q at position %d", src, s[i], i)
		}
	}
	return p, nil
}

// all returns every value the path selects in doc.
func (p *jsonPath) all(doc any) []any {
	if p == nil {
		return nil
	}
	cur := []any{doc}
	for _, st := range p.steps {
		var next []any
		for _, v := range cur {
			switch {
			case st.wildcard:
				switch x := v.(type) {
				case []any:
					next = append(next, x...)
				case map[string]any:
					for _, k := range slices.Sorted(maps.Keys(x)) {
						next = append(next, x[k])
					}
				}
			case st.isIndex:
				if a, ok := v.([]any); ok && st.index < len(a) {
					next = append(next, a[st.index])
				}
			default:
				if m, ok := v.(map[string]any); ok {
					if x, ok := m[st.key]; ok {
						next = append(next, x)
					}
				}
			}
		}
		cur = next
	}
	return cur
}

// first returns the first value the path selects.
func (p *jsonPath) first(doc any) (any, bool) {
	vs := p.all(doc)
	if len(vs) == 0 {
		return nil, false
	}
	return vs[0], true
}

// text returns the first selected value as text ("" when absent or null).
func (p *jsonPath) text(doc any) string {
	v, ok := p.first(doc)
	if !ok {
		return ""
	}
	return strings.TrimSpace(stringify(v))
}

// decodeJSON decodes JSON keeping numbers exact.
func decodeJSON(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}
