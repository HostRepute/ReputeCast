package offers

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is the address a single-use offer code was issued to; the code
// stays for the offer's counts (review 2026-11 DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "offers",
		Tables: []erasure.Table{
			{Name: "earn_offer_codes", Action: "clear", Columns: []string{"issued_to"},
				Kept: map[string]string{"contact_id": "the offer's issued and redeemed counts; the contact is scrubbed"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`UPDATE earn_offer_codes SET issued_to = '' WHERE issued_to <> '' AND (contact_id = ANY ($1) OR (workspace_id = $4 AND lower(issued_to) = ANY ($2)))`)
		},
	}
}
