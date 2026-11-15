package mcpconnect

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/jobs"
)

type RetentionArgs struct{}

func (RetentionArgs) Kind() string { return "mcp_connect.retention" }
func (RetentionArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: time.Minute}}
}

type RetentionWorker struct {
	river.WorkerDefaults[RetentionArgs]
	S *Service
}

func (w *RetentionWorker) Work(ctx context.Context, _ *river.Job[RetentionArgs]) error {
	return w.S.Purge(ctx)
}

// Purge runs bounded batches. Refresh replay hashes live until the absolute
// family expiry, including after rotation. Closed grants stay readable another
// seven days for recovery; their audit is retained independently.
func (s *Service) Purge(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	now := s.now()
	for _, step := range []struct {
		remove func(context.Context, time.Time) (int64, error)
		before time.Time
	}{
		{s.Q.MCPConnectPurgeRequests, now},
		{s.Q.MCPConnectPurgeCodes, now},
		{s.Q.MCPConnectPurgeTokens, now},
		{s.Q.MCPConnectPurgeLimits, now.Add(-20 * time.Minute)},
		{s.Q.MCPConnectPurgeSources, now},
		{func(ctx context.Context, at time.Time) (int64, error) {
			return s.Q.MCPConnectPurgeProposalContent(ctx, &at)
		}, now},
		{s.Q.MCPConnectPurgeGrants, now.Add(-7 * 24 * time.Hour)},
		{func(ctx context.Context, before time.Time) (int64, error) {
			return s.Q.MCPConnectPurgeClients(ctx, &before)
		}, now},
	} {
		for range 10 {
			n, err := step.remove(ctx, step.before)
			if err != nil {
				return err
			}
			if n < 500 {
				break
			}
		}
	}
	return nil
}

func RegisterJobs(reg *jobs.Registry, s *Service) {
	river.AddWorker(reg.Workers, &RetentionWorker{S: s})
	reg.Every(time.Minute, func() (river.JobArgs, *river.InsertOpts) { return RetentionArgs{}, nil }, true)
}
