package proxycheck_test

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/proxycheck"
	"github.com/hostrepute/reputecast/internal/store"
)

// TestUntrustedProxy: requests forwarded by a proxy ReputeCast doesn't
// trust are noticed, and the instance's admins are told to list it in
// http.trusted_proxies (review AN-8), once it has forwarded nearly all its
// requests for ten windows: a burst through a NAT names nobody (security
// review).
func TestUntrustedProxy(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	admin := env.CreateUser(apptest.UserOpts{Email: "root@example.org", Name: "Root", InstanceAdmin: true})
	member := env.CreateUser(apptest.UserOpts{Email: "member@example.org", Name: "Member"})

	// A reverse proxy on the same machine forwards readers' requests
	// (the test server's peer is 127.0.0.1, which isn't trusted); each
	// round ends a window.
	forward := func(n int) {
		t.Helper()
		for range n {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, env.Server.URL+"/healthz", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("X-Forwarded-For", "198.51.100.7")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			_ = res.Body.Close()
		}
		if err := env.App.ProxyDetector().Flush(ctx); err != nil {
			t.Fatal(err)
		}
	}
	svc := &proxycheck.Service{Pool: env.App.Pool, Q: env.App.Q}
	forward(25)
	if n, err := svc.Notify(ctx); err != nil || n != 0 {
		t.Fatalf("one window: admins told about %d proxies: %v", n, err)
	}
	for range 9 {
		forward(3)
	}
	if n, err := svc.Notify(ctx); err != nil || n != 1 {
		t.Fatalf("ten windows: admins told about %d proxies: %v", n, err)
	}
	var body string
	if err := env.App.Pool.QueryRow(ctx, `SELECT body FROM notifications WHERE user_id = $1 AND kind = $2`, admin.ID, proxycheck.KindUntrustedProxy).Scan(&body); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"127.0.0.1", "X-Forwarded-For", "http.trusted_proxies", "REPUTECAST_HTTP__TRUSTED_PROXIES"} {
		if !strings.Contains(body, want) {
			t.Errorf("the notification doesn't say %q: %s", want, body)
		}
	}
	var others int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND kind = $2`, member.ID, proxycheck.KindUntrustedProxy).Scan(&others); err != nil || others != 0 {
		t.Fatalf("someone who isn't an admin was told: %d %v", others, err)
	}
	// Told once a week while it lasts.
	if n, err := svc.Notify(ctx); err != nil || n != 0 {
		t.Fatalf("told again at once: %d %v", n, err)
	}

	// The admin API lists it with the setting that fixes it; others can't read it.
	c := env.Client()
	c.SignIn("root@example.org", "")
	var list struct {
		Proxies []struct {
			Address, Header string
			Requests        int64
		}
		Setting, Env string
	}
	c.Do("GET", "/admin/untrusted-proxies", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Proxies) != 1 || list.Proxies[0].Address != "127.0.0.1" || list.Proxies[0].Header != "X-Forwarded-For" || list.Proxies[0].Requests < 25 ||
		list.Setting != "http.trusted_proxies" || list.Env != "REPUTECAST_HTTP__TRUSTED_PROXIES" {
		t.Fatalf("list: %+v", list)
	}
	m := env.Client()
	m.SignIn("member@example.org", "")
	m.Do("GET", "/admin/untrusted-proxies", nil).MustStatus(t, 403)

	// Once the proxy is trusted, it isn't listed or told about any more.
	trusted := &proxycheck.Service{Pool: env.App.Pool, Q: env.App.Q, Trusted: []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}}
	out, err := trusted.List(ctx)
	if err != nil || len(out.Proxies) != 0 {
		t.Fatalf("a trusted proxy listed: %+v %v", out, err)
	}
}

// TestPublicProxyIsOneToCheck: anyone can send forwarding headers, so a
// public address is never one admins are told to add, only one to check;
// and admins hear about three proxies a run and ten a day, those on a
// private network first (security review).
func TestPublicProxyIsOneToCheck(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	admin := env.CreateUser(apptest.UserOpts{Email: "root@example.org", Name: "Root", InstanceAdmin: true})
	now := time.Now()
	svc := &proxycheck.Service{Pool: env.App.Pool, Q: env.App.Q, Now: func() time.Time { return now }}
	seen := func(peer string) {
		t.Helper()
		if err := env.App.Q.ProxySeen(ctx, store.ProxySeenParams{Peer: netip.MustParseAddr(peer), Header: "X-Forwarded-For", Requests: 500, SeenAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	told := func(peer string) bool {
		t.Helper()
		var n int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND kind = $2 AND data->>'peer' = $3`,
			admin.ID, proxycheck.KindUntrustedProxy, peer).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n > 0
	}

	// An address on the internet sent most of the traffic with forwarding
	// headers: a CDN, or anyone who wants to be trusted.
	seen("203.0.113.66")
	if n, err := svc.Notify(ctx); err != nil || n != 1 {
		t.Fatalf("admins told about %d proxies: %v", n, err)
	}
	var title, body string
	if err := env.App.Pool.QueryRow(ctx, `SELECT title, body FROM notifications WHERE user_id = $1 AND kind = $2`, admin.ID, proxycheck.KindUntrustedProxy).Scan(&title, &body); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "Add 203.0.113.66") || strings.Contains(title, "doesn't trust") {
		t.Errorf("admins are told to trust a public address: %s: %s", title, body)
	}
	for _, want := range []string{"203.0.113.66", "X-Forwarded-For", "CDN", "http.trusted_proxies", "REPUTECAST_HTTP__TRUSTED_PROXIES"} {
		if !strings.Contains(body, want) {
			t.Errorf("the notification doesn't say %q: %s", want, body)
		}
	}
	out, err := svc.List(ctx)
	if err != nil || len(out.Proxies) != 1 || strings.HasPrefix(out.Advice, "Add these addresses") || !strings.Contains(out.Advice, "CDN") {
		t.Fatalf("list: %+v %v", out, err)
	}

	// Many proxies at once: another public address and twelve on the
	// private network.
	seen("203.0.113.77")
	for i := range 12 {
		seen(fmt.Sprintf("10.1.0.%d", i+1))
	}
	for run, want := range []int{3, 3, 3, 0} {
		if n, err := svc.Notify(ctx); err != nil || n != want {
			t.Fatalf("run %d: admins told about %d proxies, want %d: %v", run, n, want, err)
		}
	}
	if told("203.0.113.77") || !told("10.1.0.1") || told("10.1.0.12") {
		t.Fatal("the private proxies aren't told about first")
	}
	// A day later, the rest.
	now = now.Add(25 * time.Hour)
	seen("203.0.113.77")
	for i := range 12 {
		seen(fmt.Sprintf("10.1.0.%d", i+1))
	}
	for run, want := range []int{3, 1, 0} {
		if n, err := svc.Notify(ctx); err != nil || n != want {
			t.Fatalf("the next day, run %d: admins told about %d proxies, want %d: %v", run, n, want, err)
		}
	}
	if !told("203.0.113.77") || !told("10.1.0.12") {
		t.Fatal("a proxy was never told about")
	}
}
