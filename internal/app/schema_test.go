package app_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/apptest"
)

// A schema behind the binary turns the web app read-only until the
// migrations land (review 2026-11 OP-2, OP-3): reads work and say so,
// writes answer 503 maintenance, sign-in works, /readyz is 503.
func TestPendingMigrationsServeReadOnly(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var latest int64
	if err := env.App.Pool.QueryRow(ctx, `SELECT max(version_id) FROM goose_db_version`).Scan(&latest); err != nil {
		t.Fatal(err)
	}
	// The newest migration "isn't applied yet".
	if _, err := env.App.Pool.Exec(ctx, `UPDATE goose_db_version SET version_id = -version_id WHERE version_id = $1`, latest); err != nil {
		t.Fatal(err)
	}
	ready := app.WatchSchema(ctx, env.App, 100*time.Millisecond)
	select {
	case <-ready:
		t.Fatal("ready with a migration pending")
	default:
	}

	c := env.Client()
	c.SignInDemo("linh@acme.io") // signing in still works
	r := c.Do("GET", "/workspaces/acme-newsletter/campaigns", nil).MustStatus(t, 200)
	if r.Header.Get("X-ReputeCast-Read-Only") != "upgrade" || r.Header.Get("X-ReputeCast-Upgrade-Since") == "" {
		t.Fatalf("read headers: %v", r.Header)
	}
	p := c.Do("POST", "/workspaces/acme-newsletter/lists", map[string]any{"name": "During the upgrade"}).MustStatus(t, http.StatusServiceUnavailable).Problem(t)
	if p["code"] != "maintenance" || p["meta"] == nil {
		t.Fatalf("write: %+v", p)
	}
	if r := c.DoRaw("GET", "/readyz", "", nil); r.Status != http.StatusServiceUnavailable {
		t.Fatalf("readyz %d", r.Status)
	}
	if r := c.DoRaw("GET", "/healthz", "", nil); r.Status != http.StatusOK {
		t.Fatalf("healthz %d", r.Status)
	}

	// `migrate up` runs elsewhere: writes resume without a restart.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE goose_db_version SET version_id = -version_id WHERE version_id = $1`, -latest); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("still read-only after the migrations landed")
	}
	r = c.Do("POST", "/workspaces/acme-newsletter/lists", map[string]any{"name": "After the upgrade"}).MustStatus(t, 201)
	if r.Header.Get("X-ReputeCast-Read-Only") != "" {
		t.Fatal("still flagged read-only")
	}
	if r := c.DoRaw("GET", "/readyz", "", nil); r.Status != http.StatusOK {
		t.Fatalf("readyz after %d", r.Status)
	}
}
