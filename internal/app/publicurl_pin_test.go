package app_test

import (
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Turning on http.public_url_locked (or changing the locked address) takes
// effect without anyone saving it in Instance admin: the configured address
// becomes the saved one once, audited as the server configuration's.
func TestPinPublicURL(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	if _, err := env.App.Pool.Exec(ctx, `UPDATE instance_settings SET public_url = 'https://old.example.com', public_url_check = '{"url":"https://old.example.com"}'`); err != nil {
		t.Fatal(err)
	}
	stored := func() (url string, check bool) {
		t.Helper()
		if err := env.App.Pool.QueryRow(ctx, `SELECT public_url, public_url_check IS NOT NULL FROM instance_settings`).Scan(&url, &check); err != nil {
			t.Fatal(err)
		}
		return url, check
	}
	audits := func() int {
		t.Helper()
		var n int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'instance.public_url_pinned'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// Unlocked: nothing happens.
	if err := env.App.PinPublicURL(ctx); err != nil {
		t.Fatal(err)
	}
	if u, _ := stored(); u != "https://old.example.com" || audits() != 0 {
		t.Fatalf("unlocked: %q, %d audited", u, audits())
	}

	env.App.Cfg.HTTP.PublicURLLocked = true
	env.App.Cfg.HTTP.PublicURL = "https://news.example.com/"
	if err := env.App.PinPublicURL(ctx); err != nil {
		t.Fatal(err)
	}
	if u, check := stored(); u != "https://news.example.com" || check || env.App.URLs.Base() != "https://news.example.com" || audits() != 1 {
		t.Fatalf("locked: %q (check kept %v), links %q, %d audited", u, check, env.App.URLs.Base(), audits())
	}
	var actor, before string
	if err := env.App.Pool.QueryRow(ctx, `SELECT actor_name, before->>'public_url' FROM audit_events WHERE action = 'instance.public_url_pinned'`).Scan(&actor, &before); err != nil {
		t.Fatal(err)
	}
	if actor != "Server configuration" || before != "https://old.example.com" {
		t.Fatalf("audit: %q, before %q", actor, before)
	}

	// Once recorded, the next start has nothing to do.
	if err := env.App.PinPublicURL(ctx); err != nil {
		t.Fatal(err)
	}
	if audits() != 1 {
		t.Fatalf("pinned twice: %d", audits())
	}
}
