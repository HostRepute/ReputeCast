package webhooks

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/jobs"
)

// DeliverArgs makes one attempt at one delivery.
type DeliverArgs struct {
	DeliveryID uuid.UUID `json:"delivery_id"`
}

// Kind implements river.JobArgs.
func (DeliverArgs) Kind() string { return "webhooks.deliver" }

// InsertOpts puts deliveries on the events queue. River's own retries are
// off: the delivery row tracks attempts and backoff itself.
func (DeliverArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, MaxAttempts: 1}
}

// DeliverWorker posts a delivery and schedules the next attempt on failure.
type DeliverWorker struct {
	river.WorkerDefaults[DeliverArgs]
	S   *Service
	Log *slog.Logger
}

// Timeout bounds one attempt.
func (w *DeliverWorker) Timeout(*river.Job[DeliverArgs]) time.Duration { return 30 * time.Second }

// Work implements river.Worker.
func (w *DeliverWorker) Work(ctx context.Context, job *river.Job[DeliverArgs]) error {
	next, err := w.S.Attempt(ctx, job.Args.DeliveryID)
	if err != nil {
		return err
	}
	if next != nil {
		if c, err := river.ClientFromContextSafely[pgx.Tx](ctx); err == nil {
			if _, err := c.Insert(ctx, DeliverArgs{DeliveryID: job.Args.DeliveryID}, &river.InsertOpts{ScheduledAt: *next}); err != nil {
				// The dispatcher picks the delivery up again after 10 minutes.
				w.Log.WarnContext(ctx, "schedule webhook retry", "delivery", job.Args.DeliveryID, "err", err)
			}
		}
	}
	return nil
}

// DispatchArgs is the periodic sweep that queues due deliveries (emitted
// by other domains inside their transactions) and purges old ones.
type DispatchArgs struct{}

// Kind implements river.JobArgs.
func (DispatchArgs) Kind() string { return "webhooks.dispatch" }

// InsertOpts keeps one sweep at a time.
func (DispatchArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, UniqueOpts: river.UniqueOpts{ByPeriod: 10 * time.Second}}
}

// DispatchWorker queues jobs for due deliveries.
type DispatchWorker struct {
	river.WorkerDefaults[DispatchArgs]
	S   *Service
	Log *slog.Logger
}

// Work implements river.Worker.
func (w *DispatchWorker) Work(ctx context.Context, _ *river.Job[DispatchArgs]) error {
	q := w.S.Q
	if err := q.HooksStaleDelivering(ctx); err != nil {
		return err
	}
	ids, err := q.HooksDue(ctx, 500)
	if err != nil {
		return err
	}
	if len(ids) > 0 {
		c, err := river.ClientFromContextSafely[pgx.Tx](ctx)
		if err != nil {
			return err
		}
		params := make([]river.InsertManyParams, len(ids))
		for i, id := range ids {
			params[i] = river.InsertManyParams{Args: DeliverArgs{DeliveryID: id}}
		}
		if _, err := c.InsertMany(ctx, params); err != nil {
			return err
		}
	}
	if n, err := q.HooksPurgeDeliveries(ctx); err == nil && n > 0 {
		w.Log.InfoContext(ctx, "purged webhook deliveries", "count", n)
	}
	return nil
}

// kick queues immediate attempts from API handlers (retry now, resume).
func (s *Service) kick(ctx context.Context, ids []uuid.UUID) {
	if s.Jobs == nil || len(ids) == 0 {
		return
	}
	params := make([]river.InsertManyParams, len(ids))
	for i, id := range ids {
		params[i] = river.InsertManyParams{Args: DeliverArgs{DeliveryID: id}}
	}
	_, _ = s.Jobs.InsertMany(ctx, params)
}

// RegisterJobs adds the delivery worker and the periodic dispatcher.
func RegisterJobs(reg *jobs.Registry, s *Service, log *slog.Logger) {
	river.AddWorker(reg.Workers, &DeliverWorker{S: s, Log: log})
	river.AddWorker(reg.Workers, &DispatchWorker{S: s, Log: log})
	reg.Every(15*time.Second, func() (river.JobArgs, *river.InsertOpts) { return DispatchArgs{}, nil }, true)
}
