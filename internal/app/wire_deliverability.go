package app

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/deliverability"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/dnscheck"
	"github.com/hostrepute/reputecast/internal/hostrepute"
)

// lazyJobs inserts through a.Jobs, which exists only after the job
// registrars ran (the services are built before the River client).
type lazyJobs struct{ a *App }

func (l lazyJobs) InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return l.a.Jobs.InsertTx(ctx, tx, args, opts)
}

// deliverabilityServices builds the HostRepute and Deliverability services
// for an App. They hold no state, so the API and the workers each build
// their own.
func deliverabilityServices(a *App) (*hostrepute.Service, *deliverability.Service) {
	cfg := a.Cfg
	if cfg == nil { // OpenAPI generation registers operations on an empty App
		d := config.Defaults()
		cfg = &d
	}
	resolver := dnscheck.NewResolver(dnscheck.Options{Servers: cfg.DNS.Servers, Timeout: cfg.DNS.Timeout})
	if cfg.IsDevelopment() {
		// The Acme sample domains answer from the demo zone, so checks in
		// the demo look like the design instead of failing on real DNS.
		resolver = demo.DeliverabilityDNSOverlay(resolver)
	}
	jobs := lazyJobs{a}
	hr := &hostrepute.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Cfg: cfg, Log: a.Log, Jobs: jobs}
	dl := &deliverability.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Log: a.Log, HostRepute: hr, Jobs: jobs, DNS: resolver}
	hr.Status = dl.AssetStatuses
	hr.Fallback = dl.Fallback
	return hr, dl
}

func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		hr, dl := deliverabilityServices(a)
		hostrepute.Register(h, hr)
		deliverability.Register(h, dl)
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		hr, dl := deliverabilityServices(a)
		river.AddWorker(a.JobReg.Workers, &hostrepute.SyncWorker{S: hr})
		river.AddWorker(a.JobReg.Workers, &hostrepute.ScheduleWorker{S: hr})
		river.AddWorker(a.JobReg.Workers, &deliverability.CheckWorker{S: dl})
		river.AddWorker(a.JobReg.Workers, &deliverability.ScheduleWorker{S: dl})
		a.JobReg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return hostrepute.ScheduleArgs{}, nil }, true)
		a.JobReg.Every(3*time.Hour, func() (river.JobArgs, *river.InsertOpts) { return deliverability.ScheduleArgs{}, nil }, true)
	})
}
