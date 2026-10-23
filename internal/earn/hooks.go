package earn

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// The "Paid members" webhook events (brief 6). Earn emits the subscription
// and complimentary access events; gifts and conversions belong to the
// gifts and attribution work, which emits them under the same group.
func init() {
	webhooks.Events = append(webhooks.Events,
		webhooks.WebhookEventInfo{Name: "subscription.trial_started", Group: "Paid members", Description: "A paid subscription started with a free trial.", DescriptionText: msgs.M("webhook.subscription.trial_started").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "subscription.started", Group: "Paid members", Description: "A paid subscription started and its first payment was made.", DescriptionText: msgs.M("webhook.subscription.started").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "subscription.renewed", Group: "Paid members", Description: "A subscription renewed and its payment went through.", DescriptionText: msgs.M("webhook.subscription.renewed").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "subscription.payment_failed", Group: "Paid members", Description: "A payment failed; the provider retries.", DescriptionText: msgs.M("webhook.subscription.payment_failed").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "subscription.cancel_scheduled", Group: "Paid members", Description: "A subscription will end at the end of its period.", DescriptionText: msgs.M("webhook.subscription.cancel_scheduled").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "subscription.ended", Group: "Paid members", Description: "Paid access ended: canceled, unpaid, or a trial that ended without payment.", DescriptionText: msgs.M("webhook.subscription.ended").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "subscription.changed", Group: "Paid members", Description: "A paid member switched tier or between monthly and yearly.", DescriptionText: msgs.M("webhook.subscription.changed").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "access.complimentary_granted", Group: "Paid members", Description: "A team member gave complimentary access.", DescriptionText: msgs.M("webhook.access.complimentary_granted").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "access.complimentary_ended", Group: "Paid members", Description: "Complimentary access ended or was removed.", DescriptionText: msgs.M("webhook.access.complimentary_ended").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "gift.purchased", Group: "Paid members", Description: "A reader bought a gift subscription.", DescriptionText: msgs.M("webhook.gift.purchased").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "gift.redeemed", Group: "Paid members", Description: "A gift was redeemed by its recipient.", DescriptionText: msgs.M("webhook.gift.redeemed").Keep(nil)},
		webhooks.WebhookEventInfo{Name: "conversion.recorded", Group: "Paid members", Description: "An upgrade or a conversion was credited to an email.", DescriptionText: msgs.M("webhook.conversion.recorded").Keep(nil)},
	)
}

// hookEvents maps a timeline kind to its webhook event.
var hookEvents = map[string]string{
	"trial_started":    "subscription.trial_started",
	"subscribed":       "subscription.started",
	"renewed":          "subscription.renewed",
	"payment_failed":   "subscription.payment_failed",
	"cancel_scheduled": "subscription.cancel_scheduled",
	"plan_changed":     "subscription.changed",
	"granted":          "access.complimentary_granted",
	"grant_removed":    "access.complimentary_ended",
}

// PaidMemberEvent is the data of a "Paid members" webhook event.
type PaidMemberEvent struct {
	ContactID      string         `json:"contact_id"`
	Email          string         `json:"email"`
	SubscriptionID string         `json:"subscription_id,omitempty"`
	Mode           string         `json:"mode"`
	Kind           string         `json:"kind"`
	Status         string         `json:"status,omitempty"`
	Tier           string         `json:"tier,omitempty"`
	Period         string         `json:"period,omitempty"`
	AmountCents    *int64         `json:"amount_cents,omitempty"`
	Currency       string         `json:"currency,omitempty"`
	OccurredAt     time.Time      `json:"occurred_at"`
	Detail         map[string]any `json:"detail,omitempty"`
}

// emitHook queues the webhook event of a timeline entry, in the
// transaction that records it, for every endpoint subscribed to it.
func (s *Service) emitHook(ctx context.Context, q *store.Queries, wsID, contact uuid.UUID, sub *uuid.UUID, mode, kind string, at time.Time,
	amount *int64, currency string, data map[string]any) error {
	event := hookEvents[kind]
	var row *store.PaidSubscription
	if sub != nil {
		if r, err := q.EarnSubscription(ctx, store.EarnSubscriptionParams{ID: *sub, WorkspaceID: wsID}); err == nil {
			row = &r
		}
	}
	if kind == "access_ended" {
		event = "subscription.ended"
		if row != nil && row.Kind == "complimentary" {
			event = "access.complimentary_ended"
		}
	}
	if event == "" {
		return nil
	}
	hooks, err := q.HooksMatching(ctx, store.HooksMatchingParams{WorkspaceID: wsID, Event: event})
	if err != nil || len(hooks) == 0 {
		return err
	}
	ws, err := q.GetWorkspace(ctx, wsID)
	if err != nil {
		return err
	}
	d := PaidMemberEvent{ContactID: contact.String(), Mode: mode, OccurredAt: at.UTC(), AmountCents: amount, Currency: currency, Detail: data}
	if c, err := q.EarnContactBrief(ctx, store.EarnContactBriefParams{ID: contact, WorkspaceID: wsID}); err == nil {
		d.Email = c.Email
	}
	if row != nil {
		d.SubscriptionID, d.Kind, d.Status, d.Period = row.ID.String(), row.Kind, row.Status, row.Period
		if d.AmountCents == nil && row.AmountCents > 0 {
			a := row.AmountCents
			d.AmountCents, d.Currency = &a, row.Currency
		}
		d.Tier = tierName(ctx, q, wsID, row.TierID)
	}
	eventID := newID()
	body, err := json.Marshal(webhooks.Envelope{ID: eventID.String(), Type: event, CreatedAt: s.now().UTC(), Workspace: ws.Slug, Data: d})
	if err != nil {
		return err
	}
	now := s.now()
	for _, h := range hooks {
		state := "pending"
		if !h.Enabled {
			state = "held"
		}
		if err := q.HooksInsertDelivery(ctx, store.HooksInsertDeliveryParams{ID: newID(), WebhookID: h.ID, Event: event, EventID: &eventID, Payload: body,
			State: state, NextAttemptAt: &now, MaxAttempts: webhooks.MaxAttempts}); err != nil {
			return err
		}
	}
	return nil
}
