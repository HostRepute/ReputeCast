package readiness

import (
	"github.com/hostrepute/reputecast/internal/paidaccess"
)

// Paid-only campaigns (Earn, design expansion X16, brief §3.7): Send
// Readiness says who a campaign whose audience keeps only paid members (or
// only free subscribers) goes to: "Weekly Digest · paid members only: 1,231
// recipients. Trials, complimentary and gift access are included, and 14
// paid members whose payment failed while Stripe retries."

// PaidAudienceFacts are a paid-only campaign's final recipients by paid
// status.
type PaidAudienceFacts struct {
	// Who is paid (paid members only) or free (free subscribers only).
	Who string
	// Tier is the lowest tier's short name ("Insider"), empty for any paid
	// tier; TierTop says it is the highest tier ("Founding member" rather
	// than "Founding member and above").
	Tier    string
	TierTop bool
	// What the paid members' access is.
	Trialing      int64
	Complimentary int64
	Gift          int64
	PastDue       int64
}

func (p *PaidAudienceFacts) count(m paidaccess.Member) {
	if p == nil {
		return
	}
	switch m.Status {
	case "trialing":
		p.Trialing++
	case "past_due":
		p.PastDue++
	case "complimentary":
		p.Complimentary++
	case "gift":
		p.Gift++
	}
}

// paidAudience is the Audience quality line of a paid-only campaign.
func (b *builder) paidAudience() {
	a := b.f.Audience
	p := a.Paid
	if p == nil || a.Empty {
		return
	}
	c := check{Key: "audience.paid", Group: GroupAudience, Status: Pass, Label: tx("label.paidAccess", nil)}
	group := b.groupLabel()
	if p.Who == paidaccess.Free {
		c.Title = tx("paid.title", A{"group": group, "who": tx("paid.free", nil), "count": a.Final})
		c.Detail = tx("paid.freeDetail", nil)
		c.Short = tx("paid.free", nil)
		b.add(c)
		return
	}
	who := tx("paid.members", nil)
	switch {
	case p.Tier != "" && p.TierTop:
		who = tx("paid.tierOnly", A{"tier": p.Tier})
	case p.Tier != "":
		who = tx("paid.tierAndAbove", A{"tier": p.Tier})
	}
	c.Title = tx("paid.title", A{"group": group, "who": who, "count": a.Final})
	c.Short = who
	access := p.Trialing+p.Complimentary+p.Gift > 0 || p.PastDue == 0
	switch {
	case access && p.PastDue > 0:
		c.Detail = tx("paid.includedPastDue", A{"count": p.PastDue})
	case access:
		c.Detail = tx("paid.included", nil)
	default:
		c.Detail = tx("paid.pastDue", A{"count": p.PastDue})
	}
	c.Detail = sentences(c.Detail, tx("paid.testMembers", nil))
	b.add(c)
}
