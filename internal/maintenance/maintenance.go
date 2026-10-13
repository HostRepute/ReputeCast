// Package maintenance runs housekeeping jobs: expired sessions and sign-in
// challenges, and audit-log retention.
package maintenance

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// CleanupArgs is the periodic housekeeping job.
type CleanupArgs struct{}

// Kind implements river.JobArgs.
func (CleanupArgs) Kind() string { return "maintenance.cleanup" }

// InsertOpts keeps one cleanup job at a time on the maintenance queue.
func (CleanupArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 10 * time.Minute}}
}

// CleanupWorker deletes expired auth state and audit entries past retention.
type CleanupWorker struct {
	river.WorkerDefaults[CleanupArgs]
	Pool *pgxpool.Pool
	Log  *slog.Logger
}

// Work implements river.Worker.
func (w *CleanupWorker) Work(ctx context.Context, _ *river.Job[CleanupArgs]) error {
	q := store.New(w.Pool)
	sessions, err := q.DeleteExpiredSessions(ctx)
	if err != nil {
		return err
	}
	challenges, err := q.DeleteExpiredChallenges(ctx)
	if err != nil {
		return err
	}
	purged, err := w.purgeAudit(ctx)
	if err != nil {
		return err
	}
	if sessions+challenges+purged > 0 {
		w.Log.InfoContext(ctx, "cleanup", "sessions", sessions, "challenges", challenges, "audit_events", purged)
	}
	return nil
}

// purgeAudit applies the instance retention to instance-level entries and
// each workspace's retention to its own entries.
func (w *CleanupWorker) purgeAudit(ctx context.Context) (int64, error) {
	var total int64
	err := db.Tx(ctx, w.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := q.EnableAuditPurge(ctx); err != nil {
			return err
		}
		st, err := q.GetInstanceSettings(ctx)
		if err != nil {
			return err
		}
		wss, err := q.ListWorkspaces(ctx)
		if err != nil {
			return err
		}
		for _, ws := range wss {
			id := ws.ID
			n, err := q.PurgeAuditEvents(ctx, store.PurgeAuditEventsParams{
				Before:      time.Now().AddDate(0, 0, -int(ws.AuditRetentionDays)),
				WorkspaceID: &id,
			})
			if err != nil {
				return err
			}
			total += n
		}
		// Instance-level entries keep the instance retention.
		n, err := q.PurgeInstanceAuditEvents(ctx, time.Now().AddDate(0, 0, -int(st.AuditRetentionDays)))
		if err != nil {
			return err
		}
		total += n
		// Entries older than the longest allowed retention go regardless.
		n, err = q.PurgeAuditEvents(ctx, store.PurgeAuditEventsParams{Before: time.Now().AddDate(0, 0, -int(max(st.AuditRetentionDays, 3650)))})
		total += n
		return err
	})
	return total, err
}

// Register adds the maintenance workers and schedules.
func Register(reg *jobs.Registry, pool *pgxpool.Pool, log *slog.Logger) {
	river.AddWorker(reg.Workers, &CleanupWorker{Pool: pool, Log: log})
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return CleanupArgs{}, nil }, true)
}
