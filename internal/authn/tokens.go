package authn

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// NewToken returns a random URL-safe token (32 bytes of entropy) and its
// SHA-256 hash. Only the hash is stored; the token goes to the client once.
func NewToken() (token string, hash []byte) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token)
}

// HashToken returns the SHA-256 of a token. Tokens carry 256 bits of
// entropy, so a fast hash is enough (no salt or KDF needed).
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// Prefixes of API keys and MCP agent tokens. Test-mode transactional keys
// start with rc_test_ and SMTP credentials with rc_smtp_, so a key says
// what it is wherever it is pasted.
const (
	APIKeyPrefix = "rc_live_"
	AgentPrefix  = "rc_agent_"
	TestPrefix   = "rc_test_"
	SMTPPrefix   = "rc_smtp_"
)

var keyPrefixes = []string{APIKeyPrefix, AgentPrefix, TestPrefix, SMTPPrefix}

// NewAPIKey returns a new key such as rc_live_8f2c…a91d (64 hex chars after
// the prefix), its hash, and the masked form shown in lists.
func NewAPIKey(prefix string) (key string, hash []byte, masked string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	h := hex.EncodeToString(b)
	key = prefix + h
	return key, HashToken(key), MaskAPIKey(key)
}

// MaskAPIKey shows the prefix, the first 4 and the last 4 characters.
func MaskAPIKey(key string) string {
	for _, p := range keyPrefixes {
		if strings.HasPrefix(key, p) {
			body := key[len(p):]
			if len(body) <= 8 {
				return key
			}
			return p + body[:4] + "…" + body[len(body)-4:]
		}
	}
	return "…"
}

// LooksLikeAPIKey reports whether a bearer token has an API key or agent prefix.
func LooksLikeAPIKey(s string) bool {
	for _, p := range keyPrefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// EqualHash compares two hashes in constant time.
func EqualHash(a, b []byte) bool { return subtle.ConstantTimeCompare(a, b) == 1 }
