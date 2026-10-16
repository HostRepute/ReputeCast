package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/webhooks"
)

func (a *App) webhooks() *webhooks.Service {
	return &webhooks.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Jobs: a.Jobs,
		AllowPrivate: func() bool { return a.Cfg.Webhooks.AllowPrivateNetworks },
		// Endpoints integrations manage (integrations hub, M16).
		Managed: a.integrations(),
	}
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		webhooks.Register(h, a.webhooks())
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		webhooks.RegisterJobs(a.JobReg, a.webhooks(), a.Log)
	})
}
