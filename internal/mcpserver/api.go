package mcpserver

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

// Register adds the MCP server settings operations.
func Register(h huma.API, s *Service) {
	tags := []string{"MCP agents"}
	api.Get(h, api.Op{ID: "get-mcp-server", Path: "/workspaces/{workspace}/mcp", Summary: "MCP server endpoint, settings, tools and what each agent may do", Tags: tags, Access: api.Workspace(authz.MCPManage)},
		func(ctx context.Context, _ *api.WorkspaceInput) (*api.Body[MCPOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "update-mcp-server", Path: "/workspaces/{workspace}/mcp", Summary: "Turn the MCP server on or off, set the rate limit and proposal expiry", Tags: tags, Access: api.WorkspaceHuman(authz.MCPManage)},
		func(ctx context.Context, in *struct {
			api.WorkspaceInput
			Body MCPSettingsInput
		}) (*api.Body[MCPOverview], error) {
			out, err := s.UpdateSettings(ctx, api.WorkspaceFrom(ctx), in.Body)
			return api.OK(out), err
		})
}
