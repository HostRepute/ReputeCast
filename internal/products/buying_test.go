package products_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/payments/stripe"
	"github.com/hostrepute/reputecast/internal/products"
	"github.com/hostrepute/reputecast/internal/store"
)

// liveOffer is a product's published list offer and delivery settings.
type liveOffer struct {
	OfferOn        bool   `json:"offerOn"`
	OfferListID    string `json:"offerListId"`
	OfferText      string `json:"offerText"`
	LinkDays       int32  `json:"linkDays"`
	LinkDownloads  int32  `json:"linkDownloads"`
	RevokeOnRefund bool   `json:"revokeOnRefund"`
	MinCents       int64  `json:"minCents"`
	SuggestedCents int64  `json:"suggestedCents"`
}

func (s *shop) liveOf(product uuid.UUID) liveOffer {
	s.t.Helper()
	var raw []byte
	s.scan(`SELECT live FROM products WHERE id = $1`, []any{product}, &raw)
	var l liveOffer
	if err := json.Unmarshal(raw, &l); err != nil {
		s.t.Fatal(err)
	}
	return l
}

// stripePrice is a product's current price in Stripe's live mode.
func (s *shop) stripePrice(product uuid.UUID) string {
	s.t.Helper()
	var price string
	s.scan(`SELECT coalesce(stripe->'live'->>'price', '') FROM products WHERE id = $1`, []any{product}, &price)
	return price
}

// checkoutRow is a Buy button's checkout as stored.
type checkoutRow struct {
	ID         uuid.UUID
	Status     string
	Mode       string
	Source     string
	SourceURL  string
	Email      string
	Connection *uuid.UUID
	OrderID    *uuid.UUID
	Consent    map[string]any
}

func (s *shop) checkoutOf(session string) checkoutRow {
	s.t.Helper()
	var c checkoutRow
	var consent []byte
	s.scan(`SELECT id, status, mode, source, source_url, email, connection_id, order_id, consent FROM product_checkouts WHERE session_id = $1`, []any{session},
		&c.ID, &c.Status, &c.Mode, &c.Source, &c.SourceURL, &c.Email, &c.Connection, &c.OrderID, &consent)
	_ = json.Unmarshal(consent, &c.Consent)
	return c
}

// A Buy button opens a Stripe Checkout session in payment mode for the
// product's price, in the account's currency: the demo's recorded price
// while it matches the product, or one made the first time a product sells
// without one. The list offer's tick, the page and the IP address travel
// with the checkout; a signed-in member's email is filled in on Stripe's
// page; paying charges what the price says (pay what you want: what the
// buyer chose).
func TestCheckoutSessions(t *testing.T) {
	s := newShop(t)
	fg := s.liveOf(s.fieldGuide)

	// The Field Guide, $29 one-time, from a visitor who ticks the offer.
	before := len(s.stripe.Requests())
	h, session := s.checkout(s.fieldGuide, "", true)
	if !strings.HasPrefix(session, "cs_live_") || h.ProductName != "The Deliverability Field Guide" || h.Currency != "usd" || h.Test || h.Account != "Acme Media" ||
		h.Price.PriceKind != products.PriceOneTime || h.Price.PriceCents != 2900 {
		t.Fatalf("handoff: %+v", h)
	}
	co := s.checkoutOf(session)
	page := s.site() + "/products/deliverability-field-guide"
	req, ok := s.lastRequest("POST", "/v1/checkout/sessions")
	if !ok {
		t.Fatal("no Checkout session was created")
	}
	for k, want := range map[string]string{
		"mode": "payment", "line_items[0][price]": s.stripePrice(s.fieldGuide), "line_items[0][quantity]": "1",
		"success_url": page + "/thanks?session_id={CHECKOUT_SESSION_ID}", "cancel_url": page + "?checkout=cancelled",
		"client_reference_id": co.ID.String(), "metadata[reputecast_checkout]": co.ID.String(), "metadata[reputecast_product]": s.fieldGuide.String(),
		"payment_intent_data[metadata][reputecast_checkout]": co.ID.String(), "payment_intent_data[metadata][reputecast_product]": s.fieldGuide.String(),
	} {
		if got := req.Form.Get(k); got != want {
			t.Errorf("session %s = %q, want %q", k, got, want)
		}
	}
	if req.Form.Has("customer_email") || req.Form.Has("subscription_data[metadata][reputecast_checkout]") {
		t.Errorf("visitor's session: %v", req.Form)
	}
	if (req.Form.Get("automatic_tax[enabled]") == "true") != h.Tax {
		t.Errorf("tax at checkout %v, session %q", h.Tax, req.Form.Get("automatic_tax[enabled]"))
	}
	for _, r := range s.requestsSince(before) {
		if r.Path == "/v1/prices" {
			t.Errorf("the recorded price matched, yet a price was made: %v", r.Form)
		}
	}
	if co.Status != "open" || co.Mode != "live" || co.Source != products.SourceProductPage || co.SourceURL != page || co.Email != "" ||
		co.Connection == nil || *co.Connection != s.live || co.OrderID != nil {
		t.Errorf("checkout: %+v", co)
	}
	if co.Consent["offered"] != true || co.Consent["ticked"] != true || co.Consent["text"] != fg.OfferText || co.Consent["listId"] != s.digest.String() ||
		co.Consent["page"] != strings.TrimPrefix(page, "http://") || co.Consent["ip"] != buyerIP || co.Consent["at"] == nil {
		t.Errorf("consent at checkout: %v", co.Consent)
	}
	s.pay(session, buyer{Email: "pat@example.com"})
	if cs := s.stripe.Get(payments.Live, session); fmt.Sprint(cs["amount_total"]) != "2900" || cs["currency"] != "usd" || cs["payment_status"] != "paid" {
		t.Errorf("paid session: %v", cs)
	}

	// The Launch Kit, pay what you want, from a signed-in member who leaves
	// the offer unticked.
	lk := s.liveOf(s.launchKit)
	h, session = s.checkout(s.launchKit, "Member@Example.com", false)
	if h.Price.PriceKind != products.PricePWYW || h.Price.MinCents != lk.MinCents || h.Price.SuggestedCents != lk.SuggestedCents || h.Price.MinCents < products.MinimumCents {
		t.Fatalf("pay what you want handoff: %+v", h.Price)
	}
	req, _ = s.lastRequest("POST", "/v1/checkout/sessions")
	if req.Form.Get("customer_email") != "Member@Example.com" || req.Form.Get("line_items[0][price]") != s.stripePrice(s.launchKit) || req.Form.Get("mode") != "payment" {
		t.Errorf("member's session: %v", req.Form)
	}
	co = s.checkoutOf(session)
	if co.Email != "member@example.com" || co.Consent["offered"] != true || co.Consent["ticked"] != false {
		t.Errorf("member's checkout: %+v", co)
	}
	s.pay(session, buyer{Amount: 2000})
	if cs := s.stripe.Get(payments.Live, session); fmt.Sprint(cs["amount_total"]) != "2000" || cs["customer_details"].(map[string]any)["email"] != "Member@Example.com" {
		t.Errorf("chosen amount: %v", cs)
	}

	// The first sale where Stripe has nothing of the products yet: their
	// product and price are made from what's published, and kept.
	s.exec(`UPDATE products SET stripe = '{}' WHERE id IN ($1, $2)`, s.fieldGuide, s.launchKit)
	var include bool
	s.scan(`SELECT prices_include_tax FROM earn_settings WHERE workspace_id = $1`, []any{s.ws.ID}, &include)
	behavior := map[bool]string{true: "inclusive", false: "exclusive"}[include]
	for _, tc := range []struct {
		product uuid.UUID
		name    string
		price   map[string]string
		pay     buyer
		total   string
	}{
		{s.fieldGuide, "The Deliverability Field Guide", map[string]string{"unit_amount": "2900", "currency": "usd", "tax_behavior": behavior}, buyer{}, "2900"},
		{s.launchKit, "Newsletter Launch Kit", map[string]string{"custom_unit_amount[enabled]": "true", "custom_unit_amount[minimum]": fmt.Sprint(lk.MinCents),
			"custom_unit_amount[preset]": fmt.Sprint(lk.SuggestedCents), "currency": "usd", "tax_behavior": behavior}, buyer{}, fmt.Sprint(lk.SuggestedCents)},
	} {
		before = len(s.stripe.Requests())
		_, session = s.checkout(tc.product, "", false)
		price := s.stripePrice(tc.product)
		var made []string
		for _, r := range s.requestsSince(before) {
			made = append(made, r.Method+" "+r.Path)
			switch r.Path {
			case "/v1/products":
				if r.Form.Get("name") != tc.name || r.Form.Get("metadata[reputecast_product]") != tc.product.String() {
					t.Errorf("%s: Stripe product %v", tc.name, r.Form)
				}
			case "/v1/prices":
				for k, want := range tc.price {
					if got := r.Form.Get(k); got != want {
						t.Errorf("%s: price %s = %q, want %q", tc.name, k, got, want)
					}
				}
				if tc.product == s.launchKit && r.Form.Has("unit_amount") {
					t.Errorf("%s: a pay-what-you-want price with a unit amount: %v", tc.name, r.Form)
				}
			case "/v1/checkout/sessions":
				if r.Form.Get("line_items[0][price]") != price || price == "" {
					t.Errorf("%s: session price %q, kept %q", tc.name, r.Form.Get("line_items[0][price]"), price)
				}
			}
		}
		if strings.Join(made, ", ") != "POST /v1/products, POST /v1/prices, POST /v1/checkout/sessions" {
			t.Errorf("%s: Stripe calls %v", tc.name, made)
		}
		s.pay(session, tc.pay)
		if cs := s.stripe.Get(payments.Live, session); fmt.Sprint(cs["amount_total"]) != tc.total {
			t.Errorf("%s: paid %v, want %s", tc.name, cs["amount_total"], tc.total)
		}
		// The next Buy uses the kept price.
		before = len(s.stripe.Requests())
		s.checkout(tc.product, "", false)
		for _, r := range s.requestsSince(before) {
			if r.Path == "/v1/prices" {
				t.Errorf("%s: a second price was made", tc.name)
			}
		}
		if p := s.stripePrice(tc.product); p != price {
			t.Errorf("%s: kept price changed from %s to %s", tc.name, price, p)
		}
	}

	// A tick on a product whose offer is off isn't consent to anything.
	s.exec(`UPDATE products SET live = jsonb_set(live, '{offerOn}', 'false') WHERE id = $1`, s.fieldGuide)
	_, session = s.checkout(s.fieldGuide, "", true)
	if co := s.checkoutOf(session); co.Consent["offered"] != false || co.Consent["ticked"] != false {
		t.Errorf("tick without an offer: %v", co.Consent)
	}
}

// Only a published product sells, and only while Stripe is connected: a
// draft, an archived product or an unknown one answers ErrNotForSale before
// anything reaches Stripe, and so does every product once Stripe is
// disconnected. Stripe failing answers stripe_unavailable and keeps the
// checkout as failed.
func TestCheckoutOnlyForProductsOnSale(t *testing.T) {
	s := newShop(t)
	n := len(s.stripe.Requests())
	checkouts := s.count(`SELECT count(*) FROM product_checkouts WHERE workspace_id = $1`, s.ws.ID)
	notForSale := func(what string, product uuid.UUID) {
		t.Helper()
		if _, err := s.press(product, "", true); !errors.Is(err, products.ErrNotForSale) {
			t.Errorf("%s: %v, want ErrNotForSale", what, err)
		}
	}
	notForSale("the draft Warm-up planner", s.planner)
	notForSale("an unknown product", uuid.New())
	s.owner.Do("POST", ws+"/products/"+s.launchKit.String()+"/archive", nil).MustStatus(t, 200)
	notForSale("the archived Launch Kit", s.launchKit)
	if len(s.stripe.Requests()) != n || s.count(`SELECT count(*) FROM product_checkouts WHERE workspace_id = $1`, s.ws.ID) != checkouts {
		t.Fatal("a product not for sale reached Stripe or left a checkout")
	}
	// Selling again once unarchived.
	s.owner.Do("POST", ws+"/products/"+s.launchKit.String()+"/unarchive", nil).MustStatus(t, 200)
	s.checkout(s.launchKit, "", false)

	// Stripe down (every attempt: the client tries three times): no
	// session, the checkout kept as failed.
	s.stripe.FailNext("POST", "/v1/checkout/sessions", 503, 3)
	_, err := s.press(s.fieldGuide, "", false)
	var p *api.Problem
	if !errors.As(err, &p) || p.Status != 503 || p.Code != "stripe_unavailable" {
		t.Fatalf("Stripe down: %v", err)
	}
	if n := s.count(`SELECT count(*) FROM product_checkouts WHERE product_id = $1 AND status = 'failed' AND session_id = ''`, s.fieldGuide); n != 1 {
		t.Errorf("%d failed checkouts, want 1", n)
	}

	// Stripe disconnected: the test key, then the live one.
	type conn struct {
		ID      uuid.UUID
		Mode    string
		Account string
	}
	var conns []conn
	rows, err := s.env.App.Pool.Query(s.ctx, `SELECT id, mode, account_name FROM payment_connections WHERE workspace_id = $1 AND status = 'connected' ORDER BY mode DESC`, s.ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var c conn
		_ = rows.Scan(&c.ID, &c.Mode, &c.Account)
		conns = append(conns, c)
	}
	rows.Close()
	for _, c := range conns {
		s.owner.Do("POST", ws+"/earn/connections/"+c.ID.String()+"/disconnect", map[string]any{"confirm": c.Account}).MustStatus(t, 204)
	}
	n = len(s.stripe.Requests())
	notForSale("the Field Guide without Stripe", s.fieldGuide)
	notForSale("the Launch Kit without Stripe", s.launchKit)
	for _, r := range s.requestsSince(n) {
		t.Errorf("reached Stripe without a connection: %s %s", r.Method, r.Path)
	}
}

// orderRow checks an order's money and buyer.
func checkOrder(t *testing.T, o store.ProductOrder, product uuid.UUID, email string, amount int64) {
	t.Helper()
	if o.ProductID != product || o.Status != products.OrderPaid || o.AmountCents != amount || o.Currency != "usd" || o.BuyerEmail != email ||
		o.Mode != "live" || o.Provider != "stripe" || o.RefundedCents != 0 || o.AccessRevokedAt != nil {
		t.Fatalf("order: %+v", o)
	}
}

// Back from Stripe before its webhook, the thank-you page reads the payment
// from Stripe and makes the order: Paid, numbered, its purchase link with
// the product's days and downloads, the delivery email queued, the checkout
// completed and the sale counted on its page. Stripe's webhook then finds
// the order made and makes nothing twice.
func TestPaidOnTheThanksPage(t *testing.T) {
	s := newShop(t)
	var last int64
	s.scan(`SELECT last FROM product_order_numbers WHERE workspace_id = $1`, []any{s.ws.ID}, &last)
	_, session := s.checkout(s.fieldGuide, "", false)
	// Not paid yet: the page waits.
	if _, ok, err := s.svc.Thanks(s.ctx, &s.ws, s.fieldGuide, session); ok || err != nil {
		t.Fatalf("before paying: %v %v", ok, err)
	}
	paid := s.pay(session, buyer{Email: "Robin.Hart@Example.com", Name: "Robin Hart"})
	// Another product's thank-you page doesn't show it, nor an unknown session.
	if _, ok, _ := s.svc.Thanks(s.ctx, &s.ws, s.launchKit, session); ok {
		t.Error("the Launch Kit's thank-you page shows a Field Guide order")
	}
	if _, ok, _ := s.svc.ThanksFor(s.ctx, &s.ws, "cs_live_unknown"); ok {
		t.Error("an unknown session shows an order")
	}

	v, ok, err := s.svc.Thanks(s.ctx, &s.ws, s.fieldGuide, session)
	if err != nil || !ok {
		t.Fatalf("thank-you page: %v %v", ok, err)
	}
	o := v.Order
	checkOrder(t, o, s.fieldGuide, "robin.hart@example.com", 2900)
	fg := s.liveOf(s.fieldGuide)
	link := s.link(o.ID)
	if o.Number != last+1 || o.BuyerName != "Robin Hart" || o.PaymentIntent != paid.PaymentIntent || o.SessionID != session || o.Buyer != "none" || o.ContactID != nil ||
		o.PayWhatYouWant || o.Source != products.SourceProductPage {
		t.Errorf("order: %+v", o)
	}
	if v.ProductName != "The Deliverability Field Guide" || v.EmailMasked != "r•••@example.com" || v.Downloads != fg.LinkDownloads || v.Test || v.OfferPending ||
		!v.ExpiresAt.Equal(s.clock.now().Add(time.Duration(fg.LinkDays)*24*time.Hour)) || v.DownloadURL != s.svc.DownloadURL(s.ctx, &s.ws, o.ID, link.ID) ||
		!strings.HasSuffix(v.DownloadURL, "/d/"+s.token(o.ID, link.ID)) || v.Files.Count == 0 {
		t.Errorf("thank-you page: %+v", v)
	}
	if link.Reason != "purchase" || link.DownloadsAllowed != fg.LinkDownloads || link.DownloadsUsed != 0 || s.deliveryJobs(link.ID) != 1 {
		t.Errorf("purchase link: %+v, %d delivery jobs", link, s.deliveryJobs(link.ID))
	}
	if ev := s.events(o.ID); strings.Join(ev, ",") != "checkout_started,paid" {
		t.Errorf("activity: %v", ev)
	}
	if d := s.event(o.ID, "paid"); d["stripe"] != "checkout.session.completed" {
		t.Errorf("paid event: %v", d)
	}
	if co := s.checkoutOf(session); co.Status != "completed" || co.OrderID == nil || *co.OrderID != o.ID {
		t.Errorf("checkout after paying: %+v", co)
	}
	if n := s.count(`SELECT count(*) FROM page_conversions WHERE order_id = $1 AND kind = 'sale' AND amount_cents = 2900 AND source = 'product_page'`, o.ID); n != 1 {
		t.Errorf("%d sales counted", n)
	}

	// The page again, through the page's thank-you address: the same order,
	// no new link.
	v2, ok, err := s.svc.ThanksFor(s.ctx, &s.ws, session)
	if err != nil || !ok || v2.Order.ID != o.ID || v2.DownloadURL != v.DownloadURL || len(s.links(o.ID)) != 1 {
		t.Fatalf("thank-you page again: %v %v %+v", ok, err, v2.Order)
	}

	// Stripe's webhook arrives after: the order is made already.
	if st, out := s.hook(paid.Events[0]); st != "ignored" || out != "order_paid_already" {
		t.Errorf("webhook after the page: %s %s", st, out)
	}
	if n := s.count(`SELECT count(*) FROM product_orders WHERE checkout_id = $1`, s.checkoutOf(session).ID); n != 1 {
		t.Errorf("%d orders for one checkout", n)
	}
	if len(s.links(o.ID)) != 1 || s.deliveryJobs(link.ID) != 1 || len(s.events(o.ID)) != 2 {
		t.Errorf("the webhook made things twice: links %d, jobs %d, events %v", len(s.links(o.ID)), s.deliveryJobs(link.ID), s.events(o.ID))
	}
}

// Stripe's checkout.session.completed on the webhook makes the order Paid,
// once however often it comes. The delivery email goes through the
// Purchases category's "Product delivery" template on the Transactional
// route, to the order's address only; its download_url is sensitive, so the
// link is in none of the stored rows: not the message log, not its content,
// not the order's activity, jobs, webhooks or audit entries. The order.paid
// webhook carries no link either.
func TestPaidByTheWebhook(t *testing.T) {
	s := newShop(t)
	s.deliverable()
	hook := s.subscribeOrders()
	start := time.Now().Add(-time.Second)
	_, session := s.checkout(s.fieldGuide, "", false)
	paid := s.pay(session, buyer{Email: "jo.lindqvist@example.se", Name: "Jo Lindqvist", Tax: 232})
	id := s.deliverHook(paid.Events[0])
	if err := s.env.App.Earn().Apply(s.ctx, id); err != nil {
		t.Fatal(err)
	}
	var st, outcome string
	var contact *uuid.UUID
	var amount *int64
	s.scan(`SELECT status, outcome, contact_id, amount_cents FROM payment_events WHERE id = $1`, []any{id}, &st, &outcome, &contact, &amount)
	if st != "applied" || outcome != "order_paid" || contact != nil || amount == nil || *amount != 2900 {
		t.Fatalf("payments event log: %s %s %v %v", st, outcome, contact, amount)
	}
	o := s.orderOf(session)
	checkOrder(t, o, s.fieldGuide, "jo.lindqvist@example.se", 2900)
	if o.TaxCents != 232 || o.BuyerName != "Jo Lindqvist" || o.PaymentIntent != paid.PaymentIntent || o.Buyer != "none" {
		t.Errorf("order: %+v", o)
	}
	// Stripe sends it again: stored once, and the consumer itself makes the
	// order once.
	if again := s.deliverHook(paid.Events[0]); again != id || s.count(`SELECT count(*) FROM payment_events WHERE provider_event_id = $1`, paid.Events[0].ID) != 1 {
		t.Errorf("a retried event was stored again")
	}
	if out, ok := s.consume(paid.Events[0]); !ok || out.Outcome != "order_paid_already" || !out.Ignored {
		t.Errorf("consumed again: %+v %v", out, ok)
	}
	if n := s.count(`SELECT count(*) FROM product_orders WHERE session_id = $1`, session); n != 1 || len(s.links(o.ID)) != 1 {
		t.Fatalf("%d orders, %d links", n, len(s.links(o.ID)))
	}
	// The thank-you page shows the order the webhook made.
	if v, ok, err := s.svc.Thanks(s.ctx, &s.ws, s.fieldGuide, session); err != nil || !ok || v.Order.ID != o.ID {
		t.Fatalf("thank-you page: %v %v", ok, err)
	}

	// The delivery email.
	link := s.link(o.ID)
	s.deliver(link.ID)
	link = s.link(o.ID)
	if link.Delivery != "sent" || link.MessageID == nil {
		t.Fatalf("link after delivery: %+v", link)
	}
	msg, to := s.deliveryMessage(link.ID)
	if to != "jo.lindqvist@example.se" {
		t.Fatalf("delivery email to %s", to)
	}
	var key, category, kind, toName string
	var sensitive []string
	var data []byte
	s.scan(`SELECT template_key, category_name, category_kind, to_name, sensitive, data FROM transactional_messages WHERE message_id = $1`, []any{msg},
		&key, &category, &kind, &toName, &sensitive, &data)
	var logged map[string]any
	_ = json.Unmarshal(data, &logged)
	if key != products.TemplateKey || category != products.CategoryName || kind != "essential" || toName != "Jo Lindqvist" || strings.Join(sensitive, ",") != "download_url" {
		t.Errorf("message log: %s %s %s %s %v", key, category, kind, toName, sensitive)
	}
	if logged["download_url"] != "••••••" || logged["product_name"] != "The Deliverability Field Guide" || logged["order_number"] != fmt.Sprint(o.Number) ||
		logged["amount"] != "$29.00" || logged["first_name"] != "Jo" {
		t.Errorf("logged data: %v", logged)
	}
	if n := s.sendMail(); n != 1 {
		t.Fatalf("sent %d", n)
	}
	mail := s.sink.Messages()
	if len(mail) != 1 || len(mail[0].To) != 1 || mail[0].To[0] != "jo.lindqvist@example.se" {
		t.Fatalf("mail: %+v", mail)
	}
	token := s.token(o.ID, link.ID)
	body := unfold(mail[0].Data)
	for _, want := range []string{s.svc.DownloadURL(s.ctx, &s.ws, o.ID, link.ID), "/d/" + token, "Your download: The Deliverability Field Guide", fmt.Sprint(o.Number)} {
		if !strings.Contains(body, want) {
			t.Errorf("the delivery email lacks %q", want)
		}
	}
	if where := s.holding(o.ID, token, start); len(where) > 0 {
		t.Errorf("the download link is stored in %v", where)
	}

	// The order.paid webhook: the order, no link.
	var paidHook map[string]any
	for _, p := range s.hooked(hook) {
		if p["type"] == "order.paid" {
			paidHook, _ = p["data"].(map[string]any)
		}
	}
	if paidHook == nil || fmt.Sprint(paidHook["order"]) != fmt.Sprint(o.Number) || paidHook["email"] != o.BuyerEmail || paidHook["status"] != "paid" {
		t.Errorf("order.paid: %v (all: %v)", paidHook, s.hooked(hook))
	}

	// The order panel shows the delivery through the template.
	var d products.ProductOrderDetail
	s.owner.Do("GET", ws+"/products/orders/"+o.ID.String(), nil).MustStatus(t, 200).JSON(t, &d)
	if d.Delivery == nil || d.Delivery.Template != products.TemplateKey || d.Delivery.MessageID != msg.String() || d.Download.State != "delivered" ||
		d.Buyer.Kind != "none" || d.Payment.TaxCents != 232 {
		t.Errorf("order panel: delivery %+v, download %+v, buyer %+v", d.Delivery, d.Download, d.Buyer)
	}

	// A delivery email that can't go out (the product's template is gone):
	// the link says why, and the people who manage orders hear of it.
	s.exec(`UPDATE products SET live = jsonb_set(live, '{deliveryTemplate}', to_jsonb($2::text)) WHERE id = $1`, s.fieldGuide, uuid.Nil.String())
	o2, _ := s.buy(s.fieldGuide, buyer{Email: "kim.nodelivery@example.com"})
	failed := s.link(o2.ID)
	s.deliver(failed.ID)
	if failed = s.link(o2.ID); failed.Delivery != "failed" || failed.MessageID != nil || failed.DeliveryError == "" {
		t.Fatalf("an undeliverable link: %+v", failed)
	}
	if n := s.count(`SELECT count(*) FROM notifications WHERE workspace_id = $1 AND kind = 'product_delivery_failed' AND data->>'order_id' = $2`, s.ws.ID, o2.ID.String()); n == 0 {
		t.Error("nobody heard of the failed delivery")
	}
	var d2 products.ProductOrderDetail
	s.owner.Do("GET", ws+"/products/orders/"+o2.ID.String(), nil).MustStatus(t, 200).JSON(t, &d2)
	if d2.Download.State != "failed" || d2.Download.Error != failed.DeliveryError || d2.Delivery != nil {
		t.Errorf("order panel of the failed delivery: %+v %+v", d2.Download, d2.Delivery)
	}
}

// An order's number reads as it's written everywhere else ("Order 1311"),
// not as a quantity: in the contact's timeline, and in the notifications of
// a dispute and of a delivery email that couldn't go out.
func TestOrderNumbersInWords(t *testing.T) {
	s := newShop(t)
	reader := s.addContact("rowan.numbers@example.com", "Rowan")
	o, paid := s.buy(s.fieldGuide, buyer{Email: "rowan.numbers@example.com"})
	if o.Number < 1000 {
		t.Fatalf("order %d is too small to show a thousands separator", o.Number)
	}
	var timeline struct {
		Items []struct{ Type, Detail string }
	}
	s.owner.Do("GET", ws+"/contacts/"+reader.String()+"/activity?limit=50", nil).MustStatus(t, 200).JSON(t, &timeline)
	for _, it := range timeline.Items {
		if it.Type == "bought" && !strings.Contains(it.Detail, fmt.Sprintf("order #%d", o.Number)) {
			t.Errorf("timeline: %q", it.Detail)
		}
	}
	_, ev, err := s.stripe.Dispute(payments.Live, paid.Charge, "fraudulent", time.Now().Add(9*24*time.Hour).Unix())
	if err != nil {
		t.Fatal(err)
	}
	s.hook(ev)
	var body string
	s.scan(`SELECT body FROM notifications WHERE workspace_id = $1 AND kind = 'order_disputed' AND data->>'order_id' = $2 ORDER BY created_at LIMIT 1`,
		[]any{s.ws.ID, o.ID.String()}, &body)
	if !strings.Contains(body, fmt.Sprintf("order %d ", o.Number)) {
		t.Errorf("dispute notification: %s", body)
	}
	s.exec(`UPDATE products SET live = jsonb_set(live, '{deliveryTemplate}', to_jsonb($2::text)) WHERE id = $1`, s.fieldGuide, uuid.Nil.String())
	o2, _ := s.buy(s.fieldGuide, buyer{Email: "rowan.numbers@example.com"})
	s.deliver(s.link(o2.ID).ID)
	s.scan(`SELECT body FROM notifications WHERE workspace_id = $1 AND kind = 'product_delivery_failed' AND data->>'order_id' = $2 ORDER BY created_at LIMIT 1`,
		[]any{s.ws.ID, o2.ID.String()}, &body)
	if !strings.Contains(body, fmt.Sprintf("order %d ", o2.Number)) {
		t.Errorf("delivery failed notification: %s", body)
	}
}

// refund refunds part of a payment in Stripe (0: what's left) and applies
// Stripe's charge.refunded.
func (s *shop) refund(charge string, amount int64) (string, string) {
	s.t.Helper()
	ev, err := s.stripe.Refund(payments.Live, charge, amount)
	if err != nil {
		s.t.Fatal(err)
	}
	return s.hook(ev)
}

// downloadState is what a link's download page says.
func (s *shop) downloadState(order, link uuid.UUID) string {
	s.t.Helper()
	v, err := s.svc.Download(s.ctx, &s.ws, s.token(order, link))
	if err != nil {
		s.t.Fatalf("download page: %v", err)
	}
	return v.State
}

// Refunds happen in Stripe and arrive on its webhook: in full, the order is
// Refunded and, as the product says by default, its downloads revoked; with
// "revoke on refund" off they keep working. A partial refund leaves the
// order Partially refunded with its downloads, until the rest is refunded.
// Each is in the order's activity, the audit log (by the system), the
// order.refunded webhook and the contact's timeline; an event Stripe sends
// again changes nothing.
func TestRefunds(t *testing.T) {
	s := newShop(t)
	hook := s.subscribeOrders()
	reader := s.addContact("avery.stone@example.com", "Avery")

	// In full, revoke on refund on (the Field Guide's default).
	if !s.liveOf(s.fieldGuide).RevokeOnRefund {
		t.Fatal("the demo's Field Guide doesn't revoke on refund")
	}
	o, paid := s.buy(s.fieldGuide, buyer{Email: "avery.stone@example.com", Name: "Avery Stone"})
	if o.ContactID == nil || *o.ContactID != reader {
		t.Fatalf("buyer not linked to the contact: %+v", o)
	}
	link := s.link(o.ID)
	var buyers struct{ Total int64 }
	s.owner.Do("GET", ws+"/contacts?bought="+s.fieldGuide.String()+"&q=avery.stone", nil).MustStatus(t, 200).JSON(t, &buyers)
	if buyers.Total != 1 {
		t.Errorf("the buyer isn't among the Field Guide's buyers: %d", buyers.Total)
	}
	if st, out := s.refund(paid.Charge, 0); st != "applied" || out != "order_refunded" {
		t.Fatalf("full refund: %s %s", st, out)
	}
	o = s.order(o.ID)
	if o.Status != products.OrderRefunded || o.RefundedCents != 2900 || o.RefundedAt == nil || o.AccessRevokedAt == nil || o.AccessRevokeReason != "refund" ||
		o.ChargeID != paid.Charge {
		t.Fatalf("refunded order: %+v", o)
	}
	if ev := s.events(o.ID); strings.Join(ev[len(ev)-2:], ",") != "refunded,access_revoked" {
		t.Errorf("activity: %v", ev)
	}
	if d := s.event(o.ID, "refunded"); fmt.Sprint(d["amount"]) != "2900" || d["byStripe"] != true || d["stripe"] != "charge.refunded" {
		t.Errorf("refunded event: %v", d)
	}
	if d := s.event(o.ID, "access_revoked"); d["reason"] != "refund" || d["auto"] != true {
		t.Errorf("access_revoked event: %v", d)
	}
	a, ok := s.audited("order.refunded", o.ID)
	if !ok || a.ActorType != "system" || a.Metadata["summary"] != "Refunded in Stripe · download access revoked, as the product says" ||
		a.After["access"] != "revoked" || a.After["status"] != products.OrderRefunded || fmt.Sprint(a.After["refunded_cents"]) != "2900" {
		t.Errorf("audit: %v %+v", ok, a)
	}
	if st := s.downloadState(o.ID, link.ID); st != products.DownloadRefunded {
		t.Errorf("download page after the refund: %s", st)
	}
	// A refunded order's access can't be restored by hand.
	s.owner.Do("POST", ws+"/products/orders/"+o.ID.String()+"/restore", nil).MustStatus(t, 409)
	// The contact's timeline has both, and they no longer count as a buyer.
	var timeline struct {
		Items []struct{ Type, Subject, Detail string }
	}
	s.owner.Do("GET", ws+"/contacts/"+reader.String()+"/activity?limit=50", nil).MustStatus(t, 200).JSON(t, &timeline)
	kinds := map[string]string{}
	for _, it := range timeline.Items {
		kinds[it.Type] = it.Subject + " · " + it.Detail
	}
	// (How the order's number reads: TestOrderNumbersInWords.)
	if !strings.HasPrefix(kinds["bought"], "The Deliverability Field Guide · $29.00") || !strings.HasPrefix(kinds["refunded"], "The Deliverability Field Guide · $29.00 refunded") {
		t.Errorf("timeline: %v", timeline.Items)
	}
	s.owner.Do("GET", ws+"/contacts?bought="+s.fieldGuide.String()+"&q=avery.stone", nil).MustStatus(t, 200).JSON(t, &buyers)
	if buyers.Total != 0 {
		t.Errorf("a refunded buyer still counts as a buyer")
	}

	// Stripe sends the refund again: known, nothing changes.
	events := len(s.events(o.ID))
	last, _ := s.stripe.Refund(payments.Live, paid.Charge, 0) // nothing left: the same state again
	if st, out := s.hook(last); st != "ignored" || out != "order_refund_known" {
		t.Errorf("the refund again: %s %s", st, out)
	}
	if len(s.events(o.ID)) != events {
		t.Errorf("the refund again added activity: %v", s.events(o.ID))
	}

	// Partially, then the rest.
	o, paid = s.buy(s.fieldGuide, buyer{Email: "b.partial@example.com"})
	link = s.link(o.ID)
	if st, out := s.refund(paid.Charge, 1000); st != "applied" || out != "order_refunded" {
		t.Fatalf("partial refund: %s %s", st, out)
	}
	o = s.order(o.ID)
	if o.Status != products.OrderPartiallyRefunded || o.RefundedCents != 1000 || o.AccessRevokedAt != nil {
		t.Fatalf("partially refunded order: %+v", o)
	}
	if a, _ := s.audited("order.refunded", o.ID); a.Metadata["summary"] != "Partial refund of $10.00 in Stripe · downloads keep working" || a.After["access"] != nil {
		t.Errorf("partial refund audit: %+v", a)
	}
	if st := s.downloadState(o.ID, link.ID); st != products.DownloadOK {
		t.Errorf("download page after a partial refund: %s", st)
	}
	if st, out := s.refund(paid.Charge, 0); st != "applied" || out != "order_refunded" {
		t.Fatalf("the rest: %s %s", st, out)
	}
	o = s.order(o.ID)
	if o.Status != products.OrderRefunded || o.RefundedCents != 2900 || o.AccessRevokeReason != "refund" {
		t.Errorf("refunded in two parts: %+v", o)
	}
	if d := s.event(o.ID, "refunded"); fmt.Sprint(d["amount"]) != "1900" {
		t.Errorf("the second refund's event: %v", d)
	}

	// In full with "revoke on refund" off: the downloads keep working.
	s.exec(`UPDATE products SET live = jsonb_set(live, '{revokeOnRefund}', 'false') WHERE id = $1`, s.launchKit)
	o, paid = s.buy(s.launchKit, buyer{Email: "c.keeps@example.com", Amount: 1800})
	link = s.link(o.ID)
	if st, out := s.refund(paid.Charge, 0); st != "applied" || out != "order_refunded" {
		t.Fatalf("refund without revoking: %s %s", st, out)
	}
	o = s.order(o.ID)
	if o.Status != products.OrderRefunded || o.RefundedCents != 1800 || o.AccessRevokedAt != nil || !o.PayWhatYouWant {
		t.Fatalf("refunded, access kept: %+v", o)
	}
	if a, _ := s.audited("order.refunded", o.ID); a.Metadata["summary"] != "Refunded in Stripe · downloads keep working, as the product says" || a.After["access"] != nil {
		t.Errorf("audit: %+v", a)
	}
	if st := s.downloadState(o.ID, link.ID); st != products.DownloadOK {
		t.Errorf("download page: %s", st)
	}
	if ev := s.events(o.ID); ev[len(ev)-1] != "refunded" {
		t.Errorf("activity: %v", ev)
	}

	// The order.refunded webhooks: four refunds (the one Stripe sent again
	// isn't one).
	var refunded []string
	for _, p := range s.hooked(hook) {
		if p["type"] == "order.refunded" {
			d, _ := p["data"].(map[string]any)
			refunded = append(refunded, fmt.Sprintf("%v:%v:%v", d["order"], d["status"], d["refunded_cents"]))
		}
	}
	if len(refunded) != 4 {
		t.Errorf("order.refunded webhooks: %v", refunded)
	}
}

// Disputes arrive from Stripe too. Opened, the order is Disputed with
// Stripe's reason, fee and the date to respond by, members who manage
// orders are notified and, as the product says, the downloads are revoked
// (with "revoke on refund" off they keep working). Closed, the dispute is
// won (the order is Paid again) or lost (it stays Disputed). Each step is in
// the order's activity and the audit log.
func TestDisputes(t *testing.T) {
	s := newShop(t)
	hook := s.subscribeOrders()
	due := time.Now().Add(9 * 24 * time.Hour).Truncate(time.Second)
	// notified is the dispute notification of an order (every member who
	// manages orders gets one).
	notified := func(order uuid.UUID) string {
		t.Helper()
		var body string
		s.scan(`SELECT body FROM notifications WHERE workspace_id = $1 AND kind = 'order_disputed' AND data->>'order_id' = $2 ORDER BY created_at LIMIT 1`,
			[]any{s.ws.ID, order.String()}, &body)
		return body
	}

	open := func(charge string) string {
		t.Helper()
		dp, ev, err := s.stripe.Dispute(payments.Live, charge, "fraudulent", due.Unix())
		if err != nil {
			t.Fatal(err)
		}
		if st, out := s.hook(ev); st != "applied" || out != "order_disputed" {
			t.Fatalf("dispute opened: %s %s", st, out)
		}
		return dp
	}
	closeDispute := func(dp string, won bool) {
		t.Helper()
		ev, err := s.stripe.CloseDispute(payments.Live, dp, won)
		if err != nil {
			t.Fatal(err)
		}
		if st, out := s.hook(ev); st != "applied" || out != "order_dispute_closed" {
			t.Fatalf("dispute closed: %s %s", st, out)
		}
	}

	// Opened, then won.
	o, paid := s.buy(s.fieldGuide, buyer{Email: "dee.dispute@example.com"})
	link := s.link(o.ID)
	dp := open(paid.Charge)
	o = s.order(o.ID)
	if o.Status != products.OrderDisputed || o.DisputeStatus != "open" || o.DisputeReason != "fraudulent" || o.DisputeID != dp || o.DisputeFeeCents != 1500 ||
		o.DisputedAt == nil || o.DisputeDueBy == nil || !o.DisputeDueBy.Equal(due) || o.AccessRevokedAt == nil || o.AccessRevokeReason != "dispute" {
		t.Fatalf("disputed order: %+v", o)
	}
	if ev := s.events(o.ID); strings.Join(ev[len(ev)-2:], ",") != "dispute_opened,access_revoked" {
		t.Errorf("activity: %v", ev)
	}
	if d := s.event(o.ID, "dispute_opened"); d["reason"] != "fraudulent" || d["stripe"] != "charge.dispute.created" || d["respondBy"] == nil {
		t.Errorf("dispute_opened: %v", d)
	}
	if d := s.event(o.ID, "access_revoked"); d["reason"] != "dispute" || d["auto"] != true {
		t.Errorf("access_revoked: %v", d)
	}
	if body := notified(o.ID); !strings.Contains(body, "dee.dispute@example.com") || !strings.Contains(body, "The Deliverability Field Guide") || !strings.Contains(body, "revoked") {
		t.Errorf("notification: %s", body)
	}
	if a, ok := s.audited("order.disputed", o.ID); !ok || a.ActorType != "system" || a.After["dispute"] != "open" || a.After["access"] != "revoked" {
		t.Errorf("audit: %v %+v", ok, a)
	}
	if st := s.downloadState(o.ID, link.ID); st != products.DownloadRevoked {
		t.Errorf("download page while disputed: %s", st)
	}
	var list products.ProductOrderList
	s.owner.Do("GET", ws+"/products/orders?status=disputed&period=all&q="+fmt.Sprint(o.Number), nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].Download.State != "revoked_auto" || list.Items[0].DisputeStatus != "open" {
		t.Errorf("orders filtered as disputed: %+v", list.Items)
	}
	closeDispute(dp, true)
	o = s.order(o.ID)
	if o.Status != products.OrderPaid || o.DisputeStatus != "won" {
		t.Errorf("dispute won: %+v", o)
	}
	if d := s.event(o.ID, "dispute_closed"); d["outcome"] != "won" || d["stripe"] != "charge.dispute.closed" {
		t.Errorf("dispute_closed: %v", d)
	}
	if a, ok := s.audited("order.dispute_closed", o.ID); !ok || a.ActorType != "system" || a.After["dispute"] != "won" {
		t.Errorf("audit: %v %+v", ok, a)
	}
	// Winning gives back the access the dispute took; nothing is left for a
	// person to restore.
	if o.AccessRevokedAt != nil || o.AccessRevokeReason != "" {
		t.Errorf("access stayed revoked after the won dispute: %+v", o)
	}
	if d := s.event(o.ID, "access_restored"); d["reason"] != "dispute_won" || d["auto"] != true {
		t.Errorf("access_restored: %v", d)
	}
	if a, ok := s.audited("order.dispute_closed", o.ID); !ok || a.After["access"] != "restored" {
		t.Errorf("audit of the restore: %v %+v", ok, a)
	}
	if st := s.downloadState(o.ID, link.ID); st != products.DownloadOK {
		t.Errorf("download page once the dispute is won: %s", st)
	}
	s.owner.Do("POST", ws+"/products/orders/"+o.ID.String()+"/restore", nil).MustStatus(t, 409)

	// Opened, then lost.
	o, paid = s.buy(s.fieldGuide, buyer{Email: "lou.lost@example.com"})
	dp = open(paid.Charge)
	closeDispute(dp, false)
	o = s.order(o.ID)
	if o.Status != products.OrderDisputed || o.DisputeStatus != "lost" || o.AccessRevokeReason != "dispute" {
		t.Errorf("dispute lost: %+v", o)
	}
	if d := s.event(o.ID, "dispute_closed"); d["outcome"] != "lost" {
		t.Errorf("dispute_closed: %v", d)
	}

	// "Revoke on refund" off: the dispute leaves the downloads alone.
	s.exec(`UPDATE products SET live = jsonb_set(live, '{revokeOnRefund}', 'false') WHERE id = $1`, s.launchKit)
	o, paid = s.buy(s.launchKit, buyer{Email: "kit.keeps@example.com", Amount: 900})
	link = s.link(o.ID)
	open(paid.Charge)
	o = s.order(o.ID)
	if o.Status != products.OrderDisputed || o.AccessRevokedAt != nil {
		t.Errorf("disputed, access kept: %+v", o)
	}
	if body := notified(o.ID); body == "" || strings.Contains(body, "revoked") {
		t.Errorf("notification says revoked: %s", body)
	}
	if st := s.downloadState(o.ID, link.ID); st != products.DownloadOK {
		t.Errorf("download page: %s", st)
	}

	// One order.disputed webhook per dispute opened.
	n := 0
	for _, p := range s.hooked(hook) {
		if p["type"] == "order.disputed" {
			n++
		}
	}
	if n != 3 {
		t.Errorf("%d order.disputed webhooks, want 3", n)
	}
}

// The in-process Stripe the demo's connections use (stripetest's
// HTTPClient, internal/app/wire_earn.go) also answers requests without a
// body: reading a payment back on the thank-you page (a GET) and deleting
// the webhook when Stripe is disconnected (a DELETE).
func TestInProcessStripeTakesRequestsWithoutABody(t *testing.T) {
	s := newShop(t)
	s.env.App.Earn().Demo = func() *http.Client { return s.stripe.HTTPClient() } // as the demo wires it
	_, session := s.checkout(s.fieldGuide, "", false)
	s.pay(session, buyer{Email: "plain.client@example.com"})
	var v products.ThanksView
	var ok bool
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panicked: %v", r)
			}
		}()
		v, ok, err = s.svc.Thanks(s.ctx, &s.ws, s.fieldGuide, session)
	}()
	if err != nil || !ok || v.Order.BuyerEmail != "plain.client@example.com" {
		t.Errorf("thank-you page through the in-process Stripe: %v %v", ok, err)
	}
	var test uuid.UUID
	s.scan(`SELECT id FROM payment_connections WHERE workspace_id = $1 AND mode = 'test' AND status = 'connected'`, []any{s.ws.ID}, &test)
	// The test key goes without the typed confirmation while a live key is connected.
	s.owner.Do("POST", ws+"/earn/connections/"+test.String()+"/disconnect", map[string]any{"confirm": ""}).MustStatus(t, 204)
}

// The demo sells with its own in-process Stripe (the fake its sample
// connections use, internal/app/wire_earn.go): Buy on the Field Guide opens
// a Checkout session there.
func TestDemoStripeSellsTheDemoProducts(t *testing.T) {
	env, _ := demo(t)
	ctx := context.Background()
	var wsID, product uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT w.id, p.id FROM workspaces w JOIN products p ON p.workspace_id = w.id
		WHERE w.slug = 'acme-newsletter' AND p.slug = 'deliverability-field-guide'`).Scan(&wsID, &product); err != nil {
		t.Fatal(err)
	}
	w, err := env.App.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		t.Fatal(err)
	}
	h, err := env.App.Products().StartCheckout(ctx, &w, products.CheckoutStart{ProductID: product, Source: products.SourceProductPage,
		PageURL: "http://news.acme.io/products/deliverability-field-guide", ThanksURL: "http://news.acme.io/products/deliverability-field-guide/thanks?session_id={CHECKOUT_SESSION_ID}"})
	if err != nil || !strings.HasPrefix(h.URL, "https://checkout.stripe.com/") {
		t.Fatalf("Buy in the demo: %+v %v", h, err)
	}
}

// A checkout.session.completed whose payment hasn't arrived yet (a delayed
// method, like a bank debit) waits for Stripe's
// checkout.session.async_payment_succeeded, which the webhook subscribes
// to: no order before, one after, and the first event is ignored as
// payment_pending rather than failed and retried. The thank-you page waits
// meanwhile.
func TestDelayedPaymentWaitsForStripe(t *testing.T) {
	if !slices.Contains(stripe.Events, "checkout.session.async_payment_succeeded") {
		t.Error("the webhook doesn't subscribe to checkout.session.async_payment_succeeded")
	}
	s := newShop(t)
	_, session := s.checkout(s.fieldGuide, "", false)
	cs := s.stripe.Get(payments.Live, session)
	cs["status"], cs["payment_status"] = "complete", "unpaid"
	cs["customer_details"] = map[string]any{"email": "bank.debit@example.de", "name": "Bank Debit"}
	cs["amount_total"], cs["currency"] = int64(2900), "usd"
	ev := s.stripe.Emit(payments.Live, "checkout.session.completed", cs, nil)
	id := s.deliverHook(ev)
	if err := s.env.App.Earn().Apply(s.ctx, id); err != nil {
		t.Fatalf("an unpaid session failed the event: %v", err)
	}
	var st, outcome string
	s.scan(`SELECT status, coalesce(outcome, '') FROM payment_events WHERE id = $1`, []any{id}, &st, &outcome)
	if st != "ignored" || outcome != "payment_pending" {
		t.Errorf("an unpaid session: %s %s, want ignored payment_pending", st, outcome)
	}
	if n := s.count(`SELECT count(*) FROM product_orders WHERE session_id = $1`, session); n != 0 {
		t.Fatalf("an order before the payment arrived")
	}
	if _, ok, err := s.svc.Thanks(s.ctx, &s.ws, s.fieldGuide, session); ok || err != nil {
		t.Errorf("the thank-you page before the payment arrived: %v %v", ok, err)
	}
	cs["payment_status"] = "paid"
	if st, out := s.hook(s.stripe.Emit(payments.Live, "checkout.session.async_payment_succeeded", cs, nil)); st != "applied" || out != "order_paid" {
		t.Fatalf("payment arrived: %s %s", st, out)
	}
	checkOrder(t, s.orderOf(session), s.fieldGuide, "bank.debit@example.de", 2900)
}
