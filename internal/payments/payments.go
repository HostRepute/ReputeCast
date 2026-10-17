// Package payments is the pluggable payment-provider contract (design
// expansion phase 6, M26), on the sending-provider pattern: a catalog of
// adapters with capability badges, one connection pattern (a write-only
// key, checked before it is used), health, and the shared contract-test
// suite in internal/payments/contracttest that every adapter passes.
//
// Readers pay the publisher through the publisher's own account: an adapter
// talks to that account with the workspace's own key, nothing sits in
// between, and ReputeCast never sees or stores card details. Adapters live
// in internal/payments/<name>; core code (internal/earn) talks to these
// interfaces and finds adapters through the registry. Stripe comes first;
// Paddle and Lemon Squeezy are listed as not available yet.
package payments

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"
)

// Mode is a connection's mode: live payments or the provider's test mode,
// whose payments use test cards and are kept apart from everything live.
type Mode string

// Modes.
const (
	Live Mode = "live"
	Test Mode = "test"
)

// Capability is a badge on a provider's catalog card.
type Capability string

// Capabilities, in the order of the cards.
const (
	CapSubscriptions    Capability = "subscriptions"
	CapOneTime          Capability = "one_time"
	CapTrials           Capability = "trials"
	CapCoupons          Capability = "coupons"
	CapPromotionCodes   Capability = "promotion_codes"
	CapPortal           Capability = "portal"
	CapTax              Capability = "tax"
	CapMerchantOfRecord Capability = "merchant_of_record"
	CapTestMode         Capability = "test_mode"
)

// Permission is one access a restricted key must have, with a harmless
// call that proves it.
type Permission struct {
	// Resource names it ("customers", "checkout_sessions"); the app words it.
	Resource string `json:"resource"`
	Access   string `json:"access" enum:"read,write"`
}

// Descriptor describes an adapter in the catalog.
type Descriptor struct {
	Key  string
	Name string
	// Available is false for providers listed as "Not available yet".
	Available bool
	// MerchantOfRecord providers sell to readers on the publisher's behalf
	// and handle sales tax and VAT.
	MerchantOfRecord bool
	Capabilities     []Capability
	// Permissions the restricted key needs, in the table's order.
	Permissions []Permission
	// WebhookEvents are the events ReputeCast subscribes to.
	WebhookEvents []string
	// KeyPrefixes are the restricted keys' prefixes per mode ("rk_live_").
	KeyPrefixes map[Mode]string
	// Open connects to an account. Nil when the provider isn't available.
	Open func(cfg Config) (Client, error)
}

// Config opens a client on an account.
type Config struct {
	// Key is the restricted key (write-only: never logged or returned).
	Key string
	// BaseURL overrides the provider's API address (tests and the demo).
	BaseURL string
	// HTTP overrides the client (the demo's in-process provider).
	HTTP *http.Client
	// Now overrides the clock (tests).
	Now func() time.Time
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

// Catalog lists the adapters: available ones first, then in registration order.
func Catalog() []Descriptor {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Descriptor, 0, len(order))
	for _, k := range order {
		out = append(out, registry[k])
	}
	slices.SortStableFunc(out, func(a, b Descriptor) int {
		switch {
		case a.Available == b.Available:
			return 0
		case a.Available:
			return -1
		}
		return 1
	})
	return out
}

// Errors adapters return.
var (
	// ErrSecretKey refuses an unrestricted secret key: it gives full access
	// to the account, payouts included.
	ErrSecretKey = errors.New("payments: secret keys are refused; create a restricted key")
	// ErrKeyFormat is a key that isn't a restricted key of this provider.
	ErrKeyFormat = errors.New("payments: not a restricted key")
	// ErrKeyRejected means the provider refused the key (rolled, deleted or cut short).
	ErrKeyRejected = errors.New("payments: the provider rejected the key")
	// ErrBadSignature is a webhook whose signature doesn't verify, or one
	// signed too long ago (a replay).
	ErrBadSignature = errors.New("payments: webhook signature verification failed")
	// ErrUnavailable means the provider can't be reached or answers 5xx.
	ErrUnavailable = errors.New("payments: provider unavailable")
	// ErrNotFound is an object the provider doesn't have.
	ErrNotFound = errors.New("payments: not found")
	// ErrPermission is a call the key isn't allowed to make.
	ErrPermission = errors.New("payments: the key lacks a permission")
)

// APIError is a provider's error answer.
type APIError struct {
	Status  int
	Type    string
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%d %s: %s", e.Status, e.Code, e.Message)
	}
	return fmt.Sprintf("%d: %s", e.Status, e.Message)
}

// Is maps an API error to the package's sentinel errors.
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrKeyRejected:
		return e.Status == http.StatusUnauthorized
	case ErrPermission:
		return e.Status == http.StatusForbidden
	case ErrNotFound:
		return e.Status == http.StatusNotFound
	case ErrUnavailable:
		return e.Status >= 500 || e.Status == http.StatusTooManyRequests
	}
	return false
}

// Account is what checking a key reads about the account.
type Account struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Country  string `json:"country"`
	Currency string `json:"currency"`
	// TaxEnabled reports whether the provider calculates tax (Stripe Tax).
	TaxEnabled bool `json:"taxEnabled"`
}

// PermissionCheck is one permission's result.
type PermissionCheck struct {
	Permission
	OK bool `json:"ok"`
}

// KeyCheck is the result of checking a key.
type KeyCheck struct {
	Mode        Mode
	Account     Account
	Permissions []PermissionCheck
}

// Missing returns the permissions the key lacks.
func (k KeyCheck) Missing() []Permission {
	out := []Permission{}
	for _, p := range k.Permissions {
		if !p.OK {
			out = append(out, p.Permission)
		}
	}
	return out
}

// Webhook is an endpoint registered at the provider.
type Webhook struct {
	ID     string
	URL    string
	Events []string
	// Secret signs the provider's requests (write-only).
	Secret string
}

// PortalConfig is what the customer portal lets readers do.
type PortalConfig struct {
	UpdateCard   bool
	Invoices     bool
	BillingEmail bool
	SwitchPeriod bool
	Cancel       bool
	// Prices readers may switch between (product → prices).
	Products map[string][]string
	// ReturnURL is where the portal's "Return" link leads.
	ReturnURL string
	// TermsURL and PrivacyURL show in the portal's footer.
	TermsURL   string
	PrivacyURL string
}

// Product is a tier as the provider stores it.
type Product struct {
	ID          string
	Name        string
	Description string
	Active      bool
	Metadata    map[string]string
}

// Price is an amount per period.
type Price struct {
	ID          string
	ProductID   string
	AmountCents int64
	Currency    string
	// Interval is month or year; empty for a one-time price.
	Interval string
	Active   bool
	// TaxInclusive marks a price that includes tax.
	TaxInclusive bool
	Metadata     map[string]string
}

// Discount is a coupon applied to a subscription.
type Discount struct {
	CouponID   string     `json:"couponId,omitempty"`
	Name       string     `json:"name,omitempty"`
	PromoCode  string     `json:"promoCode,omitempty"`
	PercentOff float64    `json:"percentOff,omitempty"`
	AmountOff  int64      `json:"amountOff,omitempty"`
	Duration   string     `json:"duration,omitempty"` // once, repeating, forever
	Months     int        `json:"months,omitempty"`
	Start      *time.Time `json:"start,omitempty"`
	End        *time.Time `json:"end,omitempty"`
}

// Subscription is a reader's subscription, as the provider reports it.
type Subscription struct {
	ID         string
	CustomerID string
	// Status is the provider's: trialing, active, past_due, canceled,
	// unpaid, incomplete, incomplete_expired, paused.
	Status            string
	PriceID           string
	ProductID         string
	Interval          string
	AmountCents       int64
	Currency          string
	StartedAt         time.Time
	TrialEnd          *time.Time
	CurrentPeriodEnd  *time.Time
	CancelAt          *time.Time
	CancelAtPeriodEnd bool
	CanceledAt        *time.Time
	EndedAt           *time.Time
	Discount          *Discount
	Metadata          map[string]string
	Livemode          bool
}

// Customer is a reader as the provider knows them.
type Customer struct {
	ID       string
	Email    string
	Name     string
	Created  time.Time
	Deleted  bool
	Livemode bool
}

// Invoice is a payment, a failed payment or an upcoming renewal.
type Invoice struct {
	ID             string
	CustomerID     string
	CustomerEmail  string
	SubscriptionID string
	PriceID        string
	AmountDue      int64
	AmountPaid     int64
	Currency       string
	// BillingReason: subscription_create, subscription_cycle, subscription_update…
	BillingReason string
	PeriodEnd     *time.Time
	// A failure: the provider's reason ("Your card has insufficient funds.")
	// and when it tries again.
	FailureMessage   string
	NextAttempt      *time.Time
	AttemptCount     int
	DiscountedAmount int64
	ChargeID         string
}

// Charge is a payment's charge, for refunds and disputes.
type Charge struct {
	ID             string
	CustomerID     string
	InvoiceID      string
	Amount         int64
	AmountRefunded int64
	Currency       string
	// The latest refund's amount (the event's delta).
	RefundAmount int64
	// FailureMessage is the card issuer's reason for a failed charge.
	FailureMessage string
}

// CheckoutSession is a hosted checkout.
type CheckoutSession struct {
	ID             string
	URL            string
	Status         string // open, complete, expired
	PaymentStatus  string
	CustomerID     string
	CustomerEmail  string
	SubscriptionID string
	ClientRef      string
	Metadata       map[string]string
	Livemode       bool
}

// CheckoutRequest opens a subscription checkout.
type CheckoutRequest struct {
	PriceID       string
	CustomerID    string
	CustomerEmail string
	TrialDays     int
	// Tax adds tax at checkout and collects the address it needs.
	Tax bool
	// PromotionCode applies an offer; AllowCodes lets readers type one.
	PromotionCode string
	AllowCodes    bool
	SuccessURL    string
	CancelURL     string
	ClientRef     string
	Metadata      map[string]string
	// TrialEndCancel ends access when a trial ends without a card on file.
	TrialEndCancel bool
}

// PortalFlow opens the portal on one step (the cancel step of a subscription).
type PortalFlow struct {
	Cancel         bool
	SubscriptionID string
	ReturnURL      string
}

// EventKind is what an event means, whatever the provider calls it.
type EventKind string

// Event kinds.
const (
	EventCheckoutCompleted   EventKind = "checkout_completed"
	EventSubscriptionCreated EventKind = "subscription_created"
	EventSubscriptionUpdated EventKind = "subscription_updated"
	EventSubscriptionDeleted EventKind = "subscription_deleted"
	EventTrialEnding         EventKind = "trial_ending"
	EventInvoicePaid         EventKind = "invoice_paid"
	EventPaymentFailed       EventKind = "payment_failed"
	EventInvoiceUpcoming     EventKind = "invoice_upcoming"
	EventRefunded            EventKind = "refunded"
	EventDisputeOpened       EventKind = "dispute_opened"
	EventCustomerUpdated     EventKind = "customer_updated"
	EventCustomerDeleted     EventKind = "customer_deleted"
	EventOther               EventKind = "other"
)

// Event is one provider event, normalized; Raw keeps it as sent.
type Event struct {
	ID       string
	Type     string
	Kind     EventKind
	Created  time.Time
	Livemode bool
	// The event's object, by kind.
	Subscription *Subscription
	Invoice      *Invoice
	Charge       *Charge
	Checkout     *CheckoutSession
	Customer     *Customer
	// Previous holds the fields an update changed (Stripe's previous_attributes).
	Previous map[string]any
	Raw      []byte
}

// Page is one page of a list.
type Page[T any] struct {
	Items []T
	// Next is the cursor of the next page; "" at the end.
	Next string
}

// Client talks to one connected account.
type Client interface {
	Mode() Mode
	// CheckKey reads the account and proves each permission with a call
	// that changes nothing. ErrKeyRejected when the key isn't valid.
	CheckKey(ctx context.Context) (KeyCheck, error)
	RegisterWebhook(ctx context.Context, url string, events []string) (Webhook, error)
	// WebhookExists reports whether a registered endpoint still exists.
	WebhookExists(ctx context.Context, id string) (bool, error)
	DeleteWebhook(ctx context.Context, id string) error
	// ListEvents returns events created at or after since, oldest first.
	ListEvents(ctx context.Context, since time.Time, types []string) ([]Event, error)
	ConfigurePortal(ctx context.Context, id string, c PortalConfig) (string, error)
	PortalSession(ctx context.Context, customer string, flow PortalFlow) (string, error)
	CreateCheckout(ctx context.Context, r CheckoutRequest) (CheckoutSession, error)
	GetCheckout(ctx context.Context, id string) (CheckoutSession, error)
	// SaveProduct creates the product (empty ID) or updates it.
	SaveProduct(ctx context.Context, p Product) (Product, error)
	CreatePrice(ctx context.Context, p Price) (Price, error)
	// RetirePrice stops a price from taking new subscriptions.
	RetirePrice(ctx context.Context, id string) error
	GetPrice(ctx context.Context, id string) (Price, error)
	GetSubscription(ctx context.Context, id string) (Subscription, error)
	// GetCharge reads a charge (a failed payment's reason).
	GetCharge(ctx context.Context, id string) (Charge, error)
	// GetCustomer reads a customer (the email a subscription belongs to).
	GetCustomer(ctx context.Context, id string) (Customer, error)
	// CancelSubscription cancels at the end of the period or now.
	CancelSubscription(ctx context.Context, id string, atPeriodEnd bool) (Subscription, error)
	ListSubscriptions(ctx context.Context, cursor string) (Page[Subscription], error)
	ListCustomers(ctx context.Context, cursor string) (Page[Customer], error)
	// DashboardURL links to an object in the provider's dashboard.
	DashboardURL(kind, id string) string
}

// WebhookVerifier checks a webhook request's signature and parses it.
type WebhookVerifier interface {
	// ParseWebhook verifies the signature with the endpoint's secret
	// (ErrBadSignature, also for a request signed outside the tolerance)
	// and returns the normalized event.
	ParseWebhook(header http.Header, body []byte, secret string, now time.Time) (Event, error)
	// ParseEvent reads an event already verified (stored, or read from the
	// provider's list of events).
	ParseEvent(body []byte) (Event, error)
}

var parsers = map[string]WebhookVerifier{}

// RegisterParser adds an adapter's webhook verifier (it needs no key).
func RegisterParser(key string, v WebhookVerifier) {
	registryMu.Lock()
	parsers[key] = v
	registryMu.Unlock()
}

// ParserFor returns an adapter's webhook verifier.
func ParserFor(key string) (WebhookVerifier, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	v, ok := parsers[key]
	return v, ok
}
