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

// Result is what happened on a finished day.
type Result struct {
	Day       int     `json:"day"`
	Date      string  `json:"date"`
	Cap       int64   `json:"cap"`
	Sent      int64   `json:"sent"`
	Bounce    float64 `json:"bounce"`
	Complaint float64 `json:"complaint"`
	// Slowed: a guard tripped on this day.
	Slowed bool `json:"slowed,omitempty"`
	// Held: this day's cap was held at the previous day's level.
	Held bool `json:"held,omitempty"`
}

// Plan is a parsed schedule with its progress.
type Plan struct {
	Schedule []Day
	Progress []Result
}

// ParsePlan decodes the schedule and progress columns.
func ParsePlan(schedule, progress json.RawMessage) Plan {
	var p Plan
	_ = json.Unmarshal(schedule, &p.Schedule)
	_ = json.Unmarshal(progress, &p.Progress)
	slices.SortFunc(p.Schedule, func(a, b Day) int { return a.Day - b.Day })
	slices.SortFunc(p.Progress, func(a, b Result) int { return a.Day - b.Day })
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

// Result returns the recorded result of a day.
func (p Plan) Result(d int) (Result, bool) {
	for _, r := range p.Progress {
		if r.Day == d {
			return r, true
		}
	}
	return Result{}, false
}

// Cap is a day's volume cap: the planned volume, or the cap a slowdown set
// for it (the warm-up job records it when the guard trips).
func (p Plan) Cap(d int) int64 {
	if r, ok := p.Result(d); ok && r.Cap > 0 {
		return r.Cap
	}
	return p.day(d).Volume
}

// PlannedCap is the cap the schedule planned for a day.
func (p Plan) PlannedCap(d int) int64 { return p.day(d).Volume }

// Rate is a day's rate limit per second.
func (p Plan) Rate(d int) float64 {
	day := p.day(d)
	if day.Rate > 0 {
		return day.Rate
	}
	return RateFor(p.Cap(d))
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
