package referrals

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

// Kinds of the referral log (referral_events).
const (
	EventSignedUp         = "signed_up"         // joined through a referral link (on the referred contact)
	EventConfirmed        = "confirmed"         // confirmed: the hold starts
	EventHeld             = "held"              // held for review
	EventCounted          = "counted"           // counted (on the referrer)
	EventNotCounted       = "not_counted"       // excluded, with its reason
	EventDecided          = "decided"           // a person decided on held referrals
	EventMilestoneReached = "milestone_reached" // a referrer reached a milestone
	EventRewardSent       = "reward_sent"       // a reward became available
	EventRewardFailed     = "reward_failed"     // a webhook reward gave up
	EventRewardFulfilled  = "reward_fulfilled"  // a manual reward was sent by hand
	EventLinkPaused       = "link_paused"
	EventLinkResumed      = "link_resumed"
)

// Webhook events of the Referrals group.
const (
	HookCounted          = "referral.counted"
	HookHeld             = "referral.held"
	HookNotCounted       = "referral.not_counted"
	HookMilestoneReached = "referral.milestone_reached"
	HookRewardSent       = "referral.reward_sent"
	HookRewardFailed     = "referral.reward_failed"
)

// TriggerMilestoneReached is the automation trigger kind "Reached a
// milestone": Ref is the milestone, the payload names the program, the
// milestone's count and the reward.
const TriggerMilestoneReached = "milestone_reached"

type logEntry struct {
	program  uuid.UUID
	referral *uuid.UUID
	link     *uuid.UUID
	contact  *uuid.UUID
	kind     string
	detail   map[string]any
	at       time.Time
}

func writeLog(ctx context.Context, q *store.Queries, wsID uuid.UUID, e logEntry) error {
	detail := e.detail
	if detail == nil {
		detail = map[string]any{}
	}
	raw, _ := json.Marshal(detail)
	return q.ReferralEventInsert(ctx, store.ReferralEventInsertParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, ProgramID: e.program,
		ReferralID: e.referral, LinkID: e.link, ContactID: e.contact, Kind: e.kind, Detail: raw, OccurredAt: e.at})
}

// hookContact is a contact as webhook payloads name it.
type hookContact struct {
	ContactID string `json:"contactId"`
	Email     string `json:"email,omitempty"`
	FirstName string `json:"firstName,omitempty"`
}

// hookReferral is a referral as webhook payloads name it: never the
// referred person's address.
type hookReferral struct {
	ID          string     `json:"id"`
	Number      int        `json:"number"`
	Status      string     `json:"status"`
	Reason      string     `json:"reason,omitempty"`
	Signals     []string   `json:"signals,omitempty"`
	SignedUpAt  time.Time  `json:"signedUpAt"`
	ConfirmedAt *time.Time `json:"confirmedAt,omitempty"`
	CountsOn    *time.Time `json:"countsOn,omitempty"`
	ContactID   string     `json:"contactId,omitempty"`
}

func emitHook(ctx context.Context, tx pgx.Tx, ws *store.Workspace, event string, data any) error {
	_, err := webhooks.Emit(ctx, tx, ws.ID, ws.Slug, event, data)
	return err
}

func referralHook(r store.Referral) hookReferral {
	h := hookReferral{ID: r.ID.String(), Number: int(r.Seq), Status: r.Status, Reason: r.Reason, SignedUpAt: r.SignedUpAt,
		ConfirmedAt: r.ConfirmedAt, CountsOn: r.HoldUntil}
	if len(r.Signals) > 0 {
		h.Signals = r.Signals
	}
	if r.ContactID != nil {
		h.ContactID = r.ContactID.String()
	}
	return h
}

// emitMilestoneTrigger starts "Reached a milestone" automations for a
// referrer: once per contact per milestone.
func emitMilestoneTrigger(ctx context.Context, q *store.Queries, wsID, contactID uuid.UUID, programID uuid.UUID, m store.ReferralMilestone, at time.Time) error {
	payload, _ := json.Marshal(map[string]any{"program": programID.String(), "milestone": m.ID.String(), "count": m.Count, "reward": m.RewardName})
	return q.InsertAutomationEvent(ctx, store.InsertAutomationEventParams{WorkspaceID: wsID, Kind: TriggerMilestoneReached, ContactID: contactID,
		Ref: m.ID.String(), Payload: payload, OccurredAt: at})
}
