package reports

import (
	"testing"
	"time"
)

// TestSummaryPeriodBounds: periods are local calendar weeks and months,
// so daylight saving changes and month lengths come out right.
func TestSummaryPeriodBounds(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skip(err)
	}
	at := func(y int, m time.Month, d, h, min int) time.Time { return time.Date(y, m, d, h, min, 0, 0, berlin) }
	for _, tc := range []struct {
		name     string
		freq     string
		now      time.Time
		from, to time.Time
		hours    float64
	}{
		// Clocks go forward on Sunday 30 Mar 2025 and back on 26 Oct 2025.
		{"week ending 31 Mar 2025", "weekly", at(2025, 3, 31, 9, 30), at(2025, 3, 24, 0, 0), at(2025, 3, 31, 0, 0), 167},
		{"week ending 27 Oct 2025", "weekly", at(2025, 10, 27, 10, 0), at(2025, 10, 20, 0, 0), at(2025, 10, 27, 0, 0), 169},
		// Before 09:00 on Monday the week before is still the last one due.
		{"before Monday 09:00", "weekly", at(2025, 10, 27, 8, 59), at(2025, 10, 13, 0, 0), at(2025, 10, 20, 0, 0), 168},
		{"midweek", "weekly", at(2025, 10, 29, 15, 0), at(2025, 10, 20, 0, 0), at(2025, 10, 27, 0, 0), 169},
		{"February 2028", "monthly", at(2028, 3, 1, 9, 0), at(2028, 2, 1, 0, 0), at(2028, 3, 1, 0, 0), 29 * 24},
		// September 2025 starts on a Monday: the month, not its first week.
		{"a month starting on a Monday", "monthly", at(2025, 10, 1, 12, 0), at(2025, 9, 1, 0, 0), at(2025, 10, 1, 0, 0), 30 * 24},
		{"before 09:00 on the 1st", "monthly", at(2028, 3, 1, 8, 0), at(2028, 1, 1, 0, 0), at(2028, 2, 1, 0, 0), 31 * 24},
		{"over new year", "monthly", at(2026, 1, 1, 9, 0), at(2025, 12, 1, 0, 0), at(2026, 1, 1, 0, 0), 31 * 24},
		{"a month with the clocks going back", "monthly", at(2025, 11, 3, 9, 0), at(2025, 10, 1, 0, 0), at(2025, 11, 1, 0, 0), 31*24 + 1},
	} {
		p := lastSummaryPeriod(tc.freq, tc.now, berlin)
		if !p.From.Equal(tc.from) || !p.To.Equal(tc.to) || p.To.Sub(p.From).Hours() != tc.hours {
			t.Errorf("%s: %v → %v (%v hours), want %v → %v (%v)", tc.name, p.From, p.To, p.To.Sub(p.From).Hours(), tc.from, tc.to, tc.hours)
		}
		if want := time.Date(tc.to.Year(), tc.to.Month(), tc.to.Day(), 9, 0, 0, 0, berlin); !p.SendAt.Equal(want) || tc.now.Before(p.SendAt) {
			t.Errorf("%s: due from %v, want %v and not after %v", tc.name, p.SendAt, want, tc.now)
		}
	}
	if fr := lastSummaryPeriod("weekly", at(2025, 3, 31, 10, 0), berlin).From; fr.Weekday() != time.Monday {
		t.Errorf("weeks start on Monday: %v", fr)
	}
	// The next one: the coming Monday or 1st at 09:00.
	if n := nextSummaryAt("weekly", at(2025, 10, 29, 15, 0), berlin); !n.Equal(at(2025, 11, 3, 9, 0)) {
		t.Errorf("next weekly: %v", n)
	}
	if n := nextSummaryAt("weekly", at(2025, 10, 27, 8, 0), berlin); !n.Equal(at(2025, 10, 27, 9, 0)) {
		t.Errorf("next weekly on a Monday morning: %v", n)
	}
	if n := nextSummaryAt("monthly", at(2025, 12, 15, 9, 0), berlin); !n.Equal(at(2026, 1, 1, 9, 0)) {
		t.Errorf("next monthly: %v", n)
	}
}

// TestSummaryZone: the member's zone, else the workspace's.
func TestSummaryZone(t *testing.T) {
	if z := summaryZone("Asia/Ho_Chi_Minh", "Europe/Berlin"); z.String() != "Asia/Ho_Chi_Minh" {
		t.Errorf("member's zone: %v", z)
	}
	if z := summaryZone("", "Europe/Berlin"); z.String() != "Europe/Berlin" {
		t.Errorf("workspace's zone: %v", z)
	}
	if z := summaryZone("Nowhere/Else", ""); z != time.UTC {
		t.Errorf("unknown zones: %v", z)
	}
}

// TestPeriodMemo: members who share a period share its numbers, for ten
// minutes, in a cache that stays small.
func TestPeriodMemo(t *testing.T) {
	var m periodMemo
	now := time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)
	k := periodKey{frequency: "weekly", from: 1, to: 2}
	a := m.entry(k, now)
	if b := m.entry(k, now.Add(9*time.Minute)); b != a {
		t.Error("the same period within ten minutes is computed again")
	}
	if b := m.entry(k, now.Add(11*time.Minute)); b == a {
		t.Error("numbers older than ten minutes are reused")
	}
	for i := range periodMemoSize * 2 {
		m.entry(periodKey{frequency: "weekly", from: int64(i + 10), to: 2}, now)
	}
	if m.order.Len() != periodMemoSize || len(m.items) != periodMemoSize {
		t.Errorf("cache holds %d entries (%d keys)", m.order.Len(), len(m.items))
	}
}
