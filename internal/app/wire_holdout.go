package app

import (
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/holdout"
)

// The holdout group's results (M63 G-33): Audience › Holdout, the nightly
// measurement at 04:00 workspace time and Recompute. Its settings are Send
// Readiness's (Settings › Sending rules, wire_readiness.go).
func init() {
	apiRegistrars = append(apiRegistrars, func(a *App, h huma.API) {
		holdout.Register(h, a.holdout())
	})
	jobRegistrars = append(jobRegistrars, func(a *App) {
		river.AddWorker(a.JobReg.Workers, &holdout.MeasureWorker{S: a.holdout()})
		river.AddWorker(a.JobReg.Workers, &holdout.ScheduleWorker{S: a.holdout()})
		a.JobReg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return holdout.ScheduleArgs{}, nil }, false)
	})
}

var holdoutServices sync.Map // *App → *holdout.Service

// holdout returns the app's holdout service; it queues jobs through the
// River client, created after the job registrars run.
func (a *App) holdout() *holdout.Service {
	if v, ok := holdoutServices.Load(a); ok {
		return v.(*holdout.Service)
	}
	s := &holdout.Service{Pool: a.Pool, Q: a.Q, Jobs: lazyJobs{a}, Log: a.Log}
	v, _ := holdoutServices.LoadOrStore(a, s)
	return v.(*holdout.Service)
}

// Holdout is the holdout domain (tests set its clock and measure).
func (a *App) Holdout() *holdout.Service { return a.holdout() }
