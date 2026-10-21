package offers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/offers"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/payments/stripe/stripetest"
	"github.com/hostrepute/reputecast/internal/store"
)

const ws = "/workspaces/acme-newsletter"

// sample is the demo with its Stripe in a fake the test can read.
type sample struct {
	t    *testing.T
	env  *apptest.Env
	c    *apptest.Client
	fake *stripetest.Fake
	ws   store.Workspace
	// insider is Weekly Digest Insider's id.
	insider string
	clock   *time.Time
}

func newSample(t *testing.T) *sample {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	fake := stripetest.New()
	fake.Permissive = true
	fake.AddKey(demo.EarnLiveKey, stripetest.KeySpec{})
	fake.AddKey(demo.EarnTestKey, stripetest.KeySpec{})
	now := time.Now()
	s := &sample{t: t, env: env, fake: fake, clock: &now}
	e := env.App.Earn()
	e.Demo = func() *http.Client { return fake.HTTPClient() }
	e.Now = func() time.Time { return *s.clock }
	ctx := context.Background()
	w, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	s.ws = w
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM earn_tiers WHERE workspace_id = $1 AND name = 'Weekly Digest Insider'`, w.ID).Scan(&s.insider); err != nil {
		t.Fatal(err)
	}
	s.c = env.Client()
	s.c.SignInDemo("linh@acme.io")
	return s
}

func (s *sample) audits(action, target string) int {
	var n int
	if err := s.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1 AND target_id = $2`, action, target).Scan(&n); err != nil {
		s.t.Fatal(err)
	}
	return n
}

// An offer is a coupon and a promotion code in Stripe with its limit, date
// and "new members only"; it ends at its limit or its date (the upgrade
// page then says so), raising the limit brings it back, and ending it early
// turns its code off.
func TestOfferLimitsAndExpiry(t *testing.T) {
	s := newSample(t)
	ctx := context.Background()
	in := map[string]any{"name": "SPRING27", "kind": "signup", "discountType": "percent", "percentOff": 25, "duration": "once",
		"applies": []map[string]any{{"tierId": s.insider, "period": "year"}}, "byCode": true, "code": "spring27", "byLink": true, "limit": 2,
		"endsOn": s.clock.AddDate(0, 0, 10).In(time.UTC).Format(time.DateOnly), "newOnly": true}
	var d offers.OfferDetail
	s.c.Do(http.MethodPost, ws+"/earn/offers", in).MustStatus(t, 201).JSON(t, &d)
	if d.Code != "SPRING27" || d.Sync != "synced" || d.State != "live" || d.Stripe.Coupon != "SPRING27-25" || d.Stripe.PromotionCode == "" ||
		!strings.HasSuffix(d.Link, "upgrade?offer=SPRING27") {
		t.Fatalf("created: %+v", d.Offer)
	}
	coupon := s.fake.Get(payments.Live, "SPRING27-25")
	if coupon == nil || coupon["percent_off"] != 25.0 || coupon["duration"] != "once" {
		t.Fatalf("coupon in Stripe: %v", coupon)
	}
	promo := s.fake.Get(payments.Live, d.Stripe.PromotionCode)
	if promo == nil || promo["code"] != "SPRING27" || fmt.Sprint(promo["max_redemptions"]) != "2" || promo["expires_at"] == nil ||
		promo["restrictions"].(map[string]any)["first_time_transaction"] != true {
		t.Fatalf("promotion code in Stripe: %v", promo)
	}
	if s.audits("earn.offer_created", d.ID) != 1 {
		t.Error("creating isn't audited")
	}
	// The code is the workspace's: another offer can't take it.
	s.c.Do(http.MethodPost, ws+"/earn/offers", in).MustStatus(t, 422)

	// The link applies on the yearly price only.
	o := s.env.App.Offers()
	tier := uuid.MustParse(s.insider)
	if got := o.CheckoutCode(ctx, &s.ws, "live", "spring27", tier, "year"); got != d.Stripe.PromotionCode {
		t.Errorf("checkout code: %q", got)
	}
	if got := o.CheckoutCode(ctx, &s.ws, "live", "SPRING27", tier, "month"); got != "" {
		t.Errorf("monthly checkout got a code: %q", got)
	}
	v, err := s.env.App.Earn().Upgrade(ctx, &s.ws, earn.UpgradeRequest{Offer: "SPRING27", Lang: earn.PageLang})
	if err != nil {
		t.Fatal(err)
	}
	if v.Period != "year" || v.Offer == nil || v.OfferCode != "SPRING27" || !strings.Contains(v.Offer.Label, "$60 for your first year, then $80 a year") {
		t.Fatalf("upgrade page: period %s, offer %+v", v.Period, v.Offer)
	}

	// Two paid members take it: the limit is reached.
	if _, err := s.env.App.Pool.Exec(ctx, `UPDATE paid_subscriptions SET discount = '{"label":"SPRING27","coupon":"SPRING27-25","amountCents":6000}'::jsonb
		WHERE id IN (SELECT id FROM paid_subscriptions WHERE workspace_id = $1 AND mode = 'live' AND status = 'active' AND period = 'year' AND discount IS NULL LIMIT 2)`, s.ws.ID); err != nil {
		t.Fatal(err)
	}
	if err := o.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	s.c.Do(http.MethodGet, ws+"/earn/offers/"+d.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Status != "ended" || d.EndedReason != "limit" || d.State != "limit" || d.Results.Redemptions != 2 {
		t.Fatalf("at its limit: %+v", d.Offer)
	}
	if s.fake.Get(payments.Live, d.Stripe.PromotionCode)["active"] != false {
		t.Error("the ended offer's code is still active in Stripe")
	}
	if got := o.CheckoutCode(ctx, &s.ws, "live", "SPRING27", tier, "year"); got != "" {
		t.Errorf("an ended offer still applies: %q", got)
	}
	v, _ = s.env.App.Earn().Upgrade(ctx, &s.ws, earn.UpgradeRequest{Offer: "SPRING27", Lang: earn.PageLang})
	if v.Offer != nil || v.Notice == nil || v.Notice.Title != "This offer has ended" {
		t.Fatalf("ended on the page: %+v %+v", v.Offer, v.Notice)
	}
	if s.audits("earn.offer_ended", d.ID) != 1 {
		t.Error("ending at the limit isn't audited")
	}

	// Raising the limit brings it back with a new code for the rest.
	in["limit"] = 5
	old := d.Stripe.PromotionCode
	s.c.Do(http.MethodPut, ws+"/earn/offers/"+d.ID, in).MustStatus(t, 200).JSON(t, &d)
	if d.Status != "live" || d.Stripe.PromotionCode == old || fmt.Sprint(s.fake.Get(payments.Live, d.Stripe.PromotionCode)["max_redemptions"]) != "3" {
		t.Fatalf("raised: %+v", d.Offer)
	}

	// Its date passes: it ends again, by date.
	*s.clock = s.clock.AddDate(0, 0, 11)
	if err := o.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	s.c.Do(http.MethodGet, ws+"/earn/offers/"+d.ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Status != "ended" || d.EndedReason != "date" {
		t.Fatalf("past its date: %+v", d.Offer)
	}

	// Ending FALL26 early turns its code off; its paid members keep it.
	var list offers.OfferList
	s.c.Do(http.MethodGet, ws+"/earn/offers", nil).MustStatus(t, 200).JSON(t, &list)
	var fall Offer
	for _, it := range list.Items {
		if it.Name == "FALL26" {
			fall = it
		}
	}
	if fall.ID == "" || fall.Results.Redemptions != 45 || fall.Results.MRRCents != 24000 {
		t.Fatalf("FALL26: %+v", fall)
	}
	s.c.Do(http.MethodPost, ws+"/earn/offers/"+fall.ID+"/end", nil).MustStatus(t, 200).JSON(t, &d)
	if d.Status != "ended" || d.EndedReason != "early" || d.EndedBy != "Linh Nguyen" || s.audits("earn.offer_ended", fall.ID) != 1 {
		t.Fatalf("ended early: %+v", d.Offer)
	}
	// Sofia (Editor) reads offers but can't change them.
	sofia := s.env.Client()
	sofia.SignInDemo("sofia@acme.io")
	sofia.Do(http.MethodGet, ws+"/earn/offers", nil).MustStatus(t, 200)
	sofia.Do(http.MethodPost, ws+"/earn/offers/"+fall.ID+"/end", nil).MustStatus(t, 403)
}

// Offer is the list's item.
type Offer = offers.Offer

// The retention offer shows on the cancel page to a paid member who's
// eligible, at most once in 12 months: taking it applies its coupon at
// once; continuing opens Stripe's portal on the cancel step with the
// reason kept.
func TestOneRetentionOfferIn12Months(t *testing.T) {
	s := newSample(t)
	ctx := context.Background()
	// A monthly Insider member of six months with no discount.
	var sub store.PaidSubscription
	var contact uuid.UUID
	if err := s.env.App.Pool.QueryRow(ctx, `SELECT s.id, s.contact_id FROM paid_subscriptions s JOIN contacts c ON c.id = s.contact_id
		WHERE s.workspace_id = $1 AND s.mode = 'live' AND s.kind = 'stripe' AND s.status = 'active' AND s.period = 'month' AND s.discount IS NULL
		  AND s.tier_id = $2 AND s.started_at < now() - interval '6 months' AND s.stripe_subscription_id <> '' AND s.cancel_at IS NULL
		  AND NOT EXISTS (SELECT 1 FROM earn_offer_uses u WHERE u.contact_id = s.contact_id)
		ORDER BY s.started_at LIMIT 1`, s.ws.ID, s.insider).Scan(&sub.ID, &contact); err != nil {
		t.Fatal(err)
	}
	link := s.env.App.Earn().CancelURL(ctx, &s.ws, contact)
	path := "/p/acme-newsletter/cancel/" + link[strings.LastIndex(link, "/cancel/")+len("/cancel/"):]
	reader := s.env.Client()
	page := reader.Do(http.MethodGet, path, nil).MustStatus(t, 200)
	body := string(page.Body)
	if !strings.Contains(body, "Cancel your Insider membership?") || !strings.Contains(body, "Keep Insider for $4/month") ||
		!strings.Contains(body, "Stay for $4 a month for the next 3 months.") {
		t.Fatalf("cancel page: %s", body)
	}
	visit := between(body, `name="visit" value="`, `"`)
	form := url.Values{"visit": {visit}, "reason": {"too_expensive"}, "decision": {"keep"}}
	kept := reader.DoRaw(http.MethodPost, path, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if kept.Status != 200 || !strings.Contains(string(kept.Body), "Done. You pay $4 a month until") {
		t.Fatalf("keep: %d %s", kept.Status, kept.Body)
	}
	applied := false
	for _, r := range s.fake.Requests() {
		if r.Method == http.MethodPost && strings.HasPrefix(r.Path, "/v1/subscriptions/") && r.Form.Get("coupon") == "STAY50" {
			applied = true
		}
	}
	if !applied {
		t.Error("the offer's coupon wasn't put on the subscription in Stripe")
	}
	var decision, reason string
	if err := s.env.App.Pool.QueryRow(ctx, `SELECT decision, reason FROM earn_cancel_visits WHERE id = $1`, visit).Scan(&decision, &reason); err != nil ||
		decision != "kept" || reason != "too_expensive" {
		t.Fatalf("visit: %s %s %v", decision, reason, err)
	}
	// Within 12 months there's no offer again: only Continue to cancel.
	again := string(reader.Do(http.MethodGet, path, nil).MustStatus(t, 200).Body)
	if strings.Contains(again, "Keep Insider for") || !strings.Contains(again, "Continue to cancel") {
		t.Fatalf("second visit offered again: %s", again)
	}
	// A year and a day later it may be offered again.
	o := s.env.App.Offers()
	row, err := s.env.App.Q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: sub.ID, WorkspaceID: s.ws.ID})
	if err == nil {
		*s.clock = s.clock.AddDate(1, 0, 1)
		if ret, err := o.RetentionFor(ctx, &s.ws, "live", row); err != nil || ret == nil {
			t.Errorf("a year later: %v %v", ret, err)
		}
		*s.clock = time.Now()
	}
	// Continuing goes to Stripe's portal on the cancel step; the reason stays.
	visit = between(again, `name="visit" value="`, `"`)
	form = url.Values{"visit": {visit}, "reason": {"not_reading"}, "decision": {"continue"}}
	res := reader.DoRaw(http.MethodPost, path, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if res.Status != http.StatusSeeOther || !strings.HasPrefix(res.Header.Get("Location"), "https://billing.stripe.com/") {
		t.Fatalf("continue: %d %s", res.Status, res.Header.Get("Location"))
	}
	portal := false
	for _, r := range s.fake.Requests() {
		if r.Path == "/v1/billing_portal/sessions" && r.Form.Get("flow_data[type]") == "subscription_cancel" &&
			strings.Contains(r.Form.Get("flow_data[after_completion][redirect][return_url]"), "done=1") {
			portal = true
		}
	}
	if !portal {
		t.Error("the portal didn't open on the cancel step")
	}
	var why string
	if err := s.env.App.Pool.QueryRow(ctx, `SELECT cancel_reason FROM paid_subscriptions WHERE id = $1`, sub.ID).Scan(&why); err != nil || why != "not_reading" {
		t.Errorf("cancel reason: %q %v", why, err)
	}
	// A link whose signature no longer matches (signed with an earlier
	// instance key) has expired and offers a new one; a malformed link isn't
	// one at all.
	other, err := linktoken.NewSigner([]byte(strings.Repeat("k", 32)))
	if err != nil {
		t.Fatal(err)
	}
	old := other.Sign(linktoken.Claims{Kind: earn.TokenCancel, WorkspaceID: s.ws.ID, ContactID: contact})
	gone := reader.Do(http.MethodGet, "/p/acme-newsletter/cancel/"+old, nil)
	if gone.Status != http.StatusGone || !strings.Contains(string(gone.Body), "This link has expired") || !strings.Contains(string(gone.Body), "Email me my link") {
		t.Fatalf("expired: %d", gone.Status)
	}
	reader.Do(http.MethodGet, "/p/acme-newsletter/cancel/not-a-token", nil).MustStatus(t, http.StatusNotFound)
	form = url.Values{"decision": {"continue"}}
	if res := reader.DoRaw(http.MethodPost, "/p/acme-newsletter/cancel/not-a-token", "application/x-www-form-urlencoded", strings.NewReader(form.Encode())); res.Status != http.StatusNotFound {
		t.Errorf("malformed link, posted: %d", res.Status)
	}
}

func between(s, a, b string) string {
	i := strings.Index(s, a)
	if i < 0 {
		return ""
	}
	s = s[i+len(a):]
	if j := strings.Index(s, b); j >= 0 {
		return s[:j]
	}
	return s
}

// An Upgrade block can carry a live signup offer (ComposerEarnBlocks): the
// composer gets each tier's offers with the card's line and button, and
// once an offer ends Send Readiness finds the blocks that still offer it.
func TestUpgradeBlockOffer(t *testing.T) {
	s := newSample(t)
	ctx := context.Background()
	in := map[string]any{"name": "SPRING27", "kind": "signup", "discountType": "percent", "percentOff": 25, "duration": "once",
		"applies": []map[string]any{{"tierId": s.insider, "period": "year"}}, "byCode": true, "code": "spring27", "byLink": true,
		"endsOn": s.clock.AddDate(0, 0, 10).In(time.UTC).Format(time.DateOnly)}
	var d offers.OfferDetail
	s.c.Do(http.MethodPost, ws+"/earn/offers", in).MustStatus(t, 201).JSON(t, &d)
	type composerOffer struct {
		Code, Name, Period, Option, Line, Button string
		EndsAt                                   *time.Time
	}
	offersOf := func() []composerOffer {
		var comp struct {
			Tiers []struct {
				ID     string
				Offers []composerOffer
			}
		}
		s.c.Do(http.MethodGet, ws+"/earn/composer", nil).MustStatus(t, 200).JSON(t, &comp)
		for _, tier := range comp.Tiers {
			if tier.ID == s.insider {
				return tier.Offers
			}
		}
		t.Fatal("no Insider tier in the composer's data")
		return nil
	}
	var spring *composerOffer
	for _, o := range offersOf() {
		if o.Code == "SPRING27" {
			spring = &o
		}
	}
	if spring == nil || spring.Period != "year" || spring.Option != "SPRING27 · $60 for the first year" ||
		spring.Line != "$60 for your first year with SPRING27, then $80 a year" || spring.Button != "Get Insider for $60" || spring.EndsAt == nil {
		t.Fatalf("composer offer: %+v", spring)
	}

	// An email whose Upgrade block carries it.
	src := `<mjml><mj-body><mj-section><mj-column><mj-text>Story</mj-text></mj-column></mj-section>` +
		`<mj-section rc-upgrade='{"tier":"` + s.insider + `","offer":"SPRING27"}'><mj-column><mj-text>Card</mj-text></mj-column></mj-section></mj-body></mjml>`
	if got := s.env.App.Offers().EndedUpgradeOffers(ctx, &s.ws, src); len(got) != 0 {
		t.Fatalf("a live offer counted as ended: %+v", got)
	}
	s.c.Do(http.MethodPost, ws+"/earn/offers/"+d.ID+"/end", nil).MustStatus(t, 200)
	got := s.env.App.Offers().EndedUpgradeOffers(ctx, &s.ws, src)
	if len(got) != 1 || got[0].Code != "SPRING27" || got[0].Name != "SPRING27" || got[0].EndedOn == nil || len(got[0].Path) != 1 || got[0].Path[0] != 1 {
		t.Fatalf("ended: %+v", got)
	}
	for _, o := range offersOf() {
		if o.Code == "SPRING27" {
			t.Error("the composer still offers an ended offer")
		}
	}
}
