package products

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is the contact's product orders and checkouts: the buyer's name
// and address go, amounts stay for revenue; download links stop working
// (review 2026-11 DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "products",
		Tables: []erasure.Table{
			{Name: "product_orders", Action: "anonymize", Columns: []string{"buyer_email", "buyer_name"},
				Kept: map[string]string{"contact_id": "order and revenue counts; the contact is scrubbed",
					"buyer": "how the buyer was matched (contact, joined, none), not a person"}},
			{Name: "product_checkouts", Action: "clear", Columns: []string{"email"},
				Kept: map[string]string{"contact_id": "checkout conversion counts; the contact is scrubbed"}},
			{Name: "lead_magnet_links", Action: "clear", Kept: map[string]string{"contact_id": "download counts; the link is expired"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`UPDATE product_orders SET buyer_email = '', buyer_name = ''
				 WHERE contact_id = ANY ($1) OR (workspace_id = $4 AND lower(buyer_email) = ANY ($2))`,
				`UPDATE product_checkouts SET email = '' WHERE email <> '' AND (contact_id = ANY ($1) OR (workspace_id = $4 AND lower(email) = ANY ($2)))`,
				`UPDATE lead_magnet_links SET expires_at = least(expires_at, now()) WHERE contact_id = ANY ($1)`,
			)
		},
		Exports: []erasure.Export{
			{Section: "productOrders", Needs: authz.ProductsView, SQL: `SELECT number, product_name, mode, buyer_email, buyer_name, amount_cents, tax_cents, currency, status,
				refunded_cents, refunded_at, source_url, paid_at, created_at
				FROM product_orders WHERE contact_id = $1 OR (workspace_id = $2 AND lower(buyer_email) = $3) ORDER BY created_at`},
		},
	}
}
