// Package shopifytest is an in-memory Shopify: the store's storefront
// metadata (/meta.json), the app install (OAuth's code exchange and token
// refresh) and the subset of the GraphQL Admin API the adapter calls,
// answered from an in-memory store in the shapes Shopify returns, plus
// helpers that build signed webhook deliveries in Shopify's REST payload
// shape. Failures can be injected: an outage, throttling, a revoked token,
// a missing access scope. Tests serve it with httptest; the demo runs it in
// process (Transport), so nothing leaves the machine.
package shopifytest

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/stores/shopify"
)

// Product is a product in the fake store. Prices are decimal strings.
type Product struct {
	ID          string
	Title       string
	Handle      string
	Status      string // ACTIVE, DRAFT, ARCHIVED
	Price       string
	CompareAt   string
	Tracked     bool
	Inventory   int
	ImageURL    string
	Collections []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Collection is a collection in the fake store.
type Collection struct {
	ID     string
	Title  string
	Handle string
}

// Customer is a customer in the fake store.
type Customer struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
	Country   string
	// State is SUBSCRIBED, NOT_SUBSCRIBED, UNSUBSCRIBED or PENDING.
	State     string
	Level     string // SINGLE_OPT_IN, CONFIRMED_OPT_IN, UNKNOWN
	ConsentAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Item is an order's or a checkout's line.
type Item struct {
	ProductID string
	Title     string
	Variant   string
	Quantity  int
	Price     string
}

// Order is an order in the fake store.
type Order struct {
	ID               string
	Number           int
	Email            string
	CustomerID       string
	AcceptsMarketing bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Subtotal         string
	Total            string
	Refunded         string
	Discounts        []string
	Landing          string
	CheckoutToken    string
	Items            []Item
	Test             bool
	Cancelled        bool
}

// Checkout is an abandoned checkout in the fake store.
type Checkout struct {
	ID               string
	Token            string
	Email            string
	CustomerID       string
	AcceptsMarketing bool
	Items            []Item
	Subtotal         string
	Total            string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CompletedAt      *time.Time
}

// Webhook is a subscription the app registered.
type Webhook struct {
	ID    string
	Topic string // ORDERS_CREATE
	URI   string
}

// Recorded is one request the fake received.
type Recorded struct {
	Method    string
	Path      string
	Operation string
	Token     string
}

// Fake is the in-memory Shopify.
type Fake struct {
	mu sync.Mutex

	Name          string
	Domain        string // acme-merch.myshopify.com
	PrimaryDomain string // shop.acme.io
	Currency      string
	ClientID      string
	ClientSecret  string
	// Scopes the merchant grants on approval.
	Scopes []string

	Products    []*Product
	Collections []*Collection
	Customers   []*Customer
	Orders      []*Order
	Checkouts   []*Checkout
	Webhooks    []*Webhook

	tokens map[string]bool // valid access tokens
	codes  map[string]bool // unused approval codes
	nextID int

	// PageCap caps every page at this many items whatever the request asks
	// (tests use it to exercise paging with a few objects).
	PageCap int
	// Failures.
	outage     bool
	revoked    bool
	throttle   int
	noConsent  bool
	requests   []Recorded
	consentLog []string
	// Now is the fake's clock.
	Now func() time.Time
}

// New returns a fake store with Acme Merch's identity and no data.
func New() *Fake {
	return &Fake{Name: "Acme Merch", Domain: "acme-merch.myshopify.com", PrimaryDomain: "shop.acme.io", Currency: "USD",
		ClientID: "4f1c8e2b7d03a9e2", ClientSecret: "shpss_recorded7c1d", Scopes: []string{"read_customers", "read_orders", "read_products", "write_customers"},
		tokens: map[string]bool{}, codes: map[string]bool{}, nextID: 9000, Now: time.Now}
}

// Server serves the fake over httptest.
func (f *Fake) Server() *httptest.Server { return httptest.NewServer(f) }

// Transport answers in process (the demo): every request goes to the fake.
func (f *Fake) Transport() http.RoundTripper { return roundTrip{f} }

type roundTrip struct{ f *Fake }

func (r roundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	r.f.ServeHTTP(rec, req)
	return rec.Result(), nil
}

// IssueToken adds a valid access token (a store already installed).
func (f *Fake) IssueToken(token string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tokens[token] = true
}

// Approve is the merchant approving the app on Shopify's page: it returns
// the redirect Shopify sends the browser to, signed with the client secret.
func (f *Fake) Approve(redirect, state string) string {
	f.mu.Lock()
	code := fmt.Sprintf("code-%d", f.nextID)
	f.nextID++
	f.codes[code] = true
	f.mu.Unlock()
	q := url.Values{}
	q.Set("code", code)
	q.Set("shop", f.Domain)
	q.Set("state", state)
	q.Set("timestamp", strconv.FormatInt(f.now().Unix(), 10))
	q.Set("host", base64.RawStdEncoding.EncodeToString([]byte(f.Domain+"/admin")))
	q.Set("hmac", shopify.SignQuery(q, f.ClientSecret))
	sep := "?"
	if strings.Contains(redirect, "?") {
		sep = "&"
	}
	return redirect + sep + q.Encode()
}

// SetOutage makes every call answer 503.
func (f *Fake) SetOutage(on bool) { f.mu.Lock(); f.outage = on; f.mu.Unlock() }

// Revoke refuses every access token (the app was uninstalled).
func (f *Fake) Revoke() { f.mu.Lock(); f.revoked = true; f.mu.Unlock() }

// Throttle answers the next n GraphQL calls with THROTTLED.
func (f *Fake) Throttle(n int) { f.mu.Lock(); f.throttle = n; f.mu.Unlock() }

// DenyConsentWrites refuses consent updates (write_customers not granted).
func (f *Fake) DenyConsentWrites() { f.mu.Lock(); f.noConsent = true; f.mu.Unlock() }

// Requests returns the requests received so far.
func (f *Fake) Requests() []Recorded {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests)
}

// ConsentUpdates lists the customers whose consent the app changed ("id=STATE").
func (f *Fake) ConsentUpdates() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.consentLog)
}

// WebhookTopics lists the registered topics ("orders/create").
func (f *Fake) WebhookTopics() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []string{}
	for _, w := range f.Webhooks {
		out = append(out, shopify.TopicFromEnum(w.Topic))
	}
	sort.Strings(out)
	return out
}

// DeleteWebhook removes a subscription as the merchant would in Shopify.
func (f *Fake) DeleteWebhook(topic string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	enum := shopify.TopicEnum(topic)
	f.Webhooks = slices.DeleteFunc(f.Webhooks, func(w *Webhook) bool { return w.Topic == enum })
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

// ServeHTTP implements http.Handler.
func (f *Fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	outage := f.outage
	f.mu.Unlock()
	if outage {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"errors": "Service unavailable"})
		return
	}
	switch {
	case r.URL.Path == "/meta.json" && r.Method == http.MethodGet:
		f.record(r, "", "")
		writeJSON(w, http.StatusOK, map[string]any{"name": f.Name, "currency": f.Currency, "myshopify_domain": f.Domain,
			"domain": f.PrimaryDomain, "url": "https://" + f.PrimaryDomain})
	case r.URL.Path == "/admin/oauth/access_token" && r.Method == http.MethodPost:
		f.accessToken(w, r)
	case strings.HasPrefix(r.URL.Path, "/admin/api/") && strings.HasSuffix(r.URL.Path, "/graphql.json") && r.Method == http.MethodPost:
		f.graphql(w, r)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"errors": "Not Found"})
	}
}

func (f *Fake) record(r *http.Request, op, token string) {
	f.mu.Lock()
	f.requests = append(f.requests, Recorded{Method: r.Method, Path: r.URL.Path, Operation: op, Token: token})
	f.mu.Unlock()
}

func (f *Fake) accessToken(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Code         string `json:"code"`
		GrantType    string `json:"grant_type"`
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	f.record(r, "access_token", "")
	f.mu.Lock()
	defer f.mu.Unlock()
	if in.ClientID != f.ClientID || in.ClientSecret != f.ClientSecret {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client", "error_description": "Client credentials are invalid"})
		return
	}
	if in.GrantType == "refresh_token" {
		if !f.tokens["refresh:"+in.RefreshToken] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
			return
		}
	} else {
		if !f.codes[in.Code] {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "The authorization code was not found or was already used"})
			return
		}
		delete(f.codes, in.Code)
	}
	tok := fmt.Sprintf("shpat_fake%06d", f.nextID)
	f.nextID++
	f.tokens[tok] = true
	writeJSON(w, http.StatusOK, map[string]any{"access_token": tok, "scope": strings.Join(f.Scopes, ",")})
}

type gqlIn struct {
	Query         string         `json:"query"`
	OperationName string         `json:"operationName"`
	Variables     map[string]any `json:"variables"`
}

var opName = regexp.MustCompile(`^\s*(?:query|mutation)\s+([A-Za-z]+)`)

func (f *Fake) graphql(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Shopify-Access-Token")
	raw, _ := io.ReadAll(r.Body)
	var in gqlIn
	_ = json.Unmarshal(raw, &in)
	op := in.OperationName
	if op == "" {
		if m := opName.FindStringSubmatch(in.Query); m != nil {
			op = m[1]
		}
	}
	f.record(r, op, token)
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.revoked || !f.tokens[token] {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"errors": "[API] Invalid API key or access token (unrecognized login or wrong password)"})
		return
	}
	cost := map[string]any{"requestedQueryCost": 50, "actualQueryCost": 12,
		"throttleStatus": map[string]any{"maximumAvailable": 2000.0, "currentlyAvailable": 1988.0, "restoreRate": 100.0}}
	if f.throttle > 0 {
		f.throttle--
		writeJSON(w, http.StatusOK, map[string]any{
			"errors":     []any{map[string]any{"message": "Throttled", "extensions": map[string]any{"code": "THROTTLED"}}},
			"extensions": map[string]any{"cost": map[string]any{"requestedQueryCost": 50, "throttleStatus": map[string]any{"maximumAvailable": 2000.0, "currentlyAvailable": 40.0, "restoreRate": 1000.0}}},
		})
		return
	}
	data, errs := f.answer(op, in.Variables)
	out := map[string]any{"data": data, "extensions": map[string]any{"cost": cost}}
	if len(errs) > 0 {
		out["errors"] = errs
	}
	writeJSON(w, http.StatusOK, out)
}

func gid(kind, id string) string { return "gid://shopify/" + kind + "/" + id }

func numeric(id string) string {
	if i := strings.LastIndexByte(id, '/'); i >= 0 {
		return id[i+1:]
	}
	return id
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func intOf(v any, def int) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return def
}

var rangeTerm = regexp.MustCompile(`(created_at|updated_at):>='([^']+)'`)

// inRange applies a search query's created_at and updated_at bounds.
func inRange(query string, created, updated time.Time) bool {
	for _, m := range rangeTerm.FindAllStringSubmatch(query, -1) {
		t, err := time.Parse(time.RFC3339, m[2])
		if err != nil {
			continue
		}
		at := created
		if m[1] == "updated_at" {
			at = updated
		}
		if at.Before(t) {
			return false
		}
	}
	return true
}

func cursorOf(v any) int {
	s := str(v)
	if s == "" {
		return 0
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(string(b))
	return n
}

// page slices n items from offset; it returns the page's bounds and info.
func (f *Fake) page(total int, vars map[string]any) (int, int, map[string]any) {
	from := cursorOf(vars["after"])
	n := intOf(vars["first"], 50)
	if f.PageCap > 0 && n > f.PageCap {
		n = f.PageCap
	}
	to := min(from+n, total)
	if from > total {
		from = total
	}
	info := map[string]any{"hasNextPage": to < total, "endCursor": base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(to)))}
	return from, to, info
}

func (f *Fake) money(amount string) map[string]any {
	return map[string]any{"shopMoney": map[string]any{"amount": amount, "currencyCode": f.Currency}}
}

func (f *Fake) customerNode(c *Customer) map[string]any {
	if c == nil {
		return nil
	}
	var at any
	if c.ConsentAt != nil {
		at = c.ConsentAt.UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"id": gid("Customer", c.ID), "firstName": c.FirstName, "lastName": c.LastName,
		"createdAt": c.CreatedAt.UTC().Format(time.RFC3339), "updatedAt": c.UpdatedAt.UTC().Format(time.RFC3339),
		"defaultEmailAddress": map[string]any{"emailAddress": c.Email, "marketingState": c.State, "marketingOptInLevel": c.Level, "marketingUpdatedAt": at},
		"defaultAddress":      map[string]any{"countryCodeV2": c.Country},
	}
}

func (f *Fake) customer(id string) *Customer {
	for _, c := range f.Customers {
		if c.ID == id {
			return c
		}
	}
	return nil
}

func (f *Fake) items(items []Item) map[string]any {
	nodes := []any{}
	for _, it := range items {
		nodes = append(nodes, map[string]any{"title": it.Title, "variantTitle": it.Variant, "quantity": it.Quantity,
			"product": map[string]any{"id": gid("Product", it.ProductID)}, "variant": nil, "originalUnitPriceSet": f.money(it.Price), "image": nil})
	}
	return map[string]any{"nodes": nodes}
}

func (f *Fake) answer(op string, vars map[string]any) (map[string]any, []any) {
	query := str(vars["query"])
	switch op {
	case "Shop":
		return map[string]any{"shop": map[string]any{"name": f.Name, "currencyCode": f.Currency, "myshopifyDomain": f.Domain,
			"primaryDomain": map[string]any{"host": f.PrimaryDomain}}}, nil
	case "Count":
		p, c, o := 0, 0, 0
		for _, x := range f.Products {
			if inRange(query, x.CreatedAt, x.UpdatedAt) {
				p++
			}
		}
		for _, x := range f.Customers {
			if inRange(query, x.CreatedAt, x.UpdatedAt) {
				c++
			}
		}
		for _, x := range f.Orders {
			if inRange(query, x.CreatedAt, x.UpdatedAt) {
				o++
			}
		}
		return map[string]any{"productsCount": map[string]any{"count": p}, "customersCount": map[string]any{"count": c}, "ordersCount": map[string]any{"count": o}}, nil
	case "Products":
		list := []*Product{}
		for _, x := range f.Products {
			if inRange(query, x.CreatedAt, x.UpdatedAt) {
				list = append(list, x)
			}
		}
		from, to, info := f.page(len(list), vars)
		nodes := []any{}
		for _, p := range list[from:to] {
			cols := []any{}
			for _, c := range p.Collections {
				cols = append(cols, map[string]any{"id": gid("Collection", c)})
			}
			var media any
			if p.ImageURL != "" {
				media = map[string]any{"preview": map[string]any{"image": map[string]any{"url": p.ImageURL, "altText": p.Title}}}
			}
			var compare any
			if p.CompareAt != "" {
				compare = map[string]any{"minVariantCompareAtPrice": map[string]any{"amount": p.CompareAt, "currencyCode": f.Currency}}
			}
			nodes = append(nodes, map[string]any{
				"id": gid("Product", p.ID), "title": p.Title, "handle": p.Handle, "status": p.Status,
				"onlineStoreUrl": "https://" + f.PrimaryDomain + "/products/" + p.Handle,
				"createdAt":      p.CreatedAt.UTC().Format(time.RFC3339), "updatedAt": p.UpdatedAt.UTC().Format(time.RFC3339),
				"featuredMedia":       media,
				"priceRangeV2":        map[string]any{"minVariantPrice": map[string]any{"amount": p.Price, "currencyCode": f.Currency}},
				"compareAtPriceRange": compare,
				"tracksInventory":     p.Tracked, "totalInventory": p.Inventory,
				"collections": map[string]any{"nodes": cols},
			})
		}
		return map[string]any{"products": map[string]any{"nodes": nodes, "pageInfo": info}}, nil
	case "Collections":
		from, to, info := f.page(len(f.Collections), vars)
		nodes := []any{}
		for _, c := range f.Collections[from:to] {
			n := 0
			for _, p := range f.Products {
				if slices.Contains(p.Collections, c.ID) {
					n++
				}
			}
			nodes = append(nodes, map[string]any{"id": gid("Collection", c.ID), "title": c.Title, "handle": c.Handle, "productsCount": map[string]any{"count": n}})
		}
		return map[string]any{"collections": map[string]any{"nodes": nodes, "pageInfo": info}}, nil
	case "Customers":
		list := []*Customer{}
		for _, x := range f.Customers {
			if inRange(query, x.CreatedAt, x.UpdatedAt) {
				list = append(list, x)
			}
		}
		from, to, info := f.page(len(list), vars)
		nodes := []any{}
		for _, c := range list[from:to] {
			nodes = append(nodes, f.customerNode(c))
		}
		return map[string]any{"customers": map[string]any{"nodes": nodes, "pageInfo": info}}, nil
	case "Orders":
		list := []*Order{}
		for _, x := range f.Orders {
			if inRange(query, x.CreatedAt, x.UpdatedAt) {
				list = append(list, x)
			}
		}
		from, to, info := f.page(len(list), vars)
		nodes := []any{}
		for _, o := range list[from:to] {
			var cancelled any
			if o.Cancelled {
				cancelled = o.UpdatedAt.UTC().Format(time.RFC3339)
			}
			var journey any
			if o.Landing != "" {
				v := map[string]any{"landingPage": "https://" + f.PrimaryDomain + o.Landing, "utmParameters": nil}
				journey = map[string]any{"firstVisit": v, "lastVisit": v}
			}
			refunded := o.Refunded
			if refunded == "" {
				refunded = "0.0"
			}
			nodes = append(nodes, map[string]any{
				"id": gid("Order", o.ID), "name": "#" + strconv.Itoa(o.Number),
				"createdAt": o.CreatedAt.UTC().Format(time.RFC3339), "updatedAt": o.UpdatedAt.UTC().Format(time.RFC3339),
				"cancelledAt": cancelled, "test": o.Test, "email": o.Email, "currencyCode": f.Currency,
				"customerAcceptsMarketing": o.AcceptsMarketing, "discountCodes": nonNil(o.Discounts),
				"subtotalPriceSet": f.money(o.Subtotal), "totalPriceSet": f.money(o.Total), "totalRefundedSet": f.money(refunded),
				"customer": f.customerNode(f.customer(o.CustomerID)), "customerJourneySummary": journey, "lineItems": f.items(o.Items),
			})
		}
		return map[string]any{"orders": map[string]any{"nodes": nodes, "pageInfo": info}}, nil
	case "Checkouts":
		list := []*Checkout{}
		for _, x := range f.Checkouts {
			if inRange(query, x.CreatedAt, x.UpdatedAt) {
				list = append(list, x)
			}
		}
		from, to, info := f.page(len(list), vars)
		nodes := []any{}
		for _, c := range list[from:to] {
			var done any
			if c.CompletedAt != nil {
				done = c.CompletedAt.UTC().Format(time.RFC3339)
			}
			nodes = append(nodes, map[string]any{
				"id": gid("AbandonedCheckout", c.ID), "abandonedCheckoutUrl": "https://" + f.PrimaryDomain + "/checkouts/" + c.Token + "/recover",
				"createdAt": c.CreatedAt.UTC().Format(time.RFC3339), "updatedAt": c.UpdatedAt.UTC().Format(time.RFC3339), "completedAt": done,
				"customer": f.customerNode(f.customer(c.CustomerID)), "subtotalPriceSet": f.money(c.Subtotal), "totalPriceSet": f.money(c.Total),
				"lineItems": f.items(c.Items),
			})
		}
		return map[string]any{"abandonedCheckouts": map[string]any{"nodes": nodes, "pageInfo": info}}, nil
	case "Webhooks":
		nodes := []any{}
		for _, w := range f.Webhooks {
			nodes = append(nodes, map[string]any{"id": gid("WebhookSubscription", w.ID), "topic": w.Topic, "uri": w.URI})
		}
		return map[string]any{"webhookSubscriptions": map[string]any{"nodes": nodes}}, nil
	case "Subscribe":
		topic := str(vars["topic"])
		sub, _ := vars["sub"].(map[string]any)
		uri := str(sub["uri"])
		for _, w := range f.Webhooks {
			if w.Topic == topic && w.URI == uri {
				return map[string]any{"webhookSubscriptionCreate": map[string]any{"webhookSubscription": nil,
					"userErrors": []any{map[string]any{"field": []string{"webhookSubscription", "callbackUrl"}, "message": "Address for this topic has already been taken"}}}}, nil
			}
		}
		id := strconv.Itoa(f.nextID)
		f.nextID++
		f.Webhooks = append(f.Webhooks, &Webhook{ID: id, Topic: topic, URI: uri})
		return map[string]any{"webhookSubscriptionCreate": map[string]any{"webhookSubscription": map[string]any{"id": gid("WebhookSubscription", id), "topic": topic, "uri": uri},
			"userErrors": []any{}}}, nil
	case "Unsubscribe":
		id := numeric(str(vars["id"]))
		before := len(f.Webhooks)
		f.Webhooks = slices.DeleteFunc(f.Webhooks, func(w *Webhook) bool { return w.ID == id })
		if len(f.Webhooks) == before {
			return map[string]any{"webhookSubscriptionDelete": map[string]any{"deletedWebhookSubscriptionId": nil,
				"userErrors": []any{map[string]any{"field": []string{"id"}, "message": "Webhook subscription does not exist"}}}}, nil
		}
		return map[string]any{"webhookSubscriptionDelete": map[string]any{"deletedWebhookSubscriptionId": gid("WebhookSubscription", id), "userErrors": []any{}}}, nil
	case "Consent":
		if f.noConsent || !slices.Contains(f.Scopes, "write_customers") {
			return nil, []any{map[string]any{"message": "Access denied for customerEmailMarketingConsentUpdate field.", "extensions": map[string]any{"code": "ACCESS_DENIED"}}}
		}
		in, _ := vars["input"].(map[string]any)
		id := numeric(str(in["customerId"]))
		emc, _ := in["emailMarketingConsent"].(map[string]any)
		state := str(emc["marketingState"])
		c := f.customer(id)
		if c == nil {
			return map[string]any{"customerEmailMarketingConsentUpdate": map[string]any{"customer": nil,
				"userErrors": []any{map[string]any{"field": []string{"customerId"}, "message": "Customer not found"}}}}, nil
		}
		now := f.now()
		c.State, c.ConsentAt, c.UpdatedAt = state, &now, now
		f.consentLog = append(f.consentLog, id+"="+state)
		return map[string]any{"customerEmailMarketingConsentUpdate": map[string]any{"customer": map[string]any{"id": gid("Customer", id)}, "userErrors": []any{}}}, nil
	}
	return nil, []any{map[string]any{"message": "Unknown operation " + op}}
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// ── Webhook deliveries ─────────────────────────────────────────────────

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

// Deliver signs a REST-shaped payload as Shopify would send it on topic.
func (f *Fake) Deliver(topic string, payload any) Delivery {
	body, _ := json.Marshal(payload)
	return f.DeliverRaw(topic, body, "")
}

// DeliverRaw signs a body; eventID names the event (a retry repeats it).
func (f *Fake) DeliverRaw(topic string, body []byte, eventID string) Delivery {
	f.mu.Lock()
	if eventID == "" {
		eventID = fmt.Sprintf("evt-%d", f.nextID)
		f.nextID++
	}
	secret, domain := f.ClientSecret, f.Domain
	f.mu.Unlock()
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("X-Shopify-Topic", topic)
	h.Set("X-Shopify-Hmac-Sha256", shopify.Sign(secret, body))
	h.Set("X-Shopify-Shop-Domain", domain)
	h.Set("X-Shopify-API-Version", shopify.APIVersion)
	h.Set("X-Shopify-Webhook-Id", "wh-"+eventID)
	h.Set("X-Shopify-Event-Id", eventID)
	h.Set("X-Shopify-Triggered-At", f.now().UTC().Format(time.RFC3339Nano))
	return Delivery{Topic: topic, Header: h, Body: body}
}

// CustomerPayload is a customer in Shopify's webhook shape.
func (f *Fake) CustomerPayload(c *Customer) map[string]any {
	var at any
	if c.ConsentAt != nil {
		at = c.ConsentAt.UTC().Format(time.RFC3339)
	}
	id, _ := strconv.ParseInt(c.ID, 10, 64)
	return map[string]any{
		"id": id, "email": c.Email, "first_name": c.FirstName, "last_name": c.LastName,
		"created_at": c.CreatedAt.UTC().Format(time.RFC3339), "updated_at": c.UpdatedAt.UTC().Format(time.RFC3339),
		"email_marketing_consent": map[string]any{"state": strings.ToLower(c.State), "opt_in_level": strings.ToLower(c.Level), "consent_updated_at": at},
		"default_address":         map[string]any{"country_code": c.Country, "address1": "1 Example Street", "phone": "+1 555 0100"},
		"phone":                   "+1 555 0100", "note": "Asked about bulk orders",
	}
}

func itemsPayload(items []Item) []any {
	out := []any{}
	for i, it := range items {
		pid, _ := strconv.ParseInt(it.ProductID, 10, 64)
		out = append(out, map[string]any{"id": 1000 + i, "product_id": pid, "variant_id": nil, "title": it.Title, "variant_title": it.Variant,
			"quantity": it.Quantity, "price": it.Price})
	}
	return out
}

// OrderPayload is an order in Shopify's webhook shape (with the fields
// ReputeCast must never keep: addresses, phone, payment, note).
func (f *Fake) OrderPayload(o *Order) map[string]any {
	id, _ := strconv.ParseInt(o.ID, 10, 64)
	var cust any
	if c := f.customer(o.CustomerID); c != nil {
		cust = f.CustomerPayload(c)
	}
	var cancelled any
	if o.Cancelled {
		cancelled = o.UpdatedAt.UTC().Format(time.RFC3339)
	}
	codes := []any{}
	for _, d := range o.Discounts {
		codes = append(codes, map[string]any{"code": d, "amount": "5.00", "type": "fixed_amount"})
	}
	return map[string]any{
		"id": id, "name": "#" + strconv.Itoa(o.Number), "order_number": o.Number, "email": o.Email,
		"created_at": o.CreatedAt.UTC().Format(time.RFC3339), "updated_at": o.UpdatedAt.UTC().Format(time.RFC3339), "cancelled_at": cancelled,
		"currency": f.Currency, "subtotal_price": o.Subtotal, "total_price": o.Total, "discount_codes": codes,
		"landing_site": o.Landing, "checkout_token": o.CheckoutToken, "buyer_accepts_marketing": o.AcceptsMarketing, "test": o.Test,
		"customer": cust, "line_items": itemsPayload(o.Items), "refunds": []any{},
		"billing_address":  map[string]any{"address1": "1 Example Street", "city": "Springfield", "phone": "+1 555 0100"},
		"shipping_address": map[string]any{"address1": "1 Example Street", "city": "Springfield", "phone": "+1 555 0100"},
		"payment_details":  map[string]any{"credit_card_number": "•••• •••• •••• 4242"}, "note": "Leave at the door", "phone": "+1 555 0100",
	}
}

// CheckoutPayload is a checkout in Shopify's webhook shape.
func (f *Fake) CheckoutPayload(c *Checkout) map[string]any {
	id, _ := strconv.ParseInt(c.ID, 10, 64)
	var cust any
	if cu := f.customer(c.CustomerID); cu != nil {
		cust = f.CustomerPayload(cu)
	}
	var done any
	if c.CompletedAt != nil {
		done = c.CompletedAt.UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"id": id, "token": c.Token, "email": c.Email, "created_at": c.CreatedAt.UTC().Format(time.RFC3339), "updated_at": c.UpdatedAt.UTC().Format(time.RFC3339),
		"completed_at": done, "abandoned_checkout_url": "https://" + f.PrimaryDomain + "/checkouts/" + c.Token + "/recover",
		"currency": f.Currency, "subtotal_price": c.Subtotal, "total_price": c.Total, "buyer_accepts_marketing": c.AcceptsMarketing,
		"customer": cust, "line_items": itemsPayload(c.Items), "shipping_address": map[string]any{"address1": "1 Example Street"},
	}
}

// ProductPayload is a product in Shopify's webhook shape.
func (f *Fake) ProductPayload(p *Product) map[string]any {
	id, _ := strconv.ParseInt(p.ID, 10, 64)
	mgmt := ""
	if p.Tracked {
		mgmt = "shopify"
	}
	var img any
	if p.ImageURL != "" {
		img = map[string]any{"src": p.ImageURL, "alt": p.Title}
	}
	return map[string]any{
		"id": id, "title": p.Title, "handle": p.Handle, "status": strings.ToLower(p.Status),
		"created_at": p.CreatedAt.UTC().Format(time.RFC3339), "updated_at": p.UpdatedAt.UTC().Format(time.RFC3339), "image": img,
		"variants": []any{map[string]any{"price": p.Price, "compare_at_price": p.CompareAt, "inventory_quantity": p.Inventory,
			"inventory_management": mgmt, "inventory_policy": "deny"}},
	}
}
