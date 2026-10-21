package stripetest

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/hostrepute/reputecast/internal/payments"
)

// One-time payments (digital products, design expansion phase 7): Checkout
// in payment mode, prices with a custom amount, and the events Stripe sends
// when a buyer pays, a payment is refunded or a dispute opens and closes.

// customAmount keeps a price's custom amount (pay what you want).
func customAmount(o obj, form url.Values) {
	if form.Get("custom_unit_amount[enabled]") != "true" {
		return
	}
	minimum, _ := strconv.ParseInt(form.Get("custom_unit_amount[minimum]"), 10, 64)
	preset, _ := strconv.ParseInt(form.Get("custom_unit_amount[preset]"), 10, 64)
	o["unit_amount"] = nil
	o["custom_unit_amount"] = obj{"minimum": minimum, "preset": preset, "maximum": nil}
}

// PaymentOptions completes a payment-mode checkout the way a buyer would.
type PaymentOptions struct {
	Email string
	Name  string
	// AmountCents is the amount chosen on a pay-what-you-want price (the
	// preset when zero).
	AmountCents int64
	// TaxCents is added by Stripe Tax.
	TaxCents int64
}

// Paid is a completed payment's objects.
type Paid struct {
	SessionID     string
	PaymentIntent string
	Charge        string
	Events        []Event
}

// PayCheckout pays a payment-mode checkout session: it creates the
// PaymentIntent and its charge and sends checkout.session.completed.
func (f *Fake) PayCheckout(mode payments.Mode, sessionID string, o PaymentOptions) (Paid, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := f.modes[mode]
	live := mode == payments.Live
	cs, ok := d.objects[sessionID]
	if !ok {
		return Paid{}, fmt.Errorf("stripetest: no checkout session %s", sessionID)
	}
	if cs["mode"] != "payment" {
		return Paid{}, fmt.Errorf("stripetest: session %s isn't in payment mode", sessionID)
	}
	email := o.Email
	if email == "" {
		email, _ = cs["customer_email"].(string)
	}
	priceID, _ := cs["_price"].(string)
	price := d.objects[priceID]
	amount := o.AmountCents
	currency := f.Account.Currency
	if price != nil {
		if amount == 0 {
			amount = num(price["unit_amount"])
		}
		if amount == 0 {
			if c, ok := price["custom_unit_amount"].(obj); ok {
				amount = num(c["preset"])
				if amount == 0 {
					amount = num(c["minimum"])
				}
			}
		}
		if c, ok := price["currency"].(string); ok && c != "" {
			currency = c
		}
	}
	total := amount + o.TaxCents
	now := f.now().Unix()
	pi := newID("pi")
	ch := newID("ch")
	meta := cs["metadata"]
	d.objects[ch] = obj{"id": ch, "object": "charge", "amount": total, "amount_refunded": int64(0), "currency": currency, "payment_intent": pi,
		"customer": nil, "invoice": nil, "metadata": meta, "billing_details": obj{"email": email, "name": nilIfEmpty(o.Name)}, "failure_message": nil,
		"created": now, "livemode": live, "refunded": false}
	d.objects[pi] = obj{"id": pi, "object": "payment_intent", "amount": total, "currency": currency, "status": "succeeded", "latest_charge": ch,
		"metadata": meta, "created": now, "livemode": live}
	cs["status"], cs["payment_status"], cs["payment_intent"] = "complete", "paid", pi
	cs["amount_subtotal"], cs["amount_total"], cs["currency"] = amount, total, currency
	cs["total_details"] = obj{"amount_tax": o.TaxCents, "amount_discount": int64(0), "amount_shipping": int64(0)}
	cs["customer_details"] = obj{"email": email, "name": nilIfEmpty(o.Name)}
	ev := f.emitLocked(d, "checkout.session.completed", cs, nil, live)
	return Paid{SessionID: sessionID, PaymentIntent: pi, Charge: ch, Events: []Event{ev}}, nil
}

// Refund refunds part or all of a charge (amount 0: what's left) and sends
// charge.refunded, as a refund made in Stripe's dashboard does.
func (f *Fake) Refund(mode payments.Mode, chargeID string, amount int64) (Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := f.modes[mode]
	ch, ok := d.objects[chargeID]
	if !ok {
		return Event{}, fmt.Errorf("stripetest: no charge %s", chargeID)
	}
	before := num(ch["amount_refunded"])
	left := num(ch["amount"]) - before
	if amount <= 0 || amount > left {
		amount = left
	}
	ch["amount_refunded"] = before + amount
	ch["refunded"] = num(ch["amount_refunded"]) >= num(ch["amount"])
	return f.emitLocked(d, "charge.refunded", ch, obj{"amount_refunded": before, "refunded": false}, mode == payments.Live), nil
}

// Dispute opens a dispute on a charge (reason "fraudulent" and the like)
// and sends charge.dispute.created; it returns the dispute's id too.
func (f *Fake) Dispute(mode payments.Mode, chargeID, reason string, dueBy int64) (string, Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := f.modes[mode]
	ch, ok := d.objects[chargeID]
	if !ok {
		return "", Event{}, fmt.Errorf("stripetest: no charge %s", chargeID)
	}
	id := newID("dp")
	dp := obj{"id": id, "object": "dispute", "charge": chargeID, "payment_intent": ch["payment_intent"], "amount": ch["amount"], "currency": ch["currency"],
		"status": "needs_response", "reason": reason, "evidence_details": obj{"due_by": dueBy}, "balance_transactions": []obj{{"fee": int64(1500)}},
		"metadata": obj{}, "created": f.now().Unix(), "livemode": mode == payments.Live}
	d.objects[id] = dp
	return id, f.emitLocked(d, "charge.dispute.created", dp, nil, mode == payments.Live), nil
}

// CloseDispute closes a dispute, won or lost, and sends charge.dispute.closed.
func (f *Fake) CloseDispute(mode payments.Mode, disputeID string, won bool) (Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := f.modes[mode]
	dp, ok := d.objects[disputeID]
	if !ok {
		return Event{}, fmt.Errorf("stripetest: no dispute %s", disputeID)
	}
	prev := obj{"status": dp["status"]}
	dp["status"] = "lost"
	if won {
		dp["status"] = "won"
	}
	return f.emitLocked(d, "charge.dispute.closed", dp, prev, mode == payments.Live), nil
}
