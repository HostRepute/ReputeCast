package app

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/overview"
	"github.com/hostrepute/reputecast/internal/store"
)

// Overview: each card reads the domain that owns its data.
func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		_, dl := deliverabilityServices(a)
		overview.Register(h, &overview.Service{
			Pool: a.Pool, Q: a.Q, Deliverability: dl, Providers: a.providersService(),
			Campaigns: &campaigns.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit},
			Audience:  overview.AudienceCards{Service: a.audience(), Q: a.Q},
			NextIssue: overviewNextIssue(a),
		})
	})
}

// overviewNextIssue finds a recurring digest's next issue for "Next
// scheduled campaign" (wire_digests.go sets it).
var overviewNextIssue = func(*App) func(ctx context.Context, ws *store.Workspace) (*overview.OverviewNextIssue, error) {
	return nil
}
