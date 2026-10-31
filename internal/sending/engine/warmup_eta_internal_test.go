package engine

import (
	"testing"
	"time"
)

// A share waiting for tomorrow's warm-up cap is done once enough days'
// caps have sent it (review PR-8): calendar days in the workspace's time
// zone, so the change of clocks on 1 November doesn't move midnight.
func TestWarmETAAfter(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no time zone data:", err)
	}
	next := time.Date(2026, 10, 28, 0, 0, 0, 0, ny)
	now := next.Add(-13 * time.Hour)
	w := warmETA{cap: 300, next: next, loc: ny}
	at := func(n int64) time.Time {
		return now.Add(time.Duration(w.after(n, 1, now) * float64(time.Second)))
	}
	// Tomorrow's cap takes 100 at once: done 100 s after midnight.
	if got, want := at(100), next.Add(100*time.Second); !got.Equal(want) {
		t.Fatalf("100 done at %v, want %v", got, want)
	}
	// 1,603 at 300 a day: five days' caps, then 103 more on 2 November.
	if got, want := at(1603), time.Date(2026, 11, 2, 0, 1, 43, 0, ny); !got.Equal(want) {
		t.Fatalf("1,603 done at %v, want %v", got, want)
	}
	// Tomorrow's cap already started (the resume job hasn't run yet).
	late := warmETA{cap: 300, next: now.Add(-time.Minute), loc: ny}
	if s := late.after(50, 1, now); s != 50 {
		t.Fatalf("%v s, want 50", s)
	}
}
