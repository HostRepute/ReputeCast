package mcpconnect

import (
	"context"
	"errors"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/server"
	"github.com/hostrepute/reputecast/internal/store"
)

type MCPConsent struct {
	ClientName   string    `json:"clientName"`
	ClientID     string    `json:"clientId"`
	Verified     bool      `json:"verified"`
	CallbackHost string    `json:"callbackHost"`
	Scopes       []string  `json:"scopes"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

type MCPConsentDecision struct {
	Request       string      `json:"request" maxLength:"100"`
	Scopes        []string    `json:"scopes" maxItems:"3"`
	Lists         []uuid.UUID `json:"lists" maxItems:"100"`
	SenderDomains []string    `json:"senderDomains" maxItems:"100"`
	AllowedIPs    []string    `json:"allowedIps" maxItems:"20"`
}

type MCPConsentResult struct {
	Redirect string `json:"redirect"`
}

type MCPConsentList struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type MCPConsentOptions struct {
	Endpoint       string           `json:"endpoint"`
	Scopes         []string         `json:"scopes"`
	Lists          []MCPConsentList `json:"lists"`
	SenderDomains  []string         `json:"senderDomains"`
	DefaultLists   []uuid.UUID      `json:"defaultLists"`
	DefaultDomains []string         `json:"defaultDomains"`
}

// Options are for the signed-in person's consent form, never MCP/model context.
func (s *Service) ConsentOptions(ctx context.Context, ws *store.Workspace, p *authz.Principal) (MCPConsentOptions, error) {
	out := MCPConsentOptions{Scopes: []string{}, Lists: []MCPConsentList{}, SenderDomains: []string{}, DefaultLists: []uuid.UUID{}, DefaultDomains: []string{}}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		perms, scope, err := s.authority(ctx, q, ws.ID, p.UserID)
		if err != nil || !perms.Has(authz.MCPManage) {
			return api.Forbidden(authz.MCPManage, nil)
		}
		origin, err := s.origin()
		if err != nil {
			return err
		}
		out.Endpoint = origin + Path
		for _, capability := range capabilities {
			if canScope(capability, perms) {
				out.Scopes = append(out.Scopes, capability)
			}
		}
		lists, err := q.TeamWorkspaceLists(ctx, ws.ID)
		if err != nil {
			return err
		}
		for _, list := range lists {
			if scope.AllowsLists([]uuid.UUID{list.ID}) {
				out.Lists = append(out.Lists, MCPConsentList{ID: list.ID, Name: list.Name})
			}
		}
		domains, err := q.TeamWorkspaceSenderDomains(ctx, ws.ID)
		if err != nil {
			return err
		}
		for _, domain := range domains {
			if len(scope.SenderDomains) == 0 || slices.Contains(scope.SenderDomains, domain) {
				out.SenderDomains = append(out.SenderDomains, domain)
			}
		}
		out.DefaultLists = append(out.DefaultLists, scope.Lists...)
		out.DefaultDomains = append(out.DefaultDomains, scope.SenderDomains...)
		return nil
	})
	return out, err
}

func consentProblem(ctx context.Context) error {
	return api.Conflict(tr(ctx, "problem.consent"))
}

func (s *Service) consentRequest(ctx context.Context, q *store.Queries, raw string, bind bool) (store.McpConnectRequest, error) {
	p, session := api.PrincipalFrom(ctx), api.SessionFrom(ctx)
	if !p.IsHuman() || session == nil || !strings.HasPrefix(raw, "rc_oauth_request_") {
		return store.McpConnectRequest{}, consentProblem(ctx)
	}
	r, err := q.MCPConnectRequestLock(ctx, authn.HashToken(raw))
	if err != nil || r.Status != "pending" || !r.ExpiresAt.After(s.now()) || (!bind && r.SessionID == nil) || (r.UserID != nil && *r.UserID != p.UserID) || (r.SessionID != nil && *r.SessionID != session.ID) {
		return r, consentProblem(ctx)
	}
	current, err := q.MCPConnectSessionLock(ctx, session.ID)
	if err != nil || current.UserID != p.UserID || current.RevokedAt != nil || !current.ExpiresAt.After(s.now()) {
		return r, consentProblem(ctx)
	}
	origin, err := s.origin()
	if err != nil || r.Resource != origin+Path {
		return r, consentProblem(ctx)
	}
	if r.SessionID == nil {
		if err = q.MCPConnectBindRequest(ctx, store.MCPConnectBindRequestParams{HandleHash: r.HandleHash, UserID: &p.UserID, SessionID: &session.ID}); err != nil {
			return r, err
		}
	}
	return r, nil
}

func (s *Service) Consent(ctx context.Context, raw string) (MCPConsent, error) {
	var out MCPConsent
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := s.consentRequest(ctx, q, raw, true)
		if err != nil {
			return err
		}
		c, err := s.client(ctx, q, r.ClientID)
		if err != nil {
			return consentProblem(ctx)
		}
		u, err := callback(r.RedirectUri)
		if err != nil {
			return consentProblem(ctx)
		}
		out = MCPConsent{ClientName: c.Name, ClientID: c.ID, Verified: c.Verified, CallbackHost: u.Host, Scopes: r.Scopes, ExpiresAt: r.ExpiresAt}
		return nil
	})
	return out, err
}

func (s *Service) DecideConsent(ctx context.Context, ws *store.Workspace, in MCPConsentDecision) (MCPConsentResult, error) {
	var out MCPConsentResult
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := s.consentRequest(ctx, q, in.Request, false)
		if err != nil {
			return err
		}
		p := api.PrincipalFrom(ctx)
		perms, parent, err := s.authority(ctx, q, ws.ID, p.UserID)
		if err != nil {
			return api.Forbidden(authz.MCPManage, nil)
		}
		if !perms.Has(authz.MCPManage) {
			return api.Forbidden(authz.MCPManage, nil)
		}
		policy, err := q.MCPConnectPolicy(ctx)
		if err != nil {
			return err
		}
		if policy != "off" && !p.MFAVerified {
			return api.Err(403, api.CodeTwoFactorSetup, tr(ctx, "problem.mfa"))
		}
		c, err := s.client(ctx, q, r.ClientID)
		if err != nil || !matchesCallback(c.RedirectUris, r.RedirectUri) {
			return consentProblem(ctx)
		}
		origin, _ := s.origin()
		u, _ := url.Parse(r.RedirectUri)
		values := u.Query()
		values.Set("iss", origin)
		if r.State != "" {
			values.Set("state", r.State)
		}
		if len(in.Scopes) == 0 {
			return api.Invalid("body.scopes", tr(ctx, "problem.capability"))
		}
		chosen := []string{}
		for _, scope := range in.Scopes {
			if !slices.Contains(r.Scopes, scope) || slices.Contains(chosen, scope) || !canScope(scope, perms) {
				return api.Invalid("body.scopes", tr(ctx, "problem.capabilities"))
			}
			chosen = append(chosen, scope)
		}
		scope := authz.Scope{Lists: in.Lists, SenderDomains: in.SenderDomains}
		if !scope.Within(parent) {
			return api.Invalid("body.lists", tr(ctx, "problem.scope"))
		}
		lists, err := q.TeamWorkspaceLists(ctx, ws.ID)
		if err != nil {
			return err
		}
		for _, id := range in.Lists {
			if !slices.ContainsFunc(lists, func(l store.TeamWorkspaceListsRow) bool { return l.ID == id }) {
				return api.Invalid("body.lists", tr(ctx, "problem.lists"))
			}
		}
		domains, err := q.TeamWorkspaceSenderDomains(ctx, ws.ID)
		if err != nil {
			return err
		}
		for _, domain := range in.SenderDomains {
			if !slices.Contains(domains, domain) {
				return api.Invalid("body.senderDomains", tr(ctx, "problem.domains"))
			}
		}
		ips := []netip.Prefix{}
		for _, raw := range in.AllowedIPs {
			ip, ok := server.ParsePrefix(raw)
			if !ok {
				return api.Invalid("body.allowedIps", tr(ctx, "problem.ips"))
			}
			ips = append(ips, ip)
		}
		now := s.now()
		g, err := q.MCPConnectCreateGrant(ctx, store.MCPConnectCreateGrantParams{ID: uuid.Must(uuid.NewV7()), ClientID: c.ID, WorkspaceID: ws.ID, UserID: p.UserID, Resource: r.Resource, Scopes: chosen, Permissions: perms.Intersect(scopePermissions(chosen)).Strings(), Scope: access.ScopeJSON(scope), AllowedIps: ips, CreatedAt: now, ExpiresAt: now.Add(30 * 24 * time.Hour)})
		if err != nil {
			return err
		}
		code, hash := token(codePrefix)
		if err = q.MCPConnectCreateCode(ctx, store.MCPConnectCreateCodeParams{Hash: hash, GrantID: g.ID, RedirectUri: r.RedirectUri, Challenge: r.Challenge, ExpiresAt: now.Add(time.Minute)}); err != nil {
			return err
		}
		if err = s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "mcp_connect.granted", TargetType: "oauth_grant", TargetID: g.ID.String(), Metadata: map[string]any{"client_id": c.ID, "scopes": chosen, "via": "external_mcp"}}); err != nil {
			return err
		}
		values.Set("code", code)
		if err = q.MCPConnectDecideRequest(ctx, store.MCPConnectDecideRequestParams{HandleHash: r.HandleHash, Status: "allowed"}); err != nil {
			return err
		}
		u.RawQuery = values.Encode()
		out.Redirect = u.String()
		return nil
	})
	return out, err
}

// Denial needs no workspace, capability or MFA upgrade: it never grants access.
func (s *Service) DenyConsent(ctx context.Context, raw string) (MCPConsentResult, error) {
	var out MCPConsentResult
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		r, err := s.consentRequest(ctx, q, raw, false)
		if err != nil {
			return err
		}
		c, err := s.client(ctx, q, r.ClientID)
		if err != nil || !matchesCallback(c.RedirectUris, r.RedirectUri) {
			return consentProblem(ctx)
		}
		origin, _ := s.origin()
		u, _ := url.Parse(r.RedirectUri)
		values := u.Query()
		values.Set("iss", origin)
		values.Set("error", "access_denied")
		if r.State != "" {
			values.Set("state", r.State)
		}
		if err = q.MCPConnectDecideRequest(ctx, store.MCPConnectDecideRequestParams{HandleHash: r.HandleHash, Status: "denied"}); err != nil {
			return err
		}
		if err = s.Audit.LogWith(ctx, q, audit.Event{Action: "mcp_connect.denied", TargetType: "oauth_client", TargetID: r.ClientID}); err != nil {
			return err
		}
		u.RawQuery = values.Encode()
		out.Redirect = u.String()
		return nil
	})
	return out, err
}

type MCPConnection struct {
	ID            uuid.UUID   `json:"id"`
	ClientName    string      `json:"clientName"`
	Verified      bool        `json:"verified"`
	UserName      string      `json:"userName"`
	UserID        uuid.UUID   `json:"userId"`
	Scopes        []string    `json:"scopes"`
	Lists         []uuid.UUID `json:"lists"`
	SenderDomains []string    `json:"senderDomains"`
	AllowedIPs    []string    `json:"allowedIps"`
	CreatedAt     time.Time   `json:"createdAt"`
	ExpiresAt     time.Time   `json:"expiresAt"`
	RevokedAt     *time.Time  `json:"revokedAt,omitempty"`
	RevokedReason string      `json:"revokedReason"`
}
type MCPConnectionList struct {
	Items      []MCPConnection `json:"items"`
	NextCursor string          `json:"nextCursor,omitempty"`
}

func (s *Service) Connections(ctx context.Context, ws *store.Workspace, p *authz.Principal, cursor string) (MCPConnectionList, error) {
	out := MCPConnectionList{Items: []MCPConnection{}}
	var before *uuid.UUID
	if cursor != "" {
		id, err := uuid.Parse(cursor)
		if err != nil {
			return out, api.Invalid("cursor", tr(ctx, "problem.cursor"))
		}
		before = &id
	}
	rows, err := s.Q.MCPConnectGrants(ctx, store.MCPConnectGrantsParams{WorkspaceID: ws.ID, UserID: p.UserID, Manager: p.Can(authz.MCPManage), BeforeID: before})
	if err != nil {
		return out, err
	}
	if len(rows) > 50 {
		rows = rows[:50]
		out.NextCursor = rows[49].ID.String()
	}
	for _, r := range rows {
		out.Items = append(out.Items, connectionView(store.MCPConnectConnectionRow(r)))
	}
	return out, nil
}

func connectionView(r store.MCPConnectConnectionRow) MCPConnection {
	scope := access.ParseScope(r.Scope)
	ips := []string{}
	for _, ip := range r.AllowedIps {
		ips = append(ips, ip.String())
	}
	return MCPConnection{ID: r.ID, ClientName: r.ClientName, Verified: r.Verified, UserName: r.UserName, UserID: r.UserID, Scopes: r.Scopes, Lists: append([]uuid.UUID{}, scope.Lists...), SenderDomains: append([]string{}, scope.SenderDomains...), AllowedIPs: ips, CreatedAt: r.CreatedAt, ExpiresAt: r.ExpiresAt, RevokedAt: r.RevokedAt, RevokedReason: r.RevokedReason}
}

func (s *Service) Connection(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (MCPConnection, error) {
	r, err := s.Q.MCPConnectConnection(ctx, store.MCPConnectConnectionParams{ID: id, WorkspaceID: ws.ID, UserID: p.UserID, Manager: p.Can(authz.MCPManage)})
	if err != nil {
		return MCPConnection{}, api.MapError(err, "Connection")
	}
	return connectionView(r), nil
}

func (s *Service) RevokeConnection(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		g, err := q.MCPConnectGrantLock(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) || g.WorkspaceID != ws.ID {
			return api.NotFound("Connection")
		}
		if err != nil {
			return err
		}
		r, err := s.membership(ctx, q, ws.ID, p.UserID)
		if err != nil {
			return api.NotFound("Connection")
		}
		if !authz.Effective(r.RoleKey, r.Builtin, r.Permissions).Has(authz.MCPManage) && g.UserID != p.UserID {
			return api.NotFound("Connection")
		}
		return s.revoke(ctx, q, g, "human_revoked")
	})
}
