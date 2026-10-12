package roles_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/store"
)

const ws = "/workspaces/acme-newsletter"

type roleList struct {
	Roles []struct {
		ID, Key, Name string
		Builtin       bool
		Permissions   []string
		Members       []struct{ Name, Status string }
		Active        int
		Deactivated   int
		Invited       int
		CreatedBy     string
	}
	Permissions []struct{ Key, Group string }
	Held        []string
	CanManage   bool
}

func role(t *testing.T, l roleList, key string) string {
	t.Helper()
	for _, r := range l.Roles {
		if r.Key == key {
			return r.ID
		}
	}
	t.Fatalf("no role %s", key)
	return ""
}

func TestRolesListAndEditor(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var l roleList
	c.Do("GET", ws+"/roles", nil).MustStatus(t, 200).JSON(t, &l)
	if len(l.Roles) != 5 || !l.CanManage || len(l.Permissions) < 35 {
		t.Fatalf("roles: %d manage=%v perms=%d", len(l.Roles), l.CanManage, len(l.Permissions))
	}
	for _, r := range l.Roles {
		switch r.Key {
		case "admin":
			if r.Active != 1 || r.Deactivated != 1 {
				t.Fatalf("admin members: %+v", r)
			}
		case "editor":
			if r.Active != 1 || r.Invited != 1 {
				t.Fatalf("editor members: %+v", r)
			}
		case "content-writer":
			if r.Builtin || r.CreatedBy != "Linh Nguyen" || len(r.Members) != 1 || r.Members[0].Name != "Daniel Park" {
				t.Fatalf("content writer: %+v", r)
			}
		}
	}

	// Built-in names are taken; built-in roles are read-only.
	c.Do("POST", ws+"/roles", map[string]any{"name": "Editor", "permissions": []string{"campaigns:view"}}).MustStatus(t, 422)
	c.Do("PUT", ws+"/roles/"+role(t, l, "editor"), map[string]any{"name": "Editor", "permissions": []string{"campaigns:view"}}).MustStatus(t, 409)
	c.Do("POST", ws+"/roles", map[string]any{"name": "Owner-ish", "permissions": []string{"workspace:delete"}}).MustStatus(t, 422)

	var created struct{ ID string }
	c.Do("POST", ws+"/roles", map[string]any{"name": "Support", "description": "Answers subscribers", "permissions": []string{"subscribers:edit"}}).MustStatus(t, 201).JSON(t, &created)
	var detail struct {
		Role struct {
			Name        string
			Permissions []string
		}
	}
	c.Do("GET", ws+"/roles/"+created.ID, nil).MustStatus(t, 200).JSON(t, &detail)
	if detail.Role.Name != "Support" || strings.Join(detail.Role.Permissions, ",") != "subscribers:view,subscribers:edit" {
		t.Fatalf("view is included by edit: %+v", detail.Role)
	}

	// Impact preview: adding campaigns:send to Content Writer reaches Daniel.
	var imp struct {
		Added, AddedSensitive []string
		Members               []struct{ Name, Effect string }
		MembersAffected       int
		Sensitive, Approvals  string
		Summary               string
	}
	cw := role(t, l, "content-writer")
	perms := []string{"campaigns:view", "campaigns:edit", "campaigns:send", "templates:view", "templates:edit", "automations:view", "subscribers:view", "reports:view", "deliverability:view", "providers:view", "team:view"}
	c.Do("POST", ws+"/roles/impact", map[string]any{"roleId": cw, "permissions": perms}).MustStatus(t, 200).JSON(t, &imp)
	if strings.Join(imp.AddedSensitive, ",") != "campaigns:send" || imp.MembersAffected != 1 || imp.Members[0].Name != "Daniel Park" || !strings.Contains(imp.Members[0].Effect, "IdP") {
		t.Fatalf("impact: %+v", imp)
	}
	// Removing edit from a role with holders explains who loses what.
	c.Do("POST", ws+"/roles/impact", map[string]any{"roleId": cw, "permissions": []string{"campaigns:view"}}).MustStatus(t, 200).JSON(t, &imp)
	if !strings.HasPrefix(imp.Summary, "1 member will lose") {
		t.Fatalf("summary: %q", imp.Summary)
	}
	// A new role with edit but no send can request approval.
	c.Do("POST", ws+"/roles/impact", map[string]any{"permissions": []string{"campaigns:edit"}}).MustStatus(t, 200).JSON(t, &imp)
	if !strings.Contains(imp.Approvals, "request approval") || imp.Sensitive != "None selected · two-factor not required" {
		t.Fatalf("new role impact: %+v", imp)
	}

	c.Do("PUT", ws+"/roles/"+cw, map[string]any{"name": "Content Writer", "permissions": perms}).MustStatus(t, 204)
	var n int
	_ = env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action IN ('role.created','role.updated')`).Scan(&n)
	if n != 2 {
		t.Fatalf("audit entries: %d", n)
	}

	// Can't delete a role somebody holds; can delete one nobody holds.
	r := c.Do("DELETE", ws+"/roles/"+cw, nil).MustStatus(t, 409)
	if !strings.Contains(r.Problem(t)["detail"].(string), "Move Daniel Park to another role first") {
		t.Fatalf("delete held: %s", r.Body)
	}
	c.Do("DELETE", ws+"/roles/"+created.ID, nil).MustStatus(t, 204)
}

func TestEscalationGuard(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	c.Do("POST", ws+"/roles", map[string]any{"name": "Team lead", "permissions": []string{"team:manage", "campaigns:edit"}}).MustStatus(t, 201)
	ctx := context.Background()
	w, _ := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	lead, err := env.App.Q.GetRoleByKey(ctx, store.GetRoleByKeyParams{WorkspaceID: w.ID, Key: "team-lead"})
	if err != nil {
		t.Fatal(err)
	}
	u := env.CreateUser(apptest.UserOpts{Email: "lead@acme.io", Name: "Lee Lead"})
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: w.ID, UserID: u.ID, RoleID: lead.ID, Scope: json.RawMessage(`{}`), Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	lc := env.Client()
	lc.SignInDemo("lead@acme.io")
	r := lc.Do("POST", ws+"/roles", map[string]any{"name": "Analysts", "permissions": []string{"segments:sql", "team:view"}}).MustStatus(t, 403)
	p := r.Problem(t)
	if p["code"] != "escalation_denied" || p["permission"] != "segments:sql" || !strings.Contains(p["detail"].(string), "Ask Linh Nguyen") {
		t.Fatalf("escalation: %v", p)
	}
	lc.Do("POST", ws+"/roles", map[string]any{"name": "Writers", "permissions": []string{"campaigns:edit"}}).MustStatus(t, 201)
	var imp struct {
		Locked []struct{ Key, Reason string }
	}
	lc.Do("POST", ws+"/roles/impact", map[string]any{"permissions": []string{}}).MustStatus(t, 200).JSON(t, &imp)
	found := false
	for _, l := range imp.Locked {
		if l.Key == "providers:manage" && strings.HasPrefix(l.Reason, "You can't grant providers:manage") {
			found = true
		}
	}
	if !found {
		t.Fatalf("locked: %+v", imp.Locked)
	}
	// Viewers can read roles but not change them.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", ws+"/roles", nil).MustStatus(t, 200)
	v.Do("POST", ws+"/roles", map[string]any{"name": "X", "permissions": []string{"campaigns:view"}}).MustStatus(t, 403)
}
