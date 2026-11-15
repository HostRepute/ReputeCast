package mcpconnect

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type consentInput struct {
	Request string `query:"request" required:"true" maxLength:"100"`
}
type consentDecisionInput struct {
	api.WorkspaceInput
	Body MCPConsentDecision
}
type consentDenyInput struct {
	Body struct {
		Request string `json:"request" maxLength:"100"`
	}
}
type connectionListInput struct {
	api.WorkspaceInput
	Cursor string `query:"cursor" maxLength:"36"`
}
type connectionPath struct {
	api.WorkspaceInput
	ID uuid.UUID `path:"id"`
}
type sourceSelectInput struct {
	api.WorkspaceInput
	ID   uuid.UUID `path:"id"`
	Body MCPSourceSelection
}
type proposalDecisionInput struct {
	api.WorkspaceInput
	ID   uuid.UUID `path:"id"`
	Body MCPProposalDecision
}
type preregisterInput struct{ Body ClientRegistration }
type consentResponse struct {
	CacheControl   string `header:"Cache-Control"`
	ReferrerPolicy string `header:"Referrer-Policy"`
	Body           MCPConsent
}
type decisionResponse struct {
	CacheControl   string `header:"Cache-Control"`
	ReferrerPolicy string `header:"Referrer-Policy"`
	Body           MCPConsentResult
}
type registrationResponse struct {
	CacheControl string `header:"Cache-Control"`
	Body         RegisteredClient
}

// Keep opaque handles and client credentials out of validation responses,
// including Huma's errors raised before a domain handler runs.
type privateAPI struct{ huma.API }

func (h privateAPI) Transform(ctx huma.Context, status string, value any) (any, error) {
	if p, ok := value.(*api.Problem); ok && p.Code == api.CodeValidation {
		value = api.Err(p.Status, p.Code, tr(ctx.Context(), "problem.fields"))
	}
	return h.API.Transform(ctx, status, value)
}

func (h privateAPI) Middlewares() huma.Middlewares {
	return append(huma.Middlewares{func(ctx huma.Context, next func(huma.Context)) {
		ctx.SetHeader("Cache-Control", "no-store")
		ctx.SetHeader("Referrer-Policy", "no-referrer")
		next(ctx)
	}}, h.API.Middlewares()...)
}

func Register(h huma.API, s *Service) {
	h = privateAPI{h}
	tags := []string{"External MCP"}
	api.Get(h, api.Op{ID: "get-mcp-consent", Path: "/mcp-connect/consent", Summary: "Review and bind an external connection request to this session", Tags: tags, Access: api.Access{Kind: api.AccessUser, HumansOnly: true, AllowPending: true}}, func(ctx context.Context, in *consentInput) (*consentResponse, error) {
		out, err := s.Consent(ctx, in.Request)
		return &consentResponse{CacheControl: "no-store", ReferrerPolicy: "no-referrer", Body: out}, err
	})
	api.Post(h, api.Op{ID: "deny-mcp-consent", Path: "/mcp-connect/consent/deny", Summary: "Deny a connection without granting workspace access", Tags: tags, Access: api.Access{Kind: api.AccessUser, HumansOnly: true, AllowPending: true}}, func(ctx context.Context, in *consentDenyInput) (*decisionResponse, error) {
		out, err := s.DenyConsent(ctx, in.Body.Request)
		return &decisionResponse{CacheControl: "no-store", ReferrerPolicy: "no-referrer", Body: out}, err
	})
	api.Get(h, api.Op{ID: "get-mcp-consent-options", Path: "/workspaces/{workspace}/mcp-connect/options", Summary: "Read the capabilities and resource scope this person may grant", Tags: tags, Access: api.WorkspaceHuman(authz.MCPManage)}, func(ctx context.Context, in *api.WorkspaceInput) (*api.Body[MCPConsentOptions], error) {
		out, err := s.ConsentOptions(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
		return api.OK(out), err
	})
	api.Post(h, api.Op{ID: "decide-mcp-consent", Path: "/workspaces/{workspace}/mcp-connect/consent", Summary: "Allow selected external capabilities in one workspace", Tags: tags, Access: api.WorkspaceHuman(authz.MCPManage)}, func(ctx context.Context, in *consentDecisionInput) (*decisionResponse, error) {
		out, err := s.DecideConsent(ctx, api.WorkspaceFrom(ctx), in.Body)
		return &decisionResponse{CacheControl: "no-store", ReferrerPolicy: "no-referrer", Body: out}, err
	})
	api.Get(h, api.Op{ID: "list-mcp-connections", Path: "/workspaces/{workspace}/mcp-connect/connections", Summary: "List own external connections, or permitted connections managed by this member", Tags: tags, Access: api.WorkspaceHuman("")}, func(ctx context.Context, in *connectionListInput) (*api.Body[MCPConnectionList], error) {
		out, err := s.Connections(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Cursor)
		return api.OK(out), err
	})
	api.Get(h, api.Op{ID: "get-mcp-connection", Path: "/workspaces/{workspace}/mcp-connect/connections/{id}", Summary: "Check saved connection status without repeating a decision", Tags: tags, Access: api.WorkspaceHuman("")}, func(ctx context.Context, in *connectionPath) (*api.Body[MCPConnection], error) {
		out, err := s.Connection(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID)
		return api.OK(out), err
	})
	api.Delete(h, api.Op{ID: "revoke-mcp-connection", Path: "/workspaces/{workspace}/mcp-connect/connections/{id}", Summary: "Revoke an external connection without deleting accepted drafts", Tags: tags, Access: api.WorkspaceHuman("")}, func(ctx context.Context, in *connectionPath) (*api.Empty, error) {
		return api.NoContent, s.RevokeConnection(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID)
	})
	api.Get(h, api.Op{ID: "list-mcp-selected-sources", Path: "/workspaces/{workspace}/mcp-connect/connections/{id}/sources", Summary: "Check saved source selections without sharing another resource", Tags: tags, Access: api.WorkspaceHuman("")}, func(ctx context.Context, in *connectionPath) (*api.Body[MCPSelectedSources], error) {
		out, err := s.SelectedSources(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID)
		return api.OK(out), err
	})
	api.Post(h, api.Op{ID: "select-mcp-source", Path: "/workspaces/{workspace}/mcp-connect/connections/{id}/sources", Summary: "Select one editorial source for this external connection", Tags: tags, Access: api.WorkspaceHuman("")}, func(ctx context.Context, in *sourceSelectInput) (*api.Body[MCPSourceHandle], error) {
		out, err := s.SelectSource(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
		return api.OK(out), err
	})
	api.Get(h, api.Op{ID: "list-mcp-proposals", Path: "/workspaces/{workspace}/mcp-connect/proposals", Summary: "List permitted external proposals for human review", Tags: tags, Access: api.WorkspaceHuman("")}, func(ctx context.Context, in *connectionListInput) (*api.Body[MCPProposalList], error) {
		out, err := s.Proposals(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Cursor)
		return api.OK(out), err
	})
	api.Get(h, api.Op{ID: "get-mcp-proposal", Path: "/workspaces/{workspace}/mcp-connect/proposals/{id}", Summary: "Review an external proposal without applying it", Tags: tags, Access: api.WorkspaceHuman("")}, func(ctx context.Context, in *connectionPath) (*api.Body[MCPProposalView], error) {
		out, err := s.Proposal(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID)
		return api.OK(out), err
	})
	api.Post(h, api.Op{ID: "decide-mcp-proposal", Path: "/workspaces/{workspace}/mcp-connect/proposals/{id}/decision", Summary: "Apply or reject an immutable external proposal as a person", Tags: tags, Access: api.WorkspaceHuman("")}, func(ctx context.Context, in *proposalDecisionInput) (*api.Empty, error) {
		return api.NoContent, s.DecideProposal(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
	})
	api.Post(h, api.Op{ID: "register-verified-mcp-client", Path: "/admin/mcp-connect/clients", Summary: "Pre-register an operator-verified OAuth client and disclose its secret once", Tags: tags, Access: api.InstanceAdmin}, func(ctx context.Context, in *preregisterInput) (*registrationResponse, error) {
		out, err := s.RegisterClient(ctx, in.Body, true)
		if errors.Is(err, errRequest) {
			err = api.Invalid("body", tr(ctx, "problem.client"))
		}
		return &registrationResponse{CacheControl: "no-store", Body: out}, err
	})
}
