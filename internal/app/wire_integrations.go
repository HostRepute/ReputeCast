package app

import (
	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/integrations"
)

// integrations is the integrations hub (M16). It holds no state, so the
// API, the authenticator, the webhooks service and the workers each use
// their own.
func (a *App) integrations() *integrations.Service {
	cfg := a.Cfg
	if cfg == nil { // OpenAPI generation registers operations on an empty App
		d := config.Defaults()
		cfg = &d
	}
	return &integrations.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Log: a.Log, Cfg: cfg,
		PublicURL:    a.URLs.Base,
		URLCheck:     a.accountsForSSO(),
		Confirm:      a.audience(),
		AllowPrivate: func() bool { return cfg.Webhooks.AllowPrivateNetworks },
		Languages:    draftLanguages{languagesService(a)},
		Sources:      map[string]integrations.GroupSource{"verification": verificationCatalog(a)},
	}
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		integrations.Register(h, a.integrations())
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		integrations.RegisterJobs(a.JobReg, a.integrations())
	})
	// Integration keys work only for their integration's operations, and
	// their calls go to its call log.
	startHooks = append(startHooks, func(a *App) error {
		a.Auth.Integrations = a.integrations()
		return nil
	})
}
