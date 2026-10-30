package demo

import (
	"testing"
	"time"
)

// Whatever the day the demo is seeded, Beta apply launched inside the
// cohort months and before the last 30 days, and every other source of
// natives before the cohorts: on 29 Oct 150 days back was 1 Jun, inside
// them, and the Subscribe page's natives swelled every cohort month.
func TestFormLaunchesAroundCohorts(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	for now := time.Date(2026, 1, 1, 0, 30, 0, 0, ny); now.Year() < 2030; now = now.Add(5 * time.Hour) {
		start := sgCohortStart(now, ny)
		if l := now.In(ny); start.Day() != 1 || start.AddDate(0, 4, 0).Month() != l.Month() {
			t.Fatalf("%s: cohorts start %s", now, start)
		}
		for src, at := range sgFormLaunches(now, start) {
			if src == sgBeta {
				if !at.After(start) || !at.Before(now.AddDate(0, 0, -30)) {
					t.Fatalf("%s: Beta apply launched %s, cohorts from %s", now, at, start)
				}
			} else if !at.Before(start) {
				t.Fatalf("%s: %s launched %s, inside the cohorts from %s", now, src, at, start)
			}
		}
	}
}
