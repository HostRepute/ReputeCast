package sessionext_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/store"
)

type timing struct {
	IdleTimeoutSeconds int64      `json:"idleTimeoutSeconds"`
	LastSeenAt         time.Time  `json:"lastSeenAt"`
	IdleExpiresAt      *time.Time `json:"idleExpiresAt"`
	ExpiresAt          time.Time  `json:"expiresAt"`
	EndsAt             time.Time  `json:"endsAt"`
	EndsBy             string     `json:"endsBy"`
	Extendable         bool       `json:"extendable"`
	ExtendsTo          *time.Time `json:"extendsTo"`
	ServerTime         time.Time  `json:"serverTime"`
}

func near(t *testing.T, what string, got, want time.Time) {
	t.Helper()
	if d := got.Sub(want); d > 5*time.Second || d < -5*time.Second {
		t.Fatalf("%s = %s, want about %s", what, got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

// signedIn returns a client and its user with a session that lives 14 days,
// like a real sign-in (the demo helper's sessions last an hour).
func signedIn(t *testing.T, env *apptest.Env, email string) (*apptest.Client, uuid.UUID) {
	t.Helper()
	c := env.Client()
	c.SignInDemo(email)
	u, err := env.App.Q.GetUserByEmail(context.Background(), email)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE sessions SET expires_at = now() + interval '14 days' WHERE user_id = $1`, u.ID); err != nil {
		t.Fatal(err)
	}
	return c, u.ID
}

func TestSessionTimingAndExtend(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c, userID := signedIn(t, env, "linh@acme.io")

	// A fresh session ends by the idle timeout (12 hours by default).
	var got timing
	c.Do("GET", "/me/session", nil).MustStatus(t, 200).JSON(t, &got)
	now := time.Now()
	if got.IdleTimeoutSeconds != 12*3600 || got.EndsBy != "idle" {
		t.Fatalf("timing: %+v", got)
	}
	near(t, "endsAt", got.EndsAt, now.Add(12*time.Hour))
	near(t, "expiresAt", got.ExpiresAt, now.Add(14*24*time.Hour))
	if got.Extendable {
		t.Fatal("extending a session that was just used changes nothing, so it isn't offered")
	}

	// 11 h 58 min of inactivity: the warning's moment. Extending restarts the idle clock.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE sessions SET last_seen_at = now() - interval '11 hours 58 minutes' WHERE user_id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	var ext timing
	c.Do("POST", "/me/session/extend", nil).MustStatus(t, 200).JSON(t, &ext)
	near(t, "lastSeenAt after extend", ext.LastSeenAt, time.Now())
	near(t, "endsAt after extend", ext.EndsAt, time.Now().Add(12*time.Hour))
	near(t, "expiresAt never moves", ext.ExpiresAt, got.ExpiresAt)

	// The extension is in the audit log, with the person as the actor.
	var action, actorID, targetType string
	if err := env.App.Pool.QueryRow(ctx, `SELECT action, actor_id::text, target_type FROM audit_events WHERE action = 'session.extend' ORDER BY occurred_at DESC LIMIT 1`).Scan(&action, &actorID, &targetType); err != nil {
		t.Fatalf("audit entry: %v", err)
	}
	if actorID != userID.String() || targetType != "session" {
		t.Fatalf("audit entry: actor %s target %s", actorID, targetType)
	}
}

func TestSessionEndsByLifetimeWhenThatComesFirst(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c, userID := signedIn(t, env, "linh@acme.io")
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE sessions SET expires_at = now() + interval '1 hour' WHERE user_id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	var got timing
	c.Do("GET", "/me/session", nil).MustStatus(t, 200).JSON(t, &got)
	if got.EndsBy != "lifetime" || got.Extendable {
		t.Fatalf("a session at the end of its lifetime can't be extended: %+v", got)
	}
	near(t, "endsAt", got.EndsAt, time.Now().Add(time.Hour))
	if got.ExtendsTo == nil || !got.ExtendsTo.Equal(got.ExpiresAt) {
		t.Fatalf("extendsTo: %v", got.ExtendsTo)
	}
	// Extending still restarts the idle clock but never moves the lifetime.
	var ext timing
	c.Do("POST", "/me/session/extend", nil).MustStatus(t, 200).JSON(t, &ext)
	if !ext.ExpiresAt.Equal(got.ExpiresAt) || ext.EndsBy != "lifetime" {
		t.Fatalf("extend after lifetime: %+v", ext)
	}
}

func TestIdleSessionCannotBeExtended(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c, userID := signedIn(t, env, "aisha@acme.io")
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE sessions SET last_seen_at = now() - interval '13 hours' WHERE user_id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	r := c.Do("POST", "/me/session/extend", nil).MustStatus(t, 401)
	if code := r.Problem(t)["code"]; code != "session_expired" {
		t.Fatalf("code %v", code)
	}
	// The expired session is revoked: a later read is a plain 401.
	c.Do("GET", "/me/session", nil).MustStatus(t, 401)
}

func TestSessionTimingNeedsABrowserSession(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()

	env.Client().Do("GET", "/me/session", nil).MustStatus(t, 401)
	env.Client().Do("POST", "/me/session/extend", nil).MustStatus(t, 401)

	// API keys have no session to extend.
	u, err := env.App.Q.GetUserByEmail(ctx, "linh@acme.io")
	if err != nil {
		t.Fatal(err)
	}
	var wsID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID); err != nil {
		t.Fatal(err)
	}
	key, hash, _ := authn.NewAPIKey(authn.APIKeyPrefix)
	if _, err := env.App.Q.CreateAPIKey(ctx, store.CreateAPIKeyParams{
		ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, Kind: "api_key", Name: "test", Prefix: authn.APIKeyPrefix,
		TokenHash: hash, Permissions: []string{"campaigns:view"}, Scope: json.RawMessage(`{}`), CreatedBy: &u.ID,
	}); err != nil {
		t.Fatal(err)
	}
	k := env.Client()
	k.Bearer = key
	r := k.Do("POST", "/me/session/extend", nil).MustStatus(t, 403)
	if code := r.Problem(t)["code"]; code != "humans_only" {
		t.Fatalf("code %v", code)
	}
}

func TestPendingUsersCanReadTheirSession(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "new@example.com", MustChange: true})
	c := env.Client()
	c.SignIn("new@example.com", "")
	var got timing
	c.Do("GET", "/me/session", nil).MustStatus(t, 200).JSON(t, &got)
	if got.EndsAt.IsZero() {
		t.Fatalf("timing: %+v", got)
	}
}
