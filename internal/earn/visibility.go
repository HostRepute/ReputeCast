package earn

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/paidaccess"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/visibility"
)

// Paid access in the Visibility setting (brief §3.7): a block shown to any
// paid tier, to a tier and above, or to free subscribers only, in a group
// of its own in the condition picker. Paid access includes trialing,
// complimentary and gift access, and past due while Stripe retries; a
// higher tier includes a lower tier's paid content. It is read in bulk once
// per count, send or preview (paidaccess), never per contact.

// The paid access condition: its kind, group and operators.
const (
	KindPaid  = "paid"
	GroupPaid = "paid"
	// OpPaidAny shows the block to any paid tier.
	OpPaidAny = "any"
	// OpPaidTier shows it to the tier in the condition's value and above.
	OpPaidTier = "at_least"
	// OpPaidFree shows it to free subscribers only (no paid access).
	OpPaidFree = "free"
	// FieldUpgradeBlock marks the Upgrade block's own condition: free
	// subscribers only, and only while a live payment provider is connected
	// (the block is hidden until then).
	FieldUpgradeBlock = "upgrade"
)

func init() {
	visibility.RegisterKind(visibility.Kind{Key: KindPaid, Group: GroupPaid,
		Ops:  []visibility.Operator{{Key: OpPaidAny, NoValue: true}, {Key: OpPaidTier}, {Key: OpPaidFree, NoValue: true}},
		Load: loadPaid, Match: matchPaid, Check: checkPaid, Describe: describePaid, Why: whyPaid, Options: paidOptions, Trait: paidTrait})
}

// paidFact is a contact's paid access as conditions read it
// (Facts.Extra[KindPaid]); ranks are the access ranks of the tiers the
// conditions name, shared by every contact of a load.
type paidFact struct {
	Has    bool
	Rank   int
	Tier   string
	Short  string
	Status string
	ranks  map[string]int
	// live: a live payment provider is connected (the Upgrade block shows).
	live bool
}

func paidOf(f *visibility.Facts) paidFact {
	p, _ := f.Extra[KindPaid].(paidFact)
	return p
}

// loadPaid reads the contacts' paid access and the ranks of the tiers the
// conditions name.
func loadPaid(ctx context.Context, q *store.Queries, wsID uuid.UUID, conds []visibility.VisibilityCondition, facts map[uuid.UUID]*visibility.Facts) error {
	ids := make([]uuid.UUID, 0, len(facts))
	for id, f := range facts {
		if _, done := f.Extra[KindPaid]; done {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	ranks := map[string]int{}
	for _, c := range conds {
		if c.Op != OpPaidTier {
			continue
		}
		if _, ok := ranks[c.Value]; ok {
			continue
		}
		r, err := paidaccess.MinRank(ctx, q, wsID, c.Value)
		if err != nil {
			return err
		}
		ranks[c.Value] = r
	}
	access, err := paidaccess.Facts(ctx, q, wsID, ids)
	if err != nil {
		return err
	}
	live := false
	if conns, err := q.PayConnections(ctx, wsID); err == nil {
		for _, c := range conns {
			live = live || (c.PaymentConnection.Mode == "live" && c.PaymentConnection.Status == "connected")
		}
	} else {
		return err
	}
	tiers, err := paidaccess.Tiers(ctx, q, wsID)
	if err != nil {
		return err
	}
	short := map[uuid.UUID]string{}
	for _, t := range tiers {
		short[t.ID] = t.Short
	}
	for _, id := range ids {
		f := facts[id]
		if f.Extra == nil {
			f.Extra = map[string]any{}
		}
		p := paidFact{ranks: ranks, live: live}
		if a, ok := access[id]; ok {
			p = paidFact{Has: true, Rank: a.Rank, Tier: a.Tier, Short: short[a.TierID], Status: a.Status, ranks: ranks, live: live}
			if p.Short == "" {
				p.Short = a.Tier
			}
		}
		f.Extra[KindPaid] = p
	}
	return nil
}

func matchPaid(c visibility.VisibilityCondition, f *visibility.Facts) bool {
	p := paidOf(f)
	switch c.Op {
	case OpPaidFree:
		if c.Field == FieldUpgradeBlock && !p.live {
			return false
		}
		return !p.Has
	case OpPaidTier:
		need, ok := p.ranks[c.Value]
		return p.Has && ok && p.Rank >= need
	}
	return p.Has
}

func checkPaid(c visibility.VisibilityCondition) error {
	if c.Op != OpPaidTier {
		return nil
	}
	if _, err := uuid.Parse(c.Value); err != nil {
		return errors.New("pick a tier")
	}
	return nil
}

var (
	mDescPaidAny     = msgs.M("visibility.describeAny")
	mDescPaidTier    = msgs.M("visibility.describeTier")
	mDescPaidFree    = msgs.M("visibility.describeFree")
	mWhyNoAccess     = msgs.M("visibility.whyNoAccess")
	mWhyBelow        = msgs.M("visibility.whyBelow")
	mWhyPaid         = msgs.M("visibility.whyPaid")
	mWhyNotConnected = msgs.M("visibility.whyNotConnected")
	mTraitFree       = msgs.M("visibility.traitFree")
	mTraitPaid       = msgs.M("visibility.traitPaid")
	mPaidATier       = msgs.M("visibility.aTier")
)

// tierLabel is the tier choice a condition names, as it was picked
// ("Insider and above", or the top tier alone: "Founding member").
func tierLabel(c visibility.VisibilityCondition, l i18n.Lang) string {
	if c.Label != "" {
		return c.Label
	}
	return mPaidATier.Text(l, nil)
}

// tierNeeded is the tier a condition needs, without "and above".
func tierNeeded(c visibility.VisibilityCondition, l i18n.Lang) string {
	return strings.TrimSuffix(tierLabel(c, l), " and above")
}

func describePaid(c visibility.VisibilityCondition, n *visibility.Names) string {
	switch c.Op {
	case OpPaidFree:
		return mDescPaidFree.Text(n.Lang, nil)
	case OpPaidTier:
		return mDescPaidTier.Text(n.Lang, i18n.Args{"tier": tierLabel(c, n.Lang)})
	}
	return mDescPaidAny.Text(n.Lang, nil)
}

func whyPaid(c visibility.VisibilityCondition, f *visibility.Facts, matched bool, n *visibility.Names) string {
	p := paidOf(f)
	switch {
	case c.Op == OpPaidFree && c.Field == FieldUpgradeBlock && !p.live:
		return mWhyNotConnected.Text(n.Lang, nil)
	case !p.Has:
		return mWhyNoAccess.Text(n.Lang, i18n.Args{"name": f.Name()})
	case c.Op == OpPaidTier && !matched:
		return mWhyBelow.Text(n.Lang, i18n.Args{"name": f.Name(), "tier": p.Short, "needed": tierNeeded(c, n.Lang)})
	}
	return mWhyPaid.Text(n.Lang, i18n.Args{"name": f.Name(), "tier": p.Short})
}

// paidTrait is what the preview says about a contact's paid access:
// "Free subscriber", or "Insider member · trialing".
func paidTrait(_ visibility.VisibilityCondition, f *visibility.Facts, n *visibility.Names) string {
	p := paidOf(f)
	if !p.Has {
		return mTraitFree.Text(n.Lang, nil)
	}
	return mTraitPaid.Text(n.Lang, i18n.Args{"tier": p.Short, "status": p.Status})
}

// paidOptions are the tiers a condition can name, lowest first, with how
// many contacts have access at each and above.
func paidOptions(ctx context.Context, q *store.Queries, wsID uuid.UUID, _ *visibility.Names) ([]visibility.VisibilityOption, error) {
	tiers, err := paidaccess.Tiers(ctx, q, wsID)
	if err != nil || len(tiers) == 0 {
		return nil, err
	}
	rows, err := q.PaidAccessRows(ctx, store.PaidAccessRowsParams{WorkspaceID: wsID, MinRank: 1})
	if err != nil {
		return nil, err
	}
	out := make([]visibility.VisibilityOption, 0, len(tiers))
	for _, t := range tiers {
		var n int64
		for _, r := range rows {
			if int(r.TierRank) >= t.Rank {
				n++
			}
		}
		out = append(out, visibility.VisibilityOption{Value: t.ID.String(), Label: t.Name, Contacts: &n})
	}
	return out, nil
}
