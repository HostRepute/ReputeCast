package demo

import (
	"testing"
	"time"
)

// List health counts the contacts entering Sunset and the Broadcasts
// stopping in the next 7 days with Postgres adding days to a timestamptz in
// a UTC session: 181 and 30 days of 24 hours. The demo's dates have to land
// in that window at any time of day, also when the 30 or 181 days go back
// across the change from summer time in New York. CI on Friday 13 November
// 2026 at 06:46 UTC (01:46 in New York) found 374 Broadcasts stopping
// instead of 893; a Wednesday morning did the same to those entering.
func TestSunsetWeekAcrossTheTimeChange(t *testing.T) {
	for _, at := range []time.Time{
		time.Date(2026, 11, 13, 6, 46, 0, 0, time.UTC),  // Friday 01:46 in New York
		time.Date(2026, 11, 11, 14, 20, 0, 0, time.UTC), // Wednesday 09:20 in New York
		time.Date(2026, 7, 10, 6, 46, 0, 0, time.UTC),   // a summer Friday
		time.Date(2026, 3, 13, 6, 46, 0, 0, time.UTC),   // the Friday before the clocks go forward
	} {
		g := &audGen{s: &Seeder{Now: at}, rng: audRand(1, 2), now: at}
		a := &audSeed{}
		for range 1204 {
			a.contacts = append(a.contacts, &audContact{tier: "at_risk", created: at.AddDate(-2, 0, 0)})
		}
		old := at.AddDate(-1, 0, 0)
		for range 893 {
			a.contacts = append(a.contacts, &audContact{tier: "sunset", created: at.AddDate(-2, 0, 0), lastHuman: &old, lastOpen: &old})
		}
		audShapeSunset(g, a)
		in := func(t time.Time) bool { return t.After(at) && !t.After(at.Add(7*24*time.Hour)) }
		entering, stopping := 0, 0
		for _, c := range a.contacts {
			switch c.tier {
			case "at_risk":
				if in(c.lastHuman.Add(181 * 24 * time.Hour)) {
					entering++
				}
			case "sunset":
				if c.sunsetAt != nil && in(c.sunsetAt.Add(30*24*time.Hour)) {
					stopping++
				}
			}
		}
		if entering != 1204 || stopping != 893 {
			t.Errorf("%s: %d entering and %d stopping this week, want 1204 and 893", at.Format(time.RFC3339), entering, stopping)
		}
	}
}
