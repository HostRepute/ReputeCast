package proxycheck_test

import (
	"context"
	"net/http"
	"net/netip"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/proxycheck"
)

// TestUntrustedProxy: requests forwarded by a proxy ReputeCast doesn't
// trust are noticed, and the instance's admins are told to list it in
// http.trusted_proxies (review AN-8).
func TestUntrustedProxy(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	admin := env.CreateUser(apptest.UserOpts{Email: "root@example.org", Name: "Root", InstanceAdmin: true})
	member := env.CreateUser(apptest.UserOpts{Email: "member@example.org", Name: "Member"})

	// A reverse proxy on the same machine forwards readers' requests
	// (the test server's peer is 127.0.0.1, which isn't trusted).
	for range 25 {
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
	svc := &proxycheck.Service{Pool: env.App.Pool, Q: env.App.Q}
	if n, err := svc.Notify(ctx); err != nil || n != 1 {
		t.Fatalf("admins told about %d proxies: %v", n, err)
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
