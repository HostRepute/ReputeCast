package pages

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/jobs"
)

// DueArgs publishes the scheduled pages whose time came.
type DueArgs struct{}

// Kind implements river.JobArgs.
func (DueArgs) Kind() string { return "pages.due" }

// InsertOpts runs one at a time.
func (DueArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueSystem, UniqueOpts: river.UniqueOpts{ByPeriod: time.Minute}}
}

type dueWorker struct {
	river.WorkerDefaults[DueArgs]
	S *Service
}

func (w *dueWorker) Work(ctx context.Context, _ *river.Job[DueArgs]) error {
	_, err := w.S.PublishDue(ctx)
	return err
}

// RegisterJobs adds the scheduled publisher.
func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &dueWorker{S: s})
	reg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return DueArgs{}, nil }, true)
}
