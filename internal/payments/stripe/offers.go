package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/payments"
)

// Offers and gifts (X17): coupons and promotion codes, a coupon applied to
// a subscription, and a checkout for one payment.

var (
	_ payments.Offers          = (*Client)(nil)
	_ payments.OneTimePayments = (*Client)(nil)
)

type couponJSON struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	PercentOff       float64           `json:"percent_off"`
	AmountOff        int64             `json:"amount_off"`
	Currency         string            `json:"currency"`
	Duration         string            `json:"duration"`
	DurationInMonths int               `json:"duration_in_months"`
	Valid            bool              `json:"valid"`
	Metadata         map[string]string `json:"metadata"`
	AppliesTo        *struct {
		Products []string `json:"products"`
	} `json:"applies_to"`
}

func (c couponJSON) coupon() payments.Coupon {
	out := payments.Coupon{ID: c.ID, Name: c.Name, PercentOff: c.PercentOff, AmountOff: c.AmountOff, Currency: strings.ToLower(c.Currency), Duration: c.Duration,
		Months: c.DurationInMonths, Valid: c.Valid, Metadata: c.Metadata}
	if c.AppliesTo != nil {
		out.Products = c.AppliesTo.Products
	}
	return out
}

type promoJSON struct {
	ID             string            `json:"id"`
	Code           string            `json:"code"`
	Coupon         json.RawMessage   `json:"coupon"`
	Active         bool              `json:"active"`
	MaxRedemptions int               `json:"max_redemptions"`
	ExpiresAt      int64             `json:"expires_at"`
	TimesRedeemed  int               `json:"times_redeemed"`
	Metadata       map[string]string `json:"metadata"`
	Restrictions   struct {
		FirstTimeTransaction bool `json:"first_time_transaction"`
	} `json:"restrictions"`
}

func (p promoJSON) promo() payments.PromotionCode {
	out := payments.PromotionCode{ID: p.ID, Code: p.Code, CouponID: idOf(p.Coupon), Active: p.Active, MaxRedemptions: p.MaxRedemptions,
		FirstTimeOnly: p.Restrictions.FirstTimeTransaction, TimesRedeemed: p.TimesRedeemed, Metadata: p.Metadata}
	if p.ExpiresAt > 0 {
		t := time.Unix(p.ExpiresAt, 0).UTC()
		out.ExpiresAt = &t
	}
	return out
}

func setMetadata(f url.Values, prefix string, m map[string]string) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		f.Set(prefix+"["+k+"]", m[k])
	}
}

// CreateCoupon implements payments.Offers.
func (c *Client) CreateCoupon(ctx context.Context, cp payments.Coupon) (payments.Coupon, error) {
	f := url.Values{}
	if cp.ID != "" {
		f.Set("id", cp.ID)
	}
	if cp.Name != "" {
		f.Set("name", cp.Name)
	}
	switch {
	case cp.PercentOff > 0:
		f.Set("percent_off", strconv.FormatFloat(cp.PercentOff, 'f', -1, 64))
	case cp.AmountOff > 0:
		f.Set("amount_off", strconv.FormatInt(cp.AmountOff, 10))
		f.Set("currency", strings.ToLower(cp.Currency))
	default:
		return payments.Coupon{}, fmt.Errorf("stripe: a coupon needs a percentage or an amount")
	}
	f.Set("duration", cp.Duration)
	if cp.Duration == "repeating" {
		f.Set("duration_in_months", strconv.Itoa(cp.Months))
	}
	for i, p := range cp.Products {
		f.Set(fmt.Sprintf("applies_to[products][%d]", i), p)
	}
	setMetadata(f, "metadata", cp.Metadata)
	var out couponJSON
	if err := c.call(ctx, http.MethodPost, "/v1/coupons", f, &out); err != nil {
		return payments.Coupon{}, err
	}
	return out.coupon(), nil
}

// GetCoupon implements payments.Offers. A deleted coupon is
// payments.APIError 404.
func (c *Client) GetCoupon(ctx context.Context, id string) (payments.Coupon, error) {
	var out couponJSON
	if err := c.call(ctx, http.MethodGet, "/v1/coupons/"+url.PathEscape(id), nil, &out); err != nil {
		return payments.Coupon{}, err
	}
	return out.coupon(), nil
}

// CreatePromotionCode implements payments.Offers.
func (c *Client) CreatePromotionCode(ctx context.Context, p payments.PromotionCode) (payments.PromotionCode, error) {
	f := url.Values{}
	f.Set("coupon", p.CouponID)
	if p.Code != "" {
		f.Set("code", p.Code)
	}
	if p.MaxRedemptions > 0 {
		f.Set("max_redemptions", strconv.Itoa(p.MaxRedemptions))
	}
	if p.ExpiresAt != nil {
		f.Set("expires_at", strconv.FormatInt(p.ExpiresAt.Unix(), 10))
	}
	if p.FirstTimeOnly {
		f.Set("restrictions[first_time_transaction]", "true")
	}
	setMetadata(f, "metadata", p.Metadata)
	var out promoJSON
	if err := c.call(ctx, http.MethodPost, "/v1/promotion_codes", f, &out); err != nil {
		return payments.PromotionCode{}, err
	}
	return out.promo(), nil
}

// SetPromotionCodeActive implements payments.Offers.
func (c *Client) SetPromotionCodeActive(ctx context.Context, id string, active bool) error {
	f := url.Values{}
	f.Set("active", strconv.FormatBool(active))
	return c.call(ctx, http.MethodPost, "/v1/promotion_codes/"+url.PathEscape(id), f, nil)
}

// ApplyCoupon implements payments.Offers.
func (c *Client) ApplyCoupon(ctx context.Context, subscriptionID, couponID string) (payments.Subscription, error) {
	f := url.Values{}
	f.Set("coupon", couponID)
	var raw json.RawMessage
	if err := c.call(ctx, http.MethodPost, "/v1/subscriptions/"+url.PathEscape(subscriptionID), f, &raw); err != nil {
		return payments.Subscription{}, err
	}
	return parseSubscription(raw)
}

// CreatePaymentCheckout implements payments.OneTimePayments.
func (c *Client) CreatePaymentCheckout(ctx context.Context, r payments.PaymentCheckoutRequest) (payments.CheckoutSession, error) {
	f := url.Values{}
	f.Set("mode", "payment")
	f.Set("line_items[0][price_data][currency]", strings.ToLower(r.Currency))
	f.Set("line_items[0][price_data][unit_amount]", strconv.FormatInt(r.AmountCents, 10))
	f.Set("line_items[0][price_data][product_data][name]", r.Name)
	if r.Description != "" {
		f.Set("line_items[0][price_data][product_data][description]", r.Description)
	}
	f.Set("line_items[0][quantity]", "1")
	f.Set("success_url", r.SuccessURL)
	f.Set("cancel_url", r.CancelURL)
	if r.CustomerEmail != "" {
		f.Set("customer_email", r.CustomerEmail)
	}
	if r.Tax {
		f.Set("automatic_tax[enabled]", "true")
		f.Set("line_items[0][price_data][tax_behavior]", "exclusive")
	}
	if r.ClientRef != "" {
		f.Set("client_reference_id", r.ClientRef)
	}
	setMetadata(f, "metadata", r.Metadata)
	setMetadata(f, "payment_intent_data[metadata]", r.Metadata)
	var raw json.RawMessage
	if err := c.call(ctx, http.MethodPost, "/v1/checkout/sessions", f, &raw); err != nil {
		return payments.CheckoutSession{}, err
	}
	return parseCheckout(raw)
}
