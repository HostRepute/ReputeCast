package whatsapp

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/jobs"
)

// Jobs: whatsapp.normalize turns a stored webhook into events;
// whatsapp.start plans scheduled sends; whatsapp.dispatch sends to Meta;
// whatsapp.sync checks each account and its templates every hour;
// whatsapp.sweep purges raw webhooks and people's messages after 90 days.

// One job of each kind waits or runs at a time.
var activeStates = []rivertype.JobState{rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRunning,
	rivertype.JobStateScheduled, rivertype.JobStateRetryable}

// dispatchBudget is how long one dispatcher run sends before it snoozes.
const dispatchBudget = 20 * time.Second

func interrupted(ctx context.Context, err error) error {
	if err != nil && ctx.Err() != nil && errors.Is(err, context.Canceled) {
		return river.JobSnooze(0)
	}
	return err
}

// NormalizeArgs normalizes one stored webhook.
type NormalizeArgs struct {
	RawEventID uuid.UUID `json:"raw_event_id"`
}

// Kind implements river.JobArgs.
func (NormalizeArgs) Kind() string { return "whatsapp.normalize" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (NormalizeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 10}
}

// NormalizeWorker runs NormalizeArgs.
type NormalizeWorker struct {
	river.WorkerDefaults[NormalizeArgs]
	S *Service
}

// Work implements river.Worker.
func (w *NormalizeWorker) Work(ctx context.Context, job *river.Job[NormalizeArgs]) error {
	return w.S.Normalize(ctx, job.Args.RawEventID)
}

// DispatchArgs sends the messages that are due.
type DispatchArgs struct{}

// Kind implements river.JobArgs.
func (DispatchArgs) Kind() string { return "whatsapp.dispatch" }

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

// StartArgs starts the scheduled sends whose time has come.
type StartArgs struct{}

// Kind implements river.JobArgs.
func (StartArgs) Kind() string { return "whatsapp.start" }

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

// SyncArgs checks every account and syncs its templates.
type SyncArgs struct{}

// Kind implements river.JobArgs.
func (SyncArgs) Kind() string { return "whatsapp.sync" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (SyncArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Minute}}
}

// SyncWorker runs SyncArgs.
type SyncWorker struct {
	river.WorkerDefaults[SyncArgs]
	S *Service
}

// Timeout bounds one pass.
func (w *SyncWorker) Timeout(*river.Job[SyncArgs]) time.Duration { return 10 * time.Minute }

// Work implements river.Worker.
func (w *SyncWorker) Work(ctx context.Context, _ *river.Job[SyncArgs]) error {
	return w.S.SyncAll(ctx)
}

// SyncAll checks every connected account and syncs its templates.
func (s *Service) SyncAll(ctx context.Context) error {
	accounts, err := s.Q.WhatsAppAccountsAll(ctx)
	if err != nil {
		return err
	}
	for _, a := range accounts {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ws, err := s.Q.GetWorkspace(ctx, a.WorkspaceID)
		if err != nil {
			continue
		}
		if err := s.check(ctx, &ws, a); err != nil {
			s.log().WarnContext(ctx, "whatsapp check", "account", a.ID, "err", err)
			continue
		}
		if _, err := s.Sync(ctx, &ws, nil, true); err != nil {
			s.log().WarnContext(ctx, "whatsapp sync", "account", a.ID, "err", err)
		}
	}
	return nil
}

// SweepArgs purges what's past its 90 days.
type SweepArgs struct{}

// Kind implements river.JobArgs.
func (SweepArgs) Kind() string { return "whatsapp.sweep" }

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
		`DELETE FROM whatsapp_raw_events WHERE received_at < $1::timestamptz - interval '90 days'`,
		`DELETE FROM whatsapp_inbound WHERE received_at < $1::timestamptz - interval '90 days'`,
	} {
		if _, err := w.S.Pool.Exec(ctx, q, now); err != nil {
			return err
		}
	}
	return nil
}

// RegisterJobs adds the WhatsApp workers and their schedules.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &NormalizeWorker{S: s})
	river.AddWorker(reg.Workers, &DispatchWorker{S: s})
	river.AddWorker(reg.Workers, &StartWorker{S: s})
	river.AddWorker(reg.Workers, &SyncWorker{S: s})
	river.AddWorker(reg.Workers, &SweepWorker{S: s})
	reg.Every(30*time.Second, func() (river.JobArgs, *river.InsertOpts) { return StartArgs{}, nil }, true)
	reg.Every(30*time.Second, func() (river.JobArgs, *river.InsertOpts) { return DispatchArgs{}, nil }, true)
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return SyncArgs{}, nil }, false)
	reg.Every(6*time.Hour, func() (river.JobArgs, *river.InsertOpts) { return SweepArgs{}, nil }, false)
}
