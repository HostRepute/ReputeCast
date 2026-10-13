package connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers/manage"
)

// PollArgs polls the events API of providers whose events can't come by
// webhook ("Use polling for now").
type PollArgs struct{}

// Kind implements river.JobArgs.
func (PollArgs) Kind() string { return "connect.poll_events" }

// InsertOpts runs one poll a minute.
func (PollArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

// PollWorker runs PollArgs through the ingest backfill.
type PollWorker struct {
	river.WorkerDefaults[PollArgs]
	S *Service
	// Backfill records the status of a provider's recent messages from its
	// events API (ingest.Normalizer.Backfill).
	Backfill func(ctx context.Context, providerID uuid.UUID) (int, error)
}

// Timeout bounds one poll.
func (w *PollWorker) Timeout(*river.Job[PollArgs]) time.Duration { return 2 * time.Minute }

// Work implements river.Worker.
func (w *PollWorker) Work(ctx context.Context, _ *river.Job[PollArgs]) error {
	provs, err := w.S.Q.ProvListAllActive(ctx)
	if err != nil {
		return err
	}
	var errs []error
	for _, p := range provs {
		var cfg struct {
			Events string `json:"events"`
		}
		_ = json.Unmarshal(p.Config, &cfg)
		if cfg.Events != "polling" || manage.IsDemo(p) || w.Backfill == nil {
			continue
		}
		if _, err := w.Backfill(ctx, p.ID); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p.Name, err))
		}
	}
	return errors.Join(errs...)
}
