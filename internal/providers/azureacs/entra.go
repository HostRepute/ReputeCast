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
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

// EventGridRole is the app role Event Grid's service principal holds on a
// Microsoft Entra ID protected webhook application; its tokens carry it.
const EventGridRole = "AzureEventGridSecureWebhookSubscriber"

// skew is the clock difference tolerated on token times.
const skew = 5 * time.Minute

// verifyEntraToken checks the RS256 JWT Event Grid sends to a webhook
// protected with Microsoft Entra ID. The tenant's key set is cached, and read
// again for an unseen key and every webhookauth.MaxAge so keys Microsoft
// retired stop being trusted.
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
	if tenant == "" {
		return fmt.Errorf("%w: no Entra ID tenant is configured for Event Grid", providers.ErrBadSignature)
	}
	authority := strings.TrimRight(cmp.Or(p.s.AuthorityURL, DefaultAuthority), "/")
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	err = keySets.Verify(ctx, webhookauth.Key(authority, tenant), func(ctx context.Context) (string, error) {
		return p.fetchKeySet(ctx, authority, tenant)
	}, func(set string) error {
		key, err := findKey(set, head.Kid)
		if err != nil {
			return err
		}
		if rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], sig) != nil {
			return fmt.Errorf("%w: the token signature doesn't match", providers.ErrBadSignature)
		}
		return nil
	})
	if err != nil {
		return err
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

// keySets caches each tenant's key set (the JSON Web Key Set document),
// shared by every adapter instance.
var keySets webhookauth.Material

// jwks is a JSON Web Key Set.
type jwks struct {
	Keys []struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

// fetchKeySet reads the tenant's signing keys from its discovery endpoint.
func (p *Provider) fetchKeySet(ctx context.Context, authority, tenant string) (string, error) {
	res, err := apiclient.New(authority, p.http).Exec(ctx, apiclient.Request{Op: "GET /discovery/v2.0/keys", Method: http.MethodGet,
		Path: "/" + url.PathEscape(tenant) + "/discovery/v2.0/keys"})
	if err != nil {
		return "", err
	}
	var set jwks
	if err := json.Unmarshal(res.Body, &set); err != nil || len(set.Keys) == 0 {
		return "", errors.New("the tenant's key set is empty or unreadable")
	}
	return string(res.Body), nil
}

// findKey returns the RSA key kid of a key set.
func findKey(set, kid string) (*rsa.PublicKey, error) {
	var ks jwks
	if err := json.Unmarshal([]byte(set), &ks); err != nil {
		return nil, fmt.Errorf("%w: unreadable key set", providers.ErrBadSignature)
	}
	for _, k := range ks.Keys {
		if k.Kty != "RSA" || k.Kid != kid {
			continue
		}
		n, err1 := base64.RawURLEncoding.DecodeString(strings.TrimRight(k.N, "="))
		e, err2 := base64.RawURLEncoding.DecodeString(strings.TrimRight(k.E, "="))
		if err1 != nil || err2 != nil || len(n) == 0 || len(e) == 0 || len(e) > 4 {
			break
		}
		return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}, nil
	}
	return nil, fmt.Errorf("%w: unknown signing key %q", providers.ErrBadSignature, kid)
}
