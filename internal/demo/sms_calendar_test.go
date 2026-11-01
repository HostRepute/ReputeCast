package demo

import (
	"testing"
	"time"
)

// The SMS sample's texts fall in the month the providers page counts
// whenever the month so far can hold them, at the design's clock times, and
// never in the future. A month's first days can't (the campaign is written
// the day before it goes out, and texts reach the Pacific coast inside quiet
// hours only from 12:00 New York time): until the design's today is the 3rd
// they stay together in the month before. Every half hour from late
// September 2026 to April 2027: both daylight saving changes on each side
// of the Atlantic, and months of 28, 30 and 31 days.
func TestSMSCalendar(t *testing.T) {
	ny, chicago, london, la := smsLoc("America/New_York"), smsLoc("America/Chicago"), smsLoc("Europe/London"), smsLoc("America/Los_Angeles")
	if ny == time.UTC {
		t.Skip("no time zone data")
	}
	zones := []*time.Location{smsLoc("Europe/Berlin"), london, smsLoc("America/Halifax"), ny, chicago, smsLoc("America/Denver"), la}
	end := time.Date(2027, 4, 5, 0, 0, 0, 0, ny)
	for now := time.Date(2026, 9, 28, 0, 0, 0, 0, ny); now.Before(end); now = now.Add(30 * time.Minute) {
		c := smsCalendarAt(now, ny)
		n := now.In(ny)
		month := time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, ny)
		// What the month counts: Marcus's first test, the registrations
		// (the confirmations follow them, inside quiet hours or at 09:00
		// the next morning), the campaign's tests and the campaign, the
		// last of it on the Pacific coast; the replies and clicks come
		// within the hour.
		texts := []time.Time{c.textDay(1, 10, 31, chicago), c.textDay(2, 9, 0, chicago), c.textDay(2, 14, 6, chicago), c.textDay(2, 17, 58, london),
			c.textDay(3, 11, 14, chicago), c.textDay(4, 16, 39, chicago), c.textDay(13, 15, 41, chicago), c.textDay(13, 20, 0, chicago),
			c.textDay(14, 9, 31, chicago)}
		for _, z := range zones {
			texts = append(texts, c.textDay(14, 10, 0, z))
		}
		last := c.day(14, 10, 0, la).Add(time.Hour)
		texts = append(texts, last)
		first := texts[0]
		for _, x := range texts {
			if x.Before(first) {
				first = x
			}
		}
		// The days keep their order, never come before the design's, and
		// the campaign keeps its own.
		for d := 1; d <= 14; d++ {
			at := c.textDay(d, 12, 0, ny)
			if at.Before(c.day(d, 12, 0, ny)) || (d > 1 && at.Before(c.textDay(d-1, 12, 0, ny))) || (d >= 13 && !at.Equal(c.day(d, 12, 0, ny))) {
				t.Fatalf("at %s: day %d's texts fall on %s", now, d, at)
			}
		}
		// Today's draft and the registrations read this morning are past too.
		if last.After(now) || c.day(15, 9, 0, chicago).After(now) {
			t.Fatalf("at %s: the sample reaches %s", now, last)
		}
		if !first.Before(month) {
			continue
		}
		// The month before holds them all, in a month's first days only.
		if third := time.Date(n.Year(), n.Month(), 3, 11, 0, 0, 0, ny); !now.Before(third) {
			t.Fatalf("at %s: the first text (%s) falls before the month", now, first)
		}
		if !last.Before(month) || first.Before(month.AddDate(0, -1, 0)) {
			t.Fatalf("at %s: the texts span %s to %s, across the month's turn", now, first, last)
		}
	}
}

// On the 14th the design's first days are last month's: the texts of days 1
// to 13 spread over the 1st to the 12th, and the campaign still went out
// yesterday. On the 20th everything is on the design's days.
func TestSMSCalendarDays(t *testing.T) {
	ny, chicago := smsLoc("America/New_York"), smsLoc("America/Chicago")
	if ny == time.UTC {
		t.Skip("no time zone data")
	}
	day := func(t time.Time) string { return t.In(chicago).Format("Jan 2 15:04") }
	for _, c := range []struct {
		now                          time.Time
		first, register, tests, sent string
	}{
		{time.Date(2026, 11, 14, 12, 0, 0, 0, ny), "Nov 1 10:31", "Nov 2 09:00", "Nov 12 15:41", "Nov 13 09:00"},
		{time.Date(2026, 11, 20, 12, 0, 0, 0, ny), "Nov 6 10:31", "Nov 7 09:00", "Nov 18 15:41", "Nov 19 09:00"},
		// Before 11:00 New York time the design's today is still yesterday.
		{time.Date(2026, 11, 3, 10, 0, 0, 0, ny), "Oct 18 10:31", "Oct 19 09:00", "Oct 30 15:41", "Oct 31 09:00"},
		{time.Date(2026, 11, 3, 12, 0, 0, 0, ny), "Nov 1 10:31", "Nov 1 09:00", "Nov 1 15:41", "Nov 2 09:00"},
		// A month's first days keep them in the month before.
		{time.Date(2026, 11, 1, 5, 0, 0, 0, ny), "Oct 17 10:31", "Oct 18 09:00", "Oct 29 15:41", "Oct 30 09:00"},
		{time.Date(2026, 11, 2, 12, 0, 0, 0, ny), "Oct 18 10:31", "Oct 19 09:00", "Oct 30 15:41", "Oct 31 09:00"},
	} {
		cal := smsCalendarAt(c.now, ny)
		got := []string{day(cal.textDay(1, 10, 31, chicago)), day(cal.textDay(2, 9, 0, chicago)), day(cal.textDay(13, 15, 41, chicago)),
			day(cal.textDay(14, 9, 0, chicago))}
		if want := []string{c.first, c.register, c.tests, c.sent}; got[0] != want[0] || got[1] != want[1] || got[2] != want[2] || got[3] != want[3] {
			t.Errorf("at %s: %v, want %v", c.now, got, want)
		}
	}
}
