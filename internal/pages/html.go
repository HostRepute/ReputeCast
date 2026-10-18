package pages

import (
	"bytes"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// A page is stored as the HTML of its blocks (the page builder's GrapesJS
// components), checked against this allow-list when it's saved: only the
// elements and attributes the blocks use survive. There are no styles
// (blocks only offer the theme's colours, fonts and spacing, as classes),
// no scripts, frames or forms (a signup form, a product and the other
// blocks that show live data are custom elements the site draws), and
// links and images are http(s) or on the site. The public site runs under
// a strict Content-Security-Policy as well.

// elements are the allowed elements and their attributes beyond the
// common ones.
var elements = map[string][]string{
	"section": nil, "div": nil, "span": nil, "p": nil, "h1": nil, "h2": nil, "h3": nil, "h4": nil,
	"ul": nil, "ol": {"start"}, "li": nil, "a": {"href", "title", "target", "rel"},
	"strong": nil, "b": nil, "em": nil, "i": nil, "u": nil, "s": nil, "br": nil, "small": nil, "sup": nil, "sub": nil,
	"img":    {"src", "alt", "width", "height", "loading", "title"},
	"figure": nil, "figcaption": nil, "blockquote": nil, "details": {"open"}, "summary": nil, "hr": nil,
	"time": {"datetime"},
}

// common are the attributes every allowed element may carry: classes (the
// theme's), an id (anchors such as #buy), the block's kind, id and name in
// the layers, and accessible names.
var common = []string{"class", "id", "data-pb", "data-pb-id", "data-pb-name", "data-pb-bg", "data-pb-space", "aria-label", "aria-hidden", "role", "lang", "dir"}

var (
	classRe = regexp.MustCompile(`^[a-z][a-z0-9-]{0,40}$`)
	idRe    = regexp.MustCompile(`^[a-z][a-z0-9-]{0,60}$`)
	roleRe  = regexp.MustCompile(`^(img|list|listitem|note|presentation|region|group|timer)$`)
	wordRe  = regexp.MustCompile(`^[a-z0-9_:-]{0,80}$`)
)

// voids never have content.
var voids = map[string]bool{"br": true, "img": true, "hr": true}

// dropWithContent are elements removed together with everything inside.
var dropWithContent = map[string]bool{"script": true, "style": true, "iframe": true, "object": true, "embed": true, "template": true,
	"noscript": true, "form": true, "input": true, "button": true, "select": true, "textarea": true, "svg": true, "math": true,
	"video": true, "audio": true, "canvas": true, "head": true, "title": true, "meta": true, "link": true, "base": true}

// Sanitize keeps what the page blocks may contain. Unknown elements are
// unwrapped (their text stays); dangerous ones are dropped with their
// content. It is idempotent.
func Sanitize(src string) string {
	nodes, err := html.ParseFragment(strings.NewReader(src), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return ""
	}
	var buf bytes.Buffer
	for _, n := range nodes {
		writeClean(&buf, n)
	}
	return strings.TrimSpace(buf.String())
}

func writeClean(buf *bytes.Buffer, n *html.Node) {
	switch n.Type {
	case html.TextNode:
		buf.WriteString(html.EscapeString(n.Data))
		return
	case html.ElementNode:
	default:
		// Comments, doctypes: dropped.
		return
	}
	name := strings.ToLower(n.Data)
	if dropWithContent[name] {
		return
	}
	spec, isBlock := blockSpecs[name]
	extra, known := elements[name]
	if !known && !isBlock {
		// Unwrapped: the text inside stays.
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			writeClean(buf, c)
		}
		return
	}
	buf.WriteByte('<')
	buf.WriteString(name)
	seen := map[string]bool{}
	type kv struct{ k, v string }
	var out []kv
	for _, a := range n.Attr {
		key := strings.ToLower(a.Key)
		if a.Namespace != "" || seen[key] {
			continue
		}
		val, ok := cleanAttr(name, key, a.Val, extra, spec, isBlock)
		if !ok {
			continue
		}
		seen[key] = true
		out = append(out, kv{key, val})
	}
	if name == "a" && seen["target"] && !seen["rel"] {
		out = append(out, kv{"rel", "noopener noreferrer"})
	}
	// Attributes in the order they came, except rel right after target, so
	// a second pass writes the same bytes.
	slices.SortStableFunc(out, func(a, b kv) int { return attrRank(a.k) - attrRank(b.k) })
	for _, a := range out {
		buf.WriteByte(' ')
		buf.WriteString(a.k)
		buf.WriteString(`="`)
		buf.WriteString(html.EscapeString(a.v))
		buf.WriteByte('"')
	}
	buf.WriteByte('>')
	if voids[name] {
		return
	}
	if !isBlock {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			writeClean(buf, c)
		}
	}
	buf.WriteString("</")
	buf.WriteString(name)
	buf.WriteByte('>')
}

// attrRank puts target and rel last, in that order, whatever order they came in.
func attrRank(k string) int {
	switch k {
	case "target":
		return 1
	case "rel":
		return 2
	}
	return 0
}

func cleanAttr(el, key, val string, extra []string, spec blockSpec, isBlock bool) (string, bool) {
	val = strings.TrimSpace(val)
	if strings.HasPrefix(key, "on") || key == "style" {
		return "", false
	}
	if isBlock {
		if key == "data-pb-id" || key == "data-pb-name" || key == "class" || key == "id" {
			return commonAttr(key, val)
		}
		return spec.clean(key, val)
	}
	allowed := false
	for _, k := range common {
		if k == key {
			allowed = true
			break
		}
	}
	if allowed {
		return commonAttr(key, val)
	}
	for _, k := range extra {
		if k != key {
			continue
		}
		switch key {
		case "href":
			return SafeURL(val, true)
		case "src":
			return SafeURL(val, false)
		case "target":
			return "_blank", val == "_blank"
		case "rel":
			return "noopener noreferrer", true
		case "width", "height", "start":
			if n := digits(val); n != "" && len(n) <= 5 {
				return n, true
			}
			return "", false
		case "loading":
			return "lazy", val == "lazy"
		case "open":
			return "", true
		case "datetime":
			return val, len(val) <= 40 && wordRe.MatchString(strings.ToLower(strings.NewReplacer("+", "", ".", "", " ", "").Replace(val)))
		default:
			return clip(val, 300), true
		}
	}
	return "", false
}

func commonAttr(key, val string) (string, bool) {
	switch key {
	case "class":
		var keep []string
		for _, c := range strings.Fields(val) {
			if classRe.MatchString(c) && !strings.HasPrefix(c, "gjs-") {
				keep = append(keep, c)
			}
		}
		if len(keep) == 0 {
			return "", false
		}
		return strings.Join(keep, " "), true
	case "id", "data-pb-id":
		v := strings.ToLower(val)
		return v, idRe.MatchString(v)
	case "data-pb", "data-pb-bg", "data-pb-space":
		v := strings.ToLower(val)
		return v, v != "" && wordRe.MatchString(v)
	case "role":
		return val, roleRe.MatchString(val)
	case "aria-hidden":
		return "true", val == "true"
	case "lang", "dir":
		return val, len(val) <= 12 && wordRe.MatchString(strings.ToLower(val))
	default:
		return clip(val, 300), true
	}
}

// SafeURL keeps http(s) addresses and paths on the site (and mailto: and
// tel: for links); anything else ("javascript:", "data:") is refused.
func SafeURL(raw string, link bool) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 2000 || strings.ContainsAny(raw, "\x00\r\n\t<>\"\\") {
		return "", false
	}
	if strings.HasPrefix(raw, "#") {
		return raw, link && idRe.MatchString(strings.TrimPrefix(raw, "#"))
	}
	if strings.HasPrefix(raw, "/") {
		return raw, !strings.HasPrefix(raw, "//")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	switch strings.ToLower(u.Scheme) {
	case "https", "http":
		return raw, u.Host != ""
	case "mailto", "tel":
		return raw, link
	}
	return "", false
}

func digits(s string) string {
	s = strings.TrimSuffix(strings.TrimSpace(s), "px")
	for _, r := range s {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return s
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Don't cut a UTF-8 sequence in half.
	for n > 0 && n < len(s) && s[n]&0xC0 == 0x80 {
		n--
	}
	return s[:n]
}

// parse reads stored page HTML into nodes (a fragment in a div).
func parse(src string) []*html.Node {
	nodes, err := html.ParseFragment(strings.NewReader(src), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return nil
	}
	return nodes
}

// attr reads an attribute of an element.
func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClass(n *html.Node, class string) bool {
	for _, c := range strings.Fields(attr(n, "class")) {
		if c == class {
			return true
		}
	}
	return false
}

// textOf is the text inside a node, spaces collapsed.
func textOf(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}

// walk visits every element node under the nodes, depth first.
func walk(nodes []*html.Node, fn func(n *html.Node) bool) {
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && !fn(n) {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	for _, n := range nodes {
		visit(n)
	}
}
