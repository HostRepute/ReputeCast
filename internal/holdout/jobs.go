package holdout

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
)

// MeasureHour is the workspace's local hour of the nightly measurement.
const MeasureHour = 4

// MeasureArgs measures one workspace's holdout group (holdout.measure).
type MeasureArgs struct {
	WorkspaceID uuid.UUID `json:"workspace_id"`
}

// Kind implements river.JobArgs.
func (MeasureArgs) Kind() string { return "holdout.measure" }

// InsertOpts implements river.JobArgsWithInsertOpts: one per workspace in
// any 10 minutes.
func (MeasureArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: MeasureEvery}}
}

// MeasureWorker runs MeasureArgs.
type MeasureWorker struct {
	river.WorkerDefaults[MeasureArgs]
	S *Service
}

// Timeout implements river.Worker: a large workspace takes a while.
func (w *MeasureWorker) Timeout(*river.Job[MeasureArgs]) time.Duration { return 10 * time.Minute }

// Work implements river.Worker.
func (w *MeasureWorker) Work(ctx context.Context, job *river.Job[MeasureArgs]) error {
	return w.S.Measure(ctx, job.Args.WorkspaceID)
}

// ScheduleArgs is the hourly tick that queues the nightly measurements
// (holdout.schedule).
type ScheduleArgs struct{}

// Kind implements river.JobArgs.
func (ScheduleArgs) Kind() string { return "holdout.schedule" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ScheduleArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 30 * time.Minute}}
}

// ScheduleWorker runs ScheduleArgs.
type ScheduleWorker struct {
	river.WorkerDefaults[ScheduleArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ScheduleWorker) Work(ctx context.Context, _ *river.Job[ScheduleArgs]) error {
	_, err := w.S.ScheduleMeasures(ctx)
	return err
}

// due reports whether a workspace's nightly measurement should run now:
// in its 04:00 hour once a day, or as soon as a day's run was missed.
func due(now time.Time, timezone string, last time.Time) bool {
	loc, err := time.LoadLocation(timezone)
	if err != nil || timezone == "" {
		loc = time.UTC
	}
	age := now.Sub(last)
	if now.In(loc).Hour() == MeasureHour && age > 20*time.Hour {
		return true
	}
	// A run that didn't happen (the server was off at 04:00).
	return last.After(time.Unix(0, 0)) && age > 26*time.Hour
}

// ScheduleMeasures queues the measurements that are due; it returns how
// many.
func (s *Service) ScheduleMeasures(ctx context.Context) (int, error) {
	rows, err := s.Q.HoldoutMeasureDue(ctx)
	if err != nil {
		return 0, err
	}
	now := s.now()
	n := 0
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		for _, r := range rows {
			if !due(now, r.Timezone, r.LastComputed) {
				continue
			}
			if _, err := s.Jobs.InsertTx(ctx, tx, MeasureArgs{WorkspaceID: r.WorkspaceID}, nil); err != nil {
				return err
			}
			n++
		}
		return nil
	})
	if err != nil {
		s.log().WarnContext(ctx, "holdout measurements not queued", "err", err)
	}
	return n, err
}
