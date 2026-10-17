package scoring

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

var ny, _ = time.LoadLocation("America/New_York")

// now is the brief's sample time: Wed 15 Oct 2026, 10:30 New York.
var now = time.Date(2026, 10, 15, 10, 30, 0, 0, ny)

func day(d, h int) time.Time { return time.Date(2026, 10, 15-d, h, 0, 0, 0, ny) }

func rule(kind string, pts int, capKind string, capPts int) ScoringRule {
	return ScoringRule{ID: uuid.Must(uuid.NewV7()).String(), Kind: kind, Points: pts, Cap: ScoringRuleCap{Kind: capKind, Points: capPts}, Enabled: true, Values: []string{}}
}

func cfg(rules ...ScoringRule) ScoringConfig {
	return ScoringConfig{Rules: rules, Decay: ScoringDecay{Enabled: false, AfterWeeks: 4, Percent: 10}, Bands: DefaultBands}
}

func open(at time.Time) Event { return Event{At: at, Kind: KindOpen, Counted: true} }
func click(at time.Time, url string) Event {
	return Event{At: at, Kind: KindClick, Ref: url, Counted: true}
}

func TestOpensCapPerWeek(t *testing.T) {
	// +2 per human open, at most 6 points a week (Monday to Sunday).
	e := NewEngine(cfg(rule(KindOpen, 2, CapPointsWeek, 6)), ny)
	// Mon 13 Oct to Wed 15 Oct: four opens this week, one last week.
	evs := []Event{open(day(8, 9)), open(day(2, 9)), open(day(2, 12)), open(day(1, 9)), open(day(0, 8))}
	r := e.Replay(evs, now, 0)
	if r.Score != 8 {
		t.Fatalf("score = %d, want 8 (2 last week + 6 capped this week)", r.Score)
	}
	tl := r.Rules[0]
	if tl.Count != 5 || tl.Counted != 4 || tl.Points != 8 {
		t.Fatalf("tally = %+v", tl)
	}
	// 15 Oct 2026 is a Thursday: the week started on Monday 12 Oct.
	if tl.CapReached == nil || !tl.CapReached.Equal(time.Date(2026, 10, 12, 0, 0, 0, 0, ny)) {
		t.Fatalf("cap reached = %v, want the week of Mon 12 Oct", tl.CapReached)
	}
}

func TestOneEventCountsForEveryRuleItMatches(t *testing.T) {
	pricing := rule(KindClick, 15, CapOnceMonth, 0)
	pricing.Contains = "acme.io/pricing"
	e := NewEngine(cfg(rule(KindClick, 5, CapPointsWeek, 20), pricing), ny)
	r := e.Replay([]Event{click(day(6, 10), "https://acme.io/pricing?plan=team"), click(day(5, 10), "https://ACME.io/Pricing")}, now, 0)
	// Both clicks give +5; only the first pricing click of October gives +15.
	if r.Score != 25 {
		t.Fatalf("score = %d, want 25", r.Score)
	}
	if r.Rules[1].Points != 15 || r.Rules[1].CapReached == nil {
		t.Fatalf("pricing tally = %+v", r.Rules[1])
	}
	// Next month the pricing rule counts again.
	r = e.Replay([]Event{click(day(20, 10), "https://acme.io/pricing"), click(day(6, 10), "https://acme.io/pricing")}, now, 0)
	if r.Score != 40 {
		t.Fatalf("score = %d, want 40 (5+15 in September, 5+15 in October)", r.Score)
	}
}

func TestOnceCap(t *testing.T) {
	f := rule(KindForm, 20, CapOnce, 0)
	f.FormID = "01a14479-5832-73bd-8703-a2169da63724"
	e := NewEngine(cfg(f), ny)
	ev := func(at time.Time) Event { return Event{At: at, Kind: KindForm, Ref: f.FormID, Counted: true} }
	r := e.Replay([]Event{ev(day(30, 9)), ev(day(3, 9)), {At: day(2, 9), Kind: KindForm, Ref: "other", Counted: true}}, now, 0)
	if r.Score != 20 || r.Rules[0].Count != 2 || r.Rules[0].CapReached == nil {
		t.Fatalf("score = %d, tally = %+v", r.Score, r.Rules[0])
	}
}

func TestPointsCapGivesWhatIsLeft(t *testing.T) {
	e := NewEngine(cfg(rule(KindClick, 5, CapPointsDay, 7)), ny)
	r := e.Replay([]Event{click(day(1, 9), "a"), click(day(1, 10), "b"), click(day(1, 11), "c")}, now, 0)
	if r.Score != 7 {
		t.Fatalf("score = %d, want 7", r.Score)
	}
}

func TestScoreStaysBetween0And100(t *testing.T) {
	e := NewEngine(cfg(rule(KindAPIEvent, 50, CapNone, 0), rule(KindUnsubscribe, -20, CapNone, 0)), ny)
	e.rules[0].Event = "trial_started"
	api := func(at time.Time) Event {
		return Event{At: at, Kind: KindAPIEvent, Ref: "trial_started", Counted: true}
	}
	unsub := func(at time.Time) Event {
		return Event{At: at, Kind: KindUnsubscribe, Ref: uuid.NewString(), Counted: true}
	}
	r := e.Replay([]Event{api(day(9, 9)), api(day(8, 9)), api(day(7, 9))}, now, 0)
	if r.Score != 100 || !r.Rules[0].AtMax || r.Rules[0].Points != 100 {
		t.Fatalf("score = %d, tally = %+v", r.Score, r.Rules[0])
	}
	r = e.Replay([]Event{unsub(day(9, 9)), api(day(8, 9))}, now, 0)
	// The unsubscribe can't take the score below 0; it takes nothing.
	if r.Score != 50 || r.Rules[1].Points != 0 {
		t.Fatalf("score = %d, unsubscribe tally = %+v", r.Score, r.Rules[1])
	}
	r = e.Replay([]Event{api(day(9, 9)), unsub(day(8, 9))}, now, 0)
	if r.Score != 30 || r.Rules[1].Points != -20 {
		t.Fatalf("score = %d, unsubscribe tally = %+v", r.Score, r.Rules[1])
	}
}

func TestDecayAfterWeeksWithoutActivity(t *testing.T) {
	c := cfg(rule(KindClick, 10, CapNone, 0))
	c.Decay = ScoringDecay{Enabled: true, AfterWeeks: 4, Percent: 10}
	e := NewEngine(c, ny)
	// Five clicks 40 days ago: 50 points; decay at day 28 and day 35.
	start := now.AddDate(0, 0, -40)
	evs := []Event{}
	for i := 0; i < 5; i++ {
		evs = append(evs, click(start.Add(time.Duration(i)*time.Minute), "x"))
	}
	r := e.Replay(evs, now, 0)
	// 50 → 45 → 40.
	if r.Score != 40 || r.DecayLost != -10 {
		t.Fatalf("score = %d, decay = %d", r.Score, r.DecayLost)
	}
	want := evs[4].At.Add(6 * 7 * 24 * time.Hour)
	if r.NextCheck == nil || !r.NextCheck.Equal(want) {
		t.Fatalf("next check = %v, want %v", r.NextCheck, want)
	}
	// A click last week resets the clock: no decay since.
	evs = append(evs, click(now.AddDate(0, 0, -7), "y"))
	r = e.Replay(evs, now, 0)
	if r.Score != 55 {
		t.Fatalf("score = %d, want 45 (decayed once before the click), then +10 = 55", r.Score)
	}
}

func TestDecayTakesAShareOfNewPointsOnce(t *testing.T) {
	c := cfg(rule(KindClick, 10, CapNone, 0), rule(KindAPIEvent, 20, CapNone, 0))
	c.Rules[1].Event = "trial_started"
	c.Decay = ScoringDecay{Enabled: true, AfterWeeks: 4, Percent: 10}
	e := NewEngine(c, ny)
	clickAt := now.AddDate(0, 0, -100)
	apiAt := now.AddDate(0, 0, -3)
	r := e.Replay([]Event{click(clickAt, "x"), {At: apiAt, Kind: KindAPIEvent, Ref: "trial_started", Counted: true}}, now, 0)
	// The click's 10 decayed to 0 long ago. The API event's 20 came while
	// the contact was quiet: the next weekly step, a day later, takes 10%.
	if r.Score != 18 {
		t.Fatalf("score = %d, want 18", r.Score)
	}
}

func TestLookback(t *testing.T) {
	e := NewEngine(cfg(rule(KindOpen, 2, CapNone, 0)), ny)
	old := now.AddDate(0, 0, -181)
	r := e.Replay([]Event{open(old), open(now.AddDate(0, 0, -179))}, now, 0)
	if r.Score != 2 || len(r.Events) != 1 {
		t.Fatalf("score = %d, events = %d", r.Score, len(r.Events))
	}
	// The remaining open leaves the window on day 180.
	want := now.AddDate(0, 0, -179).AddDate(0, 0, LookbackDays)
	if r.NextCheck == nil || !r.NextCheck.Equal(want) {
		t.Fatalf("next check = %v, want %v", r.NextCheck, want)
	}
}

func TestSoftBounceDeliveredAfterRetryCountsNothing(t *testing.T) {
	e := NewEngine(cfg(rule(KindOpen, 2, CapNone, 0), rule(KindSoftBounce, -5, CapNone, 0)), ny)
	r := e.Replay([]Event{open(day(10, 9)), {At: day(9, 9), Kind: KindSoftBounce, Counted: false}}, now, 0)
	if r.Score != 2 || r.Rules[1].Zero != 1 || r.Rules[1].Points != 0 {
		t.Fatalf("score = %d, tally = %+v", r.Score, r.Rules[1])
	}
	r = e.Replay([]Event{open(day(10, 9)), {At: day(9, 9), Kind: KindSoftBounce, Counted: true}}, now, 0)
	if r.Score != 0 || r.Rules[1].Points != -2 {
		t.Fatalf("score = %d, tally = %+v", r.Score, r.Rules[1])
	}
}

func TestOpensOfOneList(t *testing.T) {
	list := uuid.Must(uuid.NewV7())
	r1 := rule(KindOpen, 3, CapNone, 0)
	r1.ListID = list.String()
	e := NewEngine(cfg(r1), ny)
	r := e.Replay([]Event{{At: day(3, 9), Kind: KindOpen, Lists: []uuid.UUID{list}, Counted: true}, {At: day(2, 9), Kind: KindOpen, Lists: []uuid.UUID{uuid.Must(uuid.NewV7())}, Counted: true}}, now, 0)
	if r.Score != 3 {
		t.Fatalf("score = %d, want 3", r.Score)
	}
}

func TestDisabledRulesCountNothing(t *testing.T) {
	r := rule(KindOpen, 2, CapNone, 0)
	r.Enabled = false
	e := NewEngine(cfg(r), ny)
	if got := e.Replay([]Event{open(day(1, 9))}, now, 10).Score; got != 10 {
		t.Fatalf("score = %d, want only the fields part", got)
	}
}

func TestFieldsPartIsAddedAndKeptWithin100(t *testing.T) {
	e := NewEngine(cfg(rule(KindOpen, 50, CapNone, 0)), ny)
	r := e.Replay([]Event{open(day(2, 9)), open(day(1, 9))}, now, 10)
	if r.EventsPart != 100 || r.Score != 100 {
		t.Fatalf("events = %d, score = %d", r.EventsPart, r.Score)
	}
	if got := e.Replay(nil, now, -10).Score; got != 0 {
		t.Fatalf("a negative fields part alone gives %d, want 0", got)
	}
}

// Sam Rivera's sample (brief §4): 11 human opens, 4 clicks, plan is team,
// a soft bounce delivered after a retry: 52, Hot since his click on Thu 9 Oct.
func TestSamRivera(t *testing.T) {
	c := cfg(rule(KindOpen, 2, CapPointsWeek, 6), rule(KindClick, 5, CapPointsWeek, 20), rule(KindSoftBounce, -5, CapNone, 0))
	c.Decay = ScoringDecay{Enabled: true, AfterWeeks: 4, Percent: 10}
	e := NewEngine(c, ny)
	var evs []Event
	// Weekly opens from the Brevo history (#131–#138), then #139's soft
	// bounce, #140 and #141, the welcome email, and October's click.
	for _, d := range []int{75, 68, 61, 54, 47, 40, 33, 26} {
		evs = append(evs, open(day(d, 9)))
	}
	for _, d := range []int{68, 47, 26} {
		evs = append(evs, click(day(d, 10), "https://acme.io/blog"))
	}
	evs = append(evs, open(day(23, 10)), Event{At: day(19, 9), Kind: KindSoftBounce, Counted: false}, open(day(12, 10)), open(day(5, 9)))
	evs = append(evs, click(day(3, 10), "https://docs.acme.io/flags"))
	slicesSort(evs)
	r := e.Replay(evs, now, 10)
	if r.Score != 52 {
		t.Fatalf("score = %d, want 52 (opens %d, clicks %d)", r.Score, r.Rules[0].Points, r.Rules[1].Points)
	}
	if r.Rules[0].Points != 22 || r.Rules[1].Points != 20 || r.Rules[2].Points != 0 || r.Rules[2].Zero != 1 || r.DecayLost != 0 {
		t.Fatalf("breakdown = %+v, decay %d", r.Rules, r.DecayLost)
	}
	last := r.Steps[len(r.Steps)-1]
	if last.After+10 != 52 || last.Before+10 != 47 {
		t.Fatalf("last change %d → %d", last.Before+10, last.After+10)
	}
	since := BandSince(r, DefaultBands)
	if since == nil || !since.Equal(day(3, 10)) {
		t.Fatalf("hot since %v, want Thu 9 Oct 10:00", since)
	}
}

func TestBands(t *testing.T) {
	b := DefaultBands
	for score, want := range map[int]string{0: BandCold, 19: BandCold, 20: BandWarm, 49: BandWarm, 50: BandHot, 100: BandHot} {
		if got := b.Band(score); got != want {
			t.Errorf("band(%d) = %s, want %s", score, got, want)
		}
	}
}

func slicesSort(evs []Event) {
	for i := 1; i < len(evs); i++ {
		for j := i; j > 0 && evs[j].At.Before(evs[j-1].At); j-- {
			evs[j], evs[j-1] = evs[j-1], evs[j]
		}
	}
}
