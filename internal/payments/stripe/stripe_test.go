package stripe_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/payments/contracttest"
	"github.com/hostrepute/reputecast/internal/payments/stripe"
	"github.com/hostrepute/reputecast/internal/payments/stripe/stripetest"
)

const (
	liveKey     = "rk_live_51ContractLiveKey9fQ2"
	testKey     = "rk_test_51ContractTestKey3mZ8"
	missingKey  = "rk_live_51ContractMissingKeyX1"
	rejectedKey = "rk_live_51ContractRejectedKeyX2"
)

func newEnv(t *testing.T) (*contracttest.Env, *stripetest.Fake) {
	f := stripetest.New()
	f.AddKey(liveKey, stripetest.KeySpec{})
	f.AddKey(testKey, stripetest.KeySpec{})
	f.AddKey(missingKey, stripetest.KeySpec{Missing: []string{"webhook_endpoints"}})
	f.AddKey(rejectedKey, stripetest.KeySpec{Rejected: true})
	srv := f.Server(t)
	env := &contracttest.Env{
		Open: func(key string) (payments.Client, error) {
			return stripe.New(payments.Config{Key: key, BaseURL: srv.URL})
		},
		LiveKey: liveKey, TestKey: testKey, MissingKey: missingKey, RejectedKey: rejectedKey, MissingPermission: "webhook_endpoints",
		SecretKey: "sk_live_51ContractSecretKeyZ9",
		Requests:  func() int { return len(f.Requests()) },
		Event: func(t *testing.T, _ payments.Client) []byte {
			return f.Emit(payments.Live, "invoice.paid", map[string]any{"id": "in_contract", "object": "invoice", "customer": "cus_contract",
				"subscription": "sub_contract", "amount_paid": 800, "amount_due": 800, "currency": "usd", "billing_reason": "subscription_cycle"}, nil).Body
		},
		Sign: func(secret string, body []byte, ts time.Time) http.Header {
			h := http.Header{}
			h.Set("Stripe-Signature", stripe.SignatureHeader(secret, ts.Unix(), body))
			return h
		},
		Outage: func() { f.FailNext(http.MethodPost, "/v1/", 503, 100) },
	}
	return env, f
}

func TestContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{Adapter: stripe.Key, Env: func(t *testing.T) *contracttest.Env {
		env, _ := newEnv(t)
		return env
	}})
}

func TestModeOfAndHint(t *testing.T) {
	cases := []struct {
		key  string
		mode payments.Mode
		err  error
	}{
		{"rk_live_51AcmeMediaKey9fQ2", payments.Live, nil},
		{"rk_test_51AcmeMediaKey3mZ8", payments.Test, nil},
		{"sk_live_51AcmeMediaSecret", "", payments.ErrSecretKey},
		{"sk_test_51AcmeMediaSecret", "", payments.ErrSecretKey},
		{"pk_live_51Publishable", "", payments.ErrKeyFormat},
		{"rk_live_", "", payments.ErrKeyFormat},
	}
	for _, c := range cases {
		m, err := stripe.ModeOf(c.key)
		if m != c.mode || !errors.Is(err, c.err) {
			t.Errorf("%s: %q %v", c.key, m, err)
		}
	}
	if h := stripe.Hint("rk_live_51AcmeMediaKey9fQ2"); h != "rk_live_…9fQ2" {
		t.Fatalf("hint %q", h)
	}
}

func TestEventsParse(t *testing.T) {
	f := stripetest.New()
	sub := map[string]any{"id": "sub_1Q8xRb2Va7", "object": "subscription", "customer": "cus_QmT4wE8sLd", "status": "active", "start_date": 1791331920,
		"current_period_end": 1822867920, "cancel_at_period_end": true, "trial_end": nil,
		"items": map[string]any{"data": []map[string]any{{"price": map[string]any{"id": "price_1Q8mL5cD", "product": "prod_QmA8x2Lw", "unit_amount": 8000, "currency": "usd",
			"recurring": map[string]any{"interval": "year"}}}}},
		"discount": map[string]any{"coupon": map[string]any{"id": "FALL26", "name": "FALL26", "percent_off": 20, "duration": "once"}, "start": 1791331920, "end": 1822867920}}
	e := f.Emit(payments.Live, "customer.subscription.updated", sub, map[string]any{"cancel_at_period_end": false})
	secret := "whsec_test"
	now := time.Now()
	h := http.Header{}
	h.Set("Stripe-Signature", stripe.SignatureHeader(secret, now.Unix(), e.Body))
	ev, err := stripe.Verifier{}.ParseWebhook(h, e.Body, secret, now)
	if err != nil {
		t.Fatal(err)
	}
	s := ev.Subscription
	if ev.Kind != payments.EventSubscriptionUpdated || s == nil || s.PriceID != "price_1Q8mL5cD" || s.Interval != "year" || s.AmountCents != 8000 {
		t.Fatalf("subscription: %+v %+v", ev, s)
	}
	if s.Discount == nil || s.Discount.PercentOff != 20 || s.CancelAt == nil || !s.CancelAt.Equal(time.Unix(1822867920, 0)) {
		t.Fatalf("discount and scheduled cancel: %+v %v", s.Discount, s.CancelAt)
	}
	if ev.Previous["cancel_at_period_end"] != false {
		t.Fatalf("previous attributes: %v", ev.Previous)
	}
	ref := f.Emit(payments.Live, "charge.refunded", map[string]any{"id": "ch_1", "object": "charge", "customer": "cus_OkW7pL3dNa", "amount": 8000,
		"amount_refunded": 8000, "currency": "usd"}, map[string]any{"amount_refunded": 0})
	h.Set("Stripe-Signature", stripe.SignatureHeader(secret, now.Unix(), ref.Body))
	ev, err = stripe.Verifier{}.ParseWebhook(h, ref.Body, secret, now)
	if err != nil || ev.Charge == nil || ev.Charge.RefundAmount != 8000 {
		t.Fatalf("refund: %+v %v", ev.Charge, err)
	}
}

func TestRetriesOnOutage(t *testing.T) {
	f := stripetest.New()
	f.AddKey(liveKey, stripetest.KeySpec{})
	srv := f.Server(t)
	c, _ := stripe.New(payments.Config{Key: liveKey, BaseURL: srv.URL})
	// Two failures, then an answer: the call succeeds on its third try.
	f.FailNext(http.MethodPost, "/v1/products", 503, 2)
	p, err := c.SaveProduct(context.Background(), payments.Product{Name: "Weekly Digest Insider", Active: true})
	if err != nil || p.ID == "" {
		t.Fatalf("retry: %v", err)
	}
	n := 0
	for _, r := range f.Requests() {
		if r.Path == "/v1/products" {
			n++
		}
	}
	if n != 3 {
		t.Fatalf("tries: %d", n)
	}
	// The key never travels in the clear outside the Authorization header.
	for _, r := range f.Requests() {
		if strings.Contains(r.Form.Encode(), "rk_live_") {
			t.Fatal("the key leaked into a request body")
		}
	}
}
