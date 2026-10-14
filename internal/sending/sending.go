// Package sending runs campaign sends: it builds the recipient list,
// applies suppressions, routes each recipient to a sending provider, throttles
// per provider and per mailbox provider, sends in batches through River jobs,
// updates live counters and pauses automatically when thresholds trip.
package sending

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNotReady is returned when Send Readiness blocks the send at start.
var ErrNotReady = errors.New("send readiness is blocked")

// Starter starts or schedules a campaign send. The review & send screen
// calls it after confirmation (and after approval, when required).
type Starter interface {
	Start(ctx context.Context, campaignID uuid.UUID, startedBy uuid.UUID) error
	Pause(ctx context.Context, campaignID uuid.UUID, by uuid.UUID, reason string) error
	Resume(ctx context.Context, campaignID uuid.UUID, by uuid.UUID) error
	Cancel(ctx context.Context, campaignID uuid.UUID, by uuid.UUID) error
}

// ErrUnavailable is returned by Unavailable, before the sending domain is wired.
var ErrUnavailable = errors.New("sending is not available in this process")

// Unavailable is the Starter used until the sending domain replaces it.
type Unavailable struct{}

// Start always fails with ErrUnavailable.
func (Unavailable) Start(context.Context, uuid.UUID, uuid.UUID) error { return ErrUnavailable }

// Pause always fails with ErrUnavailable.
func (Unavailable) Pause(context.Context, uuid.UUID, uuid.UUID, string) error { return ErrUnavailable }

// Resume always fails with ErrUnavailable.
func (Unavailable) Resume(context.Context, uuid.UUID, uuid.UUID) error { return ErrUnavailable }

// Cancel always fails with ErrUnavailable.
func (Unavailable) Cancel(context.Context, uuid.UUID, uuid.UUID) error { return ErrUnavailable }

// Engagement is one open-pixel fetch or link click, as the tracking
// endpoints (and the demo transport) report it. Link is the click's link
// position; UserAgent, IP and Method decide human or machine.
type Engagement struct {
	MessageID uuid.UUID
	Click     bool
	Link      int
	UserAgent string
	IP        string
	Method    string
	At        time.Time
}

// Recorder records opens and clicks (the tracking domain implements it).
type Recorder interface {
	RecordEngagement(ctx context.Context, evs []Engagement) error
}

// Readiness reruns Send Readiness when a send starts (the send-time
// recheck, add-on §3.8) and supplies what Send Readiness computed for the
// send: per-mailbox-provider throttles. The readiness package satisfies it;
// until it is wired the engine uses ReadyAlways.
type Readiness interface {
	Recheck(ctx context.Context, campaignID uuid.UUID) (Recheck, error)
}

// Recheck is the result of the send-time recheck.
type Recheck struct {
	// Verdict is "ready", "warnings" or "blocked".
	Verdict string
	// Blocking lists the titles of the checks that block.
	Blocking []string
	// Throttles are messages per second per mailbox provider ("outlook": 5);
	// nil keeps the engine's defaults.
	Throttles map[string]float64
}

// ReadyAlways never blocks and keeps the default throttles.
type ReadyAlways struct{}

// Recheck implements Readiness.
func (ReadyAlways) Recheck(context.Context, uuid.UUID) (Recheck, error) {
	return Recheck{Verdict: "ready"}, nil
}

// Segments resolves saved segments to contact ids (the audience domain).
type Segments interface {
	ContactIDs(ctx context.Context, workspaceID, segmentID uuid.UUID) ([]uuid.UUID, error)
}
