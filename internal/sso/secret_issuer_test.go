package sso_test

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/sso/ssotest"
)

// The client secret goes to the issuer's token endpoint, so a saved one is
// sent only to the issuer it was saved with: saving another issuer needs
// the secret pasted again, and Test discovery tries another issuer without
// saving it.
func TestSavedSecretStaysWithItsIssuer(t *testing.T) {
	env, admin := setup(t)
	idp := ssotest.New()
	defer idp.Close()
	other := ssotest.New()
	defer other.Close()
	// The instance lets SSO create accounts for acme.io and acme-labs.io.
	d := addProvider(t, env, admin, idp, func(b map[string]any) { b["allowedDomains"] = []string{"acme-labs.io"} })

	body := providerBody(env, t, idp)
	body["allowedDomains"] = []string{"acme-labs.io"}
	delete(body, "clientSecret")
	body["issuerUrl"] = other.Issuer()
	r := admin.Do("PUT", "/admin/sso/"+d.ID, body).MustStatus(t, 422)
	var p struct {
		Errors []struct{ Location, Message string }
	}
	r.JSON(t, &p)
	if len(p.Errors) != 1 || p.Errors[0].Location != "body.clientSecret" ||
		p.Errors[0].Message != "The issuer changed: paste the client secret again. ReputeCast doesn't send a saved secret to a new issuer." {
		t.Fatalf("another issuer without the secret: %s", r.Body)
	}

	// Test discovery answers for the other issuer but keeps the saved one.
	admin.Do("POST", "/admin/sso/"+d.ID+"/discovery", map[string]any{"issuerUrl": other.Issuer()}).MustStatus(t, 200).JSON(t, &d)
	if d.Issuer != idp.Issuer() || d.Discovery == nil || !d.Discovery.OK || !strings.HasPrefix(d.Discovery.URL, other.Issuer()) {
		t.Fatalf("discovery of another issuer: issuer %s, %+v", d.Issuer, d.Discovery)
	}
	var saved detail
	admin.Do("GET", "/admin/sso/"+d.ID, nil).MustStatus(t, 200).JSON(t, &saved)
	if saved.Issuer != idp.Issuer() || saved.Discovery == nil || !strings.HasPrefix(saved.Discovery.URL, idp.Issuer()) {
		t.Fatalf("saved: issuer %s, %+v", saved.Issuer, saved.Discovery)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE action = 'sso.provider.discovery_tested' AND target_id = $1
		AND metadata->>'summary' = 'new issuer tested, not saved · discovery OK · 4 endpoints · 1 signing key' AND metadata->>'issuer' = $2 AND after->>'issuer' IS NULL`,
		d.ID, other.Issuer()); n != 1 {
		t.Fatalf("%d discovery tests of the other issuer audited", n)
	}

	// Sign-in still uses the saved issuer and secret.
	idp.Claims = map[string]any{"sub": "stay-1", "email": "stay@acme-labs.io", "email_verified": true, "name": "Stay Put", "amr": []string{"mfa"}}
	b := newBrowser(t, env)
	if loc := b.follow("/api/v1/auth/sso/" + d.ID + "/start"); !strings.HasPrefix(loc, "/w/") || b.me() != "stay@acme-labs.io" {
		t.Fatalf("sign-in after the refused change: %s (%v)", loc, b.Visited)
	}

	// The same issuer keeps the saved secret; another one with its secret
	// pasted is saved.
	body["issuerUrl"] = idp.Issuer()
	admin.Do("PUT", "/admin/sso/"+d.ID, body).MustStatus(t, 200)
	body["issuerUrl"], body["clientSecret"] = other.Issuer(), other.ClientSecret
	admin.Do("PUT", "/admin/sso/"+d.ID, body).MustStatus(t, 200).JSON(t, &saved)
	if saved.Issuer != other.Issuer() {
		t.Fatalf("issuer with its secret: %s", saved.Issuer)
	}
}
