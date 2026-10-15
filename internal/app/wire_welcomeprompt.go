package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/welcomeprompt"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		welcomeprompt.Register(h, &welcomeprompt.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit})
	})
}
