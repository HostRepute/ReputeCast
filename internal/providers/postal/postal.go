// Package postal is the adapter for a self-hosted Postal server
// (postalserver.io). Postal's HTTP API sends raw messages and reports message
// deliveries; webhooks are configured in Postal's web interface and signed
// with the server's RSA key (X-Postal-Signature-256, RS256 over the body),
// whose public half Postal publishes at /.well-known/jwks.json.
//
// Postal has no API for domains, suppression lists or webhook management, so
// those capabilities are off: ReputeCast enforces the shared suppression list
// before handing mail to Postal, and DNS records are set up by hand.
package postal

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // Postal's legacy X-Postal-Signature header is RSA-SHA1
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/providers/bounce"
	"github.com/hostrepute/reputecast/internal/providers/mimemsg"
	"github.com/hostrepute/reputecast/internal/version"
)

// Adapter is the registry key.
const Adapter = "postal"

// TagPrefix marks the X-Postal-Tag that carries the ReputeCast message id.
const TagPrefix = "rc:"

// Settings are the non-secret connection settings.
type Settings struct {
	ServerURL string `json:"server_url"`
	// SigningJWKS is Postal's webhook signing key set, fetched from
	// /.well-known/jwks.json when the provider is connected.
	SigningJWKS json.RawMessage `json:"signing_jwks,omitempty"`
	// SigningKey is a PEM or base64 DER public key for Postal versions that
	// show the key instead of publishing a JWKS.
	SigningKey string `json:"signing_key,omitempty"`
	Version    string `json:"version,omitempty"`
}

// Credentials are write-only.
type Credentials struct {
	APIKey string `json:"api_key"`
}

func init() { providers.Register(Descriptor()) }

// Descriptor describes the adapter for the catalog.
func Descriptor() providers.Descriptor {
	return providers.Descriptor{
		Adapter: Adapter,
		Name:    "Postal (self-hosted)",
		Kind:    "self_hosted",
		Capabilities: providers.Capabilities{
			Bulk:      providers.BulkYourPolicy,
			Webhooks:  "Registered",
			BatchSize: 1,
			Pricing:   "self-hosted",
		},
		Version:    version.Version,
		Maintainer: "HostRepute",
		Terms: []providers.Terms{{Variant: "self-hosted", Bulk: providers.BulkYourPolicy, Note: "You run the server, so your hosting provider's acceptable-use policy applies instead.",
			Streams: []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem}}},
		Events:          "Webhooks",
		DomainSetup:     "Manual DNS",
		SuppressionMode: "Local only",
		Fields: []providers.Field{
			{Key: "server_url", Label: "Server URL", Required: true, Help: "The address of your Postal web interface, e.g. https://mta.acme.io."},
			{Key: "api_key", Label: "API key", Secret: true, Required: true, Help: "Postal › your mail server › Credentials › API."},
		},
		New: New,
	}
}

// Provider talks to one Postal mail server.
type Provider struct {
	cfg  providers.Config
	s    Settings
	c    Credentials
	api  *apiclient.Client
	keys *keyCache
}

var _ providers.Provider = (*Provider)(nil)

// New builds the adapter.
func New(cfg providers.Config) (providers.Provider, error) {
	p := &Provider{cfg: cfg}
	if len(cfg.Settings) > 0 {
		if err := json.Unmarshal(cfg.Settings, &p.s); err != nil {
			return nil, fmt.Errorf("postal settings: %w", err)
		}
	}
	if len(cfg.Credentials) > 0 {
		if err := json.Unmarshal(cfg.Credentials, &p.c); err != nil {
			return nil, fmt.Errorf("postal credentials: %w", err)
		}
	}
	p.s.ServerURL = strings.TrimRight(strings.TrimSpace(p.s.ServerURL), "/")
	p.api = apiclient.New(p.s.ServerURL, cfg.HTTPClient)
	p.api.Header.Set("X-Server-API-Key", p.c.APIKey)
	p.keys = sharedKeys(p.s.ServerURL)
	if err := p.keys.seed(p.s.SigningJWKS, p.s.SigningKey); err != nil {
		return nil, fmt.Errorf("postal signing key: %w", err)
	}
	return p, nil
}

// Adapter implements providers.Provider.
func (p *Provider) Adapter() string { return Adapter }

// Capabilities implements providers.Provider.
func (p *Provider) Capabilities() providers.Capabilities { return Descriptor().Capabilities }

type apiResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// call posts to the legacy API, which always answers 200 with a status field.
func (p *Provider) call(ctx context.Context, op, path string, in, out any) error {
	var res apiResponse
	if err := p.api.Do(ctx, op, http.MethodPost, path, in, &res); err != nil {
		return err
	}
	if res.Status == "success" {
		if out != nil {
			return json.Unmarshal(res.Data, out)
		}
		return nil
	}
	var e apiError
	_ = json.Unmarshal(res.Data, &e)
	ce := &providers.CallError{Op: op, Status: http.StatusOK, Code: e.Code, Message: e.Message}
	if res.Status == "parameter-error" {
		ce.Code = "ParameterError"
	}
	switch e.Code {
	case "AccessDenied", "InvalidServerAPIKey":
		ce.Auth = true
		if ce.Message == "" {
			ce.Message = "The API key was not accepted."
		}
	case "ServerSuspended":
		ce.Temporary = true
		if ce.Message == "" {
			ce.Message = "The Postal mail server is suspended."
		}
	}
	return ce
}

// Send implements providers.Provider through POST /api/v1/send/raw, one
// message per call (Postal has no batch endpoint).
func (p *Provider) Send(ctx context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	results := make([]providers.SendResult, 0, len(msgs))
	for _, m := range msgs {
		raw, _, err := mimemsg.Build(m, time.Now(), map[string]string{"X-Postal-Tag": TagPrefix + m.ID.String()})
		if err != nil {
			results = append(results, providers.SendResult{MessageID: m.ID, Err: &providers.SendError{Code: "Render", Message: err.Error()}})
			continue
		}
		body := map[string]any{"mail_from": m.From.Address, "rcpt_to": []string{m.To.Address}, "data": base64.StdEncoding.EncodeToString(raw)}
		var data struct {
			MessageID string `json:"message_id"`
			Messages  map[string]struct {
				ID    json.Number `json:"id"`
				Token string      `json:"token"`
			} `json:"messages"`
		}
		err = p.call(ctx, "POST /api/v1/send/raw", "/api/v1/send/raw", body, &data)
		if err != nil {
			var ce *providers.CallError
			if !errors.As(err, &ce) || ce.Auth || ce.Status != http.StatusOK || ce.Code == "ServerSuspended" {
				// Credentials or the server itself failed. Nothing of this
				// call went out: fail it, or the rest of it when earlier
				// messages were accepted.
				if len(results) == 0 {
					return nil, err
				}
				for _, rest := range msgs[len(results):] {
					results = append(results, providers.SendResult{MessageID: rest.ID, Err: &providers.SendError{Temporary: true, Code: "Unavailable", Message: err.Error()}})
				}
				return results, nil
			}
			results = append(results, providers.SendResult{MessageID: m.ID, Err: &providers.SendError{Code: ce.Code, Message: ce.Message}})
			continue
		}
		res := providers.SendResult{MessageID: m.ID}
		for rcpt, info := range data.Messages {
			if strings.EqualFold(rcpt, m.To.Address) || len(data.Messages) == 1 {
				res.ProviderMessageID = info.ID.String()
			}
		}
		results = append(results, res)
	}
	return results, nil
}

// Verify implements providers.Verifier: looking up message 0 answers
// MessageNotFound with a valid key and InvalidServerAPIKey otherwise, without
// side effects.
func (p *Provider) Verify(ctx context.Context) (providers.Quota, error) {
	err := p.call(ctx, "POST /api/v1/messages/message", "/api/v1/messages/message", map[string]any{"id": 0}, nil)
	var ce *providers.CallError
	if err != nil && (!errors.As(err, &ce) || ce.Code != "MessageNotFound") {
		return providers.Quota{}, err
	}
	acct := "Postal"
	if p.s.Version != "" {
		acct += " " + p.s.Version
	}
	return providers.Quota{Account: acct + " · reachable"}, nil
}

// VerifyWebhook checks X-Postal-Signature-256 (or the legacy SHA-1
// signature) against the server's signing keys.
func (p *Provider) VerifyWebhook(ctx context.Context, r *http.Request, body []byte) error {
	sig256 := r.Header.Get("X-Postal-Signature-256")
	sig1 := r.Header.Get("X-Postal-Signature")
	if sig256 == "" && sig1 == "" {
		return providers.ErrBadSignature
	}
	keys := p.keys.lookup(ctx, p.api.HTTP, p.s.ServerURL, r.Header.Get("X-Postal-Signature-KID"))
	if len(keys) == 0 {
		return providers.ErrBadSignature
	}
	for _, k := range keys {
		if sig256 != "" {
			if sig, err := base64.StdEncoding.DecodeString(sig256); err == nil {
				sum := sha256.Sum256(body)
				if rsa.VerifyPKCS1v15(k, crypto.SHA256, sum[:], sig) == nil {
					return nil
				}
			}
			continue
		}
		if sig, err := base64.StdEncoding.DecodeString(sig1); err == nil {
			sum := sha1.Sum(body) //nolint:gosec // legacy signature format
			if rsa.VerifyPKCS1v15(k, crypto.SHA1, sum[:], sig) == nil {
				return nil
			}
		}
	}
	return providers.ErrBadSignature
}

type envelope struct {
	Event     string          `json:"event"`
	Timestamp float64         `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
	UUID      string          `json:"uuid"`
}

type messageInfo struct {
	ID        json.Number `json:"id"`
	Token     string      `json:"token"`
	Direction string      `json:"direction"`
	MessageID string      `json:"message_id"`
	To        string      `json:"to"`
	From      string      `json:"from"`
	Subject   string      `json:"subject"`
	Timestamp float64     `json:"timestamp"`
	Tag       string      `json:"tag"`
}

type statusPayload struct {
	Message     messageInfo `json:"message"`
	Status      string      `json:"status"`
	Details     string      `json:"details"`
	Output      string      `json:"output"`
	SentWithSSL bool        `json:"sent_with_ssl"`
	Timestamp   float64     `json:"timestamp"`
}

// ParseWebhook implements providers.Provider.
func (p *Provider) ParseWebhook(ctx context.Context, r *http.Request, body []byte) ([]providers.Event, error) {
	if !providers.SignatureVerified(ctx) {
		if err := p.VerifyWebhook(ctx, r, body); err != nil {
			return nil, err
		}
	}
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("postal webhook: %w", err)
	}
	raw := append(json.RawMessage(nil), body...)
	switch env.Event {
	case "MessageSent", "MessageDelayed", "MessageDeliveryFailed", "MessageHeld":
		var sp statusPayload
		if err := json.Unmarshal(env.Payload, &sp); err != nil {
			return nil, fmt.Errorf("postal %s: %w", env.Event, err)
		}
		if sp.Message.Direction == "incoming" {
			return []providers.Event{}, nil
		}
		ev := statusEvent(env.Event, sp)
		ev.Raw = raw
		return []providers.Event{ev}, nil
	case "MessageBounced":
		var bp struct {
			Original messageInfo `json:"original_message"`
			Bounce   messageInfo `json:"bounce"`
		}
		if err := json.Unmarshal(env.Payload, &bp); err != nil {
			return nil, fmt.Errorf("postal MessageBounced: %w", err)
		}
		// Postal received a bounce message for mail it had accepted. It
		// doesn't classify the bounce, so it counts as soft: nothing is
		// suppressed on a guess.
		ev := baseEvent(bp.Original)
		ev.Type = providers.EventSoftBounce
		ev.OriginalType = "MessageBounced"
		ev.Diagnostic = "Bounce message received by Postal (message " + bp.Bounce.ID.String() + ")"
		ev.BounceClass = bounce.Other
		ev.OccurredAt = floatTime(env.Timestamp)
		ev.Raw = raw
		return []providers.Event{ev}, nil
	case "MessageLinkClicked", "MessageLoaded":
		var lp struct {
			Message   messageInfo `json:"message"`
			URL       string      `json:"url"`
			UserAgent string      `json:"user_agent"`
		}
		if err := json.Unmarshal(env.Payload, &lp); err != nil {
			return nil, fmt.Errorf("postal %s: %w", env.Event, err)
		}
		ev := baseEvent(lp.Message)
		ev.Type, ev.OriginalType = providers.EventOpen, env.Event
		if env.Event == "MessageLinkClicked" {
			ev.Type, ev.URL = providers.EventClick, lp.URL
		}
		ev.UserAgent = lp.UserAgent
		ev.OccurredAt = floatTime(env.Timestamp)
		ev.Raw = raw
		return []providers.Event{ev}, nil
	}
	// DomainDNSError and future events carry no delivery information.
	return []providers.Event{}, nil
}

func baseEvent(m messageInfo) providers.Event {
	ev := providers.Event{ProviderMessageID: m.ID.String(), Recipient: strings.ToLower(m.To)}
	if strings.HasPrefix(m.Tag, TagPrefix) {
		if id, err := uuid.Parse(strings.TrimPrefix(m.Tag, TagPrefix)); err == nil {
			ev.MessageID = &id
		}
	}
	return ev
}

func statusEvent(event string, sp statusPayload) providers.Event {
	ev := baseEvent(sp.Message)
	ev.OriginalType = event + " · " + sp.Status
	ev.OccurredAt = floatTime(sp.Timestamp)
	out := strings.TrimSpace(sp.Output)
	code, rest := bounce.Split(out)
	ev.OriginalCode = code
	ev.Diagnostic = rest
	if ev.Diagnostic == "" {
		ev.Diagnostic = strings.TrimSpace(sp.Details)
	}
	switch event {
	case "MessageSent":
		ev.Type = providers.EventDelivered
		ev.OriginalType = event
	case "MessageDelayed":
		ev.Type = providers.EventDeferred
		ev.BounceClass = bounce.Classify(code, out+" "+sp.Details)
	case "MessageHeld":
		ev.Type = providers.EventDeferred
		ev.BounceClass = bounce.Other
	case "MessageDeliveryFailed":
		// Postal reports HardFail after retries too; only a permanent SMTP
		// reply makes it a hard bounce.
		if sp.Status == "HardFail" && (code == "" || bounce.Permanent(code)) {
			ev.Type = providers.EventHardBounce
		} else {
			ev.Type = providers.EventSoftBounce
		}
		ev.BounceClass = bounce.Classify(code, out+" "+sp.Details)
	}
	return ev
}

func floatTime(f float64) time.Time {
	if f <= 0 {
		return time.Now().UTC()
	}
	sec := int64(f)
	return time.Unix(sec, int64((f-float64(sec))*1e9)).UTC()
}

// Backfill implements providers.EventBackfiller with
// /api/v1/messages/deliveries: the latest delivery of each message.
func (p *Provider) Backfill(ctx context.Context, ids []string) ([]providers.Event, error) {
	out := []providers.Event{}
	for _, id := range ids {
		n, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			continue
		}
		var deliveries []struct {
			Status    string  `json:"status"`
			Details   string  `json:"details"`
			Output    string  `json:"output"`
			Timestamp float64 `json:"timestamp"`
		}
		if err := p.call(ctx, "POST /api/v1/messages/deliveries", "/api/v1/messages/deliveries", map[string]any{"id": n}, &deliveries); err != nil {
			var ce *providers.CallError
			if errors.As(err, &ce) && ce.Code == "MessageNotFound" {
				continue
			}
			return out, err
		}
		if len(deliveries) == 0 {
			continue
		}
		var info struct {
			Details struct {
				RcptTo string `json:"rcpt_to"`
				Tag    string `json:"tag"`
			} `json:"details"`
		}
		_ = p.call(ctx, "POST /api/v1/messages/message", "/api/v1/messages/message", map[string]any{"id": n, "_expansions": []string{"details"}}, &info)
		d := deliveries[len(deliveries)-1]
		event := map[string]string{"Sent": "MessageSent", "SoftFail": "MessageDelayed", "HardFail": "MessageDeliveryFailed", "Held": "MessageHeld"}[d.Status]
		if event == "" {
			continue
		}
		sp := statusPayload{Message: messageInfo{ID: json.Number(id), To: info.Details.RcptTo, Tag: info.Details.Tag}, Status: d.Status, Details: d.Details, Output: d.Output, Timestamp: d.Timestamp}
		ev := statusEvent(event, sp)
		raw, _ := json.Marshal(map[string]any{"event": event, "source": "backfill", "payload": sp})
		ev.Raw = raw
		out = append(out, ev)
	}
	return out, nil
}

// keyCache holds a server's signing keys, shared by every adapter instance
// for that server so the JWKS is fetched at most every few minutes.
type keyCache struct {
	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey // by kid ("" for a pinned PEM key)
	fetched time.Time
}

var (
	cachesMu sync.Mutex
	caches   = map[string]*keyCache{}
)

func sharedKeys(server string) *keyCache {
	cachesMu.Lock()
	defer cachesMu.Unlock()
	c, ok := caches[server]
	if !ok {
		c = &keyCache{keys: map[string]*rsa.PublicKey{}}
		caches[server] = c
	}
	return c
}

func (c *keyCache) seed(jwks json.RawMessage, pemKey string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(jwks) > 0 {
		keys, err := ParseJWKS(jwks)
		if err != nil {
			return err
		}
		for kid, k := range keys {
			c.keys[kid] = k
		}
	}
	if strings.TrimSpace(pemKey) != "" {
		k, err := ParsePublicKey(pemKey)
		if err != nil {
			return err
		}
		c.keys[""] = k
	}
	return nil
}

// lookup returns the key for kid, fetching the server's JWKS when the kid
// is unknown (at most every five minutes); with no kid every key is tried.
func (c *keyCache) lookup(ctx context.Context, hc *http.Client, server, kid string) []*rsa.PublicKey {
	c.mu.Lock()
	defer c.mu.Unlock()
	if kid != "" {
		if k, ok := c.keys[kid]; ok {
			return []*rsa.PublicKey{k}
		}
	}
	if (kid != "" || len(c.keys) == 0) && server != "" && time.Since(c.fetched) > 5*time.Minute {
		c.fetched = time.Now()
		if keys, err := FetchJWKS(ctx, hc, server); err == nil {
			for id, k := range keys {
				c.keys[id] = k
			}
		}
		if k, ok := c.keys[kid]; ok && kid != "" {
			return []*rsa.PublicKey{k}
		}
	}
	out := make([]*rsa.PublicKey, 0, len(c.keys))
	for _, k := range c.keys {
		out = append(out, k)
	}
	return out
}

// FetchJWKS downloads a Postal server's webhook signing keys.
func FetchJWKS(ctx context.Context, hc *http.Client, server string) (map[string]*rsa.PublicKey, error) {
	c := apiclient.New(server, hc)
	raw, err := c.DoRaw(ctx, "GET /.well-known/jwks.json", http.MethodGet, "/.well-known/jwks.json", nil, nil)
	if err != nil {
		return nil, err
	}
	return ParseJWKS(raw)
}

// ParseJWKS parses RSA keys from a JWK set.
func ParseJWKS(raw []byte) (map[string]*rsa.PublicKey, error) {
	var set struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(raw, &set); err != nil {
		return nil, err
	}
	out := map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		n, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(k.N, "="))
		if err != nil {
			return nil, fmt.Errorf("jwk %s: %w", k.Kid, err)
		}
		e, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(k.E, "="))
		if err != nil {
			return nil, fmt.Errorf("jwk %s: %w", k.Kid, err)
		}
		out[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	if len(out) == 0 {
		return nil, errors.New("no RSA keys in the JWKS")
	}
	return out, nil
}

// ParsePublicKey reads a PEM ("BEGIN PUBLIC KEY" or "BEGIN RSA PUBLIC KEY")
// or base64 DER RSA public key.
func ParsePublicKey(s string) (*rsa.PublicKey, error) {
	s = strings.TrimSpace(s)
	var der []byte
	if block, _ := pem.Decode([]byte(s)); block != nil {
		der = block.Bytes
	} else {
		b, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(s), ""))
		if err != nil {
			return nil, errors.New("not a PEM or base64 public key")
		}
		der = b
	}
	if k, err := x509.ParsePKIXPublicKey(der); err == nil {
		if rk, ok := k.(*rsa.PublicKey); ok {
			return rk, nil
		}
		return nil, errors.New("not an RSA key")
	}
	return x509.ParsePKCS1PublicKey(der)
}
