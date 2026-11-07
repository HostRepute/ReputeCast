package accounts_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
)

// signedInAgo moves the sessions of an email back in time.
func signedInAgo(t *testing.T, env *apptest.Env, email string, minutes int) {
	t.Helper()
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE sessions SET created_at = now() - make_interval(mins => $2)
		WHERE user_id = (SELECT id FROM users WHERE email = $1)`, email, minutes); err != nil {
		t.Fatal(err)
	}
}

// enforceAcme turns domain enforcement on for acme.io (the demo's Google
// Workspace provider).
func enforceAcme(t *testing.T, env *apptest.Env) {
	t.Helper()
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE sso_providers SET enforce_domains = ARRAY['acme.io'] WHERE preset = 'google'`); err != nil {
		t.Fatal(err)
	}
}

// An account without a password gets a first one only from a session that
// signed in moments ago, and never where single sign-on manages it: a
// stolen session cookie mustn't buy a way in that outlives the identity
// provider.
func TestFirstPasswordNeedsARecentSignInOutsideEnforcedSSO(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	pw := map[string]any{"password": "a long enough new passphrase"}

	// Marcus signs in with Google Workspace SSO and has no password.
	old := env.Client()
	old.SignInDemo("marcus@acme.io")
	signedInAgo(t, env, "marcus@acme.io", 20)
	if p := old.Do("POST", "/me/password", pw).MustStatus(t, 403).Problem(t); p["code"] != "reauth_required" {
		t.Fatalf("old session: %v", p)
	}
	fresh := env.Client()
	fresh.SignInDemo("marcus@acme.io")
	fresh.Do("POST", "/me/password", pw).MustStatus(t, 204)
	if n := countRows(t, env, `SELECT count(*) FROM audit_events WHERE action = 'account.password_change' AND target_name = 'marcus@acme.io'`); n != 1 {
		t.Fatalf("%d password changes audited", n)
	}

	// Once acme.io enforces SSO, Sofia (SSO only) can't add one; Kenji, a
	// break-glass Owner, still can.
	enforceAcme(t, env)
	sofia := env.Client()
	sofia.SignInDemo("sofia@acme.io")
	if p := sofia.Do("POST", "/me/password", pw).MustStatus(t, 403).Problem(t); p["code"] != "sso_required" {
		t.Fatalf("enforced domain: %v", p)
	}
	kenji := env.Client()
	kenji.SignInDemo("kenji@acme.io")
	kenji.Do("POST", "/me/password", pw).MustStatus(t, 204)
}

// An account managed by single sign-on keeps its email at a domain the
// provider manages, and an account without a password moves its email only
// from a fresh sign-in.
func TestEmailChangeStaysWhereSSOManagesIt(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	menv := withMail(t, env, &fakeMail{available: true})
	enforceAcme(t, env)

	sofia := menv.Client()
	sofia.SignInDemo("sofia@acme.io")
	p := sofia.Do("POST", "/me/email", map[string]any{"email": "sofia@example.org"}).MustStatus(t, 422).Problem(t)
	if errs, _ := p["errors"].([]any); len(errs) == 0 || errs[0].(map[string]any)["location"] != "body.email" {
		t.Fatalf("leaving the enforced domain: %v", p)
	}
	sofia.Do("POST", "/me/email", map[string]any{"email": "sofia.rossi@acme.io"}).MustStatus(t, 200)
	signedInAgo(t, env, "sofia@acme.io", 20)
	if p := sofia.Do("POST", "/me/email", map[string]any{"email": "s.rossi@acme.io"}).MustStatus(t, 403).Problem(t); p["code"] != "reauth_required" {
		t.Fatalf("old session: %v", p)
	}

	// A break-glass Owner may move, from a fresh sign-in.
	kenji := menv.Client()
	kenji.SignInDemo("kenji@acme.io")
	kenji.Do("POST", "/me/email", map[string]any{"email": "kenji@example.org"}).MustStatus(t, 200)
}

// Account security answers people only: an API key gets a refusal, not a
// crash.
func TestAccountSecurityIsForPeople(t *testing.T) {
	env := apptest.New(t)
	u := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	ws := env.CreateWorkspace("Acme Newsletter", u.ID)
	key, hash, masked := authn.NewAPIKey(authn.APIKeyPrefix)
	if _, err := env.App.Pool.Exec(context.Background(), `INSERT INTO api_keys (id, workspace_id, name, prefix, token_hash, permissions, created_by) VALUES ($1, $2, 'k', $3, $4, ARRAY['reports:view'], $5)`,
		uuid.Must(uuid.NewV7()), ws.ID, masked, hash, u.ID); err != nil {
		t.Fatal(err)
	}
	k := env.Client()
	k.Bearer = key
	for _, r := range []struct {
		path string
		body any
	}{
		{"/me/password", map[string]any{"password": "a long enough new passphrase"}},
		{"/me/totp", nil},
		{"/me/totp/confirm", map[string]any{"challenge": "not-a-real-challenge", "code": "123456"}},
		{"/me/recovery-codes", nil},
	} {
		if p := k.Do("POST", r.path, r.body).MustStatus(t, 403).Problem(t); p["code"] != "humans_only" {
			t.Fatalf("%s: %v", r.path, p)
		}
	}
}

func countRows(t *testing.T, env *apptest.Env, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return n
}
