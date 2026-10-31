package whatsapp

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is what WhatsApp keeps beyond the shared channel tables: replies
// from the contact's number and the template parameters of messages to them
// (review 2026-11 DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "whatsapp",
		Tables: []erasure.Table{
			{Name: "whatsapp_inbound", Action: "clear", Columns: []string{"phone", "name", "body", "contact_ids"}},
			{Name: "whatsapp_messages", Action: "clear", Columns: []string{"params"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`UPDATE whatsapp_raw_events SET body = ''::bytea, headers = '{}', summary = '' WHERE id IN (
				    SELECT i.raw_event_id FROM whatsapp_inbound i WHERE i.raw_event_id IS NOT NULL AND (i.contact_ids && $1::uuid[]
				      OR (i.workspace_id = $4 AND i.phone = ANY ($3) AND NOT EXISTS (
				          SELECT 1 FROM contacts c WHERE c.workspace_id = $4 AND c.phone = i.phone AND c.erased_at IS NULL AND c.id <> ALL ($1)))))`,
				`UPDATE whatsapp_inbound i SET phone = '', name = '', body = '', contact_ids = (SELECT coalesce(array_agg(x), '{}') FROM unnest(i.contact_ids) x WHERE x <> ALL ($1))
				 WHERE i.contact_ids && $1::uuid[]
				    OR (i.workspace_id = $4 AND i.phone = ANY ($3) AND NOT EXISTS (
				        SELECT 1 FROM contacts c WHERE c.workspace_id = $4 AND c.phone = i.phone AND c.erased_at IS NULL AND c.id <> ALL ($1)))`,
				`UPDATE whatsapp_messages SET params = '[]' WHERE message_id IN (SELECT id FROM channel_messages WHERE contact_id = ANY ($1)) AND params <> '[]'`,
			)
		},
		Exports: []erasure.Export{
			{Section: "whatsappReplies", SQL: `SELECT phone, name, body, kind, keyword, received_at FROM whatsapp_inbound WHERE contact_ids @> ARRAY[$1::uuid] ORDER BY received_at`},
		},
	}
}
