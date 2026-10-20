package channels

import (
	"errors"
	"testing"
	"time"
)

func zone(t *testing.T, name string) *time.Location {
	t.Helper()
	l, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestWindowValidate(t *testing.T) {
	cases := []struct {
		w    Window
		want error
	}{
		{DefaultWindow, nil},
		{Window{From: 8 * 60, To: 21 * 60}, nil},
		{Window{From: 9 * 60, To: 22 * 60}, ErrWindowTooLate},
		{Window{From: 7*60 + 30, To: 20 * 60}, ErrWindowTooEarly},
		{Window{From: 12 * 60, To: 12 * 60}, ErrWindowEmpty},
		{Window{From: 20 * 60, To: 9 * 60}, ErrWindowEmpty},
	}
	for _, c := range cases {
		if got := c.w.Validate(); !errors.Is(got, c.want) || (got == nil) != (c.want == nil) {
			t.Errorf("%s: %v, want %v", c.w, got, c.want)
		}
	}
	// Narrowing is always allowed; widening never.
	if (Window{From: 10 * 60, To: 18 * 60}).Validate() != nil {
		t.Error("a narrower window is valid")
	}
	if w := (Window{From: 6 * 60, To: 23 * 60}).Clamp(); w.From != HardFrom || w.To != HardTo {
		t.Errorf("clamp %s", w)
	}
}

// The route tester's sample: +44 7700 900456 at 21:40 London time waits
// until 09:00.
func TestQuietHoursLondonEvening(t *testing.T) {
	london := zone(t, "Europe/London")
	now := time.Date(2026, 10, 14, 21, 40, 0, 0, london)
	h := DefaultWindow.Check(now, london)
	if !h.Held {
		t.Fatal("21:40 is outside 09:00–20:00")
	}
	want := time.Date(2026, 10, 15, 9, 0, 0, 0, london)
	if !h.Until.Equal(want) {
		t.Fatalf("held until %s, want %s", h.Until, want)
	}
}

// One moment, several zones: 13:40 UTC on 14 Oct is 09:40 in New York,
// 14:40 in London and 15:40 in Berlin (inside the window), but 08:40 in
// Chicago, 06:40 in Los Angeles and 22:40 in Tokyo (held).
func TestQuietHoursAcrossZones(t *testing.T) {
	moment := time.Date(2026, 10, 14, 13, 40, 0, 0, time.UTC)
	cases := []struct {
		zone  string
		held  bool
		until time.Time
	}{
		{"America/New_York", false, moment},
		{"America/Chicago", true, time.Date(2026, 10, 14, 14, 0, 0, 0, time.UTC)},
		{"America/Los_Angeles", true, time.Date(2026, 10, 14, 16, 0, 0, 0, time.UTC)},
		{"Europe/London", false, moment},
		{"Europe/Berlin", false, moment},
		{"Asia/Tokyo", true, time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)}, // 22:40 JST → 09:00 JST next day
	}
	for _, c := range cases {
		loc := zone(t, c.zone)
		h := DefaultWindow.Check(moment, loc)
		if h.Held != c.held || !h.Until.Equal(c.until) {
			t.Errorf("%s: held %v until %s, want %v until %s", c.zone, h.Held, h.Until.UTC(), c.held, c.until)
		}
	}
}

// "At 10:00 in each recipient's time zone" on 14 Oct: from 08:00 UTC in
// Germany and the Netherlands to 17:00 UTC on the US Pacific coast.
func TestLocalSendTimes(t *testing.T) {
	day := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	cases := map[string]time.Time{
		"Europe/Berlin":       time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC),
		"Europe/Amsterdam":    time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC),
		"Europe/London":       time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC),
		"America/Halifax":     time.Date(2026, 10, 14, 13, 0, 0, 0, time.UTC),
		"America/New_York":    time.Date(2026, 10, 14, 14, 0, 0, 0, time.UTC),
		"America/Chicago":     time.Date(2026, 10, 14, 15, 0, 0, 0, time.UTC),
		"America/Denver":      time.Date(2026, 10, 14, 16, 0, 0, 0, time.UTC),
		"America/Los_Angeles": time.Date(2026, 10, 14, 17, 0, 0, 0, time.UTC),
	}
	for name, want := range cases {
		got, ok := DefaultWindow.LocalSend(day, 10*60, zone(t, name))
		if !ok || !got.Equal(want) {
			t.Errorf("%s: %s (%v), want %s", name, got.UTC(), ok, want)
		}
	}
	if _, ok := DefaultWindow.LocalSend(day, 21*60, time.UTC); ok {
		t.Error("21:00 is outside 09:00–20:00 and must not be offered")
	}
}

// Across a DST change the window keeps its wall-clock times: Europe/London
// leaves summer time on 25 Oct 2026.
func TestQuietHoursAcrossDST(t *testing.T) {
	london := zone(t, "Europe/London")
	sat := time.Date(2026, 10, 24, 22, 0, 0, 0, london)
	h := DefaultWindow.Check(sat, london)
	want := time.Date(2026, 10, 25, 9, 0, 0, 0, london)
	if !h.Held || !h.Until.Equal(want) || h.Until.UTC().Hour() != 9 {
		t.Fatalf("held until %s (%s UTC), want 09:00 GMT", h.Until, h.Until.UTC())
	}
	// The narrowest valid window still holds at 08:59 and opens at 09:00.
	w := Window{From: 9 * 60, To: 9*60 + 1}
	if !w.Inside(time.Date(2026, 10, 25, 9, 0, 30, 0, london), london) || w.Inside(time.Date(2026, 10, 25, 8, 59, 0, 0, london), london) {
		t.Fatal("minute-wide window")
	}
}

func TestStatusesAndSwitch(t *testing.T) {
	if SwitchShown([]ChannelStatus{{Channel: Email, SetUp: true}, {Channel: SMS}}) {
		t.Fatal("one channel set up: no switch yet")
	}
	if !SwitchShown([]ChannelStatus{{Channel: Email, SetUp: true}, {Channel: SMS, SetUp: true}}) {
		t.Fatal("two channels: the switch shows")
	}
}
