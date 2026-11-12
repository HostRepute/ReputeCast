package sponsors

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/earn"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Paying for an approved request: the status page's Pay button opens a
// Stripe Checkout session for the booking's price (one payment, the
// sponsor's address filled in), refused once the deadline passed. Stripe's
// event reaches the booking through Earn's webhook pipeline
// (earn.RegisterPaymentConsumer): paid, the issue is booked and its
// creative ready, and the sponsor is told. Stripe emails the receipt.
// Payment events go to the payment event log, never the audit log.

// MetaBooking is the Checkout session's metadata naming the booking.
const MetaBooking = "reputecast_sponsor_booking"

// ErrPayClosed is a payment asked for a booking that can't be paid any
// more (not approved, or past its deadline).
var ErrPayClosed = errors.New("sponsors: the booking can't be paid")

// ErrPaymentsUnavailable is a checkout that couldn't be opened.
var ErrPaymentsUnavailable = errors.New("sponsors: payments are unavailable")

// Pay opens the checkout of an approved request; back is its status page's
// address, where Stripe returns the sponsor.
func (s *Service) Pay(ctx context.Context, ws *store.Workspace, token, back string) (string, error) {
	st, err := s.Status(ctx, ws, token)
	if err != nil {
		return "", err
	}
	if !st.CanPay {
		return "", ErrPayClosed
	}
	if s.Earn == nil || s.Earn() == nil {
		return "", ErrPaymentsUnavailable
	}
	sel, ok, err := s.Earn().Selling(ctx, ws)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrPaymentsUnavailable
	}
	client, err := s.Earn().SellingClient(ctx, ws, sel.ID)
	if err != nil {
		return "", err
	}
	one, ok := client.(payments.OneTimePayments)
	if !ok {
		return "", ErrPaymentsUnavailable
	}
	b, err := s.Q.SponsorBookingGet(ctx, store.SponsorBookingGetParams{ID: st.id, WorkspaceID: ws.ID})
	if err != nil {
		return "", err
	}
	d, _ := parseRequest(b.Request)
	l := requestLang(d)
	args := i18n.Args{"list": st.List, "issue": st.Issue, "slot": st.Slot, "date": shortDate(l, st.IssueOn.In(location(ws)))}
	cs, err := one.CreatePaymentCheckout(ctx, payments.PaymentCheckoutRequest{AmountCents: b.PriceCents, Currency: b.Currency,
		Name: msgs.M("checkout.name").Text(l, args), Description: msgs.M("checkout.description").Text(l, args), CustomerEmail: d.Email,
		Tax: sel.TaxAtCheckout, SuccessURL: back + "?paid=1", CancelURL: back, ClientRef: b.ID.String(), Metadata: map[string]string{MetaBooking: b.ID.String()}})
	if err != nil {
		s.log().WarnContext(ctx, "sponsor checkout", "booking", b.ID, "err", err)
		return "", ErrPaymentsUnavailable
	}
	if err := s.Q.SponsorBookingSetCheckout(ctx, store.SponsorBookingSetCheckoutParams{ID: b.ID, CheckoutID: cs.ID, At: s.now()}); err != nil {
		return "", err
	}
	return cs.URL, nil
}

// Returned is the status page after Stripe sent the sponsor back: a
// payment Stripe has but whose event hasn't arrived yet is applied now (the
// event then finds it applied). It reports whether the booking is paid.
func (s *Service) Returned(ctx context.Context, ws *store.Workspace, token string) (bool, error) {
	b, err := s.Q.SponsorBookingByToken(ctx, store.SponsorBookingByTokenParams{WorkspaceID: ws.ID, TokenHash: authn.HashToken(token)})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNoBooking
	}
	if err != nil {
		return false, err
	}
	if b.PaymentStatus == "paid" {
		return true, nil
	}
	if b.CheckoutID == "" || s.Earn == nil || s.Earn() == nil {
		return false, nil
	}
	sel, ok, err := s.Earn().Selling(ctx, ws)
	if err != nil || !ok {
		return false, err
	}
	client, err := s.Earn().SellingClient(ctx, ws, sel.ID)
	if err != nil {
		return false, err
	}
	reader, ok := client.(payments.OneTimePrices)
	if !ok {
		return false, nil
	}
	pay, err := reader.GetPayment(ctx, b.CheckoutID)
	if err != nil || !pay.Paid || pay.Metadata[MetaBooking] != b.ID.String() {
		return false, nil //nolint:nilerr // the webhook applies it once it arrives
	}
	paid := false
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		bk, err := store.New(tx).SponsorBookingLock(ctx, b.ID)
		if err != nil {
			return err
		}
		out, err := s.applyPaid(ctx, tx, ws, bk, pay)
		paid = err == nil && out.Outcome != ""
		return err
	})
	return paid, err
}

// Consume is the payment consumer (earn.RegisterPaymentConsumer): the
// one-time payment events of sponsor bookings.
func (s *Service) Consume(ctx context.Context, tx pgx.Tx, e earn.PaymentEvent) (earn.Consumed, bool, error) {
	id, err := uuid.Parse(e.Payment.Metadata[MetaBooking])
	if err != nil {
		return earn.Consumed{}, false, nil //nolint:nilerr // not a sponsor booking's payment
	}
	q := store.New(tx)
	b, err := q.SponsorBookingLock(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return earn.Consumed{Outcome: "sponsor_booking_unknown", Ignored: true, Detail: map[string]any{"booking": id.String()}}, true, nil
	}
	if err != nil {
		return earn.Consumed{}, true, err
	}
	if b.WorkspaceID != e.Workspace.ID {
		return earn.Consumed{}, false, nil
	}
	detail := map[string]any{"booking": b.ID.String(), "issue": b.IssueNumber, "session": e.Payment.SessionID}
	switch e.Payment.Kind {
	case payments.PaymentCompleted:
		if !e.Payment.Paid {
			// A delayed payment method (a bank debit): the booking waits
			// for checkout.session.async_payment_succeeded.
			return earn.Consumed{Outcome: "sponsor_payment_pending", Ignored: true, Detail: detail}, true, nil
		}
		out, err := s.applyPaid(ctx, tx, e.Workspace, b, e.Payment)
		return out, true, err
	case payments.PaymentFailed:
		if err := s.event(ctx, q, b.ID, nil, "payment_failed", map[string]any{"session": e.Payment.SessionID}); err != nil {
			return earn.Consumed{}, true, err
		}
		return earn.Consumed{Outcome: "sponsor_payment_failed", Detail: detail}, true, nil
	case payments.PaymentRefunded:
		// Payments are final, but the publisher may still refund one in
		// Stripe: the booking keeps it in its activity.
		if err := s.event(ctx, q, b.ID, nil, "refunded", map[string]any{"amountCents": e.Payment.RefundCents, "refundedCents": e.Payment.RefundedCents}); err != nil {
			return earn.Consumed{}, true, err
		}
		amount := -e.Payment.RefundCents
		return earn.Consumed{Outcome: "sponsor_payment_refunded", Detail: detail, AmountCents: &amount, Currency: e.Payment.Currency}, true, nil
	case payments.PaymentDisputeOpened, payments.PaymentDisputeUpdated, payments.PaymentDisputeClosed:
		if err := s.event(ctx, q, b.ID, nil, "disputed", map[string]any{"status": e.Payment.DisputeStatus, "reason": e.Payment.DisputeReason}); err != nil {
			return earn.Consumed{}, true, err
		}
		return earn.Consumed{Outcome: "sponsor_payment_disputed", Detail: detail}, true, nil
	}
	return earn.Consumed{Outcome: "sponsor_payment_other", Ignored: true, Detail: detail}, true, nil
}

// applyPaid books a paid booking, once per checkout session: paid by
// Stripe, its creative ready, the sponsor emailed. A payment that arrives
// after the hold ended books the issue when it's still open; otherwise,
// and for a second payment, the booking keeps it and the people who manage
// sponsors are told to refund it in Stripe or book another issue.
func (s *Service) applyPaid(ctx context.Context, tx pgx.Tx, ws *store.Workspace, b store.SponsorBooking, pay payments.Payment) (earn.Consumed, error) {
	q := store.New(tx)
	amount := pay.AmountCents
	detail := map[string]any{"booking": b.ID.String(), "issue": b.IssueNumber, "session": pay.SessionID}
	paid := earn.Consumed{Detail: detail, AmountCents: &amount, Currency: pay.Currency}
	if b.PaymentStatus == "paid" {
		if b.PaidSource == "stripe" && b.CheckoutID == pay.SessionID {
			paid.Outcome, paid.Ignored = "sponsor_paid_already", true
			return paid, nil
		}
		// Paid again (two checkouts, or by hand before): the money is the
		// publisher's to send back.
		if err := s.event(ctx, q, b.ID, nil, "paid_again", map[string]any{"amountCents": amount, "session": pay.SessionID}); err != nil {
			return earn.Consumed{}, err
		}
		if err := s.warnPaid(ctx, tx, ws, b, "notify.paidTwiceTitle", "notify.paidTwiceBody", amount, pay.Currency); err != nil {
			return earn.Consumed{}, err
		}
		paid.Outcome = "sponsor_paid_twice"
		return paid, nil
	}
	now := s.now()
	status := b.Status
	switch b.Status {
	case "awaiting_payment":
		status = "booked"
	case "expired", "canceled", "declined":
		// Paid after the hold ended: book the issue if it's still open.
		if _, err := q.SponsorBookingOfIssue(ctx, store.SponsorBookingOfIssueParams{SlotID: b.SlotID, IssueNumber: b.IssueNumber}); errors.Is(err, pgx.ErrNoRows) &&
			!civil(b.IssueOn).Before(s.today(ws)) {
			status = "booked"
		} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return earn.Consumed{}, err
		}
	}
	paidAt := pay.Created
	if paidAt.IsZero() {
		paidAt = now
	}
	params := store.SponsorBookingStripePaidParams{ID: b.ID, Status: status, PaidCents: &amount, PaidOn: localDay(paidAt, ws), CheckoutID: pay.SessionID, At: now}
	// A booking the issue's other booking beat meanwhile keeps its status:
	// the savepoint keeps the event's transaction alive.
	sp, err := tx.Begin(ctx)
	if err != nil {
		return earn.Consumed{}, err
	}
	if err := store.New(sp).SponsorBookingStripePaid(ctx, params); err != nil {
		_ = sp.Rollback(ctx)
		if !isUnique(err) || status == b.Status {
			return earn.Consumed{}, err
		}
		params.Status, status = b.Status, b.Status
		if err := q.SponsorBookingStripePaid(ctx, params); err != nil {
			return earn.Consumed{}, err
		}
	} else if err := sp.Commit(ctx); err != nil {
		return earn.Consumed{}, err
	}
	if err := s.event(ctx, q, b.ID, nil, "paid", map[string]any{"amountCents": amount, "source": "stripe", "session": pay.SessionID}); err != nil {
		return earn.Consumed{}, err
	}
	if status != "booked" {
		if err := s.warnPaid(ctx, tx, ws, b, "notify.paidLateTitle", "notify.paidLateBody", amount, pay.Currency); err != nil {
			return earn.Consumed{}, err
		}
		paid.Outcome = "sponsor_paid_late"
		return paid, nil
	}
	if err := s.event(ctx, q, b.ID, nil, "booked", map[string]any{"priceCents": b.PriceCents, "paidBy": "stripe"}); err != nil {
		return earn.Consumed{}, err
	}
	d, _ := parseRequest(b.Request)
	args := i18n.Args{"company": firstNonEmpty(d.Company, ""), "issue": b.IssueNumber, "amount": money(amount, pay.Currency)}
	if err := notify.ToMembersWith(ctx, tx, ws.ID, authz.SponsorsManage, notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindSponsorRequest, Severity: "ok",
		TitleText: msgs.M("notify.paidTitle").Keep(args), BodyText: msgs.M("notify.paidBody").Keep(args), Link: "/w/" + ws.Slug + "/earn/sponsors?booking=" + b.ID.String(),
		Data: map[string]any{"booking": b.ID.String()}}); err != nil {
		return earn.Consumed{}, err
	}
	if err := s.queueMail(ctx, tx, MailArgs{WorkspaceID: ws.ID, BookingID: b.ID, Email: MailBooked}); err != nil {
		return earn.Consumed{}, err
	}
	paid.Outcome = "sponsor_paid"
	return paid, nil
}

// warnPaid tells the people who manage sponsors about a payment the
// booking can't use: theirs to refund in Stripe, or to book another issue
// with.
func (s *Service) warnPaid(ctx context.Context, tx pgx.Tx, ws *store.Workspace, b store.SponsorBooking, title, body string, amount int64, currency string) error {
	d, _ := parseRequest(b.Request)
	args := i18n.Args{"company": d.Company, "issue": b.IssueNumber, "amount": money(amount, currency)}
	return notify.ToMembersWith(ctx, tx, ws.ID, authz.SponsorsManage, notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindSponsorPayment, Severity: "warn",
		TitleText: msgs.M(title).Keep(args), BodyText: msgs.M(body).Keep(args), Link: "/w/" + ws.Slug + "/earn/sponsors?booking=" + b.ID.String(),
		Data: map[string]any{"booking": b.ID.String()}})
}

// payDue is when an approved request's payment must arrive: pay_days
// after the approval, and at the latest when its creative is due (the
// start of that day in the workspace's time zone).
func payDue(now time.Time, payDays int32, issueOn time.Time, creativeDays int32, ws *store.Workspace) time.Time {
	due := now.Add(time.Duration(payDays) * 24 * time.Hour)
	creative, _ := dayRange(civil(issueOn).AddDate(0, 0, -int(creativeDays)), ws)
	if creative.Before(due) {
		return creative
	}
	return due
}
