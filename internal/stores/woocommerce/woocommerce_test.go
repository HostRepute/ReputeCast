package woocommerce_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/stores"
	"github.com/hostrepute/reputecast/internal/stores/contracttest"
	"github.com/hostrepute/reputecast/internal/stores/woocommerce"
	"github.com/hostrepute/reputecast/internal/stores/woocommerce/wootest"
)

var since = time.Date(2025, 10, 15, 0, 0, 0, 0, time.UTC)

const contractSecret = "whsec-contract-woo"

func seed(f *wootest.Fake) {
	at := func(d int) time.Time { return since.AddDate(0, 0, d) }
	stock := func(n int) *int { return &n }
	f.Plugin = true
	f.Categories = []*wootest.Category{{ID: 15, Name: "Apparel", Slug: "apparel"}, {ID: 16, Name: "New arrivals", Slug: "new-arrivals"}}
	f.Products = []*wootest.Product{
		{ID: 4120, Name: "Acme Logo Hoodie", Slug: "acme-logo-hoodie", Status: "publish", Price: "58", Regular: "58", InStock: true, Stock: stock(21), Categories: []int64{15, 16}, Created: at(300), Modified: at(360)},
		{ID: 4131, Name: "Ship It mug", Slug: "ship-it-mug", Status: "publish", Price: "18", Regular: "18", InStock: true, Categories: []int64{16}, Created: at(301), Modified: at(360)},
		{ID: 4140, Name: "Feature Flag sticker pack", Slug: "feature-flag-stickers", Status: "publish", Price: "9", Regular: "9", InStock: true, Categories: []int64{16}, Created: at(302), Modified: at(360)},
		{ID: 4101, Name: "Boring Infrastructure tee", Slug: "boring-infrastructure-tee", Status: "publish", Price: "28", Regular: "32", Sale: "28", InStock: true, Categories: []int64{15}, Created: at(200), Modified: at(360)},
		{ID: 4099, Name: contracttest.SoldOut, Slug: "deploy-friday-cap", Status: "publish", Price: "24", Regular: "24", InStock: false, Stock: stock(0), Categories: []int64{15}, Created: at(100), Modified: at(360)},
	}
	consentAt := at(365)
	f.Orders = []*wootest.Order{
		{ID: 1201, Number: "1201", Email: contracttest.Subscribed, FirstName: "Grace", LastName: "Liu", Country: "US", CustomerID: 311, Created: since.AddDate(0, -1, 0), Modified: since.AddDate(0, -1, 0),
			Total: "28.00", Items: []wootest.Item{{ProductID: 4101, Name: "Boring Infrastructure tee", Quantity: 1, Price: 28}}},
		{ID: 1840, Number: "1840", Email: "sam.rivera@outlook.com", FirstName: "Sam", LastName: "Rivera", Country: "US", Created: at(363), Modified: at(363),
			Total: "58.00", Items: []wootest.Item{{ProductID: 4120, Name: "Acme Logo Hoodie", Quantity: 1, Price: 58}}},
		{ID: 1842, Number: "1842", Email: contracttest.Subscribed, FirstName: "Grace", LastName: "Liu", Country: "US", CustomerID: 311, Created: at(365), Modified: at(365),
			Total: "76.00", Items: []wootest.Item{{ProductID: 4120, Name: "Acme Logo Hoodie", Quantity: 1, Price: 58}, {ProductID: 4131, Name: "Ship It mug", Quantity: 1, Price: 18}},
			Marketing: true, ConsentText: "Email me about new products and offers from the store. Unsubscribe any time.", ConsentAt: &consentAt, ClickToken: contracttest.ClickToken, Checkout: "wc-checkout-9f1e"},
		{ID: 1843, Number: "1843", Email: contracttest.NotSubscribed, FirstName: "Theo", LastName: "Martin", Country: "FR", Created: at(365), Modified: at(365),
			Total: "64.00", Items: []wootest.Item{{ProductID: 4120, Name: "Acme Logo Hoodie", Quantity: 1, Price: 58}, {ProductID: 4140, Name: "Feature Flag sticker pack", Quantity: 1, Price: 6}}},
	}
	f.Webhooks = []*wootest.Webhook{
		{ID: 1, Topic: "order.created", URL: "https://reputecast.example/hooks/woocommerce/x", Secret: contractSecret, Status: "active"},
		{ID: 2, Topic: "product.updated", URL: "https://reputecast.example/hooks/woocommerce/x", Secret: contractSecret, Status: "active"},
		{ID: 3, Topic: "product.deleted", URL: "https://reputecast.example/hooks/woocommerce/x", Secret: contractSecret, Status: "active"},
	}
}

func newEnv(t *testing.T) *contracttest.Env {
	f := wootest.New()
	f.PageCap = 2
	seed(f)
	srv := f.Server()
	t.Cleanup(srv.Close)
	recorded := map[string][2]string{
		stores.EventOrder:          {"order.created", "order.created.json"},
		stores.EventProduct:        {"product.updated", "product.updated.json"},
		stores.EventProductDeleted: {"product.deleted", "product.deleted.json"},
	}
	return &contracttest.Env{
		Open: func() (stores.Client, error) {
			return woocommerce.New(stores.Config{Domain: "store.example.com", BaseURL: srv.URL, Key: f.Key, Secret: f.Secret, HTTP: srv.Client(), RetryWait: time.Millisecond})
		},
		Revoke:   f.Revoke,
		Outage:   f.SetOutage,
		Throttle: f.Throttle,
		Secret:   func() string { return contractSecret },
		Deliver: func(kind string) (http.Header, []byte) {
			r := recorded[kind]
			body, err := os.ReadFile("testdata/" + r[1])
			if err != nil {
				t.Fatal(err)
			}
			var payload json.RawMessage = body
			d := f.Deliver(r[0], payload)
			return d.Header, d.Body
		},
		Since: since,
	}
}

func TestContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{Adapter: woocommerce.Key, Env: newEnv})
}

func TestRecordedOrder(t *testing.T) {
	body, err := os.ReadFile("testdata/order.created.json")
	if err != nil {
		t.Fatal(err)
	}
	h := http.Header{}
	h.Set("X-WC-Webhook-Topic", "order.created")
	h.Set("X-WC-Webhook-Delivery-ID", "d1")
	ev, err := woocommerce.Webhooks{}.Parse(h, body)
	if err != nil {
		t.Fatal(err)
	}
	o := ev.Order
	if o.Name != "#1842" || o.Total != 7100 || o.Subtotal != 7600 || len(o.Items) != 2 || o.Items[0].Price != 5300 || o.Discounts[0] != "fall5" ||
		o.ClickToken != "rcTOKEN123" || o.CheckoutToken != "wc-checkout-9f1e" || o.UTM.Campaign != "october-product-update" || !o.AcceptsMarketing {
		t.Fatalf("order: %+v", o)
	}
	c := o.Customer
	if c.ID != "311" || c.Email != "grace.liu@example.com" || !c.Consent.Accepted() || c.Consent.Text == "" || c.Consent.UpdatedAt == nil || c.Country != "US" {
		t.Fatalf("buyer: %+v", c)
	}
	out, _ := json.Marshal(ev)
	for _, private := range []string{"Orchard", "+1646", "gift wrap", "4242", "ch_3Q7", "10002"} {
		if strings.Contains(string(out), private) {
			t.Errorf("kept %q", private)
		}
	}
	// WooCommerce's delivery check when a webhook is created.
	ping, err := woocommerce.Webhooks{}.Parse(h, []byte("webhook_id=12"))
	if err != nil || ping.Kind != stores.EventPing {
		t.Fatalf("ping: %+v %v", ping, err)
	}
}

func TestProbeAndKeys(t *testing.T) {
	ctx := context.Background()
	f := wootest.New()
	srv := f.Server()
	defer srv.Close()
	cfg := stores.Config{Domain: "store.example.com", BaseURL: srv.URL, HTTP: srv.Client()}
	p, err := woocommerce.ProbeSite(ctx, cfg)
	if err != nil || !p.WooCommerce || p.Plugin {
		t.Fatalf("probe without the plugin: %+v %v", p, err)
	}
	f.Plugin = true
	if p, _ := woocommerce.ProbeSite(ctx, cfg); !p.Plugin {
		t.Fatal("the plugin isn't seen")
	}
	if !woocommerce.KeyFormat("ck_0123456789abcdef", "cs_0123456789abcdef") || woocommerce.KeyFormat("cs_x", "ck_y") {
		t.Fatal("key format")
	}
	f.ReadOnly = true
	c, _ := woocommerce.New(stores.Config{Domain: "store.example.com", BaseURL: srv.URL, HTTP: srv.Client(), Key: f.Key, Secret: f.Secret})
	if _, err := c.Subscribe(ctx, "order.created", "https://reputecast.example/hooks/woocommerce/t", "s"); !errors.Is(err, stores.ErrUnauthorized) && !errors.Is(err, stores.ErrPermission) {
		t.Fatalf("a read-only key creating a webhook: %v", err)
	}
	if _, err := c.Shop(ctx); err != nil {
		t.Fatalf("a read-only key reads: %v", err)
	}
}
