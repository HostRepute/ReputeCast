// Package contracttest is the shared suite every store adapter passes, like
// the sending and payment adapters' (internal/providers/contracttest,
// internal/payments/contracttest): the registry entry, reading the shop,
// products with paging and availability, collections, customers with their
// marketing consent, orders in a range with the landing page's click
// token, abandoned checkouts, webhooks (subscribe, list, remove), consent
// write-back, webhook signatures (a bad or tampered one refused) and the
// recorded payloads' parsing, refused credentials, throttling and an
// outage — against the adapter's fake of the platform, with no network.
//
//	func TestContract(t *testing.T) {
//		contracttest.Run(t, contracttest.Suite{Adapter: "shopify", Env: newEnv})
//	}
package contracttest

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/stores"
)

// Env is the adapter's fake store for one test, seeded with the suite's
// sample (Seed).
type Env struct {
	// Open connects to the fake with valid credentials.
	Open func() (stores.Client, error)
	// Revoke makes the fake refuse the credentials.
	Revoke func()
	// Outage makes the fake answer 5xx; Throttle throttles n calls.
	Outage   func(on bool)
	Throttle func(n int)
	// Secret signs the fake's deliveries; Deliver signs a recorded payload
	// of a kind (EventOrder, EventProduct, EventProductDeleted,
	// EventCustomer, EventCheckout) as the platform would send it.
	Secret  func() string
	Deliver func(kind string) (http.Header, []byte)
	// Since is the creation date the sample's recent orders start at
	// (older ones are before it).
	Since time.Time
}

// The sample every fake is seeded with (Seed in each adapter's test):
const (
	// SampleProducts products, SoldOut of them out of stock.
	SampleProducts = 5
	SoldOut        = "Deploy Friday cap"
	// SampleCollections collections.
	SampleCollections = 2
	// SampleOrders orders, RecentOrders of them created at or after Since,
	// one of which arrived with ClickToken in its landing page.
	SampleOrders = 4
	RecentOrders = 3
	ClickToken   = "rcTOKEN123"
	// Subscribed is a customer who accepted marketing; NotSubscribed one
	// who didn't.
	Subscribed    = "grace.liu@example.com"
	NotSubscribed = "theo.martin@example.com"
)

// Suite describes one adapter.
type Suite struct {
	Adapter string
	Env     func(t *testing.T) *Env
}

// Run runs the contract.
func Run(t *testing.T, s Suite) {
	t.Helper()
	d, ok := stores.Lookup(s.Adapter)
	if !ok {
		t.Fatalf("adapter %q isn't registered", s.Adapter)
	}
	ctx := context.Background()

	t.Run("registry", func(t *testing.T) {
		if d.Name == "" || d.Open == nil || d.Webhooks == nil {
			t.Fatalf("incomplete descriptor: %+v", d)
		}
		for _, c := range []stores.Capability{stores.CapOrders, stores.CapProducts, stores.CapAttribution} {
			if !d.Has(c) {
				t.Errorf("every store has %s", c)
			}
		}
		if len(d.Topics) == 0 {
			t.Error("no webhook topics")
		}
	})

	t.Run("shop", func(t *testing.T) {
		env := s.Env(t)
		c := open(t, env)
		shop, err := c.Shop(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if shop.Name == "" || shop.Domain == "" || shop.Currency == "" {
			t.Fatalf("shop: %+v", shop)
		}
	})

	t.Run("products", func(t *testing.T) {
		env := s.Env(t)
		c := open(t, env)
		all := pages(t, func(cursor string) (stores.Page[stores.Product], error) {
			return c.Products(ctx, cursor, stores.Range{})
		})
		if len(all) != SampleProducts {
			t.Fatalf("products across pages: %d, want %d", len(all), SampleProducts)
		}
		seen := map[string]bool{}
		for _, p := range all {
			if seen[p.ID] {
				t.Errorf("product %s listed twice", p.ID)
			}
			seen[p.ID] = true
			if p.Title == "" || p.Price <= 0 || p.Currency == "" {
				t.Errorf("product without title, price or currency: %+v", p)
			}
			if p.Title == SoldOut && p.Available {
				t.Errorf("%s is sold out but reads as available", SoldOut)
			}
			if p.Title != SoldOut && !p.Available {
				t.Errorf("%s reads as sold out", p.Title)
			}
		}
		n, err := c.Count(ctx, stores.CountProducts, stores.Range{})
		if err != nil || n != SampleProducts {
			t.Errorf("count products: %d %v", n, err)
		}
	})

	t.Run("collections", func(t *testing.T) {
		env := s.Env(t)
		c := open(t, env)
		all := pages(t, func(cursor string) (stores.Page[stores.Collection], error) { return c.Collections(ctx, cursor) })
		if len(all) != SampleCollections {
			t.Fatalf("collections: %d, want %d", len(all), SampleCollections)
		}
	})

	t.Run("orders", func(t *testing.T) {
		env := s.Env(t)
		c := open(t, env)
		all := pages(t, func(cursor string) (stores.Page[stores.Order], error) { return c.Orders(ctx, cursor, stores.Range{}) })
		if len(all) != SampleOrders {
			t.Fatalf("orders: %d, want %d", len(all), SampleOrders)
		}
		recent := pages(t, func(cursor string) (stores.Page[stores.Order], error) {
			return c.Orders(ctx, cursor, stores.Range{CreatedSince: env.Since})
		})
		if len(recent) != RecentOrders {
			t.Fatalf("orders since %s: %d, want %d", env.Since, len(recent), RecentOrders)
		}
		tokens := 0
		for _, o := range recent {
			if o.Total <= 0 || len(o.Items) == 0 || o.Name == "" {
				t.Errorf("order without total, items or name: %+v", o)
			}
			if o.ClickToken == ClickToken {
				tokens++
			}
			if o.Customer == nil || o.Customer.Email == "" {
				t.Errorf("order %s without its buyer", o.Name)
			}
		}
		if tokens != 1 {
			t.Errorf("orders with the click token: %d, want 1", tokens)
		}
	})

	t.Run("customers", func(t *testing.T) {
		if !d.Has(stores.CapCustomers) {
			t.Skip("no customers")
		}
		env := s.Env(t)
		c := open(t, env)
		_, err := c.Customers(ctx, "", stores.Range{})
		if errors.Is(err, stores.ErrUnsupported) {
			// Consent comes with each order (the WooCommerce plugin).
			orders := pages(t, func(cursor string) (stores.Page[stores.Order], error) { return c.Orders(ctx, cursor, stores.Range{}) })
			// The latest order says what the customer chose last.
			checkConsent(t, func(email string) *stores.Consent {
				var last *stores.Order
				for i, o := range orders {
					if o.Customer != nil && o.Customer.Email == email && (last == nil || o.PlacedAt.After(last.PlacedAt)) {
						last = &orders[i]
					}
				}
				if last == nil {
					return nil
				}
				return &last.Customer.Consent
			})
			return
		}
		all := pages(t, func(cursor string) (stores.Page[stores.Customer], error) {
			return c.Customers(ctx, cursor, stores.Range{})
		})
		checkConsent(t, func(email string) *stores.Consent {
			for _, cu := range all {
				if cu.Email == email {
					return &cu.Consent
				}
			}
			return nil
		})
	})

	t.Run("checkouts", func(t *testing.T) {
		env := s.Env(t)
		c := open(t, env)
		page, err := c.Checkouts(ctx, "", stores.Range{})
		if errors.Is(err, stores.ErrUnsupported) {
			return // the platform reports them another way (the plugin)
		}
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Items) == 0 {
			t.Fatal("no abandoned checkout")
		}
		for _, ch := range page.Items {
			if ch.ID == "" || ch.RecoveryURL == "" || len(ch.Items) == 0 || ch.Total <= 0 {
				t.Errorf("incomplete checkout: %+v", ch)
			}
		}
	})

	t.Run("webhooks", func(t *testing.T) {
		env := s.Env(t)
		c := open(t, env)
		for _, topic := range d.Topics {
			if _, err := c.Subscribe(ctx, topic, "https://reputecast.example/hooks/x/token", "whsec-contract"); err != nil {
				t.Fatalf("subscribe %s: %v", topic, err)
			}
		}
		hooks, err := c.Webhooks(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for _, topic := range d.Topics {
			if !slices.ContainsFunc(hooks, func(w stores.Webhook) bool { return w.Topic == topic }) {
				t.Errorf("%s isn't listed after subscribing", topic)
			}
		}
		if err := c.Unsubscribe(ctx, hooks[0].ID); err != nil {
			t.Fatal(err)
		}
		after, _ := c.Webhooks(ctx)
		if len(after) != len(hooks)-1 {
			t.Errorf("after removing one: %d webhooks, want %d", len(after), len(hooks)-1)
		}
		if err := c.Unsubscribe(ctx, hooks[0].ID); err != nil {
			t.Errorf("removing a webhook that's gone: %v", err)
		}
	})

	t.Run("consent write-back", func(t *testing.T) {
		env := s.Env(t)
		c := open(t, env)
		err := c.SetConsent(ctx, CustomerID, stores.ConsentUnsubscribed, time.Now())
		if d.Has(stores.CapWriteBack) && err != nil {
			t.Fatalf("write-back: %v", err)
		}
		if !d.Has(stores.CapWriteBack) && !errors.Is(err, stores.ErrUnsupported) {
			t.Fatalf("without write-back: %v, want ErrUnsupported", err)
		}
	})

	t.Run("signatures", func(t *testing.T) {
		env := s.Env(t)
		h, body := env.Deliver(stores.EventOrder)
		if err := d.Webhooks.Verify(h, body, env.Secret()); err != nil {
			t.Fatalf("a good signature: %v", err)
		}
		tampered := append(slices.Clone(body[:len(body)-1]), ' ', body[len(body)-1])
		if err := d.Webhooks.Verify(h, tampered, env.Secret()); !errors.Is(err, stores.ErrBadSignature) {
			t.Errorf("a tampered body: %v", err)
		}
		if err := d.Webhooks.Verify(h, body, "another-secret"); !errors.Is(err, stores.ErrBadSignature) {
			t.Errorf("another secret: %v", err)
		}
		bare := http.Header{}
		topic, _ := d.Webhooks.Topic(h)
		if topic == "" {
			t.Error("no topic in the delivery's headers")
		}
		if err := d.Webhooks.Verify(bare, body, env.Secret()); !errors.Is(err, stores.ErrBadSignature) {
			t.Errorf("no signature: %v", err)
		}
	})

	t.Run("payloads", func(t *testing.T) {
		env := s.Env(t)
		for _, kind := range []string{stores.EventOrder, stores.EventProduct, stores.EventProductDeleted} {
			h, body := env.Deliver(kind)
			ev, err := d.Webhooks.Parse(h, body)
			if err != nil {
				t.Fatalf("%s: %v", kind, err)
			}
			if ev.Kind != kind {
				t.Errorf("%s parsed as %s", kind, ev.Kind)
			}
			_, delivery := d.Webhooks.Topic(h)
			if ev.DeliveryID == "" || delivery != ev.DeliveryID {
				t.Errorf("%s without its delivery id", kind)
			}
			switch kind {
			case stores.EventOrder:
				o := ev.Order
				if o == nil || o.Total <= 0 || len(o.Items) == 0 || o.Email == "" {
					t.Fatalf("order: %+v", o)
				}
				if o.ClickToken != ClickToken {
					t.Errorf("click token %q, want %q", o.ClickToken, ClickToken)
				}
			case stores.EventProduct:
				if ev.Product == nil || ev.Product.ID == "" || ev.Product.Title == "" {
					t.Errorf("product: %+v", ev.Product)
				}
			case stores.EventProductDeleted:
				if ev.DeletedID == "" {
					t.Error("deleted product without its id")
				}
			}
		}
	})

	t.Run("refused credentials", func(t *testing.T) {
		env := s.Env(t)
		c := open(t, env)
		env.Revoke()
		if _, err := c.Shop(ctx); !errors.Is(err, stores.ErrUnauthorized) {
			t.Fatalf("revoked: %v, want ErrUnauthorized", err)
		}
	})

	t.Run("throttling", func(t *testing.T) {
		env := s.Env(t)
		c := open(t, env)
		env.Throttle(1)
		_, err := c.Products(ctx, "", stores.Range{})
		wait, ok := stores.IsRateLimited(err)
		if !ok || wait <= 0 {
			t.Fatalf("throttled: %v (wait %s)", err, wait)
		}
		if _, err := c.Products(ctx, "", stores.Range{}); err != nil {
			t.Fatalf("after the throttle: %v", err)
		}
	})

	t.Run("outage", func(t *testing.T) {
		env := s.Env(t)
		c := open(t, env)
		env.Outage(true)
		if _, err := c.Orders(ctx, "", stores.Range{}); !errors.Is(err, stores.ErrUnavailable) {
			t.Fatalf("outage: %v, want ErrUnavailable", err)
		}
	})
}

// CustomerID is the sample customer whose consent write-back is tried.
const CustomerID = "7001"

func open(t *testing.T, env *Env) stores.Client {
	t.Helper()
	c, err := env.Open()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func pages[T any](t *testing.T, list func(cursor string) (stores.Page[T], error)) []T {
	t.Helper()
	var out []T
	cursor := ""
	for i := 0; i < 50; i++ {
		p, err := list(cursor)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, p.Items...)
		if p.Next == "" {
			return out
		}
		cursor = p.Next
	}
	t.Fatal("more than 50 pages")
	return nil
}

func checkConsent(t *testing.T, of func(email string) *stores.Consent) {
	t.Helper()
	yes := of(Subscribed)
	if yes == nil || !yes.Accepted() {
		t.Errorf("%s accepted marketing: %+v", Subscribed, yes)
	}
	no := of(NotSubscribed)
	if no == nil || no.Accepted() {
		t.Errorf("%s didn't accept marketing: %+v", NotSubscribed, no)
	}
}
