package engine

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

// One job of each kind per campaign may wait or run at a time; a finished
// one doesn't block the next (a resumed send kicks a new run).
var activeStates = []rivertype.JobState{rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRunning,
	rivertype.JobStateScheduled, rivertype.JobStateRetryable}

func uniqueOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueSending, UniqueOpts: river.UniqueOpts{ByArgs: true, ByState: activeStates}}
}

// Every sending job is safe to run again (sends are claimed per message,
// the audience build continues from its cursor), and each kind is unique
// while it waits: a job left to River's growing back-off would hold its
// campaign, or every demo event, for minutes to hours. So a job cut short
// by shutdown snoozes to run as soon as a worker starts, and a failed one
// runs again within seconds.

// interrupted turns a job cancelled by shutdown into an immediate snooze.
func interrupted(ctx context.Context, err error) error {
	if err != nil && ctx.Err() != nil && errors.Is(err, context.Canceled) {
		return river.JobSnooze(0)
	}
	return err
}

// quickRetry is when a failed sending job runs again: 2 s more per attempt,
// at most 30 s.
func quickRetry(attempt int) time.Time {
	return time.Now().Add(min(time.Duration(max(attempt, 1))*2*time.Second, 30*time.Second))
}

// StartArgs starts a scheduled campaign at its time (with the send-time
// readiness recheck).
type StartArgs struct {
	CampaignID uuid.UUID `json:"campaign_id" river:"unique"`
	At         time.Time `json:"at" river:"unique"`
	By         uuid.UUID `json:"by"`
}

// Kind implements river.JobArgs.
func (StartArgs) Kind() string { return "sending.start" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (StartArgs) InsertOpts() river.InsertOpts { return uniqueOpts() }

// StartWorker runs StartArgs.
type StartWorker struct {
	river.WorkerDefaults[StartArgs]
	E *Engine
}

// Work implements river.Worker.
func (w *StartWorker) Work(ctx context.Context, job *river.Job[StartArgs]) error {
	c, err := w.E.Q.SendingCampaign(ctx, job.Args.CampaignID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // deleted
	}
	if err != nil {
		return interrupted(ctx, err)
	}
	// Rescheduled or unscheduled since: a newer job (or none) owns it.
	if c.Status != "scheduled" || c.ScheduledAt == nil || c.ScheduledAt.Sub(job.Args.At).Abs() > time.Second {
		return nil
	}
	// Blocked at its time: the send is held, and Review & send's minute job
	// rechecks it and starts it once a recheck passes.
	err = w.E.StartWith(ctx, c.ID, StartOptions{By: job.Args.By, Now: true, Scheduled: true})
	if errors.Is(err, sending.ErrNotReady) || errors.Is(err, ErrState) || errors.Is(err, ErrNotFound) {
		return nil
	}
	return interrupted(ctx, err)
}

// NextRetry implements river.Worker.
func (w *StartWorker) NextRetry(job *river.Job[StartArgs]) time.Time { return quickRetry(job.Attempt) }

// PrepareArgs builds a campaign's recipient list.
type PrepareArgs struct {
	CampaignID uuid.UUID `json:"campaign_id" river:"unique"`
}

// Kind implements river.JobArgs.
func (PrepareArgs) Kind() string { return "sending.prepare" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (PrepareArgs) InsertOpts() river.InsertOpts { return uniqueOpts() }

// PrepareWorker runs PrepareArgs.
type PrepareWorker struct {
	river.WorkerDefaults[PrepareArgs]
	E *Engine
}

// Timeout allows a large audience build.
func (w *PrepareWorker) Timeout(*river.Job[PrepareArgs]) time.Duration { return 20 * time.Minute }

// Work implements river.Worker. While another process holds the send (the
// demo command's fast-forward), it comes back a little later.
func (w *PrepareWorker) Work(ctx context.Context, job *river.Job[PrepareArgs]) error {
	err := w.E.Prepare(ctx, job.Args.CampaignID)
	if errors.Is(err, errBusy) {
		return river.JobSnooze(5 * time.Second)
	}
	return interrupted(ctx, err)
}

// NextRetry implements river.Worker.
func (w *PrepareWorker) NextRetry(job *river.Job[PrepareArgs]) time.Time {
	return quickRetry(job.Attempt)
}

// RunArgs runs a campaign's send loop for a while, then snoozes.
type RunArgs struct {
	CampaignID uuid.UUID `json:"campaign_id" river:"unique"`
}

// Kind implements river.JobArgs.
func (RunArgs) Kind() string { return "sending.run" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (RunArgs) InsertOpts() river.InsertOpts { return uniqueOpts() }

// RunWorker runs RunArgs.
type RunWorker struct {
	river.WorkerDefaults[RunArgs]
	E *Engine
}

// runBudget is how long one run of the loop sends before yielding.
const runBudget = 50 * time.Second

// Timeout bounds one run.
func (w *RunWorker) Timeout(*river.Job[RunArgs]) time.Duration { return 2 * time.Minute }

// Work implements river.Worker.
func (w *RunWorker) Work(ctx context.Context, job *river.Job[RunArgs]) error {
	again, done, err := w.E.Run(ctx, job.Args.CampaignID, runBudget)
	if err != nil || done {
		return interrupted(ctx, err)
	}
	return river.JobSnooze(max(again, 10*time.Millisecond))
}

// NextRetry implements river.Worker.
func (w *RunWorker) NextRetry(job *river.Job[RunArgs]) time.Time { return quickRetry(job.Attempt) }

// SimArgs applies the demo transport's events as they come due.
type SimArgs struct{}

// Kind implements river.JobArgs.
func (SimArgs) Kind() string { return "sending.sim" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (SimArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, UniqueOpts: river.UniqueOpts{ByState: activeStates}}
}

// SimWorker runs SimArgs.
type SimWorker struct {
	river.WorkerDefaults[SimArgs]
	E *Engine
}

// Timeout bounds one run.
func (w *SimWorker) Timeout(*river.Job[SimArgs]) time.Duration { return 2 * time.Minute }

// Work implements river.Worker.
func (w *SimWorker) Work(ctx context.Context, _ *river.Job[SimArgs]) error {
	start := time.Now()
	for time.Since(start) < runBudget {
		n, next, err := w.E.PumpSim(ctx, w.E.now())
		if err != nil {
			return interrupted(ctx, err)
		}
		if n > 0 {
			continue
		}
		if next == nil {
			return nil
		}
		// Never sleep long: a new send's events, due in seconds, would wait
		// behind this job (it is unique) until a late open of an earlier one.
		wait := time.Until(*next)
		if wait > 5*time.Second {
			return river.JobSnooze(min(wait, 10*time.Second))
		}
		select {
		case <-ctx.Done():
			return interrupted(ctx, ctx.Err())
		case <-time.After(max(wait, 200*time.Millisecond)):
		}
	}
	return river.JobSnooze(10 * time.Millisecond)
}

// NextRetry implements river.Worker.
func (w *SimWorker) NextRetry(job *river.Job[SimArgs]) time.Time { return quickRetry(job.Attempt) }

// OutboxArgs sends queued automation emails.
type OutboxArgs struct{}

// Kind implements river.JobArgs.
func (OutboxArgs) Kind() string { return "sending.outbox" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (OutboxArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueSending, UniqueOpts: river.UniqueOpts{ByState: activeStates}}
}

// OutboxWorker runs OutboxArgs.
type OutboxWorker struct {
	river.WorkerDefaults[OutboxArgs]
	E *Engine
}

// Timeout bounds one run.
func (w *OutboxWorker) Timeout(*river.Job[OutboxArgs]) time.Duration { return 2 * time.Minute }

// Work implements river.Worker.
func (w *OutboxWorker) Work(ctx context.Context, _ *river.Job[OutboxArgs]) error {
	again, done, err := w.E.RunOutbox(ctx, runBudget)
	if err != nil || done {
		return interrupted(ctx, err)
	}
	return river.JobSnooze(max(again, 10*time.Millisecond))
}

// NextRetry implements river.Worker.
func (w *OutboxWorker) NextRetry(job *river.Job[OutboxArgs]) time.Time {
	return quickRetry(job.Attempt)
}

// WatchArgs is the periodic safety net: it restarts send loops whose job
// was lost, starts scheduled campaigns whose start job was lost, wakes the
// outbox and the demo transport, and purges old rate windows.
type WatchArgs struct{}

// Kind implements river.JobArgs.
func (WatchArgs) Kind() string { return "sending.watch" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (WatchArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 25 * time.Second}}
}

// WatchWorker runs WatchArgs.
type WatchWorker struct {
	river.WorkerDefaults[WatchArgs]
	E *Engine
}

// Work implements river.Worker.
func (w *WatchWorker) Work(ctx context.Context, _ *river.Job[WatchArgs]) error {
	return w.E.Watch(ctx)
}

// Watch is the watchdog's pass.
func (e *Engine) Watch(ctx context.Context) error {
	now := e.now()
	active, err := e.Q.SendingActiveSends(ctx)
	if err != nil {
		return err
	}
	for _, a := range active {
		switch {
		case a.Phase == "preparing" && (a.Status == "sending" || a.Status == "paused"):
			e.insert(ctx, PrepareArgs{CampaignID: a.CampaignID}, nil)
		case a.Phase == "sending" && a.Status == "sending":
			e.kick(ctx, a.CampaignID)
		case a.Phase == "sending" && a.Status == "paused" && !a.ManualPause && !a.OpenIncident:
			// It waited only for a decision that was taken elsewhere (the
			// providers screen moved the held share, or the provider
			// recovered while set to wait): send again.
			if err := e.resumeDecided(ctx, a.CampaignID, now); err != nil {
				e.log().WarnContext(ctx, "resume a decided send", "campaign", a.CampaignID, "err", err)
			}
		}
	}
	due, err := e.Q.SendingDueScheduled(ctx, &now)
	if err != nil {
		return err
	}
	for _, id := range due {
		c, err := e.Q.SendingCampaign(ctx, id)
		if err != nil || c.ScheduledAt == nil {
			continue
		}
		e.insert(ctx, StartArgs{CampaignID: id, At: *c.ScheduledAt, By: uuidOr(c.UpdatedBy)}, nil)
	}
	if next, err := e.Q.SendingNextSimEvent(ctx); err == nil && !next.IsZero() {
		e.insert(ctx, SimArgs{}, nil)
	}
	if ids, err := e.Q.SendingOutboxProviders(ctx); err == nil && len(ids) > 0 {
		e.insert(ctx, OutboxArgs{}, nil)
	}
	e.wakeStuck(ctx)
	return e.Q.SendingPurgeWindows(ctx, now.Add(-time.Minute).Unix())
}

// wakeStuck runs now any sending job (other than a scheduled start) that
// waits more than a minute: none of them snoozes that long, so it is a
// retry back-off, and while it waits the job's uniqueness keeps the work
// from being picked up again.
func (e *Engine) wakeStuck(ctx context.Context) {
	c := e.jobs()
	if c == nil {
		return
	}
	rows, err := e.Pool.Query(ctx, `SELECT id FROM river_job
		WHERE kind IN ('sending.prepare', 'sending.run', 'sending.sim', 'sending.outbox')
		  AND state IN ('scheduled', 'retryable') AND scheduled_at > now() + interval '1 minute'`)
	if err != nil {
		return
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		if _, err := c.JobRetry(ctx, id); err != nil {
			e.log().WarnContext(ctx, "wake a waiting send job", "job", id, "err", err)
		}
	}
}

// resumeDecided sends a campaign again once nothing holds it any more.
func (e *Engine) resumeDecided(ctx context.Context, id uuid.UUID, now time.Time) error {
	resumed := false
	err := db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		c, err := q.SendingCampaignForUpdate(ctx, id)
		if err != nil {
			return err
		}
		snd, err := q.SendingGetSendForUpdate(ctx, id)
		if err != nil {
			return err
		}
		open, err := q.SendingOpenIncidents(ctx, id)
		if err != nil {
			return err
		}
		if c.Status != "paused" || snd.PausedAt != nil || len(open) > 0 {
			return nil
		}
		if err := q.SendingSetCampaignStatus(ctx, store.SendingSetCampaignStatusParams{ID: id, Status: "sending", At: now}); err != nil {
			return err
		}
		resumed = true
		if err := emit(ctx, tx, c.WorkspaceID, "campaign.resumed", campaignData(id, c.Name, "resumed_by", "System", "scope", "campaign")); err != nil {
			return err
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.resumed", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
			Reason: "The decision on the held share was taken; nothing holds the send any more.", Actor: systemActor,
			Before: map[string]any{"status": "paused"}, After: map[string]any{"status": "sending"}})
	})
	if err == nil && resumed {
		e.kick(ctx, id)
	}
	return err
}

func uuidOr(p *uuid.UUID) uuid.UUID {
	if p == nil {
		return uuid.Nil
	}
	return *p
}

// RegisterJobs adds the sending workers and the watchdog.
func RegisterJobs(reg *jobs.Registry, e *Engine) {
	river.AddWorker(reg.Workers, &StartWorker{E: e})
	river.AddWorker(reg.Workers, &PrepareWorker{E: e})
	river.AddWorker(reg.Workers, &RunWorker{E: e})
	river.AddWorker(reg.Workers, &SimWorker{E: e})
	river.AddWorker(reg.Workers, &OutboxWorker{E: e})
	river.AddWorker(reg.Workers, &WatchWorker{E: e})
	reg.Every(30*time.Second, func() (river.JobArgs, *river.InsertOpts) { return WatchArgs{}, nil }, true)
}
