package remote

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

type runtimeInput struct {
	api.WorkspaceInput
	AdapterID string `path:"adapterId" format:"uuid"`
}

type runtimeUpdateInput struct {
	api.WorkspaceInput
	AdapterID string `path:"adapterId" format:"uuid"`
	Body      AdapterRuntimeInput
}

// Register adds the community adapter runtime operations. Adapters apply to
// every workspace, so on top of providers:manage they need an instance
// admin, like installing one.
func Register(h huma.API, s *Service) {
	tags := []string{"Adapters"}
	access := api.WorkspaceHuman(authz.ProvidersManage)
	base := "/workspaces/{workspace}/adapters/{adapterId}/runtime"
	api.Get(h, api.Op{ID: "get-adapter-runtime", Path: base, Summary: "Where a community adapter runs, its secret's age and its health (instance admins)",
		Description: "The signing secret is never returned.", Tags: tags, Access: access},
		func(ctx context.Context, in *runtimeInput) (*api.Body[AdapterRuntime], error) {
			r, err := s.adminRow(ctx, in.AdapterID)
			if err != nil {
				return nil, err
			}
			out, err := s.View(ctx, r)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-adapter-runtime", Path: base, Summary: "Set where a community adapter's process listens (instance admins)",
		Description: "Loopback and private addresses need providers.allow_private_networks; plain http:// only reaches them.", Tags: tags, Access: access},
		func(ctx context.Context, in *runtimeUpdateInput) (*api.Body[AdapterRuntime], error) {
			r, err := s.adminRow(ctx, in.AdapterID)
			if err != nil {
				return nil, err
			}
			ws := api.WorkspaceFrom(ctx).ID
			if r, err = s.SetEndpoint(ctx, &ws, r, in.Body.EndpointURL, nil); err != nil {
				return nil, err
			}
			out, err := s.View(ctx, r)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "issue-adapter-secret", Path: base + "/secret", Summary: "Issue a new signing secret for a community adapter; it is shown once (instance admins)",
		Description: "The previous secret keeps working for 24 hours, so the adapter process can be updated without downtime.", Tags: tags, Access: access},
		func(ctx context.Context, in *runtimeInput) (*api.Body[AdapterRuntimeSecret], error) {
			r, err := s.adminRow(ctx, in.AdapterID)
			if err != nil {
				return nil, err
			}
			ws := api.WorkspaceFrom(ctx).ID
			out, err := s.IssueSecret(ctx, &ws, r, nil)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "check-adapter-runtime", Path: base + "/check", Summary: "Call a community adapter's health operation now (instance admins)", Tags: tags, Access: access},
		func(ctx context.Context, in *runtimeInput) (*api.Body[AdapterRuntime], error) {
			r, err := s.adminRow(ctx, in.AdapterID)
			if err != nil {
				return nil, err
			}
			ws := api.WorkspaceFrom(ctx).ID
			if r, err = s.Check(ctx, &ws, r, nil); err != nil {
				return nil, err
			}
			out, err := s.View(ctx, r)
			return api.OK(out), err
		})
}

func (s *Service) adminRow(ctx context.Context, id string) (store.RemoteAdapterGetRow, error) {
	if err := instanceAdmin(ctx); err != nil {
		return store.RemoteAdapterGetRow{}, err
	}
	aid, err := uuid.Parse(id)
	if err != nil {
		return store.RemoteAdapterGetRow{}, api.NotFound("Community adapter")
	}
	return s.row(ctx, aid)
}
