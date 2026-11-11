package importer

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/url"
	"regexp"
	"strings"
)

// WordPress's export (Tools › Export) is WXR, WordPress eXtended RSS: one
// <item> a post, page, attachment or menu item, with namespaced elements —
// content:encoded (the body), excerpt:encoded, dc:creator (the author's
// login, named by a channel's wp:author), wp:post_id, wp:post_date_gmt,
// wp:post_name (the slug), wp:status (publish, draft, pending, future,
// private, trash, inherit), wp:post_type, wp:post_password, wp:postmeta
// (_thumbnail_id: the attachment that is the post's cover), and
// <category domain="category|post_tag">. The namespaces' scheme and
// version vary (http://wordpress.org/export/1.2/, https:// on
// WordPress.com), so elements are told apart by their local names.

// wxrElem is one element of an item, its text and children.
type wxrElem struct {
	XMLName xml.Name
	Attr    []xml.Attr `xml:",any,attr"`
	Text    string     `xml:",chardata"`
	Kids    []wxrElem  `xml:",any"`
}

func (e *wxrElem) attr(name string) string {
	for _, a := range e.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// child is the first child named local in a namespace containing ns ("" any).
func (e *wxrElem) child(ns, local string) *wxrElem {
	for i := range e.Kids {
		k := &e.Kids[i]
		if k.XMLName.Local == local && strings.Contains(k.XMLName.Space, ns) {
			return k
		}
	}
	return nil
}

func (e *wxrElem) text(ns, local string) string {
	if c := e.child(ns, local); c != nil {
		return strings.TrimSpace(c.Text)
	}
	return ""
}

const (
	nsWP      = "wordpress.org/export/"
	nsContent = "purl.org/rss/1.0/modules/content"
	nsExcerpt = "/excerpt/"
	nsDC      = "purl.org/dc/elements"
)

// item is a WXR item as the parser keeps it.
type wxrItem struct {
	id, kind, status, password, title, link, slug, date, creator, content, excerpt string
	thumbnail                                                                      string
	categories                                                                     []string
	attachmentURL, alt                                                             string
}

// ParseWXR reads a WordPress export. Published posts, drafts and posts
// waiting (pending, scheduled) come in; private, password-protected and
// trashed ones stay out, as do pages, attachments and menus.
func ParseWXR(data []byte) ([]Post, error) {
	body := trimBOM(data)
	if !looksLikeWXR(body) {
		return nil, ErrNotExport
	}
	dec := xml.NewDecoder(bytes.NewReader(body))
	dec.Strict = false
	dec.CharsetReader = func(_ string, r io.Reader) (io.Reader, error) { return r, nil }
	authors := map[string]string{}
	attachments := map[string]wxrItem{}
	var items []wxrItem
	base := ""
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, ErrNotExport
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch {
		case start.Name.Local == "item":
			var e wxrElem
			if err := dec.DecodeElement(&e, &start); err != nil {
				return nil, ErrNotExport
			}
			it := readItem(&e)
			switch it.kind {
			case "attachment":
				attachments[it.id] = it
			case "post":
				if len(items) == MaxPosts {
					return nil, ErrTooManyPosts
				}
				items = append(items, it)
			}
		case start.Name.Local == "author" && strings.Contains(start.Name.Space, nsWP):
			var e wxrElem
			if err := dec.DecodeElement(&e, &start); err != nil {
				return nil, ErrNotExport
			}
			if login := e.text(nsWP, "author_login"); login != "" {
				authors[login] = firstOf(e.text(nsWP, "author_display_name"), login)
			}
		case start.Name.Local == "base_blog_url" && strings.Contains(start.Name.Space, nsWP):
			var s string
			if err := dec.DecodeElement(&s, &start); err == nil {
				base = strings.TrimSpace(s)
			}
		}
	}
	out := []Post{}
	for _, it := range items {
		switch it.status {
		case "publish", "draft", "pending", "future":
		default:
			// private, trash, auto-draft, inherit.
			continue
		}
		if it.password != "" {
			continue
		}
		p := Post{SourceID: it.id, Title: unescape(it.title), Draft: it.status != "publish", PublishedAt: parseTime(it.date),
			Visibility: Public, TierRefs: []string{}, Tags: it.categories, Excerpt: textOf(it.excerpt), SourceURL: it.link,
			HTML: wpContent(it.content)}
		if slug, err := url.PathUnescape(it.slug); err == nil && slug != "" {
			p.Slug = strings.ToLower(slug)
		}
		if p.Draft && p.PublishedAt != nil && it.status != "future" {
			// A draft's date is when it was started, not when it went out.
			p.PublishedAt = nil
		}
		p.Author = it.creator
		if name, ok := authors[it.creator]; ok {
			p.Author = name
		}
		if a, ok := attachments[it.thumbnail]; ok && a.attachmentURL != "" {
			p.CoverURL, p.CoverAlt = a.attachmentURL, a.alt
		}
		if p.Title == "" {
			p.Title = p.Slug
		}
		if u, err := url.Parse(it.link); err == nil && u.Host != "" {
			if old, err := url.PathUnescape(u.Path); err == nil {
				if k := LegacyKey(old, ""); k != "" {
					p.LegacyPaths = append(p.LegacyPaths, k)
				}
			}
		}
		if it.id != "" {
			p.LegacyPaths = append(p.LegacyPaths, "?p="+it.id)
		}
		if p.SourceURL == "" && base != "" && it.id != "" {
			p.SourceURL = strings.TrimRight(base, "/") + "/?p=" + it.id
		}
		out = append(out, p)
	}
	return out, nil
}

func readItem(e *wxrElem) wxrItem {
	it := wxrItem{id: e.text(nsWP, "post_id"), kind: e.text(nsWP, "post_type"), status: e.text(nsWP, "status"), password: e.text(nsWP, "post_password"),
		title: e.text("", "title"), link: e.text("", "link"), slug: e.text(nsWP, "post_name"), date: e.text(nsWP, "post_date_gmt"),
		creator: e.text(nsDC, "creator"), attachmentURL: e.text(nsWP, "attachment_url")}
	if c := e.child(nsContent, "encoded"); c != nil {
		it.content = c.Text
	}
	if c := e.child(nsExcerpt, "encoded"); c != nil {
		it.excerpt = c.Text
	}
	seen := map[string]bool{}
	for i := range e.Kids {
		k := &e.Kids[i]
		switch {
		case k.XMLName.Local == "category" && k.XMLName.Space == "":
			d, name := k.attr("domain"), strings.TrimSpace(unescape(k.Text))
			if (d != "category" && d != "post_tag") || name == "" || k.attr("nicename") == "uncategorized" || seen[strings.ToLower(name)] {
				continue
			}
			seen[strings.ToLower(name)] = true
			it.categories = append(it.categories, name)
		case k.XMLName.Local == "postmeta" && strings.Contains(k.XMLName.Space, nsWP):
			switch k.text(nsWP, "meta_key") {
			case "_thumbnail_id":
				it.thumbnail = k.text(nsWP, "meta_value")
			case "_wp_attachment_image_alt":
				it.alt = k.text(nsWP, "meta_value")
			}
		}
	}
	if it.categories == nil {
		it.categories = []string{}
	}
	return it
}

var (
	captionRe = regexp.MustCompile(`(?s)\[(?:caption|wp_caption)\b[^\]]*\](.*?)\[/(?:caption|wp_caption)\]`)
	embedRe   = regexp.MustCompile(`(?s)\[embed\b[^\]]*\](.*?)\[/embed\]`)
	// mediaRe is [audio …] and [video …] with their file, and [gallery …]
	// and [playlist …], which need the site's own media library.
	mediaRe = regexp.MustCompile(`\[(audio|video|gallery|playlist)\b([^\]]*)\](?:\s*\[/(?:audio|video)\])?`)
	srcRe   = regexp.MustCompile(`\b(?:src|mp3|m4a|ogg|wav|mp4|m4v|webm|ogv)="([^"]+)"`)
	// A URL alone on its line: WordPress shows it as an embed.
	bareURLRe   = regexp.MustCompile(`(?m)^[ \t]*(https?://[^\s<>"']+)[ \t]*$`)
	paragraphRe = regexp.MustCompile(`\n[ \t]*\n`)
	blockOpenRe = regexp.MustCompile(`^\s*<(?i:p|div|h[1-6]|ul|ol|li|blockquote|pre|table|figure|hr|img|iframe|section|dl|form|address|style|script|!--|video|audio|object|embed)\b`)
)

// wpContent is a post's body as WordPress shows it: a classic post's
// paragraphs (blank lines) and line breaks become HTML (WordPress's
// wpautop), [caption] a figure, [embed], [audio], [video] and a known
// platform's URL alone on its line an embed. Block editor posts are HTML
// already. [gallery] and [playlist] leave a note the converter reports.
func wpContent(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = captionRe.ReplaceAllStringFunc(s, func(m string) string {
		inner := strings.TrimSpace(captionRe.FindStringSubmatch(m)[1])
		caption := ""
		if i := strings.LastIndex(inner, ">"); i >= 0 {
			inner, caption = inner[:i+1], strings.TrimSpace(inner[i+1:])
		}
		return "<figure>" + inner + "<figcaption>" + caption + "</figcaption></figure>"
	})
	s = embedRe.ReplaceAllStringFunc(s, func(m string) string { return embedFigure(strings.TrimSpace(embedRe.FindStringSubmatch(m)[1])) })
	s = mediaRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := mediaRe.FindStringSubmatch(m)
		src := ""
		if a := srcRe.FindStringSubmatch(sub[2]); a != nil {
			src = a[1]
		}
		switch {
		case sub[1] == "audio" && src != "":
			return `<audio src="` + src + `"></audio>`
		case sub[1] == "video" && src != "":
			return `<video src="` + src + `"></video>`
		}
		return "<!--rc:shortcode " + sub[1] + "-->"
	})
	if strings.Contains(s, "<!-- wp:") {
		return s
	}
	s = bareURLRe.ReplaceAllStringFunc(s, func(m string) string {
		u := strings.TrimSpace(m)
		if _, ok := platformOf(u); !ok {
			return m
		}
		return embedFigure(u)
	})
	var b strings.Builder
	for _, chunk := range paragraphRe.Split(s, -1) {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		if blockOpenRe.MatchString(chunk) {
			b.WriteString(chunk + "\n")
			continue
		}
		b.WriteString("<p>" + strings.ReplaceAll(chunk, "\n", "<br>\n") + "</p>\n")
	}
	return b.String()
}

// embedFigure is an embed as WordPress's block editor writes it.
func embedFigure(u string) string {
	return `<figure class="wp-block-embed"><div class="wp-block-embed__wrapper">` + u + `</div></figure>`
}
