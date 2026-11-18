package mcpclient

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type workspaceInput struct {
	api.WorkspaceInput
}

type createInput struct {
	api.WorkspaceInput
	Body RemoteMCPServerInput
}

type serverPath struct {
	api.WorkspaceInput
	Server uuid.UUID `path:"server" format:"uuid" doc:"The MCP server's id"`
}

type updateInput struct {
	api.WorkspaceInput
	Server uuid.UUID `path:"server" format:"uuid" doc:"The MCP server's id"`
	Body   RemoteMCPServerUpdate
}

type decideInput struct {
	api.WorkspaceInput
	Server uuid.UUID `path:"server" format:"uuid" doc:"The MCP server's id"`
	Tool   uuid.UUID `path:"tool" format:"uuid" doc:"The tool's id"`
	Body   RemoteMCPToolDecision
}

type callbackInput struct {
	MCPOAuthCallbackInput
}

type redirect struct {
	Location string `header:"Location"`
}

// Register adds the MCP servers' operations: managers add, sync and
// review servers; members sign in to OAuth servers and pick the servers
// their threads use. People only: API keys and MCP agents can't.
func Register(h huma.API, s *Service) {
	tags := []string{"AI assistant · MCP servers"}
	base := "/workspaces/{workspace}/ai/mcp/servers"
	view := api.WorkspaceHuman(authz.IntegrationsView)
	manage := api.WorkspaceHuman(authz.IntegrationsManage)
	use := api.WorkspaceHuman(authz.AIUse)

	api.Get(h, api.Op{ID: "list-remote-mcp-servers", Path: base, Summary: "The workspace's MCP servers and their tools",
		Description: "Never their secrets: headerSet and clientSecretSet say whether one is stored. A tool is offered to threads once a manager marked it read-only or writes at its current hash; changed says it moved since and is off.",
		Tags:        tags, Access: view},
		func(ctx context.Context, in *workspaceInput) (*api.Body[RemoteMCPServerList], error) {
			out, err := s.Servers(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "create-remote-mcp-server", Path: base, Summary: "Add a remote MCP server",
		Description: "Streamable HTTP, or HTTP with SSE; never a local process. Its address must pass the SSRF guard. Authentication: none, a shared header (write-only), or OAuth per member. It has no tools until a sync. Audited as mcp_server.created, without secrets.",
		Tags:        tags, Access: manage},
		func(ctx context.Context, in *createInput) (*api.Body[RemoteMCPServer], error) {
			out, err := s.Create(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Patch(h, api.Op{ID: "update-remote-mcp-server", Path: base + "/{server}", Summary: "Rename an MCP server, switch it, change who may use it or replace its secrets",
		Description: "Audited as mcp_server.updated, without secrets.", Tags: tags, Access: manage},
		func(ctx context.Context, in *updateInput) (*api.Body[RemoteMCPServer], error) {
			out, err := s.Update(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Server, in.Body)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "delete-remote-mcp-server", Path: base + "/{server}", Status: http.StatusNoContent, Summary: "Remove an MCP server, its tools and every member's sign-in to it",
		Description: "Threads stop offering its tools; a call waiting for confirmation fails. Audited as mcp_server.deleted.", Tags: tags, Access: manage},
		func(ctx context.Context, in *serverPath) (*api.Empty, error) {
			return api.NoContent, s.Delete(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Server)
		})
	api.Post(h, api.Op{ID: "sync-remote-mcp-server", Path: base + "/{server}/sync", Summary: "List an MCP server's tools into its catalog",
		Description: "As the caller (an OAuth server needs their own sign-in). New tools wait for a decision; a tool whose description, schema or annotations changed after a decision is off, and managers are notified. Audited as mcp_server.updated.",
		Tags:        tags, Access: manage},
		func(ctx context.Context, in *serverPath) (*api.Body[RemoteMCPServer], error) {
			out, err := s.Sync(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Server)
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "decide-remote-mcp-tool", Path: base + "/{server}/tools/{tool}", Summary: "Mark a tool read-only, writes or blocked",
		Description: "At the hash the manager reviewed: a tool that changed since is a 409 (mcp_tool_reviewed). Annotations are the server's hints, not evidence. A read-only tool runs without asking; each call of a writing tool waits for its person's confirmation. Audited as mcp_tool.approved or mcp_tool.revoked.",
		Tags:        tags, Access: manage},
		func(ctx context.Context, in *decideInput) (*api.Body[RemoteMCPServer], error) {
			out, err := s.Decide(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Server, in.Tool, in.Body)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "start-remote-mcp-oauth", Path: base + "/{server}/oauth", Summary: "Begin the caller's own sign-in to an OAuth MCP server",
		Description: "Returns the authorization server's address to open in the browser; it sends the person back to remote-mcp-oauth-callback. The sign-in is the caller's only, and only from this session.",
		Tags:        tags, Access: use},
		func(ctx context.Context, in *serverPath) (*api.Body[RemoteMCPOAuthStart], error) {
			out, err := s.StartOAuth(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Server)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "disconnect-remote-mcp-oauth", Path: base + "/{server}/oauth", Status: http.StatusNoContent, Summary: "Remove the caller's own sign-in to an OAuth MCP server",
		Description: "Audited as mcp_server.disconnect.", Tags: tags, Access: use},
		func(ctx context.Context, in *serverPath) (*api.Empty, error) {
			return api.NoContent, s.Disconnect(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Server)
		})
	api.Get(h, api.Op{ID: "list-ai-assistant-mcp-servers", Path: "/workspaces/{workspace}/ai/assistant/mcp-servers", Summary: "The MCP servers the caller may pick for their threads",
		Description: "Enabled servers for everyone with Use AI features or for the caller's role, with whether the caller still has to sign in (connected).",
		Tags:        tags, Access: use},
		func(ctx context.Context, in *workspaceInput) (*api.Body[RemoteMCPServerList], error) {
			out, err := s.Usable(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "remote-mcp-oauth-callback", Path: "/ai/mcp/oauth/callback", Status: http.StatusSeeOther, Summary: "Where an OAuth MCP server's authorization server sends the person back",
		Description: "The state must be the caller's, from the session that started the sign-in, unused and at most ten minutes old (400 mcp_oauth_state otherwise). Redirects to Settings › MCP servers with connected=1 or error=denied|token. Audited as mcp_server.connect.",
		Tags:        tags, Access: api.Authenticated},
		func(ctx context.Context, in *callbackInput) (*redirect, error) {
			to, err := s.Callback(ctx, api.PrincipalFrom(ctx), in.MCPOAuthCallbackInput)
			if err != nil {
				return nil, err
			}
			return &redirect{Location: to}, nil
		})
}
