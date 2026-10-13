package app

import (
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/instance"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/sso"
)

// W14 · Instance admin, SSO and social sign-in, updates.

// accountsForSSO builds the accounts service the SSO callbacks create
// sessions with (the same one password sign-in uses; it holds no state).
func (a *App) accountsForSSO() *accounts.Service {
	return &accounts.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Cfg: a.Cfg, Mail: a.notifyMailer(), Log: a.Log,
		Breach: authn.NewBreachChecker(), OnPolicyChange: a.InvalidatePolicy,
	}
}

func (a *App) ssoService() *sso.Service {
	return &sso.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Cfg: a.Cfg, Log: a.Log,
		Accounts: a.accountsForSSO(), OnPolicyChange: a.InvalidatePolicy}
}

func (a *App) instanceService() *instance.Service {
	return &instance.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Cfg: a.Cfg, Log: a.Log,
		Accounts: a.accountsForSSO(), SSO: a.ssoService(),
		Mail:           func() mailer.Sender { return a.notifyMailer() },
		Jobs:           func() *jobs.Client { return a.Jobs },
		OnPolicyChange: a.InvalidatePolicy}
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		sso.Register(h, a.ssoService())
		instance.Register(h, a.instanceService())
	})
	publicRegistrars = append(publicRegistrars, func(a *App, r chi.Router) {
		a.ssoService().Routes(r)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		svc := a.instanceService()
		river.AddWorker(a.JobReg.Workers, &instance.ReleaseCheckWorker{S: svc})
		river.AddWorker(a.JobReg.Workers, &instance.WorkspacePurgeWorker{Pool: a.Pool, Audit: a.Audit, Log: a.Log})
		// Once a day; the worker makes no request when the check is off.
		a.JobReg.Every(24*time.Hour, func() (river.JobArgs, *river.InsertOpts) { return instance.ReleaseCheckArgs{}, nil }, true)
	})
}
