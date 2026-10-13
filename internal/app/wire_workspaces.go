package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/workspaces"
)

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		workspaces.Register(h, &workspaces.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit})
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		workspaces.RegisterJobs(a.JobReg, a.Pool, a.Log)
	})
}
