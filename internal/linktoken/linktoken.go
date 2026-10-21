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
	"fmt"
	"io"
	"time"

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
	// Polls and surveys (design expansion, X11: 110–119, numbered after
	// the workstream so parallel kinds never collide). Link is the poll's
	// or the survey's number. A poll link without a message and a contact
	// is the web archive's: its answers count apart, as non-contacts'. A
	// test link's MessageID names the test address (poll_test_recipients).
	Poll       Kind = 110
	Survey     Kind = 111
	PollTest   Kind = 112
	SurveyTest Kind = 113
	// Grow (design expansion, X15: 150–159): a referrer's hub, the
	// recommendations page and a reward's download. Their Link holds the
	// day they were issued (IssueDay), so they expire.
	ReferralHub     Kind = 150
	Recommendations Kind = 151
	RewardDownload  Kind = 152
)

const version = 1

// ErrInvalid is returned for malformed or forged tokens.
var ErrInvalid = errors.New("linktoken: invalid token")

// ErrSignature is returned for a well-formed token of the wanted kind whose
// signature doesn't match: signed with an earlier instance key, or forged.
// It is also ErrInvalid.
var ErrSignature = fmt.Errorf("%w: the signature doesn't match", ErrInvalid)

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
	// The shape first: only a token that could have been signed can have a
	// signature that no longer matches.
	link, n := binary.Uvarint(body[50:])
	if n <= 0 || 50+n != len(body) || link > 1<<31 {
		return Claims{}, ErrInvalid
	}
	if !hmac.Equal(sig, s.mac(body)) {
		return Claims{}, ErrSignature
	}
	var c Claims
	c.Kind = Kind(body[1])
	copy(c.WorkspaceID[:], body[2:18])
	copy(c.MessageID[:], body[18:34])
	copy(c.ContactID[:], body[34:50])
	c.Link = uint32(link)
	return c, nil
}

// Digest is a keyed hash of parts that must not be reversible, such as the
// daily key that counts a referral link's visits once per network and
// browser without cookies. It is stable for the instance.
func (s *Signer) Digest(parts ...[]byte) []byte {
	m := hmac.New(sha256.New, append([]byte("digest:"), s.key...))
	for _, p := range parts {
		m.Write(binary.AppendUvarint(nil, uint64(len(p))))
		m.Write(p)
	}
	return m.Sum(nil)
}

// issueEpoch is day 0 of IssueDay.
var issueEpoch = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

// IssueDay is the day a link was issued, for tokens that expire (their
// Link field).
func IssueDay(t time.Time) uint32 {
	d := t.UTC().Sub(issueEpoch) / (24 * time.Hour)
	if d < 0 {
		return 0
	}
	return uint32(d)
}

// IssuedOn is the day an expiring token was issued.
func IssuedOn(day uint32) time.Time { return issueEpoch.Add(time.Duration(day) * 24 * time.Hour) }

func (s *Signer) mac(b []byte) []byte {
	m := hmac.New(sha256.New, s.key)
	m.Write(b)
	return m.Sum(nil)[:16]
}
