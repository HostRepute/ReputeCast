package importer_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"flag"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/website/importer"
	"github.com/hostrepute/reputecast/internal/website/render"
)

var update = flag.Bool("update", false, "rewrite the golden MJML files")

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func at(s string) *time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return &t
}

func bySlug(t *testing.T, posts []importer.Post, slug string) importer.Post {
	t.Helper()
	for _, p := range posts {
		if p.Slug == slug {
			return p
		}
	}
	t.Fatalf("no post %q in %d posts", slug, len(posts))
	return importer.Post{}
}

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// The format comes from the content, whatever the file is called.
func TestDetect(t *testing.T) {
	ghost := fixture(t, "ghost.json")
	for name, c := range map[string]struct {
		data []byte
		want string
	}{
		"substack zip":            {fixture(t, "substack.zip"), importer.Substack},
		"ghost json":              {ghost, importer.Ghost},
		"ghost json with a BOM":   {append([]byte("\xef\xbb\xbf\n"), ghost...), importer.Ghost},
		"ghost zip":               {zipOf(t, map[string]string{"acme-weekly.ghost.2026-11-10.json": string(ghost), "content/images/a.png": "x"}), importer.Ghost},
		"ghost zip in a folder":   {zipOf(t, map[string]string{"export/acme.json": string(ghost)}), importer.Ghost},
		"substack in a folder":    {zipOf(t, map[string]string{"export/posts.csv": "post_id\n", "export/posts/1.a.html": "<p>a</p>"}), importer.Substack},
		"wordpress":               {fixture(t, "wordpress.xml"), importer.WordPress},
		"old ghost (data at top)": {[]byte(`{"meta":{"version":"1.25.0"},"data":{"posts":[]}}`), importer.Ghost},
	} {
		if got, err := importer.Detect(c.data); err != nil || got != c.want {
			t.Errorf("%s: %q %v, want %q", name, got, err, c.want)
		}
	}
	for name, data := range map[string][]byte{
		"json":            []byte(`{"posts":[{"title":"x"}],"meta":{}}`),
		"json with db":    []byte(`{"db":[{"data":{"members":[]}}]}`),
		"csv":             []byte("email,name\nsam@example.com,Sam\n"),
		"rss":             []byte(`<?xml version="1.0"?><rss version="2.0"><channel><title>x</title></channel></rss>`),
		"zip of pictures": zipOf(t, map[string]string{"a.png": "x", "notes.json": `{"a":1}`}),
		"empty":           {},
		"broken zip":      []byte("PK\x03\x04broken"),
	} {
		if got, err := importer.Detect(data); !errors.Is(err, importer.ErrNotExport) {
			t.Errorf("%s: %q %v, want ErrNotExport", name, got, err)
		}
	}
}

// A Substack export: posts.csv's rows with their HTML; threads and pages
// aren't posts; audiences give the visibility and the tier.
func TestParseSubstack(t *testing.T) {
	posts, err := importer.ParseSubstack(fixture(t, "substack.zip"))
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 5 {
		t.Fatalf("%d posts, want 5 (no thread, no page)", len(posts))
	}
	p := bySlug(t, posts, "why-we-moved-our-queue-to-postgres")
	if p.SourceID != "150120001.why-we-moved-our-queue-to-postgres" || p.Title != "Why we moved our queue to Postgres" || p.Draft ||
		p.Visibility != importer.Public || len(p.TierRefs) != 0 || !p.PublishedAt.Equal(*at("2026-03-12T16:00:00Z")) ||
		p.Excerpt != "A job and the row it updates now commit together" || !slices.Equal(p.LegacyPaths, []string{"p/why-we-moved-our-queue-to-postgres"}) ||
		p.SourceURL != "/p/why-we-moved-our-queue-to-postgres" || p.Author != "" || len(p.Tags) != 0 || !strings.Contains(p.HTML, "youtube-wrap") {
		t.Fatalf("public post: %+v", p)
	}
	if p := bySlug(t, posts, "the-2025-deliverability-report"); p.Visibility != importer.Paid || !slices.Equal(p.TierRefs, []string{"Paid subscribers"}) ||
		p.Excerpt != "“Authentication stopped being optional”" {
		t.Fatalf("paid post: %s %v %q", p.Visibility, p.TierRefs, p.Excerpt)
	}
	if p := bySlug(t, posts, "founders-letter-year-two"); p.Visibility != importer.Paid || !slices.Equal(p.TierRefs, []string{"Founding members"}) ||
		p.Title != "Founders’ letter: year two" {
		t.Fatalf("founding post: %s %v %q", p.Visibility, p.TierRefs, p.Title)
	}
	if p := bySlug(t, posts, "ask-acme-12-warming-up-a-new-domain"); p.Visibility != importer.Members {
		t.Fatalf("free subscribers' post: %s", p.Visibility)
	}
	if p := bySlug(t, posts, "q1-planning-notes"); !p.Draft {
		t.Fatal("an unpublished post is a draft")
	}
	if _, err := importer.ParseSubstack(fixture(t, "ghost.json")); !errors.Is(err, importer.ErrNotExport) {
		t.Fatalf("not a zip: %v", err)
	}
}

// A Ghost export: posts with their tags (internal ones left out), first
// author, tiers, SEO and cover; pages aren't posts; the site's own
// address becomes root-relative.
func TestParseGhost(t *testing.T) {
	posts, err := importer.ParseGhost(fixture(t, "ghost.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 5 {
		t.Fatalf("%d posts, want 5 (no page)", len(posts))
	}
	p := bySlug(t, posts, "my-old-slug")
	want := importer.Post{SourceID: "65f1c0a2b7e4d90001a3c401", Title: "Shipping feature flags without fear", Slug: "my-old-slug",
		PublishedAt: at("2025-09-09T14:00:00Z"), Visibility: importer.Public, TierRefs: []string{}, Tags: []string{"Engineering"}, Author: "Jordan Ellis",
		Excerpt: "How we roll a change out to 1% of accounts, watch it, and widen it.", CoverURL: "https://images.unsplash.com/photo-1518770660439-4636190af475?w=2000",
		CoverAlt: "A switchboard of feature flags", SEOTitle: "Feature flags without fear · Acme Weekly", SEODescription: "How Acme rolls out a change: 1%, then 10%, then everyone.",
		SourceURL: "/my-old-slug/", LegacyPaths: []string{"my-old-slug"}, HTML: p.HTML}
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("post:\n got %+v\nwant %+v", p, want)
	}
	if strings.Contains(p.HTML, "__GHOST_URL__") || !strings.Contains(p.HTML, `src="/content/images/2025/09/rollout.png"`) || !strings.Contains(p.HTML, `href="/rate-limits-explained/"`) {
		t.Fatalf("the site's address stays in the HTML: %s", p.HTML)
	}
	if p := bySlug(t, posts, "the-2025-deliverability-report"); p.Visibility != importer.Paid || len(p.TierRefs) != 0 || p.Author != "Sofia Rossi" {
		t.Fatalf("paid post: %s %v %q", p.Visibility, p.TierRefs, p.Author)
	}
	if p := bySlug(t, posts, "founders-letter-year-two"); p.Visibility != importer.Paid || !slices.Equal(p.TierRefs, []string{"Founding member"}) {
		t.Fatalf("a tier's post: %s %v", p.Visibility, p.TierRefs)
	}
	if p := bySlug(t, posts, "ask-acme-12-warming-up-a-new-domain"); p.Visibility != importer.Members || !slices.Equal(p.Tags, []string{"Ask Acme"}) {
		t.Fatalf("members' post: %s %v", p.Visibility, p.Tags)
	}
	if p := bySlug(t, posts, "q1-planning-notes"); !p.Draft || p.PublishedAt != nil {
		t.Fatalf("draft: %+v", p)
	}
	// In a zip, with the site's images: they can be copied from it.
	z := zipOf(t, map[string]string{"acme.ghost.json": string(fixture(t, "ghost.json")), "content/images/2025/09/rollout.png": "png bytes"})
	zposts, err := importer.ParseGhost(z)
	if err != nil || len(zposts) != 5 {
		t.Fatalf("zip: %d posts %v", len(zposts), err)
	}
	if b, ok := importer.File(z, "/content/images/2025/09/rollout.png"); !ok || string(b) != "png bytes" {
		t.Fatalf("the zip's image: %q %v", b, ok)
	}
	if _, ok := importer.File(z, "/content/images/../../acme.ghost.json"); ok {
		t.Fatal("a path out of the images was read")
	}
	if _, ok := importer.File(fixture(t, "ghost.json"), "/content/images/2025/09/rollout.png"); ok {
		t.Fatal("a JSON export holds no images")
	}
}

// A WordPress export: published posts, drafts; private, password-protected
// posts, pages and menus stay out; categories and tags as tags (not
// Uncategorized); the cover from _thumbnail_id; the old addresses.
func TestParseWXR(t *testing.T) {
	posts, err := importer.ParseWXR(fixture(t, "wordpress.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 3 {
		t.Fatalf("%d posts, want 3", len(posts))
	}
	p := bySlug(t, posts, "why-we-moved-our-queue-to-postgres")
	if p.SourceID != "42" || p.Author != "Sofia Rossi" || !slices.Equal(p.Tags, []string{"Engineering", "Postgres"}) || p.Draft ||
		!p.PublishedAt.Equal(*at("2026-03-12T16:00:00Z")) || p.CoverURL != "https://blog.acme.io/wp-content/uploads/2026/03/queue-dashboard.png" ||
		p.CoverAlt != "The queue dashboard after the move" || p.Excerpt != "A job and the row it updates now commit together." ||
		p.SourceURL != "https://blog.acme.io/2026/03/12/why-we-moved-our-queue-to-postgres/" ||
		!slices.Equal(p.LegacyPaths, []string{"2026/03/12/why-we-moved-our-queue-to-postgres", "?p=42"}) {
		t.Fatalf("classic post: %+v", p)
	}
	if !strings.Contains(p.HTML, "<p>The reason is simple: a job and the row it updates now commit together.<br>") || !strings.Contains(p.HTML, "<figcaption>Jobs per second, the week of the move.</figcaption>") ||
		!strings.Contains(p.HTML, `wp-block-embed__wrapper">https://www.youtube.com/watch?v=dQw4w9WgXcQ<`) || !strings.Contains(p.HTML, "<!--rc:shortcode gallery-->") {
		t.Fatalf("classic content as WordPress shows it:\n%s", p.HTML)
	}
	if b := bySlug(t, posts, "inside-a-550-5-7-1-bounce"); b.Author != "Jordan Ellis" || len(b.Tags) != 0 || b.CoverURL != "" {
		t.Fatalf("block post: %+v", b)
	}
	var draft importer.Post
	for _, p := range posts {
		if p.SourceID == "51" {
			draft = p
		}
	}
	if !draft.Draft || draft.PublishedAt != nil || draft.Slug != "" || draft.Title != "Spring launch checklist" {
		t.Fatalf("draft: %+v", draft)
	}
	if k := importer.LegacyKey("/My-Old-Slug/", ""); k != "my-old-slug" {
		t.Fatalf("legacy key: %q", k)
	}
	if k := importer.LegacyKey("/", "p=42&utm_source=x"); k != "?p=42" {
		t.Fatalf("?p= key: %q", k)
	}
	if k := importer.LegacyKey("/", "p=abc"); k != "" {
		t.Fatalf("a p that isn't an id: %q", k)
	}
}

// Golden MJML: text grouped in mj-text, images as mj-image (rewritten,
// alt kept, an empty alt reported), embeds as links (reported), the paywall
// marker between blocks, Liquid written as entities, the old platform's
// signup blocks left out. Every result parses and renders.
func TestConvertGolden(t *testing.T) {
	sources := map[string]func([]byte) ([]importer.Post, error){importer.Substack: importer.ParseSubstack, importer.Ghost: importer.ParseGhost, importer.WordPress: importer.ParseWXR}
	files := map[string]string{importer.Substack: "substack.zip", importer.Ghost: "ghost.json", importer.WordPress: "wordpress.xml"}
	rewrite := func(src string) (string, bool) {
		if strings.Contains(src, "rollout.png") {
			// An image that couldn't be copied keeps its address.
			return "", false
		}
		return "/media/0193d7a2-0000-7000-8000-000000000000/" + path.Base(strings.SplitN(src, "?", 2)[0]), true
	}
	for _, c := range []struct {
		source, slug string
		problems     []importer.Problem
		marker       bool
		has          []string
		hasNot       []string
	}{
		{importer.Substack, "why-we-moved-our-queue-to-postgres", []importer.Problem{{Code: importer.ProblemEmbed, Detail: "YouTube"}}, false,
			[]string{"Watch on YouTube:", "Read the changelog", "&#123;&#123; first_name }}", "&#123;% if paid %}"}, []string{"Subscribe now", "Thanks for reading Acme Weekly"}},
		{importer.Substack, "the-2025-deliverability-report", []importer.Problem{{Code: importer.ProblemEmbed, Detail: "X"}}, true,
			[]string{"View on X:", "https://twitter.com/acme/status/1730000000000000000"}, []string{"Twitter avatar"}},
		{importer.Substack, "ask-acme-12-warming-up-a-new-domain", []importer.Problem{{Code: importer.ProblemAlt,
			Detail: "https://substackcdn.com/image/fetch/w_1456,c_limit,f_auto/https%3A%2F%2Fsubstack-post-media.s3.amazonaws.com%2Fpublic%2Fimages%2Fwarmup-curve.webp"}}, false,
			[]string{`alt=""`}, nil},
		{importer.Substack, "founders-letter-year-two", []importer.Problem{{Code: importer.ProblemEmbed, Detail: "Substack"}}, true,
			[]string{"Read on Substack:"}, nil},
		{importer.Ghost, "my-old-slug", []importer.Problem{{Code: importer.ProblemImage, Detail: "/content/images/2025/09/rollout.png"}, {Code: importer.ProblemEmbed, Detail: "YouTube"}}, false,
			[]string{`src="/content/images/2025/09/rollout.png"`, "https://www.youtube.com/watch?v=hmH3XMlms8E", `href="/rate-limits-explained/"`, "&#123;&#123; first_name }}", "Four days from 1% to everyone."}, nil},
		{importer.Ghost, "the-2025-deliverability-report", []importer.Problem{{Code: importer.ProblemEmbed, Detail: "dmarc.org"}}, true,
			[]string{"View on dmarc.org:"}, []string{"kg-bookmark"}},
		{importer.WordPress, "why-we-moved-our-queue-to-postgres", []importer.Problem{{Code: importer.ProblemEmbed, Detail: "YouTube"}, {Code: importer.ProblemShortcode, Detail: "gallery"}}, false,
			[]string{`<mj-image src="/media/0193d7a2-0000-7000-8000-000000000000/queue-dashboard.png" alt="The queue dashboard after the move"`, "&#123;&#123; reply_to }}", "Jobs per second, the week of the move."}, []string{"[gallery"}},
		{importer.WordPress, "inside-a-550-5-7-1-bounce", []importer.Problem{{Code: importer.ProblemAlt, Detail: "https://blog.acme.io/wp-content/uploads/2025/03/bounce.png"}, {Code: importer.ProblemEmbed, Detail: "Vimeo"}}, false,
			[]string{"<mj-divider></mj-divider>", "Watch on Vimeo:", "https://vimeo.com/76979871"}, []string{"wp:paragraph"}},
	} {
		posts, err := sources[c.source](fixture(t, files[c.source]))
		if err != nil {
			t.Fatal(err)
		}
		p := bySlug(t, posts, c.slug)
		out, problems := importer.Convert(p.HTML, rewrite)
		name := c.source + "-" + c.slug
		golden := filepath.Join("testdata", "golden", name+".mjml")
		if *update {
			if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(golden, []byte(out), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatalf("%s: %v (run with -update)", name, err)
		}
		if out != string(want) {
			t.Errorf("%s: MJML differs from %s:\n%s", name, golden, out)
		}
		if !reflect.DeepEqual(problems, c.problems) {
			t.Errorf("%s: problems %+v, want %+v", name, problems, c.problems)
		}
		for _, s := range c.has {
			if !strings.Contains(out, s) {
				t.Errorf("%s: no %q in\n%s", name, s, out)
			}
		}
		for _, s := range c.hasNot {
			if strings.Contains(out, s) {
				t.Errorf("%s: %q left in\n%s", name, s, out)
			}
		}
		if _, err := mjml.Parse(out); err != nil {
			t.Errorf("%s: mjml.Parse: %v", name, err)
		}
		res, err := render.Render(out, render.Options{Title: p.Title})
		if err != nil {
			t.Errorf("%s: render: %v", name, err)
			continue
		}
		if res.HasMarker != c.marker {
			t.Errorf("%s: marker %v, want %v", name, res.HasMarker, c.marker)
		}
		if len(res.Fallbacks) != 0 {
			t.Errorf("%s: Liquid ran in imported text: %+v", name, res.Fallbacks)
		}
		if strings.Contains(out, "{{") || strings.Contains(out, "{%") {
			t.Errorf("%s: Liquid delimiters left in the MJML", name)
		}
	}
}

// The embed links read in the site's language; Images lists what Convert
// rewrites; Summary is the first paragraph.
func TestConvertDetails(t *testing.T) {
	src := `<p>Intro</p><iframe src="https://open.spotify.com/embed/episode/7makk4oTQel546B0PZlDM5"></iframe><p><img src="//cdn.example.com/a.png" alt="A"> inline</p><img src="data:image/png;base64,AAAA" alt="x"><img src="https://cdn.example.com/a.png" alt="A">`
	out, problems := importer.ConvertIn(i18n.For(i18n.Vietnamese), src, nil)
	if !strings.Contains(out, "Nghe trên Spotify: <a href=\"https://open.spotify.com/episode/7makk4oTQel546B0PZlDM5\">") {
		t.Fatalf("Vietnamese embed link:\n%s", out)
	}
	if !slices.Contains(problems, importer.Problem{Code: importer.ProblemImage, Detail: "data:image/png;base64,AAAA"}) {
		t.Fatalf("an image the site can't serve is reported: %+v", problems)
	}
	if imgs := importer.Images(src); !slices.Equal(imgs, []string{"https://cdn.example.com/a.png", "data:image/png;base64,AAAA"}) {
		t.Fatalf("images: %v", imgs)
	}
	if s := importer.Summary(`<div class="subscription-widget-wrap"><p>Subscribe!</p></div><p></p><p>First <b>words</b> here.</p><p>Second.</p>`); s != "First words here." {
		t.Fatalf("summary: %q", s)
	}
}
