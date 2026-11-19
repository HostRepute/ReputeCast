// Package mcptest has fake remote MCP servers and a fake OAuth
// authorization server for tests of the assistant's MCP client. They are
// built on the go-sdk's own server side and run on httptest servers.
package mcptest

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	credentials []string
	calls       map[string][]json.RawMessage
}

// NewServer starts a server without tools; header and value, when set,
// are a shared header every request must carry.
func NewServer(t testing.TB, header, value string) *Server {
	s := &Server{MCP: mcp.NewServer(&mcp.Implementation{Name: "fake", Version: "1"}, nil), calls: map[string][]json.RawMessage{}, Header: header, Value: value}
	streamable := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s.MCP }, nil)
	sse := mcp.NewSSEHandler(func(*http.Request) *mcp.Server { return s.MCP }, nil)
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Requests.Add(1)
		cred := r.Header.Get("Authorization")
		if s.Header != "" {
			cred = r.Header.Get(s.Header)
		}
		s.mu.Lock()
		s.credentials = append(s.credentials, cred)
		bearer, metadata := s.bearer, s.metadata
		s.mu.Unlock()
		if metadata != nil && r.URL.Path == "/.well-known/oauth-protected-resource/mcp" {
			writeJSON(w, http.StatusOK, metadata)
			return
		}
		switch {
		case s.Unauthorized.Load(),
			s.Header != "" && cred != s.Value,
			bearer != nil && !bearer(strings.TrimPrefix(cred, "Bearer ")):
			w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+s.URL+`/.well-known/oauth-protected-resource/mcp"`)
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
	schema := map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}, "title": map[string]any{"type": "string"}}}
	s.MCP.AddTool(&mcp.Tool{Name: name, Description: description, InputSchema: schema, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: readOnly}},
		func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			raw := req.Params.Arguments
			s.mu.Lock()
			s.calls[name] = append(s.calls[name], raw)
			s.mu.Unlock()
			var args map[string]any
			_ = json.Unmarshal(raw, &args)
			return result(args), nil
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
// authorization codes bound to their PKCE challenge, redirect URI and
// resource (RFC 8707), and refresh tokens that rotate: a used one is
// refused.
type Auth struct {
	*httptest.Server
	// Resource is the resource tokens are for; Registered the clients.
	Resource   string
	mu         sync.Mutex
	clients    map[string][]string // client id → redirect URIs
	codes      map[string]grant
	access     map[string]string // access token → user hint
	refresh    map[string]string // live refresh token → user hint
	used       map[string]bool   // refresh tokens already used
	n          int
	ExpiresIn  int
	Refreshes  atomic.Int32
	Replays    atomic.Int32
	Exchanges  atomic.Int32
	LastResult string
	// refreshDelay holds each refresh before it is answered, so two that
	// overlap are seen.
	refreshDelay atomic.Int64
}

// DelayRefreshes holds each refresh for d before it is answered.
func (a *Auth) DelayRefreshes(d time.Duration) { a.refreshDelay.Store(int64(d)) }

type grant struct {
	client, redirect, challenge, resource, user string
}

// NewAuth starts an authorization server and serves the protected
// resource metadata of srv's endpoint on srv itself.
func NewAuth(t testing.TB, srv *Server) *Auth {
	a := &Auth{Resource: srv.Endpoint(), clients: map[string][]string{}, codes: map[string]grant{}, access: map[string]string{}, refresh: map[string]string{},
		used: map[string]bool{}, ExpiresIn: 3600}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"issuer": a.URL, "authorization_endpoint": a.URL + "/authorize", "token_endpoint": a.URL + "/token",
			"registration_endpoint": a.URL + "/register", "code_challenge_methods_supported": []string{"S256"}, "response_types_supported": []string{"code"},
			"grant_types_supported": []string{"authorization_code", "refresh_token"}, "token_endpoint_auth_methods_supported": []string{"none"}})
	})
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			RedirectURIs []string `json:"redirect_uris"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		a.mu.Lock()
		a.n++
		id := "client-" + strconv.Itoa(a.n)
		a.clients[id] = in.RedirectURIs
		a.mu.Unlock()
		writeJSON(w, http.StatusCreated, map[string]any{"client_id": id, "redirect_uris": in.RedirectURIs, "token_endpoint_auth_method": "none"})
	})
	mux.HandleFunc("/token", a.token)
	a.Server = httptest.NewServer(mux)
	t.Cleanup(a.Close)
	srv.mu.Lock()
	srv.metadata = map[string]any{"resource": a.Resource, "authorization_servers": []string{a.URL}, "scopes_supported": []string{"read", "write"}}
	srv.bearer = a.Valid
	srv.mu.Unlock()
	return a
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
	a.mu.Lock()
	defer a.mu.Unlock()
	uris, ok := a.clients[q.Get("client_id")]
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
		if !ok || g.client != r.Form.Get("client_id") || g.redirect != r.Form.Get("redirect_uri") || g.resource != r.Form.Get("resource") ||
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
