package approvals

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/readiness"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
)

// RecheckLead is how long before a scheduled send Send Readiness runs again.
const RecheckLead = 15 * time.Minute

// HeldRetry is how often a held send is rechecked.
const HeldRetry = 15 * time.Minute

// TickArgs is the minute job of Review & send: checks run again 15 minutes
// before scheduled sends, held sends are rechecked and resumed when a
// recheck passes, and open approvals are re-checked for audience drift.
type TickArgs struct{}

// Kind implements river.JobArgs.
func (TickArgs) Kind() string { return "approvals.tick" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (TickArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 1, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

type TickWorker struct {
	river.WorkerDefaults[TickArgs]
	S *Service
}

// Work implements river.Worker.
func (w *TickWorker) Work(ctx context.Context, _ *river.Job[TickArgs]) error { return w.S.Tick(ctx) }

// Timeout implements river.Worker.
func (w *TickWorker) Timeout(*river.Job[TickArgs]) time.Duration { return 5 * time.Minute }

// Tick runs the periodic work once.
func (s *Service) Tick(ctx context.Context) error {
	now := s.now()
	var errs []error
	// Checks run again 15 minutes before a scheduled send.
	due, err := s.Q.ReadinessScheduledDue(ctx, store.ReadinessScheduledDueParams{Until: now.Add(RecheckLead), Lead: RecheckLead})
	if err != nil {
		return err
	}
	for _, d := range due {
		ws, err := s.Q.GetWorkspace(ctx, d.WorkspaceID)
		if err != nil {
			continue
		}
		run, err := s.Readiness.Run(ctx, &ws, d.ID, readiness.RunOptions{Trigger: readiness.TriggerSchedule})
		if errors.Is(err, readiness.ErrRunning) {
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if run.Verdict == readiness.VerdictBlocked {
			c, err := s.Readiness.Campaign(ctx, &ws, d.ID)
			if err == nil {
				errs = append(errs, s.hold(ctx, &ws, c, &run, firstBlock(run)))
			}
		}
	}
	// Held sends: recheck every 15 minutes; resume when the recheck passes.
	held, err := s.Q.ApprovalHeldCampaigns(ctx)
	if err != nil {
		return err
	}
	for _, h := range held {
		latest, err := s.Readiness.Latest(ctx, h.ID)
		if err != nil || (latest != nil && now.Sub(latest.RanAt) < HeldRetry) {
			continue
		}
		err = s.CheckSendTime(ctx, h.WorkspaceID, h.ID)
		if errors.Is(err, sending.ErrNotReady) {
			continue
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if h.ScheduledAt != nil && !h.ScheduledAt.After(now) {
			// The recheck passed after the scheduled time: start the send.
			if err := s.starter().Start(ctx, h.ID, uuid.Nil); err != nil && !errors.Is(err, sending.ErrUnavailable) {
				errs = append(errs, err)
			}
		}
	}
	// Open approvals: the audience can move without anyone editing the campaign.
	if now.Minute()%5 == 0 {
		open, err := s.Q.ApprovalOpenRequests(ctx)
		if err != nil {
			return err
		}
		for _, r := range open {
			if r.CampaignID == nil {
				continue
			}
			ws, err := s.Q.GetWorkspace(ctx, r.WorkspaceID)
			if err != nil {
				continue
			}
			if _, err := s.Revalidate(ctx, &ws, *r.CampaignID); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}
