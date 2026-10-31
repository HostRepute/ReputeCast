package recommendations

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure unlinks signups a partner sent for the contact; the requests
// stay counted (review 2026-11 DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "recommendations",
		Tables: []erasure.Table{
			{Name: "partner_requests_in", Action: "clear", Columns: []string{"contact_id"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b, `UPDATE partner_requests_in SET contact_id = NULL WHERE contact_id = ANY ($1)`)
		},
	}
}
