package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/overview"
)

// Overview: each card reads the domain that owns its data.
func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		_, dl := deliverabilityServices(a)
		overview.Register(h, &overview.Service{
			Pool: a.Pool, Q: a.Q, Deliverability: dl, Providers: a.providersService(),
			Campaigns: &campaigns.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit},
			Audience:  overview.AudienceCards{Service: a.audience(), Q: a.Q},
		})
	})
}
