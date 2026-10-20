package texting

import (
	"context"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/store"
)

// ContactActivity puts a contact's texts on their timeline (implements
// audience.ActivitySource): each text with its status, clicks on its links
// and replies, and every change of SMS consent with its evidence.
func (s *Service) ContactActivity(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, before *time.Time, beforeID string, limit int) ([]audience.ContactActivityItem, int64, error) {
	union := `
		SELECT m.id, 'text'::text AS src, 'sms_' || m.status AS kind,
			coalesce(m.failed_at, m.delivered_at, m.sent_at, m.created_at) AS at,
			coalesce(c.name, a.name, '') AS subject, coalesce(p.name, '') AS provider, coalesce(p.adapter, '') AS adapter,
			m.sender AS detail, m.provider_code AS code, m.reason AS note, ''::text AS list, m.kind AS source, NULL::inet AS ip, ''::text AS actor,
			''::text AS version, false AS machine, ''::text AS url
		FROM channel_messages m
		LEFT JOIN campaigns c ON c.id = m.campaign_id
		LEFT JOIN automations a ON a.id = m.automation_id
		LEFT JOIN sms_providers p ON p.id = m.provider_id
		WHERE m.contact_id = $1 AND m.workspace_id = $2 AND m.channel = 'sms' AND m.status IN ('sent','delivered','undelivered','failed','skipped','held','cancelled')
		UNION ALL
		SELECT k.id, 'click', 'sms_click', k.occurred_at, coalesce(c.name, a.name, ''), '', '', '', '', '', '', '', NULL, '', '', k.machine, l.original
		FROM sms_link_clicks k
		JOIN sms_links l ON l.code = k.code
		LEFT JOIN campaigns c ON c.id = l.campaign_id
		LEFT JOIN automations a ON a.id = l.automation_id
		WHERE k.contact_id = $1 AND k.workspace_id = $2
		UNION ALL
		SELECT i.id, 'inbound', 'sms_' || i.kind, i.received_at, coalesce(c.name, a.name, ''), coalesce(p.name, ''), coalesce(p.adapter, ''),
			i.keyword, '', i.body, coalesce(i.effect ->> 'lists', ''), '', NULL, '', '', false, ''
		FROM sms_inbound i
		LEFT JOIN campaigns c ON c.id = i.campaign_id
		LEFT JOIN automations a ON a.id = i.automation_id
		LEFT JOIN sms_providers p ON p.id = i.provider_id
		WHERE $1 = ANY(i.contact_ids) AND i.workspace_id = $2
		UNION ALL
		SELECT e.id, 'consent', 'sms_consent_' || e.action, e.occurred_at, '', '', '', e.page, e.keyword, e.note, coalesce(l.name, ''), e.source, e.ip,
			coalesce(u.name, ''), e.consent_version, false, ''
		FROM channel_consent_events e
		LEFT JOIN lists l ON l.id = e.list_id
		LEFT JOIN users u ON u.id = e.actor_id
		WHERE e.contact_id = $1 AND e.workspace_id = $2 AND e.channel = 'sms'`
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
	rows, err := s.Pool.Query(ctx, "SELECT * FROM ("+union+") ev"+cur+" ORDER BY ev.at DESC, ev.id DESC LIMIT $"+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []audience.ContactActivityItem{}
	for rows.Next() {
		var it audience.ContactActivityItem
		var id uuid.UUID
		var src, provider, adapter, url string
		var ip *netip.Addr
		if err := rows.Scan(&id, &src, &it.Type, &it.At, &it.Subject, &provider, &adapter, &it.Detail, &it.Code, &it.Note, &it.List, &it.Source, &ip,
			&it.Actor, &it.ConsentVersion, &it.Machine, &url); err != nil {
			return nil, 0, err
		}
		it.ID = id.String()
		switch src {
		case "consent":
			it.Category = "consent"
			if ip != nil {
				it.IP = ip.String()
			}
		case "click", "inbound":
			it.Category = "engagement"
			if src == "inbound" {
				it.Answer, it.Note = it.Note, ""
				it.List = strings.Trim(strings.ReplaceAll(strings.ReplaceAll(it.List, `"`, ""), ",", ", "), "[]")
			}
		default:
			it.Category = "delivery"
			it.Detail = senderShown(it.Detail)
			if it.Subject == "" {
				it.Subject = contextOf(it.Source)
			}
		}
		if provider != "" {
			it.Provider = &audience.ContactActivityProvider{Name: provider, Adapter: adapter}
		}
		if url != "" {
			it.URL = strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "http://")
		}
		out = append(out, it)
	}
	return out, total, rows.Err()
}
