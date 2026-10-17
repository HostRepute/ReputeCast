package earn

import (
	"context"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// PaidMemberSummary is what agents read about paid members: counts by
// status and by tier, and the MRR, live only and without names.
type PaidMemberSummary struct {
	Counts   EarnMemberCounts `json:"counts"`
	MRRCents int64            `json:"mrr_cents"`
	Currency string           `json:"currency"`
	Tiers    []PaidTierCount  `json:"tiers"`
	// Mode is the mode the upgrade page sells in ("" without a connection).
	Mode string `json:"mode"`
}

// PaidTierCount is one tier's paid members.
type PaidTierCount struct {
	Tier          string `json:"tier"`
	Paying        int64  `json:"paying"`
	Trialing      int64  `json:"trialing"`
	Complimentary int64  `json:"complimentary"`
	Gift          int64  `json:"gift"`
}

// PaidMemberCounts summarizes the paid members (the MCP tool
// get_paid_member_counts).
func (s *Service) PaidMemberCounts(ctx context.Context, ws *store.Workspace) (PaidMemberSummary, error) {
	out := PaidMemberSummary{Tiers: []PaidTierCount{}}
	counts, mrr, err := s.countMembers(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Counts, out.MRRCents = counts, mrr
	st, err := s.settingsRow(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	out.Currency = st.Currency
	if conns, err := s.connections(ctx, ws.ID); err == nil {
		if c, ok := active(conns); ok {
			out.Mode = c.Mode
		}
	}
	rows, err := s.Pool.Query(ctx, `SELECT s.tier_id,
		count(*) FILTER (WHERE s.kind = 'stripe' AND s.status IN ('active','past_due')),
		count(*) FILTER (WHERE s.status = 'trialing'),
		count(*) FILTER (WHERE s.status = 'complimentary'),
		count(*) FILTER (WHERE s.status = 'gift')
		FROM paid_subscriptions s WHERE s.workspace_id = $1 AND s.mode = 'live' AND s.ended_at IS NULL AND s.status <> 'incomplete' AND s.tier_id IS NOT NULL
		GROUP BY s.tier_id`, ws.ID)
	if err != nil {
		return out, err
	}
	byTier := map[uuid.UUID]PaidTierCount{}
	for rows.Next() {
		var id uuid.UUID
		var c PaidTierCount
		if err := rows.Scan(&id, &c.Paying, &c.Trialing, &c.Complimentary, &c.Gift); err != nil {
			rows.Close()
			return out, err
		}
		byTier[id] = c
	}
	rows.Close()
	tiers, err := s.Q.EarnTiers(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, t := range tiers {
		c := byTier[t.ID]
		c.Tier = t.Name
		out.Tiers = append(out.Tiers, c)
	}
	return out, nil
}
