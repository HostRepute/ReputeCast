package website_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

// A member limited to some sender domains makes a campaign of a web-only
// post that starts from a sender of theirs, as their new drafts do, not
// from the workspace's default on a domain they can't send from.
func TestSendAsCampaignStartsFromTheMembersSender(t *testing.T) {
	env, c := demoSite(t)
	turnOn(t, env, c)
	ctx := context.Background()
	var wsID, editor uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT w.id, r.id FROM workspaces w
		JOIN roles r ON r.workspace_id = w.id AND r.key = 'editor'
		WHERE w.slug = 'acme-newsletter'`).Scan(&wsID, &editor); err != nil {
		t.Fatal(err)
	}
	// Kim may send from acme.io only (Marcus's sender), not news.acme.io.
	u := env.CreateUser(apptest.UserOpts{Email: "kim@contractor.dev", Name: "Kim Contractor"})
	scope, _ := json.Marshal(authz.Scope{SenderDomains: []string{"acme.io"}})
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, UserID: u.ID, RoleID: editor, Scope: scope, Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	kim := env.Client()
	kim.SignInDemo("kim@contractor.dev")

	var p website.WebsitePost
	c.Do("POST", ws+"/website/posts", map[string]any{"title": "A post for Kim"}).MustStatus(t, 201).JSON(t, &p)
	var made website.WebsiteSentAsCampaign
	kim.Do("POST", ws+"/website/posts/"+p.ID+"/send-as-campaign", nil).MustStatus(t, 201).JSON(t, &made)
	var email string
	if err := env.App.Pool.QueryRow(ctx, `SELECT s.email FROM campaigns c JOIN senders s ON s.id = c.sender_id WHERE c.id = $1`, made.CampaignID).Scan(&email); err != nil {
		t.Fatal(err)
	}
	if email != "marcus@acme.io" {
		t.Fatalf("Kim's draft starts from %s", email)
	}
}
