package commerce

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is what synced stores told ReputeCast about the contact: the
// customer's name and address, the landing page and free-form UTM parts of
// orders, abandoned-cart links and the activity feed. Orders and customers
// stay, without them, for revenue and customer counts (review 2026-11 DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "commerce",
		Tables: []erasure.Table{
			{Name: "store_customers", Action: "anonymize", Columns: []string{"email", "first_name", "last_name", "country"},
				Kept: map[string]string{"contact_id": "customer counts; the contact is scrubbed"}},
			{Name: "store_orders", Action: "clear", Columns: []string{"landing_page", "utm_content", "utm_term", "name"},
				Kept: map[string]string{"contact_id": "revenue and attribution; the contact is scrubbed",
					"name": "the store's order number (#1042), not a person's name"}},
			{Name: "store_checkouts", Action: "clear", Columns: []string{"recovery_url"},
				Kept: map[string]string{"contact_id": "abandoned-cart counts; the contact is scrubbed", "has_email": "a boolean, not an address"}},
			{Name: "store_activity", Action: "delete"},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`UPDATE store_customers SET email = '', first_name = '', last_name = '', country = ''
				 WHERE contact_id = ANY ($1) OR (workspace_id = $4 AND lower(email) = ANY ($2))`,
				`UPDATE store_orders SET landing_page = '', utm_content = '', utm_term = '' WHERE contact_id = ANY ($1)`,
				`UPDATE store_checkouts SET recovery_url = '' WHERE contact_id = ANY ($1) AND recovery_url <> ''`,
				`DELETE FROM store_activity WHERE contact_id = ANY ($1)`,
			)
		},
		Exports: []erasure.Export{
			{Section: "storeOrders", SQL: `SELECT s.name AS store, o.name AS order, o.placed_at, o.currency, o.total_cents, o.refunded_cents, o.landing_page,
				o.utm_source, o.utm_medium, o.utm_campaign, o.utm_content, o.utm_term
				FROM store_orders o JOIN stores s ON s.id = o.store_id WHERE o.contact_id = $1 ORDER BY o.placed_at`},
			{Section: "storeCustomers", SQL: `SELECT s.name AS store, c.email, c.first_name, c.last_name, c.country, c.marketing, c.consent_at
				FROM store_customers c JOIN stores s ON s.id = c.store_id WHERE c.contact_id = $1`},
		},
	}
}
