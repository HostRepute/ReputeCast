package demo

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/testdb"
)

// sampleDay is when the tests seed the demo: noon in New York on the 15th
// of this month, or of the month before until then. The tests read it as of
// that moment. Growth's cohorts are calendar months beside a rolling 30
// days, so around a month's turn they can't all be the boards' numbers (the
// 30 days hold more of the month before than its cohort, or a 30-day mark
// is still to come); mid-month they are, whatever the day the tests run.
func sampleDay(now time.Time) time.Time {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	l := now.In(ny)
	at := time.Date(l.Year(), l.Month(), 15, 12, 0, 0, 0, ny)
	if at.After(now) {
		at = at.AddDate(0, -1, 0)
	}
	// In UTC, as a server's clock reads: the store's last 30 days are then
	// 30 times 24 hours, where in New York one of them can have 25.
	return at.UTC()
}

// seededSample returns a database of its own with the demo seeded at
// sampleDay, the box that sealed the demo's secrets, and the seed's moment.
// The demo is seeded once for the run (testdb.Shared), minutes under -race,
// and cloned for each test.
func seededSample(t *testing.T) (*pgxpool.Pool, *secrets.Box, time.Time) {
	t.Helper()
	at := sampleDay(time.Now())
	pool, box := seededAt(t, at)
	return pool, box, at
}

// seededAt is seededSample for a seed at another moment.
func seededAt(t *testing.T, at time.Time) (*pgxpool.Pool, *secrets.Box) {
	t.Helper()
	if testing.Short() {
		t.Skip("seeds the whole demo")
	}
	box, err := secrets.New(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	name, _, err := testdb.Shared(context.Background(), "sample"+at.UTC().Format("20060102t1504"), func(pool *pgxpool.Pool, dir string) error {
		cfg := config.Defaults()
		cfg.Files.Dir = dir // the sample product files (X19)
		_, err := seedAt(context.Background(), pool, &cfg, box, slog.New(slog.NewTextHandler(io.Discard, nil)), at)
		return err
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return testdb.NewFrom(t, name), box
}
