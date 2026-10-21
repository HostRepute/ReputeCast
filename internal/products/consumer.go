package products

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// Stripe's events about one-time payments reach products through Earn's
// webhook pipeline (earn.RegisterPaymentConsumer): a completed checkout
// becomes a paid order with its delivery email, a refund or a dispute
// updates the order and, when the product says so, revokes its downloads.
// Each is applied once, inside the transaction that files the event.

func init() {
	webhooks.Events = append(webhooks.Events,
		webhooks.WebhookEventInfo{Name: "order.paid", Group: "Products and orders", Description: "A buyer paid for a digital product; the delivery email is on its way."},
		webhooks.WebhookEventInfo{Name: "order.refunded", Group: "Products and orders", Description: "An order was refunded in Stripe, in full or in part."},
		webhooks.WebhookEventInfo{Name: "order.disputed", Group: "Products and orders", Description: "A buyer disputed an order's payment with their bank."},
	)
}

// Flows makes contacts leave "Bought a product" automations when their
// payment is refunded (internal/automations).
type Flows interface {
	Refunded(ctx context.Context, tx pgx.Tx, wsID, contactID, productID uuid.UUID, at time.Time) error
}

// systemActor writes what Stripe's events change in the audit log.
var systemActor = &audit.Actor{Type: authz.ActorSystem, Name: "System"}

// Consume is the payment consumer (registered with earn.RegisterPaymentConsumer).
func (s *Service) Consume(ctx context.Context, tx pgx.Tx, e earn.PaymentEvent) (earn.Consumed, bool, error) {
	switch e.Payment.Kind {
	case payments.PaymentCompleted:
		return s.consumePaid(ctx, tx, e)
	case payments.PaymentRefunded:
		return s.consumeRefund(ctx, tx, e)
	case payments.PaymentDisputeOpened, payments.PaymentDisputeUpdated, payments.PaymentDisputeClosed:
		return s.consumeDispute(ctx, tx, e)
	}
	return earn.Consumed{}, false, nil
}

// checkoutOf finds the product checkout a payment belongs to.
func checkoutOf(ctx context.Context, q *store.Queries, p payments.Payment) (store.ProductCheckout, bool, error) {
	ref := p.Metadata["reputecast_checkout"]
	if ref == "" {
		ref = p.ClientRef
	}
	if id, err := uuid.Parse(ref); err == nil {
		c, err := q.ProductCheckoutGet(ctx, id)
		if err == nil {
			return c, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return c, false, err
		}
	}
	if p.SessionID != "" {
		c, err := q.ProductCheckoutBySession(ctx, p.SessionID)
		if err == nil {
			return c, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return c, false, err
		}
	}
	return store.ProductCheckout{}, false, nil
}

func (s *Service) consumePaid(ctx context.Context, tx pgx.Tx, e earn.PaymentEvent) (earn.Consumed, bool, error) {
	q := store.New(tx)
	co, ok, err := checkoutOf(ctx, q, e.Payment)
	if err != nil || !ok {
		return earn.Consumed{}, false, err
	}
	if co.WorkspaceID != e.Workspace.ID {
		return earn.Consumed{}, false, nil
	}
	if !e.Payment.Paid {
		// A delayed payment method (a bank debit): the order waits for
		// checkout.session.async_payment_succeeded.
		return earn.Consumed{Outcome: "payment_pending", Ignored: true, Detail: map[string]any{"session": e.Payment.SessionID}}, true, nil
	}
	o, created, err := s.applyPaid(ctx, tx, e.Workspace, co, e.Payment, e.Type)
	if err != nil {
		return earn.Consumed{}, true, err
	}
	amount := o.AmountCents
	out := earn.Consumed{Outcome: "order_paid", Detail: map[string]any{"order": o.Number, "product": o.ProductName}, ContactID: o.ContactID,
		AmountCents: &amount, Currency: o.Currency}
	if !created {
		out.Outcome, out.Ignored = "order_paid_already", true
	}
	return out, true, nil
}

// applyPaid turns a paid checkout into an order, once: its buyer and
// consent, its numbers, its first download link and delivery email, the
// "Bought a product" trigger and the order.paid webhook.
func (s *Service) applyPaid(ctx context.Context, tx pgx.Tx, ws *store.Workspace, co store.ProductCheckout, pay payments.Payment, eventType string) (store.ProductOrder, bool, error) {
	q := store.New(tx)
	if o, err := q.ProductOrderBySession(ctx, pay.SessionID); err == nil {
		return o, false, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return o, false, err
	}
	if !pay.Paid {
		return store.ProductOrder{}, false, fmt.Errorf("products: checkout %s isn't paid", pay.SessionID)
	}
	p, err := q.ProductAny(ctx, co.ProductID)
	if err != nil {
		return store.ProductOrder{}, false, err
	}
	l, ok := liveOf(p)
	if !ok {
		l = draftOf(p)
	}
	now := s.now()
	email := strings.ToLower(strings.TrimSpace(pay.CustomerEmail))
	if email == "" {
		email = strings.ToLower(co.Email)
	}
	var consent checkoutConsent
	_ = json.Unmarshal(co.Consent, &consent)

	// The buyer: a contact already, a contact now through the list offer,
	// or nobody's contact (the order keeps the email for delivery only).
	buyer := "none"
	var contactID *uuid.UUID
	if c, err := q.ProductBuyerByEmail(ctx, store.ProductBuyerByEmailParams{WorkspaceID: ws.ID, Email: email}); err == nil {
		id := c.ID
		contactID, buyer = &id, "contact"
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return store.ProductOrder{}, false, err
	}
	var confirm *uuid.UUID
	if consent.Ticked && consent.ListID != "" && s.Audience != nil && s.Audience() != nil {
		lid, _ := uuid.Parse(consent.ListID)
		lst, err := q.GetList(ctx, store.GetListParams{WorkspaceID: ws.ID, ID: lid})
		if err == nil {
			var ip *netip.Addr
			if a, err := netip.ParseAddr(consent.IP); err == nil {
				ip = &a
			}
			first, last := splitName(pay.CustomerName)
			res, err := s.Audience().Signup(ctx, tx, ws, audience.SignupInput{Email: email, FirstName: first, LastName: last, Lists: []uuid.UUID{lid},
				DoubleOptIn: lst.DoubleOptIn, Evidence: audience.PublicEvidence{Source: "product_checkout:" + l.Slug, IP: ip, ConsentText: consent.Text,
					Note: word(i18n.For(i18n.English), "consent.note", map[string]any{"text": consent.Text}), At: consent.at(now)}})
			if err != nil {
				var fe *audience.FieldError
				if !errors.As(err, &fe) {
					return store.ProductOrder{}, false, err
				}
			} else if !res.Blocked {
				id := res.ContactID
				if contactID == nil {
					buyer = "joined"
				}
				contactID = &id
				if res.NeedsConfirmation() {
					confirm = &id
				}
			}
		}
	}
	number, err := q.ProductOrderNextNumber(ctx, ws.ID)
	if err != nil {
		return store.ProductOrder{}, false, err
	}
	var campaign, automation *uuid.UUID
	by := ""
	if contactID != nil {
		campaign, automation, by = s.attribute(ctx, q, ws, *contactID, now)
	}
	amount := pay.AmountCents - pay.TaxCents
	if amount < 0 {
		amount = pay.AmountCents
	}
	currency := pay.Currency
	if currency == "" {
		currency = p.Currency
	}
	id := newID()
	started := co.CreatedAt
	o := store.ProductOrderInsertParams{ID: id, WorkspaceID: ws.ID, Number: number, ProductID: p.ID, ProductName: l.Name, Mode: co.Mode,
		ConnectionID: co.ConnectionID, CheckoutID: &co.ID, BuyerEmail: email, BuyerName: strings.TrimSpace(pay.CustomerName), ContactID: contactID, Buyer: buyer,
		AmountCents: amount, TaxCents: pay.TaxCents, Currency: currency, PayWhatYouWant: l.PriceKind == PricePWYW, Provider: "stripe",
		SessionID: pay.SessionID, PaymentIntent: pay.PaymentIntentID, ChargeID: pay.ChargeID, Source: co.Source, SourcePageID: co.SourcePageID,
		SourceUrl: co.SourceUrl, Consent: co.Consent, CampaignID: campaign, AutomationID: automation, AttributedBy: by, CheckoutStartedAt: &started, PaidAt: now}
	if err := q.ProductOrderInsert(ctx, o); err != nil {
		return store.ProductOrder{}, false, err
	}
	if err := q.ProductCheckoutCompleted(ctx, store.ProductCheckoutCompletedParams{ID: co.ID, OrderID: &id, At: now}); err != nil {
		return store.ProductOrder{}, false, err
	}
	if err := q.PageConversionInsert(ctx, store.PageConversionInsertParams{ID: newID(), WorkspaceID: ws.ID, PageID: co.SourcePageID, Source: co.Source,
		Kind: "sale", ContactID: contactID, OrderID: &id, AmountCents: amount, OccurredAt: now}); err != nil {
		return store.ProductOrder{}, false, err
	}
	if err := s.event(ctx, q, id, "checkout_started", started, nil, map[string]any{"page": consent.Page, "ticked": consent.Ticked}); err != nil {
		return store.ProductOrder{}, false, err
	}
	if err := s.event(ctx, q, id, "paid", now, nil, map[string]any{"stripe": firstNonEmpty(eventType, "checkout.session.completed")}); err != nil {
		return store.ProductOrder{}, false, err
	}
	order, err := q.ProductOrderGet(ctx, store.ProductOrderGetParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return order, false, err
	}
	if _, err := s.newLink(ctx, tx, ws, order, "purchase", nil); err != nil {
		return order, false, err
	}
	if confirm != nil && s.Jobs != nil && s.Jobs() != nil {
		if _, err := s.Jobs().InsertTx(ctx, tx, ConfirmArgs{WorkspaceID: ws.ID, ContactID: *confirm}, nil); err != nil {
			return order, false, err
		}
	}
	if contactID != nil {
		// liquid is what the flow's emails can write ({{ order.product_name }}).
		payload, _ := json.Marshal(map[string]any{"product_id": p.ID.String(), "product_name": l.Name, "order_id": id.String(), "order_number": number,
			"amount": moneyText(amount, currency), "amount_cents": amount, "currency": currency, "test": co.Mode == string(payments.Test),
			"liquid": map[string]any{"order": map[string]any{"product_name": l.Name, "amount": moneyText(amount, currency), "number": number,
				"currency": strings.ToUpper(currency), "paid_on": dayText(ws, now)}}})
		if err := q.InsertAutomationEvent(ctx, store.InsertAutomationEventParams{WorkspaceID: ws.ID, Kind: "bought_product", ContactID: *contactID,
			Ref: "product:" + p.ID.String(), Payload: payload, OccurredAt: now}); err != nil {
			return order, false, err
		}
	}
	if _, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "order.paid", orderHook(order)); err != nil {
		return order, false, err
	}
	return order, true, nil
}

// orderHook is an order in webhook deliveries (no consent IP, no links).
func orderHook(o store.ProductOrder) map[string]any {
	out := map[string]any{"order": o.Number, "id": o.ID.String(), "product_id": o.ProductID.String(), "product": o.ProductName, "email": o.BuyerEmail,
		"amount_cents": o.AmountCents, "currency": o.Currency, "status": o.Status, "test": o.Mode == string(payments.Test), "paid_at": o.PaidAt.UTC().Format(time.RFC3339)}
	if o.ContactID != nil {
		out["contact_id"] = o.ContactID.String()
	}
	if o.RefundedCents > 0 {
		out["refunded_cents"] = o.RefundedCents
	}
	if o.DisputeStatus != "" {
		out["dispute"] = map[string]any{"status": o.DisputeStatus, "reason": o.DisputeReason}
	}
	return out
}

// attribute credits a sale to the email the buyer clicked last within the
// attribution window.
func (s *Service) attribute(ctx context.Context, q *store.Queries, ws *store.Workspace, contactID uuid.UUID, now time.Time) (*uuid.UUID, *uuid.UUID, string) {
	days := int32(7)
	if st, err := q.EarnSettings(ctx, ws.ID); err == nil && st.AttributionDays > 0 {
		days = st.AttributionDays
	}
	c, err := q.ProductLastClick(ctx, store.ProductLastClickParams{WorkspaceID: ws.ID, ContactID: &contactID, Since: now.Add(-time.Duration(days) * 24 * time.Hour), Until: now})
	if err != nil {
		return nil, nil, ""
	}
	return c.CampaignID, c.AutomationID, "email"
}

func splitName(name string) (string, string) {
	name = strings.TrimSpace(name)
	if i := strings.IndexByte(name, ' '); i > 0 {
		return name[:i], strings.TrimSpace(name[i+1:])
	}
	return name, ""
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// orderOf finds the order a refund or dispute is about.
func orderOf(ctx context.Context, q *store.Queries, wsID uuid.UUID, p payments.Payment) (store.ProductOrder, bool, error) {
	if p.PaymentIntentID != "" {
		o, err := q.ProductOrderByIntent(ctx, store.ProductOrderByIntentParams{WorkspaceID: wsID, PaymentIntent: p.PaymentIntentID})
		if err == nil {
			return o, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return o, false, err
		}
	}
	if p.ChargeID != "" {
		o, err := q.ProductOrderByCharge(ctx, store.ProductOrderByChargeParams{WorkspaceID: wsID, ChargeID: p.ChargeID})
		if err == nil {
			return o, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return o, false, err
		}
	}
	return store.ProductOrder{}, false, nil
}

func (s *Service) consumeRefund(ctx context.Context, tx pgx.Tx, e earn.PaymentEvent) (earn.Consumed, bool, error) {
	q := store.New(tx)
	o, ok, err := orderOf(ctx, q, e.Workspace.ID, e.Payment)
	if err != nil || !ok {
		return earn.Consumed{}, false, err
	}
	ws := e.Workspace
	now := s.now()
	refunded := e.Payment.RefundedCents
	if refunded > o.AmountCents+o.TaxCents {
		refunded = o.AmountCents + o.TaxCents
	}
	// Refunds count against the price, tax aside.
	if refunded > o.AmountCents {
		refunded = o.AmountCents
	}
	delta := refunded - o.RefundedCents
	if delta <= 0 {
		return earn.Consumed{Outcome: "order_refund_known", Ignored: true}, true, nil
	}
	if e.Payment.ChargeID != "" {
		_ = q.ProductOrderSetCharge(ctx, store.ProductOrderSetChargeParams{ID: o.ID, ChargeID: e.Payment.ChargeID})
	}
	if err := q.ProductOrderRefund(ctx, store.ProductOrderRefundParams{ID: o.ID, RefundedCents: refunded, At: now}); err != nil {
		return earn.Consumed{}, true, err
	}
	if err := s.event(ctx, q, o.ID, "refunded", now, nil, map[string]any{"stripe": e.Type, "amount": delta, "currency": o.Currency, "byStripe": true}); err != nil {
		return earn.Consumed{}, true, err
	}
	full := refunded >= o.AmountCents
	p, err := q.ProductAny(ctx, o.ProductID)
	if err != nil {
		return earn.Consumed{}, true, err
	}
	l, _ := liveOf(p)
	revoke := full && (l.RevokeOnRefund || len(p.Live) == 0) && o.AccessRevokedAt == nil
	if revoke {
		if err := q.ProductOrderRevoke(ctx, store.ProductOrderRevokeParams{ID: o.ID, At: now, Reason: "refund"}); err != nil {
			return earn.Consumed{}, true, err
		}
		if err := s.event(ctx, q, o.ID, "access_revoked", now, nil, map[string]any{"reason": "refund", "auto": true}); err != nil {
			return earn.Consumed{}, true, err
		}
	}
	if full && o.ContactID != nil && s.Flows != nil {
		if err := s.Flows.Refunded(ctx, tx, ws.ID, *o.ContactID, o.ProductID, s.now()); err != nil {
			return earn.Consumed{}, true, err
		}
	}
	after := map[string]any{"refunded_cents": refunded, "status": statusAfterRefund(o, refunded)}
	if revoke {
		after["access"] = "revoked"
	}
	if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Actor: systemActor, Action: "order.refunded", TargetType: "product_order",
		TargetID: o.ID.String(), TargetName: orderName(o), Before: map[string]any{"refunded_cents": o.RefundedCents, "status": o.Status}, After: after,
		Metadata: map[string]any{"source": "stripe", "summary": refundSummary(full, revoke, delta, o.Currency), "event": e.Type}}); err != nil {
		return earn.Consumed{}, true, err
	}
	updated, err := q.ProductOrderGet(ctx, store.ProductOrderGetParams{WorkspaceID: ws.ID, ID: o.ID})
	if err != nil {
		return earn.Consumed{}, true, err
	}
	if _, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "order.refunded", orderHook(updated)); err != nil {
		return earn.Consumed{}, true, err
	}
	neg := -delta
	return earn.Consumed{Outcome: "order_refunded", Detail: map[string]any{"order": o.Number, "revoked": revoke}, ContactID: o.ContactID, AmountCents: &neg, Currency: o.Currency}, true, nil
}

func statusAfterRefund(o store.ProductOrder, refunded int64) string {
	switch {
	case o.DisputeStatus == "open":
		return OrderDisputed
	case refunded >= o.AmountCents:
		return OrderRefunded
	}
	return OrderPartiallyRefunded
}

// disputeStatus is Stripe's dispute status in three words.
func disputeStatus(p payments.Payment) string {
	switch p.DisputeStatus {
	case "won":
		return "won"
	case "lost":
		return "lost"
	case "warning_closed", "charge_refunded":
		return "won"
	}
	if p.Kind == payments.PaymentDisputeClosed {
		return "lost"
	}
	return "open"
}

func (s *Service) consumeDispute(ctx context.Context, tx pgx.Tx, e earn.PaymentEvent) (earn.Consumed, bool, error) {
	q := store.New(tx)
	o, ok, err := orderOf(ctx, q, e.Workspace.ID, e.Payment)
	if err != nil || !ok {
		return earn.Consumed{}, false, err
	}
	ws := e.Workspace
	now := s.now()
	status := disputeStatus(e.Payment)
	reason := strings.ReplaceAll(firstNonEmpty(e.Payment.DisputeReason, o.DisputeReason), "_", " ")
	opened := o.DisputeStatus == ""
	if err := q.ProductOrderDispute(ctx, store.ProductOrderDisputeParams{ID: o.ID, DisputeID: firstNonEmpty(e.Payment.DisputeID, o.DisputeID), DisputeStatus: status,
		DisputeReason: reason, DisputeFeeCents: max(e.Payment.DisputeFee, o.DisputeFeeCents), At: now, DisputeDueBy: e.Payment.DisputeDueBy}); err != nil {
		return earn.Consumed{}, true, err
	}
	restored := false
	switch {
	case opened:
		if err := s.event(ctx, q, o.ID, "dispute_opened", now, nil, map[string]any{"stripe": e.Type, "reason": reason, "respondBy": e.Payment.DisputeDueBy}); err != nil {
			return earn.Consumed{}, true, err
		}
	case e.Payment.Kind == payments.PaymentDisputeClosed:
		if err := s.event(ctx, q, o.ID, "dispute_closed", now, nil, map[string]any{"stripe": e.Type, "outcome": status}); err != nil {
			return earn.Consumed{}, true, err
		}
		// A won dispute gives back the access the dispute took.
		if status == "won" && o.AccessRevokedAt != nil && o.AccessRevokeReason == "dispute" {
			if err := q.ProductOrderRestore(ctx, o.ID); err != nil {
				return earn.Consumed{}, true, err
			}
			if err := s.event(ctx, q, o.ID, "access_restored", now, nil, map[string]any{"reason": "dispute_won", "auto": true}); err != nil {
				return earn.Consumed{}, true, err
			}
			restored = true
		}
	default:
		return earn.Consumed{Outcome: "order_dispute_updated", Detail: map[string]any{"order": o.Number, "status": status}}, true, nil
	}
	revoked := false
	if opened {
		p, err := q.ProductAny(ctx, o.ProductID)
		if err != nil {
			return earn.Consumed{}, true, err
		}
		l, _ := liveOf(p)
		if (l.RevokeOnRefund || len(p.Live) == 0) && o.AccessRevokedAt == nil {
			if err := q.ProductOrderRevoke(ctx, store.ProductOrderRevokeParams{ID: o.ID, At: now, Reason: "dispute"}); err != nil {
				return earn.Consumed{}, true, err
			}
			revoked = true
			if err := s.event(ctx, q, o.ID, "access_revoked", now, nil, map[string]any{"reason": "dispute", "auto": true}); err != nil {
				return earn.Consumed{}, true, err
			}
		}
		lang := i18n.For(i18n.English)
		body := word(lang, "notify.disputeBodyNoDue", map[string]any{"email": o.BuyerEmail, "order": orderNo(o.Number), "product": o.ProductName,
			"amount": moneyText(o.AmountCents, o.Currency), "reason": reason})
		if e.Payment.DisputeDueBy != nil {
			body = word(lang, "notify.disputeBody", map[string]any{"email": o.BuyerEmail, "order": orderNo(o.Number), "product": o.ProductName,
				"amount": moneyText(o.AmountCents, o.Currency), "reason": reason, "due": dayText(ws, *e.Payment.DisputeDueBy)})
		}
		if revoked {
			body += " " + word(lang, "notify.disputeRevoked", nil)
		}
		n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindOrderDisputed, Severity: "bad", Title: word(lang, "notify.disputeTitle", nil), Body: body,
			Link: "/w/" + ws.Slug + "/earn/products?order=" + o.ID.String(), Data: map[string]any{"order_id": o.ID.String(), "dispute": e.Payment.DisputeID}}
		if err := notify.ToMembersWith(ctx, tx, ws.ID, authz.OrdersManage, n); err != nil {
			return earn.Consumed{}, true, err
		}
	}
	action := "order.disputed"
	if !opened {
		action = "order.dispute_closed"
	}
	after := map[string]any{"dispute": status, "reason": reason}
	if revoked {
		after["access"] = "revoked"
	}
	if restored {
		after["access"] = "restored"
	}
	if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Actor: systemActor, Action: action, TargetType: "product_order", TargetID: o.ID.String(),
		TargetName: orderName(o), Before: map[string]any{"dispute": o.DisputeStatus, "status": o.Status}, After: after,
		Metadata: map[string]any{"source": "stripe", "event": e.Type}}); err != nil {
		return earn.Consumed{}, true, err
	}
	updated, err := q.ProductOrderGet(ctx, store.ProductOrderGetParams{WorkspaceID: ws.ID, ID: o.ID})
	if err != nil {
		return earn.Consumed{}, true, err
	}
	if opened {
		if _, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, "order.disputed", orderHook(updated)); err != nil {
			return earn.Consumed{}, true, err
		}
	}
	key := "order_disputed"
	if !opened {
		key = "order_dispute_closed"
	}
	return earn.Consumed{Outcome: key, Detail: map[string]any{"order": o.Number, "status": status, "revoked": revoked}, ContactID: o.ContactID}, true, nil
}

// checkoutConsent is the list offer's state as it travelled with the session.
type checkoutConsent struct {
	Offered bool       `json:"offered"`
	Ticked  bool       `json:"ticked"`
	Text    string     `json:"text"`
	ListID  string     `json:"listId"`
	Page    string     `json:"page"`
	IP      string     `json:"ip"`
	At      *time.Time `json:"at"`
}

func (c checkoutConsent) at(now time.Time) time.Time {
	if c.At != nil {
		return *c.At
	}
	return now
}

// refundSummary is the audit entry's line: "Refunded in Stripe · download
// access revoked, as the product says".
func refundSummary(full, revoked bool, delta int64, currency string) string {
	switch {
	case !full:
		return fmt.Sprintf("Partial refund of %s in Stripe · downloads keep working", moneyText(delta, currency))
	case revoked:
		return "Refunded in Stripe · download access revoked, as the product says"
	}
	return "Refunded in Stripe · downloads keep working, as the product says"
}
