package azureacs

import (
	"cmp"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
)

// EventGridRole is the app role Event Grid's service principal holds on a
// Microsoft Entra ID protected webhook application; its tokens carry it.
const EventGridRole = "AzureEventGridSecureWebhookSubscriber"

// skew is the clock difference tolerated on token times.
const skew = 5 * time.Minute

// verifyEntraToken checks the bearer token Event Grid sends to a webhook
// protected with Microsoft Entra ID: an RS256 JWT signed with one of the
// tenant's keys, issued by the tenant (v1 or v2 issuer), for the webhook
// application (audience), with the Event Grid subscriber role, not
// expired. Keys are fetched from the tenant's discovery endpoint and cached.
func (p *Provider) verifyEntraToken(ctx context.Context, token string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("%w: not a JWT", providers.ErrBadSignature)
	}
	var head struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := decodeSegment(parts[0], &head); err != nil || head.Alg != "RS256" || head.Kid == "" {
		return fmt.Errorf("%w: unexpected token header", providers.ErrBadSignature)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("%w: bad token signature", providers.ErrBadSignature)
	}
	tenant := p.eventGridTenant()
	key, err := p.signingKey(ctx, tenant, head.Kid)
	if err != nil {
		return fmt.Errorf("%w: %w", providers.ErrBadSignature, err)
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], sig) != nil {
		return fmt.Errorf("%w: the token signature doesn't match", providers.ErrBadSignature)
	}
	var claims struct {
		Iss   string   `json:"iss"`
		Aud   string   `json:"aud"`
		Tid   string   `json:"tid"`
		Exp   int64    `json:"exp"`
		Nbf   int64    `json:"nbf"`
		Roles []string `json:"roles"`
	}
	if err := decodeSegment(parts[1], &claims); err != nil {
		return fmt.Errorf("%w: unreadable token claims", providers.ErrBadSignature)
	}
	n := now()
	switch {
	case claims.Exp == 0 || n.After(time.Unix(claims.Exp, 0).Add(skew)):
		return fmt.Errorf("%w: the token has expired", providers.ErrBadSignature)
	case claims.Nbf != 0 && n.Before(time.Unix(claims.Nbf, 0).Add(-skew)):
		return fmt.Errorf("%w: the token isn't valid yet", providers.ErrBadSignature)
	case !strings.EqualFold(claims.Tid, tenant) ||
		(claims.Iss != "https://sts.windows.net/"+claims.Tid+"/" && claims.Iss != "https://login.microsoftonline.com/"+claims.Tid+"/v2.0"):
		return fmt.Errorf("%w: the token was issued by another tenant", providers.ErrBadSignature)
	case !audienceMatches(claims.Aud, p.s.EventGridAudience):
		return fmt.Errorf("%w: the token is for another application", providers.ErrBadSignature)
	case !slices.Contains(claims.Roles, EventGridRole):
		return fmt.Errorf("%w: the token lacks the %s role", providers.ErrBadSignature, EventGridRole)
	}
	return nil
}

// audienceMatches accepts the application ID or its "api://" ID URI, as
// Event Grid takes either.
func audienceMatches(aud, want string) bool {
	if aud == "" || want == "" {
		return false
	}
	norm := func(s string) string {
		return strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(s, "api://"), "/"))
	}
	return norm(aud) == norm(want)
}

func (p *Provider) eventGridTenant() string {
	return cmp.Or(p.s.EventGridTenantID, p.s.TenantID)
}

func decodeSegment(seg string, v any) error {
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(seg, "="))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// keySet caches one tenant's signing keys, fetched at most once a minute
// when a token names a key not seen yet.
type keySet struct {
	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

var (
	keySetsMu sync.Mutex
	keySets   = map[string]*keySet{}
)

func (p *Provider) signingKey(ctx context.Context, tenant, kid string) (*rsa.PublicKey, error) {
	if tenant == "" {
		return nil, errors.New("no Entra ID tenant is configured for Event Grid")
	}
	authority := strings.TrimRight(cmp.Or(p.s.AuthorityURL, DefaultAuthority), "/")
	id := authority + "|" + tenant
	keySetsMu.Lock()
	ks, ok := keySets[id]
	if !ok {
		ks = &keySet{keys: map[string]*rsa.PublicKey{}}
		keySets[id] = ks
	}
	keySetsMu.Unlock()
	ks.mu.Lock()
	defer ks.mu.Unlock()
	if k, ok := ks.keys[kid]; ok {
		return k, nil
	}
	if time.Since(ks.fetched) < time.Minute {
		return nil, fmt.Errorf("unknown signing key %q", kid)
	}
	ks.fetched = time.Now()
	c := apiclient.New(authority, p.http)
	var set struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := c.Do(ctx, "GET /discovery/v2.0/keys", http.MethodGet, "/"+url.PathEscape(tenant)+"/discovery/v2.0/keys", nil, &set); err != nil {
		return nil, err
	}
	for _, k := range set.Keys {
		if k.Kty != "RSA" || k.Kid == "" {
			continue
		}
		n, err1 := base64.RawURLEncoding.DecodeString(strings.TrimRight(k.N, "="))
		e, err2 := base64.RawURLEncoding.DecodeString(strings.TrimRight(k.E, "="))
		if err1 != nil || err2 != nil || len(e) > 4 {
			continue
		}
		ks.keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	if k, ok := ks.keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("unknown signing key %q", kid)
}
