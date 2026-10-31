package scoring

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is the contact's score and the events that made it; the nightly
// job would drop them for an erased contact anyway (review 2026-11 DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "scoring",
		Tables: []erasure.Table{
			{Name: "contact_scores", Action: "delete"},
			{Name: "score_events", Action: "delete"},
			{Name: "scoring_dirty", Action: "delete"},
			{Name: "scoring_recalc_scores", Action: "delete"},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`DELETE FROM score_events WHERE contact_id = ANY ($1)`,
				`DELETE FROM scoring_dirty WHERE contact_id = ANY ($1)`,
				`DELETE FROM scoring_recalc_scores WHERE contact_id = ANY ($1)`,
				`DELETE FROM contact_scores WHERE contact_id = ANY ($1)`,
			)
		},
		Exports: []erasure.Export{
			{Section: "score", SQL: `SELECT score, events_part, fields_part, active_at, changed_at, computed_at FROM contact_scores WHERE contact_id = $1`},
		},
	}
}
