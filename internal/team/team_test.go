package team_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/store"
)

const ws = "/workspaces/acme-newsletter"

type member struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Status string `json:"status"`
	Role   struct {
		ID, Key, Name string
	} `json:"role"`
	Scope struct {
		Label string `json:"label"`
		All   bool   `json:"all"`
	} `json:"scope"`
	Notes  []string                    `json:"notes"`
	SignIn []access.MemberSignInMethod `json:"signIn"`
	TwoFA  struct {
		State     string `json:"state"`
		GraceDays *int   `json:"graceDays"`
	} `json:"twoFA"`
	LastOwner bool `json:"lastOwner"`
}

type memberList struct {
	Items  []member `json:"items"`
	Counts struct {
		Members, Active, Deactivated, Invited int
	} `json:"counts"`
	Roles     []struct{ ID, Key, Name string } `json:"roles"`
	CanInvite bool                             `json:"canInvite"`
	CanManage bool                             `json:"canManage"`
	Reason    string                           `json:"reason"`
}

func byEmail(t *testing.T, l memberList, email string) member {
	t.Helper()
	for _, m := range l.Items {
		if m.Email == email {
			return m
		}
	}
	t.Fatalf("no member %s in %+v", email, l.Items)
	return member{}
}

func signInLabels(ms []access.MemberSignInMethod) string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Label
	}
	return strings.Join(out, " · ")
}

func roleID(t *testing.T, l memberList, key string) string {
	t.Helper()
	for _, r := range l.Roles {
		if r.Key == key {
			return r.ID
		}
	}
	t.Fatalf("no role %s", key)
	return ""
}

func auditCount(t *testing.T, env *apptest.Env, action string) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1`, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestMembersList(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var l memberList
	c.Do("GET", ws+"/members", nil).MustStatus(t, 200).JSON(t, &l)
	if l.Counts.Members != 6 || l.Counts.Active != 5 || l.Counts.Deactivated != 1 || l.Counts.Invited != 1 {
		t.Fatalf("counts: %+v", l.Counts)
	}
	linh := byEmail(t, l, "linh@acme.io")
	if linh.Role.Key != "owner" || !linh.LastOwner || strings.Join(linh.Notes, ",") != "instance admin,break-glass" || linh.TwoFA.State != "on" {
		t.Fatalf("linh: %+v", linh)
	}
	marcus := byEmail(t, l, "marcus@acme.io")
	if marcus.TwoFA.State != "idp" {
		t.Fatalf("marcus: %+v", marcus)
	}
	// The Sign-in column shows the member's own credentials when they have
	// any, otherwise their SSO or social sign-in; the member page lists all.
	if got := signInLabels(linh.SignIn); got != "Password · passkey" {
		t.Fatalf("linh sign-in: %s", got)
	}
	if got := signInLabels(marcus.SignIn); got != "Google Workspace SSO" {
		t.Fatalf("marcus sign-in: %s", got)
	}
	for _, m := range l.Items {
		var d struct {
			SignInMethods []access.MemberSignInMethod `json:"signInMethods"`
		}
		c.Do("GET", ws+"/members/"+m.ID, nil).MustStatus(t, 200).JSON(t, &d)
		if got, want := signInLabels(m.SignIn), signInLabels(access.PrimarySignIn(d.SignInMethods)); got != want {
			t.Fatalf("%s: the list shows %q, the rule on %+v gives %q", m.Email, got, d.SignInMethods, want)
		}
	}
	aisha := byEmail(t, l, "aisha@acme.io")
	if aisha.TwoFA.State != "required" || aisha.TwoFA.GraceDays == nil || *aisha.TwoFA.GraceDays != 3 {
		t.Fatalf("aisha 2FA: %+v", aisha.TwoFA)
	}
	if tom := byEmail(t, l, "tom@acme.io"); tom.Status != "deactivated" {
		t.Fatalf("tom: %+v", tom)
	}
	if daniel := byEmail(t, l, "daniel@acme.io"); daniel.Role.Name != "Content Writer" {
		t.Fatalf("daniel: %+v", daniel)
	}
	if !l.CanInvite || !l.CanManage {
		t.Fatal("owner can invite and manage")
	}

	c.Do("GET", ws+"/members?twofa=idp", nil).MustStatus(t, 200).JSON(t, &l)
	if len(l.Items) != 3 {
		t.Fatalf("2FA filter: %d", len(l.Items))
	}
	c.Do("GET", ws+"/members?q=aisha", nil).MustStatus(t, 200).JSON(t, &l)
	if len(l.Items) != 1 {
		t.Fatalf("search: %d", len(l.Items))
	}

	// The Viewer sees the team read-only, with the reason.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", ws+"/members", nil).MustStatus(t, 200).JSON(t, &l)
	if l.CanInvite || l.CanManage || !strings.Contains(l.Reason, "team:invite") || !strings.Contains(l.Reason, "Ask Linh Nguyen or Marcus Chen") {
		t.Fatalf("viewer: %+v", l)
	}
	v.Do("GET", ws+"/invitations", nil).MustStatus(t, 200)
	v.Do("POST", ws+"/invitations", map[string]any{"emails": []string{"x@example.com"}, "roleId": roleID(t, l, "viewer")}).MustStatus(t, 403)
}

func TestInvitations(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var l memberList
	c.Do("GET", ws+"/members", nil).MustStatus(t, 200).JSON(t, &l)

	var opts struct {
		Roles []struct {
			ID, Key   string
			Grantable bool
			Hint      string
		}
		Lists     []struct{ ID, Name string }
		CanEmail  bool `json:"canEmail"`
		JIT       []struct{ Domain, Provider, RoleName string }
		SignIn    string `json:"signInSummary"`
		DefaultID string `json:"defaultRoleId"`
	}
	c.Do("GET", ws+"/invitations/options", nil).MustStatus(t, 200).JSON(t, &opts)
	var beta string
	for _, li := range opts.Lists {
		if li.Name == "Beta Program" {
			beta = li.ID
		}
	}
	if opts.CanEmail || len(opts.JIT) != 1 || opts.JIT[0].Domain != "acme.io" || opts.DefaultID != roleID(t, l, "editor") || beta == "" {
		t.Fatalf("options: %+v", opts)
	}
	for _, r := range opts.Roles {
		if r.Key == "owner" && r.Grantable {
			t.Fatal("the Owner role is never granted by an invite")
		}
		if r.Key == "editor" && r.Hint != "Editors can send; two-factor becomes required (7-day grace)." {
			t.Fatalf("editor hint: %q", r.Hint)
		}
	}

	var res struct {
		Invitations []struct{ ID, Email, Link string }
		Emailed     bool
		Fallback    string
	}
	before := auditCount(t, env, "team.invited")
	c.Do("POST", ws+"/invitations", map[string]any{
		"emails": []string{"priya@acme.io", "Kim@Example.com"}, "roleId": roleID(t, l, "editor"),
		"scope": map[string]any{"lists": []string{beta}}, "message": "Welcome aboard",
	}).MustStatus(t, 201).JSON(t, &res)
	if len(res.Invitations) != 2 || res.Emailed || res.Fallback != "no_system_provider" || !strings.Contains(res.Invitations[0].Link, "/invite/") {
		t.Fatalf("invite: %+v", res)
	}
	if auditCount(t, env, "team.invited") != before+2 {
		t.Fatal("each invite writes an audit entry")
	}
	// The copyable link is the same one that was handed out.
	var link struct{ Link string }
	c.Do("POST", ws+"/invitations/"+res.Invitations[0].ID+"/link", nil).MustStatus(t, 200).JSON(t, &link)
	if link.Link != res.Invitations[0].Link {
		t.Fatalf("link %q != %q", link.Link, res.Invitations[0].Link)
	}
	token := link.Link[strings.LastIndex(link.Link, "/")+1:]
	inv, err := env.App.Q.GetInvitationByToken(context.Background(), hashToken(token))
	if err != nil || inv.Email != "priya@acme.io" || inv.RoleKey != "editor" {
		t.Fatalf("token lookup: %+v %v", inv, err)
	}

	// Validation: duplicates, members and bad addresses.
	c.Do("POST", ws+"/invitations", map[string]any{"emails": []string{"priya@acme.io"}, "roleId": roleID(t, l, "viewer")}).MustStatus(t, 422)
	c.Do("POST", ws+"/invitations", map[string]any{"emails": []string{"marcus@acme.io"}, "roleId": roleID(t, l, "viewer")}).MustStatus(t, 422)
	c.Do("POST", ws+"/invitations", map[string]any{"emails": []string{"not-an-email"}, "roleId": roleID(t, l, "viewer")}).MustStatus(t, 422)
	c.Do("POST", ws+"/invitations", map[string]any{"emails": []string{"new@example.com"}, "roleId": roleID(t, l, "owner")}).MustStatus(t, 403)

	var resend struct {
		Emailed  bool
		Fallback string
	}
	c.Do("POST", ws+"/invitations/"+res.Invitations[1].ID+"/resend", nil).MustStatus(t, 200).JSON(t, &resend)
	if resend.Emailed || resend.Fallback != "no_system_provider" {
		t.Fatalf("resend: %+v", resend)
	}
	c.Do("DELETE", ws+"/invitations/"+res.Invitations[1].ID, nil).MustStatus(t, 204)
	c.Do("DELETE", ws+"/invitations/"+res.Invitations[1].ID, nil).MustStatus(t, 404)
	var list struct{ Items []struct{ Email string } }
	c.Do("GET", ws+"/invitations", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 2 { // jo@contractor.dev and priya@acme.io
		t.Fatalf("pending: %+v", list.Items)
	}
}

func TestChangeRoleAndEscalation(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var l memberList
	c.Do("GET", ws+"/members", nil).MustStatus(t, 200).JSON(t, &l)
	linh := byEmail(t, l, "linh@acme.io")

	// The last Owner can't be demoted, removed or deactivated.
	r := c.Do("PATCH", ws+"/members/"+linh.ID, map[string]any{"roleId": roleID(t, l, "admin")}).MustStatus(t, 409)
	if p := r.Problem(t); p["code"] != "last_owner" {
		t.Fatalf("last owner: %v", p)
	}
	c.Do("POST", ws+"/members/"+linh.ID+"/deactivate", map[string]any{"revokeKeys": false}).MustStatus(t, 409)

	aisha := byEmail(t, l, "aisha@acme.io")
	c.Do("PATCH", ws+"/members/"+aisha.ID, map[string]any{"roleId": roleID(t, l, "editor")}).MustStatus(t, 204)
	if auditCount(t, env, "member.role_changed") != 1 {
		t.Fatal("role change is audited")
	}

	// A custom "Team lead" holds team:manage but not segments:sql: they can't
	// give anyone the Admin role (which has it) or change an Admin.
	var created struct{ ID string }
	c.Do("POST", ws+"/roles", map[string]any{"name": "Team lead", "permissions": []string{"team:manage", "campaigns:view"}}).MustStatus(t, 201).JSON(t, &created)
	u := env.CreateUser(apptest.UserOpts{Email: "lead@acme.io", Name: "Lee Lead"})
	wsRow, err := env.App.Q.GetWorkspaceBySlug(context.Background(), "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	role, err := env.App.Q.GetRoleByKey(context.Background(), store.GetRoleByKeyParams{WorkspaceID: wsRow.ID, Key: "team-lead"})
	if err != nil {
		t.Fatal(err)
	}
	addMember(t, env, wsRow.ID, u.ID, role.ID)
	lead := env.Client()
	lead.SignInDemo("lead@acme.io")
	sofia := byEmail(t, l, "sofia@acme.io")
	lead.Do("PATCH", ws+"/members/"+sofia.ID, map[string]any{"roleId": roleID(t, l, "admin")}).MustStatus(t, 403)
	marcus := byEmail(t, l, "marcus@acme.io")
	lead.Do("PATCH", ws+"/members/"+marcus.ID, map[string]any{"roleId": roleID(t, l, "viewer")}).MustStatus(t, 403)
	lead.Do("POST", ws+"/members/"+marcus.ID+"/deactivate", map[string]any{}).MustStatus(t, 403)
}

func TestDeactivateReactivateRemove(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var l memberList
	c.Do("GET", ws+"/members", nil).MustStatus(t, 200).JSON(t, &l)
	sofia := byEmail(t, l, "sofia@acme.io")
	var res struct {
		SessionsRevoked int64    `json:"sessionsRevoked"`
		KeysRevoked     []string `json:"keysRevoked"`
	}
	c.Do("POST", ws+"/members/"+sofia.ID+"/deactivate", map[string]any{"revokeKeys": true, "reason": "Contract ended"}).MustStatus(t, 200).JSON(t, &res)
	if res.SessionsRevoked != 2 {
		t.Fatalf("sessions: %+v", res)
	}
	var reason string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT reason FROM audit_events WHERE action = 'member.deactivated'`).Scan(&reason); err != nil || reason != "Contract ended" {
		t.Fatalf("audit reason %q %v", reason, err)
	}
	c.Do("POST", ws+"/members/"+sofia.ID+"/deactivate", map[string]any{}).MustStatus(t, 409)
	c.Do("POST", ws+"/members/"+sofia.ID+"/reactivate", nil).MustStatus(t, 204)

	// Tom's account is deactivated on the instance; Linh is an instance admin.
	tom := byEmail(t, l, "tom@acme.io")
	c.Do("POST", ws+"/members/"+tom.ID+"/reactivate", nil).MustStatus(t, 204)
	c.Do("GET", ws+"/members", nil).MustStatus(t, 200).JSON(t, &l)
	if byEmail(t, l, "tom@acme.io").Status != "active" {
		t.Fatal("tom reactivated")
	}

	daniel := byEmail(t, l, "daniel@acme.io")
	var detail struct {
		Member      member
		Permissions []struct {
			Key, Source, IncludedBy, Scope string
			Granted                        bool
		}
		KeysNote string
		Sessions int64
		Activity []struct{ Action, Result string }
	}
	c.Do("GET", ws+"/members/"+daniel.ID, nil).MustStatus(t, 200).JSON(t, &detail)
	found := false
	for _, p := range detail.Permissions {
		if p.Key == "campaigns:view" {
			found = p.Granted && p.Source == "Role · Content Writer" && p.IncludedBy == "campaigns:edit" && p.Scope == "all lists"
		}
		if p.Key == "campaigns:send" && p.Granted {
			t.Fatal("content writers can't send")
		}
	}
	if !found || !strings.Contains(detail.KeysNote, "apikeys:manage") || detail.Sessions != 2 || len(detail.Activity) == 0 {
		t.Fatalf("detail: %+v", detail)
	}
	c.Do("POST", ws+"/members/"+daniel.ID+"/sign-out", nil).MustStatus(t, 200)
	c.Do("DELETE", ws+"/members/"+daniel.ID, nil).MustStatus(t, 204)
	c.Do("GET", ws+"/members/"+daniel.ID, nil).MustStatus(t, 404)

	// Viewers can't manage.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("POST", ws+"/members/"+sofia.ID+"/deactivate", map[string]any{}).MustStatus(t, 403)
}

func TestBulk(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var l memberList
	c.Do("GET", ws+"/members", nil).MustStatus(t, 200).JSON(t, &l)
	aisha, sofia := byEmail(t, l, "aisha@acme.io"), byEmail(t, l, "sofia@acme.io")
	var res struct {
		Changed int
		Skipped []string
	}
	c.Do("POST", ws+"/members/bulk", map[string]any{"memberIds": []string{aisha.ID, sofia.ID}, "action": "require_2fa"}).MustStatus(t, 200).JSON(t, &res)
	if res.Changed != 1 || len(res.Skipped) != 1 {
		t.Fatalf("require 2FA: %+v", res)
	}
	c.Do("POST", ws+"/members/bulk", map[string]any{"memberIds": []string{aisha.ID, sofia.ID}, "action": "change_role", "roleId": roleID(t, l, "editor")}).MustStatus(t, 200).JSON(t, &res)
	if res.Changed != 1 || len(res.Skipped) != 1 {
		t.Fatalf("bulk role: %+v", res)
	}
}

// The second demo workspace matches InstanceUsers and ShellSwitcher: Kenji
// Watanabe owns it, Linh is an Admin, Marcus an Editor, and Sofia isn't a member.
func TestSecondWorkspaceMembers(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var l memberList
	c.Do("GET", "/workspaces/acme-engineering-blog/members", nil).MustStatus(t, 200).JSON(t, &l)
	roles := map[string]string{}
	for _, m := range l.Items {
		roles[m.Email] = m.Role.Key
	}
	if roles["kenji@acme.io"] != "owner" || roles["linh@acme.io"] != "admin" || roles["marcus@acme.io"] != "editor" || len(roles) != 3 {
		t.Fatalf("members: %v", roles)
	}
	kenji := byEmail(t, l, "kenji@acme.io")
	if signInLabels(kenji.SignIn) != "Passkey" || kenji.TwoFA.State != "on" || !kenji.LastOwner {
		t.Fatalf("kenji: %+v", kenji)
	}
}
