package accounts_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/store"
)

// invite stores an invitation the way the Team screen does and returns
// its link token.
func invite(t *testing.T, env *apptest.Env, ws store.Workspace, by uuid.UUID, email, role string, scope string, expires time.Duration) (string, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	r, err := env.App.Q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: ws.ID, Key: role})
	if err != nil {
		t.Fatal(err)
	}
	token, hash := authn.NewToken()
	now := time.Now()
	inv, err := env.App.Q.CreateInvitation(ctx, store.CreateInvitationParams{
		ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Email: email, RoleID: r.ID, Scope: json.RawMessage(scope), TokenHash: hash,
		InvitedBy: &by, LastSentAt: &now, ExpiresAt: now.Add(expires),
	})
	if err != nil {
		t.Fatal(err)
	}
	return token, inv.ID
}

func TestAcceptInvitationAsNewMember(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	linh := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	ws := env.CreateWorkspace("Acme Newsletter", linh.ID)
	beta := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO lists (id, workspace_id, name) VALUES ($1, $2, 'Beta Program')`, beta, ws.ID); err != nil {
		t.Fatal(err)
	}
	token, _ := invite(t, env, ws, linh.ID, "jo@contractor.dev", "editor", `{"lists":["`+beta.String()+`"]}`, 5*24*time.Hour)

	c := env.Client()
	var v map[string]any
	c.Do("GET", "/invitations/"+token, nil).MustStatus(t, 200).JSON(t, &v)
	if v["status"] != "pending" || v["inviterName"] != "Linh Nguyen" || v["workspaceName"] != "Acme Newsletter" || v["roleName"] != "Editor" || v["scope"] != "Beta Program list only" {
		t.Fatalf("invitation: %v", v)
	}
	if !strings.Contains(v["description"].(string), "create, edit and send campaigns for the Beta Program list") ||
		!strings.Contains(v["description"].(string), "Sends over 10,000 recipients need a second person's approval.") {
		t.Fatalf("description: %v", v["description"])
	}
	tf := v["twoFactor"].(map[string]any)
	if tf["required"] != true || tf["graceDays"].(float64) != 7 || !strings.Contains(tf["reason"].(string), "send campaigns") {
		t.Fatalf("two-factor: %v", tf)
	}
	if v["accountExists"] != false || v["passwordAllowed"] != true {
		t.Fatalf("new member: %v", v)
	}

	c.Do("POST", "/invitations/"+token+"/accept", map[string]any{"name": "Jo Lindqvist", "password": "short"}).MustStatus(t, 422)
	var res map[string]any
	r := c.Do("POST", "/invitations/"+token+"/accept", map[string]any{"name": "Jo Lindqvist", "password": "a long enough passphrase"}).MustStatus(t, 200)
	r.JSON(t, &res)
	if res["newAccount"] != true || res["workspace"] != ws.Slug || res["hasSecondFactor"] != false || res["twoFactor"].(map[string]any)["graceUntil"] == nil {
		t.Fatalf("accepted: %v", res)
	}
	if !strings.Contains(r.Header.Get("Set-Cookie"), "HttpOnly") {
		t.Fatal("joining signs the new member in")
	}
	var wme map[string]any
	c.Do("GET", "/workspaces/"+ws.Slug+"/me", nil).MustStatus(t, 200).JSON(t, &wme)
	if wme["workspace"].(map[string]any)["roleKey"] != "editor" || len(wme["scope"].(map[string]any)["lists"].([]any)) != 1 {
		t.Fatalf("membership: %v", wme)
	}
	// The link works once.
	if p := env.Client().Do("POST", "/invitations/"+token+"/accept", map[string]any{"name": "X", "password": "a long enough passphrase"}).MustStatus(t, 410).Problem(t); p["code"] != "invite_accepted" {
		t.Fatalf("used invite: %v", p)
	}
	// The inviter is notified, and the join is audited.
	var title, link string
	if err := env.App.Pool.QueryRow(ctx, `SELECT title, link FROM notifications WHERE user_id = $1 AND kind = 'invite_accepted'`, linh.ID).Scan(&title, &link); err != nil {
		t.Fatalf("inviter notification: %v", err)
	}
	if title != "Invite accepted" || link != "/w/"+ws.Slug+"/settings/team" {
		t.Fatalf("notification: %s %s", title, link)
	}
	var n int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'invitation.accept' AND workspace_id = $1`, ws.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("audit: %d", n)
	}
}

func TestInvitationStates(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	linh := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	ws := env.CreateWorkspace("Acme Newsletter", linh.ID)
	aisha := env.CreateUser(apptest.UserOpts{Email: "aisha@acme.io", Name: "Aisha Bello"})
	_ = aisha

	// Expired: shown, can't be accepted, the invitee can ask for a new one once a day.
	expired, _ := invite(t, env, ws, linh.ID, "jo@contractor.dev", "editor", `{}`, -time.Hour)
	c := env.Client()
	var v map[string]any
	c.Do("GET", "/invitations/"+expired, nil).MustStatus(t, 200).JSON(t, &v)
	if v["status"] != "expired" {
		t.Fatalf("expired: %v", v)
	}
	if p := c.Do("POST", "/invitations/"+expired+"/accept", map[string]any{"name": "Jo", "password": "a long enough passphrase"}).MustStatus(t, 410).Problem(t); p["code"] != "invite_expired" {
		t.Fatalf("accept expired: %v", p)
	}
	var req map[string]any
	c.Do("POST", "/invitations/"+expired+"/request", nil).MustStatus(t, 200).JSON(t, &req)
	if req["status"] != "sent" || req["inviter"] != "Linh Nguyen" {
		t.Fatalf("request: %v", req)
	}
	c.Do("POST", "/invitations/"+expired+"/request", nil).MustStatus(t, 200).JSON(t, &req)
	if req["status"] != "already_sent" {
		t.Fatalf("second request the same day: %v", req)
	}
	var asked int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND kind = 'invite_requested'`, linh.ID).Scan(&asked)
	if asked != 1 {
		t.Fatalf("inviter asked %d times", asked)
	}

	// Revoked.
	revoked, revokedID := invite(t, env, ws, linh.ID, "kim@example.com", "viewer", `{}`, 24*time.Hour)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE invitations SET revoked_at = now() WHERE id = $1`, revokedID); err != nil {
		t.Fatal(err)
	}
	c.Do("GET", "/invitations/"+revoked, nil).MustStatus(t, 200).JSON(t, &v)
	if v["status"] != "revoked" || v["revokedAt"] == nil {
		t.Fatalf("revoked: %v", v)
	}

	// Signed in as a different email: refused, and the person can ask for
	// an invitation of their own.
	pending, _ := invite(t, env, ws, linh.ID, "jo@contractor.dev", "editor", `{}`, 24*time.Hour)
	other := env.Client()
	other.SignIn("aisha@acme.io", "")
	other.Do("GET", "/invitations/"+pending, nil).MustStatus(t, 200).JSON(t, &v)
	if v["signedInAs"] != "aisha@acme.io" || v["email"] != "jo@contractor.dev" {
		t.Fatalf("different account: %v", v)
	}
	if p := other.Do("POST", "/invitations/"+pending+"/accept", map[string]any{}).MustStatus(t, 409).Problem(t); p["code"] != "invite_wrong_account" {
		t.Fatalf("wrong account: %v", p)
	}
	other.Do("POST", "/invitations/"+pending+"/request", nil).MustStatus(t, 200).JSON(t, &req)
	if req["status"] != "sent" {
		t.Fatalf("ask for own invite: %v", req)
	}

	// An existing account accepts while signed in.
	forAisha, _ := invite(t, env, ws, linh.ID, "aisha@acme.io", "viewer", `{}`, 24*time.Hour)
	if p := env.Client().Do("POST", "/invitations/"+forAisha+"/accept", map[string]any{"name": "Aisha", "password": "a long enough passphrase"}).MustStatus(t, 409).Problem(t); p["code"] != "invite_sign_in" {
		t.Fatalf("existing account must sign in: %v", p)
	}
	var res map[string]any
	other.Do("POST", "/invitations/"+forAisha+"/accept", map[string]any{}).MustStatus(t, 200).JSON(t, &res)
	if res["newAccount"] != false || res["workspace"] != ws.Slug {
		t.Fatalf("accept signed in: %v", res)
	}

	// Unknown links say so.
	env.Client().Do("GET", "/invitations/not-a-real-token-at-all", nil).MustStatus(t, 404)
}
