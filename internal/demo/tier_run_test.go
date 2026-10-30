package demo

import (
	"testing"
	"time"
)

// The demo's last tier run leaves room for the Cooling contacts who opened
// after it: a seed a few minutes after 03:00 uses the day before's run.
func TestTierRunLeavesRoom(t *testing.T) {
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("no time zone data")
	}
	day := func(d, h, m int) time.Time { return time.Date(2026, 10, d, h, m, 0, 0, ny) }
	for _, c := range []struct{ now, want time.Time }{
		{day(27, 2, 0), day(26, 3, 0)},
		{day(27, 3, 2), day(26, 3, 0)},
		{day(27, 3, 15), day(27, 3, 0)},
		{day(27, 18, 0), day(27, 3, 0)},
	} {
		if got := audTierRun(&Seeder{Now: c.now}); !got.Equal(c.want) {
			t.Errorf("at %s: %s, want %s", c.now.Format("15:04"), got, c.want)
		}
	}
}
