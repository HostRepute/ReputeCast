package payments

import (
	"context"
	"time"
)

// One-time payments (digital products, design expansion phase 7, M29): a
// price paid once, fixed or chosen by the buyer above a minimum (pay what
// you want), and a hosted checkout in payment mode. Adapters that support
// them implement OneTimePrices on their Client and PaymentParser on
// their webhook verifier; callers check with a type assertion, so the
// subscription contract above doesn't change.

// CustomAmount lets the buyer choose the amount: at least Minimum, with
// Preset filled in on the provider's page.
type CustomAmount struct {
	MinimumCents int64
	PresetCents  int64
}

// OneTimePrice is a price paid once: AmountCents, or a custom amount.
type OneTimePrice struct {
	ProductID   string
	AmountCents int64
	Custom      *CustomAmount
	Currency    string
	// TaxInclusive marks a price that includes tax.
	TaxInclusive bool
	Metadata     map[string]string
}

// PriceCheckoutRequest opens a checkout for one one-time price.
type PriceCheckoutRequest struct {
	PriceID       string
	CustomerEmail string
	// Tax adds tax at checkout and collects the address it needs.
	Tax        bool
	SuccessURL string
	CancelURL  string
	ClientRef  string
	// Metadata goes on the session and on the payment, so refunds and
	// disputes can be traced back.
	Metadata map[string]string
}

// OneTimePrices is implemented by clients that sell one-time prices.
type OneTimePrices interface {
	CreateOneTimePrice(ctx context.Context, p OneTimePrice) (Price, error)
	CreatePriceCheckout(ctx context.Context, r PriceCheckoutRequest) (CheckoutSession, error)
	// GetPayment reads a payment checkout as its completion would report
	// it (the thank-you page, before the webhook arrives).
	GetPayment(ctx context.Context, sessionID string) (Payment, error)
}

// PaymentKind is what a one-time payment event says.
type PaymentKind string

// Payment event kinds.
const (
	PaymentCompleted      PaymentKind = "completed"
	PaymentRefunded       PaymentKind = "refunded"
	PaymentDisputeOpened  PaymentKind = "dispute_opened"
	PaymentDisputeUpdated PaymentKind = "dispute_updated"
	PaymentDisputeClosed  PaymentKind = "dispute_closed"
	PaymentOther          PaymentKind = "other"
)

// Payment is a one-time payment as an event, or a checkout, reports it.
type Payment struct {
	Kind PaymentKind
	// The checkout session (completed), the payment and its charge.
	SessionID       string
	PaymentIntentID string
	ChargeID        string
	// Mode is the checkout's: "payment" for one-time payments.
	Mode          string
	Paid          bool
	ClientRef     string
	CustomerEmail string
	CustomerName  string
	CustomerID    string
	AmountCents   int64
	TaxCents      int64
	Currency      string
	// Refunds: the total refunded so far, and this refund's part.
	RefundedCents int64
	RefundCents   int64
	// Disputes: the provider's id, status (needs_response,
	// under_review, won, lost…), reason ("fraudulent") and the date to
	// respond by; the fee the provider charges for it.
	DisputeID     string
	DisputeStatus string
	DisputeReason string
	DisputeDueBy  *time.Time
	DisputeFee    int64
	Metadata      map[string]string
	Livemode      bool
	Created       time.Time
}

// PaymentParser reads the one-time payment in an event already verified
// (Kind PaymentOther when the event isn't about one).
type PaymentParser interface {
	ParsePayment(body []byte) (Payment, error)
}
