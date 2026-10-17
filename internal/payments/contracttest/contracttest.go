// Package contracttest is the shared suite every payment adapter passes,
// like the sending adapters' (internal/providers/contracttest): the
// catalog entry, the key rules (restricted keys only, secret keys refused
// before any call), the permission check, the webhook's registration and
// signature (bad signatures and replays refused), checkout, products and
// prices, and an outage — against the adapter's fake of the provider's
// API, with no network.
//
//	func TestContract(t *testing.T) {
//		contracttest.Run(t, contracttest.Suite{Adapter: "stripe", Env: newEnv})
//	}
package contracttest

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/payments"
)

// Env is the adapter's fake provider for one test.
type Env struct {
	// Open connects to the fake with key.
	Open func(key string) (payments.Client, error)
	// Keys the fake knows: a live and a test restricted key, a restricted
	// key that lacks MissingPermission, and one the provider rejects.
	LiveKey, TestKey, MissingKey, RejectedKey string
	MissingPermission                         string
	// SecretKey is an unrestricted secret key of the provider's format.
	SecretKey string
	// Requests counts the calls the fake received.
	Requests func() int
	// Event is a recorded event (raw body) and Sign signs a body with a
	// webhook secret at ts, as the provider's request headers would.
	Event func(t *testing.T, c payments.Client) []byte
	Sign  func(secret string, body []byte, ts time.Time) http.Header
	// Outage makes the fake answer 5xx to every call.
	Outage func()
}

// Suite describes one adapter.
type Suite struct {
	Adapter string
	Env     func(t *testing.T) *Env
}

// Run runs the contract.
func Run(t *testing.T, s Suite) {
	t.Helper()
	d, ok := payments.Lookup(s.Adapter)
	if !ok {
		t.Fatalf("adapter %q isn't registered", s.Adapter)
	}
	ctx := context.Background()

	t.Run("catalog", func(t *testing.T) {
		if !d.Available || d.Open == nil || d.Name == "" {
			t.Fatalf("descriptor: %+v", d)
		}
		if len(d.Permissions) == 0 || len(d.WebhookEvents) == 0 || d.KeyPrefixes[payments.Live] == "" || d.KeyPrefixes[payments.Test] == "" {
			t.Fatalf("descriptor needs permissions, webhook events and key prefixes: %+v", d)
		}
		if !slices.Contains(d.Capabilities, payments.CapTestMode) || !slices.Contains(d.Capabilities, payments.CapSubscriptions) {
			t.Fatalf("capabilities: %v", d.Capabilities)
		}
		if _, ok := payments.ParserFor(s.Adapter); !ok {
			t.Fatal("no webhook verifier registered")
		}
	})

	t.Run("keys", func(t *testing.T) {
		env := s.Env(t)
		before := env.Requests()
		if _, err := env.Open(env.SecretKey); !errors.Is(err, payments.ErrSecretKey) {
			t.Fatalf("secret key: %v", err)
		}
		if _, err := env.Open("not-a-key"); !errors.Is(err, payments.ErrKeyFormat) {
			t.Fatalf("garbage key: %v", err)
		}
		if env.Requests() != before {
			t.Fatal("a refused key must never reach the provider")
		}
		live, err := env.Open(env.LiveKey)
		if err != nil || live.Mode() != payments.Live {
			t.Fatalf("live key: %v", err)
		}
		test, err := env.Open(env.TestKey)
		if err != nil || test.Mode() != payments.Test {
			t.Fatalf("test key: %v", err)
		}
	})

	t.Run("check key", func(t *testing.T) {
		env := s.Env(t)
		c, _ := env.Open(env.LiveKey)
		chk, err := c.CheckKey(ctx)
		if err != nil {
			t.Fatalf("check: %v", err)
		}
		if len(chk.Permissions) != len(d.Permissions) || len(chk.Missing()) != 0 {
			t.Fatalf("every permission should pass: %+v", chk.Permissions)
		}
		if chk.Account.ID == "" || chk.Account.Name == "" || chk.Account.Currency == "" {
			t.Fatalf("account: %+v", chk.Account)
		}
		m, _ := env.Open(env.MissingKey)
		chk, err = m.CheckKey(ctx)
		if err != nil {
			t.Fatalf("check missing: %v", err)
		}
		miss := chk.Missing()
		if len(miss) != 1 || miss[0].Resource != env.MissingPermission {
			t.Fatalf("missing permission: %+v", miss)
		}
		r, _ := env.Open(env.RejectedKey)
		if _, err := r.CheckKey(ctx); !errors.Is(err, payments.ErrKeyRejected) {
			t.Fatalf("rejected key: %v", err)
		}
	})

	t.Run("webhook", func(t *testing.T) {
		env := s.Env(t)
		c, _ := env.Open(env.LiveKey)
		w, err := c.RegisterWebhook(ctx, "https://reputecast.example/hooks/x/token", d.WebhookEvents)
		if err != nil || w.ID == "" || w.Secret == "" {
			t.Fatalf("register: %+v %v", w, err)
		}
		if ok, err := c.WebhookExists(ctx, w.ID); err != nil || !ok {
			t.Fatalf("exists: %v %v", ok, err)
		}
		if err := c.DeleteWebhook(ctx, w.ID); err != nil {
			t.Fatalf("delete: %v", err)
		}
		if ok, err := c.WebhookExists(ctx, w.ID); err != nil || ok {
			t.Fatalf("after delete: %v %v", ok, err)
		}
	})

	t.Run("signature", func(t *testing.T) {
		env := s.Env(t)
		c, _ := env.Open(env.LiveKey)
		v, _ := payments.ParserFor(s.Adapter)
		body := env.Event(t, c)
		now := time.Now()
		ev, err := v.ParseWebhook(env.Sign("whsec_contract", body, now), body, "whsec_contract", now)
		if err != nil || ev.ID == "" || ev.Kind == "" {
			t.Fatalf("valid signature: %+v %v", ev, err)
		}
		tampered := append([]byte{}, body...)
		tampered[len(tampered)-2] ^= 1
		if _, err := v.ParseWebhook(env.Sign("whsec_contract", body, now), tampered, "whsec_contract", now); !errors.Is(err, payments.ErrBadSignature) {
			t.Fatalf("tampered body: %v", err)
		}
		if _, err := v.ParseWebhook(env.Sign("whsec_other", body, now), body, "whsec_contract", now); !errors.Is(err, payments.ErrBadSignature) {
			t.Fatalf("other secret: %v", err)
		}
		old := now.Add(-time.Hour)
		if _, err := v.ParseWebhook(env.Sign("whsec_contract", body, old), body, "whsec_contract", now); !errors.Is(err, payments.ErrBadSignature) {
			t.Fatalf("a replay signed an hour ago must fail: %v", err)
		}
		if _, err := v.ParseWebhook(http.Header{}, body, "whsec_contract", now); !errors.Is(err, payments.ErrBadSignature) {
			t.Fatalf("no signature: %v", err)
		}
	})

	t.Run("products and checkout", func(t *testing.T) {
		env := s.Env(t)
		c, _ := env.Open(env.TestKey)
		p, err := c.SaveProduct(ctx, payments.Product{Name: "Contract tier", Active: true})
		if err != nil || p.ID == "" {
			t.Fatalf("product: %v", err)
		}
		if p, err = c.SaveProduct(ctx, payments.Product{ID: p.ID, Name: "Contract tier, renamed", Active: true}); err != nil || p.Name != "Contract tier, renamed" {
			t.Fatalf("update product: %+v %v", p, err)
		}
		price, err := c.CreatePrice(ctx, payments.Price{ProductID: p.ID, AmountCents: 800, Currency: "usd", Interval: "month"})
		if err != nil || price.ID == "" || price.AmountCents != 800 || price.Interval != "month" {
			t.Fatalf("price: %+v %v", price, err)
		}
		cs, err := c.CreateCheckout(ctx, payments.CheckoutRequest{PriceID: price.ID, CustomerEmail: "reader@contract.test", TrialDays: 14,
			SuccessURL: "https://pub.example/upgrade/welcome?session={CHECKOUT_SESSION_ID}", CancelURL: "https://pub.example/upgrade?canceled=1", AllowCodes: true})
		if err != nil || cs.ID == "" || cs.URL == "" {
			t.Fatalf("checkout: %+v %v", cs, err)
		}
		got, err := c.GetCheckout(ctx, cs.ID)
		if err != nil || got.ID != cs.ID || got.Status != "open" {
			t.Fatalf("get checkout: %+v %v", got, err)
		}
		if err := c.RetirePrice(ctx, price.ID); err != nil {
			t.Fatalf("retire: %v", err)
		}
		if got, err := c.GetPrice(ctx, price.ID); err != nil || got.Active {
			t.Fatalf("a retired price stays inactive: %+v %v", got, err)
		}
	})

	t.Run("outage", func(t *testing.T) {
		env := s.Env(t)
		c, _ := env.Open(env.LiveKey)
		env.Outage()
		if _, err := c.SaveProduct(ctx, payments.Product{Name: "Down", Active: true}); !errors.Is(err, payments.ErrUnavailable) {
			t.Fatalf("outage: %v", err)
		}
	})
}
