package sso

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// cachedProvider is a discovered issuer kept for an hour; go-oidc caches
// the signing keys of each provider itself.
type cachedProvider struct {
	provider *oidc.Provider
	doc      discovery
	at       time.Time
}

const discoveryTTL = time.Hour

// providers caches discovered issuers for every Service of the process
// (the API and the callbacks build their own).
var providers struct {
	mu sync.Mutex
	m  map[string]cachedProvider
}

// discover fetches {issuer}/.well-known/openid-configuration and reports
// what the setup screen shows: status, latency, endpoints and key count.
func (s *Service) discover(ctx context.Context, issuer string) discovery {
	d := discovery{CheckedAt: s.now()}
	issuer = strings.TrimSpace(issuer)
	u, err := url.Parse(issuer)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		d.Error = "Enter the issuer as a full URL, like https://login.microsoftonline.com/<tenant-id>/v2.0."
		return d
	}
	if u.Scheme == "http" && !isLocalHost(u.Hostname()) {
		d.Error = "The issuer must use https. Plain http is only accepted for localhost."
		return d
	}
	d.URL = strings.TrimRight(issuer, "/") + "/.well-known/openid-configuration"
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, d.URL, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := s.client().Do(req)
	d.Millis = time.Since(start).Milliseconds()
	if err != nil {
		d.Error = "The issuer didn't answer: " + networkError(err)
		return d
	}
	defer func() { _ = resp.Body.Close() }()
	d.Status = resp.StatusCode
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		d.Error = fmt.Sprintf("returned %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		return d
	}
	var doc discovery
	if err := json.Unmarshal(body, &doc); err != nil {
		d.Error = "returned something that isn't a discovery document (invalid JSON)."
		return d
	}
	d.Issuer, d.AuthURL, d.TokenURL, d.UserInfo, d.JWKSURL, d.Algs = doc.Issuer, doc.AuthURL, doc.TokenURL, doc.UserInfo, doc.JWKSURL, doc.Algs
	switch {
	case d.Issuer == "":
		d.Error = "The discovery document has no issuer."
		return d
	case !sameIssuer(d.Issuer, issuer) && !strings.Contains(d.Issuer, "{tenantid}"):
		d.Error = fmt.Sprintf("The document's issuer is %s, not %s. Use exactly the issuer it reports.", d.Issuer, issuer)
		return d
	case d.AuthURL == "" || d.TokenURL == "" || d.JWKSURL == "":
		d.Error = "The discovery document is missing the authorization, token or JWKS endpoint."
		return d
	}
	// The signing keys must be reachable too, or no token can be verified.
	kreq, _ := http.NewRequestWithContext(ctx, http.MethodGet, d.JWKSURL, nil)
	kresp, err := s.client().Do(kreq)
	if err != nil {
		d.Error = "The JWKS endpoint didn't answer: " + networkError(err)
		return d
	}
	defer func() { _ = kresp.Body.Close() }()
	var keys struct {
		Keys []json.RawMessage `json:"keys"`
	}
	kb, _ := io.ReadAll(io.LimitReader(kresp.Body, 1<<20))
	if kresp.StatusCode != http.StatusOK || json.Unmarshal(kb, &keys) != nil || len(keys.Keys) == 0 {
		d.Error = fmt.Sprintf("The JWKS endpoint returned %d without signing keys.", kresp.StatusCode)
		return d
	}
	d.Keys = len(keys.Keys)
	d.OK = true
	return d
}

func sameIssuer(a, b string) bool {
	return strings.TrimRight(a, "/") == strings.TrimRight(b, "/")
}

func isLocalHost(h string) bool {
	return h == "localhost" || h == "127.0.0.1" || h == "::1" || strings.HasSuffix(h, ".localhost")
}

func networkError(err error) string {
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "no answer within 8 seconds."
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "no such host"):
		return "the host name doesn't resolve (DNS)."
	case strings.Contains(msg, "connection refused"):
		return "the connection was refused."
	case strings.Contains(msg, "certificate"):
		return "its TLS certificate isn't valid (" + msg + ")."
	}
	return msg + "."
}

// provider returns a go-oidc provider for the issuer, discovering it when
// it isn't cached. With loose set, the document's issuer isn't compared
// with the URL: Microsoft's endpoints report a tenant (or a template)
// instead, and the caller checks each token's issuer against its tenant.
func (s *Service) provider(ctx context.Context, issuer string, loose bool) (*oidc.Provider, error) {
	key := issuer + "|" + fmt.Sprint(loose)
	providers.mu.Lock()
	if c, ok := providers.m[key]; ok && s.now().Sub(c.at) < discoveryTTL {
		providers.mu.Unlock()
		return c.provider, nil
	}
	providers.mu.Unlock()
	cctx := oidc.ClientContext(ctx, s.client())
	var p *oidc.Provider
	var err error
	if loose {
		d := s.discover(ctx, issuer)
		if d.Issuer == "" || d.AuthURL == "" || d.TokenURL == "" || d.JWKSURL == "" {
			return nil, errors.New(firstNonEmpty(d.Error, "discovery failed"))
		}
		p = (&oidc.ProviderConfig{IssuerURL: d.Issuer, AuthURL: d.AuthURL, TokenURL: d.TokenURL, UserInfoURL: d.UserInfo, JWKSURL: d.JWKSURL, Algorithms: d.Algs}).NewProvider(cctx)
	} else {
		p, err = oidc.NewProvider(cctx, strings.TrimRight(issuer, "/"))
		if err != nil {
			// Retry with the trailing slash some issuers (Authentik) use.
			if p2, err2 := oidc.NewProvider(cctx, strings.TrimRight(issuer, "/")+"/"); err2 == nil {
				p, err = p2, nil
			}
		}
		if err != nil {
			return nil, err
		}
	}
	providers.mu.Lock()
	if providers.m == nil {
		providers.m = map[string]cachedProvider{}
	}
	providers.m[key] = cachedProvider{provider: p, at: s.now()}
	providers.mu.Unlock()
	return p, nil
}

// forget drops a cached issuer after its settings change.
func (s *Service) forget(issuer string) {
	providers.mu.Lock()
	defer providers.mu.Unlock()
	for k := range providers.m {
		if strings.HasPrefix(k, issuer+"|") {
			delete(providers.m, k)
		}
	}
}

// exchange trades the authorization code for tokens (with the PKCE
// verifier) and verifies the ID token: signature, issuer, audience, expiry
// and nonce. It returns the ID token's claims merged with userinfo claims
// the token doesn't carry.
func (s *Service) exchange(ctx context.Context, p *oidc.Provider, conf *oauth2.Config, code, verifier, nonce string, skipIssuer bool) (map[string]any, error) {
	cctx := oidc.ClientContext(ctx, s.client())
	tok, err := conf.Exchange(cctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, &flowError{Code: "token_exchange_failed", Detail: "The token exchange failed: " + oauthError(err)}
	}
	raw, ok := tok.Extra("id_token").(string)
	if !ok || raw == "" {
		return nil, &flowError{Code: "no_id_token", Detail: "The identity provider returned no ID token. Check that the openid scope is requested."}
	}
	// The provider's shared key set caches the signing keys between sign-ins.
	v := p.Verifier(&oidc.Config{ClientID: conf.ClientID, SkipIssuerCheck: skipIssuer, Now: s.now})
	idt, err := v.Verify(cctx, raw)
	if err != nil {
		return nil, &flowError{Code: "invalid_id_token", Detail: "The ID token didn't verify: " + err.Error()}
	}
	if idt.Nonce != nonce {
		return nil, &flowError{Code: "nonce_mismatch", Detail: "The ID token's nonce doesn't match this sign-in. Start again."}
	}
	claims := map[string]any{}
	if err := idt.Claims(&claims); err != nil {
		return nil, &flowError{Code: "invalid_id_token", Detail: "The ID token's claims couldn't be read."}
	}
	// Some IdPs (Entra ID without optional claims, Keycloak with lean
	// tokens) keep email or groups in userinfo only.
	if p.UserInfoEndpoint() != "" {
		if ui, err := p.UserInfo(cctx, oauth2.StaticTokenSource(tok)); err == nil && ui.Subject == idt.Subject {
			extra := map[string]any{}
			if ui.Claims(&extra) == nil {
				for k, val := range extra {
					if _, ok := claims[k]; !ok {
						claims[k] = val
					}
				}
			}
		}
	}
	return claims, nil
}

func oauthError(err error) string {
	var re *oauth2.RetrieveError
	if errors.As(err, &re) {
		if re.ErrorCode != "" {
			msg := re.ErrorCode
			if re.ErrorDescription != "" {
				msg += ": " + firstLine(re.ErrorDescription)
			}
			return msg
		}
		return fmt.Sprintf("HTTP %d", re.Response.StatusCode)
	}
	return err.Error()
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}

// flowError is a sign-in failure with a stable code (shown on the error
// page with a reference id) and an explanation for the audit log.
type flowError struct {
	Code   string
	Detail string
	// NoAccount sends the browser to "No account for this email" instead
	// of the IdP error page.
	NoAccount bool
	Email     string
}

func (e *flowError) Error() string { return e.Detail }

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
