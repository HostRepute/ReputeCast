package overview

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/store"
)

// NextCampaign is the next scheduled campaign with its latest Send Readiness
// run, recipients, route split and the cost and duration estimates.
func (s *Service) NextCampaign(ctx context.Context, ws *store.Workspace) (OverviewNextCampaign, error) {
	var out OverviewNextCampaign
	drafts, err := s.Q.OverviewDraftCount(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	out.Drafts = drafts
	if s.NextIssue != nil {
		if out.Issue, err = s.NextIssue(ctx, ws); err != nil {
			return out, err
		}
	}
	// A send that should have started minutes ago is still "next".
	since := s.now().Add(-time.Hour)
	c, err := s.Q.OverviewNextScheduled(ctx, store.OverviewNextScheduledParams{WorkspaceID: ws.ID, Since: &since})
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	sc := &OverviewScheduledCampaign{ID: c.ID.String(), Name: c.Name, Routes: []OverviewRouteShare{}, Timezone: ws.Timezone}
	if c.ScheduledAt != nil {
		sc.ScheduledAt = *c.ScheduledAt
	}
	var sched struct {
		Timezone string `json:"timezone"`
	}
	if json.Unmarshal(c.Schedule, &sched) == nil && sched.Timezone != "" {
		sc.Timezone = sched.Timezone
	}
	if r, err := s.Q.OverviewLatestReadiness(ctx, c.ID); err == nil {
		rd := &OverviewReadiness{Verdict: r.Verdict, Issues: []OverviewReadinessIssue{}, RanAt: r.RanAt}
		var checks struct {
			Checks []struct {
				Status string `json:"status"`
				Title  string `json:"title"`
			} `json:"checks"`
		}
		_ = json.Unmarshal(r.Checks, &checks)
		// Blocks first, then warnings, as Send Readiness lists them.
		for _, want := range []string{"block", "warn"} {
			for _, ch := range checks.Checks {
				if ch.Status == want {
					rd.Issues = append(rd.Issues, OverviewReadinessIssue{Status: ch.Status, Title: ch.Title})
				}
			}
		}
		sc.Readiness = rd
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	routes, err := s.Q.OverviewCampaignRoutes(ctx, c.ID)
	if err != nil {
		return out, err
	}
	var planned int64
	for _, r := range routes {
		planned += r.Planned
	}
	rates := map[string]float64{}
	if len(routes) > 0 && s.Providers != nil {
		po, err := s.Providers.Overview(ctx, ws)
		if err != nil {
			return out, err
		}
		for _, card := range po.Providers {
			rates[card.ID] = card.Rate.Max
		}
	}
	var cost, longest float64
	known := len(routes) > 0
	for _, r := range routes {
		share := OverviewRouteShare{Short: state.Short(r.ProviderName, r.Adapter), Adapter: r.Adapter, Color: state.Color(r.Adapter), Planned: r.Planned}
		if planned > 0 {
			share.Percent = round(float64(r.Planned)/float64(planned)*100, 1)
		}
		sc.Routes = append(sc.Routes, share)
		cost += float64(r.CostMicros) / 1e6
		// Providers send in parallel: the slowest route sets the duration.
		if rate := rates[r.ProviderID.String()]; rate > 0 {
			longest = math.Max(longest, float64(r.Planned)/rate/60)
		} else if r.Planned > 0 {
			known = false
		}
	}
	sc.Cost = round(cost, 2)
	if known {
		d := math.Ceil(longest)
		sc.DurationMinutes = &d
	}
	sc.Recipients = planned
	if c.AudienceCount != nil && *c.AudienceCount > 0 {
		sc.Recipients = *c.AudienceCount
	}
	out.Campaign = sc
	return out, nil
}
