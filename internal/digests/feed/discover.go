package feed

import (
	"bytes"
	"net/url"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// FeedLink is a feed a page links to in its head.
type FeedLink struct {
	URL   string
	Title string
	// Kind is the feed's kind from the link's type (rss, atom, json).
	Kind string
}

var feedTypes = map[string]string{
	"application/rss+xml":   KindRSS,
	"application/atom+xml":  KindAtom,
	"application/feed+json": KindJSON,
	"application/json":      KindJSON,
	"application/rdf+xml":   KindRSS,
}

// Discover lists the feeds a page links to with <link rel="alternate">,
// in the page's order. Comment feeds come last: WordPress links a site's
// comments feed right after its posts feed.
func Discover(page []byte, base *url.URL) []FeedLink {
	z := xhtml.NewTokenizer(bytes.NewReader(page))
	var out, comments []FeedLink
	for {
		tt := z.Next()
		if tt == xhtml.ErrorToken {
			break
		}
		if tt != xhtml.StartTagToken && tt != xhtml.SelfClosingTagToken {
			continue
		}
		name, hasAttr := z.TagName()
		a := atom.Lookup(name)
		if a == atom.Body {
			break // feeds are linked in the head
		}
		if a != atom.Link || !hasAttr {
			continue
		}
		var rel, typ, href, title string
		for {
			k, v, more := z.TagAttr()
			switch string(k) {
			case "rel":
				rel = strings.ToLower(string(v))
			case "type":
				typ = strings.ToLower(strings.TrimSpace(string(v)))
			case "href":
				href = string(v)
			case "title":
				title = string(v)
			}
			if !more {
				break
			}
		}
		kind, ok := feedTypes[typ]
		if !ok || !containsWord(rel, "alternate") || href == "" {
			continue
		}
		u := resolve(base, href)
		if u == "" {
			continue
		}
		l := FeedLink{URL: u, Title: collapse(title), Kind: kind}
		if strings.Contains(strings.ToLower(title), "comment") || strings.Contains(strings.ToLower(u), "/comments/") {
			comments = append(comments, l)
		} else {
			out = append(out, l)
		}
	}
	return append(out, comments...)
}

func containsWord(s, word string) bool {
	for _, f := range strings.Fields(s) {
		if f == word {
			return true
		}
	}
	return false
}
