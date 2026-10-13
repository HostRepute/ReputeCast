package reports

import "math"

// rate is n/d in percent, rounded to four decimals; nil without a base.
func rate(n, d int64) *float64 {
	if d <= 0 {
		return nil
	}
	v := math.Round(float64(n)/float64(d)*1e6) / 1e4
	return &v
}

func round(v float64, digits int) float64 {
	p := math.Pow(10, float64(digits))
	return math.Round(v*p) / p
}

// ZTest is a two-proportion z-test of x1/n1 against x2/n2. It returns the
// difference p2 - p1 in percentage points, z and the two-sided p-value.
// ok is false when either sample is empty or the pooled proportion is 0 or 1
// (no variance, nothing to test).
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

// Significance is the threshold below which a difference counts as real.
const Significance = 0.05
