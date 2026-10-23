// Package authn implements sign-in primitives: argon2id password hashing,
// opaque session and API tokens (only SHA-256 hashes are stored), TOTP,
// recovery codes, the optional breached-password check, passkeys and OIDC.
package authn

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Argon2id parameters (OWASP: m=46 MiB, t=1, p=1). Stored with each hash, so
// they can be raised later; NeedsRehash tells when a hash uses old ones.
const (
	argonMemory  = 46 * 1024
	argonTime    = 1
	argonThreads = 1
	argonKeyLen  = 32
	argonSaltLen = 16
)

// ErrInvalidHash is returned for a malformed stored hash.
var ErrInvalidHash = errors.New("authn: invalid password hash")

// HashPassword returns a PHC-formatted argon2id hash.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

type argonParams struct {
	memory  uint32
	time    uint32
	threads uint8
	salt    []byte
	key     []byte
}

func parseHash(encoded string) (argonParams, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return argonParams{}, ErrInvalidHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return argonParams{}, ErrInvalidHash
	}
	var p argonParams
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return argonParams{}, ErrInvalidHash
	}
	var err error
	if p.salt, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil {
		return argonParams{}, ErrInvalidHash
	}
	if p.key, err = base64.RawStdEncoding.DecodeString(parts[5]); err != nil {
		return argonParams{}, ErrInvalidHash
	}
	return p, nil
}

// VerifyPassword reports whether password matches the stored hash, in constant time.
func VerifyPassword(password, encoded string) (bool, error) {
	p, err := parseHash(encoded)
	if err != nil {
		return false, err
	}
	key := argon2.IDKey([]byte(password), p.salt, p.time, p.memory, p.threads, uint32(len(p.key)))
	return subtle.ConstantTimeCompare(key, p.key) == 1, nil
}

// dummyHash is verified against when an account doesn't exist, so the
// response time doesn't reveal whether the email is registered.
var dummyHash, _ = HashPassword("reputecast-dummy-password")

// VerifyDummy burns the same time as a real verification.
func VerifyDummy(password string) { _, _ = VerifyPassword(password, dummyHash) }

// NeedsRehash reports whether a hash was made with weaker parameters.
func NeedsRehash(encoded string) bool {
	p, err := parseHash(encoded)
	if err != nil {
		return true
	}
	return p.memory < argonMemory || p.time < argonTime || len(p.key) < argonKeyLen
}

// PasswordProblem explains why a password is rejected, or "" when it's
// fine, in English (the CLI's).
func PasswordProblem(password string, minLength int) string {
	return PasswordProblemIn(i18n.Default, password, minLength)
}

// PasswordProblemIn explains why a password is rejected in a language, or
// "" when it's fine.
func PasswordProblemIn(l i18n.Lang, password string, minLength int) string {
	n := utf8.RuneCountInString(password)
	switch {
	case n < minLength:
		return msgs.M("password.short").Text(l, i18n.Args{"min": minLength})
	case n > 256:
		return msgs.M("password.long").Text(l, nil)
	case strings.TrimSpace(password) == "":
		return msgs.M("password.spaces").Text(l, nil)
	}
	return ""
}
