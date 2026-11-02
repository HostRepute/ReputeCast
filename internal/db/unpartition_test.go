package db_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/db"
)

// unpartition turns delivery_events back into one table by the rollback
// steps in docs/deploy/partitioning.md, as an instance on the release
// before review 2026-11 OP-11 has it. With dedupe it also gets that
// release's unique dedupe index back.
func unpartition(t *testing.T, ctx context.Context, pool *pgxpool.Pool, dedupe bool) {
	t.Helper()
	err := db.Tx(ctx, pool, func(tx pgx.Tx) error {
		steps := []string{
			`LOCK TABLE delivery_events IN ACCESS EXCLUSIVE MODE`,
			`ALTER TABLE delivery_events DETACH PARTITION delivery_events_legacy`,
			`ALTER TABLE delivery_events_legacy DROP CONSTRAINT IF EXISTS delivery_events_before_partitions`,
			`INSERT INTO delivery_events_legacy SELECT * FROM delivery_events`,
			`DROP TABLE delivery_events`,
			`ALTER TABLE delivery_events_legacy RENAME TO delivery_events`,
			`ALTER TABLE delivery_events DROP CONSTRAINT delivery_events_legacy_pkey`,
			`ALTER TABLE delivery_events ADD CONSTRAINT delivery_events_pkey PRIMARY KEY (id)`,
		}
		for _, s := range steps {
			if _, err := tx.Exec(ctx, s); err != nil {
				return err
			}
		}
		rows, err := tx.Query(ctx, `SELECT c.relname FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid
			WHERE i.indrelid = 'delivery_events'::regclass AND c.relname LIKE '%\_legacy'`)
		if err != nil {
			return err
		}
		names, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return err
		}
		for _, n := range names {
			if _, err := tx.Exec(ctx, `ALTER INDEX `+n+` RENAME TO `+strings.TrimSuffix(n, "_legacy")); err != nil {
				return err
			}
		}
		if dedupe {
			_, err = tx.Exec(ctx, `CREATE UNIQUE INDEX delivery_events_dedupe_key ON delivery_events (provider_id, dedupe_key) WHERE dedupe_key <> ''`)
		}
		return err
	})
	if err != nil {
		t.Fatalf("unpartition: %v", err)
	}
	if ok, err := db.EventsPartitioned(ctx, pool); ok || err != nil {
		t.Fatalf("still partitioned: %v", err)
	}
}
