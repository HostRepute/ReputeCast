package app

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/mcpconnect"
)

func (a *App) mcpConnect() *mcpconnect.Service {
	s := &mcpconnect.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Now: a.now, PublicURL: func() string { return a.URLs.Base() },
		Campaigns: a.campaigns(), Website: websiteService(a), Automations: newAutomations(a), Segments: a.segments()}
	if a.Cfg != nil {
		s.Development = a.Cfg.IsDevelopment() || a.Cfg.Env == "test"
	}
	return s
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) { mcpconnect.Register(h, a.mcpConnect()) })
	publicRegistrars = append(publicRegistrars, func(a *App, r chi.Router) { a.mcpConnect().RegisterPublic(r) })
	jobRegistrars = append(jobRegistrars, func(a *App) { mcpconnect.RegisterJobs(a.JobReg, a.mcpConnect()) })
}
