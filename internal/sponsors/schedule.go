package sponsors

import (
	"time"

	"github.com/hostrepute/reputecast/internal/store"
)

// A slot's issues follow its list's schedule: issue n of a slot is
// first_number's date plus every_days for each number after it. Dates are
// the workspace's local calendar days (stored as midnight UTC).

// civil is a calendar day: midnight UTC of y-m-d.
func civil(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// issueOn is the date of issue n of a slot.
func issueOn(firstNumber int32, firstOn time.Time, everyDays int32, n int) time.Time {
	return civil(firstOn).AddDate(0, 0, (n-int(firstNumber))*int(everyDays))
}

// IssueOn is the date of issue n of a slot.
func IssueOn(sl store.SponsorSlot, n int) time.Time {
	return issueOn(sl.FirstNumber, sl.FirstOn, sl.EveryDays, n)
}

// NextNumber is the number of the first issue on or after day.
func NextNumber(sl store.SponsorSlot, day time.Time) int {
	every := int(sl.EveryDays)
	if every <= 0 {
		every = 7
	}
	diff := int(civil(day).Sub(civil(sl.FirstOn)).Hours() / 24)
	k := diff / every
	if diff > 0 && diff%every != 0 {
		k++
	}
	if diff < 0 {
		// Floor towards the past, then step to the first one on or after.
		k = -((-diff) / every)
	}
	return int(sl.FirstNumber) + k
}

// NumberOn is the issue falling on day, if any.
func NumberOn(sl store.SponsorSlot, day time.Time) (int, bool) {
	n := NextNumber(sl, day)
	return n, IssueOn(sl, n).Equal(civil(day))
}

// noonOn is a calendar day as a moment the app formats in the workspace's
// time zone without moving to another day: its noon there.
func noonOn(day time.Time, ws *store.Workspace) time.Time {
	d := civil(day)
	return time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, location(ws))
}

func noonPtr(day *time.Time, ws *store.Workspace) *time.Time {
	if day == nil {
		return nil
	}
	t := noonOn(*day, ws)
	return &t
}

// localDay is the workspace's calendar day of a moment.
func localDay(t time.Time, ws *store.Workspace) time.Time {
	return civil(t.In(location(ws)))
}

// dayRange is the moments of a local calendar day.
func dayRange(day time.Time, ws *store.Workspace) (time.Time, time.Time) {
	d := civil(day)
	from := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, location(ws))
	return from, from.AddDate(0, 0, 1)
}
