package routing

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type getInput struct {
	api.WorkspaceInput
}

type saveInput struct {
	api.WorkspaceInput
	Body RoutingSaveInput
}

type testInput struct {
	api.WorkspaceInput
	Body RouteTestInput
}

// Register adds the routing operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Routing"}
	api.Get(h, api.Op{ID: "get-routing", Path: "/workspaces/{workspace}/routing", Summary: "Routing rules, fallbacks and failover policy per stream",
		Tags: tags, Access: api.Workspace(authz.ProvidersView)},
		func(ctx context.Context, _ *getInput) (*api.Body[RoutingConfig], error) {
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "save-routing", Path: "/workspaces/{workspace}/routing", Summary: "Replace the routing of one or more streams",
		Description: "Validates splits, fallbacks and conditions against the connected providers. A Broadcast route that a provider's terms " +
			"don't allow is refused with code bulk_terms unless acknowledgeBulkTerms is set; the decision is audited.",
		Tags: tags, Access: api.Workspace(authz.ProvidersManage)},
		func(ctx context.Context, in *saveInput) (*api.Body[RoutingConfig], error) {
			out, err := s.Save(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "test-route-campaign", Path: "/workspaces/{workspace}/routing/test-campaign",
		Summary: "Where the next broadcast's recipients go, on the saved routing and an unsaved one", Tags: tags, Access: api.Workspace(authz.ProvidersView)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body RouteCampaignInput
		}) (*api.Body[RouteCampaignResult], error) {
			out, err := s.SimulateCampaign(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "test-route", Path: "/workspaces/{workspace}/routing/test", Summary: "Which provider would send this email?",
		Tags: tags, Access: api.Workspace(authz.ProvidersView)},
		func(ctx context.Context, in *testInput) (*api.Body[RouteTestResult], error) {
			out, err := s.Test(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
}
