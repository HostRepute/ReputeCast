package publicpages_test

import (
	"encoding/json"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/pages"
)

var icsLink = regexp.MustCompile(`href="([^"]+/event\.ics\?t=[^"]+)"`)

// spSignup is a page form's post, its timing token a minute old.
func spSignup(e *spEnv, form, publicID, block, email string) url.Values {
	return url.Values{"_t": {formToken(e.Env, publicID, time.Minute)}, "_form": {form}, "_block": {block}, "email": {email},
		"first_name": {"Nadia"}, "consent": {"1"}, "website": {""}}
}

func spPost(t *testing.T, e *spEnv, slug string, v url.Values) resp {
	t.Helper()
	return do(t, e.Env, http.MethodPost, spSite+slug+"/form", v, hdr{"Origin": e.Server.URL})
}

// A page's signup form posts to the page itself, through the form's own
// pipeline: a webinar registration answers "You're registered" with Add to
// calendar, adds the registrant to the session's list, counts for the page
// and writes the session's date to the date field the reminders use; a lead
// magnet's double opt-in waits for the click; a failed signup comes back to
// the page with its error in place.
func TestPageSignupForms(t *testing.T) {
	e := spDemo(t)
	dmarcForm := e.formID("frm_e7b302")
	dmarcPage := e.pageID("dmarc-live")
	page := get(t, e.Env, spSite+"dmarc-live")
	must(t, page, http.StatusOK, `action="`+e.Server.URL+spSite+`dmarc-live/form"`, `name="_form" value="`+dmarcForm+`"`, `name="_block" value="register"`,
		"Save your seat", `name="_t" value="`)

	const email = "nadia.haddad@fastmail.com"
	ev, start := e.event()
	registrations := `SELECT count(*) FROM page_conversions WHERE page_id = $1 AND kind = 'registration' AND source = 'page'`
	before := count(t, e.Env, registrations, dmarcPage)
	reg := spPost(t, e, "dmarc-live", spSignup(e, dmarcForm, "frm_e7b302", "register", email))
	contact, _ := contactID(t, e.Env, e.ws.ID, email)

	t.Run("a webinar registration", func(t *testing.T) {
		must(t, reg, http.StatusOK, "You're registered", "Live: Fix your DMARC setup in 30 minutes", start.Format("2 January 2006")+" · 17:00–17:30 UTC",
			"We emailed the joining link to "+email+". Reminders come 1 day and 1 hour before.", "Add to calendar", "Google Calendar",
			"https://calendar.google.com/calendar/render?")
		if cc := reg.Header.Get("Cache-Control"); !strings.Contains(cc, "no-store") {
			t.Errorf("the registered page is cached: %q", cc)
		}
		if contact == uuid.Nil {
			t.Fatal("no contact for the registrant")
		}
		list := one[uuid.UUID](t, e.Env, `SELECT id FROM lists WHERE workspace_id = $1 AND name LIKE 'DMARC live · %' AND deleted_at IS NULL`, e.ws.ID)
		if st := membership(t, e.Env, contact, list); st != "subscribed" {
			t.Errorf("the session's list: %s", st)
		}
		if st := membership(t, e.Env, contact, e.ids.digest); st != "none" {
			t.Errorf("a registrant joined the Weekly Digest: %s", st)
		}
		if n := count(t, e.Env, registrations, dmarcPage); n != before+1 {
			t.Errorf("registrations recorded for the page: %d → %d", before, n)
		}
	})

	t.Run("the registrant's conversion, date field and calendar file", func(t *testing.T) {
		if n := count(t, e.Env, `SELECT count(*) FROM page_conversions WHERE page_id = $1 AND kind = 'registration' AND contact_id = $2`, dmarcPage, contact); n != 1 {
			t.Errorf("%d registrations recorded for the registrant", n)
		}
		if d := one[string](t, e.Env, `SELECT coalesce(attributes->>'dmarc_live_at', '') FROM contacts WHERE id = $1`, contact); d != ev.Date {
			t.Errorf("the date field %q, want %s", d, ev.Date)
		}
		// The calendar file: signed for the registrant, with the joining link.
		m := icsLink.FindStringSubmatch(reg.Body)
		if m == nil {
			t.Fatalf("no calendar file link:\n%s", clip(reg.Body))
		}
		u, err := url.Parse(html.UnescapeString(m[1]))
		if err != nil {
			t.Fatal(err)
		}
		ics := get(t, e.Env, u.RequestURI())
		stamp := func(at time.Time) string { return at.UTC().Format("20060102T150405Z") }
		must(t, ics, http.StatusOK, "BEGIN:VCALENDAR", "DTSTART:"+stamp(start), "DTEND:"+stamp(start.Add(30*time.Minute)), "SUMMARY:Live: Fix your DMARC setup in 30 minutes",
			"URL:https://acme.zoom.us/j/88412093371", "UID:"+dmarcPage+"-"+contact.String()+"@reputecast")
		if ics.Header.Get("Content-Type") != "text/calendar; charset=utf-8" || ics.Header.Get("Content-Disposition") != `attachment; filename="dmarc-live.ics"` ||
			!strings.Contains(ics.Header.Get("Cache-Control"), "no-store") {
			t.Errorf("calendar headers %v", ics.Header)
		}
		// The token opens this page's file only.
		must(t, get(t, e.Env, spSite+"about/event.ics?"+u.RawQuery), http.StatusNotFound)
	})

	t.Run("a calendar file needs its signed link", func(t *testing.T) {
		must(t, get(t, e.Env, spSite+"dmarc-live/event.ics"), http.StatusNotFound)
		must(t, get(t, e.Env, spSite+"dmarc-live/event.ics?t=forged"), http.StatusNotFound)
	})

	t.Run("a lead magnet's double opt-in", func(t *testing.T) {
		const email = "omar.farouk@fastmail.com"
		dns := e.pageID("dns-checklist")
		before := count(t, e.Env, `SELECT count(*) FROM page_conversions WHERE page_id = $1 AND kind = 'signup'`, dns)
		r := spPost(t, e, "dns-checklist", spSignup(e, e.formID("frm_d4c9a1"), "frm_d4c9a1", "form", email))
		must(t, r, http.StatusOK, "Check your inbox", email)
		contact, ok := contactID(t, e.Env, e.ws.ID, email)
		if !ok {
			t.Fatal("no contact")
		}
		if st := membership(t, e.Env, contact, e.ids.digest); st != "pending" {
			t.Errorf("Weekly Digest is %s until the click", st)
		}
		if n := count(t, e.Env, `SELECT count(*) FROM page_conversions WHERE page_id = $1 AND kind = 'signup'`, dns); n != before+1 {
			t.Errorf("signups recorded for the page: %d → %d", before, n)
		}
		if sent := e.mail.to(email); len(sent) != 1 || sent[0].Kind != "double_opt_in" {
			t.Errorf("confirmation email: %+v", sent)
		}
	})

	t.Run("only the page's own forms, on live pages", func(t *testing.T) {
		must(t, spPost(t, e, "dmarc-live", spSignup(e, e.ids.signup.String(), "frm_8c2e1a", "register", "x.y@fastmail.com")), http.StatusNotFound)
		must(t, spPost(t, e, "sponsor", spSignup(e, dmarcForm, "frm_e7b302", "register", "x.y@fastmail.com")), http.StatusNotFound)
		if _, ok := contactID(t, e.Env, e.ws.ID, "x.y@fastmail.com"); ok {
			t.Error("a refused post made a contact")
		}
	})

	t.Run("a failed signup comes back to the page", func(t *testing.T) {
		r := spPost(t, e, "dmarc-live", spSignup(e, dmarcForm, "frm_e7b302", "register", "not-an-address"))
		must(t, r, http.StatusUnprocessableEntity, "Live: Fix your DMARC setup in 30 minutes", `aria-invalid="true"`, `role="alert"`, `value="not-an-address"`)
		if !strings.Contains(r.Header.Get("Cache-Control"), "no-store") {
			t.Errorf("a page with a reader's answers is cached: %q", r.Header.Get("Cache-Control"))
		}
	})
}

// A webinar's reminders go a day and an hour before the session, also when
// the session's date in its own zone isn't the workspace's date: 01:00 UTC
// on 3 December is 20:00 on 2 December in New York. Registrations write the
// date field; the flow's date-field trigger fires on that date plus its
// offset, at its time in the workspace's zone (automations'
// fireDateTriggers).
func TestWebinarRemindersAroundMidnight(t *testing.T) {
	e := spDemo(t)
	form := e.formID("frm_e7b302")
	// A date field of its own (the demo's dmarc_live_at has its flow).
	e.exec(`INSERT INTO custom_fields (id, workspace_id, key, label, type, created_at) VALUES ($1, $2, 'clinic_at', 'DMARC clinic', 'date', now())`,
		uuid.Must(uuid.NewV7()), e.ws.ID)
	body := spSection("b-hero", `<div class="prose"><h1>Late-night DMARC clinic</h1></div>`) +
		spSection("b-body", `<rc-event data-pb-id="event" date="2026-12-03" start="01:00" end="01:30" tz="UTC" where="online" platform="Zoom" link="https://acme.zoom.us/j/555"`+
			` title="Late-night DMARC clinic" field="clinic_at"></rc-event><rc-form data-pb-id="register" form="`+form+`" heading="Save your seat"></rc-form>`)
	d := e.newPage("Late-night DMARC clinic", body)
	e.publish(d.ID)
	var rem pages.PageReminders
	e.api.Do(http.MethodPost, wsAPI+"/pages/"+d.ID+"/reminder-automation", nil).MustStatus(t, http.StatusCreated).JSON(t, &rem)
	if rem.Flow == nil {
		t.Fatalf("no automation: %+v", rem)
	}
	var trig struct {
		At         string `json:"at"`
		OffsetDays int    `json:"offsetDays"`
		Field      string `json:"field"`
	}
	if err := json.Unmarshal(one[[]byte](t, e.Env, `SELECT trigger FROM automations WHERE id = $1`, rem.Flow.ID), &trig); err != nil {
		t.Fatal(err)
	}
	const email = "kai.mori@fastmail.com"
	must(t, spPost(t, e, "late-night-dmarc-clinic", spSignup(e, form, "frm_e7b302", "register", email)), http.StatusOK, "You're registered")
	contact, ok := contactID(t, e.Env, e.ws.ID, email)
	if !ok {
		t.Fatal("no contact")
	}
	date := one[string](t, e.Env, `SELECT coalesce(attributes->>'clinic_at', '') FROM contacts WHERE id = $1`, contact)
	if date == "" || trig.Field != "clinic_at" {
		// (An empty date field is the contactOf bug: see TestPageSignupForms.)
		t.Fatalf("the registration wrote the date field %q; trigger %+v", date, trig)
	}
	loc, err := time.LoadLocation(e.ws.Timezone)
	if err != nil {
		t.Fatal(err)
	}
	day, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		t.Fatalf("date field %q: %v", date, err)
	}
	at, err := time.Parse("15:04", trig.At)
	if err != nil {
		t.Fatalf("trigger time %q: %v", trig.At, err)
	}
	fire := time.Date(day.Year(), day.Month(), day.Day()+trig.OffsetDays, at.Hour(), at.Minute(), 0, 0, loc)
	start := time.Date(2026, 12, 3, 1, 0, 0, 0, time.UTC)
	if want := start.Add(-24 * time.Hour); !fire.Equal(want) {
		t.Fatalf("the day-before reminder goes at %s (date field %s, trigger %+v); the session starts at %s",
			fire.UTC().Format(time.RFC3339), date, trig, start.Format(time.RFC3339))
	}
}
