package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/appearance"
)

// Appearance: the member's light or dark theme (M54).

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		appearance.Register(h, &appearance.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit})
	})
}
