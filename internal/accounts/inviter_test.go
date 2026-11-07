package accounts_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// An invitation lasts only as long as the person who sent it could still
// send it: an active member whose role holds the invited role's
// permissions, with a scope at least as broad. Otherwise it answers like a
// revoked invitation, whichever way the invitee joins.
func TestInvitationNeedsAnInviterWhoCouldStillSendIt(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	linh := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	ws := env.CreateWorkspace("Acme Newsletter", linh.ID)
	marcus := env.CreateUser(apptest.UserOpts{Email: "marcus@acme.io", Name: "Marcus Chen"})
	env.AddMember(ws, marcus.ID, "admin")
	token, _ := invite(t, env, ws, marcus.ID, "jo@contractor.dev", "admin", `{}`, 24*time.Hour)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	setRole := func(key string) {
		exec(`UPDATE workspace_members SET role_id = (SELECT id FROM roles WHERE workspace_id = $1 AND key = $2) WHERE workspace_id = $1 AND user_id = $3`, ws.ID, key, marcus.ID)
	}
	refused := func(why string, body map[string]any) {
		t.Helper()
		c := env.Client()
		var v map[string]any
		c.Do("GET", "/invitations/"+token, nil).MustStatus(t, 200).JSON(t, &v)
		if v["status"] != "revoked" {
			t.Fatalf("%s: the invitation reads %v", why, v["status"])
		}
		if p := c.Do("POST", "/invitations/"+token+"/accept", body).MustStatus(t, 410).Problem(t); p["code"] != "invite_revoked" {
			t.Fatalf("%s: %v", why, p)
		}
	}
	join := map[string]any{"name": "Jo Lindqvist", "password": "a long enough passphrase"}

	setRole("editor")
	refused("demoted below the invited role", join)
	refused("demoted, joining with a passkey", map[string]any{"name": "Jo Lindqvist", "passkey": true})
	setRole("admin")
	exec(`UPDATE workspace_members SET status = 'deactivated' WHERE workspace_id = $1 AND user_id = $2`, ws.ID, marcus.ID)
	refused("deactivated in the workspace", join)
	exec(`UPDATE workspace_members SET status = 'active' WHERE workspace_id = $1 AND user_id = $2`, ws.ID, marcus.ID)
	exec(`UPDATE users SET status = 'deactivated' WHERE id = $1`, marcus.ID)
	refused("account deactivated", join)
	exec(`UPDATE users SET status = 'active' WHERE id = $1`, marcus.ID)
	exec(`UPDATE workspace_members SET scope = jsonb_build_object('lists', jsonb_build_array($3::text)) WHERE workspace_id = $1 AND user_id = $2`, ws.ID, marcus.ID, uuid.Must(uuid.NewV7()).String())
	refused("scoped to one list", join)
	if countRows(t, env, `SELECT count(*) FROM users WHERE email = 'jo@contractor.dev'`) != 0 {
		t.Fatal("a refused invitation created an account")
	}

	// With his authority back, the same link works.
	exec(`UPDATE workspace_members SET scope = '{}' WHERE workspace_id = $1 AND user_id = $2`, ws.ID, marcus.ID)
	env.Client().Do("POST", "/invitations/"+token+"/accept", join).MustStatus(t, 200)
}
