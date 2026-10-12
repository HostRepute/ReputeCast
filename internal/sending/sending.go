// Package sending runs campaign sends: it builds the recipient list,
// applies suppressions, routes each recipient to a sending provider, throttles
// per provider and per mailbox provider, sends in batches through River jobs,
// updates live counters and pauses automatically when thresholds trip.
package sending

import (
	"context"
	"errors"

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
