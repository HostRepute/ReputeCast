package earn_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/payments/stripe/stripetest"
)

// G-11: Earn queues the paid-member automation events from the rows'
// states, whatever Stripe event moved them, once per subscription for
// "Becomes a paid member" and "Stops paying", in live mode only.
func TestAutomationEventsForPaidMembers(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	svc := env.App.Earn()
	ctx := context.Background()
	fake := stripetest.New()
	hook := env.Server.URL + "/hooks/stripe/7Gk2pQx9LmT4vB8sNw3eRfJh"
	deliver := func(ev stripetest.Event) {
		t.Helper()
		if code, err := stripetest.Deliver(ctx, nil, hook, "whsec_demoAcmeLiveSigningSecret0001", ev, time.Now()); err != nil || code != http.StatusOK {
			t.Fatalf("deliver %s: %d %v", ev.Type, code, err)
		}
		if _, err := svc.ApplyPending(ctx, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	count := func(kind, sub string, extra string) int {
		t.Helper()
		var n int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM automation_events e JOIN paid_subscriptions s ON e.ref = 'sub:' || s.id::text
			WHERE e.kind = $1 AND s.stripe_subscription_id = $2 AND e.contact_id = s.contact_id`+extra, kind, sub).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	price := func(id string, amount int64, interval, product string) map[string]any {
		return map[string]any{"id": id, "object": "price", "product": product, "unit_amount": amount, "currency": "usd", "active": true, "recurring": map[string]any{"interval": interval}}
	}
	subscription := func(id, customer, status string, p map[string]any, extra map[string]any) map[string]any {
		o := map[string]any{"id": id, "object": "subscription", "customer": customer, "status": status, "start_date": time.Date(2026, 10, 7, 7, 52, 0, 0, time.UTC).Unix(),
			"current_period_end": time.Now().AddDate(0, 6, 0).Unix(), "items": map[string]any{"data": []map[string]any{{"price": p}}}}
		for k, v := range extra {
			o[k] = v
		}
		return o
	}
	insiderYear := price("price_1Q8mL5cD", 8000, "year", "prod_QmA8x2Lw")
	insiderMonth := price("price_1Q8mL2aB", 800, "month", "prod_QmA8x2Lw")
	founding := price("price_1Q8mM1eF", 20000, "year", "prod_QmA9fM3k")

	// Sam paid before this release: his renewal is no new paid member.
	deliver(fake.Emit(payments.Live, "invoice.paid", map[string]any{"id": "in_samRenewal", "object": "invoice", "customer": "cus_QmT4wE8sLd", "subscription": "sub_1Q8xRb2Va7",
		"amount_paid": 8000, "amount_due": 8000, "currency": "usd", "billing_reason": "subscription_cycle", "status": "paid"}, nil))
	if n := count("paid_started", "sub_1Q8xRb2Va7", ""); n != 0 {
		t.Fatalf("a renewal from before the release started %d paid-member flows", n)
	}

	// Moving up to Founding member while scheduling the cancellation: the
	// timeline shows the cancellation, the tier change is still queued.
	deliver(fake.Emit(payments.Live, "customer.subscription.updated", subscription("sub_1Q8xRb2Va7", "cus_QmT4wE8sLd", "active", founding,
		map[string]any{"cancel_at_period_end": true}), map[string]any{"items": map[string]any{"data": []map[string]any{{"price": insiderYear}}}}))
	if n := count("paid_tier_changed", "sub_1Q8xRb2Va7", ` AND e.payload->>'direction' = 'upgrade' AND e.payload->>'from_tier' = 'Weekly Digest Insider'
		AND e.payload->'liquid'->'subscription'->>'tier' = 'Founding member'`); n != 1 {
		t.Fatalf("upgrade events: %d", n)
	}

	// Ending, the same event again, then a later update that says the same.
	deleted := fake.Emit(payments.Live, "customer.subscription.deleted", subscription("sub_1Q8xRb2Va7", "cus_QmT4wE8sLd", "canceled", founding,
		map[string]any{"ended_at": time.Now().Unix(), "canceled_at": time.Now().Unix()}), nil)
	deliver(deleted)
	deliver(deleted)
	deliver(fake.Emit(payments.Live, "customer.subscription.updated", subscription("sub_1Q8xRb2Va7", "cus_QmT4wE8sLd", "canceled", founding,
		map[string]any{"ended_at": time.Now().Unix()}), nil))
	if n := count("paid_ended", "sub_1Q8xRb2Va7", ` AND e.payload->>'reason' = 'canceled'`); n != 1 {
		t.Fatalf("paid_ended events: %d", n)
	}
	if n := count("paid_ended", "sub_1Q8xRb2Va7", ""); n != 1 {
		t.Fatalf("paid_ended events of any reason: %d", n)
	}

	// Diego's trial converts: a paid member, from a trial.
	var diegoSub, diegoCus string
	if err := env.App.Pool.QueryRow(ctx, `SELECT s.stripe_subscription_id, s.stripe_customer_id FROM paid_subscriptions s JOIN contacts c ON c.id = s.contact_id
		WHERE c.email = 'd.alvarez@yahoo.com' AND s.status = 'trialing' AND s.mode = 'live'`).Scan(&diegoSub, &diegoCus); err != nil {
		t.Fatal(err)
	}
	deliver(fake.Emit(payments.Live, "customer.subscription.updated", subscription(diegoSub, diegoCus, "active", insiderMonth, nil), map[string]any{"status": "trialing"}))
	if n := count("paid_started", diegoSub, ` AND (e.payload->>'from_trial')::boolean`); n != 1 {
		t.Fatalf("paid_started from a trial: %d", n)
	}
	// His renewal after that is no second start.
	deliver(fake.Emit(payments.Live, "invoice.paid", map[string]any{"id": "in_diegoFirst", "object": "invoice", "customer": diegoCus, "subscription": diegoSub,
		"amount_paid": 800, "amount_due": 800, "currency": "usd", "billing_reason": "subscription_cycle", "status": "paid"}, nil))
	if n := count("paid_started", diegoSub, ""); n != 1 {
		t.Fatalf("paid_started after a renewal: %d", n)
	}
}

// Test mode never starts a paid-member flow.
func TestAutomationEventsSkipTestMode(t *testing.T) {
	f := newFlow(t)
	f.connect()
	f.createTier()
	form := url.Values{"tier": {f.tier}, "period": {"month"}}
	r := f.c.DoRaw(http.MethodPost, "/p/"+f.ws.Slug+"/upgrade/checkout", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	session := strings.TrimPrefix(r.Header.Get("Location"), "https://checkout.stripe.com/c/pay/")
	evs, err := f.fake.CompleteCheckout(payments.Test, session, stripetest.CheckoutOptions{Email: "tester@flow.example"})
	if err != nil {
		t.Fatal(err)
	}
	f.deliver(time.Now(), evs...)
	f.apply()
	if len(f.members("with_access")) != 1 {
		t.Fatal("the test checkout made no paid member")
	}
	if n := f.count(`SELECT count(*) FROM automation_events WHERE workspace_id = $1`, f.ws.ID); n != 0 {
		t.Fatalf("test mode queued %d automation events", n)
	}
	var marked int
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM paid_subscriptions WHERE workspace_id = $1 AND automation_started_at IS NOT NULL`, f.ws.ID).Scan(&marked); err != nil {
		t.Fatal(err)
	}
	if marked != 0 {
		t.Fatalf("test mode marked %d subscriptions", marked)
	}
}
