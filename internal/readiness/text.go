package readiness

import (
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Number formats in the product voice: "31,604", "0.6%", "$2.84", "09:40".
// They write English only (API clients, approvals' own texts); the texts
// of a run format their numbers with its messages, in each reader's region.

// Count formats n with thousands separators.
func Count(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if neg {
		return "−" + b.String()
	}
	return b.String()
}

// Pct formats a percentage with up to two decimals, trailing zeros trimmed:
// 0.6 → "0.6%", 0.04 → "0.04%", 5 → "5%".
func Pct(v float64) string {
	return trimNumber(v, 2) + "%"
}

// Pct1 formats a percentage with one decimal: "3.1%".
func Pct1(v float64) string {
	return strconv.FormatFloat(math.Round(v*10)/10, 'f', 1, 64) + "%"
}

// Money formats US dollars: "$2.84".
func Money(v float64) string {
	return "$" + strconv.FormatFloat(math.Round(v*100)/100, 'f', 2, 64)
}

// Per1000 formats a metered price: "$0.10 per 1,000".
func Per1000(v float64) string {
	return Money(v) + " per 1,000"
}

// Rate formats messages per second: "14 /s", "2.5 /s".
func Rate(v float64) string {
	return trimNumber(v, 1) + " /s"
}

func trimNumber(v float64, digits int) string {
	s := strconv.FormatFloat(v, 'f', digits, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}

// Clock formats a time in the workspace zone: "09:40" today, "Mon 12 Oct,
// 09:40" on another day.
func Clock(t, now time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	lt, ln := t.In(loc), now.In(loc)
	if lt.Year() == ln.Year() && lt.YearDay() == ln.YearDay() {
		return lt.Format("15:04")
	}
	return lt.Format("Mon 2 Jan, 15:04")
}

// Plural picks the singular or plural form.
func Plural(n int64, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// JoinAnd joins words as "A, B and C".
func JoinAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

// Minutes rounds a duration in seconds up to whole minutes (at least 1 when
// anything is sent).
func Minutes(seconds float64) int {
	if seconds <= 0 {
		return 0
	}
	return int(math.Max(1, math.Round(seconds/60)))
}

// lowerFirst lower-cases the first letter unless the first word looks like
// a name (two capitals, a dot or a digit inside).
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	word := s
	if i := strings.IndexAny(s, " :"); i > 0 {
		word = s[:i]
	}
	if strings.ContainsAny(word, ".@0123456789\"“`") {
		return s
	}
	upper := 0
	for _, r := range word {
		if unicode.IsUpper(r) {
			upper++
		}
	}
	if upper > 1 {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[n:]
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}
