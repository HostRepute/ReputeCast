package products

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/jobs"
)

// SweepArgs drops unfinished uploads and closes checkouts nobody finished.
type SweepArgs struct{}

// Kind implements river.JobArgs.
func (SweepArgs) Kind() string { return "products.sweep" }

// InsertOpts runs one sweep at a time.
func (SweepArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 30 * time.Minute}}
}

type sweepWorker struct {
	river.WorkerDefaults[SweepArgs]
	S *Service
}

func (w *sweepWorker) Work(ctx context.Context, _ *river.Job[SweepArgs]) error {
	return w.S.Sweep(ctx)
}

// Sweep drops uploads older than a day that never finished, and marks
// checkouts older than a day expired (Stripe's sessions last 24 hours).
func (s *Service) Sweep(ctx context.Context) error {
	day := s.now().Add(-24 * time.Hour)
	s.Files.SweepUploads(day)
	return s.Q.ProductCheckoutsExpire(ctx, day)
}

// RegisterJobs adds the delivery, confirmation and sweep workers.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &deliverWorker{S: s})
	river.AddWorker(reg.Workers, &confirmWorker{S: s})
	river.AddWorker(reg.Workers, &sweepWorker{S: s})
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return SweepArgs{}, nil }, false)
}
