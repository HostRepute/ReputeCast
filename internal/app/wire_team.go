package app

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/team"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		team.Register(h, &team.Service{
			Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Mail: a.Mail,
			PublicURL: func(context.Context) string { return a.Cfg.HTTP.PublicURL },
		})
	})
}
