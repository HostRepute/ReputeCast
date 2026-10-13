package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/audience"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		audience.Register(h, a.audience(), a.Secrets)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		audience.RegisterJobs(a.JobReg, a.audience(), a.Log)
	})
}

// audience returns the audience service.
func (a *App) audience() *audience.Service {
	return &audience.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Segments: a.segments(), Mail: a.Mail, Links: a.Links, URLs: a.URLs}
}
