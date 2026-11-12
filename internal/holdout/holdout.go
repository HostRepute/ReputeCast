// Package holdout is the holdout group (M63 G-33): while a period is open,
// a random share of the workspace's contacts gets no campaigns and/or no
// automation messages, and a nightly measurement compares what they did
// with everyone else.
//
// Who is held out is decided by one SQL function, rc_holdout_bucket(contact,
// seed) < basis_points (migration 03540): the sending engine, Send
// Readiness, automations, text, push and WhatsApp campaigns, the segment
// field and the measurement all ask it, so they never disagree. Raising the
// share keeps everyone already held out; "Start over" draws a new seed.
// Transactional and system mail are never held out.
package holdout

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
)

// Limits of the share, in basis points (hundredths of a percent).
const (
	MinBasisPoints = 10   // 0.1%
	MaxBasisPoints = 2000 // 20%
)

// Percent is a share in basis points as a percentage (500 → 5).
func Percent(bp int32) float64 { return float64(bp) / 100 }

// Open is the workspace's open period; nil when the holdout group is off.
func Open(ctx context.Context, q *store.Queries, workspaceID uuid.UUID) (*store.HoldoutPeriod, error) {
	p, err := q.HoldoutOpenPeriod(ctx, workspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Share is who an audience query holds out: contacts whose bucket for Seed
// is under BasisPoints. The zero Share holds out nobody.
type Share struct {
	Seed        int64 `json:"seed"`
	BasisPoints int32 `json:"basis_points"`
}

// On reports whether the share holds anyone out.
func (s Share) On() bool { return s.BasisPoints > 0 }

// Percent is the share as a percentage.
func (s Share) Percent() float64 { return Percent(s.BasisPoints) }

// ForCampaigns is the share the open period holds out of campaigns; the
// zero Share when there is none or it holds out automations only.
func ForCampaigns(ctx context.Context, q *store.Queries, workspaceID uuid.UUID) (Share, error) {
	p, err := Open(ctx, q, workspaceID)
	if err != nil || p == nil || !p.Campaigns {
		return Share{}, err
	}
	return Share{Seed: p.Seed, BasisPoints: p.BasisPoints}, nil
}

// HeldFromCampaigns lists which of the contacts the open period holds out
// of campaigns (text, push and WhatsApp campaigns skip them). Nil when
// nobody is.
func HeldFromCampaigns(ctx context.Context, q *store.Queries, workspaceID uuid.UUID, contacts []uuid.UUID) (map[uuid.UUID]bool, error) {
	sh, err := ForCampaigns(ctx, q, workspaceID)
	if err != nil || !sh.On() || len(contacts) == 0 {
		return nil, err
	}
	return sh.Held(ctx, q, contacts)
}

// Held lists which of the contacts the share holds out.
func (s Share) Held(ctx context.Context, q *store.Queries, contacts []uuid.UUID) (map[uuid.UUID]bool, error) {
	if !s.On() || len(contacts) == 0 {
		return nil, nil
	}
	ids, err := q.HoldoutHeldContacts(ctx, store.HoldoutHeldContactsParams{ContactIds: contacts, Seed: s.Seed, BasisPoints: s.BasisPoints})
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

// FromAutomations reports whether the open period holds the contact out of
// automation messages.
func FromAutomations(ctx context.Context, q *store.Queries, workspaceID, contactID uuid.UUID) (bool, error) {
	return q.HoldoutContactHeldOut(ctx, store.HoldoutContactHeldOutParams{WorkspaceID: workspaceID, Automations: true, ContactID: contactID})
}

// Ended reports whether a period has ended.
func Ended(p store.HoldoutPeriod) bool { return p.EndedAt != nil }

// End is when a period ends for its measurement: its end, else now.
func End(p store.HoldoutPeriod, now time.Time) time.Time {
	if p.EndedAt != nil {
		return *p.EndedAt
	}
	return now
}
