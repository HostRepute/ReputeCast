package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/auditlog"
)

func (a *App) auditlog() *auditlog.Service {
	return &auditlog.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets,
		AllowPrivate: func() bool { return a.Cfg.Webhooks.AllowPrivateNetworks },
	}
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		auditlog.Register(h, a.auditlog())
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		auditlog.RegisterJobs(a.JobReg, a.auditlog(), a.Log)
	})
}
