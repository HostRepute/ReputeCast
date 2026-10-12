package audience

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/store"
)

// contactExport is everything ReputeCast stores about one person, for a
// data subject access request ("Export data (GDPR)").
type contactExport struct {
	ExportedAt  time.Time           `json:"exportedAt"`
	Workspace   string              `json:"workspace"`
	Contact     ContactProfile      `json:"contact"`
	Attributes  json.RawMessage     `json:"customFields"`
	Evidence    json.RawMessage     `json:"consentEvidence"`
	Messages    []exportMessage     `json:"messages"`
	Events      []ActivityItem      `json:"events"`
	Suppression *ProfileSuppression `json:"suppression,omitempty"`
}

type exportMessage struct {
	Stream      string     `json:"stream"`
	Email       string     `json:"email"`
	Campaign    string     `json:"campaign,omitempty"`
	Provider    string     `json:"provider,omitempty"`
	Status      string     `json:"status"`
	QueuedAt    time.Time  `json:"queuedAt"`
	DeliveredAt *time.Time `json:"deliveredAt,omitempty"`
	OpenedAt    *time.Time `json:"openedAt,omitempty"`
	ClickedAt   *time.Time `json:"clickedAt,omitempty"`
}

// ExportContact returns a JSON document with all data about one contact.
// The export is logged.
func (s *Service) ExportContact(ctx context.Context, ws *store.Workspace, id uuid.UUID) ([]byte, string, error) {
	prof, err := s.Profile(ctx, ws, id)
	if err != nil {
		return nil, "", err
	}
	c, err := s.getContact(ctx, s.Q, ws, id)
	if err != nil {
		return nil, "", err
	}
	out := contactExport{ExportedAt: s.now().UTC(), Workspace: ws.Name, Contact: prof, Attributes: c.Attributes, Evidence: c.ConsentEvidence,
		Messages: []exportMessage{}, Events: []ActivityItem{}, Suppression: prof.Suppression}
	rows, err := s.Pool.Query(ctx, `SELECT m.stream, m.email, coalesce(cp.name, a.name, ''), coalesce(p.name, ''), m.status, m.queued_at,
		m.delivered_at, m.opened_at, m.clicked_at
		FROM messages m LEFT JOIN campaigns cp ON cp.id = m.campaign_id LEFT JOIN automations a ON a.id = m.automation_id
		LEFT JOIN providers p ON p.id = m.provider_id
		WHERE m.contact_id = $1 ORDER BY m.queued_at DESC LIMIT 10000`, id)
	if err != nil {
		return nil, "", err
	}
	for rows.Next() {
		var m exportMessage
		if err := rows.Scan(&m.Stream, &m.Email, &m.Campaign, &m.Provider, &m.Status, &m.QueuedAt, &m.DeliveredAt, &m.OpenedAt, &m.ClickedAt); err != nil {
			rows.Close()
			return nil, "", err
		}
		out.Messages = append(out.Messages, m)
	}
	rows.Close()
	cursor := ""
	for i := 0; i < 100; i++ {
		page, err := s.Activity(ctx, ws, id, "all", cursor, 100)
		if err != nil {
			return nil, "", err
		}
		out.Events = append(out.Events, page.Items...)
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, "", err
	}
	if err := s.Audit.Log(ctx, audit.Event{WorkspaceID: &ws.ID, Action: "contact.export", TargetType: "contact", TargetID: id.String(), TargetName: c.Email,
		Metadata: map[string]any{"messages": len(out.Messages), "events": len(out.Events)}}); err != nil {
		return nil, "", err
	}
	return b, "contact-" + id.String()[:8] + ".json", nil
}
