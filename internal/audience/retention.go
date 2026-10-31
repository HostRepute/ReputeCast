package audience

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/erasure"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// Data retention (review 2026-11 DP-4, DP-6, decision D3), per workspace
// from Settings › General › Data retention, every hour:
//   - the IP address and user agent of opens and clicks in campaigns and
//     automations go after event_detail_days (12 months by default); the
//     events stay, so counts, scores, A/B decisions and reports don't move;
//   - double opt-in signups never confirmed are erased after
//     unconfirmed_erase_days (30 by default), through the erasure registry;
//   - claims of the confirmation email limit older than a day go.

// retentionBatch bounds one statement; retentionRounds bounds one run per
// workspace, so a large backlog is worked off over several hours.
const (
	retentionBatch  = 5000
	retentionRounds = 20
)

// RetentionArgs runs the data retention of every workspace.
type RetentionArgs struct{}

// Kind implements river.JobArgs.
func (RetentionArgs) Kind() string { return "audience.retention" }

// InsertOpts runs on the maintenance queue, one run at a time.
func (RetentionArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 30 * time.Minute}}
}

// RetentionWorker runs RetentionArgs.
type RetentionWorker struct {
	river.WorkerDefaults[RetentionArgs]
	Svc *Service
	Log *slog.Logger
}

// Timeout gives a large backlog room.
func (w *RetentionWorker) Timeout(*river.Job[RetentionArgs]) time.Duration { return 20 * time.Minute }

// Work implements river.Worker.
func (w *RetentionWorker) Work(ctx context.Context, _ *river.Job[RetentionArgs]) error {
	return w.Svc.RunRetention(ctx, w.Log)
}

// RetentionRun is what one run did in one workspace.
type RetentionRun struct {
	DetailCleared int64
	Erased        int
}

// RunRetention is one pass over every workspace.
func (s *Service) RunRetention(ctx context.Context, log *slog.Logger) error {
	now := s.now()
	if _, err := s.Q.RetentionOptInSendsPurge(ctx, now.Add(-24*time.Hour)); err != nil {
		return err
	}
	rows, err := s.Q.RetentionWorkspaces(ctx)
	if err != nil {
		return err
	}
	for _, r := range rows {
		res, err := s.RetainWorkspace(ctx, r.ID, r.UnconfirmedEraseDays, r.EventDetailDays)
		if err != nil {
			return fmt.Errorf("retention for %s: %w", r.ID, err)
		}
		if log != nil && (res.DetailCleared > 0 || res.Erased > 0) {
			log.InfoContext(ctx, "data retention", "workspace", r.ID, "event_detail_cleared", res.DetailCleared, "unconfirmed_erased", res.Erased)
		}
	}
	return nil
}

// RetainWorkspace applies one workspace's retention settings.
func (s *Service) RetainWorkspace(ctx context.Context, wsID uuid.UUID, unconfirmedDays, detailDays int32) (RetentionRun, error) {
	var out RetentionRun
	now := s.now()
	if detailDays > 0 {
		cutoff := now.AddDate(0, 0, -int(detailDays))
		for range retentionRounds {
			n, err := s.Q.RetentionClearEventDetail(ctx, store.RetentionClearEventDetailParams{WorkspaceID: wsID, Cutoff: cutoff, Lim: retentionBatch})
			if err != nil {
				return out, err
			}
			out.DetailCleared += n
			if n < retentionBatch {
				break
			}
		}
	}
	if unconfirmedDays > 0 {
		before := now.AddDate(0, 0, -int(unconfirmedDays))
		for range retentionRounds {
			ids, err := s.Q.RetentionUnconfirmedDue(ctx, store.RetentionUnconfirmedDueParams{WorkspaceID: wsID, Before: before, Lim: retentionBatch})
			if err != nil || len(ids) == 0 {
				return out, err
			}
			err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
				erased, _, err := s.erase(ctx, tx, wsID, ids, erasure.ReasonUnconfirmed, false)
				if err != nil || len(erased) == 0 {
					return err
				}
				out.Erased += len(erased)
				return s.Audit.LogWith(ctx, store.New(tx), audit.Event{WorkspaceID: &wsID, Action: "contacts.erase", TargetType: "workspace", TargetID: wsID.String(),
					Actor:    &audit.Actor{Type: authz.ActorSystem, Name: "System"},
					Metadata: map[string]any{"count": len(erased), "reason": string(erasure.ReasonUnconfirmed), "afterDays": unconfirmedDays}})
			})
			if err != nil || len(ids) < retentionBatch {
				return out, err
			}
		}
	}
	return out, nil
}
