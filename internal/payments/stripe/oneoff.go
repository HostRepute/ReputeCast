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

	"github.com/hostrepute/reputecast/internal/payments"
)

// One-time payments (payments.OneTimePrices, digital products): prices
// paid once, fixed or with a custom amount the buyer chooses (Stripe's
// custom_unit_amount), Checkout in payment mode with the metadata on the
// session and on its PaymentIntent, and the events that report them. The
// restricted key's permissions already cover them: Checkout Sessions and
// Prices to write, Charges and Events to read.

var (
	_ payments.OneTimePrices = (*Client)(nil)
	_ payments.PaymentParser = Verifier{}
)

// CreateOneTimePrice implements payments.OneTimePrices.
func (c *Client) CreateOneTimePrice(ctx context.Context, p payments.OneTimePrice) (payments.Price, error) {
	f := url.Values{}
	f.Set("product", p.ProductID)
	f.Set("currency", strings.ToLower(p.Currency))
	if p.Custom != nil {
		f.Set("custom_unit_amount[enabled]", "true")
		f.Set("custom_unit_amount[minimum]", strconv.FormatInt(p.Custom.MinimumCents, 10))
		if p.Custom.PresetCents > 0 {
			f.Set("custom_unit_amount[preset]", strconv.FormatInt(p.Custom.PresetCents, 10))
		}
	} else {
		f.Set("unit_amount", strconv.FormatInt(p.AmountCents, 10))
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

// CreatePriceCheckout implements payments.OneTimePrices.
func (c *Client) CreatePriceCheckout(ctx context.Context, r payments.PriceCheckoutRequest) (payments.CheckoutSession, error) {
	f := url.Values{}
	f.Set("mode", "payment")
	f.Set("line_items[0][price]", r.PriceID)
	f.Set("line_items[0][quantity]", "1")
	f.Set("success_url", r.SuccessURL)
	f.Set("cancel_url", r.CancelURL)
	if r.CustomerEmail != "" {
		f.Set("customer_email", r.CustomerEmail)
	}
	if r.Tax {
		f.Set("automatic_tax[enabled]", "true")
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
		f.Set("payment_intent_data[metadata]["+k+"]", r.Metadata[k])
	}
	var raw json.RawMessage
	if err := c.call(ctx, http.MethodPost, "/v1/checkout/sessions", f, &raw); err != nil {
		return payments.CheckoutSession{}, err
	}
	return parseCheckout(raw)
}

// GetPayment implements payments.OneTimePrices.
func (c *Client) GetPayment(ctx context.Context, sessionID string) (payments.Payment, error) {
	var raw json.RawMessage
	q := url.Values{}
	q.Add("expand[]", "payment_intent")
	if err := c.call(ctx, http.MethodGet, "/v1/checkout/sessions/"+url.PathEscape(sessionID), q, &raw); err != nil {
		return payments.Payment{}, err
	}
	p, err := parsePaymentSession(raw)
	p.Kind = payments.PaymentCompleted
	return p, err
}

// ParsePayment implements payments.PaymentParser.
func (Verifier) ParsePayment(body []byte) (payments.Payment, error) {
	var e struct {
		Type     string `json:"type"`
		Created  int64  `json:"created"`
		Livemode bool   `json:"livemode"`
		Data     struct {
			Object   json.RawMessage `json:"object"`
			Previous map[string]any  `json:"previous_attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &e); err != nil {
		return payments.Payment{}, fmt.Errorf("stripe: unreadable event: %w", err)
	}
	var (
		p   payments.Payment
		err error
	)
	switch e.Type {
	case "checkout.session.completed", "checkout.session.async_payment_succeeded":
		p, err = parsePaymentSession(e.Data.Object)
		p.Kind = payments.PaymentCompleted
		if p.Mode != "payment" {
			p.Kind = payments.PaymentOther
		}
	case "charge.refunded":
		p, err = parsePaymentCharge(e.Data.Object, e.Data.Previous)
		p.Kind = payments.PaymentRefunded
	case "charge.dispute.created", "charge.dispute.updated", "charge.dispute.closed":
		p, err = parseDispute(e.Data.Object)
		p.Kind = map[string]payments.PaymentKind{"charge.dispute.created": payments.PaymentDisputeOpened,
			"charge.dispute.updated": payments.PaymentDisputeUpdated, "charge.dispute.closed": payments.PaymentDisputeClosed}[e.Type]
	default:
		p.Kind = payments.PaymentOther
	}
	p.Livemode = e.Livemode
	p.Created = unix(e.Created)
	return p, err
}

type paymentIntent struct {
	ID       string            `json:"id"`
	Metadata map[string]string `json:"metadata"`
	Charge   json.RawMessage   `json:"latest_charge"`
}

// intentOf reads a payment_intent field: an id, or the expanded object.
func intentOf(raw json.RawMessage) (id, charge string, meta map[string]string) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", "", nil
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s, "", nil
	}
	var pi paymentIntent
	_ = json.Unmarshal(raw, &pi)
	return pi.ID, idOf(pi.Charge), pi.Metadata
}

func parsePaymentSession(raw []byte) (payments.Payment, error) {
	var cs struct {
		ID              string          `json:"id"`
		Mode            string          `json:"mode"`
		PaymentStatus   string          `json:"payment_status"`
		PaymentIntent   json.RawMessage `json:"payment_intent"`
		Customer        json.RawMessage `json:"customer"`
		CustomerEmail   string          `json:"customer_email"`
		CustomerDetails *struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"customer_details"`
		ClientRef    string `json:"client_reference_id"`
		AmountTotal  *int64 `json:"amount_total"`
		Currency     string `json:"currency"`
		TotalDetails *struct {
			AmountTax int64 `json:"amount_tax"`
		} `json:"total_details"`
		Metadata map[string]string `json:"metadata"`
		Livemode bool              `json:"livemode"`
	}
	if err := json.Unmarshal(raw, &cs); err != nil {
		return payments.Payment{}, fmt.Errorf("stripe: unreadable checkout session: %w", err)
	}
	pi, charge, piMeta := intentOf(cs.PaymentIntent)
	out := payments.Payment{SessionID: cs.ID, Mode: cs.Mode, Paid: cs.PaymentStatus == "paid" || cs.PaymentStatus == "no_payment_required",
		PaymentIntentID: pi, ChargeID: charge, ClientRef: cs.ClientRef, CustomerID: idOf(cs.Customer), CustomerEmail: strings.ToLower(strings.TrimSpace(cs.CustomerEmail)),
		Currency: strings.ToLower(cs.Currency), Metadata: cs.Metadata, Livemode: cs.Livemode}
	if out.Metadata == nil {
		out.Metadata = piMeta
	}
	if cs.CustomerDetails != nil {
		if cs.CustomerDetails.Email != "" {
			out.CustomerEmail = strings.ToLower(strings.TrimSpace(cs.CustomerDetails.Email))
		}
		out.CustomerName = cs.CustomerDetails.Name
	}
	if cs.AmountTotal != nil {
		out.AmountCents = *cs.AmountTotal
	}
	if cs.TotalDetails != nil {
		out.TaxCents = cs.TotalDetails.AmountTax
	}
	return out, nil
}

func parsePaymentCharge(raw []byte, previous map[string]any) (payments.Payment, error) {
	var ch struct {
		ID             string            `json:"id"`
		PaymentIntent  json.RawMessage   `json:"payment_intent"`
		Customer       json.RawMessage   `json:"customer"`
		Amount         int64             `json:"amount"`
		AmountRefunded int64             `json:"amount_refunded"`
		Currency       string            `json:"currency"`
		Metadata       map[string]string `json:"metadata"`
		BillingDetails *struct {
			Email string `json:"email"`
		} `json:"billing_details"`
	}
	if err := json.Unmarshal(raw, &ch); err != nil {
		return payments.Payment{}, fmt.Errorf("stripe: unreadable charge: %w", err)
	}
	pi, _, _ := intentOf(ch.PaymentIntent)
	out := payments.Payment{ChargeID: ch.ID, PaymentIntentID: pi, CustomerID: idOf(ch.Customer), AmountCents: ch.Amount, Currency: strings.ToLower(ch.Currency),
		RefundedCents: ch.AmountRefunded, RefundCents: ch.AmountRefunded, Metadata: ch.Metadata}
	if prev, ok := previous["amount_refunded"].(float64); ok {
		out.RefundCents = ch.AmountRefunded - int64(prev)
	}
	if ch.BillingDetails != nil {
		out.CustomerEmail = strings.ToLower(strings.TrimSpace(ch.BillingDetails.Email))
	}
	return out, nil
}

func parseDispute(raw []byte) (payments.Payment, error) {
	var d struct {
		ID              string          `json:"id"`
		Charge          json.RawMessage `json:"charge"`
		PaymentIntent   json.RawMessage `json:"payment_intent"`
		Amount          int64           `json:"amount"`
		Currency        string          `json:"currency"`
		Status          string          `json:"status"`
		Reason          string          `json:"reason"`
		EvidenceDetails *struct {
			DueBy *int64 `json:"due_by"`
		} `json:"evidence_details"`
		BalanceTransactions []struct {
			Fee int64 `json:"fee"`
		} `json:"balance_transactions"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		return payments.Payment{}, fmt.Errorf("stripe: unreadable dispute: %w", err)
	}
	pi, _, _ := intentOf(d.PaymentIntent)
	out := payments.Payment{DisputeID: d.ID, ChargeID: idOf(d.Charge), PaymentIntentID: pi, AmountCents: d.Amount, Currency: strings.ToLower(d.Currency),
		DisputeStatus: d.Status, DisputeReason: d.Reason, Metadata: d.Metadata}
	if d.EvidenceDetails != nil {
		out.DisputeDueBy = unixPtr(d.EvidenceDetails.DueBy)
	}
	for _, b := range d.BalanceTransactions {
		out.DisputeFee += b.Fee
	}
	return out, nil
}
