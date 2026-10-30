package approvals

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/notify"
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

// expire takes a held send back to Draft (Approved when its approval still
// stands), keeping its schedule and A/B test for the next attempt, and
// tells who scheduled it and the approvers.
func (s *Service) expire(ctx context.Context, wsID, campaignID uuid.UUID) error {
	w, err := s.Q.GetWorkspace(ctx, wsID)
	if err != nil {
		return err
	}
	ws := &w
	c, err := s.Readiness.Campaign(ctx, ws, campaignID)
	if err != nil {
		return err
	}
	if err := s.starter().Cancel(ctx, c.ID, uuid.Nil); err != nil && !errors.Is(err, sending.ErrUnavailable) {
		return err
	}
	status := "draft"
	if open, err := s.open(ctx, c.ID); err == nil && open != nil && open.Status == "approved" {
		status = "approved"
	}
	ms, err := s.members(ctx, s.Q, ws.ID)
	if err != nil {
		return err
	}
	now := s.now()
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.ApprovalUnschedule(ctx, store.ApprovalUnscheduleParams{Status: status, Now: now, WorkspaceID: ws.ID, ID: c.ID}); err != nil {
			return err
		}
		if err := q.ApprovalResolveHold(ctx, store.ApprovalResolveHoldParams{ResolvedAt: &now, Resolution: "expired", CampaignID: c.ID}); err != nil {
			return err
		}
		if err := s.Readiness.Forget(ctx, q, c.ID); err != nil {
			return err
		}
		if err := s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "campaign.send_expired", TargetType: "campaign", TargetID: c.ID.String(), TargetName: c.Name,
			Actor: &audit.Actor{Type: authz.ActorSystem, Name: "System"}, Before: map[string]any{"status": "scheduled", "scheduled_at": c.ScheduledAt},
			After: map[string]any{"status": status}, Metadata: map[string]any{"expired_after": readiness.HeldExpiry.String()}}); err != nil {
			return err
		}
		targets := []uuid.UUID{}
		if c.UpdatedBy != nil {
			targets = append(targets, *c.UpdatedBy)
		}
		for _, a := range approversOf(ms) {
			targets = append(targets, a.ID)
		}
		n := notify.Notification{WorkspaceID: &ws.ID, Kind: notify.KindSendBlocked, Severity: "warn", Title: "Held send expired",
			Link: "/w/" + ws.Slug + "/campaigns/" + c.ID.String() + "/review", Data: map[string]any{"campaign_id": c.ID.String()}, TitleText: mNotifyExpiredTitle.Keep(nil)}
		seen := map[uuid.UUID]bool{}
		for _, id := range targets {
			if seen[id] {
				continue
			}
			seen[id] = true
			args := i18n.Args{"name": c.Name, "hours": int(readiness.HeldExpiry.Hours())}
			n.Summary, n.Body = english(mNotifyExpiredSummary, args), english(mNotifyExpiredBody, args)
			n.SummaryText, n.BodyText = mNotifyExpiredSummary.Keep(args), mNotifyExpiredBody.Keep(args)
			if err := notify.ToUser(ctx, tx, id, n); err != nil {
				return err
			}
		}
		return nil
	})
}

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
		// Blocks, and what the person didn't confirm (SR-3), hold the send.
		check, err := s.sendTimeCheck(ctx, &ws, d.ID, run)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if check.Held() {
			reason := check.Holds[0].English()
			if check.Holds[0].Kind == readiness.HoldBlocked {
				reason = firstBlock(run)
			}
			if c, err := s.Readiness.Campaign(ctx, &ws, d.ID); err == nil {
				errs = append(errs, s.hold(ctx, &ws, c, &run, reason, &check))
			}
		}
	}
	// Held sends: recheck every 15 minutes; resume when the recheck passes.
	held, err := s.Q.ApprovalHeldCampaigns(ctx)
	if err != nil {
		return err
	}
	for _, h := range held {
		// A held send waits for a person until HeldExpiry after its time;
		// then it goes back to Draft instead of starting days late (SR-3).
		if h.ScheduledAt != nil && now.After(h.ScheduledAt.Add(readiness.HeldExpiry)) {
			if err := s.expire(ctx, h.WorkspaceID, h.ID); err != nil {
				errs = append(errs, err)
			}
			continue
		}
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
			// The engine kept the canary choice on the campaign (C1).
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
