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
	if testing.Short() {
		t.Skip("seeds the whole demo")
	}
	box, err := secrets.New(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	at := sampleDay(time.Now())
	name, _, err := testdb.Shared(context.Background(), "demosample"+at.Format("20060102"), func(pool *pgxpool.Pool, dir string) error {
		cfg := config.Defaults()
		cfg.Files.Dir = dir // the sample product files (X19)
		defer func(clock func() time.Time) { seedClock = clock }(seedClock)
		seedClock = func() time.Time { return at }
		_, err := Seed(context.Background(), pool, &cfg, box, slog.New(slog.NewTextHandler(io.Discard, nil)))
		return err
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return testdb.NewFrom(t, name), box, at
}
