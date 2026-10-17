package digests

import (
	"testing"
	"time"
)

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestNextSendWeekly(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	s := DigestSchedule{Frequency: FreqWeekly, Weekday: int(time.Friday), Time: "08:00", Timezone: "America/New_York"}
	// Wed 14 Oct 2026 (the boards date their sample "Wed 15 Oct"), 10:30:
	// the next issue is Fri 16 Oct at 08:00.
	now := time.Date(2026, 10, 14, 10, 30, 0, 0, ny)
	if got, want := s.NextSend(now), time.Date(2026, 10, 16, 8, 0, 0, 0, ny); !got.Equal(want) {
		t.Fatalf("next: %v, want %v", got, want)
	}
	// At the send time itself, the next one is a week later.
	if got, want := s.NextSend(time.Date(2026, 10, 16, 8, 0, 0, 0, ny)), time.Date(2026, 10, 23, 8, 0, 0, 0, ny); !got.Equal(want) {
		t.Fatalf("at send time: %v, want %v", got, want)
	}
	// Friday 07:59 is still the same day.
	if got, want := s.NextSend(time.Date(2026, 10, 16, 7, 59, 0, 0, ny)), time.Date(2026, 10, 16, 8, 0, 0, 0, ny); !got.Equal(want) {
		t.Fatalf("just before: %v, want %v", got, want)
	}
	// The time zone is the schedule's, not the caller's: Friday 08:00 in New
	// York is 12:00 UTC (EDT), and 13:00 UTC once clocks go back.
	if got := s.NextSend(time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)); got.UTC().Hour() != 12 {
		t.Fatalf("EDT: %v", got.UTC())
	}
	if got := s.NextSend(time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)); got.UTC().Hour() != 13 || got.In(ny).Hour() != 8 {
		t.Fatalf("EST: %v", got.UTC())
	}
}

func TestNextSendDailyAndMonthly(t *testing.T) {
	berlin := mustLoc(t, "Europe/Berlin")
	daily := DigestSchedule{Frequency: FreqDaily, Time: "07:15", Timezone: "Europe/Berlin"}
	fri := time.Date(2026, 10, 16, 9, 0, 0, 0, berlin)
	if got := daily.NextSend(fri); !got.Equal(time.Date(2026, 10, 17, 7, 15, 0, 0, berlin)) {
		t.Fatalf("daily: %v", got)
	}
	daily.WeekdaysOnly = true
	if got := daily.NextSend(fri); !got.Equal(time.Date(2026, 10, 19, 7, 15, 0, 0, berlin)) {
		t.Fatalf("weekdays: %v (a Friday's next is Monday)", got)
	}
	monthly := DigestSchedule{Frequency: FreqMonthly, MonthRule: "day", MonthDay: 31, Time: "09:00", Timezone: "Europe/Berlin"}
	// November has 30 days: day 31 becomes the last day.
	if got := monthly.NextSend(time.Date(2026, 11, 2, 0, 0, 0, 0, berlin)); !got.Equal(time.Date(2026, 11, 30, 9, 0, 0, 0, berlin)) {
		t.Fatalf("day 31 in November: %v", got)
	}
	if got := monthly.NextSend(time.Date(2027, 2, 1, 0, 0, 0, 0, berlin)); !got.Equal(time.Date(2027, 2, 28, 9, 0, 0, 0, berlin)) {
		t.Fatalf("day 31 in February: %v", got)
	}
	first := DigestSchedule{Frequency: FreqMonthly, MonthRule: "first", Weekday: int(time.Monday), Time: "09:00", Timezone: "Europe/Berlin"}
	if got := first.NextSend(time.Date(2026, 10, 15, 0, 0, 0, 0, berlin)); !got.Equal(time.Date(2026, 11, 2, 9, 0, 0, 0, berlin)) {
		t.Fatalf("first Monday: %v", got)
	}
}

func TestNextSendAcrossDST(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	// 02:30 doesn't exist on Sun 8 Mar 2026 in New York (clocks jump from
	// 02:00 to 03:00): the issue goes out once, an hour into the new time.
	s := DigestSchedule{Frequency: FreqDaily, Time: "02:30", Timezone: "America/New_York"}
	got := s.NextSend(time.Date(2026, 3, 7, 12, 0, 0, 0, ny))
	if !got.Equal(time.Date(2026, 3, 8, 3, 30, 0, 0, ny)) && !got.Equal(time.Date(2026, 3, 8, 1, 30, 0, 0, ny)) {
		t.Fatalf("spring forward: %v", got)
	}
	next := s.NextSend(got)
	if !next.Equal(time.Date(2026, 3, 9, 2, 30, 0, 0, ny)) {
		t.Fatalf("after the gap: %v", next)
	}
	// 01:30 happens twice on Sun 1 Nov 2026: the first one (EDT) is used,
	// and the issue goes out only once that day.
	s.Time = "01:30"
	got = s.NextSend(time.Date(2026, 10, 31, 12, 0, 0, 0, ny))
	if got.UTC() != time.Date(2026, 11, 1, 5, 30, 0, 0, time.UTC) {
		t.Fatalf("fall back: %v (%v UTC)", got, got.UTC())
	}
	if next := s.NextSend(got); !next.Equal(time.Date(2026, 11, 2, 1, 30, 0, 0, ny)) {
		t.Fatalf("once per day across the fold: %v", next)
	}
	// A weekly 08:00 keeps 08:00 local through the change: 7 days and one
	// hour apart in absolute time.
	w := DigestSchedule{Frequency: FreqWeekly, Weekday: int(time.Friday), Time: "08:00", Timezone: "America/New_York"}
	a := w.NextSend(time.Date(2026, 10, 29, 0, 0, 0, 0, ny))
	b := w.NextSend(a)
	if b.Sub(a) != 7*24*time.Hour+time.Hour || b.In(ny).Hour() != 8 {
		t.Fatalf("weekly across DST: %v → %v", a, b)
	}
}

func TestItemsSendAt(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	s := DigestSchedule{Frequency: FreqItems, Count: 3, WindowFrom: "08:00", WindowUntil: "18:00", Timezone: "America/New_York"}
	lead := 3 * time.Hour
	// The third post at 22:40 is prepared then and goes out at 08:00.
	if got := s.ItemsSendAt(time.Date(2026, 10, 15, 22, 40, 0, 0, ny), lead, nil); !got.Equal(time.Date(2026, 10, 16, 8, 0, 0, 0, ny)) {
		t.Fatalf("night: %v", got)
	}
	// In the morning: three hours later, inside the window.
	if got := s.ItemsSendAt(time.Date(2026, 10, 15, 9, 0, 0, 0, ny), lead, nil); !got.Equal(time.Date(2026, 10, 15, 12, 0, 0, 0, ny)) {
		t.Fatalf("morning: %v", got)
	}
	// Before dawn: the window's start.
	if got := s.ItemsSendAt(time.Date(2026, 10, 15, 2, 0, 0, 0, ny), lead, nil); !got.Equal(time.Date(2026, 10, 15, 8, 0, 0, 0, ny)) {
		t.Fatalf("dawn: %v", got)
	}
	// At most once a day: an issue already sent today moves it to tomorrow.
	sent := time.Date(2026, 10, 15, 8, 0, 0, 0, ny)
	if got := s.ItemsSendAt(time.Date(2026, 10, 15, 9, 0, 0, 0, ny), lead, &sent); !got.Equal(time.Date(2026, 10, 16, 8, 0, 0, 0, ny)) {
		t.Fatalf("once a day: %v", got)
	}
}

func TestScheduleValidate(t *testing.T) {
	ok := DigestSchedule{Frequency: FreqWeekly, Weekday: 5, Time: "08:00", Timezone: "America/New_York"}
	if f, _ := ok.Validate(); f != "" {
		t.Fatalf("valid schedule refused: %s", f)
	}
	for _, c := range []struct {
		s     DigestSchedule
		field string
	}{
		{DigestSchedule{Frequency: "hourly", Time: "08:00", Timezone: "UTC"}, "frequency"},
		{DigestSchedule{Frequency: FreqWeekly, Time: "8am", Timezone: "UTC"}, "time"},
		{DigestSchedule{Frequency: FreqWeekly, Time: "08:00", Timezone: "Mars/Olympus"}, "timezone"},
		{DigestSchedule{Frequency: FreqItems, Count: 0, WindowFrom: "08:00", WindowUntil: "18:00", Timezone: "UTC"}, "count"},
		{DigestSchedule{Frequency: FreqItems, Count: 3, WindowFrom: "18:00", WindowUntil: "08:00", Timezone: "UTC"}, "windowUntil"},
		{DigestSchedule{Frequency: FreqMonthly, MonthRule: "day", MonthDay: 0, Time: "08:00", Timezone: "UTC"}, "monthDay"},
	} {
		if f, _ := c.s.Validate(); f != c.field {
			t.Fatalf("%+v: field %q, want %q", c.s, f, c.field)
		}
	}
}
