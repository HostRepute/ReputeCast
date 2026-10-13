package sso_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/sso"
	"github.com/hostrepute/reputecast/internal/sso/ssotest"
)

func TestDiscovery(t *testing.T) {
	env, admin := setup(t)
	idp := ssotest.New()
	defer idp.Close()
	var d detail
	admin.Do("POST", "/admin/sso", map[string]any{"preset": "okta"}).MustStatus(t, 201).JSON(t, &d)
	if d.Name != "Okta" || d.Status != "draft" {
		t.Fatalf("created: %+v", d)
	}
	// Nothing to discover yet.
	admin.Do("POST", "/admin/sso/"+d.ID+"/discovery", map[string]any{}).MustStatus(t, 422)

	// A working issuer: endpoints and signing keys, and the issuer is saved.
	admin.Do("POST", "/admin/sso/"+d.ID+"/discovery", map[string]any{"issuerUrl": idp.Issuer()}).MustStatus(t, 200).JSON(t, &d)
	if !d.Discovery.OK || d.Discovery.Status != 200 || d.Discovery.Keys != 1 || strings.Join(d.Discovery.Endpoints, ",") != "authorization,token,userinfo,JWKS" || d.Issuer != idp.Issuer() {
		t.Fatalf("discovery: %+v issuer %s", d.Discovery, d.Issuer)
	}

	// SSOStates "Discovery failed": a wrong path answers 404.
	wrong := idp.Server.URL + "/organizations/v2.0"
	admin.Do("POST", "/admin/sso/"+d.ID+"/discovery", map[string]any{"issuerUrl": wrong}).MustStatus(t, 200).JSON(t, &d)
	if d.Discovery.OK || d.Discovery.Status != 404 || d.Discovery.Error != "returned 404 Not Found" || d.Discovery.URL != wrong+"/.well-known/openid-configuration" {
		t.Fatalf("404 discovery: %+v", d.Discovery)
	}
	if len(d.EnableBlockers) == 0 || !strings.Contains(d.EnableBlockers[0], "discovery") {
		t.Fatalf("blockers: %v", d.EnableBlockers)
	}
	if p := admin.Do("POST", "/admin/sso/"+d.ID+"/enable", nil).MustStatus(t, 409).Problem(t); p["code"] != "sso_incomplete" {
		t.Fatalf("enable with failed discovery: %v", p)
	}
	// Each test is audited; the issuer change shows before and after.
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'sso.provider.discovery_tested' AND target_id = $1
		AND metadata->>'summary' = 'issuer changed · discovery failed: returned 404 Not Found' AND after->>'issuer' = $2 AND before->>'issuer' = $3`,
		d.ID, wrong, idp.Issuer()); n != 1 {
		t.Fatalf("%d failed discovery tests audited", n)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'sso.provider.discovery_tested' AND target_id = $1
		AND metadata->>'summary' = 'issuer changed · discovery OK · 4 endpoints · 1 signing key'`, d.ID); n != 1 {
		t.Fatalf("%d passed discovery tests audited", n)
	}

	// An issuer that doesn't match its document, plain http off localhost,
	// and a closed port.
	other := ssotest.New()
	defer other.Close()
	other.DiscoveryIssuer = "https://idp.example.com"
	admin.Do("POST", "/admin/sso/"+d.ID+"/discovery", map[string]any{"issuerUrl": other.Issuer()}).MustStatus(t, 200).JSON(t, &d)
	if d.Discovery.OK || !strings.Contains(d.Discovery.Error, "The document's issuer is https://idp.example.com") {
		t.Fatalf("mismatch: %+v", d.Discovery)
	}
	admin.Do("POST", "/admin/sso/"+d.ID+"/discovery", map[string]any{"issuerUrl": "http://idp.example.com"}).MustStatus(t, 200).JSON(t, &d)
	if d.Discovery.OK || !strings.Contains(d.Discovery.Error, "must use https") {
		t.Fatalf("plain http: %+v", d.Discovery)
	}
	admin.Do("POST", "/admin/sso/"+d.ID+"/discovery", map[string]any{"issuerUrl": "http://127.0.0.1:1"}).MustStatus(t, 200).JSON(t, &d)
	if d.Discovery.OK || d.Discovery.Error != "The issuer didn't answer: the connection was refused." {
		t.Fatalf("closed port: %+v", d.Discovery)
	}
	body := providerBody(env, t, idp)
	body["issuerUrl"] = "http://idp.example.com"
	admin.Do("PUT", "/admin/sso/"+d.ID, body).MustStatus(t, 422)

	// A live provider keeps its working issuer when a new one fails.
	live := addProvider(t, env, admin, idp, nil)
	admin.Do("POST", "/admin/sso/"+live.ID+"/discovery", map[string]any{"issuerUrl": wrong}).MustStatus(t, 200).JSON(t, &live)
	if live.Issuer != idp.Issuer() || live.Discovery.OK || live.Discovery.URL != wrong+"/.well-known/openid-configuration" {
		t.Fatalf("live provider: issuer %s, discovery %+v", live.Issuer, live.Discovery)
	}
}

func TestSecretsAreWriteOnly(t *testing.T) {
	env, admin := setup(t)
	idp := ssotest.New()
	defer idp.Close()
	idp.ClientSecret = "s3cr3t-value-for-the-write-only-test"
	d := addProvider(t, env, admin, idp, nil)
	if !d.Secret.Set {
		t.Fatalf("secret not set: %+v", d.Secret)
	}
	r := admin.Do("GET", "/admin/sso/"+d.ID, nil).MustStatus(t, 200)
	if strings.Contains(string(r.Body), idp.ClientSecret) {
		t.Fatal("the client secret was returned")
	}
	// Saving without a secret keeps the saved one: sign-in still works.
	body := providerBody(env, t, idp)
	delete(body, "clientSecret")
	body["allowedDomains"] = []string{"acme-labs.io"}
	admin.Do("PUT", "/admin/sso/"+d.ID, body).MustStatus(t, 200)
	idp.Claims = map[string]any{"sub": "keep-1", "email": "keep@acme-labs.io", "email_verified": true, "name": "Kept Secret", "amr": []string{"mfa"}}
	b := newBrowser(t, env)
	if loc := b.follow("/api/v1/auth/sso/" + d.ID + "/start"); !strings.HasPrefix(loc, "/w/") || b.me() != "keep@acme-labs.io" {
		t.Fatalf("sign-in after saving without the secret: %s (%v)", loc, b.Visited)
	}

	// Social secrets too.
	r = admin.Do("PUT", "/admin/social/github", map[string]any{"enabled": true, "clientId": "Iv23-test", "clientSecret": "gh-secret-for-the-write-only-test", "org": "acme-eng"}).MustStatus(t, 200)
	r2 := admin.Do("GET", "/admin/social", nil).MustStatus(t, 200)
	for _, b := range [][]byte{r.Body, r2.Body} {
		if strings.Contains(string(b), "gh-secret-for-the-write-only-test") || !strings.Contains(string(b), `"secretSet":true`) {
			t.Fatalf("social secret: %s", b)
		}
	}
	for _, secret := range []string{idp.ClientSecret, "gh-secret-for-the-write-only-test"} {
		if n := count(t, env, `SELECT count(*) FROM audit_events WHERE concat(before::text, after::text, metadata::text, reason) LIKE '%' || $1 || '%'`, secret); n != 0 {
			t.Fatalf("%d audit entries contain a secret", n)
		}
		if n := count(t, env, `SELECT count(*) FROM sso_providers WHERE position(convert_to($1, 'UTF8') in coalesce(client_secret_enc, ''::bytea)) > 0`, secret); n != 0 {
			t.Fatal("a client secret is stored in the clear")
		}
		if n := count(t, env, `SELECT count(*) FROM instance_settings WHERE social_providers::text LIKE '%' || $1 || '%'`, secret); n != 0 {
			t.Fatal("a social secret is stored in the clear")
		}
	}
}

func TestSignInCreatesAccountJustInTime(t *testing.T) {
	env, admin := setup(t)
	idp := ssotest.New()
	defer idp.Close()
	d := addProvider(t, env, admin, idp, func(b map[string]any) { b["allowedDomains"] = []string{"acme-labs.io"} })
	idp.Claims = map[string]any{"sub": "jit-1", "email": "New.Person@acme-labs.io", "email_verified": true, "name": "New Person", "amr": []string{"pwd", "mfa"}}

	b := newBrowser(t, env)
	loc := b.follow("/api/v1/auth/sso/" + d.ID + "/start?login_hint=new.person@acme-labs.io&next=/w/acme-newsletter/campaigns")
	if loc != "/w/acme-newsletter/campaigns" || b.me() != "new.person@acme-labs.io" {
		t.Fatalf("landed on %s as %q (%v)", loc, b.me(), b.Visited)
	}
	// State, nonce and PKCE went to the IdP, with the login hint.
	q := query(t, b.Visited[0])
	if q.Get("state") == "" || q.Get("nonce") == "" || q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" || q.Get("login_hint") != "new.person@acme-labs.io" {
		t.Fatalf("authorize request: %s", b.Visited[0])
	}
	if n := count(t, env, `SELECT count(*) FROM workspace_members m JOIN roles r ON r.id = m.role_id JOIN users u ON u.id = m.user_id
		JOIN workspaces w ON w.id = m.workspace_id WHERE u.email = 'new.person@acme-labs.io' AND w.slug = 'acme-newsletter' AND r.key = 'viewer' AND m.source = 'sso_jit'`); n != 1 {
		t.Fatal("no JIT membership as Viewer")
	}
	// The IdP asserted mfa (amr), so the session counts as two-factor.
	if n := count(t, env, `SELECT count(*) FROM sessions s JOIN users u ON u.id = s.user_id WHERE u.email = 'new.person@acme-labs.io' AND s.auth_method = 'sso' AND s.mfa_verified`); n != 1 {
		t.Fatal("no two-factor SSO session")
	}
	for _, action := range []string{"user.created", "member.joined", "signin.succeeded"} {
		if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = $1 AND (target_name = 'new.person@acme-labs.io' OR target_name = 'New Person') AND result = 'allowed'`, action); n != 1 {
			t.Fatalf("%d %s audit entries", n, action)
		}
	}
	// A second sign-in reuses the identity.
	b2 := newBrowser(t, env)
	b2.follow("/api/v1/auth/sso/" + d.ID + "/start")
	if b2.me() != "new.person@acme-labs.io" || count(t, env, `SELECT count(*) FROM users WHERE email = 'new.person@acme-labs.io'`) != 1 ||
		count(t, env, `SELECT count(*) FROM user_identities WHERE subject = 'jit-1'`) != 1 {
		t.Fatal("the second sign-in didn't reuse the account")
	}

	// The instance's JIT allowlist (acme.io, acme-labs.io) still applies.
	body := providerBody(env, t, idp)
	body["allowedDomains"] = []string{"acme-labs.io", "example.com"}
	admin.Do("PUT", "/admin/sso/"+d.ID, body).MustStatus(t, 200)
	idp.Claims = map[string]any{"sub": "jit-2", "email": "someone@example.com", "email_verified": true}
	loc = newBrowser(t, env).follow("/api/v1/auth/sso/" + d.ID + "/start")
	if q := query(t, loc); q.Get("error") != "no_account" || q.Get("email") != "someone@example.com" {
		t.Fatalf("outside the JIT allowlist: %s", loc)
	}
	// And with JIT off nobody new gets in.
	body["jit"] = map[string]any{"enabled": false}
	body["allowedDomains"] = []string{"acme-labs.io"}
	admin.Do("PUT", "/admin/sso/"+d.ID, body).MustStatus(t, 200)
	idp.Claims = map[string]any{"sub": "jit-3", "email": "late@acme-labs.io", "email_verified": true}
	if q := query(t, newBrowser(t, env).follow("/api/v1/auth/sso/"+d.ID+"/start")); q.Get("error") != "no_account" {
		t.Fatalf("JIT off: %v", q)
	}
	if count(t, env, `SELECT count(*) FROM users WHERE email IN ('someone@example.com', 'late@acme-labs.io')`) != 0 {
		t.Fatal("an account was created")
	}
}

func TestSignInLinksVerifiedEmail(t *testing.T) {
	env, admin := setup(t)
	idp := ssotest.New()
	defer idp.Close()
	d := addProvider(t, env, admin, idp, func(b map[string]any) { b["allowedDomains"] = []string{"acme.io"} })

	// Unverified: refused, nothing linked.
	idp.Claims = map[string]any{"sub": "aisha-1", "email": "aisha@acme.io", "email_verified": false, "amr": []string{"mfa"}}
	q := query(t, newBrowser(t, env).follow("/api/v1/auth/sso/"+d.ID+"/start"))
	if q.Get("error") != "idp" || q.Get("code") != "email_not_verified" || !strings.HasPrefix(q.Get("ref"), "auth_") || q.Get("provider") != "Test IdP" {
		t.Fatalf("unverified: %v", q)
	}
	// Verified (or no claim at an allowed domain): linked to Aisha's account.
	idp.Claims = map[string]any{"sub": "aisha-1", "email": "aisha@acme.io", "amr": []string{"mfa"}}
	b := newBrowser(t, env)
	b.follow("/api/v1/auth/sso/" + d.ID + "/start")
	if b.me() != "aisha@acme.io" {
		t.Fatalf("not signed in as Aisha: %v", b.Visited)
	}
	if count(t, env, `SELECT count(*) FROM user_identities i JOIN users u ON u.id = i.user_id WHERE u.email = 'aisha@acme.io' AND i.kind = 'sso' AND i.sso_provider_id = $1 AND i.subject = 'aisha-1'`, d.ID) != 1 ||
		count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'account.identity_link' AND target_name = 'aisha@acme.io'`) != 1 {
		t.Fatal("identity not linked and audited")
	}
	// A deactivated account can't sign in.
	idp.Claims = map[string]any{"sub": "tom-1", "email": "tom@acme.io", "email_verified": true, "amr": []string{"mfa"}}
	if q := query(t, newBrowser(t, env).follow("/api/v1/auth/sso/"+d.ID+"/start")); q.Get("error") != "deactivated" || q.Get("email") != "tom@acme.io" {
		t.Fatalf("deactivated: %v", q)
	}
}

func TestGroupMappings(t *testing.T) {
	env, admin := setup(t)
	idp := ssotest.New()
	defer idp.Close()
	newsletter, eng := workspaceID(t, env, "acme-newsletter").String(), workspaceID(t, env, "acme-engineering-blog").String()
	mappings := []map[string]any{
		{"group": "g-marketing", "label": "Marketing", "workspaceId": newsletter, "roleKey": "editor"},
		{"group": "g-eng-leads", "label": "Eng Leads", "workspaceId": eng, "roleKey": "admin"},
	}
	d := addProvider(t, env, admin, idp, func(b map[string]any) {
		b["allowedDomains"] = []string{"acme.io"}
		b["groupMappings"] = mappings
	})
	role := func(email, slug string) string {
		var key, source string
		err := env.App.Pool.QueryRow(context.Background(), `SELECT r.key, m.source FROM workspace_members m JOIN roles r ON r.id = m.role_id JOIN users u ON u.id = m.user_id
			JOIN workspaces w ON w.id = m.workspace_id WHERE u.email = $1 AND w.slug = $2`, email, slug).Scan(&key, &source)
		if err != nil {
			return ""
		}
		return key + "/" + source
	}
	signIn := func(email string, groups ...string) {
		t.Helper()
		idp.Claims = map[string]any{"sub": "sub-" + email, "email": email, "email_verified": true, "groups": groups, "amr": []string{"mfa"}}
		b := newBrowser(t, env)
		b.follow("/api/v1/auth/sso/" + d.ID + "/start")
		if b.me() != email {
			t.Fatalf("%s didn't sign in: %v", email, b.Visited)
		}
	}
	// Sofia joins Acme Engineering Blog as Admin; Daniel moves from
	// Content Writer to Editor; Marcus (Admin) is mapped down to Editor.
	signIn("sofia@acme.io", "g-eng-leads")
	signIn("daniel@acme.io", "g-marketing")
	signIn("marcus@acme.io", "g-marketing", "unrelated")
	if got := role("sofia@acme.io", "acme-engineering-blog"); got != "admin/sso_group" {
		t.Fatalf("sofia: %s", got)
	}
	if got := role("sofia@acme.io", "acme-newsletter"); got != "editor/sso_jit" {
		t.Fatalf("sofia keeps her newsletter role: %s", got)
	}
	if got := role("daniel@acme.io", "acme-newsletter"); got != "editor/sso_group" {
		t.Fatalf("daniel: %s", got)
	}
	if got := role("marcus@acme.io", "acme-newsletter"); got != "editor/sso_group" {
		t.Fatalf("marcus: %s", got)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'member.role_changed' AND metadata->>'source' = 'sso_group' AND reason LIKE 'Test IdP group “Marketing” maps to Editor'`); n != 2 {
		t.Fatalf("%d role change audit entries", n)
	}
	// Kenji, an Owner, keeps Owner even when a group maps him lower.
	signIn("kenji@acme.io", "g-eng-leads")
	if got := role("kenji@acme.io", "acme-engineering-blog"); !strings.HasPrefix(got, "owner/") {
		t.Fatalf("kenji: %s", got)
	}

	// Mappings are re-applied at every sign-in: leaving a group ends the
	// role it gave. Outside the JIT workspace the membership goes; in it,
	// the JIT default role applies.
	signIn("sofia@acme.io")
	if got := role("sofia@acme.io", "acme-engineering-blog"); got != "" {
		t.Fatalf("sofia left Eng Leads but stays in the blog: %s", got)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'member.removed' AND metadata->>'group' = 'Eng Leads' AND actor_type = 'system'
		AND reason = 'no longer in Test IdP group “Eng Leads”, which granted Admin'`); n != 1 {
		t.Fatalf("%d removals audited", n)
	}
	if got := role("sofia@acme.io", "acme-newsletter"); got != "editor/sso_jit" {
		t.Fatalf("sofia's newsletter role didn't come from a group: %s", got)
	}
	signIn("marcus@acme.io", "unrelated")
	if got := role("marcus@acme.io", "acme-newsletter"); got != "viewer/sso_jit" {
		t.Fatalf("marcus falls back to the JIT default: %s", got)
	}
	// A role someone set by hand after the mapping stays.
	signIn("sofia@acme.io", "g-eng-leads")
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE workspace_members m SET role_id = r.id FROM roles r, users u
		WHERE r.workspace_id = m.workspace_id AND r.key = 'editor' AND u.id = m.user_id AND u.email = 'sofia@acme.io' AND m.workspace_id = $1`, eng); err != nil {
		t.Fatal(err)
	}
	signIn("sofia@acme.io")
	if got := role("sofia@acme.io", "acme-engineering-blog"); got != "editor/sso_group" {
		t.Fatalf("a role set by hand changed: %s", got)
	}
	// Without a groups claim the person's groups aren't known: nothing changes.
	idp.Claims = map[string]any{"sub": "sub-daniel@acme.io", "email": "daniel@acme.io", "email_verified": true, "amr": []string{"mfa"}}
	b := newBrowser(t, env)
	b.follow("/api/v1/auth/sso/" + d.ID + "/start")
	if b.me() != "daniel@acme.io" {
		t.Fatalf("daniel didn't sign in: %v", b.Visited)
	}
	if got := role("daniel@acme.io", "acme-newsletter"); got != "editor/sso_group" {
		t.Fatalf("daniel without a groups claim: %s", got)
	}

	// Owner can never be mapped, and the escalation guard stops admins
	// from granting roles they don't hold.
	body := providerBody(env, t, idp)
	body["allowedDomains"] = []string{"acme.io"}
	// A scope only takes lists of the mapping's workspace.
	var digest string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT id::text FROM lists WHERE workspace_id = $1 AND name = 'Weekly Digest'`, newsletter).Scan(&digest); err != nil {
		t.Fatal(err)
	}
	body["groupMappings"] = []map[string]any{{"group": "g-x", "workspaceId": eng, "roleKey": "viewer", "lists": []string{digest}}}
	if p := admin.Do("PUT", "/admin/sso/"+d.ID, body).MustStatus(t, 422).Problem(t); !strings.Contains(fmt.Sprint(p), "Pick lists of Acme Engineering Blog") {
		t.Fatalf("list of another workspace: %v", p)
	}
	body["groupMappings"] = []map[string]any{{"group": "g-x", "workspaceId": newsletter, "roleKey": "owner"}}
	if p := admin.Do("PUT", "/admin/sso/"+d.ID, body).MustStatus(t, 403).Problem(t); p["code"] != "escalation_denied" {
		t.Fatalf("owner mapping: %v", p)
	}
	env.CreateUser(apptest.UserOpts{Email: "root@example.com", Name: "Root Admin", InstanceAdmin: true})
	root := env.Client()
	root.SignInDemo("root@example.com")
	var rd detail
	root.Do("GET", "/admin/sso/"+d.ID, nil).MustStatus(t, 200).JSON(t, &rd)
	for _, w := range rd.Workspaces {
		for _, r := range w.Roles {
			if r.Grantable {
				t.Fatalf("root can grant %s in %s", r.Key, w.Slug)
			}
		}
	}
	// Root may keep the mappings Linh made, but not add one.
	body["groupMappings"] = mappings
	root.Do("PUT", "/admin/sso/"+d.ID, body).MustStatus(t, 200)
	body["groupMappings"] = append(mappings, map[string]any{"group": "g-new", "workspaceId": eng, "roleKey": "viewer"})
	p := root.Do("PUT", "/admin/sso/"+d.ID, body).MustStatus(t, 403).Problem(t)
	if p["code"] != "escalation_denied" || !strings.Contains(p["detail"].(string), "aren't a member of Acme Engineering Blog") {
		t.Fatalf("escalation: %v", p)
	}
}

func TestSignInFailures(t *testing.T) {
	env, admin := setup(t)
	idp := ssotest.New()
	defer idp.Close()
	d := addProvider(t, env, admin, idp, func(b map[string]any) { b["allowedDomains"] = []string{"acme-labs.io"} })
	idp.Claims = map[string]any{"sub": "state-1", "email": "state@acme-labs.io", "email_verified": true, "amr": []string{"mfa"}}

	// A domain the provider doesn't allow: "No account for this email",
	// and the rejection is listed with its reference.
	idp.Claims["email"] = "kim@gmail.com"
	loc := newBrowser(t, env).follow("/api/v1/auth/sso/" + d.ID + "/start")
	if q := query(t, loc); q.Get("error") != "no_account" || q.Get("email") != "kim@gmail.com" {
		t.Fatalf("domain: %s", loc)
	}
	var got detail
	admin.Do("GET", "/admin/sso/"+d.ID, nil).MustStatus(t, 200).JSON(t, &got)
	if len(got.Rejections) != 1 || got.Rejections[0].Email != "kim@gmail.com" || got.Rejections[0].Reason != "email domain not allowed for Test IdP SSO" || !strings.HasPrefix(got.Rejections[0].Reference, "auth_") {
		t.Fatalf("rejections: %+v", got.Rejections)
	}
	idp.Claims["email"] = "state@acme-labs.io"

	// The callback only works once, in the browser that started it.
	a := newBrowser(t, env)
	start := a.get("/api/v1/auth/sso/" + d.ID + "/start")
	callback := a.get(start.Location).Location
	other := newBrowser(t, env)
	if q := query(t, other.get(callback).Location); q.Get("code") != "state_mismatch" {
		t.Fatalf("other browser: %v", q)
	}
	if q := query(t, a.get(callback).Location); q.Get("code") != "state_expired" {
		t.Fatalf("used state: %v", q)
	}
	if other.me() != "" || a.me() != "" {
		t.Fatal("someone got signed in")
	}
	// A token for another sign-in (nonce), an IdP error, a disabled provider.
	idp.Nonce = "not-this-sign-in"
	if q := query(t, newBrowser(t, env).follow("/api/v1/auth/sso/"+d.ID+"/start")); q.Get("code") != "nonce_mismatch" {
		t.Fatalf("nonce: %v", q)
	}
	idp.Nonce, idp.Error = "", "access_denied"
	if q := query(t, newBrowser(t, env).follow("/api/v1/auth/sso/"+d.ID+"/start")); q.Get("code") != "access_denied" || q.Get("provider") != "Test IdP" {
		t.Fatalf("idp error: %v", q)
	}
	idp.Error = ""
	admin.Do("POST", "/admin/sso/"+d.ID+"/disable", nil).MustStatus(t, 200)
	if q := query(t, newBrowser(t, env).follow("/api/v1/auth/sso/"+d.ID+"/start")); q.Get("code") != "provider_disabled" {
		t.Fatalf("disabled: %v", q)
	}
	// Every failure is in the audit log with a reference.
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'signin.rejected' AND result = 'denied' AND metadata->>'reference' LIKE 'auth_%'`); n < 6 {
		t.Fatalf("%d denied sign-ins audited", n)
	}
	if n := count(t, env, `SELECT count(*) FROM users WHERE email IN ('state@acme-labs.io', 'kim@gmail.com')`); n != 0 {
		t.Fatal("a failed sign-in created an account")
	}
}

func TestSignInAsksForOwnSecondFactor(t *testing.T) {
	env, admin := setup(t)
	idp := ssotest.New()
	defer idp.Close()
	d := addProvider(t, env, admin, idp, func(b map[string]any) { b["allowedDomains"] = []string{"acme.io"} })
	// No amr: Linh confirms with her own second factor.
	idp.Claims = map[string]any{"sub": "linh-1", "email": "linh@acme.io", "email_verified": true, "amr": []string{"pwd"}}
	b := newBrowser(t, env)
	q := query(t, b.follow("/api/v1/auth/sso/"+d.ID+"/start"))
	if q.Get("challenge") == "" || !strings.Contains(q.Get("methods"), "totp") || q.Get("email") != "linh@acme.io" || q.Get("provider") != "Test IdP" {
		t.Fatalf("two-factor step: %v", q)
	}
	if b.me() != "" {
		t.Fatal("signed in before the second factor")
	}
	var res struct{ Status string }
	if s := b.do("POST", "/api/v1/auth/two-factor", map[string]any{"challenge": q.Get("challenge"), "method": "recovery", "code": demo.DemoRecoveryCodes[0]}, &res); s != 200 || res.Status != "signed_in" {
		t.Fatalf("two-factor: %d %+v", s, res)
	}
	if b.me() != "linh@acme.io" {
		t.Fatal("not signed in after the second factor")
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'signin.succeeded' AND target_name = 'linh@acme.io' AND metadata->>'method' = 'sso+recovery' AND metadata->>'provider' = 'Test IdP'`); n != 1 {
		t.Fatalf("%d sso+recovery sign-ins audited", n)
	}
	// With the IdP trusted for two-factor, no second step.
	body := providerBody(env, t, idp)
	body["allowedDomains"] = []string{"acme.io"}
	body["twoFactorMode"] = "trusted"
	admin.Do("PUT", "/admin/sso/"+d.ID, body).MustStatus(t, 200)
	b2 := newBrowser(t, env)
	if loc := b2.follow("/api/v1/auth/sso/" + d.ID + "/start"); strings.Contains(loc, "challenge") || b2.me() != "linh@acme.io" {
		t.Fatalf("trusted IdP: %s", loc)
	}
}

func TestTestSignIn(t *testing.T) {
	env, admin := setup(t)
	idp := ssotest.New()
	defer idp.Close()
	var d detail
	admin.Do("POST", "/admin/sso", map[string]any{"preset": "entra"}).MustStatus(t, 201).JSON(t, &d)
	// Not ready: no issuer, client or secret.
	if p := admin.Do("POST", "/admin/sso/"+d.ID+"/test", map[string]any{"workspace": "acme-newsletter"}).MustStatus(t, 409).Problem(t); p["code"] != "sso_incomplete" {
		t.Fatalf("incomplete: %v", p)
	}
	body := providerBody(env, t, idp)
	body["name"] = "Microsoft Entra ID"
	body["allowedDomains"] = []string{"acme.io"}
	body["groupMappings"] = []map[string]any{{"group": demo.EntraGroupMarketing, "label": "Marketing", "workspaceId": workspaceID(t, env, "acme-newsletter").String(), "roleKey": "editor"}}
	admin.Do("PUT", "/admin/sso/"+d.ID, body).MustStatus(t, 200)
	admin.Do("POST", "/admin/sso/"+d.ID+"/discovery", map[string]any{}).MustStatus(t, 200)
	users := count(t, env, `SELECT count(*) FROM users`)

	run := func() detail {
		t.Helper()
		r := admin.Do("POST", "/admin/sso/"+d.ID+"/test", map[string]any{"workspace": "acme-newsletter"}).MustStatus(t, 200)
		var start struct {
			AuthorizeURL string `json:"authorizeUrl"`
		}
		r.JSON(t, &start)
		b := newBrowser(t, env)
		b.setCookies(r.Header)
		if q := query(t, start.AuthorizeURL); q.Get("prompt") != "login" || q.Get("code_challenge") == "" {
			t.Fatalf("test authorize URL: %s", start.AuthorizeURL)
		}
		if loc := b.follow(start.AuthorizeURL); loc != "/w/acme-newsletter/admin/sso/"+d.ID+"?test=done" {
			t.Fatalf("test returned to %s", loc)
		}
		if b.me() != "" {
			t.Fatal("a test sign-in signed someone in")
		}
		var out detail
		admin.Do("GET", "/admin/sso/"+d.ID, nil).MustStatus(t, 200).JSON(t, &out)
		return out
	}

	// SSOStates "Groups claim missing": Marcus without a groups claim.
	idp.Claims = map[string]any{"sub": "marcus-entra", "email": "marcus@acme.io", "name": "Marcus Chen", "preferred_username": "marcus@acme.io", "amr": []string{"pwd", "mfa"}}
	got := run()
	if got.Test == nil || got.Test.Error != "" || !got.Test.GroupsMissing || !strings.Contains(got.Test.Claims, `"email": "marcus@acme.io"`) || strings.Contains(got.Test.Claims, "nonce") {
		t.Fatalf("test view: %+v", got.Test)
	}
	if o := got.outcome("Existing account"); o != "ok: marcus@acme.io matches Marcus Chen · no account created" {
		t.Fatalf("account outcome: %q", o)
	}
	if o := got.outcome("Mappings inactive"); o != "warn: No groups claim · the group mapping doesn't apply" {
		t.Fatalf("mappings outcome: %q", o)
	}
	for _, s := range got.Test.Samples {
		if s.Field == "groups" && (s.Found || s.Missing != "not sent by Entra ID") {
			t.Fatalf("groups sample: %+v", s)
		}
	}
	// With the group: Marcus (Admin) would become Editor, flagged.
	idp.Claims["groups"] = []string{demo.EntraGroupMarketing}
	got = run()
	if got.Test.GroupsMissing || !strings.HasPrefix(got.outcome("Role change"), "warn: Acme Newsletter: Admin → Editor from group “Marketing”") {
		t.Fatalf("role change outcome: %+v", got.Test.Outcome)
	}
	// An IdP error is shown, not a sign-in.
	idp.Error = "access_denied"
	got = run()
	if got.Test.Error != "Microsoft Entra ID returned access_denied: The fake IdP was told to fail" || got.Test.Claims != "" {
		t.Fatalf("test error: %+v", got.Test)
	}
	if count(t, env, `SELECT count(*) FROM users`) != users || count(t, env, `SELECT count(*) FROM user_identities WHERE sso_provider_id = $1`, d.ID) != 0 {
		t.Fatal("a test sign-in created an account or an identity")
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'sso.provider.tested' AND target_id = $1`, d.ID); n != 3 {
		t.Fatalf("%d test sign-ins audited", n)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'sso.provider.tested' AND result = 'error'`); n != 1 {
		t.Fatalf("%d failed test sign-ins audited", n)
	}
}

func TestDomainEnforcement(t *testing.T) {
	env, admin := setup(t)
	idp := ssotest.New()
	defer idp.Close()
	d := addProvider(t, env, admin, idp, func(b map[string]any) { b["allowedDomains"] = []string{"acme.io"} })

	var pv struct {
		Members []struct {
			Email, Today, After string
			BreakGlass          bool
		} `json:"members"`
		BreakGlass []struct{ Name, Email string } `json:"breakGlass"`
		Conflict   string                         `json:"conflict"`
	}
	admin.Do("GET", "/admin/sso/"+d.ID+"/enforcement?domain=acme.io", nil).MustStatus(t, 200).JSON(t, &pv)
	glass := []string{}
	for _, o := range pv.BreakGlass {
		glass = append(glass, o.Email)
	}
	if strings.Join(glass, ",") != "linh@acme.io,kenji@acme.io" && strings.Join(glass, ",") != "kenji@acme.io,linh@acme.io" {
		t.Fatalf("break-glass: %v", glass)
	}
	for _, m := range pv.Members {
		if m.Email == "aisha@acme.io" && (m.Today != "Password" || m.After != "Test IdP only") {
			t.Fatalf("aisha: %+v", m)
		}
		if m.Email == "tom@acme.io" {
			t.Fatal("deactivated people are listed")
		}
	}

	admin.Do("PUT", "/admin/sso/"+d.ID+"/enforcement", map[string]any{"domains": []string{"acme.io"}}).MustStatus(t, 422)
	admin.Do("PUT", "/admin/sso/"+d.ID+"/enforcement", map[string]any{"domains": []string{"acme-labs.io"}, "confirm": true}).MustStatus(t, 422)
	var got detail
	admin.Do("PUT", "/admin/sso/"+d.ID+"/enforcement", map[string]any{"domains": []string{"acme.io"}, "confirm": true}).MustStatus(t, 200).JSON(t, &got)
	if strings.Join(got.Enforced, ",") != "acme.io" {
		t.Fatalf("enforced: %v", got.Enforced)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'sso.enforcement.enabled' AND target_id = $1`, d.ID); n != 1 {
		t.Fatal("enforcement not audited")
	}

	// Aisha's password stops working; identify sends her to the IdP.
	c := env.Client()
	p := c.Do("POST", "/auth/password", map[string]any{"email": "aisha@acme.io", "password": demo.Password}).MustStatus(t, 403).Problem(t)
	if p["code"] != "sso_required" {
		t.Fatalf("password under enforcement: %v", p)
	}
	var idf struct{ Next string }
	c.Do("POST", "/auth/identify", map[string]any{"email": "aisha@acme.io"}).MustStatus(t, 200).JSON(t, &idf)
	if idf.Next != "sso" {
		t.Fatalf("identify: %+v", idf)
	}
	// Linh is break-glass: her password still works and is logged as such.
	var step struct{ Status, Challenge string }
	c.Do("POST", "/auth/password", map[string]any{"email": "linh@acme.io", "password": demo.Password}).MustStatus(t, 200).JSON(t, &step)
	if step.Status != "two_factor" {
		t.Fatalf("break-glass password: %+v", step)
	}
	c.Do("POST", "/auth/two-factor", map[string]any{"challenge": step.Challenge, "method": "recovery", "code": demo.DemoRecoveryCodes[0]}).MustStatus(t, 200)
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'signin.succeeded' AND target_name = 'linh@acme.io' AND metadata->>'break_glass' = 'true' AND metadata->>'enforced_by' = 'Test IdP'`); n != 1 {
		t.Fatalf("%d break-glass sign-ins logged", n)
	}

	// Social sign-in at the domain goes through the enforcing IdP.
	google := ssotest.New()
	defer google.Close()
	socialEndpoints(t, sso.SocialEndpoints{GoogleIssuer: google.Issuer()})
	admin.Do("PUT", "/admin/social/google", map[string]any{"enabled": true, "clientId": google.ClientID, "clientSecret": google.ClientSecret}).MustStatus(t, 200)
	google.Claims = map[string]any{"sub": "g-aisha", "email": "aisha@acme.io", "email_verified": true}
	idp.Claims = map[string]any{"sub": "sso-aisha", "email": "aisha@acme.io", "email_verified": true, "amr": []string{"mfa"}}
	b := newBrowser(t, env)
	b.follow("/api/v1/auth/social/google/start")
	viaIdP := false
	for _, v := range b.Visited {
		if strings.HasPrefix(v, idp.Issuer()+"/auth") {
			viaIdP = true
		}
	}
	if !viaIdP || b.me() != "aisha@acme.io" || count(t, env, `SELECT count(*) FROM sessions s JOIN users u ON u.id = s.user_id WHERE u.email = 'aisha@acme.io' AND s.auth_method = 'sso'`) != 1 {
		t.Fatalf("social under enforcement: %v", b.Visited)
	}

	// A second provider can't enforce the same domain.
	other := ssotest.New()
	defer other.Close()
	second := addProvider(t, env, admin, other, func(b map[string]any) { b["name"] = "Second IdP"; b["allowedDomains"] = []string{"acme.io"} })
	admin.Do("PUT", "/admin/sso/"+second.ID+"/enforcement", map[string]any{"domains": []string{"acme.io"}, "confirm": true}).MustStatus(t, 409)

	// Disabling the provider ends enforcement.
	admin.Do("POST", "/admin/sso/"+d.ID+"/disable", nil).MustStatus(t, 200).JSON(t, &got)
	if len(got.Enforced) != 0 {
		t.Fatalf("still enforced: %v", got.Enforced)
	}
	env.Client().Do("POST", "/auth/password", map[string]any{"email": "aisha@acme.io", "password": demo.Password}).MustStatus(t, 200)
}

func TestSocialGoogle(t *testing.T) {
	env, admin := setup(t)
	google := ssotest.New()
	defer google.Close()
	socialEndpoints(t, sso.SocialEndpoints{GoogleIssuer: google.Issuer()})
	admin.Do("PUT", "/admin/social/google", map[string]any{"enabled": true, "clientId": google.ClientID, "clientSecret": google.ClientSecret}).MustStatus(t, 200)

	// A verified Google email signs in to the account with that email.
	google.Claims = map[string]any{"sub": "g-aisha", "email": "aisha@acme.io", "email_verified": true, "name": "Aisha Bello"}
	b := newBrowser(t, env)
	if loc := b.follow("/api/v1/auth/social/google/start"); !strings.HasPrefix(loc, "/w/") || b.me() != "aisha@acme.io" {
		t.Fatalf("google sign-in: %s %v", loc, b.Visited)
	}
	if count(t, env, `SELECT count(*) FROM user_identities i JOIN users u ON u.id = i.user_id WHERE u.email = 'aisha@acme.io' AND i.kind = 'google' AND i.subject = 'g-aisha'`) != 1 {
		t.Fatal("google identity not linked")
	}
	// Social sign-in never creates an account, and unverified emails don't link.
	for _, claims := range []map[string]any{
		{"sub": "g-new", "email": "nobody@example.com", "email_verified": true},
		{"sub": "g-sofia", "email": "sofia@acme.io", "email_verified": false},
	} {
		google.Claims = claims
		if q := query(t, newBrowser(t, env).follow("/api/v1/auth/social/google/start")); q.Get("error") != "no_account" {
			t.Fatalf("%v: %v", claims, q)
		}
	}
	if count(t, env, `SELECT count(*) FROM users WHERE email = 'nobody@example.com'`) != 0 {
		t.Fatal("social sign-in created an account")
	}
	// Turned off: the start link explains.
	admin.Do("PUT", "/admin/social/google", map[string]any{"enabled": false, "clientId": google.ClientID}).MustStatus(t, 200)
	if q := query(t, newBrowser(t, env).follow("/api/v1/auth/social/google/start")); q.Get("code") != "provider_disabled" {
		t.Fatalf("disabled: %v", q)
	}
	// Enabling needs a client ID and secret.
	admin.Do("PUT", "/admin/social/microsoft", map[string]any{"enabled": true, "clientId": "x"}).MustStatus(t, 422)
}

func TestSocialGitHubOrganization(t *testing.T) {
	env, admin := setup(t)
	gh := ssotest.NewGitHub()
	defer gh.Close()
	socialEndpoints(t, sso.SocialEndpoints{GitHubAuthURL: gh.AuthURL(), GitHubTokenURL: gh.TokenURL(), GitHubAPI: gh.API()})
	admin.Do("PUT", "/admin/social/github", map[string]any{"enabled": true, "clientId": gh.ClientID, "clientSecret": gh.ClientSecret, "org": "not a valid org"}).MustStatus(t, 422)
	admin.Do("PUT", "/admin/social/github", map[string]any{"enabled": true, "clientId": gh.ClientID, "clientSecret": gh.ClientSecret, "org": "acme-eng"}).MustStatus(t, 200)
	gh.Emails = []map[string]any{{"email": "old@example.com", "primary": false, "verified": true}, {"email": "aisha@acme.io", "primary": true, "verified": true}}
	gh.User = map[string]any{"id": 777, "login": "aisha-b", "name": "Aisha Bello"}

	gh.Orgs = []string{"other-org"}
	if q := query(t, newBrowser(t, env).follow("/api/v1/auth/social/github/start")); q.Get("code") != "not_in_org" {
		t.Fatalf("not in org: %v", q)
	}
	gh.OrgStatus = 403
	if q := query(t, newBrowser(t, env).follow("/api/v1/auth/social/github/start")); q.Get("code") != "org_restricted" {
		t.Fatalf("org restricts apps: %v", q)
	}
	gh.OrgStatus, gh.Orgs = 0, []string{"acme-eng"}
	b := newBrowser(t, env)
	b.follow("/api/v1/auth/social/github/start")
	if b.me() != "aisha@acme.io" {
		t.Fatalf("github sign-in: %v", b.Visited)
	}
	if n := count(t, env, `SELECT count(*) FROM user_identities WHERE kind = 'github' AND subject = '777' AND username = 'aisha-b' AND details->>'org' = 'acme-eng'`); n != 1 {
		t.Fatal("github identity not linked with its org")
	}
	// Only a verified primary email counts.
	gh.User = map[string]any{"id": 778, "login": "someone"}
	gh.Emails = []map[string]any{{"email": "sofia@acme.io", "primary": true, "verified": false}}
	if q := query(t, newBrowser(t, env).follow("/api/v1/auth/social/github/start")); q.Get("code") != "email_not_verified" {
		t.Fatalf("unverified primary: %v", q)
	}
}

func TestSocialMicrosoftNeverLinksByEmail(t *testing.T) {
	env, admin := setup(t)
	ms := ssotest.New()
	defer ms.Close()
	ms.Path = "/common/v2.0"
	ms.DiscoveryIssuer = ms.Server.URL + "/{tenantid}/v2.0"
	ms.TokenIssuer = ms.Server.URL + "/tid-1/v2.0"
	socialEndpoints(t, sso.SocialEndpoints{MicrosoftBase: ms.Server.URL})
	admin.Do("PUT", "/admin/social/microsoft", map[string]any{"enabled": true, "clientId": ms.ClientID, "clientSecret": ms.ClientSecret, "tenant": "common"}).MustStatus(t, 200)
	ms.Claims = map[string]any{"sub": "ms-aisha", "tid": "tid-1", "email": "aisha@acme.io", "preferred_username": "aisha@acme.io", "name": "Aisha Bello"}

	// Microsoft's email claim isn't verified: no link by email (nOAuth).
	if q := query(t, newBrowser(t, env).follow("/api/v1/auth/social/microsoft/start")); q.Get("error") != "no_account" {
		t.Fatalf("linked by an unverified email: %v", q)
	}
	// Aisha links it herself from Account & security, then it signs her in.
	b := newBrowser(t, env)
	b.signInAs("aisha@acme.io")
	if loc := b.follow("/api/v1/auth/social/microsoft/start?intent=link"); !strings.HasSuffix(loc, "/account?linked=microsoft") {
		t.Fatalf("link: %s", loc)
	}
	b2 := newBrowser(t, env)
	b2.follow("/api/v1/auth/social/microsoft/start")
	if b2.me() != "aisha@acme.io" {
		t.Fatalf("microsoft sign-in after linking: %v", b2.Visited)
	}
	// A token whose issuer isn't its tenant's is refused.
	ms.TokenIssuer = ms.Server.URL + "/tid-2/v2.0"
	if q := query(t, newBrowser(t, env).follow("/api/v1/auth/social/microsoft/start")); q.Get("code") != "invalid_issuer" {
		t.Fatalf("issuer: %v", q)
	}
}

func TestInvitationWithSSO(t *testing.T) {
	env, admin := setup(t)
	idp := ssotest.New()
	defer idp.Close()
	d := addProvider(t, env, admin, idp, func(b map[string]any) {
		b["allowedDomains"] = []string{"acme-labs.io"}
		b["jit"] = map[string]any{"enabled": false}
	})
	token := invite(t, env, "acme-engineering-blog", "new.hire@acme-labs.io", "editor")

	// Another account can't take the invitation.
	idp.Claims = map[string]any{"sub": "nh-0", "email": "someone.else@acme-labs.io", "email_verified": true, "amr": []string{"mfa"}}
	if q := query(t, newBrowser(t, env).follow("/api/v1/auth/sso/"+d.ID+"/start?invite="+token)); q.Get("code") != "invite_wrong_account" {
		t.Fatalf("wrong account: %v", q)
	}
	// The invitee joins with their IdP account, even with JIT off.
	idp.Claims = map[string]any{"sub": "nh-1", "email": "new.hire@acme-labs.io", "email_verified": true, "name": "New Hire", "amr": []string{"mfa"}}
	b := newBrowser(t, env)
	if loc := b.follow("/api/v1/auth/sso/" + d.ID + "/start?invite=" + token); loc != "/w/acme-engineering-blog" || b.me() != "new.hire@acme-labs.io" {
		t.Fatalf("invite: %s (%v)", loc, b.Visited)
	}
	if n := count(t, env, `SELECT count(*) FROM workspace_members m JOIN roles r ON r.id = m.role_id JOIN users u ON u.id = m.user_id WHERE u.email = 'new.hire@acme-labs.io' AND r.key = 'editor' AND m.source = 'invite'`); n != 1 {
		t.Fatal("invitee didn't join as Editor")
	}
	if n := count(t, env, `SELECT count(*) FROM user_identities WHERE sso_provider_id = $1 AND subject = 'nh-1'`, d.ID); n != 1 {
		t.Fatal("invitee's identity not linked")
	}
}

func TestInstanceAdminsOnly(t *testing.T) {
	env, _ := setup(t)
	for _, email := range []string{"aisha@acme.io", "daniel@acme.io", "marcus@acme.io"} {
		c := env.Client()
		c.SignInDemo(email)
		for _, r := range []struct{ method, path string }{
			{"GET", "/admin/sso"}, {"POST", "/admin/sso"}, {"GET", "/admin/social"}, {"PUT", "/admin/social/github"},
		} {
			var body any
			if r.method != "GET" {
				body = map[string]any{"preset": "oidc", "enabled": false, "clientId": ""}
			}
			if p := c.Do(r.method, r.path, body).MustStatus(t, 403).Problem(t); p["code"] != "permission_denied" {
				t.Fatalf("%s %s %s: %v", email, r.method, r.path, p)
			}
		}
	}
	env.Client().Do("GET", "/admin/sso", nil).MustStatus(t, 401)
}
