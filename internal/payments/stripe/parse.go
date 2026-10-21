package stripe

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/payments"
)

// eventKinds maps Stripe's event types onto the contract's kinds.
var eventKinds = map[string]payments.EventKind{
	"checkout.session.completed":           payments.EventCheckoutCompleted,
	"customer.subscription.created":        payments.EventSubscriptionCreated,
	"customer.subscription.updated":        payments.EventSubscriptionUpdated,
	"customer.subscription.deleted":        payments.EventSubscriptionDeleted,
	"customer.subscription.trial_will_end": payments.EventTrialEnding,
	"invoice.paid":                         payments.EventInvoicePaid,
	"invoice.payment_failed":               payments.EventPaymentFailed,
	"invoice.upcoming":                     payments.EventInvoiceUpcoming,
	"charge.refunded":                      payments.EventRefunded,
	"charge.dispute.created":               payments.EventDisputeOpened,
	"customer.updated":                     payments.EventCustomerUpdated,
	"customer.deleted":                     payments.EventCustomerDeleted,
}

// idOf reads a field that is an id or an expanded object.
func idOf(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var o struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(raw, &o)
	return o.ID
}

func unixPtr(n *int64) *time.Time {
	if n == nil || *n == 0 {
		return nil
	}
	t := time.Unix(*n, 0).UTC()
	return &t
}

func unix(n int64) time.Time {
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(n, 0).UTC()
}

func parseEvent(raw []byte) (payments.Event, error) {
	var e struct {
		ID       string `json:"id"`
		Object   string `json:"object"`
		Type     string `json:"type"`
		Created  int64  `json:"created"`
		Livemode bool   `json:"livemode"`
		Data     struct {
			Object   json.RawMessage `json:"object"`
			Previous map[string]any  `json:"previous_attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		return payments.Event{}, fmt.Errorf("stripe: unreadable event: %w", err)
	}
	if e.ID == "" || e.Type == "" {
		return payments.Event{}, fmt.Errorf("stripe: not an event")
	}
	out := payments.Event{ID: e.ID, Type: e.Type, Created: unix(e.Created), Livemode: e.Livemode, Previous: e.Data.Previous, Raw: raw, Kind: payments.EventOther}
	if k, ok := eventKinds[e.Type]; ok {
		out.Kind = k
	}
	var err error
	switch out.Kind {
	case payments.EventSubscriptionCreated, payments.EventSubscriptionUpdated, payments.EventSubscriptionDeleted, payments.EventTrialEnding:
		var s payments.Subscription
		if s, err = parseSubscription(e.Data.Object); err == nil {
			out.Subscription = &s
		}
	case payments.EventInvoicePaid, payments.EventPaymentFailed, payments.EventInvoiceUpcoming:
		var inv payments.Invoice
		if inv, err = parseInvoice(e.Data.Object); err == nil {
			out.Invoice = &inv
		}
	case payments.EventRefunded:
		var ch payments.Charge
		if ch, err = parseCharge(e.Data.Object, e.Data.Previous); err == nil {
			out.Charge = &ch
		}
	case payments.EventDisputeOpened:
		var d struct {
			Charge   json.RawMessage `json:"charge"`
			Amount   int64           `json:"amount"`
			Currency string          `json:"currency"`
		}
		if err = json.Unmarshal(e.Data.Object, &d); err == nil {
			out.Charge = &payments.Charge{ID: idOf(d.Charge), Amount: d.Amount, Currency: d.Currency}
		}
	case payments.EventCheckoutCompleted:
		var cs payments.CheckoutSession
		if cs, err = parseCheckout(e.Data.Object); err == nil {
			out.Checkout = &cs
		}
	case payments.EventCustomerUpdated, payments.EventCustomerDeleted:
		var cu payments.Customer
		if cu, err = parseCustomer(e.Data.Object); err == nil {
			if out.Kind == payments.EventCustomerDeleted {
				cu.Deleted = true
			}
			out.Customer = &cu
		}
	}
	return out, err
}

type stripePrice struct {
	ID          string          `json:"id"`
	Product     json.RawMessage `json:"product"`
	UnitAmount  *int64          `json:"unit_amount"`
	Currency    string          `json:"currency"`
	Active      bool            `json:"active"`
	TaxBehavior string          `json:"tax_behavior"`
	Recurring   *struct {
		Interval string `json:"interval"`
	} `json:"recurring"`
	Metadata map[string]string `json:"metadata"`
}

func (p stripePrice) price() payments.Price {
	out := payments.Price{ID: p.ID, ProductID: idOf(p.Product), Currency: strings.ToLower(p.Currency), Active: p.Active, TaxInclusive: p.TaxBehavior == "inclusive", Metadata: p.Metadata}
	if p.UnitAmount != nil {
		out.AmountCents = *p.UnitAmount
	}
	if p.Recurring != nil {
		out.Interval = p.Recurring.Interval
	}
	return out
}

func parsePrice(raw []byte) (payments.Price, error) {
	var p stripePrice
	if err := json.Unmarshal(raw, &p); err != nil {
		return payments.Price{}, fmt.Errorf("stripe: unreadable price: %w", err)
	}
	return p.price(), nil
}

type stripeDiscount struct {
	Coupon struct {
		ID         string   `json:"id"`
		Name       string   `json:"name"`
		PercentOff *float64 `json:"percent_off"`
		AmountOff  *int64   `json:"amount_off"`
		Duration   string   `json:"duration"`
		Months     *int     `json:"duration_in_months"`
	} `json:"coupon"`
	PromotionCode json.RawMessage `json:"promotion_code"`
	Start         *int64          `json:"start"`
	End           *int64          `json:"end"`
}

func (d *stripeDiscount) discount() *payments.Discount {
	if d == nil || d.Coupon.ID == "" {
		return nil
	}
	out := &payments.Discount{CouponID: d.Coupon.ID, Name: d.Coupon.Name, Duration: d.Coupon.Duration, Start: unixPtr(d.Start), End: unixPtr(d.End)}
	if d.Coupon.PercentOff != nil {
		out.PercentOff = *d.Coupon.PercentOff
	}
	if d.Coupon.AmountOff != nil {
		out.AmountOff = *d.Coupon.AmountOff
	}
	if d.Coupon.Months != nil {
		out.Months = *d.Coupon.Months
	}
	if out.Name == "" {
		out.Name = out.CouponID
	}
	// The promotion code's own code reads better than its id when expanded.
	var pc struct {
		Code string `json:"code"`
	}
	if json.Unmarshal(d.PromotionCode, &pc) == nil && pc.Code != "" {
		out.PromoCode = pc.Code
	}
	return out
}

func parseSubscription(raw []byte) (payments.Subscription, error) {
	var s struct {
		ID                string          `json:"id"`
		Customer          json.RawMessage `json:"customer"`
		Status            string          `json:"status"`
		StartDate         int64           `json:"start_date"`
		Created           int64           `json:"created"`
		TrialEnd          *int64          `json:"trial_end"`
		CurrentPeriodEnd  *int64          `json:"current_period_end"`
		CancelAt          *int64          `json:"cancel_at"`
		CancelAtPeriodEnd bool            `json:"cancel_at_period_end"`
		CanceledAt        *int64          `json:"canceled_at"`
		EndedAt           *int64          `json:"ended_at"`
		Livemode          bool            `json:"livemode"`
		Discount          *stripeDiscount `json:"discount"`
		Discounts         []json.RawMessage
		Items             struct {
			Data []struct {
				Price            stripePrice `json:"price"`
				CurrentPeriodEnd *int64      `json:"current_period_end"`
			} `json:"data"`
		} `json:"items"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return payments.Subscription{}, fmt.Errorf("stripe: unreadable subscription: %w", err)
	}
	out := payments.Subscription{ID: s.ID, CustomerID: idOf(s.Customer), Status: s.Status, StartedAt: unix(s.StartDate), TrialEnd: unixPtr(s.TrialEnd),
		CurrentPeriodEnd: unixPtr(s.CurrentPeriodEnd), CancelAt: unixPtr(s.CancelAt), CancelAtPeriodEnd: s.CancelAtPeriodEnd, CanceledAt: unixPtr(s.CanceledAt),
		EndedAt: unixPtr(s.EndedAt), Livemode: s.Livemode, Discount: s.Discount.discount(), Metadata: s.Metadata}
	if out.StartedAt.IsZero() {
		out.StartedAt = unix(s.Created)
	}
	if len(s.Items.Data) > 0 {
		p := s.Items.Data[0].Price.price()
		out.PriceID, out.ProductID, out.Interval, out.AmountCents, out.Currency = p.ID, p.ProductID, p.Interval, p.AmountCents, p.Currency
		// Newer API versions keep the period on the item.
		if out.CurrentPeriodEnd == nil {
			out.CurrentPeriodEnd = unixPtr(s.Items.Data[0].CurrentPeriodEnd)
		}
	}
	if out.CancelAtPeriodEnd && out.CancelAt == nil {
		out.CancelAt = out.CurrentPeriodEnd
	}
	return out, nil
}

func parseInvoice(raw []byte) (payments.Invoice, error) {
	var inv struct {
		ID            string          `json:"id"`
		Customer      json.RawMessage `json:"customer"`
		CustomerEmail string          `json:"customer_email"`
		Subscription  json.RawMessage `json:"subscription"`
		AmountDue     int64           `json:"amount_due"`
		AmountPaid    int64           `json:"amount_paid"`
		Currency      string          `json:"currency"`
		BillingReason string          `json:"billing_reason"`
		PeriodEnd     *int64          `json:"period_end"`
		NextAttempt   *int64          `json:"next_payment_attempt"`
		AttemptCount  int             `json:"attempt_count"`
		Charge        json.RawMessage `json:"charge"`
		Parent        *struct {
			SubscriptionDetails *struct {
				Subscription json.RawMessage `json:"subscription"`
			} `json:"subscription_details"`
		} `json:"parent"`
		Lines struct {
			Data []struct {
				Price  *stripePrice `json:"price"`
				Period *struct {
					End *int64 `json:"end"`
				} `json:"period"`
			} `json:"data"`
		} `json:"lines"`
		TotalDiscounts []struct {
			Amount int64 `json:"amount"`
		} `json:"total_discount_amounts"`
	}
	if err := json.Unmarshal(raw, &inv); err != nil {
		return payments.Invoice{}, fmt.Errorf("stripe: unreadable invoice: %w", err)
	}
	out := payments.Invoice{ID: inv.ID, CustomerID: idOf(inv.Customer), CustomerEmail: strings.ToLower(inv.CustomerEmail), SubscriptionID: idOf(inv.Subscription),
		AmountDue: inv.AmountDue, AmountPaid: inv.AmountPaid, Currency: strings.ToLower(inv.Currency), BillingReason: inv.BillingReason,
		PeriodEnd: unixPtr(inv.PeriodEnd), NextAttempt: unixPtr(inv.NextAttempt), AttemptCount: inv.AttemptCount, ChargeID: idOf(inv.Charge)}
	if out.SubscriptionID == "" && inv.Parent != nil && inv.Parent.SubscriptionDetails != nil {
		out.SubscriptionID = idOf(inv.Parent.SubscriptionDetails.Subscription)
	}
	if len(inv.Lines.Data) > 0 {
		l := inv.Lines.Data[0]
		if l.Price != nil {
			out.PriceID = l.Price.ID
		}
		if l.Period != nil && l.Period.End != nil {
			out.PeriodEnd = unixPtr(l.Period.End)
		}
	}
	for _, d := range inv.TotalDiscounts {
		out.DiscountedAmount += d.Amount
	}
	return out, nil
}

func parseCharge(raw []byte, previous map[string]any) (payments.Charge, error) {
	var ch struct {
		ID             string          `json:"id"`
		Customer       json.RawMessage `json:"customer"`
		Invoice        json.RawMessage `json:"invoice"`
		Amount         int64           `json:"amount"`
		AmountRefunded int64           `json:"amount_refunded"`
		Currency       string          `json:"currency"`
		FailureMessage string          `json:"failure_message"`
		PaymentIntent  json.RawMessage `json:"payment_intent"`
	}
	if err := json.Unmarshal(raw, &ch); err != nil {
		return payments.Charge{}, fmt.Errorf("stripe: unreadable charge: %w", err)
	}
	out := payments.Charge{ID: ch.ID, CustomerID: idOf(ch.Customer), InvoiceID: idOf(ch.Invoice), Amount: ch.Amount, AmountRefunded: ch.AmountRefunded,
		Currency: strings.ToLower(ch.Currency), FailureMessage: ch.FailureMessage, PaymentIntentID: idOf(ch.PaymentIntent)}
	// The refund this event reports: what amount_refunded grew by.
	out.RefundAmount = ch.AmountRefunded
	if prev, ok := previous["amount_refunded"].(float64); ok {
		out.RefundAmount = ch.AmountRefunded - int64(prev)
	}
	return out, nil
}

func parseCheckout(raw []byte) (payments.CheckoutSession, error) {
	var cs struct {
		ID              string          `json:"id"`
		URL             string          `json:"url"`
		Status          string          `json:"status"`
		PaymentStatus   string          `json:"payment_status"`
		Customer        json.RawMessage `json:"customer"`
		CustomerEmail   string          `json:"customer_email"`
		CustomerDetails *struct {
			Email string `json:"email"`
		} `json:"customer_details"`
		Subscription  json.RawMessage   `json:"subscription"`
		ClientRef     string            `json:"client_reference_id"`
		Metadata      map[string]string `json:"metadata"`
		Livemode      bool              `json:"livemode"`
		Mode          string            `json:"mode"`
		AmountTotal   int64             `json:"amount_total"`
		Currency      string            `json:"currency"`
		PaymentIntent json.RawMessage   `json:"payment_intent"`
	}
	if err := json.Unmarshal(raw, &cs); err != nil {
		return payments.CheckoutSession{}, fmt.Errorf("stripe: unreadable checkout session: %w", err)
	}
	out := payments.CheckoutSession{ID: cs.ID, URL: cs.URL, Status: cs.Status, PaymentStatus: cs.PaymentStatus, CustomerID: idOf(cs.Customer),
		CustomerEmail: strings.ToLower(cs.CustomerEmail), SubscriptionID: idOf(cs.Subscription), ClientRef: cs.ClientRef, Metadata: cs.Metadata, Livemode: cs.Livemode,
		Mode: cs.Mode, AmountTotal: cs.AmountTotal, Currency: strings.ToLower(cs.Currency), PaymentIntentID: idOf(cs.PaymentIntent)}
	if cs.CustomerDetails != nil && cs.CustomerDetails.Email != "" {
		out.CustomerEmail = strings.ToLower(cs.CustomerDetails.Email)
	}
	return out, nil
}

func parseCustomer(raw []byte) (payments.Customer, error) {
	var cu struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Created  int64  `json:"created"`
		Deleted  bool   `json:"deleted"`
		Livemode bool   `json:"livemode"`
	}
	if err := json.Unmarshal(raw, &cu); err != nil {
		return payments.Customer{}, fmt.Errorf("stripe: unreadable customer: %w", err)
	}
	return payments.Customer{ID: cu.ID, Email: strings.ToLower(strings.TrimSpace(cu.Email)), Name: cu.Name, Created: unix(cu.Created), Deleted: cu.Deleted, Livemode: cu.Livemode}, nil
}
