package stripe_test

import (
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/payments/stripe"
	"github.com/hostrepute/reputecast/internal/payments/stripe/stripetest"
)

// One-time payments: a fixed and a pay-what-you-want price, a payment-mode
// checkout, and the events of paying, refunding and disputing it.
func TestOneTimePayments(t *testing.T) {
	ctx := context.Background()
	env, f := newEnv(t)
	c, err := env.Open(liveKey)
	if err != nil {
		t.Fatal(err)
	}
	pay, ok := c.(payments.OneTimePayments)
	if !ok {
		t.Fatal("the Stripe client sells one-time prices")
	}
	prod, err := c.SaveProduct(ctx, payments.Product{Name: "The Deliverability Field Guide", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	fixed, err := pay.CreateOneTimePrice(ctx, payments.OneTimePrice{ProductID: prod.ID, AmountCents: 2900, Currency: "USD"})
	if err != nil || fixed.AmountCents != 2900 || fixed.Interval != "" {
		t.Fatalf("fixed price: %+v %v", fixed, err)
	}
	pwyw, err := pay.CreateOneTimePrice(ctx, payments.OneTimePrice{ProductID: prod.ID, Custom: &payments.CustomAmount{MinimumCents: 500, PresetCents: 1500}, Currency: "usd"})
	if err != nil {
		t.Fatal(err)
	}
	if o := f.Get(payments.Live, pwyw.ID); o["custom_unit_amount"] == nil {
		t.Fatalf("custom amount kept: %+v", o)
	}

	cs, err := pay.CreatePaymentCheckout(ctx, payments.PaymentCheckoutRequest{PriceID: pwyw.ID, CustomerEmail: "marco.bianchi@example.it", Tax: true,
		SuccessURL: "https://news.acme.io/products/launch-kit/thanks?session={CHECKOUT_SESSION_ID}", CancelURL: "https://news.acme.io/launch-kit?checkout=cancelled",
		ClientRef: "chk_1", Metadata: map[string]string{"reputecast_checkout": "chk_1"}})
	if err != nil || cs.URL == "" {
		t.Fatalf("checkout: %+v %v", cs, err)
	}
	req := f.Requests()[len(f.Requests())-1]
	if req.Form.Get("mode") != "payment" || req.Form.Get("payment_intent_data[metadata][reputecast_checkout]") != "chk_1" || req.Form.Get("subscription_data[metadata][reputecast_checkout]") != "" {
		t.Fatalf("payment mode form: %v", req.Form)
	}
	paid, err := f.CompletePayment(payments.Live, cs.ID, stripePaymentOptions(2000))
	if err != nil {
		t.Fatal(err)
	}
	parser := stripe.Verifier{}
	p, err := parser.ParsePayment(paid.Events[0].Body)
	if err != nil {
		t.Fatal(err)
	}
	if p.Kind != payments.PaymentCompleted || p.Mode != "payment" || !p.Paid || p.AmountCents != 2000 || p.PaymentIntentID != paid.PaymentIntent ||
		p.ClientRef != "chk_1" || p.CustomerEmail != "marco.bianchi@example.it" || p.Metadata["reputecast_checkout"] != "chk_1" || !p.Livemode {
		t.Fatalf("completed: %+v", p)
	}
	got, err := pay.GetPayment(ctx, cs.ID)
	if err != nil || got.PaymentIntentID != paid.PaymentIntent || got.AmountCents != 2000 {
		t.Fatalf("read back: %+v %v", got, err)
	}

	ref, err := f.Refund(payments.Live, paid.Charge, 500)
	if err != nil {
		t.Fatal(err)
	}
	r, err := parser.ParsePayment(ref.Body)
	if err != nil || r.Kind != payments.PaymentRefunded || r.RefundCents != 500 || r.RefundedCents != 500 || r.PaymentIntentID != paid.PaymentIntent {
		t.Fatalf("refund: %+v %v", r, err)
	}
	dp, opened, err := f.Dispute(payments.Live, paid.Charge, "fraudulent", 1792800000)
	if err != nil {
		t.Fatal(err)
	}
	o, err := parser.ParsePayment(opened.Body)
	if err != nil || o.Kind != payments.PaymentDisputeOpened || o.DisputeReason != "fraudulent" || o.DisputeStatus != "needs_response" || o.DisputeDueBy == nil ||
		o.PaymentIntentID != paid.PaymentIntent || o.DisputeFee != 1500 {
		t.Fatalf("dispute opened: %+v %v", o, err)
	}
	closed, err := f.CloseDispute(payments.Live, dp, false)
	if err != nil {
		t.Fatal(err)
	}
	cl, err := parser.ParsePayment(closed.Body)
	if err != nil || cl.Kind != payments.PaymentDisputeClosed || cl.DisputeStatus != "lost" || cl.DisputeID != dp {
		t.Fatalf("dispute closed: %+v %v", cl, err)
	}
	// A subscription's checkout isn't a one-time payment.
	ev := f.Emit(payments.Live, "checkout.session.completed", map[string]any{"id": "cs_sub", "object": "checkout.session", "mode": "subscription"}, nil)
	if s, _ := parser.ParsePayment(ev.Body); s.Kind != payments.PaymentOther {
		t.Fatalf("subscription checkout: %+v", s)
	}
}

func stripePaymentOptions(amount int64) stripetest.PaymentOptions {
	return stripetest.PaymentOptions{Email: "marco.bianchi@example.it", Name: "Marco Bianchi", AmountCents: amount}
}
