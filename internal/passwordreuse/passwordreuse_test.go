package passwordreuse_test

import (
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/demo"
)

func TestReuseRule(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	// No breach lookups from tests.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE instance_settings SET breached_check = false, password_reuse = 5`); err != nil {
		t.Fatal(err)
	}
	env.App.InvalidatePolicy()
	c := env.Client()
	c.SignInDemo("aisha@acme.io")
	change := func(current, next string) int {
		return c.Do("POST", "/me/password", map[string]any{"current": current, "password": next}).Status
	}
	if s := change(demo.Password, "first new passphrase"); s != 204 {
		t.Fatalf("first change: %d", s)
	}
	if s := change("first new passphrase", demo.Password); s != 422 {
		t.Fatalf("re-using the previous password: %d", s)
	}
	if s := change("first new passphrase", "second new passphrase"); s != 204 {
		t.Fatalf("second change: %d", s)
	}
	var n int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM user_password_history h JOIN users u ON u.id = h.user_id WHERE u.email = 'aisha@acme.io'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("history: %d %v", n, err)
	}
	// With re-use allowed, an old password is fine again.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE instance_settings SET password_reuse = 0`); err != nil {
		t.Fatal(err)
	}
	env.App.InvalidatePolicy()
	if s := change("second new passphrase", "first new passphrase"); s != 204 {
		t.Fatalf("re-use allowed: %d", s)
	}
}
