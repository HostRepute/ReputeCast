package contentcheck

import (
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	spaceRe     = regexp.MustCompile(`[ \t\f\r\x{00a0}]+`)
	blankLineRe = regexp.MustCompile(`\n{3,}`)
	fontSizeRe  = regexp.MustCompile(`font-size:\s*(\d+(?:\.\d+)?)px`)
	fontWeight  = regexp.MustCompile(`font-weight:\s*(bold|[6-9]00)`)
)

var blockTags = map[atom.Atom]bool{
	atom.P: true, atom.Div: true, atom.Tr: true, atom.Table: true, atom.Li: true, atom.Ul: true, atom.Ol: true,
	atom.H1: true, atom.H2: true, atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true, atom.Blockquote: true,
	atom.Section: true, atom.Article: true, atom.Header: true, atom.Footer: true, atom.Center: true,
}

// PlainText turns email HTML into the text/plain part: one block per line,
// headings in capitals, buttons and standalone links as "Label: URL", the
// footer after a "--" separator, hidden preview text and comments dropped.
func PlainText(src string) string {
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return ""
	}
	var b strings.Builder
	// The footer is the last section, when it holds the unsubscribe link.
	var footer *html.Node
	if c := SectionContainer(doc); c != nil {
		var last *html.Node
		for x := c.FirstChild; x != nil; x = x.NextSibling {
			if x.Type == html.ElementNode {
				last = x
			}
		}
		if last != nil && unsubscribeText.MatchString(textOf(last)+linkHrefs(last)) {
			footer = last
		}
	}
	var walk func(n *html.Node, upper bool)
	newline := func() {
		s := b.String()
		if s != "" && !strings.HasSuffix(s, "\n") {
			b.WriteString("\n")
		}
	}
	paragraph := func() {
		newline()
		s := b.String()
		if s != "" && !strings.HasSuffix(s, "\n\n") {
			b.WriteString("\n")
		}
	}
	walk = func(n *html.Node, upper bool) {
		switch n.Type {
		case html.TextNode:
			t := spaceRe.ReplaceAllString(strings.ReplaceAll(n.Data, "\n", " "), " ")
			if strings.TrimSpace(t) == "" {
				if b.Len() > 0 && !strings.HasSuffix(b.String(), " ") && !strings.HasSuffix(b.String(), "\n") {
					b.WriteString(" ")
				}
				return
			}
			if strings.HasSuffix(b.String(), "\n") || b.Len() == 0 {
				t = strings.TrimLeft(t, " ")
			}
			if upper {
				t = strings.ToUpper(t)
			}
			b.WriteString(t)
			return
		case html.CommentNode:
			return
		case html.ElementNode:
			switch n.DataAtom {
			case atom.Head, atom.Style, atom.Script, atom.Title:
				return
			case atom.Br:
				newline()
				return
			case atom.Img:
				return
			}
			style := attr(n, "style")
			if strings.Contains(strings.ReplaceAll(style, " ", ""), "display:none") {
				return // the hidden preview text
			}
			if n.DataAtom == atom.A {
				href := attr(n, "href")
				label := strings.TrimSpace(spaceRe.ReplaceAllString(textOf(n), " "))
				if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "mailto:") && label == strings.TrimPrefix(href, "mailto:") {
					break
				}
				if label == "" {
					return
				}
				if strings.Contains(href, "{{") || isStandalone(n) {
					newline()
					b.WriteString(label + ": " + href)
					newline()
					return
				}
				b.WriteString(label + " (" + href + ")")
				return
			}
			heading := n.DataAtom == atom.H1 || n.DataAtom == atom.H2 || n.DataAtom == atom.H3
			if m := fontSizeRe.FindStringSubmatch(style); m != nil {
				if size, _ := strconv.ParseFloat(m[1], 64); size >= 16 && fontWeight.MatchString(style) {
					heading = true
				}
			}
			if n == footer {
				paragraph()
				b.WriteString("--\n")
			} else if blockTags[n.DataAtom] {
				paragraph()
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c, upper || heading)
			}
			if blockTags[n.DataAtom] {
				paragraph()
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, upper)
		}
	}
	walk(doc, false)
	lines := strings.Split(b.String(), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	out := blankLineRe.ReplaceAllString(strings.Join(lines, "\n"), "\n\n")
	out = strings.ReplaceAll(out, "--\n\n", "--\n")
	return strings.TrimSpace(out)
}

var unsubscribeText = regexp.MustCompile(`(?i)unsubscribe`)

// SectionContainer returns the element whose children are the email's
// top-level sections in MJML output (body > div > div…), or nil.
func SectionContainer(doc *html.Node) *html.Node {
	var body *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if body != nil {
			return
		}
		if n.Type == html.ElementNode && n.DataAtom == atom.Body {
			body = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			find(c)
		}
	}
	find(doc)
	if body == nil {
		return nil
	}
	cur := body
	for {
		var only *html.Node
		count := 0
		for c := cur.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && !hidden(c) {
				count++
				only = c
			}
		}
		if count == 1 && only.DataAtom == atom.Div {
			cur = only
			continue
		}
		if count == 0 {
			return nil
		}
		return cur
	}
}

func hidden(n *html.Node) bool {
	return strings.Contains(strings.ReplaceAll(attr(n, "style"), " ", ""), "display:none")
}

func linkHrefs(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.ElementNode && x.DataAtom == atom.A {
			b.WriteString(" " + attr(x, "href"))
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// isStandalone reports whether a link is the only content of its block
// (a button, a "Read more" line), as opposed to a link inside a sentence.
func isStandalone(a *html.Node) bool {
	p := a.Parent
	for p != nil && p.Type == html.ElementNode && !blockTags[p.DataAtom] && p.DataAtom != atom.Td {
		p = p.Parent
	}
	if p == nil {
		return true
	}
	return strings.TrimSpace(textOf(p)) == strings.TrimSpace(textOf(a))
}

func attr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

func hasAttr(n *html.Node, name string) bool {
	for _, a := range n.Attr {
		if a.Key == name {
			return true
		}
	}
	return false
}

func textOf(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.TextNode {
			b.WriteString(x.Data)
		}
		if x.Type == html.ElementNode && (x.DataAtom == atom.Style || x.DataAtom == atom.Script) {
			return
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

var (
	interTagSpace = regexp.MustCompile(`>\s+<`)
	commentRe     = regexp.MustCompile(`<!--[\s\S]*?-->`)
	styleAttrRe   = regexp.MustCompile(`style="([^"]*)"`)
)

// Minify shrinks email HTML without changing how it renders: whitespace
// between tags and inside style attributes, and comments other than
// Outlook's conditional comments.
func Minify(src string) string {
	out := commentRe.ReplaceAllStringFunc(src, func(c string) string {
		if strings.HasPrefix(c, "<!--[if") || strings.HasPrefix(c, "<!--<![endif]") || strings.Contains(c, "<![endif]") {
			return c
		}
		// Conditional blocks' markers (internal/visibility cuts each
		// recipient's version at them).
		if strings.HasPrefix(c, "<!--rc:if ") || strings.HasPrefix(c, "<!--rc:fi ") {
			return c
		}
		return ""
	})
	out = interTagSpace.ReplaceAllString(out, "><")
	out = styleAttrRe.ReplaceAllStringFunc(out, func(s string) string {
		v := s[len(`style="`) : len(s)-1]
		parts := strings.Split(v, ";")
		kept := parts[:0]
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if i := strings.IndexByte(p, ':'); i > 0 {
				p = strings.TrimSpace(p[:i]) + ":" + strings.TrimSpace(p[i+1:])
			}
			kept = append(kept, p)
		}
		return `style="` + strings.Join(kept, ";") + `"`
	})
	return strings.TrimSpace(out)
}
