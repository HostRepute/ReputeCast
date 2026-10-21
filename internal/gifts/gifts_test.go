package gifts_test

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
	"github.com/hostrepute/reputecast/internal/gifts"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/payments/stripe/stripetest"
	"github.com/hostrepute/reputecast/internal/store"
)

const ws = "/workspaces/acme-newsletter"

// The demo's live connection's webhook (internal/demo/seed_earn.go).
const (
	hookToken  = "7Gk2pQx9LmT4vB8sNw3eRfJh"
	hookSecret = "whsec_demoAcmeLiveSigningSecret0001"
)

// sample is the demo with its Stripe in a fake the test can read, and a
// clock the test moves.
type sample struct {
	t     *testing.T
	env   *apptest.Env
	c     *apptest.Client
	fake  *stripetest.Fake
	ws    store.Workspace
	clock *time.Time
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
	w, err := env.App.Q.GetWorkspaceBySlug(context.Background(), "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	s.ws = w
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

// deliver posts Stripe's event to the connection's webhook and applies it.
func (s *sample) deliver(ev stripetest.Event) {
	s.t.Helper()
	status, err := stripetest.Deliver(context.Background(), s.env.Server.Client(), s.env.Server.URL+"/hooks/stripe/"+hookToken, hookSecret, ev, time.Now())
	if err != nil || status != http.StatusOK {
		s.t.Fatalf("webhook: %d %v", status, err)
	}
	if _, err := s.env.App.Earn().ApplyPending(context.Background(), time.Now().Add(time.Minute)); err != nil {
		s.t.Fatal(err)
	}
}

func (s *sample) gift(code string) store.EarnGift {
	s.t.Helper()
	row, err := s.env.App.Q.GiftByCode(context.Background(), store.GiftByCodeParams{WorkspaceID: s.ws.ID, Code: code})
	if err != nil {
		s.t.Fatalf("gift %s: %v", code, err)
	}
	return row.EarnGift
}

func (s *sample) giftBySession(session string) store.EarnGift {
	s.t.Helper()
	row, err := s.env.App.Q.GiftBySession(context.Background(), store.GiftBySessionParams{WorkspaceID: s.ws.ID, Session: session})
	if err != nil {
		s.t.Fatalf("gift of %s: %v", session, err)
	}
	return row.EarnGift
}

// buy buys a gift on the gift page and pays for it.
func (s *sample) buy(in gifts.BuyInput) store.EarnGift {
	s.t.Helper()
	ctx := context.Background()
	g := s.env.App.Gifts()
	to, v, err := g.Buy(ctx, &s.ws, in)
	if err != nil || v != nil {
		s.t.Fatalf("buy: %v %+v", err, v)
	}
	session := to[strings.LastIndex(to, "/")+1:]
	ev, err := s.fake.CompletePayment(payments.Live, session, in.BuyerEmail)
	if err != nil {
		s.t.Fatal(err)
	}
	s.deliver(ev)
	return s.giftBySession(session)
}

func (s *sample) yearOption() string {
	s.t.Helper()
	var v gifts.GiftsView
	s.c.Do(http.MethodGet, ws+"/earn/gifts", nil).MustStatus(s.t, 200).JSON(s.t, &v)
	for _, o := range v.Options {
		if o.Months == 12 {
			return o.ID
		}
	}
	s.t.Fatal("no yearly gift option")
	return ""
}

func (s *sample) events(gift uuid.UUID) map[string]int {
	s.t.Helper()
	rows, err := s.env.App.Pool.Query(context.Background(), `SELECT kind FROM earn_gift_events WHERE gift_id = $1`, gift)
	if err != nil {
		s.t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			s.t.Fatal(err)
		}
		out[k]++
	}
	return out
}

// A gift is bought on the gift page with one payment through Stripe; it's
// delivered on its date with the redeem link, redeemed once without a
// card (the recipient joins the tier's lists with the page's text as
// consent), and its message is kept until the gift ends plus 30 days.
func TestGiftPurchaseDeliveryAndRedemption(t *testing.T) {
	s := newSample(t)
	ctx := context.Background()
	g := s.env.App.Gifts()
	year := s.yearOption()

	// The page asks for what's missing.
	_, v, err := g.Buy(ctx, &s.ws, gifts.BuyInput{Option: year, To: "email", When: "date", Date: "2020-01-01", Time: "09:00", BuyerEmail: "not-an-address"})
	if err != nil || v == nil {
		t.Fatalf("an incomplete form went through: %v", err)
	}
	for _, f := range []string{"recipientEmail", "date", "buyerName", "buyerEmail"} {
		if v.Errors[f] == "" {
			t.Errorf("no error for %s: %v", f, v.Errors)
		}
	}

	// Bought for tomorrow 09:00 New York time.
	ny, _ := time.LoadLocation("America/New_York")
	day := s.clock.In(ny).AddDate(0, 0, 1).Format(time.DateOnly)
	gf := s.buy(gifts.BuyInput{Option: year, To: "email", RecipientName: "Ana Lima", RecipientEmail: "Ana.Lima@example.com", When: "date", Date: day, Time: "09:00",
		Timezone: "America/New_York", Message: "Enjoy the deep dives!", BuyerName: "Bea Costa", BuyerEmail: "bea@example.com"})
	if gf.Status != gifts.StatusPaid || gf.PaidAt == nil || gf.PaymentIntent == "" || gf.AmountCents != 8000 || gf.Months != 12 || gf.RecipientEmail != "ana.lima@example.com" {
		t.Fatalf("paid gift: %+v", gf)
	}
	cs := s.fake.Get(payments.Live, gf.StripeSessionID)
	if cs == nil || cs["mode"] != "payment" || fmt.Sprint(cs["amount_total"]) != "8000" {
		t.Fatalf("the checkout in Stripe: %v", cs)
	}
	var outcome string
	if err := s.env.App.Pool.QueryRow(ctx, `SELECT outcome FROM payment_events WHERE workspace_id = $1 AND type = 'checkout.session.completed' ORDER BY received_at DESC LIMIT 1`,
		s.ws.ID).Scan(&outcome); err != nil || outcome != "gift_paid" {
		t.Errorf("the payment event's outcome: %q %v", outcome, err)
	}
	thanks, err := g.Thanks(ctx, &s.ws, gf.StripeSessionID)
	if err != nil || thanks.State != "scheduled" || !strings.Contains(thanks.Heading, "Your gift to Ana arrives on") || !strings.Contains(thanks.To, "with your message") {
		t.Fatalf("after paying: %+v %v", thanks, err)
	}

	// The receipt goes at once; the gift waits for its date.
	if err := g.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	gf = s.gift(gf.Code)
	if gf.ReceiptSentAt == nil || gf.Status != gifts.StatusPaid || s.events(gf.ID)["receipt"] != 1 {
		t.Fatalf("receipt: %+v %v", gf, s.events(gf.ID))
	}
	*s.clock = gf.DeliverAt.Add(time.Minute)
	if err := g.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	gf = s.gift(gf.Code)
	if gf.Status != gifts.StatusDelivered || gf.DeliveredAt == nil || gf.LinkExpiresAt == nil || gf.LinkExpiresAt.Before(s.clock.AddDate(0, 11, 27)) {
		t.Fatalf("delivered: %+v", gf)
	}

	// The redeem page: who gave it, the message, the address it's for.
	page, err := g.RedeemPage(ctx, &s.ws, strings.ToLower(gf.Code))
	if err != nil || page.State != "redeem" || page.Heading != "Bea Costa gave you a year of Weekly Digest Insider" || page.Message != "Enjoy the deep dives!" ||
		page.Email != "ana.lima@example.com" || !strings.Contains(page.Fine, "No card needed.") {
		t.Fatalf("redeem page: %+v %v", page, err)
	}
	// Someone with Insider already is told so and can pass it on.
	have, err := g.Redeem(ctx, &s.ws, gf.Code, gifts.RedeemInput{Email: "sam.rivera@outlook.com"})
	if err != nil || have.State != "have" || !strings.Contains(have.LinkURL, "/gift/redeem/"+gf.Code) {
		t.Fatalf("already paid: %+v %v", have, err)
	}
	if s.gift(gf.Code).Status != gifts.StatusDelivered {
		t.Fatal("the gift was used by someone who has Insider")
	}
	done, err := g.Redeem(ctx, &s.ws, gf.Code, gifts.RedeemInput{Email: "ana.lima@example.com"})
	if err != nil || done.State != "redeemed" || !strings.HasPrefix(done.Heading, "You're an Insider until ") {
		t.Fatalf("redeemed: %+v %v", done, err)
	}
	gf = s.gift(gf.Code)
	if gf.Status != gifts.StatusRedeemed || gf.RedeemedContact == nil || gf.SubscriptionID == nil {
		t.Fatalf("redeemed gift: %+v", gf)
	}
	var kind, status string
	var ends time.Time
	if err := s.env.App.Pool.QueryRow(ctx, `SELECT kind, status, current_period_end FROM paid_subscriptions WHERE id = $1`, *gf.SubscriptionID).Scan(&kind, &status, &ends); err != nil ||
		kind != "gift" || status != "gift" || ends.Before(s.clock.AddDate(1, 0, -1)) {
		t.Fatalf("gift access: %s %s %s %v", kind, status, ends, err)
	}
	var member, consent string
	if err := s.env.App.Pool.QueryRow(ctx, `SELECT m.status, coalesce(ce.consent_text, '') FROM list_memberships m JOIN lists l ON l.id = m.list_id
		LEFT JOIN consent_events ce ON ce.contact_id = m.contact_id AND ce.list_id = m.list_id AND ce.source = 'gift_redeem'
		WHERE m.contact_id = $1 AND l.name = 'Weekly Digest'`, *gf.RedeemedContact).Scan(&member, &consent); err != nil || member != "subscribed" ||
		!strings.Contains(consent, "Redeeming adds you to the Weekly Digest") {
		t.Fatalf("joined the list: %q %q %v", member, consent, err)
	}
	var timeline int
	if err := s.env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM paid_events WHERE contact_id = $1 AND kind = 'gift_redeemed'`, *gf.RedeemedContact).Scan(&timeline); err != nil || timeline != 1 {
		t.Errorf("timeline: %d %v", timeline, err)
	}
	// A link works once.
	again, err := g.Redeem(ctx, &s.ws, gf.Code, gifts.RedeemInput{Email: "someone.else@example.com"})
	if err != nil || again.State != "used" {
		t.Fatalf("redeemed twice: %+v %v", again, err)
	}

	// "Gift ending" waits while gift-ending isn't published.
	*s.clock = ends.Add(-6 * 24 * time.Hour)
	if err := g.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if s.gift(gf.Code).EndingSentAt != nil {
		t.Error("gift-ending went out unpublished")
	}
	// The message goes 30 days after the gift ends.
	*s.clock = ends.Add(31 * 24 * time.Hour)
	if _, err := s.env.App.Pool.Exec(ctx, `UPDATE paid_subscriptions SET ended_at = current_period_end WHERE id = $1`, *gf.SubscriptionID); err != nil {
		t.Fatal(err)
	}
	if err := g.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if gf = s.gift(gf.Code); gf.Message != "" || gf.MessagePurgedAt == nil {
		t.Fatalf("message kept: %+v", gf)
	}
}

// A gift not redeemed a week after delivery is sent again once; its link
// expires after the redemption window; a refunded gift's link stops
// working and a scheduled one isn't delivered.
func TestGiftRemindersExpiryAndRefunds(t *testing.T) {
	s := newSample(t)
	ctx := context.Background()
	g := s.env.App.Gifts()
	year := s.yearOption()

	// The pages as readers get them: the form, what to correct, the
	// provider's checkout, and the redeem page of a demo gift.
	reader := s.env.Client()
	page := reader.Do(http.MethodGet, "/p/acme-newsletter/gift", nil).MustStatus(t, 200)
	if body := string(page.Body); !strings.Contains(body, "Give the Weekly Digest Insider") || !strings.Contains(body, `name="option"`) || !strings.Contains(body, "Continue to payment") {
		t.Fatalf("gift page: %s", body)
	}
	form := url.Values{"option": {year}, "to": {"email"}, "when": {"now"}, "buyer_name": {"Fay Lund"}, "buyer_email": {"fay@example.com"}}
	bad := reader.DoRaw(http.MethodPost, "/p/acme-newsletter/gift", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if bad.Status != http.StatusUnprocessableEntity || !strings.Contains(string(bad.Body), "Enter their email address") {
		t.Fatalf("missing recipient: %d %s", bad.Status, bad.Body)
	}
	form.Set("recipient_email", "gus@example.com")
	ok := reader.DoRaw(http.MethodPost, "/p/acme-newsletter/gift", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if ok.Status != http.StatusSeeOther || !strings.HasPrefix(ok.Header.Get("Location"), "https://checkout.stripe.com/") {
		t.Fatalf("checkout: %d %s", ok.Status, ok.Header.Get("Location"))
	}
	redeem := reader.Do(http.MethodGet, "/p/acme-newsletter/gift/redeem/k4rm-hd24-dec8", nil).MustStatus(t, 200)
	if body := string(redeem.Body); !strings.Contains(body, "Omar Haddad gave you a year of Weekly Digest Insider") || !strings.Contains(body, "Start my Insider year") {
		t.Fatalf("redeem page: %s", body)
	}
	reader.Do(http.MethodGet, "/p/acme-newsletter/gift/redeem/NOPE-NOPE-NOPE", nil).MustStatus(t, 404)

	now := gifts.BuyInput{Option: year, To: "email", RecipientEmail: "chris@example.com", When: "now", BuyerName: "Dana Reyes", BuyerEmail: "dana@example.com"}
	gf := s.buy(now)
	if err := g.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if gf = s.gift(gf.Code); gf.Status != gifts.StatusDelivered {
		t.Fatalf("a gift for now isn't delivered: %+v", gf)
	}
	*s.clock = gf.DeliveredAt.Add(gifts.RemindAfter + time.Minute)
	if err := g.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if err := g.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if gf = s.gift(gf.Code); gf.RemindedAt == nil || s.events(gf.ID)["reminded"] != 1 {
		t.Fatalf("reminder: %+v %v", gf, s.events(gf.ID))
	}
	*s.clock = gf.LinkExpiresAt.Add(time.Minute)
	if err := g.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if gf = s.gift(gf.Code); gf.Status != gifts.StatusExpired {
		t.Fatalf("not expired: %+v", gf)
	}
	if p, _ := g.RedeemPage(ctx, &s.ws, gf.Code); p.State != "expired" || !strings.Contains(p.Text, "Gift links work for 12 months") {
		t.Fatalf("expired page: %+v", p)
	}

	// Refunded in Stripe before its date: nothing goes out.
	*s.clock = time.Now()
	ny, _ := time.LoadLocation("America/New_York")
	later := gifts.BuyInput{Option: year, To: "person", When: "date", Date: s.clock.In(ny).AddDate(0, 0, 3).Format(time.DateOnly), Time: "10:00", Timezone: "America/New_York",
		BuyerName: "Eli Park", BuyerEmail: "eli@example.com"}
	gr := s.buy(later)
	s.deliver(s.fake.RefundPayment(payments.Live, gr.PaymentIntent, 0))
	if gr = s.gift(gr.Code); gr.Status != gifts.StatusRefunded || gr.RefundedAt == nil {
		t.Fatalf("refund: %+v", gr)
	}
	*s.clock = gr.DeliverAt.Add(time.Minute)
	if err := g.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if gr = s.gift(gr.Code); gr.DeliveredAt != nil {
		t.Fatal("a refunded gift was delivered")
	}
	if p, _ := g.RedeemPage(ctx, &s.ws, gr.Code); p.State != "refunded" {
		t.Fatalf("refunded page: %+v", p)
	}
}

// Earn › Gifts: the numbers of the sample, the options and the gift page
// (earn:edit), and the team's changes to a gift (earn:manage), each
// audited; a viewer reads only.
func TestGiftsTab(t *testing.T) {
	s := newSample(t)
	var v gifts.GiftsView
	s.c.Do(http.MethodGet, ws+"/earn/gifts", nil).MustStatus(t, 200).JSON(t, &v)
	n := v.Numbers
	if n.Sold != 11 || n.SoldCents != 65600 || n.Delivered != 9 || n.Redeemed != 8 || n.Waiting != 1 || n.Scheduled != 2 || n.WithAccess != 8 || n.NextAt == nil {
		t.Fatalf("numbers: %+v", n)
	}
	if len(v.Options) != 2 || !v.Page.On || v.Page.LinkMonths != 12 || len(v.Emails) != 3 || v.Ending == nil || !v.CanEdit || !v.CanManage {
		t.Fatalf("tab: %d options, page %+v, %d emails, ending %+v", len(v.Options), v.Page, len(v.Emails), v.Ending)
	}
	var list gifts.GiftList
	s.c.Do(http.MethodGet, ws+"/earn/gifts/list", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Counts.All != 11 || list.Counts.Scheduled != 2 || list.Counts.NotRedeemed != 1 || list.Counts.Redeemed != 8 || len(list.Items) != 11 ||
		list.Items[0].Buyer != "Omar Haddad" || list.Items[0].Status != "scheduled" {
		t.Fatalf("list: %+v, first %+v", list.Counts, list.Items[0])
	}
	s.c.Do(http.MethodGet, ws+"/earn/gifts/list?status=scheduled&q=karim", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].RecipientName != "Karim Haddad" {
		t.Fatalf("search: %+v", list.Items)
	}
	karim := list.Items[0]

	// Options and the page.
	var o gifts.GiftOption
	s.c.Do(http.MethodPost, ws+"/earn/gifts/options", map[string]any{"tierId": v.Options[0].TierID, "months": 6, "amountCents": 4500}).MustStatus(t, 201).JSON(t, &o)
	if o.Months != 6 || o.AmountCents != 4500 || s.audits("earn.gift_option_created", o.ID) != 1 {
		t.Fatalf("option: %+v", o)
	}
	s.c.Do(http.MethodPost, ws+"/earn/gifts/options", map[string]any{"tierId": v.Options[0].TierID, "months": 6, "amountCents": 4000}).MustStatus(t, 422)
	s.c.Do(http.MethodPut, ws+"/earn/gifts/options/"+o.ID, map[string]any{"tierId": v.Options[0].TierID, "months": 6, "amountCents": 4200}).MustStatus(t, 200).JSON(t, &o)
	if o.AmountCents != 4200 || s.audits("earn.gift_option_updated", o.ID) != 1 {
		t.Fatalf("option changed: %+v", o)
	}
	s.c.Do(http.MethodDelete, ws+"/earn/gifts/options/"+o.ID, nil).MustStatus(t, 204)
	var pg gifts.GiftPage
	s.c.Do(http.MethodPut, ws+"/earn/gifts/page", map[string]any{"on": false, "linkMonths": 6}).MustStatus(t, 200).JSON(t, &pg)
	if pg.On || pg.LinkMonths != 6 || pg.ChangedBy != "Linh Nguyen" || s.audits("earn.gift_page_updated", s.ws.ID.String()) != 1 {
		t.Fatalf("page: %+v", pg)
	}
	if p, err := s.env.App.Gifts().Page(context.Background(), &s.ws, gifts.BuyInput{}, false); err != nil || p.State != "off" {
		t.Fatalf("the page while off: %+v %v", p, err)
	}

	// A scheduled gift moves; its link is copied; only delivered gifts go again.
	var row gifts.GiftRow
	s.c.Do(http.MethodPost, ws+"/earn/gifts/"+karim.ID+"/delivery", map[string]any{"date": "2026-12-23", "time": "08:00", "timezone": "America/Chicago"}).MustStatus(t, 200).JSON(t, &row)
	if row.DeliverAt == nil || row.DeliverAt.UTC().Format(time.RFC3339) != "2026-12-23T14:00:00Z" || s.audits("earn.gift_delivery_changed", karim.ID) != 1 {
		t.Fatalf("delivery moved: %+v", row)
	}
	var link gifts.GiftLink
	s.c.Do(http.MethodPost, ws+"/earn/gifts/"+karim.ID+"/link", nil).MustStatus(t, 200).JSON(t, &link)
	if !strings.Contains(link.URL, "/gift/redeem/") || s.audits("earn.gift_link_copied", karim.ID) != 1 {
		t.Fatalf("link: %+v", link)
	}
	s.c.Do(http.MethodPost, ws+"/earn/gifts/"+karim.ID+"/resend", nil).MustStatus(t, 409)

	// A viewer reads only.
	viewer := s.env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do(http.MethodGet, ws+"/earn/gifts", nil).MustStatus(t, 200).JSON(t, &v)
	if v.CanEdit || v.CanManage {
		t.Error("a viewer can change gifts")
	}
	viewer.Do(http.MethodPut, ws+"/earn/gifts/page", map[string]any{"on": true, "linkMonths": 12}).MustStatus(t, 403)
	viewer.Do(http.MethodPost, ws+"/earn/gifts/"+karim.ID+"/link", nil).MustStatus(t, 403)
}
