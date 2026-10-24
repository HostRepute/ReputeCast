package testdb

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// sharedMaxAge is how long a run reuses a shared template before it fills a
// new one, so data placed relative to when it was filled (last opens, decay
// steps, "today") stays close to the clock.
const sharedMaxAge = 10 * time.Minute

// runOwner is the `go test` process that runs every test binary of a run.
// Shared templates carry its PID as their owner: the sweep keeps them while
// the run lasts and drops them once it has ended.
var runOwner = os.Getppid()

// Shared is Template filled once per `go test` run rather than once per test
// binary: the first binary to ask fills it, the others wait for it and reuse
// it. fill writes the database through pool and any files into dir, which
// the caller copies for each clone. key names the fill and must change with
// anything that changes what fill writes.
func Shared(ctx context.Context, key string, fill func(pool *pgxpool.Pool, dir string) error) (name, dir string, err error) {
	once.Do(setup)
	if initErr != nil {
		return "", "", initErr
	}
	admin, err := pgx.Connect(ctx, withDB(baseURL, "postgres"))
	if err != nil {
		return "", "", err
	}
	defer func() { _ = admin.Close(ctx) }()
	prefix := "rc_fill_" + key + "_" + hostTag + "_" + strconv.Itoa(runOwner) + "_"
	sum := sha256.Sum256([]byte(prefix))
	lock := int64(binary.BigEndian.Uint64(sum[:8]))
	if _, err := admin.Exec(ctx, `SELECT pg_advisory_lock($1)`, lock); err != nil {
		return "", "", err
	}
	defer admin.Exec(ctx, `SELECT pg_advisory_unlock($1)`, lock) //nolint:errcheck

	if name, ok := newestShared(ctx, admin, prefix, time.Now()); ok {
		return name, sharedDir(name), nil
	}
	name = prefix + strconv.FormatInt(time.Now().Unix(), 10) + "_" + randHex(3)
	dir = sharedDir(name)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", "", err
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+name+` TEMPLATE `+templateName); err != nil {
		return "", "", err
	}
	pool, err := pgxpool.New(ctx, withDB(baseURL, name))
	if err == nil {
		err = fill(pool, dir)
		pool.Close()
	}
	if err == nil {
		_, err = admin.Exec(ctx, `COMMENT ON DATABASE `+name+` IS 'filled:`+strconv.FormatInt(time.Now().Unix(), 10)+`'`)
	}
	if err != nil {
		_, _ = admin.Exec(ctx, `DROP DATABASE IF EXISTS `+name+` WITH (FORCE)`)
		_ = os.RemoveAll(dir)
		return "", "", fmt.Errorf("fill %s: %w", key, err)
	}
	return name, dir, nil
}

// newestShared finds the run's most recent complete template for prefix, if
// it is younger than sharedMaxAge.
func newestShared(ctx context.Context, admin *pgx.Conn, prefix string, now time.Time) (string, bool) {
	rows, err := admin.Query(ctx, `SELECT datname, coalesce(shobj_description(oid, 'pg_database'), '') FROM pg_database
		WHERE starts_with(datname, $1) AND NOT datistemplate`, prefix)
	if err != nil {
		return "", false
	}
	var best string
	var bestAt time.Time
	for rows.Next() {
		var name, comment string
		if rows.Scan(&name, &comment) != nil {
			continue
		}
		v, ok := strings.CutPrefix(comment, "filled:")
		if !ok {
			continue
		}
		ts, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			continue
		}
		if at := time.Unix(ts, 0); at.After(bestAt) {
			best, bestAt = name, at
		}
	}
	rows.Close()
	if best == "" || now.Sub(bestAt) >= sharedMaxAge {
		return "", false
	}
	return best, true
}

// sharedDir is where a shared template's files live.
func sharedDir(name string) string {
	return filepath.Join(os.TempDir(), "reputecast-test", name)
}
