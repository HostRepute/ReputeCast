package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/integrations"
	"github.com/hostrepute/reputecast/internal/languages"
	"github.com/hostrepute/reputecast/internal/reports"
)

// ai is the AI assistant with the workspace's own model (M22). It holds no
// state, so the API, the catalog and the worker each build their own.
func (a *App) ai() *ai.Service {
	cfg := a.Cfg
	if cfg == nil { // OpenAPI generation registers operations on an empty App
		d := config.Defaults()
		cfg = &d
	}
	return &ai.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log,
		// Model servers on a private address work when the instance allows
		// private addresses, as SMTP servers and community adapters do.
		AllowPrivate: func() bool { return cfg.Providers.AllowPrivateNetworks },
		// The demo's sample connection answers from its recordings in
		// development and in tests, never from Anthropic.
		Development: cfg.IsDevelopment() || cfg.Env == "test",
		Segments:    a.segments(),
		Reports:     &reports.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, URLs: a.URLs, Languages: &languages.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit}},
		Languages:   languagesService(a),
	}
}

// catalogSources fill the integrations hub's later groups: AI models.
func catalogSources(a *App) map[string]integrations.GroupSource {
	return map[string]integrations.GroupSource{"ai": a.ai().CatalogEntries}
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		ai.Register(h, a.ai())
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		ai.RegisterJobs(a.JobReg, a.ai())
	})
}
