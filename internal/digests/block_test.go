package digests

import (
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/templates"
)

func sampleItems() []BlockItem {
	at := func(d, h, m int) *time.Time {
		t := time.Date(2026, 10, d, h, m, 0, 0, time.UTC)
		return &t
	}
	return []BlockItem{
		{ID: "a", Title: "A year of on-call in numbers", URL: "https://blog.acme.io/on-call-in-numbers/", Author: "Joel Weber", Published: at(15, 11, 30),
			Categories: []string{"Engineering"}, Excerpt: "We were paged 1,412 times in 2025, and a third of those pages came from the same three alerts. Here's what we changed, what we stopped alerting on, and why the Monday morning spike finally went away after we moved our checks.",
			Image: "/media/0192/on-call-2025.png", ImageAlt: "Heat map of on-call pages"},
		{ID: "b", Title: "How we test database migrations & locks", URL: "https://blog.acme.io/testing-migrations/", Author: "Mei Tanaka", Published: at(14, 20, 30),
			Categories: []string{"Infrastructure"}, Excerpt: "Every migration runs against a fresh copy of production before it ships.", Image: "/media/0193/migration-pipeline.png"},
		{ID: "c", Title: "Postgres partitioning without downtime", URL: "https://blog.acme.io/partitioning/", Author: "Mei Tanaka", Published: at(6, 13, 0),
			Excerpt: "Our events table passed 3 billion rows in August."},
	}
}

func TestRenderBlockLayouts(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	items := sampleItems()
	b := DefaultBlock()
	b.Fallback, b.FallbackName = "/media/0199/blog-fallback.png", "blog-fallback.png"
	cards := RenderBlock(b, items, ny, i18n.Lang{})
	if n := strings.Count(cards, `css-class="rc-feed-item"`); n != 3 {
		t.Fatalf("cards: %d items", n)
	}
	for _, want := range []string{
		`<mj-image src="/media/0192/on-call-2025.png" alt="Heat map of on-call pages" width="552px" href="https://blog.acme.io/on-call-in-numbers/"`,
		// Alt text from the title when the image has none; the title is escaped.
		`alt="How we test database migrations &amp; locks"`,
		// A post without an image shows the fallback, with its title as alt text.
		`<mj-image src="/media/0199/blog-fallback.png" alt="Postgres partitioning without downtime"`,
		"Thursday 15 October 2026 · Joel Weber",
		`>Read the post</mj-button>`,
	} {
		if !strings.Contains(cards, want) {
			t.Fatalf("cards miss %q:\n%s", want, cards)
		}
	}
	// The excerpt is cut at 40 words.
	if !strings.Contains(cards, "after we moved…") && !strings.Contains(cards, "spike finally went away after we…") {
		if strings.Count(cards, "…") == 0 {
			t.Fatalf("excerpt not cut: %s", cards)
		}
	}
	b.Layout = "list"
	list := RenderBlock(b, items, ny, i18n.Lang{})
	if strings.Contains(list, "<mj-image") || strings.Contains(list, "<mj-button") || !strings.Contains(list, "text-decoration:underline") {
		t.Fatalf("list: %s", list)
	}
	b.Layout = "grid"
	grid := RenderBlock(b, items, ny, i18n.Lang{})
	if strings.Count(grid, `css-class="rc-feed-item"`) != 2 || strings.Count(grid, `width="50%"`) != 3 || !strings.Contains(grid, `width="264px"`) || !strings.Contains(grid, "Wed 14 Oct · Mei Tanaka") {
		t.Fatalf("grid: %s", grid)
	}
	// Fields left out stay out.
	b.Layout, b.Fields = "cards", []string{FieldTitle}
	bare := RenderBlock(b, items[:1], ny, i18n.Lang{})
	if strings.Contains(bare, "<mj-image") || strings.Contains(bare, "Joel Weber") || strings.Contains(bare, "mj-button") {
		t.Fatalf("title only: %s", bare)
	}
}

func TestSelect(t *testing.T) {
	items := sampleItems()
	b := DefaultBlock()
	b.Max = 2
	shown, rest := b.Select(items)
	if len(shown) != 2 || shown[0].ID != "a" || shown[1].ID != "b" || len(rest) != 1 || rest[0].ID != "c" {
		t.Fatalf("drop: %v %v", ids(shown), ids(rest))
	}
	// Keep the rest for the next issue: the oldest go first.
	b.Overflow = "keep"
	shown, rest = b.Select(items)
	if ids(shown) != "b,c" || ids(rest) != "a" {
		t.Fatalf("keep: %v %v", ids(shown), ids(rest))
	}
	b.Overflow, b.Order = "drop", "oldest"
	shown, _ = b.Select(items)
	if ids(shown) != "b,a" {
		t.Fatalf("oldest first: %v", ids(shown))
	}
}

func ids(items []BlockItem) string {
	out := []string{}
	for _, it := range items {
		out = append(out, it.ID)
	}
	return strings.Join(out, ",")
}

func TestStarterExpandsAndCompiles(t *testing.T) {
	src, ok := templates.StarterMJML(templates.BlogDigestStarter, "Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105")
	if !ok {
		t.Fatal("no blog digest starter")
	}
	b, found := BlockOf(src)
	if !found || b.Layout != "cards" || b.Max != 6 || b.ButtonText != "Read the post" {
		t.Fatalf("block: %+v %v", b, found)
	}
	// The template compiles as it is: the block is an empty Raw HTML element.
	if res := mjml.Compile(src); len(res.Errors) > 0 {
		t.Fatalf("template: %v", res.Errors)
	}
	issue := ExpandBlock(src, sampleItems(), time.UTC, i18n.Lang{})
	if strings.Contains(issue, "rc:feed-items") || !strings.Contains(issue, `rc:feed-issue {"layout":"cards","items":3}`) {
		t.Fatalf("expand: %s", issue)
	}
	res := mjml.Compile(issue)
	if len(res.Errors) > 0 {
		t.Fatalf("issue: %v", res.Errors)
	}
	for _, want := range []string{"A year of on-call in numbers", "https://blog.acme.io/testing-migrations/", "Read the post"} {
		if !strings.Contains(res.HTML, want) {
			t.Fatalf("compiled issue misses %q", want)
		}
	}
}

func TestMarkerRoundTrip(t *testing.T) {
	b := DefaultBlock()
	b.ButtonText = "Read -- now"
	m := MarkerFor(b)
	if strings.Contains(strings.TrimSuffix(strings.TrimPrefix(m, "<mj-raw><!--"), "--></mj-raw>"), "--") {
		t.Fatalf("marker keeps --: %s", m)
	}
	got, ok := BlockOf("<mjml><mj-body>" + m + "</mj-body></mjml>")
	if !ok || got.ButtonText != "Read -- now" {
		t.Fatalf("round trip: %+v %v", got, ok)
	}
}
