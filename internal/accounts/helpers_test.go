package accounts_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/fxamacker/cbor/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"

	"github.com/hostrepute/reputecast/internal/accounts"
	"github.com/hostrepute/reputecast/internal/app"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/server"
)

// fakeMail records system mail; available=false behaves like an instance
// without a System-stream provider.
type fakeMail struct {
	mu        sync.Mutex
	available bool
	sent      []mailer.Message
}

func (f *fakeMail) SendSystem(_ context.Context, m mailer.Message) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.available {
		return "", mailer.ErrNoSystemProvider
	}
	f.sent = append(f.sent, m)
	return "Postmark", nil
}

func (f *fakeMail) SystemAvailable(context.Context, *uuid.UUID) bool { return f.available }

func (f *fakeMail) last(t *testing.T) mailer.Message {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.sent) == 0 {
		t.Fatal("no email was sent")
	}
	return f.sent[len(f.sent)-1]
}

// withMail serves the accounts operations with a fake mailer, on the same
// database and authenticator as env (the app's own server keeps the
// "no System provider" mailer).
func withMail(t *testing.T, env *apptest.Env, m mailer.Sender) *apptest.Env {
	t.Helper()
	r := chi.NewRouter()
	r.Use(server.AuditContext)
	h := humachi.New(r, app.NewHumaConfig())
	h.UseMiddleware(env.App.Auth.Middleware(h))
	accounts.Register(h, &accounts.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Cfg: env.App.Cfg,
		Mail: m, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Breach: authn.NewBreachChecker()})
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return &apptest.Env{T: t, App: env.App, Server: srv}
}

// totpCode returns the current code for a secret shown as "JBSW Y3DP …".
func totpCode(t *testing.T, secret string) string {
	t.Helper()
	code, err := totp.GenerateCode(strings.ReplaceAll(secret, " ", ""), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return code
}

// linkToken extracts the token after prefix ("/reset-password/") in a text.
func linkToken(t *testing.T, text, prefix string) string {
	t.Helper()
	i := strings.Index(text, prefix)
	if i < 0 {
		t.Fatalf("no %q link in %q", prefix, text)
	}
	tok := text[i+len(prefix):]
	if j := strings.IndexAny(tok, " \n\"<&"); j >= 0 {
		tok = tok[:j]
	}
	return tok
}

// softAuthenticator is a minimal WebAuthn authenticator for tests: one
// ES256 key, "none" attestation, user verification always performed.
type softAuthenticator struct {
	priv       *ecdsa.PrivateKey
	credID     []byte
	userHandle []byte
	rpID       string
	origin     string
	counter    uint32
}

func newSoftAuthenticator(t *testing.T, rpID, origin string) *softAuthenticator {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	return &softAuthenticator{priv: priv, credID: id, rpID: rpID, origin: origin}
}

var b64 = base64.RawURLEncoding

func (a *softAuthenticator) clientData(t *testing.T, typ string, options map[string]any) []byte {
	t.Helper()
	b, _ := json.Marshal(map[string]any{"type": typ, "challenge": options["challenge"], "origin": a.origin, "crossOrigin": false})
	return b
}

func (a *softAuthenticator) authData(flags byte, attested []byte) []byte {
	h := sha256.Sum256([]byte(a.rpID))
	out := append([]byte{}, h[:]...)
	out = append(out, flags)
	cnt := make([]byte, 4)
	binary.BigEndian.PutUint32(cnt, a.counter)
	out = append(out, cnt...)
	return append(out, attested...)
}

// register answers navigator.credentials.create() options.
func (a *softAuthenticator) register(t *testing.T, options map[string]any) map[string]any {
	t.Helper()
	user := options["user"].(map[string]any)
	handle, err := b64.DecodeString(user["id"].(string))
	if err != nil {
		t.Fatalf("user handle: %v", err)
	}
	a.userHandle = handle
	pub := a.priv.PublicKey
	x, y := make([]byte, 32), make([]byte, 32)
	pub.X.FillBytes(x)
	pub.Y.FillBytes(y)
	cose, err := cbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: x, -3: y})
	if err != nil {
		t.Fatal(err)
	}
	attested := make([]byte, 16) // zero AAGUID
	l := make([]byte, 2)
	binary.BigEndian.PutUint16(l, uint16(len(a.credID)))
	attested = append(attested, l...)
	attested = append(attested, a.credID...)
	attested = append(attested, cose...)
	att, err := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": a.authData(0x45, attested)})
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{
		"id": b64.EncodeToString(a.credID), "rawId": b64.EncodeToString(a.credID), "type": "public-key",
		"response": map[string]any{
			"clientDataJSON":    b64.EncodeToString(a.clientData(t, "webauthn.create", options)),
			"attestationObject": b64.EncodeToString(att),
			"transports":        []string{"internal"},
		},
		"clientExtensionResults":  map[string]any{},
		"authenticatorAttachment": "platform",
	}
}

// assert answers navigator.credentials.get() options.
func (a *softAuthenticator) assert(t *testing.T, options map[string]any) map[string]any {
	t.Helper()
	a.counter++
	auth := a.authData(0x05, nil)
	cd := a.clientData(t, "webauthn.get", options)
	sum := sha256.Sum256(cd)
	digest := sha256.Sum256(append(append([]byte{}, auth...), sum[:]...))
	sig, err := ecdsa.SignASN1(rand.Reader, a.priv, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{
		"id": b64.EncodeToString(a.credID), "rawId": b64.EncodeToString(a.credID), "type": "public-key",
		"response": map[string]any{
			"clientDataJSON":    b64.EncodeToString(cd),
			"authenticatorData": b64.EncodeToString(auth),
			"signature":         b64.EncodeToString(sig),
			"userHandle":        b64.EncodeToString(a.userHandle),
		},
		"clientExtensionResults":  map[string]any{},
		"authenticatorAttachment": "platform",
	}
}
