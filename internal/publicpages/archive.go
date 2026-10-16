package publicpages

// The public archive (board PublicArchive): every issue of the archived
// lists, searchable and filterable by list, an HTML page per issue, an RSS
// feed and a sitemap. Search engines may index it (the Archive tab can say
// no); search results and later pages ask them not to.

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/store"
)

// archiveLists are the lists the archive publishes.
func (h *Handler) archiveLists(ctx context.Context, s *site) []uuid.UUID {
	lists, err := h.Forms.ArchiveLists(ctx, h.Q, s.ws, s.set)
	if err != nil {
		h.log().WarnContext(ctx, "archive lists", "err", err)
		return []uuid.UUID{}
	}
	return lists
}

type filterView struct {
	Label, Href string
	On          bool
}

type issueView struct {
	List, Date, ISO, Label, Title, Excerpt, Href, Lang string
}

type archiveBody struct {
	Summary, FeedHref, Action, Query, FilterList, SubscribeAction string
	Filters                                                       []filterView
	Issues                                                        []issueView
	Range, NewerHref, OlderHref, Empty, PrefsHref, Lang           string
}

// andNames joins list names: "the Weekly Digest and Product Updates".
func andNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// archiveSummary is the line under the heading: "Every issue of the Weekly
// Digest and Product Updates since 2024. 142 issues."
func archiveSummary(names []string, first time.Time, n int64) string {
	count := formatCount(n) + " issues."
	if n == 1 {
		count = "1 issue."
	}
	if n == 0 {
		return "Issues appear here as they're sent."
	}
	what := "Every issue"
	if len(names) > 0 {
		what += " of the " + andNames(names)
	}
	return fmt.Sprintf("%s since %d. %s", what, first.Year(), count)
}

func archiveDate(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("Monday 2 January 2006")
}

// archiveParams reads ?q, ?list (a list's slug) and ?page.
type archiveParams struct {
	q, list string
	page    int
	listID  *uuid.UUID
}

func archiveParamsOf(v url.Values, lists map[string]uuid.UUID) archiveParams {
	p := archiveParams{q: strings.TrimSpace(v.Get("q")), list: v.Get("list"), page: 1}
	if r := []rune(p.q); len(r) > 100 {
		p.q = string(r[:100])
	}
	if n, err := strconv.Atoi(v.Get("page")); err == nil && n > 1 && n < 100000 {
		p.page = n
	}
	if id, ok := lists[p.list]; ok {
		p.listID = &id
	} else {
		p.list = ""
	}
	return p
}

func (p archiveParams) href(base string, page int, list string, q string) string {
	v := url.Values{}
	if q != "" {
		v.Set("q", q)
	}
	if list != "" {
		v.Set("list", list)
	}
	if page > 1 {
		v.Set("page", strconv.Itoa(page))
	}
	if len(v) == 0 {
		return base
	}
	return base + "?" + v.Encode()
}

func (h *Handler) archive(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if !s.set.Pages.Archive.Enabled {
		h.notFound(w, r, s)
		return
	}
	v, status, err := h.archiveView(r.Context(), s, r.URL.Query(), false)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	h.render(w, r, s, "archive", status, v, csp{}, false)
}

// archiveView builds the archive page; preview renders it for the app.
func (h *Handler) archiveView(ctx context.Context, s *site, query url.Values, preview bool) (*view, int, error) {
	publisher := s.set.Branding.PublisherName
	lists, err := h.Audience.PublicLists(ctx, h.Q, s.ws)
	if err != nil {
		return nil, 0, err
	}
	archived := h.archiveLists(ctx, s)
	bySlug := map[string]uuid.UUID{}
	names := map[uuid.UUID]string{}
	slugs := map[uuid.UUID]string{}
	for _, l := range lists {
		names[l.ID] = l.Name
		sl := forms.Slugify(l.Name, 60)
		if sl == "" {
			sl = l.ID.String()
		}
		bySlug[sl], slugs[l.ID] = l.ID, sl
	}
	p := archiveParamsOf(query, bySlug)
	entries, total, err := h.Forms.Archive(ctx, s.ws, s.set, forms.ArchiveQuery{Search: p.q, List: p.listID, Page: p.page})
	if err != nil {
		return nil, 0, err
	}
	sum, err := h.Q.ArchiveSummary(ctx, store.ArchiveSummaryParams{WorkspaceID: s.ws.ID, Lists: archived})
	if err != nil {
		return nil, 0, err
	}
	counts, err := h.Q.ArchiveListCounts(ctx, store.ArchiveListCountsParams{WorkspaceID: s.ws.ID, Lists: archived})
	if err != nil {
		return nil, 0, err
	}
	withIssues := map[uuid.UUID]bool{}
	for _, c := range counts {
		withIssues[c.ListID] = c.Issues > 0
	}
	var listNames []string
	for _, id := range archived {
		if withIssues[id] {
			listNames = append(listNames, names[id])
		}
	}
	base := s.links.Archive()
	b := archiveBody{Summary: archiveSummary(listNames, sum.FirstAt, sum.Issues), FeedHref: s.links.Feed(), Action: base, Query: p.q,
		FilterList: p.list, PrefsHref: s.links.PreferencesRequest(), Issues: []issueView{}}
	if s.set.Pages.Archive.SubscribeBox {
		b.SubscribeAction = s.links.Subscribe("")
	}
	if len(listNames) > 1 {
		b.Filters = append(b.Filters, filterView{Label: "All", Href: p.href(base, 1, "", p.q), On: p.list == ""})
		for _, id := range archived {
			if withIssues[id] {
				b.Filters = append(b.Filters, filterView{Label: names[id], Href: p.href(base, 1, slugs[id], p.q), On: p.list == slugs[id]})
			}
		}
	}
	loc := location(s.ws)
	for _, e := range entries {
		label := e.Label
		iv := issueView{Date: archiveDate(e.SentAt, loc), ISO: e.SentAt.UTC().Format(time.RFC3339), Label: label, Title: e.Title,
			Excerpt: e.Excerpt, Href: s.links.Issue(e.Slug)}
		if len(listNames) > 1 || e.ListName != "" {
			iv.List = e.ListName
		}
		b.Issues = append(b.Issues, iv)
	}
	size := int64(forms.ArchivePageSize)
	from := int64(p.page-1)*size + 1
	to := min(from+int64(len(entries))-1, total)
	if len(entries) > 0 {
		b.Range = fmt.Sprintf("%s to %s of %s", formatCount(from), formatCount(to), formatCount(total))
	}
	if to < total {
		b.OlderHref = p.href(base, p.page+1, p.list, p.q)
	}
	if p.page > 1 {
		b.NewerHref = p.href(base, p.page-1, p.list, p.q)
	}
	status := http.StatusOK
	switch {
	case len(entries) == 0 && p.q != "":
		b.Empty = "No issues match “" + p.q + "”. Try fewer words, or browse every issue."
	case len(entries) == 0 && p.page > 1:
		b.Empty = "There are no older issues."
		status = http.StatusNotFound
	case len(entries) == 0:
		b.Empty = "No issues yet. The first one appears here as soon as it's sent."
	}
	v := h.view(s, publisher+" newsletter archive", b)
	v.Description = b.Summary
	v.Canonical = p.href(s.canon.Archive(), p.page, p.list, "")
	v.Preview = preview
	if s.set.Pages.Archive.Index && p.q == "" && !preview {
		v.Robots = "index, follow"
	} else {
		v.Robots = "noindex, follow"
	}
	if p.page > 1 {
		v.Title += fmt.Sprintf(" · page %d", p.page)
	}
	v.OG = &og{Type: "website", SiteName: publisher, Title: v.Title, Description: v.Description, URL: v.Canonical}
	v.Feed, v.FeedTitle = s.links.Feed(), publisher+" newsletter archive"
	ld := map[string]any{"@context": "https://schema.org", "@type": "Blog", "name": publisher + " newsletter archive", "url": s.canon.Archive(),
		"description": b.Summary, "publisher": map[string]any{"@type": "Organization", "name": publisher}}
	var posts []map[string]any
	for _, e := range entries {
		posts = append(posts, map[string]any{"@type": "BlogPosting", "headline": e.Title, "datePublished": e.SentAt.UTC().Format(time.RFC3339),
			"url": s.canon.Issue(e.Slug)})
	}
	if len(posts) > 0 {
		ld["blogPost"] = posts
	}
	v.JSONLD = ld
	return v, status, nil
}

func (h *Handler) issue(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if !s.set.Pages.Archive.Enabled {
		h.notFound(w, r, s)
		return
	}
	slug := chi.URLParam(r, "issue")
	if err := h.Forms.SyncArchive(r.Context(), s.ws, s.set); err != nil {
		h.log().WarnContext(r.Context(), "sync archive", "err", err)
	}
	row, err := h.Q.ArchiveIssueBySlug(r.Context(), store.ArchiveIssueBySlugParams{WorkspaceID: s.ws.ID, Slug: slug, Lists: h.archiveLists(r.Context(), s)})
	if errors.Is(err, pgx.ErrNoRows) {
		h.archiveMissing(w, r, s)
		return
	}
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	doc := h.issueDocument(r.Context(), s, row)
	emailHeaders(w, false)
	if !s.set.Pages.Archive.Index {
		w.Header().Set("X-Robots-Tag", "noindex")
	}
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(doc)) //nolint:gosec // the publisher's own email, served under a no-script CSP; inserted values are escaped
	}
}

func (h *Handler) archiveMissing(w http.ResponseWriter, r *http.Request, s *site) {
	h.message(w, r, s, http.StatusNotFound, "Issue not found · "+s.set.Branding.PublisherName, messageBody{
		Heading: "This issue isn't in the archive",
		Text:    "It may have been removed, or the link is cut short. Every published issue is listed in the archive.",
		Actions: []action{{Label: "Newsletter archive", Href: s.links.Archive()}},
	}, false)
}

// winnerContent is the subject and HTML readers of the archive see: the
// winning variant of an A/B test, else the campaign itself.
func winnerContent(subject, body string, ab json.RawMessage) (string, string) {
	var t struct {
		Winner string `json:"winner"`
	}
	if len(ab) == 0 || json.Unmarshal(ab, &t) != nil {
		return subject, body
	}
	return variantContent(subject, body, ab, t.Winner)
}

// variantContent is the subject and HTML of one variant of a campaign
// ("B" of a subject or content test); A and unknown variants are the
// campaign itself.
func variantContent(subject, body string, ab json.RawMessage, variant string) (string, string) {
	if len(ab) == 0 || variant == "" || variant == "A" {
		return subject, body
	}
	var t struct {
		Kind     string `json:"kind"`
		Variants []struct {
			Key     string `json:"key"`
			Subject string `json:"subject"`
			HTML    string `json:"html"`
		} `json:"variants"`
	}
	if json.Unmarshal(ab, &t) != nil {
		return subject, body
	}
	for _, v := range t.Variants {
		if v.Key != variant {
			continue
		}
		if v.Subject != "" {
			subject = v.Subject
		}
		if t.Kind == "content" && v.HTML != "" {
			body = v.HTML
		}
	}
	return subject, body
}

// issueDocument renders an archived issue: the email itself under a bar
// that links back to the archive and to the subscribe page.
func (h *Handler) issueDocument(ctx context.Context, s *site, row store.ArchiveIssueBySlugRow) string {
	publisher := s.set.Branding.PublisherName
	cp, err := h.Q.PublicCampaign(ctx, store.PublicCampaignParams{WorkspaceID: s.ws.ID, ID: row.CampaignID})
	ab := json.RawMessage(nil)
	if err == nil {
		ab = cp.AbTest
	}
	subject, body := winnerContent(row.Subject, row.Html, ab)
	links := map[string]string{
		"unsubscribe_url":     s.links.PreferencesRequest(),
		"preferences_url":     s.links.PreferencesRequest(),
		"view_in_browser_url": s.links.Issue(row.Slug),
	}
	if strings.TrimSpace(body) == "" {
		body = plainDocument(row.PlainText)
	}
	subject, doc := renderEmail(subject, body, "", "", "", nil, links, h.URLs.Base())
	title := strings.TrimSpace(row.Title)
	if title == "" {
		title = subject
	}
	if title == "" {
		title = row.Name
	}
	desc := strings.TrimSpace(row.Excerpt)
	if desc == "" {
		desc = forms.RenderTitle(row.Preheader)
	}
	if desc == "" {
		desc = forms.Excerpt(row.PlainText, 180)
	}
	canon := s.canon.Issue(row.Slug)
	sent := row.SentAt.UTC().Format(time.RFC3339)
	image := firstImage(doc)
	ld := map[string]any{
		"@context": "https://schema.org", "@type": "BlogPosting", "headline": title, "datePublished": sent,
		"dateModified": row.UpdatedAt.UTC().Format(time.RFC3339), "url": canon, "mainEntityOfPage": canon,
		"author":    map[string]any{"@type": "Organization", "name": publisher},
		"publisher": map[string]any{"@type": "Organization", "name": publisher},
		"isPartOf":  map[string]any{"@type": "Blog", "name": publisher + " newsletter archive", "url": s.canon.Archive()},
	}
	if desc != "" {
		ld["description"] = desc
	}
	if image != "" {
		ld["image"] = image
	}
	robots := "index, follow"
	if !s.set.Pages.Archive.Index {
		robots = "noindex, follow"
	}
	head := emailHead{Title: title + " · " + publisher + " newsletter", Description: desc, Robots: robots, Canonical: canon,
		OG:   &og{Type: "article", SiteName: publisher, Title: title, Description: desc, URL: canon, Image: image, Published: sent},
		Feed: s.links.Feed(), FeedTitle: publisher + " newsletter archive", JSONLD: ld}
	if s.set.Branding.Logo != nil {
		head.Icon = s.links.Logo(s.set.Branding.Logo.Hash)
	}
	doc = injectHead(doc, head)
	meta := []string{}
	if row.ListName != "" {
		meta = append(meta, row.ListName)
	}
	meta = append(meta, archiveDate(row.SentAt, location(s.ws)))
	if row.Label != "" {
		meta = append(meta, row.Label)
	}
	return injectTop(withStylesheet(doc, s.links.Static("issue.css?v="+h.version)), h.issueBar(s, strings.Join(meta, " · ")))
}

// withStylesheet links the issue bar's stylesheet in the document's head.
func withStylesheet(doc, href string) string {
	link := `<link rel="stylesheet" href="` + html.EscapeString(href) + `">` + "\n"
	if loc := headOpen.FindStringIndex(doc); loc != nil {
		return doc[:loc[1]] + "\n" + link + doc[loc[1]:]
	}
	return link + doc
}

// issueBar is the strip above an archived issue.
func (h *Handler) issueBar(s *site, meta string) string {
	esc := html.EscapeString
	b := s.set.Branding
	mark := `<span class="rcia-mark" aria-hidden="true">` + esc(b.Initial()) + `</span>`
	if b.Logo != nil {
		mark = `<span class="rcia-mark rcia-logo"><img src="` + esc(s.links.Logo(b.Logo.Hash)) + `" alt="" width="28" height="28"></span>`
	}
	p := b.Palette
	style := fmt.Sprintf(`<style>.rcia{--rcia-accent:%s;--rcia-hover:%s;--rcia-on:%s;--rcia-ground:%s}</style>`, p.Accent, p.Hover, p.OnAccent, p.Background)
	return "\n" + style + `<header class="rcia"><div class="rcia-in"><a class="rcia-home" href="` + esc(s.links.Archive()) + `">` + mark +
		`<span>` + esc(b.PublisherName) + ` · Newsletter archive</span></a><a class="rcia-sub" href="` + esc(s.links.Subscribe("")) + `">Subscribe</a></div>` +
		`<div class="rcia-meta">` + esc(meta) + `</div></header>` + "\n"
}

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Atom    string     `xml:"xmlns:atom,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	// The atom self link comes first: readers that ignore namespaces then
	// take the channel's own <link>.
	Self        rssAtom   `xml:"atom:link"`
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Language    string    `xml:"language"`
	Updated     string    `xml:"lastBuildDate,omitempty"`
	Items       []rssItem `xml:"item"`
}

type rssAtom struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type rssItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	GUID        rssGUID `xml:"guid"`
	PubDate     string  `xml:"pubDate"`
	Category    string  `xml:"category,omitempty"`
	Description string  `xml:"description,omitempty"`
}

type rssGUID struct {
	Value     string `xml:",chardata"`
	Permalink string `xml:"isPermaLink,attr"`
}

func (h *Handler) feed(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	if !s.set.Pages.Archive.Enabled {
		h.notFound(w, r, s)
		return
	}
	entries, _, err := h.Forms.Archive(r.Context(), s.ws, s.set, forms.ArchiveQuery{Page: 1, PageSize: 50})
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	publisher := s.set.Branding.PublisherName
	ch := rssChannel{Title: publisher + " newsletter", Link: s.canon.Archive(), Description: "Every issue of the " + publisher + " newsletter.",
		Language: "en", Self: rssAtom{Href: s.canon.Feed(), Rel: "self", Type: "application/rss+xml"}, Items: []rssItem{}}
	for i, e := range entries {
		if i == 0 {
			ch.Updated = e.SentAt.UTC().Format(time.RFC1123Z)
		}
		link := s.canon.Issue(e.Slug)
		ch.Items = append(ch.Items, rssItem{Title: e.Title, Link: link, GUID: rssGUID{Value: link, Permalink: "true"},
			PubDate: e.SentAt.UTC().Format(time.RFC1123Z), Category: e.ListName, Description: e.Excerpt})
	}
	out, err := xml.MarshalIndent(rssFeed{Version: "2.0", Atom: "http://www.w3.org/2005/Atom", Channel: ch}, "", "  ")
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

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	NS      string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

func (h *Handler) sitemap(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	set := sitemapURLSet{NS: "http://www.sitemaps.org/schemas/sitemap/0.9", URLs: []sitemapURL{}}
	if s.set.Pages.Subscribe.Index {
		set.URLs = append(set.URLs, sitemapURL{Loc: s.canon.Subscribe(""), LastMod: s.set.UpdatedAt().UTC().Format("2006-01-02")})
	}
	if s.set.Pages.Archive.Enabled && s.set.Pages.Archive.Index {
		var newest time.Time
		var issues []sitemapURL
		for page := 1; page <= 100; page++ {
			entries, total, err := h.Forms.Archive(r.Context(), s.ws, s.set, forms.ArchiveQuery{Page: page, PageSize: 500})
			if err != nil {
				h.serverError(w, r, s, err)
				return
			}
			for _, e := range entries {
				if e.UpdatedAt.After(newest) {
					newest = e.UpdatedAt
				}
				issues = append(issues, sitemapURL{Loc: s.canon.Issue(e.Slug), LastMod: e.UpdatedAt.UTC().Format("2006-01-02")})
			}
			if int64(page*500) >= total {
				break
			}
		}
		arch := sitemapURL{Loc: s.canon.Archive()}
		if !newest.IsZero() {
			arch.LastMod = newest.UTC().Format("2006-01-02")
		}
		set.URLs = append(set.URLs, arch)
		set.URLs = append(set.URLs, issues...)
	}
	out, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	hd := w.Header()
	hd.Set("Content-Type", "application/xml; charset=utf-8")
	hd.Set("Cache-Control", "public, max-age=3600")
	hd.Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(out)
}

// robots is /robots.txt on a custom domain: the subscribe page and the
// archive may be crawled, the signed-link pages may not.
func (h *Handler) robots(w http.ResponseWriter, r *http.Request) {
	s, ok := h.siteOr404(w, r)
	if !ok {
		return
	}
	var b strings.Builder
	b.WriteString("User-agent: *\n")
	for _, p := range []string{"/confirm/", "/preferences", "/unsubscribe/", "/view/", "/subscribe/challenge"} {
		b.WriteString("Disallow: " + p + "\n")
	}
	if !s.set.Pages.Archive.Index {
		b.WriteString("Disallow: /archive\n")
	}
	b.WriteString("\nSitemap: " + s.links.Sitemap() + "\n")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write([]byte(b.String()))
}
