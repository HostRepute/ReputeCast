// Package ssotest is a fake OpenID Connect issuer (and a fake GitHub) for
// tests and local screenshots: discovery, signing keys, an authorization
// endpoint that approves at once, a token endpoint that checks the client
// secret and the PKCE verifier and returns a signed ID token with the
// nonce, and userinfo. Real identity providers are never called in tests.
package ssotest

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc/oidctest"
)

// IdP is a fake OpenID Connect provider.
type IdP struct {
	// Server is the HTTP server; its URL is the issuer (plus Path).
	Server *httptest.Server
	// Path is appended to the server URL to form the issuer ("/tenant/v2.0").
	Path         string
	ClientID     string
	ClientSecret string
	// Claims are the claims of the next sign-in (sub, email, groups…).
	// Users picks claims by login_hint instead when it has an entry.
	Claims map[string]any
	Users  map[string]map[string]any
	// Error makes the authorization endpoint answer with ?error=…
	Error string
	// DiscoveryStatus overrides the discovery document's HTTP status (404).
	DiscoveryStatus int
	// DiscoveryIssuer and TokenIssuer override the issuer the discovery
	// document and the ID tokens report (Microsoft's "{tenantid}" template
	// and per-tenant issuers).
	DiscoveryIssuer string
	TokenIssuer     string
	// Nonce, when set, replaces the nonce the client sent (a replayed token).
	Nonce string

	key     *rsa.PrivateKey
	keys    *oidctest.Server
	control bool
	mu      sync.Mutex
	codes   map[string]grant
	// Requests records the paths the fake served, for assertions.
	Requests []string
}

type grant struct {
	claims      map[string]any
	nonce       string
	challenge   string
	redirectURI string
	clientID    string
}

// New starts a fake issuer on a random local port. Close it with Close.
func New() *IdP {
	idp := newIdP()
	idp.Server = httptest.NewServer(idp)
	idp.keys.SetIssuer(idp.Server.URL)
	return idp
}

// NewAt starts a fake issuer on addr ("127.0.0.1:9555") for local runs,
// with path as the issuer path. POST /_claims (a JSON object) changes the
// claims of the next sign-ins and POST /_error?error=… makes them fail.
func NewAt(addr, path string) (*IdP, error) {
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", addr)
	if err != nil {
		return nil, err
	}
	idp := newIdP()
	idp.Path, idp.control = path, true
	idp.Server = httptest.NewUnstartedServer(idp)
	_ = idp.Server.Listener.Close()
	idp.Server.Listener = l
	idp.Server.Start()
	idp.keys.SetIssuer(idp.Server.URL)
	return idp, nil
}

func newIdP() *IdP {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	idp := &IdP{ClientID: "test-client", ClientSecret: "test-secret", key: key, codes: map[string]grant{},
		Claims: map[string]any{"sub": "user-1", "email": "person@example.com", "email_verified": true, "name": "Test Person"}}
	idp.keys = &oidctest.Server{PublicKeys: []oidctest.PublicKey{{PublicKey: key.Public(), KeyID: "k1", Algorithm: "RS256"}}}
	return idp
}

// ServeHTTP serves discovery, keys, authorization, token and userinfo.
func (i *IdP) ServeHTTP(w http.ResponseWriter, r *http.Request) { i.serve(w, r) }

// Set replaces the claims of the next sign-ins and the error to answer
// with, safely while the server runs.
func (i *IdP) Set(claims map[string]any, idpError string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.Claims, i.Error = claims, idpError
}

// Close stops the server.
func (i *IdP) Close() { i.Server.Close() }

// Issuer is the issuer URL to configure.
func (i *IdP) Issuer() string { return i.Server.URL + i.Path }

func (i *IdP) issuer() string {
	if i.TokenIssuer != "" {
		return i.TokenIssuer
	}
	return i.Issuer()
}

func (i *IdP) serve(w http.ResponseWriter, r *http.Request) {
	i.mu.Lock()
	i.Requests = append(i.Requests, r.Method+" "+r.URL.Path)
	i.mu.Unlock()
	p := strings.TrimPrefix(r.URL.Path, i.Path)
	switch {
	case i.control && r.Method == http.MethodPost && r.URL.Path == "/_claims":
		var next map[string]any
		if err := json.NewDecoder(r.Body).Decode(&next); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		i.Set(next, "")
		w.WriteHeader(http.StatusNoContent)
	case i.control && r.Method == http.MethodPost && r.URL.Path == "/_error":
		i.mu.Lock()
		i.Error = r.URL.Query().Get("error")
		i.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	case p == "/.well-known/openid-configuration":
		if i.DiscoveryStatus != 0 {
			http.Error(w, "not found", i.DiscoveryStatus)
			return
		}
		iss := i.Issuer()
		if i.DiscoveryIssuer != "" {
			iss = i.DiscoveryIssuer
		}
		writeJSON(w, map[string]any{
			"issuer": iss, "authorization_endpoint": i.Issuer() + "/auth", "token_endpoint": i.Issuer() + "/token",
			"userinfo_endpoint": i.Issuer() + "/userinfo", "jwks_uri": i.Server.URL + "/keys",
			"id_token_signing_alg_values_supported": []string{"RS256"}, "response_types_supported": []string{"code"},
			"subject_types_supported": []string{"public"},
		})
	case r.URL.Path == "/keys":
		i.keys.ServeHTTP(w, r)
	case p == "/auth":
		i.authorize(w, r)
	case p == "/token":
		i.token(w, r)
	case p == "/userinfo":
		writeJSON(w, map[string]any{"sub": i.Claims["sub"]})
	default:
		http.NotFound(w, r)
	}
}

func (i *IdP) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	redirect := q.Get("redirect_uri")
	back, err := url.Parse(redirect)
	if err != nil || redirect == "" {
		http.Error(w, "bad redirect_uri", http.StatusBadRequest)
		return
	}
	out := back.Query()
	out.Set("state", q.Get("state"))
	i.mu.Lock()
	idpError, claims := i.Error, i.Claims
	if u, ok := i.Users[strings.ToLower(q.Get("login_hint"))]; ok {
		claims = u
	}
	i.mu.Unlock()
	if idpError != "" {
		out.Set("error", idpError)
		out.Set("error_description", "The fake IdP was told to fail")
		back.RawQuery = out.Encode()
		http.Redirect(w, r, back.String(), http.StatusFound) //nolint:gosec // G710: a fake IdP redirects wherever its test client asks
		return
	}
	if q.Get("client_id") != i.ClientID {
		http.Error(w, "unknown client", http.StatusBadRequest)
		return
	}
	code := randomCode()
	i.mu.Lock()
	i.codes[code] = grant{claims: claims, nonce: q.Get("nonce"), challenge: q.Get("code_challenge"), redirectURI: redirect, clientID: q.Get("client_id")}
	i.mu.Unlock()
	out.Set("code", code)
	back.RawQuery = out.Encode()
	http.Redirect(w, r, back.String(), http.StatusFound) //nolint:gosec // G710: a fake IdP redirects wherever its test client asks
}

func (i *IdP) token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	id, secret, ok := r.BasicAuth()
	if !ok {
		id, secret = r.PostForm.Get("client_id"), r.PostForm.Get("client_secret")
	}
	if id != i.ClientID || secret != i.ClientSecret {
		w.WriteHeader(http.StatusUnauthorized)
		writeJSON(w, map[string]any{"error": "invalid_client", "error_description": "Client authentication failed"})
		return
	}
	i.mu.Lock()
	g, found := i.codes[r.PostForm.Get("code")]
	delete(i.codes, r.PostForm.Get("code"))
	i.mu.Unlock()
	if !found || g.redirectURI != r.PostForm.Get("redirect_uri") {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]any{"error": "invalid_grant", "error_description": "Unknown or used code"})
		return
	}
	sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
	if g.challenge == "" || base64.RawURLEncoding.EncodeToString(sum[:]) != g.challenge {
		w.WriteHeader(http.StatusBadRequest)
		writeJSON(w, map[string]any{"error": "invalid_grant", "error_description": "PKCE verification failed"})
		return
	}
	nonce := g.nonce
	if i.Nonce != "" {
		nonce = i.Nonce
	}
	claims := map[string]any{"iss": i.issuer(), "aud": g.clientID, "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "nonce": nonce}
	for k, v := range g.claims {
		claims[k] = v
	}
	raw, _ := json.Marshal(claims)
	writeJSON(w, map[string]any{"access_token": "at-" + randomCode(), "token_type": "Bearer", "expires_in": 3600,
		"id_token": oidctest.SignIDToken(i.key, "k1", "RS256", string(raw))})
}

// GitHub is a fake GitHub OAuth app and API.
type GitHub struct {
	Server       *httptest.Server
	ClientID     string
	ClientSecret string
	// User is the signed-in GitHub user; Emails their addresses; Orgs the
	// organizations they are an active member of.
	User   map[string]any
	Emails []map[string]any
	Orgs   []string
	// OrgStatus overrides the membership answer (403 for app restrictions).
	OrgStatus int

	mu    sync.Mutex
	codes map[string]string // code → PKCE challenge
}

// NewGitHub starts a fake GitHub.
func NewGitHub() *GitHub {
	g := &GitHub{ClientID: "gh-client", ClientSecret: "gh-secret", codes: map[string]string{},
		User:   map[string]any{"id": 4242, "login": "octo", "name": "Octo Cat"},
		Emails: []map[string]any{{"email": "octo@example.com", "primary": true, "verified": true}}}
	g.Server = httptest.NewServer(http.HandlerFunc(g.serve))
	return g
}

// Close stops the server.
func (g *GitHub) Close() { g.Server.Close() }

// AuthURL, TokenURL and API are the endpoints to configure.
func (g *GitHub) AuthURL() string  { return g.Server.URL + "/login/oauth/authorize" }
func (g *GitHub) TokenURL() string { return g.Server.URL + "/login/oauth/access_token" }
func (g *GitHub) API() string      { return g.Server.URL + "/api" }

func (g *GitHub) serve(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/login/oauth/authorize":
		q := r.URL.Query()
		back, _ := url.Parse(q.Get("redirect_uri"))
		code := randomCode()
		g.mu.Lock()
		g.codes[code] = q.Get("code_challenge")
		g.mu.Unlock()
		out := back.Query()
		out.Set("code", code)
		out.Set("state", q.Get("state"))
		back.RawQuery = out.Encode()
		http.Redirect(w, r, back.String(), http.StatusFound) //nolint:gosec // G710: a fake IdP redirects wherever its test client asks
	case r.URL.Path == "/login/oauth/access_token":
		_ = r.ParseForm()
		g.mu.Lock()
		challenge, ok := g.codes[r.PostForm.Get("code")]
		delete(g.codes, r.PostForm.Get("code"))
		g.mu.Unlock()
		sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
		if !ok || r.PostForm.Get("client_secret") != g.ClientSecret || base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
			writeJSON(w, map[string]any{"error": "bad_verification_code"})
			return
		}
		writeJSON(w, map[string]any{"access_token": "gho_" + randomCode(), "token_type": "bearer", "scope": "read:user,user:email,read:org"})
	case r.URL.Path == "/api/user":
		writeJSON(w, g.User)
	case r.URL.Path == "/api/user/emails":
		writeJSON(w, g.Emails)
	case strings.HasPrefix(r.URL.Path, "/api/user/memberships/orgs/"):
		if g.OrgStatus != 0 {
			w.WriteHeader(g.OrgStatus)
			writeJSON(w, map[string]any{"message": "restricted"})
			return
		}
		org := strings.TrimPrefix(r.URL.Path, "/api/user/memberships/orgs/")
		for _, o := range g.Orgs {
			if strings.EqualFold(o, org) {
				writeJSON(w, map[string]any{"state": "active", "role": "member"})
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		writeJSON(w, map[string]any{"message": "Not Found"})
	default:
		http.NotFound(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func randomCode() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
