package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/campaigns"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		campaigns.Register(h, &campaigns.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit})
	})
}
