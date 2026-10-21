package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/time/rate"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/apikeys"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/version"
)

// Path is where the MCP endpoint is mounted.
const Path = "/mcp"

// agentCall is what the token check learns about a request, carried to the
// tools in the token info.
type agentCall struct {
	Principal *authz.Principal
	Workspace store.Workspace
	Masked    string
	Request   audit.Request
	Settings  Settings
}

const callKey = "reputecast.call"

type limiter struct {
	lim    *rate.Limiter
	perMin int
}

func (s *Service) allow(agent uuid.UUID, perMin int) bool {
	v, _ := s.limiters.Load(agent)
	l, _ := v.(*limiter)
	if l == nil || l.perMin != perMin {
		l = &limiter{lim: rate.NewLimiter(rate.Limit(float64(perMin)/60), perMin), perMin: perMin}
		s.limiters.Store(agent, l)
	}
	return l.lim.Allow()
}

// verify checks an agent token: known, an MCP agent, not revoked or expired,
// MCP on in its workspace.
func (s *Service) verify(ctx context.Context, token string, r *http.Request) (*auth.TokenInfo, error) {
	req := audit.RequestFrom(ctx)
	if !strings.HasPrefix(token, authn.AgentPrefix) {
		return nil, fmt.Errorf("%w: use an MCP agent token (rc_agent_…) from Settings › Keys & MCP", auth.ErrInvalidToken)
	}
	k, err := s.Q.KeysByToken(ctx, authn.HashToken(token))
	if err != nil || k.Kind != "mcp_agent" {
		return nil, fmt.Errorf("%w: unknown agent token", auth.ErrInvalidToken)
	}
	switch {
	case k.RevokedAt != nil:
		apikeys.RecordRejection(ctx, s.Q, k.ID, req.IP, r.UserAgent(), "revoked")
		return nil, fmt.Errorf("%w: this agent was revoked", auth.ErrInvalidToken)
	case k.ExpiresAt != nil && !k.ExpiresAt.After(s.now()):
		apikeys.RecordRejection(ctx, s.Q, k.ID, req.IP, r.UserAgent(), "expired")
		return nil, fmt.Errorf("%w: this agent token expired on %s; rotate it in Settings › Keys & MCP", auth.ErrInvalidToken, k.ExpiresAt.Format("Jan 2, 2006"))
	}
	st := ParseSettings(k.McpSettings)
	if !st.Enabled {
		apikeys.RecordRejection(ctx, s.Q, k.ID, req.IP, r.UserAgent(), "disabled")
		return nil, fmt.Errorf("%w: the MCP server is off in %s", auth.ErrInvalidToken, k.WorkspaceName)
	}
	// An agent never does more than its creator can do right now: nothing
	// once they have left or been deactivated in the workspace (as for API
	// keys in api.Authenticator).
	perms := authz.ParseSet(k.Permissions)
	if k.CreatedBy == nil {
		perms = authz.Set{}
	} else if m, err := s.Q.GetMembership(ctx, store.GetMembershipParams{WorkspaceID: k.WorkspaceID, UserID: *k.CreatedBy}); err == nil && m.Status == "active" {
		perms = perms.Intersect(authz.Effective(m.RoleKey, m.RoleBuiltin, m.RolePermissions))
	} else {
		perms = authz.Set{}
	}
	ws, err := s.Q.GetWorkspace(ctx, k.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("%w: workspace not found", auth.ErrInvalidToken)
	}
	creator := uuid.Nil
	if k.CreatedBy != nil {
		creator = *k.CreatedBy
	}
	masked := k.Prefix
	p := &authz.Principal{Type: authz.ActorMCPAgent, ID: k.ID, Name: k.Name, UserID: creator, WorkspaceID: k.WorkspaceID,
		Permissions: perms.HoldableBy(authz.ActorMCPAgent), Scope: access.ParseScope(k.Scope), MFAVerified: true}
	return &auth.TokenInfo{
		UserID: k.ID.String(),
		Extra:  map[string]any{callKey: &agentCall{Principal: p, Workspace: ws, Masked: masked, Request: req, Settings: st}},
	}, nil
}

// Handler serves the MCP endpoint: bearer-token check, a per-agent rate
// limit, then the streamable HTTP transport.
func (s *Service) Handler(log *slog.Logger) http.Handler {
	server := s.newServer(log)
	stream := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{
		Logger:         log,
		SessionTimeout: 30 * time.Minute,
		// The instance runs behind its public URL; the Host header check for
		// loopback listeners would reject requests proxied to 127.0.0.1.
		DisableLocalhostProtection: true,
	})
	limited := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ti := auth.TokenInfoFromContext(r.Context()); ti != nil {
			if c, ok := ti.Extra[callKey].(*agentCall); ok && !s.allow(c.Principal.ID, c.Settings.RateLimitPerMinute) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, fmt.Sprintf("Rate limit reached: %d calls per minute per agent. Calls resume within a minute.", c.Settings.RateLimitPerMinute), http.StatusTooManyRequests)
				return
			}
		}
		stream.ServeHTTP(w, r)
	})
	return auth.RequireBearerToken(s.verify, &auth.RequireBearerTokenOptions{AllowMissingExpiration: true})(limited)
}

func (s *Service) newServer(log *slog.Logger) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "reputecast", Title: "ReputeCast", Version: version.Get().Version}, &mcp.ServerOptions{
		Instructions: "ReputeCast is a self-hosted newsletter platform. Tools act inside one workspace with the permissions of this agent; " +
			"anything that would send mail is proposed to a team member instead of running. Campaigns can be referenced by name or id.",
		Logger:       log,
		Capabilities: &mcp.ServerCapabilities{},
	})
	s.addTools(server)
	s.addAccessibilityTools(server)
	s.addGrowthTools(server)
	s.addDigestTools(server)
	s.addPollTools(server)
	s.addEarnTools(server)
	s.addPlacementTools(server)
	s.addWebsiteTools(server)
	s.addPagesTools(server)
	s.addX17Tools(server)
	s.addGrowTools(server)
	s.addResources(server)
	s.addPrompts(server)
	return server
}

// call returns the agent behind a request, with a context that carries the
// principal for audit entries.
func call(ctx context.Context, extra *mcp.RequestExtra) (context.Context, *agentCall, error) {
	if extra == nil || extra.TokenInfo == nil {
		return ctx, nil, errors.New("not authenticated")
	}
	c, ok := extra.TokenInfo.Extra[callKey].(*agentCall)
	if !ok {
		return ctx, nil, errors.New("not authenticated")
	}
	ctx = audit.WithRequest(ctx, c.Request)
	ctx = api.WithPrincipal(ctx, c.Principal)
	return ctx, c, nil
}

// entry is one audit entry for a tool call.
type entry struct {
	action, targetType, targetID, targetName, detail, summary string
	before, after                                             any
}

// record writes the audit entry of a call and touches the agent's last use.
func (s *Service) record(ctx context.Context, c *agentCall, tool string, perm authz.Permission, e entry, result, reason, note string) {
	meta := map[string]any{"tool": tool, "source": "mcp", "via": c.Masked, "permission": string(perm),
		"summary": fmt.Sprintf("MCP agent “%s” · tool %s", c.Principal.Name, tool)}
	if e.detail != "" {
		meta["detail"] = e.detail
	}
	// outcome is what the call did, shown in the agents' recent actions.
	if e.summary != "" {
		meta["outcome"] = e.summary
	}
	if note != "" {
		meta["note"] = note
	}
	wsID := c.Workspace.ID
	_ = s.Audit.Log(ctx, audit.Event{WorkspaceID: &wsID, Action: e.action, TargetType: e.targetType, TargetID: e.targetID, TargetName: e.targetName,
		Result: result, Reason: reason, Before: e.before, After: e.after, Metadata: meta})
	_ = s.Q.KeysTouch(ctx, store.KeysTouchParams{ID: c.Principal.ID, Ip: c.Request.IP, Client: apikeys.ClientLabel(c.Request.UserAgent), LastCall: tool})
}

// deny records a denied call, proposes it to the members who hold the
// permission and returns the error the agent sees.
func (s *Service) deny(ctx context.Context, c *agentCall, tool string, perm authz.Permission, e entry) error {
	helpers := access.Helpers(ctx, s.Q, c.Workspace.ID, perm)
	note := "needs " + string(perm)
	msg := fmt.Sprintf("Denied: %s needs the %s permission, which this agent doesn't hold.", tool, perm)
	if len(helpers) > 0 {
		note += " · proposal recorded for " + access.OrList(helpers[:1])
		msg += fmt.Sprintf(" A proposal was recorded for %s to review in ReputeCast.", access.OrList(helpers[:1]))
		s.propose(ctx, c, tool, perm, e)
	}
	s.record(ctx, c, tool, perm, e, audit.Denied, "missing "+string(perm), note)
	return errors.New(msg)
}

func ref(r string) string { return strings.TrimSpace(r) }
