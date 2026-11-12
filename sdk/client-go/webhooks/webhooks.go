// Package webhooks verifies the webhooks ReputeCast delivers to your
// endpoints.
//
// Each delivery is a JSON POST signed with the endpoint's secret
// (whsec_…): X-ReputeCast-Signature holds the hex HMAC-SHA256 of
// "<timestamp>.<raw body>" (two comma-separated signatures for 24 hours
// after the secret is rotated) and X-ReputeCast-Timestamp the Unix time in
// seconds.
//
//	body, err := io.ReadAll(r.Body)
//	…
//	if err := webhooks.Verify([]string{secret}, body, r.Header, time.Now()); err != nil {
//		http.Error(w, "invalid signature", http.StatusUnauthorized)
//		return
//	}
package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// The headers of a delivery.
const (
	SignatureHeader = "X-ReputeCast-Signature"
	TimestampHeader = "X-ReputeCast-Timestamp"
)

// Tolerance is how far a delivery's timestamp may be from the receiver's
// clock.
const Tolerance = 5 * time.Minute

// The reasons Verify refuses a delivery; answer it with 400 or 401.
var (
	ErrMissingHeader = errors.New("webhooks: the delivery has no " + SignatureHeader + " or " + TimestampHeader + " header")
	ErrTimestamp     = errors.New("webhooks: the delivery's timestamp is more than 5 minutes from now")
	ErrSignature     = errors.New("webhooks: no signature of the delivery matches the signing secret")
)

// Verify checks that a delivery was signed by ReputeCast with one of
// secrets: body is the raw request body, exactly as received, h its
// headers, now the receiver's clock (time.Now()). Signatures are compared
// in constant time. It returns ErrMissingHeader, ErrTimestamp or
// ErrSignature when the delivery fails.
func Verify(secrets []string, body []byte, h http.Header, now time.Time) error {
	ts, sig := strings.TrimSpace(h.Get(TimestampHeader)), h.Get(SignatureHeader)
	if ts == "" || sig == "" {
		return ErrMissingHeader
	}
	unix, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || unix < 0 || ts[0] == '+' {
		return ErrTimestamp
	}
	if d := now.Sub(time.Unix(unix, 0)); d > Tolerance || d < -Tolerance {
		return ErrTimestamp
	}
	candidates := strings.Split(sig, ",")
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		m := hmac.New(sha256.New, []byte(secret))
		m.Write([]byte(ts))
		m.Write([]byte("."))
		m.Write(body)
		want := []byte(hex.EncodeToString(m.Sum(nil)))
		for _, c := range candidates {
			if hmac.Equal(want, []byte(strings.TrimSpace(c))) {
				return nil
			}
		}
	}
	return ErrSignature
}
