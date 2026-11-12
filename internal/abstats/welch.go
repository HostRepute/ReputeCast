package abstats

import "math"

// NormalDF is the degrees of freedom above which WelchT reads its p-value
// from the normal distribution instead of Student's t: the two differ by
// less than 0.0005 there, and the holdout group's samples are thousands.
const NormalDF = 1000

// WelchT is Welch's two-sample t-test (unequal variances) of mean B
// against mean A, from each sample's mean, unbiased variance (n − 1 in
// the denominator) and size, as scipy.stats.ttest_ind(b, a,
// equal_var=False) computes it. It returns the difference meanB − meanA,
// t and the two-sided p-value: Student's t with the Welch–Satterthwaite
// degrees of freedom, through the regularised incomplete beta function,
// or the normal distribution when they pass NormalDF. ok is false when a
// sample has fewer than two values or neither has any variance (nothing
// to test).
func WelchT(meanA, varA float64, nA int64, meanB, varB float64, nB int64) (diff, t, p float64, ok bool) {
	diff = meanB - meanA
	if nA < 2 || nB < 2 || varA < 0 || varB < 0 || math.IsNaN(diff) || math.IsInf(diff, 0) {
		return diff, 0, 1, false
	}
	a, b := varA/float64(nA), varB/float64(nB)
	se2 := a + b
	if se2 <= 0 || math.IsNaN(se2) || math.IsInf(se2, 0) {
		return diff, 0, 1, false
	}
	t = diff / math.Sqrt(se2)
	df := se2 * se2 / (a*a/float64(nA-1) + b*b/float64(nB-1))
	return diff, t, studentTwoSided(t, df), true
}

// studentTwoSided is P(|T| ≥ |t|) for Student's t with df degrees of
// freedom: I_{df/(df+t²)}(df/2, 1/2).
func studentTwoSided(t, df float64) float64 {
	if math.IsInf(df, 0) || math.IsNaN(df) || df > NormalDF {
		return math.Erfc(math.Abs(t) / math.Sqrt2)
	}
	x := df / (df + t*t)
	return regIncBeta(x, df/2, 0.5)
}

// regIncBeta is the regularised incomplete beta function I_x(a, b), by its
// continued fraction (Lentz's method), on the side where it converges fast.
func regIncBeta(x, a, b float64) float64 {
	switch {
	case x <= 0:
		return 0
	case x >= 1:
		return 1
	}
	la, _ := math.Lgamma(a)
	lb, _ := math.Lgamma(b)
	lab, _ := math.Lgamma(a + b)
	front := math.Exp(lab - la - lb + a*math.Log(x) + b*math.Log1p(-x))
	if x < (a+1)/(a+b+2) {
		return front * betaCF(x, a, b) / a
	}
	return 1 - front*betaCF(1-x, b, a)/b
}

// betaCF evaluates the continued fraction of the incomplete beta function.
func betaCF(x, a, b float64) float64 {
	const (
		maxIter = 10000
		eps     = 1e-16
		tiny    = 1e-300
	)
	qab, qap, qam := a+b, a+1, a-1
	c, d := 1.0, 1-qab*x/qap
	if math.Abs(d) < tiny {
		d = tiny
	}
	d = 1 / d
	h := d
	for m := 1; m <= maxIter; m++ {
		fm := float64(m)
		m2 := 2 * fm
		aa := fm * (b - fm) * x / ((qam + m2) * (a + m2))
		d = 1 + aa*d
		if math.Abs(d) < tiny {
			d = tiny
		}
		c = 1 + aa/c
		if math.Abs(c) < tiny {
			c = tiny
		}
		d = 1 / d
		h *= d * c
		aa = -(a + fm) * (qab + fm) * x / ((a + m2) * (qap + m2))
		d = 1 + aa*d
		if math.Abs(d) < tiny {
			d = tiny
		}
		c = 1 + aa/c
		if math.Abs(c) < tiny {
			c = tiny
		}
		d = 1 / d
		del := d * c
		h *= del
		if math.Abs(del-1) < eps {
			break
		}
	}
	return h
}
