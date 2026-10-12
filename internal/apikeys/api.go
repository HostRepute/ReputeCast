package apikeys

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type keyInput struct {
	api.WorkspaceInput
	KeyID string `path:"keyId" format:"uuid"`
}

func (in *keyInput) id(kind string) (uuid.UUID, error) {
	id, err := uuid.Parse(in.KeyID)
	if err != nil {
		return uuid.Nil, api.NotFound(noun(kind))
	}
	return id, nil
}

type actionsInput struct {
	api.WorkspaceInput
	Limit int `query:"limit" minimum:"1" maximum:"100" default:"20"`
}

// Register adds the API key and MCP agent operations.
func Register(h huma.API, s *Service) {
	for _, k := range []struct {
		kind, path, tag, idPrefix, title string
		perm                             authz.Permission
	}{
		{KindAPIKey, "/workspaces/{workspace}/api-keys", "API keys", "api-key", "API key", authz.APIKeysManage},
		{KindAgent, "/workspaces/{workspace}/mcp-agents", "MCP agents", "mcp-agent", "MCP agent", authz.MCPManage},
	} {
		tags := []string{k.tag}
		api.Get(h, api.Op{ID: "list-" + k.idPrefix + "s", Path: k.path, Summary: "List " + k.title + "s with permissions, scope, creator, last use and expiry", Tags: tags, Access: api.Workspace(k.perm)},
			func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[APIKeyList], error) {
				out, err := s.List(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), k.kind)
				return api.OK(out), err
			})
		api.Get(h, api.Op{ID: "get-" + k.idPrefix, Path: k.path + "/{keyId}", Summary: "One " + k.title + " with the calls rejected since it expired", Tags: tags, Access: api.Workspace(k.perm)},
			func(ctx context.Context, in *keyInput) (*api.Body[APIKeyView], error) {
				id, err := in.id(k.kind)
				if err != nil {
					return nil, err
				}
				out, err := s.Get(ctx, api.WorkspaceFrom(ctx), k.kind, id)
				return api.OK(out), err
			})
		api.Post(h, api.Op{ID: "create-" + k.idPrefix, Path: k.path, Summary: "Create an " + k.title + "; the secret is shown once", Tags: tags, Access: api.WorkspaceHuman(k.perm), Status: 201},
			func(ctx context.Context, in *struct {
				api.WorkspaceInput
				Body APIKeyCreateInput
			}) (*api.Body[APIKeyCreated], error) {
				out, err := s.Create(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), k.kind, in.Body)
				return api.OK(out), err
			})
		api.Post(h, api.Op{ID: "rotate-" + k.idPrefix, Path: k.path + "/{keyId}/rotate", Summary: "Rotate (or replace an expired) " + k.title + "; the new secret is shown once", Tags: tags, Access: api.WorkspaceHuman(k.perm)},
			func(ctx context.Context, in *keyInput) (*api.Body[APIKeyCreated], error) {
				id, err := in.id(k.kind)
				if err != nil {
					return nil, err
				}
				out, err := s.Rotate(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), k.kind, id)
				return api.OK(out), err
			})
		api.Post(h, api.Op{ID: "revoke-" + k.idPrefix, Path: k.path + "/{keyId}/revoke", Summary: "Revoke an " + k.title + " immediately", Tags: tags, Access: api.Workspace(k.perm), Status: 204},
			func(ctx context.Context, in *keyInput) (*api.Empty, error) {
				id, err := in.id(k.kind)
				if err != nil {
					return nil, err
				}
				return api.NoContent, s.Revoke(ctx, api.WorkspaceFrom(ctx), k.kind, id)
			})
		api.Delete(h, api.Op{ID: "delete-" + k.idPrefix, Path: k.path + "/{keyId}", Summary: "Delete an expired or revoked " + k.title, Tags: tags, Access: api.Workspace(k.perm)},
			func(ctx context.Context, in *keyInput) (*api.Empty, error) {
				id, err := in.id(k.kind)
				if err != nil {
					return nil, err
				}
				return api.NoContent, s.Delete(ctx, api.WorkspaceFrom(ctx), k.kind, id)
			})
	}
	api.Get(h, api.Op{ID: "list-agent-actions", Path: "/workspaces/{workspace}/mcp-agents/actions", Summary: "Recent MCP agent tool calls from the audit log", Tags: []string{"MCP agents"}, Access: api.Workspace(authz.MCPManage)},
		func(ctx context.Context, in *actionsInput) (*api.Body[MCPAgentActionList], error) {
			out, err := s.AgentActions(ctx, api.WorkspaceFrom(ctx), in.Limit)
			return api.OK(out), err
		})
}
