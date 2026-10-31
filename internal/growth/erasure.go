package growth

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is where the contact came from. Source, medium, campaign, page and
// referrer stay: they are kept as a host and path only and Audience › Growth
// counts joins by them. The free-form UTM content and term go (review
// 2026-11 DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "growth",
		Tables: []erasure.Table{
			{Name: "contact_growth", Action: "clear", Columns: []string{"utm_content", "utm_term"},
				Kept: map[string]string{"contact_id": "growth counts by source; the contact is scrubbed",
					"referrer": "a host and path only, counted on Audience › Growth"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`UPDATE contact_growth SET utm_content = '', utm_term = '' WHERE contact_id = ANY ($1) AND (utm_content <> '' OR utm_term <> '')`)
		},
		Exports: []erasure.Export{
			{Section: "signupSource", SQL: `SELECT source_kind, source_id, joined_at, left_at, left_kind, utm_source, utm_medium, utm_campaign, utm_content, utm_term,
				page, referrer FROM contact_growth WHERE contact_id = $1`},
		},
	}
}
