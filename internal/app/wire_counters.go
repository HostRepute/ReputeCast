package app

import (
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/counters"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/reports"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// trackerEngagement counts the opens and clicks a provider reports with
// the App's tracking service (review AN-4), looked up when the first one
// arrives: the tracker's own services build normalizers.
type trackerEngagement struct{ a *App }

// Begin implements ingest.Engagement.
func (t trackerEngagement) Begin(l *counters.Ledger) ingest.EngagementBatch {
	return sendingTracker(t.a).Begin(l)
}

// Honest counters and bot clicks (review AN-3, AN-2): campaigns' counters
// are rebuilt from their messages and events once their results are
// final; opens and clicks the tracking writer couldn't record are applied
// again; clicks settle before they count; instance admins keep the list
// of scanner and privacy-proxy networks.
func init() {
	trackingHooks = append(trackingHooks, func(a *App, t *tracking.Service) { t.Audit = a.Audit })
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		tracking.RegisterNetworks(h, sendingTracker(a))
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		jobsFn := func() *jobs.Client { return a.Jobs }
		t := sendingTracker(a)
		river.AddWorker(a.JobReg.Workers, &reports.RebuildWorker{Pool: a.Pool})
		river.AddWorker(a.JobReg.Workers, &reports.RebuildSweepWorker{Q: a.Q, Jobs: jobsFn})
		river.AddWorker(a.JobReg.Workers, &tracking.RetryWorker{S: t})
		river.AddWorker(a.JobReg.Workers, &tracking.SettleWorker{S: t})
		a.JobReg.Every(10*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return reports.RebuildSweepArgs{}, nil }, false)
		a.JobReg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return tracking.RetryArgs{}, nil }, false)
		a.JobReg.Every(15*time.Second, func() (river.JobArgs, *river.InsertOpts) { return tracking.SettleArgs{}, nil }, false)
	})
}
