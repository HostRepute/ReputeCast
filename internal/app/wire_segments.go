package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/segments"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		segments.Register(h, a.segments())
	})
}

// segments returns the segments service (shared by the audience and the
// tier job, which recounts segments nightly).
func (a *App) segments() *segments.Service {
	return &segments.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, SQL: &segments.SQLRunner{Pool: a.Pool}}
}
