package webhooks_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// An endpoint that batches gets its email.* events as email.batch
// deliveries of up to 100, oldest first; other events and other endpoints
// are unchanged (review 2026-11 EN-8).
func TestEmailEventsInBatches(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Cfg.Webhooks.AllowPrivateNetworks = true
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	ctx := context.Background()
	w, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var got []struct {
		event, sig, ts string
		body           []byte
	}
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		got = append(got, struct {
			event, sig, ts string
			body           []byte
		}{r.Header.Get(webhooks.HeaderEvent), r.Header.Get(webhooks.HeaderSignature), r.Header.Get(webhooks.HeaderTimestamp), b})
		mu.Unlock()
	}))
	defer srv.Close()

	var batched, plain struct {
		Webhook struct {
			ID         string
			BatchEmail bool
		}
		Secret string
	}
	c.Do("POST", ws+"/webhooks", map[string]any{"name": "Batches", "url": srv.URL + "/batch", "events": []string{"email.*", "campaign.started"}, "batchEmail": true}).
		MustStatus(t, 201).JSON(t, &batched)
	c.Do("POST", ws+"/webhooks", map[string]any{"name": "One by one", "url": srv.URL + "/plain", "events": []string{"email.delivered"}}).
		MustStatus(t, 201).JSON(t, &plain)
	if !batched.Webhook.BatchEmail || plain.Webhook.BatchEmail {
		t.Fatalf("created: %+v %+v", batched.Webhook, plain.Webhook)
	}
	var audited string
	if err := env.App.Pool.QueryRow(ctx, `SELECT after::text FROM audit_events WHERE action = 'webhook.created' AND target_id = $1`, batched.Webhook.ID).Scan(&audited); err != nil || !contains([]byte(audited), `"batchEmail": true`) {
		t.Fatalf("audit: %s %v", audited, err)
	}
	bid, pid := uuid.MustParse(batched.Webhook.ID), uuid.MustParse(plain.Webhook.ID)

	emit := func(n int, event string) {
		t.Helper()
		err := pgx.BeginFunc(ctx, env.App.Pool, func(tx pgx.Tx) error {
			for i := range n {
				if _, err := webhooks.Emit(ctx, tx, w.ID, w.Slug, event, map[string]any{"seq": i}); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	count := func(hook uuid.UUID, state, event string) int {
		t.Helper()
		var n int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM webhook_deliveries WHERE webhook_id = $1 AND state = $2 AND event = $3`, hook, state, event).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	emit(250, webhooks.EmailDelivered)
	emit(1, "campaign.started")
	if b, p := count(bid, "batching", webhooks.EmailDelivered), count(pid, "pending", webhooks.EmailDelivered); b != 250 || p != 250 {
		t.Fatalf("waiting: batched %d, one by one %d", b, p)
	}
	if n := count(bid, "pending", "campaign.started"); n != 1 {
		t.Fatalf("other events aren't batched: %d", n)
	}
	var list struct {
		Items []struct {
			ID     string
			Queued int64
		}
	}
	c.Do("GET", ws+"/webhooks", nil).MustStatus(t, 200).JSON(t, &list)
	for _, it := range list.Items {
		if it.ID == batched.Webhook.ID && it.Queued != 251 {
			t.Fatalf("queued counts the waiting events: %d", it.Queued)
		}
	}

	s := service(env)
	made, err := s.FoldBatches(ctx, 500)
	if err != nil || made != 3 {
		t.Fatalf("fold: %d %v", made, err)
	}
	if n := count(bid, "batching", webhooks.EmailDelivered); n != 0 {
		t.Fatalf("left waiting: %d", n)
	}
	rows, err := env.App.Pool.Query(ctx, `SELECT id, payload FROM webhook_deliveries WHERE webhook_id = $1 AND event = $2 AND state = 'pending' ORDER BY created_at, id`, bid, webhooks.EmailBatch)
	if err != nil {
		t.Fatal(err)
	}
	type envelope struct {
		Type string
		Data struct {
			Count  int
			Events []struct {
				Type string
				Data struct{ Seq int }
			}
		}
	}
	var ids []uuid.UUID
	var sizes []int
	next := 0
	for rows.Next() {
		var id uuid.UUID
		var body []byte
		if err := rows.Scan(&id, &body); err != nil {
			t.Fatal(err)
		}
		var e envelope
		if err := json.Unmarshal(body, &e); err != nil {
			t.Fatal(err)
		}
		if e.Type != webhooks.EmailBatch || e.Data.Count != len(e.Data.Events) {
			t.Fatalf("batch: %s %d/%d", e.Type, e.Data.Count, len(e.Data.Events))
		}
		for _, ev := range e.Data.Events {
			if ev.Type != webhooks.EmailDelivered || ev.Data.Seq != next {
				t.Fatalf("event %d: %s seq %d", next, ev.Type, ev.Data.Seq)
			}
			next++
		}
		ids = append(ids, id)
		sizes = append(sizes, e.Data.Count)
	}
	rows.Close()
	if len(sizes) != 3 || sizes[0] != 100 || sizes[1] != 100 || sizes[2] != 50 {
		t.Fatalf("batch sizes: %v", sizes)
	}

	// A batch goes out as one signed POST.
	if _, err := s.Attempt(ctx, ids[0]); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	if len(got) != 1 || got[0].event != webhooks.EmailBatch {
		t.Fatalf("posted: %d", len(got))
	}
	ts, _ := strconv.ParseInt(got[0].ts, 10, 64)
	if !webhooks.Verify(batched.Secret, got[0].sig, ts, got[0].body, time.Now(), 5*time.Minute) {
		t.Fatal("the batch is signed like any delivery")
	}
	mu.Unlock()

	// A paused endpoint's batch is held until it resumes.
	c.Do("POST", ws+"/webhooks/"+batched.Webhook.ID+"/pause", nil).MustStatus(t, 204)
	emit(3, webhooks.EmailOpened)
	if made, err := s.FoldBatches(ctx, 500); err != nil || made != 1 || count(bid, "held", webhooks.EmailBatch) != 1 {
		t.Fatalf("paused: %d %v", made, err)
	}
	c.Do("POST", ws+"/webhooks/"+batched.Webhook.ID+"/resume", nil).MustStatus(t, 204)
	if n := count(bid, "held", webhooks.EmailBatch); n != 0 {
		t.Fatalf("resumed: %d held", n)
	}

	// The sweep's limit: 250 events and a limit of one make one batch.
	emit(250, webhooks.EmailClicked)
	if made, err := s.FoldBatches(ctx, 1); err != nil || made != 1 || count(bid, "batching", webhooks.EmailClicked) != 150 {
		t.Fatalf("limit: %d %v", made, err)
	}

	// Switching batching off sends what was waiting, and later events one by one.
	c.Do("PUT", ws+"/webhooks/"+batched.Webhook.ID, map[string]any{"name": "Batches", "url": srv.URL + "/batch", "events": []string{"email.*"}}).MustStatus(t, 200)
	if err := env.App.Pool.QueryRow(ctx, `SELECT before::text || after::text FROM audit_events WHERE action = 'webhook.updated' AND target_id = $1`, batched.Webhook.ID).Scan(&audited); err != nil ||
		!contains([]byte(audited), `"batchEmail": true`) || !contains([]byte(audited), `"batchEmail": false`) {
		t.Fatalf("update audit: %s %v", audited, err)
	}
	emit(1, webhooks.EmailBounced)
	if n := count(bid, "pending", webhooks.EmailBounced); n != 1 {
		t.Fatalf("after switching off: %d", n)
	}
	if made, err := s.FoldBatches(ctx, 500); err != nil || made != 2 || count(bid, "batching", webhooks.EmailClicked) != 0 {
		t.Fatalf("waiting events after switching off: %d %v", made, err)
	}
}
