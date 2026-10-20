package webpush

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// VAPID (RFC 8292): the application server signs a JWT with its P-256 key
// and sends it with the public key in the Authorization header. The push
// service checks the signature, the audience (its own origin) and the
// expiry, at most 24 hours ahead.

// MaxTokenLifetime is RFC 8292's limit on "exp": at most 24 hours from the
// time of the request.
const MaxTokenLifetime = 24 * time.Hour

// TokenLifetime is what ReputeCast signs for. Tokens are reused for an hour
// per push service, so a token is always well inside the limit.
const TokenLifetime = 12 * time.Hour

// Claims are a VAPID token's claims: the push service's origin, the expiry
// and the contact address ("mailto:support@acme.io").
type Claims struct {
	Aud string `json:"aud"`
	Exp int64  `json:"exp"`
	Sub string `json:"sub,omitempty"`
}

// jwtHeader is the only header VAPID uses.
const jwtHeader = `{"typ":"JWT","alg":"ES256"}`

// Token signs a VAPID JWT for the push service at aud.
func (k *VAPIDKey) Token(aud, sub string, exp time.Time) (string, error) {
	if aud == "" {
		return "", errors.New("webpush: a token needs the push service's origin")
	}
	claims, err := json.Marshal(Claims{Aud: aud, Exp: exp.Unix(), Sub: sub})
	if err != nil {
		return "", err
	}
	input := Encode([]byte(jwtHeader)) + "." + Encode(claims)
	sum := sha256.Sum256([]byte(input))
	der, err := ecdsa.SignASN1(rand.Reader, k.priv, sum[:])
	if err != nil {
		return "", err
	}
	var sig struct{ R, S *big.Int }
	if _, err := asn1.Unmarshal(der, &sig); err != nil {
		return "", err
	}
	// JWS (RFC 7518 §3.4) writes R and S as two 32-byte big-endian halves.
	rs := make([]byte, 64)
	sig.R.FillBytes(rs[:32])
	sig.S.FillBytes(rs[32:])
	return input + "." + Encode(rs), nil
}

// Authorization is the header value of a push request: "vapid t=<JWT>,
// k=<public key>".
func (k *VAPIDKey) Authorization(aud, sub string, exp time.Time) (string, error) {
	t, err := k.Token(aud, sub, exp)
	if err != nil {
		return "", err
	}
	return "vapid t=" + t + ", k=" + k.PublicKeyString(), nil
}

// Errors from verifying a token.
var (
	ErrAuthorization = errors.New("webpush: not a vapid authorization")
	ErrSignature     = errors.New("webpush: the token's signature doesn't verify")
	ErrClaims        = errors.New("webpush: the token's claims are invalid")
)

// ParseAuthorization splits "vapid t=…, k=…" into the token and the key.
func ParseAuthorization(h string) (token string, key []byte, err error) {
	scheme, rest, ok := strings.Cut(strings.TrimSpace(h), " ")
	if !ok || !strings.EqualFold(scheme, "vapid") {
		return "", nil, ErrAuthorization
	}
	var k string
	for _, part := range strings.Split(rest, ",") {
		name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "t":
			token = strings.TrimSpace(value)
		case "k":
			k = strings.TrimSpace(value)
		}
	}
	if token == "" || k == "" {
		return "", nil, ErrAuthorization
	}
	key, err = Decode(k)
	if err != nil {
		return "", nil, ErrAuthorization
	}
	return token, key, nil
}

// VerifyToken checks an ES256 JWT against an uncompressed P-256 public key
// and returns its claims. It checks the signature only; Verify checks the
// audience and the expiry too.
func VerifyToken(token string, pub []byte) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrAuthorization
	}
	hb, err := Decode(parts[0])
	if err != nil {
		return Claims{}, ErrAuthorization
	}
	var hdr struct {
		Alg string `json:"alg"`
	}
	if json.Unmarshal(hb, &hdr) != nil || hdr.Alg != "ES256" {
		return Claims{}, ErrAuthorization
	}
	rs, err := Decode(parts[2])
	if err != nil || len(rs) != 64 {
		return Claims{}, ErrSignature
	}
	key, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), pub)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %v", ErrSignature, err)
	}
	der, err := asn1.Marshal(struct{ R, S *big.Int }{new(big.Int).SetBytes(rs[:32]), new(big.Int).SetBytes(rs[32:])})
	if err != nil {
		return Claims{}, ErrSignature
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !ecdsa.VerifyASN1(key, sum[:], der) {
		return Claims{}, ErrSignature
	}
	cb, err := Decode(parts[1])
	if err != nil {
		return Claims{}, ErrClaims
	}
	var c Claims
	if err := json.Unmarshal(cb, &c); err != nil {
		return Claims{}, ErrClaims
	}
	return c, nil
}

// Verify checks a request's Authorization header as a push service does:
// the signature, the audience and an expiry in the future and at most 24
// hours away. It returns the claims and the signing key.
func Verify(header, aud string, now time.Time) (Claims, []byte, error) {
	token, key, err := ParseAuthorization(header)
	if err != nil {
		return Claims{}, nil, err
	}
	c, err := VerifyToken(token, key)
	if err != nil {
		return Claims{}, nil, err
	}
	exp := time.Unix(c.Exp, 0)
	switch {
	case c.Aud != aud:
		return c, key, fmt.Errorf("%w: audience %q, expected %q", ErrClaims, c.Aud, aud)
	case !exp.After(now):
		return c, key, fmt.Errorf("%w: expired", ErrClaims)
	case exp.Sub(now) > MaxTokenLifetime:
		return c, key, fmt.Errorf("%w: expires more than 24 hours ahead", ErrClaims)
	}
	return c, key, nil
}
