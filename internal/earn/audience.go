package earn

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/paidaccess"
	"github.com/hostrepute/reputecast/internal/store"
)

// Paid-only campaigns (brief §3.7, ComposerEarnBlocks): the composer's
// audience popover asks who on a list gets the campaign: everyone, paid
// members only (any paid tier or a tier and above) or free subscribers
// only. Drafting needs only campaigns:edit, so a Content Writer can make a
// paid-only draft; the counts are totals, never names.

// EarnAudienceTier is a tier the popover offers, lowest first.
type EarnAudienceTier struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Short is how readers name it ("Insider"); Top is the highest tier
	// ("Founding member" rather than "Founding member and above").
	Short string `json:"short"`
	Top   bool   `json:"top"`
}

// EarnAudienceLeftOut says why paid members aren't recipients.
type EarnAudienceLeftOut struct {
	// NotOnList: paid members who never joined the list.
	NotOnList int64 `json:"notOnList"`
	// Unsubscribed from the list's emails (they keep their access).
	Unsubscribed int64 `json:"unsubscribed"`
	Suppressed   int64 `json:"suppressed"`
	// Inactive contacts (bounced, complained, paused by the team).
	Inactive int64 `json:"inactive"`
}

// EarnAudience is the popover's numbers for a list.
type EarnAudience struct {
	// Who on the list: everyone, paid members at the tier and above, free
	// subscribers (no paid access). Paid and free add up to everyone with
	// any paid tier.
	Everyone int64 `json:"everyone"`
	Paid     int64 `json:"paid"`
	Free     int64 `json:"free"`
	// WithAccess counts every paid member at the tier and above (test paid
	// members never count); LeftOut says why some aren't recipients.
	WithAccess int64               `json:"withAccess"`
	LeftOut    EarnAudienceLeftOut `json:"leftOut"`
	// PastDue counts the recipients whose payment failed while Stripe
	// retries (they keep access); Trialing, Complimentary and Gift the
	// others included.
	PastDue       int64              `json:"pastDue"`
	Trialing      int64              `json:"trialing"`
	Complimentary int64              `json:"complimentary"`
	Gift          int64              `json:"gift"`
	Tiers         []EarnAudienceTier `json:"tiers"`
	// Connected is false until a payment provider is connected.
	Connected bool `json:"connected"`
}

type audienceInput struct {
	api.WorkspaceInput
	List string `query:"list" format:"uuid" doc:"The campaign's list"`
	Tier string `query:"tier" doc:"The lowest tier of paid members; empty for any paid tier"`
}

func registerAudience(h huma.API, s *Service, tags []string, base string) {
	api.Get(h, api.Op{ID: "get-earn-audience", Path: base + "/audience", Summary: "Who on a list a paid-only campaign reaches: everyone, paid members only or free subscribers only",
		Description: "Counts only. Drafting a paid-only campaign needs campaigns:edit, like any draft.", Tags: tags, Access: api.Workspace(authz.CampaignsEdit)},
		func(ctx context.Context, in *audienceInput) (*api.Body[EarnAudience], error) {
			out, err := s.Audience(ctx, api.WorkspaceFrom(ctx), in.List, in.Tier)
			return api.OK(out), err
		})
}

// Audience counts who on a list a paid-only campaign reaches.
func (s *Service) Audience(ctx context.Context, ws *store.Workspace, list, tier string) (EarnAudience, error) {
	out := EarnAudience{Tiers: []EarnAudienceTier{}}
	tiers, err := paidaccess.Tiers(ctx, s.Q, ws.ID)
	if err != nil {
		return out, err
	}
	for _, t := range tiers {
		out.Tiers = append(out.Tiers, EarnAudienceTier{ID: t.ID.String(), Name: t.Name, Short: t.Short, Top: t.Top})
	}
	conns, err := s.connections(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	_, out.Connected = connected(conns, "live")
	lid, err := uuid.Parse(list)
	if err != nil {
		return out, api.Invalid("list", "Pick a list of this workspace.")
	}
	n, err := paidaccess.CountLists(ctx, s.Q, ws.ID, []uuid.UUID{lid}, nil, tier)
	if err != nil {
		return out, err
	}
	out.Everyone, out.Paid, out.Free = n.Everyone, n.Paid, n.Free
	rank, err := paidaccess.MinRank(ctx, s.Q, ws.ID, tier)
	if err != nil {
		return out, err
	}
	rows, err := s.Q.PaidAccessAudience(ctx, store.PaidAccessAudienceParams{WorkspaceID: ws.ID, ListID: lid})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		if int(r.TierRank) < rank {
			continue
		}
		out.WithAccess++
		switch {
		case r.Unsubscribed && !r.Subscribed:
			out.LeftOut.Unsubscribed++
		case !r.Subscribed:
			out.LeftOut.NotOnList++
		case r.Suppressed:
			out.LeftOut.Suppressed++
		case !r.Active:
			out.LeftOut.Inactive++
		default:
			switch r.Status {
			case StatusPastDue:
				out.PastDue++
			case StatusTrialing:
				out.Trialing++
			case StatusComplimentary:
				out.Complimentary++
			case StatusGift:
				out.Gift++
			}
		}
	}
	return out, nil
}
