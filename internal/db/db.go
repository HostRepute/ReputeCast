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
	cfg.ConnConfig.RuntimeParams["application_name"] = "reputecast"
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

// MigrateUp applies all pending goose migrations, then River's migrations.
func MigrateUp(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
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

// MigrateDown rolls back the latest goose migration.
func MigrateDown(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
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
