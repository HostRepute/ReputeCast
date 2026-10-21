package stripetest

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hostrepute/reputecast/internal/payments"
)

// Coupons, promotion codes, a coupon applied to a subscription and a
// checkout for one payment (offers and gifts, X17).

func (f *Fake) createCoupon(w http.ResponseWriter, d *data, form url.Values, live bool) {
	id := form.Get("id")
	if id == "" {
		id = newID("co")
	}
	if _, exists := d.objects[id]; exists {
		stripeError(w, 400, "invalid_request_error", "resource_already_exists", "Coupon already exists.")
		return
	}
	pct, _ := strconv.ParseFloat(form.Get("percent_off"), 64)
	amount, _ := strconv.ParseInt(form.Get("amount_off"), 10, 64)
	months, _ := strconv.Atoi(form.Get("duration_in_months"))
	o := obj{"id": id, "object": "coupon", "name": nilIfEmpty(form.Get("name")), "duration": form.Get("duration"), "valid": true, "times_redeemed": int64(0),
		"metadata": prefixed(form, "metadata"), "livemode": live, "created": f.now().Unix(), "percent_off": nil, "amount_off": nil, "currency": nil, "duration_in_months": nil}
	if pct > 0 {
		o["percent_off"] = pct
	}
	if amount > 0 {
		o["amount_off"], o["currency"] = amount, form.Get("currency")
	}
	if months > 0 {
		o["duration_in_months"] = months
	}
	var products []string
	for i := 0; ; i++ {
		v := form.Get(fmt.Sprintf("applies_to[products][%d]", i))
		if v == "" {
			break
		}
		products = append(products, v)
	}
	if len(products) > 0 {
		o["applies_to"] = obj{"products": products}
	}
	d.objects[id] = o
	writeJSON(w, 200, public(o))
}

func (f *Fake) createPromotionCode(w http.ResponseWriter, d *data, form url.Values, live bool) {
	coupon, ok := d.objects[form.Get("coupon")]
	if !ok || coupon["object"] != "coupon" {
		stripeError(w, 400, "invalid_request_error", "resource_missing", "No such coupon: '"+form.Get("coupon")+"'")
		return
	}
	code := strings.ToUpper(form.Get("code"))
	if code == "" {
		code = strings.ToUpper(newID("C")[2:10])
	}
	for _, o := range d.objects {
		if o["object"] == "promotion_code" && o["active"] == true && o["code"] == code {
			stripeError(w, 400, "invalid_request_error", "resource_already_exists", "An active promotion code with `code: "+code+"` already exists.")
			return
		}
	}
	id := newID("promo")
	o := obj{"id": id, "object": "promotion_code", "code": code, "coupon": public(coupon), "active": true, "times_redeemed": int64(0), "max_redemptions": nil,
		"expires_at": nil, "restrictions": obj{"first_time_transaction": form.Get("restrictions[first_time_transaction]") == "true"},
		"metadata": prefixed(form, "metadata"), "livemode": live, "created": f.now().Unix()}
	if n, err := strconv.Atoi(form.Get("max_redemptions")); err == nil && n > 0 {
		o["max_redemptions"] = int64(n)
	}
	if t, err := strconv.ParseInt(form.Get("expires_at"), 10, 64); err == nil && t > 0 {
		o["expires_at"] = t
	}
	d.objects[id] = o
	writeJSON(w, 200, public(o))
}

// applyCoupon puts a coupon on a subscription and sends the update.
func (f *Fake) applyCoupon(d *data, sub obj, couponID string, live bool) {
	coupon, ok := d.objects[couponID]
	if !ok {
		coupon = obj{"id": couponID, "object": "coupon", "valid": true}
	}
	now := f.now().Unix()
	disc := obj{"object": "discount", "coupon": public(coupon), "start": now, "end": nil}
	if m := num(coupon["duration_in_months"]); m > 0 && coupon["duration"] == "repeating" {
		disc["end"] = now + m*30*86400
	}
	prev := obj{"discount": sub["discount"]}
	sub["discount"] = disc
	f.emitLocked(d, "customer.subscription.updated", sub, prev, live)
}

// CompletePayment pays a one-time checkout (a gift): Stripe sends
// checkout.session.completed with the payment.
func (f *Fake) CompletePayment(mode payments.Mode, sessionID, email string) (Event, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := f.modes[mode]
	cs, ok := d.objects[sessionID]
	if !ok || cs["mode"] != "payment" {
		return Event{}, fmt.Errorf("stripetest: no payment checkout session %s", sessionID)
	}
	if email == "" {
		email, _ = cs["customer_email"].(string)
	}
	cus := newID("cus")
	d.objects[cus] = obj{"id": cus, "object": "customer", "email": email, "created": f.now().Unix(), "livemode": mode == payments.Live}
	pi := newID("pi")
	cs["status"], cs["payment_status"], cs["customer"], cs["payment_intent"] = "complete", "paid", cus, pi
	cs["customer_details"] = obj{"email": email}
	return f.emitLocked(d, "checkout.session.completed", cs, nil, mode == payments.Live), nil
}

// RefundPayment refunds a one-time payment in full or in part: Stripe sends
// charge.refunded with the charge of the payment.
func (f *Fake) RefundPayment(mode payments.Mode, paymentIntent string, amount int64) Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	d := f.modes[mode]
	var total int64
	currency := f.Account.Currency
	cus := ""
	for _, o := range d.objects {
		if o["object"] == "checkout.session" && o["payment_intent"] == paymentIntent {
			total, cus = num(o["amount_total"]), fmt.Sprint(o["customer"])
			if c, ok := o["currency"].(string); ok && c != "" {
				currency = c
			}
		}
	}
	if amount <= 0 {
		amount = total
	}
	ch := obj{"id": newID("ch"), "object": "charge", "customer": cus, "payment_intent": paymentIntent, "amount": total, "amount_refunded": amount,
		"currency": currency, "invoice": nil, "livemode": mode == payments.Live}
	return f.emitLocked(d, "charge.refunded", ch, obj{"amount_refunded": int64(0)}, mode == payments.Live)
}
