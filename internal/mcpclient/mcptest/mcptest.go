// Package mcptest has fake remote MCP servers and a fake OAuth
// authorization server for tests of the assistant's MCP client. They are
// built on the go-sdk's own server side and run on httptest servers.
package mcptest

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server is a remote MCP server (Streamable HTTP at /mcp, HTTP with SSE at
// /sse). It records every credential it was sent and every call of its
// tools.
type Server struct {
	*httptest.Server
	MCP *mcp.Server
	// Header and Value: a shared header every request must carry.
	Header, Value string
	// Unauthorized answers every request with 401.
	Unauthorized atomic.Bool
	// Requests counts the HTTP requests it got.
	Requests atomic.Int32

	mu          sync.Mutex
	bearer      func(token string) bool
	metadata    map[string]any
	metaPath    string // where metadata is served
	challenge   string // its 401s' WWW-Authenticate
	credentials []string
	calls       map[string][]json.RawMessage
}

// NewServer starts a server without tools; header and value, when set,
// are a shared header every request must carry.
func NewServer(t testing.TB, header, value string) *Server { return newServer(t, header, value, false) }

// NewLegacyServer is a server that speaks the 2025-11-25 protocol at the
// newest (initialize, and notifications on the session's own stream).
func NewLegacyServer(t testing.TB) *Server {
	s := newServer(t, "", "", false)
	s.MCP = mcp.NewServer(&mcp.Implementation{Name: "fake", Version: "1"}, &mcp.ServerOptions{SupportedProtocolVersions: []string{"2025-11-25"}})
	return s
}

// NewTLSServer is NewServer over https, with httptest's certificate (a
// client trusts it through TrustTLS).
func NewTLSServer(t testing.TB, header, value string) *Server {
	return newServer(t, header, value, true)
}

// TrustTLS makes httptest's certificate the only trusted root of the test
// process's TLS clients: call it from TestMain, before any TLS
// connection is made.
func TrustTLS() (func(), error) {
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	defer srv.Close()
	dir, err := os.MkdirTemp("", "mcptest-ca")
	if err != nil {
		return nil, err
	}
	file := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(file, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw}), 0o600); err != nil {
		return nil, err
	}
	if err := os.Setenv("SSL_CERT_FILE", file); err != nil {
		return nil, err
	}
	return func() { _ = os.RemoveAll(dir) }, nil
}

func newServer(t testing.TB, header, value string, secure bool) *Server {
	s := &Server{MCP: mcp.NewServer(&mcp.Implementation{Name: "fake", Version: "1"}, nil), calls: map[string][]json.RawMessage{}, Header: header, Value: value}
	streamable := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s.MCP }, nil)
	sse := mcp.NewSSEHandler(func(*http.Request) *mcp.Server { return s.MCP }, nil)
	start := httptest.NewServer
	if secure {
		start = httptest.NewTLSServer
	}
	s.Server = start(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Requests.Add(1)
		cred := r.Header.Get("Authorization")
		if s.Header != "" {
			cred = r.Header.Get(s.Header)
		}
		s.mu.Lock()
		s.credentials = append(s.credentials, cred)
		bearer, metadata, metaPath, challenge := s.bearer, s.metadata, s.metaPath, s.challenge
		s.mu.Unlock()
		if metadata != nil && r.URL.Path == metaPath {
			writeJSON(w, http.StatusOK, metadata)
			return
		}
		switch {
		case s.Unauthorized.Load(),
			s.Header != "" && cred != s.Value,
			bearer != nil && !bearer(strings.TrimPrefix(cred, "Bearer ")):
			if challenge == "" {
				challenge = `Bearer resource_metadata="` + s.URL + `/.well-known/oauth-protected-resource/mcp"`
			}
			w.Header().Set("WWW-Authenticate", challenge)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/sse") {
			sse.ServeHTTP(w, r)
			return
		}
		streamable.ServeHTTP(w, r)
	}))
	t.Cleanup(s.Close)
	return s
}

// Endpoint is its Streamable HTTP endpoint.
func (s *Server) Endpoint() string { return s.URL + "/mcp" }

// Tool adds or replaces a tool; result answers each call with the call's
// arguments.
func (s *Server) Tool(name, description string, readOnly bool, result func(args map[string]any) *mcp.CallToolResult) {
	s.ToolCtx(name, description, readOnly, func(_ context.Context, args map[string]any) *mcp.CallToolResult { return result(args) })
}

// ToolCtx is Tool, result getting the call's context (cancelled when the
// client cancels the call).
func (s *Server) ToolCtx(name, description string, readOnly bool, result func(ctx context.Context, args map[string]any) *mcp.CallToolResult) {
	schema := map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}, "title": map[string]any{"type": "string"}}}
	s.MCP.AddTool(&mcp.Tool{Name: name, Description: description, InputSchema: schema, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: readOnly}},
		func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			raw := req.Params.Arguments
			s.mu.Lock()
			s.calls[name] = append(s.calls[name], raw)
			s.mu.Unlock()
			var args map[string]any
			_ = json.Unmarshal(raw, &args)
			return result(ctx, args), nil
		})
}

// Text is a result of one text.
func Text(text string) func(map[string]any) *mcp.CallToolResult {
	return func(map[string]any) *mcp.CallToolResult {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
	}
}

// Slow answers after d, or when the call is cancelled.
func Slow(d time.Duration, text string) func(map[string]any) *mcp.CallToolResult {
	return func(map[string]any) *mcp.CallToolResult {
		time.Sleep(d)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
	}
}

// Calls are a tool's calls, their arguments in order.
func (s *Server) Calls(name string) []json.RawMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]json.RawMessage(nil), s.calls[name]...)
}

// Credentials are the credentials requests carried, in order.
func (s *Server) Credentials() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.credentials...)
}

// Auth is an OAuth authorization server for a Server's endpoint: protected
// resource metadata (RFC 9728) on the resource's host, authorization
// server metadata (RFC 8414), dynamic client registration (RFC 7591),
// clients whose id is their metadata document's URL (Client ID Metadata
// Documents), client authentication at its token and revocation
// endpoints, authorization codes bound to their PKCE challenge, redirect
// URI and resource (RFC 8707), and refresh tokens that rotate: a used one
// is refused.
type Auth struct {
	*httptest.Server
	// Resource is the resource tokens are for.
	Resource string
	// Registrations counts dynamic registrations.
	Registrations atomic.Int32
	mu            sync.Mutex
	clients       map[string]client
	methods       []string // token_endpoint_auth_methods_supported
	cimd          bool
	documents     []Document
	auths         []string // "<endpoint> <method>" of each token and revocation request
	codes         map[string]grant
	access        map[string]string // access token → user hint
	refresh       map[string]string // live refresh token → user hint
	used          map[string]bool   // refresh tokens already used
	n             int
	ExpiresIn     int
	Refreshes     atomic.Int32
	Replays       atomic.Int32
	Exchanges     atomic.Int32
	LastResult    string
	// refreshDelay holds each refresh before it is answered, so two that
	// overlap are seen.
	refreshDelay atomic.Int64
	// RevokeStatus is how the revocation endpoint answers (200 when 0).
	RevokeStatus atomic.Int32
	revoked      []string // "<token> <hint> <client>[ <secret>]"
	scopes       []string // the scope each authorization asked for
}

// AuthOptions are how the protected resource and its authorization server
// are found (MCP authorization, 2025-11-25).
type AuthOptions struct {
	// Metadata is where the server's protected resource metadata is: ""
	// (at the endpoint's path-inserted well-known address, named in its
	// 401s), "root" (at the root well-known address, for the server's
	// origin, not named), or "header" (at an address only its 401s name).
	Metadata string
	// Resource is another resource for the metadata to name.
	Resource string
	// Scope is the scope the server's 401s ask for.
	Scope string
	// OIDC: the issuer has a path and only OpenID Connect discovery,
	// appended to it.
	OIDC bool
	// TLS serves the authorization server over https; Plain names one of
	// its endpoints its metadata gives as plain http instead (token,
	// registration, authorization or revocation).
	TLS   bool
	Plain string
	// Methods are its token_endpoint_auth_methods_supported (["none"]
	// when nil). CIMD: it takes clients whose id is their metadata
	// document's https URL. NoRegistration: it has no registration
	// endpoint. RegisterPost: a dynamic registration gets a secret, to be
	// sent with client_secret_post.
	Methods        []string
	CIMD           bool
	NoRegistration bool
	RegisterPost   bool
}

// Revoked are the tokens revoked, as "<token> <token_type_hint> <client_id>",
// then the client's secret when it authenticated with one.
func (a *Auth) Revoked() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.revoked...)
}

// Scopes are the scopes the authorizations asked for, in order.
func (a *Auth) Scopes() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.scopes...)
}

// DelayRefreshes holds each refresh for d before it is answered.
func (a *Auth) DelayRefreshes(d time.Duration) { a.refreshDelay.Store(int64(d)) }

type grant struct {
	client, redirect, challenge, resource, user string
}

// client is a registered client: its redirect URIs, its secret (none for
// a public client) and, when it registered one, its authentication method.
type client struct {
	uris           []string
	secret, method string
}

// Document is a client metadata document the server fetched: its
// response's Content-Type and Cache-Control, and its fields.
type Document struct {
	ContentType, CacheControl string
	Fields                    map[string]any
}

// Documents are the client metadata documents fetched, in order.
func (a *Auth) Documents() []Document {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]Document(nil), a.documents...)
}

// ClientAuth is how each token and revocation request authenticated its
// client, in order: "token|refresh|revoke none|client_secret_basic|client_secret_post".
func (a *Auth) ClientAuth() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.auths...)
}

// Preregister adds a client registered out of band, as a manager does,
// with its secret and its one redirect URI.
func (a *Auth) Preregister(id, secret, redirect string) {
	a.mu.Lock()
	a.clients[id] = client{uris: []string{redirect}, secret: secret}
	a.mu.Unlock()
}

// authenticate checks a token or revocation request's client
// authentication (RFC 6749 §2.3.1): one method, one the server lists (or
// the client registered), and the client's secret. It records the method.
func (a *Auth) authenticate(endpoint string, r *http.Request) (string, bool) {
	id, secret, basic := r.BasicAuth()
	if basic {
		id, _ = url.QueryUnescape(id)
		secret, _ = url.QueryUnescape(secret)
	}
	method := "none"
	switch {
	case basic && r.Form.Has("client_secret"):
		a.auths = append(a.auths, endpoint+" two methods")
		return "", false
	case basic:
		method = "client_secret_basic"
	case r.Form.Has("client_secret"):
		method, secret = "client_secret_post", r.Form.Get("client_secret")
	}
	a.auths = append(a.auths, endpoint+" "+method)
	if id == "" {
		id = r.Form.Get("client_id")
	}
	c, ok := a.clients[id]
	switch {
	case !ok || (r.Form.Get("client_id") != "" && r.Form.Get("client_id") != id):
		return "", false
	case c.secret == "":
		return id, method == "none"
	case c.method != "" && method != c.method:
		return "", false
	case method == "none" || !slices.Contains(a.methods, method):
		return "", false
	}
	return id, secret == c.secret
}

// NewAuth starts an authorization server and serves the protected
// resource metadata of srv's endpoint on srv itself.
func NewAuth(t testing.TB, srv *Server) *Auth { return NewAuthWith(t, srv, AuthOptions{}) }

// NewAuthWith is NewAuth, with the metadata found as o says.
func NewAuthWith(t testing.TB, srv *Server, o AuthOptions) *Auth {
	a := &Auth{Resource: srv.Endpoint(), clients: map[string]client{}, codes: map[string]grant{}, access: map[string]string{}, refresh: map[string]string{},
		used: map[string]bool{}, ExpiresIn: 3600, methods: o.Methods, cimd: o.CIMD}
	if a.methods == nil {
		a.methods = []string{"none"}
	}
	if o.Metadata == "root" {
		a.Resource = srv.URL
	}
	mux := http.NewServeMux()
	meta := func(w http.ResponseWriter, r *http.Request) {
		issuer := a.URL
		if o.OIDC {
			issuer += "/tenant"
		}
		endpoint := func(name, path string) string {
			if o.Plain == name {
				return strings.Replace(a.URL, "https://", "http://", 1) + path
			}
			return a.URL + path
		}
		m := map[string]any{"issuer": issuer, "authorization_endpoint": endpoint("authorization", "/authorize"), "token_endpoint": endpoint("token", "/token"),
			"registration_endpoint": endpoint("registration", "/register"), "revocation_endpoint": endpoint("revocation", "/revoke"), "code_challenge_methods_supported": []string{"S256"},
			"response_types_supported": []string{"code"}, "grant_types_supported": []string{"authorization_code", "refresh_token"},
			"token_endpoint_auth_methods_supported": a.methods}
		if o.NoRegistration {
			delete(m, "registration_endpoint")
		}
		if o.CIMD {
			m["client_id_metadata_document_supported"] = true
		}
		writeJSON(w, http.StatusOK, m)
	}
	if o.OIDC {
		mux.HandleFunc("/tenant/.well-known/openid-configuration", meta)
	} else {
		mux.HandleFunc("/.well-known/oauth-authorization-server", meta)
	}
	mux.HandleFunc("/revoke", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		a.mu.Lock()
		entry := r.Form.Get("token") + " " + r.Form.Get("token_type_hint") + " " + r.Form.Get("client_id")
		if _, secret, ok := r.BasicAuth(); ok {
			entry += " " + secret
		} else if r.Form.Has("client_secret") {
			entry += " " + r.Form.Get("client_secret")
		}
		a.revoked = append(a.revoked, entry)
		_, ok := a.authenticate("revoke", r)
		a.mu.Unlock()
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid_client"})
			return
		}
		if st := a.RevokeStatus.Load(); st != 0 {
			w.WriteHeader(int(st))
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if o.NoRegistration {
			http.NotFound(w, r)
			return
		}
		var in struct {
			RedirectURIs []string `json:"redirect_uris"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		a.Registrations.Add(1)
		a.mu.Lock()
		a.n++
		id := "client-" + strconv.Itoa(a.n)
		c := client{uris: in.RedirectURIs}
		out := map[string]any{"client_id": id, "redirect_uris": in.RedirectURIs, "token_endpoint_auth_method": "none"}
		if o.RegisterPost {
			c.secret, c.method = "registered-secret-"+strconv.Itoa(a.n), "client_secret_post"
			out["client_secret"], out["token_endpoint_auth_method"] = c.secret, c.method
		}
		a.clients[id] = c
		a.mu.Unlock()
		writeJSON(w, http.StatusCreated, out)
	})
	mux.HandleFunc("/token", a.token)
	if o.TLS {
		a.Server = httptest.NewTLSServer(mux)
	} else {
		a.Server = httptest.NewServer(mux)
	}
	t.Cleanup(a.Close)
	issuer := a.URL
	if o.OIDC {
		issuer += "/tenant"
	}
	resource := a.Resource
	if o.Resource != "" {
		resource = o.Resource
	}
	srv.mu.Lock()
	srv.metadata = map[string]any{"resource": resource, "authorization_servers": []string{issuer}, "scopes_supported": []string{"read", "write"}}
	srv.bearer = a.Valid
	params := []string{}
	switch o.Metadata {
	case "root":
		srv.metaPath = "/.well-known/oauth-protected-resource"
	case "header":
		srv.metaPath = "/meta/resource.json"
		params = append(params, `resource_metadata="`+srv.URL+srv.metaPath+`"`)
	default:
		srv.metaPath = "/.well-known/oauth-protected-resource/mcp"
		params = append(params, `resource_metadata="`+srv.URL+srv.metaPath+`"`)
	}
	if o.Scope != "" {
		params = append(params, `scope="`+o.Scope+`"`)
	}
	srv.challenge = strings.TrimSpace("Bearer " + strings.Join(params, ", "))
	srv.mu.Unlock()
	return a
}

// fetchDocument fetches a client metadata document as an authorization
// server does (draft-ietf-oauth-client-id-metadata-document-00), checks
// it and registers its client: JSON, its client_id the URL exactly, a
// client_name, its redirect URIs, and a public client (shared secrets
// aren't allowed).
func (a *Auth) fetchDocument(t testing.TB, id string) {
	t.Helper()
	res, err := http.Get(id) //nolint:gosec,noctx // the test's own app
	if err != nil {
		t.Fatalf("client metadata document %s: %v", id, err)
	}
	defer res.Body.Close() //nolint:errcheck
	var doc map[string]any
	if res.StatusCode != http.StatusOK || json.NewDecoder(res.Body).Decode(&doc) != nil {
		t.Fatalf("client metadata document %s: %s", id, res.Status)
	}
	uris := []string{}
	list, _ := doc["redirect_uris"].([]any)
	for _, u := range list {
		if s, ok := u.(string); ok {
			uris = append(uris, s)
		}
	}
	method, _ := doc["token_endpoint_auth_method"].(string)
	if name, _ := doc["client_name"].(string); doc["client_id"] != id || name == "" || len(uris) == 0 || strings.HasPrefix(method, "client_secret") {
		t.Fatalf("client metadata document %s: %v", id, doc)
	}
	a.mu.Lock()
	a.documents = append(a.documents, Document{ContentType: res.Header.Get("Content-Type"), CacheControl: res.Header.Get("Cache-Control"), Fields: doc})
	a.clients[id] = client{uris: uris}
	a.mu.Unlock()
}

// Authorize is the person signing in at the authorization URL: it checks
// the request as an authorization server does and returns the code the
// browser carries back, with the state.
func (a *Auth) Authorize(t testing.TB, authorizationURL, user string) (code, state, redirect string) {
	t.Helper()
	u, err := url.Parse(authorizationURL)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if strings.HasPrefix(q.Get("client_id"), "https://") && a.cimd {
		a.fetchDocument(t, q.Get("client_id"))
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	c, ok := a.clients[q.Get("client_id")]
	uris := c.uris
	switch {
	case !strings.HasPrefix(authorizationURL, a.URL+"/authorize?"):
		t.Fatalf("authorization URL %s isn't the server's", authorizationURL)
	case !ok:
		t.Fatalf("unknown client %q", q.Get("client_id"))
	case len(uris) != 1 || uris[0] != q.Get("redirect_uri"):
		t.Fatalf("redirect URI %q isn't the registered %v", q.Get("redirect_uri"), uris)
	case q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "":
		t.Fatalf("no PKCE S256: %v", q)
	case q.Get("resource") != a.Resource:
		t.Fatalf("resource %q, want %q", q.Get("resource"), a.Resource)
	case q.Get("response_type") != "code" || q.Get("state") == "":
		t.Fatalf("authorization request: %v", q)
	}
	a.n++
	a.scopes = append(a.scopes, q.Get("scope"))
	code = "code-" + strconv.Itoa(a.n)
	a.codes[code] = grant{client: q.Get("client_id"), redirect: q.Get("redirect_uri"), challenge: q.Get("code_challenge"), resource: q.Get("resource"), user: user}
	return code, q.Get("state"), q.Get("redirect_uri")
}

func (a *Auth) token(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if r.Form.Get("grant_type") == "refresh_token" {
		time.Sleep(time.Duration(a.refreshDelay.Load()))
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	fail := func(code string) {
		a.LastResult = code
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": code})
	}
	endpoint := "token"
	if r.Form.Get("grant_type") == "refresh_token" {
		endpoint = "refresh"
	}
	id, ok := a.authenticate(endpoint, r)
	if !ok {
		fail("invalid_client")
		return
	}
	if r.Form.Get("resource") != a.Resource {
		fail("invalid_target")
		return
	}
	user := ""
	switch r.Form.Get("grant_type") {
	case "authorization_code":
		a.Exchanges.Add(1)
		g, ok := a.codes[r.Form.Get("code")]
		delete(a.codes, r.Form.Get("code"))
		sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
		if !ok || g.client != id || g.redirect != r.Form.Get("redirect_uri") || g.resource != r.Form.Get("resource") ||
			base64.RawURLEncoding.EncodeToString(sum[:]) != g.challenge {
			fail("invalid_grant")
			return
		}
		user = g.user
	case "refresh_token":
		a.Refreshes.Add(1)
		old := r.Form.Get("refresh_token")
		if a.used[old] {
			a.Replays.Add(1)
			fail("invalid_grant")
			return
		}
		u, ok := a.refresh[old]
		if !ok {
			fail("invalid_grant")
			return
		}
		delete(a.refresh, old)
		a.used[old] = true
		user = u
	default:
		fail("unsupported_grant_type")
		return
	}
	a.n++
	access, refresh := "access-"+user+"-"+strconv.Itoa(a.n), "refresh-"+user+"-"+strconv.Itoa(a.n)
	a.access[access], a.refresh[refresh] = user, user
	a.LastResult = "ok"
	writeJSON(w, http.StatusOK, map[string]any{"access_token": access, "refresh_token": refresh, "token_type": "Bearer", "expires_in": a.ExpiresIn})
}

// Valid reports whether an access token was issued (and not revoked).
func (a *Auth) Valid(token string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.access[token]
	return ok
}

// Revoke makes every access token invalid, as when a sign-in is revoked
// at the authorization server.
func (a *Auth) Revoke() {
	a.mu.Lock()
	a.access = map[string]string{}
	a.mu.Unlock()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WantResource makes the server issue tokens for another resource only,
// as one that serves several resources does.
func (a *Auth) WantResource(resource string) {
	a.mu.Lock()
	a.Resource = resource
	a.mu.Unlock()
}
