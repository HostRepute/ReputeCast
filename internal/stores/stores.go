// Package stores is the store-platform contract (design expansion phase
// 7, M30): what ReputeCast reads from a merchant's own Shopify or
// WooCommerce shop and how it is told about changes, on the pattern of the
// sending providers and the payment providers. Each adapter implements
// Client against its platform's API over plain HTTP, verifies and parses
// its webhooks (WebhookParser), and registers itself with Register; core
// code (internal/commerce) talks only to these interfaces. Every adapter
// passes the shared suite in internal/stores/contracttest against its fake.
//
// Never read or kept, whatever the platform sends: addresses, phone
// numbers, payment details and order notes (brief §3.6). The parsers drop
// them before anything else sees the payload.
package stores

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

// Capability is a badge on a store's catalog card and page (brief §3.6).
type Capability string

// Capabilities, in the order of the cards.
const (
	CapOrders      Capability = "orders"
	CapProducts    Capability = "products"
	CapCustomers   Capability = "customers"   // customers who accepted marketing
	CapCheckouts   Capability = "checkouts"   // abandoned checkouts
	CapWriteBack   Capability = "write_back"  // unsubscribes passed back
	CapAttribution Capability = "attribution" // orders credited to emails
)

// Descriptor describes an adapter.
type Descriptor struct {
	Key  string
	Name string
	// Capabilities with everything set up (WooCommerce's consent and
	// abandoned checkouts need the ReputeCast plugin).
	Capabilities []Capability
	// Topics are the webhooks ReputeCast registers, in the order of the
	// store page's table.
	Topics []string
	// Open connects to a store.
	Open func(cfg Config) (Client, error)
	// Webhooks verifies and parses deliveries.
	Webhooks WebhookParser
}

// Config opens a client on a store.
type Config struct {
	// Domain is the store's canonical host: acme-merch.myshopify.com, or
	// the WooCommerce site's host.
	Domain string
	// BaseURL overrides https://{Domain} (tests; the fakes).
	BaseURL string
	// HTTP overrides the client (the SSRF guard's in production).
	HTTP *http.Client
	// Token is a Shopify access token (write-only, sealed at rest).
	Token string
	// Key and Secret are a WooCommerce REST API key (sealed at rest).
	Key, Secret string
	// APIVersion overrides the adapter's pinned API version.
	APIVersion string
	// Now overrides the clock (tests).
	Now func() time.Time
	// RetryWait overrides the pause between retries of 5xx answers.
	RetryWait time.Duration
}

// Shop is what connecting reads about the store.
type Shop struct {
	Name string `json:"name"`
	// Domain is the canonical host (acme-merch.myshopify.com).
	Domain string `json:"domain"`
	// PublicDomain is the storefront's own domain (shop.acme.io).
	PublicDomain string `json:"publicDomain"`
	Currency     string `json:"currency"`
}

// Collection is a group of products (Shopify's collections, WooCommerce's
// product categories).
type Collection struct {
	ID       string
	Title    string
	Handle   string
	Products int
}

// Product is one product, with the price of its cheapest variant.
type Product struct {
	ID       string
	Title    string
	Handle   string
	URL      string
	ImageURL string
	ImageAlt string
	// Price and CompareAt are in minor units of Currency; CompareAt is 0
	// when the product isn't on sale.
	Price     int64
	CompareAt int64
	Currency  string
	// Available reports whether it can be bought; Inventory is the stock
	// when the store tracks it (nil otherwise).
	Available bool
	Inventory *int
	// Status: active, draft or archived (removed products arrive as a
	// deletion, never as a status).
	Status      string
	Collections []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Marketing consent states.
const (
	ConsentSubscribed    = "subscribed"
	ConsentNotSubscribed = "not_subscribed"
	ConsentUnsubscribed  = "unsubscribed"
	ConsentPending       = "pending"
)

// Consent is a customer's email-marketing consent in the store.
type Consent struct {
	// State: subscribed, not_subscribed, unsubscribed or pending.
	State string
	// Level: single_opt_in, confirmed_opt_in or unknown.
	Level     string
	UpdatedAt *time.Time
	// Text is the exact checkbox text, when the platform records it (the
	// ReputeCast plugin at a WooCommerce checkout).
	Text string
	// IP is the address the consent was given from, when recorded.
	IP string
}

// Accepted reports whether the customer accepted marketing.
func (c Consent) Accepted() bool { return c.State == ConsentSubscribed }

// Customer is a store customer. Only an email, a name and a country are
// read; never an address or a phone number.
type Customer struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
	Country   string
	Consent   Consent
	CreatedAt time.Time
	UpdatedAt time.Time
}

// LineItem is one line of an order or a checkout.
type LineItem struct {
	ProductID string `json:"productId"`
	VariantID string `json:"variantId,omitempty"`
	Title     string `json:"title"`
	Variant   string `json:"variant,omitempty"`
	Quantity  int    `json:"quantity"`
	// Price is the unit price, in minor units.
	Price    int64  `json:"price"`
	ImageURL string `json:"imageUrl,omitempty"`
}

// UTM are an order's landing page parameters.
type UTM struct {
	Source   string `json:"source,omitempty"`
	Medium   string `json:"medium,omitempty"`
	Campaign string `json:"campaign,omitempty"`
	Content  string `json:"content,omitempty"`
	Term     string `json:"term,omitempty"`
}

// Order is one order. Email is read to match the buyer to a contact and
// never stored.
type Order struct {
	ID     string
	Name   string // "#4409"
	Number string // "4409"
	Email  string
	// Customer is the buyer as the order carries them, with the consent
	// given at checkout.
	Customer         *Customer
	AcceptsMarketing bool
	PlacedAt         time.Time
	UpdatedAt        time.Time
	Currency         string
	// Subtotal, Total and Refunded are in minor units; Refunds lists each
	// refund (an order's refunds are counted once each, whichever event
	// brings them).
	Subtotal  int64
	Total     int64
	Refunded  int64
	Refunds   []Refund
	Discounts []string
	Items     []LineItem
	// LandingPage is the page the buyer arrived on (path and query, or a
	// full URL), with its UTM parameters and ReputeCast's click token.
	LandingPage string
	UTM         UTM
	ClickToken  string
	// CheckoutToken links the order to the checkout it completed.
	CheckoutToken string
	Cancelled     bool
	// Test orders (Shopify's test mode, Bogus gateway) are kept apart.
	Test bool
}

// Checkout is a checkout with an email address (a candidate abandoned
// checkout until it becomes an order).
type Checkout struct {
	ID               string
	Email            string
	Customer         *Customer
	AcceptsMarketing bool
	Items            []LineItem
	Subtotal         int64
	Total            int64
	Currency         string
	// RecoveryURL is the store's own link back to the checkout.
	RecoveryURL string
	ClickToken  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}

// Refund is money returned on an order.
type Refund struct {
	ID      string
	OrderID string
	Amount  int64
	At      time.Time
}

// Webhook is a webhook subscription in the store.
type Webhook struct {
	ID      string
	Topic   string
	Address string
}

// Page is one page of a list; Next is empty on the last page.
type Page[T any] struct {
	Items []T
	Next  string
}

// Range limits what a list returns.
type Range struct {
	// CreatedSince keeps what was created at or after it (the first
	// import's 3, 6 or 12 months).
	CreatedSince time.Time
	// UpdatedSince keeps what changed at or after it (the nightly
	// reconciliation and polling a missing webhook).
	UpdatedSince time.Time
}

// Client is a store reached with the workspace's own credentials.
type Client interface {
	// Shop reads the store's name, domains and currency.
	Shop(ctx context.Context) (Shop, error)
	// Count counts products, customers or orders in a range, for the
	// first import's progress.
	Count(ctx context.Context, kind string, r Range) (int, error)
	Products(ctx context.Context, cursor string, r Range) (Page[Product], error)
	Collections(ctx context.Context, cursor string) (Page[Collection], error)
	Customers(ctx context.Context, cursor string, r Range) (Page[Customer], error)
	Orders(ctx context.Context, cursor string, r Range) (Page[Order], error)
	// Checkouts lists checkouts with an email that didn't become orders
	// (ErrUnsupported where the platform has none of its own).
	Checkouts(ctx context.Context, cursor string, r Range) (Page[Checkout], error)
	Webhooks(ctx context.Context) ([]Webhook, error)
	Subscribe(ctx context.Context, topic, address, secret string) (Webhook, error)
	Unsubscribe(ctx context.Context, id string) error
	// SetConsent tells the store a customer unsubscribed (consent
	// write-back; ErrUnsupported or ErrPermission without it).
	SetConsent(ctx context.Context, customerID, state string, at time.Time) error
}

// Kinds Count accepts.
const (
	CountProducts  = "products"
	CountCustomers = "customers"
	CountOrders    = "orders"
)

// Event kinds, normalized across platforms.
const (
	EventOrder          = "order"
	EventOrderDeleted   = "order_deleted"
	EventRefund         = "refund"
	EventCustomer       = "customer"
	EventCheckout       = "checkout"
	EventProduct        = "product"
	EventProductDeleted = "product_deleted"
	EventUninstalled    = "uninstalled"
	// EventPing is WooCommerce's delivery check when a webhook is created.
	EventPing = "ping"
)

// Event is one verified webhook delivery, normalized.
type Event struct {
	Kind string
	// Topic is the platform's own topic ("orders/create", "order.updated").
	Topic string
	// DeliveryID identifies the delivery; a retry carries the same one.
	DeliveryID string
	// TriggeredAt is when the platform sent it (zero when unknown).
	TriggeredAt time.Time
	Order       *Order
	Customer    *Customer
	Checkout    *Checkout
	Product     *Product
	Refund      *Refund
	// DeletedID is a deleted product's or order's id.
	DeletedID string
}

// CurrencyHeader carries the store's currency into Parse, for payloads
// that don't name theirs (Shopify's products and refunds): the service sets
// it on the stored headers before parsing.
const CurrencyHeader = "X-ReputeCast-Store-Currency"

// WebhookParser verifies and parses one adapter's deliveries.
type WebhookParser interface {
	// Verify checks the signature against the store's secret.
	Verify(h http.Header, body []byte, secret string) error
	// Topic reads the delivery's topic and id from its headers.
	Topic(h http.Header) (topic, delivery string)
	// Parse normalizes a verified delivery.
	Parse(h http.Header, body []byte) (Event, error)
}

// Errors adapters return.
var (
	// ErrUnauthorized: the store refused the credentials (a revoked token,
	// an uninstalled app, a deleted key).
	ErrUnauthorized = errors.New("store: credentials refused")
	// ErrPermission: the credentials lack an access scope.
	ErrPermission = errors.New("store: missing access")
	// ErrUnavailable: the store can't be reached or answers 5xx.
	ErrUnavailable = errors.New("store: unavailable")
	// ErrNotFound: an object the store doesn't have.
	ErrNotFound = errors.New("store: not found")
	// ErrNotAStore: the address doesn't answer like a store of the platform.
	ErrNotAStore = errors.New("store: not a store of this platform")
	// ErrBadSignature: a webhook whose signature doesn't verify.
	ErrBadSignature = errors.New("store: webhook signature verification failed")
	// ErrUnsupported: the platform can't do it.
	ErrUnsupported = errors.New("store: not supported by the platform")
)

// RateLimited is a request the store throttled: wait RetryAfter.
type RateLimited struct {
	RetryAfter time.Duration
}

func (e *RateLimited) Error() string {
	return fmt.Sprintf("store: rate limited, retry after %s", e.RetryAfter)
}

// IsRateLimited reports whether err is a throttle, and how long to wait.
func IsRateLimited(err error) (time.Duration, bool) {
	var rl *RateLimited
	if errors.As(err, &rl) {
		return rl.RetryAfter, true
	}
	return 0, false
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Descriptor{}
	order      []string
)

// Register adds an adapter; adapters call it from their package's init.
func Register(d Descriptor) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, ok := registry[d.Key]; !ok {
		order = append(order, d.Key)
	}
	registry[d.Key] = d
}

// Lookup returns a registered adapter.
func Lookup(key string) (Descriptor, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	d, ok := registry[key]
	return d, ok
}

// Adapters lists the registered adapters in registration order.
func Adapters() []Descriptor {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Descriptor, 0, len(order))
	for _, k := range order {
		out = append(out, registry[k])
	}
	return out
}

// Has reports whether d has a capability.
func (d Descriptor) Has(c Capability) bool { return slices.Contains(d.Capabilities, c) }

// NormalizeHost turns what a person typed ("https://Shop.Acme.io/",
// "acme-merch.myshopify.com/admin") into a bare lowercase host.
func NormalizeHost(address string) string {
	a := strings.TrimSpace(strings.ToLower(address))
	a = strings.TrimPrefix(a, "https://")
	a = strings.TrimPrefix(a, "http://")
	if i := strings.IndexAny(a, "/?#"); i >= 0 {
		a = a[:i]
	}
	return strings.TrimSuffix(a, ".")
}
