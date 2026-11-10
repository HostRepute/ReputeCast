package automations

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// Triggers for paid members (M57, G-11). internal/earn queues their events
// for Stripe subscriptions in live mode (earn/automation.go): the
// subscription is the ref ("sub:<id>"), the payload names the tier
// (tier_id), a tier change's direction (upgrade, downgrade or lateral, from
// the tiers' access ranks) and why access ended (reason). Its "liquid"
// object becomes the flow's email data ({{ subscription.tier }},
// {{ subscription.previous_tier }}, {{ subscription.trial_ends_at }}).
// "Becomes a paid member" and "Stops paying" come once per subscription;
// only a flow's first trigger can be one of these.
const (
	TriggerPaidStarted     = "paid_started"
	TriggerPaidTierChanged = "paid_tier_changed"
	TriggerPaidEnded       = "paid_ended"
	TriggerPaidTrialEnding = "paid_trial_ending"
)

// paidEndReasons are the reasons "Stops paying" can be limited to, in the
// order labels list them.
var paidEndReasons = []string{"canceled", "payment_failed", "trial_ended"}

func isPaidTrigger(kind string) bool {
	return kind == TriggerPaidStarted || kind == TriggerPaidTierChanged || kind == TriggerPaidEnded || kind == TriggerPaidTrialEnding
}

// AutomationTierOption is a paid tier a paid-member trigger can name.
type AutomationTierOption struct {
	ID   string `json:"id"`
	Name string `json:"name" example:"Weekly Digest Insider"`
}

var (
	mProblemPickTier      = msg("problem.pickTier")
	mProblemTierDirection = msg("problem.tierDirection")
	mProblemEndReasons    = msg("problem.endReasons")

	mRefAnyTier          = msg("ref.anyTier")
	mRefDeletedTier      = msg("ref.deletedTier")
	mRefTierUpgrade      = msg("ref.tierUpgrade")
	mRefTierDowngrade    = msg("ref.tierDowngrade")
	mRefTierChange       = msg("ref.tierChange")
	mRefAnyTierChange    = msg("ref.anyTierChange")
	mRefAnyUpgrade       = msg("ref.anyUpgrade")
	mRefAnyDowngrade     = msg("ref.anyDowngrade")
	mRefAnyReason        = msg("ref.anyReason")
	mRefReasonCanceled   = msg("ref.reasonCanceled")
	mRefReasonFailed     = msg("ref.reasonPaymentFailed")
	mRefReasonTrialEnded = msg("ref.reasonTrialEnded")
)

// checkPaidTrigger checks a paid-member trigger's settings: a tier that
// exists, a direction only on Changes tier, end reasons only on Stops
// paying.
func checkPaidTrigger(t *AutomationTrigger, refs *Refs) []configProblem {
	var out []configProblem
	if t.TierID != "" && refs != nil && refs.Tiers != nil {
		if _, ok := refs.Tiers[t.TierID]; !ok {
			out = append(out, configProblem{"missing_ref", txt(mProblemPickTier)})
		}
	}
	switch {
	case t.Kind == TriggerPaidTierChanged:
		if t.Direction != "" && t.Direction != "any" && t.Direction != "upgrade" && t.Direction != "downgrade" {
			out = append(out, configProblem{"invalid_direction", txt(mProblemTierDirection)})
		}
	case t.Direction != "":
		out = append(out, configProblem{"invalid_direction", txt(mProblemTierDirection)})
	}
	if len(t.EndReasons) > 0 && (t.Kind != TriggerPaidEnded || slices.ContainsFunc(t.EndReasons, func(r string) bool { return !slices.Contains(paidEndReasons, r) })) {
		out = append(out, configProblem{"invalid_reason", txt(mProblemEndReasons)})
	}
	return out
}

// paidTriggerMatches decides whether a paid-member event starts a flow:
// its tier (any when none is chosen), a tier change's direction and why
// access ended.
func paidTriggerMatches(t AutomationTrigger, ev store.ClaimAutomationEventsRow) bool {
	var p struct {
		TierID    string `json:"tier_id"`
		Direction string `json:"direction"`
		Reason    string `json:"reason"`
	}
	if json.Unmarshal(ev.Payload, &p) != nil {
		return false
	}
	if t.TierID != "" && p.TierID != t.TierID {
		return false
	}
	switch t.Kind {
	case TriggerPaidTierChanged:
		return t.Direction == "" || t.Direction == "any" || t.Direction == p.Direction
	case TriggerPaidEnded:
		return len(t.EndReasons) == 0 || slices.Contains(t.EndReasons, p.Reason)
	}
	return true
}

// tierRef names a trigger's tier, or a deleted one.
func tierRef(id string, refs *Refs) i18n.Text {
	if refs != nil {
		if n, ok := refs.Tiers[id]; ok {
			return name(n)
		}
	}
	return txt(mRefDeletedTier)
}

// paidTriggerRef reads "Any tier", "Weekly Digest Insider", "Upgrades to
// Founding member", "canceled or payment failed".
func paidTriggerRef(t *AutomationTrigger, refs *Refs) i18n.Text {
	switch t.Kind {
	case TriggerPaidTierChanged:
		if t.TierID == "" {
			switch t.Direction {
			case "upgrade":
				return txt(mRefAnyUpgrade)
			case "downgrade":
				return txt(mRefAnyDowngrade)
			}
			return txt(mRefAnyTierChange)
		}
		tier := tierRef(t.TierID, refs)
		switch t.Direction {
		case "upgrade":
			return txt(mRefTierUpgrade, "tier", tier)
		case "downgrade":
			return txt(mRefTierDowngrade, "tier", tier)
		}
		return txt(mRefTierChange, "tier", tier)
	case TriggerPaidEnded:
		reasons := txt(mRefAnyReason)
		if len(t.EndReasons) > 0 {
			words := map[string]i18n.Msg{"canceled": mRefReasonCanceled, "payment_failed": mRefReasonFailed, "trial_ended": mRefReasonTrialEnded}
			var items []i18n.Text
			for _, r := range paidEndReasons {
				if slices.Contains(t.EndReasons, r) {
					items = append(items, txt(words[r]))
				}
			}
			reasons = orList(items)
		}
		if t.TierID == "" {
			return reasons
		}
		return joined(" · ", []i18n.Text{tierRef(t.TierID, refs), reasons})
	}
	if t.TierID == "" {
		return txt(mRefAnyTier)
	}
	return tierRef(t.TierID, refs)
}

// loadTierRefs adds the workspace's paid tiers to refs (archived ones too:
// their members can still change or stop paying).
func loadTierRefs(ctx context.Context, q *store.Queries, wsID uuid.UUID, r *Refs) error {
	rows, err := q.EarnTiers(ctx, wsID)
	if err != nil {
		return err
	}
	r.Tiers = make(map[string]string, len(rows))
	for _, t := range rows {
		r.Tiers[t.ID.String()] = t.Name
	}
	return nil
}

// paidOptions are the tiers the paid-member triggers offer, and whether
// the workspace has a live payment connection (the editor shows the
// "Paid members" triggers only then).
func paidOptions(ctx context.Context, q *store.Queries, wsID uuid.UUID) ([]AutomationTierOption, bool, error) {
	tiers := []AutomationTierOption{}
	rows, err := q.EarnTiers(ctx, wsID)
	if err != nil {
		return tiers, false, err
	}
	for _, t := range rows {
		tiers = append(tiers, AutomationTierOption{ID: t.ID.String(), Name: t.Name})
	}
	conns, err := q.PayConnections(ctx, wsID)
	if err != nil {
		return tiers, false, err
	}
	connected := slices.ContainsFunc(conns, func(c store.PayConnectionsRow) bool {
		return c.PaymentConnection.Mode == "live" && c.PaymentConnection.Status == "connected"
	})
	return tiers, connected, nil
}
