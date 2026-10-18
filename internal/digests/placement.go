package digests

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// NextIssueCampaign is the campaign of a digest's next issue, for a
// placement test of it (internal/placement, X14): the one already built
// for a check or a test while the dialog is open, else built now. fresh
// builds it again from the feed and the template, as starting a test does,
// so the copies are the issue as it would go out at that moment.
func (s *Service) NextIssueCampaign(ctx context.Context, ws *store.Workspace, p *authz.Principal, digestID uuid.UUID, fresh bool) (uuid.UUID, error) {
	if !fresh {
		if _, err := s.Q.DigestGet(ctx, store.DigestGetParams{WorkspaceID: ws.ID, ID: digestID}); err == nil {
			if issue, err := s.Q.DigestIssueOpen(ctx, digestID); err == nil && issue.CampaignID != nil {
				return *issue.CampaignID, nil
			}
		}
	}
	return s.nextCampaign(ctx, ws, p, digestID)
}
