package sso

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/i18n"
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
	pl := signInPlan{User: &u, MFA: true, MFANote: keep("mfa.amr", nil), MFAAmr: true, Changes: []memberChange{
		{WorkspaceName: "Acme Newsletter", FromKey: "admin", FromName: "Admin", To: store.SSOWorkspaceRolesRow{RoleKey: "editor", RoleName: "Editor"}, Group: "Marketing", MemberID: &u.ID},
		{WorkspaceName: "Acme Engineering Blog", FromKey: "editor", FromName: "Editor", To: store.SSOWorkspaceRolesRow{RoleKey: "admin", RoleName: "Admin"}, Group: "Eng Leads", MemberID: &u.ID},
	}}
	got := []string{}
	for _, o := range describe(i18n.Default, id, pl) {
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
	rej := describe(i18n.Default, identity{Email: "kim@gmail.com"}, signInPlan{Reject: failure("domain_not_allowed", "flow.domain", i18n.Args{"name": p.Name})})
	if len(rej) != 1 || rej[0].Detail != "kim@gmail.com: email domain not allowed for Microsoft Entra ID SSO" {
		t.Fatalf("reject: %+v", rej)
	}

	// Leaving a mapped group: the membership ends, or falls back to the
	// JIT default role in the JIT workspace.
	exits := signInPlan{User: &u, MFA: true, MFANote: keep("mfa.amr", nil), MFAAmr: true, Changes: []memberChange{
		{WorkspaceName: "Acme Engineering Blog", FromKey: "admin", FromName: "Admin", LeftGroup: "Eng Leads", Remove: true, MemberID: &u.ID},
		{WorkspaceName: "Acme Newsletter", FromKey: "editor", FromName: "Editor", To: store.SSOWorkspaceRolesRow{RoleKey: "viewer", RoleName: "Viewer"}, LeftGroup: "Marketing", MemberID: &u.ID},
	}}
	got = got[:0]
	for _, o := range describe(i18n.Default, id, exits)[1:3] {
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

// The test sign-in's outcome reads in the admin's language, built-in roles
// too; a flow error from before its message was kept reads in English.
func TestDescribeInVietnamese(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	u := store.User{ID: uuid.New(), Email: "marcus@acme.io", Name: "Marcus Chen"}
	pl := signInPlan{User: &u, MFA: true, MFANote: keep("mfa.amr", nil), MFAAmr: true, Changes: []memberChange{
		{WorkspaceName: "Acme Newsletter", FromKey: "admin", FromName: "Admin", To: store.SSOWorkspaceRolesRow{RoleKey: "editor", RoleName: "Editor"}, Group: "Marketing", MemberID: &u.ID},
	}, GroupsMissing: true, Mappings: 2}
	got := []string{}
	for _, o := range describe(vi, identity{Email: "marcus@acme.io"}, pl) {
		got = append(got, o.Label+": "+o.Detail+" ["+o.Strong+o.Code+"]")
	}
	want := []string{
		"Tài khoản hiện có: marcus@acme.io khớp với Marcus Chen · không tạo tài khoản mới []",
		"Đổi vai trò: Acme Newsletter: Quản trị viên → Biên tập viên theo nhóm “Marketing”. Vai trò được gán trực tiếp sẽ bị thay ở lần đăng nhập tiếp theo của họ; hãy xem lại ánh xạ nếu đó không phải điều bạn muốn. [Quản trị viên → Biên tập viên]",
		"Ánh xạ không hoạt động: Không có claim groups · 2 ánh xạ nhóm không được áp dụng []",
		"Xác thực hai lớp: amr có mfa · được tính là xác thực hai lớp qua IdP [amr]",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got\n%s", strings.Join(got, "\n"))
	}
	rej := describe(vi, identity{Email: "kim@gmail.com"}, signInPlan{Reject: failure("domain_not_allowed", "flow.domain", i18n.Args{"name": "Okta"})})
	if rej[0].Label != "Bị từ chối" || rej[0].Detail != "kim@gmail.com: tên miền email không được phép với SSO Okta" {
		t.Fatalf("reject: %+v", rej)
	}
	old := describe(vi, identity{Email: "kim@gmail.com"}, signInPlan{Reject: &flowError{Detail: "email domain not allowed for Okta SSO"}})
	if old[0].Detail != "kim@gmail.com: email domain not allowed for Okta SSO" {
		t.Fatalf("old reject: %+v", old)
	}
}

// The presets' English is their messages' English, so English readers read
// exactly the strings in Presets; every other language has them all.
func TestPresetsInEveryLanguage(t *testing.T) {
	for _, p := range Presets {
		en := p.In(i18n.Default)
		fields := map[string][2]string{"app": {p.App, en.App}, "discoveryHint": {p.DiscoveryHint, en.DiscoveryHint}, "groupsHint": {p.GroupsHint, en.GroupsHint},
			"groupsMissingHint": {p.GroupsMissingHint, en.GroupsMissingHint}, "trustedTwoFactor": {p.TrustedTwoFA, en.TrustedTwoFA}, "docsLabel": {p.DocsLabel, en.DocsLabel}}
		for field, v := range fields {
			if v[0] == "" {
				continue
			}
			m, err := msgs.Lookup("preset." + p.Key + "." + field)
			if err != nil {
				t.Errorf("%s.%s has no message", p.Key, field)
				continue
			}
			if got := m.Text(i18n.Default, nil); got != v[0] {
				t.Errorf("%s.%s: message %q, preset %q", p.Key, field, got, v[0])
			}
		}
		vi := p.In(i18n.For(i18n.Vietnamese))
		if vi.DiscoveryHint == p.DiscoveryHint || vi.TrustedTwoFA == p.TrustedTwoFA || vi.Name != p.Name && p.Key != "oidc" {
			t.Errorf("%s in Vietnamese: %+v", p.Key, vi)
		}
	}
	if g := PresetFor("oidc"); g.In(i18n.Default).Name != "Generic OIDC" || g.In(i18n.For(i18n.Vietnamese)).Name != "OIDC chung" || g.In(i18n.For(i18n.Vietnamese)).ShortName != "IdP" {
		t.Errorf("generic OIDC: %+v", g.In(i18n.For(i18n.Vietnamese)))
	}
	if trustedTwoFA("google").In(i18n.Default) != PresetFor("google").TrustedTwoFA {
		t.Error("trusted two-factor note")
	}
}

// A discovery failure's summary keeps its error as a part of the sentence,
// in each reader's language, without its final period.
func TestDiscoverySummary(t *testing.T) {
	d := discovery{}.failed("discovery.noAnswer", i18n.Args{"why": "refused", "error": "dial tcp: connection refused"})
	if d.Error != "The issuer didn't answer: the connection was refused." {
		t.Fatalf("error: %q", d.Error)
	}
	meta := audit.KeepSummary(nil, discoverySummary(d, true)...)
	if meta["summary"] != "issuer changed · discovery failed: The issuer didn't answer: the connection was refused" {
		t.Fatalf("English: %q", meta["summary"])
	}
	raw, _ := json.Marshal(meta)
	if got := audit.KeptIn(raw).SummaryIn(i18n.For(i18n.Vietnamese), ""); got != "đã đổi nhà phát hành · discovery thất bại: Nhà phát hành không trả lời: kết nối bị từ chối" {
		t.Fatalf("Vietnamese: %q", got)
	}
	ok := discovery{OK: true, AuthURL: "a", TokenURL: "t", JWKSURL: "j", Keys: 1}
	meta = audit.KeepSummary(nil, discoverySummary(ok, false)...)
	if meta["summary"] != "discovery OK · 3 endpoints · 1 signing key" {
		t.Fatalf("OK: %q", meta["summary"])
	}
}
