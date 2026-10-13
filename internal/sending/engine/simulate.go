package engine

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Drive runs a send on a virtual clock from start for d: one step a
// second, applying the demo transport's events as they come due, exactly as
// the worker would in real time. Tests use it, and the demo command uses
// it to fast-forward a send so the monitor opens mid-send with consistent,
// back-dated history. It returns the virtual time it stopped at and
// whether the send ended (finished, paused or cancelled).
func (e *Engine) Drive(ctx context.Context, campaignID uuid.UUID, start time.Time, d time.Duration) (time.Time, bool, error) {
	end := start.Add(d)
	t := start
	rs, err := e.load(ctx, campaignID, t)
	if errors.Is(err, errStopped) {
		return t, true, e.drain(ctx, t, end)
	}
	if err != nil {
		return t, false, err
	}
	for ; t.Before(end); t = t.Add(time.Second) {
		if err := ctx.Err(); err != nil {
			return t, false, err
		}
		if _, err := e.Step(ctx, rs, t); err != nil {
			if errors.Is(err, errStopped) {
				return t, true, e.drain(ctx, t, end)
			}
			return t, false, err
		}
		if _, _, err := e.PumpSim(ctx, t); err != nil {
			return t, false, err
		}
	}
	return t, false, nil
}

// drain applies the demo transport's events up to end once nothing sends
// any more (deliveries and opens keep arriving after the last message).
func (e *Engine) drain(ctx context.Context, from, end time.Time) error {
	for t := from; t.Before(end); t = t.Add(5 * time.Second) {
		if _, _, err := e.PumpSim(ctx, t); err != nil {
			return err
		}
	}
	_, _, err := e.PumpSim(ctx, end)
	return err
}

// DrivePrepare builds the recipient list on a virtual clock (tests and the
// demo command).
func (e *Engine) DrivePrepare(ctx context.Context, campaignID uuid.UUID, now time.Time) error {
	return e.prepare(ctx, campaignID, now)
}
