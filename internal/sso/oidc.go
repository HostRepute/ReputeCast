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

	"github.com/hostrepute/reputecast/internal/i18n"
)

// cachedProvider is a discovered issuer kept for an hour; go-oidc caches
// the signing keys of each provider itself.
type cachedProvider struct {
	provider *oidc.Provider
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
		return d.failed("discovery.url", i18n.Args{"example": "https://login.microsoftonline.com/<tenant-id>/v2.0"})
	}
	if u.Scheme == "http" && !isLocalHost(u.Hostname()) {
		return d.failed("discovery.https", nil)
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
		return d.failed("discovery.noAnswer", networkError(err))
	}
	defer func() { _ = resp.Body.Close() }()
	d.Status = resp.StatusCode
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return d.failed("discovery.status", i18n.Args{"status": resp.StatusCode, "statusText": http.StatusText(resp.StatusCode)})
	}
	var doc discovery
	if err := json.Unmarshal(body, &doc); err != nil {
		return d.failed("discovery.json", nil)
	}
	d.Issuer, d.AuthURL, d.TokenURL, d.UserInfo, d.JWKSURL, d.Algs = doc.Issuer, doc.AuthURL, doc.TokenURL, doc.UserInfo, doc.JWKSURL, doc.Algs
	switch {
	case d.Issuer == "":
		return d.failed("discovery.noIssuer", nil)
	case !sameIssuer(d.Issuer, issuer) && !strings.Contains(d.Issuer, "{tenantid}"):
		return d.failed("discovery.issuerMismatch", i18n.Args{"reported": d.Issuer, "entered": issuer})
	case d.AuthURL == "" || d.TokenURL == "" || d.JWKSURL == "":
		return d.failed("discovery.endpoints", nil)
	}
	// The signing keys must be reachable too, or no token can be verified.
	kreq, _ := http.NewRequestWithContext(ctx, http.MethodGet, d.JWKSURL, nil)
	kresp, err := s.client().Do(kreq)
	if err != nil {
		return d.failed("discovery.jwksNoAnswer", networkError(err))
	}
	defer func() { _ = kresp.Body.Close() }()
	var keys struct {
		Keys []json.RawMessage `json:"keys"`
	}
	kb, _ := io.ReadAll(io.LimitReader(kresp.Body, 1<<20))
	if kresp.StatusCode != http.StatusOK || json.Unmarshal(kb, &keys) != nil || len(keys.Keys) == 0 {
		return d.failed("discovery.jwksKeys", i18n.Args{"status": kresp.StatusCode})
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

// failed records why a discovery test failed: the English, and the
// message kept for later readers.
func (d discovery) failed(key string, args i18n.Args) discovery {
	d.ErrorText = keep(key, args)
	d.Error = d.ErrorText.In(i18n.Default)
	return d
}

// networkError is why a request got no answer, as the arguments of
// discovery.noAnswer: why (timeout, dns, refused, tls or other) and the
// library's error.
func networkError(err error) i18n.Args {
	var ue *url.Error
	if errors.As(err, &ue) {
		err = ue.Err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return i18n.Args{"why": "timeout", "error": ""}
	}
	msg := err.Error()
	why := "other"
	switch {
	case strings.Contains(msg, "no such host"):
		why = "dns"
	case strings.Contains(msg, "connection refused"):
		why = "refused"
	case strings.Contains(msg, "certificate"):
		why = "tls"
	}
	return i18n.Args{"why": why, "error": msg}
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
		return nil, failure("token_exchange_failed", "flow.tokenExchange", i18n.Args{"error": oauthError(err)})
	}
	raw, ok := tok.Extra("id_token").(string)
	if !ok || raw == "" {
		return nil, failure("no_id_token", "flow.noIDToken", nil)
	}
	// The provider's shared key set caches the signing keys between sign-ins.
	v := p.Verifier(&oidc.Config{ClientID: conf.ClientID, SkipIssuerCheck: skipIssuer, Now: s.now})
	idt, err := v.Verify(cctx, raw)
	if err != nil {
		return nil, failure("invalid_id_token", "flow.idTokenInvalid", i18n.Args{"error": err.Error()})
	}
	if idt.Nonce != nonce {
		return nil, failure("nonce_mismatch", "flow.nonce", nil)
	}
	claims := map[string]any{}
	if err := idt.Claims(&claims); err != nil {
		return nil, failure("invalid_id_token", "flow.claims", nil)
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
// page with a reference id) and an explanation for the audit log: Detail
// in English, Text the message kept for later readers (none when the
// explanation came from elsewhere, like another package's problem).
type flowError struct {
	Code   string
	Detail string
	Text   i18n.Text
	// NoAccount sends the browser to "No account for this email" instead
	// of the IdP error page.
	NoAccount bool
	Email     string
}

func (e *flowError) Error() string { return e.Detail }

// failure is a flowError explained by one of the package's messages.
func failure(code, key string, args i18n.Args) *flowError {
	t := keep(key, args)
	return &flowError{Code: code, Detail: t.In(i18n.Default), Text: t}
}

// about names the email the failure is about.
func (e *flowError) about(email string) *flowError {
	e.Email = email
	return e
}

// noAccount sends the browser to "No account for this email".
func (e *flowError) noAccount(email string) *flowError {
	e.NoAccount, e.Email = true, email
	return e
}

// in is the explanation in a language.
func (e *flowError) in(l i18n.Lang) string { return textIn(e.Text, l, e.Detail) }

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
