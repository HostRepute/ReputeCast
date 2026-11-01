package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/search"
)

// The command palette's search (review 2026-11 FE-1).

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		search.Register(h, &search.Service{Q: a.Q})
	})
}
