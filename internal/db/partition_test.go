package db_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// delivery_events turns into monthly partitions without losing or copying
// a row; reads, writes, dedupe and the detail retention keep working
// (review 2026-11 OP-8).
func TestPartitionDeliveryEvents(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	pool := env.App.Pool
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	// An upgrade converts the table (OP-11); this starts from one that
	// hasn't been, as the opt-in command found it.
	unpartition(t, ctx, pool, true)
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	counts := func() map[string]int64 {
		t.Helper()
		rows, err := pool.Query(ctx, `SELECT type, count(*) FROM delivery_events GROUP BY 1`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[string]int64{}
		for rows.Next() {
			var k string
			var n int64
			if err := rows.Scan(&k, &n); err != nil {
				t.Fatal(err)
			}
			out[k] = n
		}
		return out
	}
	var campaign string
	if err := pool.QueryRow(ctx, `SELECT c.id::text FROM campaigns c JOIN delivery_events e ON e.campaign_id = c.id GROUP BY c.id ORDER BY count(*) DESC LIMIT 1`).Scan(&campaign); err != nil {
		t.Fatal(err)
	}
	report := func() string {
		t.Helper()
		return string(c.Do("GET", "/workspaces/acme-newsletter/campaigns/"+campaign+"/report", nil).MustStatus(t, 200).Body)
	}
	before, reportBefore := counts(), report()
	var provider uuid.UUID
	var oldKey string
	if err := pool.QueryRow(ctx, `SELECT provider_id, dedupe_key FROM delivery_events WHERE dedupe_key <> '' AND provider_id IS NOT NULL ORDER BY received_at DESC LIMIT 1`).Scan(&provider, &oldKey); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	res, err := db.PartitionDeliveryEvents(ctx, pool, log, now)
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := db.EventsPartitioned(ctx, pool); !ok || res.Already || res.KeysCopied == 0 {
		t.Fatalf("partitioned: %v %+v", ok, res)
	}
	parts, err := db.EventPartitions(ctx, pool)
	if err != nil || len(parts) != db.EventPartitionsAhead+1 && len(parts) != db.EventPartitionsAhead {
		t.Fatalf("partitions: %+v %v", parts, err)
	}
	if !parts[0].From.Equal(res.FirstMonth) {
		t.Fatalf("first month %v, partitions %+v", res.FirstMonth, parts)
	}
	after := counts()
	for k, n := range before {
		if after[k] != n {
			t.Fatalf("%s: %d before, %d after", k, n, after[k])
		}
	}
	if got := report(); got != reportBefore {
		t.Fatalf("the report reads differently:\n%s\n%s", reportBefore, got)
	}
	c.Do("GET", "/workspaces/acme-newsletter/deliverability/events", nil)

	// Writes: a new event lands in the current month; a key seen before the
	// conversion, or twice, is stored once.
	insert := func(key string, at time.Time) bool {
		t.Helper()
		var ws uuid.UUID
		_ = pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&ws)
		_, err := env.App.Q.IngestInsertEvent(ctx, store.IngestInsertEventParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws, ProviderID: &provider,
			Type: "delivered", OccurredAt: at, MailboxProvider: "gmail", ReceivedAt: at, DedupeKey: key})
		return err == nil
	}
	if insert(oldKey, now) {
		t.Fatal("an event seen before the conversion was stored again")
	}
	if !insert("m47-new-key", now) || insert("m47-new-key", now) {
		t.Fatal("a new key: stored once")
	}
	// Months from now: the partition made ahead takes it.
	if !insert("m47-later", res.FirstMonth.Add(36*time.Hour)) {
		t.Fatal("an event next month")
	}

	// Again: nothing to do but keep the partitions ahead.
	again, err := db.PartitionDeliveryEvents(ctx, pool, log, now)
	if err != nil || !again.Already {
		t.Fatalf("again: %+v %v", again, err)
	}
	made, err := db.EnsureEventPartitions(ctx, pool, now.AddDate(0, 4, 0))
	if err != nil || len(made) == 0 {
		t.Fatalf("partitions ahead later: %v %v", made, err)
	}

	// DP-6 by partition: once next month's partition is past a workspace's
	// cutoff, its opens and clicks lose their IP address and user agent in
	// one pass.
	if _, err := pool.Exec(ctx, `UPDATE workspaces SET event_detail_days = 30 WHERE slug = 'acme-newsletter'`); err != nil {
		t.Fatal(err)
	}
	var open uuid.UUID
	err = pool.QueryRow(ctx, `
		INSERT INTO delivery_events (id, workspace_id, type, occurred_at, received_at, user_agent, ip)
		SELECT gen_random_uuid(), id, 'open', $1, $1, 'Mozilla/5.0', '198.51.100.7' FROM workspaces WHERE slug = 'acme-newsletter'
		RETURNING id`, res.FirstMonth.Add(48*time.Hour)).Scan(&open)
	if err != nil {
		t.Fatal(err)
	}
	later := res.FirstMonth.AddDate(0, 3, 0)
	n, err := env.App.Audience().ClearDetailByPartition(ctx, later)
	if err != nil || n < 1 {
		t.Fatalf("cleared %d: %v", n, err)
	}
	var ua string
	var ip *string
	_ = pool.QueryRow(ctx, `SELECT user_agent, host(ip) FROM delivery_events WHERE id = $1`, open).Scan(&ua, &ip)
	if ua != "" || ip != nil {
		t.Fatalf("detail kept: %q %v", ua, ip)
	}
	if n, _ := env.App.Audience().ClearDetailByPartition(ctx, later); n != 0 {
		t.Fatalf("a partition is cleared once: %d", n)
	}
	c.Do("GET", "/workspaces/acme-newsletter/campaigns/"+campaign+"/report", nil).MustStatus(t, http.StatusOK)
}
