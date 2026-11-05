package team_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Sessions belong to the instance, not to one workspace. A workspace's
// admin ends them only for someone who uses no other workspace and doesn't
// administer the instance; for anyone else only an instance admin can, and
// deactivating them here closes this workspace at once anyway.
func TestWorkspaceSignOutsStayInTheWorkspace(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	linh := env.Client() // Owner and instance admin
	linh.SignInDemo("linh@acme.io")
	marcus := env.Client() // Admin, not an instance admin
	marcus.SignInDemo("marcus@acme.io")
	var l memberList
	linh.Do("GET", ws+"/members", nil).MustStatus(t, 200).JSON(t, &l)

	// Aisha also belongs to the engineering blog.
	eng, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-engineering-blog")
	if err != nil {
		t.Fatal(err)
	}
	u, err := env.App.Q.GetUserByEmail(ctx, "aisha@acme.io")
	if err != nil {
		t.Fatal(err)
	}
	env.AddMember(eng, u.ID, "viewer")
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	am := byEmail(t, l, "aisha@acme.io")

	type detail struct {
		SessionsShared bool   `json:"sessionsShared"`
		SignOutReason  string `json:"signOutReason"`
	}
	var d detail
	marcus.Do("GET", ws+"/members/"+am.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if !d.SessionsShared || !strings.Contains(d.SignOutReason, "only an instance admin") {
		t.Fatalf("member detail: %+v", d)
	}
	if p := marcus.Do("POST", ws+"/members/"+am.ID+"/sign-out", nil).MustStatus(t, 403).Problem(t); p["code"] != "permission_denied" {
		t.Fatalf("sign-out from the workspace: %v", p)
	}
	aisha.Do("GET", "/me", nil).MustStatus(t, 200)

	// Deactivating her here closes this workspace at once; her other
	// workspace keeps working.
	var res struct {
		SessionsRevoked int64 `json:"sessionsRevoked"`
	}
	marcus.Do("POST", ws+"/members/"+am.ID+"/deactivate", map[string]any{}).MustStatus(t, 200).JSON(t, &res)
	if res.SessionsRevoked != 0 {
		t.Fatalf("deactivation ended %d sessions", res.SessionsRevoked)
	}
	aisha.Do("GET", ws+"/me", nil).MustStatus(t, 404)
	aisha.Do("GET", "/workspaces/acme-engineering-blog/me", nil).MustStatus(t, 200)

	// An instance admin can end them from here.
	linh.Do("POST", ws+"/members/"+am.ID+"/reactivate", nil).MustStatus(t, 204)
	var ld detail
	linh.Do("GET", ws+"/members/"+am.ID, nil).MustStatus(t, 200).JSON(t, &ld)
	if !ld.SessionsShared || ld.SignOutReason != "" {
		t.Fatalf("instance admin: %+v", ld)
	}
	linh.Do("POST", ws+"/members/"+am.ID+"/sign-out", nil).MustStatus(t, 200).JSON(t, &res)
	if res.SessionsRevoked == 0 {
		t.Fatal("the instance admin's sign-out ended nothing")
	}
	aisha.Do("GET", "/me", nil).MustStatus(t, 401)

	// An instance admin who is only a member here keeps their sessions too.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE users SET is_instance_admin = true WHERE email = 'daniel@acme.io'`); err != nil {
		t.Fatal(err)
	}
	if p := marcus.Do("POST", ws+"/members/"+byEmail(t, l, "daniel@acme.io").ID+"/sign-out", nil).MustStatus(t, 403).Problem(t); !strings.Contains(p["detail"].(string), "instance admin") {
		t.Fatalf("instance admin target: %v", p)
	}

	// Someone who only uses this workspace is signed out as before.
	sofia := env.Client()
	sofia.SignInDemo("sofia@acme.io")
	marcus.Do("POST", ws+"/members/"+byEmail(t, l, "sofia@acme.io").ID+"/sign-out", nil).MustStatus(t, 200).JSON(t, &res)
	if res.SessionsRevoked == 0 {
		t.Fatal("nothing ended")
	}
	sofia.Do("GET", "/me", nil).MustStatus(t, 401)
}
