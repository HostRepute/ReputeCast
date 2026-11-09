package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// The reset link the server prints uses the address links use: the one an
// admin saved, unless the server's operator pins http.public_url.
func TestResetPasswordLinkAddress(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE instance_settings SET public_url = 'https://old.example.com'`); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REPUTECAST_CONFIG", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REPUTECAST_DATABASE__URL", env.App.Cfg.Database.URL)
	t.Setenv("REPUTECAST_SECRETS__KEY", env.App.Cfg.Secrets.Key)
	t.Setenv("REPUTECAST_PUBLIC_URL", "https://news.example.com")
	link := func(locked string) string {
		t.Helper()
		t.Setenv("REPUTECAST_HTTP__PUBLIC_URL_LOCKED", locked)
		cmd := newUserCmd(&rootOpts{})
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetArgs([]string{"reset-password", "--email", "linh@acme.io"})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("reset-password: %v\n%s", err, out.String())
		}
		return out.String()
	}
	if out := link("false"); !strings.Contains(out, "https://old.example.com/reset-password/") {
		t.Fatalf("unlocked:\n%s", out)
	}
	if out := link("true"); !strings.Contains(out, "https://news.example.com/reset-password/") {
		t.Fatalf("locked:\n%s", out)
	}
}
