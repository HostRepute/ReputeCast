// Package jobs wires River, the Postgres-backed job queue. Domains add their
// workers and periodic jobs to a Registry; the worker role runs them, other
// roles get an insert-only client so they can still enqueue work.
package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
)

// Queue names. Sending has its own queue so a large broadcast never starves
// webhook normalization or system mail.
const (
	QueueDefault     = river.QueueDefault
	QueueSending     = "sending"
	QueueEvents      = "events"
	QueueSystem      = "system"
	QueueMaintenance = "maintenance"
)

// Client is River's client bound to pgx.
type Client = river.Client[pgx.Tx]

// Registry collects workers and periodic jobs from every domain.
type Registry struct {
	Workers  *river.Workers
	Periodic []*river.PeriodicJob
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry { return &Registry{Workers: river.NewWorkers()} }

// Every schedules a job built by fn at a fixed interval.
func (r *Registry) Every(d time.Duration, fn func() (river.JobArgs, *river.InsertOpts), runOnStart bool) {
	r.Periodic = append(r.Periodic, river.NewPeriodicJob(river.PeriodicInterval(d), fn, &river.PeriodicJobOpts{RunOnStart: runOnStart}))
}

// Options configures the client.
type Options struct {
	Run         bool // run workers (worker role); false = insert-only
	Concurrency int
	Logger      *slog.Logger
}

// NewClient builds the River client.
func NewClient(pool *pgxpool.Pool, reg *Registry, o Options) (*Client, error) {
	cfg := &river.Config{
		Logger:       o.Logger,
		Workers:      reg.Workers,
		JobTimeout:   10 * time.Minute,
		MaxAttempts:  12,
		ErrorHandler: &errorHandler{log: o.Logger},
	}
	if o.Run {
		c := max(o.Concurrency, 4)
		cfg.Queues = map[string]river.QueueConfig{
			QueueDefault:     {MaxWorkers: max(c/4, 2)},
			QueueSending:     {MaxWorkers: c},
			QueueEvents:      {MaxWorkers: max(c/2, 2)},
			QueueSystem:      {MaxWorkers: max(c/5, 2)},
			QueueMaintenance: {MaxWorkers: 2},
		}
		cfg.PeriodicJobs = reg.Periodic
	}
	return river.NewClient(riverpgxv5.New(pool), cfg)
}

type errorHandler struct{ log *slog.Logger }

func (h *errorHandler) HandleError(ctx context.Context, job *rivertype.JobRow, err error) *river.ErrorHandlerResult {
	h.log.WarnContext(ctx, "job failed", "kind", job.Kind, "id", job.ID, "attempt", job.Attempt, "err", err)
	return nil
}

func (h *errorHandler) HandlePanic(ctx context.Context, job *rivertype.JobRow, panicVal any, trace string) *river.ErrorHandlerResult {
	h.log.ErrorContext(ctx, "job panicked", "kind", job.Kind, "id", job.ID, "panic", panicVal, "trace", trace)
	return nil
}
