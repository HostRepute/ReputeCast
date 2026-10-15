package transactional

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// Retention of transactional mail: the rendered content after the
// workspace's content retention (0 to 90 days, 30 by default), the message
// log's metadata after 90 days, idempotency keys after 24 hours, and
// refused calls (the first-send guide's and the publish check's) after 30
// days. Sensitive values are never stored, so nothing else holds them.

// RejectedCallRetention is how long refused calls are kept.
const RejectedCallRetention = 30 * 24 * time.Hour

const retentionBatch = 5000

// RetentionResult is what one retention pass removed.
type RetentionResult struct {
	Content, Messages, Idempotency, RejectedCalls int64
}

// Retain removes what is past its retention.
func (s *Service) Retain(ctx context.Context) (RetentionResult, error) {
	now := s.now()
	var out RetentionResult
	wss, err := s.Q.TxnRetentionWorkspaces(ctx)
	if err != nil {
		return out, err
	}
	for _, w := range wss {
		cutoff := now.Add(-time.Duration(w.ContentRetentionDays) * 24 * time.Hour)
		for {
			ids, err := s.Q.TxnRetentionContentDue(ctx, store.TxnRetentionContentDueParams{WorkspaceID: w.WorkspaceID, Now: now, Cutoff: cutoff, Lim: retentionBatch})
			if err != nil {
				return out, err
			}
			if len(ids) == 0 {
				break
			}
			if err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
				q := store.New(tx)
				if err := q.TxnRetentionContentRemove(ctx, ids); err != nil {
					return err
				}
				return q.TxnRetentionContentRemoved(ctx, store.TxnRetentionContentRemovedParams{Now: now, Ids: ids})
			}); err != nil {
				return out, err
			}
			out.Content += int64(len(ids))
			if len(ids) < retentionBatch {
				break
			}
		}
	}
	for {
		n, err := s.Q.TxnRetentionMessages(ctx, store.TxnRetentionMessagesParams{Before: now.Add(-MetadataRetentionDays * 24 * time.Hour), Lim: retentionBatch})
		if err != nil {
			return out, err
		}
		out.Messages += n
		if n < retentionBatch {
			break
		}
	}
	if out.Idempotency, err = s.Q.TxnIdemPurge(ctx, now.Add(-IdempotencyTTL)); err != nil {
		return out, err
	}
	if out.RejectedCalls, err = s.Q.TxnRetentionRejectedCalls(ctx, now.Add(-RejectedCallRetention)); err != nil {
		return out, err
	}
	return out, nil
}

// RetentionArgs runs a retention pass.
type RetentionArgs struct{}

// Kind implements river.JobArgs.
func (RetentionArgs) Kind() string { return "transactional.retention" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (RetentionArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: 30 * time.Minute}}
}

// RetentionWorker runs RetentionArgs.
type RetentionWorker struct {
	river.WorkerDefaults[RetentionArgs]
	S *Service
}

// Timeout bounds one pass.
func (w *RetentionWorker) Timeout(*river.Job[RetentionArgs]) time.Duration { return 10 * time.Minute }

// Work implements river.Worker.
func (w *RetentionWorker) Work(ctx context.Context, _ *river.Job[RetentionArgs]) error {
	res, err := w.S.Retain(ctx)
	if err != nil {
		return err
	}
	if res.Content+res.Messages > 0 && w.S.Log != nil {
		w.S.Log.InfoContext(ctx, "transactional retention", "content", res.Content, "messages", res.Messages, "idempotency", res.Idempotency, "rejected_calls", res.RejectedCalls)
	}
	return nil
}

// RegisterJobs adds the transactional domain's jobs.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &RetentionWorker{S: s})
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return RetentionArgs{}, nil }, true)
	river.AddWorker(reg.Workers, &AlertsWorker{S: s})
	reg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return AlertsArgs{}, nil }, false)
}
