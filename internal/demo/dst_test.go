package demo

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/config"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/testdb"
)

// The demo seeded in the hour after the clocks went back in its workspace's
// zone: at 04:30 UTC on 3 November 2026 it is 23:30 on 2 November in New
// York, the day after the change. The audience history then went back in
// 24-hour steps and wrote 1 November twice, and the seed failed on its key.
// It holds one row per day of the workspace's calendar, and the days run
// without a gap up to yesterday.
func TestSeedAfterTheClocksGoBack(t *testing.T) {
	if testing.Short() {
		t.Skip("seeds the whole demo")
	}
	box, err := secrets.New(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.New(t)
	cfg := config.Defaults()
	cfg.Files.Dir = t.TempDir()
	defer func(clock func() time.Time) { seedClock = clock }(seedClock)
	seedClock = func() time.Time { return time.Date(2026, 11, 3, 4, 30, 0, 0, time.UTC) }
	ctx := context.Background()
	if _, err := Seed(ctx, pool, &cfg, box, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("seed: %v", err)
	}
	var days, distinct int
	var first, last time.Time
	if err := pool.QueryRow(ctx, `SELECT count(*), count(DISTINCT day), min(day), max(day) FROM audience_daily WHERE scope = 'workspace'`).Scan(&days, &distinct, &first, &last); err != nil {
		t.Fatal(err)
	}
	if days != 95 || distinct != 95 {
		t.Fatalf("%d workspace rows on %d days, want 95 on 95", days, distinct)
	}
	if want := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC); !last.Equal(want) {
		t.Errorf("the last day is %s, want yesterday in New York, %s", last.Format(time.DateOnly), want.Format(time.DateOnly))
	}
	if gap := last.Sub(first); gap != 94*24*time.Hour {
		t.Errorf("the days span %v, want 94 days with none missing", gap)
	}
}
