package campaigns

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// SampleRender is the saved campaign (one A/B variant: A, B or C) as the
// synthetic sample contact gets it, and the saved version it comes from.
// Real renders (internal/renders, CO-13) send it to a service outside
// ReputeCast, so no contact's data leaves the instance.
func (s *Service) SampleRender(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, variant string) (ComposerPreview, int, error) {
	c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return ComposerPreview{}, 0, api.MapError(err, "Campaign")
	}
	in := ComposerPreviewRequest{synthetic: true}
	if variant == "B" || variant == "C" {
		in.Variant = variant
	}
	out, err := s.Preview(ctx, ws, p, id, in)
	return out, int(c.Version), err
}

// CampaignName is a campaign's name, or a 404 when the workspace has no
// such campaign.
func (s *Service) CampaignName(ctx context.Context, ws *store.Workspace, id uuid.UUID) (string, error) {
	c, err := s.Q.GetComposerCampaign(ctx, store.GetComposerCampaignParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return "", api.MapError(err, "Campaign")
	}
	return c.Name, nil
}
