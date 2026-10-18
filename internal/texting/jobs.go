package texting

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
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

// DispatchArgs sends the texts that are due.
type DispatchArgs struct{}

// Kind implements river.JobArgs.
func (DispatchArgs) Kind() string { return "sms.dispatch" }

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

// NormalizeArgs normalizes one stored SMS webhook.
type NormalizeArgs struct {
	RawEventID uuid.UUID `json:"raw_event_id"`
}

// Kind implements river.JobArgs.
func (NormalizeArgs) Kind() string { return "sms.normalize" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (NormalizeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, MaxAttempts: 8}
}

// NormalizeWorker runs NormalizeArgs.
type NormalizeWorker struct {
	river.WorkerDefaults[NormalizeArgs]
	S *Service
}

// Work implements river.Worker.
func (w *NormalizeWorker) Work(ctx context.Context, job *river.Job[NormalizeArgs]) error {
	err := w.S.Normalize(ctx, job.Args.RawEventID)
	if errors.Is(err, errUnmatched) {
		// The send call hasn't returned the provider's id yet.
		return river.JobSnooze(10 * time.Second)
	}
	return err
}

// NextRetry implements river.Worker.
func (w *NormalizeWorker) NextRetry(job *river.Job[NormalizeArgs]) time.Time {
	return time.Now().Add(time.Duration(job.Attempt*job.Attempt) * 10 * time.Second)
}

// HealthArgs is the provider health job's SMS pass: credentials, receipts,
// registration statuses, prices and opt-out lists.
type HealthArgs struct{}

// Kind implements river.JobArgs.
func (HealthArgs) Kind() string { return "sms.health" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (HealthArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

// HealthWorker runs HealthArgs.
type HealthWorker struct {
	river.WorkerDefaults[HealthArgs]
	S *Service
}

// Timeout bounds one pass.
func (w *HealthWorker) Timeout(*river.Job[HealthArgs]) time.Duration { return 5 * time.Minute }

// Work implements river.Worker.
func (w *HealthWorker) Work(ctx context.Context, _ *river.Job[HealthArgs]) error {
	return w.S.healthTick(ctx)
}

// SweepArgs re-queues webhooks whose normalization never finished, purges
// text content past the workspace's retention and events past 90 days,
// and expires short links after 12 months.
type SweepArgs struct{}

// Kind implements river.JobArgs.
func (SweepArgs) Kind() string { return "sms.sweep" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (SweepArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 4 * time.Minute}}
}

// SweepWorker runs SweepArgs.
type SweepWorker struct {
	river.WorkerDefaults[SweepArgs]
	S *Service
}

// Work implements river.Worker.
func (w *SweepWorker) Work(ctx context.Context, _ *river.Job[SweepArgs]) error {
	return w.S.Sweep(ctx)
}

// EventRetention is how long delivery events and raw webhooks are kept.
const EventRetention = 90 * 24 * time.Hour

// Sweep is the SMS log's housekeeping.
func (s *Service) Sweep(ctx context.Context) error {
	now := s.now()
	ids, err := s.Q.SmsRawPending(ctx, now.Add(-2*time.Minute))
	if err != nil {
		return err
	}
	if c := s.jobs(); c != nil && len(ids) > 0 {
		params := make([]river.InsertManyParams, 0, len(ids))
		for _, id := range ids {
			params = append(params, river.InsertManyParams{Args: NormalizeArgs{RawEventID: id}})
		}
		if _, err := c.InsertMany(ctx, params); err != nil {
			return err
		}
	}
	// Text content: each workspace's retention (0 to 90 days).
	if _, err := s.Pool.Exec(ctx, `UPDATE channel_messages m SET body = NULL
		FROM workspaces w LEFT JOIN sms_settings st ON st.workspace_id = w.id
		WHERE m.workspace_id = w.id AND m.body IS NOT NULL AND m.status NOT IN ('queued','held','sending')
		AND m.created_at < $1::timestamptz - make_interval(days => coalesce(st.retention_days, 30))`, now); err != nil {
		return err
	}
	if _, err := s.Pool.Exec(ctx, `UPDATE sms_inbound i SET body = ''
		FROM workspaces w LEFT JOIN sms_settings st ON st.workspace_id = w.id
		WHERE i.workspace_id = w.id AND i.body <> '' AND i.received_at < $1::timestamptz - make_interval(days => coalesce(st.retention_days, 30))`, now); err != nil {
		return err
	}
	cut := now.Add(-EventRetention)
	for _, q := range []string{
		`DELETE FROM sms_raw_events WHERE received_at < $1`,
		`DELETE FROM channel_message_events WHERE occurred_at < $1`,
		`DELETE FROM sms_link_clicks WHERE occurred_at < $1`,
		`DELETE FROM sms_inbound WHERE received_at < $1`,
		`DELETE FROM channel_messages WHERE created_at < $1 AND status NOT IN ('queued','held','sending') AND NOT EXISTS (SELECT 1 FROM sms_links l WHERE l.message_id = channel_messages.id AND l.expires_at > now())`,
	} {
		if _, err := s.Pool.Exec(ctx, q, cut); err != nil {
			return err
		}
	}
	_, err = s.Pool.Exec(ctx, `DELETE FROM sms_links WHERE expires_at < $1`, now)
	return err
}

// RegisterJobs adds the SMS workers and their schedules.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &DispatchWorker{S: s})
	river.AddWorker(reg.Workers, &NormalizeWorker{S: s})
	river.AddWorker(reg.Workers, &HealthWorker{S: s})
	river.AddWorker(reg.Workers, &SweepWorker{S: s})
	reg.Every(30*time.Second, func() (river.JobArgs, *river.InsertOpts) { return DispatchArgs{}, nil }, true)
	reg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return HealthArgs{}, nil }, false)
	reg.Every(5*time.Minute, func() (river.JobArgs, *river.InsertOpts) { return SweepArgs{}, nil }, false)
}
