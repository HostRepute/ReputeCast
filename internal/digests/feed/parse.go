// Package feed reads the feeds recurring digests are built from: RSS 2.0
// (and the RSS 1.0 and 0.9x it grew from), Atom 1.0 and JSON Feed 1.0 and
// 1.1. It parses with the standard library's XML and JSON decoders and
// golang.org/x/net/html (already a dependency) rather than a feed library:
// a digest needs a dozen fields per item, and parsing them ourselves keeps
// the limits (size, items, nesting) and the failure messages under our
// control. It also finds the feed a web page links to (discover.go) and
// fetches feeds through the SSRF guard (fetch.go).
package feed

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html/charset"
)

// Kinds of feed.
const (
	KindRSS  = "rss"
	KindAtom = "atom"
	KindJSON = "json"
)

// MaxItems is how many items of a feed are kept, newest first. A feed's
// back catalogue can be long; a digest only ever looks at its newest posts.
const MaxItems = 200

// Feed is a parsed feed.
type Feed struct {
	Kind  string
	Title string
	// Link is the site the feed belongs to.
	Link  string
	Items []Item
}

// Item is one entry of a feed.
type Item struct {
	// Key identifies the item across reads: its guid or id, else its link,
	// so an edited post isn't sent twice.
	Key       string
	URL       string
	Title     string
	Author    string
	Published *time.Time
	// Categories and Tags are the item's terms. A term is a tag when the
	// feed says so (an RSS category domain or an Atom scheme naming a tag
	// taxonomy); JSON Feed tags are categories.
	Categories []string
	Tags       []string
	// Excerpt is plain text: HTML, scripts and styles are stripped.
	Excerpt string
	// Image is the item's picture (media, enclosure, else the first image
	// of its content) and ImageAlt that image's own alt text, when it has one.
	Image    string
	ImageAlt string
}

// NotFeedError means the document isn't a feed: an HTML page (a maintenance
// page, the site's home page), or XML or JSON of another kind.
type NotFeedError struct {
	// HTML is true when the document is a web page.
	HTML bool
	// Lines are the document's first lines, for showing what answered.
	Lines []string
	// Reason describes what was found instead ("an XML document whose root is <urlset>").
	Reason string
}

func (e *NotFeedError) Error() string {
	if e.HTML {
		return "the document is an HTML page, not a feed"
	}
	if e.Reason != "" {
		return "not a feed: " + e.Reason
	}
	return "not a feed"
}

// IsNotFeed reports whether err says the document isn't a feed.
func IsNotFeed(err error) bool {
	var nf *NotFeedError
	return errors.As(err, &nf)
}

// FirstLines returns up to n of the document's first non-empty lines, each
// at most 120 characters, for error messages.
func FirstLines(data []byte, n int) []string {
	out := []string{}
	for _, line := range strings.Split(string(data[:min(len(data), 4096)]), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\uFEFF"))
		if line == "" {
			continue
		}
		if !utf8.ValidString(line) {
			line = strings.ToValidUTF8(line, "�")
		}
		if r := []rune(line); len(r) > 120 {
			line = string(r[:119]) + "…"
		}
		out = append(out, line)
		if len(out) == n {
			break
		}
	}
	return out
}

// LooksLikeHTML reports whether a document is a web page.
func LooksLikeHTML(data []byte) bool {
	head := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(string(data[:min(len(data), 1024)]), "\uFEFF")))
	switch {
	case strings.HasPrefix(head, "<!doctype html"), strings.HasPrefix(head, "<html"):
		return true
	case strings.HasPrefix(head, "<!--") && strings.Contains(head, "<html"):
		return true
	}
	return strings.Contains(head, "<head") && strings.Contains(head, "<body")
}

// Parse reads a feed. base resolves relative links (the feed's own URL).
func Parse(data []byte, base *url.URL) (*Feed, error) {
	trimmed := bytes.TrimSpace(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")))
	if len(trimmed) == 0 {
		return nil, &NotFeedError{Reason: "the document is empty"}
	}
	var (
		f   *Feed
		err error
	)
	switch trimmed[0] {
	case '{':
		f, err = parseJSON(trimmed, base)
	case '<':
		if LooksLikeHTML(trimmed) {
			return nil, &NotFeedError{HTML: true, Lines: FirstLines(data, 4)}
		}
		f, err = parseXML(trimmed, base)
	default:
		return nil, &NotFeedError{Lines: FirstLines(data, 4), Reason: "the document is neither XML nor JSON"}
	}
	if err != nil {
		var nf *NotFeedError
		if errors.As(err, &nf) && nf.Lines == nil {
			nf.Lines = FirstLines(data, 4)
		}
		return nil, err
	}
	finish(f)
	return f, nil
}

// finish de-duplicates items by key, orders them newest first and keeps MaxItems.
func finish(f *Feed) {
	seen := map[string]bool{}
	items := f.Items[:0]
	for _, it := range f.Items {
		if it.Key == "" {
			it.Key = fallbackKey(it)
		}
		if it.Title == "" && it.URL == "" {
			continue
		}
		if seen[it.Key] {
			continue
		}
		seen[it.Key] = true
		items = append(items, it)
	}
	// Newest first; items without a date keep the feed's order after dated ones.
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i].Published, items[j].Published
		switch {
		case a == nil:
			return false
		case b == nil:
			return true
		}
		return a.After(*b)
	})
	if len(items) > MaxItems {
		items = items[:MaxItems]
	}
	f.Items = items
	f.Title = cleanText(f.Title)
}

func fallbackKey(it Item) string {
	h := sha256.New()
	h.Write([]byte(it.Title))
	if it.Published != nil {
		h.Write([]byte(it.Published.UTC().Format(time.RFC3339)))
	}
	return "hash:" + hex.EncodeToString(h.Sum(nil))[:24]
}

// ---- XML (RSS and Atom) ----

// node is an element of a small XML tree: feeds are small (5 MB at most)
// and their elements vary between producers, so they are read into a tree
// and searched by local name and namespace.
type node struct {
	Space, Name string
	Attr        []xml.Attr
	// Text is the element's own character data; Inner its whole content as
	// HTML-ish text in document order (Atom xhtml content keeps its words,
	// images and paragraph breaks this way).
	Text     strings.Builder
	Inner    strings.Builder
	Children []*node
}

// maxText bounds what is kept of one element's text.
const maxText = 1 << 20

// Namespaces of the elements read: the URI, and the usual prefix for feeds
// that use one without declaring it.
var (
	nsContent = []string{"http://purl.org/rss/1.0/modules/content/", "content"}
	nsDC      = []string{"http://purl.org/dc/elements/1.1/", "dc"}
	nsMedia   = []string{"http://search.yahoo.com/mrss/", "http://search.yahoo.com/mrss", "media"}
	nsAtom    = []string{"http://www.w3.org/2005/Atom", "atom", ""}
	// RSS elements: no namespace, or RSS 1.0's and 0.9's.
	nsRSS = []string{"", "http://purl.org/rss/1.0/", "http://my.netscape.com/rdf/simple/0.9/", "http://channel.netscape.com/rdf/simple/0.9/"}
)

func (n *node) attr(name string) string {
	if n == nil {
		return ""
	}
	for _, a := range n.Attr {
		if strings.EqualFold(a.Name.Local, name) {
			return strings.TrimSpace(a.Value)
		}
	}
	return ""
}

// child returns the first child with the local name in one of the namespaces.
func (n *node) child(name string, spaces []string) *node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if strings.EqualFold(c.Name, name) && spaceMatches(c.Space, spaces) {
			return c
		}
	}
	return nil
}

func (n *node) all(name string, spaces []string) []*node {
	if n == nil {
		return nil
	}
	var out []*node
	for _, c := range n.Children {
		if strings.EqualFold(c.Name, name) && spaceMatches(c.Space, spaces) {
			out = append(out, c)
		}
	}
	return out
}

func (n *node) text() string {
	if n == nil {
		return ""
	}
	return strings.TrimSpace(n.Text.String())
}

func spaceMatches(space string, want []string) bool {
	for _, w := range want {
		if strings.EqualFold(space, w) {
			return true
		}
	}
	return false
}

// maxDepth guards against pathologically nested documents.
const maxDepth = 64

// blockNames break text: their content reads as separate words.
var blockNames = map[string]bool{"p": true, "div": true, "br": true, "li": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"blockquote": true, "tr": true, "td": true, "figcaption": true}

func readTree(data []byte) (*node, error) {
	d := xml.NewDecoder(bytes.NewReader(data))
	// Feeds in the wild aren't always well-formed XML: unknown entities and
	// stray ampersands are tolerated. (No auto-closing: RSS's <link> has
	// content, unlike HTML's.)
	d.Strict = false
	d.Entity = xml.HTMLEntity
	d.CharsetReader = func(label string, r io.Reader) (io.Reader, error) {
		return charset.NewReaderLabel(label, r)
	}
	root := &node{}
	stack := []*node{root}
	inner := func(s string) {
		for _, n := range stack[1:] {
			if n.Inner.Len() < maxText {
				n.Inner.WriteString(s)
			}
		}
	}
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if len(root.Children) > 0 && len(stack) <= 2 {
				break // trailing garbage after the document
			}
			return nil, &NotFeedError{Reason: "the XML doesn't parse: " + err.Error()}
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if len(stack) > maxDepth {
				return nil, &NotFeedError{Reason: "the XML is nested too deeply"}
			}
			n := &node{Space: t.Name.Space, Name: t.Name.Local, Attr: t.Attr}
			name := strings.ToLower(t.Name.Local)
			switch {
			case name == "img":
				inner(fmt.Sprintf(` <img src="%s" alt="%s"> `, xmlAttrEscape(n.attr("src")), xmlAttrEscape(n.attr("alt"))))
			case blockNames[name]:
				inner(" ")
			}
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, n)
			stack = append(stack, n)
		case xml.EndElement:
			// Close the nearest open element of that name; an end tag that
			// matches nothing open is ignored.
			for i := len(stack) - 1; i > 0; i-- {
				if stack[i].Name == t.Name.Local {
					if blockNames[strings.ToLower(t.Name.Local)] {
						inner(" ")
					}
					stack = stack[:i]
					break
				}
			}
		case xml.CharData:
			top := stack[len(stack)-1]
			if top.Text.Len() < maxText {
				top.Text.Write(t)
			}
			inner(html.EscapeString(string(t)))
		}
	}
	if len(root.Children) == 0 {
		return nil, &NotFeedError{Reason: "the XML has no elements"}
	}
	return root.Children[0], nil
}

func parseXML(data []byte, base *url.URL) (*Feed, error) {
	root, err := readTree(data)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(root.Name) {
	case "rss":
		ch := root.child("channel", nsRSS)
		if ch == nil {
			return nil, &NotFeedError{Reason: "the RSS document has no channel"}
		}
		return parseRSS(ch, ch.all("item", nsRSS), base), nil
	case "rdf":
		// RSS 1.0: items are siblings of the channel.
		return parseRSS(root.child("channel", nsRSS), root.all("item", nsRSS), base), nil
	case "feed":
		return parseAtom(root, base), nil
	}
	return nil, &NotFeedError{Reason: fmt.Sprintf("an XML document whose root is <%s>", root.Name)}
}

func parseRSS(ch *node, items []*node, base *url.URL) *Feed {
	f := &Feed{Kind: KindRSS, Title: ch.child("title", nsRSS).text(), Link: resolve(base, ch.child("link", nsRSS).text())}
	for _, it := range items {
		item := Item{Title: cleanText(it.child("title", nsRSS).text())}
		item.URL = resolve(base, it.child("link", nsRSS).text())
		if item.URL == "" {
			// Some feeds only give an atom:link.
			if l := it.child("link", nsAtom); l != nil {
				item.URL = resolve(base, l.attr("href"))
			}
		}
		if g := it.child("guid", nsRSS); g != nil && g.text() != "" {
			item.Key = g.text()
			if item.URL == "" && !strings.EqualFold(g.attr("isPermaLink"), "false") && strings.HasPrefix(g.text(), "http") {
				item.URL = g.text()
			}
		} else if item.URL != "" {
			item.Key = item.URL
		}
		for _, c := range []*node{it.child("pubDate", nsRSS), it.child("date", nsDC), it.child("published", nsAtom), it.child("updated", nsAtom)} {
			if t, ok := ParseDate(c.text()); ok {
				item.Published = &t
				break
			}
		}
		item.Author = authorOf(it)
		for _, c := range it.all("category", nsRSS) {
			term := cleanText(c.text())
			if term == "" {
				continue
			}
			if isTagTaxonomy(c.attr("domain")) {
				item.Tags = appendUnique(item.Tags, term)
			} else {
				item.Categories = appendUnique(item.Categories, term)
			}
		}
		for _, c := range it.all("subject", nsDC) {
			if term := cleanText(c.text()); term != "" {
				item.Categories = appendUnique(item.Categories, term)
			}
		}
		content := it.child("encoded", nsContent).text()
		desc := it.child("description", nsRSS).text()
		summary := desc
		if summary == "" {
			summary = content
		}
		item.Excerpt = Excerpt(summary)
		item.Image, item.ImageAlt = mediaImage(it, base)
		if item.Image == "" {
			for _, enc := range it.all("enclosure", nsRSS) {
				if strings.HasPrefix(strings.ToLower(enc.attr("type")), "image/") {
					item.Image = resolve(base, enc.attr("url"))
					break
				}
			}
		}
		if item.Image == "" {
			item.Image, item.ImageAlt = firstImage(content+" "+desc, base)
		}
		f.Items = append(f.Items, item)
	}
	return f
}

func parseAtom(root *node, base *url.URL) *Feed {
	f := &Feed{Kind: KindAtom, Title: htmlTitle(root.child("title", nsAtom))}
	for _, l := range root.all("link", nsAtom) {
		if rel := strings.ToLower(l.attr("rel")); rel == "" || rel == "alternate" {
			f.Link = resolve(base, l.attr("href"))
			break
		}
	}
	feedAuthor := ""
	if a := root.child("author", nsAtom); a != nil {
		feedAuthor = cleanText(a.child("name", nsAtom).text())
	}
	for _, e := range root.all("entry", nsAtom) {
		item := Item{Title: cleanText(htmlTitle(e.child("title", nsAtom)))}
		for _, l := range e.all("link", nsAtom) {
			rel := strings.ToLower(l.attr("rel"))
			switch {
			case (rel == "" || rel == "alternate") && item.URL == "":
				item.URL = resolve(base, l.attr("href"))
			case rel == "enclosure" && strings.HasPrefix(strings.ToLower(l.attr("type")), "image/") && item.Image == "":
				item.Image = resolve(base, l.attr("href"))
			}
		}
		item.Key = e.child("id", nsAtom).text()
		if item.Key == "" {
			item.Key = item.URL
		}
		for _, name := range []string{"published", "updated", "issued", "modified"} {
			if t, ok := ParseDate(e.child(name, nsAtom).text()); ok {
				item.Published = &t
				break
			}
		}
		item.Author = feedAuthor
		if a := e.child("author", nsAtom); a != nil {
			if name := cleanText(a.child("name", nsAtom).text()); name != "" {
				item.Author = name
			}
		}
		for _, c := range e.all("category", nsAtom) {
			term := cleanText(c.attr("label"))
			if term == "" {
				term = cleanText(c.attr("term"))
			}
			if term == "" {
				continue
			}
			if isTagTaxonomy(c.attr("scheme")) {
				item.Tags = appendUnique(item.Tags, term)
			} else {
				item.Categories = appendUnique(item.Categories, term)
			}
		}
		summary := htmlText(e.child("summary", nsAtom))
		content := htmlText(e.child("content", nsAtom))
		if summary == "" {
			summary = content
		}
		item.Excerpt = Excerpt(summary)
		if item.Image == "" {
			item.Image, item.ImageAlt = mediaImage(e, base)
		}
		if item.Image == "" {
			item.Image, item.ImageAlt = firstImage(content+" "+summary, base)
		}
		f.Items = append(f.Items, item)
	}
	return f
}

// htmlTitle is an Atom title as plain text (type "html" escapes markup).
func htmlTitle(n *node) string {
	if n == nil {
		return ""
	}
	if t := strings.ToLower(n.attr("type")); t == "html" || t == "xhtml" {
		return Excerpt(htmlText(n))
	}
	return n.text()
}

// htmlText is an Atom text construct as HTML: text constructs of type
// "xhtml" keep their markup as child elements, the others as text.
func htmlText(n *node) string {
	if n == nil {
		return ""
	}
	if strings.EqualFold(n.attr("type"), "xhtml") {
		return strings.TrimSpace(n.Inner.String())
	}
	return n.text()
}

func xmlAttrEscape(s string) string {
	return strings.NewReplacer(`&`, "&amp;", `"`, "&quot;", `<`, "&lt;", `>`, "&gt;").Replace(s)
}

func authorOf(it *node) string {
	if c := it.child("creator", nsDC); c != nil && c.text() != "" {
		return cleanText(c.text())
	}
	if a := it.child("author", nsAtom); a != nil {
		if name := a.child("name", nsAtom); name != nil {
			return cleanText(name.text())
		}
		// The RSS author is an email address, often "jo@example.com (Jo Doe)".
		v := cleanText(a.text())
		if i := strings.Index(v, "("); i >= 0 && strings.HasSuffix(v, ")") {
			return strings.TrimSpace(v[i+1 : len(v)-1])
		}
		return v
	}
	return ""
}

// mediaImage reads Media RSS: media:content (an image), media:thumbnail,
// also inside media:group.
func mediaImage(n *node, base *url.URL) (string, string) {
	var walk func(*node) (string, string)
	walk = func(c *node) (string, string) {
		for _, m := range c.all("content", nsMedia) {
			medium, typ := strings.ToLower(m.attr("medium")), strings.ToLower(m.attr("type"))
			if medium == "image" || strings.HasPrefix(typ, "image/") || (medium == "" && typ == "" && looksLikeImage(m.attr("url"))) {
				alt := m.child("description", nsMedia).text()
				if alt == "" {
					alt = m.child("title", nsMedia).text()
				}
				return resolve(base, m.attr("url")), Excerpt(alt)
			}
		}
		if t := c.child("thumbnail", nsMedia); t != nil && t.attr("url") != "" {
			return resolve(base, t.attr("url")), ""
		}
		if g := c.child("group", nsMedia); g != nil {
			return walk(g)
		}
		return "", ""
	}
	return walk(n)
}

func looksLikeImage(u string) bool {
	l := strings.ToLower(u)
	if i := strings.IndexAny(l, "?#"); i >= 0 {
		l = l[:i]
	}
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".webp"} {
		if strings.HasSuffix(l, ext) {
			return true
		}
	}
	return false
}

// isTagTaxonomy reports whether an RSS category domain or an Atom category
// scheme names a tag taxonomy (WordPress "post_tag", ".../tag/").
func isTagTaxonomy(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "post_tag" || v == "tag" || v == "tags" || strings.HasSuffix(strings.TrimRight(v, "/"), "/tag") || strings.Contains(v, "/tag/")
}

func appendUnique(xs []string, v string) []string {
	for _, x := range xs {
		if strings.EqualFold(x, v) {
			return xs
		}
	}
	return append(xs, v)
}

// ---- JSON Feed ----

type jsonAuthor struct {
	Name string `json:"name"`
}

type jsonItem struct {
	ID            json.RawMessage `json:"id"`
	URL           string          `json:"url"`
	ExternalURL   string          `json:"external_url"`
	Title         string          `json:"title"`
	ContentHTML   string          `json:"content_html"`
	ContentText   string          `json:"content_text"`
	Summary       string          `json:"summary"`
	Image         string          `json:"image"`
	BannerImage   string          `json:"banner_image"`
	DatePublished string          `json:"date_published"`
	DateModified  string          `json:"date_modified"`
	Author        *jsonAuthor     `json:"author"`
	Authors       []jsonAuthor    `json:"authors"`
	Tags          []string        `json:"tags"`
}

type jsonFeed struct {
	Version string       `json:"version"`
	Title   string       `json:"title"`
	Home    string       `json:"home_page_url"`
	Items   []jsonItem   `json:"items"`
	Author  *jsonAuthor  `json:"author"`
	Authors []jsonAuthor `json:"authors"`
}

func parseJSON(data []byte, base *url.URL) (*Feed, error) {
	var jf jsonFeed
	if err := json.Unmarshal(data, &jf); err != nil {
		return nil, &NotFeedError{Reason: "the JSON doesn't parse: " + err.Error()}
	}
	if !strings.Contains(jf.Version, "jsonfeed.org/version/") {
		return nil, &NotFeedError{Reason: "a JSON document that isn't a JSON Feed (it has no jsonfeed.org version)"}
	}
	f := &Feed{Kind: KindJSON, Title: jf.Title, Link: resolve(base, jf.Home)}
	feedAuthor := ""
	if len(jf.Authors) > 0 {
		feedAuthor = jf.Authors[0].Name
	} else if jf.Author != nil {
		feedAuthor = jf.Author.Name
	}
	for _, it := range jf.Items {
		item := Item{Title: cleanText(it.Title), URL: resolve(base, firstNonEmpty(it.URL, it.ExternalURL))}
		// The id is a string in 1.1; 1.0 feeds sometimes use numbers.
		var id string
		if json.Unmarshal(it.ID, &id) != nil {
			id = strings.Trim(string(it.ID), `"`)
		}
		item.Key = strings.TrimSpace(id)
		if item.Key == "" || item.Key == "null" {
			item.Key = item.URL
		}
		for _, d := range []string{it.DatePublished, it.DateModified} {
			if t, ok := ParseDate(d); ok {
				item.Published = &t
				break
			}
		}
		switch {
		case len(it.Authors) > 0:
			item.Author = cleanText(it.Authors[0].Name)
		case it.Author != nil:
			item.Author = cleanText(it.Author.Name)
		default:
			item.Author = cleanText(feedAuthor)
		}
		for _, t := range it.Tags {
			if t = cleanText(t); t != "" {
				item.Categories = appendUnique(item.Categories, t)
			}
		}
		switch {
		case it.Summary != "":
			item.Excerpt = Excerpt(it.Summary)
		case it.ContentText != "":
			item.Excerpt = collapse(it.ContentText)
		default:
			item.Excerpt = Excerpt(it.ContentHTML)
		}
		item.Image = resolve(base, firstNonEmpty(it.Image, it.BannerImage))
		if item.Image == "" {
			item.Image, item.ImageAlt = firstImage(it.ContentHTML, base)
		}
		f.Items = append(f.Items, item)
	}
	return f, nil
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// resolve makes a link absolute against the feed's address; anything but
// http(s) is dropped.
func resolve(base *url.URL, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if base != nil {
		u = base.ResolveReference(u)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return u.String()
}

// ParseDate reads the date formats feeds use: RFC 1123 and its variants
// (RSS), RFC 3339 (Atom, JSON Feed, dc:date) and a few common slips.
func ParseDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	// Go reads a zone name it can't place ("EDT") as a zero offset: the
	// North American names RSS allows are written as offsets first.
	if i := strings.LastIndex(s, " "); i > 0 {
		if off, ok := zoneOffsets[strings.ToUpper(s[i+1:])]; ok {
			s = s[:i] + " " + off
		}
	}
	for _, l := range dateLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

var zoneOffsets = map[string]string{"EST": "-0500", "EDT": "-0400", "CST": "-0600", "CDT": "-0500", "MST": "-0700", "MDT": "-0600", "PST": "-0800",
	"PDT": "-0700", "UT": "+0000", "UTC": "+0000", "GMT": "+0000", "Z": "+0000"}

var dateLayouts = []string{
	time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05Z0700", "2006-01-02T15:04:05", "2006-01-02 15:04:05Z07:00", "2006-01-02 15:04:05",
	time.RFC1123Z, "Mon, 2 Jan 2006 15:04:05 -0700", "Mon, 02 Jan 06 15:04:05 -0700", "Mon, 2 Jan 2006 15:04 -0700", "2 Jan 2006 15:04:05 -0700",
	"02 Jan 2006 15:04:05 -0700", "Monday, 02-Jan-06 15:04:05 -0700", time.RFC1123, "Mon, 2 Jan 2006 15:04:05 MST", "Mon Jan _2 15:04:05 2006", "2006-01-02",
}
