package revenue

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is what a conversion's sender attached about the contact; the
// conversion stays for attributed revenue (review 2026-11 DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "revenue",
		Tables: []erasure.Table{
			{Name: "conversions", Action: "clear", Columns: []string{"metadata"},
				Kept: map[string]string{"contact_id": "attributed revenue; the contact is scrubbed"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b, `UPDATE conversions SET metadata = '{}' WHERE contact_id = ANY ($1) AND metadata <> '{}'`)
		},
		Exports: []erasure.Export{
			{Section: "conversions", SQL: `SELECT name, value, value_cents, currency, occurred_at, metadata FROM conversions WHERE contact_id = $1 ORDER BY occurred_at`},
		},
	}
}
