package manage

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// campaignSender is a campaign sender's verification per provider ID, for
// vetting where its held share could move (review GS-5); nil when the
// campaign has no sender.
func campaignSender(ctx context.Context, q *store.Queries, wsID, campaignID uuid.UUID) map[string]string {
	cc, err := q.SendingCampaignContext(ctx, campaignID)
	if err != nil || cc.WorkspaceID != wsID || cc.SenderID == nil {
		return nil
	}
	s, err := q.SenderGet(ctx, store.SenderGetParams{WorkspaceID: wsID, ID: *cc.SenderID})
	if err != nil {
		return nil
	}
	return state.SenderVerification(s.Verification)
}
