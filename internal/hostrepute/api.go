package hostrepute

import (
	"context"
	"net/http"
	"net/url"

	"github.com/danielgtaylor/huma/v2"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type workspaceOnly struct {
	api.WorkspaceInput
}

type hostReputeAPIKeyInput struct {
	api.WorkspaceInput
	Body struct {
		APIKey string `json:"apiKey" minLength:"1" maxLength:"512" doc:"A HostRepute API key (hr_live_…) with the monitor:read, alerts:read and assets:write scopes. Write-only: it is stored encrypted and never returned."`
	}
}

// HostReputeOAuthStart is where to send the browser for HostRepute's consent page.
type HostReputeOAuthStart struct {
	AuthorizeURL string `json:"authorizeUrl" format:"uri"`
}

type callbackInput struct {
	Code             string `query:"code" maxLength:"2048"`
	State            string `query:"state" maxLength:"512"`
	Error            string `query:"error" maxLength:"200"`
	ErrorDescription string `query:"error_description" maxLength:"1000"`
}

type redirectOutput struct {
	Status       int
	Location     string `header:"Location"`
	CacheControl string `header:"Cache-Control"`
}

type hostReputeAddAssetInput struct {
	api.WorkspaceInput
	Body struct {
		Asset string `json:"asset" minLength:"1" maxLength:"253" doc:"A sending domain, organization domain, dedicated IP or server hostname of this workspace"`
	}
}

type hostReputeSwapAssetInput struct {
	api.WorkspaceInput
	Body struct {
		Remove string `json:"remove" minLength:"1" maxLength:"253"`
		Add    string `json:"add" minLength:"1" maxLength:"253"`
	}
}

// Register adds the HostRepute and Settings › Integrations operations.
func Register(h huma.API, s *Service) {
	tags := []string{"Integrations"}
	api.Get(h, api.Op{ID: "get-integrations", Path: "/workspaces/{workspace}/integrations", Summary: "Settings › Integrations: HostRepute, webhooks, MCP, Postmaster and SNDS",
		Tags: tags, Access: api.Workspace(authz.DeliverabilityView)},
		func(ctx context.Context, _ *workspaceOnly) (*api.Body[HostReputeIntegrationsOverview], error) {
			out, err := s.Overview(ctx, api.WorkspaceFrom(ctx).ID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "start-hostrepute-oauth", Path: "/workspaces/{workspace}/integrations/hostrepute/oauth",
		Summary:     "Start connecting HostRepute (OAuth 2 with PKCE)",
		Description: "Returns HostRepute's consent URL. The browser comes back to /api/v1/integrations/hostrepute/callback.",
		Tags:        tags, Access: api.WorkspaceHuman(authz.IntegrationsHostRepute)},
		func(ctx context.Context, _ *workspaceOnly) (*api.Body[HostReputeOAuthStart], error) {
			u, err := s.StartOAuth(ctx, api.WorkspaceFrom(ctx), api.UserFrom(ctx))
			return api.OK(HostReputeOAuthStart{AuthorizeURL: u}), err
		})
	// One callback per instance: the single-use state names the workspace
	// and the person, and the handler checks integrations:hostrepute there.
	api.Get(h, api.Op{ID: "hostrepute-oauth-callback", Path: "/integrations/hostrepute/callback",
		Summary:     "OAuth return from HostRepute (browser redirect)",
		Description: "Exchanges the code, stores the encrypted tokens and redirects to the connect screen. Requires the signed-in person who started the handoff, holding integrations:hostrepute in its workspace.",
		Tags:        tags, Access: api.Access{Kind: api.AccessPublic, Permission: authz.IntegrationsHostRepute}},
		func(ctx context.Context, in *callbackInput) (*redirectOutput, error) {
			self := CallbackPath + "?" + url.Values{"code": {in.Code}, "state": {in.State}, "error": {in.Error}}.Encode()
			loc := s.CompleteOAuth(ctx, api.UserFrom(ctx), CallbackParams{Code: in.Code, State: in.State, Error: in.Error, ErrorDescription: in.ErrorDescription, Self: self})
			return &redirectOutput{Status: http.StatusFound, Location: loc, CacheControl: "no-store"}, nil
		})
	api.Post(h, api.Op{ID: "connect-hostrepute-api-key", Path: "/workspaces/{workspace}/integrations/hostrepute/api-key",
		Summary: "Connect HostRepute with an API key", Tags: tags, Access: api.Workspace(authz.IntegrationsHostRepute)},
		func(ctx context.Context, in *hostReputeAPIKeyInput) (*api.Body[HostReputeConnection], error) {
			out, err := s.ConnectAPIKey(ctx, api.WorkspaceFrom(ctx), in.Body.APIKey)
			return api.OK(out), err
		})
	api.Delete(h, api.Op{ID: "disconnect-hostrepute", Path: "/workspaces/{workspace}/integrations/hostrepute",
		Summary: "Disconnect HostRepute", Description: "Forgets (and revokes) the token and the data synced from HostRepute. The HostRepute team keeps monitoring on hostrepute.com.",
		Tags: tags, Access: api.Workspace(authz.IntegrationsHostRepute)},
		func(ctx context.Context, _ *workspaceOnly) (*api.Empty, error) {
			return api.NoContent, s.Disconnect(ctx, api.WorkspaceFrom(ctx))
		})
	api.Post(h, api.Op{ID: "sync-hostrepute", Path: "/workspaces/{workspace}/integrations/hostrepute/sync",
		Summary: "Sync HostRepute data now", Tags: tags, Access: api.Workspace(authz.IntegrationsHostRepute), Status: http.StatusAccepted},
		func(ctx context.Context, _ *workspaceOnly) (*api.Body[HostReputeSyncQueued], error) {
			out, err := s.SyncNow(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "add-hostrepute-asset", Path: "/workspaces/{workspace}/integrations/hostrepute/assets",
		Summary:     "Add a domain, IP or hostname to HostRepute monitoring",
		Description: "Answers 409 with code monitor_limit_reached (meta: limit, used, team, asset, swap) when the HostRepute plan is full.",
		Tags:        tags, Access: api.Workspace(authz.IntegrationsHostRepute)},
		func(ctx context.Context, in *hostReputeAddAssetInput) (*api.Body[HostReputeConnection], error) {
			out, err := s.AddAsset(ctx, api.WorkspaceFrom(ctx), in.Body.Asset)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "swap-hostrepute-asset", Path: "/workspaces/{workspace}/integrations/hostrepute/assets/swap",
		Summary: "Stop monitoring one asset and monitor another instead", Tags: tags, Access: api.Workspace(authz.IntegrationsHostRepute)},
		func(ctx context.Context, in *hostReputeSwapAssetInput) (*api.Body[HostReputeConnection], error) {
			out, err := s.SwapAsset(ctx, api.WorkspaceFrom(ctx), in.Body.Remove, in.Body.Add)
			return api.OK(out), err
		})
}
