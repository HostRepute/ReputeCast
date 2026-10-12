package app

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/maintenance"
	"github.com/hostrepute/reputecast/internal/meta"
)

// registerAPI adds every domain's Huma operations. Each domain lives in its
// own package and exposes Register(huma.API, *Service).
func registerAPI(a *App, h huma.API) {
	meta.Register(h, &meta.Service{Q: a.Q})
	accounts.Register(h, &accounts.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Cfg: a.Cfg, Mail: a.Mail, Log: a.Log,
		Breach: authn.NewBreachChecker(), OnPolicyChange: a.InvalidatePolicy,
	})
	campaigns.Register(h, &campaigns.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit})
}

// registerPublic adds server-rendered public pages (subscribe, confirm,
// preferences, unsubscribe, archive) under /p/.
func registerPublic(a *App, r chi.Router) {
	_ = a
	_ = r
}

// registerIngest adds provider webhook endpoints under /hooks/.
func registerIngest(a *App, r chi.Router) {
	_ = a
	_ = r
}

// registerJobs adds River workers and periodic jobs.
func registerJobs(a *App) {
	maintenance.Register(a.JobReg, a.Pool, a.Log)
}

// RegisterOperations registers every operation on h without a database, for
// generating the OpenAPI document. Handlers are never called.
func RegisterOperations(h huma.API) {
	registerAPI(&App{}, h)
}
