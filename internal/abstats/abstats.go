// Package abstats is the A/B test's decision rule, shared by the sending
// engine (which decides) and the report (which explains the decision), so
// the two can't disagree (review AN-1).
//
// A variant wins only when every variant reached the minimum sample and
// the leader's lead over the runner-up is significant: a two-sided
// two-proportion z-test at the chosen confidence (95% unless the campaign
// says otherwise). Otherwise there is no clear winner, and the rest of the
// audience gets the default variant.
package abstats

import "math"

// Defaults of the rule (contract C14: campaigns.ab_test.min_sample and
// confidence).
const (
	DefaultMinSample  = 500
	DefaultConfidence = 0.95
)

// Outcomes of a decision.
const (
	// Winner: the leader won.
	Winner = "winner"
	// NotSignificant: the lead could be chance.
	NotSignificant = "not_significant"
	// TooSmall: a variant didn't reach the minimum sample.
	TooSmall = "too_small"
	// Manual: a person picked the variant.
	Manual = "manual"
)

// Variant is one variant's result on the test's metric: the recipients
// whose result could be measured and how many of them converted.
type Variant struct {
	Key         string
	Eligible    int64
	Conversions int64
}

// Rule is what a winner needs.
type Rule struct {
	MinSample  int64
	Confidence float64
}

// Normalized fills in the defaults.
func (r Rule) Normalized() Rule {
	if r.MinSample <= 0 {
		r.MinSample = DefaultMinSample
	}
	if r.Confidence <= 0 || r.Confidence >= 1 {
		r.Confidence = DefaultConfidence
	}
	return r
}

// Result is the rule applied to the variants.
type Result struct {
	// Leader has the best rate, RunnerUp the best of the rest (ties go to
	// the earlier variant).
	Leader, RunnerUp string
	// Tested: the z-test could run (both had results and the rates
	// weren't all 0% or all 100%).
	Tested bool
	// Lift is the leader's lead in percentage points; P the two-sided
	// p-value; Confidence is 1 - P in percent.
	Lift, Z, P, Confidence float64
	// Enough: every variant reached the minimum sample.
	Enough bool
	// Outcome is Winner, NotSignificant or TooSmall.
	Outcome string
}

// Rate is n/d in percent; ok is false without a base.
func Rate(n, d int64) (float64, bool) {
	if d <= 0 {
		return 0, false
	}
	return float64(n) / float64(d) * 100, true
}

// Decide applies the rule to the variants (at least two).
func Decide(r Rule, vs []Variant) Result {
	r = r.Normalized()
	out := Result{Outcome: TooSmall, P: 1}
	if len(vs) < 2 {
		return out
	}
	rate := func(v Variant) float64 {
		x, ok := Rate(v.Conversions, v.Eligible)
		if !ok {
			return -1
		}
		return x
	}
	// The leader and the runner-up: the best rate, then the best of the
	// rest (ties go to the earlier variant).
	a, b := vs[1], vs[0]
	if rate(vs[1]) > rate(vs[0]) {
		a, b = vs[0], vs[1]
	}
	for _, v := range vs[2:] {
		switch {
		case rate(v) > rate(b):
			a, b = b, v
		case rate(v) > rate(a):
			a = v
		}
	}
	out.Leader, out.RunnerUp = b.Key, a.Key
	out.Enough = true
	for _, v := range vs {
		if v.Eligible < r.MinSample {
			out.Enough = false
		}
	}
	if lift, z, p, ok := ZTest(a.Conversions, a.Eligible, b.Conversions, b.Eligible); ok {
		out.Tested, out.Lift, out.Z, out.P = true, lift, z, p
		out.Confidence = (1 - p) * 100
	} else if lr, ok1 := Rate(b.Conversions, b.Eligible); ok1 {
		if rr, ok2 := Rate(a.Conversions, a.Eligible); ok2 {
			out.Lift = lr - rr
		}
	}
	switch {
	case !out.Enough:
		out.Outcome = TooSmall
	case out.Tested && out.Lift > 0 && out.P < 1-r.Confidence:
		out.Outcome = Winner
	default:
		out.Outcome = NotSignificant
	}
	return out
}

// ZTest is a two-proportion z-test of x1/n1 against x2/n2. It returns the
// difference p2 - p1 in percentage points, z and the two-sided p-value.
// ok is false when either sample is empty or the pooled proportion is 0 or
// 1 (no variance, nothing to test).
func ZTest(x1, n1, x2, n2 int64) (lift, z, p float64, ok bool) {
	if n1 <= 0 || n2 <= 0 || x1 < 0 || x2 < 0 || x1 > n1 || x2 > n2 {
		return 0, 0, 1, false
	}
	p1 := float64(x1) / float64(n1)
	p2 := float64(x2) / float64(n2)
	pooled := float64(x1+x2) / float64(n1+n2)
	if pooled <= 0 || pooled >= 1 {
		return (p2 - p1) * 100, 0, 1, false
	}
	se := math.Sqrt(pooled * (1 - pooled) * (1/float64(n1) + 1/float64(n2)))
	z = (p2 - p1) / se
	p = math.Erfc(math.Abs(z) / math.Sqrt2)
	return (p2 - p1) * 100, z, p, true
}
