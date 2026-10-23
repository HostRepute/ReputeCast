package whatsapp

import (
	"context"
	"net/netip"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/store"
)

// ContactActivity puts a contact's WhatsApp on their timeline (implements
// audience.ActivitySource): each message with its status (types
// whatsapp_sent, whatsapp_delivered, whatsapp_failed…, read in Detail),
// what they wrote (whatsapp_reply, whatsapp_button, whatsapp_opt_out,
// whatsapp_opt_in, whatsapp_help) and every change of WhatsApp consent
// with its evidence (whatsapp_consent_<action>).
func (s *Service) ContactActivity(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, before *time.Time, beforeID string, limit int) ([]audience.ContactActivityItem, int64, error) {
	union := `
		SELECT m.id, 'message'::text AS src, 'whatsapp_' || m.status AS kind,
			coalesce(m.failed_at, m.delivered_at, m.sent_at, m.created_at) AS at,
			coalesce(c.name, a.name, '') AS subject, w.template AS detail, m.provider_code AS code, m.reason AS note, ''::text AS list,
			m.kind AS source, NULL::inet AS ip, ''::text AS actor, ''::text AS version, (w.read_at IS NOT NULL) AS read, coalesce(m.body, '') AS body
		FROM channel_messages m
		JOIN whatsapp_messages w ON w.message_id = m.id
		LEFT JOIN campaigns c ON c.id = m.campaign_id
		LEFT JOIN automations a ON a.id = m.automation_id
		WHERE m.contact_id = $1 AND m.workspace_id = $2 AND m.channel = 'whatsapp' AND m.kind <> 'test'
		  AND m.status IN ('sent','delivered','undelivered','failed','skipped','held','cancelled')
		UNION ALL
		SELECT i.id, 'inbound', 'whatsapp_' || i.kind, i.received_at, coalesce(c.name, a.name, ''), i.keyword, '', '', coalesce(i.effect ->> 'lists', ''),
			'', NULL, '', '', false, i.body
		FROM whatsapp_inbound i
		LEFT JOIN campaigns c ON c.id = i.campaign_id
		LEFT JOIN automations a ON a.id = i.automation_id
		WHERE $1 = ANY(i.contact_ids) AND i.workspace_id = $2
		UNION ALL
		SELECT e.id, 'consent', 'whatsapp_consent_' || e.action, e.occurred_at, '', e.page, e.keyword, e.note, coalesce(l.name, ''), e.source, e.ip,
			coalesce(u.name, ''), e.consent_version, false, ''
		FROM channel_consent_events e
		LEFT JOIN lists l ON l.id = e.list_id
		LEFT JOIN users u ON u.id = e.actor_id
		WHERE e.contact_id = $1 AND e.workspace_id = $2 AND e.channel = 'whatsapp'`
	var total int64
	if err := s.Pool.QueryRow(ctx, "SELECT count(*) FROM ("+union+") ev", contactID, ws.ID).Scan(&total); err != nil {
		return nil, 0, err
	}
	args := []any{contactID, ws.ID}
	cur := ""
	if before != nil {
		bid, _ := uuid.Parse(beforeID)
		args = append(args, *before, bid)
		cur = " WHERE (ev.at, ev.id) < ($3, $4)"
	}
	args = append(args, limit)
	rows, err := s.Pool.Query(ctx, "SELECT * FROM ("+union+") ev"+cur+" ORDER BY ev.at DESC, ev.id DESC LIMIT $"+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []audience.ContactActivityItem{}
	l := langOf(ctx)
	for rows.Next() {
		var it audience.ContactActivityItem
		var id uuid.UUID
		var src, body string
		var ip *netip.Addr
		var read bool
		if err := rows.Scan(&id, &src, &it.Type, &it.At, &it.Subject, &it.Detail, &it.Code, &it.Note, &it.List, &it.Source, &ip, &it.Actor,
			&it.ConsentVersion, &read, &body); err != nil {
			return nil, 0, err
		}
		it.ID = id.String()
		switch src {
		case "consent":
			it.Category = "consent"
			if ip != nil {
				it.IP = ip.String()
			}
			// A keyword's consent change was written in English: it reads in
			// the reader's language. What a person or a form wrote stays.
			it.Detail, it.Note = storedIn(l, it.Detail), storedIn(l, it.Note)
		case "inbound":
			it.Category = "engagement"
			it.Answer = clip(body, 300)
		default:
			it.Category = "delivery"
			it.Answer = clip(body, 300)
			if read && it.Note == "" {
				// Only people who share read receipts report it.
				it.Note = "read"
			}
			it.Provider = &audience.ContactActivityProvider{Name: "Meta", Adapter: "whatsapp"}
		}
		out = append(out, it)
	}
	return out, total, rows.Err()
}
