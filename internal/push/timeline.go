package push

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/store"
)

// ContactActivity puts a contact's notifications on their timeline
// (implements audience.ActivitySource): each notification to a linked
// subscription with how far it got (push_sent, push_shown, push_clicked,
// push_failed, push_skipped), and each subscription linked to them or
// ended (push_subscribed, push_unsubscribed), with its site and browser.
func (s *Service) ContactActivity(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, before *time.Time, beforeID string, limit int) ([]audience.ContactActivityItem, int64, error) {
	union := `
		SELECT m.id, 'message'::text AS src,
			'push_' || CASE WHEN p.clicked_at IS NOT NULL THEN 'clicked' WHEN p.shown_at IS NOT NULL THEN 'shown' WHEN m.status = 'delivered' THEN 'sent' ELSE m.status END AS kind,
			coalesce(p.clicked_at, p.shown_at, m.failed_at, m.sent_at, m.created_at) AS at,
			coalesce(c.name, a.name, '') AS subject, coalesce(st.host, '') AS detail, p.browser AS code, m.reason AS note, coalesce(m.body::jsonb ->> 'title', '') AS title
		FROM channel_messages m
		JOIN push_messages p ON p.message_id = m.id
		LEFT JOIN push_sites st ON st.id = p.site_id
		LEFT JOIN campaigns c ON c.id = m.campaign_id
		LEFT JOIN automations a ON a.id = m.automation_id
		WHERE m.contact_id = $1 AND m.workspace_id = $2 AND m.channel = 'push' AND m.kind IN ('campaign','automation')
		  AND m.status IN ('sent','delivered','failed','skipped','held')
		UNION ALL
		SELECT s.id, 'subscription', 'push_subscribed', coalesce(s.linked_at, s.created_at), '', st.host, s.browser, s.linked_by, s.prompt_text
		FROM push_subscriptions s JOIN push_sites st ON st.id = s.site_id
		WHERE s.contact_id = $1 AND s.workspace_id = $2
		UNION ALL
		SELECT md5(s.id::text || ':removed')::uuid, 'subscription', 'push_unsubscribed', s.removed_at, '', st.host, s.browser, s.removed_reason, ''
		FROM push_subscriptions s JOIN push_sites st ON st.id = s.site_id
		WHERE s.contact_id = $1 AND s.workspace_id = $2 AND s.removed_at IS NOT NULL`
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
	for rows.Next() {
		var it audience.ContactActivityItem
		var id uuid.UUID
		var src, title string
		if err := rows.Scan(&id, &src, &it.Type, &it.At, &it.Subject, &it.Detail, &it.Code, &it.Note, &title); err != nil {
			return nil, 0, err
		}
		it.ID = id.String()
		switch src {
		case "subscription":
			it.Category, it.Source = "consent", it.Note
			if it.Type == "push_subscribed" {
				// The exact prompt text the reader allowed.
				it.Answer = title
			}
		default:
			it.Category = "delivery"
			if it.Type == "push_clicked" || it.Type == "push_shown" {
				it.Category = "engagement"
			}
			it.Answer = title
		}
		it.Provider = &audience.ContactActivityProvider{Name: "Web push", Adapter: "push"}
		out = append(out, it)
	}
	return out, total, rows.Err()
}
