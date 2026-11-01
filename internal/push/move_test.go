package push_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/push"
)

// The calendar moves a scheduled push campaign that hasn't started
// (review 2026-11 CA-9); a time in each recipient's zone is refused, push
// going out at one time for everyone.
func TestCalendarMovesPush(t *testing.T) {
	f := newFixture(t)
	env := f.env
	ctx := context.Background()
	f.pushOn(t, 5)
	f.subscribe(t, "chrome", chromeUA, nil)
	var c push.PushCampaign
	f.owner.Do(http.MethodPost, f.path+"/push-campaigns", map[string]any{"name": "X22 later", "title": "Later",
		"message": "How we ship to 2% of accounts first.", "link": "help.x22.test/blog/flags"}).MustStatus(t, 201).JSON(t, &c)
	day := time.Now().AddDate(0, 0, 10)
	sendAt := time.Date(day.Year(), day.Month(), day.Day(), 15, 0, 0, 0, time.UTC)
	var rv push.PushReview
	f.owner.Do(http.MethodPut, f.path+"/push-campaigns/"+c.ID+"/send-settings", map[string]any{"mode": "at", "sendAt": sendAt, "outside": "hold"}).
		MustStatus(t, 200).JSON(t, &rv)
	var res push.PushSendResult
	f.owner.Do(http.MethodPost, f.path+"/push-campaigns/"+c.ID+"/send", map[string]any{"version": rv.Version}).MustStatus(t, 200).JSON(t, &res)
	if res.Status != "scheduled" || !res.ScheduledAt.Equal(sendAt) {
		t.Fatalf("send: %+v", res)
	}
	move := f.path + "/campaigns/" + c.ID + "/reschedule"
	future := day.AddDate(0, 0, 1).Format(time.DateOnly)
	f.owner.Do(http.MethodPost, move, map[string]any{"mode": "local", "date": future, "time": "10:30", "timezone": "UTC"}).MustStatus(t, http.StatusUnprocessableEntity)
	f.owner.Do(http.MethodPost, move, map[string]any{"mode": "at", "date": future, "time": "10:30", "timezone": "UTC"}).MustStatus(t, http.StatusNoContent)
	want := time.Date(day.Year(), day.Month(), day.Day()+1, 10, 30, 0, 0, time.UTC)
	id := uuid.MustParse(c.ID)
	if at := one[time.Time](t, env, `SELECT scheduled_at FROM push_sends WHERE campaign_id = $1`, id); !at.Equal(want) {
		t.Fatalf("the send starts at %v, want %v", at, want)
	}
	if at := one[time.Time](t, env, `SELECT scheduled_at FROM campaigns WHERE id = $1`, id); !at.Equal(want) {
		t.Fatalf("the campaign shows %v", at)
	}
	if at := one[time.Time](t, env, `SELECT send_at FROM push_campaigns WHERE campaign_id = $1`, id); !at.Equal(want) {
		t.Fatalf("Review & send shows %v", at)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM audit_events WHERE action = 'campaign.rescheduled' AND target_id = $1 AND metadata->>'channel' = 'push'`, c.ID); n != 1 {
		t.Fatalf("audited moves: %d", n)
	}
	// Nothing went before its new time; once started it can't move.
	f.run(t)
	if n := one[int64](t, env, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1`, id); n != 0 {
		t.Fatalf("%d notifications before the new time", n)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE push_sends SET phase = 'sending' WHERE campaign_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	f.owner.Do(http.MethodPost, move, map[string]any{"mode": "at", "date": future, "time": "11:30", "timezone": "UTC"}).MustStatus(t, http.StatusConflict)
}
