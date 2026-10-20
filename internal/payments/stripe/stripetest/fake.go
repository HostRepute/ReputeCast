// Package stripetest is an in-memory Stripe: the subset of Stripe's API the
// adapter calls, answered from recorded object shapes, with restricted keys
// and their permissions, failure injection and helpers that complete a
// checkout or change a subscription and sign the webhooks Stripe would
// send. Tests serve it with httptest (no network); the demo runs it in
// process for its sample account, so nothing leaves the machine.
package stripetest

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/payments/stripe"
)

// KeySpec is a restricted key the fake accepts.
type KeySpec struct {
	Mode payments.Mode
	// Missing are permission resources the key lacks ("webhook_endpoints").
	Missing []string
	// Rejected answers 401 to every call (a rolled or deleted key).
	Rejected bool
}

// Account is the account every key belongs to.
type Account struct {
	ID        string
	Name      string
	Country   string
	Currency  string
	TaxActive bool
}

// Recorded is one request the fake received.
type Recorded struct {
	Method string
	Path   string
	Key    string
	Form   url.Values
}

type failRule struct {
	method, prefix string
	status         int
	left           int
}

// Fake is the in-memory Stripe.
type Fake struct {
	mu       sync.Mutex
	Account  Account
	keys     map[string]KeySpec
	modes    map[payments.Mode]*data
	fails    []failRule
	requests []Recorded
	// Now is the fake's clock.
	Now func() time.Time
	// Permissive answers any object id the fake doesn't know as if it
	// existed (the demo, whose sample objects aren't all loaded).
	Permissive bool
}

type obj = map[string]any

type data struct {
	objects map[string]obj // by id, every kind
	events  []obj          // oldest first
}

// New returns a fake for the sample account, Acme Media (US, USD, Stripe
// Tax on).
func New() *Fake {
	return &Fake{
		Account: Account{ID: "acct_1AcmeMediaQ2x", Name: "Acme Media", Country: "US", Currency: "usd", TaxActive: true},
		keys:    map[string]KeySpec{},
		modes:   map[payments.Mode]*data{payments.Live: {objects: map[string]obj{}}, payments.Test: {objects: map[string]obj{}}},
		Now:     time.Now,
	}
}

// AddKey makes a restricted key valid.
func (f *Fake) AddKey(key string, spec KeySpec) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if spec.Mode == "" {
		spec.Mode = payments.Live
		if strings.HasPrefix(key, "rk_test_") {
			spec.Mode = payments.Test
		}
	}
	f.keys[key] = spec
}

// RejectKey makes a key answer 401 from now on.
func (f *Fake) RejectKey(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := f.keys[key]
	k.Rejected = true
	f.keys[key] = k
}

// FailNext answers the next n requests matching method and path prefix
// with status (503 for an outage).
func (f *Fake) FailNext(method, prefix string, status, n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fails = append(f.fails, failRule{method: method, prefix: prefix, status: status, left: n})
}

// Requests returns what the fake received.
func (f *Fake) Requests() []Recorded {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests)
}

// Server serves the fake on a loopback port until the test ends.
func (f *Fake) Server(t interface{ Cleanup(func()) }) *httptest.Server {
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return srv
}

// HTTPClient reaches the fake in process, whatever URL is asked for.
func (f *Fake) HTTPClient() *http.Client {
	return &http.Client{Transport: roundTrip{f}, Timeout: 10 * time.Second}
}

type roundTrip struct{ f *Fake }

func (r roundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	// A client's GET has no body, where a server's request always has one.
	if req.Body == nil {
		req = req.Clone(req.Context())
		req.Body = http.NoBody
	}
	rec := httptest.NewRecorder()
	r.f.ServeHTTP(rec, req)
	return rec.Result(), nil
}

func (f *Fake) now() time.Time {
	if f.Now != nil {
		return f.Now()
	}
	return time.Now()
}

func newID(prefix string) string {
	b := make([]byte, 7)
	_, _ = rand.Read(b)
	return prefix + "_" + strings.ToUpper(hex.EncodeToString(b))[:6] + hex.EncodeToString(b)[6:]
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func stripeError(w http.ResponseWriter, status int, typ, code, msg string) {
	e := obj{"type": typ, "message": msg}
	if code != "" {
		e["code"] = code
	}
	writeJSON(w, status, obj{"error": e})
}

// permissionOf names the permission a request needs.
func permissionOf(method, path string) (resource, access string) {
	access = "write"
	if method == http.MethodGet {
		access = "read"
	}
	switch {
	case strings.HasPrefix(path, "/v1/customers"):
		return "customers", access
	case strings.HasPrefix(path, "/v1/checkout/sessions"):
		return "checkout_sessions", access
	case strings.HasPrefix(path, "/v1/billing_portal"):
		return "customer_portal", access
	case strings.HasPrefix(path, "/v1/products"):
		return "products", access
	case strings.HasPrefix(path, "/v1/prices"):
		return "prices", access
	case strings.HasPrefix(path, "/v1/coupons"):
		return "coupons", access
	case strings.HasPrefix(path, "/v1/promotion_codes"):
		return "promotion_codes", access
	case strings.HasPrefix(path, "/v1/subscriptions"):
		return "subscriptions", access
	case strings.HasPrefix(path, "/v1/invoices"):
		return "invoices", access
	case strings.HasPrefix(path, "/v1/charges"), strings.HasPrefix(path, "/v1/refunds"):
		return "charges", access
	case strings.HasPrefix(path, "/v1/webhook_endpoints"):
		return "webhook_endpoints", access
	case strings.HasPrefix(path, "/v1/events"):
		return "events", access
	case strings.HasPrefix(path, "/v1/tax/settings"):
		return "tax_settings", access
	}
	return "", ""
}

// ServeHTTP answers one API request.
func (f *Fake) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	body, _ := io.ReadAll(r.Body)
	form, _ := url.ParseQuery(string(body))
	if r.Method == http.MethodGet || r.Method == http.MethodDelete {
		form = r.URL.Query()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, Recorded{Method: r.Method, Path: r.URL.Path, Key: key, Form: form})
	for i := range f.fails {
		fr := &f.fails[i]
		if fr.left > 0 && fr.method == r.Method && strings.HasPrefix(r.URL.Path, fr.prefix) {
			fr.left--
			stripeError(w, fr.status, "api_error", "", http.StatusText(fr.status))
			return
		}
	}
	spec, ok := f.keys[key]
	if !ok || spec.Rejected {
		masked := key
		if len(key) > 4 {
			masked = key[:min(8, len(key))] + "****" + key[len(key)-4:]
		}
		stripeError(w, http.StatusUnauthorized, "invalid_request_error", "", "Invalid API Key provided: "+masked)
		return
	}
	if res, access := permissionOf(r.Method, r.URL.Path); res != "" && slices.Contains(spec.Missing, res) {
		stripeError(w, http.StatusForbidden, "invalid_request_error", "",
			fmt.Sprintf("The provided key does not have the required permissions for this endpoint. Having the 'rak_%s_%s' permission would allow this request to continue.", res, access))
		return
	}
	d := f.modes[spec.Mode]
	live := spec.Mode == payments.Live
	path := r.URL.Path
	seg := strings.Split(strings.Trim(path, "/"), "/")
	switch {
	case r.Method == http.MethodGet && path == "/v1/account":
		writeJSON(w, 200, obj{"id": f.Account.ID, "object": "account", "country": f.Account.Country, "default_currency": f.Account.Currency,
			"settings": obj{"dashboard": obj{"display_name": f.Account.Name}}, "business_profile": obj{"name": f.Account.Name}})
	case r.Method == http.MethodGet && path == "/v1/tax/settings":
		status := "pending"
		if f.Account.TaxActive {
			status = "active"
		}
		writeJSON(w, 200, obj{"object": "tax.settings", "status": status})
	case r.Method == http.MethodGet && (path == "/v1/invoices" || path == "/v1/charges" || path == "/v1/refunds"):
		writeJSON(w, 200, obj{"object": "list", "data": []obj{}, "has_more": false})
	case r.Method == http.MethodGet && path == "/v1/events":
		f.listEvents(w, d, form)
	case r.Method == http.MethodGet && path == "/v1/subscriptions":
		f.list(w, d, "subscription", form)
	case r.Method == http.MethodGet && path == "/v1/customers":
		f.list(w, d, "customer", form)
	case r.Method == http.MethodPost && path == "/v1/webhook_endpoints":
		id := newID("we")
		o := obj{"id": id, "object": "webhook_endpoint", "url": form.Get("url"), "enabled_events": form["enabled_events[]"], "status": "enabled",
			"api_version": form.Get("api_version"), "livemode": live, "secret": "whsec_" + hex.EncodeToString([]byte(id))[:24]}
		d.objects[id] = o
		writeJSON(w, 200, o)
	case r.Method == http.MethodPost && path == "/v1/billing_portal/configurations":
		id := newID("bpc")
		o := obj{"id": id, "object": "billing_portal.configuration", "livemode": live, "features": formTree(form)}
		d.objects[id] = o
		writeJSON(w, 200, o)
	case r.Method == http.MethodPost && path == "/v1/billing_portal/sessions":
		id := newID("bps")
		writeJSON(w, 200, obj{"id": id, "object": "billing_portal.session", "customer": form.Get("customer"), "url": "https://billing.stripe.com/p/session/" + id,
			"return_url": form.Get("return_url"), "livemode": live})
	case r.Method == http.MethodPost && path == "/v1/checkout/sessions":
		id := newID("cs_" + string(spec.Mode))
		o := obj{"id": id, "object": "checkout.session", "url": "https://checkout.stripe.com/c/pay/" + id, "status": "open", "payment_status": "unpaid",
			"mode": sessionMode(form), "customer": nilIfEmpty(form.Get("customer")), "customer_email": nilIfEmpty(form.Get("customer_email")), "subscription": nil,
			"client_reference_id": nilIfEmpty(form.Get("client_reference_id")), "metadata": prefixed(form, "metadata"), "livemode": live,
			"success_url": form.Get("success_url"), "cancel_url": form.Get("cancel_url"),
			"_price": form.Get("line_items[0][price]"), "_trial_days": form.Get("subscription_data[trial_period_days]"),
			"_promotion_code": form.Get("discounts[0][promotion_code]"), "_tax": form.Get("automatic_tax[enabled]")}
		d.objects[id] = o
		writeJSON(w, 200, public(o))
	case r.Method == http.MethodPost && path == "/v1/products":
		id := newID("prod")
		o := obj{"id": id, "object": "product", "name": form.Get("name"), "description": form.Get("description"), "active": form.Get("active") != "false",
			"metadata": prefixed(form, "metadata"), "livemode": live}
		d.objects[id] = o
		writeJSON(w, 200, o)
	case r.Method == http.MethodPost && path == "/v1/prices":
		id := newID("price")
		amount, _ := strconv.ParseInt(form.Get("unit_amount"), 10, 64)
		o := obj{"id": id, "object": "price", "product": form.Get("product"), "unit_amount": amount, "currency": form.Get("currency"), "active": true,
			"tax_behavior": form.Get("tax_behavior"), "metadata": prefixed(form, "metadata"), "livemode": live}
		if iv := form.Get("recurring[interval]"); iv != "" {
			o["recurring"] = obj{"interval": iv, "interval_count": 1}
			o["type"] = "recurring"
		} else {
			o["type"] = "one_time"
		}
		customAmount(o, form)
		d.objects[id] = o
		writeJSON(w, 200, o)
	case len(seg) >= 3 && seg[0] == "v1":
		f.object(w, r, d, seg, form, live)
	default:
		stripeError(w, 404, "invalid_request_error", "resource_missing", "Unrecognized request URL ("+r.Method+": "+path+").")
	}
}

// object answers the requests about one object: GET reads it, POST updates
// it, DELETE removes it, and 404 when it doesn't exist (the permission
// probes rely on that).
func (f *Fake) object(w http.ResponseWriter, r *http.Request, d *data, seg []string, form url.Values, live bool) {
	kind := seg[1]
	id := seg[2]
	if kind == "checkout" && len(seg) >= 4 {
		kind, id = "checkout_sessions", seg[3]
	}
	if kind == "billing_portal" && len(seg) >= 4 {
		kind, id = "portal_configurations", seg[3]
	}
	o, ok := d.objects[id]
	if !ok && f.Permissive && !strings.Contains(id, "reputecast_check") {
		o = f.synthesize(d, kind, id, live)
		ok = o != nil
	}
	if !ok {
		stripeError(w, 404, "invalid_request_error", "resource_missing", fmt.Sprintf("No such %s: '%s'", strings.TrimSuffix(kind, "s"), id))
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, 200, public(o))
	case http.MethodDelete:
		switch kind {
		case "subscriptions":
			now := f.now().Unix()
			prev := obj{"status": o["status"]}
			o["status"], o["canceled_at"], o["ended_at"] = "canceled", now, now
			f.emitLocked(d, "customer.subscription.deleted", o, prev, live)
			writeJSON(w, 200, public(o))
		default:
			delete(d.objects, id)
			writeJSON(w, 200, obj{"id": id, "deleted": true})
		}
	case http.MethodPost:
		switch kind {
		case "subscriptions":
			if v := form.Get("cancel_at_period_end"); v != "" {
				prev := obj{"cancel_at_period_end": o["cancel_at_period_end"]}
				o["cancel_at_period_end"] = v == "true"
				if v == "true" {
					o["cancel_at"] = o["current_period_end"]
					o["canceled_at"] = f.now().Unix()
				} else {
					o["cancel_at"], o["canceled_at"] = nil, nil
				}
				f.emitLocked(d, "customer.subscription.updated", o, prev, live)
			}
		case "prices":
			if v := form.Get("active"); v != "" {
				o["active"] = v == "true"
			}
		case "products":
			for _, k := range []string{"name", "description"} {
				if form.Has(k) {
					o[k] = form.Get(k)
				}
			}
			if v := form.Get("active"); v != "" {
				o["active"] = v == "true"
			}
		case "checkout_sessions":
			if len(seg) >= 5 && seg[4] == "expire" {
				o["status"] = "expired"
			}
		case "portal_configurations":
			o["features"] = formTree(form)
		}
		writeJSON(w, 200, public(o))
	}
}

// synthesize makes up an object the demo refers to but the fake never
// created (Permissive).
func (f *Fake) synthesize(d *data, kind, id string, live bool) obj {
	now := f.now().Unix()
	switch kind {
	case "subscriptions":
		o := obj{"id": id, "object": "subscription", "customer": "cus_demo", "status": "active", "start_date": now, "current_period_end": now + 30*86400,
			"cancel_at_period_end": false, "cancel_at": nil, "canceled_at": nil, "ended_at": nil, "trial_end": nil, "livemode": live,
			"items": obj{"data": []obj{}}, "metadata": obj{}}
		d.objects[id] = o
		return o
	case "customers":
		o := obj{"id": id, "object": "customer", "email": nil, "created": now, "livemode": live}
		d.objects[id] = o
		return o
	case "prices":
		o := obj{"id": id, "object": "price", "active": true, "unit_amount": 0, "currency": f.Account.Currency, "livemode": live}
		d.objects[id] = o
		return o
	case "charges":
		o := obj{"id": id, "object": "charge", "amount": 0, "amount_refunded": 0, "currency": f.Account.Currency, "failure_message": nil, "livemode": live}
		d.objects[id] = o
		return o
	case "webhook_endpoints":
		o := obj{"id": id, "object": "webhook_endpoint", "status": "enabled", "livemode": live}
		d.objects[id] = o
		return o
	}
	return nil
}

func (f *Fake) list(w http.ResponseWriter, d *data, object string, form url.Values) {
	items := []obj{}
	for _, o := range d.objects {
		if o["object"] == object {
			items = append(items, o)
		}
	}
	// Newest first, as Stripe lists.
	slices.SortFunc(items, func(a, b obj) int {
		if c := num(b["created"]) - num(a["created"]); c != 0 {
			return int(c)
		}
		return strings.Compare(fmt.Sprint(b["id"]), fmt.Sprint(a["id"]))
	})
	if object == "subscription" && form.Get("status") != "all" {
		items = slices.DeleteFunc(items, func(o obj) bool { return o["status"] == "canceled" })
	}
	page(w, items, form)
}

func (f *Fake) listEvents(w http.ResponseWriter, d *data, form url.Values) {
	since, _ := strconv.ParseInt(form.Get("created[gte]"), 10, 64)
	types := form["types[]"]
	items := []obj{}
	for i := len(d.events) - 1; i >= 0; i-- {
		e := d.events[i]
		if num(e["created"]) < since || (len(types) > 0 && !slices.Contains(types, fmt.Sprint(e["type"]))) {
			continue
		}
		items = append(items, e)
	}
	page(w, items, form)
}

func page(w http.ResponseWriter, items []obj, form url.Values) {
	limit, _ := strconv.Atoi(form.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	if after := form.Get("starting_after"); after != "" {
		for i, o := range items {
			if o["id"] == after {
				items = items[i+1:]
				break
			}
		}
	}
	more := len(items) > limit
	if more {
		items = items[:limit]
	}
	out := make([]obj, len(items))
	for i, o := range items {
		out[i] = public(o)
	}
	writeJSON(w, 200, obj{"object": "list", "data": out, "has_more": more})
}

func num(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	}
	return 0
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// public is an object without the fake's own bookkeeping (keys starting with _).
func public(o obj) obj {
	out := obj{}
	for k, v := range o {
		if !strings.HasPrefix(k, "_") {
			out[k] = v
		}
	}
	return out
}

func prefixed(form url.Values, prefix string) obj {
	out := obj{}
	for k, v := range form {
		if strings.HasPrefix(k, prefix+"[") && strings.HasSuffix(k, "]") && len(v) > 0 {
			out[strings.TrimSuffix(strings.TrimPrefix(k, prefix+"["), "]")] = v[0]
		}
	}
	return out
}

func formTree(form url.Values) obj {
	out := obj{}
	for k, v := range form {
		out[k] = strings.Join(v, ",")
	}
	return out
}

// Get returns a copy of an object of mode.
func (f *Fake) Get(mode payments.Mode, id string) map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	o, ok := f.modes[mode].objects[id]
	if !ok {
		return nil
	}
	return public(o)
}

// Objects returns the objects of one kind ("subscription", "price").
func (f *Fake) Objects(mode payments.Mode, object string) []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []map[string]any{}
	for _, o := range f.modes[mode].objects {
		if o["object"] == object {
			out = append(out, public(o))
		}
	}
	slices.SortFunc(out, func(a, b map[string]any) int { return strings.Compare(fmt.Sprint(a["id"]), fmt.Sprint(b["id"])) })
	return out
}

// Put stores an object as is (recorded shapes for tests and the demo).
func (f *Fake) Put(mode payments.Mode, o map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := o["livemode"]; !ok {
		o["livemode"] = mode == payments.Live
	}
	if _, ok := o["created"]; !ok {
		o["created"] = f.now().Unix()
	}
	f.modes[mode].objects[fmt.Sprint(o["id"])] = o
}

// DeleteObject removes an object (a webhook endpoint deleted in Stripe's dashboard).
func (f *Fake) DeleteObject(mode payments.Mode, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.modes[mode].objects, id)
}

// Event is one event the fake created, as Stripe sends it.
type Event struct {
	ID   string
	Type string
	Body []byte
}

func (f *Fake) emitLocked(d *data, typ string, o obj, previous obj, live bool) Event {
	id := newID("evt")
	data := obj{"object": public(o)}
	if previous != nil {
		data["previous_attributes"] = previous
	}
	e := obj{"id": id, "object": "event", "type": typ, "created": f.now().Unix(), "livemode": live, "api_version": stripe.APIVersion, "data": data}
	d.events = append(d.events, e)
	b, _ := json.Marshal(e)
	return Event{ID: id, Type: typ, Body: b}
}

// Emit records an event about o (and stores o) and returns it as sent.
func (f *Fake) Emit(mode payments.Mode, typ string, o map[string]any, previous map[string]any) Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := f.modes[mode]
	if id, ok := o["id"].(string); ok && !strings.HasPrefix(typ, "invoice.upcoming") {
		if _, exists := d.objects[id]; !exists {
			o["livemode"] = mode == payments.Live
		}
		d.objects[id] = o
	}
	return f.emitLocked(d, typ, o, previous, mode == payments.Live)
}

// Events returns the events of mode, oldest first.
func (f *Fake) Events(mode payments.Mode) []Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []Event{}
	for _, e := range f.modes[mode].events {
		b, _ := json.Marshal(e)
		out = append(out, Event{ID: fmt.Sprint(e["id"]), Type: fmt.Sprint(e["type"]), Body: b})
	}
	return out
}

// CheckoutOptions completes a checkout the way a reader would.
type CheckoutOptions struct {
	Email string
	// Customer reuses an existing customer.
	Customer string
	// Paid makes the first payment at once (no trial).
	Interval string
}

// CompleteCheckout pays a checkout session: it creates the customer and
// the subscription (trialing when the session has a trial), and the
// events Stripe sends in order: customer.subscription.created,
// checkout.session.completed and, without a trial, invoice.paid.
func (f *Fake) CompleteCheckout(mode payments.Mode, sessionID string, o CheckoutOptions) ([]Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := f.modes[mode]
	live := mode == payments.Live
	cs, ok := d.objects[sessionID]
	if !ok {
		return nil, fmt.Errorf("stripetest: no checkout session %s", sessionID)
	}
	now := f.now().Unix()
	email := o.Email
	if email == "" {
		email, _ = cs["customer_email"].(string)
	}
	cus := o.Customer
	if cus == "" {
		cus, _ = cs["customer"].(string)
	}
	if cus == "" {
		cus = newID("cus")
		d.objects[cus] = obj{"id": cus, "object": "customer", "email": email, "name": nil, "created": now, "livemode": live}
	}
	priceID, _ := cs["_price"].(string)
	price := d.objects[priceID]
	if price == nil {
		price = obj{"id": priceID, "object": "price", "unit_amount": int64(0), "currency": f.Account.Currency, "recurring": obj{"interval": "month"}}
	}
	interval := "month"
	if rec, ok := price["recurring"].(obj); ok {
		interval, _ = rec["interval"].(string)
	}
	period := int64(30 * 86400)
	if interval == "year" {
		period = 365 * 86400
	}
	trial, _ := strconv.Atoi(fmt.Sprint(cs["_trial_days"]))
	sub := newID("sub")
	s := obj{"id": sub, "object": "subscription", "customer": cus, "status": "active", "start_date": now, "created": now, "livemode": live,
		"cancel_at_period_end": false, "cancel_at": nil, "canceled_at": nil, "ended_at": nil, "trial_end": nil, "current_period_end": now + period,
		"items": obj{"object": "list", "data": []obj{{"price": public(price), "current_period_end": now + period}}}, "metadata": cs["metadata"], "discount": nil}
	if trial > 0 {
		s["status"] = "trialing"
		s["trial_end"] = now + int64(trial)*86400
		s["current_period_end"] = now + int64(trial)*86400
	}
	if code, _ := cs["_promotion_code"].(string); code != "" {
		if promo, ok := d.objects[code]; ok {
			s["discount"] = obj{"coupon": promo["coupon"], "promotion_code": code, "start": now, "end": nil}
		}
	}
	d.objects[sub] = s
	cs["status"], cs["payment_status"], cs["customer"], cs["subscription"] = "complete", "paid", cus, sub
	cs["customer_details"] = obj{"email": email}
	if trial > 0 {
		cs["payment_status"] = "no_payment_required"
	}
	out := []Event{f.emitLocked(d, "customer.subscription.created", s, nil, live), f.emitLocked(d, "checkout.session.completed", cs, nil, live)}
	if trial == 0 {
		amount := num(price["unit_amount"])
		inv := obj{"id": newID("in"), "object": "invoice", "customer": cus, "customer_email": email, "subscription": sub, "amount_due": amount,
			"amount_paid": amount, "currency": price["currency"], "billing_reason": "subscription_create", "status": "paid", "period_end": now + period,
			"charge": newID("ch"), "lines": obj{"data": []obj{{"price": public(price), "period": obj{"end": now + period}}}}, "livemode": live}
		d.objects[fmt.Sprint(inv["id"])] = inv
		out = append(out, f.emitLocked(d, "invoice.paid", inv, nil, live))
	}
	return out, nil
}

// Session returns a checkout session's bookkeeping: the price, trial and
// promotion code it was opened with.
func (f *Fake) Session(mode payments.Mode, id string) (price string, trialDays int, promo string, tax bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cs := f.modes[mode].objects[id]
	if cs == nil {
		return "", 0, "", false
	}
	price, _ = cs["_price"].(string)
	trialDays, _ = strconv.Atoi(fmt.Sprint(cs["_trial_days"]))
	promo, _ = cs["_promotion_code"].(string)
	tax = cs["_tax"] == "true"
	return
}

// Deliver posts an event to a webhook URL with a valid signature (or one
// signed at ts, for replays), and returns the answer's status.
func Deliver(ctx context.Context, client *http.Client, target, secret string, e Event, ts time.Time) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(e.Body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Stripe-Signature", stripe.SignatureHeader(secret, ts.Unix(), e.Body))
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	_ = resp.Body.Close()
	return resp.StatusCode, nil
}
