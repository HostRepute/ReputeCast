package assistant

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// Retention (assistant brief §3.7): stream events go 24 hours after their
// turn ends; a thread's content 30 days after its last activity, leaving a
// tombstone with its usage; tombstones 90 days after that.
const (
	eventsKept     = 24 * time.Hour
	tombstonesKept = 90 * 24 * time.Hour
)

// RetentionArgs is the hourly retention job.
type RetentionArgs struct{}

// Kind implements river.JobArgs.
func (RetentionArgs) Kind() string { return "ai.assistant_retention" }

// InsertOpts runs it once at a time.
func (RetentionArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour}}
}

// RetentionWorker runs the retention.
type RetentionWorker struct {
	river.WorkerDefaults[RetentionArgs]
	S *Service
}

// Work implements river.Worker.
func (w *RetentionWorker) Work(ctx context.Context, _ *river.Job[RetentionArgs]) error {
	r, err := w.S.Purge(ctx)
	if err == nil && (r.Events+r.Expired+r.Tombstones) > 0 {
		w.S.log().InfoContext(ctx, "assistant: retention", "events", r.Events, "expired", r.Expired, "tombstones", r.Tombstones)
	}
	return err
}

// PurgeResult is what a retention run removed.
type PurgeResult struct {
	Events, Expired, Tombstones int64
}

// Purge applies the retention, in batches. An expiring thread is taken in
// the threads' lock order and skipped while someone holds it; its active
// turns end first (their version moves on), so a worker that checks in
// afterwards restores nothing.
func (s *Service) Purge(ctx context.Context) (PurgeResult, error) {
	var r PurgeResult
	now := s.now()
	for {
		before := now.Add(-eventsKept)
		n, err := s.Q.AIAssistantEventsPurge(ctx, &before)
		r.Events += n
		if err != nil || n < 5000 {
			if err != nil {
				return r, err
			}
			break
		}
	}
	for {
		n := 0
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			rows, err := q.AIAssistantExpiring(ctx, now)
			if err != nil {
				return err
			}
			n = len(rows)
			for _, t := range rows {
				if _, err := q.AIAssistantThreadTurnsEnd(ctx, store.AIAssistantThreadTurnsEndParams{ThreadID: t.ID, Now: &now}); err != nil {
					return err
				}
				for _, purge := range []func(context.Context, uuid.UUID) error{q.AIAssistantMessagesPurge, q.AIAssistantContextsPurge, q.AIAssistantThreadEventsPurge, q.AIAssistantThreadSlotPurge,
					q.AIAssistantConfirmationsPurge} {
					if err := purge(ctx, t.ID); err != nil {
						return err
					}
				}
				if err := q.AIAssistantThreadExpire(ctx, store.AIAssistantThreadExpireParams{ID: t.ID, Now: now}); err != nil {
					return err
				}
			}
			return nil
		})
		r.Expired += int64(n)
		if err != nil {
			return r, err
		}
		if n < 100 {
			break
		}
	}
	for {
		n, err := s.Q.AIAssistantTombstonesPurge(ctx, now.Add(-tombstonesKept))
		r.Tombstones += n
		if err != nil {
			return r, err
		}
		if n < 1000 {
			break
		}
	}
	return r, nil
}
