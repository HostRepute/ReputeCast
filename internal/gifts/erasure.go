package gifts

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is gifts the contact bought or received: their side's name and
// address go, with the message and the gift's event details; amounts and
// dates stay for revenue (review 2026-11 DP-1). A gift still to be sent
// whose recipient is erased goes to the buyer, as an in-person gift does.
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "gifts",
		Tables: []erasure.Table{
			{Name: "earn_gifts", Action: "anonymize", Columns: []string{"buyer_name", "buyer_email", "recipient_name", "recipient_email", "message", "redeemed_email", "stripe_customer"},
				Kept: map[string]string{"redeemed_contact": "gift redemption counts; the contact is scrubbed"}},
			{Name: "earn_gift_events", Action: "clear", Columns: []string{"data"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`UPDATE earn_gift_events SET data = '{}' WHERE data <> '{}' AND gift_id IN (SELECT id FROM earn_gifts WHERE workspace_id = $4
				    AND (redeemed_contact = ANY ($1) OR lower(buyer_email) = ANY ($2) OR lower(recipient_email) = ANY ($2) OR lower(redeemed_email) = ANY ($2)))`,
				`UPDATE earn_gifts SET buyer_name = '', buyer_email = '', stripe_customer = '', message = '', updated_at = now()
				 WHERE workspace_id = $4 AND lower(buyer_email) = ANY ($2)`,
				`UPDATE earn_gifts SET recipient_name = '', recipient_email = '', redeemed_email = '', message = '', updated_at = now()
				 WHERE workspace_id = $4 AND (redeemed_contact = ANY ($1) OR lower(recipient_email) = ANY ($2) OR lower(redeemed_email) = ANY ($2))`,
			)
		},
		Exports: []erasure.Export{
			{Section: "gifts", Needs: authz.EarnView, SQL: `SELECT mode, months, amount_cents, currency, status, buyer_name, buyer_email, recipient_name, recipient_email,
				message, deliver_at, paid_at, delivered_at, redeemed_at
				FROM earn_gifts WHERE workspace_id = $2 AND (redeemed_contact = $1 OR lower(buyer_email) = $3 OR lower(recipient_email) = $3 OR lower(redeemed_email) = $3)
				ORDER BY created_at`},
		},
	}
}
