package publicpages_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/linktoken"
)

// The demo's Earn pages: the upgrade page and its checkout, the welcome
// page after it, a paid member's billing link, the editor's test checkout,
// and the paid parts of the preference center and the unsubscribe page.
func TestEarnPages(t *testing.T) {
	env, ids := demo(t)
	ctx := context.Background()
	insider := one[uuid.UUID](t, env, `SELECT id FROM earn_tiers WHERE workspace_id = $1 AND name = 'Weekly Digest Insider'`, ids.ws)
	free := one[uuid.UUID](t, env, `SELECT c.id FROM contacts c JOIN list_memberships m ON m.contact_id = c.id AND m.list_id = $2 AND m.status = 'subscribed'
		WHERE c.workspace_id = $1 AND c.status = 'active' AND NOT EXISTS (SELECT 1 FROM paid_subscriptions p WHERE p.contact_id = c.id) ORDER BY c.email LIMIT 1`, ids.ws, ids.digest)
	earnToken := func(kind linktoken.Kind, contact uuid.UUID) string {
		return env.App.Links.Sign(linktoken.Claims{Kind: kind, WorkspaceID: ids.ws, ContactID: contact})
	}

	// The upgrade page, for anyone; checkout is Stripe's page.
	r := get(t, env, "/p/"+wsSlug+"/upgrade")
	must(t, r, 200, "Weekly Digest Insider", "Founding member", "Most popular", "Subscribe free", "Prices in US dollars, before tax")
	lacks(t, r, "Test mode")
	if csp := r.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "https://checkout.stripe.com") {
		t.Errorf("CSP doesn't allow the checkout form: %s", csp)
	}
	// A paid member's own link says they're a member instead.
	must(t, get(t, env, "/p/"+wsSlug+"/upgrade?r="+earnToken(earn.TokenUpgrade, ids.sam)), 200, "You're an Insider", "Manage your subscription")
	r = do(t, env, http.MethodPost, "/p/"+wsSlug+"/upgrade/checkout", url.Values{"tier": {insider.String()}, "period": {"year"}}, nil)
	if r.Status != http.StatusSeeOther || !strings.HasPrefix(r.Header.Get("Location"), "https://checkout.stripe.com/") {
		t.Fatalf("checkout: %d %s", r.Status, r.Header.Get("Location"))
	}
	r = do(t, env, http.MethodPost, "/p/"+wsSlug+"/upgrade/checkout", url.Values{"tier": {uuid.NewString()}, "period": {"year"}}, nil)
	must(t, r, http.StatusServiceUnavailable, "Payments aren’t available right now", "Nothing was charged")

	// The welcome page after checkout.
	must(t, get(t, env, "/p/"+wsSlug+"/upgrade/welcome?session=cs_live_demoSamRiveraFALL26"), 200, "Sam", "What's included", "Read past issues")
	must(t, get(t, env, "/p/"+wsSlug+"/upgrade/welcome?session=cs_live_nobodyEverPaid"), 404)

	// Billing: Sam's link opens Stripe's portal; a free reader has nothing to manage.
	r = get(t, env, "/p/"+wsSlug+"/billing/"+earnToken(earn.TokenBilling, ids.sam))
	if r.Status != http.StatusSeeOther || !strings.HasPrefix(r.Header.Get("Location"), "https://billing.stripe.com/") {
		t.Fatalf("billing: %d %s", r.Status, r.Header.Get("Location"))
	}
	must(t, get(t, env, "/p/"+wsSlug+"/billing/"+earnToken(earn.TokenBilling, free)), 200, "No paid subscription to manage")
	must(t, get(t, env, "/p/"+wsSlug+"/billing/"+earnToken(earn.TokenCancel, ids.sam)), 404)
	must(t, get(t, env, "/p/"+wsSlug+"/cancel/"+earnToken(earn.TokenBilling, ids.sam)+"x"), 404)

	// The preference center: Sam's subscription, and the offer for a free reader.
	must(t, get(t, env, "/p/"+wsSlug+"/preferences/"+sign(env, linktoken.Preferences, ids.ws, ids.sam, uuid.Nil)), 200,
		"Your paid subscription", "Manage billing", "Cancel subscription")
	r = get(t, env, "/p/"+wsSlug+"/preferences/"+sign(env, linktoken.Preferences, ids.ws, free, uuid.Nil))
	must(t, r, 200, "Get more with Insider")
	lacks(t, r, "Manage billing")
	// Unsubscribing doesn't cancel, and the page says so to a paid member only.
	must(t, get(t, env, "/p/"+wsSlug+"/unsubscribe/"+sign(env, linktoken.Unsubscribe, ids.ws, ids.sam, uuid.Nil)), 200, "doesn’t cancel your subscription")
	lacks(t, get(t, env, "/p/"+wsSlug+"/unsubscribe/"+sign(env, linktoken.Unsubscribe, ids.ws, free, uuid.Nil)), "doesn’t cancel your subscription")

	// The editor's test checkout opens the page in test mode, on the custom domain.
	ws, err := env.App.Q.GetWorkspace(ctx, ids.ws)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(env.App.Earn().TestCheckoutURL(ctx, &ws))
	if err != nil || u.Query().Get("test") == "" {
		t.Fatalf("test checkout link: %v %v", u, err)
	}
	r = do(t, env, http.MethodGet, u.RequestURI(), nil, hdr{"Host": u.Host})
	must(t, r, 200, "Test mode · no real payments", "Weekly Digest Insider")
	if r.Header.Get("X-Robots-Tag") == "" && !strings.Contains(r.Body, "noindex") {
		t.Error("a test checkout page is indexable")
	}
	// A forged or expired test link is the ordinary page.
	q := u.Query()
	q.Set("test", q.Get("test")+"x")
	lacks(t, do(t, env, http.MethodGet, u.Path+"?"+q.Encode(), nil, hdr{"Host": u.Host}), "Test mode · no real payments")
}
