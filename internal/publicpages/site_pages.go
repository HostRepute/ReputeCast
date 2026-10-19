package publicpages

import (
	"context"
	"encoding/xml"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/website"
)

// The website's lists and unavailable pages (PublicSiteStates): a tag's
// page and feed, the site's feed, the archive in the site's style with
// search, 410 for unpublished posts, the site's own 404 and error page.

const sitePageSize = 10

// listBody is a list of posts: a tag's page, the archive, a gone post's
// latest posts.
type listBody struct {
	Kind string // tag, archive, gone, missing
	// The tag's head.
	Over, Heading, Text string
	Count               string
	FeedHref, FeedText  string
	// The archive's search and tag filter (Tag: the tag picked).
	SearchAction, SearchLabel, Query, Tag string
	Chips                                 []linkAction
	Result                                string
	Cards                                 []postCardView
	Empty                                 string
	// Paging.
	PageText             string
	NewerHref, OlderHref string
	NewerText, OlderText string
	ClearHref, ClearText string
	PagesLabel           string
	Actions              []linkAction
}

// siteFail is the site's error page: "This page didn't load". Nothing
// about the reader or the post shows.
func (h *Handler) siteFail(w http.ResponseWriter, r *http.Request, s *site, err error) {
	h.log().ErrorContext(r.Context(), "website page", "path", r.URL.Path, "err", err)
	if h.Website == nil {
		h.serverError(w, r, s, err)
		return
	}
	v, verr := h.siteViewIn(r.Context(), s, h.visitorLang(w, r, s, nil))
	if verr != nil {
		h.serverError(w, r, s, err)
		return
	}
	v.Title = v.T("site.title", "page", v.T("site.failTitle"), "site", v.Frame.Title)
	v.Robots = "noindex, nofollow"
	v.private = true
	v.Body = listBody{Kind: "fail", Heading: v.T("site.failTitle"), Text: v.T("site.failText"),
		Actions: []linkAction{{Label: v.T("site.tryAgain"), Href: r.URL.RequestURI(), Primary: true}, {Label: v.T("signin.home"), Href: s.path()}}}
	h.renderSite(w, r, s, "fail", http.StatusInternalServerError, v)
}

// siteNotFound is a missing page of the site, in its frame while it is on.
func (h *Handler) siteNotFound(w http.ResponseWriter, r *http.Request, s *site) {
	if h.Website == nil || !h.siteOn(r.Context(), s) {
		h.notFound(w, r, s)
		return
	}
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	v.Title = v.T("site.title", "page", v.T("site.missingTitle"), "site", v.Frame.Title)
	v.Robots = "noindex, follow"
	b, err := h.latestBody(r.Context(), s, v, "missing", v.T("site.missingTitle"), v.T("site.missingText"))
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	v.Body = b
	h.renderSite(w, r, s, "gone", http.StatusNotFound, v)
}

// siteGone is an unpublished or deleted post: 410, and the latest posts.
func (h *Handler) siteGone(w http.ResponseWriter, r *http.Request, s *site) {
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	v.Title = v.T("site.title", "page", v.T("site.goneTitle"), "site", v.Frame.Title)
	v.Robots = "noindex, follow"
	b, err := h.latestBody(r.Context(), s, v, "gone", v.T("site.goneTitle"), v.T("site.goneText"))
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	v.Body = b
	h.renderSite(w, r, s, "gone", http.StatusGone, v)
}

// latestBody is a heading, a line and the three latest posts.
func (h *Handler) latestBody(ctx context.Context, s *site, v *siteView, kind, heading, text string) (listBody, error) {
	b := listBody{Kind: kind, Heading: heading, Text: text,
		Actions: []linkAction{{Label: v.T("site.browseArchive"), Href: s.links.Archive(), Primary: true}, {Label: v.T("signin.home"), Href: s.path()}}}
	posts, _, err := h.Website.PublicPosts(ctx, s.ws, website.PublicQuery{Limit: 3}, v.labels.Lang())
	if err != nil {
		return b, err
	}
	for _, p := range posts {
		b.Cards = append(b.Cards, h.card(s, v, p, nil, "row"))
	}
	return b, nil
}

// pageOf reads ?page= (1 and up).
func pageOf(r *http.Request) int {
	n, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || n < 1 || n > 10000 {
		return 1
	}
	return n
}

func withPage(base string, q url.Values, page int) string {
	v := url.Values{}
	for k, vs := range q {
		if k != "page" {
			v[k] = vs
		}
	}
	if page > 1 {
		v.Set("page", strconv.Itoa(page))
	}
	if len(v) == 0 {
		return base
	}
	return base + "?" + v.Encode()
}

// paging fills a list's pager.
func paging(v *siteView, b *listBody, base string, q url.Values, page int, total int64) {
	pages := int((total + sitePageSize - 1) / sitePageSize)
	if pages <= 1 {
		return
	}
	b.PageText, b.PagesLabel = v.T("site.pageOf", "page", page, "pages", pages), v.T("site.pages")
	if page > 1 {
		b.NewerHref, b.NewerText = withPage(base, q, page-1), v.T("site.newer")
	}
	if page < pages {
		b.OlderHref, b.OlderText = withPage(base, q, page+1), v.T("site.older")
	}
}

// siteTag is a tag's page: its name, description and feed, then its posts.
func (h *Handler) siteTag(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Website == nil {
		h.notFound(w, r, s)
		return
	}
	ctx := r.Context()
	tag, err := h.Website.TagBySlug(ctx, s.ws, chi.URLParam(r, "slug"))
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	if tag == nil {
		h.siteNotFound(w, r, s)
		return
	}
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	page := pageOf(r)
	posts, total, err := h.Website.PublicPosts(ctx, s.ws, website.PublicQuery{Tag: &tag.ID, Limit: sitePageSize, Offset: (page - 1) * sitePageSize}, v.labels.Lang())
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	base := s.path("tag", tag.Slug)
	b := listBody{Kind: "tag", Over: v.T("site.tagOver"), Heading: tag.Name, Text: tag.Description, Count: v.T("site.tagPosts", "count", total),
		FeedHref: s.path("tag", tag.Slug, "feed"), FeedText: v.T("site.feedLink")}
	for _, p := range posts {
		b.Cards = append(b.Cards, h.card(s, v, p, nil, "row"))
	}
	if len(posts) == 0 {
		b.Empty = v.T("site.noPosts")
	}
	paging(v, &b, base, r.URL.Query(), page, total)
	v.Body = b
	title := tag.Name
	if page > 1 {
		title += " · " + v.T("site.page", "page", page)
		v.Robots = "noindex, follow"
	}
	v.Title = v.T("site.title", "page", title, "site", v.Frame.Title)
	v.Description = tag.Description
	v.Canonical = s.canonPath("tag", tag.Slug)
	v.Feeds = []siteFeed{{Title: v.T("site.feedOf", "tag", tag.Name, "site", v.Frame.Title), Href: s.canonPath("tag", tag.Slug, "feed")}}
	v.mark("tag:"+tag.Slug, v.Frame)
	h.Website.CountView(ctx, s.ws, r, website.PageTag, tag.ID)
	h.renderSite(w, r, s, "tag", http.StatusOK, v)
}

// siteArchive is the archive while the website is on: every post, a
// search over what visitors may read, and the tags.
func (h *Handler) siteArchive(w http.ResponseWriter, r *http.Request, s *site) {
	ctx := r.Context()
	v, ok := h.siteViewFor(w, r, s)
	if !ok {
		return
	}
	q := r.URL.Query()
	query := strings.TrimSpace(q.Get("q"))
	if len([]rune(query)) > 100 {
		query = string([]rune(query)[:100])
	}
	page := pageOf(r)
	f := website.PublicQuery{Search: query, Limit: sitePageSize, Offset: (page - 1) * sitePageSize}
	tags, err := h.Website.SiteTags(ctx, s.ws)
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	current := q.Get("tag")
	var currentName string
	for _, t := range tags {
		if t.Slug == current {
			id := t.ID
			f.Tag, currentName = &id, t.Name
		}
	}
	posts, total, err := h.Website.PublicPosts(ctx, s.ws, f, v.labels.Lang())
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	_, all, err := h.Website.PublicPosts(ctx, s.ws, website.PublicQuery{Limit: 1}, v.labels.Lang())
	if err != nil {
		h.siteFail(w, r, s, err)
		return
	}
	base := s.links.Archive()
	host := strings.TrimPrefix(strings.TrimPrefix(strings.TrimSuffix(s.canonPath(), "/"), "https://"), "http://")
	b := listBody{Kind: "archive", Heading: v.T("site.archive"), Text: v.T("site.archiveText", "host", host, "count", all), SearchAction: base,
		SearchLabel: v.T("site.searchPosts"), Query: query}
	chip := func(label, slug string) linkAction {
		qq := url.Values{}
		if query != "" {
			qq.Set("q", query)
		}
		if slug != "" {
			qq.Set("tag", slug)
		}
		href := base
		if len(qq) > 0 {
			href += "?" + qq.Encode()
		}
		return linkAction{Label: label, Href: href, Primary: slug == current || (slug == "" && f.Tag == nil)}
	}
	b.Chips = append(b.Chips, chip(v.T("site.all"), ""))
	for i, t := range tags {
		if i >= 8 {
			break
		}
		b.Chips = append(b.Chips, chip(t.Name, t.Slug))
	}
	var hl *regexp.Regexp
	if query != "" {
		hl, _ = regexp.Compile("(?i)" + regexp.QuoteMeta(query))
	}
	for _, p := range posts {
		c := h.card(s, v, p, nil, "row")
		if hl != nil {
			c.TitleHTML, c.ExcerptHTML = highlight(hl, c.Title), highlight(hl, c.Excerpt)
		}
		b.Cards = append(b.Cards, c)
	}
	switch {
	case query != "" && currentName != "":
		b.Result = v.T("site.matchesIn", "count", total, "query", query, "tag", currentName)
	case query != "":
		b.Result = v.T("site.matches", "count", total, "query", query)
	}
	if query != "" || f.Tag != nil {
		b.ClearHref, b.ClearText = base, v.T("site.clearSearch")
	}
	if len(posts) == 0 {
		b.Empty = v.T("site.noMatches")
	}
	paging(v, &b, base, q, page, total)
	if b.PageText == "" && query != "" && total > 0 {
		b.PageText = v.T("site.endOfResults", "count", total)
	}
	if f.Tag != nil {
		b.Tag = current
	}
	b.PagesLabel = v.T("site.pages")
	v.Body = b
	v.Title = v.T("site.title", "page", v.T("site.archive"), "site", v.Frame.Title)
	v.Canonical = s.canon.Archive()
	if query != "" || page > 1 || f.Tag != nil {
		v.Robots = "noindex, follow"
	}
	v.Feeds = []siteFeed{{Title: v.Frame.Title, Href: s.canonPath("feed")}}
	v.mark("archive", v.Frame)
	h.Website.CountView(ctx, s.ws, r, website.PageArchive, website.HomeID)
	h.renderSite(w, r, s, "archive", http.StatusOK, v)
}

// highlight marks a search's matches in a text.
func highlight(re *regexp.Regexp, text string) template.HTML {
	var b strings.Builder
	last := 0
	for _, m := range re.FindAllStringIndex(text, -1) {
		b.WriteString(template.HTMLEscapeString(text[last:m[0]]))
		b.WriteString(`<mark class="st-hl">` + template.HTMLEscapeString(text[m[0]:m[1]]) + `</mark>`)
		last = m[1]
	}
	b.WriteString(template.HTMLEscapeString(text[last:]))
	return template.HTML(b.String()) //nolint:gosec // every piece is escaped above
}

// siteFeedHandler is /feed: the site's posts, public ones whole, gated
// ones only as far as visitors may read them.
func (h *Handler) siteFeed(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Website == nil || !h.siteOn(r.Context(), s) {
		h.notFound(w, r, s)
		return
	}
	h.writeFeed(w, r, s, nil)
}

// siteTagFeed is a tag's feed.
func (h *Handler) siteTagFeed(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if h.Website == nil {
		h.notFound(w, r, s)
		return
	}
	tag, err := h.Website.TagBySlug(r.Context(), s.ws, chi.URLParam(r, "slug"))
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	if tag == nil {
		h.notFound(w, r, s)
		return
	}
	h.writeFeed(w, r, s, tag)
}

// rssContent is an item's whole text (content:encoded).
type rssContent struct {
	Value string `xml:",cdata"`
}

type siteRSSItem struct {
	Title       string      `xml:"title"`
	Link        string      `xml:"link"`
	GUID        rssGUID     `xml:"guid"`
	PubDate     string      `xml:"pubDate"`
	Categories  []string    `xml:"category"`
	Description string      `xml:"description"`
	Content     *rssContent `xml:"content:encoded,omitempty"`
}

type siteRSSChannel struct {
	Title       string        `xml:"title"`
	Link        string        `xml:"link"`
	Description string        `xml:"description"`
	Language    string        `xml:"language,omitempty"`
	Updated     string        `xml:"lastBuildDate,omitempty"`
	Self        rssAtom       `xml:"atom:link"`
	Items       []siteRSSItem `xml:"item"`
}

type siteRSS struct {
	XMLName xml.Name       `xml:"rss"`
	Version string         `xml:"version,attr"`
	Atom    string         `xml:"xmlns:atom,attr"`
	Content string         `xml:"xmlns:content,attr"`
	Channel siteRSSChannel `xml:"channel"`
}

func (h *Handler) writeFeed(w http.ResponseWriter, r *http.Request, s *site, tag *website.PublicTag) {
	ctx := r.Context()
	var tagID *uuid.UUID
	if tag != nil {
		tagID = &tag.ID
	}
	items, err := h.Website.Feed(ctx, s.ws, tagID, 30)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	frame, err := h.Website.Frame(ctx, s.ws, s.set)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	title, link, self := frame.Title, s.canonPath(), s.canonPath("feed")
	desc := frame.Description
	if desc == "" {
		desc = frame.Tagline
	}
	if tag != nil {
		title, link, self = tag.Name+" · "+frame.Title, s.canonPath("tag", tag.Slug), s.canonPath("tag", tag.Slug, "feed")
		if tag.Description != "" {
			desc = tag.Description
		}
	}
	ch := siteRSSChannel{Title: title, Link: link, Description: desc, Language: h.defaultLang(ctx, s), Self: rssAtom{Href: self, Rel: "self", Type: "application/rss+xml"}}
	for i, it := range items {
		if i == 0 {
			ch.Updated = it.Post.PublishedAt.UTC().Format(time.RFC1123Z)
		}
		u := s.canonPath("posts", it.Post.Slug)
		item := siteRSSItem{Title: it.Post.Title, Link: u, GUID: rssGUID{Value: u, Permalink: "true"}, PubDate: it.Post.PublishedAt.UTC().Format(time.RFC1123Z),
			Description: it.Post.Excerpt}
		for _, t := range it.Post.Tags {
			item.Categories = append(item.Categories, t.Name)
		}
		if strings.TrimSpace(it.HTML) != "" {
			item.Content = &rssContent{Value: it.HTML}
		}
		ch.Items = append(ch.Items, item)
	}
	out, err := xml.MarshalIndent(siteRSS{Version: "2.0", Atom: "http://www.w3.org/2005/Atom", Content: "http://purl.org/rss/1.0/modules/content/", Channel: ch}, "", "  ")
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	hd := w.Header()
	hd.Set("Content-Type", "application/rss+xml; charset=utf-8")
	hd.Set("Cache-Control", "public, max-age=600")
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(out)
}

// siteSitemap adds the website's addresses to the sitemap: the home page,
// the archive, every post that isn't hidden from search and the tags.
func (h *Handler) siteSitemap(ctx context.Context, s *site, set *sitemapURLSet) error {
	if h.Website == nil || !h.siteOn(ctx, s) {
		return nil
	}
	frame, err := h.Website.Frame(ctx, s.ws, s.set)
	if err != nil || !frame.Index {
		return err
	}
	entries, err := h.Website.Sitemap(ctx, s.ws)
	if err != nil {
		return err
	}
	set.URLs = append(set.URLs, sitemapURL{Loc: s.canonPath()})
	for _, e := range entries {
		u := sitemapURL{Loc: s.canonPath(e.Path...)}
		if !e.LastMod.IsZero() {
			u.LastMod = e.LastMod.UTC().Format("2006-01-02")
		}
		set.URLs = append(set.URLs, u)
	}
	return nil
}
