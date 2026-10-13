// Package secrets encrypts provider credentials, OIDC client secrets, TOTP
// seeds and other write-only values at rest with AES-256-GCM.
//
// Ciphertext layout: version (1 byte) | key id (4 bytes) | nonce (12) | sealed.
// The key id is the first 4 bytes of SHA-256(key), so data sealed with a
// previous key still opens after rotation. Callers pass associated data that
// names the row and field ("provider:<id>:credentials"), which stops a
// ciphertext from being copied onto another row.
package secrets

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const formatV1 = 1

// ErrDecrypt is returned when a value can't be opened with any known key.
var ErrDecrypt = errors.New("secrets: cannot decrypt value")

// Box seals and opens values. The zero value is not usable; use New.
type Box struct {
	current key
	keys    []key
}

type key struct {
	id   [4]byte
	aead cipher.AEAD
}

func newKey(raw []byte) (key, error) {
	if len(raw) != 32 {
		return key{}, fmt.Errorf("secrets: key must be 32 bytes, got %d", len(raw))
	}
	block, err := aes.NewCipher(raw)
	if err != nil {
		return key{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return key{}, err
	}
	sum := sha256.Sum256(raw)
	var k key
	copy(k.id[:], sum[:4])
	k.aead = aead
	return k, nil
}

// New returns a Box that seals with current and opens with current or any previous key.
func New(current []byte, previous ...[]byte) (*Box, error) {
	ck, err := newKey(current)
	if err != nil {
		return nil, err
	}
	b := &Box{current: ck, keys: []key{ck}}
	for _, p := range previous {
		pk, err := newKey(p)
		if err != nil {
			return nil, err
		}
		b.keys = append(b.keys, pk)
	}
	return b, nil
}

// Seal encrypts plaintext bound to associated data ad.
func (b *Box) Seal(plaintext []byte, ad string) ([]byte, error) {
	nonce := make([]byte, b.current.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	out := make([]byte, 0, 1+4+len(nonce)+len(plaintext)+b.current.aead.Overhead())
	out = append(out, formatV1)
	out = append(out, b.current.id[:]...)
	out = append(out, nonce...)
	return b.current.aead.Seal(out, nonce, plaintext, []byte(ad)), nil
}

// SealString is Seal for strings.
func (b *Box) SealString(s, ad string) ([]byte, error) { return b.Seal([]byte(s), ad) }

// Open decrypts a value sealed with Seal and the same associated data.
func (b *Box) Open(ciphertext []byte, ad string) ([]byte, error) {
	if len(ciphertext) < 1+4+12 || ciphertext[0] != formatV1 {
		return nil, ErrDecrypt
	}
	id := ciphertext[1:5]
	for _, k := range b.keys {
		if !bytes.Equal(k.id[:], id) {
			continue
		}
		ns := k.aead.NonceSize()
		nonce := ciphertext[5 : 5+ns]
		pt, err := k.aead.Open(nil, nonce, ciphertext[5+ns:], []byte(ad))
		if err != nil {
			return nil, ErrDecrypt
		}
		return pt, nil
	}
	return nil, ErrDecrypt
}

// OpenString is Open for strings.
func (b *Box) OpenString(ciphertext []byte, ad string) (string, error) {
	pt, err := b.Open(ciphertext, ad)
	return string(pt), err
}

// NeedsRotation reports whether ciphertext was sealed with a previous key.
func (b *Box) NeedsRotation(ciphertext []byte) bool {
	return len(ciphertext) >= 5 && !bytes.Equal(ciphertext[1:5], b.current.id[:])
}

// GenerateKey returns a new random 32-byte key, base64-encoded.
func GenerateKey() string {
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(k)
}

// LoadOrCreateKeyFile reads a base64 key from path, creating the file with a
// fresh key (mode 0600) when it doesn't exist yet.
func LoadOrCreateKeyFile(path string) (string, bool, error) {
	b, err := os.ReadFile(path) //nolint:gosec // the path is the operator's secrets.key_file setting
	if err == nil {
		return strings.TrimSpace(string(b)), false, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return "", false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", false, err
	}
	k := GenerateKey()
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // the operator's secrets.key_file setting
	if err != nil {
		return "", false, err
	}
	if _, err := f.WriteString(k + "\n"); err != nil {
		_ = f.Close()
		return "", false, err
	}
	if err := f.Close(); err != nil {
		return "", false, err
	}
	return k, true, nil
}
