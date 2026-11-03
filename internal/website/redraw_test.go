package website_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// A post published from an email with a poll before a poll stayed in the
// inbox kept the poll's answers, as links to /0, /1 … on the site. The
// redraw takes them out, from the content the post was published with, and
// leaves its version, hash and publication as they were; a second run finds
// nothing to do.
func TestRedrawTakesThePollOffPublishedPosts(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	const slug = "the-case-for-boring-infrastructure"
	var mjml string
	if err := env.App.Pool.QueryRow(ctx, `SELECT published_mjml FROM website_posts WHERE slug = $1 AND status = 'published'`, slug).Scan(&mjml); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(mjml, "rc-poll") {
		t.Fatal("the demo's post has no poll to test with")
	}
	if body := page(t, env.Client(), "/posts/"+slug, 200); strings.Contains(body, `href="/0"`) {
		t.Fatal("a post published now shows the poll's answers")
	}

	// The post as an earlier release drew it.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE website_posts SET body_html = body_html || '<p><a href="/0">0</a><a href="/10">10</a></p>',
		render_notes = jsonb_set(render_notes, '{hidden}', '[]') WHERE slug = $1`, slug); err != nil {
		t.Fatal(err)
	}
	var version int32
	var hash string
	if err := env.App.Pool.QueryRow(ctx, `SELECT published_version, rendered_hash FROM website_posts WHERE slug = $1`, slug).Scan(&version, &hash); err != nil {
		t.Fatal(err)
	}
	if body := page(t, env.Client(), "/posts/"+slug, 200); !strings.Contains(body, `href="/0"`) {
		t.Fatal("the old drawing didn't take")
	}

	n, err := env.App.Website().RedrawPollPosts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("redrew %d posts, want the one with the poll", n)
	}
	if body := page(t, env.Client(), "/posts/"+slug, 200); strings.Contains(body, `href="/0"`) || !strings.Contains(body, "From the changelog") {
		t.Fatalf("after the redraw the post shows the poll's answers or lost its content:\n%.2000s", body)
	}
	var version2 int32
	var hash2 string
	var polls int
	if err := env.App.Pool.QueryRow(ctx, `SELECT published_version, rendered_hash,
		(SELECT count(*) FROM jsonb_array_elements(render_notes->'hidden') h WHERE h->>'kind' = 'poll')
		FROM website_posts WHERE slug = $1`, slug).Scan(&version2, &hash2, &polls); err != nil {
		t.Fatal(err)
	}
	if version2 != version || hash2 != hash || polls != 1 {
		t.Fatalf("after the redraw: version %d (was %d), hash changed %v, poll notes %d", version2, version, hash2 != hash, polls)
	}
	if n, err := env.App.Website().RedrawPollPosts(ctx); err != nil || n != 0 {
		t.Fatalf("a second redraw drew %d posts (%v), want none", n, err)
	}
}
