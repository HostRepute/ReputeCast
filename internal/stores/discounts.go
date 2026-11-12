package stores

import (
	"context"
	"time"
)

// Discount codes made in the store (M59 G-22): one discount with many
// single-use codes, so each email reader gets a code of their own that the
// store's checkout accepts. An adapter that can create them implements
// DiscountCreator; core code finds it with a type assertion on the Client.

// Discount kinds.
const (
	DiscountPercentage   = "percentage"
	DiscountFixedAmount  = "fixed_amount"
	DiscountFreeShipping = "free_shipping"
)

// DiscountSpec is what every code of a discount gives: each code works
// once, for one customer.
type DiscountSpec struct {
	Title string
	// Kind is percentage, fixed_amount or free_shipping.
	Kind string
	// Value is the percentage (1–100) or the amount off in minor units of
	// Currency; free shipping has none.
	Value    int64
	Currency string
	// MinimumCents is the order subtotal the code needs (0: none).
	MinimumCents int64
	// EndsAt is when the codes stop working (nil: never).
	EndsAt *time.Time
	// CollectionIDs limit the discount to collections (product categories
	// in WooCommerce); none applies it to every product.
	CollectionIDs []string
}

// DiscountCodes is where adding codes stands.
type DiscountCodes struct {
	// JobID names the store's job that adds them ("" when the codes were
	// added at once).
	JobID string
	Done  bool
	// IDs are the store's ids of the codes it created, by code, when it
	// gives them.
	IDs map[string]string
	// Failed are the codes the store refused (a code already taken).
	Failed []string
}

// DiscountCreator creates discounts and their codes. Errors are the
// contract's: ErrPermission when the credentials can't write discounts (a
// scope not granted, a read-only key), *RateLimited when throttled.
type DiscountCreator interface {
	// MaxCodesPerCall is the most codes AddCodes takes at once.
	MaxCodesPerCall() int
	// CreateDiscount creates the discount with its first code. Platforms
	// without a parent discount return "" as its id.
	CreateDiscount(ctx context.Context, spec DiscountSpec, firstCode string) (discountID, firstCodeID string, err error)
	// AddCodes adds codes to a discount; the store may add them in the
	// background (Done false), CodesStatus then follows its job.
	AddCodes(ctx context.Context, discountID string, spec DiscountSpec, codes []string) (DiscountCodes, error)
	// CodesStatus reports on a job AddCodes started for codes.
	CodesStatus(ctx context.Context, discountID, jobID string, codes []string) (DiscountCodes, error)
	// RemoveCodes deletes codes from the store; ids are the store's ids of
	// codes, in the same order ("" where unknown). Codes already gone are
	// not an error.
	RemoveCodes(ctx context.Context, discountID string, codes []string, ids []string) error
	// DisableDiscount stops a discount's codes from working.
	DisableDiscount(ctx context.Context, discountID string) error
}
