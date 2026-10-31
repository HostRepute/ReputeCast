package renders

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type connectInput struct {
	api.WorkspaceInput
	Body RenderingConnectInput
}

// CampaignPath is the {workspace}/{campaignId} part of render operations
// (exported: huma reads path parameters of embedded structs only when
// they're exported).
type CampaignPath struct {
	api.WorkspaceInput
	CampaignID string `path:"campaignId" format:"uuid"`
}

func (in CampaignPath) campaign(ctx context.Context, s *Service) (uuid.UUID, error) {
	id, err := uuid.Parse(in.CampaignID)
	if err != nil {
		return uuid.Nil, api.NotFound("Campaign")
	}
	if _, err := s.Content.CampaignName(ctx, api.WorkspaceFrom(ctx), id); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

type renderInput struct {
	CampaignPath
	Body CampaignRenderInput
}

// Register adds the rendering service operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Rendering"}
	base := "/workspaces/{workspace}/rendering-service"
	api.Get(h, api.Op{ID: "get-rendering-service", Path: base, Summary: "The workspace's rendering service (Litmus or Mailgun Inspect)",
		Description: "Whether real renders are available, from which service, and its email clients. The API key is never returned.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, _ *struct{ api.WorkspaceInput }) (*api.Body[RenderingService], error) {
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "connect-rendering-service", Path: base, Summary: "Connect Litmus or Mailgun Inspect with the workspace's own API key",
		Description: "The key is tested by listing the account's email clients, then sealed at rest. It is never returned; a new key replaces the old one.",
		Tags:        tags, Access: api.Workspace(authz.IntegrationsManage)},
		func(ctx context.Context, in *connectInput) (*api.Body[RenderingService], error) {
			out, err := s.Connect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "test-rendering-service", Path: base + "/test", Summary: "Test the rendering service's key again",
		Description: "Lists the account's email clients with the stored key; status and lastError say how it went.",
		Tags:        tags, Access: api.Workspace(authz.IntegrationsManage)},
		func(ctx context.Context, _ *struct{ api.WorkspaceInput }) (*api.Body[RenderingService], error) {
			out, err := s.Test(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "disconnect-rendering-service", Path: base, Summary: "Disconnect the rendering service and forget its key",
		Description: "Renders already made stay with their campaigns.",
		Tags:        tags, Access: api.Workspace(authz.IntegrationsManage)},
		func(ctx context.Context, _ *struct{ api.WorkspaceInput }) (*api.Empty, error) {
			return api.NoContent, s.Disconnect(ctx, api.WorkspaceFrom(ctx))
		})
	api.Get(h, api.Op{ID: "list-campaign-renders", Path: "/workspaces/{workspace}/campaigns/{campaignId}/renders", Summary: "A campaign's latest real renders",
		Description: "Newest first. Running renders are refreshed from the service at most every few seconds.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsView)},
		func(ctx context.Context, in *CampaignPath) (*api.Body[RenderList], error) {
			id, err := in.campaign(ctx, s)
			if err != nil {
				return nil, err
			}
			out, err := s.List(ctx, api.WorkspaceFrom(ctx), id)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "render-campaign", Path: "/workspaces/{workspace}/campaigns/{campaignId}/renders", Summary: "Render the saved campaign in real email clients",
		Description: "Sends the saved version (one A/B variant), personalized for the sample contact, to the connected service. Each client is one preview on the account's plan.",
		Tags:        tags, Access: api.Workspace(authz.CampaignsEdit), Status: 201},
		func(ctx context.Context, in *renderInput) (*api.Body[Render], error) {
			id, err := in.campaign(ctx, s)
			if err != nil {
				return nil, err
			}
			out, err := s.Start(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body)
			return api.OK(out), err
		})
}
