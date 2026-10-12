// Package testdb gives integration tests a fresh, migrated Postgres database.
//
// The first call starts Postgres 18 with testcontainers (or uses the server
// in REPUTECAST_TEST_DATABASE_URL, which is faster locally: point it at the
// docker compose Postgres), migrates a template database once, and then
// clones it per test with CREATE DATABASE … TEMPLATE, which takes a few ms.
package testdb

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/migrations"
)

const templateName = "reputecast_test_template"

var (
	once    sync.Once
	baseURL string
	initErr error
)

// Image is the Postgres image used when no server URL is configured.
var Image = "postgres:18-alpine"

func setup() {
	ctx := context.Background()
	baseURL = os.Getenv("REPUTECAST_TEST_DATABASE_URL")
	if baseURL == "" {
		c, err := tcpostgres.Run(ctx, Image,
			tcpostgres.WithDatabase("postgres"),
			tcpostgres.WithUsername("reputecast"),
			tcpostgres.WithPassword("reputecast"),
			testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second)),
		)
		if err != nil {
			initErr = fmt.Errorf("start postgres container: %w", err)
			return
		}
		baseURL, err = c.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			initErr = err
			return
		}
	}
	initErr = ensureTemplate(ctx)
}

// ensureTemplate creates and migrates the template database once. Several
// test binaries may race here, so the work runs under an advisory lock.
func ensureTemplate(ctx context.Context) error {
	admin, err := pgx.Connect(ctx, withDB(baseURL, "postgres"))
	if err != nil {
		return fmt.Errorf("connect to test postgres: %w", err)
	}
	defer admin.Close(ctx)
	if _, err := admin.Exec(ctx, `SELECT pg_advisory_lock(727272)`); err != nil {
		return err
	}
	defer admin.Exec(ctx, `SELECT pg_advisory_unlock(727272)`) //nolint:errcheck
	want := migrationsHash()
	var exists bool
	var have string
	if err := admin.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1),
		coalesce((SELECT shobj_description(oid, 'pg_database') FROM pg_database WHERE datname = $1), '')`, templateName).Scan(&exists, &have); err != nil {
		return err
	}
	// The template is rebuilt whenever a migration is added or changed.
	if exists && have == want && os.Getenv("REPUTECAST_TEST_REBUILD_TEMPLATE") == "" {
		return nil
	}
	if exists {
		if _, err := admin.Exec(ctx, `DROP DATABASE `+templateName+` WITH (FORCE)`); err != nil {
			return err
		}
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+templateName); err != nil {
		return err
	}
	pool, err := pgxpool.New(ctx, withDB(baseURL, templateName))
	if err != nil {
		return err
	}
	defer pool.Close()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := db.MigrateUp(ctx, pool, log); err != nil {
		return fmt.Errorf("migrate template: %w", err)
	}
	pool.Close()
	_, err = admin.Exec(ctx, `COMMENT ON DATABASE `+templateName+` IS '`+want+`'`)
	return err
}

// migrationsHash fingerprints every embedded migration file.
func migrationsHash() string {
	h := sha256.New()
	entries, _ := fs.ReadDir(migrations.FS, ".")
	for _, e := range entries {
		b, _ := fs.ReadFile(migrations.FS, e.Name())
		h.Write([]byte(e.Name()))
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil))[:32]
}

// New returns a pool connected to a fresh database cloned from the migrated
// template. The database is dropped when the test ends.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("integration test: needs Postgres (run without -short)")
	}
	once.Do(setup)
	if initErr != nil {
		t.Fatalf("testdb: %v", initErr)
	}
	ctx := context.Background()
	name := "rc_test_" + randHex(6)
	admin, err := pgx.Connect(ctx, withDB(baseURL, "postgres"))
	if err != nil {
		t.Fatalf("testdb connect: %v", err)
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+name+` TEMPLATE `+templateName); err != nil {
		_ = admin.Close(ctx)
		t.Fatalf("testdb create: %v", err)
	}
	_ = admin.Close(ctx)
	pool, err := pgxpool.New(ctx, withDB(baseURL, name))
	if err != nil {
		t.Fatalf("testdb pool: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		a, err := pgx.Connect(context.Background(), withDB(baseURL, "postgres"))
		if err == nil {
			_, _ = a.Exec(context.Background(), `DROP DATABASE IF EXISTS `+name+` WITH (FORCE)`)
			_ = a.Close(context.Background())
		}
	})
	return pool
}

// URL returns the connection URL of the database behind pool's config.
func URL(pool *pgxpool.Pool) string { return pool.Config().ConnString() }

func withDB(raw, name string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Path = "/" + strings.TrimPrefix(name, "/")
	return u.String()
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
