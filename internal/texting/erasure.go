package texting

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/erasure"
)

// Erasure is what SMS, WhatsApp and push keep about a contact in the shared
// channel tables: consents with their proof, messages and their events,
// replies and link clicks. Messages stay, without the number or text, so
// costs and campaign counts still add up (review 2026-11 DP-1).
func Erasure() erasure.Source {
	return erasure.Source{
		Name: "channels",
		Tables: []erasure.Table{
			{Name: "channel_consents", Action: "delete"},
			{Name: "channel_consent_events", Action: "delete", Columns: []string{"phone", "ip", "user_agent", "consent_text", "page", "note"}},
			{Name: "channel_messages", Action: "clear", Columns: []string{"recipient", "body"},
				Kept: map[string]string{"contact_id": "campaign and cost counts; the contact is scrubbed",
					"recipient": "replaced by erased:<contact>, so distinct-recipient counts still add up"}},
			{Name: "channel_message_events", Action: "clear", Columns: []string{"detail"}},
			{Name: "sms_inbound", Action: "clear", Columns: []string{"phone", "body", "contact_ids"}},
			{Name: "sms_link_clicks", Action: "clear", Columns: []string{"user_agent"},
				Kept: map[string]string{"contact_id": "counts distinct clickers; the contact is scrubbed"}},
			{Name: "sms_links", Action: "keep", Kept: map[string]string{"contact_id": "a message's short links; no personal data beyond the scrubbed contact"}},
		},
		Erase: func(ctx context.Context, tx pgx.Tx, b *erasure.Batch) error {
			return erasure.Exec(ctx, tx, b,
				`DELETE FROM channel_consents WHERE contact_id = ANY ($1)`,
				`DELETE FROM channel_consent_events WHERE contact_id = ANY ($1)`,
				`UPDATE channel_message_events SET detail = '{}' WHERE message_id IN (SELECT id FROM channel_messages WHERE contact_id = ANY ($1)) AND detail <> '{}'`,
				`UPDATE channel_messages SET recipient = 'erased:' || contact_id, body = NULL WHERE contact_id = ANY ($1)`,
				// A reply from the contact's number: linked to them, or from
				// their number when no other live contact has it.
				`UPDATE sms_raw_events SET body = ''::bytea, headers = '{}', summary = '' WHERE id IN (
				    SELECT i.raw_event_id FROM sms_inbound i WHERE i.raw_event_id IS NOT NULL AND (i.contact_ids && $1::uuid[]
				      OR (i.workspace_id = $4 AND i.phone = ANY ($3) AND NOT EXISTS (
				          SELECT 1 FROM contacts c WHERE c.workspace_id = $4 AND c.phone = i.phone AND c.erased_at IS NULL AND c.id <> ALL ($1)))))`,
				`UPDATE sms_inbound i SET phone = '', body = '', contact_ids = (SELECT coalesce(array_agg(x), '{}') FROM unnest(i.contact_ids) x WHERE x <> ALL ($1))
				 WHERE i.contact_ids && $1::uuid[]
				    OR (i.workspace_id = $4 AND i.phone = ANY ($3) AND NOT EXISTS (
				        SELECT 1 FROM contacts c WHERE c.workspace_id = $4 AND c.phone = i.phone AND c.erased_at IS NULL AND c.id <> ALL ($1)))`,
				`UPDATE sms_link_clicks SET user_agent = '' WHERE contact_id = ANY ($1) AND user_agent <> ''`,
			)
		},
		Exports: []erasure.Export{
			{Section: "channelConsents", SQL: `SELECT channel, l.name AS list, status, source, keyword, stopped_at, changed_at
				FROM channel_consents cc LEFT JOIN lists l ON l.id = cc.list_id WHERE cc.contact_id = $1 ORDER BY changed_at`},
			{Section: "channelConsentEvents", SQL: `SELECT channel, action, phone, source, page, consent_text, keyword, host(ip) AS ip, user_agent, note, occurred_at
				FROM channel_consent_events WHERE contact_id = $1 ORDER BY occurred_at`},
			{Section: "channelMessages", SQL: `SELECT channel, kind, recipient, body, country, status, reason, created_at, sent_at, delivered_at, failed_at
				FROM channel_messages WHERE contact_id = $1 ORDER BY created_at`},
			{Section: "smsReplies", SQL: `SELECT phone, body, kind, keyword, received_at FROM sms_inbound WHERE contact_ids @> ARRAY[$1::uuid] ORDER BY received_at`},
		},
	}
}
