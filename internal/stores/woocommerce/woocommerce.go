// Package woocommerce is the WooCommerce store adapter: WooCommerce's REST
// API (wc/v3) over plain HTTPS with a REST API key the merchant creates
// with Read/Write access (sealed at rest, never shown again), and webhooks
// registered with a secret whose signatures are checked on arrival.
//
// WooCommerce has no marketing consent of its own: the ReputeCast
// WordPress plugin adds the checkout checkbox with its exact text and
// records the consent, the click token and the checkout on the order's
// meta data, which this adapter reads. Abandoned checkouts are reported by
// the plugin to ReputeCast directly (WooCommerce has none to list).
// Addresses, phone numbers, payment details and customer notes are never
// read: the structs below have no field for them.
package woocommerce

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/stores"
)

// Key is the adapter's key in the stores registry.
const Key = "woocommerce"

// Topics are the 6 webhooks ReputeCast registers (refunds arrive as
// order.updated).
var Topics = []string{
	"order.created",
	"order.updated",
	"order.deleted",
	"product.created",
	"product.updated",
	"product.deleted",
}

// Order meta the ReputeCast plugin writes at checkout.
const (
	MetaMarketing   = "_reputecast_marketing"    // "yes" when the box was ticked
	MetaConsentText = "_reputecast_consent_text" // the checkbox's exact text
	MetaConsentAt   = "_reputecast_consent_at"   // RFC 3339
	MetaClickToken  = "_reputecast_click"        // the email's click token
	MetaCheckout    = "_reputecast_checkout"     // the reported checkout's token
	// WooCommerce's own order attribution (8.5+).
	metaEntry  = "_wc_order_attribution_session_entry"
	metaSource = "_wc_order_attribution_utm_source"
	metaMedium = "_wc_order_attribution_utm_medium"
	metaCamp   = "_wc_order_attribution_utm_campaign"
	metaCont   = "_wc_order_attribution_utm_content"
	metaTerm   = "_wc_order_attribution_utm_term"
)

// PluginNamespace is the ReputeCast plugin's REST namespace on the site.
const PluginNamespace = "reputecast/v1"

func init() {
	stores.Register(stores.Descriptor{
		Key:  Key,
		Name: "WooCommerce",
		Capabilities: []stores.Capability{stores.CapOrders, stores.CapProducts, stores.CapCustomers, stores.CapCheckouts,
			stores.CapAttribution},
		Topics:   Topics,
		Open:     func(cfg stores.Config) (stores.Client, error) { return New(cfg) },
		Webhooks: Webhooks{},
	})
}

// Client is a store reached with a REST API key.
type Client struct {
	domain    string
	base      string
	key       string
	secret    string
	hc        *http.Client
	now       func() time.Time
	retryWait time.Duration
	// cur caches the store's currency.
	cur string
}

// New opens a client on a store.
func New(cfg stores.Config) (*Client, error) {
	domain := stores.NormalizeHost(cfg.Domain)
	if domain == "" {
		return nil, fmt.Errorf("woocommerce: %w: no store address", stores.ErrNotAStore)
	}
	c := &Client{domain: domain, base: "https://" + domain, key: strings.TrimSpace(cfg.Key), secret: strings.TrimSpace(cfg.Secret),
		hc: cfg.HTTP, now: cfg.Now, retryWait: 500 * time.Millisecond}
	if cfg.BaseURL != "" {
		c.base = strings.TrimRight(cfg.BaseURL, "/")
		c.retryWait = 5 * time.Millisecond
	}
	if cfg.RetryWait > 0 {
		c.retryWait = cfg.RetryWait
	}
	if c.hc == nil {
		c.hc = &http.Client{Timeout: 30 * time.Second}
	}
	if c.now == nil {
		c.now = time.Now
	}
	return c, nil
}

// KeyFormat checks a REST API key's shape (ck_… and cs_…).
func KeyFormat(key, secret string) bool {
	key, secret = strings.TrimSpace(key), strings.TrimSpace(secret)
	return strings.HasPrefix(key, "ck_") && len(key) > 10 && strings.HasPrefix(secret, "cs_") && len(secret) > 10
}

// call sends one request to wc/v3; 5xx answers are retried twice.
func (c *Client) call(ctx context.Context, method, path string, q url.Values, body, out any) (http.Header, error) {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.retryWait * time.Duration(attempt)):
			}
		}
		h, err := c.once(ctx, method, path, q, body, out)
		if err == nil {
			return h, nil
		}
		last = err
		if !errors.Is(err, stores.ErrUnavailable) {
			return nil, err
		}
	}
	return nil, last
}

func (c *Client) once(ctx context.Context, method, path string, q url.Values, body, out any) (http.Header, error) {
	target := c.base + "/wp-json/wc/v3" + path
	if len(q) > 0 {
		target += "?" + q.Encode()
	}
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, rd)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.key, c.secret)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ReputeCast")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", stores.ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", stores.ErrUnavailable, err)
	}
	if err := statusError(resp, raw); err != nil {
		return nil, err
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return nil, fmt.Errorf("woocommerce: unreadable answer to %s %s: %w", method, path, err)
		}
	}
	return resp.Header, nil
}

func statusError(resp *http.Response, raw []byte) error {
	if resp.StatusCode < 300 {
		return nil
	}
	var e struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(raw, &e)
	msg := e.Message
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return fmt.Errorf("%w: %s %s", stores.ErrUnauthorized, e.Code, msg)
	case resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("%w: %s %s", stores.ErrPermission, e.Code, msg)
	case resp.StatusCode == http.StatusNotFound:
		if e.Code == "rest_no_route" {
			return fmt.Errorf("%w: WooCommerce's REST API isn't there", stores.ErrNotAStore)
		}
		return fmt.Errorf("%w: %s", stores.ErrNotFound, msg)
	case resp.StatusCode == http.StatusTooManyRequests:
		wait := 2 * time.Second
		if s := resp.Header.Get("Retry-After"); s != "" {
			if n, err := strconv.Atoi(s); err == nil && n > 0 {
				wait = time.Duration(n) * time.Second
			}
		}
		return &stores.RateLimited{RetryAfter: wait}
	case resp.StatusCode >= 500:
		return fmt.Errorf("%w: %d", stores.ErrUnavailable, resp.StatusCode)
	}
	return fmt.Errorf("woocommerce: %d %s", resp.StatusCode, msg)
}

// Probe is what a site's REST index says about it.
type Probe struct {
	Name string
	// WooCommerce: the wc/v3 namespace is there.
	WooCommerce bool
	// Plugin: the ReputeCast plugin's namespace is there.
	Plugin bool
}

// ProbeSite reads the site's REST index (no key needed).
func ProbeSite(ctx context.Context, cfg stores.Config) (Probe, error) {
	base := "https://" + stores.NormalizeHost(cfg.Domain)
	if cfg.BaseURL != "" {
		base = strings.TrimRight(cfg.BaseURL, "/")
	}
	hc := cfg.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/wp-json/", nil)
	if err != nil {
		return Probe{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ReputeCast")
	resp, err := hc.Do(req)
	if err != nil {
		return Probe{}, fmt.Errorf("%w: %w", stores.ErrNotAStore, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return Probe{}, stores.ErrNotAStore
	}
	var idx struct {
		Name       string   `json:"name"`
		Namespaces []string `json:"namespaces"`
	}
	if err := json.Unmarshal(raw, &idx); err != nil {
		return Probe{}, stores.ErrNotAStore
	}
	p := Probe{Name: idx.Name}
	for _, ns := range idx.Namespaces {
		switch ns {
		case "wc/v3":
			p.WooCommerce = true
		case PluginNamespace:
			p.Plugin = true
		}
	}
	if !p.WooCommerce {
		return p, stores.ErrNotAStore
	}
	return p, nil
}

// Shop implements stores.Client.
func (c *Client) Shop(ctx context.Context) (stores.Shop, error) {
	var cur struct {
		Value string `json:"value"`
	}
	if _, err := c.call(ctx, http.MethodGet, "/settings/general/woocommerce_currency", nil, nil, &cur); err != nil {
		return stores.Shop{}, err
	}
	p, _ := ProbeSite(ctx, stores.Config{Domain: c.domain, BaseURL: strings.TrimSuffix(c.base, "/"), HTTP: c.hc})
	name := p.Name
	if name == "" {
		name = c.domain
	}
	return stores.Shop{Name: name, Domain: c.domain, PublicDomain: c.domain, Currency: strings.ToUpper(cur.Value)}, nil
}

func listQuery(page int, r stores.Range, orderby string) url.Values {
	q := url.Values{}
	q.Set("per_page", "100")
	q.Set("page", strconv.Itoa(page))
	if orderby != "" {
		q.Set("orderby", orderby)
		q.Set("order", "asc")
	}
	if !r.CreatedSince.IsZero() {
		q.Set("after", r.CreatedSince.UTC().Format("2006-01-02T15:04:05"))
		q.Set("dates_are_gmt", "true")
	}
	if !r.UpdatedSince.IsZero() {
		q.Set("modified_after", r.UpdatedSince.UTC().Format("2006-01-02T15:04:05"))
		q.Set("dates_are_gmt", "true")
	}
	return q
}

// pageOf turns a cursor ("2") into a page number.
func pageOf(cursor string) int {
	if n, err := strconv.Atoi(cursor); err == nil && n > 0 {
		return n
	}
	return 1
}

func nextPage(h http.Header, page, got int) string {
	if total, err := strconv.Atoi(h.Get("X-WP-TotalPages")); err == nil {
		if page < total {
			return strconv.Itoa(page + 1)
		}
		return ""
	}
	if got == 100 {
		return strconv.Itoa(page + 1)
	}
	return ""
}

// Count implements stores.Client (X-WP-Total of a one-item page).
func (c *Client) Count(ctx context.Context, kind string, r stores.Range) (int, error) {
	path := "/" + kind
	switch kind {
	case stores.CountProducts, stores.CountOrders:
	case stores.CountCustomers:
		return 0, stores.ErrUnsupported
	default:
		return 0, fmt.Errorf("woocommerce: nothing to count as %q", kind)
	}
	q := listQuery(1, r, "")
	q.Set("per_page", "1")
	var items []json.RawMessage
	h, err := c.call(ctx, http.MethodGet, path, q, nil, &items)
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(h.Get("X-WP-Total"))
	if err != nil {
		return len(items), nil
	}
	return n, nil
}

type wcProduct struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Permalink       string `json:"permalink"`
	Status          string `json:"status"`
	Price           string `json:"price"`
	RegularPrice    string `json:"regular_price"`
	SalePrice       string `json:"sale_price"`
	StockStatus     string `json:"stock_status"`
	StockQuantity   *int   `json:"stock_quantity"`
	ManageStock     any    `json:"manage_stock"`
	DateCreatedGMT  string `json:"date_created_gmt"`
	DateModifiedGMT string `json:"date_modified_gmt"`
	Images          []struct {
		Src string `json:"src"`
		Alt string `json:"alt"`
	} `json:"images"`
	Categories []struct {
		ID int64 `json:"id"`
	} `json:"categories"`
}

func gmt(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse("2006-01-02T15:04:05", s); err == nil {
		return t.UTC()
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC()
	}
	return time.Time{}
}

func (p wcProduct) product(currency string) stores.Product {
	out := stores.Product{ID: strconv.FormatInt(p.ID, 10), Title: p.Name, Handle: p.Slug, URL: p.Permalink, Currency: currency,
		Price: stores.ParseAmount(p.Price, currency), CreatedAt: gmt(p.DateCreatedGMT), UpdatedAt: gmt(p.DateModifiedGMT), Collections: []string{}}
	switch p.Status {
	case "publish":
		out.Status = "active"
	case "draft", "pending", "private":
		out.Status = "draft"
	default:
		out.Status = p.Status
	}
	if reg := stores.ParseAmount(p.RegularPrice, currency); p.SalePrice != "" && reg > out.Price {
		out.CompareAt = reg
	}
	if len(p.Images) > 0 {
		out.ImageURL, out.ImageAlt = p.Images[0].Src, p.Images[0].Alt
	}
	out.Available = p.StockStatus != "outofstock"
	if managed, _ := p.ManageStock.(bool); managed && p.StockQuantity != nil {
		n := *p.StockQuantity
		out.Inventory = &n
	}
	for _, c := range p.Categories {
		out.Collections = append(out.Collections, strconv.FormatInt(c.ID, 10))
	}
	return out
}

// Products implements stores.Client.
func (c *Client) Products(ctx context.Context, cursor string, r stores.Range) (stores.Page[stores.Product], error) {
	page := pageOf(cursor)
	var items []wcProduct
	h, err := c.call(ctx, http.MethodGet, "/products", listQuery(page, r, "modified"), nil, &items)
	if err != nil {
		return stores.Page[stores.Product]{}, err
	}
	cur := c.currency(ctx)
	out := stores.Page[stores.Product]{Items: make([]stores.Product, 0, len(items)), Next: nextPage(h, page, len(items))}
	for _, p := range items {
		out.Items = append(out.Items, p.product(cur))
	}
	return out, nil
}

// currency reads the store's currency once per client.
func (c *Client) currency(ctx context.Context) string {
	if c.cur != "" {
		return c.cur
	}
	var cur struct {
		Value string `json:"value"`
	}
	if _, err := c.call(ctx, http.MethodGet, "/settings/general/woocommerce_currency", nil, nil, &cur); err != nil || cur.Value == "" {
		return "USD"
	}
	c.cur = strings.ToUpper(cur.Value)
	return c.cur
}

// Collections implements stores.Client: product categories.
func (c *Client) Collections(ctx context.Context, cursor string) (stores.Page[stores.Collection], error) {
	page := pageOf(cursor)
	var items []struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Slug  string `json:"slug"`
		Count int    `json:"count"`
	}
	q := url.Values{"per_page": {"100"}, "page": {strconv.Itoa(page)}}
	h, err := c.call(ctx, http.MethodGet, "/products/categories", q, nil, &items)
	if err != nil {
		return stores.Page[stores.Collection]{}, err
	}
	out := stores.Page[stores.Collection]{Items: []stores.Collection{}, Next: nextPage(h, page, len(items))}
	for _, it := range items {
		out.Items = append(out.Items, stores.Collection{ID: strconv.FormatInt(it.ID, 10), Title: it.Name, Handle: it.Slug, Products: it.Count})
	}
	return out, nil
}

// Customers implements stores.Client. WooCommerce customers carry no
// consent: customers come from their orders' consent (MetaMarketing).
func (c *Client) Customers(context.Context, string, stores.Range) (stores.Page[stores.Customer], error) {
	return stores.Page[stores.Customer]{}, stores.ErrUnsupported
}

type wcMeta struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

type wcOrder struct {
	ID              int64  `json:"id"`
	Number          string `json:"number"`
	Status          string `json:"status"`
	Currency        string `json:"currency"`
	DateCreatedGMT  string `json:"date_created_gmt"`
	DateModifiedGMT string `json:"date_modified_gmt"`
	Total           string `json:"total"`
	CustomerID      int64  `json:"customer_id"`
	Billing         struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Country   string `json:"country"`
	} `json:"billing"`
	LineItems []struct {
		ProductID   int64   `json:"product_id"`
		VariationID int64   `json:"variation_id"`
		Name        string  `json:"name"`
		Quantity    int     `json:"quantity"`
		Price       float64 `json:"price"`
		Subtotal    string  `json:"subtotal"`
		Image       *struct {
			Src string `json:"src"`
		} `json:"image"`
	} `json:"line_items"`
	CouponLines []struct {
		Code string `json:"code"`
	} `json:"coupon_lines"`
	Refunds []struct {
		ID    int64  `json:"id"`
		Total string `json:"total"`
	} `json:"refunds"`
	MetaData []wcMeta `json:"meta_data"`
}

func metaString(meta []wcMeta, key string) string {
	for _, m := range meta {
		if m.Key == key {
			switch v := m.Value.(type) {
			case string:
				return v
			case float64:
				return strconv.FormatFloat(v, 'f', -1, 64)
			case bool:
				if v {
					return "yes"
				}
				return "no"
			}
		}
	}
	return ""
}

func (o wcOrder) order() stores.Order {
	cur := strings.ToUpper(o.Currency)
	email := strings.ToLower(strings.TrimSpace(o.Billing.Email))
	out := stores.Order{
		ID: strconv.FormatInt(o.ID, 10), Number: o.Number, Name: "#" + o.Number, Email: email,
		PlacedAt: gmt(o.DateCreatedGMT), UpdatedAt: gmt(o.DateModifiedGMT), Currency: cur, Total: stores.ParseAmount(o.Total, cur),
		Cancelled: o.Status == "cancelled" || o.Status == "failed", Discounts: []string{}, Items: []stores.LineItem{},
	}
	if out.Number == "" {
		out.Number = out.ID
		out.Name = "#" + out.ID
	}
	for _, li := range o.LineItems {
		item := stores.LineItem{ProductID: strconv.FormatInt(li.ProductID, 10), Title: li.Name, Quantity: li.Quantity,
			Price: stores.ParseAmount(strconv.FormatFloat(li.Price, 'f', -1, 64), cur)}
		if li.VariationID > 0 {
			item.VariantID = strconv.FormatInt(li.VariationID, 10)
		}
		if li.Image != nil {
			item.ImageURL = li.Image.Src
		}
		out.Items = append(out.Items, item)
		out.Subtotal += stores.ParseAmount(li.Subtotal, cur)
	}
	for _, cl := range o.CouponLines {
		out.Discounts = append(out.Discounts, cl.Code)
	}
	for _, r := range o.Refunds {
		amt := stores.ParseAmount(r.Total, cur)
		if amt < 0 {
			amt = -amt
		}
		out.Refunded += amt
	}
	// Consent, the click token and the checkout, from the plugin.
	consent := stores.Consent{State: stores.ConsentNotSubscribed, Level: "unknown"}
	if metaString(o.MetaData, MetaMarketing) == "yes" {
		consent = stores.Consent{State: stores.ConsentSubscribed, Level: "single_opt_in", Text: metaString(o.MetaData, MetaConsentText)}
		if at := metaString(o.MetaData, MetaConsentAt); at != "" {
			if t, err := time.Parse(time.RFC3339, at); err == nil {
				consent.UpdatedAt = &t
			}
		}
	}
	out.AcceptsMarketing = consent.Accepted()
	custID := "guest:" + email
	if o.CustomerID > 0 {
		custID = strconv.FormatInt(o.CustomerID, 10)
	}
	out.Customer = &stores.Customer{ID: custID, Email: email, FirstName: o.Billing.FirstName, LastName: o.Billing.LastName,
		Country: o.Billing.Country, Consent: consent, CreatedAt: out.PlacedAt, UpdatedAt: out.UpdatedAt}
	out.ClickToken = metaString(o.MetaData, MetaClickToken)
	out.CheckoutToken = metaString(o.MetaData, MetaCheckout)
	out.LandingPage = metaString(o.MetaData, metaEntry)
	utm, token := stores.ParseLanding(out.LandingPage)
	if out.ClickToken == "" {
		out.ClickToken = token
	}
	if u := (stores.UTM{Source: metaString(o.MetaData, metaSource), Medium: metaString(o.MetaData, metaMedium), Campaign: metaString(o.MetaData, metaCamp),
		Content: metaString(o.MetaData, metaCont), Term: metaString(o.MetaData, metaTerm)}); u != (stores.UTM{}) {
		utm = u
	}
	out.UTM = utm
	return out
}

// Orders implements stores.Client.
func (c *Client) Orders(ctx context.Context, cursor string, r stores.Range) (stores.Page[stores.Order], error) {
	page := pageOf(cursor)
	var items []wcOrder
	h, err := c.call(ctx, http.MethodGet, "/orders", listQuery(page, r, "date"), nil, &items)
	if err != nil {
		return stores.Page[stores.Order]{}, err
	}
	out := stores.Page[stores.Order]{Items: make([]stores.Order, 0, len(items)), Next: nextPage(h, page, len(items))}
	for _, o := range items {
		out.Items = append(out.Items, o.order())
	}
	return out, nil
}

// Checkouts implements stores.Client: WooCommerce has none of its own; the
// ReputeCast plugin reports them.
func (c *Client) Checkouts(context.Context, string, stores.Range) (stores.Page[stores.Checkout], error) {
	return stores.Page[stores.Checkout]{}, stores.ErrUnsupported
}

type wcWebhook struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Topic       string `json:"topic"`
	DeliveryURL string `json:"delivery_url"`
}

// Webhooks implements stores.Client. Paused or disabled webhooks count as
// missing: WooCommerce disables a webhook after repeated failures.
func (c *Client) Webhooks(ctx context.Context) ([]stores.Webhook, error) {
	out := []stores.Webhook{}
	for page := 1; page <= 10; page++ {
		var items []wcWebhook
		h, err := c.call(ctx, http.MethodGet, "/webhooks", url.Values{"per_page": {"100"}, "page": {strconv.Itoa(page)}}, nil, &items)
		if err != nil {
			return nil, err
		}
		for _, w := range items {
			if w.Status != "active" {
				continue
			}
			out = append(out, stores.Webhook{ID: strconv.FormatInt(w.ID, 10), Topic: w.Topic, Address: w.DeliveryURL})
		}
		if nextPage(h, page, len(items)) == "" {
			break
		}
	}
	return out, nil
}

// Subscribe implements stores.Client: WooCommerce signs each delivery with
// the secret given here.
func (c *Client) Subscribe(ctx context.Context, topic, address, secret string) (stores.Webhook, error) {
	var w wcWebhook
	body := map[string]any{"name": "ReputeCast · " + topic, "topic": topic, "delivery_url": address, "secret": secret, "status": "active"}
	if _, err := c.call(ctx, http.MethodPost, "/webhooks", nil, body, &w); err != nil {
		return stores.Webhook{}, err
	}
	return stores.Webhook{ID: strconv.FormatInt(w.ID, 10), Topic: w.Topic, Address: w.DeliveryURL}, nil
}

// Unsubscribe implements stores.Client. A webhook that's already gone is
// not an error.
func (c *Client) Unsubscribe(ctx context.Context, id string) error {
	_, err := c.call(ctx, http.MethodDelete, "/webhooks/"+url.PathEscape(id), url.Values{"force": {"true"}}, nil, nil)
	if errors.Is(err, stores.ErrNotFound) {
		return nil
	}
	return err
}

// SetConsent implements stores.Client: WooCommerce keeps no consent to
// write back to (the plugin's checkbox asks again at each checkout).
func (c *Client) SetConsent(context.Context, string, string, time.Time) error {
	return stores.ErrUnsupported
}

// Webhooks verifies and parses WooCommerce's deliveries
// (X-WC-Webhook-Signature: base64 HMAC-SHA256 of the raw body).
type Webhooks struct{}

var _ stores.WebhookParser = Webhooks{}

// Sign is the X-WC-Webhook-Signature value of a body (tests and the fake).
func Sign(secret string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	return base64.StdEncoding.EncodeToString(m.Sum(nil))
}

// Verify implements stores.WebhookParser.
func (Webhooks) Verify(h http.Header, body []byte, secret string) error {
	got := strings.TrimSpace(h.Get("X-WC-Webhook-Signature"))
	if got == "" || secret == "" {
		return stores.ErrBadSignature
	}
	sig, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		return stores.ErrBadSignature
	}
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	if !hmac.Equal(sig, m.Sum(nil)) {
		return stores.ErrBadSignature
	}
	return nil
}

// Topic implements stores.WebhookParser.
func (Webhooks) Topic(h http.Header) (string, string) {
	return strings.ToLower(strings.TrimSpace(h.Get("X-WC-Webhook-Topic"))), h.Get("X-WC-Webhook-Delivery-ID")
}

// Parse implements stores.WebhookParser.
func (w Webhooks) Parse(h http.Header, body []byte) (stores.Event, error) {
	topic, delivery := w.Topic(h)
	ev := stores.Event{Topic: topic, DeliveryID: delivery}
	// WooCommerce's delivery check when a webhook is created: "webhook_id=12".
	if bytes.HasPrefix(bytes.TrimSpace(body), []byte("webhook_id=")) {
		ev.Kind = stores.EventPing
		return ev, nil
	}
	switch topic {
	case "order.created", "order.updated", "order.restored":
		var o wcOrder
		if err := json.Unmarshal(body, &o); err != nil {
			return ev, fmt.Errorf("woocommerce: unreadable order: %w", err)
		}
		order := o.order()
		ev.Kind, ev.Order = stores.EventOrder, &order
	case "order.deleted":
		var o struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal(body, &o); err != nil {
			return ev, fmt.Errorf("woocommerce: unreadable order deletion: %w", err)
		}
		ev.Kind, ev.DeletedID = stores.EventOrderDeleted, strconv.FormatInt(o.ID, 10)
	case "product.created", "product.updated", "product.restored":
		var p wcProduct
		if err := json.Unmarshal(body, &p); err != nil {
			return ev, fmt.Errorf("woocommerce: unreadable product: %w", err)
		}
		prod := p.product(h.Get(stores.CurrencyHeader))
		ev.Kind, ev.Product = stores.EventProduct, &prod
	case "product.deleted":
		var p struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal(body, &p); err != nil {
			return ev, fmt.Errorf("woocommerce: unreadable product deletion: %w", err)
		}
		ev.Kind, ev.DeletedID = stores.EventProductDeleted, strconv.FormatInt(p.ID, 10)
	default:
		return ev, fmt.Errorf("woocommerce: topic %q isn't one ReputeCast subscribes to", topic)
	}
	return ev, nil
}
