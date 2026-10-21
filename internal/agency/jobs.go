package agency

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/jobs"
)

// RefreshArgs scores every workspace of the agency and rolls up their
// usage (hourly).
type RefreshArgs struct{}

// Kind implements river.JobArgs.
func (RefreshArgs) Kind() string { return "agency.refresh" }

// InsertOpts keeps one refresh at a time.
func (RefreshArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour}}
}

// RefreshWorker runs the hourly refresh.
type RefreshWorker struct {
	river.WorkerDefaults[RefreshArgs]
	S   *Service
	Log *slog.Logger
}

// Timeout gives the refresh time for a large agency.
func (w *RefreshWorker) Timeout(*river.Job[RefreshArgs]) time.Duration { return 30 * time.Minute }

// Work implements river.Worker.
func (w *RefreshWorker) Work(ctx context.Context, _ *river.Job[RefreshArgs]) error {
	if err := w.S.RefreshAll(ctx); err != nil {
		w.Log.WarnContext(ctx, "agency refresh failed", "err", err)
		return err
	}
	return nil
}

// RegisterJobs adds the hourly refresh of health and usage.
func RegisterJobs(reg *jobs.Registry, s *Service, log *slog.Logger) {
	river.AddWorker(reg.Workers, &RefreshWorker{S: s, Log: log})
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return RefreshArgs{}, nil }, true)
}
