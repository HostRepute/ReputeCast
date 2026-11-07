package sso_test

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/sso"
	"github.com/hostrepute/reputecast/internal/sso/ssotest"
)

// socialInvites turns on Google and Microsoft (any tenant) at fake servers.
func socialInvites(t *testing.T, admin *apptest.Client) (google, ms *ssotest.IdP) {
	t.Helper()
	google = ssotest.New()
	t.Cleanup(google.Close)
	ms = ssotest.New()
	t.Cleanup(ms.Close)
	ms.Path = "/common/v2.0"
	ms.DiscoveryIssuer = ms.Server.URL + "/{tenantid}/v2.0"
	ms.TokenIssuer = ms.Server.URL + "/tid-1/v2.0"
	socialEndpoints(t, sso.SocialEndpoints{GoogleIssuer: google.Issuer(), MicrosoftBase: ms.Server.URL})
	admin.Do("PUT", "/admin/social/google", map[string]any{"enabled": true, "clientId": google.ClientID, "clientSecret": google.ClientSecret}).MustStatus(t, 200)
	admin.Do("PUT", "/admin/social/microsoft", map[string]any{"enabled": true, "clientId": ms.ClientID, "clientSecret": ms.ClientSecret, "tenant": "common"}).MustStatus(t, 200)
	return google, ms
}

// memberOf counts the memberships of an email in a workspace.
func memberOf(t *testing.T, env *apptest.Env, email, slug string) int {
	t.Helper()
	return count(t, env, `SELECT count(*) FROM workspace_members m JOIN users u ON u.id = m.user_id JOIN workspaces w ON w.id = m.workspace_id
		WHERE u.email = $1 AND w.slug = $2`, email, slug)
}

// An invitation for an email that already has an account can't be taken
// over with someone's identity provider account: the IdP's email proves
// nothing about the account (Microsoft's isn't even verified). The person
// signs in first, then accepts.
func TestInvitationForAnExistingAccountNeedsItsOwnSignIn(t *testing.T) {
	env, admin := setup(t)
	google, ms := socialInvites(t, admin)
	token := invite(t, env, "acme-engineering-blog", "aisha@acme.io", "editor")

	ms.Claims = map[string]any{"sub": "ms-attacker", "tid": "tid-1", "email": "aisha@acme.io", "preferred_username": "attacker@evil.example"}
	google.Claims = map[string]any{"sub": "g-not-aisha", "email": "aisha@acme.io", "email_verified": true}
	for _, kind := range []string{"microsoft", "google"} {
		b := newBrowser(t, env)
		loc := b.follow("/api/v1/auth/social/" + kind + "/start?invite=" + token)
		q := query(t, loc)
		if !strings.HasPrefix(loc, "/sign-in?") || q.Get("error") != "idp" || q.Get("code") != "invite_sign_in" || q.Get("email") != "aisha@acme.io" || q.Get("next") != "/invite/"+token {
			t.Fatalf("%s: %s", kind, loc)
		}
		if b.me() != "" {
			t.Fatalf("%s signed the browser in", kind)
		}
	}
	if n := count(t, env, `SELECT count(*) FROM user_identities i JOIN users u ON u.id = i.user_id WHERE u.email = 'aisha@acme.io' AND i.kind IN ('google', 'microsoft')`); n != 0 {
		t.Fatalf("%d identities linked to Aisha", n)
	}
	if memberOf(t, env, "aisha@acme.io", "acme-engineering-blog") != 0 || count(t, env, `SELECT count(*) FROM invitations WHERE email = 'aisha@acme.io' AND accepted_at IS NULL`) != 1 {
		t.Fatal("the invitation was taken")
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'signin.rejected' AND metadata->>'code' = 'invite_sign_in'`); n != 2 {
		t.Fatalf("%d refusals audited", n)
	}

	// Aisha signs in as usual and accepts while signed in.
	b := newBrowser(t, env)
	b.signInAs("aisha@acme.io")
	var res struct {
		Workspace  string `json:"workspace"`
		NewAccount bool   `json:"newAccount"`
	}
	if s := b.do("POST", "/api/v1/invitations/"+token+"/accept", map[string]any{}, &res); s != 200 || res.Workspace != "acme-engineering-blog" || res.NewAccount {
		t.Fatalf("accept signed in: %d %+v", s, res)
	}
}

// An identity the account already signs in with accepts the invitation, as
// it would sign in.
func TestInvitationForAnExistingAccountWithItsIdentity(t *testing.T) {
	env, admin := setup(t)
	google, _ := socialInvites(t, admin)
	google.Claims = map[string]any{"sub": "g-aisha", "email": "aisha@acme.io", "email_verified": true}
	linker := newBrowser(t, env)
	linker.signInAs("aisha@acme.io")
	if loc := linker.follow("/api/v1/auth/social/google/start?intent=link"); !strings.Contains(loc, "linked=google") {
		t.Fatalf("link: %s", loc)
	}
	token := invite(t, env, "acme-engineering-blog", "aisha@acme.io", "editor")
	b := newBrowser(t, env)
	if loc := b.follow("/api/v1/auth/social/google/start?invite=" + token); loc != "/w/acme-engineering-blog" || b.me() != "aisha@acme.io" {
		t.Fatalf("accept with a linked identity: %s %v", loc, b.Visited)
	}
	if memberOf(t, env, "aisha@acme.io", "acme-engineering-blog") != 1 {
		t.Fatal("Aisha didn't join")
	}
}

// A new account needs an email the identity provider vouches for, under the
// rules sign-in uses.
func TestInvitationNewAccountNeedsAVouchedEmail(t *testing.T) {
	env, admin := setup(t)
	google, ms := socialInvites(t, admin)
	idp := ssotest.New()
	defer idp.Close()
	d := addProvider(t, env, admin, idp, func(b map[string]any) {
		b["allowedDomains"] = []string{"acme-labs.io"}
		b["jit"] = map[string]any{"enabled": false}
	})
	refused := func(start, email string) {
		t.Helper()
		if q := query(t, newBrowser(t, env).follow(start)); q.Get("code") != "email_not_verified" {
			t.Fatalf("%s: %v", start, q)
		}
		if count(t, env, `SELECT count(*) FROM users WHERE email = $1`, email) != 0 {
			t.Fatalf("an account was created for %s", email)
		}
	}
	joined := func(start, email string) {
		t.Helper()
		b := newBrowser(t, env)
		if loc := b.follow(start); loc != "/w/acme-engineering-blog" || b.me() != email {
			t.Fatalf("%s: %s %v", start, loc, b.Visited)
		}
	}

	// Google: only email_verified.
	tok := invite(t, env, "acme-engineering-blog", "new.one@example.com", "viewer")
	google.Claims = map[string]any{"sub": "g-new-1", "email": "new.one@example.com", "email_verified": false}
	refused("/api/v1/auth/social/google/start?invite="+tok, "new.one@example.com")
	google.Claims = map[string]any{"sub": "g-new-1b", "email": "new.one@example.com"}
	refused("/api/v1/auth/social/google/start?invite="+tok, "new.one@example.com")
	google.Claims = map[string]any{"sub": "g-new-1c", "email": "new.one@example.com", "email_verified": true}
	joined("/api/v1/auth/social/google/start?invite="+tok, "new.one@example.com")

	// Microsoft: the email claim counts only when it is the sign-in name.
	tok = invite(t, env, "acme-engineering-blog", "new.two@example.com", "viewer")
	ms.Claims = map[string]any{"sub": "ms-new-2", "tid": "tid-1", "email": "new.two@example.com", "preferred_username": "someone@other.example"}
	refused("/api/v1/auth/social/microsoft/start?invite="+tok, "new.two@example.com")
	ms.Claims = map[string]any{"sub": "ms-new-2b", "tid": "tid-1", "email": "new.two@example.com", "preferred_username": "New.Two@example.com"}
	joined("/api/v1/auth/social/microsoft/start?invite="+tok, "new.two@example.com")

	// SSO without email_verified: only at the provider's own domains.
	tok = invite(t, env, "acme-engineering-blog", "new.three@example.com", "viewer")
	idp.Claims = map[string]any{"sub": "s-3", "email": "new.three@example.com", "amr": []string{"mfa"}}
	refused("/api/v1/auth/sso/"+d.ID+"/start?invite="+tok, "new.three@example.com")
	tok = invite(t, env, "acme-engineering-blog", "new.four@acme-labs.io", "viewer")
	idp.Claims = map[string]any{"sub": "s-4", "email": "new.four@acme-labs.io", "amr": []string{"mfa"}}
	joined("/api/v1/auth/sso/"+d.ID+"/start?invite="+tok, "new.four@acme-labs.io")

	// One identity provider account signs in to one account.
	tok = invite(t, env, "acme-engineering-blog", "new.five@example.com", "viewer")
	google.Claims = map[string]any{"sub": "g-new-1c", "email": "new.five@example.com", "email_verified": true}
	if q := query(t, newBrowser(t, env).follow("/api/v1/auth/social/google/start?invite="+tok)); q.Get("code") != "identity_in_use" {
		t.Fatalf("identity in use: %v", q)
	}
}

// At a domain under single sign-on, only its provider joins people: another
// provider's sign-in goes on to it with the invitation.
func TestInvitationFollowsDomainEnforcement(t *testing.T) {
	env, admin := setup(t)
	google, _ := socialInvites(t, admin)
	idp := ssotest.New()
	defer idp.Close()
	d := addProvider(t, env, admin, idp, func(b map[string]any) {
		b["allowedDomains"] = []string{"acme-labs.io"}
		b["jit"] = map[string]any{"enabled": false}
	})
	admin.Do("PUT", "/admin/sso/"+d.ID+"/enforcement", map[string]any{"domains": []string{"acme-labs.io"}, "confirm": true}).MustStatus(t, 200)
	token := invite(t, env, "acme-engineering-blog", "new.hire@acme-labs.io", "editor")

	google.Claims = map[string]any{"sub": "g-nh", "email": "new.hire@acme-labs.io", "email_verified": true}
	idp.Claims = map[string]any{"sub": "nh-1", "email": "new.hire@acme-labs.io", "email_verified": true, "amr": []string{"mfa"}}
	b := newBrowser(t, env)
	loc := b.follow("/api/v1/auth/social/google/start?invite=" + token)
	viaIdP := false
	for _, v := range b.Visited {
		if strings.HasPrefix(v, idp.Issuer()+"/auth") {
			viaIdP = true
		}
	}
	if loc != "/w/acme-engineering-blog" || !viaIdP || b.me() != "new.hire@acme-labs.io" {
		t.Fatalf("enforced domain: %s %v", loc, b.Visited)
	}
	if count(t, env, `SELECT count(*) FROM user_identities WHERE kind = 'google' AND subject = 'g-nh'`) != 0 ||
		count(t, env, `SELECT count(*) FROM user_identities WHERE sso_provider_id = $1 AND subject = 'nh-1'`, d.ID) != 1 {
		t.Fatal("the invitee joined with the wrong identity")
	}
}
