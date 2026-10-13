package accounts_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// passkeyEnv points the instance's public URL at localhost (an IP address
// can't be a WebAuthn relying party) and returns the matching origin.
func passkeyEnv(t *testing.T) (*apptest.Env, string) {
	t.Helper()
	env := apptest.New(t)
	port := env.Server.URL[strings.LastIndex(env.Server.URL, ":")+1:]
	origin := "http://localhost:" + port
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE instance_settings SET public_url = $1`, origin); err != nil {
		t.Fatal(err)
	}
	return env, origin
}

func TestPasskeyRegistrationSignInAndSecondFactor(t *testing.T) {
	env, origin := passkeyEnv(t)
	u := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	env.CreateWorkspace("Acme Newsletter", u.ID)
	c := env.Client()
	c.SignIn("linh@acme.io", "")

	// Add a passkey: discoverable, user verification required.
	var opts map[string]any
	c.Do("POST", "/me/passkeys/options", nil).MustStatus(t, 200).JSON(t, &opts)
	options := opts["options"].(map[string]any)
	sel := options["authenticatorSelection"].(map[string]any)
	if sel["residentKey"] != "required" || sel["userVerification"] != "required" {
		t.Fatalf("registration must ask for a discoverable, verified passkey: %v", sel)
	}
	if rp := options["rp"].(map[string]any); rp["id"] != "localhost" {
		t.Fatalf("relying party comes from the public URL: %v", rp)
	}
	key := newSoftAuthenticator(t, "localhost", origin)
	cred := key.register(t, options)
	var pk map[string]any
	c.Do("POST", "/me/passkeys", map[string]any{"challenge": opts["challenge"], "credential": cred, "name": "Work laptop"}).MustStatus(t, 201).JSON(t, &pk)
	if pk["name"] != "Work laptop" || pk["kind"] != "platform" {
		t.Fatalf("passkey: %v", pk)
	}
	// The registration challenge works once.
	c.Do("POST", "/me/passkeys", map[string]any{"challenge": opts["challenge"], "credential": cred}).MustStatus(t, 400)

	// Sign in with the passkey alone: no email.
	anon := env.Client()
	var lo map[string]any
	anon.Do("POST", "/auth/passkey/options", nil).MustStatus(t, 200).JSON(t, &lo)
	if lo["options"].(map[string]any)["allowCredentials"] != nil {
		t.Fatalf("discoverable sign-in must not name credentials: %v", lo["options"])
	}
	answer := key.assert(t, lo["options"].(map[string]any))
	var res map[string]any
	anon.Do("POST", "/auth/passkey", map[string]any{"challenge": lo["challenge"], "credential": answer}).MustStatus(t, 200).JSON(t, &res)
	if res["status"] != "signed_in" || res["email"] != "linh@acme.io" {
		t.Fatalf("passkey sign-in: %v", res)
	}
	var acct map[string]any
	anon.Do("GET", "/me/account", nil).MustStatus(t, 200).JSON(t, &acct)
	keys := acct["passkeys"].([]any)
	if len(keys) != 1 || keys[0].(map[string]any)["thisSession"] != true || keys[0].(map[string]any)["lastUsedAt"] == nil {
		t.Fatalf("the session should be marked as using the passkey: %v", keys)
	}
	// A replayed answer is refused: the challenge was consumed.
	env.Client().Do("POST", "/auth/passkey", map[string]any{"challenge": lo["challenge"], "credential": answer}).MustStatus(t, 400)
	// An unknown passkey is refused with a generic message.
	stranger := newSoftAuthenticator(t, "localhost", origin)
	stranger.userHandle = []byte("nobody-here")
	var lo2 map[string]any
	anon2 := env.Client()
	anon2.Do("POST", "/auth/passkey/options", nil).MustStatus(t, 200).JSON(t, &lo2)
	p := anon2.Do("POST", "/auth/passkey", map[string]any{"challenge": lo2["challenge"], "credential": stranger.assert(t, lo2["options"].(map[string]any))}).MustStatus(t, 401).Problem(t)
	if p["code"] != "passkey_failed" {
		t.Fatalf("unknown passkey: %v", p)
	}

	// A password sign-in now asks for the passkey as the second factor.
	c3 := env.Client()
	var step map[string]any
	c3.Do("POST", "/auth/password", map[string]any{"email": "linh@acme.io", "password": "correct horse battery staple"}).MustStatus(t, 200).JSON(t, &step)
	if step["status"] != "two_factor" || !contains(step["methods"], "passkey") || step["passkeyHint"] != "Work laptop" {
		t.Fatalf("second factor: %v", step)
	}
	var po map[string]any
	c3.Do("POST", "/auth/two-factor/passkey/options", map[string]any{"challenge": step["challenge"]}).MustStatus(t, 200).JSON(t, &po)
	allow := po["options"].(map[string]any)["allowCredentials"].([]any)
	if len(allow) != 1 {
		t.Fatalf("the second factor names the account's passkeys: %v", allow)
	}
	var done map[string]any
	c3.Do("POST", "/auth/two-factor/passkey", map[string]any{"challenge": step["challenge"], "passkeyChallenge": po["challenge"], "credential": key.assert(t, po["options"].(map[string]any))}).
		MustStatus(t, 200).JSON(t, &done)
	if done["status"] != "signed_in" {
		t.Fatalf("passkey second factor: %v", done)
	}

	// Rename. The Owner holds sensitive permissions, so the last second
	// factor can't be removed while two-factor is required.
	id := pk["id"].(string)
	c.Do("PATCH", "/me/passkeys/"+id, map[string]any{"name": "MacBook Pro · Touch ID"}).MustStatus(t, 204)
	if p := c.Do("DELETE", "/me/passkeys/"+id, nil).MustStatus(t, 409).Problem(t); p["code"] != "last_second_factor" {
		t.Fatalf("last second factor: %v", p)
	}
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE instance_settings SET twofa_enforcement = 'off'`); err != nil {
		t.Fatal(err)
	}
	c.Do("DELETE", "/me/passkeys/"+id, nil).MustStatus(t, 204)
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action IN ('account.passkey_add','account.passkey_rename','account.passkey_remove')`).Scan(&n); err != nil || n != 3 {
		t.Fatalf("audit entries: %d %v", n, err)
	}
}

func TestLastPasskeyCantBeRemoved(t *testing.T) {
	env, origin := passkeyEnv(t)
	u := env.CreateUser(apptest.UserOpts{Email: "sofia@acme.io"})
	env.CreateWorkspace("Acme Newsletter", u.ID)
	c := env.Client()
	c.SignIn("sofia@acme.io", "")
	var opts map[string]any
	c.Do("POST", "/me/passkeys/options", nil).MustStatus(t, 200).JSON(t, &opts)
	key := newSoftAuthenticator(t, "localhost", origin)
	var pk map[string]any
	c.Do("POST", "/me/passkeys", map[string]any{"challenge": opts["challenge"], "credential": key.register(t, opts["options"].(map[string]any))}).MustStatus(t, 201).JSON(t, &pk)
	// Without a password, the passkey is the only way in.
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE users SET password_hash = NULL WHERE id = $1`, u.ID); err != nil {
		t.Fatal(err)
	}
	p := c.Do("DELETE", "/me/passkeys/"+pk["id"].(string), nil).MustStatus(t, 409).Problem(t)
	if p["code"] != "last_method" {
		t.Fatalf("last method: %v", p)
	}
}

func contains(list any, want string) bool {
	items, _ := list.([]any)
	for _, it := range items {
		if it == want {
			return true
		}
	}
	return false
}
