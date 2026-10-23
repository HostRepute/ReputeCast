package audience

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/paidaccess"
	"github.com/hostrepute/reputecast/internal/store"
)

// Paid status in the contacts table (Earn, design expansion X16, brief §6):
// the Paid status filter (any paid tier with access or a tier and above,
// trialing, past due, complimentary, gift, ended, or free subscribers only),
// the Paid column and the tier chip next to a paid member's name. Paid
// status needs earn:view on top of subscribers:view; test paid members
// never count.

// PaidFilter narrows contacts by paid status. Saved views keep it.
type PaidFilter struct {
	// Paid is a paid status: with_access (any paid tier, or the tier in
	// PaidTier and above), trialing, past_due, complimentary, gift, ended
	// (access ended, none now) or free (no paid access).
	Paid     string `json:"paid,omitempty" enum:"with_access,trialing,past_due,complimentary,gift,ended,free"`
	PaidTier string `json:"paidTier,omitempty" doc:"With paid=with_access: a tier's id, for that tier and above"`
}

// PaidQuery is the same filter as query parameters.
type PaidQuery struct {
	Paid     string `query:"paid" enum:"with_access,trialing,past_due,complimentary,gift,ended,free,"`
	PaidTier string `query:"paidTier" doc:"With paid=with_access: a tier's id, for that tier and above"`
}

func (q PaidQuery) filter() PaidFilter { return PaidFilter(q) }

// accessCond is "s has paid access now" over paid_subscriptions s of
// contact c.
const accessCond = `s.contact_id = c.id AND s.mode = 'live' AND s.ended_at IS NULL
	AND (s.status IN ('trialing', 'active', 'complimentary', 'gift') OR (s.status = 'past_due' AND NOT s.access_paused))`

// canSeePaid reports whether the caller may read paid status.
func canSeePaid(ctx context.Context) bool {
	p := api.PrincipalFrom(ctx)
	return p == nil || p.Can(authz.EarnView)
}

// paidFilter appends the paid status conditions of f.
func (s *Service) paidFilter(ctx context.Context, wsID uuid.UUID, out *filterSQL, f PaidFilter) error {
	if f.Paid == "" {
		return nil
	}
	if !canSeePaid(ctx) {
		return api.Err(403, "forbidden", say(ctx, mPaidNeedsEarn, nil))
	}
	access := func(cond string, vals ...any) {
		out.add("EXISTS (SELECT 1 FROM paid_subscriptions s LEFT JOIN earn_tiers t ON t.id = s.tier_id WHERE "+accessCond+cond+")", vals...)
	}
	switch f.Paid {
	case "with_access":
		rank, err := paidaccess.MinRank(ctx, s.Q, wsID, f.PaidTier)
		if err != nil {
			return err
		}
		access(" AND coalesce(t.access_rank, 1) >= ?", rank)
	case "trialing", "past_due":
		access(" AND s.status = ?", f.Paid)
	case "complimentary", "gift":
		access(" AND s.kind = ?", f.Paid)
	case "free":
		out.add("NOT EXISTS (SELECT 1 FROM paid_subscriptions s WHERE " + accessCond + ")")
	case "ended":
		out.add("NOT EXISTS (SELECT 1 FROM paid_subscriptions s WHERE " + accessCond + ")" +
			" AND EXISTS (SELECT 1 FROM paid_subscriptions s WHERE s.contact_id = c.id AND s.mode = 'live' AND s.ended_at IS NOT NULL)")
	default:
		return api.Invalid("paid", say(ctx, mPickPaid, nil))
	}
	return nil
}

// ContactPaid is a contact's paid status in the contacts table: the tier's
// short name for the chip, the status for the Paid column, and the paid
// subscription that opens the paid member.
type ContactPaid struct {
	Tier     string `json:"tier" doc:"The tier's short name (Insider)"`
	Status   string `json:"status" enum:"trialing,active,past_due,complimentary,gift,ended"`
	MemberID string `json:"memberId"`
	// EndedAt is when access ended (status ended).
	EndedAt *time.Time `json:"endedAt,omitempty"`
}

// contactPaid reads the paid status of a page of contacts: the highest
// tier with access, else the latest ended subscription.
func (s *Service) contactPaid(ctx context.Context, wsID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*ContactPaid, error) {
	out := map[uuid.UUID]*ContactPaid{}
	if len(ids) == 0 || !canSeePaid(ctx) {
		return out, nil
	}
	tiers, err := paidaccess.Tiers(ctx, s.Q, wsID)
	if err != nil {
		return nil, err
	}
	short := map[uuid.UUID]string{}
	for _, t := range tiers {
		short[t.ID] = t.Short
	}
	rows, err := s.Pool.Query(ctx, `SELECT DISTINCT ON (s.contact_id) s.contact_id, s.id, s.status, s.tier_id, coalesce(t.name, ''), s.ended_at,
			(s.ended_at IS NULL AND (s.status IN ('trialing', 'active', 'complimentary', 'gift') OR (s.status = 'past_due' AND NOT s.access_paused))) AS access
		FROM paid_subscriptions s LEFT JOIN earn_tiers t ON t.id = s.tier_id
		WHERE s.workspace_id = $1 AND s.contact_id = ANY ($2) AND s.mode = 'live' AND s.status <> 'incomplete'
		ORDER BY s.contact_id, access DESC, coalesce(t.access_rank, 1) DESC, coalesce(s.ended_at, s.started_at) DESC`, wsID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var contact, member uuid.UUID
		var tier *uuid.UUID
		var status, tierName string
		var ended *time.Time
		var access bool
		if err := rows.Scan(&contact, &member, &status, &tier, &tierName, &ended, &access); err != nil {
			return nil, err
		}
		p := &ContactPaid{Tier: tierName, Status: status, MemberID: member.String()}
		if tier != nil && short[*tier] != "" {
			p.Tier = short[*tier]
		}
		if !access {
			if ended == nil {
				continue
			}
			p.Status, p.EndedAt = "ended", ended
		}
		out[contact] = p
	}
	return out, rows.Err()
}

// AudiencePaidTier is a tier the Paid status filter offers, lowest first.
type AudiencePaidTier struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Short string `json:"short"`
	Top   bool   `json:"top"`
}

// paidTiers are the tiers of the Paid status filter; none without earn:view.
func (s *Service) paidTiers(ctx context.Context, q *store.Queries, wsID uuid.UUID) ([]AudiencePaidTier, error) {
	out := []AudiencePaidTier{}
	if !canSeePaid(ctx) {
		return out, nil
	}
	tiers, err := paidaccess.Tiers(ctx, q, wsID)
	if err != nil {
		return nil, err
	}
	for _, t := range tiers {
		out = append(out, AudiencePaidTier{ID: t.ID.String(), Name: t.Name, Short: t.Short, Top: t.Top})
	}
	return out, nil
}
