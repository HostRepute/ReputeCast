package integrations

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is the bodies of integration calls about the contact (kept 7 days)
// and contact events not yet sent to endpoints; contact.erased is queued
// after this runs (review 2026-11 DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "integrations",
		Tables: []erasure.Table{
			{Name: "integration_calls", Action: "clear", Columns: []string{"request", "response", "target", "ip"},
				Kept: map[string]string{"contact_id": "the integration's activity counts; the contact is scrubbed"}},
			{Name: "webhook_contact_events", Action: "delete"},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`UPDATE integration_calls SET request = NULL, response = NULL, target = '{}', ip = NULL WHERE contact_id = ANY ($1)`,
				`DELETE FROM webhook_contact_events WHERE contact_id = ANY ($1)`,
			)
		},
	}
}
