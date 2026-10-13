package overview

import (
	"context"
	"sort"
	"time"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/store"
)

// RecentLimit is how many campaigns "Recent campaigns" shows.
const RecentLimit = 4

// Recent returns the latest campaigns that went out (sending or sent), as
// the campaigns list shows them, so both screens agree.
func (s *Service) Recent(ctx context.Context, ws *store.Workspace) (OverviewRecentCampaigns, error) {
	out := OverviewRecentCampaigns{Items: []campaigns.CampaignListItem{}}
	for _, group := range []string{"sending", "sent"} {
		l, err := s.Campaigns.List(ctx, ws, campaigns.ListParams{Status: group, PageSize: RecentLimit})
		if err != nil {
			return out, err
		}
		out.Items = append(out.Items, l.Items...)
	}
	started := func(c campaigns.CampaignListItem) time.Time {
		if c.StartedAt != nil {
			return *c.StartedAt
		}
		return c.UpdatedAt
	}
	sort.SliceStable(out.Items, func(i, j int) bool { return started(out.Items[i]).After(started(out.Items[j])) })
	if len(out.Items) > RecentLimit {
		out.Items = out.Items[:RecentLimit]
	}
	return out, nil
}
