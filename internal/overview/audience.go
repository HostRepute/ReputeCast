package overview

import (
	"context"
	"time"

	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// AudienceCards builds "Subscriber growth" and "Engagement tiers" on the
// audience domain: the counts of the Audience header (contacts, tiers,
// lists), each list's joins and departures as List health counts them, and
// the sunset policy.
type AudienceCards struct {
	Service *audience.Service
	Q       *store.Queries
	Now     func() time.Time
}

func (a AudienceCards) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

// Audience returns the subscribers (list memberships) with their net change
// over the period and a point a day, the unique contacts, each list's size
// and change, the engagement tiers and the sunset policy.
func (a AudienceCards) Audience(ctx context.Context, ws *store.Workspace, days int) (OverviewAudience, error) {
	out := OverviewAudience{Days: days, Series: []OverviewAudiencePoint{}, Lists: []OverviewListGrowth{}, Tiers: []OverviewTier{}}
	sum, err := a.Service.Summary(ctx, ws)
	if err != nil {
		return out, err
	}
	out.Total = sum.Contacts
	now := a.now()
	since := now.AddDate(0, 0, -days)

	// Each list's size and its joins minus departures, as on List health.
	growth, err := a.Q.ListGrowthSince(ctx, store.ListGrowthSinceParams{WorkspaceID: ws.ID, Since: &since})
	if err != nil {
		return out, err
	}
	net := map[string]int64{}
	for _, g := range growth {
		net[g.ListID.String()] = g.Joined - g.Departed
	}
	for _, l := range sum.Lists {
		// The newsletter's lists: like "All subscribers", not the customers
		// a connected store adds (X20).
		if l.Origin == "store" {
			continue
		}
		out.Lists = append(out.Lists, OverviewListGrowth{ID: l.ID, Name: l.Name, Contacts: l.Subscribed, Net: net[l.ID]})
		out.Subscribers += l.Subscribed
		out.Net += net[l.ID]
	}

	// The line: today's subscribers, walked back a day at a time by the
	// joins and departures of each 24 hours, so it starts at today minus
	// the net change.
	steps, err := a.Q.OverviewListChanges(ctx, store.OverviewListChangesParams{WorkspaceID: ws.ID, Since: since})
	if err != nil {
		return out, err
	}
	change := make([]int64, days)
	for _, s := range steps {
		k := min(max(int(s.Step), 0), days-1)
		change[k] += s.Joined - s.Departed
	}
	loc := state.Location(ws.Timezone)
	points := make([]OverviewAudiencePoint, days+1)
	v := out.Subscribers
	points[days] = OverviewAudiencePoint{Day: now.In(loc).Format(time.DateOnly), Subscribers: v}
	for k := days - 1; k >= 0; k-- {
		v -= change[k]
		points[k] = OverviewAudiencePoint{Day: since.AddDate(0, 0, k).In(loc).Format(time.DateOnly), Subscribers: v}
	}
	out.Series = points
	if start := out.Subscribers - out.Net; start > 0 {
		pct := round(float64(out.Net)/float64(start)*100, 1)
		out.NetPercent = &pct
	}

	p := audience.ParsePolicy(ws.SunsetPolicy)
	l := i18n.LangFrom(ctx)
	counts := []int64{sum.Tiers.Active, sum.Tiers.Cooling, sum.Tiers.AtRisk, sum.Tiers.Sunset}
	for i, t := range []string{"active", "cooling", "at_risk", "sunset"} {
		tier := OverviewTier{Tier: t, Count: counts[i], Description: describeTier(l, p, t)}
		if sum.Contacts > 0 {
			tier.Pct = round(float64(counts[i])/float64(sum.Contacts)*100, 1)
		}
		out.Tiers = append(out.Tiers, tier)
	}
	out.Policy, out.PolicyEnabled = policySentence(l, p), p.Enabled
	return out, nil
}

// describeTier says what a tier means under the policy, in the reader's
// language: "opened in the last 30 days" (the thresholds audience.TierFor
// applies).
func describeTier(l i18n.Lang, p audience.SunsetPolicy, tier string) string {
	switch tier {
	case "active":
		return mTierActive.Text(l, i18n.Args{"days": p.ActiveDays})
	case "cooling":
		return mTierRange.Text(l, i18n.Args{"from": p.ActiveDays + 1, "to": p.AtRiskDays - 1})
	case "at_risk":
		return mTierRange.Text(l, i18n.Args{"from": p.AtRiskDays, "to": p.SunsetDays - 1})
	}
	return mTierSunset.Text(l, i18n.Args{"days": p.SunsetDays - 1})
}

// policySentence is the sunset policy in one sentence, in the reader's
// language.
func policySentence(l i18n.Lang, p audience.SunsetPolicy) string {
	if !p.Enabled {
		return mPolicyOff.Text(l, nil)
	}
	mode := "keep"
	switch {
	case p.Reengage && p.StopBroadcasts:
		mode = "both"
	case p.StopBroadcasts:
		mode = "stop"
	case p.Reengage:
		mode = "reengage"
	}
	erase := "no"
	if p.DeleteAfter {
		erase = "yes"
	}
	return mPolicyOn.Text(l, i18n.Args{"mode": mode, "days": p.StopAfterDays, "erase": erase, "months": p.DeleteAfterMonths})
}
