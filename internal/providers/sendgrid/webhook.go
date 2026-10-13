package sendgrid

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/bounce"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

// Headers of the signed Event Webhook.
const (
	SignatureHeader = "X-Twilio-Email-Event-Webhook-Signature"
	TimestampHeader = "X-Twilio-Email-Event-Webhook-Timestamp"
)

// Verify checks a signed Event Webhook payload: an ECDSA P-256 signature
// (base64 ASN.1) over the timestamp header followed by the raw body,
// verified with the webhook's public key (base64 DER). It checks the
// signature only; VerifyWebhook also bounds the timestamp's age (Window).
func Verify(publicKey, signature, timestamp string, body []byte) error {
	der, err := base64.StdEncoding.DecodeString(strings.TrimSpace(publicKey))
	if err != nil {
		return fmt.Errorf("%w: the verification key is not base64", providers.ErrBadSignature)
	}
	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return fmt.Errorf("%w: the verification key is not a public key", providers.ErrBadSignature)
	}
	pub, ok := parsed.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("%w: the verification key is not an ECDSA key", providers.ErrBadSignature)
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature))
	if err != nil || len(sig) == 0 {
		return providers.ErrBadSignature
	}
	h := sha256.New()
	h.Write([]byte(timestamp))
	h.Write(body)
	if !ecdsa.VerifyASN1(pub, h.Sum(nil), sig) {
		return providers.ErrBadSignature
	}
	return nil
}

// Window is how old a signed timestamp may be. SendGrid retries a delivery
// for 24 hours and doesn't document whether a retry is signed again, so the
// window spans the retries. The signature covers the body: a replay within
// it repeats events ingest's dedupe drops.
const Window = 26 * time.Hour

// now is the clock, replaced in tests.
var now = time.Now

// VerifyWebhook implements providers.WebhookVerifier.
func (p *Provider) VerifyWebhook(ctx context.Context, r *http.Request, body []byte) error {
	sig, ts := r.Header.Get(SignatureHeader), r.Header.Get(TimestampHeader)
	if sig == "" || ts == "" {
		return fmt.Errorf("%w: the request is not signed; turn on signature verification for the Event Webhook", providers.ErrBadSignature)
	}
	check := func(key string) error {
		if err := Verify(key, sig, ts, body); err != nil {
			return err
		}
		return webhookauth.Within(ts, now(), Window)
	}
	if k := strings.TrimSpace(p.s.WebhookVerificationKey); k != "" {
		return check(k)
	}
	path := r.URL.Path
	if path == "" || path == "/" {
		return fmt.Errorf("%w: no verification key is configured", providers.ErrBadSignature)
	}
	// Signing may have been turned off and on again (a new key): the
	// cache reads the key again when a signature doesn't verify.
	return verificationKeys.Verify(ctx, p.keyID(path), func(ctx context.Context) (string, error) { return p.fetchVerificationKey(ctx, path) }, check)
}

// verificationKeys caches the public keys read from SendGrid's API, shared
// by every adapter instance.
var verificationKeys webhookauth.Material

func (p *Provider) keyID(path string) string {
	return webhookauth.Key(p.api.BaseURL, p.c.APIKey, strings.TrimRight(path, "/"))
}

// fetchVerificationKey reads the public key of the webhook whose URL ends
// with path.
func (p *Provider) fetchVerificationKey(ctx context.Context, path string) (string, error) {
	hooks, err := p.listWebhooks(ctx)
	if err != nil {
		return "", err
	}
	want := strings.TrimRight(path, "/")
	for _, h := range hooks {
		hp := strings.TrimRight(pathOf(h.URL), "/")
		// The suffix allows a reverse proxy that strips a path prefix.
		if hp != want && !strings.HasSuffix(hp, want) {
			continue
		}
		key := h.PublicKey
		if key == "" {
			var signed struct {
				PublicKey string `json:"public_key"`
			}
			if err := p.api.Do(ctx, "GET /v3/user/webhooks/event/settings/signed/{id}", http.MethodGet, "/v3/user/webhooks/event/settings/signed/"+url.PathEscape(h.ID), nil, &signed); err != nil {
				return "", err
			}
			key = signed.PublicKey
		}
		if key == "" {
			return "", errors.New("signature verification is off for this Event Webhook; register the webhook again to turn it on")
		}
		return key, nil
	}
	return "", errors.New("no SendGrid Event Webhook points at this URL; register it again or enter its verification key")
}

func pathOf(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return parsed.Path
}

// event is one Event Webhook record. Custom args arrive as top-level keys.
type event struct {
	Email          string          `json:"email"`
	Timestamp      json.RawMessage `json:"timestamp"`
	Event          string          `json:"event"`
	SGMessageID    string          `json:"sg_message_id"`
	Reason         string          `json:"reason"`
	Status         string          `json:"status"`
	Response       string          `json:"response"`
	Type           string          `json:"type"`
	Classification string          `json:"bounce_classification"`
	URL            string          `json:"url"`
	UserAgent      string          `json:"useragent"`
	RCMessage      string          `json:"rc_message"`
}

// ParseWebhook implements providers.Provider: the body is a JSON array of
// events.
func (p *Provider) ParseWebhook(ctx context.Context, r *http.Request, body []byte) ([]providers.Event, error) {
	if !providers.SignatureVerified(ctx) {
		if err := p.VerifyWebhook(ctx, r, body); err != nil {
			return nil, err
		}
	}
	var items []json.RawMessage
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("sendgrid webhook: %w", err)
	}
	out := []providers.Event{}
	for _, raw := range items {
		var e event
		if err := json.Unmarshal(raw, &e); err != nil {
			continue
		}
		if ev, ok := normalize(e); ok {
			ev.Raw = append(json.RawMessage(nil), raw...)
			out = append(out, ev)
		}
	}
	return out, nil
}

// MessageID returns the X-Message-Id an event's sg_message_id starts with
// ("XBg2anf2TqCy6WXKQFhieQ.filter0905p1mdw1-…" → "XBg2anf2TqCy6WXKQFhieQ").
func MessageID(sgMessageID string) string {
	s := strings.TrimSpace(sgMessageID)
	for _, marker := range []string{".filter", ".recvd", ".geopod"} {
		if i := strings.Index(s, marker); i > 0 {
			return s[:i]
		}
	}
	if i := strings.IndexByte(s, '.'); i > 0 {
		return s[:i]
	}
	return s
}

func normalize(e event) (providers.Event, bool) {
	ev := providers.Event{ProviderMessageID: MessageID(e.SGMessageID), Recipient: strings.ToLower(strings.TrimSpace(e.Email)),
		OccurredAt: unixTime(e.Timestamp), OriginalType: e.Event}
	if id, err := uuid.Parse(e.RCMessage); err == nil {
		ev.MessageID = &id
	}
	switch e.Event {
	case "delivered":
		ev.Type = providers.EventDelivered
		ev.OriginalCode, ev.Diagnostic = bounce.Split(e.Response)
	case "deferred":
		ev.Type = providers.EventDeferred
		ev.OriginalCode, ev.Diagnostic = bounce.Split(e.Response)
		ev.BounceClass = bounce.Classify(ev.OriginalCode, e.Response)
		if ev.BounceClass == bounce.Other {
			ev.BounceClass = bounce.Temporary
		}
	case "bounce":
		code, rest := bounce.Split(e.Reason)
		if bounce.Enhanced(code) == "" && e.Status != "" {
			code = strings.TrimSpace(code + " " + e.Status)
		}
		ev.OriginalCode, ev.Diagnostic = code, rest
		kind := e.Type
		if kind == "bounce" {
			kind = ""
		}
		ev.OriginalType = joinNonEmpty(" · ", "bounce", kind, e.Classification)
		ev.BounceClass = classOf(e.Classification, code, e.Reason)
		ev.Type = providers.EventHardBounce
		switch {
		case kind == "blocked":
			// A block is about the sender (IP, content), not the address.
			ev.Type = providers.EventSoftBounce
		case e.Classification == "Content" || e.Classification == "Reputation" || e.Classification == "Frequency/Volume":
			ev.Type = providers.EventSoftBounce
		case code != "" && !bounce.Permanent(code):
			ev.Type = providers.EventSoftBounce
		}
	case "dropped":
		ev.Type = providers.EventRejected
		ev.OriginalType = joinNonEmpty(" · ", "dropped", e.Reason)
		ev.OriginalCode = e.Status
		ev.Diagnostic = "SendGrid dropped the message: " + e.Reason
		switch e.Reason {
		case "Bounced Address", "Invalid":
			ev.BounceClass = bounce.MailboxUnknown
		case "Spam Reporting Address", "Unsubscribed Address":
			ev.BounceClass = bounce.PolicyBlock
		default:
			ev.BounceClass = bounce.Other
		}
	case "spamreport":
		ev.Type = providers.EventComplaint
		ev.Diagnostic = "Spam report via SendGrid"
	case "unsubscribe", "group_unsubscribe":
		ev.Type = providers.EventUnsubscribe
	case "open":
		ev.Type = providers.EventOpen
		ev.UserAgent = e.UserAgent
	case "click":
		ev.Type, ev.URL, ev.UserAgent = providers.EventClick, e.URL, e.UserAgent
	default:
		// processed, group_resubscribe, account_status_change.
		return ev, false
	}
	if ev.OccurredAt.IsZero() {
		ev.OccurredAt = time.Now().UTC()
	}
	return ev, true
}

// classOf maps SendGrid's bounce classification to the shared taxonomy,
// reading the SMTP reply where the classification is broad.
func classOf(classification, code, reason string) string {
	switch classification {
	case "Invalid Address":
		return bounce.MailboxUnknown
	case "Content":
		return bounce.SpamBlock
	case "Reputation", "Frequency/Volume":
		return bounce.Reputation
	case "Mailbox Unavailable":
		if c := bounce.Classify(code, reason); c != bounce.Other {
			return c
		}
		return bounce.MailboxFull
	}
	return bounce.Classify(code, reason)
}

// unixTime reads Unix seconds (or milliseconds), as a number or a string.
func unixTime(raw json.RawMessage) time.Time {
	s := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= 0 || math.IsInf(f, 0) {
		return time.Time{}
	}
	if f > 1e12 {
		f /= 1000
	}
	return time.Unix(int64(f), 0).UTC()
}

func joinNonEmpty(sep string, parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, strings.TrimSpace(p))
		}
	}
	return strings.Join(out, sep)
}
