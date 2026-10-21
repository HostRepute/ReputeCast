package stores

import (
	"strconv"
	"strings"
)

// zeroDecimal are currencies without minor units.
var zeroDecimal = map[string]bool{
	"BIF": true, "CLP": true, "DJF": true, "GNF": true, "ISK": true, "JPY": true, "KMF": true, "KRW": true,
	"PYG": true, "RWF": true, "UGX": true, "VND": true, "VUV": true, "XAF": true, "XOF": true, "XPF": true,
}

// threeDecimal are currencies with three decimal places.
var threeDecimal = map[string]bool{"BHD": true, "IQD": true, "JOD": true, "KWD": true, "LYD": true, "OMR": true, "TND": true}

// Exponent is how many decimals a currency's minor unit has.
func Exponent(currency string) int {
	c := strings.ToUpper(currency)
	switch {
	case zeroDecimal[c]:
		return 0
	case threeDecimal[c]:
		return 3
	}
	return 2
}

// ParseAmount reads a decimal amount ("58.00", "58.0", "1234") into minor
// units of currency, rounding half away from zero past its precision. An
// empty or unreadable amount is 0.
func ParseAmount(s, currency string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(strings.TrimPrefix(s, "-"), "+")
	whole, frac, _ := strings.Cut(s, ".")
	exp := Exponent(currency)
	w, err := strconv.ParseInt(nonEmpty(whole), 10, 64)
	if err != nil {
		return 0
	}
	f := frac
	round := false
	if len(f) > exp {
		round = f[exp] >= '5'
		f = f[:exp]
	}
	for len(f) < exp {
		f += "0"
	}
	var minor int64
	if exp > 0 {
		minor, err = strconv.ParseInt(f, 10, 64)
		if err != nil {
			return 0
		}
	}
	v := w
	for range exp {
		v *= 10
	}
	v += minor
	if round {
		v++
	}
	if neg {
		v = -v
	}
	return v
}

// FormatAmount writes minor units as a decimal amount ("58.00").
func FormatAmount(v int64, currency string) string {
	exp := Exponent(currency)
	neg := v < 0
	if neg {
		v = -v
	}
	s := strconv.FormatInt(v, 10)
	if exp > 0 {
		for len(s) <= exp {
			s = "0" + s
		}
		s = s[:len(s)-exp] + "." + s[len(s)-exp:]
	}
	if neg {
		s = "-" + s
	}
	return s
}

func nonEmpty(s string) string {
	if s == "" {
		return "0"
	}
	return s
}
