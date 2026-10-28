// Package contracttest is the shared test suite every sending-provider
// adapter must pass (TECH_STACK note 10): registration and capabilities,
// the texts people read about the adapter (in every app language), Send
// (success, permanent rejection, outage, rejected credentials) against a
// fake of the provider's API, and ParseWebhook (signature verification and
// normalization) against recorded payloads.
//
// An adapter's test calls Run with a Suite:
//
//	func TestContract(t *testing.T) {
//		contracttest.Run(t, contracttest.Suite{Adapter: "postmark", API: fakeAPI, New: newForTest, Webhooks: cases})
//	}
//
// The fake API reads Env.Mode ("", ModeOutage, ModeAuth) and the recipient
// address (RejectAddress) to choose its answer, and the suite records every
// request it receives.
package contracttest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

// Modes the fake API simulates.
const (
	ModeOK     = ""
	ModeOutage = "outage" // 5xx or connection errors on every call
	ModeAuth   = "auth"   // credentials rejected on every call
)

// RejectAddress is a recipient the fake API must reject permanently.
const RejectAddress = "rejected@contract.test"

// Env is shared between the suite and the fake API.
type Env struct {
	// URL is the fake API's base URL (empty when the suite has no API).
	URL string
	mu  sync.Mutex
	// Mode selects the failure the fake simulates.
	mode     string
	requests []Recorded
}

// Recorded is one request the fake API received.
type Recorded struct {
	Method string
	Path   string
	Header http.Header
	Body   []byte
}

// Mode returns the simulated failure.
func (e *Env) Mode() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.mode
}

// SetMode sets the simulated failure.
func (e *Env) SetMode(m string) {
	e.mu.Lock()
	e.mode = m
	e.mu.Unlock()
}

// Requests returns the requests received so far.
func (e *Env) Requests() []Recorded {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.requests)
}

func (e *Env) reset() {
	e.mu.Lock()
	e.requests = nil
	e.mode = ModeOK
	e.mu.Unlock()
}

// Webhook is one recorded provider event.
type Webhook struct {
	Name string
	// Request returns a correctly signed request and its body.
	Request func(t *testing.T, env *Env) (*http.Request, []byte)
	// Want are the expected events (compared on type, recipient, provider
	// message id, message id, original type and code, bounce class).
	Want []providers.Event
}

// Suite describes one adapter.
type Suite struct {
	Adapter string
	// API is the fake provider API; nil when the adapter doesn't use HTTP (SMTP).
	API func(env *Env) http.Handler
	// New builds the adapter pointed at env (env.URL is the fake API).
	New func(t *testing.T, env *Env) providers.Provider
	// SendReady reports whether Send can be exercised (for example Mailpit
	// is running); a false result skips the Send tests with the reason.
	SendReady func(t *testing.T) (bool, string)
	// Delivered checks that the messages reached the provider with their
	// headers; it gets the messages sent and the requests recorded.
	Delivered func(t *testing.T, env *Env, msgs []providers.Message, results []providers.SendResult)
	// SkipFailureModes skips the outage/auth send tests (SMTP against Mailpit
	// can't simulate them; its dial errors are tested separately).
	SkipFailureModes bool
	Webhooks         []Webhook
	// Tamper breaks a valid signed request; the default flips a body byte.
	Tamper func(r *http.Request, body []byte) (*http.Request, []byte)
	// IdempotencyKey reads the idempotency key a recorded send request
	// carries ("" for none). An adapter whose provider recognizes a message
	// it already took (providers.IdempotentSender) must say where it is.
	IdempotencyKey func(r Recorded) string
}

// NewEnv starts the fake API (when api is not nil) for adapter-specific
// tests outside Run.
func NewEnv(t *testing.T, api func(env *Env) http.Handler) *Env {
	t.Helper()
	env := &Env{}
	if api != nil {
		h := api(env)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			env.mu.Lock()
			env.requests = append(env.requests, Recorded{Method: r.Method, Path: r.URL.RequestURI(), Header: r.Header.Clone(), Body: body})
			env.mu.Unlock()
			r.Body = io.NopCloser(bytes.NewReader(body))
			h.ServeHTTP(w, r)
		}))
		t.Cleanup(srv.Close)
		env.URL = srv.URL
	}
	return env
}

func Run(t *testing.T, s Suite) {
	t.Helper()
	env := NewEnv(t, s.API)

	t.Run("Registration", func(t *testing.T) {
		d, ok := providers.Lookup(s.Adapter)
		if !ok {
			t.Fatalf("adapter %q is not registered", s.Adapter)
		}
		if d.Name == "" || d.New == nil || d.Kind == "" || d.Maintainer == "" {
			t.Fatalf("incomplete descriptor: %+v", d)
		}
		seen := map[string]bool{}
		for _, f := range d.Fields {
			if f.Key == "" || f.Label == "" || seen[f.Key] {
				t.Fatalf("field %+v: empty or duplicate key/label", f)
			}
			seen[f.Key] = true
			if (strings.Contains(f.Key, "password") || strings.Contains(f.Key, "token") || strings.Contains(f.Key, "secret") || strings.HasSuffix(f.Key, "api_key")) && !f.Secret {
				t.Fatalf("field %q looks secret but is not marked Secret", f.Key)
			}
		}
		checkCapabilities(t, d.Capabilities)
	})

	// What people read about the adapter reads in their language: every
	// message says what its English says, and a built-in adapter leaves no
	// text without one, translated into every app language.
	t.Run("Texts", func(t *testing.T) {
		d, ok := providers.Lookup(s.Adapter)
		if !ok {
			t.Fatalf("adapter %q is not registered", s.Adapter)
		}
		CheckTexts(t, d)
		env.reset()
		p := s.New(t, env)
		CheckTexts(t, providers.Descriptor{Adapter: d.Adapter, Capabilities: p.Capabilities(), Messages: d.Messages})
	})

	t.Run("Capabilities", func(t *testing.T) {
		env.reset()
		p := s.New(t, env)
		if p.Adapter() != s.Adapter {
			t.Fatalf("Adapter() = %q, want %q", p.Adapter(), s.Adapter)
		}
		checkCapabilities(t, p.Capabilities())
		if _, ok := p.(providers.WebhookUnregistrar); ok != p.Capabilities().RemovesWebhook {
			t.Fatalf("RemovesWebhook is %v but the adapter implements WebhookUnregistrar: %v", p.Capabilities().RemovesWebhook, ok)
		}
		if _, ok := p.(providers.WebhookRegistrar); p.Capabilities().RemovesWebhook && !ok {
			t.Fatal("an adapter that removes its webhook must register it too")
		}
	})

	// The webhook ReputeCast registers goes when the provider is removed.
	t.Run("WebhookLifecycle", func(t *testing.T) {
		env.reset()
		p := s.New(t, env)
		wr, okR := p.(providers.WebhookRegistrar)
		wu, okU := p.(providers.WebhookUnregistrar)
		if !okR || !okU {
			t.Skip("the adapter doesn't register and remove webhooks")
		}
		ctx := context.Background()
		hook := "https://reputecast.contract.test/hooks/" + s.Adapter + "/contract-token"
		inspect := func() (providers.WebhookStatus, bool) {
			wi, ok := p.(providers.WebhookInspector)
			if !ok {
				return providers.WebhookStatus{}, false
			}
			st, err := wi.InspectWebhook(ctx, hook)
			if err != nil {
				t.Fatalf("inspect: %v", err)
			}
			return st, true
		}
		if err := wr.RegisterWebhook(ctx, hook); err != nil {
			t.Fatalf("register: %v", err)
		}
		if st, ok := inspect(); ok && !st.Registered {
			t.Fatalf("after RegisterWebhook the provider has no webhook for %s: %+v", hook, st)
		}
		if err := wu.UnregisterWebhook(ctx, hook); err != nil {
			t.Fatalf("unregister: %v", err)
		}
		if st, ok := inspect(); ok && st.Registered {
			t.Fatalf("after UnregisterWebhook the provider still has a webhook for %s: %+v", hook, st)
		}
		if err := wu.UnregisterWebhook(ctx, hook); err != nil {
			t.Fatalf("removing a webhook that isn't there: %v", err)
		}
	})

	t.Run("Send", func(t *testing.T) {
		ready(t, s)
		env.reset()
		p := s.New(t, env)
		n := min(3, max(p.Capabilities().BatchSize, 1))
		msgs := make([]providers.Message, n)
		for i := range msgs {
			msgs[i] = Message(i)
		}
		res, err := p.Send(context.Background(), msgs)
		if err != nil {
			t.Fatalf("Send: %v", err)
		}
		if len(res) != len(msgs) {
			t.Fatalf("Send returned %d results for %d messages", len(res), len(msgs))
		}
		for i, r := range res {
			if r.MessageID != msgs[i].ID {
				t.Fatalf("result %d is for %s, want %s (results must keep the input order)", i, r.MessageID, msgs[i].ID)
			}
			if r.Err != nil {
				t.Fatalf("message %d failed: %v", i, r.Err)
			}
			if r.ProviderMessageID == "" {
				t.Fatalf("message %d has no provider message id", i)
			}
		}
		if s.Delivered != nil {
			s.Delivered(t, env, msgs, res)
		}
	})

	t.Run("SendRejected", func(t *testing.T) {
		ready(t, s)
		if s.API == nil {
			t.Skip("rejections are simulated by the fake API")
		}
		env.reset()
		p := s.New(t, env)
		m := Message(0)
		m.To = mail.Address{Address: RejectAddress}
		res, err := p.Send(context.Background(), []providers.Message{m})
		if err != nil {
			t.Fatalf("a rejected recipient must not fail the whole call: %v", err)
		}
		if len(res) != 1 || res[0].Err == nil {
			t.Fatalf("want a per-message error, got %+v", res)
		}
		if res[0].Err.Temporary {
			t.Fatalf("a permanent rejection was reported as temporary: %+v", res[0].Err)
		}
		if res[0].Err.Code == "" && res[0].Err.Message == "" {
			t.Fatal("the rejection must keep the provider's code or message")
		}
	})

	t.Run("SendOutage", func(t *testing.T) {
		ready(t, s)
		if s.API == nil || s.SkipFailureModes {
			t.Skip("outages are simulated by the fake API")
		}
		env.reset()
		env.SetMode(ModeOutage)
		p := s.New(t, env)
		res, err := p.Send(context.Background(), []providers.Message{Message(0)})
		if err == nil {
			if len(res) == 1 && res[0].Err != nil && res[0].Err.Temporary {
				return
			}
			t.Fatalf("an outage must fail the call or the message temporarily, got %+v", res)
		}
		if !errors.Is(err, providers.ErrUnavailable) {
			t.Fatalf("outage error must wrap providers.ErrUnavailable: %v", err)
		}
	})

	t.Run("SendAuth", func(t *testing.T) {
		ready(t, s)
		if s.API == nil || s.SkipFailureModes {
			t.Skip("credential failures are simulated by the fake API")
		}
		env.reset()
		env.SetMode(ModeAuth)
		p := s.New(t, env)
		_, err := p.Send(context.Background(), []providers.Message{Message(0)})
		if !errors.Is(err, providers.ErrUnauthorized) {
			t.Fatalf("rejected credentials must fail the call with providers.ErrUnauthorized, got %v", err)
		}
	})

	// An adapter that says its provider recognizes a message it already took
	// (providers.IdempotentSender) lets the engine send a message again
	// after a lost answer (review GS-1), so the promise must hold message by
	// message: one message per call, each with a key made from its id alone,
	// the same key every time, and the provider's answer to a repeated
	// message read as sent. A provider's key covers its request: a batch's
	// key would cover the batch, and a message sent again in another batch
	// would be delivered twice.
	t.Run("Idempotency", func(t *testing.T) {
		env.reset()
		p := s.New(t, env)
		if providers.IdempotencyWindow(p) == 0 {
			t.Skip("the provider doesn't recognize a message it already took")
		}
		ready(t, s)
		if s.API == nil {
			t.Skip("keys are checked against the fake API")
		}
		if s.IdempotencyKey == nil {
			t.Fatal("the adapter is a providers.IdempotentSender: set Suite.IdempotencyKey so the suite can read its keys")
		}
		if n := p.Capabilities().BatchSize; n != 1 {
			t.Fatalf("BatchSize %d: an idempotent sender hands each message over in its own call", n)
		}
		a, b := Message(0), Message(1)
		for i, m := range []providers.Message{a, a, b} {
			res, err := p.Send(context.Background(), []providers.Message{m})
			if err != nil || len(res) != 1 || res[0].Err != nil || res[0].MessageID != m.ID {
				t.Fatalf("send %d: %+v, %v (a message sent again must be answered as sent)", i, res, err)
			}
		}
		var keys []string
		for _, r := range env.Requests() {
			if k := s.IdempotencyKey(r); k != "" {
				keys = append(keys, k)
			}
		}
		if len(keys) != 3 {
			t.Fatalf("%d keyed requests for 3 sends: %q", len(keys), keys)
		}
		if keys[0] != keys[1] || keys[0] == keys[2] || !strings.Contains(keys[0], a.ID.String()) || !strings.Contains(keys[2], b.ID.String()) {
			t.Fatalf("keys %q: want each message's own id, the same when it is sent again", keys)
		}
	})

	for _, w := range s.Webhooks {
		t.Run("Webhook/"+w.Name, func(t *testing.T) {
			env.reset()
			p := s.New(t, env)
			req, body := w.Request(t, env)
			if v, ok := p.(providers.WebhookVerifier); ok {
				if err := v.VerifyWebhook(context.Background(), req, body); err != nil {
					t.Fatalf("VerifyWebhook rejected a valid request: %v", err)
				}
			}
			got, err := p.ParseWebhook(context.Background(), req, body)
			if err != nil {
				t.Fatalf("ParseWebhook: %v", err)
			}
			compare(t, got, w.Want)

			// The normalization job parses stored payloads again without the
			// original transport: a verified context must be enough.
			req2, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://ingest.test/", bytes.NewReader(body))
			req2.Header = req.Header.Clone()
			again, err := p.ParseWebhook(providers.WithSignatureVerified(context.Background()), req2, body)
			if err != nil {
				t.Fatalf("ParseWebhook with a verified context: %v", err)
			}
			compare(t, again, w.Want)

			tamper := s.Tamper
			if tamper == nil {
				tamper = flipByte
			}
			bad, badBody := tamper(req.Clone(context.Background()), slices.Clone(body))
			if _, err := p.ParseWebhook(context.Background(), bad, badBody); !errors.Is(err, providers.ErrBadSignature) {
				t.Fatalf("a tampered request must fail with providers.ErrBadSignature, got %v", err)
			}
			if v, ok := p.(providers.WebhookVerifier); ok {
				if err := v.VerifyWebhook(context.Background(), bad, badBody); !errors.Is(err, providers.ErrBadSignature) {
					t.Fatalf("VerifyWebhook accepted a tampered request: %v", err)
				}
			}
		})
	}
}

// CheckTexts checks the texts of a descriptor (docs/adapters.md, "Text for
// people"): each message must say what the English it was kept with says,
// and a built-in adapter of a kind the catalog shows (api, smtp,
// self_hosted, http) names its catalog, gives every text a message from it
// and has the catalog translated into every app language. A community
// adapter's texts are English; the messages ReputeCast adds to them are
// checked the same way.
func CheckTexts(t *testing.T, d providers.Descriptor) {
	t.Helper()
	switch d.Kind {
	case "api", "smtp", "self_hosted", "http":
		if d.Messages == nil {
			t.Fatalf("%s keeps no catalog: a built-in adapter declares its texts in messages/<locale>.json and calls WithMessages in its Descriptor", d.Adapter)
		}
	}
	for _, x := range d.Texts() {
		if x.Text.IsZero() {
			if d.Messages != nil {
				t.Errorf("%s %q has no message in %s's catalog, so it reads in English in every language", x.What, x.English, d.Messages.Namespace)
			}
			continue
		}
		if got := x.Text.In(i18n.Default); got != x.English {
			t.Errorf("%s: the message %s says %q, but the English beside it is %q", x.What, x.Text.ID, got, x.English)
		}
	}
	if d.Messages == nil {
		return
	}
	for _, l := range i18n.Locales() {
		if l == i18n.English || l == i18n.Pseudo {
			continue
		}
		if missing := d.Messages.Missing(l); len(missing) > 0 {
			t.Errorf("%s's catalog isn't translated into %s: %s", d.Messages.Namespace, l.Name(), strings.Join(missing, ", "))
		}
	}
}

// CheckText checks one text an adapter returned with its message (an
// account standing, a tracking detail, a provisioning row): the message
// must exist and say what the English says.
func CheckText(t *testing.T, what, english string, text i18n.Text) {
	t.Helper()
	switch {
	case english == "":
		return
	case text.IsZero():
		t.Errorf("%s %q has no message, so it reads in English in every language", what, english)
	case text.In(i18n.Default) != english:
		t.Errorf("%s: the message %s says %q, but the English beside it is %q", what, text.ID, text.In(i18n.Default), english)
	}
}

func ready(t *testing.T, s Suite) {
	t.Helper()
	if s.SendReady != nil {
		if ok, why := s.SendReady(t); !ok {
			t.Skip(why)
		}
	}
}

func checkCapabilities(t *testing.T, c providers.Capabilities) {
	t.Helper()
	switch c.Bulk {
	case providers.BulkAllowed, providers.BulkBroadcastStreams, providers.BulkForbidden, providers.BulkYourPolicy:
	default:
		t.Fatalf("Capabilities.Bulk %q is not a known policy", c.Bulk)
	}
	if c.BatchSize < 1 {
		t.Fatalf("Capabilities.BatchSize must be at least 1, got %d", c.BatchSize)
	}
	if c.Pricing == "" {
		t.Fatal("Capabilities.Pricing must describe the pricing model")
	}
	if c.RateDiscovery && !c.QuotaDiscovery {
		t.Fatal("rate discovery without quota discovery")
	}
}

func flipByte(r *http.Request, body []byte) (*http.Request, []byte) {
	if len(body) > 0 {
		i := len(body) / 2
		body[i] ^= 0x01
		if body[i] == '"' || body[i] == '\\' {
			body[i] ^= 0x03
		}
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return r, body
}

func compare(t *testing.T, got, want []providers.Event) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Type != w.Type || g.Recipient != w.Recipient || g.ProviderMessageID != w.ProviderMessageID ||
			g.OriginalType != w.OriginalType || g.OriginalCode != w.OriginalCode || g.BounceClass != w.BounceClass {
			t.Fatalf("event %d:\n got  %+v\n want %+v", i, summary(g), summary(w))
		}
		if (w.MessageID == nil) != (g.MessageID == nil) || w.MessageID != nil && *w.MessageID != *g.MessageID {
			t.Fatalf("event %d message id: got %v, want %v", i, g.MessageID, w.MessageID)
		}
		if g.OccurredAt.IsZero() {
			t.Fatalf("event %d has no time", i)
		}
		if len(g.Raw) == 0 {
			t.Fatalf("event %d does not keep the raw payload", i)
		}
	}
}

func summary(e providers.Event) map[string]any {
	return map[string]any{"type": e.Type, "recipient": e.Recipient, "providerMessageId": e.ProviderMessageID, "messageId": e.MessageID,
		"originalType": e.OriginalType, "originalCode": e.OriginalCode, "diagnostic": e.Diagnostic, "bounceClass": e.BounceClass}
}

// Message returns a test message with the headers the sending pipeline sets.
func Message(i int) providers.Message {
	id := uuid.Must(uuid.NewV7())
	return providers.Message{
		ID:      id,
		Stream:  providers.StreamBroadcast,
		From:    mail.Address{Name: "Acme Newsletter", Address: "news@news.acme.io"},
		To:      mail.Address{Name: "Contract Test", Address: "reader" + string(rune('a'+i)) + "@contract.test"},
		Subject: "Contract test " + id.String(),
		HTML:    "<p>Hello from the contract suite.</p>",
		Text:    "Hello from the contract suite.",
		Headers: map[string]string{
			"List-Unsubscribe":      "<https://news.acme.io/p/acme/unsubscribe/abc>, <mailto:unsubscribe@news.acme.io>",
			"List-Unsubscribe-Post": "List-Unsubscribe=One-Click",
			"Feedback-ID":           "contract:acme:reputecast",
		},
		Tags: map[string]string{"campaign": "contract"},
	}
}

// UUID parses s or fails the test.
func UUID(t *testing.T, s string) *uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return &id
}

// Time is a fixed time for fixtures.
var Time = time.Date(2026, 10, 12, 13, 41, 2, 0, time.UTC)
