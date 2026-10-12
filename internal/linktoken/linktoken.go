// Package linktoken signs the tokens embedded in emails: open pixel, click
// redirect, one-click unsubscribe (RFC 8058), preference center, view online
// and double opt-in confirmation links. Tokens are compact, unforgeable
// (HMAC-SHA256 truncated to 128 bits) and carry no personal data in clear.
package linktoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"

	"github.com/google/uuid"
	"golang.org/x/crypto/hkdf"
)

// Kind is what a token is for; a token for one purpose never works for another.
type Kind byte

// Token kinds.
const (
	Open        Kind = 1
	Click       Kind = 2
	Unsubscribe Kind = 3
	Preferences Kind = 4
	ViewOnline  Kind = 5
	Confirm     Kind = 6
)

const version = 1

// ErrInvalid is returned for malformed or forged tokens.
var ErrInvalid = errors.New("linktoken: invalid token")

// Claims identify the message (or contact, for links outside a message).
type Claims struct {
	Kind        Kind
	WorkspaceID uuid.UUID
	MessageID   uuid.UUID // uuid.Nil when not tied to a message
	ContactID   uuid.UUID
	// Link is the index of the link in campaign_links for Click tokens.
	Link uint32
}

// Signer signs and verifies tokens with a key derived from the instance key.
type Signer struct{ key []byte }

// NewSigner derives the signing key from the instance master key with HKDF.
func NewSigner(master []byte) (*Signer, error) {
	if len(master) < 32 {
		return nil, errors.New("linktoken: master key too short")
	}
	key := make([]byte, 32)
	if _, err := io.ReadFull(hkdf.New(sha256.New, master, nil, []byte("reputecast/linktoken/v1")), key); err != nil {
		return nil, err
	}
	return &Signer{key: key}, nil
}

// Sign returns a URL-safe token.
func (s *Signer) Sign(c Claims) string {
	buf := make([]byte, 0, 2+48+binary.MaxVarintLen32+16)
	buf = append(buf, version, byte(c.Kind))
	buf = append(buf, c.WorkspaceID[:]...)
	buf = append(buf, c.MessageID[:]...)
	buf = append(buf, c.ContactID[:]...)
	buf = binary.AppendUvarint(buf, uint64(c.Link))
	buf = append(buf, s.mac(buf)...)
	return base64.RawURLEncoding.EncodeToString(buf)
}

// Verify checks a token and returns its claims; want limits the accepted kind.
func (s *Signer) Verify(token string, want Kind) (Claims, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) < 2+48+1+16 || raw[0] != version || Kind(raw[1]) != want {
		return Claims{}, ErrInvalid
	}
	body, sig := raw[:len(raw)-16], raw[len(raw)-16:]
	if !hmac.Equal(sig, s.mac(body)) {
		return Claims{}, ErrInvalid
	}
	var c Claims
	c.Kind = Kind(body[1])
	copy(c.WorkspaceID[:], body[2:18])
	copy(c.MessageID[:], body[18:34])
	copy(c.ContactID[:], body[34:50])
	link, n := binary.Uvarint(body[50:])
	if n <= 0 || 50+n != len(body) || link > 1<<31 {
		return Claims{}, ErrInvalid
	}
	c.Link = uint32(link)
	return c, nil
}

func (s *Signer) mac(b []byte) []byte {
	m := hmac.New(sha256.New, s.key)
	m.Write(b)
	return m.Sum(nil)[:16]
}
