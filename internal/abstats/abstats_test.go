package abstats

import (
	"math"
	"testing"
)

func TestZTest(t *testing.T) {
	// 710/2690 vs 785/2690: lift +2.8 points, p ≈ 0.022 (98% confidence).
	lift, z, p, ok := ZTest(710, 2690, 785, 2690)
	if !ok || math.Abs(lift-2.788) > 0.01 || math.Abs(z-2.28) > 0.01 || math.Abs(p-0.0225) > 0.001 {
		t.Fatalf("lift %.3f z %.3f p %.4f", lift, z, p)
	}
	if _, _, p, ok := ZTest(400, 1000, 400, 1000); !ok || p < 0.99 {
		t.Fatalf("equal samples: p %.3f", p)
	}
	if _, _, _, ok := ZTest(0, 0, 3, 10); ok {
		t.Fatal("empty sample")
	}
	if _, _, _, ok := ZTest(0, 10, 0, 10); ok {
		t.Fatal("no variance")
	}
}

func TestDecide(t *testing.T) {
	rule := Rule{}.Normalized()
	if rule.MinSample != 500 || rule.Confidence != 0.95 {
		t.Fatalf("defaults %+v", rule)
	}
	tests := []struct {
		name    string
		vs      []Variant
		outcome string
		leader  string
		conf    float64 // approximate, percent
	}{
		// The Report board: B 3.4%, A 2.6% on human clicks, about 4,750
		// delivered each: 98% confidence.
		{"significant lead", []Variant{{"A", 4750, 124}, {"B", 4750, 162}}, Winner, "B", 98},
		// ReportStates: B 4.0%, A 3.6%, 3,160 each: no clear winner.
		{"lead within chance", []Variant{{"A", 3125, 113}, {"B", 3125, 125}}, NotSignificant, "B", 57},
		// A big gap on 100 recipients each is still too small to decide.
		{"too small", []Variant{{"A", 100, 10}, {"B", 100, 30}}, TooSmall, "B", 99.9},
		// Ties go to the earlier variant.
		{"tie", []Variant{{"A", 1000, 50}, {"B", 1000, 50}, {"C", 1000, 20}}, NotSignificant, "A", 0},
		// Three variants: the leader against the best of the rest.
		{"three variants", []Variant{{"A", 2000, 60}, {"B", 2000, 120}, {"C", 2000, 70}}, Winner, "B", 99.9},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := Decide(rule, tc.vs)
			if r.Outcome != tc.outcome || r.Leader != tc.leader {
				t.Fatalf("outcome %s, leader %s (%+v); want %s, %s", r.Outcome, r.Leader, r, tc.outcome, tc.leader)
			}
			if tc.conf > 0 && math.Abs(r.Confidence-tc.conf) > 1.5 {
				t.Fatalf("confidence %.1f, want about %.1f", r.Confidence, tc.conf)
			}
		})
	}
	if r := Decide(rule, []Variant{{"A", 600, 30}}); r.Outcome != TooSmall {
		t.Fatalf("one variant: %+v", r)
	}
	// A stricter rule.
	if r := Decide(Rule{MinSample: 5000, Confidence: 0.99}, []Variant{{"A", 4750, 124}, {"B", 4750, 162}}); r.Outcome != TooSmall {
		t.Fatalf("5,000 per variant needed: %+v", r)
	}
	if r := Decide(Rule{MinSample: 500, Confidence: 0.99}, []Variant{{"A", 4750, 124}, {"B", 4750, 162}}); r.Outcome != NotSignificant {
		t.Fatalf("99%% confidence needed: %+v", r)
	}
}
