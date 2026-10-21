package recommendations

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/jobs"
)

// SendArgs is the minute job, and the kick after a partner request
// arrives: subscribe requests to send or retry, partner signups'
// confirmation emails, confirmations to tell partners, daily limits that
// reset, and partners that can't be reached.
type SendArgs struct{}

// Kind implements river.JobArgs.
func (SendArgs) Kind() string { return "recommendations.send" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (SendArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 1, UniqueOpts: river.UniqueOpts{ByPeriod: 5 * time.Second}}
}

// SendWorker runs SendArgs.
type SendWorker struct {
	river.WorkerDefaults[SendArgs]
	S *Service
}

// Work implements river.Worker.
func (w *SendWorker) Work(ctx context.Context, _ *river.Job[SendArgs]) error { return w.S.Tick(ctx) }

// Timeout implements river.Worker.
func (w *SendWorker) Timeout(*river.Job[SendArgs]) time.Duration { return 5 * time.Minute }

// PruneArgs is the daily job: the partner request logs keep 90 days.
type PruneArgs struct{}

// Kind implements river.JobArgs.
func (PruneArgs) Kind() string { return "recommendations.prune" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (PruneArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, MaxAttempts: 2, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Minute}}
}

// PruneWorker runs PruneArgs.
type PruneWorker struct {
	river.WorkerDefaults[PruneArgs]
	S *Service
}

// Work implements river.Worker.
func (w *PruneWorker) Work(ctx context.Context, _ *river.Job[PruneArgs]) error { return w.S.Prune(ctx) }

// RegisterJobs adds the recommendations' workers and schedules.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &SendWorker{S: s})
	river.AddWorker(reg.Workers, &PruneWorker{S: s})
	reg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return SendArgs{}, nil }, false)
	reg.Every(time.Hour, func() (river.JobArgs, *river.InsertOpts) { return PruneArgs{}, nil }, true)
}

// Tick is one pass of the minute job.
func (s *Service) Tick(ctx context.Context) error {
	var errs []error
	if _, err := s.SendDue(ctx); err != nil {
		errs = append(errs, err)
	}
	if _, err := s.SendOptIns(ctx); err != nil {
		errs = append(errs, err)
	}
	if _, err := s.SendNotices(ctx); err != nil {
		errs = append(errs, err)
	}
	now := s.now()
	if _, err := s.Pool.Exec(ctx, `UPDATE partner_links SET limit_until = NULL, updated_at = $1 WHERE limit_until IS NOT NULL AND limit_until <= $1`, now); err != nil {
		errs = append(errs, err)
	}
	if err := s.notifyUnreachable(ctx); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// notifyUnreachable tells members with grow:connect about partners whose
// requests have failed for a while.
func (s *Service) notifyUnreachable(ctx context.Context) error {
	rows, err := s.Pool.Query(ctx, `SELECT id, workspace_id FROM partner_links WHERE status = 'connected' AND failing_since IS NOT NULL
		AND failing_since <= $1 AND notified_at IS NULL`, s.now().Add(-UnreachableAfter))
	if err != nil {
		return err
	}
	type due struct{ id, ws uuid.UUID }
	var items []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.id, &d.ws); err != nil {
			rows.Close()
			return err
		}
		items = append(items, d)
	}
	rows.Close()
	for _, d := range items {
		ws, err := s.Q.GetWorkspace(ctx, d.ws)
		if err != nil {
			continue
		}
		s.notifyProblem(ctx, &ws, d.id)
	}
	return nil
}

// Prune is one pass of the daily job.
func (s *Service) Prune(ctx context.Context) error {
	now := s.now()
	var errs []error
	if err := s.Q.PartnerRequestsPrune(ctx, now.Add(-LogKept)); err != nil {
		errs = append(errs, err)
	}
	// Messages never delivered within a week don't count as shown.
	if _, err := s.Pool.Exec(ctx, `DELETE FROM recommendation_message_shown WHERE created_at < $1`, now.Add(-7*24*time.Hour)); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
