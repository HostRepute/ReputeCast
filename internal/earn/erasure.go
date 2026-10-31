package earn

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/erasure"
)

var mErasePaidMembers = msgs.M("erase.paidMembers")

// Erasure is a contact's paid membership. A contact who still pays through
// Stripe can't be erased: Stripe would go on charging someone ReputeCast no
// longer knows, so the subscription is canceled first (Earn › Members).
// Complimentary and gift access ends. Subscriptions stay, without the
// billing address, Stripe ids or anything a person wrote, so MRR, churn and
// revenue history still add up; the paid timeline goes, as the Earn brief
// says (§3.16).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "earn",
		Tables: []erasure.Table{
			{Name: "paid_subscriptions", Action: "anonymize",
				Columns: []string{"billing_email", "stripe_customer_id", "stripe_subscription_id", "cancel_reason", "grant_note", "failure_reason"},
				Kept:    map[string]string{"contact_id": "MRR, churn and revenue history (earn_mrr_changes cascades from these rows); the contact is scrubbed"}},
			{Name: "paid_events", Action: "delete"},
			{Name: "payment_events", Action: "clear", Columns: []string{"payload"},
				Kept: map[string]string{"contact_id": "the event log's counts; the payload with the customer's details is gone"}},
			{Name: "earn_checkouts", Action: "clear", Columns: []string{"email"},
				Kept: map[string]string{"contact_id": "checkout conversion counts; the contact is scrubbed"}},
			{Name: "earn_import_rows", Action: "clear", Columns: []string{"email", "name", "customer_id", "stripe_customer_id", "stripe_subscription_id", "candidates", "subscription"},
				Kept: map[string]string{"contact_id": "the import's report; the contact is scrubbed"}},
			{Name: "earn_mrr_changes", Action: "keep", Kept: map[string]string{"contact_id": "MRR history; amounts only, the contact is scrubbed"}},
			{Name: "earn_offer_uses", Action: "keep", Kept: map[string]string{"contact_id": "offer limits and counts; the contact is scrubbed"}},
			{Name: "earn_cancel_visits", Action: "keep", Kept: map[string]string{"contact_id": "cancel-page counts and chosen reasons; the contact is scrubbed"}},
		},
		Guard: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) (*erasure.Refusal, error) {
			rows, err := tx.Query(ctx, `SELECT DISTINCT contact_id FROM paid_subscriptions
				WHERE contact_id = ANY ($1) AND kind = 'stripe' AND mode = 'live' AND ended_at IS NULL AND status IN ('trialing', 'active', 'past_due')`, b.IDs())
			if err != nil {
				return nil, err
			}
			ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
			if err != nil || len(ids) == 0 {
				return nil, err
			}
			return &erasure.Refusal{IDs: ids, Code: "paid_members", Message: mErasePaidMembers}, nil
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				// Test-mode rows aren't revenue; they go.
				`DELETE FROM paid_events WHERE contact_id = ANY ($1)`,
				`DELETE FROM paid_subscriptions WHERE contact_id = ANY ($1) AND mode = 'test'`,
				`UPDATE paid_subscriptions SET status = 'canceled', ended_at = now(), end_reason = 'grant_removed', updated_at = now()
				 WHERE contact_id = ANY ($1) AND kind IN ('complimentary', 'gift') AND ended_at IS NULL`,
				`UPDATE paid_subscriptions SET billing_email = '', stripe_customer_id = '', stripe_subscription_id = '', cancel_reason = '',
				    grant_note = '', failure_reason = '', updated_at = now()
				 WHERE contact_id = ANY ($1)`,
				`UPDATE payment_events SET payload = '{}' WHERE contact_id = ANY ($1) AND payload IS NOT NULL`,
				`UPDATE earn_checkouts SET email = '' WHERE email <> '' AND (contact_id = ANY ($1) OR (workspace_id = $4 AND lower(email) = ANY ($2)))`,
				`UPDATE earn_import_rows r SET email = '', name = '', customer_id = '', stripe_customer_id = '', stripe_subscription_id = '', candidates = '[]', subscription = '{}'
				 WHERE r.contact_id = ANY ($1)
				    OR (lower(r.email) = ANY ($2) AND r.import_id IN (SELECT id FROM earn_imports WHERE workspace_id = $4))`,
			)
		},
		Exports: []erasure.Export{
			{Section: "paidSubscriptions", Needs: authz.EarnView, SQL: `SELECT s.mode, s.kind, t.name AS tier, s.status, s.period, s.amount_cents, s.currency, s.started_at,
				s.trial_end, s.current_period_end, s.cancel_at, s.canceled_at, s.ended_at, s.end_reason, s.cancel_reason, s.paid_cents, s.refunded_cents,
				s.billing_email, s.stripe_customer_id, s.source_label, s.grant_reason, s.grant_note
				FROM paid_subscriptions s LEFT JOIN earn_tiers t ON t.id = s.tier_id WHERE s.contact_id = $1 ORDER BY s.started_at`},
			{Section: "paidTimeline", Needs: authz.EarnView, SQL: `SELECT mode, kind, occurred_at, amount_cents, currency, data
				FROM paid_events WHERE contact_id = $1 ORDER BY occurred_at`},
			{Section: "paidCheckouts", Needs: authz.EarnView, SQL: `SELECT mode, period, email, status, offer, created_at, completed_at
				FROM earn_checkouts WHERE contact_id = $1 ORDER BY created_at`},
		},
	}
}
