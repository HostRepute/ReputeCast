package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/templates"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		templates.Register(h, &templates.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Brand: a.brandKit()})
	})
}
