package mcpclient

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// OAuth per member (MCP authorization, 2025-11-25): discovery of the
// protected resource (RFC 9728) and its authorization server (RFC 8414),
// a client a manager entered, else the instance's Client ID Metadata
// Document when the authorization server takes one, else a client
// registered dynamically (RFC 7591); the client authenticates with none,
// client_secret_basic or client_secret_post as its server lists. PKCE
// S256, the resource indicator (RFC 8707), a state bound to the person's
// session and used once, an exact redirect URI on the instance's address.
// Tokens are sealed for (server, person); a refresh replaces a rotated
// refresh token. No redirect is ever followed: every request goes through
// the SSRF guard's client.

const stateTTL = 10 * time.Minute

func tokenAD(server, user uuid.UUID, kind string) string {
	return "mcp_token:" + server.String() + ":" + user.String() + ":" + kind
}

func stateAD(id uuid.UUID) string { return "mcp_oauth_state:" + id.String() }

func random() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func hashOf(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

func (s *Service) oauthClient() *http.Client { return s.policy().Client(15 * time.Second) }

func oauthProblem(ctx context.Context, srv store.McpServer) error {
	return problem(ctx, http.StatusBadGateway, codeOAuth, "problem.oauth", i18n.Args{"server": srv.Name})
}

// Client authentication at the token and revocation endpoints (RFC 6749
// §2.3.1, RFC 8414's token_endpoint_auth_methods_supported): none for a
// client without a secret; with one, client_secret_basic when the
// authorization server offers it (or lists nothing: RFC 8414's default),
// else client_secret_post (MCP authorization, 2025-11-25). private_key_jwt
// and the TLS methods aren't supported.
const (
	authNone  = "none"
	authBasic = "client_secret_basic"
	authPost  = "client_secret_post"
)

// pickAuth is the method a client uses, from the methods its
// authorization server lists; false when it lists only methods ReputeCast
// doesn't support (a client with a secret needs basic or post).
func pickAuth(methods []string, secret bool) (string, bool) {
	offers := func(m string) bool { return len(methods) == 0 || slices.Contains(methods, m) }
	switch {
	case !secret && (offers(authNone) || offers(authBasic) || offers(authPost)):
		return authNone, true
	case secret && offers(authBasic):
		return authBasic, true
	case secret && offers(authPost):
		return authPost, true
	}
	return "", false
}

// methodOf is a client's method as stored: a server set up before the
// method was kept sends its secret, when it has one, with Basic.
func methodOf(method string, secret []byte) string {
	switch {
	case method != "":
		return method
	case len(secret) > 0:
		return authBasic
	}
	return authNone
}

func clientAuthProblem(ctx context.Context, status int, srv store.McpServer) error {
	return problem(ctx, status, codeClientAuth, "problem.clientAuth", i18n.Args{"server": srv.Name})
}

// postForm posts a form to a token or revocation endpoint as the client:
// its client_id in the form, and its secret as its method says (Basic
// authentication, or client_secret in the form). Through the guard's
// client: no redirect is followed.
func (s *Service) postForm(ctx context.Context, endpoint string, form url.Values, method, client string, sealed []byte, server uuid.UUID) (*http.Response, error) {
	form.Set("client_id", client)
	secret := ""
	if method != authNone && len(sealed) > 0 {
		var err error
		if secret, err = s.Box.OpenString(sealed, secretAD(server)); err != nil {
			return nil, err
		}
	}
	if method == authPost && secret != "" {
		form.Set("client_secret", secret)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, errors.New("bad endpoint address")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if method == authBasic && secret != "" {
		req.SetBasicAuth(url.QueryEscape(client), url.QueryEscape(secret))
	}
	return s.oauthClient().Do(req)
}

// clientMetadataPath is the instance's Client ID Metadata Document, under
// the API.
const clientMetadataPath = "/ai/mcp/oauth/client-metadata.json"

// clientMetadataURL is the document's address, the client_id of a client
// that uses it: only on an https public URL ("" otherwise).
func (s *Service) clientMetadataURL() string {
	if s.PublicURL == nil {
		return ""
	}
	base := strings.TrimRight(s.PublicURL(), "/")
	if u, err := url.Parse(base); err != nil || u.Scheme != "https" || u.Host == "" {
		return ""
	}
	return base + api.Version + clientMetadataPath
}

// MCPClientMetadataDocument is ReputeCast's OAuth client as an
// authorization server reads it (OAuth Client ID Metadata Document,
// draft-ietf-oauth-client-id-metadata-document-00): client_id is this
// document's own address. A public client: PKCE, no secret.
type MCPClientMetadataDocument struct {
	ClientID                string   `json:"client_id"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// ClientMetadataDocument is the document, when the public URL is https.
func (s *Service) ClientMetadataDocument() (MCPClientMetadataDocument, bool) {
	id := s.clientMetadataURL()
	if id == "" {
		return MCPClientMetadataDocument{}, false
	}
	return MCPClientMetadataDocument{ClientID: id, ClientName: "ReputeCast", RedirectURIs: []string{s.redirectURI()},
		GrantTypes: []string{"authorization_code", "refresh_token"}, ResponseTypes: []string{"code"}, TokenEndpointAuthMethod: authNone}, true
}

// wellKnown is a well-known address for a URL, with its path appended
// (RFC 8414 §3.1, RFC 9728 §3.1).
func wellKnown(raw, suffix string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	path := strings.TrimRight(u.Path, "/")
	return u.Scheme + "://" + u.Host + "/.well-known/" + suffix + path
}

// originOf is a URL's scheme and host.
func originOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// resourceOf is the resource tokens are asked for (RFC 8707): the server's
// endpoint, or its origin when its metadata names that (found at the root
// well-known address).
func resourceOf(srv store.McpServer) string {
	if srv.Resource != "" {
		return srv.Resource
	}
	return srv.Url
}

// secure reports whether an address of a server's authorization may be
// used: https, or plain http only for a plain http server where the
// instance allows private addresses (a development setup).
func (s *Service) secure(srv store.McpServer, raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return true
	case "http":
		server, err := url.Parse(srv.Url)
		return err == nil && strings.EqualFold(server.Scheme, "http") && s.policy().AllowPrivate
	}
	return false
}

// challenges are the WWW-Authenticate challenges of the server's answer to
// a request without credentials (MCP authorization: the 401 names its
// resource metadata and the scope to ask for). None when it doesn't
// answer 401.
func (s *Service) challenges(ctx context.Context, srv store.McpServer) []oauthex.Challenge {
	method, body := http.MethodPost, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"ReputeCast","version":"1"}}}`
	if srv.Transport == "sse" {
		method, body = http.MethodGet, ""
	}
	req, err := http.NewRequestWithContext(ctx, method, srv.Url, strings.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/json, text/event-stream")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := s.oauthClient().Do(req)
	if err != nil {
		return nil
	}
	defer res.Body.Close() //nolint:errcheck
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 64<<10))
	if res.StatusCode != http.StatusUnauthorized {
		return nil
	}
	cs, _ := oauthex.ParseWWWAuthenticate(res.Header.Values("WWW-Authenticate"))
	return cs
}

// challenge is a parameter of the first Bearer challenge that has it.
func challenge(cs []oauthex.Challenge, param string) string {
	for _, c := range cs {
		if (c.Scheme == "bearer" || param == "resource_metadata") && c.Params[param] != "" {
			return c.Params[param]
		}
	}
	return ""
}

// resourceMetadata is the server's protected resource metadata (RFC 9728)
// and the resource it names, from (in order) the address its 401 names,
// the well-known address at its endpoint's path, and the root well-known
// address (MCP authorization, 2025-11-25). The metadata must name the
// endpoint, or the origin at the root address: never another resource.
func (s *Service) resourceMetadata(ctx context.Context, srv store.McpServer, cs []oauthex.Challenge) (*oauthex.ProtectedResourceMetadata, error) {
	type candidate struct{ url, resource string }
	var list []candidate
	if u := challenge(cs, "resource_metadata"); u != "" {
		if _, err := s.policy().ValidateURL(u); err == nil && s.secure(srv, u) {
			list = append(list, candidate{u, srv.Url})
		}
	}
	root := originOf(srv.Url)
	list = append(list, candidate{wellKnown(srv.Url, "oauth-protected-resource"), srv.Url}, candidate{root + "/.well-known/oauth-protected-resource", root})
	var last error
	for _, c := range list {
		prm, err := oauthex.GetProtectedResourceMetadata(ctx, c.url, c.resource, s.oauthClient())
		if err != nil || prm == nil {
			last = err
			continue
		}
		if len(prm.AuthorizationServers) == 0 {
			return nil, errors.New("protected resource metadata names no authorization server")
		}
		return prm, nil
	}
	return nil, last
}

// authServerMetadata is the authorization server's metadata: OAuth (RFC
// 8414), then OpenID Connect discovery, with the issuer's path inserted
// and, for OpenID Connect, appended (MCP authorization, 2025-11-25).
func (s *Service) authServerMetadata(ctx context.Context, issuer string) (*oauthex.AuthServerMeta, error) {
	u, err := url.Parse(issuer)
	if err != nil {
		return nil, err
	}
	base, path := u.Scheme+"://"+u.Host, strings.Trim(u.Path, "/")
	urls := []string{base + "/.well-known/oauth-authorization-server", base + "/.well-known/openid-configuration"}
	if path != "" {
		urls = []string{base + "/.well-known/oauth-authorization-server/" + path, base + "/.well-known/openid-configuration/" + path,
			base + "/" + path + "/.well-known/openid-configuration"}
	}
	var last error
	for _, m := range urls {
		asm, err := oauthex.GetAuthServerMeta(ctx, m, issuer, s.oauthClient())
		if err == nil && asm != nil {
			return asm, nil
		}
		last = err
	}
	if last == nil {
		last = errors.New("no authorization server metadata")
	}
	return nil, last
}

// discover finds a server's authorization server and registers a client
// when none was entered. Every request goes through the SSRF guard and
// follows no redirect.
func (s *Service) discover(ctx context.Context, srv *store.McpServer) error {
	cs := s.challenges(ctx, *srv)
	prm, err := s.resourceMetadata(ctx, *srv, cs)
	if err != nil || prm == nil {
		s.log().WarnContext(ctx, "mcp: protected resource metadata", "server", srv.ID, "err", err)
		return oauthProblem(ctx, *srv)
	}
	issuer := prm.AuthorizationServers[0]
	if _, err := s.policy().ValidateURL(issuer); err != nil || !s.secure(*srv, issuer) {
		return oauthProblem(ctx, *srv)
	}
	asm, err := s.authServerMetadata(ctx, issuer)
	if err != nil {
		s.log().WarnContext(ctx, "mcp: authorization server metadata", "server", srv.ID, "err", err)
		return oauthProblem(ctx, *srv)
	}
	s256 := false
	for _, m := range asm.CodeChallengeMethodsSupported {
		s256 = s256 || m == "S256"
	}
	if !s256 {
		return oauthProblem(ctx, *srv)
	}
	// The browser opens the authorization endpoint; ReputeCast posts to the
	// token endpoint (through the guard, as every request). Codes, tokens
	// and client secrets travel over https only: metadata naming a plain
	// http endpoint is refused, whichever it is.
	for _, e := range []string{asm.AuthorizationEndpoint, asm.TokenEndpoint, asm.RegistrationEndpoint, asm.RevocationEndpoint} {
		if e != "" && !s.secure(*srv, e) {
			return oauthProblem(ctx, *srv)
		}
	}
	if a, err := url.Parse(asm.AuthorizationEndpoint); err != nil || a.Host == "" {
		return oauthProblem(ctx, *srv)
	}
	if _, err := s.policy().ValidateURL(asm.TokenEndpoint); err != nil {
		return oauthProblem(ctx, *srv)
	}
	srv.Issuer, srv.AuthorizationEndpoint, srv.TokenEndpoint = issuer, asm.AuthorizationEndpoint, asm.TokenEndpoint
	srv.Resource = ""
	if prm.Resource != srv.Url {
		srv.Resource = prm.Resource
	}
	// The scope the server's 401 asks for, else every scope it supports.
	srv.Scopes = challenge(cs, "scope")
	if srv.Scopes == "" {
		srv.Scopes = strings.Join(prm.ScopesSupported, " ")
	}
	srv.RevocationEndpoint = ""
	if _, err := s.policy().ValidateURL(asm.RevocationEndpoint); asm.RevocationEndpoint != "" && err == nil {
		srv.RevocationEndpoint = asm.RevocationEndpoint
	}
	srv.TokenAuthMethods = authMethods(asm.TokenEndpointAuthMethodsSupported)
	// The client (MCP authorization, 2025-11-25): the one a manager
	// entered; else the instance's Client ID Metadata Document when the
	// authorization server takes one (and the public URL is https); else a
	// client registered dynamically; else a manager has to enter one.
	methods := srv.TokenAuthMethods
	switch {
	case srv.ClientID != "":
	case asm.ClientIDMetadataDocumentSupported && s.clientMetadataURL() != "":
		srv.ClientID, srv.ClientMetadata, srv.ClientRegistered, srv.ClientSecretSealed = s.clientMetadataURL(), true, false, nil
	case asm.RegistrationEndpoint != "":
		reg, err := oauthex.RegisterClient(ctx, asm.RegistrationEndpoint, &oauthex.ClientRegistrationMetadata{RedirectURIs: []string{s.redirectURI()},
			TokenEndpointAuthMethod: authNone, GrantTypes: []string{"authorization_code", "refresh_token"}, ResponseTypes: []string{"code"}, ClientName: "ReputeCast"}, s.oauthClient())
		if err != nil {
			s.log().WarnContext(ctx, "mcp: client registration", "server", srv.ID, "err", err)
			return oauthProblem(ctx, *srv)
		}
		srv.ClientID, srv.ClientRegistered, srv.ClientMetadata, srv.ClientSecretSealed = reg.ClientID, true, false, nil
		if reg.ClientSecret != "" {
			if srv.ClientSecretSealed, err = s.Box.SealString(reg.ClientSecret, secretAD(srv.ID)); err != nil {
				return err
			}
		}
		// The registration says how its client authenticates.
		if reg.TokenEndpointAuthMethod != "" {
			methods = []string{reg.TokenEndpointAuthMethod}
		}
	default:
		return problem(ctx, http.StatusUnprocessableEntity, codeNoClient, "problem.noClient", i18n.Args{"server": srv.Name})
	}
	method, ok := pickAuth(methods, len(srv.ClientSecretSealed) > 0)
	if !ok {
		return clientAuthProblem(ctx, http.StatusBadGateway, *srv)
	}
	srv.TokenAuthMethod = method
	return nil
}

// authMethods are the client authentication methods an authorization
// server lists, bounded (they are kept with the server).
func authMethods(listed []string) []string {
	out := []string{}
	for _, m := range listed {
		if len(out) < 20 && len(m) <= 64 && m != "" && !slices.Contains(out, m) {
			out = append(out, m)
		}
	}
	return out
}

// RemoteMCPOAuthStart is where the person signs in.
type RemoteMCPOAuthStart struct {
	AuthorizationURL string `json:"authorizationUrl"`
}

// StartOAuth begins a person's sign-in to an OAuth server: they open the
// returned address in their browser; the authorization server sends them
// back to the callback.
func (s *Service) StartOAuth(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (RemoteMCPOAuthStart, error) {
	var out RemoteMCPOAuthStart
	fresh, err := manager(ctx, s.Q, ws, p, authz.AIUse)
	if err != nil {
		return out, err
	}
	if p.SessionID == uuid.Nil {
		return out, api.Forbidden(authz.AIUse, nil)
	}
	srv, err := s.Q.MCPServer(ctx, store.MCPServerParams{WorkspaceID: ws.ID, ID: id})
	if err != nil {
		return out, api.MapError(err, "MCP server")
	}
	if ok, err := MayUse(ctx, s.Q, ws.ID, fresh.UserID, srv.AccessRoles); err != nil || !ok || !srv.Enabled || srv.Auth != "oauth" {
		if err != nil {
			return out, err
		}
		return out, api.NotFound("MCP server")
	}
	if srv.TokenEndpoint == "" || srv.ClientID == "" {
		if err := s.discover(ctx, &srv); err != nil {
			return out, err
		}
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			cur, err := q.MCPServerLock(ctx, store.MCPServerLockParams{WorkspaceID: ws.ID, ID: id})
			if err != nil {
				return err
			}
			cur.Issuer, cur.AuthorizationEndpoint, cur.TokenEndpoint, cur.Scopes = srv.Issuer, srv.AuthorizationEndpoint, srv.TokenEndpoint, srv.Scopes
			cur.Resource, cur.RevocationEndpoint, cur.TokenAuthMethods = srv.Resource, srv.RevocationEndpoint, srv.TokenAuthMethods
			if cur.ClientID == "" {
				cur.ClientID, cur.ClientRegistered, cur.ClientSecretSealed, cur.ClientMetadata = srv.ClientID, srv.ClientRegistered, srv.ClientSecretSealed, srv.ClientMetadata
			}
			if cur.ClientID == srv.ClientID {
				cur.TokenAuthMethod = srv.TokenAuthMethod
			}
			srv = cur
			return s.save(ctx, q, cur)
		})
		if err != nil {
			return out, err
		}
	}
	state, verifier := random(), random()
	sid := uuid.Must(uuid.NewV7())
	sealed, err := s.Box.SealString(verifier, stateAD(sid))
	if err != nil {
		return out, err
	}
	now := s.now()
	if err := s.Q.MCPOAuthStateInsert(ctx, store.MCPOAuthStateInsertParams{ID: sid, StateHash: hashOf(state), ServerID: srv.ID, UserID: fresh.UserID,
		SessionID: p.SessionID, VerifierSealed: sealed, Now: now, ExpiresAt: now.Add(stateTTL)}); err != nil {
		return out, err
	}
	challenge := sha256.Sum256([]byte(verifier))
	v := url.Values{"response_type": {"code"}, "client_id": {srv.ClientID}, "redirect_uri": {s.redirectURI()}, "state": {state},
		"code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}, "code_challenge_method": {"S256"}, "resource": {resourceOf(srv)}}
	if srv.Scopes != "" {
		v.Set("scope", srv.Scopes)
	}
	sep := "?"
	if strings.Contains(srv.AuthorizationEndpoint, "?") {
		sep = "&"
	}
	out.AuthorizationURL = srv.AuthorizationEndpoint + sep + v.Encode()
	return out, nil
}

// tokenResponse is an authorization server's token answer.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Error        string `json:"error"`
}

// token asks the token endpoint, always with the resource indicator, as
// the client authenticates (postForm).
func (s *Service) token(ctx context.Context, srv store.McpServer, form url.Values) (tokenResponse, error) {
	var out tokenResponse
	form.Set("resource", resourceOf(srv))
	res, err := s.postForm(ctx, srv.TokenEndpoint, form, methodOf(srv.TokenAuthMethod, srv.ClientSecretSealed), srv.ClientID, srv.ClientSecretSealed, srv.ID)
	if err != nil {
		return out, err
	}
	defer res.Body.Close() //nolint:errcheck
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	_ = json.Unmarshal(raw, &out)
	if res.StatusCode != http.StatusOK || out.AccessToken == "" {
		if out.Error == "" {
			out.Error = res.Status
		}
		return out, errors.New("token endpoint: " + out.Error)
	}
	return out, nil
}

func (s *Service) saveToken(ctx context.Context, q *store.Queries, srv store.McpServer, user uuid.UUID, t tokenResponse, keepRefresh []byte) error {
	access, err := s.Box.SealString(t.AccessToken, tokenAD(srv.ID, user, "access"))
	if err != nil {
		return err
	}
	refresh := keepRefresh
	if t.RefreshToken != "" {
		if refresh, err = s.Box.SealString(t.RefreshToken, tokenAD(srv.ID, user, "refresh")); err != nil {
			return err
		}
	}
	var expires *time.Time
	if t.ExpiresIn > 0 {
		e := s.now().Add(time.Duration(t.ExpiresIn) * time.Second)
		expires = &e
	}
	return q.MCPTokenSave(ctx, store.MCPTokenSaveParams{ServerID: srv.ID, UserID: user, WorkspaceID: srv.WorkspaceID, AccessSealed: access, RefreshSealed: refresh,
		ExpiresAt: expires, Now: s.now()})
}

// MCPOAuthCallbackInput is the authorization server's answer.
type MCPOAuthCallbackInput struct {
	Code  string `query:"code" maxLength:"4000"`
	State string `query:"state" maxLength:"200"`
	Error string `query:"error" maxLength:"200"`
}

// Callback finishes a sign-in: the state must be this person's, from this
// session, unused and fresh. It returns where the browser goes next.
func (s *Service) Callback(ctx context.Context, p *authz.Principal, in MCPOAuthCallbackInput) (string, error) {
	if p == nil || !p.IsHuman() || p.SessionID == uuid.Nil || in.State == "" {
		return "", problem(ctx, http.StatusBadRequest, codeState, "problem.state", nil)
	}
	st, err := s.Q.MCPOAuthStateUse(ctx, store.MCPOAuthStateUseParams{Now: s.now(), StateHash: hashOf(in.State), UserID: p.UserID, SessionID: p.SessionID})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", problem(ctx, http.StatusBadRequest, codeState, "problem.state", nil)
	}
	if err != nil {
		return "", err
	}
	var srv store.McpServer
	if err := s.Pool.QueryRow(ctx, `SELECT workspace_id FROM mcp_servers WHERE id = $1`, st.ServerID).Scan(&srv.WorkspaceID); err != nil {
		return "", api.MapError(err, "MCP server")
	}
	ws, err := s.Q.GetWorkspace(ctx, srv.WorkspaceID)
	if err != nil {
		return "", err
	}
	if srv, err = s.Q.MCPServer(ctx, store.MCPServerParams{WorkspaceID: ws.ID, ID: st.ServerID}); err != nil {
		return "", err
	}
	back := "/w/" + ws.Slug + "/settings/integrations/mcp-servers?server=" + srv.ID.String()
	member := *p
	member.WorkspaceID = ws.ID
	if _, err := manager(ctx, s.Q, &ws, &member, authz.AIUse); err != nil {
		return "", err
	}
	if in.Error != "" || in.Code == "" {
		return back + "&error=denied", nil
	}
	verifier, err := s.Box.OpenString(st.VerifierSealed, stateAD(st.ID))
	if err != nil {
		return "", err
	}
	t, err := s.token(ctx, srv, url.Values{"grant_type": {"authorization_code"}, "code": {in.Code}, "redirect_uri": {s.redirectURI()}, "code_verifier": {verifier}})
	if err != nil {
		s.log().WarnContext(ctx, "mcp: token exchange", "server", srv.ID, "err", err)
		return back + "&error=token", nil
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := s.saveToken(ctx, q, srv, p.UserID, t, nil); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Actor: &audit.Actor{Type: authz.ActorUser, ID: &p.UserID, Name: p.Name},
			Action: "mcp_server.connect", TargetType: "mcp_server", TargetID: srv.ID.String(), TargetName: srv.Name, Metadata: map[string]any{"via": "ai_assistant"}})
	})
	if err != nil {
		return "", err
	}
	return back + "&connected=1", nil
}

// Disconnect removes the person's own sign-in to a server, then revokes
// its refresh token at the authorization server when it has a revocation
// endpoint (best effort: the sign-in is gone whatever it answers).
func (s *Service) Disconnect(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := manager(ctx, q, ws, p, authz.AIUse)
		if err != nil {
			return err
		}
		srv, err := q.MCPServer(ctx, store.MCPServerParams{WorkspaceID: ws.ID, ID: id})
		if err != nil {
			return api.MapError(err, "MCP server")
		}
		n, err := q.MCPTokenDelete(ctx, store.MCPTokenDeleteParams{ServerID: srv.ID, UserID: fresh.UserID})
		if err != nil || n == 0 {
			return err
		}
		return s.Audit.LogWith(ctx, q, audit.Event{WorkspaceID: &ws.ID, Action: "mcp_server.disconnect", TargetType: "mcp_server", TargetID: srv.ID.String(),
			TargetName: srv.Name, Metadata: map[string]any{"via": "ai_assistant"}})
	})
	if err != nil {
		return err
	}
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	_, _ = s.Revoke(rctx, 10)
	return nil
}

// Revoke revokes up to limit refresh tokens queued when sign-ins were
// removed (RFC 7009), each tried once: a token is off the queue before it
// is sent, and a failure is logged, never the token.
func (s *Service) Revoke(ctx context.Context, limit int32) (int, error) {
	rows, err := s.Q.MCPTokenRevocationsTake(ctx, limit)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, r := range rows {
		if r.RevocationEndpoint == "" {
			continue
		}
		token, err := s.Box.OpenString(r.RefreshSealed, tokenAD(r.ServerID, r.UserID, "refresh"))
		if err == nil {
			err = s.revoke(ctx, r, token)
		}
		if err != nil {
			s.log().WarnContext(ctx, "mcp: token revocation", "server", r.ServerID, "err", err)
			continue
		}
		n++
	}
	return n, nil
}

// revoke asks the authorization server to revoke a refresh token, as the
// client the token was issued to authenticates.
func (s *Service) revoke(ctx context.Context, r store.MCPTokenRevocationsTakeRow, token string) error {
	form := url.Values{"token": {token}, "token_type_hint": {"refresh_token"}}
	res, err := s.postForm(ctx, r.RevocationEndpoint, form, methodOf(r.TokenAuthMethod, r.ClientSecretSealed), r.ClientID, r.ClientSecretSealed, r.ServerID)
	if err != nil {
		return errors.New("revocation endpoint unreachable")
	}
	defer res.Body.Close() //nolint:errcheck
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 64<<10))
	if res.StatusCode != http.StatusOK {
		return errors.New("revocation endpoint: " + res.Status)
	}
	return nil
}

// RevokeArgs is the minute's revocation of removed sign-ins' tokens.
type RevokeArgs struct{}

// Kind implements river.JobArgs.
func (RevokeArgs) Kind() string { return "mcp.tokens_revoke" }

// InsertOpts runs it once at a time.
func (RevokeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: time.Minute}}
}

// RevokeWorker runs the revocations, 100 a run.
type RevokeWorker struct {
	river.WorkerDefaults[RevokeArgs]
	S *Service
}

// Work implements river.Worker.
func (w *RevokeWorker) Work(ctx context.Context, _ *river.Job[RevokeArgs]) error {
	_, err := w.S.Revoke(ctx, 100)
	return err
}

// accessToken is a person's access token for a server, refreshed (under
// the token's lock, so a rotated refresh token is never sent twice) when
// it expires within a minute. A refused refresh removes the sign-in.
func (s *Service) accessToken(ctx context.Context, srv store.McpServer, user uuid.UUID) (string, error) {
	var token string
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		t, err := q.MCPTokenLock(ctx, store.MCPTokenLockParams{ServerID: srv.ID, UserID: user})
		if errors.Is(err, pgx.ErrNoRows) {
			return signIn(ctx, srv)
		}
		if err != nil {
			return err
		}
		if t.ExpiresAt == nil || t.ExpiresAt.After(s.now().Add(time.Minute)) {
			token, err = s.Box.OpenString(t.AccessSealed, tokenAD(srv.ID, user, "access"))
			return err
		}
		if len(t.RefreshSealed) == 0 {
			return signIn(ctx, srv)
		}
		refresh, err := s.Box.OpenString(t.RefreshSealed, tokenAD(srv.ID, user, "refresh"))
		if err != nil {
			return err
		}
		res, err := s.token(ctx, srv, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresh}})
		if err != nil {
			s.log().WarnContext(ctx, "mcp: refresh", "server", srv.ID, "err", err)
			return signIn(ctx, srv)
		}
		if err := s.saveToken(ctx, q, srv, user, res, t.RefreshSealed); err != nil {
			return err
		}
		token = res.AccessToken
		return nil
	})
	var p *api.Problem
	if errors.As(err, &p) && p.Code == CodeSignIn {
		// The sign-in is gone for good: commit that, then ask for a new one.
		_, _ = s.Q.MCPTokenDelete(ctx, store.MCPTokenDeleteParams{ServerID: srv.ID, UserID: user})
	}
	return token, err
}

// SignedOut removes a person's sign-in the server refused (a 401).
func (s *Service) SignedOut(ctx context.Context, srv uuid.UUID, user uuid.UUID) {
	_, _ = s.Q.MCPTokenDelete(ctx, store.MCPTokenDeleteParams{ServerID: srv, UserID: user})
}

// PurgeArgs is the hourly removal of sign-ins that were never finished.
type PurgeArgs struct{}

// Kind implements river.JobArgs.
func (PurgeArgs) Kind() string { return "mcp.oauth_states_purge" }

// InsertOpts runs it once at a time.
func (PurgeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour}}
}

// PurgeWorker runs the purge.
type PurgeWorker struct {
	river.WorkerDefaults[PurgeArgs]
	S *Service
}

// Work implements river.Worker.
func (w *PurgeWorker) Work(ctx context.Context, _ *river.Job[PurgeArgs]) error {
	_, err := w.S.Q.MCPOAuthStatesPurge(ctx, w.S.now().Add(-stateTTL))
	return err
}
