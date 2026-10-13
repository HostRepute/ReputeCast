package accounts_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/apptest"
)

func TestPasswordResetWithoutSystemProvider(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "aisha@acme.io"})
	c := env.Client()
	var known, unknown map[string]any
	c.Do("POST", "/auth/password-reset", map[string]any{"email": "Aisha@acme.io"}).MustStatus(t, 200).JSON(t, &known)
	c.Do("POST", "/auth/password-reset", map[string]any{"email": "ghost@acme.io"}).MustStatus(t, 200).JSON(t, &unknown)
	if known["status"] != "email_unavailable" || known["command"] != "reputecast user reset-password --email aisha@acme.io" {
		t.Fatalf("no System provider: %v", known)
	}
	if unknown["status"] != known["status"] {
		t.Fatalf("the answer must not reveal whether an account exists: %v vs %v", known, unknown)
	}
}

func TestPasswordResetByEmail(t *testing.T) {
	env := apptest.New(t)
	u := env.CreateUser(apptest.UserOpts{Email: "aisha@acme.io", Password: "old passphrase 123"})
	mail := &fakeMail{available: true}
	menv := withMail(t, env, mail)
	c := menv.Client()

	// A signed-in session elsewhere is signed out by the reset.
	other := env.Client()
	other.SignIn("aisha@acme.io", "old passphrase 123")

	var res map[string]any
	c.Do("POST", "/auth/password-reset", map[string]any{"email": "aisha@acme.io"}).MustStatus(t, 200).JSON(t, &res)
	if res["status"] != "sent" || res["expiresInMinutes"].(float64) != 15 {
		t.Fatalf("reset request: %v", res)
	}
	msg := mail.last(t)
	if msg.To != "aisha@acme.io" || !strings.Contains(msg.Text, "/reset-password/") {
		t.Fatalf("reset email: %+v", msg)
	}
	token := linkToken(t, msg.Text, "/reset-password/")
	// Unknown emails get the same answer and no email.
	var ghost map[string]any
	c.Do("POST", "/auth/password-reset", map[string]any{"email": "ghost@acme.io"}).MustStatus(t, 200).JSON(t, &ghost)
	if ghost["status"] != "sent" || len(mail.sent) != 1 {
		t.Fatalf("unknown email: %v (%d emails)", ghost, len(mail.sent))
	}

	var info map[string]any
	c.Do("GET", "/auth/password-reset/"+token, nil).MustStatus(t, 200).JSON(t, &info)
	if info["email"] != "aisha@acme.io" || info["minLength"].(float64) != 12 {
		t.Fatalf("reset info: %v", info)
	}
	c.Do("POST", "/auth/password-reset/"+token, map[string]any{"password": "short"}).MustStatus(t, 422)
	var done map[string]any
	c.Do("POST", "/auth/password-reset/"+token, map[string]any{"password": "a brand new passphrase"}).MustStatus(t, 200).JSON(t, &done)
	if done["email"] != "aisha@acme.io" {
		t.Fatalf("reset: %v", done)
	}
	// One use only; the old password no longer works, the new one does.
	p := c.Do("POST", "/auth/password-reset/"+token, map[string]any{"password": "another new passphrase"}).MustStatus(t, 410).Problem(t)
	if p["code"] != "reset_expired" {
		t.Fatalf("used link: %v", p)
	}
	c.Do("GET", "/auth/password-reset/"+token, nil).MustStatus(t, 410)
	other.Do("GET", "/me", nil).MustStatus(t, 401)
	env.Client().Do("POST", "/auth/password", map[string]any{"email": "aisha@acme.io", "password": "old passphrase 123"}).MustStatus(t, 401)
	env.Client().SignIn("aisha@acme.io", "a brand new passphrase")

	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action IN ('account.password_reset_request','account.password_reset') AND target_id = $1`, u.ID.String()).Scan(&n); err != nil || n != 2 {
		t.Fatalf("audit entries: %d %v", n, err)
	}
}

func TestPasswordResetExpiresAndSSOAccounts(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	u := env.CreateUser(apptest.UserOpts{Email: "aisha@acme.io"})
	c := env.Client()
	// A link older than 15 minutes is refused.
	token, _, err := accounts.CreateResetToken(ctx, env.App.Q, u.ID, "cli", time.Now().Add(-20*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if p := c.Do("GET", "/auth/password-reset/"+token, nil).MustStatus(t, 410).Problem(t); p["code"] != "reset_expired" {
		t.Fatalf("expired link: %v", p)
	}

	// acme.io signs in with Google Workspace: passwords are managed there,
	// except for break-glass Owners, who keep one.
	sso := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO sso_providers (id, preset, name, status, enforce_domains, allowed_domains) VALUES ($1, 'google', 'Google Workspace', 'enabled', ARRAY['acme.io'], ARRAY['acme.io'])`, sso); err != nil {
		t.Fatal(err)
	}
	var res map[string]any
	c.Do("POST", "/auth/password-reset", map[string]any{"email": "sofia@acme.io"}).MustStatus(t, 200).JSON(t, &res)
	if res["status"] != "sso" || res["sso"].(map[string]any)["name"] != "Google Workspace" {
		t.Fatalf("SSO domain: %v", res)
	}
	owner := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", InstanceAdmin: true})
	if _, err := env.App.Pool.Exec(ctx, `UPDATE users SET break_glass = true WHERE id = $1`, owner.ID); err != nil {
		t.Fatal(err)
	}
	c.Do("POST", "/auth/password-reset", map[string]any{"email": "linh@acme.io"}).MustStatus(t, 200).JSON(t, &res)
	if res["status"] != "email_unavailable" {
		t.Fatalf("break-glass Owner keeps a password to reset: %v", res)
	}

	// An SSO-only account outside an enforced domain gets the answer any
	// email gets, so the answer doesn't reveal it; the email says how it
	// signs in.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE sso_providers SET enforce_domains = '{}'`); err != nil {
		t.Fatal(err)
	}
	marcus := env.CreateUser(apptest.UserOpts{Email: "marcus@acme.io"})
	if _, err := env.App.Pool.Exec(ctx, `UPDATE users SET password_hash = NULL WHERE id = $1`, marcus.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO user_identities (id, user_id, kind, sso_provider_id, subject, email) VALUES ($1, $2, 'sso', $3, 'g-123', 'marcus@acme.io')`, uuid.Must(uuid.NewV7()), marcus.ID, sso); err != nil {
		t.Fatal(err)
	}
	mail := &fakeMail{available: true}
	mc := withMail(t, env, mail).Client()
	var unknown, ssoOnly map[string]any
	mc.Do("POST", "/auth/password-reset", map[string]any{"email": "nobody@acme.io"}).MustStatus(t, 200).JSON(t, &unknown)
	mc.Do("POST", "/auth/password-reset", map[string]any{"email": "marcus@acme.io"}).MustStatus(t, 200).JSON(t, &ssoOnly)
	if ssoOnly["status"] != "sent" || unknown["status"] != "sent" || ssoOnly["sso"] != nil {
		t.Fatalf("SSO-only account: %v, unknown email: %v", ssoOnly, unknown)
	}
	if msg := mail.last(t); msg.To != "marcus@acme.io" || !strings.Contains(msg.Text, "signs in with Google Workspace") || strings.Contains(msg.Text, "/reset-password/") {
		t.Fatalf("reminder: %+v", msg)
	}
	// Like reset emails, one reminder a minute at most; each is audited.
	sent := len(mail.sent)
	mc.Do("POST", "/auth/password-reset", map[string]any{"email": "marcus@acme.io"}).MustStatus(t, 200)
	if len(mail.sent) != sent {
		t.Fatalf("a second reminder within the minute: %d emails", len(mail.sent))
	}
	var n int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'account.sso_reminder' AND target_id = $1`, marcus.ID.String()).Scan(&n); err != nil || n != 1 {
		t.Fatalf("reminder audit entries: %d %v", n, err)
	}
}
