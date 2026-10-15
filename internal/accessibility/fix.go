package accessibility

import (
	"errors"
	"fmt"
	"html"
	"io"
	"regexp"
	"strconv"
	"strings"

	xhtml "golang.org/x/net/html"

	"github.com/hostrepute/reputecast/internal/mjml"
)

// Fix kinds that FixRequest takes.
const (
	FixAltText    = "alt_text"
	FixDecorative = "decorative"
	FixLinkText   = "link_text"
	FixColor      = "color"
	FixHeading    = "heading"
	FixTextSize   = "text_size"
)

// FixRequest is one fix to apply to an email's MJML. Fixes that need no
// words (color, heading, text size, decorative) go to every language
// version; alt text and link text to the version of Lang.
type FixRequest struct {
	Kind string
	// Path is the element (mj-body element-index path); Index the nth link
	// or image inside a text or Raw HTML block.
	Path  []int
	Index int
	// Value is the alt text, the link text or the color.
	Value string
	// Target says which color changes: "background" or "text"; From is the
	// color it replaces inside the element.
	Target string
	From   string
	// Headings are the level changes of a heading fix.
	Headings []AccessibilityHeadingChange
	// ToPx is the text size of a text-size fix.
	ToPx int
}

// ErrFixTarget means the element a fix points at isn't in the email any more.
var ErrFixTarget = errors.New("the element isn't in the email any more")

// ApplyFix applies one fix to an MJML document.
func ApplyFix(src string, f FixRequest) (string, error) {
	d, err := parseDoc(src)
	if err != nil {
		return src, err
	}
	switch f.Kind {
	case FixHeading:
		return fixHeadings(d, f.Headings)
	}
	n := d.node(f.Path)
	if n == nil {
		return src, ErrFixTarget
	}
	switch f.Kind {
	case FixAltText, FixDecorative:
		alt := strings.TrimSpace(f.Value)
		if f.Kind == FixDecorative {
			alt = ""
		} else if alt == "" {
			return src, errors.New("write the alt text first")
		}
		return fixAlt(d, n, f.Index, alt)
	case FixLinkText:
		text := strings.TrimSpace(f.Value)
		if text == "" {
			return src, errors.New("write the link text first")
		}
		return fixLinkText(d, n, f.Index, text)
	case FixColor:
		c, ok := ParseColor(f.Value)
		if !ok {
			return src, fmt.Errorf("%q isn't a color", f.Value)
		}
		return fixColor(d, n, f.Target, f.From, c.Hex())
	case FixTextSize:
		if f.ToPx < MinMinTextPx || f.ToPx > MaxMinTextPx {
			return src, fmt.Errorf("text size must be %d to %d px", MinMinTextPx, MaxMinTextPx)
		}
		return fixTextSize(d, n, f.ToPx)
	}
	return src, fmt.Errorf("unknown fix %q", f.Kind)
}

// setNodeAttr sets an attribute on an MJML element's opening tag.
func setNodeAttr(src string, n *mjml.Node, name, value string) (string, bool) {
	e, ok := setAttrEdit(src, n.Start, name, value)
	if !ok {
		return src, false
	}
	return applyEdits(src, []edit{e}), true
}

// innerRange is where an ending tag's content sits in the source.
func innerRange(src string, n *mjml.Node) (int, int, bool) {
	_, _, open, ok := openTag(src, n.Start)
	if !ok || open > n.End {
		return 0, 0, false
	}
	rel := strings.LastIndex(strings.ToLower(src[open:n.End]), "</"+n.Tag)
	if rel < 0 {
		return 0, 0, false
	}
	return open, open + rel, true
}

// htmlElement is a start tag inside an element's HTML content.
type htmlElement struct {
	tag            string
	start, nameEnd int // offsets in the content
	tagEnd         int // after '>'
	closeStart     int // the matching end tag, -1 when it has none
	closeEnd       int
	href           bool
}

// elementsOf lists the elements of an HTML fragment with their offsets.
func elementsOf(content string) []*htmlElement {
	var out []*htmlElement
	var stack []*htmlElement
	z := xhtml.NewTokenizer(strings.NewReader(content))
	off := 0
	for {
		tt := z.Next()
		if tt == xhtml.ErrorToken {
			if z.Err() != io.EOF {
				return out
			}
			return out
		}
		raw := len(z.Raw())
		start := off
		off += raw
		switch tt {
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			name, more := z.TagName()
			tag := strings.ToLower(string(name))
			el := &htmlElement{tag: tag, start: start, nameEnd: start + 1 + len(name), tagEnd: off, closeStart: -1, closeEnd: -1}
			for more {
				var k []byte
				k, _, more = z.TagAttr()
				if strings.EqualFold(string(k), "href") {
					el.href = true
				}
			}
			out = append(out, el)
			if tt == xhtml.StartTagToken && !voidElements[tag] {
				stack = append(stack, el)
			}
		case xhtml.EndTagToken:
			name, _ := z.TagName()
			tag := strings.ToLower(string(name))
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].tag == tag {
					stack[i].closeStart, stack[i].closeEnd = start, off
					stack = stack[:i]
					break
				}
			}
		}
	}
}

func nth(els []*htmlElement, match func(*htmlElement) bool, index int) *htmlElement {
	i := 0
	for _, e := range els {
		if match(e) {
			if i == index {
				return e
			}
			i++
		}
	}
	return nil
}

// replaceInner rebuilds the source with new content for an ending tag.
func replaceInner(src string, n *mjml.Node, edits []edit) (string, error) {
	s, e, ok := innerRange(src, n)
	if !ok {
		return src, ErrFixTarget
	}
	content := applyEdits(src[s:e], edits)
	return src[:s] + content + src[e:], nil
}

func fixAlt(d *doc, n *mjml.Node, index int, alt string) (string, error) {
	switch n.Tag {
	case "mj-image", "mj-social-element", "mj-carousel-image":
		out, ok := setNodeAttr(d.src, n, "alt", alt)
		if !ok {
			return d.src, ErrFixTarget
		}
		return out, nil
	case "mj-text", "mj-raw", "mj-button", "mj-table":
		s, e, ok := innerRange(d.src, n)
		if !ok {
			return d.src, ErrFixTarget
		}
		content := d.src[s:e]
		img := nth(elementsOf(content), func(x *htmlElement) bool { return x.tag == "img" }, index)
		if img == nil {
			return d.src, ErrFixTarget
		}
		ed, ok := setAttrEdit(content, img.start, "alt", alt)
		if !ok {
			return d.src, ErrFixTarget
		}
		return replaceInner(d.src, n, []edit{ed})
	}
	return d.src, ErrFixTarget
}

func fixLinkText(d *doc, n *mjml.Node, index int, text string) (string, error) {
	esc := html.EscapeString(text)
	switch n.Tag {
	case "mj-button", "mj-navbar-link":
		return replaceInner(d.src, n, []edit{{0, len(innerOf(d.src, n)), esc}})
	case "mj-text", "mj-raw", "mj-table":
		s, e, ok := innerRange(d.src, n)
		if !ok {
			return d.src, ErrFixTarget
		}
		a := nth(elementsOf(d.src[s:e]), func(x *htmlElement) bool { return x.tag == "a" && x.href }, index)
		if a == nil || a.closeStart < 0 {
			return d.src, ErrFixTarget
		}
		return replaceInner(d.src, n, []edit{{a.tagEnd, a.closeStart, esc}})
	}
	return d.src, ErrFixTarget
}

func innerOf(src string, n *mjml.Node) string {
	s, e, ok := innerRange(src, n)
	if !ok {
		return ""
	}
	return src[s:e]
}

// colorPattern matches a color value in a style or attribute.
func colorPattern(hex string) *regexp.Regexp {
	c, ok := ParseColor(hex)
	if !ok {
		return regexp.MustCompile(`$^`)
	}
	alts := []string{regexp.QuoteMeta(strings.ToLower(c.Hex()))}
	h := strings.ToLower(c.Hex())
	if h[1] == h[2] && h[3] == h[4] && h[5] == h[6] {
		alts = append(alts, regexp.QuoteMeta("#"+string(h[1])+string(h[3])+string(h[5])))
	}
	alts = append(alts, fmt.Sprintf(`rgba?\(\s*%d\s*,\s*%d\s*,\s*%d\s*(,\s*1(\.0+)?\s*)?\)`, c.R, c.G, c.B))
	for name, nc := range namedColors {
		if nc == c {
			alts = append(alts, `\b`+name+`\b`)
		}
	}
	return regexp.MustCompile(`(?i)(` + strings.Join(alts, "|") + `)`)
}

func fixColor(d *doc, n *mjml.Node, target, from, to string) (string, error) {
	attr := "color"
	if target == "background" {
		attr = "background-color"
		if n.Tag == "mj-text" {
			attr = "container-background-color"
		}
	}
	out, ok := setNodeAttr(d.src, n, attr, to)
	if !ok {
		return d.src, ErrFixTarget
	}
	// Text colored inside the content (a link's or a span's inline color)
	// changes too.
	if from != "" && n.Tag != "mj-button" {
		d2, err := parseDoc(out)
		if err != nil {
			return out, nil
		}
		n2 := d2.nodes[pathKeyOf(d, n)]
		if n2 == nil {
			return out, nil
		}
		s, e, ok := innerRange(out, n2)
		if ok {
			prop := "color"
			if target == "background" {
				prop = "background(-color)?"
			}
			re := regexp.MustCompile(`(?i)((?:^|[;"'\s])` + prop + `\s*:\s*)` + colorPattern(from).String())
			content := re.ReplaceAllString(out[s:e], "${1}"+to)
			out = out[:s] + content + out[e:]
		}
	}
	return out, nil
}

// pathKeyOf finds a node's path key in its document.
func pathKeyOf(d *doc, n *mjml.Node) string {
	for k, x := range d.nodes {
		if x == n {
			return k
		}
	}
	return ""
}

var fontSizeDecl = regexp.MustCompile(`(?i)(font-size\s*:\s*)(\d+(?:\.\d+)?)(px|pt)`)

// mjmlFontSize is MJML's own default font size of the text components.
const mjmlFontSize = 13

// defaultFontSize is the size an element gets when it sets none: the
// head's mj-attributes for its tag or mj-all, then MJML's default.
func defaultFontSize(d *doc, tag string) float64 {
	size := float64(mjmlFontSize)
	head := mjml.Head(d.root)
	if head == nil {
		return size
	}
	for _, c := range head.Children {
		if c.Tag != "mj-attributes" {
			continue
		}
		for _, a := range c.Children {
			if (a.Tag == "mj-all" || a.Tag == tag) && a.Attr("font-size") != "" {
				if v, ok := cssLength(a.Attr("font-size"), size); ok {
					size = v
				}
			}
		}
	}
	return size
}

func fixTextSize(d *doc, n *mjml.Node, to int) (string, error) {
	out := d.src
	if n.Tag != "mj-raw" {
		size, ok := cssLength(n.Attr("font-size"), defaultFontSize(d, n.Tag))
		if !ok {
			size = defaultFontSize(d, n.Tag)
		}
		if size < float64(to) {
			var set bool
			out, set = setNodeAttr(out, n, "font-size", strconv.Itoa(to)+"px")
			if !set {
				return d.src, ErrFixTarget
			}
		}
	}
	if !endingTag(n.Tag) {
		return out, nil
	}
	d2, err := parseDoc(out)
	if err != nil {
		return out, nil
	}
	n2 := d2.nodes[pathKeyOf(d, n)]
	if n2 == nil {
		return out, nil
	}
	s, e, ok := innerRange(out, n2)
	if !ok {
		return out, nil
	}
	content := fontSizeDecl.ReplaceAllStringFunc(out[s:e], func(m string) string {
		p := fontSizeDecl.FindStringSubmatch(m)
		v, _ := strconv.ParseFloat(p[2], 64)
		if strings.EqualFold(p[3], "pt") {
			v = v * 4 / 3
		}
		if v >= float64(to) {
			return m
		}
		return p[1] + strconv.Itoa(to) + "px"
	})
	return out[:s] + content + out[e:], nil
}

func endingTag(tag string) bool {
	switch tag {
	case "mj-text", "mj-button", "mj-raw", "mj-table", "mj-navbar-link", "mj-social-element", "mj-accordion-text", "mj-accordion-title":
		return true
	}
	return false
}

// UA defaults of each heading level, in em: font size and vertical margin.
var headingDefaults = map[int][2]string{1: {"2em", "0.67em"}, 2: {"1.5em", "0.83em"}, 3: {"1.17em", "1em"}, 4: {"1em", "1.33em"}, 5: {"0.83em", "1.67em"}, 6: {"0.67em", "2.33em"}}

// fixHeadings changes heading levels. The text keeps its look: a heading
// that relied on the old level's default size or margin gets them written
// out.
func fixHeadings(d *doc, changes []AccessibilityHeadingChange) (string, error) {
	if len(changes) == 0 {
		return d.src, errors.New("no heading to change")
	}
	byPath := map[string][]AccessibilityHeadingChange{}
	var order []string
	for _, c := range changes {
		if c.From < 1 || c.From > 6 || c.To < 1 || c.To > 6 {
			return d.src, fmt.Errorf("heading levels go from 1 to 6")
		}
		k := pathKey(c.Path)
		if _, ok := byPath[k]; !ok {
			order = append(order, k)
		}
		byPath[k] = append(byPath[k], c)
	}
	var all []edit
	for _, k := range order {
		n := d.nodes[k]
		if n == nil || !endingTag(n.Tag) {
			return d.src, ErrFixTarget
		}
		s, e, ok := innerRange(d.src, n)
		if !ok {
			return d.src, ErrFixTarget
		}
		content := d.src[s:e]
		els := elementsOf(content)
		for _, c := range byPath[k] {
			h := nth(els, func(x *htmlElement) bool {
				return len(x.tag) == 2 && x.tag[0] == 'h' && x.tag[1] >= '1' && x.tag[1] <= '6'
			}, c.Index)
			if h == nil || h.tag != "h"+strconv.Itoa(c.From) {
				return d.src, ErrFixTarget
			}
			newTag := "h" + strconv.Itoa(c.To)
			all = append(all, edit{s + h.start + 1, s + h.nameEnd, newTag})
			if h.closeStart >= 0 {
				all = append(all, edit{s + h.closeStart + 2, s + h.closeStart + 2 + len(h.tag), newTag})
			}
			// Keep the look of the old level where the style doesn't set it.
			style := ""
			_, attrs, _, _ := openTag(content, h.start)
			for _, a := range attrs {
				if a.name == "style" {
					style = content[a.valStart:a.valEnd]
				}
			}
			decls := declarations(style)
			has := func(p string) bool {
				for _, dcl := range decls {
					if dcl[0] == p || (p == "margin" && strings.HasPrefix(dcl[0], "margin")) || (p == "font-size" && dcl[0] == "font") {
						return true
					}
				}
				return false
			}
			var extra []string
			def := headingDefaults[c.From]
			if !has("font-size") {
				extra = append(extra, "font-size:"+def[0])
			}
			if !has("margin") {
				extra = append(extra, "margin:"+def[1]+" 0")
			}
			if len(extra) > 0 {
				next := strings.TrimSpace(style)
				if next != "" && !strings.HasSuffix(next, ";") {
					next += ";"
				}
				next += strings.Join(extra, ";")
				if ed, ok := setAttrEdit(content, h.start, "style", next); ok {
					ed.start += s
					ed.end += s
					all = append(all, ed)
				}
			}
		}
	}
	return applyEdits(d.src, all), nil
}
