// Package webhookauth holds the webhook authentication schemes several
// sending providers share, so each adapter verifies them the same way:
// Standard Webhooks (the Svix scheme Resend uses), HMAC signatures over the
// body, and shared secrets sent as basic auth or in a header. Every check
// is constant-time and fails with providers.ErrBadSignature.
package webhookauth

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // some providers still sign with HMAC-SHA1
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
)

// Tolerance is how far a signed timestamp may be from now before the
// request is treated as a replay.
const Tolerance = 5 * time.Minute

// Standard verifies a Standard Webhooks request (standardwebhooks.com, the
// scheme Svix signs with): the headers webhook-id, webhook-timestamp and
// webhook-signature (or their svix- equivalents) and an HMAC-SHA256 over
// "id.timestamp.body" with the base64 secret that follows "whsec_". The
// signature header lists one or more "v1,<base64>" entries; any may match,
// so secrets can be rotated. A zero now skips the timestamp check.
func Standard(secret string, h http.Header, body []byte, now time.Time) error {
	id, ts, sigs := h.Get("webhook-id"), h.Get("webhook-timestamp"), h.Get("webhook-signature")
	if id == "" && ts == "" && sigs == "" {
		id, ts, sigs = h.Get("svix-id"), h.Get("svix-timestamp"), h.Get("svix-signature")
	}
	if id == "" || ts == "" || sigs == "" {
		return providers.ErrBadSignature
	}
	key, err := StandardKey(secret)
	if err != nil {
		return err
	}
	if !now.IsZero() {
		if err := Fresh(ts, now); err != nil {
			return err
		}
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + ts + "."))
	mac.Write(body)
	want := mac.Sum(nil)
	for _, entry := range strings.Fields(sigs) {
		version, sig, ok := strings.Cut(entry, ",")
		if !ok || version != "v1" {
			continue
		}
		got, err := base64.StdEncoding.DecodeString(sig)
		if err == nil && hmac.Equal(got, want) {
			return nil
		}
	}
	return providers.ErrBadSignature
}

// StandardKey decodes a Standard Webhooks secret ("whsec_" + base64).
func StandardKey(secret string) ([]byte, error) {
	s := strings.TrimSpace(secret)
	if s == "" {
		return nil, fmt.Errorf("%w: no signing secret is configured", providers.ErrBadSignature)
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(s, "whsec_"))
	if err != nil || len(key) == 0 {
		return nil, fmt.Errorf("%w: the signing secret is not a whsec_ secret", providers.ErrBadSignature)
	}
	return key, nil
}

// SignStandard returns the webhook-signature value for a Standard Webhooks
// payload, for tests and for webhook test events.
func SignStandard(secret, id, ts string, body []byte) (string, error) {
	key, err := StandardKey(secret)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + ts + "."))
	mac.Write(body)
	return "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

// Fresh checks a Unix timestamp in seconds (or milliseconds) is within
// Tolerance of now.
func Fresh(ts string, now time.Time) error {
	f, err := strconv.ParseFloat(strings.TrimSpace(ts), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return fmt.Errorf("%w: bad timestamp", providers.ErrBadSignature)
	}
	if f > 1e12 {
		f /= 1000
	}
	t := time.Unix(int64(f), 0)
	if d := now.Sub(t); d > Tolerance || d < -Tolerance {
		return fmt.Errorf("%w: timestamp outside the %s window", providers.ErrBadSignature, Tolerance)
	}
	return nil
}

// Hash returns the hash for an HMAC algorithm name: sha256 (also the
// default for ""), sha1 or sha512.
func Hash(alg string) (func() hash.Hash, error) {
	switch strings.ToLower(strings.TrimPrefix(strings.ToLower(alg), "hmac-")) {
	case "", "sha256":
		return sha256.New, nil
	case "sha1":
		return sha1.New, nil
	case "sha512":
		return sha512.New, nil
	}
	return nil, fmt.Errorf("unknown HMAC algorithm %q (use sha256, sha1 or sha512)", alg)
}

// HMAC computes the HMAC of the parts with key.
func HMAC(h func() hash.Hash, key []byte, parts ...[]byte) []byte {
	mac := hmac.New(h, key)
	for _, p := range parts {
		mac.Write(p)
	}
	return mac.Sum(nil)
}

// Decode decodes a signature in hex (default) or base64 (standard or URL
// alphabet, padded or not).
func Decode(sig, encoding string) ([]byte, error) {
	sig = strings.TrimSpace(sig)
	switch strings.ToLower(encoding) {
	case "", "hex":
		return hex.DecodeString(sig)
	case "base64":
		for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
			if b, err := enc.DecodeString(sig); err == nil {
				return b, nil
			}
		}
		return nil, fmt.Errorf("signature is not base64")
	}
	return nil, fmt.Errorf("unknown signature encoding %q (use hex or base64)", encoding)
}

// Equal compares two secrets in constant time. An empty expected secret
// never matches, so a provider without a configured secret rejects
// everything instead of accepting everything.
func Equal(got, want string) bool {
	if want == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

// Basic checks HTTP basic auth against user and password.
func Basic(r *http.Request, user, password string) error {
	u, p, ok := r.BasicAuth()
	if !ok || !Equal(u, user) || !Equal(p, password) {
		return providers.ErrBadSignature
	}
	return nil
}
