package publicpages_test

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mailer"
)

var (
	billingLinkIn = regexp.MustCompile(`https?://\S+/billing/[A-Za-z0-9_-]+`)
	cancelLinkIn  = regexp.MustCompile(`https?://\S+/cancel/[A-Za-z0-9_-]+`)
)

// open requests an absolute link of the workspace, on its custom domain
// when it has one.
func open(t *testing.T, env *apptest.Env, link string) resp {
	t.Helper()
	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	return do(t, env, http.MethodGet, u.RequestURI(), nil, hdr{"Host": u.Host})
}

// A paid member's billing and cancel links open Stripe's portal (the
// card, the invoices and the address on them, cancelling) for a while
// only: an hour when asked for, 30 days in an email. The pages a
// forwarded email reaches (the preference center, which every "view
// online" page links to, the unsubscribe page, the upgrade page) don't
// open the portal: they email the member fresh links, to the address on
// file, and say the same whatever happens.
func TestBillingLinks(t *testing.T) {
	env, ids := demo(t)
	ctx := context.Background()
	mail := &fakeMail{}
	e := env.App.Earn()
	e.Mail = func() mailer.Sender { return mail }
	ws, err := env.App.Q.GetWorkspace(ctx, ids.ws)
	if err != nil {
		t.Fatal(err)
	}
	later := func(d time.Duration) { e.Now = func() time.Time { return time.Now().Add(d) } }
	defer later(0)
	const sam = "sam.rivera@outlook.com"
	prefs := sign(env, linktoken.Preferences, ids.ws, ids.sam, uuid.Nil)

	// The pages ask for the links by email.
	r := get(t, env, "/p/"+wsSlug+"/preferences/"+prefs)
	must(t, r, 200, "Your paid subscription", "Manage billing", "Cancel subscription", `/p/acme-newsletter/billing"`, `name="t" value="`,
		"For your safety, we email the link to sam.rivera@outlook.com. It works for an hour.")
	lacks(t, r, "/billing/", "/cancel/", "billing.stripe.com")
	r = get(t, env, "/p/"+wsSlug+"/unsubscribe/"+sign(env, linktoken.Unsubscribe, ids.ws, ids.sam, uuid.Nil))
	must(t, r, 200, "doesn’t cancel your subscription", "Cancel subscription", `/p/acme-newsletter/billing"`)
	lacks(t, r, "/billing/", "/cancel/")
	upgrade := env.App.Links.Sign(linktoken.Claims{Kind: earn.TokenUpgrade, WorkspaceID: ids.ws, ContactID: ids.sam})
	r = get(t, env, "/p/"+wsSlug+"/upgrade?r="+upgrade)
	must(t, r, 200, "You're an Insider", "Manage your subscription", `/p/acme-newsletter/billing"`, `name="t" value="`+upgrade+`"`)
	lacks(t, r, "/billing/", "/cancel/")
	if len(mail.sent) != 0 {
		t.Fatalf("opening the pages sent %d emails", len(mail.sent))
	}

	// Asking sends both links to the address on file.
	post := func(form url.Values) resp { return do(t, env, http.MethodPost, "/p/"+wsSlug+"/billing", form, nil) }
	must(t, post(url.Values{"t": {prefs}}), 200, "Check your inbox", "If sam.rivera@outlook.com has a paid subscription, a link to manage it is on its way.")
	got := mail.to(sam)
	if len(got) != 1 || got[0].Kind != "billing_link" || !strings.Contains(got[0].Text, "It works for an hour.") {
		t.Fatalf("the billing link email: %+v", got)
	}
	billing, cancel := billingLinkIn.FindString(got[0].Text), cancelLinkIn.FindString(got[0].Text)
	if billing == "" || cancel == "" || !strings.Contains(got[0].HTML, billing) {
		t.Fatalf("links in the email:\n%s", got[0].Text)
	}
	if r := open(t, env, billing); r.Status != http.StatusSeeOther || !strings.HasPrefix(r.Header.Get("Location"), "https://billing.stripe.com/") {
		t.Fatalf("the emailed billing link: %d %s", r.Status, r.Header.Get("Location"))
	}
	must(t, open(t, env, cancel), 200, "Cancel your Insider membership?")
	// Asking again soon sends nothing more, and says the same.
	must(t, post(url.Values{"t": {upgrade}}), 200, "Check your inbox", "If sam.rivera@outlook.com has a paid subscription")
	if n := len(mail.to(sam)); n != 1 {
		t.Errorf("%d emails after asking twice", n)
	}

	// An hour later they've expired: their pages ask for an address to
	// email new ones, and the portal stays shut.
	later(earn.LinkValidAsked + time.Minute)
	for _, link := range []string{billing, cancel} {
		r := open(t, env, link)
		must(t, r, http.StatusGone, "This link has expired", "Your email", "Email me my link", `/billing" novalidate`)
		lacks(t, r, "Continue to cancel")
	}
	// A link in an email works for 30 days: a member email's, a campaign's.
	later(0)
	inEmail := e.BillingURL(ctx, &ws, ids.sam, earn.LinkValidInEmail)
	links, err := e.ReaderLinks(ctx, ids.ws)
	if err != nil {
		t.Fatal(err)
	}
	inCampaign := ""
	for _, l := range links {
		if l.Var == earn.VarManageURL {
			inCampaign = l.URL(uuid.Nil, ids.sam)
		}
	}
	later(29 * 24 * time.Hour)
	for _, link := range []string{inEmail, inCampaign} {
		if r := open(t, env, link); r.Status != http.StatusSeeOther {
			t.Errorf("%s after 29 days: %d", link, r.Status)
		}
	}
	later(earn.LinkValidInEmail + time.Minute)
	for _, link := range []string{inEmail, inCampaign} {
		must(t, open(t, env, link), http.StatusGone, "This link has expired")
	}
	later(0)
	// Links from before links had a time are expired.
	for _, kind := range []linktoken.Kind{earn.TokenBilling, earn.TokenCancel} {
		tok := env.App.Links.Sign(linktoken.Claims{Kind: kind, WorkspaceID: ids.ws, ContactID: ids.sam})
		route := map[linktoken.Kind]string{earn.TokenBilling: "/billing/", earn.TokenCancel: "/cancel/"}[kind]
		must(t, get(t, env, "/p/"+wsSlug+route+tok), http.StatusGone, "This link has expired")
	}

	// The expired page's address: another paid member gets their links.
	var other string
	if err := env.App.Pool.QueryRow(ctx, `SELECT c.email FROM paid_subscriptions s JOIN contacts c ON c.id = s.contact_id
		WHERE s.workspace_id = $1 AND s.kind = 'stripe' AND s.mode = 'live' AND s.ended_at IS NULL AND s.status = 'active' AND s.stripe_customer_id <> ''
		  AND c.id <> $2 AND c.status = 'active' ORDER BY c.email LIMIT 1`, ids.ws, ids.sam).Scan(&other); err != nil {
		t.Fatal(err)
	}
	must(t, post(url.Values{"email": {other}, "_lang": {"en"}}), 200, "Check your inbox", "If "+other+" has a paid subscription")
	if got := mail.to(other); len(got) != 1 || billingLinkIn.FindString(got[0].Text) == "" {
		t.Errorf("by address: %+v", got)
	}
	// Anything else gets the same answer and no email.
	before := len(mail.sent)
	must(t, post(url.Values{"email": {"nobody.here@fastmail.com"}}), 200, "If nobody.here@fastmail.com has a paid subscription")
	free := one[uuid.UUID](t, env, `SELECT c.id FROM contacts c WHERE c.workspace_id = $1 AND c.status = 'active'
		AND NOT EXISTS (SELECT 1 FROM paid_subscriptions p WHERE p.contact_id = c.id) ORDER BY c.email LIMIT 1`, ids.ws)
	must(t, post(url.Values{"t": {sign(env, linktoken.Preferences, ids.ws, free, uuid.Nil)}}), 200, "Check your inbox")
	if len(mail.sent) != before {
		t.Errorf("%d emails for addresses without a paid subscription", len(mail.sent)-before)
	}
	// Neither a member's link nor an address isn't a request.
	must(t, post(url.Values{"t": {"not-a-token"}}), 404)
	must(t, post(url.Values{}), 404)
}
