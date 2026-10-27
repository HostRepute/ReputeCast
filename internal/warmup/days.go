package warmup

import (
	"errors"
	"math"
	"time"
)

// How a plan moves (review PR-8): a plan has a number of counted days, not
// calendar days. A day counts once at least CountShare of its cap went out
// and the guards held; the guards judge a day only after the plan's
// minimum sample (500 messages by default), so one complaint out of 200
// never holds the ramp. A day that doesn't count repeats the next day. A
// pause freezes the plan: it resumes on the day it stopped. After a guard
// trips, the cap holds (or steps back, as the plan says) until HealthyDays
// counted days in a row.

const (
	// CountShare is the share of a day's cap that must go out for the day
	// to count.
	CountShare = 0.8
	// HealthyDays is how many counted days in a row end a slowdown's hold.
	HealthyDays = 2
	// DefaultMinSample is the messages a day needs before the guards judge it.
	DefaultMinSample = 500
)

// DayStats is what a warming provider sent on one of the workspace's days.
// Complained and Delivered count only mailbox providers that report
// complaints per message (review F-2): Gmail's and iCloud's mail is in
// neither, so their silence is never read as a 0% complaint rate.
type DayStats struct {
	Sent, Bounced         int64
	Complained, Delivered int64
}

// Guards are a plan's slowdown settings.
type Guards struct {
	// Bounce and Complaint are the rates (percent) that slow the plan down.
	Bounce, Complaint float64
	MinSample         int64
	// Action is what a tripped guard does: hold, step_back or pause.
	Action string
}

// Verdict is how one day was judged.
type Verdict struct {
	// Counted: the day moved the plan on.
	Counted bool
	// Judged: the guards had enough messages to judge the day.
	Judged bool
	// Tripped: a guard tripped (Metric says which).
	Tripped bool
	Metric  string
	// Bounce and Complaint are the day's rates in percent, nil when there
	// was nothing to measure.
	Bounce, Complaint *float64
	// Short: too little of the cap went out for the day to count.
	Short bool
}

// Judge decides whether a day counts.
func Judge(st DayStats, cap int64, g Guards) Verdict {
	v := Verdict{}
	if st.Sent > 0 {
		b := round1(float64(st.Bounced) / float64(st.Sent) * 100)
		v.Bounce = &b
	}
	if st.Delivered > 0 {
		c := round2(float64(st.Complained) / float64(st.Delivered) * 100)
		v.Complaint = &c
	}
	minSample := g.MinSample
	if minSample <= 0 {
		minSample = DefaultMinSample
	}
	v.Judged = st.Sent >= minSample
	if v.Judged {
		switch {
		case v.Bounce != nil && *v.Bounce > g.Bounce:
			v.Tripped, v.Metric = true, "bounce"
		case v.Complaint != nil && *v.Complaint > g.Complaint:
			v.Tripped, v.Metric = true, "complaint"
		}
	}
	need := int64(math.Ceil(float64(cap) * CountShare))
	v.Short = cap <= 0 || st.Sent < need
	v.Counted = !v.Short && !v.Tripped
	return v
}

// Hold is a slowdown's hold on the cap: the cap it holds and the counted
// days so far.
type Hold struct {
	Cap    *int64
	Streak int
}

// CapFor is a day's cap: the held cap while a slowdown holds it, otherwise
// the planned one.
func (h Hold) CapFor(planned int64) int64 {
	if h.Cap != nil {
		return *h.Cap
	}
	return planned
}

// HoldStep is what judging a day did to the hold.
type HoldStep string

// Hold steps.
const (
	HoldNone    HoldStep = ""
	HoldStarted HoldStep = "started" // a guard tripped: the cap holds (or steps back)
	HoldHealthy HoldStep = "healthy" // a counted day under the hold, more needed
	HoldEnded   HoldStep = "ended"   // enough healthy days: the ramp resumes
	HoldPause   HoldStep = "pause"   // a guard tripped and the plan pauses
)

// After returns the hold once a day was judged: a tripped guard holds the
// day's cap (or the cap of the day before it for step_back, or pauses the
// plan); each counted day under a hold is a healthy day, and HealthyDays in
// a row end it. A day that didn't count for lack of volume is no healthy
// day, but doesn't break the run either.
func (h Hold) After(v Verdict, dayCap, prevCap int64, action string) (Hold, HoldStep) {
	switch {
	case v.Tripped:
		c := dayCap
		switch action {
		case "pause":
			return Hold{Cap: &c}, HoldPause
		case "step_back":
			if prevCap > 0 && prevCap < c {
				c = prevCap
			}
		}
		return Hold{Cap: &c}, HoldStarted
	case h.Cap != nil && v.Counted:
		h.Streak++
		if h.Streak >= HealthyDays {
			return Hold{}, HoldEnded
		}
		return h, HoldHealthy
	}
	return h, HoldNone
}

// MailboxShares are the parts of a day's cap per mailbox provider when a
// plan caps them separately: strict providers get their own lower caps.
var MailboxShares = []struct {
	Mailbox string
	Percent int
}{{"gmail", 40}, {"outlook", 25}, {"yahoo", 15}, {"icloud", 10}, {"other", 10}}

// MailboxCaps splits a day's cap per mailbox provider; the parts add up to
// the cap exactly (rounding goes to Gmail).
func MailboxCaps(cap int64) map[string]int64 {
	out := map[string]int64{}
	var used int64
	for _, s := range MailboxShares {
		n := cap * int64(s.Percent) / 100
		out[s.Mailbox] = n
		used += n
	}
	out["gmail"] += cap - used
	return out
}

// MailboxOf is the warm-up bucket of a mailbox provider key (anything
// unknown is "other").
func MailboxOf(mbp string) string {
	for _, s := range MailboxShares {
		if s.Mailbox == mbp {
			return mbp
		}
	}
	return "other"
}

// ErrTooShort refuses a plan length shorter than the days it already
// counted.
var ErrTooShort = errors.New("a plan can't be shorter than the days it already counted")

// Reschedule gives a plan a new length from where it stands: the days
// before today keep their volumes, today keeps its cap, and the rest ramps
// from it to the same last volume. Shortening a plan never makes today's
// cap jump; it only makes the coming days steeper.
func Reschedule(sched []Day, today, days int, todayCap int64) ([]Day, error) {
	if days < today {
		return nil, ErrTooShort
	}
	last := lastVolume(sched)
	first := int64(200)
	if len(sched) > 0 {
		first = sched[0].Volume
	}
	if today <= 1 {
		return Ramp(days, first, last), nil
	}
	if todayCap <= 0 {
		todayCap = first
	}
	full := Ramp(days, first, last) // tiers and shares by the new length
	out := make([]Day, 0, days)
	for _, d := range sched {
		if d.Day < today {
			out = append(out, d)
		}
	}
	steps := days - today
	growth := 1.0
	if steps > 0 && last > todayCap {
		growth = math.Pow(float64(last)/float64(todayCap), 1/float64(steps))
	}
	for d := today; d <= days; d++ {
		vol := roundNice(int64(float64(todayCap) * math.Pow(growth, float64(d-today))))
		switch d {
		case today:
			vol = todayCap
		case days:
			vol = max(last, todayCap)
		}
		nd := full[d-1]
		nd.Volume, nd.Rate = vol, RateFor(vol)
		out = append(out, nd)
	}
	return out, nil
}

// DayOn is a plan's day at a time, on the workspace's calendar: start is
// start_date (the date day 1 would have been if every day had counted; it
// moves on by every day that didn't count), and a paused plan stays on the
// day it stopped (pausedAt, nil unless the plan is paused). Callers clamp
// it to the plan's length.
func DayOn(start time.Time, pausedAt *time.Time, now time.Time, loc *time.Location) int {
	today := dateIn(now, loc)
	if pausedAt != nil {
		if p := dateIn(*pausedAt, loc); p.Before(today) {
			today = p
		}
	}
	return dayNumber(localDate(start, loc), today)
}

// pausedAt is a plan's pause time while it is paused.
func pausedAt(status string, at *time.Time) *time.Time {
	if status != "paused" {
		return nil
	}
	return at
}

// dayNumber is the plan's day on a date: start is the date day 1 would
// have been if every day had counted (start_date), both at midnight in the
// workspace's time zone.
func dayNumber(start, date time.Time) int {
	return int(date.Sub(start).Hours()/24+0.5) + 1
}

// localDate is a stored date (a date column, read as UTC midnight) as
// midnight in the workspace's time zone.
func localDate(d time.Time, loc *time.Location) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
}
