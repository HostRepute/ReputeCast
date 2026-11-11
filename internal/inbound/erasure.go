package inbound

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is the personal data received mail holds about contacts: the
// messages from them or linked to them, and what arrived for those
// messages (the raw request or message).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "inbound",
		Tables: []erasure.Table{
			{Name: "inbound_messages", Action: "delete"},
			{Name: "inbound_raw", Action: "delete"},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				// What arrived for their messages first: it refers to them.
				`DELETE FROM inbound_raw r WHERE r.id IN (
				    SELECT im.raw_id FROM inbound_messages im
				    WHERE im.workspace_id = $4 AND im.raw_id IS NOT NULL AND (im.contact_id = ANY ($1) OR lower(im.from_email) = ANY ($2)))`,
				`DELETE FROM inbound_messages im WHERE im.workspace_id = $4 AND (im.contact_id = ANY ($1) OR lower(im.from_email) = ANY ($2))`,
			)
		},
		Exports: []erasure.Export{
			{Section: "replies", SQL: `SELECT received_at, kind, from_email, subject, text_body FROM inbound_messages
				WHERE workspace_id = $2 AND (contact_id = $1 OR lower(from_email) = $3) ORDER BY received_at`},
		},
	}
}
