package publicpages_test

import (
	"fmt"
	"net/url"
	"path"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// headFacts is what a page's markup says to browsers and search engines.
type headFacts struct {
	lang, title, robots, description, canonical string
	viewport                                    bool
	h1                                          int
	imgsWithoutAlt                              []string
	links                                       []string
}

func readHead(body string) headFacts {
	var f headFacts
	doc, err := html.Parse(strings.NewReader(body))
	if err != nil {
		return f
	}
	attr := func(n *html.Node, key string) (string, bool) {
		for _, a := range n.Attr {
			if strings.EqualFold(a.Key, key) {
				return a.Val, true
			}
		}
		return "", false
	}
	var text func(n *html.Node) string
	text = func(n *html.Node) string {
		if n.Type == html.TextNode {
			return n.Data
		}
		var b strings.Builder
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			b.WriteString(text(c))
		}
		return b.String()
	}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "html":
				f.lang, _ = attr(n, "lang")
			case "title":
				f.title = strings.TrimSpace(text(n))
			case "meta":
				name, _ := attr(n, "name")
				content, _ := attr(n, "content")
				switch strings.ToLower(name) {
				case "robots":
					f.robots = content
				case "description":
					f.description = strings.TrimSpace(content)
				case "viewport":
					f.viewport = strings.Contains(content, "width=device-width")
				}
			case "link":
				if rel, _ := attr(n, "rel"); rel == "canonical" {
					f.canonical, _ = attr(n, "href")
				}
			case "h1":
				f.h1++
			case "img":
				if _, ok := attr(n, "alt"); !ok {
					src, _ := attr(n, "src")
					f.imgsWithoutAlt = append(f.imgsWithoutAlt, src)
				}
			case "a":
				if href, ok := attr(n, "href"); ok {
					f.links = append(f.links, href)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return f
}

// Every public page a reader or a search engine can reach from the
// publication's front doors says its language, has a title, fits a phone's
// width and says whether it may be indexed. A page that may be indexed also
// has a description, a canonical address and one main heading, and every
// image says what it shows (or that it shows nothing, alt=""). The crawl
// follows the site's own links from the home page, the subscribe page and
// the archive on the demo's custom domain.
func TestPublicPagesHead(t *testing.T) {
	env, _ := demo(t)
	const host = "news.acme.io"
	queue := []string{"/", "/subscribe", "/archive"}
	seen := map[string]bool{}
	indexable, checked := 0, 0
	for len(queue) > 0 && checked < 120 {
		p := queue[0]
		queue = queue[1:]
		if seen[p] {
			continue
		}
		seen[p] = true
		r := do(t, env, "GET", p, nil, hdr{"Host": host})
		if r.Status != 200 || !strings.HasPrefix(r.Header.Get("Content-Type"), "text/html") {
			continue
		}
		checked++
		f := readHead(r.Body)
		problems := []string{}
		if f.lang == "" {
			problems = append(problems, "no lang on <html>")
		}
		if f.title == "" {
			problems = append(problems, "no title")
		}
		if !f.viewport {
			problems = append(problems, "no width=device-width viewport")
		}
		if f.robots == "" && !strings.Contains(r.Header.Get("X-Robots-Tag"), "noindex") {
			problems = append(problems, "doesn't say whether it may be indexed")
		}
		if len(f.imgsWithoutAlt) > 0 {
			problems = append(problems, "images without alt: "+strings.Join(f.imgsWithoutAlt, ", "))
		}
		if strings.HasPrefix(f.robots, "index") {
			indexable++
			if f.description == "" {
				problems = append(problems, "indexable without a description")
			}
			if u, err := url.Parse(f.canonical); err != nil || !u.IsAbs() {
				problems = append(problems, "indexable without an absolute canonical address")
			}
			if f.h1 != 1 {
				problems = append(problems, fmt.Sprintf("indexable with %d main headings, want one", f.h1))
			}
		}
		if len(problems) > 0 {
			t.Errorf("%s (%q): %s", p, f.title, strings.Join(problems, "; "))
		}
		for _, href := range f.links {
			u, err := url.Parse(href)
			if err != nil || (u.Host != "" && u.Host != host) || (u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https") {
				continue
			}
			next := u.Path
			if !strings.HasPrefix(next, "/") {
				next = path.Join(path.Dir(p), next)
			}
			if u.RawQuery != "" {
				next += "?" + u.RawQuery
			}
			switch {
			case next == "", strings.HasPrefix(next, "/p/_/"), strings.HasPrefix(next, "/media/"), strings.HasPrefix(next, "/t/"),
				strings.HasSuffix(next, ".rss"), strings.HasSuffix(next, ".xml"), strings.HasSuffix(next, ".css"), strings.HasSuffix(next, ".js"):
				continue
			}
			if !seen[next] {
				queue = append(queue, next)
			}
		}
	}
	if checked < 20 || indexable < 10 {
		t.Fatalf("the crawl reached %d pages, %d of them indexable: too few to say much", checked, indexable)
	}
	t.Logf("%d public pages checked, %d of them indexable", checked, indexable)
}
