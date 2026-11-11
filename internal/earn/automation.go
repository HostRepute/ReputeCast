package earn

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/payments"
	"github.com/hostrepute/reputecast/internal/store"
)

// Automation triggers for paid members (M57, G-11): Earn queues their
// events, internal/automations matches them to flows (the same kind
// strings; neither package imports the other). Only Stripe subscriptions
// in live mode start flows: complimentary access, gifts and test mode
// never do.
//
// Each event's ref is the subscription ("sub:<id>"), its payload names the
// tier (tier_id, tier) and carries "liquid", the flow's email data:
// {{ subscription.tier }}, {{ subscription.previous_tier }} and
// {{ subscription.trial_ends_at }}.
const (
	// AutomationPaidStarted: a subscription is active or past due for
	// the first time ("Becomes a paid member"); from_trial when a trial
	// came first.
	AutomationPaidStarted = "paid_started"
	// AutomationPaidTierChanged: the subscription moved to another tier
	// ("Changes tier"), with the direction from the tiers' access ranks.
	// A monthly↔yearly switch inside one tier is no change.
	AutomationPaidTierChanged = "paid_tier_changed"
	// AutomationPaidEnded: paid access ended ("Stops paying"), with the
	// reason.
	AutomationPaidEnded = "paid_ended"
	// AutomationPaidTrialEnding: Stripe's notice three days before a
	// trial ends ("Trial ends soon").
	AutomationPaidTrialEnding = "paid_trial_ending"
	// AutomationPaidPaused and AutomationPaidResumed: the subscription's
	// payments paused, and started again (M65, "Pauses payments",
	// "Resumes payments"), with the resume date.
	AutomationPaidPaused  = "paid_paused"
	AutomationPaidResumed = "paid_resumed"
)

// automationEndReasons are the ends "Stops paying" starts on. Other ends
// (a disconnected account) mark the subscription and start nothing.
var automationEndReasons = []string{"canceled", "payment_failed", "trial_ended"}

// paidTier is a tier as the events name it.
type paidTier struct {
	id    string
	name  string
	rank  int32
	found bool
}

func paidTierOf(ctx context.Context, q *store.Queries, wsID uuid.UUID, id *uuid.UUID) (paidTier, error) {
	if id == nil {
		return paidTier{}, nil
	}
	t, err := q.EarnTier(ctx, store.EarnTierParams{ID: *id, WorkspaceID: wsID})
	if errors.Is(err, pgx.ErrNoRows) {
		return paidTier{id: id.String()}, nil
	}
	if err != nil {
		return paidTier{}, err
	}
	return paidTier{id: t.ID.String(), name: t.Name, rank: t.AccessRank, found: true}, nil
}

// marked reads a mark query's answer: no row means it was set before.
func marked(_ uuid.UUID, err error) (bool, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// paidTransitions queues a subscription's automation events after a write
// to it, comparing the row before the write (nil when the write created
// it) with the row now, whatever the write was. The marks on the row make
// "Becomes a paid member" and "Stops paying" happen once per subscription,
// and everything runs in the write's transaction: a Stripe redelivery or
// replay queues nothing twice, and a rollback queues nothing.
func (s *Service) paidTransitions(ctx context.Context, q *store.Queries, ws *store.Workspace, mode string, before *store.PaidSubscription, subID uuid.UUID, at time.Time) error {
	if mode != string(payments.Live) {
		return nil
	}
	row, err := q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: subID, WorkspaceID: ws.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if row.Mode != string(payments.Live) || row.Kind != "stripe" {
		return nil
	}
	tier, err := paidTierOf(ctx, q, ws.ID, row.TierID)
	if err != nil {
		return err
	}
	started := false
	if row.Status == StatusActive || row.Status == StatusPastDue {
		if started, err = marked(q.EarnSubscriptionMarkStarted(ctx, store.EarnSubscriptionMarkStartedParams{ID: row.ID, At: at})); err != nil {
			return err
		}
		if started {
			fromTrial := row.TrialEnd != nil || (before != nil && before.Status == StatusTrialing)
			if err := s.paidEvent(ctx, q, ws, row, AutomationPaidStarted, at, tier, "", map[string]any{"from_trial": fromTrial}); err != nil {
				return err
			}
		}
	}
	if before != nil && before.TierID != nil && row.TierID != nil && *before.TierID != *row.TierID {
		from, err := paidTierOf(ctx, q, ws.ID, before.TierID)
		if err != nil {
			return err
		}
		direction := "lateral"
		if from.found && tier.found {
			switch {
			case tier.rank > from.rank:
				direction = "upgrade"
			case tier.rank < from.rank:
				direction = "downgrade"
			}
		}
		if err := s.paidEvent(ctx, q, ws, row, AutomationPaidTierChanged, at, tier, from.name,
			map[string]any{"direction": direction, "from_tier_id": from.id, "from_tier": from.name}); err != nil {
			return err
		}
	}
	if before != nil && row.EndedAt == nil {
		switch {
		case before.PausedAt == nil && row.PausedAt != nil:
			extra := map[string]any{}
			if row.PauseResumesAt != nil {
				extra["resumes_at"] = row.PauseResumesAt.UTC().Format(time.RFC3339)
			}
			if err := s.paidEvent(ctx, q, ws, row, AutomationPaidPaused, at, tier, "", extra); err != nil {
				return err
			}
		case before.PausedAt != nil && row.PausedAt == nil:
			if err := s.paidEvent(ctx, q, ws, row, AutomationPaidResumed, at, tier, "", nil); err != nil {
				return err
			}
		}
	}
	if row.EndedAt != nil {
		ended, err := marked(q.EarnSubscriptionMarkEnded(ctx, store.EarnSubscriptionMarkEndedParams{ID: row.ID, At: at}))
		if err != nil {
			return err
		}
		// Only someone who had access stops paying: a checkout that
		// never completed (incomplete, then expired) ends silently.
		hadAccess := started || row.AutomationStartedAt != nil || row.TrialEnd != nil || (before != nil && before.Status == StatusTrialing)
		if ended && hadAccess && slices.Contains(automationEndReasons, row.EndReason) {
			if err := s.paidEvent(ctx, q, ws, row, AutomationPaidEnded, at, tier, "", map[string]any{"reason": row.EndReason}); err != nil {
				return err
			}
		}
	}
	return nil
}

// paidTrialEnding queues "Trial ends soon" for a live Stripe trial that
// hasn't ended. Each Stripe notice is applied once (Apply's status guard).
func (s *Service) paidTrialEnding(ctx context.Context, q *store.Queries, ws *store.Workspace, mode string, row store.PaidSubscription, trialEnd *time.Time, at time.Time) error {
	if mode != string(payments.Live) || row.Mode != string(payments.Live) || row.Kind != "stripe" || row.EndedAt != nil {
		return nil
	}
	tier, err := paidTierOf(ctx, q, ws.ID, row.TierID)
	if err != nil {
		return err
	}
	extra := map[string]any{}
	if trialEnd != nil {
		row.TrialEnd = trialEnd
		extra["trial_end"] = trialEnd.UTC().Format(time.RFC3339)
	}
	return s.paidEvent(ctx, q, ws, row, AutomationPaidTrialEnding, at, tier, "", extra)
}

// paidEvent queues one automation event of a subscription.
func (s *Service) paidEvent(ctx context.Context, q *store.Queries, ws *store.Workspace, row store.PaidSubscription, kind string, at time.Time, tier paidTier, previous string, extra map[string]any) error {
	vars := map[string]any{"tier": tier.name, "previous_tier": previous, "trial_ends_at": ""}
	if row.TrialEnd != nil {
		vars["trial_ends_at"] = longDate(ws, *row.TrialEnd)
	}
	payload := map[string]any{"tier_id": tier.id, "tier": tier.name, "liquid": map[string]any{"subscription": vars}}
	maps.Copy(payload, extra)
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return q.InsertAutomationEvent(ctx, store.InsertAutomationEventParams{WorkspaceID: ws.ID, Kind: kind, ContactID: row.ContactID,
		Ref: "sub:" + row.ID.String(), Payload: b, OccurredAt: at})
}

// markPaidHistory marks a subscription brought in by an import as history,
// as the M57 migration marked the subscriptions it found: someone who paid,
// or whose access ended, before ReputeCast knew them never starts "Becomes
// a paid member" or "Stops paying". A trial still to convert stays open.
func markPaidHistory(ctx context.Context, q *store.Queries, row store.PaidSubscription) error {
	if row.Kind != "stripe" {
		return nil
	}
	if row.Status == StatusActive || row.Status == StatusPastDue || row.Status == StatusCanceled {
		if _, err := marked(q.EarnSubscriptionMarkStarted(ctx, store.EarnSubscriptionMarkStartedParams{ID: row.ID, At: row.StartedAt})); err != nil {
			return err
		}
	}
	if row.EndedAt != nil {
		if _, err := marked(q.EarnSubscriptionMarkEnded(ctx, store.EarnSubscriptionMarkEndedParams{ID: row.ID, At: *row.EndedAt})); err != nil {
			return err
		}
	}
	return nil
}
