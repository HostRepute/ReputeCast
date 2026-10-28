package warmup_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/warmup"
)

const ws = "/workspaces/acme-newsletter"

func TestRampGrowsToTheTarget(t *testing.T) {
	days := warmup.Ramp(21, 200, 48000)
	if len(days) != 21 || days[0].Volume != 200 || days[20].Volume != 48000 {
		t.Fatalf("ramp ends %d … %d", days[0].Volume, days[20].Volume)
	}
	for i := 1; i < len(days); i++ {
		if days[i].Volume < days[i-1].Volume {
			t.Fatalf("day %d shrinks: %d < %d", i+1, days[i].Volume, days[i-1].Volume)
		}
		if days[i].Share < days[i-1].Share {
			t.Fatalf("share drops on day %d", i+1)
		}
	}
	if days[0].Share != 10 || days[20].Share != 40 || len(days[20].Tiers) != len(warmup.Tiers) || len(days[0].Tiers) != 1 {
		t.Fatalf("first %+v, last %+v", days[0], days[20])
	}
	if warmup.RateFor(200) != 1 || warmup.RateFor(6000) != 2 {
		t.Fatalf("rates %v %v", warmup.RateFor(200), warmup.RateFor(6000))
	}
}

// Progress rows are kept by date; rows from before counted days (without a
// date) are dropped.
func TestParsePlanKeepsDatedDays(t *testing.T) {
	sched, _ := json.Marshal(warmup.Ramp(21, 200, 48000))
	progress, _ := json.Marshal([]warmup.Result{{Day: 2, Date: "2026-10-03", Cap: 400, Sent: 400, Counted: true}, {Day: 1, Date: "2026-10-02", Cap: 200, Sent: 200, Counted: true},
		{Day: 3, Cap: 700, Sent: 700}})
	p := warmup.ParsePlan(sched, progress)
	if len(p.Progress) != 2 || p.Progress[0].Date != "2026-10-02" || p.Counted() != 2 {
		t.Fatalf("progress %+v", p.Progress)
	}
	if r, ok := p.OnDate("2026-10-03"); !ok || r.Day != 2 {
		t.Fatalf("on date %+v", r)
	}
	if p.PlannedCap(9) != warmup.Ramp(21, 200, 48000)[8].Volume || p.PlannedCap(40) != 48000 {
		t.Fatal("planned caps")
	}
}

// The demo plan reads as ProvidersWarmup draws it: started eleven days ago,
// eight days counted, so today is day 9 of 21 at 6,000. The slowdown's hold
// kept the plan on day 6 until its second healthy day (R3.L6d).
func TestWarmupEndpoints(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	var o warmup.WarmupOverview
	owner.Do("GET", ws+"/warmup", nil).MustStatus(t, 200).JSON(t, &o)
	if len(o.Plans) != 1 {
		t.Fatalf("plans %+v", o.Plans)
	}
	p := o.Plans[0]
	if p.Day != 9 || p.Days != 21 || p.Counted != 8 || p.CalendarDays != 11 || p.TodayCap != 6000 || p.NeedToday != 4800 || p.Progress != 38.1 {
		t.Fatalf("plan day %d/%d counted %d of %d calendar days, cap %d need %d, progress %v", p.Day, p.Days, p.Counted, p.CalendarDays, p.TodayCap, p.NeedToday, p.Progress)
	}
	if p.SentTotal < 16898 || p.Slowdowns != 1 || p.LastSlowdown == nil || p.LastSlowdown.Day != 6 || p.LastSlowdown.Metric != "bounce" {
		t.Fatalf("sent %d, slowdowns %d %+v", p.SentTotal, p.Slowdowns, p.LastSlowdown)
	}
	if p.BounceRate == nil || *p.BounceRate != 0.9 || p.ComplaintRate == nil || *p.ComplaintRate != 0.05 {
		t.Fatalf("7-day rates %v %v", p.BounceRate, p.ComplaintRate)
	}
	if p.RateToday != 2 || p.NextRate == nil || p.NextRate.Day != 12 || p.NextRate.Value != 4 {
		t.Fatalf("rate %v next %+v", p.RateToday, p.NextRate)
	}
	want := []string{"counted", "counted", "counted", "counted", "not_counted", "counted", "not_counted", "not_counted", "counted", "counted", "counted", "today"}
	if len(p.Rows) < len(want)+12 {
		t.Fatalf("%d rows", len(p.Rows))
	}
	for i, st := range want {
		if p.Rows[i].Status != st {
			t.Fatalf("row %d (day %d) is %s, want %s", i, p.Rows[i].Day, p.Rows[i].Status, st)
		}
	}
	if r := p.Rows[4]; r.Day != 5 || r.Reason != "short" || r.Judged {
		t.Fatalf("Sunday: %+v", r)
	}
	if r := p.Rows[6]; r.Day != 6 || r.Reason != "guard" || r.Metric != "bounce" || !r.Judged {
		t.Fatalf("the slowdown: %+v", r)
	}
	// The first healthy day proves the held cap without moving the plan on;
	// the second completes day 6, and the ramp resumes one step up.
	if r := p.Rows[7]; r.Day != 6 || !r.Held || r.Healthy != 1 || r.Reason != "held" {
		t.Fatalf("first healthy day: %+v", r)
	}
	if r := p.Rows[8]; r.Day != 6 || r.Cap != 2600 || r.PlannedCap != 0 || r.Healthy != 2 || r.Status != "counted" {
		t.Fatalf("second healthy day: %+v", r)
	}
	if r := p.Rows[9]; r.Day != 7 || r.Cap != 3600 || r.Held {
		t.Fatalf("the day after the hold: %+v", r)
	}
	if r := p.Rows[11]; r.Day != 9 || r.Cap != 6000 || r.Need != 4800 || r.Judged {
		t.Fatalf("today: %+v", r)
	}
	if r := p.Rows[12]; r.Day != 9 || !r.Repeat || r.Status != "planned" {
		t.Fatalf("tomorrow repeats day 9: %+v", r)
	}
	last := p.Rows[len(p.Rows)-1]
	if last.Day != 21 || !last.Eligible || last.Status != "planned" || last.Cap != 48000 || !last.Date.Equal(p.FinishesOn) {
		t.Fatalf("last day %+v, finishes %v", last, p.FinishesOn)
	}
	caps := map[string]int64{}
	var total int64
	for _, c := range p.MailboxCaps {
		caps[c.Mailbox] = c.Cap
		total += c.Cap
	}
	if total != 6000 || caps["gmail"] != 2400 || caps["outlook"] != 1500 || caps["yahoo"] != 900 || caps["icloud"] != 600 || caps["other"] != 600 {
		t.Fatalf("mailbox caps %+v", p.MailboxCaps)
	}
	if p.OverflowTo != "Amazon SES" || p.Settings.Overflow == nil || !*p.Settings.Overflow || len(p.Holds) != 0 {
		t.Fatalf("overflow %q %v, holds %d", p.OverflowTo, p.Settings.Overflow, len(p.Holds))
	}
	// Nothing is scheduled today (the demo's next send is in two days), so
	// the log opens with today's line.
	if !p.RepeatsTomorrow || len(p.Log) != 6 || p.Log[0].Kind != "today" || p.Log[0].Body != "Nothing is scheduled today, so day 9 doesn't count and repeats tomorrow." {
		t.Fatalf("repeats tomorrow %v, log %+v", p.RepeatsTomorrow, p.Log)
	}
	path := ws + "/warmup/" + p.ID

	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("GET", ws+"/warmup", nil).MustStatus(t, 200)
	off := false
	settings := warmup.WarmupSettings{Days: 21, EngagedFirst: true, GuardBounce: 1.5, GuardComplaint: 0.08, GuardAction: "step_back", SpreadMailbox: true, Overflow: &off}
	viewer.Do("PUT", path, settings).MustStatus(t, 403)
	bad := settings
	bad.GuardBounce = 50
	owner.Do("PUT", path, bad).MustStatus(t, 422)
	var saved warmup.WarmupPlanView
	owner.Do("PUT", path, settings).MustStatus(t, 200).JSON(t, &saved)
	if saved.Settings.GuardBounce != 1.5 || saved.Settings.GuardAction != "step_back" || *saved.Settings.Overflow {
		t.Fatalf("settings %+v", saved.Settings)
	}
	// A shorter plan keeps today's cap and reaches the same last volume
	// sooner (PR-8).
	settings.Days = 14
	owner.Do("PUT", path, settings).MustStatus(t, 200).JSON(t, &saved)
	end := saved.Rows[len(saved.Rows)-1]
	if saved.Days != 14 || saved.Day != 9 || saved.TodayCap != 6000 || end.Cap != 48000 || end.Day != 14 {
		t.Fatalf("shortened: day %d/%d cap %d, last %+v", saved.Day, saved.Days, saved.TodayCap, end)
	}

	viewer.Do("POST", path+"/pause", nil).MustStatus(t, 403)
	owner.Do("POST", path+"/pause", nil).MustStatus(t, 200)
	owner.Do("GET", ws+"/warmup", nil).MustStatus(t, 200).JSON(t, &o)
	if o.Plans[0].Status != "paused" || o.Plans[0].Day != 9 {
		t.Fatalf("after pause: %s day %d", o.Plans[0].Status, o.Plans[0].Day)
	}
	owner.Do("POST", path+"/resume", nil).MustStatus(t, 200).JSON(t, &saved)
	if saved.Status != "active" || saved.Day != 9 {
		t.Fatalf("after resume: %s day %d", saved.Status, saved.Day)
	}

	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action IN ('warmup.update','warmup.pause','warmup.resume')`).Scan(&n); err != nil || n != 4 {
		t.Fatalf("warm-up audit = %d %v", n, err)
	}

	// The other providers can start a plan; Postal already has one.
	if len(o.Candidates) == 0 {
		t.Fatal("no warm-up candidates")
	}
	var created warmup.WarmupPlanView
	owner.Do("POST", ws+"/warmup", map[string]any{"providerId": o.Candidates[0].ProviderID, "days": 14, "engagedFirst": true, "guardBounce": 2, "guardComplaint": 0.1, "guardAction": "hold", "spreadMailbox": true}).
		MustStatus(t, 201).JSON(t, &created)
	if created.Days != 14 || created.Day != 1 || created.Counted != 0 || created.Rows[0].Status != "today" || created.Settings.Overflow == nil || !*created.Settings.Overflow {
		t.Fatalf("created %+v", created)
	}
}

// harness drives a plan's days on the workspace's calendar: messages a
// provider sent on a day, and the daily job at a given time.
type harness struct {
	t   *testing.T
	env *apptest.Env
	ctx context.Context
	ws  store.Workspace
	loc *time.Location
	at  time.Time
	svc *warmup.Service
}

func newHarness(t *testing.T) *harness {
	env := apptest.New(t)
	env.SeedDemo()
	h := &harness{t: t, env: env, ctx: context.Background()}
	var err error
	if h.ws, err = env.App.Q.GetWorkspaceBySlug(h.ctx, "acme-newsletter"); err != nil {
		t.Fatal(err)
	}
	// The workspace's day is Ho Chi Minh City's (UTC+7), not the server's.
	if h.loc, err = time.LoadLocation("Asia/Ho_Chi_Minh"); err != nil {
		t.Skip(err)
	}
	h.exec(`UPDATE workspaces SET timezone = 'Asia/Ho_Chi_Minh' WHERE id = $1`, h.ws.ID)
	h.ws.Timezone = "Asia/Ho_Chi_Minh"
	h.svc = &warmup.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Now: func() time.Time { return h.at }}
	return h
}

func (h *harness) exec(sql string, args ...any) {
	h.t.Helper()
	if _, err := h.env.App.Pool.Exec(h.ctx, sql, args...); err != nil {
		h.t.Fatalf("%s: %v", sql, err)
	}
}

func (h *harness) count(sql string, args ...any) int64 {
	h.t.Helper()
	var n int64
	if err := h.env.App.Pool.QueryRow(h.ctx, sql, args...).Scan(&n); err != nil {
		h.t.Fatalf("%s: %v", sql, err)
	}
	return n
}

// send records n messages the provider sent at a time: bounced of them
// bounced and complained complained.
func (h *harness) send(provider uuid.UUID, at time.Time, mailbox string, n, bounced, complained int) {
	h.t.Helper()
	h.exec(`INSERT INTO messages (id, workspace_id, stream, email, provider_id, mailbox_provider, status, queued_at, sent_at, delivered_at)
		SELECT gen_random_uuid(), $1, 'broadcast', 'warm.' || g || '.' || extract(epoch from $4::timestamptz)::bigint || '@' || $3 || '.example', $2, $3,
		       CASE WHEN g <= $5 THEN 'bounced' WHEN g <= $5 + $6 THEN 'complained' ELSE 'delivered' END, $4, $4,
		       CASE WHEN g <= $5 THEN NULL ELSE $4::timestamptz END
		FROM generate_series(1, $7::int) g`, h.ws.ID, provider, mailbox, at, bounced, complained, n)
}

// day is the start of a day of the workspace, a month ahead of the demo's data.
func (h *harness) day(i int) time.Time {
	base := time.Now().In(h.loc).AddDate(0, 1, 0)
	return time.Date(base.Year(), base.Month(), base.Day()+i, 0, 0, 0, 0, h.loc)
}

// plan starts a 14-day plan for SES with simple caps (600, 800, 1,000,
// 1,200, …) on day 0.
func (h *harness) plan() (uuid.UUID, uuid.UUID) {
	h.t.Helper()
	var ses uuid.UUID
	if err := h.env.App.Pool.QueryRow(h.ctx, `SELECT id FROM providers WHERE workspace_id = $1 AND adapter = 'ses' AND disconnected_at IS NULL`, h.ws.ID).Scan(&ses); err != nil {
		h.t.Fatal(err)
	}
	h.at = h.day(0).Add(9 * time.Hour)
	off := false
	v, err := h.svc.Create(h.ctx, &h.ws, warmup.NewPlanInput{ProviderID: ses.String(), WarmupSettings: warmup.WarmupSettings{Days: 14, EngagedFirst: true,
		GuardBounce: 2, GuardComplaint: 0.1, GuardAction: "hold", SpreadMailbox: true, Overflow: &off}, TargetVolume: 20000})
	if err != nil {
		h.t.Fatal(err)
	}
	sched := make([]warmup.Day, 14)
	for i := range sched {
		vol := int64(600 + 200*i)
		sched[i] = warmup.Day{Day: i + 1, Volume: vol, Rate: warmup.RateFor(vol), Tiers: []string{"active"}, Share: 10}
	}
	b, _ := json.Marshal(sched)
	id := uuid.MustParse(v.ID)
	h.exec(`UPDATE warmup_plans SET schedule = $2 WHERE id = $1`, id, b)
	return id, ses
}

func (h *harness) view(id uuid.UUID) warmup.WarmupPlanView {
	h.t.Helper()
	o, err := h.svc.Overview(h.ctx, &h.ws)
	if err != nil {
		h.t.Fatal(err)
	}
	for _, p := range o.Plans {
		if p.ID == id.String() {
			return p
		}
	}
	h.t.Fatalf("plan %s not listed", id)
	return warmup.WarmupPlanView{}
}

// PR-8: the daily job moves a plan by counted days. A day counts once 80%
// of its cap went out and the guards held; the guards judge a day only
// after 500 messages; a slowdown holds the cap until two healthy days in a
// row, with the plan staying on its day meanwhile, so the ramp resumes one
// step above the held cap (R3.L6d); a day that didn't count repeats.
func TestPlanMovesByCountedDays(t *testing.T) {
	h := newHarness(t)
	id, ses := h.plan()
	noon := func(i int) time.Time { return h.day(i).Add(12 * time.Hour) }
	h.send(ses, noon(0), "outlook", 600, 3, 0)  // day 1, cap 600: counts
	h.send(ses, noon(1), "outlook", 800, 24, 0) // day 2, cap 800: bounces 3% trip the guard
	h.send(ses, noon(2), "outlook", 800, 2, 0)  // day 2 again, held at 800: healthy day 1, the plan stays on day 2
	h.send(ses, noon(3), "outlook", 800, 2, 0)  // day 2 again, held at 800: healthy day 2 ends the hold and completes day 2
	h.send(ses, noon(4), "yahoo", 300, 150, 0)  // day 3, cap 1,000 (one step up): too few to judge, too few to count
	h.at = h.day(5).Add(10 * time.Hour)
	if err := h.svc.Evaluate(h.ctx, id); err != nil {
		t.Fatal(err)
	}
	v := h.view(id)
	if v.Counted != 2 || v.Day != 3 || v.TodayCap != 1000 || v.Held != nil || v.Status != "active" || v.CalendarDays != 5 {
		t.Fatalf("after five days: counted %d, day %d, cap %d, held %v, status %s, calendar %d", v.Counted, v.Day, v.TodayCap, v.Held, v.Status, v.CalendarDays)
	}
	want := []struct {
		day            int
		status, reason string
		cap, planned   int64
		held           bool
		healthy        int
		judged         bool
	}{
		{1, "counted", "", 600, 0, false, 0, true},
		{2, "not_counted", "guard", 800, 0, false, 0, true},
		{2, "not_counted", "held", 800, 0, true, 1, true},
		{2, "counted", "", 800, 0, true, 2, true},
		{3, "not_counted", "short", 1000, 0, false, 0, false},
	}
	for i, w := range want {
		r := v.Rows[i]
		if r.Day != w.day || r.Status != w.status || r.Reason != w.reason || r.Cap != w.cap || r.PlannedCap != w.planned || r.Held != w.held || r.Healthy != w.healthy || r.Judged != w.judged {
			t.Fatalf("row %d: %+v, want %+v", i, r, w)
		}
	}
	if r := v.Rows[1]; r.Metric != "bounce" || r.Bounce == nil || *r.Bounce != 3 {
		t.Fatalf("the slowdown: %+v", r)
	}
	if r := v.Rows[5]; r.Status != "today" || r.Day != 3 || !r.Date.Equal(h.day(5)) {
		t.Fatalf("today: %+v", r)
	}
	// The log keeps what changed (started, the slowdown, its end, the day
	// that repeated) under today's line: nothing is scheduled today. The
	// hold's end names the next step, day 3 at 1,000.
	if v.Slowdowns != 1 || len(v.Log) != 5 || v.Log[0].Kind != "today" || v.Log[1].Body != "300 of 1,000 sent, so day 3 repeated on "+h.day(5).Weekday().String()+"." ||
		v.Log[2].Kind != "resumed" || !strings.Contains(v.Log[2].Body, "the ramp resumes with day 3 at 1,000") {
		t.Fatalf("slowdowns %d, log %+v", v.Slowdowns, v.Log)
	}
	var startDate time.Time
	if err := h.env.App.Pool.QueryRow(h.ctx, `SELECT start_date FROM warmup_plans WHERE id = $1`, id).Scan(&startDate); err != nil {
		t.Fatal(err)
	}
	if !time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, h.loc).Equal(h.day(3)) {
		t.Fatalf("day 1 moved to %v, want three days later (the trip, the first healthy day, the short day)", startDate)
	}
	// The job runs every hour: running it again changes nothing.
	if err := h.svc.Evaluate(h.ctx, id); err != nil {
		t.Fatal(err)
	}
	if again := h.view(id); again.Counted != 2 || again.Day != 3 || len(again.Rows) != len(v.Rows) || len(again.Log) != 5 {
		t.Fatalf("a second run moved the plan: counted %d day %d", again.Counted, again.Day)
	}

	// Today's allowance counts the workspace's day, whatever clock the
	// caller keeps: 23:30 yesterday in Ho Chi Minh City is yesterday.
	h.send(ses, h.day(5).Add(-30*time.Minute), "gmail", 50, 0, 0)
	h.send(ses, h.day(5).Add(30*time.Minute), "gmail", 100, 0, 0)
	a, ok, err := warmup.AllowanceFor(h.ctx, h.env.App.Q, ses, h.at, time.UTC)
	if err != nil || !ok {
		t.Fatalf("allowance: %v %v", ok, err)
	}
	if a.Cap != 1000 || a.Sent != 100 || a.Left() != 900 || a.MailboxCaps["gmail"] != 400 || a.MailboxLeft("gmail") != 300 || a.Day != 3 || !a.EngagedFirst || a.Overflow {
		t.Fatalf("allowance %+v", a)
	}
}

// PR-8: a pause freezes the plan. Its days aren't judged and it resumes
// on the day it stopped; a guard can pause it too.
func TestPauseFreezesThePlan(t *testing.T) {
	h := newHarness(t)
	id, ses := h.plan()
	h.send(ses, h.day(0).Add(12*time.Hour), "gmail", 600, 0, 0)
	h.at = h.day(1).Add(8 * time.Hour)
	if err := h.svc.Evaluate(h.ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.SetPaused(h.ctx, &h.ws, id.String(), true); err != nil {
		t.Fatal(err)
	}
	a, _, _ := warmup.AllowanceFor(h.ctx, h.env.App.Q, ses, h.at, nil)
	if !a.Paused || a.Left() != 0 {
		t.Fatalf("a paused plan sends nothing: %+v", a)
	}
	// Three days later nothing was judged.
	h.at = h.day(4).Add(8 * time.Hour)
	if err := h.svc.Evaluate(h.ctx, id); err != nil {
		t.Fatal(err)
	}
	if v := h.view(id); v.Status != "paused" || v.Counted != 1 || len(v.Rows) < 2 || v.Rows[1].Status != "today" {
		t.Fatalf("while paused: %s counted %d rows %+v", v.Status, v.Counted, v.Rows)
	}
	v, err := h.svc.SetPaused(h.ctx, &h.ws, id.String(), false)
	if err != nil {
		t.Fatal(err)
	}
	if v.Status != "active" || v.Day != 2 || v.TodayCap != 800 || v.Counted != 1 {
		t.Fatalf("resumed on day %d (cap %d, %s, counted %d), want day 2", v.Day, v.TodayCap, v.Status, v.Counted)
	}
	if h.count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action IN ('warmup.pause', 'warmup.resume')`, id.String()) != 2 {
		t.Fatal("pause and resume aren't audited")
	}
	// The paused days are never judged later; the day it resumed didn't
	// count either.
	h.at = h.day(5).Add(8 * time.Hour)
	if err := h.svc.Evaluate(h.ctx, id); err != nil {
		t.Fatal(err)
	}
	if v := h.view(id); v.Counted != 1 || v.Day != 2 || len(v.Rows) < 3 || v.Rows[1].Reason != "short" || v.Rows[2].Status != "today" {
		t.Fatalf("after the pause: counted %d day %d rows %+v", v.Counted, v.Day, v.Rows)
	}

	// A guard set to pause pauses the plan by itself.
	h.exec(`UPDATE warmup_plans SET guard_action = 'pause' WHERE id = $1`, id)
	h.send(ses, h.day(5).Add(12*time.Hour), "gmail", 800, 40, 0)
	h.at = h.day(6).Add(8 * time.Hour)
	if err := h.svc.Evaluate(h.ctx, id); err != nil {
		t.Fatal(err)
	}
	if v := h.view(id); v.Status != "paused" || v.Day != 2 {
		t.Fatalf("after the guard: %s day %d", v.Status, v.Day)
	}
	if h.count(`SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'warmup.status'`, id.String()) == 0 {
		t.Fatal("the guard's pause isn't audited")
	}
}

// GS-9's warm-up hold: a share held at the cap waits for tomorrow's cap,
// moves to a provider that passes the checks, or its rest is cancelled.
func TestCapHoldChoices(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, sql, args...); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	count := func(sql string, args ...any) int64 {
		t.Helper()
		var n int64
		if err := env.App.Pool.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		return n
	}
	var wsID, postal, ses, campaign, other uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT w.id, p.id, s.id FROM workspaces w JOIN providers p ON p.workspace_id = w.id AND p.adapter = 'postal'
		JOIN providers s ON s.workspace_id = w.id AND s.adapter = 'ses' WHERE w.slug = 'acme-newsletter'`).Scan(&wsID, &postal, &ses); err != nil {
		t.Fatal(err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM campaigns WHERE workspace_id = $1 AND name = 'Weekly Digest #142'`, wsID).Scan(&campaign); err != nil {
		t.Fatal(err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM campaigns WHERE workspace_id = $1 AND id <> $2 AND deleted_at IS NULL ORDER BY created_at LIMIT 1`, wsID, campaign).Scan(&other); err != nil {
		t.Fatal(err)
	}
	hold := func(c uuid.UUID, until time.Time, n int) {
		t.Helper()
		detail, _ := json.Marshal(map[string]any{"reason": "warmup_cap", "until": until})
		exec(`INSERT INTO campaign_routes (campaign_id, provider_id, planned, status, status_detail) VALUES ($1, $2, $3, 'held', $4)
			ON CONFLICT (campaign_id, provider_id) DO UPDATE SET status = 'held', status_detail = EXCLUDED.status_detail, planned = EXCLUDED.planned`, c, postal, n, string(detail))
		exec(`INSERT INTO messages (id, workspace_id, stream, campaign_id, email, provider_id, mailbox_provider, status)
			SELECT gen_random_uuid(), $1, 'broadcast', $2, 'held.' || g || '.' || $4 || '@example.org', $3, 'gmail', 'queued' FROM generate_series(1, $5::int) g`,
			wsID, c, postal, c.String()[:8], n)
		exec(`INSERT INTO campaign_incidents (id, campaign_id, kind, scope, provider_id, metric, held, detail, tripped_at)
			VALUES (gen_random_uuid(), $1, 'throttled', 'route', $2, 'warmup_cap', $3, '{}', now())`, c, postal, n)
	}
	hold(campaign, time.Now().Add(10*time.Hour), 20)

	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	var o warmup.WarmupOverview
	owner.Do("GET", ws+"/warmup", nil).MustStatus(t, 200).JSON(t, &o)
	p := o.Plans[0]
	if len(p.Holds) != 1 || p.Holds[0].Held != 20 || p.Holds[0].Campaign != "Weekly Digest #142" || p.Holds[0].ResumeAt == nil {
		t.Fatalf("holds %+v", p.Holds)
	}
	var sesAlt *warmup.WarmupHoldAlternative
	for i, a := range p.Holds[0].Alternatives {
		if a.ProviderID == ses.String() {
			sesAlt = &p.Holds[0].Alternatives[i]
		}
	}
	if sesAlt == nil || !sesAlt.Allowed || len(sesAlt.Checks) == 0 {
		t.Fatalf("SES as an alternative: %+v", p.Holds[0].Alternatives)
	}
	path := ws + "/warmup/" + p.ID + "/held/" + campaign.String()

	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do("POST", path, map[string]any{"action": "wait"}).MustStatus(t, 403)

	var res warmup.WarmupHoldDecisionResult
	owner.Do("POST", path, map[string]any{"action": "wait"}).MustStatus(t, 200).JSON(t, &res)
	if count(`SELECT count(*) FROM campaign_routes WHERE campaign_id = $1 AND provider_id = $2 AND status = 'held' AND status_detail LIKE '%"decision":"wait"%'`, campaign, postal) != 1 {
		t.Fatal("waiting isn't recorded")
	}
	owner.Do("POST", path, map[string]any{"action": "move", "targetProviderId": postal.String()}).MustStatus(t, 422)
	owner.Do("POST", path, map[string]any{"action": "move", "targetProviderId": ses.String()}).MustStatus(t, 200).JSON(t, &res)
	if res.Moved != 20 {
		t.Fatalf("moved %d", res.Moved)
	}
	if count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND provider_id = $2 AND status = 'queued' AND email LIKE 'held.%'`, campaign, ses) != 20 {
		t.Fatal("the held messages didn't move to SES")
	}
	if count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND metric = 'warmup_cap' AND resolution = 'moved' AND resolved_at IS NOT NULL`, campaign) != 1 {
		t.Fatal("the hold isn't resolved")
	}
	owner.Do("POST", path, map[string]any{"action": "move", "targetProviderId": ses.String()}).MustStatus(t, 409)

	// Cancelling drops the held rest of this share only.
	hold(other, time.Now().Add(10*time.Hour), 7)
	owner.Do("POST", ws+"/warmup/"+p.ID+"/held/"+other.String(), map[string]any{"action": "cancel"}).MustStatus(t, 200).JSON(t, &res)
	if res.Cancelled != 7 || count(`SELECT count(*) FROM messages WHERE campaign_id = $1 AND provider_id = $2 AND status = 'cancelled' AND email LIKE 'held.%'`, other, postal) != 7 {
		t.Fatalf("cancelled %d", res.Cancelled)
	}
	if count(`SELECT count(*) FROM audit_events WHERE action IN ('campaign.warmup_hold_wait', 'campaign.warmup_hold_move', 'campaign.warmup_hold_cancel')`) != 3 {
		t.Fatal("the decisions aren't audited")
	}

	// A share nobody moved goes on by itself once tomorrow's cap starts.
	exec(`DELETE FROM campaign_incidents WHERE campaign_id = $1`, other)
	hold(other, time.Now().Add(-time.Minute), 5)
	svc := &warmup.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}
	if err := svc.ResumeCapHolds(ctx); err != nil {
		t.Fatal(err)
	}
	if count(`SELECT count(*) FROM campaign_routes WHERE campaign_id = $1 AND provider_id = $2 AND status = 'sending'`, other, postal) != 1 ||
		count(`SELECT count(*) FROM campaign_incidents WHERE campaign_id = $1 AND resolution = 'auto_recovered'`, other) != 1 {
		t.Fatal("the hold didn't end with the new day")
	}
}
