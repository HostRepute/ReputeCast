package reports

import (
	"slices"
	"testing"
)

// TestCompareTiles: every tile compares with the median of the last issues
// (review AN-11): the count sent in percent, the rates in points, each read
// as good or bad by metric; an issue counted before privacy-protected opens
// were counted apart isn't compared on opens.
func TestCompareTiles(t *testing.T) {
	issue := func(sent, delivered, human, privacy, machine, clicked, unsub, hard int64) counts {
		return counts{sent: sent, delivered: delivered, human: human, privacy: privacy, machine: machine, clicked: clicked, unsub: unsub, hard: hard}
	}
	now := issue(47980, 47548, 19590, 10436, 1240, 2996, 112, 288)
	pastCounts := []counts{
		issue(46000, 45600, 18000, 9500, 1100, 3000, 90, 200),
		issue(47000, 46500, 18300, 9900, 1000, 3100, 95, 210),
		issue(46500, 46100, 18100, 9700, 1050, 3050, 100, 220),
		// Imported: Apple Mail's prefetches are in its machine opens.
		issue(45000, 44500, 15000, 0, 11000, 2900, 120, 400),
	}
	if !legacyOpens(pastCounts[3]) || legacyOpens(pastCounts[0]) || legacyOpens(counts{}) {
		t.Fatal("legacy opens")
	}
	past := []issueRates{}
	for _, c := range pastCounts {
		r := ratesOf(c, counts{})
		if legacyOpens(c) {
			r.humanMeasurable, r.privacy = nil, nil
		}
		past = append(past, r)
	}
	tiles := map[string]ReportComparison{}
	for _, c := range compareTiles(ratesOf(now, counts{delivered: 27000, complained: 19}), past) {
		tiles[c.Metric] = c
	}
	// Sent: median of 45,000, 46,000, 46,500 and 47,000 is 46,250.
	if s := tiles["sent"]; s.Unit != "percent" || s.Median != 46250 || s.Change != 3.7 || s.Tone != "neutral" {
		t.Fatalf("sent: %+v", s)
	}
	// Human opens of the measurable: the three issues counted with privacy
	// opens apart (49.86%, 50.0%, 49.73%); the imported one's 33.7% isn't
	// in the median.
	if h := tiles["humanOpens"]; h.Unit != "points" || h.Median != 49.86 || h.Change != 2.92 || h.Tone != "good" {
		t.Fatalf("human opens: %+v", h)
	}
	if p := tiles["privacyOpens"]; p.Tone != "neutral" || p.Median != 21.04 {
		t.Fatalf("privacy: %+v", p)
	}
	if c := tiles["clicks"]; c.Tone != "bad" || c.Change >= 0 {
		t.Fatalf("clicks: %+v", c)
	}
	if u := tiles["unsubscribes"]; u.Tone != "bad" || u.Change <= 0 {
		t.Fatalf("unsubscribes: %+v", u)
	}
	if b := tiles["bounces"]; b.Tone != "bad" || b.Change <= 0 {
		t.Fatalf("bounces: %+v", b)
	}
	// No earlier issue with a mailbox provider that reports complaints:
	// complaints aren't compared.
	if _, ok := tiles["complaints"]; ok || len(tiles) != 7 {
		t.Fatalf("tiles: %+v", tiles)
	}
	// Each tile's trend line: the earlier issues oldest first (they're read
	// newest first), then this issue.
	if got := tiles["sent"].Trend; !slices.Equal(got, []float64{45000, 46500, 47000, 46000, 47980}) {
		t.Fatalf("sent trend %v", got)
	}
	// The imported issue has no human opens of the measurable: it isn't a
	// point of that trend.
	if got := tiles["humanOpens"].Trend; !slices.Equal(got, []float64{49.7253, 50, 49.8615, 52.7862}) {
		t.Fatalf("human opens trend %v", got)
	}
	if got := tiles["bounces"].Trend; !slices.Equal(got, []float64{0.8889, 0.4731, 0.4468, 0.4348, 0.6003}) {
		t.Fatalf("bounces trend %v", got)
	}
	for metric, c := range tiles {
		if n := len(c.Trend); n < 2 || c.Trend[n-1] != map[string]float64{"sent": 47980, "delivered": 99.0996, "humanOpens": 52.7862, "privacyOpens": 21.9483,
			"clicks": 6.301, "unsubscribes": 0.2356, "bounces": 0.6003}[metric] {
			t.Fatalf("%s trend %v doesn't end with this issue", metric, c.Trend)
		}
	}
	// Nothing earlier: nothing compared.
	if got := compareTiles(ratesOf(now, counts{}), nil); len(got) != 0 {
		t.Fatalf("without history: %+v", got)
	}
}
