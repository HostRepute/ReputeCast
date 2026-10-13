package app

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/maintenance"
	"github.com/hostrepute/reputecast/internal/meta"
)

// Domains plug themselves in from their own file in this package
// (wire_<domain>.go) with an init function that appends to these lists, so
// adding a domain never touches a shared file:
//
//	func init() {
//		apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
//			campaigns.Register(h, &campaigns.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit})
//		})
//	}
var (
	apiRegistrars    []func(*App, huma.API)
	publicRegistrars []func(*App, chi.Router)
	ingestRegistrars []func(*App, chi.Router)
	jobRegistrars    []func(*App)
	// startHooks run after dependencies exist and before serving, in file
	// order (wire domains that need each other, such as the system mailer).
	startHooks []func(*App) error
)

// registerAPI adds every domain's Huma operations.
func registerAPI(a *App, h huma.API) {
	meta.Register(h, &meta.Service{Q: a.Q})
	accounts.Register(h, &accounts.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Cfg: a.Cfg, Mail: a.Mail, Log: a.Log,
		Breach: authn.NewBreachChecker(), OnPolicyChange: a.InvalidatePolicy,
		OnPublicURL: func(ctx context.Context, before, after string) { _, _ = a.publicURLSaved(ctx, before, after) },
	})
	for _, r := range apiRegistrars {
		r(a, h)
	}
}

// registerPublic adds server-rendered public pages (subscribe, confirm,
// preferences, unsubscribe, archive) under /p/ and tracking under /t/.
func registerPublic(a *App, r chi.Router) {
	for _, reg := range publicRegistrars {
		reg(a, r)
	}
}

// registerIngest adds provider webhook endpoints under /hooks/.
func registerIngest(a *App, r chi.Router) {
	for _, reg := range ingestRegistrars {
		reg(a, r)
	}
}

// registerJobs adds River workers and periodic jobs.
func registerJobs(a *App) {
	maintenance.Register(a.JobReg, a.Pool, a.Log)
	for _, r := range jobRegistrars {
		r(a)
	}
}

// RegisterOperations registers every operation on h without a database, for
// generating the OpenAPI document. Handlers are never called.
func RegisterOperations(h huma.API) {
	registerAPI(&App{}, h)
}
