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
