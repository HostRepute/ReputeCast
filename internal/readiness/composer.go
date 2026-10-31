package readiness

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// ComposerReadiness is what the composer's checks show from Send Readiness
// (Composer board: the Workspace rules and Sender reputation rows), from
// the campaign's latest run: the checks aren't run again for it.
type ComposerReadiness struct {
	// RanAt is when the latest run ran; nil when the checks never ran.
	RanAt *time.Time `json:"ranAt,omitempty"`
	// CheckedAt is when HostRepute's hourly blocklist check ran.
	CheckedAt *time.Time `json:"checkedAt,omitempty"`
	// Rules is nil when no workspace rule applies to the campaign.
	Rules      *ComposerReadinessGroup `json:"rules,omitempty"`
	Reputation *ComposerReadinessGroup `json:"reputation,omitempty"`
}

// ComposerReadinessGroup is how a group stands: its warnings and blocks,
// with their checks in the reader's language.
type ComposerReadinessGroup struct {
	Warnings int              `json:"warnings"`
	Blocks   int              `json:"blocks"`
	Checks   []ReadinessCheck `json:"checks"`
}

// ComposerFindings is a run's workspace rules and sender reputation for the
// composer's checks.
func ComposerFindings(run *ReadinessRun) ComposerReadiness {
	out := ComposerReadiness{}
	if run == nil {
		return out
	}
	at := run.RanAt
	out.RanAt, out.CheckedAt = &at, run.Estimates.HourlyAt
	group := func(key string) *ComposerReadinessGroup {
		var g *ComposerReadinessGroup
		for _, c := range run.Checks {
			if c.Group != key {
				continue
			}
			if g == nil {
				g = &ComposerReadinessGroup{Checks: []ReadinessCheck{}}
			}
			switch c.Status {
			case Warn:
				g.Warnings++
				g.Checks = append(g.Checks, c)
			case Block:
				g.Blocks++
				g.Checks = append(g.Checks, c)
			}
		}
		return g
	}
	out.Rules, out.Reputation = group(GroupRules), group(GroupReputation)
	return out
}

// registerComposer adds the composer's read of Send Readiness.
func registerComposer(h huma.API, s *Service, tags []string) {
	api.Get(h, api.Op{ID: "get-composer-readiness", Path: "/workspaces/{workspace}/campaigns/{campaignId}/readiness/composer",
		Summary:     "Workspace rules and sender reputation for the composer's checks",
		Description: "From the campaign's latest Send Readiness run, without running the checks: how the workspace's sending rules and the sender's reputation stand, their warnings and blocks, and when the blocklists were checked.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *ReadinessCampaignPath) (*api.Body[ComposerReadiness], error) {
			id, err := in.ID()
			if err != nil {
				return nil, err
			}
			if ws, err := s.Q.ReadinessCampaignWorkspace(ctx, id); err != nil || ws != api.WorkspaceFrom(ctx).ID {
				return nil, api.NotFound("Campaign")
			}
			run, err := s.Latest(ctx, id)
			if err != nil {
				return nil, err
			}
			return api.OK(ComposerFindings(ForReader(ctx, run))), nil
		})
}
