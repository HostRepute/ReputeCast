// Package db opens the Postgres pool and runs schema migrations (goose for
// ReputeCast's own tables, rivermigrate for River's job tables).
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/hostrepute/reputecast/migrations"
)

// Open creates a connection pool and verifies the database is reachable.
func Open(ctx context.Context, url string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("database url: %w", err)
	}
	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}
	cfg.MaxConnIdleTime = 5 * time.Minute
	Configure(cfg)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	return pool, nil
}

// Configure sets up a pool's connections the way ReputeCast uses them;
// Open does it, and so do the test databases (internal/testdb).
//
// JIT is off: Postgres compiles a plan whose estimated cost passes
// jit_above_cost with JIT, again on every execution, and ReputeCast's
// queries are short (counting a 48,000-contact audience for Review & send
// spent 0.5 s compiling and 0.1 s running). It is set once connected rather
// than in the startup packet, which poolers such as PgBouncer refuse for
// settings they don't know. UUIDs travel as bytes (registerUUID).
func Configure(cfg *pgxpool.Config) {
	cfg.ConnConfig.RuntimeParams["application_name"] = "reputecast"
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		registerUUID(c.TypeMap())
		_, err := c.Exec(ctx, "SET jit = off")
		return err
	}
}

// MigrationStatus describes one goose migration.
type MigrationStatus struct {
	Version   int64
	Name      string
	Applied   bool
	AppliedAt time.Time
}

func gooseProvider(pool *pgxpool.Pool, logger *slog.Logger) (*goose.Provider, *sql.DB, error) {
	sqlDB := stdlib.OpenDBFromPool(pool)
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS,
		goose.WithSlog(logger),
		goose.WithAllowOutofOrder(true),
	)
	if err != nil {
		_ = sqlDB.Close()
		return nil, nil, err
	}
	return p, sqlDB, nil
}

// migrateLockKey is the session advisory lock (pg_try_advisory_lock(int,
// int)) that one migrator holds at a time, so replicas started together with
// --auto-migrate never run a migration twice.
const (
	migrateLockClass = 0x7263 // "rc"
	migrateLockID    = 1
)

// migrateLockPoll and migrateLockWait bound how a second migrator waits.
var (
	migrateLockPoll = 2 * time.Second
	migrateLockWait = 15 * time.Minute
)

// withMigrateLock runs fn while this process holds the migration lock. It
// polls pg_try_advisory_lock on a dedicated connection rather than blocking
// in pg_advisory_lock: a blocked waiter holds a snapshot, and a migration
// building an index CONCURRENTLY would wait for it forever. The lock covers
// goose and River together.
func withMigrateLock(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger, fn func() error) error {
	if pool.Config().MaxConns < 2 {
		return errors.New("migrate: the database pool needs at least 2 connections")
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("migrate lock: %w", err)
	}
	defer conn.Release()
	deadline := time.Now().Add(migrateLockWait)
	logged := false
	for {
		var got bool
		if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1, $2)", migrateLockClass, migrateLockID).Scan(&got); err != nil {
			return fmt.Errorf("migrate lock: %w", err)
		}
		if got {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("migrate: another migrator held the lock for %s", migrateLockWait)
		}
		if !logged {
			logger.Info("waiting for another migrator to finish")
			logged = true
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(migrateLockPoll):
		}
	}
	defer func() {
		_, _ = conn.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1, $2)", migrateLockClass, migrateLockID)
	}()
	return fn()
}

// MigrateUp applies all pending goose migrations, then River's migrations,
// holding the migration lock throughout.
//
// After the schema's migrations it does the upgrade steps a goose migration
// can't (review 2026-11 OP-11): delivery_events becomes monthly partitions
// when `reputecast migrate partition-events` hasn't made it so already, and
// the unpartitioned table's unique dedupe index goes.
func MigrateUp(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	return withMigrateLock(ctx, pool, logger, func() error {
		if err := migrateUp(ctx, pool, logger); err != nil {
			return err
		}
		return upgradeEvents(ctx, pool, logger, time.Now())
	})
}

func migrateUp(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	p, sqlDB, err := gooseProvider(pool, logger)
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()
	results, err := p.Up(ctx)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	for _, r := range results {
		logger.Info("migration applied", "version", r.Source.Version, "duration", r.Duration)
	}
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("river migrator: %w", err)
	}
	res, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return fmt.Errorf("river migrate: %w", err)
	}
	for _, v := range res.Versions {
		logger.Info("river migration applied", "version", v.Version)
	}
	return nil
}

// MigrateDown rolls back the latest goose migration, holding the migration
// lock.
func MigrateDown(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	return withMigrateLock(ctx, pool, logger, func() error { return migrateDown(ctx, pool, logger) })
}

func migrateDown(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	p, sqlDB, err := gooseProvider(pool, logger)
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()
	if _, err := p.Down(ctx); err != nil && !errors.Is(err, goose.ErrNoNextVersion) {
		return err
	}
	return nil
}

// Status lists goose migrations and whether they are applied.
func Status(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) ([]MigrationStatus, error) {
	p, sqlDB, err := gooseProvider(pool, logger)
	if err != nil {
		return nil, err
	}
	defer func() { _ = sqlDB.Close() }()
	st, err := p.Status(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MigrationStatus, 0, len(st))
	for _, s := range st {
		out = append(out, MigrationStatus{
			Version:   s.Source.Version,
			Name:      s.Source.Path,
			Applied:   s.State == goose.StateApplied,
			AppliedAt: s.AppliedAt,
		})
	}
	return out, nil
}

// HasPending reports whether any goose migration is not yet applied.
func HasPending(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) (bool, error) {
	p, sqlDB, err := gooseProvider(pool, logger)
	if err != nil {
		return false, err
	}
	defer func() { _ = sqlDB.Close() }()
	return p.HasPending(ctx)
}

// Tx runs fn inside a transaction, committing on success and rolling back on
// error or panic.
func Tx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SchemaState compares the database's schema with the migrations this
// binary carries (review 2026-11 OP-2, OP-3).
type SchemaState struct {
	// Expected is the newest migration this binary carries; Applied the
	// newest the database has.
	Expected, Applied int64
	// Pending are this binary's migrations the database doesn't have yet.
	Pending []int64
	// Newer is true when the database has migrations this binary doesn't
	// know: a newer release migrated it.
	Newer bool
	// RiverCurrent is true when the job queue's own migrations are applied.
	RiverCurrent bool
}

// Current reports whether every migration this binary needs is applied.
func (s SchemaState) Current() bool { return len(s.Pending) == 0 && s.RiverCurrent }

// Schema reads the schema's state. It is one small query per table, cheap
// enough for a readiness probe.
func Schema(ctx context.Context, pool *pgxpool.Pool) (SchemaState, error) {
	var st SchemaState
	carried, err := migrationVersions()
	if err != nil {
		return st, err
	}
	if len(carried) > 0 {
		st.Expected = carried[len(carried)-1]
	}
	applied := map[int64]bool{}
	rows, err := pool.Query(ctx, `SELECT DISTINCT ON (version_id) version_id, is_applied FROM goose_db_version ORDER BY version_id, id DESC`)
	if err != nil {
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "42P01" { // an empty database has no table yet
			return st, err
		}
	} else {
		for rows.Next() {
			var v int64
			var ok bool
			if err := rows.Scan(&v, &ok); err != nil {
				rows.Close()
				return st, err
			}
			if ok && v > 0 {
				applied[v] = true
				st.Applied = max(st.Applied, v)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return st, err
		}
	}
	known := make(map[int64]bool, len(carried))
	for _, v := range carried {
		known[v] = true
		if !applied[v] {
			st.Pending = append(st.Pending, v)
		}
	}
	for v := range applied {
		if !known[v] {
			st.Newer = true
			break
		}
	}
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return st, err
	}
	if existing, err := migrator.ExistingVersions(ctx); err == nil {
		all := migrator.AllVersions()
		st.RiverCurrent = len(all) == 0 || len(existing) > 0 && existing[len(existing)-1].Version >= all[len(all)-1].Version
	}
	return st, nil
}

// migrationVersions lists the versions of the embedded migrations, in
// order.
func migrationVersions() ([]int64, error) {
	names, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(names))
	for _, n := range names {
		digits, _, _ := strings.Cut(n, "_")
		v, err := strconv.ParseInt(digits, 10, 64)
		if err != nil {
			continue
		}
		out = append(out, v)
	}
	slices.Sort(out)
	return out, nil
}

// SetUpSegmentsRole runs again what migrations 00008 and 00151 do for SQL
// segments: create the reputecast_segments role when the app's user may,
// grant it to the app's user, and own the function segments run through.
// pg_dump doesn't carry roles, so a database restored on a new server
// needs it once the role exists there (docs/deploy/backup-restore.md).
func SetUpSegmentsRole(ctx context.Context, pool *pgxpool.Pool) error {
	for _, name := range []string{"00008_segment_role.sql", "00151_segment_runner.sql"} {
		b, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			return err
		}
		body := string(b)
		start := strings.Index(body, "-- +goose StatementBegin")
		end := strings.Index(body, "-- +goose StatementEnd")
		if start < 0 || end < start {
			return fmt.Errorf("%s: no statement block", name)
		}
		if _, err := pool.Exec(ctx, body[start+len("-- +goose StatementBegin"):end]); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}
