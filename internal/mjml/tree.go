// Package mjml works with MJML documents on the server: a tolerant parser
// for outlines, warnings and wireframes, the server-side compiler (gomjml,
// MJML 4.15) used when content arrives as MJML only (API and automation
// sends), and head helpers that keep <mj-title>/<mj-preview> in step with
// the subject and preheader.
//
// The composer compiles in the browser with mjml-browser (grapesjs-mjml);
// both engines target MJML 4 and parity is checked by tests.
package mjml

import (
	"fmt"
	"html"
	"strings"
)

// Attr is one attribute with its decoded value.
type Attr struct {
	Name  string
	Value string
}

// Node is one element of an MJML document.
type Node struct {
	Tag      string
	Attrs    []Attr
	Children []*Node
	// Inner is the raw content of ending tags (mj-text, mj-button, mj-raw…),
	// which hold HTML rather than MJML elements.
	Inner string
	// Marker is the ReputeCast block a preceding <!-- rc:… --> comment names
	// ("footer", "header-logo"), empty otherwise.
	Marker string
	// Line is the 1-based line of the opening tag.
	Line int
	// Start and End are byte offsets of the element in the source.
	Start, End int
}

// Get returns an attribute value and whether it is set.
func (n *Node) Get(name string) (string, bool) {
	for _, a := range n.Attrs {
		if a.Name == name {
			return a.Value, true
		}
	}
	return "", false
}

// Attr returns an attribute value or "".
func (n *Node) Attr(name string) string {
	v, _ := n.Get(name)
	return v
}

// Find returns the first descendant (or n itself) with the tag.
func (n *Node) Find(tag string) *Node {
	if n == nil {
		return nil
	}
	if n.Tag == tag {
		return n
	}
	for _, c := range n.Children {
		if f := c.Find(tag); f != nil {
			return f
		}
	}
	return nil
}

// Walk calls fn for n and every descendant, depth first, with the path of
// element indices from n.
func (n *Node) Walk(fn func(node *Node, path []int)) {
	var walk func(*Node, []int)
	walk = func(x *Node, path []int) {
		fn(x, path)
		for i, c := range x.Children {
			walk(c, append(append([]int(nil), path...), i))
		}
	}
	walk(n, nil)
}

// ParseError is a syntax problem with its line.
type ParseError struct {
	Line    int
	Message string
}

func (e *ParseError) Error() string { return fmt.Sprintf("Line %d: %s", e.Line, e.Message) }

// endingTags hold HTML (or text) instead of MJML children.
var endingTags = map[string]bool{
	"mj-text": true, "mj-button": true, "mj-raw": true, "mj-table": true, "mj-navbar-link": true,
	"mj-social-element": true, "mj-title": true, "mj-preview": true, "mj-style": true,
	"mj-accordion-title": true, "mj-accordion-text": true,
}

// voidTags may be written without a closing tag; they are closed
// implicitly when their parent closes, as the MJML parser does.
var voidTags = map[string]bool{
	"mj-image": true, "mj-divider": true, "mj-spacer": true, "mj-carousel-image": true, "mj-breakpoint": true,
	"mj-font": true, "mj-all": true, "mj-class": true, "mj-html-attribute": true, "mj-include": true,
}

type parser struct {
	src    string
	pos    int
	lines  []int // byte offset of each line start
	marker string
}

// Parse reads an MJML document and returns its <mjml> root. It is lenient
// about whitespace, attribute quoting, comments and void tags, and reports
// tags that are never closed with their line.
func Parse(src string) (*Node, error) {
	p := &parser{src: src}
	p.lines = append(p.lines, 0)
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			p.lines = append(p.lines, i+1)
		}
	}
	root := &Node{Tag: "#document"}
	stack := []*Node{root}
	for p.pos < len(src) {
		lt := strings.IndexByte(src[p.pos:], '<')
		if lt < 0 {
			break
		}
		lt += p.pos
		switch {
		case strings.HasPrefix(src[lt:], "<!--"):
			end := strings.Index(src[lt+4:], "-->")
			if end < 0 {
				return nil, &ParseError{Line: p.line(lt), Message: "unclosed comment"}
			}
			body := strings.TrimSpace(src[lt+4 : lt+4+end])
			if strings.HasPrefix(body, "rc:") {
				p.marker = strings.TrimSpace(strings.TrimPrefix(body, "rc:"))
			}
			p.pos = lt + 4 + end + 3
			continue
		case strings.HasPrefix(src[lt:], "</"):
			gt := strings.IndexByte(src[lt:], '>')
			if gt < 0 {
				return nil, &ParseError{Line: p.line(lt), Message: "unfinished closing tag"}
			}
			tag := strings.ToLower(strings.TrimSpace(src[lt+2 : lt+gt]))
			p.pos = lt + gt + 1
			// Close implicitly-closed void tags, then the matching element.
			for len(stack) > 1 && stack[len(stack)-1].Tag != tag && voidTags[stack[len(stack)-1].Tag] {
				top := stack[len(stack)-1]
				top.End = lt
				stack = stack[:len(stack)-1]
			}
			top := stack[len(stack)-1]
			if top.Tag != tag {
				if len(stack) == 1 {
					return nil, &ParseError{Line: p.line(lt), Message: fmt.Sprintf("</%s> has no opening tag", tag)}
				}
				return nil, &ParseError{Line: top.Line, Message: fmt.Sprintf("unclosed <%s>", top.Tag)}
			}
			top.End = p.pos
			stack = stack[:len(stack)-1]
			continue
		case lt+1 < len(src) && (src[lt+1] == '!' || src[lt+1] == '?'):
			gt := strings.IndexByte(src[lt:], '>')
			if gt < 0 {
				p.pos = len(src)
				break
			}
			p.pos = lt + gt + 1
			continue
		}
		node, selfClosing, err := p.openTag(lt)
		if err != nil {
			return nil, err
		}
		if node == nil {
			p.pos = lt + 1
			continue
		}
		parent := stack[len(stack)-1]
		// A void tag left open is closed by its next sibling.
		if voidTags[parent.Tag] && len(stack) > 1 {
			parent.End = lt
			stack = stack[:len(stack)-1]
			parent = stack[len(stack)-1]
		}
		parent.Children = append(parent.Children, node)
		if selfClosing {
			node.End = p.pos
			continue
		}
		if endingTags[node.Tag] {
			closeTag := "</" + node.Tag
			idx := indexFold(src[p.pos:], closeTag)
			if idx < 0 {
				return nil, &ParseError{Line: node.Line, Message: fmt.Sprintf("unclosed <%s>", node.Tag)}
			}
			node.Inner = src[p.pos : p.pos+idx]
			gt := strings.IndexByte(src[p.pos+idx:], '>')
			if gt < 0 {
				return nil, &ParseError{Line: node.Line, Message: fmt.Sprintf("unclosed <%s>", node.Tag)}
			}
			p.pos = p.pos + idx + gt + 1
			node.End = p.pos
			continue
		}
		stack = append(stack, node)
	}
	for len(stack) > 1 {
		top := stack[len(stack)-1]
		if !voidTags[top.Tag] {
			return nil, &ParseError{Line: top.Line, Message: fmt.Sprintf("unclosed <%s>", top.Tag)}
		}
		top.End = len(src)
		stack = stack[:len(stack)-1]
	}
	for _, c := range root.Children {
		if c.Tag == "mjml" {
			return c, nil
		}
	}
	if len(root.Children) == 0 {
		return nil, &ParseError{Line: 1, Message: "the document is empty"}
	}
	return nil, &ParseError{Line: root.Children[0].Line, Message: "the document must start with <mjml>"}
}

// openTag parses the tag starting at lt. It returns nil for text that only
// looks like a tag ("a < b").
func (p *parser) openTag(lt int) (*Node, bool, error) {
	src := p.src
	i := lt + 1
	start := i
	for i < len(src) && (isNameChar(src[i])) {
		i++
	}
	if i == start {
		return nil, false, nil
	}
	n := &Node{Tag: strings.ToLower(src[start:i]), Line: p.line(lt), Start: lt, Marker: p.marker}
	p.marker = ""
	for {
		for i < len(src) && isSpace(src[i]) {
			i++
		}
		if i >= len(src) {
			return nil, false, &ParseError{Line: n.Line, Message: fmt.Sprintf("unfinished <%s>", n.Tag)}
		}
		if src[i] == '>' {
			p.pos = i + 1
			return n, false, nil
		}
		if src[i] == '/' && i+1 < len(src) && src[i+1] == '>' {
			p.pos = i + 2
			return n, true, nil
		}
		ns := i
		for i < len(src) && !isSpace(src[i]) && src[i] != '=' && src[i] != '>' && (src[i] != '/' || i+1 >= len(src) || src[i+1] != '>') {
			i++
		}
		name := strings.ToLower(src[ns:i])
		for i < len(src) && isSpace(src[i]) {
			i++
		}
		val := ""
		if i < len(src) && src[i] == '=' {
			i++
			for i < len(src) && isSpace(src[i]) {
				i++
			}
			if i < len(src) && (src[i] == '"' || src[i] == '\'') {
				q := src[i]
				end := strings.IndexByte(src[i+1:], q)
				if end < 0 {
					return nil, false, &ParseError{Line: p.line(i), Message: fmt.Sprintf("unclosed quote in <%s>", n.Tag)}
				}
				val = src[i+1 : i+1+end]
				i = i + 1 + end + 1
			} else {
				vs := i
				for i < len(src) && !isSpace(src[i]) && src[i] != '>' {
					i++
				}
				val = src[vs:i]
			}
		}
		if name != "" {
			n.Attrs = append(n.Attrs, Attr{Name: name, Value: html.UnescapeString(val)})
		}
	}
}

func (p *parser) line(off int) int {
	lo, hi := 0, len(p.lines)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if p.lines[mid] <= off {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo + 1
}

func isNameChar(c byte) bool {
	return c == '-' || c == '_' || c == ':' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' }

func indexFold(s, sub string) int {
	n := len(sub)
	for i := 0; i+n <= len(s); i++ {
		if strings.EqualFold(s[i:i+n], sub) {
			return i
		}
	}
	return -1
}

// Body returns the <mj-body> of a document, or nil.
func Body(root *Node) *Node {
	if root == nil {
		return nil
	}
	for _, c := range root.Children {
		if c.Tag == "mj-body" {
			return c
		}
	}
	return nil
}

// Head returns the <mj-head> of a document, or nil.
func Head(root *Node) *Node {
	if root == nil {
		return nil
	}
	for _, c := range root.Children {
		if c.Tag == "mj-head" {
			return c
		}
	}
	return nil
}
