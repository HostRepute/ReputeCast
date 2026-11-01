// Package opsmetrics holds the operational metrics every role exports on
// /metrics (review 2026-11 OP-1): background jobs (depth, lag, failures by
// kind), messages handed to sending providers by adapter and result,
// webhook ingest lag, outgoing webhook deliveries and the database pool.
//
// Counters are package variables, counted where the work happens; the
// queue gauges are read from the database at scrape time, at most every
// 15 seconds, so a busy Prometheus never loads the database.
package opsmetrics

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

// Counters, counted where the work happens.
var (
	// ProviderMessages counts the messages handed to sending providers:
	// accepted, rejected (permanently), deferred (a temporary refusal,
	// retried) or failed (the whole call failed).
	ProviderMessages = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "reputecast_provider_messages_total",
		Help: "Messages handed to sending providers, by adapter and result (accepted, rejected, deferred, failed).",
	}, []string{"provider", "result"})
	// ProviderCalls counts send calls to providers, by adapter and result
	// (ok or error).
	ProviderCalls = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "reputecast_provider_calls_total",
		Help: "Send calls to sending providers, by adapter and result (ok, error).",
	}, []string{"provider", "result"})
	// JobFailures counts failed job attempts by kind: retried, discarded
	// (out of attempts) or panicked.
	JobFailures = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "reputecast_job_failures_total",
		Help: "Failed background job attempts, by kind and outcome (retry, discarded, panic).",
	}, []string{"kind", "outcome"})
	// WebhookDeliveries counts attempts to deliver outgoing webhooks.
	WebhookDeliveries = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "reputecast_webhook_deliveries_total",
		Help: "Outgoing webhook delivery attempts, by result (delivered, retrying, dead).",
	}, []string{"result"})
	// IngestRequests counts provider webhook requests received.
	IngestRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "reputecast_ingest_requests_total",
		Help: "Provider webhook requests received, by channel (email, sms, whatsapp) and result (stored, rejected).",
	}, []string{"channel", "result"})
)

// counters are registered with every registry (tests build several apps).
var counters = []prometheus.Collector{ProviderMessages, ProviderCalls, JobFailures, WebhookDeliveries, IngestRequests}

// Register adds the counters and the database collectors to reg. queues
// are the job queues whose lag always shows, 0 when nothing waits.
func Register(reg prometheus.Registerer, pool *pgxpool.Pool, queues ...string) {
	for _, c := range counters {
		if err := reg.Register(c); err != nil {
			var already prometheus.AlreadyRegisteredError
			if !errors.As(err, &already) {
				panic(err)
			}
		}
	}
	if pool != nil {
		reg.MustRegister(newQueueCollector(pool, queues), poolCollector{pool})
	}
}

// Message results.
const (
	Accepted = "accepted"
	Rejected = "rejected"
	Deferred = "deferred"
	Failed   = "failed"
)

// queueCollector reads job, ingest and outgoing webhook queues.
type queueCollector struct {
	pool   *pgxpool.Pool
	ttl    time.Duration
	queues []string

	jobs, jobLag, ingestPending, ingestLag, hooksPending, hooksLag, scrapeErrors *prometheus.Desc

	mu      sync.Mutex
	at      time.Time
	metrics []prometheus.Metric
}

func newQueueCollector(pool *pgxpool.Pool, queues []string) *queueCollector {
	return &queueCollector{
		pool:   pool,
		ttl:    15 * time.Second,
		queues: queues,
		jobs: prometheus.NewDesc("reputecast_jobs", "Background jobs by queue, kind and state (available, scheduled, retryable, running, discarded).",
			[]string{"queue", "kind", "state"}, nil),
		jobLag: prometheus.NewDesc("reputecast_job_queue_lag_seconds", "How long the oldest job that is due has waited, by queue (0 when none waits).",
			[]string{"queue"}, nil),
		ingestPending: prometheus.NewDesc("reputecast_ingest_pending", "Provider webhook events stored and not normalized yet, by channel.",
			[]string{"channel"}, nil),
		ingestLag: prometheus.NewDesc("reputecast_ingest_lag_seconds", "Age of the oldest provider webhook event not normalized yet, by channel.",
			[]string{"channel"}, nil),
		hooksPending: prometheus.NewDesc("reputecast_webhook_deliveries_pending", "Outgoing webhook deliveries waiting for an attempt or a retry.",
			nil, nil),
		hooksLag: prometheus.NewDesc("reputecast_webhook_deliveries_lag_seconds", "How long the oldest outgoing webhook delivery that is due has waited.",
			nil, nil),
		scrapeErrors: prometheus.NewDesc("reputecast_metrics_scrape_errors", "Queue readings that failed at the last scrape (the database was slow or down).",
			nil, nil),
	}
}

func (c *queueCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range []*prometheus.Desc{c.jobs, c.jobLag, c.ingestPending, c.ingestLag, c.hooksPending, c.hooksLag, c.scrapeErrors} {
		ch <- d
	}
}

func (c *queueCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.metrics == nil || time.Since(c.at) >= c.ttl {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		c.metrics = c.read(ctx)
		cancel()
		c.at = time.Now()
	}
	for _, m := range c.metrics {
		ch <- m
	}
}

// read takes one reading of every queue. A reading that fails is left out
// and counted in reputecast_metrics_scrape_errors.
func (c *queueCollector) read(ctx context.Context) []prometheus.Metric {
	var out []prometheus.Metric
	failed := 0
	gauge := func(d *prometheus.Desc, v float64, labels ...string) {
		out = append(out, prometheus.MustNewConstMetric(d, prometheus.GaugeValue, v, labels...))
	}

	// Jobs. Completed and cancelled jobs are left out: River keeps them a
	// day, and nothing waits on them.
	rows, err := c.pool.Query(ctx, `
		SELECT queue, kind, state::text, count(*)::float8
		FROM river_job
		WHERE state IN ('available', 'scheduled', 'retryable', 'running', 'discarded')
		GROUP BY 1, 2, 3`)
	if err == nil {
		for rows.Next() {
			var queue, kind, state string
			var n float64
			if rows.Scan(&queue, &kind, &state, &n) == nil {
				gauge(c.jobs, n, queue, kind, state)
			}
		}
		rows.Close()
		err = rows.Err()
	}
	if err != nil {
		failed++
	}
	lags := map[string]float64{}
	for _, q := range c.queues {
		lags[q] = 0
	}
	rows, err = c.pool.Query(ctx, `
		SELECT queue, extract(epoch FROM now() - min(scheduled_at))::float8
		FROM river_job
		WHERE state = 'available' AND scheduled_at <= now()
		GROUP BY queue`)
	if err == nil {
		for rows.Next() {
			var queue string
			var lag float64
			if rows.Scan(&queue, &lag) == nil {
				lags[queue] = max(lag, 0)
			}
		}
		rows.Close()
		err = rows.Err()
	}
	if err != nil {
		failed++
	}
	for q, lag := range lags {
		gauge(c.jobLag, lag, q)
	}

	// Webhook ingest: stored, waiting to be normalized.
	for _, src := range []struct{ channel, table string }{
		{"email", "provider_raw_events"}, {"sms", "sms_raw_events"}, {"whatsapp", "whatsapp_raw_events"},
	} {
		var n, lag float64
		// The table names are constants above.
		err := c.pool.QueryRow(ctx, `SELECT count(*)::float8, coalesce(extract(epoch FROM now() - min(received_at)), 0)::float8 FROM `+
			src.table+` WHERE normalized_at IS NULL`).Scan(&n, &lag)
		if err != nil {
			failed++
			continue
		}
		gauge(c.ingestPending, n, src.channel)
		gauge(c.ingestLag, max(lag, 0), src.channel)
	}

	// Outgoing webhooks waiting for an attempt.
	var pending, lag float64
	err = c.pool.QueryRow(ctx, `
		SELECT count(*)::float8,
		       coalesce(extract(epoch FROM now() - min(coalesce(next_attempt_at, created_at)) FILTER (WHERE coalesce(next_attempt_at, created_at) <= now())), 0)::float8
		FROM webhook_deliveries WHERE state IN ('pending', 'retrying')`).Scan(&pending, &lag)
	if err == nil {
		gauge(c.hooksPending, pending)
		gauge(c.hooksLag, max(lag, 0))
	} else {
		failed++
	}
	gauge(c.scrapeErrors, float64(failed))
	return out
}

// poolCollector reports the database pool's connections and waits.
type poolCollector struct{ pool *pgxpool.Pool }

var (
	poolConns = prometheus.NewDesc("reputecast_db_pool_connections", "Database pool connections by state (acquired, idle, constructing).",
		[]string{"state"}, nil)
	poolMax      = prometheus.NewDesc("reputecast_db_pool_max_connections", "The database pool's size limit.", nil, nil)
	poolAcquires = prometheus.NewDesc("reputecast_db_pool_acquires_total", "Connections acquired from the pool.", nil, nil)
	poolEmpty    = prometheus.NewDesc("reputecast_db_pool_empty_acquires_total", "Acquires that had to wait because no connection was idle.", nil, nil)
	poolCanceled = prometheus.NewDesc("reputecast_db_pool_canceled_acquires_total", "Acquires cancelled while waiting for a connection.", nil, nil)
	poolWait     = prometheus.NewDesc("reputecast_db_pool_acquire_wait_seconds_total", "Time spent waiting for a connection.", nil, nil)
)

func (poolCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range []*prometheus.Desc{poolConns, poolMax, poolAcquires, poolEmpty, poolCanceled, poolWait} {
		ch <- d
	}
}

func (p poolCollector) Collect(ch chan<- prometheus.Metric) {
	s := p.pool.Stat()
	ch <- prometheus.MustNewConstMetric(poolConns, prometheus.GaugeValue, float64(s.AcquiredConns()), "acquired")
	ch <- prometheus.MustNewConstMetric(poolConns, prometheus.GaugeValue, float64(s.IdleConns()), "idle")
	ch <- prometheus.MustNewConstMetric(poolConns, prometheus.GaugeValue, float64(s.ConstructingConns()), "constructing")
	ch <- prometheus.MustNewConstMetric(poolMax, prometheus.GaugeValue, float64(s.MaxConns()))
	ch <- prometheus.MustNewConstMetric(poolAcquires, prometheus.CounterValue, float64(s.AcquireCount()))
	ch <- prometheus.MustNewConstMetric(poolEmpty, prometheus.CounterValue, float64(s.EmptyAcquireCount()))
	ch <- prometheus.MustNewConstMetric(poolCanceled, prometheus.CounterValue, float64(s.CanceledAcquireCount()))
	ch <- prometheus.MustNewConstMetric(poolWait, prometheus.CounterValue, s.AcquireDuration().Seconds())
}
