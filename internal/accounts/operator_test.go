package accounts_test

import (
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// The first sign-in on a server whose operator pins the public URL
// (http.public_url_locked) and turns the update check off (updates.check).
func TestSetupWithOperatorSettings(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	env.App.Cfg.HTTP.PublicURLLocked = true
	env.App.Cfg.Updates.Check = false
	pinned := env.App.Cfg.HTTP.PublicURL
	env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Password: "temp-pass-1234", InstanceAdmin: true, MustChange: true})
	if _, err := env.App.Pool.Exec(ctx, `UPDATE instance_settings SET setup_completed_at = NULL, basics_confirmed_at = NULL, public_url = 'https://old.acme.io', update_check = false`); err != nil {
		t.Fatal(err)
	}
	c := env.Client()
	c.SignIn("linh@acme.io", "temp-pass-1234")
	c.Do("POST", "/me/password", map[string]any{"current": "temp-pass-1234", "password": "a much longer passphrase"}).MustStatus(t, 204)
	var totpSetup map[string]any
	c.Do("POST", "/me/totp", nil).MustStatus(t, 200).JSON(t, &totpSetup)
	c.Do("POST", "/me/totp/confirm", map[string]any{"challenge": totpSetup["challenge"], "code": totpCode(t, totpSetup["secret"].(string))}).MustStatus(t, 200)
	c.Do("POST", "/me/recovery-codes/saved", nil).MustStatus(t, 204)

	var st struct {
		Current  string `json:"current"`
		Instance struct {
			PublicURL          string `json:"publicUrl"`
			PublicURLLocked    bool   `json:"publicUrlLocked"`
			UpdateCheckAllowed bool   `json:"updateCheckAllowed"`
		} `json:"instance"`
	}
	c.Do("GET", "/setup", nil).MustStatus(t, 200).JSON(t, &st)
	if st.Current != "instance" || st.Instance.PublicURL != pinned || !st.Instance.PublicURLLocked || st.Instance.UpdateCheckAllowed {
		t.Fatalf("setup: %+v", st)
	}

	basics := map[string]any{"name": "Acme ReputeCast", "publicUrl": "https://evil.example", "timezone": "UTC", "updateCheck": true}
	if p := c.Do("PUT", "/setup/instance", basics).MustStatus(t, 409).Problem(t); p["code"] != "operator_locked" {
		t.Fatalf("another address: %v", p)
	}
	basics["publicUrl"] = pinned + "/"
	c.Do("PUT", "/setup/instance", basics).MustStatus(t, 204)
	var stored string
	var check bool
	if err := env.App.Pool.QueryRow(ctx, `SELECT public_url, update_check FROM instance_settings`).Scan(&stored, &check); err != nil {
		t.Fatal(err)
	}
	// The pinned address is used without being stored, and the update
	// check stays off whatever the form sent.
	if stored != "https://old.acme.io" || check {
		t.Fatalf("stored: %q, update check %v", stored, check)
	}
	// What the app tells the browser names the pinned address too.
	var meta struct {
		PublicURL string `json:"publicUrl"`
	}
	c.Do("GET", "/meta", nil).MustStatus(t, 200).JSON(t, &meta)
	var me struct {
		Instance struct {
			PublicURL string `json:"publicUrl"`
		} `json:"instance"`
	}
	c.Do("GET", "/me", nil).MustStatus(t, 200).JSON(t, &me)
	if meta.PublicURL != pinned || me.Instance.PublicURL != pinned {
		t.Fatalf("meta %q, me %q", meta.PublicURL, me.Instance.PublicURL)
	}
}
