package domains

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// CheckArgs is the periodic DNS re-verification of sending domains: every
// domain is checked again every 30 minutes, so a change shows up in Send
// Readiness within the hour.
type CheckArgs struct{}

// Kind implements river.JobArgs.
func (CheckArgs) Kind() string { return "domains.check" }

// InsertOpts keeps one sweep at a time.
func (CheckArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 4 * time.Minute}}
}

// CheckWorker re-verifies the domains whose last check is older than
// CheckEvery, at most 200 per run.
type CheckWorker struct {
	river.WorkerDefaults[CheckArgs]
	S *Service
}

// Timeout bounds one sweep.
func (w *CheckWorker) Timeout(*river.Job[CheckArgs]) time.Duration { return 4 * time.Minute }

// Work implements river.Worker.
func (w *CheckWorker) Work(ctx context.Context, _ *river.Job[CheckArgs]) error {
	before := w.S.now().Add(-CheckEvery)
	ids, err := w.S.Q.DomDue(ctx, store.DomDueParams{Before: &before, Lim: 200})
	if err != nil {
		return err
	}
	var errs []error
	for _, id := range ids {
		d, err := w.S.Q.DomGetAny(ctx, id)
		if err != nil {
			continue
		}
		cctx, cancel := context.WithTimeout(ctx, time.Minute)
		_, err = w.S.Check(cctx, d)
		cancel()
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", d.Domain, err))
		}
	}
	return errors.Join(errs...)
}
