// Package wootest is an in-memory WooCommerce site: the REST index (with
// the ReputeCast plugin's namespace when it's active), the wc/v3 endpoints
// the adapter calls (currency, products, categories, orders, webhooks)
// behind a REST API key, and helpers that build signed webhook deliveries
// in WooCommerce's payload shape, with the plugin's order meta. Failures
// can be injected: an outage, throttling, a deleted key, a read-only key.
package wootest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/stores/woocommerce"
)

// Product is a product on the fake site. Prices are decimal strings.
type Product struct {
	ID         int64
	Name       string
	Slug       string
	Status     string // publish, draft
	Price      string
	Regular    string
	Sale       string
	InStock    bool
	Stock      *int
	Categories []int64
	ImageURL   string
	Created    time.Time
	Modified   time.Time
}

// Category is a product category.
type Category struct {
	ID   int64
	Name string
	Slug string
}

// Item is an order line.
type Item struct {
	ProductID int64
	Name      string
	Quantity  int
	Price     float64
}

// Order is an order, with what the ReputeCast plugin recorded at checkout.
type Order struct {
	ID         int64
	Number     string
	Status     string
	Email      string
	FirstName  string
	LastName   string
	Country    string
	CustomerID int64
	Created    time.Time
	Modified   time.Time
	Total      string
	Items      []Item
	Coupons    []string
	Refunded   string
	// Plugin meta: the consent checkbox, its text and time, the click
	// token and the reported checkout.
	Marketing   bool
	ConsentText string
	ConsentAt   *time.Time
	ClickToken  string
	Checkout    string
	// WooCommerce's own order attribution.
	Entry string
}

// Webhook is a webhook the key registered.
type Webhook struct {
	ID       int64
	Topic    string
	URL      string
	Secret   string
	Status   string
	Created  time.Time
	Disabled bool
}

// Fake is the in-memory WooCommerce site.
type Fake struct {
	mu sync.Mutex

	Name     string
	Currency string
	Key      string
	Secret   string
	// ReadOnly keys can't create webhooks.
	ReadOnly bool
	// Plugin: the ReputeCast plugin is active.
	Plugin bool

	Products   []*Product
	Categories []*Category
	Orders     []*Order
	Webhooks   []*Webhook

	PageCap  int
	outage   bool
	revoked  bool
	throttle int
	nextID   int64
	requests []string
	Now      func() time.Time
}

// New returns a fake site with a key and no data.
func New() *Fake {
	return &Fake{Name: "Example Store", Currency: "USD", Key: "ck_recorded0123456789", Secret: "cs_recorded0123456789", nextID: 500, Now: time.Now}
}

// Server serves the fake over httptest.
func (f *Fake) Server() *httptest.Server { return httptest.NewServer(f) }

// Transport answers in process.
func (f *Fake) Transport() http.RoundTripper { return roundTrip{f} }

type roundTrip struct{ f *Fake }

func (r roundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	r.f.ServeHTTP(rec, req)
	return rec.Result(), nil
}

// SetOutage makes every call answer 503.
func (f *Fake) SetOutage(on bool) { f.mu.Lock(); f.outage = on; f.mu.Unlock() }

// Revoke deletes the key: every call answers 401.
func (f *Fake) Revoke() { f.mu.Lock(); f.revoked = true; f.mu.Unlock() }

// Throttle answers the next n calls 429.
func (f *Fake) Throttle(n int) { f.mu.Lock(); f.throttle = n; f.mu.Unlock() }

// Requests lists the calls received ("GET /wp-json/wc/v3/orders").
func (f *Fake) Requests() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests)
}

// WebhookTopics lists the active webhooks' topics.
func (f *Fake) WebhookTopics() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []string{}
	for _, w := range f.Webhooks {
		if w.Status == "active" {
			out = append(out, w.Topic)
		}
	}
	slices.Sort(out)
	return out
}

// SecretFor is the secret a topic's webhook was registered with.
func (f *Fake) SecretFor(topic string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, w := range f.Webhooks {
		if w.Topic == topic {
			return w.Secret
		}
	}
	return ""
}

func (f *Fake) now() time.Time {
	if f.Now != nil {
		return f.Now()
	}
	return time.Now()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func errJSON(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{"code": code, "message": msg, "data": map[string]any{"status": status}})
}

// ServeHTTP implements http.Handler.
func (f *Fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	if f.outage {
		errJSON(w, http.StatusServiceUnavailable, "service_unavailable", "Service unavailable")
		return
	}
	if r.URL.Path == "/wp-json/" || r.URL.Path == "/wp-json" {
		ns := []string{"oembed/1.0", "wp/v2", "wc/v3", "wc/store/v1"}
		if f.Plugin {
			ns = append(ns, woocommerce.PluginNamespace)
		}
		writeJSON(w, http.StatusOK, map[string]any{"name": f.Name, "url": "https://store.example.com", "namespaces": ns})
		return
	}
	path, ok := strings.CutPrefix(r.URL.Path, "/wp-json/wc/v3")
	if !ok {
		errJSON(w, http.StatusNotFound, "rest_no_route", "No route was found matching the URL and request method.")
		return
	}
	key, secret, _ := r.BasicAuth()
	if f.revoked || key != f.Key || secret != f.Secret {
		errJSON(w, http.StatusUnauthorized, "woocommerce_rest_authentication_error", "Consumer key is invalid.")
		return
	}
	if f.throttle > 0 {
		f.throttle--
		w.Header().Set("Retry-After", "1")
		errJSON(w, http.StatusTooManyRequests, "rest_too_many_requests", "Too many requests.")
		return
	}
	switch {
	case path == "/settings/general/woocommerce_currency" && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"id": "woocommerce_currency", "value": f.Currency})
	case path == "/products" && r.Method == http.MethodGet:
		list := []any{}
		for _, p := range f.Products {
			if f.inRange(r, p.Created, p.Modified) {
				list = append(list, f.productJSON(p))
			}
		}
		f.page(w, r, list)
	case path == "/products/categories" && r.Method == http.MethodGet:
		list := []any{}
		for _, c := range f.Categories {
			n := 0
			for _, p := range f.Products {
				if slices.Contains(p.Categories, c.ID) {
					n++
				}
			}
			list = append(list, map[string]any{"id": c.ID, "name": c.Name, "slug": c.Slug, "count": n})
		}
		f.page(w, r, list)
	case path == "/orders" && r.Method == http.MethodGet:
		list := []any{}
		for _, o := range f.Orders {
			if f.inRange(r, o.Created, o.Modified) {
				list = append(list, f.OrderPayload(o))
			}
		}
		f.page(w, r, list)
	case path == "/webhooks" && r.Method == http.MethodGet:
		list := []any{}
		for _, h := range f.Webhooks {
			list = append(list, map[string]any{"id": h.ID, "name": "ReputeCast · " + h.Topic, "status": h.Status, "topic": h.Topic, "delivery_url": h.URL})
		}
		f.page(w, r, list)
	case path == "/webhooks" && r.Method == http.MethodPost:
		if f.ReadOnly {
			errJSON(w, http.StatusUnauthorized, "woocommerce_rest_cannot_create", "Sorry, you are not allowed to create resources.")
			return
		}
		var in struct {
			Topic       string `json:"topic"`
			DeliveryURL string `json:"delivery_url"`
			Secret      string `json:"secret"`
			Status      string `json:"status"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		f.nextID++
		h := &Webhook{ID: f.nextID, Topic: in.Topic, URL: in.DeliveryURL, Secret: in.Secret, Status: "active", Created: f.now()}
		f.Webhooks = append(f.Webhooks, h)
		writeJSON(w, http.StatusCreated, map[string]any{"id": h.ID, "name": "ReputeCast · " + h.Topic, "status": h.Status, "topic": h.Topic, "delivery_url": h.URL})
	case strings.HasPrefix(path, "/webhooks/") && r.Method == http.MethodDelete:
		id, _ := strconv.ParseInt(strings.TrimPrefix(path, "/webhooks/"), 10, 64)
		before := len(f.Webhooks)
		f.Webhooks = slices.DeleteFunc(f.Webhooks, func(h *Webhook) bool { return h.ID == id })
		if len(f.Webhooks) == before {
			errJSON(w, http.StatusNotFound, "woocommerce_rest_webhook_invalid_id", "Invalid ID.")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": id})
	default:
		errJSON(w, http.StatusNotFound, "rest_no_route", "No route was found matching the URL and request method.")
	}
}

func (f *Fake) inRange(r *http.Request, created, modified time.Time) bool {
	q := r.URL.Query()
	if s := q.Get("after"); s != "" {
		if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil && created.Before(t) {
			return false
		}
	}
	if s := q.Get("modified_after"); s != "" {
		if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil && modified.Before(t) {
			return false
		}
	}
	return true
}

func (f *Fake) page(w http.ResponseWriter, r *http.Request, list []any) {
	q := r.URL.Query()
	per, _ := strconv.Atoi(q.Get("per_page"))
	if per <= 0 {
		per = 10
	}
	if f.PageCap > 0 && per > f.PageCap {
		per = f.PageCap
	}
	page, _ := strconv.Atoi(q.Get("page"))
	if page <= 0 {
		page = 1
	}
	pages := (len(list) + per - 1) / per
	from := min((page-1)*per, len(list))
	to := min(from+per, len(list))
	w.Header().Set("X-WP-Total", strconv.Itoa(len(list)))
	w.Header().Set("X-WP-TotalPages", strconv.Itoa(pages))
	writeJSON(w, http.StatusOK, list[from:to])
}

func gmt(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05") }

func (f *Fake) productJSON(p *Product) map[string]any {
	cats := []any{}
	for _, id := range p.Categories {
		for _, c := range f.Categories {
			if c.ID == id {
				cats = append(cats, map[string]any{"id": c.ID, "name": c.Name, "slug": c.Slug})
			}
		}
	}
	stock := "instock"
	if !p.InStock {
		stock = "outofstock"
	}
	images := []any{}
	if p.ImageURL != "" {
		images = append(images, map[string]any{"src": p.ImageURL, "alt": p.Name})
	}
	return map[string]any{
		"id": p.ID, "name": p.Name, "slug": p.Slug, "permalink": "https://store.example.com/product/" + p.Slug, "status": p.Status,
		"price": p.Price, "regular_price": p.Regular, "sale_price": p.Sale, "stock_status": stock, "stock_quantity": p.Stock,
		"manage_stock": p.Stock != nil, "date_created_gmt": gmt(p.Created), "date_modified_gmt": gmt(p.Modified), "images": images, "categories": cats,
	}
}

// ProductPayload is a product in WooCommerce's webhook shape.
func (f *Fake) ProductPayload(p *Product) map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.productJSON(p)
}

// OrderPayload is an order in WooCommerce's shape, with the plugin's meta
// and the fields ReputeCast must never keep (addresses, phone, payment,
// the customer's note).
func (f *Fake) OrderPayload(o *Order) map[string]any {
	items := []any{}
	for i, it := range o.Items {
		items = append(items, map[string]any{"id": 100 + i, "product_id": it.ProductID, "variation_id": 0, "name": it.Name, "quantity": it.Quantity,
			"price": it.Price, "subtotal": strconv.FormatFloat(it.Price*float64(it.Quantity), 'f', 2, 64), "total": strconv.FormatFloat(it.Price*float64(it.Quantity), 'f', 2, 64)})
	}
	coupons := []any{}
	for _, c := range o.Coupons {
		coupons = append(coupons, map[string]any{"code": c})
	}
	refunds := []any{}
	if o.Refunded != "" {
		refunds = append(refunds, map[string]any{"id": 9, "total": "-" + o.Refunded})
	}
	meta := []any{}
	if f.Plugin {
		mk := "no"
		if o.Marketing {
			mk = "yes"
		}
		meta = append(meta, map[string]any{"id": 1, "key": woocommerce.MetaMarketing, "value": mk})
		if o.Marketing {
			meta = append(meta, map[string]any{"id": 2, "key": woocommerce.MetaConsentText, "value": o.ConsentText})
			if o.ConsentAt != nil {
				meta = append(meta, map[string]any{"id": 3, "key": woocommerce.MetaConsentAt, "value": o.ConsentAt.UTC().Format(time.RFC3339)})
			}
		}
		if o.ClickToken != "" {
			meta = append(meta, map[string]any{"id": 4, "key": woocommerce.MetaClickToken, "value": o.ClickToken})
		}
		if o.Checkout != "" {
			meta = append(meta, map[string]any{"id": 5, "key": woocommerce.MetaCheckout, "value": o.Checkout})
		}
	}
	if o.Entry != "" {
		meta = append(meta, map[string]any{"id": 6, "key": "_wc_order_attribution_session_entry", "value": o.Entry})
	}
	status := o.Status
	if status == "" {
		status = "processing"
	}
	return map[string]any{
		"id": o.ID, "number": o.Number, "status": status, "currency": f.Currency, "date_created_gmt": gmt(o.Created), "date_modified_gmt": gmt(o.Modified),
		"total": o.Total, "customer_id": o.CustomerID, "customer_note": "Please gift wrap",
		"billing": map[string]any{"first_name": o.FirstName, "last_name": o.LastName, "email": o.Email, "country": o.Country,
			"address_1": "1 Example Street", "phone": "+1 555 0100"},
		"shipping":       map[string]any{"address_1": "1 Example Street", "phone": "+1 555 0100"},
		"payment_method": "stripe", "transaction_id": "ch_123",
		"line_items": items, "coupon_lines": coupons, "refunds": refunds, "meta_data": meta,
	}
}

// Delivery is a signed webhook request, ready to POST.
type Delivery struct {
	Topic  string
	Header http.Header
	Body   []byte
}

// Request builds the delivery as an *http.Request to address.
func (d Delivery) Request(address string) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, address, bytes.NewReader(d.Body))
	for k, v := range d.Header {
		req.Header[k] = v
	}
	return req
}

// Deliver signs a payload with the topic's webhook secret, as WooCommerce
// sends it.
func (f *Fake) Deliver(topic string, payload any) Delivery {
	body, _ := json.Marshal(payload)
	f.mu.Lock()
	f.nextID++
	id := f.nextID
	secret := ""
	var hookID int64
	for _, w := range f.Webhooks {
		if w.Topic == topic {
			secret, hookID = w.Secret, w.ID
		}
	}
	f.mu.Unlock()
	resource, event, _ := strings.Cut(topic, ".")
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("X-WC-Webhook-Source", "https://store.example.com/")
	h.Set("X-WC-Webhook-Topic", topic)
	h.Set("X-WC-Webhook-Resource", resource)
	h.Set("X-WC-Webhook-Event", event)
	h.Set("X-WC-Webhook-Signature", woocommerce.Sign(secret, body))
	h.Set("X-WC-Webhook-ID", strconv.FormatInt(hookID, 10))
	h.Set("X-WC-Webhook-Delivery-ID", fmt.Sprintf("d%d", id))
	return Delivery{Topic: topic, Header: h, Body: body}
}
