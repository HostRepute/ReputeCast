package campaigns

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Presence (review 2026-10, CO-10): who else has a campaign open in the
// composer, and whether they're editing. The composer says it's there
// every 20 seconds while open; someone not heard from for a minute is gone.
// It isn't audited: it records no change to anything, only who is looking.

// presenceFor is how long someone counts as present after they last said so.
const presenceFor = time.Minute

// ComposerPresenceInput says the caller has the campaign open.
type ComposerPresenceInput struct {
	// Editing is true while the caller is changing the email (typed or
	// changed something in the last minute).
	Editing bool `json:"editing,omitempty"`
}

// ComposerPresent is someone else with the campaign open.
type ComposerPresent struct {
	UserID  string    `json:"userId"`
	Name    string    `json:"name"`
	Color   string    `json:"color" doc:"Avatar colour"`
	Editing bool      `json:"editing"`
	SeenAt  time.Time `json:"seenAt"`
}

// ComposerPresence is who else has the campaign open.
type ComposerPresence struct {
	Others []ComposerPresent `json:"others"`
}

// Present records that the caller has the campaign open and returns who
// else does.
func (s *Service) Present(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in ComposerPresenceInput) (ComposerPresence, error) {
	if _, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id}); err != nil {
		return ComposerPresence{}, api.MapError(err, "Campaign")
	}
	now := s.now()
	me := uuid.Nil
	if p != nil && p.UserID != uuid.Nil {
		me = p.UserID
		// Only people are present; an API key or an agent isn't.
		if err := s.Q.PresenceTouch(ctx, store.PresenceTouchParams{CampaignID: id, UserID: me, Editing: in.Editing, Now: now}); err != nil {
			return ComposerPresence{}, err
		}
	}
	rows, err := s.Q.PresenceList(ctx, store.PresenceListParams{CampaignID: id, Since: now.Add(-presenceFor), Me: me})
	if err != nil {
		return ComposerPresence{}, err
	}
	out := ComposerPresence{Others: make([]ComposerPresent, 0, len(rows))}
	for _, r := range rows {
		out.Others = append(out.Others, ComposerPresent{UserID: r.UserID.String(), Name: r.Name, Color: r.AvatarColor, Editing: r.Editing, SeenAt: r.SeenAt})
	}
	return out, nil
}

// Leave records that the caller closed the campaign.
func (s *Service) Leave(ctx context.Context, p *authz.Principal, id uuid.UUID) error {
	if p == nil || p.UserID == uuid.Nil {
		return nil
	}
	return s.Q.PresenceLeave(ctx, store.PresenceLeaveParams{CampaignID: id, UserID: p.UserID})
}
