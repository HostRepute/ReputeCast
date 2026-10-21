package channels

import (
	"errors"
	"fmt"
	"time"
)

// Quiet hours (design expansion phase 8, brief §3.5): texts, notifications
// and WhatsApp messages go out only inside a daily window in each
// recipient's own time zone. The window can be narrowed anywhere between
// 08:00 and 21:00; ReputeCast never sends outside those hard limits.

// Hard limits, in minutes after local midnight.
const (
	HardFrom = 8 * 60
	HardTo   = 21 * 60
)

// DefaultWindow is 09:00–20:00.
var DefaultWindow = Window{From: 9 * 60, To: 20 * 60}

// Window is a daily sending window, in minutes after local midnight: From
// is the first minute texts may leave, To the first one they may not.
type Window struct {
	From int `json:"from" minimum:"0" maximum:"1440"`
	To   int `json:"to" minimum:"0" maximum:"1440"`
}

// Errors from Validate. The app words them; the API names the limit.
var (
	ErrWindowEmpty     = errors.New("channels: the window ends before it starts")
	ErrWindowTooEarly  = errors.New("channels: texts can't go out before 08:00 local time")
	ErrWindowTooLate   = errors.New("channels: texts can't go out after 21:00 local time")
	errWindowMalformed = errors.New("channels: a time is outside 00:00–24:00")
)

// Validate checks the window against the hard limits.
func (w Window) Validate() error {
	switch {
	case w.From < 0 || w.To > 24*60 || w.From > 24*60 || w.To < 0:
		return errWindowMalformed
	case w.To <= w.From:
		return ErrWindowEmpty
	case w.From < HardFrom:
		return ErrWindowTooEarly
	case w.To > HardTo:
		return ErrWindowTooLate
	}
	return nil
}

// Clamp narrows the window to the hard limits. Stored windows are always
// valid; Clamp guards a window read from older data.
func (w Window) Clamp() Window {
	if w.From < HardFrom {
		w.From = HardFrom
	}
	if w.To > HardTo {
		w.To = HardTo
	}
	if w.To <= w.From {
		return DefaultWindow
	}
	return w
}

// String is "09:00–20:00".
func (w Window) String() string { return Clock(w.From) + "–" + Clock(w.To) }

// Clock writes minutes after midnight as 24-hour time: 540 → "09:00".
func Clock(m int) string { return fmt.Sprintf("%02d:%02d", m/60, m%60) }

// minuteOf is t's minute after local midnight.
func minuteOf(t time.Time) int { return t.Hour()*60 + t.Minute() }

// Inside reports whether t is inside the window in loc.
func (w Window) Inside(t time.Time, loc *time.Location) bool {
	w = w.Clamp()
	m := minuteOf(t.In(loc))
	return m >= w.From && m < w.To
}

// at is the local time on t's local day at minute m.
func at(t time.Time, loc *time.Location, m int) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), m/60, m%60, 0, 0, loc)
}

// NextOpen is the earliest moment at or after t inside the window in loc:
// t itself when the window is open, else the next opening.
func (w Window) NextOpen(t time.Time, loc *time.Location) time.Time {
	w = w.Clamp()
	if w.Inside(t, loc) {
		return t
	}
	open := at(t, loc, w.From)
	if !open.After(t) {
		// After today's window: tomorrow's opening (AddDate keeps the wall
		// clock across a DST change).
		open = at(t.In(loc).AddDate(0, 0, 1), loc, w.From)
	}
	return open
}

// Closes is when the window that contains t closes, in loc.
func (w Window) Closes(t time.Time, loc *time.Location) time.Time {
	return at(t, loc, w.Clamp().To)
}

// LocalSend is the moment a text scheduled "at a time in each recipient's
// time zone" goes out: the date and minute in loc. It reports false when
// the minute is outside the window, which the app never offers.
func (w Window) LocalSend(date time.Time, minute int, loc *time.Location) (time.Time, bool) {
	w = w.Clamp()
	t := time.Date(date.Year(), date.Month(), date.Day(), minute/60, minute%60, 0, 0, loc)
	return t, minute >= w.From && minute < w.To
}

// Hold is what quiet hours do to a text due at t: send it then, or hold it
// until the window opens.
type Hold struct {
	Held  bool      `json:"held"`
	Until time.Time `json:"until"`
}

// Check applies the window to a text due at t for a recipient in loc.
func (w Window) Check(t time.Time, loc *time.Location) Hold {
	next := w.NextOpen(t, loc)
	if next.Equal(t) {
		return Hold{Until: t}
	}
	return Hold{Held: true, Until: next}
}
