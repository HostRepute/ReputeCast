package approvals_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// TestReschedule moves a scheduled campaign (the calendar's drag, review
// 2026-11 CA-3): an approved time stays with its approval, the new time is
// in the future, and the move is audited; a local-time schedule starts in
// the earliest zone (CA-5).
func TestReschedule(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	n := counter{t, env}
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var id string
	if err := env.App.Pool.QueryRow(ctx, `SELECT id::text FROM campaigns WHERE name = 'Weekly Digest #142'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	path := "/workspaces/acme-newsletter/campaigns/" + id + "/reschedule"
	future := time.Now().AddDate(0, 0, 10).Format(time.DateOnly)
	p := linh.Do(http.MethodPost, path, map[string]any{"mode": "at", "date": future, "time": "10:30", "timezone": "America/New_York"}).MustStatus(t, http.StatusConflict).Problem(t)
	if p["code"] != "approval_covers_time" {
		t.Fatalf("an approved time: %+v", p)
	}
	// Without an approval in force, it moves.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE approval_requests SET status = 'invalidated' WHERE campaign_id = $1 AND status IN ('pending', 'approved')`, id); err != nil {
		t.Fatal(err)
	}
	linh.Do(http.MethodPost, path, map[string]any{"mode": "at", "date": "2001-01-01", "time": "10:30", "timezone": "America/New_York"}).MustStatus(t, http.StatusUnprocessableEntity)
	linh.Do(http.MethodPost, path, map[string]any{"mode": "now"}).MustStatus(t, http.StatusUnprocessableEntity)
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do(http.MethodPost, path, map[string]any{"mode": "at", "date": future, "time": "10:30", "timezone": "America/New_York"}).MustStatus(t, http.StatusForbidden)

	linh.Do(http.MethodPost, path, map[string]any{"mode": "at", "date": future, "time": "10:30", "timezone": "America/New_York"}).MustStatus(t, http.StatusNoContent)
	ny, _ := time.LoadLocation("America/New_York")
	want, _ := time.ParseInLocation("2006-01-02 15:04", future+" 10:30", ny)
	var status string
	var at time.Time
	if err := env.App.Pool.QueryRow(ctx, `SELECT status, scheduled_at FROM campaigns WHERE id = $1`, id).Scan(&status, &at); err != nil || status != "scheduled" || !at.Equal(want) {
		t.Fatalf("rescheduled: %s %v (%v), want scheduled at %v", status, at, err, want)
	}
	if n.audit("campaign.rescheduled", id) != 1 {
		t.Fatal("the move isn't in the audit log")
	}
	// Local time: the send starts when 10:30 first comes (UTC+14).
	linh.Do(http.MethodPost, path, map[string]any{"mode": "local", "date": future, "time": "10:30", "timezone": "America/New_York"}).MustStatus(t, http.StatusNoContent)
	first, _ := time.ParseInLocation("2006-01-02 15:04", future+" 10:30", time.FixedZone("UTC+14", 14*3600))
	var local string
	if err := env.App.Pool.QueryRow(ctx, `SELECT scheduled_at, schedule->>'local' FROM campaigns WHERE id = $1`, id).Scan(&at, &local); err != nil || !at.Equal(first) || local != future+" 10:30" {
		t.Fatalf("local time: %v %q (%v), want %v", at, local, err, first)
	}
	var rv struct{ Schedule struct{ Mode, Date, Time string } }
	linh.Do(http.MethodGet, "/workspaces/acme-newsletter/campaigns/"+id+"/review", nil).MustStatus(t, 200).JSON(t, &rv)
	if rv.Schedule.Mode != "local" || rv.Schedule.Date != future || rv.Schedule.Time != "10:30" {
		t.Fatalf("Review & send shows %+v", rv.Schedule)
	}
	// A draft has nothing to move.
	var draft string
	if err := env.App.Pool.QueryRow(ctx, `SELECT id::text FROM campaigns WHERE name = 'Flash sale: 48 hours only'`).Scan(&draft); err != nil {
		t.Fatal(err)
	}
	linh.Do(http.MethodPost, "/workspaces/acme-newsletter/campaigns/"+draft+"/reschedule", map[string]any{"mode": "at", "date": future, "time": "10:30", "timezone": "America/New_York"}).
		MustStatus(t, http.StatusConflict)
}
