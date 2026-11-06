package texting_test

// Short links on the short-link domains: a workspace's domain carries its
// own links, never another workspace's, whichever workspaces stored the
// same host before a host had to be free.

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/store"
)

func workspaceNamed(t *testing.T, env *apptest.Env, slug string) *store.Workspace {
	t.Helper()
	w, err := env.App.Q.GetWorkspaceBySlug(context.Background(), slug)
	if err != nil {
		t.Fatal(err)
	}
	return &w
}

func run(t *testing.T, env *apptest.Env, sql string, args ...any) {
	t.Helper()
	if _, err := env.App.Pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// shortDomain stores host as ws's verified short-link domain, verified at,
// never checked since: as two workspaces could hold one host before a host
// had to be free, and as the upgrade leaves them.
func shortDomain(t *testing.T, env *apptest.Env, ws uuid.UUID, host string, at time.Time) {
	t.Helper()
	run(t, env, `INSERT INTO sms_settings (workspace_id, short_domain, short_domain_status, short_domain_verified_at, short_domain_detail, updated_at)
		VALUES ($1, $2, 'verified', $3, 'CNAME points at reputecast.example.com.', now())
		ON CONFLICT (workspace_id) DO UPDATE SET short_domain = excluded.short_domain, short_domain_status = 'verified',
			short_domain_verified_at = excluded.short_domain_verified_at, short_domain_checked_at = NULL, short_domain_detail = excluded.short_domain_detail`, ws, host, at)
}

// shortLink adds a short link of ws to target.
func shortLink(t *testing.T, env *apptest.Env, ws uuid.UUID, code, target string) {
	t.Helper()
	run(t, env, `INSERT INTO sms_links (code, workspace_id, original, target, created_at, expires_at) VALUES ($1, $2, $3, $3, now(), now() + interval '30 days')`,
		code, ws, target)
}

// follow requests path on host ("" for the test server's own) and returns
// the status and where it redirects.
func follow(t *testing.T, env *apptest.Env, host, path string) (int, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, env.Server.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if host != "" {
		req.Host = host
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X)")
	c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode, resp.Header.Get("Location")
}

// A short-link domain carries its own workspace's links only, at its root
// and at /t/s/; the instance's own host carries every workspace's. Two
// workspaces that hold one host each answer there for their own links.
func TestShortLinksAnswerOnlyOnTheirDomain(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	acme, eng := workspaceNamed(t, env, "acme-newsletter"), workspaceNamed(t, env, "acme-engineering-blog")
	shortDomain(t, env, eng.ID, "go.blog.example", time.Now().Add(-48*time.Hour))
	shortLink(t, env, acme.ID, "Acme01", "https://acme.example/summit")
	shortLink(t, env, eng.ID, "Engi01", "https://blog.example/post")

	check := func(host, path, want string) {
		t.Helper()
		status, to := follow(t, env, host, path)
		switch {
		case want != "" && (status != http.StatusFound || to != want):
			t.Errorf("%s%s: %d to %q, want the redirect to %s", host, path, status, to, want)
		case want == "" && status != http.StatusNotFound:
			t.Errorf("%s%s: %d to %q, want 404: the code isn't this domain's workspace's", host, path, status, to)
		}
	}
	check("go.acme.io", "/Acme01", "https://acme.example/summit")
	check("go.acme.io", "/Engi01", "")
	check("go.acme.io", "/t/s/Engi01", "")
	check("go.blog.example", "/Engi01", "https://blog.example/post")
	check("go.blog.example:443", "/t/s/Engi01", "https://blog.example/post")
	check("go.blog.example", "/Acme01", "")
	check("", "/t/s/Engi01", "https://blog.example/post")
	check("", "/t/s/Acme01", "https://acme.example/summit")

	// Both stored go.blog.example before a host had to be free: until a
	// check settles whose it is, each answers for its own codes there.
	shortDomain(t, env, acme.ID, "go.blog.example", time.Now().Add(-time.Hour))
	check("go.blog.example", "/Acme01", "https://acme.example/summit")
	check("go.blog.example", "/Engi01", "https://blog.example/post")
}
