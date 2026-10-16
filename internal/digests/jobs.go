package digests

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/jobs"
)

// The digests' jobs: the minute job moves every digest along; a read job
// reads one digest's feed; a prepare job builds one issue. Read and prepare
// are unique while they wait, so a busy minute never piles them up.

var activeStates = []rivertype.JobState{rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRunning,
	rivertype.JobStateScheduled, rivertype.JobStateRetryable}

// TickArgs is the minute job.
type TickArgs struct{}

// Kind implements river.JobArgs.
func (TickArgs) Kind() string { return "digests.tick" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (TickArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 1, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

// TickWorker runs the minute job.
type TickWorker struct {
	river.WorkerDefaults[TickArgs]
	S *Service
}

// Work implements river.Worker.
func (w *TickWorker) Work(ctx context.Context, _ *river.Job[TickArgs]) error { return w.S.Tick(ctx) }

// Timeout implements river.Worker.
func (w *TickWorker) Timeout(*river.Job[TickArgs]) time.Duration { return 5 * time.Minute }

// ReadArgs reads a digest's feed.
type ReadArgs struct {
	DigestID uuid.UUID `json:"digest_id" river:"unique"`
	Via      string    `json:"via"`
}

// Kind implements river.JobArgs.
func (ReadArgs) Kind() string { return "digests.read" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ReadArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByArgs: true, ByState: activeStates}}
}

// ReadWorker reads a feed.
type ReadWorker struct {
	river.WorkerDefaults[ReadArgs]
	S *Service
}

// Work implements river.Worker.
func (w *ReadWorker) Work(ctx context.Context, job *river.Job[ReadArgs]) error {
	d, err := w.S.Q.DigestByID(ctx, job.Args.DigestID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // deleted
	}
	if err != nil {
		return err
	}
	ws, err := w.S.workspace(ctx, d.WorkspaceID)
	if err != nil {
		return err
	}
	_, err = w.S.read(ctx, ws, d, firstNonEmpty(job.Args.Via, ViaSchedule))
	return err
}

// Timeout implements river.Worker: a read takes 15 seconds at most, plus its images.
func (w *ReadWorker) Timeout(*river.Job[ReadArgs]) time.Duration { return 3 * time.Minute }

// PrepareArgs prepares an issue.
type PrepareArgs struct {
	IssueID uuid.UUID `json:"issue_id" river:"unique"`
}

// Kind implements river.JobArgs.
func (PrepareArgs) Kind() string { return "digests.prepare" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (PrepareArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByArgs: true, ByState: activeStates}}
}

// PrepareWorker prepares an issue.
type PrepareWorker struct {
	river.WorkerDefaults[PrepareArgs]
	S *Service
}

// Work implements river.Worker.
func (w *PrepareWorker) Work(ctx context.Context, job *river.Job[PrepareArgs]) error {
	return w.S.Prepare(ctx, job.Args.IssueID)
}

// Timeout implements river.Worker.
func (w *PrepareWorker) Timeout(*river.Job[PrepareArgs]) time.Duration { return 5 * time.Minute }

// RegisterJobs adds the digests' workers and the minute job.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &TickWorker{S: s})
	river.AddWorker(reg.Workers, &ReadWorker{S: s})
	river.AddWorker(reg.Workers, &PrepareWorker{S: s})
	reg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return TickArgs{}, nil }, false)
}
