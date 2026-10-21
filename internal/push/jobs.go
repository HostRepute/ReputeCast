package push

import (
	"context"
	"errors"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/jobs"
)

// One job of each kind waits or runs at a time.
var activeStates = []rivertype.JobState{rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRunning,
	rivertype.JobStateScheduled, rivertype.JobStateRetryable}

// dispatchBudget is how long one dispatcher run sends before it snoozes.
const dispatchBudget = 20 * time.Second

// interrupted turns a job cancelled by shutdown into an immediate snooze.
func interrupted(ctx context.Context, err error) error {
	if err != nil && ctx.Err() != nil && errors.Is(err, context.Canceled) {
		return river.JobSnooze(0)
	}
	return err
}

// DispatchArgs sends the notifications that are due.
type DispatchArgs struct{}

// Kind implements river.JobArgs.
func (DispatchArgs) Kind() string { return "push.dispatch" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (DispatchArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueSending, UniqueOpts: river.UniqueOpts{ByState: activeStates}}
}

// DispatchWorker runs DispatchArgs.
type DispatchWorker struct {
	river.WorkerDefaults[DispatchArgs]
	S *Service
}

// Timeout bounds one run.
func (w *DispatchWorker) Timeout(*river.Job[DispatchArgs]) time.Duration { return 2 * time.Minute }

// Work implements river.Worker.
func (w *DispatchWorker) Work(ctx context.Context, _ *river.Job[DispatchArgs]) error {
	again, done, err := w.S.Dispatch(ctx, dispatchBudget)
	if err != nil || done {
		return interrupted(ctx, err)
	}
	return river.JobSnooze(max(again, 10*time.Millisecond))
}

// NextRetry implements river.Worker: a failed run goes again within seconds.
func (w *DispatchWorker) NextRetry(job *river.Job[DispatchArgs]) time.Time {
	return time.Now().Add(min(time.Duration(max(job.Attempt, 1))*2*time.Second, 30*time.Second))
}

// StartArgs starts the scheduled sends whose time has come.
type StartArgs struct{}

// Kind implements river.JobArgs.
func (StartArgs) Kind() string { return "push.start" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (StartArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueSending, UniqueOpts: river.UniqueOpts{ByState: activeStates}}
}

// StartWorker runs StartArgs.
type StartWorker struct {
	river.WorkerDefaults[StartArgs]
	S *Service
}

// Work implements river.Worker.
func (w *StartWorker) Work(ctx context.Context, _ *river.Job[StartArgs]) error {
	return interrupted(ctx, w.S.StartDue(ctx))
}

// CheckArgs checks the snippet sites' service workers, once a day each.
type CheckArgs struct{}

// Kind implements river.JobArgs.
func (CheckArgs) Kind() string { return "push.check" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (CheckArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Minute}}
}

// CheckWorker runs CheckArgs.
type CheckWorker struct {
	river.WorkerDefaults[CheckArgs]
	S *Service
}

// Timeout bounds one pass.
func (w *CheckWorker) Timeout(*river.Job[CheckArgs]) time.Duration { return 10 * time.Minute }

// Work implements river.Worker.
func (w *CheckWorker) Work(ctx context.Context, _ *river.Job[CheckArgs]) error {
	return w.S.CheckStale(ctx, 24*time.Hour)
}

// SweepArgs purges clicks after 90 days and prompt counts after 400.
type SweepArgs struct{}

// Kind implements river.JobArgs.
func (SweepArgs) Kind() string { return "push.sweep" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (SweepArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 5 * time.Hour}}
}

// SweepWorker runs SweepArgs.
type SweepWorker struct {
	river.WorkerDefaults[SweepArgs]
	S *Service
}

// Work implements river.Worker.
func (w *SweepWorker) Work(ctx context.Context, _ *river.Job[SweepArgs]) error {
	now := w.S.now()
	for _, q := range []string{
		`DELETE FROM push_clicks WHERE occurred_at < $1::timestamptz - interval '90 days'`,
		`DELETE FROM push_prompt_days WHERE day < ($1::timestamptz - interval '400 days')::date`,
		// Removed subscriptions keep their counts for 90 days.
		`DELETE FROM push_subscriptions WHERE removed_at < $1::timestamptz - interval '90 days'`,
	} {
		if _, err := w.S.Pool.Exec(ctx, q, now); err != nil {
			return err
		}
	}
	return nil
}

// RegisterJobs adds the push workers and their schedules.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &DispatchWorker{S: s})
	river.AddWorker(reg.Workers, &StartWorker{S: s})
	river.AddWorker(reg.Workers, &CheckWorker{S: s})
	river.AddWorker(reg.Workers, &SweepWorker{S: s})
	reg.Every(30*time.Second, func() (river.JobArgs, *river.InsertOpts) { return StartArgs{}, nil }, true)
	reg.Every(30*time.Second, func() (river.JobArgs, *river.InsertOpts) { return DispatchArgs{}, nil }, true)
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return CheckArgs{}, nil }, false)
	reg.Every(6*time.Hour, func() (river.JobArgs, *river.InsertOpts) { return SweepArgs{}, nil }, false)
}

// CheckStale checks the snippet sites not checked for a while.
func (s *Service) CheckStale(ctx context.Context, age time.Duration) error {
	sites, err := s.Q.PushSitesAll(ctx)
	if err != nil {
		return err
	}
	cut := s.now().Add(-age)
	for _, st := range sites {
		if st.CheckedAt != nil && st.CheckedAt.After(cut) {
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ws, err := s.Q.GetWorkspace(ctx, st.WorkspaceID)
		if err != nil {
			continue
		}
		if err := s.check(ctx, &ws, st); err != nil && !errors.Is(err, context.Canceled) {
			s.log().WarnContext(ctx, "push site check", "site", st.ID, "err", err)
		}
	}
	return nil
}
