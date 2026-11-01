package audience

import (
	"context"
	"errors"
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
	// Exports past their seven days and fields deleted 30 days ago go too.
	return errors.Join(w.Svc.RunRetention(ctx, w.Log), w.Svc.SweepJobFiles(ctx), w.Svc.PurgeDeletedFields(ctx))
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
	// Whole monthly partitions past a cutoff first, in one pass each; the
	// per-workspace batches below then find little left (OP-8).
	if n, err := s.ClearDetailByPartition(ctx, now); err != nil {
		return fmt.Errorf("detail by partition: %w", err)
	} else if log != nil && n > 0 {
		log.InfoContext(ctx, "data retention", "event_detail_cleared_by_partition", n)
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

// ClearDetailByPartition clears open and click detail (IP address, user
// agent) a monthly partition of delivery_events at a time, when it is past
// the cutoff of the workspaces keeping detail a given number of days: one
// pass over the partition, recorded so it never runs again (review
// 2026-11 OP-8, DP-6). A day's margin covers events stored a little after
// they happened. Without partitions it does nothing.
func (s *Service) ClearDetailByPartition(ctx context.Context, now time.Time) (int64, error) {
	ok, err := db.EventsPartitioned(ctx, s.Pool)
	if err != nil || !ok {
		return 0, err
	}
	parts, err := db.EventPartitions(ctx, s.Pool)
	if err != nil {
		return 0, err
	}
	var days []int32
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT event_detail_days FROM workspaces WHERE deleted_at IS NULL AND event_detail_days > 0 ORDER BY 1`)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var d int32
		if err := rows.Scan(&d); err != nil {
			rows.Close()
			return 0, err
		}
		days = append(days, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	var total int64
	for _, p := range parts {
		for _, d := range days {
			if p.Till.After(now.AddDate(0, 0, -int(d)-1)) {
				continue
			}
			var done bool
			if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM delivery_event_partitions WHERE partition = $1 AND detail_days = $2)`, p.Name, d).Scan(&done); err != nil {
				return total, err
			}
			if done {
				continue
			}
			err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
				// The partition's name comes from the catalog and matches
				// delivery_events_YYYY_MM.
				tag, err := tx.Exec(ctx, `
					UPDATE `+p.Name+` e SET ip = NULL, user_agent = ''
					FROM workspaces w
					WHERE w.id = e.workspace_id AND w.event_detail_days = $1
					  AND e.type IN ('open', 'click') AND (e.ip IS NOT NULL OR e.user_agent <> '')
					  AND (e.message_id IS NULL OR NOT EXISTS (
					        SELECT 1 FROM messages m WHERE m.id = e.message_id AND m.stream NOT IN ('broadcast', 'automation')))`, d)
				if err != nil {
					return err
				}
				total += tag.RowsAffected()
				_, err = tx.Exec(ctx, `INSERT INTO delivery_event_partitions (partition, detail_days, cleared_at) VALUES ($1, $2, $3)
					ON CONFLICT DO NOTHING`, p.Name, d, now)
				return err
			})
			if err != nil {
				return total, err
			}
		}
	}
	return total, nil
}
