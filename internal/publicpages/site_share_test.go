package publicpages_test

import (
	"context"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/website"
)

// newsHost is the demo's verified custom domain.
var newsHost = hdr{"Host": "news.acme.io"}

// saveShareButtons saves Home & navigation as it is, with "Share buttons
// on posts" set to on, as c.
func saveShareButtons(t *testing.T, c *apptest.Client, on bool, status int) {
	t.Helper()
	var saved website.WebsiteHomeNav
	c.Do(http.MethodGet, wsAPI+"/website/home-nav", nil).MustStatus(t, 200).JSON(t, &saved)
	in := website.WebsiteHomeNavInput{Layout: saved.Layout, Featured: []string{}, Sections: saved.Sections, About: saved.About, Header: []website.WebsiteNavLink{},
		Footer: saved.Footer, PrivacyURL: saved.PrivacyURL, PoweredBy: saved.PoweredBy, Identity: saved.Identity, Index: saved.Index, ShareButtons: &on}
	for _, f := range saved.Featured {
		in.Featured = append(in.Featured, f.ID)
	}
	for _, h := range saved.Header {
		in.Header = append(in.Header, h.WebsiteNavLink)
	}
	c.Do(http.MethodPut, wsAPI+"/website/home-nav", in).MustStatus(t, status)
}

// A published post ends with its share row: the address with Copy link
// (hidden until site.js runs) and plain share links to the canonical
// address; "Share buttons on posts" off takes it away.
func TestPostShareButtons(t *testing.T) {
	env, ids := demo(t)
	const slug = "the-case-for-boring-infrastructure"
	canon := "http://news.acme.io/posts/" + slug
	path := "/posts/" + slug
	r := do(t, env, http.MethodGet, path, nil, newsHost)
	must(t, r, 200, "https://x.com/intent/post?text=The+case+for+boring+infrastructure&url="+url.QueryEscape(canon), `data-copy="share-url"`,
		`value="`+canon+`" readonly`, `aria-labelledby="share-h"`, "https://www.facebook.com/sharer/sharer.php?u="+url.QueryEscape(canon), "https://bsky.app/intent/compose?text=")
	if !regexp.MustCompile(`data-copy="share-url"[^>]* hidden>`).MatchString(r.Body) || strings.Contains(r.Body, "<script>") {
		t.Error("Copy link shows without the script, or the page runs an inline script")
	}
	// Share links open a new tab and say so; the email link opens the mail app in place.
	if n := strings.Count(r.Body, `target="_blank" rel="noopener noreferrer"`); n < 5 {
		t.Errorf("%d share links open a new tab, want 5", n)
	}
	if !strings.Contains(r.Text(), `<a class="st-btn2" href="mailto:?subject=The%20case%20for%20boring%20infrastructure&body=`) {
		t.Errorf("email link:\n%s", clip(r.Body[strings.Index(r.Body, `class="st-share"`):]))
	}
	if csp := r.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'self'") || strings.Contains(csp, "unsafe-inline") {
		t.Errorf("CSP %s", csp)
	}

	// A title can't add a header to the reader's email.
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE website_posts SET title = 'x&bcc=a@evil.io' WHERE workspace_id = $1 AND slug = $2`, ids.ws, slug); err != nil {
		t.Fatal(err)
	}
	r = do(t, env, http.MethodGet, path, nil, newsHost)
	must(t, r, 200, "mailto:?subject=x%26bcc%3Da%40evil.io&body=x%26bcc%3Da%40evil.io%20")
	m := regexp.MustCompile(`href="(mailto:[^"]*)"`).FindStringSubmatch(r.Body)
	if m == nil {
		t.Fatal("no email share link")
	}
	mail, err := url.Parse(html.UnescapeString(m[1]))
	if err != nil {
		t.Fatal(err)
	}
	if q := mail.Query(); len(q) != 2 || q.Get("subject") != "x&bcc=a@evil.io" || q.Get("body") != "x&bcc=a@evil.io "+canon {
		t.Errorf("email link %s: %v", m[1], q)
	}

	// A Viewer can't turn them off; an Owner can, and it's audited.
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	saveShareButtons(t, viewer, false, http.StatusForbidden)
	saveShareButtons(t, owner(env), false, http.StatusOK)
	r = do(t, env, http.MethodGet, path, nil, newsHost)
	must(t, r, 200)
	lacks(t, r, "https://x.com/intent/post?", `data-copy="share-url"`)
	var before, after string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT before->>'share_buttons', after->>'share_buttons' FROM audit_events
		WHERE workspace_id = $1 AND action = 'website.navigation_updated' ORDER BY occurred_at DESC LIMIT 1`, ids.ws).Scan(&before, &after); err != nil || before != "true" || after != "false" {
		t.Errorf("audit share_buttons %q → %q (%v)", before, after, err)
	}
	var nav website.WebsiteHomeNav
	owner(env).Do(http.MethodGet, wsAPI+"/website/home-nav", nil).MustStatus(t, 200).JSON(t, &nav)
	if nav.ShareButtons || nav.URLs.LLMs != "news.acme.io/llms.txt" {
		t.Errorf("home-nav share %v, llms %q", nav.ShareButtons, nav.URLs.LLMs)
	}
}

// /llms.txt lists what the sitemap does, newest first, in Markdown whose
// every value is escaped; it follows "Let search engines index the site".
func TestLLMsTxt(t *testing.T) {
	env, ids := demo(t)
	ctx := context.Background()
	if _, err := env.App.Pool.Exec(ctx, `UPDATE website_posts SET hide_from_search = true WHERE workspace_id = $1 AND slug = 'rate-limits-best-customers'`, ids.ws); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE website_posts SET title = $2 WHERE workspace_id = $1 AND slug = 'postgres-migration-runbook'`, ids.ws,
		"Evil](https://x.example)\n# Fake"); err != nil {
		t.Fatal(err)
	}
	r := do(t, env, http.MethodGet, "/llms.txt", nil, newsHost)
	if r.Status != 200 || r.Header.Get("Content-Type") != "text/markdown; charset=utf-8" || r.Header.Get("X-Content-Type-Options") != "nosniff" ||
		!strings.Contains(r.Header.Get("Cache-Control"), "public") {
		t.Fatalf("llms.txt: %d %v\n%s", r.Status, r.Header, clip(r.Body))
	}
	var nav website.WebsiteHomeNav
	owner(env).Do(http.MethodGet, wsAPI+"/website/home-nav", nil).MustStatus(t, 200).JSON(t, &nav)
	if nav.Identity.Title == "" || !strings.HasPrefix(r.Body, "# "+nav.Identity.Title+"\n\n") {
		t.Errorf("heading %q:\n%s", nav.Identity.Title, clip(r.Body))
	}
	for _, want := range []string{"## Posts\n", "](http://news.acme.io/posts/the-case-for-boring-infrastructure)",
		"- [Evil\\]\\(https://x.example\\) \\# Fake](http://news.acme.io/posts/postgres-migration-runbook): How we moved",
		"## Optional\n", "- [RSS feed](http://news.acme.io/feed)", "- [Sitemap](http://news.acme.io/sitemap.xml)"} {
		if !strings.Contains(r.Body, want) {
			t.Errorf("llms.txt lacks %q:\n%s", want, clip(r.Body))
		}
	}
	for _, line := range strings.Split(r.Body, "\n") {
		if strings.HasPrefix(line, "# Fake") || strings.HasPrefix(line, "[Fake") {
			t.Errorf("a title added a line: %q", line)
		}
	}
	for _, gone := range []string{"q4-roadmap-deep-dive", "feature-flags-preview", "feature-flags-at-scale", "rate-limits-best-customers"} {
		if strings.Contains(r.Body, "/posts/"+gone+")") {
			t.Errorf("llms.txt lists %s", gone)
		}
	}
	// Newest first: the runbook (2 days ago) before the boring-infrastructure issue.
	if strings.Index(r.Body, "postgres-migration-runbook") > strings.Index(r.Body, "the-case-for-boring-infrastructure") {
		t.Error("posts aren't newest first")
	}
	must(t, get(t, env, "/p/acme-newsletter/llms.txt"), 200, "## Posts")

	if _, err := env.App.Pool.Exec(ctx, `UPDATE website_settings SET index_site = false WHERE workspace_id = $1`, ids.ws); err != nil {
		t.Fatal(err)
	}
	must(t, do(t, env, http.MethodGet, "/llms.txt", nil, newsHost), 404)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE website_settings SET index_site = true WHERE workspace_id = $1`, ids.ws); err != nil {
		t.Fatal(err)
	}
	websiteOff(t, env, ids.ws)
	must(t, do(t, env, http.MethodGet, "/llms.txt", nil, newsHost), 404)
}
