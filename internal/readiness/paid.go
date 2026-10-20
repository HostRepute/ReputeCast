package readiness

import (
	"fmt"
	"strings"

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
	c := ReadinessCheck{Key: "audience.paid", Group: GroupAudience, Status: Pass, Label: "Paid access"}
	recipients := Count(a.Final) + " " + Plural(a.Final, "recipient", "recipients")
	if p.Who == paidaccess.Free {
		c.Title = fmt.Sprintf("%s · free subscribers only: %s", a.GroupLabel, recipients)
		c.Detail = "Paid members are left out, with trials, complimentary and gift access. Test paid members never count."
		c.Short = "free subscribers only"
		b.add(c)
		return
	}
	who := "paid members only"
	switch {
	case p.Tier != "" && p.TierTop:
		who = p.Tier + " members only"
	case p.Tier != "":
		who = p.Tier + " and above"
	}
	c.Title = fmt.Sprintf("%s · %s: %s", a.GroupLabel, who, recipients)
	c.Short = who
	var parts []string
	if p.Trialing+p.Complimentary+p.Gift > 0 || p.PastDue == 0 {
		parts = append(parts, "Trials, complimentary and gift access are included")
	}
	if p.PastDue > 0 {
		pd := fmt.Sprintf("%s paid %s whose payment failed while Stripe retries", Count(p.PastDue), Plural(p.PastDue, "member", "members"))
		if len(parts) > 0 {
			parts[0] += ", and " + pd
		} else {
			parts = append(parts, strings.ToUpper(pd[:1])+pd[1:]+" are included")
		}
	}
	c.Detail = strings.Join(parts, "") + ". Test paid members never count."
	b.add(c)
}
