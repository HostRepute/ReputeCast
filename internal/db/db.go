// Package db opens the Postgres pool and runs schema migrations (goose for
// ReputeCast's own tables, rivermigrate for River's job tables).
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
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
func MigrateUp(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	return withMigrateLock(ctx, pool, logger, func() error { return migrateUp(ctx, pool, logger) })
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
