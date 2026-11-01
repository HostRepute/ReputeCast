package texting_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/texting"
)

// The calendar moves a scheduled SMS campaign (review 2026-11 CA-9): each
// queued text gets its new moment in its recipient's zone, inside quiet
// hours, and the move is audited; once a text went out it can't move.
func TestCalendarMovesTexts(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	sofia := env.Client()
	sofia.SignInDemo("sofia@acme.io")
	list := one[string](t, env, `SELECT id::text FROM lists WHERE name = 'Acme Summit 2026'`)
	var c texting.SmsCampaign
	sofia.Do(http.MethodPost, ws+"/sms-campaigns", map[string]any{"name": "Summit: shuttle", "lists": []string{list},
		"body": "Shuttles leave the station every 15 minutes."}).MustStatus(t, 201).JSON(t, &c)
	day := time.Now().AddDate(0, 0, 10)
	future := day.Format(time.DateOnly)
	sendAt := time.Date(day.Year(), day.Month(), day.Day(), 15, 0, 0, 0, time.UTC)
	var rv texting.SmsReview
	sofia.Do(http.MethodPut, ws+"/sms-campaigns/"+c.ID+"/send-settings", map[string]any{"mode": "at", "sendAt": sendAt}).MustStatus(t, 200).JSON(t, &rv)
	acks := []string{}
	for _, ch := range rv.Checks {
		if ch.Status == "warn" {
			acks = append(acks, ch.Key)
		}
	}
	var res texting.SmsSendResult
	sofia.Do(http.MethodPost, ws+"/sms-campaigns/"+c.ID+"/send", map[string]any{"fingerprint": rv.Fingerprint, "acknowledged": acks}).MustStatus(t, 200).JSON(t, &res)
	if res.Status != "scheduled" {
		t.Fatalf("send: %+v", res)
	}
	queued := one[int64](t, env, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1 AND status = 'queued'`, c.ID)
	if queued == 0 {
		t.Fatal("nothing queued")
	}
	move := "/workspaces/acme-newsletter/campaigns/" + c.ID + "/reschedule"
	type text struct {
		zone string
		at   time.Time
	}
	texts := func() []text {
		t.Helper()
		rows, err := env.App.Pool.Query(ctx, `SELECT time_zone, send_after FROM channel_messages WHERE campaign_id = $1 AND status = 'queued'`, c.ID)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []text
		for rows.Next() {
			var x text
			if err := rows.Scan(&x.zone, &x.at); err != nil {
				t.Fatal(err)
			}
			out = append(out, x)
		}
		if int64(len(out)) != queued {
			t.Fatalf("queued texts: %d, were %d", len(out), queued)
		}
		return out
	}
	zone := func(z string) *time.Location {
		l, err := time.LoadLocation(z)
		if err != nil || z == "" {
			return time.UTC
		}
		return l
	}
	scheduledAt := func() time.Time {
		return one[time.Time](t, env, `SELECT scheduled_at FROM campaigns WHERE id = $1`, c.ID)
	}

	// 10:30 in each recipient's zone.
	sofia.Do(http.MethodPost, move, map[string]any{"mode": "local", "date": future, "time": "10:30", "timezone": "America/Chicago"}).MustStatus(t, http.StatusNoContent)
	first := time.Time{}
	zones := map[string]bool{}
	for _, x := range texts() {
		local := x.at.In(zone(x.zone))
		if local.Format("2006-01-02 15:04") != future+" 10:30" {
			t.Fatalf("a text in %q goes at %s", x.zone, local)
		}
		if first.IsZero() || x.at.Before(first) {
			first = x.at
		}
		zones[x.zone] = true
	}
	if len(zones) < 2 {
		t.Fatalf("the summit list should span zones: %v", zones)
	}
	if at := scheduledAt(); !at.Equal(first) {
		t.Fatalf("the campaign starts at %v, its first text at %v", at, first)
	}
	if m := one[string](t, env, `SELECT schedule_mode || ' ' || local_minute FROM sms_campaigns WHERE campaign_id = $1`, c.ID); m != "local 630" {
		t.Fatalf("schedule: %s", m)
	}
	var cal struct {
		Items []struct {
			ID, Channel, Kind, LocalTime string
			Local                        bool
		}
	}
	q := "?from=" + day.AddDate(0, 0, -1).Format(time.DateOnly) + "&to=" + day.AddDate(0, 0, 2).Format(time.DateOnly)
	sofia.Do(http.MethodGet, ws+"/campaigns/calendar"+q, nil).MustStatus(t, 200).JSON(t, &cal)
	found := false
	for _, it := range cal.Items {
		if it.ID == c.ID {
			found = it.Channel == "sms" && it.Kind == "scheduled" && it.Local && it.LocalTime == "10:30"
		}
	}
	if !found {
		t.Fatalf("the calendar doesn't show the text at 10:30 local: %+v", cal.Items)
	}

	// 07:00 in Chicago: quiet hours hold Chicago's texts until 09:00 there;
	// Berlin's go at that instant (14:00 there).
	sofia.Do(http.MethodPost, move, map[string]any{"mode": "at", "date": future, "time": "07:00", "timezone": "America/Chicago"}).MustStatus(t, http.StatusNoContent)
	chicago := zone("America/Chicago")
	instant := time.Date(day.Year(), day.Month(), day.Day(), 7, 0, 0, 0, chicago)
	for _, x := range texts() {
		switch x.zone {
		case "America/Chicago":
			if got := x.at.In(chicago).Format("15:04"); got != "09:00" {
				t.Fatalf("a Chicago text at %s", got)
			}
		case "Europe/Berlin":
			if !x.at.Equal(instant) {
				t.Fatalf("a Berlin text at %v, want %v", x.at, instant)
			}
		}
		if x.at.Before(instant) {
			t.Fatalf("a text in %q before the new time: %v", x.zone, x.at)
		}
	}
	if n := one[int64](t, env, `SELECT count(*) FROM audit_events WHERE action = 'campaign.rescheduled' AND target_id = $1 AND metadata->>'channel' = 'sms'`, c.ID); n != 2 {
		t.Fatalf("audited moves: %d", n)
	}

	// Outside quiet hours, by a viewer, and after a text went out: refused.
	sofia.Do(http.MethodPost, move, map[string]any{"mode": "local", "date": future, "time": "21:30", "timezone": "America/Chicago"}).MustStatus(t, http.StatusUnprocessableEntity)
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do(http.MethodPost, move, map[string]any{"mode": "local", "date": future, "time": "10:30", "timezone": "America/Chicago"}).MustStatus(t, http.StatusForbidden)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE channel_messages SET status = 'sent', sent_at = now() WHERE id = (SELECT id FROM channel_messages WHERE campaign_id = $1 AND status = 'queued' LIMIT 1)`, c.ID); err != nil {
		t.Fatal(err)
	}
	sofia.Do(http.MethodPost, move, map[string]any{"mode": "local", "date": future, "time": "11:00", "timezone": "America/Chicago"}).MustStatus(t, http.StatusConflict)
}
