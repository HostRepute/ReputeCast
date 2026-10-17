package publicpages

// The public archive (board PublicArchive): every issue of the archived
// lists, searchable and filterable by list, an HTML page per issue, an RSS
// feed and a sitemap. Search engines may index it (the Archive tab can say
// no); search results and later pages ask them not to.
//
// An issue with language versions (design expansion M17) shows a visitor
// the version their email would have been: in their language when it has
// one, else the fallback's, else the language it's written in. Each version
// has its own address (/archive/<issue>/<language>), linked from "Also in"
// and from hreflang alternates; the archive lists each issue in the version
// a reader of the visitor's language gets.

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

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/pagelabels"
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

// issueView is an issue in the archive's list. Lang is the language of its
// version when it isn't the page's.
type issueView struct {
	List, Date, ISO, Label, Title, Excerpt, Href, Lang string
}

type archiveBody struct {
	Summary, FeedHref, Action, Query, FilterList, SubscribeAction string
	Filters                                                       []filterView
	Issues                                                        []issueView
	Range, NewerHref, OlderHref, Empty, PrefsHref                 string
	// Lang keeps a language picked in the menu across a search.
	Lang string
}

// archiveSummary is the line under the heading: "Every issue of the Weekly
// Digest and Product Updates since 2024. 142 issues."
func archiveSummary(l pagelabels.Set, names []string, first time.Time, n int64) string {
	year := strconv.Itoa(first.Year())
	switch {
	case n == 0:
		return l.Text("archive.summaryEmpty", nil)
	case len(names) == 0:
		return l.Text("archive.summaryAll", i18n.Args{"year": year, "count": n})
	}
	return l.Text("archive.summary", i18n.Args{"lists": l.Join(names), "year": year, "count": n})
}

// archiveParams reads ?q, ?list (a list's slug) and ?page. lang is a
// language picked in the menu, kept in the page's links.
type archiveParams struct {
	q, list, lang string
	page          int
	listID        *uuid.UUID
	// digestID is set when ?list names a recurring digest's filter.
	digestID *uuid.UUID
}

// withDigest reads ?list as a recurring digest's slug when it isn't a list's.
func (p archiveParams) withDigest(v url.Values, digests map[string]uuid.UUID) archiveParams {
	if p.list != "" {
		return p
	}
	if id, ok := digests[v.Get("list")]; ok {
		p.list, p.digestID = v.Get("list"), &id
	}
	return p
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
	if p.lang != "" {
		v.Set(langParam, p.lang)
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
	pl := h.visitorLang(w, r, s, nil)
	v, status, err := h.archiveView(r.Context(), s, r.URL.Query(), pl, h.readerLang(r, s), false)
	if err != nil {
		h.serverError(w, r, s, err)
		return
	}
	h.setLang(v, r, s, pl, nil)
	h.render(w, r, s, "archive", status, v, csp{}, false)
}

// archiveView builds the archive page: its own text in pl, each issue in
// the version a reader of reader's language gets. preview renders it for
// the app.
func (h *Handler) archiveView(ctx context.Context, s *site, query url.Values, pl pageLang, reader string, preview bool) (*view, int, error) {
	publisher := s.set.Branding.PublisherName
	pub := i18n.Args{"publisher": publisher}
	l := pl.Labels
	pc := h.copyOf(ctx, s)
	lists, err := h.Audience.PublicLists(ctx, h.Q, s.ws)
	if err != nil {
		return nil, 0, err
	}
	archived := h.archiveLists(ctx, s)
	bySlug := map[string]uuid.UUID{}
	names := map[uuid.UUID]string{}
	slugs := map[uuid.UUID]string{}
	for _, li := range lists {
		names[li.ID] = li.Name
		if pc != nil {
			names[li.ID] = pc.ListName(pl.Code, li)
		}
		// The filter's address keeps the list's own name in every language.
		sl := forms.Slugify(li.Name, 60)
		if sl == "" {
			sl = li.ID.String()
		}
		bySlug[sl], slugs[li.ID] = li.ID, sl
	}
	// Recurring digests with "Show in the archive" on have their own filter
	// (internal/digests), named like a list's; a list keeps its name first.
	// Their newest issues are synced first, as the list below syncs them.
	if err := h.Forms.SyncArchive(ctx, s.ws, s.set); err != nil {
		return nil, 0, err
	}
	digests, err := h.Q.ArchiveDigestCounts(ctx, store.ArchiveDigestCountsParams{WorkspaceID: s.ws.ID, Lists: archived})
	if err != nil {
		return nil, 0, err
	}
	digestBySlug := map[string]uuid.UUID{}
	digestSlugs := map[uuid.UUID]string{}
	for _, dg := range digests {
		sl := forms.Slugify(dg.Name, 60)
		if _, taken := bySlug[sl]; taken || sl == "" {
			sl = dg.ID.String()
		}
		digestBySlug[sl], digestSlugs[dg.ID] = dg.ID, sl
	}
	p := archiveParamsOf(query, bySlug).withDigest(query, digestBySlug)
	if query.Get(langParam) != "" && pl.picked {
		p.lang = pl.Code
	}
	entries, total, err := h.Forms.Archive(ctx, s.ws, s.set, forms.ArchiveQuery{Search: p.q, List: p.listID, Digest: p.digestID, Page: p.page})
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
	b := archiveBody{Summary: archiveSummary(l, listNames, sum.FirstAt, sum.Issues), FeedHref: s.links.Feed(), Action: base, Query: p.q,
		FilterList: p.list, PrefsHref: s.links.PreferencesRequest(), Issues: []issueView{}, Lang: p.lang}
	if s.set.Pages.Archive.SubscribeBox {
		b.SubscribeAction = s.links.Subscribe("")
	}
	if len(listNames) > 1 || len(digests) > 0 {
		b.Filters = append(b.Filters, filterView{Label: l.Text("archive.every", nil), Href: p.href(base, 1, "", p.q), On: p.list == ""})
		for _, id := range archived {
			if withIssues[id] {
				b.Filters = append(b.Filters, filterView{Label: names[id], Href: p.href(base, 1, slugs[id], p.q), On: p.list == slugs[id]})
			}
		}
		for _, dg := range digests {
			b.Filters = append(b.Filters, filterView{Label: dg.Name, Href: p.href(base, 1, digestSlugs[dg.ID], p.q), On: p.list == digestSlugs[dg.ID]})
		}
	}
	loc := location(s.ws)
	versions := h.listVersions(ctx, s, entries, reader)
	written := s.set.CopyLanguage()
	if pc != nil {
		written = pc.Languages.Default
	}
	for _, e := range entries {
		iv := issueView{Date: l.FullDate(e.SentAt.In(loc)), ISO: e.SentAt.UTC().Format(time.RFC3339), Label: e.Label, Title: e.Title,
			Excerpt: e.Excerpt, Href: s.links.Issue(e.Slug)}
		code := written
		if ch, ok := versions[e.CampaignID]; ok {
			code = ch.code
			if ch.code != ch.source {
				if t := forms.RenderTitle(ch.subject); t != "" {
					iv.Title = t
				}
				if iv.Excerpt = forms.RenderTitle(ch.preheader); iv.Excerpt == "" {
					iv.Excerpt = forms.Excerpt(ch.text, 180)
				}
				iv.Href = issueIn(s.links, e.Slug, ch.code)
			}
		}
		if code != pl.Code {
			iv.Lang = code
		}
		if len(listNames) > 1 || e.ListName != "" {
			iv.List = e.ListName
			if e.ListID != nil && names[*e.ListID] != "" {
				iv.List = names[*e.ListID]
			}
		}
		b.Issues = append(b.Issues, iv)
	}
	size := int64(forms.ArchivePageSize)
	from := int64(p.page-1)*size + 1
	to := min(from+int64(len(entries))-1, total)
	if len(entries) > 0 {
		b.Range = l.Text("archive.range", i18n.Args{"from": l.Count(from), "to": l.Count(to), "total": l.Count(total)})
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
		b.Empty = l.Text("archive.noMatch", i18n.Args{"query": p.q})
	case len(entries) == 0 && p.page > 1:
		b.Empty = l.Text("archive.noOlder", nil)
		status = http.StatusNotFound
	case len(entries) == 0:
		b.Empty = l.Text("archive.empty", nil)
	}
	name := l.Text("archive.title", pub)
	v := h.view(s, name, b)
	v.Description = b.Summary
	v.Canonical = p.href(s.canon.Archive(), p.page, p.list, "")
	v.Preview = preview
	if s.set.Pages.Archive.Index && p.q == "" && !preview {
		v.Robots = "index, follow"
	} else {
		v.Robots = "noindex, follow"
	}
	if p.page > 1 {
		v.Title = l.Text("archive.titlePage", i18n.Args{"publisher": publisher, "page": p.page})
	}
	if p.q == "" {
		plain := p
		plain.lang = ""
		v.Alternates = h.alternates(ctx, s, plain.href(s.canon.Archive(), p.page, p.list, ""), nil)
	}
	v.OG = &og{Type: "website", SiteName: publisher, Title: v.Title, Description: v.Description, URL: v.Canonical}
	v.Feed, v.FeedTitle = s.links.Feed(), name
	ld := map[string]any{"@context": "https://schema.org", "@type": "Blog", "name": name, "url": s.canon.Archive(),
		"description": b.Summary, "inLanguage": pl.Code, "publisher": map[string]any{"@type": "Organization", "name": publisher}}
	var posts []map[string]any
	for i, e := range entries {
		post := map[string]any{"@type": "BlogPosting", "headline": b.Issues[i].Title, "datePublished": e.SentAt.UTC().Format(time.RFC3339),
			"url": s.canon.Issue(e.Slug)}
		if ch, ok := versions[e.CampaignID]; ok && ch.code != ch.source {
			post["url"], post["inLanguage"] = issueIn(s.canon, e.Slug, ch.code), ch.code
		}
		posts = append(posts, post)
	}
	if len(posts) > 0 {
		ld["blogPost"] = posts
	}
	v.JSONLD = ld
	return v, status, nil
}

// readerLang is the language a visitor reads issues in: the menu's pick,
// then the first of the browser's languages the workspace offers, then the
// default.
func (h *Handler) readerLang(r *http.Request, s *site) string {
	pc := h.copyOf(r.Context(), s)
	if pc == nil {
		return ""
	}
	for _, w := range append([]string{picked(r)}, audience.BrowserLanguages(r.Header.Get("Accept-Language"))...) {
		if code := languages.Base(w); code != "" && pc.Languages.Offers(code) {
			return code
		}
	}
	return pc.Languages.Default
}

// issueIn is the address of an issue's version in a language.
func issueIn(l forms.PageLinks, slug, code string) string {
	return l.Issue(slug) + "/" + url.PathEscape(code)
}

// issueChoice is the version of an issue a reader gets.
type issueChoice struct {
	code, source             string
	subject, preheader, text string
}

// listVersions picks the version of each listed issue a reader of a
// language gets, as their email would: theirs, else the fallback's, else
// the one the issue is written in. Nil with a single workspace language.
func (h *Handler) listVersions(ctx context.Context, s *site, entries []forms.ArchiveEntry, reader string) map[uuid.UUID]issueChoice {
	pc := h.copyOf(ctx, s)
	if pc == nil || len(pc.Languages.Languages) < 2 || len(entries) == 0 || reader == "" {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.CampaignID)
	}
	want := []string{reader}
	if f := pc.Languages.Fallback; f != "" && f != reader {
		want = append(want, f)
	}
	rows, err := h.Q.ArchiveListVersions(ctx, store.ArchiveListVersionsParams{WorkspaceID: s.ws.ID, Campaigns: ids, Languages: want})
	if err != nil {
		h.log().WarnContext(ctx, "archive versions", "err", err)
		return nil
	}
	sources := map[uuid.UUID]string{}
	found := map[uuid.UUID]map[string]store.ArchiveListVersionsRow{}
	for _, r := range rows {
		sources[r.CampaignID] = r.Source
		if r.Language == "" || !pc.Languages.Offers(r.Language) {
			continue
		}
		if found[r.CampaignID] == nil {
			found[r.CampaignID] = map[string]store.ArchiveListVersionsRow{}
		}
		found[r.CampaignID][r.Language] = r
	}
	out := make(map[uuid.UUID]issueChoice, len(ids))
	for _, id := range ids {
		src, ok := sources[id]
		if !ok {
			src = pc.Languages.Default
		}
		ch := issueChoice{code: src, source: src}
		for _, code := range want {
			if code == src {
				break
			}
			if r, ok := found[id][code]; ok {
				ch = issueChoice{code: code, source: src, subject: r.Subject, preheader: r.Preheader, text: r.PlainText}
				break
			}
		}
		out[id] = ch
	}
	return out
}

// issueLangs are the versions of an archived issue readers can see: the
// one in the language it's written in, and each translated version that
// isn't held in a language the workspace offers, in the workspace's order.
type issueLangs struct {
	source, fallback string
	versions         map[string]store.ArchiveIssueVersionsRow
	order            []string
}

func (h *Handler) issueLangs(ctx context.Context, s *site, campaignID uuid.UUID) issueLangs {
	il := issueLangs{source: s.set.CopyLanguage(), versions: map[string]store.ArchiveIssueVersionsRow{}}
	pc := h.copyOf(ctx, s)
	if pc == nil {
		il.order = []string{il.source}
		return il
	}
	il.source, il.fallback = pc.Languages.Default, pc.Languages.Fallback
	src, err := h.Q.ContentSource(ctx, store.ContentSourceParams{OwnerKind: languages.OwnerCampaign, OwnerID: campaignID})
	switch {
	case err == nil:
		il.source = src.Language
		rows, err := h.Q.ArchiveIssueVersions(ctx, store.ArchiveIssueVersionsParams{WorkspaceID: s.ws.ID, CampaignID: campaignID})
		if err != nil {
			h.log().WarnContext(ctx, "issue versions", "err", err)
		}
		for _, r := range rows {
			if pc.Languages.Offers(r.Language) {
				il.versions[r.Language] = r
			}
		}
	case !errors.Is(err, pgx.ErrNoRows):
		h.log().WarnContext(ctx, "issue language", "err", err)
	}
	il.order = []string{il.source}
	for _, code := range pc.Languages.Languages {
		if _, ok := il.versions[code]; ok {
			il.order = append(il.order, code)
		}
	}
	return il
}

func (il issueLangs) has(code string) bool {
	if code == il.source {
		return true
	}
	_, ok := il.versions[code]
	return ok
}

// pick is the version a reader of a language gets, as their email would:
// theirs, else the fallback's, else the one the issue is written in.
func (il issueLangs) pick(code string) string {
	switch {
	case code != "" && il.has(code):
		return code
	case il.fallback != "" && il.has(il.fallback):
		return il.fallback
	}
	return il.source
}

// canonical is a version's preferred address: the issue's own for the
// language it's written in.
func (il issueLangs) canonical(l forms.PageLinks, slug, code string) string {
	if code == il.source {
		return l.Issue(slug)
	}
	return issueIn(l, slug, code)
}

// alternates are the issue's versions for search engines (hreflang), the
// issue's own address as x-default. None for an issue in one language.
func (il issueLangs) alternates(l forms.PageLinks, slug string) []alternate {
	if len(il.order) < 2 {
		return nil
	}
	out := make([]alternate, 0, len(il.order)+1)
	for _, code := range il.order {
		out = append(out, alternate{Lang: code, Href: il.canonical(l, slug, code)})
	}
	return append(out, alternate{Lang: "x-default", Href: l.Issue(slug)})
}

// also links the issue's other versions, each in its own language's name.
func (il issueLangs) also(l forms.PageLinks, slug, shown string) []langLink {
	var out []langLink
	for _, code := range il.order {
		if code == shown {
			continue
		}
		meta, _ := languages.Lookup(code)
		name := meta.Native
		if name == "" {
			name = code
		}
		out = append(out, langLink{Code: code, Name: name, Href: issueIn(l, slug, code)})
	}
	return out
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
	il := h.issueLangs(r.Context(), s, row.CampaignID)
	var code string
	var pl pageLang
	if asked := chi.URLParam(r, "lang"); asked != "" {
		if !il.has(asked) {
			// Not one of the issue's versions (any longer): the issue in
			// the visitor's language.
			http.Redirect(w, r, s.links.Issue(row.Slug), http.StatusFound)
			return
		}
		// A version asked for by its address: the page around it in its
		// language when the pages are Live in it.
		code = asked
		if pc := h.copyOf(r.Context(), s); pc != nil && pc.Live(code) {
			pl = pageLang{Code: code, Labels: pc.Labels(code)}
		} else {
			pl = h.visitorLang(w, r, s, nil)
		}
	} else {
		pl = h.visitorLang(w, r, s, nil)
		code = il.pick(h.readerLang(r, s))
	}
	doc := h.issueDocument(r, s, row, il, code, pl)
	emailHeaders(w, false)
	w.Header().Add("Vary", "Accept-Language")
	w.Header().Add("Vary", "Cookie")
	if !s.set.Pages.Archive.Index {
		w.Header().Set("X-Robots-Tag", "noindex")
	}
	if r.Method != http.MethodHead {
		_, _ = w.Write([]byte(doc)) //nolint:gosec // the publisher's own email, served under a no-script CSP; inserted values are escaped
	}
}

func (h *Handler) archiveMissing(w http.ResponseWriter, r *http.Request, s *site) {
	pl := h.visitorLang(w, r, s, nil)
	l := pl.Labels
	h.messageIn(w, r, s, pl, http.StatusNotFound, pageTitle(pl, s, "archive.missing.title"), messageBody{
		Heading: l.Text("archive.missing.heading", nil),
		Text:    l.Text("archive.missing.text", nil),
		Actions: []action{{Label: l.Text("archive.heading", nil), Href: s.links.Archive()}},
	}, false)
}

// winnerContent is the subject and HTML readers of the archive see: the
// winning variant of an A/B test, else the campaign itself.
func winnerContent(subject, body string, ab json.RawMessage) (string, string) {
	return variantContent(subject, body, ab, winnerOf(ab))
}

// winnerOf is the winning variant of a campaign's A/B test ("" before one
// wins).
func winnerOf(ab json.RawMessage) string {
	var t struct {
		Winner string `json:"winner"`
	}
	if len(ab) == 0 || json.Unmarshal(ab, &t) != nil {
		return ""
	}
	return t.Winner
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

// issueDocument renders an archived issue in one of its versions: the
// email itself under a bar that links back to the archive and to the
// subscribe page, and over the language menu. pl is the language of the
// bar and the menu.
func (h *Handler) issueDocument(r *http.Request, s *site, row store.ArchiveIssueBySlugRow, il issueLangs, code string, pl pageLang) string {
	ctx := r.Context()
	publisher := s.set.Branding.PublisherName
	pub := i18n.Args{"publisher": publisher}
	l := pl.Labels
	cp, err := h.Q.PublicCampaign(ctx, store.PublicCampaignParams{WorkspaceID: s.ws.ID, ID: row.CampaignID})
	ab := json.RawMessage(nil)
	if err == nil {
		ab = cp.AbTest
	}
	// The publisher's title and excerpt for the archive are written in the
	// issue's own language; a version shows its subject and preheader.
	title, desc := strings.TrimSpace(row.Title), strings.TrimSpace(row.Excerpt)
	preheader, text := row.Preheader, row.PlainText
	var subject, body string
	if v, ok := il.versions[code]; ok && code != il.source {
		subject, body = variantContent(v.Subject, v.Html, v.AbTest, winnerOf(ab))
		preheader, text, title, desc = v.Preheader, v.PlainText, "", ""
	} else {
		subject, body = winnerContent(row.Subject, row.Html, ab)
	}
	links := map[string]string{
		"unsubscribe_url":     s.links.PreferencesRequest(),
		"preferences_url":     s.links.PreferencesRequest(),
		"view_in_browser_url": il.canonical(s.links, row.Slug, code),
	}
	if strings.TrimSpace(body) == "" {
		body = plainDocument(text)
	}
	// Conditional content and polls (design expansion, X11): a reader of the
	// archive matches no condition, and answers as someone who isn't a contact.
	if h.Browser != nil {
		body = h.Browser.BrowserHTML(ctx, s.ws, row.CampaignID, uuid.Nil, body)
		for k, v := range h.Browser.BrowserLinks(ctx, s.ws, row.CampaignID) {
			links[k] = v
		}
	}
	subject, doc := renderEmail(subject, body, "", "", "", nil, links, h.URLs.Base())
	if title == "" {
		title = subject
	}
	if title == "" {
		title = row.Name
	}
	if desc == "" {
		desc = forms.RenderTitle(preheader)
	}
	if desc == "" {
		desc = forms.Excerpt(text, 180)
	}
	canon := il.canonical(s.canon, row.Slug, code)
	sent := row.SentAt.UTC().Format(time.RFC3339)
	image := firstImage(doc)
	archiveName := l.Text("archive.title", pub)
	ld := map[string]any{
		"@context": "https://schema.org", "@type": "BlogPosting", "headline": title, "datePublished": sent,
		"dateModified": row.UpdatedAt.UTC().Format(time.RFC3339), "url": canon, "mainEntityOfPage": canon, "inLanguage": code,
		"author":    map[string]any{"@type": "Organization", "name": publisher},
		"publisher": map[string]any{"@type": "Organization", "name": publisher},
		"isPartOf":  map[string]any{"@type": "Blog", "name": archiveName, "url": s.canon.Archive()},
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
	head := emailHead{Title: title + " · " + l.Text("archive.feedName", pub), Description: desc, Robots: robots, Canonical: canon,
		OG:   &og{Type: "article", SiteName: publisher, Title: title, Description: desc, URL: canon, Image: image, Published: sent},
		Feed: s.links.Feed(), FeedTitle: archiveName, JSONLD: ld, Alternates: il.alternates(s.canon, row.Slug)}
	if s.set.Branding.Logo != nil {
		head.Icon = s.links.Logo(s.set.Branding.Logo.Hash)
	}
	doc = setDocLang(injectHead(doc, head), code)
	meta := []string{}
	if row.ListName != "" {
		name := row.ListName
		if pc := h.copyOf(ctx, s); pc != nil && row.ListID != nil {
			name = pc.ListName(pl.Code, audience.PublicList{ID: *row.ListID, Name: row.ListName})
		}
		meta = append(meta, name)
	}
	meta = append(meta, l.FullDate(row.SentAt.In(location(s.ws))))
	if row.Label != "" {
		meta = append(meta, row.Label)
	}
	doc = injectTop(withStylesheet(doc, s.links.Static("issue.css?v="+h.version)), h.issueBar(s, pl, code, strings.Join(meta, " · "), il.also(s.links, row.Slug, code)))
	menu := h.menu(r, s, pl.Code, nil)
	for i := range menu {
		// The menu opens the issue itself in a language: its version there,
		// as a reader of that language gets it.
		menu[i].Href = s.links.Issue(row.Slug) + "?" + langParam + "=" + url.QueryEscape(menu[i].Code)
	}
	return injectBottom(doc, issueFoot(pl, code, menu))
}

// withStylesheet links the issue bar's stylesheet in the document's head.
func withStylesheet(doc, href string) string {
	link := `<link rel="stylesheet" href="` + html.EscapeString(href) + `">` + "\n"
	if loc := headOpen.FindStringIndex(doc); loc != nil {
		return doc[:loc[1]] + "\n" + link + doc[loc[1]:]
	}
	return link + doc
}

// langAttrs are the lang and dir of a part of an issue page in another
// language than the email's.
func langAttrs(pl pageLang, docLang string) string {
	if pl.Code == docLang {
		return ""
	}
	return ` lang="` + html.EscapeString(pl.Code) + `" dir="` + pagelabels.Dir(pl.Code) + `"`
}

// issueBar is the strip above an archived issue, with the issue's other
// versions under "Also in".
func (h *Handler) issueBar(s *site, pl pageLang, docLang, meta string, also []langLink) string {
	esc := html.EscapeString
	l := pl.Labels
	b := s.set.Branding
	mark := `<span class="rcia-mark" aria-hidden="true">` + esc(b.Initial()) + `</span>`
	if b.Logo != nil {
		mark = `<span class="rcia-mark rcia-logo"><img src="` + esc(s.links.Logo(b.Logo.Hash)) + `" alt="" width="28" height="28"></span>`
	}
	p := b.Palette
	style := fmt.Sprintf(`<style>.rcia{--rcia-accent:%s;--rcia-hover:%s;--rcia-on:%s;--rcia-ground:%s;--rcia-text:%s}</style>`, p.Accent, p.Hover, p.OnAccent, p.Background, p.Text)
	var out strings.Builder
	out.WriteString("\n" + style + `<header class="rcia"` + langAttrs(pl, docLang) + `><div class="rcia-in"><a class="rcia-home" href="` + esc(s.links.Archive()) + `">` + mark +
		`<span>` + esc(b.PublisherName) + ` · ` + esc(l.Text("archive.heading", nil)) + `</span></a><a class="rcia-sub" href="` + esc(s.links.Subscribe("")) + `">` +
		esc(l.Text("form.button", nil)) + `</a></div>` + `<div class="rcia-meta">` + esc(meta) + `</div>`)
	if len(also) > 0 {
		out.WriteString(`<p class="rcia-also">` + esc(l.Text("archive.alsoIn", nil)))
		for i, a := range also {
			if i > 0 {
				out.WriteString(`<span aria-hidden="true">·</span>`)
			}
			out.WriteString(` <a href="` + esc(a.Href) + `" lang="` + esc(a.Code) + `" hreflang="` + esc(a.Code) + `">` + esc(a.Name) + `</a>`)
		}
		out.WriteString(`</p>`)
	}
	out.WriteString("</header>\n")
	return out.String()
}

// issueFoot is the language menu under an archived issue ("" without one).
func issueFoot(pl pageLang, docLang string, menu []langLink) string {
	if len(menu) == 0 {
		return ""
	}
	esc := html.EscapeString
	var out strings.Builder
	out.WriteString("\n" + `<footer class="rcia rcia-foot"` + langAttrs(pl, docLang) + `><nav class="rcia-langs" aria-label="` + esc(pl.Labels.Text("page.languages", nil)) + `">` + globe)
	for i, m := range menu {
		if i > 0 {
			out.WriteString(`<span aria-hidden="true">·</span>`)
		}
		current := ""
		if m.Current {
			current = ` aria-current="true"`
		}
		out.WriteString(`<a href="` + esc(m.Href) + `" lang="` + esc(m.Code) + `" hreflang="` + esc(m.Code) + `"` + current + `>` + esc(m.Name) + `</a>`)
	}
	out.WriteString("</nav></footer>\n")
	return out.String()
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
	// The feed is the issues as they're written, in the default language.
	pl := h.fixedLang(r.Context(), s, h.defaultLang(r.Context(), s), nil)
	pub := i18n.Args{"publisher": s.set.Branding.PublisherName}
	ch := rssChannel{Title: pl.Labels.Text("archive.feedName", pub), Link: s.canon.Archive(), Description: pl.Labels.Text("archive.feedDescription", pub),
		Language: pl.Code, Self: rssAtom{Href: s.canon.Feed(), Rel: "self", Type: "application/rss+xml"}, Items: []rssItem{}}
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
