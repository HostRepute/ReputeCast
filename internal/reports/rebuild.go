package reports

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/counters"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// A campaign's counters are kept in step with its messages as events come
// in (internal/counters). Once its results are final (FinalAfter), and
// whenever the counting rules change (counters.Version), they are rebuilt
// from the stored messages and events, so nothing a crash or a bug let
// through stays in a report.

// RebuildArgs rebuilds one campaign's counters.
type RebuildArgs struct {
	CampaignID uuid.UUID `json:"campaign_id"`
}

// Kind implements river.JobArgs.
func (RebuildArgs) Kind() string { return "reports.rebuild_counters" }

// InsertOpts puts rebuilds on the maintenance queue, one per campaign at a
// time.
func (RebuildArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, MaxAttempts: 5, UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: 10 * time.Minute}}
}

// RebuildWorker runs RebuildArgs.
type RebuildWorker struct {
	river.WorkerDefaults[RebuildArgs]
	Pool *pgxpool.Pool
	Now  func() time.Time
}

// Timeout bounds one rebuild (a campaign of a few hundred thousand
// messages takes seconds).
func (w *RebuildWorker) Timeout(*river.Job[RebuildArgs]) time.Duration { return 5 * time.Minute }

// Work implements river.Worker.
func (w *RebuildWorker) Work(ctx context.Context, job *river.Job[RebuildArgs]) error {
	now := time.Now()
	if w.Now != nil {
		now = w.Now()
	}
	_, err := counters.Rebuild(ctx, w.Pool, job.Args.CampaignID, now)
	if errors.Is(err, counters.ErrNotFound) {
		return nil // deleted meanwhile
	}
	return err
}

// RebuildSweepArgs queues the rebuilds that are due (periodic).
type RebuildSweepArgs struct{}

// Kind implements river.JobArgs.
func (RebuildSweepArgs) Kind() string { return "reports.rebuild_sweep" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (RebuildSweepArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 9 * time.Minute}}
}

// RebuildSweepWorker runs RebuildSweepArgs: campaigns whose results became
// final, or that were counted with older rules, a few at a time.
type RebuildSweepWorker struct {
	river.WorkerDefaults[RebuildSweepArgs]
	Q    *store.Queries
	Jobs func() *jobs.Client
	Now  func() time.Time
}

// Work implements river.Worker.
func (w *RebuildSweepWorker) Work(ctx context.Context, _ *river.Job[RebuildSweepArgs]) error {
	now := time.Now()
	if w.Now != nil {
		now = w.Now()
	}
	before := now.Add(-FinalAfter)
	ids, err := w.Q.CountersDueRebuilds(ctx, store.CountersDueRebuildsParams{FinalBefore: &before, Version: counters.Version, Lim: 20})
	if err != nil || len(ids) == 0 {
		return err
	}
	c := w.Jobs()
	if c == nil {
		return nil
	}
	params := make([]river.InsertManyParams, 0, len(ids))
	for _, id := range ids {
		params = append(params, river.InsertManyParams{Args: RebuildArgs{CampaignID: id}})
	}
	_, err = c.InsertMany(ctx, params)
	return err
}
