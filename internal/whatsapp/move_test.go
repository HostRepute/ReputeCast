package whatsapp_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/whatsapp"
)

// The calendar moves a scheduled WhatsApp campaign that hasn't started
// (review 2026-11 CA-9): to an instant, or to a time in each recipient's
// zone, when the send starts as that time first comes anywhere.
func TestCalendarMovesWhatsApp(t *testing.T) {
	f := newFixture(t)
	env, c, p := f.env, f.owner, f.path
	ctx := context.Background()
	f.connect(t)
	list, ids := f.people(t)
	c.Do(http.MethodPut, p+"/whatsapp/settings", map[string]any{"quietHours": false}).MustStatus(t, 200)
	for _, k := range []string{"sam", "ana"} {
		c.Do(http.MethodPut, p+"/contacts/"+ids[k].String()+"/whatsapp/consent", map[string]any{"listId": list, "status": "subscribed", "evidence": evidence()}).
			MustStatus(t, 200)
	}
	var ov whatsapp.WhatsAppOverview
	c.Do(http.MethodGet, p+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &ov)
	var tpl string
	for _, tp := range ov.Templates {
		if tp.Name == "summit_day_before" {
			tpl = tp.ID
		}
	}
	var camp whatsapp.WhatsAppCampaign
	c.Do(http.MethodPost, p+"/whatsapp-campaigns", map[string]any{"name": "X22 later", "templateId": tpl, "lists": []string{list.String()}}).
		MustStatus(t, 201).JSON(t, &camp)
	day := time.Now().AddDate(0, 0, 10)
	sendAt := time.Date(day.Year(), day.Month(), day.Day(), 15, 0, 0, 0, time.UTC)
	c.Do(http.MethodPatch, p+"/whatsapp-campaigns/"+camp.ID, map[string]any{"name": camp.Name, "templateId": tpl, "lists": []string{list.String()}, "mode": "at",
		"sendAt": sendAt, "version": camp.Version,
		"variables": []map[string]any{{"key": "1", "source": "field", "field": "first_name", "fallback": "there"}, {"key": "2", "source": "text", "text": "8:30"}}}).
		MustStatus(t, 200).JSON(t, &camp)
	var rv whatsapp.WhatsAppReview
	c.Do(http.MethodGet, p+"/whatsapp-campaigns/"+camp.ID+"/review", nil).MustStatus(t, 200).JSON(t, &rv)
	var res whatsapp.WhatsAppSendResult
	c.Do(http.MethodPost, p+"/whatsapp-campaigns/"+camp.ID+"/send", map[string]any{"version": rv.Version}).MustStatus(t, 200).JSON(t, &res)
	if res.Status != "scheduled" || !res.ScheduledAt.Equal(sendAt) {
		t.Fatalf("send: %+v", res)
	}
	id := uuid.MustParse(camp.ID)
	move := p + "/campaigns/" + camp.ID + "/reschedule"
	future := day.AddDate(0, 0, 1)
	date := future.Format(time.DateOnly)

	// 09:15 in each recipient's zone: the send starts when 09:15 first comes (UTC+14).
	c.Do(http.MethodPost, move, map[string]any{"mode": "local", "date": date, "time": "09:15", "timezone": "America/Chicago"}).MustStatus(t, http.StatusNoContent)
	first := time.Date(future.Year(), future.Month(), future.Day(), 9, 15, 0, 0, time.FixedZone("UTC+14", 14*3600))
	if at := one[time.Time](t, env, `SELECT scheduled_at FROM whatsapp_sends WHERE campaign_id = $1`, id); !at.Equal(first) {
		t.Fatalf("the send starts at %v, want %v", at, first)
	}
	if m := one[string](t, env, `SELECT schedule_mode || ' ' || local_date::text || ' ' || local_minute FROM whatsapp_campaigns WHERE campaign_id = $1`, id); m != "local "+date+" 555" {
		t.Fatalf("schedule: %s", m)
	}
	var cal struct {
		Items []struct {
			ID, LocalTime string
			Local         bool
		}
	}
	q := "?from=" + day.Format(time.DateOnly) + "&to=" + day.AddDate(0, 0, 3).Format(time.DateOnly)
	c.Do(http.MethodGet, p+"/campaigns/calendar"+q, nil).MustStatus(t, 200).JSON(t, &cal)
	shown := false
	for _, it := range cal.Items {
		shown = shown || (it.ID == camp.ID && it.Local && it.LocalTime == "09:15")
	}
	if !shown {
		t.Fatalf("calendar: %+v", cal.Items)
	}

	// An instant.
	c.Do(http.MethodPost, move, map[string]any{"mode": "at", "date": date, "time": "16:45", "timezone": "UTC"}).MustStatus(t, http.StatusNoContent)
	want := time.Date(future.Year(), future.Month(), future.Day(), 16, 45, 0, 0, time.UTC)
	if at := one[time.Time](t, env, `SELECT scheduled_at FROM whatsapp_sends WHERE campaign_id = $1`, id); !at.Equal(want) {
		t.Fatalf("the send starts at %v, want %v", at, want)
	}
	if m := one[string](t, env, `SELECT schedule_mode || ' ' || (local_date IS NULL)::text FROM whatsapp_campaigns WHERE campaign_id = $1`, id); m != "at true" {
		t.Fatalf("schedule: %s", m)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM audit_events WHERE action = 'campaign.rescheduled' AND target_id = $1 AND metadata->>'channel' = 'whatsapp'`, camp.ID); n != 2 {
		t.Fatalf("audited moves: %d", n)
	}
	// Nothing went before its time; once started it can't move.
	f.run(t)
	if n := len(f.fake.SentMessages()); n != 0 {
		t.Fatalf("%d messages before the new time", n)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE whatsapp_sends SET phase = 'sending' WHERE campaign_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	c.Do(http.MethodPost, move, map[string]any{"mode": "at", "date": date, "time": "17:45", "timezone": "UTC"}).MustStatus(t, http.StatusConflict)
}
