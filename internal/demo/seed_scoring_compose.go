package demo

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/scoring"
)

// Composing a contact's events: human opens and real clicks week by week
// up to its last activity (within the open and click caps, with no gap a
// decay step could fall in), plus the pricing click, the Beta apply form,
// trial_started and soft bounces where its role has them. The replay must
// come to the target; the seeder fails otherwise.

const (
	sgPricingURL = "https://acme.io/pricing"
	week         = 7 * 24 * time.Hour
)

var sgLinks = []string{
	"https://acme.io/blog", "https://acme.io/changelog", "https://docs.acme.io", "https://acme.io/customers/northwind",
	"https://acme.io/features/feature-flags-ga", "https://docs.acme.io/flags", "https://acme.io/office-hours", "https://docs.acme.io/migrate-v2",
}

// sgWeek is one Monday-to-Sunday week (workspace time) a contact can have
// activity in: [from, to].
type sgWeek struct{ from, to time.Time }

// weekStart is the Monday 00:00 of t's week.
func (w *sgWorld) weekStart(t time.Time) time.Time {
	l := t.In(w.loc)
	d := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, w.loc)
	return d.AddDate(0, 0, -((int(l.Weekday()) + 6) % 7))
}

// weeks lists the weeks from last's back to start's, newest first.
func (w *sgWorld) weeks(start, last time.Time) []sgWeek {
	var out []sgWeek
	for ws := w.weekStart(last); !ws.AddDate(0, 0, 7).Before(start) && len(out) < 26; ws = ws.AddDate(0, 0, -7) {
		from, to := ws, ws.AddDate(0, 0, 7).Add(-time.Minute)
		if from.Before(start) {
			from = start
		}
		if to.After(last) {
			to = last
		}
		if to.Sub(from) < 3*time.Hour && len(out) > 0 {
			break
		}
		out = append(out, sgWeek{from, to})
	}
	return out
}

// weekValue is a week's points from o opens and c clicks.
type weekValue struct{ o, c int }

func (v weekValue) points() int { return 2*v.o + 5*v.c }

// values lists what a week can add, with at least minO opens and minC
// clicks (no clicks at all for contacts who never click).
func values(minO, minC int, noClick bool) []weekValue {
	var out []weekValue
	maxC := 4
	if noClick {
		maxC = 0
	}
	for o := minO; o <= 3; o++ {
		for c := minC; c <= maxC; c++ {
			out = append(out, weekValue{o, c})
		}
	}
	return out
}

// reach[n] holds the sums n free weeks can add.
func reachable(n int, noClick bool) []map[int]bool {
	free := values(0, 0, noClick)
	out := make([]map[int]bool, n+1)
	out[0] = map[int]bool{0: true}
	for i := 1; i <= n; i++ {
		out[i] = map[int]bool{}
		for s := range out[i-1] {
			for _, v := range free {
				if s+v.points() <= 130 {
					out[i][s+v.points()] = true
				}
			}
		}
	}
	return out
}

var (
	sgReachClick   = reachable(26, false)
	sgReachNoClick = reachable(26, true)
)

// schedule splits r points over the weeks: the newest week holds the last
// activity (an open, and a click when one comes last or is the pricing
// click). It returns nil when r can't be made.
func (w *sgWorld) schedule(r int, weeks []sgWeek, finalClick, noClick bool) []weekValue {
	reach := sgReachClick
	if noClick {
		reach = sgReachNoClick
	}
	n := len(weeks)
	if n == 0 {
		return nil
	}
	minC := 0
	if finalClick {
		minC = 1
	}
	// Spread over as many weeks as a natural pace needs (about 6–14 points
	// a week), never fewer than the caps allow.
	want := max(1, min(n, (r+11)/12))
	per := float64(r) / float64(want)
	out := make([]weekValue, 0, want)
	rem := r
	for i := 0; i < n && rem > 0; i++ {
		var opts []weekValue
		if i == 0 {
			opts = values(1, minC, noClick)
		} else {
			opts = values(0, 0, noClick)
		}
		best, bestD := weekValue{-1, -1}, 1e9
		for _, v := range opts {
			p := v.points()
			if p > rem || (p == 0 && i > 0) {
				continue
			}
			left := rem - p
			if left > 0 && (n-1-i == 0 || !reach[n-1-i][left]) {
				continue
			}
			d := abs(float64(p) - per)
			// A little noise so contacts don't all look alike.
			d += w.rng.Float64() * 2.5
			if d < bestD {
				best, bestD = v, d
			}
		}
		if best.o < 0 {
			return nil
		}
		out = append(out, best)
		rem -= best.points()
	}
	if rem != 0 {
		return nil
	}
	return out
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// between is a random moment in [from, to].
func (w *sgWorld) between(from, to time.Time) time.Time {
	if !to.After(from) {
		return from
	}
	return from.Add(time.Duration(w.rng.Int64N(int64(to.Sub(from))))).Truncate(time.Second)
}

// compose builds and checks a contact's events.
func (w *sgWorld) compose(c *sgContact, eng *scoring.Engine) error {
	if c.board == "sam" {
		return w.composeSam(c, eng)
	}
	targets := []int{c.sc.target}
	b := 0
	for i, bk := range sgBuckets {
		if c.sc.target >= bk[0] && c.sc.target <= bk[1] {
			b = i
		}
	}
	// Other scores in the same bucket keep every count the same.
	if c.board == "" {
		lo, hi := sgBuckets[b][0], sgBuckets[b][1]
		if c.sc.role == "entrant" {
			hi = c.sc.target
			if hi > 51 {
				hi = 56
			}
			lo = 50
		}
		for s := lo; s <= hi; s++ {
			if s != c.sc.target {
				targets = append(targets, s)
			}
		}
	}
	var lastErr, firstErr error
	for _, t := range targets {
		for try := 0; try < 4; try++ {
			evs, err := w.build(c, t)
			if err != nil {
				lastErr = err
				break
			}
			res := eng.Replay(evs, w.now, c.sc.fields)
			if res.Score != t {
				var desc []string
				for _, ev := range evs {
					desc = append(desc, ev.Kind+"@"+ev.At.In(w.loc).Format("Mon 01-02 15:04"))
				}
				last, click, _ := w.lastActivity(c)
				lastErr = fmt.Errorf("replays to %d, not %d (fields %d, last %s, click %v, decay %d): %v", res.Score, t, c.sc.fields,
					last.In(w.loc).Format("Mon 01-02 15:04"), click != nil, res.DecayLost, desc)
				continue
			}
			if err := w.checkCrossing(c, eng, evs); err != nil {
				lastErr = err
				continue
			}
			c.sc.target, c.sc.events, c.sc.res = t, evs, res
			return nil
		}
		if firstErr == nil {
			firstErr = lastErr
		}
	}
	return fmt.Errorf("role %q target %d (%s, paid %v): %w (last: %w)", c.sc.role, c.sc.target, c.tier, c.sc.fields > 0, firstErr, lastErr)
}

// checkCrossing keeps "Sales-ready handoff" at its 37: entrants crossed 50
// after it went live, everyone else in the Hot band before.
func (w *sgWorld) checkCrossing(c *sgContact, eng *scoring.Engine, evs []scoring.Event) error {
	before := eng.Replay(evs, w.liveSince, c.sc.fields).Score
	now := eng.Replay(evs, w.now, c.sc.fields).Score
	switch {
	case c.sc.role == "entrant" && (before >= 50 || now < 50):
		return fmt.Errorf("entrant at %d then %d", before, now)
	case c.sc.role != "entrant" && now >= 50 && before < 50:
		return fmt.Errorf("crossed 50 after the automation went live (%d → %d)", before, now)
	}
	return nil
}

// build lays out the events for score t.
func (w *sgWorld) build(c *sgContact, t int) ([]scoring.Event, error) {
	last, click, ok := w.lastActivity(c)
	e := t - c.sc.fields
	if !ok {
		if e != 0 {
			return nil, fmt.Errorf("no activity for %d", e)
		}
		return nil, nil
	}
	k := w.decaySteps(w.finalAt(c))
	if (c.sc.role == "pricing" || c.sc.role == "mover") && k > 0 {
		return nil, fmt.Errorf("pricing clicker with decay")
	}
	var evs []scoring.Event
	add := func(at time.Time, kind, ref string, counted bool, msg uuid.UUID) {
		evs = append(evs, scoring.Event{At: at, Kind: kind, Ref: ref, Counted: counted, Message: msg})
	}
	specials := 0
	// The Beta apply form, when the contact signed up through it lately.
	if c.sc.form {
		add(c.g.joined, scoring.KindForm, w.forms["Beta apply"].String(), true, uuid.Nil)
		specials += 20
	}
	// trial_started from Acme's app, before the last activity (and before
	// the automation went live for anyone who is Hot).
	if c.sc.trialLive {
		// After the automation went live: it takes the score past 50.
		add(w.between(w.liveSince.Add(20*time.Minute), w.now.Add(-20*time.Minute)), scoring.KindAPIEvent, "trial_started", true, uuid.Nil)
		specials += 25
	} else if c.sc.trial {
		from := last.AddDate(0, 0, -20)
		if lim := w.now.AddDate(0, 0, -29); from.Before(lim) {
			from = lim
		}
		if ws := w.windowStart(c); from.Before(ws) {
			from = ws
		}
		to := last.Add(-time.Hour)
		if lim := w.liveSince.Add(-time.Hour); to.After(lim) && t >= 50 {
			to = lim
		}
		if !to.After(from) {
			return nil, fmt.Errorf("no room for trial_started")
		}
		add(w.between(from, to), scoring.KindAPIEvent, "trial_started", true, uuid.Nil)
		specials += 25
	}
	pricing := c.sc.role == "pricing" || c.sc.role == "mover"
	finalClick := click != nil || pricing
	softBounces := 0
	var p int
	if c.sc.role == "mover" {
		// Pricing interaction (+22) then soft bounces down to the target.
		for sb := 1; sb <= 6; sb++ {
			p = e + 5*sb
			r := p - 15 - specials
			if r >= 7 && w.schedule(r, w.weeks(w.windowStart(c), last), true, false) != nil {
				softBounces = sb
				break
			}
		}
		if softBounces == 0 {
			return nil, fmt.Errorf("mover %d can't be made", e)
		}
	} else {
		// The least events part that holds the last activity and the
		// role's extras and decays to e.
		qmin := 2 + specials
		if finalClick {
			qmin = 7 + specials
		}
		if pricing {
			qmin += 15
		}
		var feasible bool
		for q := qmin; q <= 100; q++ {
			if q-qmin == 1 || q-qmin == 3 {
				continue // opens and clicks can't add 1 or 3
			}
			if sgChain(q, k) == e {
				p, feasible = q, true
				break
			}
			if sgChain(q, k) > e {
				break
			}
		}
		if !feasible {
			return nil, fmt.Errorf("no events part decays to %d in %d steps", e, k)
		}
	}
	r := p - specials
	if pricing {
		r -= 15
	}
	minR := 2
	if finalClick {
		minR = 7
	}
	if r < minR {
		return nil, fmt.Errorf("%d points too few for the last activity", r)
	}
	weeks := w.weeks(w.windowStart(c), last)
	plan := w.schedule(r, weeks, finalClick, c.sc.noClick)
	if plan == nil {
		return nil, fmt.Errorf("%d points don't fit %d weeks", r, len(weeks))
	}
	// Active Cold contacts at 17–19 click at least once: Kenji, who opens
	// every week and never clicks, is the board's example of the type.
	if c.board == "" && c.tier == "active" && t >= 17 && t < 20 {
		clicks := 0
		for _, v := range plan {
			clicks += v.c
		}
		if clicks == 0 {
			return nil, fmt.Errorf("an opener without a click at %d", t)
		}
	}
	link := func() string { return sgLinks[w.rng.IntN(len(sgLinks))] }
	for i, v := range plan {
		wk := weeks[i]
		o, cl := v.o, v.c
		if i == 0 {
			// The last activity: an open at the W02 time, and its click.
			add(last, scoring.KindOpen, "", true, uuid.Nil)
			o--
			if finalClick {
				at := last.Add(3 * time.Minute)
				if click != nil {
					at = *click
				}
				ref := link()
				if pricing {
					ref = sgPricingURL
				}
				add(at, scoring.KindClick, ref, true, uuid.Nil)
				cl--
			}
		}
		end := wk.to
		if i == 0 {
			end = last.Add(-90 * time.Minute)
		}
		if end.Before(wk.from) {
			end = wk.from
		}
		var opens []time.Time
		for j := 0; j < o; j++ {
			at := w.between(wk.from, end)
			add(at, scoring.KindOpen, "", true, uuid.Nil)
			opens = append(opens, at)
		}
		for j := 0; j < cl; j++ {
			var at time.Time
			if len(opens) > 0 {
				at = opens[j%len(opens)].Add(time.Duration(2+w.rng.IntN(25)) * time.Minute)
			} else {
				at = w.between(wk.from, end)
			}
			if at.After(end) {
				at = end
			}
			add(at, scoring.KindClick, link(), true, uuid.Nil)
		}
	}
	// Soft bounces after the pricing click, retries over and not delivered.
	for j := 0; j < softBounces; j++ {
		from := last.Add(6 * time.Hour)
		to := w.now.Add(-30 * time.Minute)
		if !to.After(from) {
			from = last.Add(10 * time.Minute)
		}
		add(w.between(from, to), scoring.KindSoftBounce, "", true, uuid.Nil)
	}
	sort.SliceStable(evs, func(i, j int) bool { return evs[i].At.Before(evs[j].At) })
	return evs, nil
}

// composeSam lays out Sam Rivera's score (ScoringProfile): 11 human opens
// and 4 clicks since he came over from Brevo on 18 Sep, plan team, the soft
// bounce of Weekly Digest #139 delivered after a retry, and the last
// change, "Read the docs" in October Product Update: 47 → 52, Hot.
func (w *sgWorld) composeSam(c *sgContact, eng *scoring.Engine) error {
	s := w.s
	msgs := map[string]uuid.UUID{}
	rows, err := s.Tx.Query(w.ctx, `SELECT m.id, coalesce(ca.name, a.name, '') FROM messages m
		LEFT JOIN campaigns ca ON ca.id = m.campaign_id LEFT JOIN automations a ON a.id = m.automation_id
		WHERE m.contact_id = $1`, c.id)
	if err == nil {
		for rows.Next() {
			var id uuid.UUID
			var name string
			if rows.Scan(&id, &name) == nil && name != "" {
				msgs[name] = id
			}
		}
		rows.Close()
	}
	type ev struct {
		at    time.Time
		kind  string
		ref   string
		msg   string
		count bool
	}
	opens := []time.Time{
		s.designDay(23, 10, 31), s.designDay(21, 9, 15), s.designDay(19, 10, 30), s.designDay(17, 8, 40), s.designDay(14, 19, 5),
		s.designDay(12, 10, 12), s.designDay(10, 9, 30), s.designDay(9, 12, 10), s.designDay(7, 8, 55), s.designDay(6, 18, 20), s.designDay(5, 9, 41),
	}
	// At most three opens count in a week: move extras to the week before.
	opens = w.fitWeekly(opens, 3, w.brevoAt.Add(-14*24*time.Hour))
	list := []ev{
		{s.designDay(19, 9, 12), scoring.KindSoftBounce, "", "Weekly Digest #139", false},
		{s.designDay(17, 8, 46), scoring.KindClick, "https://acme.io/blog/weekly-digest-139", "", true},
		{s.designDay(12, 10, 20), scoring.KindClick, "https://acme.io/customers/northwind", "Weekly Digest #140", true},
		{s.designDay(5, 9, 47), scoring.KindClick, "https://acme.io/blog/weekly-digest-141", "Weekly Digest #141", true},
		{s.designDay(3, 10, 22), scoring.KindClick, "https://docs.acme.io/flags", "October Product Update", true},
	}
	byDay := map[int]string{19: "Weekly Digest #139", 12: "Weekly Digest #140", 5: "Weekly Digest #141"}
	for _, at := range opens {
		name := ""
		for d, n := range byDay {
			if at.Equal(s.designDay(d, at.In(w.loc).Hour(), at.In(w.loc).Minute())) {
				name = n
			}
		}
		list = append(list, ev{at, scoring.KindOpen, "", name, true})
	}
	var evs []scoring.Event
	for _, e := range list {
		evs = append(evs, scoring.Event{At: e.at, Kind: e.kind, Ref: e.ref, Counted: e.count, Message: msgs[e.msg]})
		if e.msg != "" {
			w.samLinks = append(w.samLinks, sgLink{e.at, e.kind, e.msg})
		}
	}
	sort.SliceStable(evs, func(i, j int) bool { return evs[i].At.Before(evs[j].At) })
	res := eng.Replay(evs, w.now, c.sc.fields)
	if res.Score != 52 {
		return fmt.Errorf("the board's Sam Rivera replays to %d", res.Score)
	}
	c.sc.events, c.sc.res = evs, res
	return nil
}

// sgLink ties one of Sam's events to the email it was about, by name.
type sgLink struct {
	at         time.Time
	kind, name string
}

// linkSam points Sam's events at his emails once the audience's late step
// has written them (his timeline: Weekly Digest #139 to #141 and the
// October Product Update), so the profile can name them.
func linkSam(ctx context.Context, s *Seeder, sam uuid.UUID, links []sgLink) error {
	for _, l := range links {
		if _, err := s.Tx.Exec(ctx, `UPDATE score_events SET message_id = (
				SELECT m.id FROM messages m JOIN campaigns ca ON ca.id = m.campaign_id
				WHERE m.contact_id = $1 AND ca.name = $4 ORDER BY m.id LIMIT 1)
			WHERE contact_id = $1 AND occurred_at = $2 AND kind = $3`, sam, l.at, l.kind, l.name); err != nil {
			return err
		}
	}
	return nil
}

// fitWeekly moves times so no Monday-week holds more than n, shifting the
// earliest extras a week back (never before floor).
func (w *sgWorld) fitWeekly(ts []time.Time, n int, floor time.Time) []time.Time {
	out := slices.Clone(ts)
	for pass := 0; pass < 10; pass++ {
		sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
		count := map[time.Time]int{}
		for _, t := range out {
			count[w.weekStart(t)]++
		}
		moved := false
		for i, t := range out {
			ws := w.weekStart(t)
			if count[ws] <= n {
				continue
			}
			// The earliest one of an overfull week goes to the previous
			// week's last free day.
			nt := t.AddDate(0, 0, -7)
			for count[w.weekStart(nt)] >= n && nt.After(floor) {
				nt = nt.AddDate(0, 0, -7)
			}
			if nt.Before(floor) {
				continue
			}
			count[ws]--
			count[w.weekStart(nt)]++
			out[i] = nt
			moved = true
			break
		}
		if !moved {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}
