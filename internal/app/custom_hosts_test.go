package app_test

// A host a workspace holds (its verified custom domain, short-link domain,
// link tracking domain or publication website) carries only that
// workspace's links and pages: another workspace's would look as if they
// came from the host's owner. The instance's own host, and a host nobody
// holds, carry every workspace's, as links in mail already sent expect.

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/linktoken"
)

const browserUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"

// onHost requests path on host ("" for the test server's own) without
// following a redirect, and returns the status and where it redirects.
func onHost(t *testing.T, env *apptest.Env, host, path string) (int, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, env.Server.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if host != "" {
		req.Host = host
	}
	req.Header.Set("User-Agent", browserUA)
	c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode, resp.Header.Get("Location")
}

func TestAHostCarriesOnlyItsWorkspacesLinks(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	scan := func(sql string, args []any, dest ...any) {
		t.Helper()
		if err := env.App.Pool.QueryRow(ctx, sql, args...).Scan(dest...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	var acme, eng uuid.UUID
	scan(`SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`, nil, &acme)
	scan(`SELECT id FROM workspaces WHERE slug = 'acme-engineering-blog'`, nil, &eng)
	// The demo's acme-newsletter holds news.acme.io (its pages) and
	// go.acme.io (its short links); the engineering blog gets blog.example.
	exec(`INSERT INTO public_page_settings (workspace_id, custom_domain, domain_status, domain_verified_at)
		VALUES ($1, 'blog.example', 'verified', now())
		ON CONFLICT (workspace_id) DO UPDATE SET custom_domain = excluded.custom_domain, domain_status = 'verified',
			domain_verified_at = excluded.domain_verified_at`, eng)

	expect := func(what, host, path string, wantStatus int, wantTo string) {
		t.Helper()
		status, to := onHost(t, env, host, path)
		if status != wantStatus || (wantTo != "" && to != wantTo) {
			t.Errorf("%s on %q: %d to %q, want %d to %q", what, host, status, to, wantStatus, wantTo)
		}
	}

	// Link tracking: a click on an email of acme-newsletter.
	var msg uuid.UUID
	var contact *uuid.UUID
	var pos int32
	var target string
	scan(`SELECT m.id, m.contact_id, l.position, l.url FROM messages m JOIN campaign_links l ON l.campaign_id = m.campaign_id
		WHERE m.workspace_id = $1 AND m.kind <> 'placement' AND m.stream <> 'transactional' AND l.url NOT LIKE '%{{%' AND l.url LIKE 'https://%'
		ORDER BY m.id, l.position LIMIT 1`, []any{acme}, &msg, &contact, &pos, &target)
	cid := uuid.Nil
	if contact != nil {
		cid = *contact
	}
	click := "/t/c/" + env.App.Links.Sign(linktoken.Claims{Kind: linktoken.Click, WorkspaceID: acme, MessageID: msg, ContactID: cid, Link: uint32(pos)})
	expect("acme's click", "", click, http.StatusFound, target)
	expect("acme's click", "news.acme.io", click, http.StatusFound, target)
	expect("acme's click", "nobody.example", click, http.StatusFound, target)
	expect("acme's click", "blog.example", click, http.StatusNotFound, "")
	expect("acme's click", "BLOG.example.:443", click, http.StatusNotFound, "")

	// Its open counts on acme's host, not on the blog's.
	tracked := func() (n int64) {
		scan(`SELECT count(*) FROM delivery_events WHERE message_id = $1 AND original_type = 'ReputeCast tracking'`, []any{msg}, &n)
		return n
	}
	before := tracked()
	open := "/t/o/" + env.App.Links.Sign(linktoken.Claims{Kind: linktoken.Open, WorkspaceID: acme, MessageID: msg, ContactID: cid})
	expect("acme's open", "blog.example", open, http.StatusOK, "")
	expect("acme's open", "news.acme.io", open, http.StatusOK, "")
	deadline := time.Now().Add(10 * time.Second)
	for tracked() == before && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if got := tracked() - before; got != 1 {
		t.Errorf("%d opens counted, want the one on acme's host", got)
	}

	// Short links, at /t/s/ on any host.
	for code, ws := range map[string]uuid.UUID{"Acme01": acme, "Engi01": eng} {
		exec(`INSERT INTO sms_links (code, workspace_id, original, target, created_at, expires_at) VALUES ($1, $2, $3, $3, now(), now() + interval '30 days')`,
			code, ws, "https://"+code+".example/")
	}
	expect("the blog's short link", "", "/t/s/Engi01", http.StatusFound, "https://Engi01.example/")
	expect("the blog's short link", "blog.example", "/t/s/Engi01", http.StatusFound, "https://Engi01.example/")
	expect("the blog's short link", "news.acme.io", "/t/s/Engi01", http.StatusNotFound, "")
	expect("the blog's short link", "go.acme.io", "/t/s/Engi01", http.StatusNotFound, "")
	expect("acme's short link", "news.acme.io", "/t/s/Acme01", http.StatusFound, "https://Acme01.example/")
	expect("acme's short link", "blog.example", "/t/s/Acme01", http.StatusNotFound, "")

	// A push notification's click.
	var pushMsg, pushWS uuid.UUID
	var token string
	scan(`SELECT m.id, m.workspace_id, p.token FROM channel_messages m JOIN push_messages p ON p.message_id = m.id
		WHERE m.status <> 'failed' ORDER BY m.id LIMIT 1`, nil, &pushMsg, &pushWS, &token)
	own, other := "news.acme.io", "blog.example"
	switch pushWS {
	case acme:
	case eng:
		own, other = other, own
	default:
		t.Fatalf("the demo's push notifications are workspace %s's", pushWS)
	}
	pushClick := "/t/p/" + pushMsg.String() + "." + token
	if status, to := onHost(t, env, own, pushClick); status != http.StatusFound || to == "" {
		t.Errorf("a push click on its workspace's host: %d to %q", status, to)
	}
	expect("a push click", other, pushClick, http.StatusNotFound, "")

	// A workspace's pages under /p/.
	expect("acme's subscribe page", "", "/p/acme-newsletter/subscribe", http.StatusOK, "")
	expect("acme's subscribe page", "news.acme.io", "/p/acme-newsletter/subscribe", http.StatusOK, "")
	expect("acme's subscribe page", "nobody.example", "/p/acme-newsletter/subscribe", http.StatusOK, "")
	expect("acme's subscribe page", "blog.example", "/p/acme-newsletter/subscribe", http.StatusNotFound, "")
	expect("acme's archive", "blog.example", "/p/acme-newsletter/archive", http.StatusNotFound, "")
	expect("acme's push settings", "blog.example", "/p/acme-newsletter/push/config", http.StatusNotFound, "")
	expect("the blog's subscribe page", "news.acme.io", "/p/acme-engineering-blog/subscribe", http.StatusNotFound, "")
	// What /p/_/ serves is nobody's.
	expect("the custom domains' TLS check", "news.acme.io", "/p/_/tls-check?domain=blog.example", http.StatusOK, "")
}
