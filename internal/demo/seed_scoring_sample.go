package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/scoring"
)

// Contact scoring (audience-insights brief §4): the eight rules Sofia
// Rossi set up on Monday, and every contact's score built from events that
// replay to it. The spread, the bands by tier, the named contacts, the
// preview of rule 3 at +25 (1,108 Warm → Hot, 204 Cold → Warm) and the 37
// contacts "Sales-ready handoff" started for all follow from the events;
// the seeder checks each replay against its target.

// sgScore is what the seeder decides about one contact's score.
type sgScore struct {
	set     bool
	target  int
	fields  int
	role    string // pricing, mover, entrant, trial…
	noClick bool
	trial   bool
	// trialLive: trial_started arrived after "Sales-ready handoff" went
	// live and took the score past 50.
	trialLive bool
	form      bool
	events    []scoring.Event
	res       scoring.Result
}

// sgBuckets are the spread's ten buckets.
var sgBuckets = [][2]int{{0, 9}, {10, 19}, {20, 29}, {30, 39}, {40, 49}, {50, 59}, {60, 69}, {70, 79}, {80, 89}, {90, 100}}

// sgCell is how many contacts of a tier land in a bucket, paid plans
// (team, business) and the rest.
type sgCell struct{ paid, other int }

// sgSpread is the board's spread split by tier (ScoringRules: bands by
// tier, buckets) with the paid plans placed so 1,532 Hot contacts are on
// one ("Hot on paid plans").
var sgSpread = map[string][10]sgCell{
	"active":  {{0, 4291}, {700, 6759}, {600, 5467}, {450, 3647}, {341, 2455}, {592, 1294}, {380, 802}, {260, 537}, {160, 337}, {92, 248}},
	"cooling": {{0, 5300}, {400, 2872}, {180, 620}, {60, 240}, {20, 78}, {34, 46}, {10, 15}, {4, 5}, {}, {}},
	"at_risk": {{0, 5200}, {13, 40}, {0, 45}, {0, 4}, {}, {}, {}, {}, {}, {}},
	"sunset":  {{0, 3615}, {}, {}, {}, {}, {}, {}, {}, {}, {}},
}

// sgPricing is how many Active contacts in each bucket clicked a link to
// acme.io/pricing once in the last month: the ones rule 3 at +25 moves.
var sgPricing = [10]int{0, 204, 379, 643, 1108, 421, 291, 181, 120, 60}

// The rule ids stay the same across versions.
var sgRuleIDs = []string{
	"019a0c4e-5a10-7000-8000-000000000001", "019a0c4e-5a10-7000-8000-000000000002", "019a0c4e-5a10-7000-8000-000000000003",
	"019a0c4e-5a10-7000-8000-000000000004", "019a0c4e-5a10-7000-8000-000000000005", "019a0c4e-5a10-7000-8000-000000000006",
	"019a0c4e-5a10-7000-8000-000000000007", "019a0c4e-5a10-7000-8000-000000000008",
}

func (w *sgWorld) scoringConfig() scoring.ScoringConfig {
	r := func(i int, rule scoring.ScoringRule) scoring.ScoringRule {
		rule.ID, rule.Enabled = sgRuleIDs[i], true
		if rule.Values == nil {
			rule.Values = []string{}
		}
		return rule
	}
	return scoring.ScoringConfig{
		Rules: []scoring.ScoringRule{
			r(0, scoring.ScoringRule{Kind: scoring.KindOpen, Points: 2, Cap: scoring.ScoringRuleCap{Kind: scoring.CapPointsWeek, Points: 6}}),
			r(1, scoring.ScoringRule{Kind: scoring.KindClick, Points: 5, Cap: scoring.ScoringRuleCap{Kind: scoring.CapPointsWeek, Points: 20}}),
			r(2, scoring.ScoringRule{Kind: scoring.KindClick, Contains: "acme.io/pricing", Points: 15, Cap: scoring.ScoringRuleCap{Kind: scoring.CapOnceMonth}}),
			r(3, scoring.ScoringRule{Kind: scoring.KindForm, FormID: w.forms["Beta apply"].String(), Points: 20, Cap: scoring.ScoringRuleCap{Kind: scoring.CapOnce}}),
			r(4, scoring.ScoringRule{Kind: scoring.KindAPIEvent, Event: "trial_started", Points: 25, Cap: scoring.ScoringRuleCap{Kind: scoring.CapOnce}}),
			r(5, scoring.ScoringRule{Kind: scoring.KindField, Field: "plan", Op: scoring.OpIsOneOf, Values: []string{"team", "business"}, Points: 10, Cap: scoring.ScoringRuleCap{Kind: scoring.CapWhileTrue}}),
			r(6, scoring.ScoringRule{Kind: scoring.KindSoftBounce, Points: -5, Cap: scoring.ScoringRuleCap{Kind: scoring.CapNone}}),
			r(7, scoring.ScoringRule{Kind: scoring.KindUnsubscribe, Points: -20, Cap: scoring.ScoringRuleCap{Kind: scoring.CapNone}}),
		},
		Decay: scoring.DefaultDecay,
		Bands: scoring.DefaultBands,
	}
}

func (w *sgWorld) scoring(ctx context.Context) error {
	cfg := w.scoringConfig()
	eng := scoring.NewEngine(cfg, w.loc)
	sofia := w.s.Users["sofia@acme.io"]
	setUp := w.s.designDay(2, 11, 40)
	if err := w.assignScores(); err != nil {
		return err
	}
	for _, c := range w.contacts {
		if c.board == "erased" {
			continue
		}
		if err := w.compose(c, eng); err != nil {
			return fmt.Errorf("%s: %w", c.email, err)
		}
	}
	// Acme's app sends other events too; no rule counts them (the rule
	// editor lists trial_started 214 · demo_booked 38 · plan_upgraded 12).
	others := w.shuffled(func(c *sgContact) bool { return c.tier == "active" && c.board == "" && c.sc.role == "" && !c.sc.trial })
	for i, c := range others[:50] {
		name := "demo_booked"
		if i >= 38 {
			name = "plan_upgraded"
		}
		from := w.windowStart(c)
		if lim := w.now.AddDate(0, 0, -28); from.Before(lim) {
			from = lim
		}
		ev := scoring.Event{At: w.between(from, w.now.Add(-time.Hour)), Kind: scoring.KindAPIEvent, Ref: name, Counted: true}
		evs := append(slices.Clone(c.sc.events), ev)
		sort.SliceStable(evs, func(i, j int) bool { return evs[i].At.Before(evs[j].At) })
		res := eng.Replay(evs, w.now, c.sc.fields)
		if res.Score != c.sc.target {
			return fmt.Errorf("%s with %s: %d, not %d", c.email, name, res.Score, c.sc.target)
		}
		c.sc.events, c.sc.res = evs, res
	}
	raw, _ := json.Marshal(cfg)
	tx := w.s.Tx
	if _, err := tx.Exec(ctx, `INSERT INTO scoring_configs (workspace_id, version, applied_version, ingested_through, nightly_at, set_up_by, set_up_at, updated_at)
		VALUES ($1, 1, 1, $2, $2, $3, $4, $4)`, w.ws.ID, w.now, sofia.ID, setUp); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO scoring_versions (workspace_id, version, config, created_by, created_at) VALUES ($1, 1, $2, $3, $4)`,
		w.ws.ID, raw, sofia.ID, setUp); err != nil {
		return err
	}
	// The set-up's recalculation, finished a minute later.
	summary, _ := json.Marshal(map[string]any{"changed": 0, "moves": []any{}, "bands": map[string]int{"cold": 29190, "warm": 14207, "hot": 4816}, "automations": []any{}})
	changes, _ := json.Marshal(map[string]any{"setup": true, "rules": []int{}, "decay": false, "bands": false})
	if _, err := tx.Exec(ctx, `INSERT INTO scoring_recalcs (id, workspace_id, version, reason, status, start_automations, total, done, label, changes, started_by, started_at, finished_at, summary)
		VALUES ($1,$2,1,'setup','finished',false,48213,48213,'Set up with 8 rules',$3,$4,$5,$6,$7)`,
		ID(), w.ws.ID, changes, sofia.ID, setUp, setUp.Add(74*time.Second), summary); err != nil {
		return err
	}
	var scores, events [][]any
	for _, c := range w.contacts {
		if c.board == "erased" {
			continue
		}
		r := c.sc.res
		scores = append(scores, []any{c.id, w.ws.ID, int16(r.Score), int16(r.EventsPart), int16(r.FieldsPart), r.ActiveAt, r.ChangedAt(), r.NextCheck, int32(1), w.now})
		for _, ev := range c.sc.events {
			var msg any
			if ev.Message != uuid.Nil {
				msg = ev.Message
			}
			lists := ev.Lists
			if lists == nil {
				lists = []uuid.UUID{}
			}
			events = append(events, []any{w.ws.ID, c.id, uuid.New(), ev.At, ev.Kind, ev.Ref, lists, msg, ev.Counted})
		}
	}
	if err := sgCopy(ctx, tx, "contact_scores", []string{"contact_id", "workspace_id", "score", "events_part", "fields_part", "active_at", "changed_at",
		"next_check_at", "version", "computed_at"}, scores); err != nil {
		return err
	}
	return sgCopy(ctx, tx, "score_events", []string{"workspace_id", "contact_id", "source_id", "occurred_at", "kind", "ref", "lists", "message_id", "counted"}, events)
}

// lastActivity is the contact's last human open or click in the window
// events can use, and whether a click came last.
func (w *sgWorld) lastActivity(c *sgContact) (time.Time, *time.Time, bool) {
	limit := w.now.AddDate(0, 0, -170)
	var last time.Time
	var click *time.Time
	if c.lastHuman != nil && c.lastHuman.After(limit) {
		last = *c.lastHuman
	}
	if c.lastClick != nil && c.lastClick.After(limit) && !c.lastClick.Before(last) && c.lastClick.Sub(last) < 3*time.Hour && !last.IsZero() {
		t := *c.lastClick
		click = &t
	}
	return last, click, !last.IsZero()
}

// finalAt is when a contact's last activity event is: the click that came
// with its last email (W02's, or the pricing click three minutes after the
// open), else the last open. Decay counts from it.
func (w *sgWorld) finalAt(c *sgContact) time.Time {
	last, click, _ := w.lastActivity(c)
	switch {
	case click != nil:
		return *click
	case c.sc.role == "pricing" || c.sc.role == "mover":
		return last.Add(3 * time.Minute)
	}
	return last
}

// decaySteps counts the weekly decay steps between the last activity and now.
func (w *sgWorld) decaySteps(last time.Time) int {
	k := 0
	for t := last.Add(28 * 24 * time.Hour); !t.After(w.now); t = t.Add(7 * 24 * time.Hour) {
		k++
	}
	return k
}

func sgChain(p, k int) int {
	for i := 0; i < k; i++ {
		p = p * 90 / 100
	}
	return p
}

// windowStart is how far back a contact's events can go: imported contacts
// keep their history (within the look-back), the rest start when they joined.
func (w *sgWorld) windowStart(c *sgContact) time.Time {
	start := c.created
	last, _, ok := w.lastActivity(c)
	switch {
	case c.importID != nil || c.source == "import:mailchimp" || c.source == "import:brevo" || c.source == "import:csv" || c.created.Before(w.now.AddDate(0, 0, -171)):
		start = w.now.AddDate(0, 0, -171)
	case ok && c.created.After(last.AddDate(0, 0, -14)):
		// A few 1.0 signups opened an email dated before their join; give
		// them three weeks before their last open.
		start = last.AddDate(0, 0, -21)
	}
	// Never after the last activity (a day before it at the latest).
	if ok && start.After(last.Add(-24*time.Hour)) {
		start = last.Add(-24 * time.Hour)
	}
	return start
}

// capacity is the most a contact's events part can be today: the weeks of
// activity before its last one, decayed.
func (w *sgWorld) capacity(c *sgContact) int {
	last, _, ok := w.lastActivity(c)
	if !ok {
		return 0
	}
	weeks := int(last.Sub(w.windowStart(c)).Hours()/(24*7)) + 1
	p := min(100, weeks*24)
	return sgChain(p, w.decaySteps(w.finalAt(c)))
}

// assignScores picks every contact's target: the named ones, the 37 who
// crossed 50 since "Sales-ready handoff" went live, the pricing clickers,
// then the rest of the spread by tier, the strongest histories on top.
func (w *sgWorld) assignScores() error {
	for _, c := range w.contacts {
		if c.board == "erased" {
			continue
		}
		if c.paid {
			c.sc.fields = 10
		}
	}
	// The board's contacts.
	named := map[string]int{"sam": 52, "priya": 78, "diego": 12, "hannah": 47, "kenji": 18, "lena": 0, "omar": 64, "grace": 0, "tomasz": 21, "aisha": 91}
	left := map[string][10]sgCell{}
	for t, cells := range sgSpread {
		left[t] = cells
	}
	bucketOf := func(s int) int {
		for i, b := range sgBuckets {
			if s >= b[0] && s <= b[1] {
				return i
			}
		}
		return 0
	}
	useCell := func(c *sgContact, b int) error {
		cells := left[c.tier]
		if c.sc.fields > 0 {
			if cells[b].paid == 0 {
				return fmt.Errorf("%s: no paid slot in %s bucket %d", c.email, c.tier, b)
			}
			cells[b].paid--
		} else {
			if cells[b].other == 0 {
				return fmt.Errorf("%s: no slot in %s bucket %d", c.email, c.tier, b)
			}
			cells[b].other--
		}
		left[c.tier] = cells
		return nil
	}
	for key, s := range named {
		c := w.named(key)
		c.sc.set, c.sc.target, c.sc.role = true, s, "named:"+key
		if key == "kenji" {
			c.sc.noClick = true
		}
		if key == "aisha" {
			c.sc.trial, c.sc.form = true, true
		}
		if err := useCell(c, bucketOf(s)); err != nil {
			return err
		}
	}

	free := func(c *sgContact) bool { return !c.sc.set && c.board == "" }
	active := func(c *sgContact) bool { return free(c) && c.tier == "active" }

	// "Sales-ready handoff": 31 on a paid plan and 6 not, who went past 50
	// after it went live: with their last open or click, or when Acme's app
	// sent trial_started for them.
	recent := func(c *sgContact) bool {
		last, _, ok := w.lastActivity(c)
		return ok && last.After(w.liveSince.Add(10*time.Minute)) && last.Before(w.now.Add(-15*time.Minute)) && w.windowStart(c).Before(w.now.AddDate(0, 0, -28))
	}
	viaTrial := func(c *sgContact) bool {
		last, _, ok := w.lastActivity(c)
		return ok && last.After(w.now.AddDate(0, 0, -24)) && last.Before(w.liveSince.Add(-2*time.Hour)) && w.windowStart(c).Before(w.now.AddDate(0, 0, -35))
	}
	for _, want := range []struct {
		paid bool
		n    int
	}{{true, 31}, {false, 6}} {
		pool := w.shuffled(func(c *sgContact) bool { return active(c) && recent(c) && (c.sc.fields > 0) == want.paid })
		pool = pool[:min(len(pool), want.n*2/3)]
		for _, c := range w.shuffled(func(c *sgContact) bool { return active(c) && viaTrial(c) && (c.sc.fields > 0) == want.paid }) {
			if len(pool) == want.n {
				break
			}
			c.sc.trial, c.sc.trialLive = true, true
			pool = append(pool, c)
		}
		if len(pool) < want.n {
			return fmt.Errorf("entrants: %d of %d", len(pool), want.n)
		}
		for _, c := range pool {
			_, click, _ := w.lastActivity(c)
			hi := 51
			if click != nil || c.sc.trialLive {
				hi = 56
			}
			c.sc.set, c.sc.role, c.sc.target = true, "entrant", 50+w.rng.IntN(hi-49)
			if err := useCell(c, 5); err != nil {
				return err
			}
		}
	}

	// Pricing clickers: Active, without a paid plan, active in the last
	// 26 days; the Hot ones were Hot before the automation went live.
	for b := 9; b >= 1; b-- {
		n := sgPricing[b]
		pool := w.shuffled(func(c *sgContact) bool {
			if !active(c) || c.sc.fields > 0 {
				return false
			}
			last, _, ok := w.lastActivity(c)
			if !ok || last.Before(w.now.AddDate(0, 0, -26)) {
				return false
			}
			if b >= 5 && w.finalAt(c).Add(3*time.Minute).After(w.liveSince.Add(-time.Hour)) {
				return false
			}
			return w.capacity(c) >= sgBuckets[b][0]+5
		})
		sort.SliceStable(pool, func(i, j int) bool { return w.capacity(pool[i]) > w.capacity(pool[j]) })
		if len(pool) < n {
			return fmt.Errorf("pricing bucket %d: %d of %d", b, len(pool), n)
		}
		for _, c := range pool[:n] {
			c.sc.set, c.sc.role = true, "pricing"
			lo, hi := sgBuckets[b][0], sgBuckets[b][1]
			if b == 1 {
				c.sc.role = "mover"
			}
			c.sc.target = lo + w.rng.IntN(hi-lo+1)
			if err := useCell(c, b); err != nil {
				return err
			}
		}
	}

	// Everyone else, tier by tier: the strongest histories take the top
	// buckets. Hot contacts were Hot before the automation went live.
	for _, tier := range []string{"active", "cooling", "at_risk", "sunset"} {
		for _, paid := range []bool{true, false} {
			pool := w.shuffled(func(c *sgContact) bool { return free(c) && c.tier == tier && (c.sc.fields > 0) == paid })
			caps := map[*sgContact]int{}
			for _, c := range pool {
				caps[c] = w.capacity(c)
			}
			sort.SliceStable(pool, func(i, j int) bool { return caps[pool[i]] > caps[pool[j]] })
			cells := left[tier]
			i := 0
			for b := 9; b >= 0; b-- {
				n := cells[b].other
				if paid {
					n = cells[b].paid
				}
				for k := 0; k < n; k++ {
					// The next contact that can reach the bucket (Hot ones
					// quiet since before the automation went live).
					j := i
					for ; j < len(pool); j++ {
						c := pool[j]
						if b >= 5 && w.finalAt(c).After(w.liveSince.Add(-time.Hour)) {
							continue
						}
						if caps[c]+c.sc.fields >= sgBuckets[b][0] {
							break
						}
					}
					if j >= len(pool) {
						return fmt.Errorf("%s (paid %v) bucket %d: out of contacts", tier, paid, b)
					}
					pool[i], pool[j] = pool[j], pool[i]
					c := pool[i]
					i++
					c.sc.set = true
					c.sc.target = w.targetIn(c, b, caps[c])
					if paid {
						cells[b].paid--
					} else {
						cells[b].other--
					}
				}
			}
			left[tier] = cells
			if i != len(pool) {
				return fmt.Errorf("%s (paid %v): %d contacts without a slot", tier, paid, len(pool)-i)
			}
		}
	}
	// trial_started in the last 30 days: Warm and Hot Active contacts.
	trials := w.shuffled(func(c *sgContact) bool {
		last, _, ok := w.lastActivity(c)
		return ok && c.tier == "active" && c.sc.role == "" && c.board == "" && c.sc.target-c.sc.fields >= 32 && w.capacity(c) >= 40 &&
			last.After(w.now.AddDate(0, 0, -27)) && (c.sc.target < 50 || last.Before(w.liveSince.Add(-26*time.Hour)))
	})
	// 214 in all, with Aisha's and the entrants'.
	want := 214
	for _, c := range w.contacts {
		if c.sc.trial {
			want--
		}
	}
	if len(trials) < want {
		return fmt.Errorf("trials: %d", len(trials))
	}
	for _, c := range trials[:want] {
		c.sc.trial = true
	}
	// Beta apply signups whose form counts: confirmed within the four
	// weeks before their last open.
	for _, c := range w.contacts {
		if c.g.src != sgBeta || c.board != "" || c.sc.role != "" || c.sc.trial {
			continue
		}
		last, _, ok := w.lastActivity(c)
		if ok && !c.g.joined.After(last.Add(-2*time.Hour)) && last.Sub(c.g.joined) < 25*24*time.Hour && c.sc.target-c.sc.fields >= 24 && w.decaySteps(w.finalAt(c)) == 0 {
			c.sc.form = true
		}
	}
	return nil
}

// targetIn picks a score in a bucket the contact can reach: lower in the
// bucket more often, and never the odd small scores opens can't make.
func (w *sgWorld) targetIn(c *sgContact, b, capacity int) int {
	lo, hi := sgBuckets[b][0], sgBuckets[b][1]
	if top := capacity + c.sc.fields; top < hi {
		hi = top
	}
	if hi < lo {
		hi = lo
	}
	// Active Cold contacts mostly open and rarely click: at most 16 without
	// a click (Kenji's 18 is the board's example).
	for tries := 0; tries < 20; tries++ {
		u := w.rng.Float64()
		s := lo + int(math.Floor(math.Pow(u, 1.35)*float64(hi-lo+1)))
		if s > hi {
			s = hi
		}
		if w.feasible(c, s) {
			return s
		}
	}
	for s := lo; s <= hi; s++ {
		if w.feasible(c, s) {
			return s
		}
	}
	return lo
}

// feasible reports whether events can replay to s for c.
func (w *sgWorld) feasible(c *sgContact, s int) bool {
	_, ok := w.preDecay(c, s)
	return ok
}

// preDecay is the events part a contact needs at its last activity so
// decay leaves s − fields today.
func (w *sgWorld) preDecay(c *sgContact, s int) (int, bool) {
	e := s - c.sc.fields
	if e < 0 {
		return 0, false
	}
	_, click, ok := w.lastActivity(c)
	if !ok {
		return 0, e == 0
	}
	k := w.decaySteps(w.finalAt(c))
	minP := 2
	if click != nil {
		minP = 7
	}
	for p := minP; p <= 100; p++ {
		if p == 3 || (p == 5 && click == nil) {
			continue
		}
		if sgChain(p, k) == e {
			return p, true
		}
		if sgChain(p, k) > e {
			break
		}
	}
	return 0, false
}
