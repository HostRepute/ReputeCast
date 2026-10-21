package shopify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/stores"
)

// The install: the merchant creates a custom app in Shopify's Dev
// Dashboard with ReputeCast's app URL and redirect URL, pastes its client
// ID and secret, then approves it on Shopify's own page (OAuth's
// authorization code grant). ReputeCast never sees the merchant's Shopify
// password.

// AuthorizeURL is Shopify's approval page for the app on a store.
func AuthorizeURL(domain, clientID, redirect, state string, scopes []string) string {
	q := url.Values{}
	q.Set("client_id", clientID)
	q.Set("scope", strings.Join(scopes, ","))
	q.Set("redirect_uri", redirect)
	q.Set("state", state)
	return "https://" + stores.NormalizeHost(domain) + "/admin/oauth/authorize?" + q.Encode()
}

// SignQuery is the hmac Shopify adds to the query strings it sends to the
// app URL and the redirect URL: hex HMAC-SHA256 of the other parameters,
// sorted, joined with "&" (tests and the fake).
func SignQuery(q url.Values, secret string) string {
	keys := make([]string, 0, len(q))
	for k := range q {
		if k == "hmac" || k == "signature" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		vals := slices.Clone(q[k])
		parts = append(parts, k+"="+strings.Join(vals, ","))
	}
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(strings.Join(parts, "&")))
	return hex.EncodeToString(m.Sum(nil))
}

// VerifyQuery checks the hmac of a query string Shopify sent, and that its
// timestamp is recent (a link replayed later is refused).
func VerifyQuery(q url.Values, secret string, now time.Time) bool {
	got := q.Get("hmac")
	if got == "" || secret == "" {
		return false
	}
	want := SignQuery(q, secret)
	if !hmac.Equal([]byte(strings.ToLower(got)), []byte(want)) {
		return false
	}
	if ts := q.Get("timestamp"); ts != "" {
		var sec int64
		if _, err := fmt.Sscan(ts, &sec); err != nil {
			return false
		}
		at := time.Unix(sec, 0)
		if now.Sub(at) > 24*time.Hour || at.Sub(now) > 10*time.Minute {
			return false
		}
	}
	return true
}

// ValidShopDomain reports whether host is a myshopify.com store domain, as
// Shopify names them in its redirects (letters, digits and hyphens).
func ValidShopDomain(host string) bool {
	h := stores.NormalizeHost(host)
	name, ok := strings.CutSuffix(h, ".myshopify.com")
	if !ok || name == "" || len(name) > 60 {
		return false
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

// Token is what the code exchange returns.
type Token struct {
	AccessToken string
	Scopes      []string
	// RefreshToken and ExpiresAt are set when Shopify issues an expiring
	// offline token.
	RefreshToken string
	ExpiresAt    *time.Time
}

// Exchange trades the approval's code for the store's access token. A
// refused client secret is stores.ErrUnauthorized (the dialog's "Shopify
// refused this client secret").
func Exchange(ctx context.Context, cfg stores.Config, clientID, secret, code string) (Token, error) {
	base := "https://" + stores.NormalizeHost(cfg.Domain)
	if cfg.BaseURL != "" {
		base = strings.TrimRight(cfg.BaseURL, "/")
	}
	hc := cfg.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 20 * time.Second}
	}
	now := time.Now
	if cfg.Now != nil {
		now = cfg.Now
	}
	body, _ := json.Marshal(map[string]string{"client_id": clientID, "client_secret": secret, "code": code})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/admin/oauth/access_token", bytes.NewReader(body))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		return Token{}, fmt.Errorf("%w: %w", stores.ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	switch {
	case resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return Token{}, fmt.Errorf("%w: %d %s", stores.ErrUnauthorized, resp.StatusCode, snippet(raw))
	case resp.StatusCode >= 300:
		return Token{}, statusError(resp, raw)
	}
	var t struct {
		AccessToken           string `json:"access_token"`
		Scope                 string `json:"scope"`
		RefreshToken          string `json:"refresh_token"`
		ExpiresIn             int64  `json:"expires_in"`
		RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in"`
	}
	if err := json.Unmarshal(raw, &t); err != nil || t.AccessToken == "" {
		return Token{}, errors.New("shopify: the code exchange returned no access token")
	}
	out := Token{AccessToken: t.AccessToken, RefreshToken: t.RefreshToken}
	for _, s := range strings.Split(t.Scope, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out.Scopes = append(out.Scopes, s)
		}
	}
	if t.ExpiresIn > 0 {
		at := now().Add(time.Duration(t.ExpiresIn) * time.Second)
		out.ExpiresAt = &at
	}
	return out, nil
}

// Refresh renews an expiring offline token.
func Refresh(ctx context.Context, cfg stores.Config, clientID, secret, refresh string) (Token, error) {
	base := "https://" + stores.NormalizeHost(cfg.Domain)
	if cfg.BaseURL != "" {
		base = strings.TrimRight(cfg.BaseURL, "/")
	}
	hc := cfg.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 20 * time.Second}
	}
	now := time.Now
	if cfg.Now != nil {
		now = cfg.Now
	}
	body, _ := json.Marshal(map[string]string{"client_id": clientID, "client_secret": secret, "grant_type": "refresh_token", "refresh_token": refresh})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/admin/oauth/access_token", bytes.NewReader(body))
	if err != nil {
		return Token{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		return Token{}, fmt.Errorf("%w: %w", stores.ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return Token{}, fmt.Errorf("%w: %d %s", stores.ErrUnauthorized, resp.StatusCode, snippet(raw))
	}
	if err := statusError(resp, raw); err != nil {
		return Token{}, err
	}
	var t struct {
		AccessToken  string `json:"access_token"`
		Scope        string `json:"scope"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(raw, &t); err != nil || t.AccessToken == "" {
		return Token{}, errors.New("shopify: the refresh returned no access token")
	}
	out := Token{AccessToken: t.AccessToken, RefreshToken: t.RefreshToken}
	if out.RefreshToken == "" {
		out.RefreshToken = refresh
	}
	for _, s := range strings.Split(t.Scope, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out.Scopes = append(out.Scopes, s)
		}
	}
	if t.ExpiresIn > 0 {
		at := now().Add(time.Duration(t.ExpiresIn) * time.Second)
		out.ExpiresAt = &at
	}
	return out, nil
}

// Storefront is what a store's public address says about it.
type Storefront struct {
	Name     string
	Domain   string // acme-merch.myshopify.com
	Currency string
}

// Resolve checks that an address answers like a Shopify store and finds
// its myshopify.com domain: a custom domain (shop.acme.io) answers
// /meta.json with it. A myshopify.com address is taken as it is when the
// storefront can't be read (a password-protected store).
func Resolve(ctx context.Context, cfg stores.Config, address string) (Storefront, error) {
	host := stores.NormalizeHost(address)
	if host == "" || !strings.Contains(host, ".") {
		return Storefront{}, stores.ErrNotAStore
	}
	base := "https://" + host
	if cfg.BaseURL != "" {
		base = strings.TrimRight(cfg.BaseURL, "/")
	}
	hc := cfg.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/meta.json", nil)
	if err != nil {
		return Storefront{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ReputeCast")
	resp, err := hc.Do(req)
	if err == nil {
		defer func() { _ = resp.Body.Close() }()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if resp.StatusCode == http.StatusOK {
			var m struct {
				Name            string `json:"name"`
				Currency        string `json:"currency"`
				MyshopifyDomain string `json:"myshopify_domain"`
			}
			if json.Unmarshal(raw, &m) == nil && ValidShopDomain(m.MyshopifyDomain) {
				return Storefront{Name: m.Name, Domain: stores.NormalizeHost(m.MyshopifyDomain), Currency: m.Currency}, nil
			}
		}
	}
	if ValidShopDomain(host) {
		return Storefront{Domain: host}, nil
	}
	if err != nil {
		return Storefront{}, fmt.Errorf("%w: %w", stores.ErrNotAStore, err)
	}
	return Storefront{}, stores.ErrNotAStore
}
