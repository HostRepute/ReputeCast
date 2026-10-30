package readiness

import (
	"context"
	"slices"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/store"
)

// A sender types the recipient count only for an unusual send (SR-13): an
// audience more than the workspace's multiple of the usual one, a first
// send to a list, from a sending domain or through a provider, or a
// reputation warning above low impact. Otherwise the warnings are
// confirmed with checkboxes.

// ConfirmFacts are what makes a send unusual.
type ConfirmFacts struct {
	// Usual is the median audience of the last sends to the campaign's
	// lists (Sends of them, at most 5); 0 without any. Multiple is the
	// workspace's threshold (Settings › Sending rules).
	Usual    int64
	Sends    int
	Multiple float64
	// FirstList names a list never sent to, FirstDomain the sending
	// domain when nothing went out from it yet, FirstProviders the route's
	// providers that never carried a send.
	FirstList      string
	FirstDomain    string
	FirstProviders []string
}

// ReadinessConfirm says whether the sender types the recipient count, and
// why ("31,192 is 2.1× your usual 15,000 for this list.").
type ReadinessConfirm struct {
	Typed   bool     `json:"typed"`
	Reasons []string `json:"reasons"`
	// Usual is the usual audience of the campaign's lists (0 without one).
	Usual int64 `json:"usual"`
}

// confirmFacts reads what makes the send unusual.
func (s *Service) confirmFacts(ctx context.Context, q *store.Queries, wsID, campaignID uuid.UUID, def AudienceDef, names audienceNames, domain string, providers []RouteShareRef) ConfirmFacts {
	out := ConfirmFacts{Multiple: sendDefaults(ctx, q, wsID).TypedMultiple}
	if len(def.Lists) > 0 {
		if rows, err := q.ReadinessUsualAudience(ctx, store.ReadinessUsualAudienceParams{WorkspaceID: wsID, CampaignID: campaignID, Lists: def.Lists}); err == nil && len(rows) > 0 {
			out.Usual, out.Sends = median(rows), len(rows)
		}
		if sent, err := q.ReadinessListsSentTo(ctx, store.ReadinessListsSentToParams{WorkspaceID: wsID, CampaignID: campaignID, Lists: def.Lists}); err == nil {
			for _, l := range def.Lists {
				if !slices.Contains(sent, l) {
					out.FirstList = names.lists[l]
					break
				}
			}
		}
	}
	if domain != "" {
		if sent, err := q.ReadinessSentFromDomain(ctx, store.ReadinessSentFromDomainParams{WorkspaceID: wsID, CampaignID: campaignID, Domain: domain}); err == nil && !sent {
			out.FirstDomain = domain
		}
	}
	if len(providers) > 0 {
		ids := make([]uuid.UUID, 0, len(providers))
		for _, p := range providers {
			ids = append(ids, p.ID)
		}
		if sent, err := q.ReadinessProvidersSentThrough(ctx, store.ReadinessProvidersSentThroughParams{WorkspaceID: wsID, CampaignID: campaignID, Providers: ids}); err == nil {
			for _, p := range providers {
				if !slices.Contains(sent, p.ID) {
					out.FirstProviders = append(out.FirstProviders, p.Short)
				}
			}
		}
	}
	return out
}

// RouteShareRef names a provider of a send's route.
type RouteShareRef struct {
	ID    uuid.UUID
	Short string
}

func median(v []int64) int64 {
	s := slices.Clone(v)
	slices.Sort(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

// confirm says whether the sender types the count, and why.
func (b *builder) confirm() (ReadinessConfirm, []Text) {
	f, a := b.f.Confirm, b.f.Audience
	out := ReadinessConfirm{Reasons: []string{}, Usual: f.Usual}
	reasons := []Text{}
	if f.Usual > 0 && f.Multiple > 0 && float64(a.Final) > f.Multiple*float64(f.Usual) {
		reasons = append(reasons, tx("confirm.larger", A{"count": a.Final, "times": argDecimal1(float64(a.Final) / float64(f.Usual)), "usual": f.Usual}))
	}
	if f.FirstList != "" {
		reasons = append(reasons, tx("confirm.firstList", A{"list": f.FirstList}))
	}
	if f.FirstDomain != "" {
		reasons = append(reasons, tx("confirm.firstDomain", A{"domain": f.FirstDomain}))
	}
	if len(f.FirstProviders) > 0 {
		reasons = append(reasons, tx("confirm.firstProvider", A{"providers": ArgAnd(f.FirstProviders)}))
	}
	for _, l := range b.f.Reputation.Listings {
		if l.Impact == "medium" || l.Impact == "high" {
			reasons = append(reasons, tx("confirm.reputation", A{"asset": l.Asset, "source": l.Source, "impact": l.Impact}))
			break
		}
	}
	out.Typed = len(reasons) > 0
	for _, r := range reasons {
		out.Reasons = append(out.Reasons, r.en())
	}
	return out, reasons
}
