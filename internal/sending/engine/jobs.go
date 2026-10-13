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
	if err != nil {
		return nil // deleted
	}
	// Rescheduled or unscheduled since: a newer job (or none) owns it.
	if c.Status != "scheduled" || c.ScheduledAt == nil || c.ScheduledAt.Sub(job.Args.At).Abs() > time.Second {
		return nil
	}
	by := job.Args.By
	err = w.E.StartWith(ctx, c.ID, StartOptions{By: by, Now: true})
	if errors.Is(err, sending.ErrNotReady) || errors.Is(err, ErrState) || errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

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
	return err
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
		return err
	}
	return river.JobSnooze(max(again, 10*time.Millisecond))
}

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
			return err
		}
		if n > 0 {
			continue
		}
		if next == nil {
			return nil
		}
		wait := time.Until(*next)
		if wait > 5*time.Second {
			return river.JobSnooze(wait)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(max(wait, 200*time.Millisecond)):
		}
	}
	return river.JobSnooze(10 * time.Millisecond)
}

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
		return err
	}
	return river.JobSnooze(max(again, 10*time.Millisecond))
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
	return e.Q.SendingPurgeWindows(ctx, now.Add(-time.Minute).Unix())
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
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.send_resume", TargetType: "campaign", TargetID: id.String(), TargetName: c.Name,
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
