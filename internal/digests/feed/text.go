package feed

import (
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Excerpt turns a feed's HTML (a description, a summary, content) into
// plain text: tags, scripts and styles go, entities are decoded and
// whitespace collapses. WordPress's "The post … appeared first on …"
// footer and a trailing "[…]" are dropped.
func Excerpt(src string) string {
	if !strings.ContainsAny(src, "<&") {
		return trimFooter(collapse(src))
	}
	z := xhtml.NewTokenizer(strings.NewReader(src))
	var b strings.Builder
	skip := 0
	for {
		tt := z.Next()
		switch tt {
		case xhtml.ErrorToken:
			return trimFooter(collapse(b.String()))
		case xhtml.TextToken:
			if skip == 0 {
				b.Write(z.Text())
			}
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			name, _ := z.TagName()
			switch atom.Lookup(name) {
			case atom.Script, atom.Style, atom.Noscript, atom.Template, atom.Iframe, atom.Svg:
				if tt == xhtml.StartTagToken {
					skip++
				}
			case atom.Br, atom.P, atom.Div, atom.Li, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Blockquote, atom.Tr, atom.Figcaption:
				b.WriteByte(' ')
			}
		case xhtml.EndTagToken:
			name, _ := z.TagName()
			switch atom.Lookup(name) {
			case atom.Script, atom.Style, atom.Noscript, atom.Template, atom.Iframe, atom.Svg:
				if skip > 0 {
					skip--
				}
			case atom.P, atom.Div, atom.Li, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Blockquote, atom.Td:
				b.WriteByte(' ')
			}
		}
		if b.Len() > 64<<10 {
			return trimFooter(collapse(b.String()))
		}
	}
}

var (
	footerRe   = regexp.MustCompile(`(?i)\s*The post .{1,300}? appeared first on .{1,200}?\.?\s*$`)
	ellipsisRe = regexp.MustCompile(`\s*(\[(…|\.\.\.|&hellip;)\]|\[\s*\.\.\.\s*\])\s*$`)
)

func trimFooter(s string) string {
	s = footerRe.ReplaceAllString(s, "")
	s = ellipsisRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// collapse joins runs of whitespace into single spaces.
func collapse(s string) string {
	var b strings.Builder
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) || r == ' ' {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteRune(r)
	}
	return b.String()
}

// cleanText is a title or a name: entities decoded (some feeds escape
// twice), tags dropped, whitespace collapsed.
func cleanText(s string) string {
	if strings.ContainsAny(s, "<&") {
		s = Excerpt(html.UnescapeString(s))
	}
	return collapse(s)
}

var imgRe = regexp.MustCompile(`(?is)<img\b[^>]*>`)

// firstImage returns the first image of an HTML fragment and its alt text.
// Tracking pixels (1×1) and emoji images are skipped.
func firstImage(src string, base *url.URL) (string, string) {
	for _, tag := range imgRe.FindAllString(src, 20) {
		z := xhtml.NewTokenizer(strings.NewReader(tag))
		z.Next()
		var srcAttr, alt, width, height, class string
		for {
			k, v, more := z.TagAttr()
			switch string(k) {
			case "src":
				srcAttr = string(v)
			case "data-src":
				if srcAttr == "" {
					srcAttr = string(v)
				}
			case "alt":
				alt = string(v)
			case "width":
				width = string(v)
			case "height":
				height = string(v)
			case "class":
				class = string(v)
			}
			if !more {
				break
			}
		}
		if srcAttr == "" || width == "1" || height == "1" || strings.Contains(class, "emoji") || strings.Contains(srcAttr, "/emoji/") {
			continue
		}
		if u := resolve(base, html.UnescapeString(srcAttr)); u != "" {
			return u, collapse(alt)
		}
	}
	return "", ""
}

// Words shortens text to at most n words, ending with "…" when it was cut.
func Words(text string, n int) string {
	if n <= 0 {
		return ""
	}
	fields := strings.Fields(text)
	if len(fields) <= n {
		return strings.Join(fields, " ")
	}
	out := strings.Join(fields[:n], " ")
	return strings.TrimRight(out, ".,;:!?—–-") + "…"
}
