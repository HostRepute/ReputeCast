// Package partnerproto is the partner-link protocol between two ReputeCast
// instances (grow brief §3.9, docs/grow/partners.md): the messages, how
// they're signed and checked, partner codes, and how the two sides agree a
// signing secret without sending it.
//
// The protocol is versioned: every message carries "protocol": 1 and the
// X-ReputeCast-Partner-Protocol header, and an instance answers
// GET /partner/v1 with the versions it speaks.
package partnerproto

import (
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Version is the protocol version this build speaks.
const Version = 1

// Prefix is where an instance serves the protocol, under its public URL.
const Prefix = "/partner/v1"

// Headers of signed messages.
const (
	HeaderProtocol  = "X-ReputeCast-Partner-Protocol"
	HeaderLink      = "X-ReputeCast-Partner-Link"
	HeaderTimestamp = "X-ReputeCast-Timestamp"
	HeaderSignature = "X-ReputeCast-Signature"
)

// MaxSkew is how far a signed timestamp may be from the receiver's clock.
const MaxSkew = 5 * time.Minute

// PreviousSecretGrace is how long the previous secret keeps working after a
// rotation, on both sides.
const PreviousSecretGrace = 24 * time.Hour

// MaxBody is the largest message either side reads.
const MaxBody = 64 << 10

// --- Partner codes ---------------------------------------------------------

// CodeAlphabet has no look-alikes (no 0, O, 1, I or L).
const CodeAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// CodeTTL is how long a partner code works.
const CodeTTL = 7 * 24 * time.Hour

// NewCode returns a partner code: three groups of four characters,
// "H3RT-6QWN-8ZKB" (about 59 bits).
func NewCode() string {
	n := big.NewInt(int64(len(CodeAlphabet)))
	var b strings.Builder
	for i := range 12 {
		if i > 0 && i%4 == 0 {
			b.WriteByte('-')
		}
		v, err := rand.Int(rand.Reader, n)
		if err != nil {
			panic(err)
		}
		b.WriteByte(CodeAlphabet[v.Int64()])
	}
	return b.String()
}

// NormalizeCode reads a code as pasted ("h3rt 6qwn 8zkb", "H3RT6QWN8ZKB")
// and reports whether it is one.
func NormalizeCode(s string) (string, bool) {
	var raw []byte
	for _, r := range strings.ToUpper(s) {
		switch {
		case r == '-' || r == ' ':
		case r < 128 && strings.IndexByte(CodeAlphabet, byte(r)) >= 0:
			raw = append(raw, byte(r))
		default:
			return "", false
		}
	}
	if len(raw) != 12 {
		return "", false
	}
	return string(raw[0:4]) + "-" + string(raw[4:8]) + "-" + string(raw[8:12]), true
}

// HashCode is what an instance keeps of a code it created.
func HashCode(code string) []byte {
	h := sha256.Sum256([]byte("reputecast-partner-code:" + code))
	return h[:]
}

// CodeURL is the link people send each other: the instance's public URL
// with /p/ and the code.
func CodeURL(base, code string) string {
	return strings.TrimRight(base, "/") + "/p/" + code
}

// Code errors when reading a pasted partner code.
var (
	// ErrNotACode: the text isn't a partner code (a subscribe page, say).
	ErrNotACode = errors.New("partnerproto: not a partner code")
)

// ParseCodeURL splits a pasted partner code into the partner's base URL
// and the code. The base keeps any path before /p/ (an instance served
// under a path).
func ParseCodeURL(raw string) (base, code string, err error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return "", "", ErrNotACode
	}
	path := strings.TrimRight(u.EscapedPath(), "/")
	i := strings.LastIndex(path, "/p/")
	if i < 0 {
		return "", "", ErrNotACode
	}
	code, ok := NormalizeCode(path[i+3:])
	if !ok {
		return "", "", ErrNotACode
	}
	return u.Scheme + "://" + u.Host + path[:i], code, nil
}

// --- Key agreement ----------------------------------------------------------

// KeyPair is one side's X25519 key pair for agreeing a secret.
type KeyPair struct {
	priv *ecdh.PrivateKey
}

// NewKeyPair generates a key pair.
func NewKeyPair() (KeyPair, error) {
	k, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair{priv: k}, nil
}

// Public is the public key as sent in a message (base64url).
func (k KeyPair) Public() string {
	return base64.RawURLEncoding.EncodeToString(k.priv.PublicKey().Bytes())
}

// Secret derives the shared signing secret from the other side's public
// key. salt binds it to what both sides know (the partner code, or the
// rotation's request), info to the two links.
func (k KeyPair) Secret(peer string, salt []byte, recommenderLink, partnerLink string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(peer)
	if err != nil {
		return "", fmt.Errorf("partnerproto: the other side's key isn't readable: %w", err)
	}
	pub, err := ecdh.X25519().NewPublicKey(raw)
	if err != nil {
		return "", fmt.Errorf("partnerproto: the other side's key isn't an X25519 key: %w", err)
	}
	shared, err := k.priv.ECDH(pub)
	if err != nil {
		return "", err
	}
	info := "reputecast-partner-v1|" + recommenderLink + "|" + partnerLink
	key, err := hkdf.Key(sha256.New, shared, salt, info, 32)
	if err != nil {
		return "", err
	}
	return "rcps_" + base64.RawURLEncoding.EncodeToString(key), nil
}

// ConnectSalt binds a connection's secret to its partner code.
func ConnectSalt(code string) []byte {
	h := sha256.Sum256([]byte("connect:" + code))
	return h[:]
}

// RotateSalt binds a rotated secret to the rotation request.
func RotateSalt(rotation string) []byte {
	h := sha256.Sum256([]byte("rotate:" + rotation))
	return h[:]
}

// --- Signatures -------------------------------------------------------------

// canonical is what a signature covers: the timestamp, the method, the
// path and the body, so a signed body can't be replayed at another
// endpoint. Responses sign "RESPONSE <status>" in place of the method and
// the request's path.
func canonical(timestamp int64, method, path string, body []byte) []byte {
	var b []byte
	b = strconv.AppendInt(b, timestamp, 10)
	b = append(b, '\n')
	b = append(b, method...)
	b = append(b, '\n')
	b = append(b, path...)
	b = append(b, '\n')
	return append(b, body...)
}

// Sign is the lowercase hex HMAC-SHA256 of the canonical message.
func Sign(secret string, timestamp int64, method, path string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(canonical(timestamp, method, path, body))
	return hex.EncodeToString(m.Sum(nil))
}

// SignatureHeader is the X-ReputeCast-Signature value: one "v1=" entry per
// live secret, current first.
func SignatureHeader(secrets []string, timestamp int64, method, path string, body []byte) string {
	parts := make([]string, 0, len(secrets))
	for _, s := range secrets {
		if s != "" {
			parts = append(parts, "v1="+Sign(s, timestamp, method, path, body))
		}
	}
	return strings.Join(parts, ", ")
}

// Signature errors.
var (
	ErrNoSignature    = errors.New("partnerproto: the message isn't signed")
	ErrStaleTimestamp = errors.New("partnerproto: the signature's timestamp is more than 5 minutes off")
	ErrBadSignature   = errors.New("partnerproto: the signature doesn't match")
)

// Verify checks a signature header against any of the live secrets.
func Verify(secrets []string, timestamp, header, method, path string, body []byte, now time.Time) error {
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
			if s != "" && hmac.Equal([]byte(Sign(s, ts, method, path, body)), []byte(sig)) {
				return nil
			}
		}
	}
	return ErrBadSignature
}

// ResponseMethod is what a response's signature covers in place of the
// method.
func ResponseMethod(status int) string { return "RESPONSE " + strconv.Itoa(status) }

// --- Messages ---------------------------------------------------------------

// Hello answers GET /partner/v1: the versions an instance speaks.
type Hello struct {
	Protocols []int  `json:"protocols"`
	Instance  string `json:"instance"`
	Software  string `json:"software"`
}

// Profile is a newsletter as partners show it.
type Profile struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	LogoURL     string `json:"logoUrl,omitempty"`
	// List is the list the partner's readers join (the partner side's).
	List string `json:"list,omitempty"`
}

// PreviewRequest asks what a partner code connects to.
type PreviewRequest struct {
	Protocol int    `json:"protocol"`
	Code     string `json:"code"`
}

// PreviewResponse is the partner's profile, before connecting.
type PreviewResponse struct {
	Protocol    int     `json:"protocol"`
	Profile     Profile `json:"profile"`
	DailyLimit  int     `json:"dailyLimit"`
	DoubleOptIn bool    `json:"doubleOptIn"`
	Instance    string  `json:"instance"`
}

// ConnectRequest uses a partner code: the recommending side's link, its
// address, its newsletter and its half of the key agreement.
type ConnectRequest struct {
	Protocol    int     `json:"protocol"`
	Code        string  `json:"code"`
	Link        string  `json:"link"`
	Instance    string  `json:"instance"`
	Recommender Profile `json:"recommender"`
	Key         string  `json:"key"`
}

// ConnectResponse is the partner's link and its half of the key agreement.
type ConnectResponse struct {
	Protocol   int     `json:"protocol"`
	Link       string  `json:"link"`
	Key        string  `json:"key"`
	Profile    Profile `json:"profile"`
	DailyLimit int     `json:"dailyLimit"`
}

// SubscribeRequest asks the partner to run its double opt-in for a reader.
// It carries only the address and the recommending newsletter's name.
type SubscribeRequest struct {
	Protocol int       `json:"protocol"`
	Request  string    `json:"request"`
	Email    string    `json:"email"`
	Source   string    `json:"source"`
	At       time.Time `json:"at"`
}

// Subscribe statuses. The partner answers "accepted" whether or not the
// reader already gets it.
const (
	StatusAccepted     = "accepted"
	StatusLimitReached = "limit_reached"
	StatusPaused       = "paused"
	StatusEnded        = "ended"
	StatusOK           = "ok"
)

// SubscribeResponse is the partner's answer to a subscribe request.
type SubscribeResponse struct {
	Protocol int        `json:"protocol"`
	Status   string     `json:"status"`
	ResetsAt *time.Time `json:"resetsAt,omitempty"`
}

// ConfirmedNotice tells the recommending side a reader confirmed. It names
// the request, never the reader.
type ConfirmedNotice struct {
	Protocol int       `json:"protocol"`
	Request  string    `json:"request"`
	At       time.Time `json:"at"`
}

// States of a link either side can set.
const (
	StateActive = "active"
	StatePaused = "paused"
	StateEnded  = "ended"
)

// StateNotice pauses, resumes or ends a link; the other side follows.
type StateNotice struct {
	Protocol int    `json:"protocol"`
	State    string `json:"state"`
}

// ProfileNotice updates what the recommending side shows: the partner's
// profile and the daily limit it gives.
type ProfileNotice struct {
	Protocol   int     `json:"protocol"`
	Profile    Profile `json:"profile"`
	DailyLimit int     `json:"dailyLimit"`
}

// RotateRequest starts a new secret: a rotation id and a new public key,
// signed with the current secret.
type RotateRequest struct {
	Protocol int    `json:"protocol"`
	Rotation string `json:"rotation"`
	Key      string `json:"key"`
}

// RotateResponse is the other side's new public key.
type RotateResponse struct {
	Protocol int    `json:"protocol"`
	Key      string `json:"key"`
}

// Problem is the body of an error answer.
type Problem struct {
	Protocol int    `json:"protocol"`
	Error    string `json:"error"`
	Detail   string `json:"detail,omitempty"`
	// At is when a code expired or was used.
	At *time.Time `json:"at,omitempty"`
}

// Error codes.
const (
	ErrCodeUnknown        = "code_unknown"
	ErrCodeExpired        = "code_expired"
	ErrCodeUsed           = "code_used"
	ErrCodeRevoked        = "code_revoked"
	ErrSignatureRejected  = "signature_rejected"
	ErrUnknownLink        = "unknown_link"
	ErrBadRequest         = "bad_request"
	ErrUnsupportedVersion = "unsupported_protocol"
	ErrRateLimited        = "rate_limited"
	ErrLinkEnded          = "ended"
)
