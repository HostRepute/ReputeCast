package abstats

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

// TestWelchT checks WelchT against scipy (testdata/welch.json, written by
// scipy.stats.ttest_ind_from_stats with equal_var=False): t within 1e-9,
// the p-value within 1e-7 with Student's t, and within the normal
// approximation's bound (5e-4) past NormalDF degrees of freedom.
func TestWelchT(t *testing.T) {
	raw, err := os.ReadFile("testdata/welch.json")
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Cases []struct {
			MeanA, VarA float64
			NA          int64
			MeanB, VarB float64
			NB          int64
			T, P, DF    float64
		}
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	if len(data.Cases) < 10 {
		t.Fatalf("only %d cases", len(data.Cases))
	}
	exact := 0
	for _, c := range data.Cases {
		diff, tv, p, ok := WelchT(c.MeanA, c.VarA, c.NA, c.MeanB, c.VarB, c.NB)
		if !ok {
			t.Errorf("%+v: not tested", c)
			continue
		}
		if math.Abs(diff-(c.MeanB-c.MeanA)) > 1e-12*math.Max(1, math.Abs(c.MeanB)) {
			t.Errorf("%+v: diff %v", c, diff)
		}
		if math.Abs(tv-c.T) > 1e-9 {
			t.Errorf("%+v: t %.12f, want %.12f", c, tv, c.T)
		}
		tol := 1e-7
		if c.DF > NormalDF {
			tol = 5e-4
		} else {
			exact++
		}
		if math.Abs(p-c.P) > tol {
			t.Errorf("df %.1f: p %.10f, want %.10f (±%g)", c.DF, p, c.P, tol)
		}
	}
	if exact < 8 {
		t.Errorf("only %d cases check Student's t exactly", exact)
	}
}

func TestWelchTUntestable(t *testing.T) {
	for _, c := range []struct {
		name           string
		mA, vA, mB, vB float64
		nA, nB         int64
	}{
		{"one value", 1, 1, 2, 1, 1, 10},
		{"no variance", 1, 0, 2, 0, 10, 10},
		{"negative variance", 1, -1, 2, 1, 10, 10},
	} {
		if _, _, p, ok := WelchT(c.mA, c.vA, c.nA, c.mB, c.vB, c.nB); ok || p != 1 {
			t.Errorf("%s: ok %v p %v", c.name, ok, p)
		}
	}
}

func TestRegIncBeta(t *testing.T) {
	// I_x(1, 1) = x; I_x(a, b) = 1 − I_{1−x}(b, a); I_0.5(a, a) = 0.5.
	for _, x := range []float64{0.1, 0.37, 0.5, 0.9} {
		if v := regIncBeta(x, 1, 1); math.Abs(v-x) > 1e-14 {
			t.Errorf("I_%v(1,1) = %v", x, v)
		}
		if v, w := regIncBeta(x, 2.5, 7), regIncBeta(1-x, 7, 2.5); math.Abs(v+w-1) > 1e-13 {
			t.Errorf("symmetry at %v: %v + %v", x, v, w)
		}
	}
	if v := regIncBeta(0.5, 40, 40); math.Abs(v-0.5) > 1e-13 {
		t.Errorf("I_0.5(40,40) = %v", v)
	}
}
