// Package remotetest is a fake community adapter process for tests: a small
// reference implementation of the adapter side of the protocol
// (docs/adapters.md) in Go. It fronts a fake provider ("Fake ESP") whose
// behaviour follows the contract suite: contracttest.RejectAddress is
// refused permanently, Mode "outage" and "auth" fail every provider call.
//
// Fake ESP webhooks are JSON events (one object or an array) signed with
// X-Fake-Signature: hex HMAC-SHA256 of the body keyed with the connection's
// webhook_signing_key credential. A body {"type":"handshake","challenge":c}
// is a subscription handshake answered with {"challenge":c}.
package remotetest

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/remote"
)

// ID is the fake adapter's id; its registry key is "community:fake-esp".
const ID = "fake-esp"

// SignatureHeader carries Fake ESP's webhook signature.
const SignatureHeader = "X-Fake-Signature"

// Manifest is the fake adapter's manifest with every capability declared.
func Manifest() remote.Manifest {
	return remote.Manifest{
		Protocol: remote.ProtocolVersion, Kind: remote.Kind, ID: ID, Name: "Fake ESP", Version: "1.2.3", Maintainer: "reputecast-tests",
		Capabilities: remote.Capabilities{Batch: 50, Webhooks: true, RegisterWebhooks: true, DomainSetup: "full", SuppressionSync: "two-way", QuotaDiscovery: "rate"},
		Terms:        remote.Terms{BulkMarketing: "allowed", Note: "Fake ESP allows newsletters."},
		Pricing:      "$0.50 per 1,000",
		Config: []remote.Field{
			{Key: "api_key", Label: "API key", Secret: true, Required: true},
			{Key: "webhook_signing_key", Label: "Webhook signing key", Secret: true},
			{Key: "region", Label: "Region", Options: []string{"eu", "us"}},
		},
		Permissions: remote.Permissions{Network: []string{"api.fake-esp.test"}},
	}
}

// ManifestJSON is Manifest as manifest.json.
func ManifestJSON(m remote.Manifest) []byte {
	b, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return b
}

// Call is one request the fake adapter accepted.
type Call struct {
	Operation      string
	IdempotencyKey string
	Connection     *remote.Connection
	Input          json.RawMessage
}

// Adapter is the fake adapter process (an http.Handler).
type Adapter struct {
	Manifest remote.Manifest
	// Secrets verify ReputeCast's requests; the first signs the answers.
	Secrets []string
	// Mode returns the simulated provider failure ("", "outage", "auth").
	Mode func() string
	Now  func() time.Time

	// Misbehaviour for tests.
	Delay    time.Duration
	Unsigned bool
	// AnswerSecret signs answers with another secret.
	AnswerSecret string
	// Rewrite changes an answer's body before it is signed.
	Rewrite func(op string, body []byte) []byte

	mu       sync.Mutex
	calls    []Call
	answered map[string][]byte
	sent     map[string]int
	pushed   []string
	webhook  string
}

// New returns a fake adapter for secret with the full manifest.
func New(secret string) *Adapter {
	return &Adapter{Manifest: Manifest(), Secrets: []string{secret}}
}

func (a *Adapter) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

func (a *Adapter) mode() string {
	if a.Mode != nil {
		return a.Mode()
	}
	return ""
}

// Calls returns the accepted requests.
func (a *Adapter) Calls() []Call {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]Call(nil), a.calls...)
}

// Sent returns how often each message id was handed to the provider.
func (a *Adapter) Sent() map[string]int {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := map[string]int{}
	for k, v := range a.sent {
		out[k] = v
	}
	return out
}

// Pushed returns the suppressions pushed to the provider.
func (a *Adapter) Pushed() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.pushed...)
}

// Webhook returns the webhook URL registered at the provider.
func (a *Adapter) Webhook() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.webhook
}

type protoErr struct {
	status int
	e      remote.WireError
}

func fail(status int, code, msg string) *protoErr {
	return &protoErr{status, remote.WireError{Code: code, Message: msg}}
}

// ServeHTTP implements the protocol.
func (a *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Path == "/healthz" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
		return
	}
	if a.Delay > 0 {
		select {
		case <-time.After(a.Delay):
		case <-r.Context().Done():
			return
		}
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 32<<20))
	if err != nil {
		return
	}
	op := strings.TrimPrefix(r.URL.Path, "/v1/")
	if r.Method != http.MethodPost || op == r.URL.Path {
		a.answer(w, "", "", nil, fail(http.StatusNotFound, remote.CodeNotSupported, "unknown path"))
		return
	}
	if err := remote.Verify(a.Secrets, r.Header.Get(remote.HeaderTimestamp), r.Header.Get(remote.HeaderSignature), body, a.now()); err != nil {
		a.answer(w, op, "", nil, fail(http.StatusUnauthorized, remote.CodeUnauthenticated, err.Error()))
		return
	}
	var env struct {
		Protocol       int                `json:"protocol"`
		Operation      string             `json:"operation"`
		Adapter        string             `json:"adapter"`
		IdempotencyKey string             `json:"idempotencyKey"`
		Connection     *remote.Connection `json:"connection"`
		Input          json.RawMessage    `json:"input"`
	}
	if err := json.Unmarshal(body, &env); err != nil || env.Protocol != remote.ProtocolVersion || env.Operation != op || env.Adapter != a.Manifest.ID || env.IdempotencyKey == "" {
		a.answer(w, op, env.IdempotencyKey, nil, fail(http.StatusBadRequest, remote.CodeInvalidRequest, "the envelope doesn't match the request"))
		return
	}
	if op != remote.OpHealth && env.Connection == nil {
		a.answer(w, op, env.IdempotencyKey, nil, fail(http.StatusBadRequest, remote.CodeInvalidRequest, "connection missing"))
		return
	}
	a.mu.Lock()
	if a.answered == nil {
		a.answered, a.sent = map[string][]byte{}, map[string]int{}
	}
	if prev, ok := a.answered[env.IdempotencyKey]; ok {
		a.mu.Unlock()
		a.write(w, op, http.StatusOK, prev)
		return
	}
	a.calls = append(a.calls, Call{Operation: op, IdempotencyKey: env.IdempotencyKey, Connection: env.Connection, Input: env.Input})
	a.mu.Unlock()
	result, perr := a.dispatch(op, env.Connection, env.Input)
	a.answer(w, op, env.IdempotencyKey, result, perr)
}

func (a *Adapter) answer(w http.ResponseWriter, op, key string, result any, perr *protoErr) {
	resp := map[string]any{"protocol": remote.ProtocolVersion, "idempotencyKey": key}
	status := http.StatusOK
	if perr != nil {
		resp["error"], status = perr.e, perr.status
	} else {
		resp["result"] = result
	}
	body, _ := json.Marshal(resp)
	if perr == nil && key != "" {
		a.mu.Lock()
		a.answered[key] = body
		a.mu.Unlock()
	}
	a.write(w, op, status, body)
}

func (a *Adapter) write(w http.ResponseWriter, op string, status int, body []byte) {
	if a.Rewrite != nil {
		body = a.Rewrite(op, body)
	}
	w.Header().Set("Content-Type", "application/json")
	if !a.Unsigned {
		secret := a.AnswerSecret
		if secret == "" && len(a.Secrets) > 0 {
			secret = a.Secrets[0]
		}
		ts := a.now().Unix()
		w.Header().Set(remote.HeaderTimestamp, strconv.FormatInt(ts, 10))
		w.Header().Set(remote.HeaderSignature, "v1="+remote.Sign(secret, ts, body))
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// providerFailure is what Fake ESP answers in a failure mode.
func (a *Adapter) providerFailure() *protoErr {
	switch a.mode() {
	case contracttest.ModeAuth:
		return &protoErr{http.StatusBadGateway, remote.WireError{Code: remote.CodeProviderAuth, Message: "Fake ESP rejected the API key.", Status: 401, ProviderCode: "invalid_api_key"}}
	case contracttest.ModeOutage:
		return &protoErr{http.StatusServiceUnavailable, remote.WireError{Code: remote.CodeProviderUnavailable, Message: "Fake ESP answered 503.", Status: 503, ProviderCode: "service_unavailable", RetryAfter: 30}}
	}
	return nil
}

func (a *Adapter) dispatch(op string, conn *remote.Connection, input json.RawMessage) (any, *protoErr) {
	caps := a.Manifest.Capabilities
	allowed := map[string]bool{}
	for _, o := range remote.Operations(caps) {
		allowed[o] = true
	}
	if !allowed[op] {
		return nil, fail(http.StatusNotImplemented, remote.CodeNotSupported, op+" is not implemented")
	}
	if op == remote.OpHealth {
		return remote.HealthOutput{Adapter: a.Manifest.ID, Version: a.Manifest.Version, Protocol: remote.ProtocolVersion, SDK: "remotetest"}, nil
	}
	if op != remote.OpWebhookVerify && op != remote.OpWebhookParse {
		if conn.Credentials["api_key"] == nil {
			return nil, &protoErr{http.StatusBadGateway, remote.WireError{Code: remote.CodeProviderAuth, Message: "No API key.", Status: 401}}
		}
		if perr := a.providerFailure(); perr != nil {
			return nil, perr
		}
	}
	switch op {
	case remote.OpVerify, remote.OpQuota:
		return remote.WireQuota{Kind: "monthly_plan", Max: 50000, Used: 1234, MaxRate: 14, Account: "Fake plan · 50,000 per month"}, nil
	case remote.OpSend:
		var in remote.SendInput
		if err := json.Unmarshal(input, &in); err != nil || len(in.Messages) == 0 || len(in.Messages) > caps.Batch {
			return nil, fail(http.StatusBadRequest, remote.CodeInvalidRequest, "bad send input")
		}
		out := remote.SendOutput{Results: []remote.WireSendResult{}}
		a.mu.Lock()
		for _, m := range in.Messages {
			if m.To.Address == contracttest.RejectAddress {
				out.Results = append(out.Results, remote.WireSendResult{MessageID: m.ID, Error: &remote.WireSendError{Code: "recipient_suppressed", Message: "The recipient is on Fake ESP's suppression list."}})
				continue
			}
			a.sent[m.ID]++
			out.Results = append(out.Results, remote.WireSendResult{MessageID: m.ID, ProviderMessageID: "fake-" + m.ID[len(m.ID)-12:]})
		}
		a.mu.Unlock()
		return out, nil
	case remote.OpWebhookVerify:
		var in remote.WebhookVerifyInput
		if err := json.Unmarshal(input, &in); err != nil {
			return nil, fail(http.StatusBadRequest, remote.CodeInvalidRequest, "bad webhook input")
		}
		if !a.signed(conn, in.Request) {
			return nil, fail(http.StatusUnprocessableEntity, remote.CodeBadSignature, "X-Fake-Signature doesn't match")
		}
		var hs struct{ Type, Challenge string }
		if json.Unmarshal(in.Request.Body, &hs) == nil && hs.Type == "handshake" {
			b, _ := json.Marshal(map[string]string{"challenge": hs.Challenge})
			return map[string]any{"handshake": map[string]string{"body": string(b)}}, nil
		}
		return map[string]any{}, nil
	case remote.OpWebhookParse:
		var in remote.WebhookParseInput
		if err := json.Unmarshal(input, &in); err != nil {
			return nil, fail(http.StatusBadRequest, remote.CodeInvalidRequest, "bad webhook input")
		}
		if !in.Verified && !a.signed(conn, in.Request) {
			return nil, fail(http.StatusUnprocessableEntity, remote.CodeBadSignature, "X-Fake-Signature doesn't match")
		}
		evs, err := Normalize(in.Request.Body)
		if err != nil {
			return nil, fail(http.StatusBadRequest, remote.CodeInvalidRequest, err.Error())
		}
		return remote.WebhookParseOutput{Events: evs}, nil
	case remote.OpWebhookRegister:
		var in remote.RegisterInput
		_ = json.Unmarshal(input, &in)
		a.mu.Lock()
		a.webhook = in.URL
		a.mu.Unlock()
		return map[string]any{}, nil
	case remote.OpDomainAdd, remote.OpDomainStatus:
		var in remote.DomainInput
		_ = json.Unmarshal(input, &in)
		status := "pending"
		if op == remote.OpDomainStatus {
			status = "verified"
		}
		return map[string]any{"records": []map[string]string{
			{"purpose": "dkim", "type": "CNAME", "name": "fk1._domainkey." + in.Domain, "value": "fk1.dkim.fake-esp.test", "status": status},
			{"purpose": "return_path", "type": "CNAME", "name": "bounces." + in.Domain, "value": "return.fake-esp.test", "status": status},
		}}, nil
	case remote.OpSuppressionsList:
		var in remote.SuppressionsListInput
		_ = json.Unmarshal(input, &in)
		if in.Cursor == nil {
			next := "page-2"
			return remote.SuppressionsListOutput{NextCursor: &next, Suppressions: []remote.WireSuppression{
				{Email: "Bounced@Example.com", Reason: "hard_bounce", Code: "bounce", CreatedAt: "2026-10-11T08:00:00Z"}}}, nil
		}
		return remote.SuppressionsListOutput{Suppressions: []remote.WireSuppression{{Email: "complainer@example.com", Reason: "complaint", Code: "spam_report"}}}, nil
	case remote.OpSuppressionsAdd:
		var in remote.SuppressionsAddInput
		_ = json.Unmarshal(input, &in)
		a.mu.Lock()
		a.pushed = append(a.pushed, in.Emails...)
		a.mu.Unlock()
		return map[string]any{}, nil
	}
	return nil, fail(http.StatusNotImplemented, remote.CodeNotSupported, op+" is not implemented")
}

// signed checks a Fake ESP webhook signature.
func (a *Adapter) signed(conn *remote.Connection, r remote.WebhookRequest) bool {
	key, _ := conn.Credentials["webhook_signing_key"].(string)
	got := ""
	for k, v := range r.Headers {
		if strings.EqualFold(k, SignatureHeader) && len(v) > 0 {
			got = v[0]
		}
	}
	return key != "" && hmac.Equal([]byte(got), []byte(SignWebhook(key, r.Body)))
}

// SignWebhook is Fake ESP's webhook signature.
func SignWebhook(key string, body []byte) string {
	m := hmac.New(sha256.New, []byte(key))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

// Event is a Fake ESP webhook event.
type Event struct {
	Event       string `json:"event"`
	MessageID   string `json:"message_id"`
	Recipient   string `json:"recipient"`
	Timestamp   string `json:"timestamp"`
	SMTPCode    string `json:"smtp_code,omitempty"`
	Description string `json:"description,omitempty"`
	BounceType  string `json:"bounce_type,omitempty"`
	RCMessage   string `json:"rc_message,omitempty"`
}

// Normalize maps a Fake ESP webhook body to protocol events.
func Normalize(body []byte) ([]remote.WireEvent, error) {
	var raws []json.RawMessage
	if err := json.Unmarshal(body, &raws); err != nil {
		raws = []json.RawMessage{body}
	}
	out := []remote.WireEvent{}
	for _, raw := range raws {
		var e Event
		if err := json.Unmarshal(raw, &e); err != nil {
			return nil, err
		}
		ev := remote.WireEvent{Recipient: e.Recipient, ProviderMessageID: e.MessageID, MessageID: e.RCMessage, OccurredAt: e.Timestamp,
			OriginalType: e.Event, OriginalCode: e.SMTPCode, Diagnostic: e.Description, Raw: raw}
		switch e.Event {
		case "delivered":
			ev.Type = "delivered"
		case "bounced":
			ev.Type, ev.OriginalType = "soft_bounce", "bounced · "+e.BounceType
			if e.BounceType == "hard" {
				ev.Type = "hard_bounce"
			}
		case "complained":
			ev.Type = "complaint"
		default:
			continue
		}
		out = append(out, ev)
	}
	return out, nil
}

// WebhookBody builds a signed Fake ESP webhook request body and headers.
func WebhookBody(key string, events ...Event) ([]byte, http.Header) {
	var body []byte
	if len(events) == 1 {
		body, _ = json.Marshal(events[0])
	} else {
		body, _ = json.Marshal(events)
	}
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set(SignatureHeader, SignWebhook(key, body))
	return body, h
}

// RandomKey returns a random-looking test key (stable per name).
func RandomKey(name string) string {
	sum := sha256.Sum256([]byte("remotetest:" + name))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
