package tracker

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is what the site tracker knows of a contact: the browsers linked
// to them, their events (with the anonymous ones of those browsers),
// their product interest, the store flows they entered and their "email
// me when it's back" requests. A visitor's own products go with it.
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "tracker",
		Tables: []erasure.Table{
			{Name: "tracker_visitors", Action: "delete"},
			{Name: "tracker_events", Action: "delete"},
			{Name: "tracker_product_views", Action: "delete"},
			{Name: "tracker_trigger_entries", Action: "delete"},
			{Name: "store_stock_requests", Action: "delete"},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`DELETE FROM tracker_events WHERE contact_id = ANY ($1)
				    OR visitor_id IN (SELECT id FROM tracker_visitors WHERE contact_id = ANY ($1))`,
				`DELETE FROM tracker_visitors WHERE contact_id = ANY ($1)`,
				`DELETE FROM tracker_product_views WHERE contact_id = ANY ($1)`,
				`DELETE FROM tracker_trigger_entries WHERE contact_id = ANY ($1)`,
				`DELETE FROM store_stock_requests WHERE contact_id = ANY ($1)`,
			)
		},
		Exports: []erasure.Export{
			{Section: "siteActivity", SQL: `SELECT e.occurred_at,
				CASE e.kind WHEN 1 THEN 'page' WHEN 2 THEN 'product' WHEN 3 THEN 'cart' ELSE 'custom' END AS kind,
				s.origin AS site, e.path, e.referrer_domain, coalesce(p.title, '') AS product, e.quantity, e.name, e.props
				FROM tracker_events e
				JOIN tracker_sites s ON s.id = e.site_id
				LEFT JOIN store_products p ON p.id = e.product_id
				WHERE e.contact_id = $1 AND e.workspace_id = $2 AND e.occurred_at >= now() - interval '90 days'
				ORDER BY e.occurred_at, e.id`},
		},
	}
}
