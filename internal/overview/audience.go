package overview

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// AudienceStats reads subscriber growth and tiers from the contacts and
// list memberships (the audience domain's tables).
type AudienceStats struct {
	Q   *store.Queries
	Now func() time.Time
}

func (a AudienceStats) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

// Audience returns the subscriber count, its daily series and net change
// over the period, each list's size and change, the engagement tiers and
// the sunset policy.
func (a AudienceStats) Audience(ctx context.Context, ws *store.Workspace, days int) (OverviewAudience, error) {
	out := OverviewAudience{Days: days, Series: []OverviewAudiencePoint{}, Lists: []OverviewListGrowth{}, Tiers: []OverviewTier{}}
	totals, err := a.Q.OverviewAudienceTotals(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Total = totals.Contacts
	loc := state.Location(ws.Timezone)
	n := a.now().In(loc)
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
	start := today.AddDate(0, 0, -days)
	series, err := a.Q.OverviewContactsByDay(ctx, store.OverviewContactsByDayParams{WorkspaceID: ws.ID, Since: start, Until: today})
	if err != nil {
		return out, err
	}
	for _, p := range series {
		out.Series = append(out.Series, OverviewAudiencePoint{Day: p.Day.Format("2006-01-02"), Contacts: p.Contacts})
	}
	if len(out.Series) > 0 {
		first := out.Series[0].Contacts
		out.Net = out.Total - first
		if first > 0 {
			v := round(float64(out.Net)/float64(first)*100, 1)
			out.NetPercent = &v
		}
	}
	since := start.Add(24 * time.Hour)
	lists, err := a.Q.OverviewListGrowth(ctx, store.OverviewListGrowthParams{Since: &since, WorkspaceID: ws.ID})
	if err != nil {
		return out, err
	}
	for _, l := range lists {
		out.Lists = append(out.Lists, OverviewListGrowth{ID: l.ID.String(), Name: l.Name, Contacts: l.Subscribed, Net: l.Joined - l.Departed})
	}
	p := parsePolicy(ws.SunsetPolicy)
	counts := []int64{totals.Active, totals.Cooling, totals.AtRisk, totals.Sunset}
	for i, t := range []string{"active", "cooling", "at_risk", "sunset"} {
		tier := OverviewTier{Tier: t, Count: counts[i], Description: p.describe(t)}
		if totals.Contacts > 0 {
			tier.Pct = round(float64(counts[i])/float64(totals.Contacts)*100, 1)
		}
		out.Tiers = append(out.Tiers, tier)
	}
	out.Policy, out.PolicyEnabled = p.sentence(), p.Enabled
	return out, nil
}

// sunsetPolicy is the workspace's sunset policy, stored as JSON on the
// workspace (the audience domain edits it), with its defaults.
type sunsetPolicy struct {
	Enabled           bool
	ActiveDays        int
	AtRiskDays        int
	SunsetDays        int
	Reengage          bool
	StopBroadcasts    bool
	StopAfterDays     int
	DeleteAfter       bool
	DeleteAfterMonths int
}

func parsePolicy(raw json.RawMessage) sunsetPolicy {
	var d struct {
		Enabled           *bool `json:"enabled"`
		ActiveDays        int   `json:"active_days"`
		AtRiskDays        int   `json:"at_risk_days"`
		SunsetDays        int   `json:"sunset_days"`
		AfterDays         int   `json:"after_days"`
		Reengage          *bool `json:"reengage"`
		StopBroadcasts    *bool `json:"stop_broadcasts"`
		StopAfterDays     int   `json:"stop_after_days"`
		DeleteAfter       bool  `json:"delete_after"`
		DeleteAfterMonths int   `json:"delete_after_months"`
	}
	_ = json.Unmarshal(raw, &d)
	p := sunsetPolicy{Enabled: true, ActiveDays: 30, AtRiskDays: 91, SunsetDays: 181, Reengage: true, StopBroadcasts: true, StopAfterDays: 30, DeleteAfterMonths: 12}
	if d.Enabled != nil {
		p.Enabled = *d.Enabled
	}
	if d.ActiveDays > 0 {
		p.ActiveDays = d.ActiveDays
	}
	if d.AtRiskDays > 0 {
		p.AtRiskDays = d.AtRiskDays
	}
	switch {
	case d.SunsetDays > 0:
		p.SunsetDays = d.SunsetDays
	case d.AfterDays > 0:
		p.SunsetDays = d.AfterDays + 1
	}
	if d.Reengage != nil {
		p.Reengage = *d.Reengage
	}
	if d.StopBroadcasts != nil {
		p.StopBroadcasts = *d.StopBroadcasts
	}
	if d.StopAfterDays > 0 {
		p.StopAfterDays = d.StopAfterDays
	}
	p.DeleteAfter = d.DeleteAfter
	if d.DeleteAfterMonths > 0 {
		p.DeleteAfterMonths = d.DeleteAfterMonths
	}
	return p
}

// describe says what a tier means: "opened in the last 30 days".
func (p sunsetPolicy) describe(tier string) string {
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

// sentence is the sunset policy in one line.
func (p sunsetPolicy) sentence() string {
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
