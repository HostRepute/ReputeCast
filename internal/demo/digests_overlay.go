package demo

import (
	"bytes"
	"embed"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"
)

// The Acme engineering blog (blog.acme.io) of the recurring digest sample:
// its posts, its RSS feed and its images. In development and tests,
// DigestFeedOverlay answers the blog's addresses offline, as the readiness
// overlay answers the sample emails' links: the blog is a real domain the
// demo has no business calling.

//go:embed assets/digests
var digestAssets embed.FS

// BlogFeedURL is the feed the WordPress plugin reports for blog.acme.io.
const BlogFeedURL = "https://blog.acme.io/feed/"

// BlogFeedETag is the version of the feed the demo seeded: a read that
// sends it back gets 304 Not Modified, so the seeded items stay as they are.
const BlogFeedETag = `"demo-blog-feed"`

// blogFeedTitle is the feed's channel title.
const blogFeedTitle = "Acme engineering blog"

// blogPost is a post of the sample blog, dated like the rest of the demo
// (daysAgo, hour and minute of designDay).
type blogPost struct {
	id                     int
	slug, title, author    string
	category               string
	tags                   []string
	daysAgo, hour, min     int
	excerpt                string
	image, imageAlt        string
	seenDays, seenH, seenM int
	seenVia                string
}

// blogPosts are the feed's 10 items, newest first (brief §4).
var blogPosts = []blogPost{
	{id: 418, slug: "scheduled-rollouts", title: "Scheduled rollouts are here", author: "Joel Weber", category: "Announcements", daysAgo: 0, hour: 10, min: 4,
		excerpt:  "You can now schedule flag changes ahead of time, in any timezone. Pick the moment, the audience and the rollback rule, and the change rolls out on its own while you sleep.",
		seenDays: 0, seenH: 10, seenM: 5, seenVia: "plugin"},
	{id: 416, slug: "a-year-of-on-call", title: "A year of on-call in numbers", author: "Joel Weber", category: "Engineering", tags: []string{"on-call"}, daysAgo: 0, hour: 7, min: 30,
		excerpt: "We were paged 1,412 times in 2025, and a third of those pages came from the same three alerts. Here's what we changed, what we stopped alerting on, and why the Monday morning spike finally went away after we moved the weekly batch jobs to Sunday night.",
		image:   "on-call-2025.png", imageAlt: "Pages per week in 2025, falling from 41 in January to 12 in December",
		seenDays: 0, seenH: 7, seenM: 31, seenVia: "plugin"},
	{id: 413, slug: "testing-database-migrations", title: "How we test database migrations", author: "Mei Tanaka", category: "Infrastructure", tags: []string{"postgres"}, daysAgo: 1, hour: 9, min: 12,
		excerpt: "Every migration runs against a fresh copy of production before it ships. We time each lock, replay the slowest queries and refuse anything that blocks writes for more than 200 ms. Here's the pipeline, and the three bugs it caught in its first month.",
		image:   "migration-pipeline.png", imageAlt: "The migration pipeline: copy production, apply, replay queries, compare lock times",
		seenDays: 1, seenH: 16, seenM: 30, seenVia: "schedule"},
	{id: 409, slug: "ingest-pipeline-rust", title: "Moving our ingest pipeline to Rust", author: "Ana Lima", category: "Engineering", tags: []string{"rust"}, daysAgo: 7, hour: 10, min: 20,
		excerpt: "Our event ingest handled 40,000 events a second in Go and spent a fifth of its time collecting garbage. We rewrote the hot path in Rust over six weeks. Throughput doubled, and the surprises were all in the build pipeline, not in the code.",
		image:   "ingest-throughput.png", imageAlt: "Ingest throughput before and after the rewrite: 40,000 and 84,000 events a second",
		seenDays: 7, seenH: 10, seenM: 21, seenVia: "plugin"},
	{id: 405, slug: "feature-flags-at-2000-teams", title: "Feature flags at 2,000 teams: what broke first", author: "Joel Weber", category: "Engineering", daysAgo: 8, hour: 9, min: 40,
		excerpt: "When the 2,000th team turned on feature flags, rule evaluation went from 2 ms to 90 ms at the 99th percentile. The culprit wasn't the rules engine but a cache we had forgotten about. Here's how we found it, and the dashboard we wish we'd had.",
		image:   "flag-latency.png", imageAlt: "Rule evaluation latency at the 99th percentile, rising from 2 ms to 90 ms",
		seenDays: 8, seenH: 9, seenM: 41, seenVia: "plugin"},
	{id: 402, slug: "postgres-partitioning", title: "Postgres partitioning without downtime", author: "Mei Tanaka", category: "Infrastructure", tags: []string{"postgres"}, daysAgo: 9, hour: 11, min: 5,
		excerpt:  "Our events table passed 3 billion rows in August. We split it into monthly partitions while it kept taking writes, using a trigger, a backfill in small batches and one short lock at the very end of the cut-over.",
		seenDays: 9, seenH: 11, seenM: 6, seenVia: "plugin"},
	{id: 398, slug: "internship-2027", title: "Our 2027 internship program", author: "Priya Raman", category: "Company", daysAgo: 13, hour: 10, min: 0,
		excerpt:  "Applications are open for twelve paid summer internships in engineering, design and developer relations, in Lisbon, Austin or remote.",
		seenDays: 13, seenH: 10, seenM: 1, seenVia: "plugin"},
	{id: 396, slug: "three-services-back-to-postgres", title: "Why we moved three services back to Postgres", author: "Ana Lima", category: "Engineering", tags: []string{"postgres"}, daysAgo: 14, hour: 9, min: 15,
		excerpt: "Two years ago we gave three services their own document stores. This summer we moved all three back to Postgres. Fewer moving parts, one backup story, and queries we can finally join. What we learned along the way, and what we'd do differently.",
		image:   "services-postgres.png", imageAlt: "Three services with their own stores, then one Postgres cluster",
		seenDays: 14, seenH: 9, seenM: 16, seenVia: "plugin"},
	{id: 391, slug: "load-testing-production-traffic", title: "Load-testing with production traffic", author: "Mei Tanaka", category: "Infrastructure", daysAgo: 15, hour: 14, min: 30,
		excerpt:  "We replay a sampled hour of real traffic against every release candidate, at twice its volume. Here's how we scrub it, shard it and read the results.",
		seenDays: 15, seenH: 14, seenM: 31, seenVia: "plugin"},
	{id: 377, slug: "hello-engineering-blog", title: "Hello from the Acme engineering team", author: "Joel Weber", category: "Engineering", daysAgo: 23, hour: 9, min: 0,
		excerpt:  "This is where the people who build Acme write about how it works: the outages, the migrations and the tools we make along the way.",
		seenDays: 20, seenH: 15, seenM: 12, seenVia: "setup"},
}

// blogImageURL is a post image's address on the blog.
func blogImageURL(file string) string {
	return "https://blog.acme.io/wp-content/uploads/2026/10/" + file
}

func (p blogPost) url() string  { return "https://blog.acme.io/" + p.slug + "/" }
func (p blogPost) guid() string { return fmt.Sprintf("https://blog.acme.io/?p=%d", p.id) }

// demoDay is designDay for a moment other than the seed's: hour:minute
// (New York time) daysAgo days before "today" as of now.
func demoDay(now time.Time, daysAgo, hour, minute int) time.Time {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	n := now.In(loc).Add(-11 * time.Hour)
	return time.Date(n.Year(), n.Month(), n.Day(), hour, minute, 0, 0, loc).AddDate(0, 0, -daysAgo)
}

// blogFeed is the blog's RSS 2.0 feed as of now.
func blogFeed(now time.Time) []byte {
	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:media="http://search.yahoo.com/mrss/">` + "\n")
	fmt.Fprintf(&b, "<channel>\n<title>%s</title>\n<link>https://blog.acme.io/</link>\n<description>How Acme is built</description>\n<language>en-US</language>\n<generator>https://wordpress.org/?v=6.9.2</generator>\n",
		html.EscapeString(blogFeedTitle))
	for _, p := range blogPosts {
		at := demoDay(now, p.daysAgo, p.hour, p.min)
		if at.After(now) {
			at = now.Add(-time.Minute)
		}
		fmt.Fprintf(&b, "<item>\n<title>%s</title>\n<link>%s</link>\n<dc:creator><![CDATA[%s]]></dc:creator>\n<pubDate>%s</pubDate>\n<category domain=\"category\"><![CDATA[%s]]></category>\n",
			html.EscapeString(p.title), p.url(), p.author, at.UTC().Format(time.RFC1123Z), p.category)
		for _, t := range p.tags {
			fmt.Fprintf(&b, "<category domain=\"post_tag\"><![CDATA[%s]]></category>\n", t)
		}
		fmt.Fprintf(&b, "<guid isPermaLink=\"false\">%s</guid>\n<description><![CDATA[%s]]></description>\n", p.guid(), p.excerpt)
		if p.image != "" {
			fmt.Fprintf(&b, "<media:content url=\"%s\" medium=\"image\" width=\"1200\" height=\"522\"><media:description>%s</media:description></media:content>\n",
				blogImageURL(p.image), html.EscapeString(p.imageAlt))
		}
		b.WriteString("</item>\n")
	}
	b.WriteString("</channel>\n</rss>\n")
	return b.Bytes()
}

// blogHome is the blog's home page: it names its feed in its head.
const blogHome = `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Acme engineering blog</title>
<link rel="alternate" type="application/rss+xml" title="Acme engineering blog &raquo; Feed" href="https://blog.acme.io/feed/">
<link rel="alternate" type="application/rss+xml" title="Acme engineering blog &raquo; Comments Feed" href="https://blog.acme.io/comments/feed/">
</head><body><h1>Acme engineering blog</h1></body></html>`

// DigestFeedOverlay answers the sample blog's feed, pages and images
// offline; everything else goes to next. A conditional read of the feed
// the demo seeded gets 304, so the seeded items keep their dates.
func DigestFeedOverlay(next http.RoundTripper) http.RoundTripper {
	return blogTransport{next: next, now: time.Now}
}

type blogTransport struct {
	next http.RoundTripper
	now  func() time.Time
}

// RoundTrip implements http.RoundTripper.
func (t blogTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if !strings.EqualFold(r.URL.Hostname(), "blog.acme.io") {
		return t.next.RoundTrip(r)
	}
	switch path := r.URL.Path; {
	case path == "/feed/" || path == "/feed":
		if r.Header.Get("If-None-Match") == BlogFeedETag {
			return blogResponse(r, http.StatusNotModified, "", nil), nil
		}
		return blogResponse(r, http.StatusOK, "application/rss+xml; charset=UTF-8", blogFeed(t.now())), nil
	case path == "/" || path == "":
		return blogResponse(r, http.StatusOK, "text/html; charset=UTF-8", []byte(blogHome)), nil
	case strings.HasPrefix(path, "/wp-content/uploads/"):
		name := path[strings.LastIndex(path, "/")+1:]
		if data, err := digestAssets.ReadFile("assets/digests/" + name); err == nil {
			return blogResponse(r, http.StatusOK, "image/png", data), nil
		}
	}
	return blogResponse(r, http.StatusNotFound, "text/html; charset=UTF-8", []byte("<h1>Not found</h1>")), nil
}

func blogResponse(r *http.Request, status int, ctype string, body []byte) *http.Response {
	h := http.Header{}
	if ctype != "" {
		h.Set("Content-Type", ctype)
	}
	if strings.HasPrefix(r.URL.Path, "/feed") && status == http.StatusOK {
		h.Set("ETag", BlogFeedETag)
	}
	return &http.Response{StatusCode: status, Status: fmt.Sprintf("%d %s", status, http.StatusText(status)), Header: h, Body: io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)), Request: r, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1}
}
