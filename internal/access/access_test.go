package access

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
)

func labels(ms []MemberSignInMethod) string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Label
	}
	return strings.Join(out, " · ")
}

func TestPrimarySignIn(t *testing.T) {
	sso := Identity{Kind: "sso", ProviderName: "Google Workspace", ProviderPreset: "google"}
	github := Identity{Kind: "github"}
	cases := []struct {
		name      string
		password  bool
		passkeys  int64
		ids       []Identity
		all, list string
	}{
		// Linh on SettingsTeam and AccountSecurity: her own credentials win in the list.
		{"own credentials and linked identities", true, 1, []Identity{github, sso}, "Password · passkey · GitHub · Google Workspace SSO", "Password · passkey"},
		{"password only", true, 0, nil, "Password", "Password"},
		{"passkey only", false, 2, []Identity{sso}, "Passkey · Google Workspace SSO", "Passkey"},
		// Marcus: no password or passkey, so the list shows his SSO sign-in.
		{"SSO only", false, 0, []Identity{sso}, "Google Workspace SSO", "Google Workspace SSO"},
		{"social only", false, 0, []Identity{github}, "GitHub", "GitHub"},
		{"SSO and social", false, 0, []Identity{sso, github}, "Google Workspace SSO · GitHub", "Google Workspace SSO · GitHub"},
		{"nothing", false, 0, nil, "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			all := SignInMethods(c.password, c.passkeys, c.ids)
			if got := labels(all); got != c.all {
				t.Errorf("all methods = %q, want %q", got, c.all)
			}
			if got := labels(PrimarySignIn(all)); got != c.list {
				t.Errorf("list = %q, want %q", got, c.list)
			}
		})
	}
}

// A member limited to some lists sends to those lists only: never to a
// segment, which can reach anyone in the workspace.
func TestAudienceInScope(t *testing.T) {
	beta, weekly := uuid.New(), uuid.New()
	scoped := authz.Scope{Lists: []uuid.UUID{beta}}
	for _, c := range []struct {
		scope    authz.Scope
		lists    []uuid.UUID
		segments int
		want     bool
	}{
		{authz.Scope{}, []uuid.UUID{weekly}, 2, true},
		{scoped, []uuid.UUID{beta}, 0, true},
		{scoped, nil, 0, true},
		{scoped, []uuid.UUID{beta, weekly}, 0, false},
		{scoped, []uuid.UUID{beta}, 1, false},
		{scoped, nil, 1, false},
	} {
		if got := AudienceInScope(c.scope, c.lists, c.segments); got != c.want {
			t.Errorf("AudienceInScope(%v, %v, %d) = %v", c.scope, c.lists, c.segments, got)
		}
	}
	if got := ParseLists([]string{beta.String(), "not-an-id"}); len(got) != 1 || got[0] != beta {
		t.Fatalf("ParseLists: %v", got)
	}
	if AudienceScopeProblem(i18n.For(i18n.Vietnamese)) == AudienceScopeProblem(i18n.Default) {
		t.Fatal("the reason reads in the member's language")
	}
}
