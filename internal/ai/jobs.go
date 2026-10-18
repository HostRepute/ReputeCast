package ai

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/jobs"
)

// RetentionArgs drops the activity log's rows older than 90 days.
type RetentionArgs struct{}

// Kind implements river.JobArgs.
func (RetentionArgs) Kind() string { return "ai.retention" }

// InsertOpts runs it on the maintenance queue, once at a time.
func (RetentionArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour}}
}

// RetentionWorker purges the activity log.
type RetentionWorker struct {
	river.WorkerDefaults[RetentionArgs]
	S *Service
}

// Work implements river.Worker.
func (w *RetentionWorker) Work(ctx context.Context, _ *river.Job[RetentionArgs]) error {
	n, err := w.S.Purge(ctx)
	if err == nil && n > 0 {
		w.S.log().Info("ai: activity log purged", "rows", n)
	}
	return err
}

// RegisterJobs adds the assistant's job: the 90-day retention, every 6 hours.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &RetentionWorker{S: s})
	reg.Every(6*time.Hour, func() (river.JobArgs, *river.InsertOpts) { return RetentionArgs{}, nil }, true)
}
