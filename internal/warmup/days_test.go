package warmup

import (
	"errors"
	"testing"
	"time"
)

// A day counts once 80% of its cap went out and the guards held; the
// guards judge a day only after the plan's minimum sample.
func TestJudge(t *testing.T) {
	g := Guards{Bounce: 2, Complaint: 0.1, MinSample: 500, Action: "hold"}
	cases := []struct {
		name                   string
		st                     DayStats
		cap                    int64
		counted, judged, short bool
		metric                 string
	}{
		{"full day, clean", DayStats{Sent: 1000, Bounced: 5, Delivered: 900, Complained: 0}, 1000, true, true, false, ""},
		{"80% of the cap is enough", DayStats{Sent: 800}, 1000, true, true, false, ""},
		{"under 80% repeats", DayStats{Sent: 799}, 1000, false, true, true, ""},
		{"nothing sent", DayStats{}, 6000, false, false, true, ""},
		{"bounces over the guard", DayStats{Sent: 1000, Bounced: 30}, 1000, false, true, false, "bounce"},
		{"complaints over the guard", DayStats{Sent: 1000, Delivered: 900, Complained: 2}, 1000, false, true, false, "complaint"},
		// One complaint out of 200 never holds the ramp: too few to judge.
		{"too few to judge", DayStats{Sent: 200, Bounced: 20, Delivered: 180, Complained: 1}, 200, true, false, false, ""},
		// Gmail's and iCloud's mail isn't in Delivered: no complaint rate.
		{"no complaint data", DayStats{Sent: 1000}, 1000, true, true, false, ""},
		{"no cap", DayStats{Sent: 10}, 0, false, false, true, ""},
	}
	for _, c := range cases {
		v := Judge(c.st, c.cap, g)
		if v.Counted != c.counted || v.Judged != c.judged || v.Short != c.short || v.Metric != c.metric || v.Tripped != (c.metric != "") {
			t.Errorf("%s: %+v", c.name, v)
		}
	}
	if v := Judge(DayStats{Sent: 1000}, 1000, g); v.Complaint != nil || v.Bounce == nil || *v.Bounce != 0 {
		t.Fatalf("rates without complaint data: %v %v", v.Bounce, v.Complaint)
	}
	// A plan's own minimum sample, and the default when it has none.
	if v := Judge(DayStats{Sent: 300, Bounced: 30}, 300, Guards{Bounce: 2, Complaint: 0.1, MinSample: 100}); !v.Tripped {
		t.Fatal("a smaller minimum sample judges sooner")
	}
	if v := Judge(DayStats{Sent: 499, Bounced: 100}, 499, Guards{Bounce: 2, Complaint: 0.1}); v.Judged {
		t.Fatal("the default minimum sample is 500")
	}
}

// After a guard trips the cap holds (or steps back, or the plan pauses)
// until two healthy days in a row.
func TestHoldAfter(t *testing.T) {
	trip := Verdict{Tripped: true, Metric: "bounce", Judged: true}
	counted := Verdict{Counted: true, Judged: true}
	short := Verdict{Short: true}

	h, step := Hold{}.After(trip, 2600, 1800, "hold")
	if step != HoldStarted || h.Cap == nil || *h.Cap != 2600 || h.Streak != 0 {
		t.Fatalf("hold: %v %+v", step, h)
	}
	if h2, step := (Hold{}).After(trip, 2600, 1800, "step_back"); step != HoldStarted || *h2.Cap != 1800 {
		t.Fatalf("step back: %v %+v", step, h2)
	}
	if h2, step := (Hold{}).After(trip, 2600, 1800, "pause"); step != HoldPause || h2.Cap == nil {
		t.Fatalf("pause: %v %+v", step, h2)
	}
	// A day too short to count is no healthy day, but doesn't break the run.
	h, step = h.After(short, 2600, 1800, "hold")
	if step != HoldNone || h.Cap == nil || h.Streak != 0 {
		t.Fatalf("short day under the hold: %v %+v", step, h)
	}
	h, step = h.After(counted, 2600, 1800, "hold")
	if step != HoldHealthy || h.Streak != 1 || h.CapFor(3600) != 2600 {
		t.Fatalf("first healthy day: %v %+v", step, h)
	}
	h, step = h.After(counted, 2600, 1800, "hold")
	if step != HoldEnded || h.Cap != nil || h.CapFor(3600) != 3600 {
		t.Fatalf("second healthy day: %v %+v", step, h)
	}
	// Another trip during a hold starts it again.
	h = Hold{Cap: ptr(int64(2600)), Streak: 1}
	if h, step = h.After(trip, 2600, 1800, "hold"); step != HoldStarted || h.Streak != 0 {
		t.Fatalf("trip during a hold: %v %+v", step, h)
	}
	if _, step := (Hold{}).After(counted, 4800, 3600, "hold"); step != HoldNone {
		t.Fatalf("a counted day without a hold: %v", step)
	}
}

func ptr[T any](v T) *T { return &v }

// The caps per mailbox provider add up to the day's cap.
func TestMailboxCaps(t *testing.T) {
	c := MailboxCaps(6000)
	want := map[string]int64{"gmail": 2400, "outlook": 1500, "yahoo": 900, "icloud": 600, "other": 600}
	for k, v := range want {
		if c[k] != v {
			t.Fatalf("%s = %d, want %d", k, c[k], v)
		}
	}
	for _, total := range []int64{1, 7, 999, 1001, 48000} {
		var sum int64
		for _, v := range MailboxCaps(total) {
			sum += v
		}
		if sum != total {
			t.Fatalf("caps of %d add up to %d", total, sum)
		}
	}
	if MailboxOf("gmail") != "gmail" || MailboxOf("proton") != "other" || MailboxOf("") != "other" {
		t.Fatal("mailbox buckets")
	}
	a := Allowance{Cap: 6000, Sent: 2000, Spread: true, MailboxCaps: c, MailboxSent: map[string]int64{"gmail": 2400, "outlook": 100}}
	if a.MailboxLeft("gmail") != 0 || a.MailboxLeft("outlook") != 1400 || a.MailboxLeft("fastmail") != 600 || a.Left() != 4000 {
		t.Fatalf("left: gmail %d outlook %d other %d all %d", a.MailboxLeft("gmail"), a.MailboxLeft("outlook"), a.MailboxLeft("fastmail"), a.Left())
	}
	a.Spread = false
	if a.MailboxLeft("gmail") != 4000 {
		t.Fatal("without separate caps only the day's cap counts")
	}
	a.Paused = true
	if a.Left() != 0 || a.MailboxLeft("outlook") != 0 {
		t.Fatal("a paused plan sends nothing")
	}
}

// A new length rebuilds the days ahead from today's cap: shortening never
// makes today's cap jump, and a plan can't get shorter than where it is.
func TestReschedule(t *testing.T) {
	sched := Ramp(21, 200, 48000)
	today := 9
	todayCap := sched[today-1].Volume
	for _, days := range []int{14, 30, 9} {
		next, err := Reschedule(sched, today, days, todayCap)
		if err != nil {
			t.Fatalf("%d days: %v", days, err)
		}
		if len(next) != days {
			t.Fatalf("%d days: %d entries", days, len(next))
		}
		for d := 1; d < today; d++ {
			if next[d-1].Volume != sched[d-1].Volume {
				t.Fatalf("%d days: day %d changed %d → %d", days, d, sched[d-1].Volume, next[d-1].Volume)
			}
		}
		if next[today-1].Volume != todayCap {
			t.Fatalf("%d days: today's cap jumped %d → %d", days, todayCap, next[today-1].Volume)
		}
		// The last day keeps the plan's last volume, unless the plan now
		// ends today: today's cap never jumps.
		if last := next[days-1].Volume; (days > today && last != 48000) || (days == today && last != todayCap) {
			t.Fatalf("%d days: last day %d", days, last)
		}
		for i := 1; i < len(next); i++ {
			if next[i].Volume < next[i-1].Volume || next[i].Day != i+1 {
				t.Fatalf("%d days: day %d shrinks or is misnumbered: %+v", days, i+1, next[i])
			}
		}
	}
	if _, err := Reschedule(sched, today, 8, todayCap); !errors.Is(err, ErrTooShort) {
		t.Fatalf("shorter than today: %v", err)
	}
	// A held cap stays today's cap in the new schedule.
	next, err := Reschedule(sched, today, 14, 2600)
	if err != nil || next[today-1].Volume != 2600 {
		t.Fatalf("held cap: %v %+v", err, next[today-1])
	}
	// Day 1 just builds a new ramp.
	if next, _ := Reschedule(sched, 1, 14, 0); len(next) != 14 || next[0].Volume != 200 {
		t.Fatalf("from day 1: %+v", next[0])
	}
}

// The plan's day is the workspace's calendar day, across a clock change.
func TestDayNumber(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip(err)
	}
	start := time.Date(2026, 10, 20, 0, 0, 0, 0, loc)
	if d := dayNumber(start, time.Date(2026, 11, 3, 0, 0, 0, 0, loc)); d != 15 {
		t.Fatalf("after the clock change: day %d", d)
	}
	if d := dayNumber(start, start); d != 1 {
		t.Fatalf("first day: %d", d)
	}
	stored := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	if got := localDate(stored, loc); !got.Equal(start) {
		t.Fatalf("stored date: %v", got)
	}
	// Late evening in New York is still the workspace's day, not UTC's.
	now := time.Date(2026, 10, 21, 2, 30, 0, 0, time.UTC) // 22:30 on the 20th in New York
	if d := dayNumber(start, dateIn(now, loc)); d != 1 {
		t.Fatalf("evening: day %d", d)
	}
}
