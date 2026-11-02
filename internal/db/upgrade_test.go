package db_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

func dedupeIndexExists(t *testing.T, ctx context.Context, pool *pgxpool.Pool) bool {
	t.Helper()
	var ok bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'delivery_events_dedupe_key' AND relkind = 'i')`).Scan(&ok); err != nil {
		t.Fatal(err)
	}
	return ok
}

// An upgrade converts delivery_events of an instance that never ran the
// opt-in command, without losing a row or a dedupe key, and drops the
// unpartitioned table's unique dedupe index; a second run changes nothing
// (review 2026-11 OP-11).
func TestMigrateUpConvertsEvents(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	pool := env.App.Pool
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	unpartition(t, ctx, pool, true)

	var rows int64
	var provider uuid.UUID
	var oldKey string
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM delivery_events`).Scan(&rows); err != nil || rows == 0 {
		t.Fatalf("the demo has no events: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT provider_id, dedupe_key FROM delivery_events WHERE dedupe_key <> '' AND provider_id IS NOT NULL
		ORDER BY received_at DESC LIMIT 1`).Scan(&provider, &oldKey); err != nil {
		t.Fatal(err)
	}
	// The release before kept every key in its unique index, not all of
	// them in delivery_event_keys.
	if _, err := pool.Exec(ctx, `DELETE FROM delivery_event_keys`); err != nil {
		t.Fatal(err)
	}

	if err := db.MigrateUp(ctx, pool, log); err != nil {
		t.Fatal(err)
	}
	if ok, err := db.EventsPartitioned(ctx, pool); !ok || err != nil {
		t.Fatalf("not partitioned after the upgrade: %v", err)
	}
	if dedupeIndexExists(t, ctx, pool) {
		t.Fatal("the legacy dedupe index is still there")
	}
	var after int64
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM delivery_events`).Scan(&after); err != nil || after != rows {
		t.Fatalf("%d events before, %d after (%v)", rows, after, err)
	}
	parts, err := db.EventPartitions(ctx, pool)
	if err != nil || len(parts) < db.EventPartitionsAhead {
		t.Fatalf("partitions ahead: %+v %v", parts, err)
	}
	var ws uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	insert := func(key string) bool {
		t.Helper()
		now := time.Now().UTC()
		_, err := env.App.Q.IngestInsertEvent(ctx, store.IngestInsertEventParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws, ProviderID: &provider,
			Type: "delivered", OccurredAt: now, MailboxProvider: "gmail", ReceivedAt: now, DedupeKey: key})
		return err == nil
	}
	if insert(oldKey) {
		t.Fatal("an event the upgrade had seen was stored again without the old index")
	}
	if !insert("op11-new-key") || insert("op11-new-key") {
		t.Fatal("a new key is stored once")
	}

	if err := db.MigrateUp(ctx, pool, log); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM delivery_events`).Scan(&after); err != nil || after != rows+1 {
		t.Fatalf("a second upgrade changed the events: %d, want %d (%v)", after, rows+1, err)
	}
}

// An instance that ran `reputecast migrate partition-events` on the release
// before keeps that conversion; the upgrade only drops the dedupe index
// its legacy partition kept (review 2026-11 OP-11).
func TestMigrateUpAfterTheOptInCommand(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	pool := env.App.Pool
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	unpartition(t, ctx, pool, true)
	res, err := db.PartitionDeliveryEvents(ctx, pool, log, time.Now())
	if err != nil || res.Already {
		t.Fatalf("opt-in conversion: %+v %v", res, err)
	}
	if !dedupeIndexExists(t, ctx, pool) {
		t.Fatal("the opt-in command dropped the dedupe index; the upgrade was meant to")
	}
	var legacyBefore int64
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM delivery_events_legacy`).Scan(&legacyBefore); err != nil {
		t.Fatal(err)
	}
	if err := db.MigrateUp(ctx, pool, log); err != nil {
		t.Fatal(err)
	}
	if dedupeIndexExists(t, ctx, pool) {
		t.Fatal("the legacy dedupe index is still there")
	}
	var legacyAfter int64
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM delivery_events_legacy`).Scan(&legacyAfter); err != nil || legacyAfter != legacyBefore {
		t.Fatalf("the legacy partition went from %d to %d rows (%v)", legacyBefore, legacyAfter, err)
	}
}
