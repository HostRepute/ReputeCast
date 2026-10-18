package shopify_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/stores"
	"github.com/hostrepute/reputecast/internal/stores/contracttest"
	"github.com/hostrepute/reputecast/internal/stores/shopify"
	"github.com/hostrepute/reputecast/internal/stores/shopify/shopifytest"
)

const token = "shpat_contract"

var since = time.Date(2025, 10, 15, 0, 0, 0, 0, time.UTC)

// seed fills a fake with the contract's sample.
func seed(f *shopifytest.Fake) {
	at := func(d int) time.Time { return since.AddDate(0, 0, d) }
	f.Collections = []*shopifytest.Collection{{ID: "501", Title: "Apparel", Handle: "apparel"}, {ID: "502", Title: "New arrivals", Handle: "new-arrivals"}}
	f.Products = []*shopifytest.Product{
		{ID: "8841203", Title: "Acme Logo Hoodie", Handle: "acme-logo-hoodie", Status: "ACTIVE", Price: "58.0", Tracked: true, Inventory: 21, Collections: []string{"501", "502"}, CreatedAt: at(300), UpdatedAt: at(360)},
		{ID: "8841207", Title: "Ship It mug", Handle: "ship-it-mug", Status: "ACTIVE", Price: "18.0", Collections: []string{"502"}, CreatedAt: at(301), UpdatedAt: at(360)},
		{ID: "8841215", Title: "Feature Flag sticker pack", Handle: "feature-flag-stickers", Status: "ACTIVE", Price: "9.0", Collections: []string{"502"}, CreatedAt: at(302), UpdatedAt: at(360)},
		{ID: "8841190", Title: "Boring Infrastructure tee", Handle: "boring-infrastructure-tee", Status: "ACTIVE", Price: "28.0", CompareAt: "32.0", Collections: []string{"501"}, CreatedAt: at(200), UpdatedAt: at(360)},
		{ID: "8841177", Title: contracttest.SoldOut, Handle: "deploy-friday-cap", Status: "ACTIVE", Price: "24.0", Tracked: true, Inventory: 0, Collections: []string{"501"}, CreatedAt: at(100), UpdatedAt: at(360)},
	}
	consent := at(140)
	f.Customers = []*shopifytest.Customer{
		{ID: contracttest.CustomerID, Email: contracttest.Subscribed, FirstName: "Grace", LastName: "Liu", Country: "US", State: "SUBSCRIBED", Level: "SINGLE_OPT_IN", ConsentAt: &consent, CreatedAt: at(140), UpdatedAt: at(364)},
		{ID: "7002", Email: contracttest.NotSubscribed, FirstName: "Theo", LastName: "Martin", Country: "FR", State: "NOT_SUBSCRIBED", Level: "UNKNOWN", CreatedAt: at(365), UpdatedAt: at(365)},
	}
	item := func(id, title, price string) shopifytest.Item {
		return shopifytest.Item{ProductID: id, Title: title, Quantity: 1, Price: price}
	}
	f.Orders = []*shopifytest.Order{
		{ID: "5800000000001", Number: 1201, Email: contracttest.Subscribed, CustomerID: contracttest.CustomerID, CreatedAt: since.AddDate(0, -1, 0), UpdatedAt: since.AddDate(0, -1, 0),
			Subtotal: "28.0", Total: "28.0", Items: []shopifytest.Item{item("8841190", "Boring Infrastructure tee", "28.0")}},
		{ID: "5800000000002", Number: 4409, Email: "sam.rivera@outlook.com", CreatedAt: at(363), UpdatedAt: at(363), Subtotal: "58.0", Total: "58.0",
			Landing: "/products/acme-logo-hoodie?utm_source=acme-newsletter&utm_medium=email&utm_campaign=october-product-update",
			Items: []shopifytest.Item{item("8841203", "Acme Logo Hoodie", "58.0")}},
		{ID: "5800000000003", Number: 4417, Email: contracttest.Subscribed, CustomerID: contracttest.CustomerID, AcceptsMarketing: true, CreatedAt: at(365), UpdatedAt: at(365),
			Subtotal: "76.0", Total: "76.0", Landing: "/collections/new-arrivals?utm_source=acme-newsletter&rc_click=" + contracttest.ClickToken,
			Items: []shopifytest.Item{item("8841203", "Acme Logo Hoodie", "58.0"), item("8841207", "Ship It mug", "18.0")}},
		{ID: "5800000000004", Number: 4421, Email: contracttest.NotSubscribed, CustomerID: "7002", CreatedAt: at(365), UpdatedAt: at(365), Subtotal: "64.0", Total: "64.0",
			Items: []shopifytest.Item{item("8841203", "Acme Logo Hoodie", "58.0"), item("8841215", "Feature Flag sticker pack", "6.0")}},
	}
	f.Checkouts = []*shopifytest.Checkout{
		{ID: "36728419123401", Token: "c4f2a9e7d1b34e6f8a0b2c3d4e5f6a7b", Email: contracttest.Subscribed, CustomerID: contracttest.CustomerID, AcceptsMarketing: true,
			Subtotal: "76.0", Total: "76.0", CreatedAt: at(364), UpdatedAt: at(364),
			Items: []shopifytest.Item{item("8841203", "Acme Logo Hoodie", "58.0"), item("8841207", "Ship It mug", "18.0")}},
	}
}

func newEnv(t *testing.T) *contracttest.Env {
	f := shopifytest.New()
	f.PageCap = 2
	seed(f)
	f.IssueToken(token)
	srv := f.Server()
	t.Cleanup(srv.Close)
	recorded := map[string][2]string{
		stores.EventOrder:          {"orders/create", "orders_create.json"},
		stores.EventProduct:        {"products/update", "products_update.json"},
		stores.EventProductDeleted: {"products/delete", "products_delete.json"},
		stores.EventCustomer:       {"customers/update", "customers_update.json"},
		stores.EventCheckout:       {"checkouts/update", "checkouts_update.json"},
	}
	return &contracttest.Env{
		Open: func() (stores.Client, error) {
			return shopify.New(stores.Config{Domain: f.Domain, BaseURL: srv.URL, Token: token, HTTP: srv.Client(), RetryWait: time.Millisecond})
		},
		Revoke:   f.Revoke,
		Outage:   f.SetOutage,
		Throttle: f.Throttle,
		Secret:   func() string { return f.ClientSecret },
		Deliver: func(kind string) (http.Header, []byte) {
			r := recorded[kind]
			body, err := os.ReadFile("testdata/" + r[1])
			if err != nil {
				t.Fatal(err)
			}
			d := f.DeliverRaw(r[0], body, "")
			return d.Header, d.Body
		},
		Since: since,
	}
}

func TestContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{Adapter: shopify.Key, Env: newEnv})
}

// The payloads Shopify sends carry addresses, phone numbers, payment
// details and notes: none of it survives parsing.
func TestParsingKeepsNothingPrivate(t *testing.T) {
	files := map[string]string{"orders/create": "orders_create.json", "customers/update": "customers_update.json",
		"checkouts/update": "checkouts_update.json", "refunds/create": "refunds_create.json", "products/update": "products_update.json"}
	for topic, file := range files {
		body, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		h := http.Header{}
		h.Set("X-Shopify-Topic", topic)
		h.Set("X-Shopify-Event-Id", "e1")
		h.Set(stores.CurrencyHeader, "USD")
		ev, err := shopify.Webhooks{}.Parse(h, body)
		if err != nil {
			t.Fatalf("%s: %v", topic, err)
		}
		out, _ := json.Marshal(ev)
		for _, private := range []string{"Orchard", "Rivoli", "+1646", "+3361", "4242", "Please ship", "VIP", "10002", "75004"} {
			if strings.Contains(string(out), private) {
				t.Errorf("%s kept %q: %s", topic, private, out)
			}
		}
	}
}

func TestRecordedPayloads(t *testing.T) {
	read := func(topic, file string) stores.Event {
		t.Helper()
		body, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		h := http.Header{}
		h.Set("X-Shopify-Topic", topic)
		h.Set("X-Shopify-Webhook-Id", "wh-1")
		h.Set("X-Shopify-Triggered-At", "2026-10-15T12:40:15.120Z")
		h.Set(stores.CurrencyHeader, "USD")
		ev, err := shopify.Webhooks{}.Parse(h, body)
		if err != nil {
			t.Fatal(err)
		}
		return ev
	}
	o := read("orders/create", "orders_create.json").Order
	if o.Name != "#4417" || o.Number != "4417" || o.Total != 7600 || o.Currency != "USD" || len(o.Items) != 2 || o.Items[1].Variant != "" ||
		o.Items[0].Variant != "M" || o.CheckoutToken != "a8d1b6c2e4f04c7e9b3a5d6f7e8c9b0a" || !o.AcceptsMarketing ||
		o.UTM.Source != "acme-newsletter" || o.UTM.Campaign != "abandoned-cart" || o.ClickToken != "rcTOKEN123" {
		t.Fatalf("order: %+v", o)
	}
	if c := o.Customer; c == nil || c.ID != "7001" || !c.Consent.Accepted() || c.Consent.Level != "single_opt_in" || c.Country != "US" || c.Consent.UpdatedAt == nil {
		t.Fatalf("buyer: %+v", o.Customer)
	}
	c := read("customers/update", "customers_update.json").Customer
	if c.Email != "theo.martin@example.com" || c.Consent.Accepted() || c.Consent.State != stores.ConsentNotSubscribed {
		t.Fatalf("customer: %+v", c)
	}
	ch := read("checkouts/update", "checkouts_update.json").Checkout
	if ch.ID != "c4f2a9e7d1b34e6f8a0b2c3d4e5f6a7b" || ch.Total != 7600 || !ch.AcceptsMarketing || !strings.Contains(ch.RecoveryURL, "/recover") || ch.CompletedAt != nil {
		t.Fatalf("checkout: %+v", ch)
	}
	r := read("refunds/create", "refunds_create.json").Refund
	if r.OrderID != "5821403913242" || r.Amount != 1800 {
		t.Fatalf("refund: %+v", r)
	}
	p := read("products/update", "products_update.json").Product
	if p.ID != "8841203" || p.Price != 5800 || !p.Available || p.Inventory == nil || *p.Inventory != 21 || p.ImageURL == "" {
		t.Fatalf("product: %+v", p)
	}
	ev := read("app/uninstalled", "app_uninstalled.json")
	if ev.Kind != stores.EventUninstalled {
		t.Fatalf("app/uninstalled: %+v", ev)
	}
	if ev.TriggeredAt.IsZero() || ev.DeliveryID != "wh-1" {
		t.Fatalf("delivery: %+v", ev)
	}
}

func TestInstall(t *testing.T) {
	ctx := context.Background()
	f := shopifytest.New()
	srv := f.Server()
	defer srv.Close()
	cfg := stores.Config{Domain: f.Domain, BaseURL: srv.URL, HTTP: srv.Client()}

	// The store's public address names its myshopify domain.
	sf, err := shopify.Resolve(ctx, cfg, "https://Shop.Acme.io/")
	if err != nil || sf.Domain != "acme-merch.myshopify.com" || sf.Name != "Acme Merch" || sf.Currency != "USD" {
		t.Fatalf("resolve: %+v %v", sf, err)
	}
	// Something else isn't a store.
	other := stores.Config{BaseURL: srv.URL + "/not-shopify", HTTP: srv.Client()}
	if _, err := shopify.Resolve(ctx, other, "acme-merch.com"); !errors.Is(err, stores.ErrNotAStore) {
		t.Fatalf("not a store: %v", err)
	}

	auth := shopify.AuthorizeURL(f.Domain, f.ClientID, "https://reputecast.example/integrations/shopify/callback", "st4te", append(shopify.Scopes, shopify.OptionalScope))
	u, _ := url.Parse(auth)
	if u.Host != "acme-merch.myshopify.com" || u.Path != "/admin/oauth/authorize" || u.Query().Get("scope") != "read_customers,read_orders,read_products,write_customers" {
		t.Fatalf("authorize: %s", auth)
	}

	back := f.Approve("https://reputecast.example/integrations/shopify/callback", "st4te")
	bu, _ := url.Parse(back)
	q := bu.Query()
	if !shopify.VerifyQuery(q, f.ClientSecret, time.Now()) {
		t.Fatal("the callback's hmac doesn't verify")
	}
	if shopify.VerifyQuery(q, "shpss_wrong", time.Now()) {
		t.Fatal("a wrong secret verified")
	}
	if shopify.VerifyQuery(q, f.ClientSecret, time.Now().Add(48*time.Hour)) {
		t.Fatal("an old callback verified")
	}
	tampered := url.Values{}
	for k, v := range q {
		tampered[k] = v
	}
	tampered.Set("shop", "evil.myshopify.com")
	if shopify.VerifyQuery(tampered, f.ClientSecret, time.Now()) {
		t.Fatal("a tampered callback verified")
	}

	// A wrong secret is refused at the exchange: the dialog's step 3 error.
	if _, err := shopify.Exchange(ctx, cfg, f.ClientID, "shpss_wrong", q.Get("code")); !errors.Is(err, stores.ErrUnauthorized) {
		t.Fatalf("wrong secret: %v", err)
	}
	tok, err := shopify.Exchange(ctx, cfg, f.ClientID, f.ClientSecret, q.Get("code"))
	if err != nil || !strings.HasPrefix(tok.AccessToken, "shpat_") || len(tok.Scopes) != 4 {
		t.Fatalf("exchange: %+v %v", tok, err)
	}
	// A code works once.
	if _, err := shopify.Exchange(ctx, cfg, f.ClientID, f.ClientSecret, q.Get("code")); !errors.Is(err, stores.ErrUnauthorized) {
		t.Fatalf("reused code: %v", err)
	}
	c, _ := shopify.New(stores.Config{Domain: f.Domain, BaseURL: srv.URL, HTTP: srv.Client(), Token: tok.AccessToken})
	if shop, err := c.Shop(ctx); err != nil || shop.Name != "Acme Merch" || shop.PublicDomain != "shop.acme.io" {
		t.Fatalf("shop with the new token: %+v %v", shop, err)
	}
	if !shopify.ValidShopDomain("acme-merch.myshopify.com") || shopify.ValidShopDomain("acme-merch.com") || shopify.ValidShopDomain("a_b.myshopify.com") {
		t.Fatal("ValidShopDomain")
	}
}

func TestTopicNames(t *testing.T) {
	if shopify.TopicEnum("orders/create") != "ORDERS_CREATE" || shopify.TopicFromEnum("CHECKOUTS_UPDATE") != "checkouts/update" ||
		shopify.TopicFromEnum("APP_UNINSTALLED") != "app/uninstalled" {
		t.Fatal("topic names")
	}
}
