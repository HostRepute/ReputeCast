package scoring

import (
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// A score is a replay: the contact's events of the last 180 days, in time
// order, under the rules. Each event adds the points of every enabled rule
// it matches, within the rule's cap; the running score stays between 0 and
// 100. After a stretch without a human open or click the score loses a
// share each week (decay). Field conditions add their points on top while
// they are true. Replaying the same events always gives the same score, so
// the incremental scorer, the nightly run, a recalculation and the impact
// preview agree, and the profile's breakdown and 90-day line come from the
// same replay.

// Event is one thing a contact did that rules can count.
type Event struct {
	At   time.Time
	Kind string // open, click, form, api_event, soft_bounce, unsubscribe
	// Ref is the link (click), form id, API event name or list id (unsubscribe).
	Ref string
	// Lists are the lists the email went to (open, click).
	Lists []uuid.UUID
	// Counted is false for a soft bounce delivered after a retry.
	Counted bool
	// Message is the email the event is about (open, click, soft bounce).
	Message uuid.UUID
}

// activity reports whether the event keeps the score from decaying.
func (e Event) activity() bool { return e.Kind == KindOpen || e.Kind == KindClick }

// Engine replays events under one configuration in one timezone.
type Engine struct {
	cfg   ScoringConfig
	rules []compiledRule
	loc   *time.Location
}

type compiledRule struct {
	ScoringRule
	index    int
	list     uuid.UUID
	contains string
}

// NewEngine compiles a configuration. Calendar caps (a day, a week from
// Monday, a month) follow loc, the workspace timezone.
func NewEngine(cfg ScoringConfig, loc *time.Location) *Engine {
	if loc == nil {
		loc = time.UTC
	}
	e := &Engine{cfg: cfg, loc: loc}
	for i, r := range cfg.Rules {
		if !r.Enabled || !r.IsEvent() {
			continue
		}
		c := compiledRule{ScoringRule: r, index: i, contains: strings.ToLower(r.Contains)}
		if r.ListID != "" {
			c.list, _ = uuid.Parse(r.ListID)
		}
		e.rules = append(e.rules, c)
	}
	return e
}

// Config returns the configuration the engine runs.
func (e *Engine) Config() ScoringConfig { return e.cfg }

func (r compiledRule) matches(ev Event) bool {
	if r.Kind != ev.Kind {
		return false
	}
	switch r.Kind {
	case KindOpen:
		return r.list == uuid.Nil || slices.Contains(ev.Lists, r.list)
	case KindClick:
		return r.contains == "" || strings.Contains(strings.ToLower(ev.Ref), r.contains)
	case KindForm:
		return strings.EqualFold(r.FormID, ev.Ref)
	case KindAPIEvent:
		return r.Event == ev.Ref
	case KindSoftBounce:
		return true
	case KindUnsubscribe:
		return r.list == uuid.Nil || strings.EqualFold(r.ListID, ev.Ref)
	}
	return false
}

// RuleTally is what one rule added in a replay.
type RuleTally struct {
	RuleID string
	// Points is what the rule actually changed the score by: within its
	// cap, the score's 0–100 range and after nothing else.
	Points int
	// Count is how many events matched; Counted how many changed the score.
	Count   int
	Counted int
	// Zero counts matching events that gave nothing because they don't
	// count (a soft bounce delivered after a retry).
	Zero     int
	LastZero *Event
	// CapReached is the start of the latest period where the cap stopped
	// points (or the first event, for "once").
	CapReached *time.Time
	// AtMax is true when 100 stopped some of its points.
	AtMax  bool
	First  *time.Time
	Latest *time.Time
}

// Step is one change of the events part: an event's rules, or decay.
type Step struct {
	At     time.Time
	Before int
	After  int
	// Event is the index in the replayed events, -1 for decay.
	Event int
	Rules []string
}

// Result is a replay's outcome at its end time.
type Result struct {
	// EventsPart is the score from events after caps and decay (0–100).
	EventsPart int
	// FieldsPart is the sum of the true field conditions' points.
	FieldsPart int
	// Score is EventsPart + FieldsPart, kept between 0 and 100.
	Score int
	// ActiveAt is the last human open or click in the look-back.
	ActiveAt *time.Time
	// NextCheck is when the score changes next without new activity: the
	// next decay step, or the oldest event leaving the look-back.
	NextCheck *time.Time
	Rules     []RuleTally // one per enabled event rule, in rule order
	// DecayLost is what decay took (0 or negative).
	DecayLost int
	Steps     []Step
	// Events are the replayed events (in the look-back), in time order.
	Events []Event
}

// ChangedAt is when the score last changed, if it ever did.
func (r Result) ChangedAt() *time.Time {
	if len(r.Steps) == 0 {
		return nil
	}
	t := r.Steps[len(r.Steps)-1].At
	return &t
}

// Total adds a fields part to an events part, kept between 0 and MaxScore.
func Total(events, fields int) int { return clamp(events + fields) }

func clamp(n int) int {
	if n < 0 {
		return 0
	}
	if n > MaxScore {
		return MaxScore
	}
	return n
}

// period returns the start of the cap period an event falls in.
func (e *Engine) period(kind string, at time.Time) time.Time {
	l := at.In(e.loc)
	switch kind {
	case CapOnceDay, CapPointsDay:
		return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, e.loc)
	case CapOnceWeek, CapPointsWeek:
		wd := (int(l.Weekday()) + 6) % 7 // Monday = 0
		d := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, e.loc)
		return d.AddDate(0, 0, -wd)
	case CapOnceMonth, CapPointsMonth:
		return time.Date(l.Year(), l.Month(), 1, 0, 0, 0, 0, e.loc)
	}
	return time.Time{}
}

type capKey struct {
	rule   int
	period time.Time
}

// Replay scores events at time now. Events outside (now−180 days, now] are
// left out; events must be in time order. fieldsPart is the sum of the
// contact's true field conditions.
func (e *Engine) Replay(events []Event, now time.Time, fieldsPart int) Result {
	from := now.AddDate(0, 0, -LookbackDays)
	res := Result{FieldsPart: fieldsPart, Rules: make([]RuleTally, len(e.rules))}
	for i, r := range e.rules {
		res.Rules[i].RuleID = r.ID
	}
	used := map[capKey]int{} // points (or events, for once_*) used per rule and period
	onceDone := map[int]bool{}
	score := 0
	var ref *time.Time // decay counts from the last activity, else the first change
	var nextDecay time.Time
	decay := e.cfg.Decay
	week := 7 * 24 * time.Hour
	resetDecay := func(t time.Time) {
		ref = &t
		nextDecay = t.Add(time.Duration(decay.AfterWeeks) * week)
	}
	// Decay steps fall every week from N weeks after the reference, whether
	// or not there is anything left to take, so points earned while a
	// contact is quiet lose a share at the next step, not every step missed.
	applyDecay := func(until time.Time) {
		if !decay.Enabled || ref == nil {
			return
		}
		for !nextDecay.After(until) {
			if score > 0 {
				before := score
				score = score * (100 - decay.Percent) / 100
				res.DecayLost += score - before
				res.Steps = append(res.Steps, Step{At: nextDecay, Before: before, After: score, Event: -1})
			}
			nextDecay = nextDecay.Add(week)
		}
	}
	for _, ev := range events {
		if !ev.At.After(from) || ev.At.After(now) {
			continue
		}
		applyDecay(ev.At)
		idx := len(res.Events)
		res.Events = append(res.Events, ev)
		before := score
		var ruleIDs []string
		for i, r := range e.rules {
			if !r.matches(ev) {
				continue
			}
			t := &res.Rules[i]
			t.Count++
			at := ev.At
			if t.First == nil {
				t.First = &at
			}
			t.Latest = &at
			if !ev.Counted {
				t.Zero++
				evc := ev
				t.LastZero = &evc
				continue
			}
			pts := r.Points
			switch r.Cap.Kind {
			case CapOnce:
				if onceDone[i] {
					capped := *t.First
					t.CapReached = &capped
					continue
				}
				onceDone[i] = true
			case CapOnceDay, CapOnceWeek, CapOnceMonth:
				k := capKey{i, e.period(r.Cap.Kind, ev.At)}
				if used[k] > 0 {
					p := k.period
					t.CapReached = &p
					continue
				}
				used[k] = 1
			case CapPointsDay, CapPointsWeek, CapPointsMonth:
				k := capKey{i, e.period(r.Cap.Kind, ev.At)}
				mag := pts
				if mag < 0 {
					mag = -mag
				}
				left := r.Cap.Points - used[k]
				if left <= 0 {
					p := k.period
					t.CapReached = &p
					continue
				}
				if mag > left {
					p := k.period
					t.CapReached = &p
					mag = left
					if pts < 0 {
						pts = -left
					} else {
						pts = left
					}
				}
				used[k] += mag
			}
			next := clamp(score + pts)
			if next-score != pts && pts > 0 {
				t.AtMax = true
			}
			applied := next - score
			score = next
			t.Points += applied
			if applied != 0 {
				t.Counted++
				ruleIDs = append(ruleIDs, r.ID)
			}
		}
		if score != before {
			res.Steps = append(res.Steps, Step{At: ev.At, Before: before, After: score, Event: idx, Rules: ruleIDs})
			if ref == nil {
				resetDecay(ev.At)
			}
		}
		if ev.activity() {
			at := ev.At
			res.ActiveAt = &at
			resetDecay(ev.At)
		}
	}
	applyDecay(now)
	res.EventsPart = score
	res.Score = Total(score, fieldsPart)
	// The next change without new activity.
	var next *time.Time
	if decay.Enabled && ref != nil && score > 0 {
		t := nextDecay
		next = &t
	}
	if len(res.Events) > 0 {
		out := res.Events[0].At.AddDate(0, 0, LookbackDays)
		if next == nil || out.Before(*next) {
			next = &out
		}
	}
	res.NextCheck = next
	return res
}

// BandSince is when the score last entered its current band, judging by
// the replay's steps with the fields part added (nil when it never left
// the band it started in).
func BandSince(res Result, bands ScoringBands) *time.Time {
	band := bands.Band(res.Score)
	var since *time.Time
	prev := bands.Band(Total(0, res.FieldsPart))
	for _, s := range res.Steps {
		b := bands.Band(Total(s.After, res.FieldsPart))
		if b == band && prev != band {
			at := s.At
			since = &at
		}
		prev = b
	}
	return since
}
