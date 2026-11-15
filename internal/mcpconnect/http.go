package mcpconnect

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/store"
)

// RegisterPublic mounts only protocol endpoints. Human consent uses the normal
// session/CSRF/MFA API middleware, never bearer credentials from this transport.
func (s *Service) RegisterPublic(r chi.Router) {
	r.Get("/.well-known/oauth-authorization-server", s.metadata)
	r.Get("/.well-known/oauth-protected-resource/mcp/connect", s.metadata)
	r.Post("/oauth/mcp/register", s.register)
	r.Get("/oauth/mcp/authorize", s.authorize)
	r.Post("/oauth/mcp/token", s.exchange)
	r.Post("/oauth/mcp/revoke", s.revokeToken)
	r.Handle(Path, s.Handler())
}

func noStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func oauthError(w http.ResponseWriter, err error) {
	noStore(w)
	code, status := "server_error", http.StatusInternalServerError
	switch err.Error() {
	case "invalid_request", "invalid_scope", "invalid_grant", "unsupported_grant_type":
		code, status = err.Error(), http.StatusBadRequest
	case "invalid_client":
		code, status = "invalid_client", http.StatusUnauthorized
		w.Header().Set("WWW-Authenticate", `Basic realm="mcp"`)
	}
	// No raw parser/database/client error can echo codes, credentials or input.
	writeJSON(w, status, map[string]string{"error": code})
}

func (s *Service) metadata(w http.ResponseWriter, r *http.Request) {
	origin, err := s.origin()
	if err != nil {
		oauthError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	if r.URL.Path == "/.well-known/oauth-protected-resource/mcp/connect" {
		writeJSON(w, 200, map[string]any{"resource": origin + Path, "authorization_servers": []string{origin}, "scopes_supported": capabilities, "bearer_methods_supported": []string{"header"}})
		return
	}
	writeJSON(w, 200, map[string]any{
		"issuer": origin, "authorization_endpoint": origin + "/oauth/mcp/authorize", "token_endpoint": origin + "/oauth/mcp/token",
		"registration_endpoint": origin + "/oauth/mcp/register", "revocation_endpoint": origin + "/oauth/mcp/revoke",
		"response_types_supported": []string{"code"}, "grant_types_supported": []string{"authorization_code", "refresh_token"},
		"token_endpoint_auth_methods_supported":      []string{"none", "client_secret_basic", "client_secret_post"},
		"revocation_endpoint_auth_methods_supported": []string{"none", "client_secret_basic", "client_secret_post"},
		"code_challenge_methods_supported":           []string{"S256"}, "scopes_supported": capabilities, "authorization_response_iss_parameter_supported": true,
	})
}

// Quotas are shared by replicas, count rejected attempts, and use hashed network
// keys. The network's own quota is checked first, so attempts it refuses never
// use up the global one. Only kinds that create rows (registration, consent
// requests) have a global quota: one network must not be able to refuse token
// refresh or revocation to every other client.
func (s *Service) limit(w http.ResponseWriter, r *http.Request, kind string, perNetwork, global int32) bool {
	ip := "unknown"
	if addr := audit.RequestFrom(r.Context()).IP; addr != nil {
		ip = addr.Unmap().String()
		if addr.Is6() && !addr.Is4In6() {
			ip = netip.PrefixFrom(*addr, 64).Masked().String()
		}
	}
	type quota struct {
		key string
		max int32
	}
	quotas := []quota{{kind + ":" + ip, perNetwork}}
	if global > 0 {
		quotas = append(quotas, quota{kind + ":global", global})
	}
	for _, item := range quotas {
		_, err := s.Q.MCPConnectLimit(r.Context(), store.MCPConnectLimitParams{Key: authn.HashToken(item.key), Bucket: s.now().Truncate(10 * time.Minute), Ceiling: item.max})
		if errors.Is(err, pgx.ErrNoRows) {
			w.Header().Set("Retry-After", "600")
			writeJSON(w, 429, map[string]string{"error": "temporarily_unavailable"})
			return false
		}
		if err != nil {
			oauthError(w, err)
			return false
		}
	}
	return true
}

// decodeObject rejects duplicate top-level fields (including escaped aliases)
// and trailing objects. RFC 7591 requires ignoring unrecognized metadata: do
// not persist, fetch or render it. Known security fields remain case-sensitive.
func decodeObject(r *http.Request, out any) error {
	typeName, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || typeName != "application/json" {
		return errRequest
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, 16385))
	if err != nil || len(b) > 16384 {
		return errRequest
	}
	d := json.NewDecoder(strings.NewReader(string(b)))
	start, err := d.Token()
	if err != nil || start != json.Delim('{') {
		return errRequest
	}
	seen := map[string]bool{}
	for d.More() {
		t, err := d.Token()
		if err != nil {
			return errRequest
		}
		key, ok := t.(string)
		if !ok || seen[key] {
			return errRequest
		}
		seen[key] = true
		for _, known := range []string{"client_name", "redirect_uris", "token_endpoint_auth_method", "grant_types", "response_types"} {
			if key != known && strings.EqualFold(key, known) {
				return errRequest
			}
		}
		var value json.RawMessage
		if err = d.Decode(&value); err != nil {
			return errRequest
		}
	}
	if _, err = d.Token(); err != nil {
		return errRequest
	}
	if _, err = d.Token(); !errors.Is(err, io.EOF) {
		return errRequest
	}
	d = json.NewDecoder(strings.NewReader(string(b)))
	if err = d.Decode(out); err != nil {
		return errRequest
	}
	return nil
}

func (s *Service) register(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if !s.limit(w, r, "registration", 10, 1000) {
		return
	}
	var in ClientRegistration
	if err := decodeObject(r, &in); err != nil {
		oauthError(w, err)
		return
	}
	out, err := s.RegisterClient(r.Context(), in, false)
	if err != nil {
		oauthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func parameters(raw string, allowed []string) (url.Values, error) {
	if len(raw) > 16384 {
		return nil, errRequest
	}
	in, err := url.ParseQuery(raw)
	if err != nil {
		return nil, errRequest
	}
	for k, v := range in {
		if len(v) != 1 || !slices.Contains(allowed, k) {
			return nil, errRequest
		}
	}
	return in, nil
}

func (s *Service) authorize(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if !s.limit(w, r, "authorization", 30, 3000) {
		return
	}
	in, err := parameters(r.URL.RawQuery, []string{"client_id", "redirect_uri", "response_type", "scope", "state", "resource", "code_challenge", "code_challenge_method"})
	if err != nil {
		oauthError(w, err)
		return
	}
	location, err := s.Authorize(r.Context(), in)
	if err != nil {
		oauthError(w, err)
		return
	}
	// Authorize returns the configured origin's fixed consent path and a
	// server-generated handle, never the request's client callback.
	// #nosec G710 -- destination is independent of untrusted redirect_uri.
	http.Redirect(w, r, location, http.StatusSeeOther)
}

func form(r *http.Request) (url.Values, error) {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/x-www-form-urlencoded" || r.URL.RawQuery != "" {
		return nil, errRequest
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, 16385))
	if err != nil {
		return nil, errRequest
	}
	return parameters(string(b), []string{"client_id", "client_secret", "grant_type", "code", "code_verifier", "redirect_uri", "resource", "refresh_token", "scope", "token", "token_type_hint"})
}

func (s *Service) authenticateClient(r *http.Request, in url.Values) (store.McpConnectClient, error) {
	id, secret, method := in.Get("client_id"), in.Get("client_secret"), "none"
	if r.Header.Get("Authorization") != "" {
		if len(r.Header.Values("Authorization")) != 1 || in.Has("client_secret") || in.Has("client_id") {
			return store.McpConnectClient{}, errClient
		}
		i, p, ok := r.BasicAuth()
		if !ok {
			return store.McpConnectClient{}, errClient
		}
		var e1, e2 error
		id, e1 = url.QueryUnescape(i)
		secret, e2 = url.QueryUnescape(p)
		if e1 != nil || e2 != nil {
			return store.McpConnectClient{}, errClient
		}
		method = "client_secret_basic"
	} else if in.Has("client_secret") {
		method = "client_secret_post"
	}
	c, err := s.client(r.Context(), s.Q, id)
	if err != nil || c.AuthMethod != method {
		return c, errClient
	}
	if method != "none" && !authn.EqualHash(c.SecretHash, authn.HashToken(secret)) {
		return c, errClient
	}
	return c, nil
}

func (s *Service) exchange(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if !s.limit(w, r, "token", 120, 0) {
		return
	}
	in, err := form(r)
	if err != nil {
		oauthError(w, err)
		return
	}
	c, err := s.authenticateClient(r, in)
	if err != nil {
		oauthError(w, err)
		return
	}
	out, err := s.Exchange(r.Context(), c, in, audit.RequestFrom(r.Context()).IP)
	if err != nil {
		oauthError(w, err)
		return
	}
	writeJSON(w, 200, out)
}

func (s *Service) revokeToken(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if !s.limit(w, r, "revoke", 120, 0) {
		return
	}
	in, err := form(r)
	if err != nil {
		oauthError(w, err)
		return
	}
	c, err := s.authenticateClient(r, in)
	if err != nil {
		oauthError(w, err)
		return
	}
	if in.Get("token") == "" {
		oauthError(w, errRequest)
		return
	}
	if err = s.RevokeToken(r.Context(), c, in.Get("token")); err != nil {
		oauthError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
