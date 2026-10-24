package testdb

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A run fills a shared template once: later calls, from this binary or its
// siblings, get the same database and files until it is sharedMaxAge old.
func TestSharedFillsOncePerRun(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: needs Postgres (run without -short)")
	}
	ctx := context.Background()
	key := "sharetest" + randHex(3)
	fills := 0
	fill := func(pool *pgxpool.Pool, dir string) error {
		fills++
		if _, err := pool.Exec(ctx, `CREATE TABLE shared_probe (n int)`); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, "probe.txt"), []byte("x"), 0o600)
	}
	name, dir, err := Shared(ctx, key, fill)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if a, err := pgx.Connect(context.Background(), withDB(baseURL, "postgres")); err == nil {
			_, _ = a.Exec(context.Background(), `DROP DATABASE IF EXISTS `+name+` WITH (FORCE)`)
			_ = a.Close(context.Background())
		}
		_ = os.RemoveAll(dir)
	})
	again, dir2, err := Shared(ctx, key, fill)
	if err != nil || again != name || dir2 != dir || fills != 1 {
		t.Fatalf("second call: %s %s %v, %d fills", again, dir2, err, fills)
	}

	var n int
	if err := NewFrom(t, name).QueryRow(ctx, `SELECT count(*) FROM shared_probe`).Scan(&n); err != nil {
		t.Fatalf("a clone carries the fill: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "probe.txt")); err != nil {
		t.Fatalf("the fill's files: %v", err)
	}

	admin, err := pgx.Connect(ctx, withDB(baseURL, "postgres"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = admin.Close(ctx) }()
	prefix := "rc_fill_" + key + "_" + hostTag + "_" + strconv.Itoa(runOwner) + "_"
	if got, ok := newestShared(ctx, admin, prefix, time.Now()); !ok || got != name {
		t.Fatalf("newest: %q %v", got, ok)
	}
	if _, ok := newestShared(ctx, admin, prefix, time.Now().Add(sharedMaxAge)); ok {
		t.Fatal("a template sharedMaxAge old is filled again")
	}
}
