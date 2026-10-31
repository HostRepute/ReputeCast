package app

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/connect"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/routing"
)

// Connect a sending provider (the catalog and the six-step setup) and
// Settings › Providers.
func (a *App) connectService() *connect.Service {
	cfg := a.Cfg
	if cfg == nil { // OpenAPI generation registers operations on an empty App
		d := config.Defaults()
		cfg = &d
	}
	return &connect.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Factory: a.providerFactory(),
		Domains:      a.domainsService(),
		Routing:      &routing.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Mailbox: a.mailboxClassifier()},
		Suppressions: a.suppressionSync(),
		Providers:    a.providersService(),
		Deliver:      a.deliverer(),
		Links:        a.Links,
		Jobs:         func() *jobs.Client { return a.Jobs },
		URLs:         a.URLs,
		TemplateURL:  cfg.Connect.SESTemplateURL,
		// Integration tests run on 127.0.0.1: let the webhook self-test
		// reach it instead of stopping at "private address".
		AllowPrivateURL:      cfg.Env == "test",
		AllowPrivateNetworks: cfg.Providers.AllowPrivateNetworks,
		OnRoutingChange: func() {
			a.forgetSystemAvailability()
		},
		Log: a.Log,
	}
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		connect.Register(h, a.connectService())
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		svc := a.connectService()
		n := a.normalizer()
		river.AddWorker(a.JobReg.Workers, &connect.ProvisionWorker{S: svc})
		river.AddWorker(a.JobReg.Workers, &connect.ScanWorker{S: svc})
		river.AddWorker(a.JobReg.Workers, &connect.PullWorker{S: svc})
		river.AddWorker(a.JobReg.Workers, &connect.PollWorker{S: svc, Backfill: func(ctx context.Context, id uuid.UUID) (int, error) { return n.Backfill(ctx, id) }})
		a.JobReg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return connect.PollArgs{}, nil }, false)
	})
}
