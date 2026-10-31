// Package eventqueue polls the event queues of connections whose events
// come through a queue ReputeCast reads instead of a webhook (review PR-4:
// SES → SNS → an SQS queue). Nothing has to reach ReputeCast from the
// internet, so it works behind NAT and firewalls.
//
// Each message is stored as a raw event, like a webhook delivery, and
// normalized later by the ingest job; it leaves the queue only once it is
// stored, so a crash or a restart loses nothing (the queue redelivers it).
package eventqueue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
)

// PollArgs polls every event queue once (about a minute of long polls).
type PollArgs struct{}

// Kind implements river.JobArgs.
func (PollArgs) Kind() string { return "providers.event_queue" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (PollArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueEvents, UniqueOpts: river.UniqueOpts{ByPeriod: 50 * time.Second}}
}

// pollFor is how long one job keeps polling a queue: long polls of up to
// 20 s back to back, so events arrive within about 20 s.
const pollFor = 50 * time.Second

// maxPerPoll bounds the messages one job stores per queue.
const maxPerPoll = 2000

// Worker polls the queues.
type Worker struct {
	river.WorkerDefaults[PollArgs]
	Pool    *pgxpool.Pool
	Q       *store.Queries
	Factory *manage.Factory
	Jobs    func() *jobs.Client
	Log     *slog.Logger
	Now     func() time.Time
	// OnPoll is told how each poll of a queue went, so a queue that can't
	// be read reaches the provider's health.
	OnPoll func(ctx context.Context, p store.Provider, err error)
	// For overrides how long a job polls each queue (tests).
	For time.Duration
}

func (w *Worker) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

func (w *Worker) log() *slog.Logger {
	if w.Log != nil {
		return w.Log
	}
	return slog.Default()
}

// Timeout bounds one job.
func (w *Worker) Timeout(*river.Job[PollArgs]) time.Duration { return 2 * time.Minute }

// Work implements river.Worker.
func (w *Worker) Work(ctx context.Context, _ *river.Job[PollArgs]) error {
	provs, err := w.Q.ProvListAllActive(ctx)
	if err != nil {
		return err
	}
	// The queues are long-polled side by side, a few at a time.
	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
	)
	sem := make(chan struct{}, 8)
	for _, p := range provs {
		if manage.IsDemo(p) || !queued(p) {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(p store.Provider) {
			defer wg.Done()
			defer func() { <-sem }()
			if _, err := w.Poll(ctx, p); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", p.Name, err))
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()
	if len(errs) > 0 {
		w.log().WarnContext(ctx, "poll event queues", "err", errors.Join(errs...))
	}
	// A queue that can't be read is the provider's health, not a job
	// failure to retry: the next job polls again within a minute.
	return nil
}

// queued reports whether a connection's events come through a queue.
func queued(p store.Provider) bool {
	var c struct {
		Events string `json:"events"`
	}
	_ = json.Unmarshal(p.Config, &c)
	return c.Events == providers.EventsQueue
}

// Poll reads a provider's queue for up to a minute: each message is stored
// as a raw event and queued for normalization, then removed from the queue.
func (w *Worker) Poll(ctx context.Context, p store.Provider) (n int, err error) {
	adapter, err := w.Factory.Open(ctx, p)
	if err != nil {
		return 0, err
	}
	eq, ok := adapter.(providers.EventQueue)
	if !ok || !eq.QueueEnabled() {
		return 0, nil
	}
	if w.OnPoll != nil {
		defer func() { w.OnPoll(ctx, p, err) }()
	}
	span := w.For
	if span <= 0 {
		span = pollFor
	}
	deadline := time.Now().Add(span)
	for n < maxPerPoll {
		wait := time.Until(deadline)
		if wait <= 0 {
			break
		}
		evs, err := eq.ReceiveEvents(ctx, 10, wait)
		if err != nil {
			return n, err
		}
		if len(evs) == 0 {
			if wait < time.Second {
				break
			}
			continue
		}
		if err := w.store(ctx, p, evs); err != nil {
			return n, err
		}
		handles := make([]string, 0, len(evs))
		for _, ev := range evs {
			handles = append(handles, ev.Handle)
		}
		if err := eq.AckEvents(ctx, handles); err != nil {
			// Stored already: the queue delivers them again, and the
			// duplicates are normalized like a webhook retried by SNS.
			return n + len(evs), err
		}
		n += len(evs)
	}
	return n, nil
}

// store keeps queued events as raw events (source "queue") and queues
// their normalization, in one transaction.
func (w *Worker) store(ctx context.Context, p store.Provider, evs []providers.QueuedEvent) error {
	hdr, _ := json.Marshal(map[string]string{"X-ReputeCast-Source": "queue"})
	return db.Tx(ctx, w.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		for _, ev := range evs {
			id := uuid.Must(uuid.NewV7())
			ct := ev.ContentType
			if ct == "" {
				ct = "application/json"
			}
			// The queue only takes messages from the connection's own topic
			// and is read with the connection's own credentials.
			if err := q.IngestInsertRaw(ctx, store.IngestInsertRawParams{ID: id, WorkspaceID: p.WorkspaceID, ProviderID: p.ID, ReceivedAt: w.now(),
				ContentType: ct, Body: ev.Body, Headers: hdr, SignatureOk: true, Source: "queue"}); err != nil {
				return err
			}
			if c := w.jobs(); c != nil {
				if _, err := c.InsertTx(ctx, tx, ingest.NormalizeArgs{RawEventID: id}, nil); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (w *Worker) jobs() *jobs.Client {
	if w.Jobs == nil {
		return nil
	}
	return w.Jobs()
}
