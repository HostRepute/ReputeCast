package appearance_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hostrepute/reputecast/internal/appearance"
	"github.com/hostrepute/reputecast/internal/apptest"
)

type meView struct {
	User struct {
		Theme string `json:"theme"`
	} `json:"user"`
}

func auditRows(t *testing.T, env *apptest.Env) []map[string]map[string]any {
	t.Helper()
	rows, err := env.App.Pool.Query(context.Background(), `SELECT coalesce(before, '{}'), coalesce(after, '{}') FROM audit_events WHERE action = 'account.appearance_update' ORDER BY occurred_at, id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []map[string]map[string]any
	for rows.Next() {
		var before, after []byte
		if err := rows.Scan(&before, &after); err != nil {
			t.Fatal(err)
		}
		row := map[string]map[string]any{}
		for k, b := range map[string][]byte{"before": before, "after": after} {
			var v map[string]any
			_ = json.Unmarshal(b, &v)
			row[k] = v
		}
		out = append(out, row)
	}
	return out
}

func TestMemberTheme(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	env.CreateUser(apptest.UserOpts{Email: "marcus@acme.io", Name: "Marcus Chen"})
	linh := env.Client()
	linh.SignIn("linh@acme.io", "")

	// Nothing chosen: the app follows the device.
	var me meView
	linh.Do("GET", "/me", nil).MustStatus(t, 200).JSON(t, &me)
	if me.User.Theme != appearance.System {
		t.Fatalf("nothing chosen yet: %q", me.User.Theme)
	}

	var saved appearance.Choice
	linh.Do("PUT", "/me/appearance", map[string]any{"theme": "dark"}).MustStatus(t, 200).JSON(t, &saved)
	if saved.Theme != appearance.Dark {
		t.Fatalf("saved: %+v", saved)
	}
	// It follows the member to another browser.
	other := env.Client()
	other.SignIn("linh@acme.io", "")
	other.Do("GET", "/me", nil).MustStatus(t, 200).JSON(t, &me)
	if me.User.Theme != appearance.Dark {
		t.Fatalf("/me in another browser: %q", me.User.Theme)
	}
	// It is the member's own: another member still follows their device.
	marcus := env.Client()
	marcus.SignIn("marcus@acme.io", "")
	var theirs meView
	marcus.Do("GET", "/me", nil).MustStatus(t, 200).JSON(t, &theirs)
	if theirs.User.Theme != appearance.System {
		t.Fatalf("another member's theme: %q", theirs.User.Theme)
	}

	rows := auditRows(t, env)
	if len(rows) != 1 || rows[0]["before"]["theme"] != "system" || rows[0]["after"]["theme"] != "dark" {
		t.Fatalf("audit: %+v", rows)
	}
	// Saving the same theme again changes nothing and logs nothing.
	linh.Do("PUT", "/me/appearance", map[string]any{"theme": "dark"}).MustStatus(t, 200)
	if n := len(auditRows(t, env)); n != 1 {
		t.Fatalf("an unchanged save was audited (%d entries)", n)
	}

	// Back to the device's setting: stored as no choice.
	linh.Do("PUT", "/me/appearance", map[string]any{"theme": "system"}).MustStatus(t, 200).JSON(t, &saved)
	linh.Do("GET", "/me", nil).MustStatus(t, 200).JSON(t, &me)
	if saved.Theme != appearance.System || me.User.Theme != appearance.System {
		t.Fatalf("back to system: saved %+v, /me %q", saved, me.User.Theme)
	}
	var stored *string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT theme FROM users WHERE email = 'linh@acme.io'`).Scan(&stored); err != nil || stored != nil {
		t.Fatalf("system is stored as NULL: %v %v", stored, err)
	}
	if rows := auditRows(t, env); len(rows) != 2 || rows[1]["before"]["theme"] != "dark" || rows[1]["after"]["theme"] != "system" {
		t.Fatalf("audit after going back: %+v", rows)
	}

	// Anything else is refused, and signed-out callers can't save.
	linh.Do("PUT", "/me/appearance", map[string]any{"theme": "sepia"}).MustStatus(t, 422)
	linh.Do("PUT", "/me/appearance", map[string]any{}).MustStatus(t, 422)
	env.Client().Do("PUT", "/me/appearance", map[string]any{"theme": "dark"}).MustStatus(t, 401)
}
