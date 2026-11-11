package inbound

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/store"
)

// ContactActivity puts a contact's replies on their timeline (implements
// audience.ActivitySource): "Replied to {campaign or automation email}",
// with the reply's id for its Open reply link. Auto-replies, spam and other
// mail stay off it.
func (s *Service) ContactActivity(ctx context.Context, ws *store.Workspace, contactID uuid.UUID, before *time.Time, beforeID string, limit int) ([]audience.ContactActivityItem, int64, error) {
	cid := contactID
	total, err := s.Q.InboundContactActivityCount(ctx, store.InboundContactActivityCountParams{WorkspaceID: ws.ID, ContactID: &cid})
	if err != nil || total == 0 {
		return []audience.ContactActivityItem{}, total, err
	}
	p := store.InboundContactActivityParams{WorkspaceID: ws.ID, ContactID: &cid, Lim: int32(limit)}
	if before != nil {
		id, _ := uuid.Parse(beforeID)
		p.BeforeAt, p.BeforeID = before, id
	}
	rows, err := s.Q.InboundContactActivity(ctx, p)
	if err != nil {
		return nil, 0, err
	}
	out := make([]audience.ContactActivityItem, 0, len(rows))
	for _, r := range rows {
		name := r.CampaignName
		if name == "" {
			name = r.AutomationName
		}
		out = append(out, audience.ContactActivityItem{ID: r.ID.String(), Category: "engagement", Type: "replied", At: r.ReceivedAt,
			Subject: name, Detail: r.Subject})
	}
	return out, total, nil
}
