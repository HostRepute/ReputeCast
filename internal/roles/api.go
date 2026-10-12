package roles

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type roleInput struct {
	api.WorkspaceInput
	RoleID string `path:"roleId" format:"uuid"`
}

func (in *roleInput) id() (uuid.UUID, error) {
	id, err := uuid.Parse(in.RoleID)
	if err != nil {
		return uuid.Nil, api.NotFound("Role")
	}
	return id, nil
}

// Register adds the roles operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Roles"}
	base := "/workspaces/{workspace}/roles"

	api.Get(h, api.Op{ID: "list-roles", Path: base, Summary: "List roles with members, and the permission catalogue", Tags: tags, Access: api.Workspace(authz.TeamView)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[RoleList], error) {
			out, err := s.List(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})

	api.Get(h, api.Op{ID: "get-role", Path: base + "/{roleId}", Summary: "One role for the role editor", Tags: tags, Access: api.Workspace(authz.TeamView)},
		func(ctx context.Context, in *roleInput) (*api.Body[RoleDetail], error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			out, err := s.Get(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id)
			return api.OK(out), api.MapError(err, "Role")
		})

	api.Post(h, api.Op{ID: "create-role", Path: base, Summary: "Create a custom role (only permissions you hold)", Tags: tags, Access: api.WorkspaceHuman(authz.TeamManage), Status: 201},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body RoleInput
		}) (*api.Body[RoleCreated], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), api.MapError(err, "Role")
		})

	api.Put(h, api.Op{ID: "update-role", Path: base + "/{roleId}", Summary: "Update a custom role; changes apply to its members at once", Tags: tags, Access: api.WorkspaceHuman(authz.TeamManage), Status: 204},
		func(ctx context.Context, in *struct {
			roleInput
			Body RoleInput
		}) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, api.MapError(s.Update(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), id, in.Body), "Role")
		})

	api.Delete(h, api.Op{ID: "delete-role", Path: base + "/{roleId}", Summary: "Delete a custom role nobody holds", Tags: tags, Access: api.WorkspaceHuman(authz.TeamManage)},
		func(ctx context.Context, in *roleInput) (*api.Empty, error) {
			id, err := in.id()
			if err != nil {
				return nil, err
			}
			return api.NoContent, api.MapError(s.Delete(ctx, api.WorkspaceFrom(ctx), id), "Role")
		})

	api.Post(h, api.Op{ID: "role-impact", Path: base + "/impact", Summary: "Preview who gains or loses what before saving a role", Tags: tags, Access: api.Workspace(authz.TeamManage)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body RoleImpactInput
		}) (*api.Body[RoleImpact], error) {
			out, err := s.Impact(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
}
