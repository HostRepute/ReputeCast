package gifts

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// The payment events of gifts, from Earn's webhook ingest (earn.OneTimeHook):
// a gift's checkout paid, and its payment refunded. They're recorded in
// the payments event log with their outcome, not in the audit log.

// CheckoutCompleted marks a gift paid when its checkout's payment went
// through; its receipt and delivery follow from the gifts job.
func (s *Service) CheckoutCompleted(ctx context.Context, tx pgx.Tx, ws *store.Workspace, mode string, cs payments.CheckoutSession, at time.Time) (bool, map[string]any, error) {
	q := store.New(tx)
	g, err := q.GiftBySessionForUpdate(ctx, store.GiftBySessionForUpdateParams{WorkspaceID: ws.ID, Mode: mode, Session: cs.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	detail := map[string]any{"gift": g.ID.String(), "months": g.Months}
	if err := s.markPaid(ctx, q, ws, g, cs, at); err != nil {
		return true, detail, err
	}
	return true, detail, nil
}

// markPaid moves a gift from checkout to paid (once), with its event and
// the gift.purchased webhook.
func (s *Service) markPaid(ctx context.Context, q *store.Queries, ws *store.Workspace, g store.EarnGift, cs payments.CheckoutSession, at time.Time) error {
	if g.Status != StatusCheckout && g.Status != StatusAbandoned {
		return nil
	}
	// An asynchronous payment (a bank debit) completes the checkout before
	// it clears: the gift waits for it.
	if cs.PaymentStatus != "" && cs.PaymentStatus != "paid" && cs.PaymentStatus != "no_payment_required" {
		return nil
	}
	if err := q.GiftPaid(ctx, store.GiftPaidParams{At: at, PaymentIntent: cs.PaymentIntentID, StripeCustomer: cs.CustomerID, BuyerEmail: cs.CustomerEmail, ID: g.ID}); err != nil {
		return err
	}
	if err := event(ctx, q, g.ID, "paid", at, nil, nil, map[string]any{"amountCents": g.AmountCents, "currency": g.Currency, "session": cs.ID}); err != nil {
		return err
	}
	tier, _, _ := s.tierOf(ctx, ws, g)
	ev := GiftEvent{GiftID: g.ID.String(), Mode: g.Mode, Tier: tier, Months: int(g.Months), AmountCents: g.AmountCents, Currency: g.Currency,
		BuyerEmail: firstNonEmpty(g.BuyerEmail, cs.CustomerEmail), InPerson: g.InPerson, DeliverAt: g.DeliverAt, OccurredAt: at.UTC(), RecipientName: g.RecipientName}
	return s.emit(ctx, q, ws, "gift.purchased", ev)
}

// Refunded marks a gift refunded when its payment is refunded in full:
// its link stops working and a scheduled gift isn't sent. A partial
// refund is noted on the gift only.
func (s *Service) Refunded(ctx context.Context, tx pgx.Tx, ws *store.Workspace, mode string, ch payments.Charge, at time.Time) (bool, map[string]any, error) {
	q := store.New(tx)
	g, err := q.GiftByIntentForUpdate(ctx, store.GiftByIntentForUpdateParams{WorkspaceID: ws.ID, Mode: mode, PaymentIntent: ch.PaymentIntentID})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, err
	}
	detail := map[string]any{"gift": g.ID.String(), "months": g.Months}
	refunded := ch.AmountRefunded
	if refunded == 0 {
		refunded = ch.RefundAmount
	}
	if refunded < g.AmountCents {
		detail["partial"] = true
		return true, detail, event(ctx, q, g.ID, "partly_refunded", at, nil, nil, map[string]any{"amountCents": refunded, "currency": ch.Currency})
	}
	if g.Status == StatusRefunded {
		return true, detail, nil
	}
	if err := q.GiftRefunded(ctx, store.GiftRefundedParams{At: at, ID: g.ID}); err != nil {
		return true, detail, err
	}
	return true, detail, event(ctx, q, g.ID, "refunded", at, nil, nil, map[string]any{"amountCents": refunded, "currency": ch.Currency, "was": g.Status})
}
