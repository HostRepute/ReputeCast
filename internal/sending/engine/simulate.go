package engine

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Drive runs a send on a virtual clock from start for d, a step a second,
// applying the demo transport's events as they come due; idle stretches
// (retry back-off, the A/B window) are skipped up to the next due event.
// Tests and the demo command's fast-forward use it. It returns the virtual
// time it stopped at and whether the send ended (finished, paused or
// cancelled).
func (e *Engine) Drive(ctx context.Context, campaignID uuid.UUID, start time.Time, d time.Duration) (time.Time, bool, error) {
	unlock, err := e.lockSend(ctx, campaignID)
	if err != nil {
		return start, false, err
	}
	defer unlock()
	return e.drive(ctx, campaignID, start, d)
}

// drive is Drive for a caller that already holds the send's lock.
func (e *Engine) drive(ctx context.Context, campaignID uuid.UUID, start time.Time, d time.Duration) (time.Time, bool, error) {
	end := start.Add(d)
	t := start
	rs, err := e.load(ctx, campaignID, t)
	if errors.Is(err, errStopped) {
		return t, true, e.drain(ctx, t, end)
	}
	if err != nil {
		return t, false, err
	}
	for t.Before(end) {
		if err := ctx.Err(); err != nil {
			return t, false, err
		}
		idle, err := e.Step(ctx, rs, t)
		if err != nil {
			if errors.Is(err, errStopped) {
				return t, true, e.drain(ctx, t, end)
			}
			return t, false, err
		}
		if _, _, err := e.PumpSim(ctx, t); err != nil {
			return t, false, err
		}
		next := t.Add(time.Second)
		if idle > time.Second {
			skip := t.Add(idle.Truncate(time.Second))
			if due, err := e.Q.SendingNextSimEvent(ctx); err == nil && due.Year() > 2000 && due.Before(skip) {
				skip = due.Truncate(time.Second).Add(time.Second)
			}
			if skip.After(next) {
				next = skip
			}
		}
		t = minTime(next, end)
	}
	return t, false, nil
}

// drain applies the demo transport's events up to end once nothing sends
// any more (deliveries and opens keep arriving after the last message),
// jumping from one due event to the next.
func (e *Engine) drain(ctx context.Context, from, end time.Time) error {
	t := from
	for t.Before(end) {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, next, err := e.PumpSim(ctx, t)
		if err != nil {
			return err
		}
		switch {
		case n > 0:
			continue // more may be due at t
		case next == nil || !next.Before(end):
			t = end
		default:
			t = minTime(maxTime(next.Truncate(time.Second).Add(time.Second), t.Add(time.Second)), end)
		}
	}
	_, _, err := e.PumpSim(ctx, end)
	return err
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// DrivePrepare builds the recipient list on a virtual clock (tests and the
// demo command).
func (e *Engine) DrivePrepare(ctx context.Context, campaignID uuid.UUID, now time.Time) error {
	return e.prepare(ctx, campaignID, now)
}
