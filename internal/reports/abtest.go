package reports

import (
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/abstats"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// The A/B card reads what production writes (review AN-1): the test as the
// campaign stores it, the decision the sending engine made when the test's
// window passed (its snapshot: what each variant had, the z-test, the rule,
// what the rest got) and each sample's results since, apart from the
// recipients who got the variant after the test (the rollout, counted as
// variant "").

// abConfig is campaigns.ab_test.
type abConfig struct {
	Kind     string `json:"kind"`
	Variants []struct {
		Key      string `json:"key"`
		Subject  string `json:"subject"`
		Label    string `json:"label"`
		SendTime string `json:"send_time"`
	} `json:"variants"`
	TestPercent   float64     `json:"test_percent"`
	Metric        string      `json:"metric"`
	DurationHours float64     `json:"duration_hours"`
	Winner        string      `json:"winner"`
	DecidedAt     *time.Time  `json:"decided_at"`
	MinSample     int64       `json:"min_sample"`
	Confidence    float64     `json:"confidence"`
	Decision      *abSnapshot `json:"decision"`
}

// OutcomeUnrecorded is the outcome of a test decided before the sending
// engine kept its decisions: the winner is known, what decided it isn't.
const OutcomeUnrecorded = "unrecorded"

// abSnapshot is the decision the engine kept (internal/sending/engine
// ab.go's abDecision).
type abSnapshot struct {
	Metric   string `json:"metric"`
	Variants []struct {
		Key         string   `json:"key"`
		Sent        int64    `json:"sent"`
		Eligible    int64    `json:"eligible"`
		Conversions int64    `json:"conversions"`
		Rate        *float64 `json:"rate"`
	} `json:"variants"`
	Leader     string    `json:"leader"`
	RunnerUp   string    `json:"runner_up"`
	Lift       *float64  `json:"lift"`
	PValue     *float64  `json:"p_value"`
	Confidence *float64  `json:"confidence"`
	MinSample  int64     `json:"min_sample"`
	Threshold  float64   `json:"threshold"`
	Outcome    string    `json:"outcome"`
	Winner     string    `json:"winner"`
	Default    string    `json:"default"`
	Hours      float64   `json:"hours"`
	DecidedAt  time.Time `json:"decided_at"`
}

// sendAB is the send's A/B state (campaign_sends.ab).
type sendAB struct {
	Phase    string     `json:"phase"`
	DecideAt *time.Time `json:"decide_at"`
	Window   int64      `json:"window"`
}

func parseAB(raw json.RawMessage) *abConfig {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var c abConfig
	if json.Unmarshal(raw, &c) != nil || c.Kind == "" {
		return nil
	}
	return &c
}

func parseSendAB(raw json.RawMessage) sendAB {
	var s sendAB
	_ = json.Unmarshal(raw, &s)
	return s
}

func ptr(v float64) *float64 { return &v }

// abResult is the A/B card: the decision when there is one, else the test
// so far, with each sample's results.
func abResult(l i18n.Lang, cfg *abConfig, snd sendAB, cells []store.ReportCounterCellsRow, c store.ReportCampaignRow, now time.Time) ReportABTest {
	metric := cfg.Metric
	if d := cfg.Decision; d != nil && d.Metric != "" {
		metric = d.Metric
	}
	if metric != "clicks" {
		metric = "human_opens"
	}
	rule := abstats.Rule{MinSample: cfg.MinSample, Confidence: cfg.Confidence}.Normalized()
	out := ReportABTest{Kind: cfg.Kind, Label: abLabel(l, cfg.Kind), Metric: metric, WindowHours: cfg.DurationHours, Variants: []ReportABVariant{},
		MinSample: rule.MinSample, Threshold: round(rule.Confidence*100, 1)}
	if out.Label == "" {
		out.Label = strings.ReplaceAll(cfg.Kind, "_", " ")
	}
	if out.Kind != "content" && out.Kind != "send_time" {
		out.Kind = "subject"
	}
	if len(cfg.Variants) > 0 {
		out.Default = cfg.Variants[0].Key
	}
	// The samples apart from the rollout ("" cells).
	byVariant := map[string]*counts{}
	var total, rollout int64
	for _, cell := range cells {
		total += cell.Sent
		if cell.Variant == "" {
			rollout += cell.Sent
			continue
		}
		if byVariant[cell.Variant] == nil {
			byVariant[cell.Variant] = &counts{}
		}
		byVariant[cell.Variant].add(cell)
	}
	// The decision's numbers, when the test was decided.
	at := map[string]abstats.Variant{}
	d := cfg.Decision
	if d != nil {
		for _, v := range d.Variants {
			at[v.Key] = abstats.Variant{Key: v.Key, Eligible: v.Eligible, Conversions: v.Conversions}
		}
		if d.MinSample > 0 {
			out.MinSample = d.MinSample
		}
		if d.Threshold > 0 {
			out.Threshold = round(d.Threshold*100, 1)
		}
		if d.Default != "" {
			out.Default = d.Default
		}
	}
	live := make([]abstats.Variant, 0, len(cfg.Variants))
	for _, v := range cfg.Variants {
		a := byVariant[v.Key]
		if a == nil {
			a = &counts{}
		}
		label := v.Subject
		if label == "" {
			label = v.Label
		}
		if label == "" && v.SendTime != "" {
			label = v.SendTime
		}
		if label == "" && v.Key == out.Default && out.Kind == "subject" {
			// The composer keeps variant A's subject on the campaign.
			label = c.Subject
		}
		rv := ReportABVariant{Key: v.Key, Label: label, Default: v.Key == out.Default, Recipients: a.sent, HumanOpenRate: rate(a.human, a.measurable()),
			ClickRate: rate(a.clicked, a.delivered), ComplaintRate: rate(a.complained, a.delivered)}
		if total > 0 {
			rv.SamplePercent = round(float64(a.sent)/float64(total)*100, 1)
		}
		lv := abstats.Variant{Key: v.Key, Eligible: a.measurable(), Conversions: a.human}
		if metric == "clicks" {
			lv.Eligible, lv.Conversions = a.delivered, a.clicked
		}
		live = append(live, lv)
		rv.MetricRate = rate(lv.Conversions, lv.Eligible)
		if dv, ok := at[v.Key]; ok {
			rv.Eligible, rv.Conversions = dv.Eligible, dv.Conversions
			rv.DecisionRate = rate(dv.Conversions, dv.Eligible)
		} else {
			rv.Eligible, rv.Conversions = lv.Eligible, lv.Conversions
		}
		out.Variants = append(out.Variants, rv)
	}
	// The statistics: the decision's when it was made, else the test so far.
	var res abstats.Result
	if d != nil && len(d.Variants) >= 2 {
		vs := make([]abstats.Variant, 0, len(d.Variants))
		for _, v := range d.Variants {
			vs = append(vs, abstats.Variant{Key: v.Key, Eligible: v.Eligible, Conversions: v.Conversions})
		}
		res = abstats.Decide(abstats.Rule{MinSample: out.MinSample, Confidence: out.Threshold / 100}, vs)
	} else {
		res = abstats.Decide(rule, live)
	}
	out.Leader, out.RunnerUp = res.Leader, res.RunnerUp
	if res.Tested {
		out.Lift, out.PValue, out.Confidence = ptr(round(res.Lift, 1)), ptr(round(res.P, 4)), ptr(round(res.Confidence, 1))
	}
	out.Significant = res.Outcome == abstats.Winner
	switch {
	case d != nil:
		out.Outcome, out.Winner = d.Outcome, d.Winner
		// A decision kept with its numbers only (imported, or kept before
		// the snapshot had the rest): the campaign's winner, the rule's
		// outcome for those numbers.
		if out.Winner == "" {
			out.Winner = cfg.Winner
		}
		if out.Outcome == "" {
			out.Outcome = res.Outcome
			if cfg.Winner != "" && res.Outcome != abstats.Winner {
				out.Outcome = abstats.Manual
			}
		}
		if out.Winner == "" {
			out.Winner = out.Default
		}
		switch {
		case !d.DecidedAt.IsZero():
			decided := d.DecidedAt
			out.DecidedAt = &decided
		case cfg.DecidedAt != nil:
			out.DecidedAt = cfg.DecidedAt
		}
		if d.Hours > 0 {
			out.DecidedAfter = ptr(d.Hours)
		} else if out.DecidedAt != nil && c.StartedAt != nil {
			out.DecidedAfter = ptr(round(out.DecidedAt.Sub(*c.StartedAt).Hours(), 1))
		}
		out.State = "no_winner"
		if out.Outcome == abstats.Winner || out.Outcome == abstats.Manual {
			out.State = "decided"
		}
		out.Remaining, out.RolloutAt = rollout, out.DecidedAt
	case cfg.Winner != "":
		// Decided before decisions were kept (review AN-1): the winner went
		// to the rest, but what decided it wasn't kept. The samples' results
		// since aren't what it was decided on, so no confidence or
		// significance is claimed for it.
		out.State, out.Winner, out.Outcome = "decided", cfg.Winner, OutcomeUnrecorded
		out.Lift, out.PValue, out.Confidence, out.Significant = nil, nil, nil, false
		out.Leader, out.RunnerUp = "", ""
		out.Remaining = rollout
		if cfg.DecidedAt != nil {
			out.DecidedAt = cfg.DecidedAt
		} else if c.StartedAt != nil && cfg.DurationHours > 0 {
			t := c.StartedAt.Add(time.Duration(cfg.DurationHours * float64(time.Hour)))
			out.DecidedAt = &t
		}
		out.RolloutAt = out.DecidedAt
	default:
		out.State = "testing"
		decide := snd.DecideAt
		if decide == nil && c.StartedAt != nil && cfg.DurationHours > 0 {
			t := c.StartedAt.Add(time.Duration(cfg.DurationHours * float64(time.Hour)))
			decide = &t
		}
		out.DecidedAt = decide
		// A test that closed without a decision (the send was cancelled)
		// has no winner.
		if c.Status == "cancelled" || c.Status == "failed" || (snd.Phase == "" && c.CompletedAt != nil) {
			out.State = "no_winner"
		}
	}
	for i := range out.Variants {
		v := &out.Variants[i]
		v.Winner = out.State != "testing" && v.Key == out.Winner
		if v.Winner {
			v.Remaining = out.Remaining
		}
	}
	out.BarMax = abBarMax(out)
	return out
}

// abBarMax is the top of the bars' scale: the next round number above the
// highest rate (10% for rates up to 10%).
func abBarMax(t ReportABTest) float64 {
	top := 0.0
	for _, v := range t.Variants {
		if v.MetricRate != nil {
			top = math.Max(top, *v.MetricRate)
		}
	}
	for _, step := range []float64{5, 10, 20, 25, 50, 75, 100} {
		if top <= step {
			return step
		}
	}
	return 100
}
