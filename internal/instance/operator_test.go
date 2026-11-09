package instance_test

import (
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/instance"
)

// An operator who runs the instance for someone else pins the public URL
// (http.public_url_locked) and says who upgrades it (updates.managed_by).
func TestOperatorManagedSettings(t *testing.T) {
	env, admin := setup(t)
	// An address an admin saved before the server locked it.
	admin.Do("PUT", "/admin/instance", map[string]any{"name": "Acme ReputeCast", "publicUrl": "https://old.acme.io", "defaultTimezone": "UTC"}).MustStatus(t, 200)

	cfg := *env.App.Cfg
	cfg.HTTP.PublicURL = "https://news.acme.io"
	cfg.HTTP.PublicURLLocked = true
	cfg.Updates.Check = false
	cfg.Updates.ManagedBy = " Northwind Hosting "
	moved := false
	e2 := serve(t, env, func(s *instance.Service) {
		s.Cfg = &cfg
		s.OnPublicURL = func(context.Context, string, string) (*instance.PublicURLWebhooks, error) {
			moved = true
			return nil, nil
		}
	})
	c := e2.Client()
	c.SignInDemo("linh@acme.io")

	var s struct {
		Name            string `json:"name"`
		PublicURL       string `json:"publicUrl"`
		PublicURLLocked bool   `json:"publicUrlLocked"`
	}
	c.Do("GET", "/admin/instance", nil).MustStatus(t, 200).JSON(t, &s)
	if s.PublicURL != "https://news.acme.io" || !s.PublicURLLocked {
		t.Fatalf("locked settings: %+v", s)
	}

	// Another address is refused; the problem says why.
	p := c.Do("PUT", "/admin/instance", map[string]any{"name": "Acme ReputeCast", "publicUrl": "https://evil.example", "defaultTimezone": "UTC"}).MustStatus(t, 409).Problem(t)
	if p["code"] != "operator_locked" {
		t.Fatalf("problem: %v", p)
	}

	// The rest still saves, with the pinned address or none; the stored
	// address isn't touched and no webhook moves.
	for _, url := range []string{"https://NEWS.acme.io/", ""} {
		c.Do("PUT", "/admin/instance", map[string]any{"name": "Acme Mail", "publicUrl": url, "defaultTimezone": "Europe/Berlin"}).MustStatus(t, 200).JSON(t, &s)
		if s.Name != "Acme Mail" || s.PublicURL != "https://news.acme.io" {
			t.Fatalf("save with %q: %+v", url, s)
		}
	}
	var stored string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT public_url FROM instance_settings`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != "https://old.acme.io" || moved {
		t.Fatalf("stored public URL: %q, webhooks moved %v", stored, moved)
	}

	// A kept URL check shows only when it is the pinned address's: not one
	// of the old address, nor one kept before checks named their address.
	var shown struct {
		URLCheck *struct {
			URL string `json:"url"`
		} `json:"urlCheck"`
	}
	for check, want := range map[string]bool{
		`{"url":"https://old.acme.io","checkedAt":"2026-11-01T10:00:00Z"}`:   false,
		`{"checkedAt":"2026-11-01T10:00:00Z"}`:                               false,
		`{"url":"https://news.acme.io/","checkedAt":"2026-11-01T10:00:00Z"}`: true,
	} {
		if _, err := env.App.Pool.Exec(context.Background(), `UPDATE instance_settings SET public_url_check = $1, public_url_checked_at = now()`, check); err != nil {
			t.Fatal(err)
		}
		shown.URLCheck = nil
		c.Do("GET", "/admin/instance", nil).MustStatus(t, 200).JSON(t, &shown)
		if (shown.URLCheck != nil) != want {
			t.Errorf("check %s shown: %v", check, shown.URLCheck != nil)
		}
	}

	var v struct {
		CheckAllowed   bool   `json:"checkAllowed"`
		ManagedBy      string `json:"managedBy"`
		UpgradeCommand string `json:"upgradeCommand"`
	}
	c.Do("GET", "/admin/updates", nil).MustStatus(t, 200).JSON(t, &v)
	if v.ManagedBy != "Northwind Hosting" || v.CheckAllowed {
		t.Fatalf("version: %+v", v)
	}

	// Without the operator's settings nothing is locked or managed.
	s.PublicURLLocked = false
	admin.Do("GET", "/admin/instance", nil).MustStatus(t, 200).JSON(t, &s)
	if s.PublicURLLocked || s.PublicURL != "https://old.acme.io" {
		t.Fatalf("unlocked settings: %+v", s)
	}
	v.ManagedBy = ""
	admin.Do("GET", "/admin/updates", nil).MustStatus(t, 200).JSON(t, &v)
	if v.ManagedBy != "" || !v.CheckAllowed {
		t.Fatalf("unmanaged version: %+v", v)
	}
}
