package sso

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

func TestFormatClaimsLikeTheDesign(t *testing.T) {
	var claims map[string]any
	_ = json.Unmarshal([]byte(`{
		"aud": "a9f3c1e2-4b7d-4e8f-9a0b-1c2d3e4f5a6b", "exp": 1, "iat": 1, "nonce": "n", "tid": "7c1a2b3d-0e4f-4a5b-9c6d-1e2f3a4b5c6d", "xms_pl": "en",
		"amr": ["pwd", "mfa"], "groups": ["3f2a9d1e-5b7c-4e21-9a8f-2d4c6e8b0c91", "8b7d44c0-1e2f-4a3b-8c9d-6e7f8a9b002e"],
		"preferred_username": "marcus@acme.io", "name": "Marcus Chen", "email": "marcus@acme.io",
		"sub": "AAAAAAAAAAAAAAAAAAAAAM4sQ8vD2fX0bYkHw9Lr", "iss": "https://login.microsoftonline.com/7c1a2b3d-0e4f-4a5b-9c6d-1e2f3a4b5c6d/v2.0"}`), &claims)
	want := `{
  "iss": "https://login.microsoftonline.com/7c1a…5c6d/v2.0",
  "sub": "AAAAAAAAAAAAAAAAAAAAAM4s…",
  "email": "marcus@acme.io",
  "name": "Marcus Chen",
  "preferred_username": "marcus@acme.io",
  "groups": ["3f2a9d1e-…-c91", "8b7d44c0-…-02e"],
  "amr": ["pwd", "mfa"]
}`
	if got := formatClaims(claims); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if got := formatClaims(map[string]any{"name": "R&D <team>"}); got != "{\n  \"name\": \"R&D <team>\"\n}" {
		t.Fatalf("escaping: %s", got)
	}
}

func TestExtract(t *testing.T) {
	p := store.SsoProvider{ID: uuid.New(), Name: "Entra", Preset: "entra", ClaimMap: json.RawMessage(`{"email":"mail","groups":"roles"}`)}
	id := extract(p, map[string]any{"sub": "s1", "preferred_username": "Marcus@Acme.io", "given_name": "Marcus", "family_name": "Chen",
		"roles": []any{"a", "b"}, "amr": []any{"pwd", "mfa"}, "email_verified": "true"})
	if id.Email != "marcus@acme.io" || id.Name != "Marcus Chen" || !id.GroupsClaim || strings.Join(id.Groups, ",") != "a,b" || !id.amrMFA() || id.EmailVerified == nil || !*id.EmailVerified {
		t.Fatalf("extract: %+v", id)
	}
	over := extract(p, map[string]any{"sub": "s2", "mail": "x@acme.io", "_claim_names": map[string]any{"roles": "src1"}})
	if over.GroupsClaim || !over.GroupsOverage || over.EmailVerified != nil {
		t.Fatalf("overage: %+v", over)
	}
	if extract(p, map[string]any{"sub": "s3", "amr": []any{"pwd"}}).amrMFA() {
		t.Fatal("pwd alone counted as two-factor")
	}
}

func TestDomainsAndPaths(t *testing.T) {
	ds, err := normalizeDomains([]string{" Acme.IO ", "@acme-labs.io", "acme.io", ""})
	if err != nil || strings.Join(ds, ",") != "acme.io,acme-labs.io" {
		t.Fatalf("normalize: %v %v", ds, err)
	}
	for _, bad := range []string{"acme", "-acme.io", "acme..io", "ac me.io", "acme.io/x"} {
		if _, err := normalizeDomains([]string{bad}); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
	for in, want := range map[string]string{"/w/acme/campaigns?x=1": "/w/acme/campaigns?x=1", "//evil.example": "", "/\\evil.example": "", "/\t/evil.example": "", "/\n/evil.example": "", "/w/a\\b": "", "https://evil.example": "", "": ""} {
		if got := safeNext(in); got != want {
			t.Fatalf("safeNext(%q) = %q", in, got)
		}
	}
	if !validGitHubOrg("acme-eng") || validGitHubOrg("acme--eng") || validGitHubOrg("-acme") || validGitHubOrg("acme eng") {
		t.Fatal("GitHub org names")
	}
}

func TestDescribe(t *testing.T) {
	p := store.SsoProvider{ID: uuid.New(), Name: "Microsoft Entra ID", Preset: "entra"}
	u := store.User{ID: uuid.New(), Email: "marcus@acme.io", Name: "Marcus Chen"}
	id := identity{Email: "marcus@acme.io"}
	pl := signInPlan{User: &u, MFA: true, MFANote: "amr includes mfa · counts as two-factor via IdP", Changes: []memberChange{
		{WorkspaceName: "Acme Newsletter", FromKey: "admin", FromName: "Admin", To: store.SSOWorkspaceRolesRow{RoleKey: "editor", RoleName: "Editor"}, Group: "Marketing", MemberID: &u.ID},
		{WorkspaceName: "Acme Engineering Blog", FromKey: "editor", FromName: "Editor", To: store.SSOWorkspaceRolesRow{RoleKey: "admin", RoleName: "Admin"}, Group: "Eng Leads", MemberID: &u.ID},
	}}
	got := []string{}
	for _, o := range describe(p, id, pl) {
		got = append(got, o.Tone+" "+o.Label+": "+o.Detail)
	}
	want := []string{
		"ok Existing account: marcus@acme.io matches Marcus Chen · no account created",
		"warn Role change: Acme Newsletter: Admin → Editor from group “Marketing”. The direct assignment is replaced at their next sign-in; review the mapping if that's not intended.",
		"ok Role: Acme Engineering Blog: Editor → Admin from group “Eng Leads”",
		"ok Two-factor: amr includes mfa · counts as two-factor via IdP",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got\n%s", strings.Join(got, "\n"))
	}
	rej := describe(p, identity{Email: "kim@gmail.com"}, signInPlan{Reject: &flowError{Detail: "email domain not allowed for Microsoft Entra ID SSO"}})
	if len(rej) != 1 || rej[0].Detail != "kim@gmail.com: email domain not allowed for Microsoft Entra ID SSO" {
		t.Fatalf("reject: %+v", rej)
	}

	// Leaving a mapped group: the membership ends, or falls back to the
	// JIT default role in the JIT workspace.
	exits := signInPlan{User: &u, MFA: true, MFANote: "amr includes mfa · counts as two-factor via IdP", Changes: []memberChange{
		{WorkspaceName: "Acme Engineering Blog", FromKey: "admin", FromName: "Admin", LeftGroup: "Eng Leads", Remove: true, MemberID: &u.ID},
		{WorkspaceName: "Acme Newsletter", FromKey: "editor", FromName: "Editor", To: store.SSOWorkspaceRolesRow{RoleKey: "viewer", RoleName: "Viewer"}, LeftGroup: "Marketing", MemberID: &u.ID},
	}}
	got = got[:0]
	for _, o := range describe(p, id, exits)[1:3] {
		got = append(got, o.Tone+" "+o.Label+": "+o.Detail+" ["+o.Strong+"]")
	}
	want = []string{
		"warn Removed: Acme Engineering Blog: leaves the workspace · no longer in group “Eng Leads”, which granted Admin [Admin]",
		"warn Role change: Acme Newsletter: Editor → Viewer (JIT default) · no longer in group “Marketing” [Editor → Viewer]",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("exits:\n%s", strings.Join(got, "\n"))
	}
}
