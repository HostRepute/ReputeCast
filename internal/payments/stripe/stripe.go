// Package stripe is the Stripe payment adapter: plain HTTPS calls to
// Stripe's API with the workspace's restricted key (no SDK), the webhook's
// signature check, and the mapping of Stripe's objects onto the payments
// contract. Secret keys are refused before any call: they give full access
// to the account, payouts included.
//
// The API version is pinned (Stripe-Version), and the webhook endpoint is
// registered with the same version, so the objects ReputeCast reads keep
// their shape whatever the account's default version is.
package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/payments"
)

// Key is the adapter's key in the payments registry.
const Key = "stripe"

// APIVersion is the Stripe API version ReputeCast speaks.
const APIVersion = "2024-06-20"

// DefaultBase is Stripe's API.
const DefaultBase = "https://api.stripe.com"

// Tolerance is how old a webhook's signature may be; older requests are
// refused as replays.
const Tolerance = 5 * time.Minute

// Permissions are the 13 permissions the restricted key needs (brief §4),
// in the order of the table on the connect page.
var Permissions = []payments.Permission{
	{Resource: "customers", Access: "write"},
	{Resource: "checkout_sessions", Access: "write"},
	{Resource: "customer_portal", Access: "write"},
	{Resource: "products", Access: "write"},
	{Resource: "prices", Access: "write"},
	{Resource: "coupons", Access: "write"},
	{Resource: "promotion_codes", Access: "write"},
	{Resource: "subscriptions", Access: "write"},
	{Resource: "invoices", Access: "read"},
	{Resource: "charges", Access: "read"},
	{Resource: "webhook_endpoints", Access: "write"},
	{Resource: "events", Access: "read"},
	{Resource: "tax_settings", Access: "read"},
}

// Events are the 12 events the webhook subscribes to (brief §4).
var Events = []string{
	"checkout.session.completed",
	"customer.subscription.created",
	"customer.subscription.updated",
	"customer.subscription.deleted",
	"customer.subscription.trial_will_end",
	"invoice.paid",
	"invoice.payment_failed",
	"invoice.upcoming",
	"charge.refunded",
	"charge.dispute.created",
	"customer.updated",
	"customer.deleted",
}

// Prefixes of restricted keys per mode, and of the secret keys refused.
const (
	livePrefix       = "rk_live_"
	testPrefix       = "rk_test_"
	secretLivePrefix = "sk_live_"
	secretTestPrefix = "sk_test_"
)

func init() {
	payments.Register(payments.Descriptor{
		Key: Key, Name: "Stripe", Available: true,
		Capabilities: []payments.Capability{payments.CapSubscriptions, payments.CapOneTime, payments.CapTrials, payments.CapCoupons,
			payments.CapPromotionCodes, payments.CapPortal, payments.CapTax, payments.CapTestMode},
		Permissions:   Permissions,
		WebhookEvents: Events,
		KeyPrefixes:   map[payments.Mode]string{payments.Live: livePrefix, payments.Test: testPrefix},
		Open:          func(cfg payments.Config) (payments.Client, error) { return New(cfg) },
		Hint:          Hint,
		Dashboard:     Dashboard,
	})
	payments.RegisterParser(Key, Verifier{})
}

// ModeOf checks a key's format: a restricted key's mode, ErrSecretKey for a
// secret key and ErrKeyFormat for anything else.
func ModeOf(key string) (payments.Mode, error) {
	key = strings.TrimSpace(key)
	switch {
	case strings.HasPrefix(key, secretLivePrefix), strings.HasPrefix(key, secretTestPrefix):
		return "", payments.ErrSecretKey
	case strings.HasPrefix(key, livePrefix) && len(key) > len(livePrefix)+8:
		return payments.Live, nil
	case strings.HasPrefix(key, testPrefix) && len(key) > len(testPrefix)+8:
		return payments.Test, nil
	}
	return "", payments.ErrKeyFormat
}

// Hint is the masked form of a key: its prefix and last four characters
// ("rk_live_…9fQ2").
func Hint(key string) string {
	key = strings.TrimSpace(key)
	prefix := livePrefix
	if strings.HasPrefix(key, testPrefix) {
		prefix = testPrefix
	}
	if len(key) < 4 {
		return prefix + "…"
	}
	return prefix + "…" + key[len(key)-4:]
}

// Client is a Stripe account reached with a restricted key.
type Client struct {
	key  string
	mode payments.Mode
	base string
	hc   *http.Client
	now  func() time.Time
	// retryWait is the pause before retrying a 5xx (tests shorten it).
	retryWait time.Duration
}

// New opens a client. The key must be a restricted key.
func New(cfg payments.Config) (*Client, error) {
	mode, err := ModeOf(cfg.Key)
	if err != nil {
		return nil, err
	}
	c := &Client{key: strings.TrimSpace(cfg.Key), mode: mode, base: DefaultBase, hc: cfg.HTTP, now: cfg.Now, retryWait: 400 * time.Millisecond}
	if cfg.BaseURL != "" {
		c.base = strings.TrimRight(cfg.BaseURL, "/")
		c.retryWait = 10 * time.Millisecond
	}
	if c.hc == nil {
		c.hc = &http.Client{Timeout: 20 * time.Second}
	}
	if c.now == nil {
		c.now = time.Now
	}
	return c, nil
}

// Mode implements payments.Client.
func (c *Client) Mode() payments.Mode { return c.mode }

// call sends one request. Form values follow Stripe's bracket notation;
// POSTs that create something carry an idempotency key so a retry never
// creates it twice. 5xx answers and network errors are retried twice.
func (c *Client) call(ctx context.Context, method, path string, form url.Values, out any) error {
	idem := ""
	if method == http.MethodPost {
		idem = randomID()
	}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryWait * time.Duration(attempt)):
			}
		}
		err := c.once(ctx, method, path, form, idem, out)
		if err == nil {
			return nil
		}
		last = err
		if !errors.Is(err, payments.ErrUnavailable) {
			return err
		}
	}
	return last
}

func (c *Client) once(ctx context.Context, method, path string, form url.Values, idem string, out any) error {
	target := c.base + path
	var body io.Reader
	if method == http.MethodGet || method == http.MethodDelete {
		if len(form) > 0 {
			target += "?" + form.Encode()
		}
	} else if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Stripe-Version", APIVersion)
	req.Header.Set("User-Agent", "ReputeCast")
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if idem != "" {
		req.Header.Set("Idempotency-Key", idem)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		// Any failure to reach Stripe (DNS, connection, timeout) is an outage.
		return fmt.Errorf("%w: %w", payments.ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("%w: %w", payments.ErrUnavailable, err)
	}
	if resp.StatusCode >= 300 {
		var e struct {
			Error struct {
				Type    string `json:"type"`
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		msg := e.Error.Message
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return &payments.APIError{Status: resp.StatusCode, Type: e.Error.Type, Code: e.Error.Code, Message: msg}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("stripe: unreadable answer to %s %s: %w", method, path, err)
	}
	return nil
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// probes are the harmless calls proving each permission: a write is an
// update of an object that doesn't exist (Stripe checks the permission
// before it looks the object up, so 404 means allowed), a read lists one
// object.
var probes = map[string]struct{ method, path string }{
	"customers":         {http.MethodPost, "/v1/customers/cus_reputecast_check"},
	"checkout_sessions": {http.MethodPost, "/v1/checkout/sessions/cs_reputecast_check/expire"},
	"customer_portal":   {http.MethodPost, "/v1/billing_portal/configurations/bpc_reputecast_check"},
	"products":          {http.MethodPost, "/v1/products/prod_reputecast_check"},
	"prices":            {http.MethodPost, "/v1/prices/price_reputecast_check"},
	"coupons":           {http.MethodDelete, "/v1/coupons/reputecast_check"},
	"promotion_codes":   {http.MethodPost, "/v1/promotion_codes/promo_reputecast_check"},
	"subscriptions":     {http.MethodPost, "/v1/subscriptions/sub_reputecast_check"},
	"invoices":          {http.MethodGet, "/v1/invoices?limit=1"},
	"charges":           {http.MethodGet, "/v1/charges?limit=1"},
	"webhook_endpoints": {http.MethodPost, "/v1/webhook_endpoints/we_reputecast_check"},
	"events":            {http.MethodGet, "/v1/events?limit=1"},
	"tax_settings":      {http.MethodGet, "/v1/tax/settings"},
}

// CheckKey implements payments.Client.
func (c *Client) CheckKey(ctx context.Context) (payments.KeyCheck, error) {
	out := payments.KeyCheck{Mode: c.mode, Permissions: make([]payments.PermissionCheck, len(Permissions))}
	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup
	for i, p := range Permissions {
		wg.Add(1)
		go func(i int, p payments.Permission) {
			defer wg.Done()
			pr := probes[p.Resource]
			path, query, _ := strings.Cut(pr.path, "?")
			form, _ := url.ParseQuery(query)
			var body map[string]any
			err := c.once(ctx, pr.method, path, form, "", &body)
			ok := err == nil || errors.Is(err, payments.ErrNotFound)
			if !ok && !errors.Is(err, payments.ErrPermission) {
				var ae *payments.APIError
				// A 400 (bad request) also passed the permission check.
				if errors.As(err, &ae) && ae.Status == http.StatusBadRequest {
					ok = true
				}
			}
			mu.Lock()
			defer mu.Unlock()
			out.Permissions[i] = payments.PermissionCheck{Permission: p, OK: ok}
			if p.Resource == "tax_settings" && err == nil {
				if s, _ := body["status"].(string); s == "active" {
					out.Account.TaxEnabled = true
				}
			}
			if err != nil && (errors.Is(err, payments.ErrKeyRejected) || errors.Is(err, payments.ErrUnavailable)) && firstErr == nil {
				firstErr = err
			}
		}(i, p)
	}
	wg.Wait()
	if firstErr != nil {
		return payments.KeyCheck{}, firstErr
	}
	var acct struct {
		ID              string `json:"id"`
		Country         string `json:"country"`
		DefaultCurrency string `json:"default_currency"`
		Settings        struct {
			Dashboard struct {
				DisplayName string `json:"display_name"`
			} `json:"dashboard"`
		} `json:"settings"`
		BusinessProfile struct {
			Name string `json:"name"`
		} `json:"business_profile"`
	}
	if err := c.call(ctx, http.MethodGet, "/v1/account", nil, &acct); err != nil {
		if errors.Is(err, payments.ErrKeyRejected) || errors.Is(err, payments.ErrUnavailable) {
			return payments.KeyCheck{}, err
		}
	}
	tax := out.Account.TaxEnabled
	out.Account = payments.Account{ID: acct.ID, Country: acct.Country, Currency: strings.ToLower(acct.DefaultCurrency), TaxEnabled: tax,
		Name: firstNonEmpty(acct.Settings.Dashboard.DisplayName, acct.BusinessProfile.Name)}
	return out, nil
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// RegisterWebhook implements payments.Client.
func (c *Client) RegisterWebhook(ctx context.Context, endpoint string, events []string) (payments.Webhook, error) {
	f := url.Values{}
	f.Set("url", endpoint)
	f.Set("api_version", APIVersion)
	f.Set("description", "ReputeCast: paid subscriptions")
	for _, e := range events {
		f.Add("enabled_events[]", e)
	}
	var w struct {
		ID            string   `json:"id"`
		URL           string   `json:"url"`
		Secret        string   `json:"secret"`
		EnabledEvents []string `json:"enabled_events"`
	}
	if err := c.call(ctx, http.MethodPost, "/v1/webhook_endpoints", f, &w); err != nil {
		return payments.Webhook{}, err
	}
	return payments.Webhook{ID: w.ID, URL: w.URL, Events: w.EnabledEvents, Secret: w.Secret}, nil
}

// WebhookExists implements payments.Client.
func (c *Client) WebhookExists(ctx context.Context, id string) (bool, error) {
	var w struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	err := c.call(ctx, http.MethodGet, "/v1/webhook_endpoints/"+url.PathEscape(id), nil, &w)
	if errors.Is(err, payments.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return w.Status != "disabled", nil
}

// DeleteWebhook implements payments.Client.
func (c *Client) DeleteWebhook(ctx context.Context, id string) error {
	err := c.call(ctx, http.MethodDelete, "/v1/webhook_endpoints/"+url.PathEscape(id), nil, nil)
	if errors.Is(err, payments.ErrNotFound) {
		return nil
	}
	return err
}

// ListEvents implements payments.Client: Stripe lists newest first, so the
// pages are read to the end and returned oldest first.
func (c *Client) ListEvents(ctx context.Context, since time.Time, types []string) ([]payments.Event, error) {
	var all []payments.Event
	after := ""
	for page := 0; page < 50; page++ {
		f := url.Values{}
		f.Set("limit", "100")
		f.Set("created[gte]", strconv.FormatInt(since.Unix(), 10))
		for _, t := range types {
			f.Add("types[]", t)
		}
		if after != "" {
			f.Set("starting_after", after)
		}
		var list struct {
			Data    []json.RawMessage `json:"data"`
			HasMore bool              `json:"has_more"`
		}
		if err := c.call(ctx, http.MethodGet, "/v1/events", f, &list); err != nil {
			return nil, err
		}
		for _, raw := range list.Data {
			ev, err := parseEvent(raw)
			if err != nil {
				return nil, err
			}
			all = append(all, ev)
			after = ev.ID
		}
		if !list.HasMore || len(list.Data) == 0 {
			break
		}
	}
	slices.SortStableFunc(all, func(a, b payments.Event) int { return a.Created.Compare(b.Created) })
	return all, nil
}

// ConfigurePortal implements payments.Client: it creates the portal
// configuration (empty id) or updates it.
func (c *Client) ConfigurePortal(ctx context.Context, id string, pc payments.PortalConfig) (string, error) {
	f := url.Values{}
	f.Set("features[payment_method_update][enabled]", strconv.FormatBool(pc.UpdateCard))
	f.Set("features[invoice_history][enabled]", strconv.FormatBool(pc.Invoices))
	f.Set("features[customer_update][enabled]", strconv.FormatBool(pc.BillingEmail))
	if pc.BillingEmail {
		f.Add("features[customer_update][allowed_updates][]", "email")
	}
	f.Set("features[subscription_cancel][enabled]", strconv.FormatBool(pc.Cancel))
	if pc.Cancel {
		f.Set("features[subscription_cancel][mode]", "at_period_end")
	}
	switchOn := pc.SwitchPeriod && len(pc.Products) > 0
	f.Set("features[subscription_update][enabled]", strconv.FormatBool(switchOn))
	if switchOn {
		f.Add("features[subscription_update][default_allowed_updates][]", "price")
		f.Set("features[subscription_update][proration_behavior]", "create_prorations")
		i := 0
		products := make([]string, 0, len(pc.Products))
		for p := range pc.Products {
			products = append(products, p)
		}
		slices.Sort(products)
		for _, p := range products {
			f.Set(fmt.Sprintf("features[subscription_update][products][%d][product]", i), p)
			for _, price := range pc.Products[p] {
				f.Add(fmt.Sprintf("features[subscription_update][products][%d][prices][]", i), price)
			}
			i++
		}
	}
	if pc.ReturnURL != "" {
		f.Set("default_return_url", pc.ReturnURL)
	}
	if pc.TermsURL != "" {
		f.Set("business_profile[terms_of_service_url]", pc.TermsURL)
	}
	if pc.PrivacyURL != "" {
		f.Set("business_profile[privacy_policy_url]", pc.PrivacyURL)
	}
	path := "/v1/billing_portal/configurations"
	if id != "" {
		path += "/" + url.PathEscape(id)
	}
	var out struct {
		ID string `json:"id"`
	}
	err := c.call(ctx, http.MethodPost, path, f, &out)
	if errors.Is(err, payments.ErrNotFound) && id != "" {
		return c.ConfigurePortal(ctx, "", pc)
	}
	return out.ID, err
}

// PortalSession implements payments.Client.
func (c *Client) PortalSession(ctx context.Context, customer string, flow payments.PortalFlow) (string, error) {
	f := url.Values{}
	f.Set("customer", customer)
	if flow.ReturnURL != "" {
		f.Set("return_url", flow.ReturnURL)
	}
	if flow.Cancel && flow.SubscriptionID != "" {
		f.Set("flow_data[type]", "subscription_cancel")
		f.Set("flow_data[subscription_cancel][subscription]", flow.SubscriptionID)
		if flow.ReturnURL != "" {
			f.Set("flow_data[after_completion][type]", "redirect")
			f.Set("flow_data[after_completion][redirect][return_url]", flow.ReturnURL)
		}
	}
	var out struct {
		URL string `json:"url"`
	}
	if err := c.call(ctx, http.MethodPost, "/v1/billing_portal/sessions", f, &out); err != nil {
		return "", err
	}
	return out.URL, nil
}

// CreateCheckout implements payments.Client.
func (c *Client) CreateCheckout(ctx context.Context, r payments.CheckoutRequest) (payments.CheckoutSession, error) {
	f := url.Values{}
	f.Set("mode", "subscription")
	f.Set("line_items[0][price]", r.PriceID)
	f.Set("line_items[0][quantity]", "1")
	f.Set("success_url", r.SuccessURL)
	f.Set("cancel_url", r.CancelURL)
	switch {
	case r.CustomerID != "":
		f.Set("customer", r.CustomerID)
		if r.Tax {
			f.Set("customer_update[address]", "auto")
		}
	case r.CustomerEmail != "":
		f.Set("customer_email", r.CustomerEmail)
	}
	if r.TrialDays > 0 {
		f.Set("subscription_data[trial_period_days]", strconv.Itoa(r.TrialDays))
		f.Set("payment_method_collection", "always")
		if r.TrialEndCancel {
			f.Set("subscription_data[trial_settings][end_behavior][missing_payment_method]", "cancel")
		}
	}
	if r.Tax {
		f.Set("automatic_tax[enabled]", "true")
	}
	switch {
	case r.PromotionCode != "":
		f.Set("discounts[0][promotion_code]", r.PromotionCode)
	case r.AllowCodes:
		f.Set("allow_promotion_codes", "true")
	}
	if r.ClientRef != "" {
		f.Set("client_reference_id", r.ClientRef)
	}
	keys := make([]string, 0, len(r.Metadata))
	for k := range r.Metadata {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		f.Set("metadata["+k+"]", r.Metadata[k])
		f.Set("subscription_data[metadata]["+k+"]", r.Metadata[k])
	}
	var raw json.RawMessage
	if err := c.call(ctx, http.MethodPost, "/v1/checkout/sessions", f, &raw); err != nil {
		return payments.CheckoutSession{}, err
	}
	return parseCheckout(raw)
}

// GetCheckout implements payments.Client.
func (c *Client) GetCheckout(ctx context.Context, id string) (payments.CheckoutSession, error) {
	var raw json.RawMessage
	if err := c.call(ctx, http.MethodGet, "/v1/checkout/sessions/"+url.PathEscape(id), nil, &raw); err != nil {
		return payments.CheckoutSession{}, err
	}
	return parseCheckout(raw)
}

// SaveProduct implements payments.Client.
func (c *Client) SaveProduct(ctx context.Context, p payments.Product) (payments.Product, error) {
	f := url.Values{}
	f.Set("name", p.Name)
	if p.Description != "" {
		f.Set("description", p.Description)
	} else if p.ID != "" {
		f.Set("description", "")
	}
	f.Set("active", strconv.FormatBool(p.Active))
	for k, v := range p.Metadata {
		f.Set("metadata["+k+"]", v)
	}
	path := "/v1/products"
	if p.ID != "" {
		path += "/" + url.PathEscape(p.ID)
	}
	var out struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Active      bool   `json:"active"`
	}
	if err := c.call(ctx, http.MethodPost, path, f, &out); err != nil {
		return payments.Product{}, err
	}
	return payments.Product{ID: out.ID, Name: out.Name, Description: out.Description, Active: out.Active, Metadata: p.Metadata}, nil
}

// CreatePrice implements payments.Client.
func (c *Client) CreatePrice(ctx context.Context, p payments.Price) (payments.Price, error) {
	f := url.Values{}
	f.Set("product", p.ProductID)
	f.Set("unit_amount", strconv.FormatInt(p.AmountCents, 10))
	f.Set("currency", strings.ToLower(p.Currency))
	if p.Interval != "" {
		f.Set("recurring[interval]", p.Interval)
	}
	if p.TaxInclusive {
		f.Set("tax_behavior", "inclusive")
	} else {
		f.Set("tax_behavior", "exclusive")
	}
	for k, v := range p.Metadata {
		f.Set("metadata["+k+"]", v)
	}
	var raw json.RawMessage
	if err := c.call(ctx, http.MethodPost, "/v1/prices", f, &raw); err != nil {
		return payments.Price{}, err
	}
	return parsePrice(raw)
}

// RetirePrice implements payments.Client.
func (c *Client) RetirePrice(ctx context.Context, id string) error {
	f := url.Values{}
	f.Set("active", "false")
	return c.call(ctx, http.MethodPost, "/v1/prices/"+url.PathEscape(id), f, nil)
}

// GetPrice implements payments.Client.
func (c *Client) GetPrice(ctx context.Context, id string) (payments.Price, error) {
	var raw json.RawMessage
	if err := c.call(ctx, http.MethodGet, "/v1/prices/"+url.PathEscape(id), nil, &raw); err != nil {
		return payments.Price{}, err
	}
	return parsePrice(raw)
}

// GetSubscription implements payments.Client.
func (c *Client) GetSubscription(ctx context.Context, id string) (payments.Subscription, error) {
	var raw json.RawMessage
	if err := c.call(ctx, http.MethodGet, "/v1/subscriptions/"+url.PathEscape(id), nil, &raw); err != nil {
		return payments.Subscription{}, err
	}
	return parseSubscription(raw)
}

// GetCharge implements payments.Client.
func (c *Client) GetCharge(ctx context.Context, id string) (payments.Charge, error) {
	var raw json.RawMessage
	if err := c.call(ctx, http.MethodGet, "/v1/charges/"+url.PathEscape(id), nil, &raw); err != nil {
		return payments.Charge{}, err
	}
	return parseCharge(raw, nil)
}

// GetCustomer implements payments.Client.
func (c *Client) GetCustomer(ctx context.Context, id string) (payments.Customer, error) {
	var raw json.RawMessage
	if err := c.call(ctx, http.MethodGet, "/v1/customers/"+url.PathEscape(id), nil, &raw); err != nil {
		return payments.Customer{}, err
	}
	return parseCustomer(raw)
}

// CancelSubscription implements payments.Client.
func (c *Client) CancelSubscription(ctx context.Context, id string, atPeriodEnd bool) (payments.Subscription, error) {
	var raw json.RawMessage
	var err error
	if atPeriodEnd {
		f := url.Values{}
		f.Set("cancel_at_period_end", "true")
		err = c.call(ctx, http.MethodPost, "/v1/subscriptions/"+url.PathEscape(id), f, &raw)
	} else {
		err = c.call(ctx, http.MethodDelete, "/v1/subscriptions/"+url.PathEscape(id), nil, &raw)
	}
	if err != nil {
		return payments.Subscription{}, err
	}
	return parseSubscription(raw)
}

// ListSubscriptions implements payments.Client (every status).
func (c *Client) ListSubscriptions(ctx context.Context, cursor string) (payments.Page[payments.Subscription], error) {
	f := url.Values{}
	f.Set("limit", "100")
	f.Set("status", "all")
	if cursor != "" {
		f.Set("starting_after", cursor)
	}
	var list struct {
		Data    []json.RawMessage `json:"data"`
		HasMore bool              `json:"has_more"`
	}
	if err := c.call(ctx, http.MethodGet, "/v1/subscriptions", f, &list); err != nil {
		return payments.Page[payments.Subscription]{}, err
	}
	out := payments.Page[payments.Subscription]{Items: make([]payments.Subscription, 0, len(list.Data))}
	for _, raw := range list.Data {
		s, err := parseSubscription(raw)
		if err != nil {
			return out, err
		}
		out.Items = append(out.Items, s)
	}
	if list.HasMore && len(out.Items) > 0 {
		out.Next = out.Items[len(out.Items)-1].ID
	}
	return out, nil
}

// ListCustomers implements payments.Client.
func (c *Client) ListCustomers(ctx context.Context, cursor string) (payments.Page[payments.Customer], error) {
	f := url.Values{}
	f.Set("limit", "100")
	if cursor != "" {
		f.Set("starting_after", cursor)
	}
	var list struct {
		Data    []json.RawMessage `json:"data"`
		HasMore bool              `json:"has_more"`
	}
	if err := c.call(ctx, http.MethodGet, "/v1/customers", f, &list); err != nil {
		return payments.Page[payments.Customer]{}, err
	}
	out := payments.Page[payments.Customer]{Items: make([]payments.Customer, 0, len(list.Data))}
	for _, raw := range list.Data {
		cu, err := parseCustomer(raw)
		if err != nil {
			return out, err
		}
		out.Items = append(out.Items, cu)
	}
	if list.HasMore && len(out.Items) > 0 {
		out.Next = out.Items[len(out.Items)-1].ID
	}
	return out, nil
}

// DashboardURL implements payments.Client.
func (c *Client) DashboardURL(kind, id string) string { return Dashboard(c.mode, kind, id) }

// Dashboard links to an object in Stripe's dashboard: customers,
// subscriptions, products, prices, webhooks, events, payments, or the
// API keys page ("apikeys") and the email settings ("emails").
func Dashboard(mode payments.Mode, kind, id string) string {
	base := "https://dashboard.stripe.com"
	if mode == payments.Test {
		base += "/test"
	}
	switch kind {
	case "apikeys":
		return base + "/apikeys"
	case "apikeys_create":
		return base + "/apikeys/create"
	case "emails":
		return "https://dashboard.stripe.com/settings/emails"
	case "webhooks":
		if id == "" {
			return base + "/webhooks"
		}
	}
	if id == "" {
		return base
	}
	return base + "/" + kind + "/" + url.PathEscape(id)
}

// Verifier checks Stripe's webhook signatures.
type Verifier struct{}

// ParseWebhook implements payments.WebhookVerifier: Stripe-Signature has
// the timestamp (t) and one or more HMAC-SHA256 signatures (v1) of
// "t.body"; any v1 matching the secret passes, within Tolerance.
func (Verifier) ParseWebhook(header http.Header, body []byte, secret string, now time.Time) (payments.Event, error) {
	if err := Verify(header.Get("Stripe-Signature"), body, secret, now); err != nil {
		return payments.Event{}, err
	}
	return parseEvent(body)
}

// ParseEvent implements payments.WebhookVerifier.
func (Verifier) ParseEvent(body []byte) (payments.Event, error) { return parseEvent(body) }

// Verify checks one Stripe-Signature header.
func Verify(sig string, body []byte, secret string, now time.Time) error {
	if secret == "" || sig == "" {
		return payments.ErrBadSignature
	}
	var ts int64
	var v1 []string
	for _, part := range strings.Split(sig, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return payments.ErrBadSignature
			}
			ts = n
		case "v1":
			v1 = append(v1, v)
		}
	}
	if ts == 0 || len(v1) == 0 {
		return payments.ErrBadSignature
	}
	if now.Sub(time.Unix(ts, 0)).Abs() > Tolerance {
		return payments.ErrBadSignature
	}
	want := Sign(secret, ts, body)
	for _, s := range v1 {
		if hmac.Equal([]byte(s), []byte(want)) {
			return nil
		}
	}
	return payments.ErrBadSignature
}

// Sign is the v1 signature of a payload at time ts.
func Sign(secret string, ts int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%d.", ts)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// SignatureHeader is a Stripe-Signature header for body at ts (tests and
// the demo's in-process Stripe).
func SignatureHeader(secret string, ts int64, body []byte) string {
	return fmt.Sprintf("t=%d,v1=%s", ts, Sign(secret, ts, body))
}
