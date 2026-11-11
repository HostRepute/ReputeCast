package db_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/testdb"
)

// A table in monthly partitions keeps its coming months, adopts the rows
// its default partition holds for a month it makes, and drops whole months
// once they're past a cutoff (tracker_events, M62).
func TestMonthlyPartitions(t *testing.T) {
	ctx := context.Background()
	pool := testdb.New(t)
	tbl := db.MonthlyTable{Name: "tracker_events", Key: "occurred_at", Ahead: 2, Adopt: true}
	now := time.Now().UTC()
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	names := func() []string {
		t.Helper()
		parts, err := db.MonthlyPartitions(ctx, pool, tbl)
		if err != nil {
			t.Fatal(err)
		}
		out := make([]string, 0, len(parts))
		for i, p := range parts {
			if !p.Till.Equal(p.From.AddDate(0, 1, 0)) || (i > 0 && !p.From.After(parts[i-1].From)) {
				t.Fatalf("partitions out of order: %+v", parts)
			}
			out = append(out, p.Name)
		}
		return out
	}
	name := func(m time.Time) string { return "tracker_events_" + m.Format("2006_01") }
	// The migration made the months around the one it ran in; whichever
	// those were, making the three before this one through two ahead
	// leaves all six, and a second pass makes nothing.
	if _, err := db.EnsureMonthlyPartitions(ctx, pool, tbl, month.AddDate(0, -3, 0), time.Time{}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.EnsureMonthlyPartitions(ctx, pool, tbl, now, time.Time{}); err != nil {
		t.Fatal(err)
	}
	for i := -3; i <= 2; i++ {
		if !slices.Contains(names(), name(month.AddDate(0, i, 0))) {
			t.Fatalf("no partition for month %d: %v", i, names())
		}
	}
	if made, err := db.EnsureMonthlyPartitions(ctx, pool, tbl, now, time.Time{}); err != nil || len(made) != 0 {
		t.Fatalf("nothing to make: %v %v", made, err)
	}

	// A row far ahead lands in the default partition; making its month
	// moves it there.
	ws := uuid.Must(uuid.NewV7())
	if _, err := pool.Exec(ctx, `INSERT INTO workspaces (id, slug, name) VALUES ($1, 'monthly-test', 'Monthly')`, ws); err != nil {
		t.Fatal(err)
	}
	later := month.AddDate(0, 6, 0)
	id := uuid.Must(uuid.NewV7())
	if _, err := pool.Exec(ctx, `INSERT INTO tracker_events (id, workspace_id, site_id, visitor_id, kind, occurred_at) VALUES ($1, $2, $3, $4, 1, $5)`,
		id, ws, uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), later.Add(36*time.Hour)); err != nil {
		t.Fatal(err)
	}
	var where string
	if err := pool.QueryRow(ctx, `SELECT tableoid::regclass::text FROM tracker_events WHERE id = $1`, id).Scan(&where); err != nil || where != "tracker_events_default" {
		t.Fatalf("row in %q (%v)", where, err)
	}
	made, err := db.EnsureMonthlyPartitions(ctx, pool, tbl, later, time.Time{})
	if err != nil || !slices.Contains(made, name(later)) {
		t.Fatalf("made %v: %v", made, err)
	}
	if err := pool.QueryRow(ctx, `SELECT tableoid::regclass::text FROM tracker_events WHERE id = $1`, id).Scan(&where); err != nil || where != name(later) {
		t.Fatalf("the default's row is in %q (%v)", where, err)
	}
	var left int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tracker_events_default`).Scan(&left); err != nil || left != 0 {
		t.Fatalf("default keeps %d rows (%v)", left, err)
	}

	// Whole months that end by the cutoff go; the month the cutoff falls in
	// stays.
	cut := month.AddDate(0, -1, 10)
	dropped, err := db.DropMonthlyPartitions(ctx, pool, tbl, cut)
	if err != nil || !slices.Contains(dropped, name(month.AddDate(0, -3, 0))) || !slices.Contains(dropped, name(month.AddDate(0, -2, 0))) {
		t.Fatalf("dropped %v: %v", dropped, err)
	}
	parts, err := db.MonthlyPartitions(ctx, pool, tbl)
	if err != nil || len(parts) == 0 || parts[0].Name != name(month.AddDate(0, -1, 0)) {
		t.Fatalf("oldest after the drop: %+v %v", parts, err)
	}
}
