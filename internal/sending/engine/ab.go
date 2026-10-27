package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/abstats"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// A/B tests end to end (review AN-1). The sample gets the variants, the
// rest wait; when the test's window has passed after the last sample
// recipient's time (a send-time test's variants go out at their time of
// day in each recipient's own time zone), the test is decided by the rule
// in internal/abstats: a winner only with every variant at the minimum
// sample and a significant lead; otherwise the rest get the default
// variant (A). The decision is a snapshot of what it was made on, kept
// with the send and the campaign, and the rest are counted apart from the
// sample (messages.ab_rollout), so the report compares the samples alone
// and says what the rest got.

// NoWinnerDefault is the only choice for a test without a clear winner:
// the default variant (A) goes to the rest (contract C14).
const NoWinnerDefault = "default"

// abRule is the decision rule a campaign's test sets (contract C14).
type abRule struct {
	MinSample  int64   `json:"min_sample,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	NoWinner   string  `json:"no_winner,omitempty"`
}

func (r abRule) stats() abstats.Rule {
	return abstats.Rule{MinSample: r.MinSample, Confidence: r.Confidence}.Normalized()
}

// abDecision is what a test was decided on: per variant, the recipients
// whose result could be measured and how many converted on the metric at
// the decision, the z-test between the leader and the runner-up, the rule
// and what the rest got.
type abDecision struct {
	Metric   string              `json:"metric"`
	Variants []abDecisionVariant `json:"variants"`
	Leader   string              `json:"leader,omitempty"`
	RunnerUp string              `json:"runner_up,omitempty"`
	// Lift is the leader's lead in percentage points, PValue the two-sided
	// p-value, Confidence 1 - p in percent (absent when no test could run).
	Lift       *float64 `json:"lift,omitempty"`
	PValue     *float64 `json:"p_value,omitempty"`
	Confidence *float64 `json:"confidence,omitempty"`
	// MinSample and Threshold are the rule (Threshold 0.95 is 95%).
	MinSample int64   `json:"min_sample"`
	Threshold float64 `json:"threshold"`
	// Outcome: winner, not_significant, too_small (the default went out)
	// or manual (a person picked).
	Outcome string `json:"outcome"`
	// Winner is the variant the rest got; Default the test's default.
	Winner  string `json:"winner"`
	Default string `json:"default"`
	// Hours is how long after the sample went out it was decided.
	Hours     float64   `json:"hours"`
	DecidedAt time.Time `json:"decided_at"`
	DecidedBy string    `json:"decided_by,omitempty"`
}

type abDecisionVariant struct {
	Key         string   `json:"key"`
	Sent        int64    `json:"sent"`
	Eligible    int64    `json:"eligible"`
	Conversions int64    `json:"conversions"`
	Rate        *float64 `json:"rate,omitempty"`
}

// abStart sets the test up once the audience is built: its rule, its
// default, and when it is decided (the window after the last sample
// recipient's time).
func (e *Engine) abStart(ctx context.Context, cc store.SendingCampaignContextRow, ab *abTest, st abState, now time.Time) (abState, error) {
	if ab == nil {
		return st, nil
	}
	if ab.Winner != "" {
		st.Kind, st.Phase, st.Winner = ab.Kind, "decided", ab.Winner
		return st, nil
	}
	if ab.TestPercent <= 0 || ab.TestPercent >= 100 || len(ab.Variants) < 2 {
		return st, nil
	}
	st.Kind, st.Metric, st.TestPercent, st.Phase = ab.Kind, ab.Metric, ab.TestPercent, "testing"
	if st.Metric == "" {
		st.Metric = "human_opens"
	}
	st.Variants = st.Variants[:0]
	for _, v := range ab.Variants {
		st.Variants = append(st.Variants, v.Key)
	}
	st.Default = ab.Variants[0].Key
	st.Rule = abRule{MinSample: ab.MinSample, Confidence: ab.Confidence, NoWinner: NoWinnerDefault}
	r := st.Rule.stats()
	st.Rule.MinSample, st.Rule.Confidence = r.MinSample, r.Confidence
	window := time.Duration(max(ab.DurationHours, 1)) * time.Hour
	if st.Window > 0 {
		window = time.Duration(st.Window) * time.Second
	}
	if st.Metric == "manual" {
		return st, nil
	}
	// The window starts when the last sample recipient is due: a send-time
	// test's latest variant time (in its recipients' time zones), a
	// best-time send's latest hour.
	from := now
	cid := cc.ID
	last, err := e.Q.ABLastSampleDue(ctx, &cid)
	if err != nil {
		return st, err
	}
	if last.After(from) {
		from = last
	}
	at := from.Add(window)
	st.DecideAt = &at
	return st, nil
}

// cellVariant is the counter cell a message counts in: the recipients who
// got the variant after the test (the rollout) count apart from the
// sample, as variant "".
func cellVariant(variant string, rollout bool) string {
	if rollout {
		return ""
	}
	return variant
}

// abSendTimes are a send-time test's variant times for recipients of one
// time zone: the next occurrence of each variant's time of day there.
type abSendTimes struct {
	ab    *abTest
	ws    *time.Location
	now   time.Time
	zones map[string]map[string]*time.Time
	locs  map[string]*time.Location
}

func newSendTimes(ab *abTest, ws *time.Location, now time.Time) *abSendTimes {
	return &abSendTimes{ab: ab, ws: ws, now: now, zones: map[string]map[string]*time.Time{}, locs: map[string]*time.Location{}}
}

// in returns the variants' times for a recipient in time zone tz (the
// workspace's when tz is empty or unknown).
func (t *abSendTimes) in(tz string) map[string]*time.Time {
	if t == nil || t.ab == nil || t.ab.Kind != "send_time" {
		return nil
	}
	if out, ok := t.zones[tz]; ok {
		return out
	}
	loc := t.location(tz)
	out := map[string]*time.Time{}
	for _, v := range t.ab.Variants {
		if at, ok := nextLocal(v.SendTime, loc, t.now); ok {
			out[v.Key] = &at
		}
	}
	t.zones[tz] = out
	return out
}

func (t *abSendTimes) location(tz string) *time.Location {
	if tz == "" {
		return t.ws
	}
	if l, ok := t.locs[tz]; ok {
		return l
	}
	l, err := time.LoadLocation(tz)
	if err != nil || l == nil {
		l = t.ws
	}
	t.locs[tz] = l
	return l
}

// nextLocal is the next time it is hh:mm in loc after now.
func nextLocal(hhmm string, loc *time.Location, now time.Time) (time.Time, bool) {
	var h, m int
	if _, err := fmt.Sscanf(strings.TrimSpace(hhmm), "%d:%d", &h, &m); err != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return time.Time{}, false
	}
	local := now.In(loc)
	at := time.Date(local.Year(), local.Month(), local.Day(), h, m, 0, 0, loc)
	if !at.After(now) {
		at = time.Date(local.Year(), local.Month(), local.Day()+1, h, m, 0, 0, loc)
	}
	return at, true
}

// checkAB decides an A/B test when its window has passed and releases the
// variant that goes out to everyone who waited.
func (e *Engine) checkAB(ctx context.Context, rs *runState, _ tallies, now time.Time) error {
	if rs.ab.Phase != "testing" || rs.ab.DecideAt == nil || now.Before(*rs.ab.DecideAt) {
		return nil
	}
	return e.decideAB(ctx, rs.campaign, "", nil, nil, "", now)
}

// abWinner is the variant with the best rate on the test's metric, with
// each variant's rate (of delivered), as the live monitor shows them while
// the test runs.
func abWinner(ab abState, ts tallies) (string, map[string]float64) {
	rates := map[string]float64{}
	best, bestRate := "", -1.0
	for _, v := range ab.Variants {
		t := ts.variant[v]
		r := 0.0
		if t != nil {
			if ab.Metric == "clicks" {
				r = rate(t.clicked, t.delivered)
			} else {
				r = rate(t.human, t.delivered)
			}
		}
		rates[v] = round2(r)
		if r > bestRate {
			best, bestRate = v, r
		}
	}
	return best, rates
}

// abSample is the test sample's results per variant, on the metric: for
// human opens, the recipients whose opens can be measured (delivered,
// less the privacy-protected) and their human opens; for clicks, the
// delivered and the clickers.
func abSample(rows []store.ABSampleCountsRow, keys []string, metric string) []abDecisionVariant {
	by := map[string]store.ABSampleCountsRow{}
	for _, r := range rows {
		by[r.Variant] = r
	}
	out := make([]abDecisionVariant, 0, len(keys))
	for _, k := range keys {
		r := by[k]
		v := abDecisionVariant{Key: k, Sent: r.Sent, Eligible: max(r.Delivered-r.OpenedPrivacy, 0), Conversions: r.OpenedHuman}
		if metric == "clicks" {
			v.Eligible, v.Conversions = r.Delivered, r.Clicked
		}
		if x, ok := abstats.Rate(v.Conversions, v.Eligible); ok {
			x = math.Round(x*100) / 100
			v.Rate = &x
		}
		out = append(out, v)
	}
	return out
}

// abDecide applies the rule to the sample, or records a person's pick.
func abDecide(st abState, rows []store.ABSampleCountsRow, pick string, started *time.Time, now time.Time) abDecision {
	metric := st.Metric
	if metric != "clicks" {
		metric = "human_opens"
	}
	rule := st.Rule.stats()
	def := st.Default
	if def == "" && len(st.Variants) > 0 {
		def = st.Variants[0]
	}
	d := abDecision{Metric: metric, Variants: abSample(rows, st.Variants, metric), MinSample: rule.MinSample, Threshold: rule.Confidence, Default: def, DecidedAt: now}
	if started != nil {
		d.Hours = math.Round(now.Sub(*started).Hours()*10) / 10
	}
	vs := make([]abstats.Variant, 0, len(d.Variants))
	for _, v := range d.Variants {
		vs = append(vs, abstats.Variant{Key: v.Key, Eligible: v.Eligible, Conversions: v.Conversions})
	}
	res := abstats.Decide(rule, vs)
	d.Leader, d.RunnerUp, d.Outcome = res.Leader, res.RunnerUp, res.Outcome
	if res.Tested {
		lift, p, conf := math.Round(res.Lift*100)/100, math.Round(res.P*10000)/10000, math.Round(res.Confidence*10)/10
		d.Lift, d.PValue, d.Confidence = &lift, &p, &conf
	}
	switch {
	case pick != "":
		d.Outcome, d.Winner = abstats.Manual, pick
	case res.Outcome == abstats.Winner:
		d.Winner = res.Leader
	default:
		d.Winner = def
	}
	return d
}

// decideAB decides the test (pick names a person's choice; "" applies the
// rule), releases the variant that goes out to everyone who waited, and
// keeps the decision with the send and the campaign.
func (e *Engine) decideAB(ctx context.Context, c store.SendingCampaignContextRow, pick string, _ map[string]float64, by *uuid.UUID, byName string, now time.Time) error {
	err := db.Tx(ctx, e.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		s, err := q.SendingGetSendForUpdate(ctx, c.ID)
		if err != nil {
			return err
		}
		ab := parseAB(s.Ab)
		if ab.Phase != "testing" {
			return nil
		}
		rows, err := q.ABSampleCounts(ctx, c.ID)
		if err != nil {
			return err
		}
		d := abDecide(ab, rows, pick, c.StartedAt, now)
		d.DecidedBy = byName
		released, err := e.releaseHeld(ctx, q, c, d.Winner, now)
		if err != nil {
			return err
		}
		rates := map[string]float64{}
		for _, v := range d.Variants {
			if v.Rate != nil {
				rates[v.Key] = *v.Rate
			}
		}
		ab.Phase, ab.Winner, ab.DecidedAt, ab.DecidedBy, ab.Rates, ab.Decision = "decided", d.Winner, &now, byName, rates, &d
		b, _ := json.Marshal(ab)
		if err := q.SendingSetAB(ctx, store.SendingSetABParams{CampaignID: c.ID, Ab: b, At: now}); err != nil {
			return err
		}
		raw, _ := json.Marshal(d)
		if err := q.ABSaveDecision(ctx, store.ABSaveDecisionParams{ID: c.ID, Winner: d.Winner, Decision: raw}); err != nil {
			return err
		}
		actor := systemActor
		if by != nil {
			actor = &audit.Actor{Type: authz.ActorUser, ID: by, Name: byName}
		}
		if audit.PrincipalFrom(ctx) != nil {
			actor = nil
		}
		after := map[string]any{"winner": d.Winner, "outcome": d.Outcome, "rates": rates, "released": released, "minSample": d.MinSample, "threshold": d.Threshold}
		if d.Confidence != nil {
			after["confidence"] = *d.Confidence
		}
		return e.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &c.WorkspaceID, Action: "campaign.ab_winner_picked", TargetType: "campaign", TargetID: c.ID.String(),
			TargetName: c.Name, Actor: actor, After: after})
	})
	if err == nil {
		e.kick(ctx, c.ID)
	}
	return err
}

// releaseHeld sends the variant to everyone who waited for the test: at
// once (a best-time send keeps each recipient's hour), or, for a send-time
// test, at the variant's time of day in each recipient's time zone.
func (e *Engine) releaseHeld(ctx context.Context, q *store.Queries, c store.SendingCampaignContextRow, variant string, now time.Time) (int64, error) {
	cid := c.ID
	sendTime := ""
	if ab := abTestOf(c.AbTest); ab != nil && ab.Kind == "send_time" {
		for _, v := range ab.Variants {
			if v.Key == variant {
				sendTime = v.SendTime
			}
		}
	}
	if sendTime == "" {
		return q.ABReleaseHeld(ctx, store.ABReleaseHeldParams{CampaignID: &cid, Variant: variant})
	}
	rows, err := q.ABHeldZones(ctx, &cid)
	if err != nil {
		return 0, err
	}
	ws := state.Location(c.WorkspaceTimezone)
	times := newSendTimes(&abTest{Kind: "send_time", Variants: []abVariant{{Key: variant, SendTime: sendTime}}}, ws, now)
	// In id order (ABHeldZones), the order writers lock messages in.
	var released int64
	for len(rows) > 0 {
		n := min(len(rows), 5000)
		p := store.ABReleaseAtParams{Variant: variant, Ids: make([]uuid.UUID, 0, n), Ats: make([]time.Time, 0, n)}
		for _, r := range rows[:n] {
			at := now
			if t := times.in(r.Timezone)[variant]; t != nil {
				at = *t
			}
			p.Ids = append(p.Ids, r.ID)
			p.Ats = append(p.Ats, at)
		}
		k, err := q.ABReleaseAt(ctx, p)
		if err != nil {
			return released, err
		}
		released += k
		rows = rows[n:]
	}
	return released, nil
}
