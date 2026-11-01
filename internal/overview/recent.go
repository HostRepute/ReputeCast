package overview

import (
	"context"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/store"
)

// RecentLimit is how many campaigns "Recent campaigns" shows.
const RecentLimit = 4

// Recent returns the latest campaigns that went out (sending or sent, latest
// start first), as the campaigns list shows them, so both screens agree:
// one page of the list without its totals (review 2026-11 CA-7).
func (s *Service) Recent(ctx context.Context, ws *store.Workspace) (OverviewRecentCampaigns, error) {
	l, err := s.Campaigns.List(ctx, ws, campaigns.ListParams{Status: "recent", PageSize: RecentLimit, ItemsOnly: true})
	if err != nil {
		return OverviewRecentCampaigns{Items: []campaigns.CampaignListItem{}}, err
	}
	return OverviewRecentCampaigns{Items: l.Items}, nil
}
