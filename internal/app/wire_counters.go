package app

import (
	"time"

	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/reports"
	"github.com/hostrepute/reputecast/internal/tracking"
)

// Honest counters (review AN-3): campaigns' counters are rebuilt from
// their messages and events once their results are final, and opens and
// clicks the tracking writer couldn't record are applied again.
func init() {
	jobRegistrars = append(jobRegistrars, func(a *App) {
		jobsFn := func() *jobs.Client { return a.Jobs }
		river.AddWorker(a.JobReg.Workers, &reports.RebuildWorker{Pool: a.Pool})
		river.AddWorker(a.JobReg.Workers, &reports.RebuildSweepWorker{Q: a.Q, Jobs: jobsFn})
		river.AddWorker(a.JobReg.Workers, &tracking.RetryWorker{S: sendingTracker(a)})
		a.JobReg.Every(10*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return reports.RebuildSweepArgs{}, nil }, false)
		a.JobReg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return tracking.RetryArgs{}, nil }, false)
	})
}
