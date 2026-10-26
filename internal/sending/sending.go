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

	"github.com/hostrepute/reputecast/internal/i18n"
)

// ErrNotReady is returned when Send Readiness blocks the send at start.
var ErrNotReady = errors.New("send readiness is blocked")

// NotReadyError is ErrNotReady with what blocked the send.
type NotReadyError struct {
	Reason   string
	Blocking []string
}

func (e *NotReadyError) Error() string { return ErrNotReady.Error() + ": " + e.Reason }

// Is makes errors.Is(err, ErrNotReady) hold.
func (e *NotReadyError) Is(target error) bool { return target == ErrNotReady }

// Explained is an error that says why in a reader's language too, as the
// engine's state errors do ("Weekly Digest #142 is already sending").
type Explained interface {
	error
	In(l i18n.Lang) string
}

type recheckKey struct{ campaign uuid.UUID }

// WithRecheck hands Start the send-time recheck its caller has just run for
// the campaign, so Send Readiness doesn't run a second time.
func WithRecheck(ctx context.Context, campaignID uuid.UUID, rc Recheck) context.Context {
	return context.WithValue(ctx, recheckKey{campaignID}, rc)
}

// RecheckFrom returns the recheck WithRecheck attached for the campaign.
func RecheckFrom(ctx context.Context, campaignID uuid.UUID) (Recheck, bool) {
	rc, ok := ctx.Value(recheckKey{campaignID}).(Recheck)
	return rc, ok
}

// Starter starts or schedules a campaign send. The review & send screen
// calls it after confirmation (and after approval, when required).
type Starter interface {
	Start(ctx context.Context, campaignID uuid.UUID, startedBy uuid.UUID) error
	Pause(ctx context.Context, campaignID uuid.UUID, by uuid.UUID, reason string) error
	Resume(ctx context.Context, campaignID uuid.UUID, by uuid.UUID) error
	Cancel(ctx context.Context, campaignID uuid.UUID, by uuid.UUID) error
}

// Outages is the sending engine's side of a provider outage that the
// provider health job found (review GS-3): every running send's share on
// that provider goes through the engine's outage path, the same as an
// outage the send loop finds itself: held with its analysis and ways to
// continue (retry with a probe, move to a vetted provider, cancel the
// share), or moved when the Broadcast failover policy allows it. The engine
// implements it; the health job type-asserts its Starter.
type Outages interface {
	// HoldShares holds the shares on a provider; reason is "provider_down"
	// or "auth".
	HoldShares(ctx context.Context, providerID uuid.UUID, reason string) error
}

// ShareMover moves what is left of a held share to another provider, vetted
// as every move is, keeping A/B and canary holdouts held (review GS-5). The
// engine implements it; the providers screen's decision on a held share
// goes through it.
type ShareMover interface {
	MoveShare(ctx context.Context, campaignID, by, from, to uuid.UUID) (int64, error)
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
// recheck, add-on §3.8) and supplies its per-mailbox-provider throttles.
// The engine falls back to ReadyAlways when none is wired.
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
	// Audience is the final recipient count the recheck counted, and RunID
	// its Send Readiness run; a hold records both.
	Audience int64
	RunID    string
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
