// Package warmup plans and runs the warm-up of a new sending provider or
// dedicated IP: a daily volume ramp, the most engaged contacts first, an
// automatic slowdown when bounces or complaints rise, and a Broadcast share
// that grows with the plan (10% → 15% → 25% → 40%). The sending pipeline
// reads each day's cap, rate and audience tiers with Allowance.
package warmup

import (
	"encoding/json"
	"math"
	"slices"
	"strings"
)

// Engagement tiers in order of engagement.
var Tiers = []string{"active", "cooling", "at_risk", "sunset"}

// Day is one day of a plan's schedule.
type Day struct {
	Day    int     `json:"day"`
	Volume int64   `json:"volume"`
	Rate   float64 `json:"rate,omitempty"`
	// Share is the provider's Broadcast share in percent from this day.
	Share int `json:"share,omitempty"`
	// Tiers are the engagement tiers the day may send to (engaged first).
	Tiers []string `json:"tiers,omitempty"`
}

// Result is how one of the workspace's days went: the plan's day on that
// date, its cap, what went out and how the day was judged (PR-8). A plan
// day that didn't count appears again on a later date.
type Result struct {
	Day  int    `json:"day"`
	Date string `json:"date"`
	Cap  int64  `json:"cap"`
	// PlannedCap is the schedule's cap when a slowdown held a lower one.
	PlannedCap int64 `json:"plannedCap,omitempty"`
	Sent       int64 `json:"sent"`
	// Bounced, Complained and Delivered are the day's counts; complaints
	// and deliveries count only mailbox providers that report complaints.
	Bounced    int64 `json:"bounced,omitempty"`
	Complained int64 `json:"complained,omitempty"`
	Delivered  int64 `json:"delivered,omitempty"`
	// Bounce and Complaint are percentages; nil when nothing was measured.
	Bounce    *float64 `json:"bounce,omitempty"`
	Complaint *float64 `json:"complaint,omitempty"`
	// Counted: the day moved the plan on. Judged: the guards had enough
	// messages to judge it. Short: too little of the cap went out.
	Counted bool `json:"counted"`
	Judged  bool `json:"judged"`
	Short   bool `json:"short,omitempty"`
	// Slowed: a guard tripped on this day (Metric says which).
	Slowed bool   `json:"slowed,omitempty"`
	Metric string `json:"metric,omitempty"`
	// Held: a slowdown held this day's cap; Healthy is the healthy day it
	// was under the hold (1 of 2, 2 of 2), 0 otherwise.
	Held    bool `json:"held,omitempty"`
	Healthy int  `json:"healthy,omitempty"`
}

// Plan is a parsed schedule with its progress.
type Plan struct {
	Schedule []Day
	Progress []Result
}

// ParsePlan decodes the schedule and progress columns. Progress rows from
// before counted days (placeholders without a date) are dropped.
func ParsePlan(schedule, progress json.RawMessage) Plan {
	var p Plan
	_ = json.Unmarshal(schedule, &p.Schedule)
	var all []Result
	_ = json.Unmarshal(progress, &all)
	for _, r := range all {
		if r.Date != "" {
			p.Progress = append(p.Progress, r)
		}
	}
	slices.SortFunc(p.Schedule, func(a, b Day) int { return a.Day - b.Day })
	slices.SortFunc(p.Progress, func(a, b Result) int { return strings.Compare(a.Date, b.Date) })
	return p
}

// day returns the schedule entry for a day (the last one past the end).
func (p Plan) day(d int) Day {
	if len(p.Schedule) == 0 {
		return Day{Day: d}
	}
	for _, s := range p.Schedule {
		if s.Day == d {
			return s
		}
	}
	if d > p.Schedule[len(p.Schedule)-1].Day {
		return p.Schedule[len(p.Schedule)-1]
	}
	return p.Schedule[0]
}

// OnDate returns the result recorded for a date ("2006-01-02").
func (p Plan) OnDate(date string) (Result, bool) {
	for _, r := range p.Progress {
		if r.Date == date {
			return r, true
		}
	}
	return Result{}, false
}

// Counted is how many days have counted.
func (p Plan) Counted() int {
	n := 0
	for _, r := range p.Progress {
		if r.Counted {
			n++
		}
	}
	return n
}

// set records a date's result, replacing an earlier one.
func (p *Plan) set(r Result) {
	for i := range p.Progress {
		if p.Progress[i].Date == r.Date {
			p.Progress[i] = r
			return
		}
	}
	p.Progress = append(p.Progress, r)
}

// PlannedCap is the cap the schedule plans for a day.
func (p Plan) PlannedCap(d int) int64 { return p.day(d).Volume }

// Cap is the cap the schedule plans for a day. Today's cap (a slowdown's
// held cap, a pause, the caps per mailbox provider) comes from
// AllowanceFor.
func (p Plan) Cap(d int) int64 { return p.PlannedCap(d) }

// Rate is a day's rate limit per second.
func (p Plan) Rate(d int) float64 {
	day := p.day(d)
	if day.Rate > 0 {
		return day.Rate
	}
	return RateFor(p.PlannedCap(d))
}

// Share is the provider's Broadcast share on a day, in percent.
func (p Plan) Share(d int) int {
	share := 0
	for _, s := range p.Schedule {
		if s.Day > d {
			break
		}
		if s.Share > 0 {
			share = s.Share
		}
	}
	return share
}

// TiersOn is the audience of a day.
func (p Plan) TiersOn(d int) []string {
	t := p.day(d).Tiers
	if len(t) == 0 {
		return Tiers
	}
	return t
}

// RateFor is the default rate for a daily cap: spread over about an hour of
// sending at the start, never under 1 /s.
func RateFor(cap int64) float64 {
	return math.Max(1, math.Round(float64(cap)/3000))
}

// Ramp builds a schedule of n days from first to last emails per day,
// growing geometrically, with the engaged tiers first and the Broadcast
// share stepping up 10 → 15 → 25 → 40%.
func Ramp(n int, first, last int64) []Day {
	if n < 1 {
		n = 1
	}
	if first < 1 {
		first = 200
	}
	if last < first {
		last = first
	}
	out := make([]Day, n)
	growth := 1.0
	if n > 1 {
		growth = math.Pow(float64(last)/float64(first), 1/float64(n-1))
	}
	for i := 0; i < n; i++ {
		v := float64(first) * math.Pow(growth, float64(i))
		vol := roundNice(int64(v))
		if i == n-1 {
			vol = last
		}
		d := Day{Day: i + 1, Volume: vol, Rate: RateFor(vol)}
		frac := float64(i+1) / float64(n)
		switch {
		case frac <= 0.25:
			d.Tiers = []string{"active"}
		case frac <= 0.5:
			d.Tiers = []string{"active", "cooling"}
		case i < n-1:
			d.Tiers = []string{"active", "cooling", "at_risk"}
		default:
			d.Tiers = Tiers
		}
		switch {
		case i == n-1:
			d.Share = 40
		case frac > 0.75:
			d.Share = 25
		case frac > 0.55:
			d.Share = 15
		default:
			d.Share = 10
		}
		out[i] = d
	}
	return out
}

func roundNice(v int64) int64 {
	switch {
	case v < 1000:
		return (v + 50) / 100 * 100
	case v < 10000:
		return (v + 250) / 500 * 500
	default:
		return (v + 500) / 1000 * 1000
	}
}
