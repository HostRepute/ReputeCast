package overview

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type wsInput struct {
	api.WorkspaceInput
}

type audienceInput struct {
	api.WorkspaceInput
	Days int `query:"days" minimum:"7" maximum:"365" default:"30"`
}

// Register adds the Overview operations: one per card, each with the
// permission of the data it shows, so a card can fail or be hidden alone.
func Register(h huma.API, s *Service) {
	tags := []string{"Overview"}
	base := "/workspaces/{workspace}/overview"
	api.Get(h, api.Op{ID: "overview-sender-health", Path: base + "/sender-health", Summary: "Sender health: a 0–100 score, its word and the reasons behind it",
		Description: "From the Deliverability checks (blocklists, authentication) and each provider's bounce and complaint rates against its own limits, warm-up and status.",
		Tags:        tags, Access: api.Workspace(authz.DeliverabilityView)},
		func(ctx context.Context, _ *wsInput) (*api.Body[OverviewSenderHealth], error) {
			out, err := s.SenderHealth(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "overview-next-campaign", Path: base + "/next-campaign", Summary: "The next scheduled campaign with its Send Readiness, route split and estimates",
		Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, _ *wsInput) (*api.Body[OverviewNextCampaign], error) {
			out, err := s.NextCampaign(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "overview-providers", Path: base + "/providers", Summary: "Each sending provider's status, rates against its limits, quota, rate, last webhook and cost",
		Tags: tags, Access: api.Workspace(authz.ProvidersView)},
		func(ctx context.Context, _ *wsInput) (*api.Body[OverviewProviders], error) {
			out, err := s.ProviderCards(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "overview-audience", Path: base + "/audience", Summary: "Subscriber growth over the period and engagement tiers with the sunset policy",
		Tags: tags, Access: api.Workspace(authz.SubscribersView)},
		func(ctx context.Context, in *audienceInput) (*api.Body[OverviewAudience], error) {
			out, err := s.Audience.Audience(ctx, api.WorkspaceFrom(ctx), in.Days)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "overview-recent-campaigns", Path: base + "/recent-campaigns", Summary: "The latest campaigns that went out, as the campaigns list shows them",
		Tags: tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, _ *wsInput) (*api.Body[OverviewRecentCampaigns], error) {
			out, err := s.Recent(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
}
