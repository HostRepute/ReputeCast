package digests

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/digests/feed"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// How the feed was read: the 30-minute schedule, at preparation, because
// the WordPress plugin reported a post, by a person (Read now), or while
// the digest was set up.
const (
	ViaSchedule    = "schedule"
	ViaPreparation = "preparation"
	ViaPlugin      = "plugin"
	ViaPerson      = "person"
	ViaSetup       = "setup"
)

// ReadEvery is how often the feed of a digest that is on is read.
const ReadEvery = 30 * time.Minute

// imagesPerRead bounds how many new images one read copies.
const imagesPerRead = 12

// ReadResult is what a read found.
type ReadResult struct {
	OK          bool
	NotModified bool
	Items       int
	New         int
	Err         error
	// Address is the address that was read.
	Address string
}

// feedAddress is the address a digest reads: its WordPress site's reported
// feed, or its own.
func (s *Service) feedAddress(ctx context.Context, d store.Digest) string {
	if d.Source == "wordpress" && d.SiteID != nil {
		if site, err := s.siteRow(ctx, d.WorkspaceID, *d.SiteID); err == nil && site != nil && site.FeedUrl != "" {
			return site.FeedUrl
		}
	}
	return d.FeedUrl
}

// Read reads a digest's feed now and keeps its items.
func (s *Service) Read(ctx context.Context, ws *store.Workspace, id uuid.UUID, via string) (ReadResult, error) {
	d, err := s.Q.DigestByID(ctx, id)
	if err != nil {
		return ReadResult{}, api.MapError(err, "Digest")
	}
	return s.read(ctx, ws, d, via)
}

func (s *Service) read(ctx context.Context, ws *store.Workspace, d store.Digest, via string) (ReadResult, error) {
	address := s.feedAddress(ctx, d)
	now := s.now()
	req := feed.Request{URL: address}
	// A conditional request only when the items were kept from that read.
	if d.FeedStatus == "ok" && d.FeedItems > 0 {
		req.ETag, req.Modified = d.FeedEtag, d.FeedModified
	}
	start := time.Now()
	res, err := s.fetcher().Read(ctx, req)
	ms := int32(time.Since(start) / time.Millisecond)
	log := store.DigestFeedReadInsertParams{DigestID: d.ID, At: now, Via: via, Hops: []byte(`[]`)}
	upd := store.DigestSetFeedParams{ID: d.ID, FeedStatus: "ok", FeedKind: d.FeedKind, FeedTitle: d.FeedTitle, FeedItems: d.FeedItems, FeedBytes: d.FeedBytes,
		FeedEtag: d.FeedEtag, FeedModified: d.FeedModified, At: &now, Via: via}
	if res != nil && res.Response != nil {
		hops, _ := json.Marshal(res.Hops)
		log.Hops, log.HttpStatus, log.Bytes, log.Ms = hops, int32(res.Status), int32(len(res.Body)), ms
		upd.FeedMovedTo = res.Moved
	}
	out := ReadResult{Address: address}
	if err != nil {
		out.Err = err
		log.Ok = false
		log.Error = err.Error()
		kind := "unreachable"
		var nf *feed.NotFeedError
		if errors.As(err, &nf) {
			kind = "invalid"
			upd.FeedSnippet = strings.Join(nf.Lines, "\n")
		}
		if fe, ok := feed.AsFetchError(err); ok {
			hops, _ := json.Marshal(fe.Hops)
			log.Hops, log.HttpStatus = hops, int32(fe.Status)
			log.Error = fe.Detail
		}
		upd.FeedStatus, upd.FeedError = kind, encodeFeedError(err, address)
		upd.FeedEtag, upd.FeedModified = "", ""
		if err := s.Q.DigestSetFeed(ctx, upd); err != nil {
			return out, err
		}
		_ = s.Q.DigestFeedReadInsert(ctx, log)
		return out, nil
	}
	out.OK = true
	log.Ok = true
	log.NotModified = res.NotModified
	if res.NotModified {
		out.NotModified, out.Items = true, int(d.FeedItems)
		log.Items, log.Kind = d.FeedItems, d.FeedKind
		if err := s.Q.DigestSetFeed(ctx, upd); err != nil {
			return out, err
		}
		_ = s.Q.DigestFeedReadInsert(ctx, log)
		return out, nil
	}
	fd := res.Feed
	keys := make([]string, 0, len(fd.Items))
	for _, it := range fd.Items {
		r, err := s.Q.DigestItemUpsert(ctx, store.DigestItemUpsertParams{ID: uuid.Must(uuid.NewV7()), DigestID: d.ID, Key: clip(it.Key, 1000), Url: clip(it.URL, 2000),
			Title: clip(it.Title, 500), Author: clip(it.Author, 200), PublishedAt: it.Published, Categories: clipAll(it.Categories), Tags: clipAll(it.Tags),
			Excerpt: clip(it.Excerpt, 4000), ImageUrl: clip(it.Image, 2000), ImageAlt: clip(it.ImageAlt, 500), Now: now, Via: via})
		if err != nil {
			return out, err
		}
		if r.Inserted {
			out.New++
		}
		keys = append(keys, clip(it.Key, 1000))
	}
	if err := s.Q.DigestItemsMarkGone(ctx, store.DigestItemsMarkGoneParams{DigestID: d.ID, Keys: keys, Now: now}); err != nil {
		return out, err
	}
	out.Items = len(fd.Items)
	upd.FeedKind, upd.FeedTitle, upd.FeedItems, upd.FeedBytes = fd.Kind, clip(fd.Title, 300), int32(len(fd.Items)), int32(len(res.Body))
	upd.FeedEtag, upd.FeedModified = clip(res.ETag, 300), clip(res.Modified, 100)
	if err := s.Q.DigestSetFeed(ctx, upd); err != nil {
		return out, err
	}
	log.Items, log.NewItems, log.Kind = int32(len(fd.Items)), int32(out.New), fd.Kind
	_ = s.Q.DigestFeedReadInsert(ctx, log)
	s.copyImages(ctx, ws, d.ID)
	return out, nil
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	r := []rune(s)
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}

func clipAll(xs []string) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		out = append(out, clip(x, 100))
	}
	if len(out) > 30 {
		out = out[:30]
	}
	return out
}

// storedFeedError is digests.feed_error: the failure, as the screen needs
// it. Detail keeps its English; Text is the same sentence as a message,
// written in each reader's language (rows from before have none).
type storedFeedError struct {
	Kind    string          `json:"kind"`
	Host    string          `json:"host"`
	URL     string          `json:"url"`
	Status  int             `json:"status,omitempty"`
	Detail  string          `json:"detail"`
	Lines   []string        `json:"lines,omitempty"`
	Hops    []DigestFeedHop `json:"hops,omitempty"`
	Seconds int             `json:"seconds,omitempty"`
	Text    *i18n.Text      `json:"text,omitempty"`
}

func encodeFeedError(err error, address string) string {
	out, text := toFeedError(err, address)
	b, _ := json.Marshal(storedFeedError{Kind: out.Kind, Host: out.Host, URL: out.URL, Status: out.Status, Detail: out.Detail, Lines: out.Lines,
		Hops: out.Hops, Seconds: out.Seconds, Text: text})
	return string(b)
}

// decodeFeedError reads digests.feed_error for a reader.
func decodeFeedError(raw string, l i18n.Lang) *DigestFeedError {
	if raw == "" {
		return nil
	}
	var e storedFeedError
	if json.Unmarshal([]byte(raw), &e) != nil {
		return &DigestFeedError{Kind: "unreachable", Detail: raw, Lines: []string{}, Hops: []DigestFeedHop{}}
	}
	out := &DigestFeedError{Kind: e.Kind, Host: e.Host, URL: e.URL, Status: e.Status, Detail: textIn(e.Text, l, e.Detail), Lines: e.Lines, Hops: e.Hops, Seconds: e.Seconds}
	if out.Lines == nil {
		out.Lines = []string{}
	}
	if out.Hops == nil {
		out.Hops = []DigestFeedHop{}
	}
	return out
}

// toFeedError describes a failed read for the screen: Detail in English,
// and the same sentence as a message for a reader's language (nil when
// the error has none).
func toFeedError(err error, address string) (DigestFeedError, *i18n.Text) {
	out := DigestFeedError{Kind: "unreachable", Host: hostOf(address), URL: address, Detail: err.Error(), Lines: []string{}, Hops: []DigestFeedHop{}}
	var text *i18n.Text
	if t := feed.TextOf(err, address); !t.IsZero() {
		text = &t
		out.Detail = t.In(i18n.Default)
	}
	var nf *feed.NotFeedError
	var nl *feed.NoFeedLinkError
	switch {
	case errors.As(err, &nf):
		out.Kind = "not_feed"
		out.Lines = nf.Lines
		if out.Lines == nil {
			out.Lines = []string{}
		}
	case errors.As(err, &nl):
		out.Kind, out.URL = "no_feed_link", nl.Page
	}
	if fe, ok := feed.AsFetchError(err); ok {
		out.Kind, out.Status, out.Detail = fe.Kind, fe.Status, fe.Detail
		if fe.URL != "" {
			out.URL, out.Host = fe.URL, fe.Host()
		}
		for _, h := range fe.Hops {
			out.Hops = append(out.Hops, DigestFeedHop{URL: h.URL, Status: h.Status, Location: h.Location})
		}
		if fe.Kind == feed.ErrTimeout {
			out.Seconds = int(feed.Timeout / time.Second)
		}
	}
	return out, text
}

// DigestFindInput is an address typed in the setup.
type DigestFindInput struct {
	Address string `json:"address" minLength:"1" maxLength:"500"`
}

// DigestFound is the feed found from an address.
type DigestFound struct {
	OK bool `json:"ok"`
	// URL is the feed's address; Page the page it was linked from.
	URL   string `json:"url,omitempty"`
	Page  string `json:"page,omitempty"`
	Kind  string `json:"kind,omitempty"`
	Title string `json:"title,omitempty"`
	Items int    `json:"items"`
	Bytes int    `json:"bytes"`
	// Moved is the address a permanent redirect leads to.
	Moved string           `json:"moved,omitempty"`
	Hops  []DigestFeedHop  `json:"hops"`
	Error *DigestFeedError `json:"error,omitempty"`
	// Preview is the feed's newest items.
	Preview []DigestFeedItem `json:"preview"`
}

// Find reads an address as a feed, or as a page linking to one. It changes
// nothing: the address is saved with the settings.
func (s *Service) Find(ctx context.Context, ws *store.Workspace, in DigestFindInput) (DigestFound, error) {
	l := i18n.LangFrom(ctx)
	u, err := feed.NormalizeURL(in.Address)
	if err != nil {
		return DigestFound{}, api.Invalid("body.address", addressProblem(l, err, in.Address))
	}
	found, err := s.fetcher().Find(ctx, u)
	if err != nil {
		fe, text := toFeedError(err, u)
		fe.Detail = textIn(text, l, fe.Detail)
		return DigestFound{Error: &fe, Hops: fe.Hops, Preview: []DigestFeedItem{}}, nil
	}
	out := DigestFound{OK: true, URL: found.URL, Page: found.Page, Kind: found.Feed.Kind, Title: found.Feed.Title, Items: len(found.Feed.Items), Bytes: len(found.Body),
		Moved: found.Moved, Hops: []DigestFeedHop{}, Preview: []DigestFeedItem{}}
	for _, h := range found.Hops {
		out.Hops = append(out.Hops, DigestFeedHop{URL: h.URL, Status: h.Status, Location: h.Location})
	}
	for i, it := range found.Feed.Items {
		if i == 6 {
			break
		}
		out.Preview = append(out.Preview, DigestFeedItem{ID: it.Key, Title: it.Title, URL: it.URL, Author: it.Author, Published: it.Published,
			Categories: nonNil(it.Categories), Tags: nonNil(it.Tags), Excerpt: feed.Words(it.Excerpt, 40), Status: "new"})
	}
	return out, nil
}

func nonNil(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

// siteRow is a connected WordPress site of the workspace.
func (s *Service) siteRow(ctx context.Context, wsID, id uuid.UUID) (*store.DigestSitesRow, error) {
	rows, err := s.Q.DigestSites(ctx, wsID)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].ID == id {
			return &rows[i], nil
		}
	}
	return nil, nil
}

// Sites lists the workspace's WordPress sites as feed sources.
func (s *Service) Sites(ctx context.Context, ws *store.Workspace) (DigestSites, error) {
	rows, err := s.Q.DigestSites(ctx, ws.ID)
	if err != nil {
		return DigestSites{}, err
	}
	out := DigestSites{Items: []DigestSite{}}
	for _, r := range rows {
		out.Items = append(out.Items, s.siteOf(ctx, ws, r))
	}
	return out, nil
}

func (s *Service) site(ctx context.Context, ws *store.Workspace, id uuid.UUID) (*DigestSite, error) {
	r, err := s.siteRow(ctx, ws.ID, id)
	if err != nil || r == nil {
		return nil, err
	}
	site := s.siteOf(ctx, ws, *r)
	return &site, nil
}

func (s *Service) siteOf(ctx context.Context, ws *store.Workspace, r store.DigestSitesRow) DigestSite {
	host := hostOf(firstNonEmpty(r.SiteUrl, r.Address))
	out := DigestSite{ID: r.ID.String(), Host: host, Name: r.Name, FeedURL: r.FeedUrl, ConnectedAt: r.ConnectedAt, ConnectedBy: r.ConnectedByName,
		PluginVersion: r.PluginVersion, LastReportAt: r.LastReportAt, Status: "ok", DraftCategories: []string{}}
	if s.LatestPlugin != nil {
		out.LatestPlugin = s.LatestPlugin(ctx)
	}
	if out.FeedURL == "" && host != "" {
		out.FeedURL = "https://" + host + "/feed/"
	}
	switch {
	case r.LastReportAt == nil || s.now().Sub(*r.LastReportAt) > 24*time.Hour:
		out.Status = "silent"
	case out.LatestPlugin != "" && newerVersion(out.LatestPlugin, r.PluginVersion):
		out.Status = "outdated"
	}
	// What the site's post-draft rule turns into drafts.
	if r.RuleEnabled {
		var rule struct {
			Categories []string `json:"categories"`
			ListIDs    []string `json:"listIds"`
		}
		_ = json.Unmarshal(r.Rule, &rule)
		var cats []struct {
			Name string `json:"name"`
			Slug string `json:"slug"`
		}
		_ = json.Unmarshal(r.Categories, &cats)
		for _, slug := range rule.Categories {
			name := slug
			for _, c := range cats {
				if c.Slug == slug {
					name = c.Name
				}
			}
			out.DraftCategories = append(out.DraftCategories, name)
		}
		if len(rule.ListIDs) > 0 {
			if lid, err := uuid.Parse(rule.ListIDs[0]); err == nil {
				if lists, err := s.Q.ListWorkspaceLists(ctx, ws.ID); err == nil {
					for _, l := range lists {
						if l.ID == lid {
							out.DraftList = l.Name
						}
					}
				}
			}
		}
	}
	items, _ := s.Q.DigestSiteFeedItems(ctx, &r.ID)
	out.Items = int(items)
	return out
}

// newerVersion reports whether a is a later version than b ("1.3.0" > "1.2.0").
func newerVersion(a, b string) bool {
	parse := func(v string) [3]int {
		var out [3]int
		for i, p := range strings.SplitN(strings.TrimPrefix(v, "v"), ".", 3) {
			n := 0
			for _, r := range p {
				if r < '0' || r > '9' {
					break
				}
				n = n*10 + int(r-'0')
			}
			out[i] = n
		}
		return out
	}
	x, y := parse(a), parse(b)
	for i := range 3 {
		if x[i] != y[i] {
			return x[i] > y[i]
		}
	}
	return false
}

// PostReported is the WordPress plugin's report of a published post: the
// feeds of the digests reading that site are read again at once, so their
// preview shows the post without waiting for the 30-minute read.
func (s *Service) PostReported(ctx context.Context, siteID uuid.UUID) {
	rows, err := s.Q.DigestsBySite(ctx, &siteID)
	if err != nil {
		return
	}
	for _, r := range rows {
		s.enqueue(ctx, ReadArgs{DigestID: r.ID, Via: ViaPlugin})
	}
}
