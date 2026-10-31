package automations

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure ends the contact's runs and drops what their triggers carried
// (order details, API event payloads); runs stay for the flows' counts
// (review 2026-11 DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "automations",
		Tables: []erasure.Table{
			{Name: "automation_runs", Action: "clear", Columns: []string{"state"},
				Kept: map[string]string{"contact_id": "the flow's step counts; the run is ended and the contact scrubbed"}},
			{Name: "automation_events", Action: "clear", Columns: []string{"payload"},
				Kept: map[string]string{"contact_id": "trigger counts; the contact is scrubbed"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`UPDATE automation_runs SET status = 'exited', exit_reason = 'erased', finished_at = now(), next_at = NULL
				 WHERE contact_id = ANY ($1) AND status IN ('active', 'waiting')`,
				`UPDATE automation_runs SET state = '{}' WHERE contact_id = ANY ($1) AND state <> '{}'`,
				`DELETE FROM automation_events WHERE contact_id = ANY ($1) AND processed_at IS NULL`,
				`UPDATE automation_events SET payload = '{}' WHERE contact_id = ANY ($1) AND payload <> '{}'`,
			)
		},
		Exports: []erasure.Export{
			{Section: "automationRuns", SQL: `SELECT a.name AS automation, r.version, r.node_id AS step, r.status, r.exit_reason, r.entered_at, r.finished_at
				FROM automation_runs r JOIN automations a ON a.id = r.automation_id WHERE r.contact_id = $1 ORDER BY r.entered_at`},
		},
	}
}
