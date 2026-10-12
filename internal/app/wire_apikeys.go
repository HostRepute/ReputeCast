package app

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/apikeys"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		apikeys.Register(h, &apikeys.Service{
			Pool: a.Pool, Q: a.Q, Audit: a.Audit,
			PublicURL: func(context.Context) string { return a.Cfg.HTTP.PublicURL },
		})
	})
}
