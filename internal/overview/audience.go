package overview

import (
	"context"
	"fmt"
	"time"

	"github.com/hostrepute/reputecast/internal/audience"
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
	counts := []int64{sum.Tiers.Active, sum.Tiers.Cooling, sum.Tiers.AtRisk, sum.Tiers.Sunset}
	for i, t := range []string{"active", "cooling", "at_risk", "sunset"} {
		tier := OverviewTier{Tier: t, Count: counts[i], Description: describeTier(p, t)}
		if sum.Contacts > 0 {
			tier.Pct = round(float64(counts[i])/float64(sum.Contacts)*100, 1)
		}
		out.Tiers = append(out.Tiers, tier)
	}
	out.Policy, out.PolicyEnabled = policySentence(p), p.Enabled
	return out, nil
}

// describeTier says what a tier means under the policy: "opened in the
// last 30 days" (the thresholds audience.TierFor applies).
func describeTier(p audience.SunsetPolicy, tier string) string {
	switch tier {
	case "active":
		return fmt.Sprintf("opened in the last %d days", p.ActiveDays)
	case "cooling":
		return fmt.Sprintf("last open %d to %d days ago", p.ActiveDays+1, p.AtRiskDays-1)
	case "at_risk":
		return fmt.Sprintf("last open %d to %d days ago", p.AtRiskDays, p.SunsetDays-1)
	}
	return fmt.Sprintf("no open in %d days", p.SunsetDays-1)
}

// policySentence is the sunset policy in one line.
func policySentence(p audience.SunsetPolicy) string {
	if !p.Enabled {
		return "No sunset policy: Sunset contacts keep receiving broadcasts."
	}
	var s string
	switch {
	case p.Reengage && p.StopBroadcasts:
		s = fmt.Sprintf("Sunset contacts get one re-engagement email, then stop receiving broadcasts after %d days", p.StopAfterDays)
	case p.StopBroadcasts:
		s = fmt.Sprintf("Sunset contacts stop receiving broadcasts after %d days", p.StopAfterDays)
	case p.Reengage:
		s = "Sunset contacts get one re-engagement email and keep receiving broadcasts"
	default:
		s = "Sunset contacts keep receiving broadcasts"
	}
	if p.DeleteAfter {
		unit := "months"
		if p.DeleteAfterMonths == 1 {
			unit = "month"
		}
		s += fmt.Sprintf("; they're erased after %d %s in Sunset", p.DeleteAfterMonths, unit)
	}
	return s + "."
}
