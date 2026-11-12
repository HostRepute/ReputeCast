package leads

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is what lead ads keep about a contact: which lead form they sent
// and when. The entries stay without the contact, so a later import of
// past leads never brings an erased person back (their lead's id is
// recorded). Meta's deliveries hold only ids, never answers, and go after
// 30 days.
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "leads",
		Tables: []erasure.Table{
			{Name: "lead_entries", Action: "anonymize", Columns: []string{"contact_id"},
				Kept: map[string]string{"leadgen_id": "Meta's id of the lead, so importing past leads never adds the erased person again"}},
			{Name: "lead_events", Action: "keep",
				Kept: map[string]string{"body": "Meta's webhook carries only ids (leadgen_id, page_id, form_id, ad_id, created_time), never answers; purged after 30 days"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b, `UPDATE lead_entries SET contact_id = NULL WHERE contact_id = ANY ($1)`)
		},
		Exports: []erasure.Export{
			{Section: "leadAds", SQL: `SELECT f.name AS form, p.name AS page, e.outcome, e.lead_at
				FROM lead_entries e JOIN lead_forms f ON f.id = e.form_id JOIN lead_pages p ON p.id = f.page_id
				WHERE e.contact_id = $1 ORDER BY e.lead_at`},
		},
	}
}
