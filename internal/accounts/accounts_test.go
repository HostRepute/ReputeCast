package accounts_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/hostrepute/reputecast/internal/apptest"
)

func TestPasswordSignInAndMe(t *testing.T) {
	env := apptest.New(t)
	u := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen", InstanceAdmin: true})
	ws := env.CreateWorkspace("Acme Newsletter", u.ID)

	c := env.Client()
	c.Do("GET", "/me", nil).MustStatus(t, 401)

	var id map[string]any
	c.Do("POST", "/auth/identify", map[string]any{"email": "Linh@Acme.io"}).MustStatus(t, 200).JSON(t, &id)
	if id["next"] != "password" || id["email"] != "linh@acme.io" {
		t.Fatalf("identify: %v", id)
	}

	r := c.Do("POST", "/auth/password", map[string]any{"email": "linh@acme.io", "password": "correct horse battery staple"}).MustStatus(t, 200)
	var res map[string]any
	r.JSON(t, &res)
	if res["status"] != "signed_in" {
		t.Fatalf("sign-in: %v", res)
	}
	if !strings.Contains(r.Header.Get("Set-Cookie"), "HttpOnly") {
		t.Fatalf("cookie must be HttpOnly: %s", r.Header.Get("Set-Cookie"))
	}

	var me map[string]any
	c.Do("GET", "/me", nil).MustStatus(t, 200).JSON(t, &me)
	wss := me["workspaces"].([]any)
	if len(wss) != 1 || wss[0].(map[string]any)["slug"] != ws.Slug || wss[0].(map[string]any)["roleKey"] != "owner" {
		t.Fatalf("me workspaces: %v", wss)
	}

	var wme map[string]any
	c.Do("GET", "/workspaces/"+ws.Slug+"/me", nil).MustStatus(t, 200).JSON(t, &wme)
	perms := wme["permissions"].([]any)
	if len(perms) < 30 {
		t.Fatalf("owner should hold every permission, got %d", len(perms))
	}

	// A stranger can't see the workspace at all.
	env.CreateUser(apptest.UserOpts{Email: "kim@example.com"})
	other := env.Client()
	other.SignIn("kim@example.com", "")
	other.Do("GET", "/workspaces/"+ws.Slug+"/me", nil).MustStatus(t, 404)

	c.Do("POST", "/auth/sign-out", nil).MustStatus(t, 204)
	c.Do("GET", "/me", nil).MustStatus(t, 401)
}

func TestWrongPasswordIsGenericAndLocks(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "aisha@acme.io"})
	_, _ = env.App.Pool.Exec(context.Background(), `UPDATE instance_settings SET lockout_attempts = 3`)
	c := env.Client()

	known := c.Do("POST", "/auth/password", map[string]any{"email": "aisha@acme.io", "password": "nope"}).MustStatus(t, 401).Problem(t)
	unknown := c.Do("POST", "/auth/password", map[string]any{"email": "ghost@acme.io", "password": "nope"}).MustStatus(t, 401).Problem(t)
	if known["detail"] != unknown["detail"] {
		t.Fatalf("messages must not reveal whether an account exists:\n%v\n%v", known["detail"], unknown["detail"])
	}
	c.Do("POST", "/auth/password", map[string]any{"email": "aisha@acme.io", "password": "nope"}).MustStatus(t, 401)
	locked := c.Do("POST", "/auth/password", map[string]any{"email": "aisha@acme.io", "password": "nope"}).MustStatus(t, 429).Problem(t)
	if locked["code"] != "locked" {
		t.Fatalf("expected locked: %v", locked)
	}
	// Even the right password is refused while locked.
	c.Do("POST", "/auth/password", map[string]any{"email": "aisha@acme.io", "password": "correct horse battery staple"}).MustStatus(t, 429)
}

func TestTemporaryPasswordThenTOTP(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Password: "temp-pass-1234", InstanceAdmin: true, MustChange: true})
	c := env.Client()
	c.SignIn("linh@acme.io", "temp-pass-1234")

	var me map[string]any
	c.Do("GET", "/me", nil).MustStatus(t, 200).JSON(t, &me)
	if me["pending"] != "password_change" {
		t.Fatalf("pending: %v", me["pending"])
	}
	// Everything except the setup endpoints is blocked until the password is replaced.
	ws := c.Do("POST", "/workspaces", map[string]any{"name": "Acme Newsletter"})
	_ = ws
	c.Do("POST", "/me/password", map[string]any{"current": "temp-pass-1234", "password": "short"}).MustStatus(t, 422)
	c.Do("POST", "/me/password", map[string]any{"current": "temp-pass-1234", "password": "a much longer passphrase"}).MustStatus(t, 204)

	var setup map[string]any
	c.Do("POST", "/me/totp", nil).MustStatus(t, 200).JSON(t, &setup)
	secret := strings.ReplaceAll(setup["secret"].(string), " ", "")
	c.Do("POST", "/me/totp/confirm", map[string]any{"challenge": setup["challenge"], "code": "000000"}).MustStatus(t, 422)
	code, _ := totp.GenerateCode(secret, time.Now())
	var codes map[string]any
	c.Do("POST", "/me/totp/confirm", map[string]any{"challenge": setup["challenge"], "code": code}).MustStatus(t, 200).JSON(t, &codes)
	rc := codes["codes"].([]any)
	if len(rc) != 10 {
		t.Fatalf("want 10 recovery codes, got %d", len(rc))
	}

	// Sign in again: now a second factor is required.
	c2 := env.Client()
	var step map[string]any
	c2.Do("POST", "/auth/password", map[string]any{"email": "linh@acme.io", "password": "a much longer passphrase"}).MustStatus(t, 200).JSON(t, &step)
	if step["status"] != "two_factor" {
		t.Fatalf("expected two_factor, got %v", step)
	}
	c2.Do("POST", "/auth/two-factor", map[string]any{"challenge": step["challenge"], "method": "recovery", "code": "aaaa-bbbb-cccc"}).MustStatus(t, 401)
	var done map[string]any
	c2.Do("POST", "/auth/two-factor", map[string]any{"challenge": step["challenge"], "method": "recovery", "code": rc[0]}).MustStatus(t, 200).JSON(t, &done)
	if done["status"] != "signed_in" || done["recoveryCodesLeft"].(float64) != 9 {
		t.Fatalf("recovery sign-in: %v", done)
	}
	// Each recovery code works once.
	c3 := env.Client()
	c3.Do("POST", "/auth/password", map[string]any{"email": "linh@acme.io", "password": "a much longer passphrase"}).MustStatus(t, 200).JSON(t, &step)
	c3.Do("POST", "/auth/two-factor", map[string]any{"challenge": step["challenge"], "method": "recovery", "code": rc[0]}).MustStatus(t, 401)
}

func TestCrossOriginWritesRejected(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "linh@acme.io"})
	c := env.Client()
	c.SignIn("linh@acme.io", "")
	req := c.Do("POST", "/auth/sign-out", nil)
	if req.Status != 204 {
		t.Fatalf("same-origin sign-out: %d", req.Status)
	}
}
