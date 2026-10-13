package accessrequests_test

import (
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/accessrequests"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
)

const ws = "/workspaces/acme-newsletter"

type suggestion struct {
	Permission  string   `json:"permission"`
	Action      string   `json:"action"`
	RoleKey     string   `json:"roleKey"`
	RoleName    string   `json:"roleName"`
	YourRole    string   `json:"yourRole"`
	Owners      []string `json:"owners"`
	Managers    []string `json:"managers"`
	RequestedAt *string  `json:"requestedAt"`
}

func TestSmallestRole(t *testing.T) {
	for perm, want := range map[authz.Permission]string{
		authz.CampaignsView:      "viewer",
		authz.CampaignsSend:      "editor",
		authz.ProvidersManage:    "admin",
		authz.TeamManage:         "admin",
		authz.SegmentsSQL:        "admin",
		authz.SubscribersEdit:    "editor",
		authz.AutomationsPublish: "editor",
	} {
		if got := accessrequests.SmallestRole(perm).Key; got != want {
			t.Errorf("%s: %s, want %s", perm, got, want)
		}
	}
}

func TestViewerAsksForAccess(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("aisha@acme.io")

	var s suggestion
	c.Do("GET", ws+"/access-requests/suggestion?permission=providers:manage", nil).MustStatus(t, 200).JSON(t, &s)
	if s.RoleName != "Admin" || s.YourRole != "Viewer" || s.Action != "Managing providers" || s.RequestedAt != nil {
		t.Fatalf("suggestion: %+v", s)
	}
	if len(s.Owners) != 1 || s.Owners[0] != "Linh Nguyen" || len(s.Managers) < 2 {
		t.Fatalf("people: owners %v managers %v", s.Owners, s.Managers)
	}
	c.Do("GET", ws+"/access-requests/suggestion?permission=campaigns:send", nil).MustStatus(t, 200).JSON(t, &s)
	if s.RoleName != "Editor" || s.Action != "Sending" {
		t.Fatalf("suggestion: %+v", s)
	}
	c.Do("GET", ws+"/access-requests/suggestion?permission=nope:nope", nil).MustStatus(t, 422)

	var res struct {
		RoleName string   `json:"roleName"`
		Notified []string `json:"notified"`
	}
	c.Do("POST", ws+"/access-requests", map[string]any{"permission": "providers:manage", "note": "I cover the routing review on Fridays."}).MustStatus(t, 202).JSON(t, &res)
	if res.RoleName != "Admin" || len(res.Notified) < 2 {
		t.Fatalf("request: %+v", res)
	}

	// The team's managers got a notification with the note; Aisha didn't.
	var n int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications n JOIN users u ON u.id = n.user_id
		WHERE n.kind = 'access_requested' AND n.body = 'I cover the routing review on Fridays.' AND n.title = 'Aisha Bello asks for Admin access'
		  AND u.email IN ('linh@acme.io','marcus@acme.io')`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("notifications to managers: %d", n)
	}
	var self int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications n JOIN users u ON u.id = n.user_id WHERE n.kind = 'access_requested' AND u.email = 'aisha@acme.io'`).Scan(&self)
	if self != 0 {
		t.Fatal("the requester must not be notified")
	}
	// Audited.
	var target string
	if err := env.App.Pool.QueryRow(ctx, `SELECT target_id FROM audit_events WHERE action = 'access.request' ORDER BY occurred_at DESC LIMIT 1`).Scan(&target); err != nil || target != "admin" {
		t.Fatalf("audit: %q %v", target, err)
	}

	// One request per role a day, whichever permission led to it.
	c.Do("POST", ws+"/access-requests", map[string]any{"permission": "routing:manage"}).MustStatus(t, 422)
	dup := c.Do("POST", ws+"/access-requests", map[string]any{"permission": "domains:manage"}).MustStatus(t, 409).Problem(t)
	if meta, _ := dup["meta"].(map[string]any); meta == nil || meta["requestedAt"] == nil {
		t.Fatalf("duplicate: %v", dup)
	}
	c.Do("GET", ws+"/access-requests/suggestion?permission=providers:manage", nil).MustStatus(t, 200).JSON(t, &s)
	if s.RequestedAt == nil {
		t.Fatal("suggestion should say when you asked")
	}
}

func TestRequestNotNeededOrNotAllowed(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()

	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	owner.Do("POST", ws+"/access-requests", map[string]any{"permission": "providers:manage"}).MustStatus(t, 409)

	env.Client().Do("POST", ws+"/access-requests", map[string]any{"permission": "providers:manage"}).MustStatus(t, 401)
}
