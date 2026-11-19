// Package mcpclient is the assistant's client of remote MCP servers
// (assistant brief D2, §3.6; milestone B5). Integration managers connect
// servers (Streamable HTTP, or HTTP with SSE; never a local process) with
// no authentication, a shared header secret, or OAuth per member; a sync
// lists their tools, each of which waits for a manager to mark it
// read-only, writes or blocked. A tool that changes after it was decided
// on is off until a manager decides again: each call lists the server's
// tools first, in its own session, and servers without a person's sign-in
// are synced every hour. A sign-in removed (disconnected, or its member
// gone) has its refresh token revoked when the authorization server can.
// Every request goes through the SSRF guard; secrets and tokens are
// sealed and never returned.
package mcpclient

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/netguard"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// Service is the workspace's MCP servers.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	Log   *slog.Logger
	// AllowPrivate allows servers on private addresses (the instance's
	// setting); PublicURL is the instance's address, for OAuth redirects.
	AllowPrivate func() bool
	PublicURL    func() string
	// Now is the clock, CallTimeout a call's (tests); zero values take the
	// wall clock and 30 seconds.
	Now         func() time.Time
	CallTimeout time.Duration
}

const (
	// MaxResult bounds a tool's result as the model reads it.
	MaxResult = 32 << 10
	// maxTools bounds a server's catalog.
	maxTools = 200
)

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) callTimeout() time.Duration {
	if s.CallTimeout > 0 {
		return s.CallTimeout
	}
	return 30 * time.Second
}

func (s *Service) policy() netguard.Policy {
	return netguard.Policy{AllowPrivate: s.AllowPrivate != nil && s.AllowPrivate()}
}

func (s *Service) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "mcpclient")

// Problem codes.
const (
	CodeUnreachable = "mcp_unreachable"
	CodeSignIn      = "mcp_sign_in"
	CodeToolChanged = "mcp_tool_changed"
	CodeToolBlocked = "mcp_tool_blocked"
	CodeTimeout     = "mcp_timeout"
	CodeTooLarge    = "mcp_too_large"
	codeURL         = "mcp_url"
	codeInput       = "mcp_input"
	codeState       = "mcp_oauth_state"
	codeOAuth       = "mcp_oauth"
	codeReviewed    = "mcp_tool_reviewed"
)

func problem(ctx context.Context, status int, code, key string, args i18n.Args) *api.Problem {
	return api.Err(status, code, msgs.M(key).Text(i18n.LangFrom(ctx), args))
}

// Problems a call or a sync can end with.
func unreachable(ctx context.Context, name string) error {
	return problem(ctx, http.StatusBadGateway, CodeUnreachable, "problem.unreachable", i18n.Args{"server": name})
}

func signIn(ctx context.Context, srv store.McpServer) error {
	return problem(ctx, http.StatusConflict, CodeSignIn, "problem.signIn", i18n.Args{"server": srv.Name}).WithMeta("serverId", srv.ID)
}

func toolChanged(ctx context.Context, tool string) error {
	return problem(ctx, http.StatusConflict, CodeToolChanged, "problem.toolChanged", i18n.Args{"tool": tool})
}

func toolBlocked(ctx context.Context, tool string) error {
	return problem(ctx, http.StatusConflict, CodeToolBlocked, "problem.toolBlocked", i18n.Args{"tool": tool})
}

// RemoteMCPServerInput adds a server. HeaderValue and ClientSecret are
// write-only.
type RemoteMCPServerInput struct {
	Name string `json:"name" minLength:"1" maxLength:"80"`
	URL  string `json:"url" minLength:"1" maxLength:"500" doc:"The server's MCP endpoint (Streamable HTTP, or HTTP with SSE)"`
	// Transport is streamable unless the server only speaks the older
	// HTTP with SSE; a sync falls back to it by itself.
	Transport string `json:"transport,omitempty" enum:"streamable,sse"`
	Auth      string `json:"auth" enum:"none,header,oauth"`
	// HeaderName and HeaderValue are the shared secret's header (auth header).
	HeaderName  string `json:"headerName,omitempty" maxLength:"100" pattern:"^[A-Za-z0-9-]*$"`
	HeaderValue string `json:"headerValue,omitempty" maxLength:"4000"`
	// ClientID and ClientSecret are a client the manager registered with
	// the authorization server; without them, ReputeCast registers one.
	ClientID     string `json:"clientId,omitempty" maxLength:"500"`
	ClientSecret string `json:"clientSecret,omitempty" maxLength:"4000"`
	// Roles are the role keys who may use it; empty: everyone with Use AI
	// features.
	Roles []string `json:"roles,omitempty" maxItems:"50"`
}

// RemoteMCPServerUpdate changes a server; absent fields stay.
type RemoteMCPServerUpdate struct {
	Name         *string   `json:"name,omitempty" minLength:"1" maxLength:"80"`
	Enabled      *bool     `json:"enabled,omitempty"`
	Roles        *[]string `json:"roles,omitempty" maxItems:"50"`
	HeaderValue  *string   `json:"headerValue,omitempty" maxLength:"4000"`
	ClientID     *string   `json:"clientId,omitempty" maxLength:"500"`
	ClientSecret *string   `json:"clientSecret,omitempty" maxLength:"4000"`
}

// RemoteMCPServer is a server as managers and members see it: never its
// secrets.
type RemoteMCPServer struct {
	ID         uuid.UUID `json:"id"`
	Key        string    `json:"key" doc:"The prefix of its tools for the model: mcp__<key>__<tool>"`
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	Transport  string    `json:"transport" enum:"streamable,sse"`
	Auth       string    `json:"auth" enum:"none,header,oauth"`
	HeaderName string    `json:"headerName,omitempty"`
	HeaderSet  bool      `json:"headerSet"`
	// ClientID is the OAuth client: entered by a manager, or registered
	// by ReputeCast (ClientRegistered).
	ClientID         string     `json:"clientId,omitempty"`
	ClientSecretSet  bool       `json:"clientSecretSet"`
	ClientRegistered bool       `json:"clientRegistered"`
	Roles            []string   `json:"roles"`
	Enabled          bool       `json:"enabled"`
	Status           string     `json:"status" enum:"new,ok,unreachable,sign_in"`
	StatusDetail     string     `json:"statusDetail,omitempty"`
	SyncedAt         *time.Time `json:"syncedAt,omitempty"`
	// Connected: the caller's own OAuth sign-in is there (auth oauth).
	Connected bool `json:"connected"`
	// RedirectURI is the address to register with the authorization server
	// for a pre-registered client.
	RedirectURI string          `json:"redirectUri,omitempty"`
	Tools       []RemoteMCPTool `json:"tools"`
	// Waiting counts the tools a manager hasn't decided on, or that changed.
	Waiting int `json:"waiting"`
}

// RemoteMCPTool is a tool of a server's catalog.
type RemoteMCPTool struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	ModelName   string          `json:"modelName"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
	// Annotations are the server's hints (readOnlyHint, destructiveHint…),
	// never evidence.
	Annotations json.RawMessage `json:"annotations"`
	// Hash identifies the description, schema and annotations reviewed.
	Hash     string `json:"hash"`
	Decision string `json:"decision" enum:"pending,read_only,writes,blocked"`
	// Changed: it changed after a manager decided on it; it is off until
	// they decide again.
	Changed   bool       `json:"changed"`
	Removed   bool       `json:"removed"`
	DecidedAt *time.Time `json:"decidedAt,omitempty"`
	// Approved is the version decided on, for a tool that changed since:
	// what a manager approved, to compare with what the server says now.
	Approved *RemoteMCPToolVersion `json:"approved,omitempty"`
}

// RemoteMCPToolVersion is a tool's description, schema and annotations as
// a manager reviewed them.
type RemoteMCPToolVersion struct {
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
	Annotations json.RawMessage `json:"annotations"`
}

// RemoteMCPServerList is the workspace's servers.
type RemoteMCPServerList struct {
	Servers []RemoteMCPServer `json:"servers"`
}

// manager is a person who may manage integrations, as they are now.
func manager(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, perm authz.Permission) (*authz.Principal, error) {
	fresh, err := brandkit.VoiceActor(ctx, q, ws, p)
	if err != nil {
		return nil, err
	}
	if !fresh.Can(perm) {
		return nil, api.Forbidden(perm, nil)
	}
	return fresh, nil
}

// MayUse reports whether a person may use a server: they use AI features,
// and the server is for everyone or for their role.
func MayUse(ctx context.Context, q *store.Queries, ws uuid.UUID, user uuid.UUID, roles []string) (bool, error) {
	if len(roles) == 0 {
		return true, nil
	}
	r, err := q.BrandVoiceActor(ctx, store.BrandVoiceActorParams{WorkspaceID: ws, UserID: user})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil && slices.Contains(roles, r.Key), err
}

func (s *Service) redirectURI() string {
	base := ""
	if s.PublicURL != nil {
		base = strings.TrimRight(s.PublicURL(), "/")
	}
	return base + api.Version + "/ai/mcp/oauth/callback"
}

func headerAD(server uuid.UUID) string { return "mcp_server:" + server.String() + ":header" }
func secretAD(server uuid.UUID) string { return "mcp_server:" + server.String() + ":client_secret" }

// toolHash is a tool's identity for review: its description, input schema
// and annotations.
func toolHash(description string, schema, annotations json.RawMessage) []byte {
	raw, _ := json.Marshal([]any{description, json.RawMessage(canonical(schema)), json.RawMessage(canonical(annotations))})
	h := sha256.Sum256(raw)
	return h[:]
}

// canonical is JSON with its keys sorted (as encoding/json writes maps).
func canonical(raw json.RawMessage) []byte {
	var v any
	if len(raw) == 0 || json.Unmarshal(raw, &v) != nil {
		return []byte("null")
	}
	out, _ := json.Marshal(v)
	return out
}

func (s *Service) view(ctx context.Context, q *store.Queries, srv store.McpServer, user uuid.UUID, withTools bool) (RemoteMCPServer, error) {
	v := RemoteMCPServer{ID: srv.ID, Key: srv.Key, Name: srv.Name, URL: srv.Url, Transport: srv.Transport, Auth: srv.Auth, HeaderName: srv.HeaderName,
		HeaderSet: len(srv.HeaderSealed) > 0, ClientID: srv.ClientID, ClientSecretSet: len(srv.ClientSecretSealed) > 0, ClientRegistered: srv.ClientRegistered,
		Roles: []string{}, Enabled: srv.Enabled, Status: srv.Status, StatusDetail: srv.StatusDetail, SyncedAt: srv.SyncedAt, Tools: []RemoteMCPTool{}}
	if srv.AccessRoles != nil {
		v.Roles = srv.AccessRoles
	}
	if srv.Auth == "oauth" {
		v.RedirectURI = s.redirectURI()
		if _, err := q.MCPToken(ctx, store.MCPTokenParams{ServerID: srv.ID, UserID: user}); err == nil {
			v.Connected = true
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return v, err
		}
	}
	tools, err := q.MCPTools(ctx, srv.ID)
	if err != nil {
		return v, err
	}
	for _, t := range tools {
		tv := toolView(t)
		if !tv.Removed && (tv.Decision == "pending" || tv.Changed) {
			v.Waiting++
		}
		if withTools {
			v.Tools = append(v.Tools, tv)
		}
	}
	return v, nil
}

func toolView(t store.McpTool) RemoteMCPTool {
	v := RemoteMCPTool{ID: t.ID, Name: t.Name, ModelName: t.ModelName, Description: t.Description, InputSchema: t.InputSchema, Annotations: t.Annotations,
		Hash: hexOf(t.Hash), Decision: t.Decision, Changed: t.Decision != "pending" && !slices.Equal(t.ApprovedHash, t.Hash), Removed: t.RemovedAt != nil,
		DecidedAt: t.DecidedAt}
	if v.Changed && t.ApprovedDescription != nil {
		v.Approved = &RemoteMCPToolVersion{Description: *t.ApprovedDescription, InputSchema: jsonOr(t.ApprovedInputSchema), Annotations: jsonOr(t.ApprovedAnnotations)}
	}
	return v
}

// jsonOr is a kept JSON value, or an empty object.
func jsonOr(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("{}")
	}
	return raw
}

func hexOf(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		out = append(out, digits[c>>4], digits[c&15])
	}
	return string(out)
}

// Servers lists the workspace's servers with their catalogs.
func (s *Service) Servers(ctx context.Context, ws *store.Workspace, p *authz.Principal) (RemoteMCPServerList, error) {
	out := RemoteMCPServerList{Servers: []RemoteMCPServer{}}
	if _, err := manager(ctx, s.Q, ws, p, authz.IntegrationsView); err != nil {
		return out, err
	}
	rows, err := s.Q.MCPServers(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		v, err := s.view(ctx, s.Q, r, p.UserID, true)
		if err != nil {
			return out, err
		}
		out.Servers = append(out.Servers, v)
	}
	return out, nil
}

// Usable lists the servers a person may choose for their threads, with
// whether they still have to sign in.
func (s *Service) Usable(ctx context.Context, ws *store.Workspace, p *authz.Principal) (RemoteMCPServerList, error) {
	out := RemoteMCPServerList{Servers: []RemoteMCPServer{}}
	if _, err := manager(ctx, s.Q, ws, p, authz.AIUse); err != nil {
		return out, err
	}
	rows, err := s.Q.MCPServers(ctx, ws.ID)
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		ok, err := MayUse(ctx, s.Q, ws.ID, p.UserID, r.AccessRoles)
		if err != nil {
			return out, err
		}
		if !ok || !r.Enabled {
			continue
		}
		v, err := s.view(ctx, s.Q, r, p.UserID, false)
		if err != nil {
			return out, err
		}
		v.HeaderName, v.ClientID, v.RedirectURI = "", "", ""
		out.Servers = append(out.Servers, v)
	}
	return out, nil
}

// serverKey is a short key for a server's tools, unique in the workspace.
func serverKey(ctx context.Context, q *store.Queries, ws uuid.UUID, name string) (string, error) {
	base := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return -1
	}, name)
	if len(base) > 12 {
		base = base[:12]
	}
	if base == "" {
		base = "server"
	}
	taken, err := q.MCPServerKeys(ctx, ws)
	if err != nil {
		return "", err
	}
	key := base
	for i := 2; slices.Contains(taken, key); i++ {
		key = base + strconv.Itoa(i)
	}
	return key, nil
}

// Create adds a server. Its URL must pass the SSRF guard; its secrets are
// sealed. It has no tools until a sync.
func (s *Service) Create(ctx context.Context, ws *store.Workspace, p *authz.Principal, in RemoteMCPServerInput) (RemoteMCPServer, error) {
	u, err := s.policy().ValidateURL(in.URL)
	if err != nil {
		return RemoteMCPServer{}, problem(ctx, http.StatusUnprocessableEntity, codeURL, "problem.url", i18n.Args{"reason": err.Error()})
	}
	if in.Transport == "" {
		in.Transport = "streamable"
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || (in.Auth == "header") != (in.HeaderName != "" && in.HeaderValue != "") || (in.Auth != "oauth" && (in.ClientID != "" || in.ClientSecret != "")) {
		return RemoteMCPServer{}, problem(ctx, http.StatusUnprocessableEntity, codeInput, "problem.input", nil)
	}
	id := uuid.Must(uuid.NewV7())
	var out RemoteMCPServer
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := manager(ctx, q, ws, p, authz.IntegrationsManage)
		if err != nil {
			return err
		}
		key, err := serverKey(ctx, q, ws.ID, name)
		if err != nil {
			return err
		}
		var header, secret []byte
		if in.Auth == "header" {
			if header, err = s.Box.SealString(in.HeaderValue, headerAD(id)); err != nil {
				return err
			}
		}
		if in.ClientSecret != "" {
			if secret, err = s.Box.SealString(in.ClientSecret, secretAD(id)); err != nil {
				return err
			}
		}
		var roles []string
		if len(in.Roles) > 0 {
			roles = in.Roles
		}
		if err := q.MCPServerInsert(ctx, store.MCPServerInsertParams{ID: id, WorkspaceID: ws.ID, Key: key, Name: name, Url: u.String(), Transport: in.Transport,
			Auth: in.Auth, HeaderName: in.HeaderName, HeaderSealed: header, ClientID: in.ClientID, ClientSecretSealed: secret, AccessRoles: roles,
			CreatedBy: &fresh.UserID, Now: s.now()}); err != nil {
			return err
		}
		srv, err := q.MCPServer(ctx, store.MCPServerParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return err
		}
		if out, err = s.view(ctx, q, srv, fresh.UserID, true); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "mcp_server.created", TargetType: "mcp_server", TargetID: id.String(), TargetName: name,
			After: map[string]any{"url": u.String(), "transport": in.Transport, "auth": in.Auth, "roles": in.Roles, "header_set": in.Auth == "header", "client_id": in.ClientID}})
	})
	return out, err
}

// Update changes a server's name, switch, access or secrets.
func (s *Service) Update(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in RemoteMCPServerUpdate) (RemoteMCPServer, error) {
	var out RemoteMCPServer
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := manager(ctx, q, ws, p, authz.IntegrationsManage)
		if err != nil {
			return err
		}
		srv, err := q.MCPServerLock(ctx, store.MCPServerLockParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "MCP server")
		}
		changed := []string{}
		if in.Name != nil && strings.TrimSpace(*in.Name) != "" {
			srv.Name, changed = strings.TrimSpace(*in.Name), append(changed, "name")
		}
		if in.Enabled != nil {
			srv.Enabled, changed = *in.Enabled, append(changed, "enabled")
		}
		if in.Roles != nil {
			srv.AccessRoles, changed = nil, append(changed, "roles")
			if len(*in.Roles) > 0 {
				srv.AccessRoles = *in.Roles
			}
		}
		if in.HeaderValue != nil {
			if srv.Auth != "header" || *in.HeaderValue == "" {
				return problem(ctx, http.StatusUnprocessableEntity, codeInput, "problem.input", nil)
			}
			if srv.HeaderSealed, err = s.Box.SealString(*in.HeaderValue, headerAD(id)); err != nil {
				return err
			}
			changed = append(changed, "header")
		}
		if in.ClientID != nil || in.ClientSecret != nil {
			if srv.Auth != "oauth" {
				return problem(ctx, http.StatusUnprocessableEntity, codeInput, "problem.input", nil)
			}
			if in.ClientID != nil {
				srv.ClientID, srv.ClientRegistered = *in.ClientID, false
			}
			srv.ClientSecretSealed = nil
			if in.ClientSecret != nil && *in.ClientSecret != "" {
				if srv.ClientSecretSealed, err = s.Box.SealString(*in.ClientSecret, secretAD(id)); err != nil {
					return err
				}
			}
			changed = append(changed, "client")
		}
		if err := s.save(ctx, q, srv); err != nil {
			return err
		}
		if srv, err = q.MCPServer(ctx, store.MCPServerParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		if out, err = s.view(ctx, q, srv, fresh.UserID, true); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "mcp_server.updated", TargetType: "mcp_server", TargetID: id.String(), TargetName: srv.Name,
			Metadata: map[string]any{"changed": changed, "enabled": srv.Enabled, "roles": out.Roles}})
	})
	return out, err
}

func (s *Service) save(ctx context.Context, q *store.Queries, srv store.McpServer) error {
	return q.MCPServerSave(ctx, store.MCPServerSaveParams{Name: srv.Name, Enabled: srv.Enabled, AccessRoles: srv.AccessRoles, HeaderSealed: srv.HeaderSealed,
		ClientID: srv.ClientID, ClientSecretSealed: srv.ClientSecretSealed, ClientRegistered: srv.ClientRegistered, Issuer: srv.Issuer,
		AuthorizationEndpoint: srv.AuthorizationEndpoint, TokenEndpoint: srv.TokenEndpoint, Scopes: srv.Scopes, Resource: srv.Resource,
		RevocationEndpoint: srv.RevocationEndpoint, Now: s.now(), WorkspaceID: srv.WorkspaceID, ID: srv.ID})
}

// Delete removes a server, its tools and every member's sign-in to it,
// queueing their refresh tokens for revocation in the same transaction.
func (s *Service) Delete(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := manager(ctx, q, ws, p, authz.IntegrationsManage); err != nil {
			return err
		}
		srv, err := q.MCPServerLock(ctx, store.MCPServerLockParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "MCP server")
		}
		// Its members' sign-ins go with it; their refresh tokens are revoked
		// by the minute's job.
		queued, err := q.MCPServerRevocationsQueue(ctx, store.MCPServerRevocationsQueueParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return err
		}
		if _, err := q.MCPServerDelete(ctx, store.MCPServerDeleteParams{WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "mcp_server.deleted", TargetType: "mcp_server", TargetID: id.String(), TargetName: srv.Name,
			Metadata: map[string]any{"refresh_tokens_queued": queued}})
	})
}
