package digests

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Frequencies of a digest's schedule.
const (
	FreqDaily   = "daily"
	FreqWeekly  = "weekly"
	FreqMonthly = "monthly"
	// FreqItems prepares an issue as soon as Count new items arrived, at
	// most once a day, inside a sending window.
	FreqItems = "items"
)

// DigestSchedule is when a digest's issues go out (digests.schedule).
type DigestSchedule struct {
	Frequency string `json:"frequency" enum:"daily,weekly,monthly,items"`
	// WeekdaysOnly: a daily digest that skips Saturdays and Sundays.
	WeekdaysOnly bool `json:"weekdaysOnly,omitempty"`
	// Weekday is the day of a weekly digest, and of a monthly digest that
	// goes out on the first such day of the month: 0 Sunday … 6 Saturday.
	Weekday int `json:"weekday" minimum:"0" maximum:"6"`
	// MonthRule is "day" (a day of the month) or "first" (the first Weekday).
	MonthRule string `json:"monthRule,omitempty" enum:"day,first,"`
	// MonthDay is 1–31; months without that day use their last day.
	MonthDay int `json:"monthDay,omitempty" minimum:"0" maximum:"31"`
	// Time is the send time, HH:MM in Timezone.
	Time     string `json:"time" pattern:"^([01][0-9]|2[0-3]):[0-5][0-9]$" example:"08:00"`
	Timezone string `json:"timezone" maxLength:"64" example:"America/New_York"`
	// Count, WindowFrom and WindowUntil describe an item-count schedule.
	Count       int    `json:"count,omitempty" minimum:"0" maximum:"50"`
	WindowFrom  string `json:"windowFrom,omitempty" pattern:"^(([01][0-9]|2[0-3]):[0-5][0-9])?$"`
	WindowUntil string `json:"windowUntil,omitempty" pattern:"^(([01][0-9]|2[0-3]):[0-5][0-9])?$"`
}

// DefaultSchedule is a new digest's: Fridays at 08:00 in the workspace's time zone.
func DefaultSchedule(tz string) DigestSchedule {
	return DigestSchedule{Frequency: FreqWeekly, Weekday: int(time.Friday), Time: "08:00", Timezone: tz, MonthRule: "day", MonthDay: 1, Count: 3, WindowFrom: "08:00", WindowUntil: "18:00"}
}

// ParseSchedule reads digests.schedule; a broken value is the default.
func ParseSchedule(raw json.RawMessage, tz string) DigestSchedule {
	s := DefaultSchedule(tz)
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &s)
	}
	if s.Timezone == "" {
		s.Timezone = tz
	}
	if s.Time == "" {
		s.Time = "08:00"
	}
	return s
}

// Validate checks a schedule and says what to change: the field, and the
// message (with its arguments) to write in the reader's language.
func (s DigestSchedule) Validate() (field string, msg i18n.Msg, args i18n.Args) {
	switch s.Frequency {
	case FreqDaily, FreqWeekly, FreqMonthly, FreqItems:
	default:
		return "frequency", mScheduleFrequency, nil
	}
	if _, err := time.LoadLocation(s.Timezone); err != nil || s.Timezone == "" {
		return "timezone", mScheduleTimezone, nil
	}
	if s.Frequency == FreqItems {
		if s.Count < 1 || s.Count > 50 {
			return "count", mScheduleCount, nil
		}
		from, ok1 := clockMinutes(s.WindowFrom)
		until, ok2 := clockMinutes(s.WindowUntil)
		if !ok1 || !ok2 {
			return "windowFrom", mScheduleWindow, nil
		}
		if until <= from {
			// A time of day, written on the reader's clock.
			return "windowUntil", mScheduleWindowOrder, i18n.Args{"from": time.Date(2000, 1, 1, from/60, from%60, 0, 0, time.UTC)}
		}
		return "", i18n.Msg{}, nil
	}
	if _, ok := clockMinutes(s.Time); !ok {
		return "time", mScheduleTime, nil
	}
	if s.Weekday < 0 || s.Weekday > 6 {
		return "weekday", mScheduleWeekday, nil
	}
	if s.Frequency == FreqMonthly && s.MonthRule != "first" && (s.MonthDay < 1 || s.MonthDay > 31) {
		return "monthDay", mScheduleMonthDay, nil
	}
	return "", i18n.Msg{}, nil
}

func (s DigestSchedule) location() *time.Location {
	if loc, err := time.LoadLocation(s.Timezone); err == nil {
		return loc
	}
	return time.UTC
}

// clockMinutes reads "HH:MM".
func clockMinutes(v string) (int, bool) {
	h, m, ok := strings.Cut(strings.TrimSpace(v), ":")
	if !ok {
		return 0, false
	}
	hh, err1 := strconv.Atoi(h)
	mm, err2 := strconv.Atoi(m)
	if err1 != nil || err2 != nil || hh < 0 || hh > 23 || mm < 0 || mm > 59 {
		return 0, false
	}
	return hh*60 + mm, true
}

// at is hh:mm on the local day of d. A time that doesn't exist that day
// (inside a daylight-saving gap) moves forward by the gap; one that exists
// twice (when clocks go back) is the first of the two.
func at(d time.Time, minutes int, loc *time.Location) time.Time {
	y, mo, day := d.Date()
	t := time.Date(y, mo, day, minutes/60, minutes%60, 0, 0, loc)
	// time.Date may pick the second of two equal wall times; prefer the first.
	if earlier := t.Add(-time.Hour); earlier.Hour() == t.Hour() && earlier.Minute() == t.Minute() && earlier.Day() == t.Day() {
		return earlier
	}
	return t
}

// matches reports whether a local day is a send day.
func (s DigestSchedule) matches(d time.Time) bool {
	switch s.Frequency {
	case FreqDaily:
		return !s.WeekdaysOnly || (d.Weekday() != time.Saturday && d.Weekday() != time.Sunday)
	case FreqWeekly:
		return int(d.Weekday()) == s.Weekday
	case FreqMonthly:
		if s.MonthRule == "first" {
			return int(d.Weekday()) == s.Weekday && d.Day() <= 7
		}
		last := time.Date(d.Year(), d.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
		return d.Day() == min(s.MonthDay, last)
	}
	return false
}

// NextSend is the first send time of a daily, weekly or monthly schedule
// strictly after t; the zero time for an item-count schedule.
func (s DigestSchedule) NextSend(t time.Time) time.Time {
	if s.Frequency == FreqItems {
		return time.Time{}
	}
	loc := s.location()
	minutes, ok := clockMinutes(s.Time)
	if !ok {
		minutes = 8 * 60
	}
	day := t.In(loc)
	day = time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, loc)
	for i := 0; i < 400; i++ {
		d := day.AddDate(0, 0, i)
		if !s.matches(d) {
			continue
		}
		if send := at(d, minutes, loc); send.After(t) {
			return send
		}
	}
	return time.Time{}
}

// ItemsSendAt is when an item-count issue prepared at prepared goes out:
// after the preparation lead, inside the sending window (a time outside it
// moves to the next window), and on another day than lastSent's.
func (s DigestSchedule) ItemsSendAt(prepared time.Time, lead time.Duration, lastSent *time.Time) time.Time {
	loc := s.location()
	from, ok1 := clockMinutes(s.WindowFrom)
	until, ok2 := clockMinutes(s.WindowUntil)
	if !ok1 || !ok2 || until <= from {
		from, until = 8*60, 18*60
	}
	t := prepared.Add(lead).In(loc)
	for i := 0; i < 400; i++ {
		startOfWindow := at(t, from, loc)
		endOfWindow := at(t, until, loc)
		switch {
		case t.Before(startOfWindow):
			t = startOfWindow
		case t.After(endOfWindow):
			next := time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, loc).AddDate(0, 0, 1)
			t = at(next, from, loc)
			continue
		}
		if lastSent != nil && sameDay(lastSent.In(loc), t) {
			next := time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, loc).AddDate(0, 0, 1)
			t = at(next, from, loc)
			continue
		}
		return t
	}
	return t
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// Window says whether an item-count schedule may prepare now (the third
// post at 22:40 is prepared then; it goes out in the next window).
func (s DigestSchedule) sendWindow() (from, until int) {
	from, ok1 := clockMinutes(s.WindowFrom)
	until, ok2 := clockMinutes(s.WindowUntil)
	if !ok1 || !ok2 || until <= from {
		return 8 * 60, 18 * 60
	}
	return from, until
}

// Describe is the schedule in a sentence, in English, for notifications
// and the audit log ("every Friday at 08:00 America/New_York").
func (s DigestSchedule) Describe() string {
	switch s.Frequency {
	case FreqDaily:
		if s.WeekdaysOnly {
			return fmt.Sprintf("every weekday at %s %s", s.Time, s.Timezone)
		}
		return fmt.Sprintf("every day at %s %s", s.Time, s.Timezone)
	case FreqWeekly:
		return fmt.Sprintf("every %s at %s %s", time.Weekday(s.Weekday), s.Time, s.Timezone)
	case FreqMonthly:
		if s.MonthRule == "first" {
			return fmt.Sprintf("the first %s of every month at %s %s", time.Weekday(s.Weekday), s.Time, s.Timezone)
		}
		return fmt.Sprintf("day %d of every month at %s %s", s.MonthDay, s.Time, s.Timezone)
	case FreqItems:
		from, until := s.sendWindow()
		return fmt.Sprintf("when %d new items arrive, at most once a day, between %02d:%02d and %02d:%02d %s", s.Count, from/60, from%60, until/60, until%60, s.Timezone)
	}
	return ""
}
