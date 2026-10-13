package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/roles"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		roles.Register(h, &roles.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit})
	})
}
