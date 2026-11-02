package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/testdb"
)

// What a transaction hands to AfterCommit runs once it has committed, from a
// savepoint too; not when it rolls back, not for the next transaction on the
// same connection; and a transaction that isn't a db.Tx is told so, so its
// caller does the work its old way (review 2026-11 OP-13).
func TestAfterCommit(t *testing.T) {
	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(testdb.Empty(t))
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 1 // every transaction below gets the same connection
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	ran := 0
	hook := func() { ran++ }

	if err := db.Tx(ctx, pool, func(tx pgx.Tx) error {
		if !db.AfterCommit(tx, hook) {
			t.Error("AfterCommit inside a db.Tx reported false")
		}
		if ran != 0 {
			t.Error("the hook ran before the commit")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if ran != 1 {
		t.Fatalf("after a commit the hook ran %d times, want 1", ran)
	}

	// Handed in a savepoint, as the automations engine does for each run.
	if err := db.Tx(ctx, pool, func(tx pgx.Tx) error {
		sp, err := tx.Begin(ctx)
		if err != nil {
			return err
		}
		if !db.AfterCommit(sp, hook) {
			t.Error("AfterCommit inside a savepoint reported false")
		}
		return sp.Commit(ctx)
	}); err != nil {
		t.Fatal(err)
	}
	if ran != 2 {
		t.Fatalf("a hook handed in a savepoint ran %d times in all, want 2", ran)
	}

	// A rollback runs nothing.
	boom := errors.New("boom")
	if err := db.Tx(ctx, pool, func(tx pgx.Tx) error {
		db.AfterCommit(tx, hook)
		return boom
	}); !errors.Is(err, boom) {
		t.Fatalf("Tx returned %v, want boom", err)
	}
	if ran != 2 {
		t.Fatalf("a rolled-back transaction ran its hook (%d runs)", ran)
	}

	// The next transaction on the same connection starts with nothing.
	if err := db.Tx(ctx, pool, func(pgx.Tx) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if ran != 2 {
		t.Fatalf("a hook leaked into the next transaction (%d runs)", ran)
	}

	// A transaction begun some other way isn't a db.Tx.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if db.AfterCommit(tx, hook) {
		t.Error("AfterCommit outside a db.Tx reported true")
	}
}
