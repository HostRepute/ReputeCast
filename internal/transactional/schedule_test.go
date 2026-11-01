package transactional_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

// TestCancelAndRescheduleScheduled: a message sent with send_at waits until
// its time and, until then, can be moved or cancelled by its key or a
// person; the log shows Scheduled and Cancelled, webhooks get
// message.cancelled and both changes are audited (review 2026-11 TX-1).
func TestCancelAndRescheduleScheduled(t *testing.T) {
	s := newSendEnv(t)
	s.webhook()
	_, secret := s.key("Production", map[string]any{"preset": "send_only"})
	at := s.clock.now().Add(2 * time.Hour)
	b := resetBody("sam.rivera@outlook.com")
	b["send_at"] = at.Format(time.RFC3339)
	var res sendResult
	s.call(secret, "/messages", "", b).MustStatus(t, 202).JSON(t, &res)

	var log struct {
		Items  []struct{ ID, Status string }
		Counts struct{ Scheduled, Cancelled int64 }
	}
	s.owner.Do("GET", base+"/messages?status=scheduled", nil).MustStatus(t, 200).JSON(t, &log)
	if len(log.Items) != 1 || log.Items[0].ID != res.ID || log.Counts.Scheduled != 1 {
		t.Fatalf("scheduled in the log: %+v", log)
	}

	// Moved by the key: an hour later.
	later := at.Add(time.Hour).UTC()
	var moved struct {
		Status string
		SendAt *time.Time
	}
	s.call(secret, "/messages/"+res.ID+"/reschedule", "", map[string]any{"sendAt": later.Format(time.RFC3339)}).MustStatus(t, 200).JSON(t, &moved)
	if moved.Status != "scheduled" || moved.SendAt == nil || !moved.SendAt.Equal(later) {
		t.Fatalf("rescheduled: %+v", moved)
	}
	s.call(secret, "/messages/"+res.ID+"/reschedule", "", map[string]any{"sendAt": s.clock.now().Add(73 * time.Hour).Format(time.RFC3339)}).MustStatus(t, 422)
	s.call(secret, "/messages/"+res.ID+"/reschedule", "", map[string]any{"sendAt": s.clock.now().Add(-time.Hour).Format(time.RFC3339)}).MustStatus(t, 422)
	s.clock.add(2*time.Hour + time.Second)
	if n := s.step(); n != 0 {
		t.Fatalf("sent at the old time: %d", n)
	}

	// Cancelled by a person.
	var done struct{ Status string }
	s.owner.Do("POST", base+"/messages/"+res.ID+"/cancel", nil).MustStatus(t, 200).JSON(t, &done)
	if done.Status != "cancelled" {
		t.Fatalf("cancel: %+v", done)
	}
	p := s.owner.Do("POST", base+"/messages/"+res.ID+"/cancel", nil).MustStatus(t, http.StatusConflict).Problem(t)
	if p["code"] != "not_scheduled" {
		t.Fatalf("twice: %+v", p)
	}
	s.clock.add(2 * time.Hour)
	if n := s.step(); n != 0 || s.status(res.ID) != "cancelled" {
		t.Fatalf("a cancelled message went: %d %s", n, s.status(res.ID))
	}
	var outbox int
	s.one(`SELECT count(*) FROM transactional_outbox WHERE message_id = $1`, &outbox, res.ID)
	if outbox != 0 {
		t.Fatal("the sealed payload stays after a cancel")
	}
	var content int
	var removed bool
	s.one(`SELECT count(*) FROM transactional_message_content WHERE message_id = $1`, &content, res.ID)
	s.one(`SELECT content_removed_at IS NOT NULL FROM transactional_messages WHERE message_id = $1`, &removed, res.ID)
	if content != 0 || !removed {
		t.Fatalf("the rendered content after a cancel: %d rows, removed %v", content, removed)
	}
	if ev := s.events(res.ID); !slices.Contains(ev, "message.cancelled") {
		t.Fatalf("webhook events: %v", ev)
	}
	var detail struct {
		Status   string
		Timeline []struct{ Kind, Detail string }
	}
	s.owner.Do("GET", base+"/messages/"+res.ID, nil).MustStatus(t, 200).JSON(t, &detail)
	last := detail.Timeline[len(detail.Timeline)-1]
	if detail.Status != "cancelled" || last.Kind != "cancelled" || last.Detail == "" {
		t.Fatalf("detail: %+v", detail)
	}
	s.owner.Do("GET", base+"/messages?status=cancelled", nil).MustStatus(t, 200).JSON(t, &log)
	if len(log.Items) != 1 || log.Counts.Cancelled != 1 {
		t.Fatalf("cancelled in the log: %+v", log)
	}
	var audits int
	s.one(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action IN ('transactional.message_cancelled', 'transactional.message_rescheduled')`, &audits, res.ID)
	if audits != 2 {
		t.Fatalf("audited %d changes, want 2", audits)
	}

	// A message that went at once can't be cancelled; an editor without
	// transactional:send can't cancel at all.
	var now sendResult
	s.call(secret, "/messages", "", resetBody("lee@outlook.com")).MustStatus(t, 202).JSON(t, &now)
	if p := s.owner.Do("POST", base+"/messages/"+now.ID+"/cancel", nil).MustStatus(t, http.StatusConflict).Problem(t); !strings.Contains(p["detail"].(string), "waiting") {
		t.Fatalf("not scheduled: %+v", p)
	}
	s.sofia.Do("POST", base+"/messages/"+now.ID+"/cancel", nil).MustStatus(t, http.StatusForbidden)
}

// TestLogLivePoll: asking only for newer messages answers them alone, with
// the status counts kept for a minute (review 2026-11 TX-3).
func TestLogLivePoll(t *testing.T) {
	s := newSendEnv(t)
	_, secret := s.key("Production", map[string]any{"preset": "send_only"})
	var first sendResult
	s.call(secret, "/messages", "", resetBody("a@contoso.example")).MustStatus(t, 202).JSON(t, &first)
	var page struct {
		Items   []struct{ ID string }
		Newest  string
		Filters struct{ Templates []struct{ ID string } }
		Counts  struct{ All int64 }
	}
	s.owner.Do("GET", base+"/messages?limit=10", nil).MustStatus(t, 200).JSON(t, &page)
	if len(page.Items) == 0 || page.Newest == "" || len(page.Filters.Templates) == 0 {
		t.Fatalf("page: %+v", page)
	}
	all := page.Counts.All
	var second sendResult
	s.call(secret, "/messages", "", resetBody("b@contoso.example")).MustStatus(t, 202).JSON(t, &second)
	var live struct {
		Items   []struct{ ID string }
		Filters struct{ Templates []struct{ ID string } }
		Counts  struct{ All int64 }
	}
	s.owner.Do("GET", base+"/messages?limit=100&after="+page.Newest, nil).MustStatus(t, 200).JSON(t, &live)
	if len(live.Items) != 1 || live.Items[0].ID != second.ID || len(live.Filters.Templates) != 0 {
		t.Fatalf("live poll: %+v", live)
	}
	if live.Counts.All != all {
		t.Fatalf("the poll recounted: %d, cached %d", live.Counts.All, all)
	}
}

// Review 2026-11 TX-7: a batch scheduled with one send_at is moved and
// cancelled together by its Idempotency-Key; a message already gone is
// skipped, and an unknown key is a 404.
func TestBatchMovedAndCancelledTogether(t *testing.T) {
	s := newSendEnv(t)
	_, secret := s.key("Production", map[string]any{"preset": "send_only"})
	at := s.clock.now().Add(2 * time.Hour)
	msgs := []any{}
	for _, to := range []string{"a@contoso.example", "b@contoso.example", "c@contoso.example"} {
		b := resetBody(to)
		b["send_at"] = at.Format(time.RFC3339)
		msgs = append(msgs, b)
	}
	var out struct {
		Results []struct{ ID string }
	}
	s.call(secret, "/messages/batch", "spring-batch", map[string]any{"messages": msgs}).MustStatus(t, 200).JSON(t, &out)
	if len(out.Results) != 3 {
		t.Fatalf("batch: %+v", out)
	}
	// One of them is cancelled on its own first.
	s.owner.Do("POST", base+"/messages/"+out.Results[0].ID+"/cancel", nil).MustStatus(t, 200)

	later := at.Add(time.Hour).UTC()
	var moved struct {
		Matched, Changed, Skipped int
		SendAt                    *time.Time
	}
	s.call(secret, "/batches/reschedule", "", map[string]any{"idempotencyKey": "spring-batch", "sendAt": later.Format(time.RFC3339)}).MustStatus(t, 200).JSON(t, &moved)
	if moved.Matched != 3 || moved.Changed != 2 || moved.Skipped != 1 || moved.SendAt == nil || !moved.SendAt.Equal(later) {
		t.Fatalf("moved: %+v", moved)
	}
	s.call(secret, "/batches/reschedule", "", map[string]any{"idempotencyKey": "spring-batch", "sendAt": s.clock.now().Add(73 * time.Hour).Format(time.RFC3339)}).MustStatus(t, 422)
	s.clock.add(2*time.Hour + time.Second)
	if n := s.step(); n != 0 {
		t.Fatalf("sent at the old time: %d", n)
	}

	var cancelled struct{ Matched, Changed, Skipped int }
	s.owner.Do("POST", base+"/batches/cancel", map[string]any{"idempotencyKey": "spring-batch"}).MustStatus(t, 200).JSON(t, &cancelled)
	if cancelled.Matched != 3 || cancelled.Changed != 2 || cancelled.Skipped != 1 {
		t.Fatalf("cancelled: %+v", cancelled)
	}
	for _, r := range out.Results {
		if st := s.status(r.ID); st != "cancelled" {
			t.Fatalf("%s is %s", r.ID, st)
		}
	}
	var audited int
	s.one(`SELECT count(*) FROM audit_events WHERE action IN ('transactional.message_rescheduled', 'transactional.message_cancelled')`, &audited)
	if audited != 5 {
		t.Fatalf("audit entries: %d, want 2 moves and 3 cancels", audited)
	}
	p := s.owner.Do("POST", base+"/batches/cancel", map[string]any{"idempotencyKey": "no-such-batch"}).MustStatus(t, 404).Problem(t)
	if p["code"] != "not_found" {
		t.Fatalf("unknown key: %+v", p)
	}
}
