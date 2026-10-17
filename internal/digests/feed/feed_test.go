package feed

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/netguard"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mustURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func TestParseWordPressRSS(t *testing.T) {
	f, err := Parse(fixture(t, "wordpress.rss"), mustURL("https://blog.acme.io/feed/"))
	if err != nil {
		t.Fatal(err)
	}
	if f.Kind != KindRSS || f.Title != "Acme engineering blog" || f.Link != "https://blog.acme.io" {
		t.Fatalf("feed: %+v", f)
	}
	// The repeated post is read once.
	if len(f.Items) != 4 {
		t.Fatalf("items: %d", len(f.Items))
	}
	rollouts, oncall, migrations, partitioning := f.Items[0], f.Items[1], f.Items[2], f.Items[3]
	if rollouts.Title != "Scheduled rollouts are here" || rollouts.Key != "https://blog.acme.io/?p=812" || rollouts.Author != "Joel Weber" {
		t.Fatalf("rollouts: %+v", rollouts)
	}
	if got := strings.Join(rollouts.Categories, ","); got != "Announcements" {
		t.Fatalf("rollouts categories: %s", got)
	}
	// The excerpt drops WordPress's "[…]".
	if rollouts.Excerpt != "You can now schedule flag changes ahead of time, in any timezone." {
		t.Fatalf("rollouts excerpt: %q", rollouts.Excerpt)
	}
	// The first image of the content, with its own alt text.
	if rollouts.Image != "https://blog.acme.io/wp-content/uploads/2026/10/scheduled-rollouts.png" || rollouts.ImageAlt != "A rollout calendar" {
		t.Fatalf("rollouts image: %q %q", rollouts.Image, rollouts.ImageAlt)
	}
	// Tags are told apart by the category's domain.
	if strings.Join(oncall.Categories, ",") != "Engineering" || strings.Join(oncall.Tags, ",") != "on-call" {
		t.Fatalf("on-call terms: %v %v", oncall.Categories, oncall.Tags)
	}
	// Scripts, styles and the "appeared first on" footer go; entities are decoded.
	if oncall.Excerpt != "We were paged 1,412 times in 2025, and a third of those pages came from the same three alerts. Here’s what we changed." {
		t.Fatalf("on-call excerpt: %q", oncall.Excerpt)
	}
	// A relative image in the content resolves against the feed.
	if oncall.Image != "https://blog.acme.io/wp-content/uploads/2026/10/on-call-2025.png" {
		t.Fatalf("on-call image: %q", oncall.Image)
	}
	if oncall.Published == nil || !oncall.Published.Equal(time.Date(2026, 10, 15, 11, 30, 0, 0, time.UTC)) {
		t.Fatalf("on-call date: %v", oncall.Published)
	}
	// Media RSS gives the image and its description.
	if migrations.Image != "https://blog.acme.io/wp-content/uploads/2026/10/migration-pipeline.png" || migrations.ImageAlt != "A diagram of the migration pipeline" {
		t.Fatalf("migrations image: %q %q", migrations.Image, migrations.ImageAlt)
	}
	if partitioning.Image != "" || partitioning.Excerpt != "Our events table passed 3 billion rows in August & kept growing." {
		t.Fatalf("partitioning: %+v", partitioning)
	}
}

func TestParseAtom(t *testing.T) {
	f, err := Parse(fixture(t, "atom.xml"), mustURL("https://notes.acme.io/atom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if f.Kind != KindAtom || f.Title != "Acme status & engineering notes" || f.Link != "https://notes.acme.io/" || len(f.Items) != 2 {
		t.Fatalf("feed: %+v", f)
	}
	// Newest first: the Rust post (8 Oct) before the flags post (7 Oct).
	rust, flags := f.Items[0], f.Items[1]
	if flags.Title != "Feature flags at 2,000 teams: what broke first" || flags.URL != "https://notes.acme.io/flags-at-2000-teams" {
		t.Fatalf("flags: %+v", flags)
	}
	if flags.Key != "tag:notes.acme.io,2026:flags-2000" || flags.Author != "Joel Weber" || flags.Image != "https://notes.acme.io/img/flag-latency.png" {
		t.Fatalf("flags: %+v", flags)
	}
	if strings.Join(flags.Categories, ",") != "Engineering" || strings.Join(flags.Tags, ",") != "flags" {
		t.Fatalf("flags terms: %v %v", flags.Categories, flags.Tags)
	}
	if flags.Excerpt != "When the 2,000th team turned on feature flags, rule evaluation went from 2 ms to 90 ms." {
		t.Fatalf("flags excerpt: %q", flags.Excerpt)
	}
	// The published date wins over updated.
	if flags.Published == nil || flags.Published.UTC().Hour() != 14 {
		t.Fatalf("flags date: %v", flags.Published)
	}
	// An entry without an author takes the feed's; xhtml content becomes text.
	if rust.Author != "Acme Engineering" || rust.Image != "https://notes.acme.io/img/ingest-throughput.png" || rust.Excerpt != "Our event ingest handled 40,000 events a second in Go." {
		t.Fatalf("rust: %+v", rust)
	}
}

func TestParseJSONFeed(t *testing.T) {
	f, err := Parse(fixture(t, "feed.json"), mustURL("https://changelog.acme.io/feed.json"))
	if err != nil {
		t.Fatal(err)
	}
	if f.Kind != KindJSON || f.Title != "Acme changelog" || len(f.Items) != 2 {
		t.Fatalf("feed: %+v", f)
	}
	a, b := f.Items[0], f.Items[1]
	if a.Key != "2026-10-01-services-back-to-postgres" || a.Author != "Ana Lima" || a.Image != "https://changelog.acme.io/img/services-postgres.png" || a.ImageAlt != "Three services, one database" {
		t.Fatalf("first: %+v", a)
	}
	if strings.Join(a.Categories, ",") != "Engineering,postgres" || a.Excerpt != "Two years ago we gave three services their own document stores." {
		t.Fatalf("first terms: %+v", a)
	}
	// A 1.0-style numeric id, the summary as excerpt, the feed's author.
	if b.Key != "42" || b.Excerpt != "We replay a sample of production traffic against every release candidate." || b.Author != "Acme Product" || b.Image != "https://changelog.acme.io/img/load.png" {
		t.Fatalf("second: %+v", b)
	}
}

func TestParseNotAFeed(t *testing.T) {
	_, err := Parse(fixture(t, "maintenance.html"), nil)
	var nf *NotFeedError
	if !errors.As(err, &nf) || !nf.HTML {
		t.Fatalf("html: %v", err)
	}
	if len(nf.Lines) != 4 || nf.Lines[0] != "<!DOCTYPE html>" || !strings.Contains(nf.Lines[2], "Down for maintenance") {
		t.Fatalf("lines: %q", nf.Lines)
	}
	for _, c := range []struct{ name, doc, want string }{
		{"broken XML", string(fixture(t, "broken.rss")), "doesn't parse"},
		{"another XML document", `<?xml version="1.0"?><urlset><url><loc>https://x.io/</loc></url></urlset>`, "root is <urlset>"},
		{"JSON that isn't a feed", `{"posts":[]}`, "isn't a JSON Feed"},
		{"plain text", "Service unavailable", "neither XML nor JSON"},
		{"empty", "   ", "empty"},
	} {
		_, err := Parse([]byte(c.doc), nil)
		if !IsNotFeed(err) || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s: %v", c.name, err)
		}
	}
}

func TestParseHugeFeedKeepsTheNewest(t *testing.T) {
	var b bytes.Buffer
	b.WriteString(`<rss version="2.0"><channel><title>Big</title>`)
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range 1000 {
		fmt.Fprintf(&b, `<item><title>Post %d</title><link>https://big.example/%d</link><pubDate>%s</pubDate><description>%s</description></item>`,
			i, i, start.Add(time.Duration(i)*time.Hour).Format(time.RFC1123Z), strings.Repeat("word ", 50))
	}
	b.WriteString(`</channel></rss>`)
	f, err := Parse(b.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Items) != MaxItems || f.Items[0].Title != "Post 999" || f.Items[MaxItems-1].Title != "Post 800" {
		t.Fatalf("kept %d, newest %q, oldest %q", len(f.Items), f.Items[0].Title, f.Items[len(f.Items)-1].Title)
	}
}

func TestDates(t *testing.T) {
	for _, s := range []string{"Wed, 15 Oct 2026 11:30:00 +0000", "Wed, 15 Oct 2026 07:30:00 EDT", "2026-10-15T07:30:00-04:00", "15 Oct 2026 11:30:00 +0000", "Wed, 15 Oct 2026 11:30:00 GMT"} {
		got, ok := ParseDate(s)
		if !ok || !got.Equal(time.Date(2026, 10, 15, 11, 30, 0, 0, time.UTC)) {
			t.Fatalf("%q: %v %v", s, got, ok)
		}
	}
	if _, ok := ParseDate("yesterday"); ok {
		t.Fatal("parsed nonsense")
	}
}

func TestExcerptAndWords(t *testing.T) {
	got := Excerpt(`<p>One&nbsp;two</p><p>three <b>four</b></p><script>var x = "five"</script><noscript>six</noscript>`)
	if got != "One two three four" {
		t.Fatalf("excerpt: %q", got)
	}
	if w := Words("one two three four five", 3); w != "one two three…" {
		t.Fatalf("words: %q", w)
	}
	if w := Words("one two, three.", 5); w != "one two, three." {
		t.Fatalf("words: %q", w)
	}
	if w := Words("one two, three four", 2); w != "one two…" {
		t.Fatalf("words with a comma: %q", w)
	}
}

func TestDiscover(t *testing.T) {
	links := Discover(fixture(t, "home.html"), mustURL("https://blog.acme.io/"))
	if len(links) != 2 || links[0].URL != "https://blog.acme.io/feed/" || links[1].URL != "https://blog.acme.io/comments/feed/" {
		t.Fatalf("links: %+v", links)
	}
}

// sites answers requests in-process: the address checks of the guard still
// run on every address and redirect, nothing touches the network.
type sites map[string]http.HandlerFunc

func (s sites) RoundTrip(r *http.Request) (*http.Response, error) {
	h, ok := s[r.URL.Host+r.URL.Path]
	if !ok {
		return nil, fmt.Errorf("dial tcp: lookup %s: no such host", r.URL.Host)
	}
	rec := &recorder{header: http.Header{}, status: 200}
	h(rec, r)
	return &http.Response{StatusCode: rec.status, Header: rec.header, Body: io.NopCloser(bytes.NewReader(rec.body.Bytes())), Request: r,
		ContentLength: int64(rec.body.Len())}, nil
}

type recorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (r *recorder) Header() http.Header         { return r.header }
func (r *recorder) Write(b []byte) (int, error) { return r.body.Write(b) }
func (r *recorder) WriteHeader(s int)           { r.status = s }

func TestFetchFollowsRedirectsAndFindsFeeds(t *testing.T) {
	rss := fixture(t, "wordpress.rss")
	conditional := 0
	f := &Fetcher{Policy: netguard.Policy{}, Transport: sites{
		"acme.io/blog/rss": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", "https://blog.acme.io/feed/")
			w.WriteHeader(http.StatusMovedPermanently)
		},
		"blog.acme.io/": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(fixture(t, "home.html")) },
		"blog.acme.io/feed/": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("If-None-Match") == `"v1"` {
				conditional++
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("ETag", `"v1"`)
			w.Header().Set("Content-Type", "application/rss+xml")
			_, _ = w.Write(rss)
		},
	}}
	ctx := context.Background()
	// A feed that moved: followed, and its new address offered.
	res, err := f.Read(ctx, Request{URL: "acme.io/blog/rss"})
	if err != nil {
		t.Fatal(err)
	}
	if res.URL != "https://blog.acme.io/feed/" || res.Moved != "https://blog.acme.io/feed/" || len(res.Hops) != 2 || res.Hops[0].Status != 301 || len(res.Feed.Items) != 4 {
		t.Fatalf("moved: url %s moved %s hops %+v items %d", res.URL, res.Moved, res.Hops, len(res.Feed.Items))
	}
	// A conditional read of an unchanged feed.
	res, err = f.Read(ctx, Request{URL: "https://blog.acme.io/feed/", ETag: `"v1"`})
	if err != nil || !res.NotModified || res.Feed != nil || conditional != 1 {
		t.Fatalf("conditional: %+v %v", res, err)
	}
	// A page address finds the feed its head links to.
	found, err := f.Find(ctx, "blog.acme.io")
	if err != nil {
		t.Fatal(err)
	}
	if found.Page != "https://blog.acme.io/" || found.URL != "https://blog.acme.io/feed/" || found.Feed.Kind != KindRSS {
		t.Fatalf("found: %+v", found)
	}
}

func TestFetchRefusesPrivateAddresses(t *testing.T) {
	f := &Fetcher{Policy: netguard.Policy{}, Transport: sites{
		"feeds.example.com/private": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", "http://10.0.4.12/feed")
			w.WriteHeader(http.StatusFound)
		},
		"feeds.example.com/metadata": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", "http://169.254.169.254/latest/meta-data/")
			w.WriteHeader(http.StatusFound)
		},
		"feeds.example.com/local": func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", "http://localhost:8080/feed")
			w.WriteHeader(http.StatusTemporaryRedirect)
		},
	}}
	ctx := context.Background()
	for _, path := range []string{"private", "metadata", "local"} {
		_, err := f.Read(ctx, Request{URL: "https://feeds.example.com/" + path})
		fe, ok := AsFetchError(err)
		if !ok || fe.Kind != ErrBlocked || len(fe.Hops) != 1 || !strings.Contains(fe.Detail, "can't be read") {
			t.Fatalf("%s: %v", path, err)
		}
	}
	// A typed private address is refused before any connection.
	_, err := f.Read(ctx, Request{URL: "http://127.0.0.1:8080/feed"})
	if fe, ok := AsFetchError(err); !ok || fe.Kind != ErrBlocked {
		t.Fatalf("loopback: %v", err)
	}
	// Allowing private networks never allows metadata addresses.
	f.Policy = netguard.Policy{AllowPrivate: true}
	_, err = f.Read(ctx, Request{URL: "https://feeds.example.com/metadata"})
	if fe, ok := AsFetchError(err); !ok || fe.Kind != ErrBlocked {
		t.Fatalf("metadata with private allowed: %v", err)
	}
}

func TestFetchLimits(t *testing.T) {
	big := bytes.Repeat([]byte("x"), 6<<20)
	hops := 0
	f := &Fetcher{Policy: netguard.Policy{}, Transport: sites{
		"feeds.example.com/huge": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(big) },
		"feeds.example.com/loop": func(w http.ResponseWriter, _ *http.Request) {
			hops++
			w.Header().Set("Location", "/loop")
			w.WriteHeader(http.StatusFound)
		},
		"feeds.example.com/gone": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
		"feeds.example.com/html": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(fixture(t, "maintenance.html")) },
	}}
	ctx := context.Background()
	_, err := f.Read(ctx, Request{URL: "https://feeds.example.com/huge"})
	if fe, ok := AsFetchError(err); !ok || fe.Kind != ErrTooLarge || !strings.Contains(fe.Detail, "more than 5 MB") {
		t.Fatalf("huge: %v", err)
	}
	_, err = f.Read(ctx, Request{URL: "https://feeds.example.com/loop"})
	if fe, ok := AsFetchError(err); !ok || fe.Kind != ErrRedirects || hops != 4 {
		t.Fatalf("redirect loop: %v (%d hops)", err, hops)
	}
	_, err = f.Read(ctx, Request{URL: "https://feeds.example.com/gone"})
	if fe, ok := AsFetchError(err); !ok || fe.Kind != ErrStatus || fe.Status != 404 || !strings.Contains(fe.Detail, "no feed at feeds.example.com/gone") {
		t.Fatalf("404: %v", err)
	}
	res, err := f.Read(ctx, Request{URL: "https://feeds.example.com/html"})
	if !IsNotFeed(err) || res == nil || res.Status != 200 {
		t.Fatalf("html page: %v", err)
	}
	_, err = f.Read(ctx, Request{URL: "https://nowhere.example.com/feed"})
	if fe, ok := AsFetchError(err); !ok || fe.Kind != ErrUnreachable || !strings.Contains(fe.Detail, "nowhere.example.com didn't answer") {
		t.Fatalf("unreachable: %v", err)
	}
}

func TestFetchTimesOut(t *testing.T) {
	f := &Fetcher{Policy: netguard.Policy{}, Timeout: 50 * time.Millisecond, Transport: slow{}}
	_, err := f.Read(context.Background(), Request{URL: "https://blog.acme.io/feed/"})
	fe, ok := AsFetchError(err)
	if !ok || fe.Kind != ErrTimeout || fe.Detail != "blog.acme.io didn't answer: connection timed out after 0 s" {
		t.Fatalf("timeout: %v", err)
	}
}

type slow struct{}

func (slow) RoundTrip(r *http.Request) (*http.Response, error) {
	<-r.Context().Done()
	return nil, r.Context().Err()
}

func TestNormalizeURL(t *testing.T) {
	for in, want := range map[string]string{"blog.acme.io": "https://blog.acme.io/", "blog.acme.io/feed/": "https://blog.acme.io/feed/", "http://x.io/rss#top": "http://x.io/rss"} {
		if got, err := NormalizeURL(in); err != nil || got != want {
			t.Fatalf("%q: %q %v", in, got, err)
		}
	}
	for _, in := range []string{"", "ftp://x.io/feed", "not an address"} {
		if _, err := NormalizeURL(in); err == nil {
			t.Fatalf("%q accepted", in)
		}
	}
}
