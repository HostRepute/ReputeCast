// Package webpush speaks the Web Push protocol with the standard library
// only (design expansion phase 8, X22):
//
//   - RFC 8030, the generic event delivery: a POST to the subscription's
//     push resource with TTL, Urgency and Topic, answered 201 Created;
//   - RFC 8292, VAPID: an ES256 JWT that identifies the workspace to the
//     push services, carried in "Authorization: vapid t=…, k=…";
//   - RFC 8291 with RFC 8188's aes128gcm content coding: the payload is
//     encrypted for the browser with a fresh ECDH key and a random salt.
//
// It knows nothing about workspaces, sites or campaigns (internal/push does
// that). The encryption is checked against RFC 8291's Appendix A and RFC
// 8188's example; webpushtest is an in-process push service that verifies
// the JWT and decrypts what it receives, for tests and the demo.
package webpush

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// b64 is base64url without padding, the encoding of every key and token
// in Web Push.
var b64 = base64.RawURLEncoding

// Encode writes bytes as unpadded base64url.
func Encode(b []byte) string { return b64.EncodeToString(b) }

// Decode reads base64url with or without padding (browsers send both).
func Decode(s string) ([]byte, error) {
	s = strings.TrimRight(strings.TrimSpace(s), "=")
	return b64.DecodeString(s)
}

// VAPIDKey is the key pair that identifies a workspace to the browsers'
// push services (RFC 8292): an ECDSA P-256 key. The public key is the
// applicationServerKey the browser subscribes with; the private key never
// leaves the server (internal/push seals it at rest).
type VAPIDKey struct {
	priv *ecdsa.PrivateKey
}

// GenerateVAPIDKey makes a new key pair.
func GenerateVAPIDKey() (*VAPIDKey, error) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &VAPIDKey{priv: k}, nil
}

// ParseVAPIDKey reads a private key stored as its 32-byte scalar.
func ParseVAPIDKey(raw []byte) (*VAPIDKey, error) {
	k, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), raw)
	if err != nil {
		return nil, fmt.Errorf("webpush: vapid private key: %w", err)
	}
	return &VAPIDKey{priv: k}, nil
}

// PrivateBytes is the private scalar, 32 bytes, for sealing at rest.
func (k *VAPIDKey) PrivateBytes() []byte {
	b, err := k.priv.Bytes()
	if err != nil {
		// A key made by GenerateKey or ParseRawPrivateKey on P-256 always
		// serializes.
		panic(err)
	}
	return b
}

// PublicKey is the uncompressed P-256 point, 65 bytes (0x04 || X || Y).
func (k *VAPIDKey) PublicKey() []byte {
	b, err := k.priv.PublicKey.Bytes()
	if err != nil {
		panic(err)
	}
	return b
}

// PublicKeyString is the public key in base64url, as browsers take it for
// applicationServerKey and as the "k" parameter carries it.
func (k *VAPIDKey) PublicKeyString() string { return Encode(k.PublicKey()) }

// Subscription is what a browser's PushManager.subscribe() returns: the
// push resource and the keys the payload is encrypted for.
type Subscription struct {
	// Endpoint is the push resource URL (https://fcm.googleapis.com/…).
	Endpoint string
	// P256dh is the browser's ECDH public key, 65 bytes uncompressed.
	P256dh []byte
	// Auth is the 16-byte authentication secret.
	Auth []byte
}

// Errors from ParseSubscription.
var (
	ErrEndpoint = errors.New("webpush: the endpoint isn't an https URL")
	ErrKeys     = errors.New("webpush: the subscription's keys are malformed")
)

// ParseSubscription checks a subscription as a browser sent it: an https
// endpoint, a P-256 public key and a 16-byte secret.
func ParseSubscription(endpoint, p256dh, auth string) (Subscription, error) {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || u.Scheme != "https" || u.Host == "" || len(endpoint) > 1024 {
		return Subscription{}, ErrEndpoint
	}
	pk, err := Decode(p256dh)
	if err != nil {
		return Subscription{}, ErrKeys
	}
	if _, err := ecdh.P256().NewPublicKey(pk); err != nil {
		return Subscription{}, ErrKeys
	}
	sec, err := Decode(auth)
	if err != nil || len(sec) != 16 {
		return Subscription{}, ErrKeys
	}
	return Subscription{Endpoint: u.String(), P256dh: pk, Auth: sec}, nil
}

// Origin is the push service's origin, the JWT's audience: the scheme and
// host of the endpoint ("https://fcm.googleapis.com").
func Origin(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", ErrEndpoint
	}
	return u.Scheme + "://" + u.Host, nil
}
