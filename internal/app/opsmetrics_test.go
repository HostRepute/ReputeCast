package app_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/opsmetrics"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
)

type failArgs struct{}

func (failArgs) Kind() string { return "opsmetrics_test.fail" }

type failWorker struct{ river.WorkerDefaults[failArgs] }

func (failWorker) Work(context.Context, *river.Job[failArgs]) error {
	return errors.New("always fails")
}

// Every operational metric shows on /metrics once its work has happened
// (review 2026-11 OP-1): jobs, provider sends, ingest, outgoing webhooks
// and the database pool.
func TestOperationalMetricsAreScraped(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Cfg.Webhooks.AllowPrivateNetworks = true
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	// A job that fails its only attempt, run by a real River client.
	reg := jobs.NewRegistry()
	river.AddWorker(reg.Workers, &failWorker{})
	client, err := jobs.NewClient(env.App.Pool, reg, jobs.Options{Run: true, Concurrency: 1, Logger: env.App.Log})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Insert(ctx, failArgs{}, &river.InsertOpts{MaxAttempts: 1, Queue: jobs.QueueSystem}); err != nil {
		t.Fatal(err)
	}
	failures := opsmetrics.JobFailures.WithLabelValues("opsmetrics_test.fail", "discarded")
	for deadline := time.Now().Add(20 * time.Second); testutil.ToFloat64(failures) < 1; {
		if time.Now().After(deadline) {
			t.Fatal("the failed job wasn't counted")
		}
		time.Sleep(50 * time.Millisecond)
	}
	stopCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	_ = client.Stop(stopCtx)
	cancel()

	// A provider call: two accepted, one bounced at once, one deferred.
	var pid uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM providers ORDER BY id LIMIT 1`).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	p, err := env.App.Q.ProvGetAny(ctx, pid)
	if err != nil {
		t.Fatal(err)
	}
	res := []providers.SendResult{{}, {}, {Err: &providers.SendError{Code: "550"}}, {Err: &providers.SendError{Temporary: true, Code: "421"}}}
	if err := manage.RecordSend(ctx, env.App.Pool, p, res, nil, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := testutil.ToFloat64(opsmetrics.ProviderMessages.WithLabelValues(p.Adapter, opsmetrics.Accepted)); got < 2 {
		t.Fatalf("accepted: %v", got)
	}

	// A provider webhook for an unknown endpoint, and one outgoing webhook.
	c.DoRaw("POST", "/hooks/ses/not-a-token", "application/json", strings.NewReader(`{}`))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	defer srv.Close()
	var hook struct{ Webhook struct{ ID string } }
	c.Do("POST", "/workspaces/acme-newsletter/webhooks", map[string]any{"name": "Metrics", "url": srv.URL, "events": []string{"email.*"}}).MustStatus(t, 201).JSON(t, &hook)
	c.Do("POST", "/workspaces/acme-newsletter/webhooks/"+hook.Webhook.ID+"/test", nil).MustStatus(t, 200)

	names := map[string]bool{}
	families, err := env.App.Metrics.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range families {
		names[f.GetName()] = true
	}
	for _, want := range []string{
		"reputecast_jobs", "reputecast_job_queue_lag_seconds", "reputecast_job_failures_total",
		"reputecast_provider_messages_total", "reputecast_provider_calls_total",
		"reputecast_ingest_requests_total", "reputecast_ingest_pending", "reputecast_ingest_lag_seconds",
		"reputecast_webhook_deliveries_total", "reputecast_webhook_deliveries_pending", "reputecast_webhook_deliveries_lag_seconds",
		"reputecast_db_pool_connections", "reputecast_db_pool_max_connections", "reputecast_db_pool_acquires_total",
		"reputecast_db_pool_empty_acquires_total", "reputecast_db_pool_canceled_acquires_total", "reputecast_db_pool_acquire_wait_seconds_total",
		"reputecast_metrics_scrape_errors",
	} {
		if !names[want] {
			t.Errorf("%s isn't scraped", want)
		}
	}
	if v := testutil.ToFloat64(opsmetrics.IngestRequests.WithLabelValues("email", "unknown")); v < 1 {
		t.Errorf("ingest unknown: %v", v)
	}
	if v := testutil.ToFloat64(opsmetrics.WebhookDeliveries.WithLabelValues("delivered")); v < 1 {
		t.Errorf("webhook delivered: %v", v)
	}
	// The discarded job is in the queue gauge, with its kind.
	body := scrape(t, env)
	if !strings.Contains(body, `reputecast_jobs{kind="opsmetrics_test.fail",queue="system",state="discarded"} 1`) {
		t.Errorf("discarded job not in the gauge:\n%s", grepLines(body, "reputecast_jobs{"))
	}
	if !strings.Contains(body, `reputecast_job_queue_lag_seconds{queue="sending"}`) || strings.Contains(body, "reputecast_metrics_scrape_errors 1") {
		t.Errorf("lag and errors:\n%s", grepLines(body, "lag_seconds", "scrape_errors"))
	}
}

func scrape(t *testing.T, env *apptest.Env) string {
	t.Helper()
	rec := httptest.NewRecorder()
	promhttp.HandlerFor(env.App.Metrics, promhttp.HandlerOpts{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	return rec.Body.String()
}

func grepLines(body string, subs ...string) string {
	var out []string
	for _, l := range strings.Split(body, "\n") {
		for _, s := range subs {
			if strings.Contains(l, s) {
				out = append(out, l)
				break
			}
		}
	}
	return strings.Join(out, "\n")
}
