package pages_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/pages"
)

// A webinar page's reminder automation: offered while its event details
// name a date field; made once, as a draft flow with a date-field trigger
// the day before at the session's time in the workspace's zone, then an
// hour before, both emails with the joining link.
func TestReminderAutomation(t *testing.T) {
	f := newFixture(t)
	// Fresh values each time: "automation" is left out when there's none.
	offer := func(path string) pages.PageReminders {
		t.Helper()
		var r pages.PageReminders
		f.owner.Do(http.MethodGet, path, nil).MustStatus(t, http.StatusOK).JSON(t, &r)
		return r
	}

	// The demo's webinar has its flow already ("DMARC live reminders", live).
	dmarc := pagesAPI + "/" + f.pageID("dmarc-live") + "/reminder-automation"
	r := offer(dmarc)
	if !r.Available || r.Field != "dmarc_live_at" || r.Title != "Live: Fix your DMARC setup in 30 minutes" || r.Reminders != "1 day and 1 hour before" ||
		r.Flow == nil || r.Flow.Name != "DMARC live reminders" || r.Flow.Status != "live" {
		t.Fatalf("the demo webinar's offer: %+v %+v", r, r.Flow)
	}
	p := mustProblem(t, f.owner.Do(http.MethodPost, dmarc, nil), http.StatusConflict, "reminders_exist")
	if p.Meta["automation"] != r.Flow.ID || !strings.Contains(p.Detail, "DMARC live reminders") {
		t.Errorf("exists: %+v", p)
	}

	// A new webinar: no date field, no offer.
	w := f.create(f.owner, pages.TemplateWebinar, "Live: BIMI in practice", "")
	path := pagesAPI + "/" + w.ID + "/reminder-automation"
	if r := offer(path); r.Available || r.Flow != nil {
		t.Fatalf("a new webinar's event has no date field yet: %+v", r)
	}
	mustProblem(t, f.owner.Do(http.MethodPost, path, nil), http.StatusConflict, "conflict")

	// With one: 17:00 in New York on 10 December.
	f.save(f.owner, w.ID, w.Draft.Version, section("b-hero", `<div class="prose"><h1>Live: BIMI in practice</h1></div>`)+
		section("b-event", `<rc-event data-pb-id="event" date="2026-12-10" start="17:00" end="17:45" tz="America/New_York" where="online" platform="Zoom"`+
			` link="https://acme.zoom.us/j/777" title="Live: BIMI in practice" reminders="1 day and 1 hour before" field="bimi_live_at"></rc-event>`))
	if r := offer(path); !r.Available || r.Field != "bimi_live_at" || r.Title != "Live: BIMI in practice" || r.Flow != nil {
		t.Fatalf("with a date field: %+v", r)
	}
	var made pages.PageReminders
	f.owner.Do(http.MethodPost, path, nil).MustStatus(t, http.StatusCreated).JSON(t, &made)
	if made.Flow == nil || made.Flow.Name != "Live: BIMI in practice · reminders" || made.Flow.Status != "draft" {
		t.Fatalf("made: %+v", made.Flow)
	}
	id := made.Flow.ID
	var trig struct {
		Kind       string `json:"kind"`
		Field      string `json:"field"`
		At         string `json:"at"`
		OffsetDays int    `json:"offsetDays"`
	}
	if err := json.Unmarshal(scan[[]byte](f, `SELECT trigger FROM automations WHERE id = $1`, id), &trig); err != nil {
		t.Fatal(err)
	}
	if trig.Kind != "date_field" || trig.Field != "bimi_live_at" || trig.OffsetDays != -1 || trig.At != "17:00" {
		t.Fatalf("trigger %+v", trig)
	}
	if n := scan[int64](f, `SELECT count(*) FROM automation_emails WHERE automation_id = $1`, id); n != 2 {
		t.Errorf("%d reminder emails", n)
	}
	if n := scan[int64](f, `SELECT count(*) FROM automation_email_versions v JOIN automation_emails e ON e.id = v.email_id
		WHERE e.automation_id = $1 AND (v.plain_text LIKE '%acme.zoom.us/j/777%' OR v.html LIKE '%acme.zoom.us/j/777%')`, id); n != 2 {
		t.Errorf("%d emails carry the joining link", n)
	}
	if n := scan[int64](f, `SELECT count(*) FROM audit_events WHERE action = 'automation.create' AND target_id = $1 AND metadata->>'source' = 'page'`, id); n != 1 {
		t.Errorf("%d audit entries for the automation", n)
	}
	// Only once.
	p = mustProblem(t, f.owner.Do(http.MethodPost, path, nil), http.StatusConflict, "reminders_exist")
	if p.Meta["automation"] != id {
		t.Errorf("exists: %+v", p)
	}
	if r := offer(path); r.Flow == nil || r.Flow.ID != id {
		t.Errorf("the offer after: %+v", r.Flow)
	}
	// Pages without event details have no offer.
	about := pagesAPI + "/" + f.pageID("about") + "/reminder-automation"
	if r := offer(about); r.Available || r.Flow != nil {
		t.Errorf("About's offer: %+v", r)
	}
	mustProblem(t, f.owner.Do(http.MethodPost, about, nil), http.StatusConflict, "conflict")
}
