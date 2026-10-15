package accessibility

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"github.com/hostrepute/reputecast/internal/mjml"
)

// The checks run on compiled HTML, and every finding has to point at the
// MJML element it came from (Show on canvas, one-click fixes). Before
// compiling, every element of mj-body gets a css-class naming its path
// (rcx-0-1-2: the element-index path from mj-body, as the composer counts
// it), and the top-level elements of Raw HTML blocks a data-rcx attribute.
// The classes only exist in the HTML the checks compile; the stored HTML is
// compiled without them.

const classPrefix = "rcx-"

// pathKey is "0-1-2" for the path [0 1 2].
func pathKey(path []int) string {
	parts := make([]string, len(path))
	for i, p := range path {
		parts[i] = strconv.Itoa(p)
	}
	return strings.Join(parts, "-")
}

// parsePathKey reads "0-1-2"; ok is false for anything else.
func parsePathKey(k string) ([]int, bool) {
	if k == "" {
		return nil, false
	}
	parts := strings.Split(k, "-")
	out := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return nil, false
		}
		out[i] = n
	}
	return out, true
}

// doc is a parsed MJML document with its elements by path.
type doc struct {
	src   string
	root  *mjml.Node
	body  *mjml.Node
	nodes map[string]*mjml.Node
	// parents maps a path to its parent's path ("" for sections).
	parents map[string]string
}

func parseDoc(src string) (*doc, error) {
	root, err := mjml.Parse(src)
	if err != nil {
		return nil, err
	}
	d := &doc{src: src, root: root, body: mjml.Body(root), nodes: map[string]*mjml.Node{}, parents: map[string]string{}}
	if d.body == nil {
		return d, nil
	}
	for i, sec := range d.body.Children {
		sec.Walk(func(x *mjml.Node, rel []int) {
			path := append([]int{i}, rel...)
			k := pathKey(path)
			d.nodes[k] = x
			if len(path) > 1 {
				d.parents[k] = pathKey(path[:len(path)-1])
			}
		})
	}
	return d, nil
}

// node returns the element at a path.
func (d *doc) node(path []int) *mjml.Node {
	if d == nil {
		return nil
	}
	return d.nodes[pathKey(path)]
}

// section returns the top-level element containing path, and its label as
// the composer names it ("hero", "footer"), when it has a ReputeCast marker.
func (d *doc) section(path []int) (*mjml.Node, string) {
	if d == nil || d.body == nil || len(path) == 0 || path[0] >= len(d.body.Children) {
		return nil, ""
	}
	sec := d.body.Children[path[0]]
	return sec, mjml.SectionLabel(sec, path[0], len(d.body.Children))
}

// edit replaces src[start:end] with text.
type edit struct {
	start, end int
	text       string
}

func applyEdits(src string, edits []edit) string {
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start < edits[j].start })
	var b strings.Builder
	last := 0
	for _, e := range edits {
		if e.start < last {
			continue
		}
		b.WriteString(src[last:e.start])
		b.WriteString(e.text)
		last = e.end
	}
	b.WriteString(src[last:])
	return b.String()
}

// tagAttr is an attribute of an opening tag with the offsets of its name
// and value (valStart == valEnd when it has none).
type tagAttr struct {
	name             string
	nameStart        int
	valStart, valEnd int
	quoted           bool
}

// openTag scans the opening tag that starts at src[start] ('<'). It
// returns the offset just after the tag name, the attributes and the
// offset after the closing '>' (or "/>").
func openTag(src string, start int) (nameEnd int, attrs []tagAttr, end int, ok bool) {
	i := start + 1
	for i < len(src) && (isNameByte(src[i])) {
		i++
	}
	nameEnd = i
	for i < len(src) {
		for i < len(src) && isSpaceByte(src[i]) {
			i++
		}
		if i >= len(src) {
			return 0, nil, 0, false
		}
		if src[i] == '>' {
			return nameEnd, attrs, i + 1, true
		}
		if src[i] == '/' && i+1 < len(src) && src[i+1] == '>' {
			return nameEnd, attrs, i + 2, true
		}
		ns := i
		for i < len(src) && !isSpaceByte(src[i]) && src[i] != '=' && src[i] != '>' && (src[i] != '/' || i+1 >= len(src) || src[i+1] != '>') {
			i++
		}
		a := tagAttr{name: strings.ToLower(src[ns:i]), nameStart: ns}
		for i < len(src) && isSpaceByte(src[i]) {
			i++
		}
		if i < len(src) && src[i] == '=' {
			i++
			for i < len(src) && isSpaceByte(src[i]) {
				i++
			}
			if i < len(src) && (src[i] == '"' || src[i] == '\'') {
				q := src[i]
				e := strings.IndexByte(src[i+1:], q)
				if e < 0 {
					return 0, nil, 0, false
				}
				a.valStart, a.valEnd, a.quoted = i+1, i+1+e, true
				i = i + 1 + e + 1
			} else {
				vs := i
				for i < len(src) && !isSpaceByte(src[i]) && src[i] != '>' {
					i++
				}
				a.valStart, a.valEnd = vs, i
			}
		} else {
			a.valStart, a.valEnd = i, i
		}
		if a.name != "" {
			attrs = append(attrs, a)
		}
		if i == ns {
			i++
		}
	}
	return 0, nil, 0, false
}

func isNameByte(c byte) bool {
	return c == '-' || c == '_' || c == ':' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func isSpaceByte(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' }

// setAttrEdit returns the edit that sets an attribute of the opening tag
// at start: it replaces an existing value or adds the attribute after the
// last one.
func setAttrEdit(src string, start int, name, value string) (edit, bool) {
	_, attrs, end, ok := openTag(src, start)
	if !ok {
		return edit{}, false
	}
	esc := html.EscapeString(value)
	for _, a := range attrs {
		if a.name == name {
			if a.quoted {
				return edit{a.valStart, a.valEnd, esc}, true
			}
			if a.valStart == a.valEnd {
				// A bare attribute (no value).
				return edit{a.valStart, a.valEnd, `="` + esc + `"`}, true
			}
			return edit{a.valStart, a.valEnd, `"` + esc + `"`}, true
		}
	}
	at := end - 1 // the '>'
	if at > start && src[at-1] == '/' {
		at--
	}
	for at > start && isSpaceByte(src[at-1]) {
		at--
	}
	return edit{at, at, " " + name + `="` + esc + `"`}, true
}

// removeAttrEdit returns the edit that removes an attribute, if present.
func removeAttrEdit(src string, start int, name string) (edit, bool) {
	_, attrs, _, ok := openTag(src, start)
	if !ok {
		return edit{}, false
	}
	for _, a := range attrs {
		if a.name != name {
			continue
		}
		// From the spaces before the name to the end of the value and its quote.
		s, e := a.nameStart, a.valEnd
		if a.quoted {
			e++
		}
		for s > start && isSpaceByte(src[s-1]) {
			s--
		}
		return edit{s, e, ""}, true
	}
	return edit{}, false
}

// classless are MJML elements that never take a css-class.
var classless = map[string]bool{"mj-raw": true, "mj-include": true}

// annotate adds the path classes to every element of mj-body.
func annotate(d *doc) string {
	if d.body == nil {
		return d.src
	}
	var edits []edit
	for k, n := range d.nodes {
		if n.Tag == "mj-raw" {
			edits = append(edits, rawEdits(d.src, n, k)...)
			continue
		}
		if classless[n.Tag] || !strings.HasPrefix(n.Tag, "mj-") {
			continue
		}
		token := classPrefix + k
		_, attrs, _, ok := openTag(d.src, n.Start)
		if !ok {
			continue
		}
		cls := ""
		for _, a := range attrs {
			if a.name == "css-class" {
				cls = html.UnescapeString(d.src[a.valStart:a.valEnd])
			}
		}
		if e, ok := setAttrEdit(d.src, n.Start, "css-class", strings.TrimSpace(token+" "+cls)); ok {
			edits = append(edits, e)
		}
	}
	return applyEdits(d.src, edits)
}

// rawEdits marks the top-level elements of a Raw HTML block with data-rcx.
func rawEdits(src string, n *mjml.Node, key string) []edit {
	open := strings.IndexByte(src[n.Start:], '>')
	if open < 0 || n.Inner == "" {
		return nil
	}
	base := n.Start + open + 1
	// Inner starts right after the opening tag.
	if !strings.HasPrefix(src[base:], n.Inner) {
		return nil
	}
	var edits []edit
	z := html.NewTokenizer(strings.NewReader(n.Inner))
	off, depth := 0, 0
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if z.Err() != io.EOF {
				return edits
			}
			return edits
		}
		raw := len(z.Raw())
		switch tt {
		case html.StartTagToken, html.SelfClosingTagToken:
			name, _ := z.TagName()
			tag := strings.ToLower(string(name))
			if depth == 0 {
				at := base + off + 1 + len(name)
				edits = append(edits, edit{at, at, fmt.Sprintf(` data-rcx="%s"`, key)})
			}
			if tt == html.StartTagToken && !voidElements[tag] {
				depth++
			}
		case html.EndTagToken:
			if depth > 0 {
				depth--
			}
		}
		off += raw
	}
}

var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true, "img": true, "input": true,
	"link": true, "meta": true, "source": true, "track": true, "wbr": true,
}

// pathOfElement reads the path an element of the checked HTML carries.
func pathOfElement(n *html.Node) (string, bool) {
	for _, a := range n.Attr {
		switch a.Key {
		case "data-rcx":
			return a.Val, true
		case "class":
			for _, c := range strings.Fields(a.Val) {
				if strings.HasPrefix(c, classPrefix) && !strings.HasSuffix(c, "-outlook") {
					return strings.TrimPrefix(c, classPrefix), true
				}
			}
		}
	}
	return "", false
}
