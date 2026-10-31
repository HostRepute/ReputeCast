package db_test

import (
	"context"
	"io"
	"io/fs"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/testdb"
	"github.com/hostrepute/reputecast/migrations"
)

// Two replicas started together with --auto-migrate: one migrates while the
// other waits for the lock, and every migration runs exactly once.
func TestConcurrentMigratorsRunEachMigrationOnce(t *testing.T) {
	defer db.SetMigrateLockPoll(100 * time.Millisecond)()
	ctx := context.Background()
	url := testdb.Empty(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		pool, err := db.Open(ctx, url, 4)
		if err != nil {
			t.Fatal(err)
		}
		defer pool.Close()
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = db.MigrateUp(ctx, pool, log)
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("migrator %d: %v", i, err)
		}
	}
	pool, err := db.Open(ctx, url, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var rows, versions int
	if err := pool.QueryRow(ctx, `SELECT count(*), count(DISTINCT version_id) FROM goose_db_version WHERE version_id > 0`).Scan(&rows, &versions); err != nil {
		t.Fatal(err)
	}
	files := 0
	entries, _ := fs.ReadDir(migrations.FS, ".")
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") || (strings.HasSuffix(e.Name(), ".go") && e.Name() != "embed.go") {
			files++
		}
	}
	if rows != versions || versions != files {
		t.Fatalf("goose_db_version has %d rows for %d versions; %d migration files", rows, versions, files)
	}
}

// A pool of one connection can't hold the lock and migrate at once.
func TestMigrateNeedsTwoConnections(t *testing.T) {
	ctx := context.Background()
	pool, err := db.Open(ctx, testdb.Empty(t), 1)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.MigrateUp(ctx, pool, slog.New(slog.NewTextHandler(io.Discard, nil))); err == nil || !strings.Contains(err.Error(), "at least 2") {
		t.Fatalf("err = %v, want the two-connection error", err)
	}
}
