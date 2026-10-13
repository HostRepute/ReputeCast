package remote

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

// Headers of the protocol.
const (
	HeaderProtocol    = "X-ReputeCast-Protocol"
	HeaderTimestamp   = "X-ReputeCast-Timestamp"
	HeaderSignature   = "X-ReputeCast-Signature"
	HeaderOperation   = "X-ReputeCast-Operation"
	HeaderIdempotency = "Idempotency-Key"
)

// MaxSkew is how far a signed timestamp may be from the receiver's clock.
const MaxSkew = 5 * time.Minute

// SecretPrefix starts every adapter signing secret.
const SecretPrefix = "rcas_"

// NewSecret returns a fresh signing secret: "rcas_" and 32 random bytes.
func NewSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return SecretPrefix + base64.RawURLEncoding.EncodeToString(b)
}

// Sign returns the lowercase hex HMAC-SHA256 of "<timestamp>.<body>".
func Sign(secret string, timestamp int64, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(strconv.FormatInt(timestamp, 10)))
	m.Write([]byte("."))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

// SignatureHeader is the X-ReputeCast-Signature value: one "v1=" entry per
// secret, current first.
func SignatureHeader(secrets []string, timestamp int64, body []byte) string {
	parts := make([]string, 0, len(secrets))
	for _, s := range secrets {
		if s != "" {
			parts = append(parts, "v1="+Sign(s, timestamp, body))
		}
	}
	return strings.Join(parts, ", ")
}

// Signature errors.
var (
	ErrNoSignature    = errors.New("the request or response is not signed")
	ErrStaleTimestamp = errors.New("the signature's timestamp is more than 5 minutes off")
	ErrBadSignature   = errors.New("the signature doesn't match")
)

// Verify checks a timestamp and signature header against any of the secrets.
func Verify(secrets []string, timestamp, header string, body []byte, now time.Time) error {
	if timestamp == "" || header == "" {
		return ErrNoSignature
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return ErrNoSignature
	}
	if d := now.Sub(time.Unix(ts, 0)); d > MaxSkew || d < -MaxSkew {
		return ErrStaleTimestamp
	}
	for _, part := range strings.Split(header, ",") {
		sig, ok := strings.CutPrefix(strings.TrimSpace(part), "v1=")
		if !ok {
			continue
		}
		for _, s := range secrets {
			if s != "" && hmac.Equal([]byte(Sign(s, ts, body)), []byte(sig)) {
				return nil
			}
		}
	}
	return ErrBadSignature
}
