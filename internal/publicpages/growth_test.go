package publicpages_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/growth"
)

// A signup through the subscribe page keeps where it came from (growth
// sources, audience-insights brief §3.4): the page's UTM parameters and the
// page before it go with the consent record, and once the contact
// confirms, Growth counts them under the page with that first touch.
func TestSubscribePageCapturesFirstTouch(t *testing.T) {
	env, ids := demo(t)
	ctx := context.Background()
	email := "rowan.ellis@fastmail.com"
	page := "https://news.acme.io/subscribe?utm_source=linkedin&utm_medium=social&utm_campaign=flags-ga&utm_content=carousel"
	must(t, do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe", url.Values{
		"_t": {formToken(env, "frm_8c2e1a", time.Minute)}, "email": {email}, "list": {ids.digest.String()}, "consent": {"1"},
		"_page": {page}, "_ref": {"https://docs.acme.io/flags?ref=nav#install"},
	}, hdr{"Origin": env.Server.URL}), 200, "Check your inbox")
	contact, ok := contactID(t, env, ids.ws, email)
	if !ok {
		t.Fatal("no contact for the signup")
	}
	var src, medium, campaign, content, pg, ref string
	if err := env.App.Pool.QueryRow(ctx, `SELECT utm_source, utm_medium, utm_campaign, utm_content, page, referrer FROM consent_events
		WHERE contact_id = $1 AND action = 'subscribed'`, contact).Scan(&src, &medium, &campaign, &content, &pg, &ref); err != nil {
		t.Fatal(err)
	}
	if src != "linkedin" || medium != "social" || campaign != "flags-ga" || content != "carousel" || pg != "news.acme.io/subscribe" || ref != "docs.acme.io/flags" {
		t.Fatalf("first touch: %q %q %q %q %q %q", src, medium, campaign, content, pg, ref)
	}
	// The confirmation click; then the next attribution run counts the join.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE list_memberships SET status = 'subscribed', subscribed_at = now() - interval '5 seconds' WHERE contact_id = $1`, contact); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO consent_events (id, contact_id, list_id, action, source, detail, occurred_at)
		VALUES ($1, $2, $3, 'confirmed', 'subscribe_page', '', now() - interval '5 seconds')`, uuid.Must(uuid.NewV7()), contact, ids.digest); err != nil {
		t.Fatal(err)
	}
	ws, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	g := &growth.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}
	if err := g.Attribute(ctx, &ws); err != nil {
		t.Fatal(err)
	}
	var kind, campaignOf, refDomain string
	var captured bool
	if err := env.App.Pool.QueryRow(ctx, `SELECT source_kind, utm_campaign, referrer_domain, captured FROM contact_growth WHERE contact_id = $1`, contact).
		Scan(&kind, &campaignOf, &refDomain, &captured); err != nil {
		t.Fatal(err)
	}
	if (kind != "subscribe_page" && kind != "form") || campaignOf != "flags-ga" || refDomain != "docs.acme.io" || !captured {
		t.Fatalf("growth row: %s %s %s %v", kind, campaignOf, refDomain, captured)
	}
	// Leaving every list is a departure.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE list_memberships SET status = 'unsubscribed', unsubscribed_at = now() - interval '4 seconds' WHERE contact_id = $1`, contact); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO consent_events (id, contact_id, list_id, action, source, detail, occurred_at)
		VALUES ($1, $2, $3, 'unsubscribed', 'one_click', '', now() - interval '4 seconds')`, uuid.Must(uuid.NewV7()), contact, ids.digest); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := g.Attribute(ctx, &ws); err != nil {
		t.Fatal(err)
	}
	var leftKind *string
	if err := env.App.Pool.QueryRow(ctx, `SELECT left_kind FROM contact_growth WHERE contact_id = $1`, contact).Scan(&leftKind); err != nil || leftKind == nil || *leftKind != "unsubscribed" {
		t.Fatalf("departure: %v %v", leftKind, err)
	}
}
