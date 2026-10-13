package app

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/sessionext"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		sessionext.Register(h, &sessionext.Service{
			Pool:  a.Pool,
			Q:     a.Q,
			Audit: a.Audit,
			// Resolved per request: a.Auth doesn't exist when only the OpenAPI
			// document is being generated.
			Policy: func(ctx context.Context) (api.Policy, error) { return a.Auth.Policy(ctx) },
		})
	})
}
