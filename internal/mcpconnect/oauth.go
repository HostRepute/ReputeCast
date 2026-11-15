// Package mcpconnect serves the separately consented, draft-only external MCP
// connection. It has no reference to the in-app AI service or sending engine.
package mcpconnect

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/access"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/server"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

const (
	Path          = "/mcp/connect"
	accessPrefix  = "rc_oauth_access_"
	refreshPrefix = "rc_oauth_refresh_"
	codePrefix    = "rc_oauth_code_"
)

var capabilities = []string{"editorial:read", "reports:read", "drafts:propose"}
var errGrant = errors.New("invalid_grant")
var errClient = errors.New("invalid_client")
var errRequest = errors.New("invalid_request")

type Service struct {
	Pool        *pgxpool.Pool
	Q           *store.Queries
	Audit       *audit.Logger
	PublicURL   func() string
	Development bool
	Now         func() time.Time
	Campaigns   *campaigns.Service
	Website     *website.Service
	Automations *automations.Service
	Segments    *segments.Service
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// origin never trusts the request's Host or forwarding headers.
func (s *Service) origin() (string, error) {
	u, err := url.Parse(s.PublicURL())
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", errRequest
	}
	switch u.Scheme {
	case "https":
	case "http":
		if !s.Development || (u.Hostname() != "localhost" && !loopback(u.Hostname())) {
			return "", errRequest
		}
	default:
		return "", errRequest
	}
	return u.Scheme + "://" + u.Host, nil
}

func loopback(host string) bool {
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.IsLoopback() && ip.Zone() == ""
}

func token(prefix string) (string, []byte) {
	raw, _ := authn.NewToken()
	raw = prefix + raw
	return raw, authn.HashToken(raw)
}

func callback(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || len(raw) > 2048 || u.Host == "" || u.Opaque != "" || u.User != nil || u.Fragment != "" || strings.ContainsAny(raw, "\\#") || strings.Contains(u.Host, "%") {
		return nil, errRequest
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil || strings.Contains(decoded, "\\") || strings.ContainsFunc(decoded, unicode.IsControl) || strings.ContainsAny(u.Host, "* ") {
		return nil, errRequest
	}
	switch u.Scheme {
	case "https":
	case "http":
		if !loopback(u.Hostname()) {
			return nil, errRequest
		}
	default:
		return nil, errRequest
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, errRequest
	}
	for _, key := range []string{"code", "state", "error", "iss"} {
		if _, ok := q[key]; ok {
			return nil, errRequest
		}
	}
	return u, nil
}

func matchesCallback(registered []string, raw string) bool {
	u, err := callback(raw)
	if err != nil {
		return false
	}
	for _, value := range registered {
		if value == raw {
			return true
		}
		v, err := callback(value)
		if err == nil && u.Scheme == "http" && v.Scheme == "http" && loopback(u.Hostname()) && u.Hostname() == v.Hostname() && u.EscapedPath() == v.EscapedPath() && u.RawQuery == v.RawQuery {
			return true
		}
	}
	return false
}

type ClientRegistration struct {
	Name          string   `json:"client_name,omitempty" maxLength:"120"`
	RedirectURIs  []string `json:"redirect_uris" minItems:"1" maxItems:"5"`
	AuthMethod    string   `json:"token_endpoint_auth_method,omitempty" enum:"none,client_secret_basic,client_secret_post"`
	GrantTypes    []string `json:"grant_types,omitempty"`
	ResponseTypes []string `json:"response_types,omitempty"`
}

type RegisteredClient struct {
	ClientRegistration
	ID              string `json:"client_id"`
	Secret          string `json:"client_secret,omitempty"`
	IssuedAt        int64  `json:"client_id_issued_at"`
	SecretExpiresAt int64  `json:"client_secret_expires_at"`
}

func (s *Service) RegisterClient(ctx context.Context, in ClientRegistration, verified bool) (RegisteredClient, error) {
	if in.Name == "" {
		in.Name = "Unnamed client"
	}
	if len(in.Name) > 120 || strings.ContainsFunc(in.Name, unicode.IsControl) || len(in.RedirectURIs) < 1 || len(in.RedirectURIs) > 5 {
		return RegisteredClient{}, errRequest
	}
	for _, raw := range in.RedirectURIs {
		if _, err := callback(raw); err != nil {
			return RegisteredClient{}, err
		}
	}
	if in.AuthMethod == "" {
		in.AuthMethod = "client_secret_basic"
	}
	if !slices.Contains([]string{"none", "client_secret_basic", "client_secret_post"}, in.AuthMethod) {
		return RegisteredClient{}, errRequest
	}
	for _, g := range in.GrantTypes {
		if g != "authorization_code" && g != "refresh_token" {
			return RegisteredClient{}, errRequest
		}
	}
	for _, r := range in.ResponseTypes {
		if r != "code" {
			return RegisteredClient{}, errRequest
		}
	}
	in.GrantTypes, in.ResponseTypes = []string{"authorization_code", "refresh_token"}, []string{"code"}
	id, _ := token("rc_oauth_client_")
	out := RegisteredClient{ClientRegistration: in, ID: id, IssuedAt: s.now().Unix()}
	var hash []byte
	if in.AuthMethod != "none" {
		out.Secret, hash = token("rc_oauth_secret_")
	}
	expires := s.now().Add(7 * 24 * time.Hour)
	var deadline *time.Time
	if !verified {
		deadline = &expires
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		_, err := q.MCPConnectCreateClient(ctx, store.MCPConnectCreateClientParams{ID: id, Name: in.Name, RedirectUris: in.RedirectURIs, AuthMethod: in.AuthMethod, SecretHash: hash, Verified: verified, ExpiresAt: deadline})
		if err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{Action: "mcp_connect.client_registered", TargetType: "oauth_client", TargetID: id, Metadata: map[string]any{"verified": verified, "auth_method": in.AuthMethod}})
	})
	return out, err
}

func (s *Service) client(ctx context.Context, q *store.Queries, id string) (store.McpConnectClient, error) {
	c, err := q.MCPConnectClient(ctx, id)
	if err != nil {
		return c, errClient
	}
	if c.ExpiresAt != nil && !c.ExpiresAt.After(s.now()) {
		used, err := q.MCPConnectClientInUse(ctx, store.MCPConnectClientInUseParams{ClientID: id, Now: s.now()})
		if err != nil || !used {
			return c, errClient
		}
	}
	return c, nil
}

func scopes(raw string) ([]string, error) {
	out := []string{}
	for _, value := range strings.Split(raw, " ") {
		if !slices.Contains(capabilities, value) || slices.Contains(out, value) {
			return nil, errors.New("invalid_scope")
		}
		out = append(out, value)
	}
	return out, nil
}

func validVerifier(value string) bool {
	if len(value) < 43 || len(value) > 128 {
		return false
	}
	return !strings.ContainsFunc(value, func(c rune) bool {
		return !strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~", c)
	})
}

func (s *Service) Authorize(ctx context.Context, in url.Values) (string, error) {
	origin, err := s.origin()
	if err != nil {
		return "", err
	}
	c, err := s.client(ctx, s.Q, in.Get("client_id"))
	if err != nil {
		return "", err
	}
	if !matchesCallback(c.RedirectUris, in.Get("redirect_uri")) || in.Get("response_type") != "code" || in.Get("resource") != origin+Path || in.Get("code_challenge_method") != "S256" || len(in.Get("state")) > 2048 {
		return "", errRequest
	}
	challenge, err := base64.RawURLEncoding.DecodeString(in.Get("code_challenge"))
	if err != nil || len(challenge) != 32 || base64.RawURLEncoding.EncodeToString(challenge) != in.Get("code_challenge") {
		return "", errRequest
	}
	requested, err := scopes(in.Get("scope"))
	if err != nil {
		return "", err
	}
	handle, hash := token("rc_oauth_request_")
	err = s.Q.MCPConnectCreateRequest(ctx, store.MCPConnectCreateRequestParams{HandleHash: hash, ClientID: c.ID, RedirectUri: in.Get("redirect_uri"), Resource: origin + Path, State: in.Get("state"), Challenge: in.Get("code_challenge"), Scopes: requested, ExpiresAt: s.now().Add(5 * time.Minute)})
	return origin + "/mcp-consent?request=" + url.QueryEscape(handle), err
}

// membership fences current authority against revocation until commit. It also
// permits recovery actions when the workspace has disabled MCP connections.
// Database errors, such as a lock wait, are returned as they are: they say
// nothing about the person's access.
func (s *Service) membership(ctx context.Context, q *store.Queries, ws, user uuid.UUID) (store.MCPConnectAuthorityRow, error) {
	r, err := q.MCPConnectAuthority(ctx, store.MCPConnectAuthorityParams{WorkspaceID: ws, UserID: user})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return r, err
	}
	if err != nil || r.MemberStatus != "active" || r.UserStatus != "active" || r.MustChangePassword || r.DeletedAt != nil {
		return r, errGrant
	}
	return r, nil
}

func (s *Service) authority(ctx context.Context, q *store.Queries, ws, user uuid.UUID) (authz.Set, authz.Scope, error) {
	r, err := s.membership(ctx, q, ws, user)
	if err != nil {
		return nil, authz.Scope{}, err
	}
	settings := struct {
		Enabled bool `json:"enabled"`
	}{Enabled: true}
	if err := json.Unmarshal(r.McpSettings, &settings); err != nil || !settings.Enabled {
		return nil, authz.Scope{}, errGrant
	}
	return authz.Effective(r.RoleKey, r.Builtin, r.Permissions), access.ParseScope(r.Scope), nil
}

func scopePermissions(scopes []string) authz.Set {
	perms := authz.Set{}
	for _, scope := range scopes {
		var keys []authz.Permission
		switch scope {
		case "editorial:read":
			keys = []authz.Permission{authz.CampaignsView, authz.TemplatesView, authz.AutomationsView, authz.WebsiteView, authz.SubscribersView}
		case "reports:read":
			keys = []authz.Permission{authz.ReportsView, authz.CampaignsView}
		case "drafts:propose":
			keys = []authz.Permission{authz.CampaignsView, authz.CampaignsEdit, authz.AutomationsView, authz.AutomationsEdit, authz.WebsiteView, authz.WebsiteEdit, authz.SubscribersView, authz.SegmentsEdit}
		}
		for _, key := range keys {
			perms[key] = struct{}{}
		}
	}
	return perms.HoldableBy(authz.ActorMCPAgent)
}

func canScope(scope string, perms authz.Set) bool {
	switch scope {
	case "editorial:read":
		return perms.Has(authz.CampaignsView) || perms.Has(authz.TemplatesView) || perms.Has(authz.AutomationsView) || perms.Has(authz.WebsiteView) || perms.Has(authz.SubscribersView)
	case "reports:read":
		return perms.HasAll(authz.ReportsView, authz.CampaignsView)
	case "drafts:propose":
		return perms.HasAll(authz.CampaignsView, authz.CampaignsEdit) || perms.HasAll(authz.AutomationsView, authz.AutomationsEdit) || perms.HasAll(authz.WebsiteView, authz.WebsiteEdit) || perms.HasAll(authz.SubscribersView, authz.SegmentsEdit)
	}
	return false
}

func (s *Service) grantPrincipal(ctx context.Context, q *store.Queries, g store.McpConnectGrant, ip *netip.Addr) (*authz.Principal, error) {
	if !server.AllowedFrom(g.AllowedIps, ip) {
		return nil, errGrant
	}
	return s.grantAuthority(ctx, q, g)
}

// Human review checks the grant's authority, not the client's egress IP.
func (s *Service) grantAuthority(ctx context.Context, q *store.Queries, g store.McpConnectGrant) (*authz.Principal, error) {
	origin, err := s.origin()
	if err != nil || g.Resource != origin+Path || g.RevokedAt != nil || !g.ExpiresAt.After(s.now()) {
		return nil, errGrant
	}
	perms, scope, err := s.authority(ctx, q, g.WorkspaceID, g.UserID)
	if err != nil {
		return nil, err
	}
	return &authz.Principal{Type: authz.ActorMCPAgent, ID: g.ID, UserID: g.UserID, WorkspaceID: g.WorkspaceID, Name: "External MCP connection", Permissions: perms.Intersect(authz.ParseSet(g.Permissions)).Intersect(scopePermissions(g.Scopes)), Scope: scope.Intersect(access.ParseScope(g.Scope)), MFAVerified: true}, nil
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
}

func (s *Service) issue(ctx context.Context, q *store.Queries, g store.McpConnectGrant) (TokenResponse, error) {
	accessToken, accessHash := token(accessPrefix)
	refreshToken, refreshHash := token(refreshPrefix)
	now := s.now()
	expires := now.Add(10 * time.Minute)
	if g.ExpiresAt.Before(expires) {
		expires = g.ExpiresAt
	}
	if err := q.MCPConnectCreateToken(ctx, store.MCPConnectCreateTokenParams{Hash: accessHash, GrantID: g.ID, Kind: "access", ExpiresAt: expires}); err != nil {
		return TokenResponse{}, err
	}
	if err := q.MCPConnectCreateToken(ctx, store.MCPConnectCreateTokenParams{Hash: refreshHash, GrantID: g.ID, Kind: "refresh", ExpiresAt: g.ExpiresAt}); err != nil {
		return TokenResponse{}, err
	}
	return TokenResponse{AccessToken: accessToken, RefreshToken: refreshToken, TokenType: "Bearer", ExpiresIn: int64(expires.Sub(now).Seconds()), Scope: strings.Join(g.Scopes, " ")}, nil
}

func (s *Service) Exchange(ctx context.Context, client store.McpConnectClient, in url.Values, ip *netip.Addr) (TokenResponse, error) {
	var out TokenResponse
	var replay bool
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		var id uuid.UUID
		var code store.McpConnectCode
		var refresh store.McpConnectToken
		var err error
		switch in.Get("grant_type") {
		case "authorization_code":
			if !strings.HasPrefix(in.Get("code"), codePrefix) || !validVerifier(in.Get("code_verifier")) {
				return errGrant
			}
			code, err = q.MCPConnectCode(ctx, authn.HashToken(in.Get("code")))
			id = code.GrantID
		case "refresh_token":
			if !strings.HasPrefix(in.Get("refresh_token"), refreshPrefix) {
				return errGrant
			}
			refresh, err = q.MCPConnectToken(ctx, authn.HashToken(in.Get("refresh_token")))
			id = refresh.GrantID
			if refresh.Kind != "refresh" {
				return errGrant
			}
		default:
			return errors.New("unsupported_grant_type")
		}
		if err != nil {
			return errGrant
		}
		g, err := q.MCPConnectGrantLock(ctx, id)
		if err != nil || g.ClientID != client.ID || in.Get("resource") != g.Resource {
			return errGrant
		}
		if _, err = s.grantPrincipal(ctx, q, g, ip); err != nil {
			return err
		}
		if in.Get("scope") != "" && in.Get("scope") != strings.Join(g.Scopes, " ") {
			return errors.New("invalid_scope")
		}
		now := s.now()
		if in.Get("grant_type") == "authorization_code" {
			sum := sha256.Sum256([]byte(in.Get("code_verifier")))
			challenge := base64.RawURLEncoding.EncodeToString(sum[:])
			if code.RedirectUri != in.Get("redirect_uri") || !authn.EqualHash([]byte(challenge), []byte(code.Challenge)) {
				return errGrant
			}
			if _, err = q.MCPConnectUseCode(ctx, store.MCPConnectUseCodeParams{Hash: code.Hash, Now: &now}); err != nil {
				return errGrant
			}
		} else {
			// Re-read under the family lock: the initial lookup may predate a
			// competing refresh's commit on another process.
			refresh, err = q.MCPConnectToken(ctx, refresh.Hash)
			if err != nil || !refresh.ExpiresAt.After(s.now()) {
				return errGrant
			}
			if refresh.UsedAt != nil {
				replay = true
				return s.revoke(ctx, q, g, "refresh_reuse")
			}
			if _, err = q.MCPConnectUseRefresh(ctx, store.MCPConnectUseRefreshParams{Hash: refresh.Hash, Now: &now}); err != nil {
				return errGrant
			}
		}
		out, err = s.issue(ctx, q, g)
		return err
	})
	if replay && err == nil {
		return TokenResponse{}, errGrant
	}
	return out, err
}

func (s *Service) revoke(ctx context.Context, q *store.Queries, g store.McpConnectGrant, reason string) error {
	if g.RevokedAt != nil {
		return nil
	}
	now := s.now()
	if err := q.MCPConnectRevoke(ctx, store.MCPConnectRevokeParams{ID: g.ID, Now: &now, Reason: reason}); err != nil {
		return err
	}
	if audit.PrincipalFrom(ctx) == nil {
		ctx = audit.WithPrincipal(ctx, &authz.Principal{Type: authz.ActorMCPAgent, ID: g.ID, UserID: g.UserID, WorkspaceID: g.WorkspaceID, Name: "External MCP connection"})
	}
	return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &g.WorkspaceID, Action: "mcp_connect.revoked", TargetType: "oauth_grant", TargetID: g.ID.String(), Metadata: map[string]any{"reason": reason, "via": "external_mcp"}})
}

func (s *Service) RevokeToken(ctx context.Context, client store.McpConnectClient, raw string) error {
	if !strings.HasPrefix(raw, accessPrefix) && !strings.HasPrefix(raw, refreshPrefix) {
		return nil
	}
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		t, err := q.MCPConnectToken(ctx, authn.HashToken(raw))
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		g, err := q.MCPConnectGrantLock(ctx, t.GrantID)
		if err != nil {
			return err
		}
		if g.ClientID != client.ID {
			return nil
		}
		return s.revoke(ctx, q, g, "client_revoked")
	})
}
