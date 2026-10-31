package verification

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is the results of checking the contact's address (review 2026-11
// DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "verification",
		Tables: []erasure.Table{
			{Name: "contact_address_checks", Action: "delete"},
			{Name: "verification_job_items", Action: "clear", Columns: []string{"email"},
				Kept: map[string]string{"contact_id": "the job's counts; the contact is scrubbed"}},
			{Name: "verification_realtime_pending", Action: "delete", Columns: []string{"email"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`DELETE FROM contact_address_checks WHERE contact_id = ANY ($1)`,
				`UPDATE verification_job_items SET email = '' WHERE contact_id = ANY ($1) AND email <> ''`,
				`DELETE FROM verification_realtime_pending WHERE workspace_id = $4 AND lower(email) = ANY ($2)`,
			)
		},
		Exports: []erasure.Export{
			{Section: "addressChecks", SQL: `SELECT service, result, reason, code, checked_at FROM contact_address_checks WHERE contact_id = $1`},
		},
	}
}
