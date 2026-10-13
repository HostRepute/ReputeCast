package mailgun

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/bounce"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

func basic(user, pass string) string {
	return base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
}

// payload is a Mailgun webhook request: the signature and the event.
type payload struct {
	Signature struct {
		Token     string          `json:"token"`
		Timestamp json.RawMessage `json:"timestamp"`
		Signature string          `json:"signature"`
	} `json:"signature"`
	EventData json.RawMessage `json:"event-data"`
}

// Sign returns the signature Mailgun computes for a webhook: hex
// HMAC-SHA256 of timestamp followed by token, keyed with the account's
// HTTP webhook signing key.
func Sign(key, timestamp, token string) string {
	return hex.EncodeToString(webhookauth.HMAC(sha256.New, []byte(key), []byte(timestamp), []byte(token)))
}

// VerifyWebhook implements providers.WebhookVerifier. Mailgun retries a
// delivery for up to eight hours with the same signature and documents no
// timestamp window, so stale timestamps are accepted; redelivered events
// are dropped by ingest's dedupe.
func (p *Provider) VerifyWebhook(ctx context.Context, _ *http.Request, body []byte) error {
	var pl payload
	if err := json.Unmarshal(body, &pl); err != nil {
		return fmt.Errorf("%w: not a Mailgun webhook", providers.ErrBadSignature)
	}
	ts := strings.Trim(strings.TrimSpace(string(pl.Signature.Timestamp)), `"`)
	if pl.Signature.Token == "" || ts == "" || pl.Signature.Signature == "" {
		return fmt.Errorf("%w: the request has no signature", providers.ErrBadSignature)
	}
	key, cached, err := p.signingKey(ctx, false)
	if err != nil {
		return fmt.Errorf("%w: %w", providers.ErrBadSignature, err)
	}
	if check(key, ts, pl.Signature.Token, pl.Signature.Signature) {
		return nil
	}
	if cached {
		// The key may have been regenerated: read it again (at most once a minute).
		if fresh, _, err := p.signingKey(ctx, true); err == nil && fresh != key && check(fresh, ts, pl.Signature.Token, pl.Signature.Signature) {
			return nil
		}
	}
	return providers.ErrBadSignature
}

func check(key, ts, token, sig string) bool {
	got, err := hex.DecodeString(strings.TrimSpace(sig))
	if err != nil {
		return false
	}
	want, _ := hex.DecodeString(Sign(key, ts, token))
	return hmac.Equal(got, want)
}

type keyEntry struct {
	mu      sync.Mutex
	key     string
	fetched time.Time
}

var (
	keysMu sync.Mutex
	keys   = map[string]*keyEntry{}
)

// signingKey returns the account's HTTP webhook signing key: the one
// entered at connect time, or the one GET /v5/accounts/http_signing_key
// reports (cached, shared by adapter instances).
func (p *Provider) signingKey(ctx context.Context, refresh bool) (string, bool, error) {
	if k := strings.TrimSpace(p.c.WebhookSigningKey); k != "" {
		return k, false, nil
	}
	sum := sha256.Sum256([]byte(p.api.BaseURL + "\x00" + p.c.APIKey))
	id := hex.EncodeToString(sum[:])
	keysMu.Lock()
	e, ok := keys[id]
	if !ok {
		e = &keyEntry{}
		keys[id] = e
	}
	keysMu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.key != "" && (!refresh || time.Since(e.fetched) < time.Minute) {
		return e.key, true, nil
	}
	var out struct {
		Key string `json:"http_signing_key"`
	}
	if err := p.api.Do(ctx, "GET /v5/accounts/http_signing_key", http.MethodGet, "/v5/accounts/http_signing_key", nil, &out); err != nil {
		return "", false, err
	}
	if out.Key == "" {
		return "", false, errors.New("no HTTP webhook signing key in Mailgun's answer")
	}
	e.key, e.fetched = out.Key, time.Now()
	return e.key, true, nil
}

// eventData is a Mailgun event, as webhooks and the events API carry it.
type eventData struct {
	Event          string          `json:"event"`
	ID             string          `json:"id"`
	Timestamp      float64         `json:"timestamp"`
	Recipient      string          `json:"recipient"`
	Severity       string          `json:"severity"`
	Reason         string          `json:"reason"`
	URL            string          `json:"url"`
	UserVariables  json.RawMessage `json:"user-variables"`
	DeliveryStatus struct {
		Code         json.RawMessage `json:"code"`
		Message      string          `json:"message"`
		Description  *string         `json:"description"`
		EnhancedCode *string         `json:"enhanced-code"`
		BounceType   string          `json:"bounce-type"`
	} `json:"delivery-status"`
	Message struct {
		Headers struct {
			MessageID string `json:"message-id"`
		} `json:"headers"`
	} `json:"message"`
	ClientInfo struct {
		UserAgent string `json:"user-agent"`
	} `json:"client-info"`
	Reject struct {
		Reason      string `json:"reason"`
		Description string `json:"description"`
	} `json:"reject"`
}

// ParseWebhook implements providers.Provider: one event per request.
func (p *Provider) ParseWebhook(ctx context.Context, r *http.Request, body []byte) ([]providers.Event, error) {
	if !providers.SignatureVerified(ctx) {
		if err := p.VerifyWebhook(ctx, r, body); err != nil {
			return nil, err
		}
	}
	var pl payload
	if err := json.Unmarshal(body, &pl); err != nil {
		return nil, fmt.Errorf("mailgun webhook: %w", err)
	}
	var ed eventData
	if err := json.Unmarshal(pl.EventData, &ed); err != nil {
		return nil, fmt.Errorf("mailgun event: %w", err)
	}
	ev, ok := normalize(ed)
	if !ok {
		return []providers.Event{}, nil
	}
	ev.Raw = append(json.RawMessage(nil), body...)
	return []providers.Event{ev}, nil
}

func normalize(ed eventData) (providers.Event, bool) {
	ev := providers.Event{ProviderMessageID: strings.Trim(strings.TrimSpace(ed.Message.Headers.MessageID), "<>"),
		Recipient: strings.ToLower(strings.TrimSpace(ed.Recipient)), OriginalType: ed.Event, OccurredAt: floatTime(ed.Timestamp)}
	if id, err := uuid.Parse(userVariable(ed.UserVariables, VarMessage)); err == nil {
		ev.MessageID = &id
	}
	ds := ed.DeliveryStatus
	code := deliveryCode(ds.Code, ds.EnhancedCode, ds.Message)
	diag := diagnostic(ds.Message, ds.Description)
	switch ed.Event {
	case "delivered":
		ev.Type, ev.OriginalCode, ev.Diagnostic = providers.EventDelivered, code, diag
	case "failed":
		ev.OriginalType = joinNonEmpty(" · ", "failed", ed.Severity, ed.Reason)
		ev.OriginalCode, ev.Diagnostic = code, diag
		ev.BounceClass = bounce.Classify(code, ds.Message+" "+diag)
		switch {
		case strings.HasPrefix(ed.Reason, "suppress-"):
			// Mailgun didn't try: the address is on its suppression list.
			ev.Type = providers.EventRejected
			ev.BounceClass = map[string]string{"suppress-bounce": bounce.MailboxUnknown, "suppress-complaint": bounce.PolicyBlock}[ed.Reason]
			if ev.BounceClass == "" {
				ev.BounceClass = bounce.Other
			}
		case ed.Severity != "permanent":
			// Mailgun retries temporary failures.
			ev.Type = providers.EventDeferred
		case ds.BounceType == "soft", ed.Reason == "old", ed.Reason == "espblock",
			ev.BounceClass == bounce.SpamBlock || ev.BounceClass == bounce.Reputation || ev.BounceClass == bounce.PolicyBlock:
			// Given up, but not because of the address: retries ran out,
			// or the receiver blocked the sender.
			ev.Type = providers.EventSoftBounce
		default:
			ev.Type = providers.EventHardBounce
		}
	case "rejected":
		ev.Type = providers.EventRejected
		ev.Diagnostic = joinNonEmpty(" · ", ed.Reject.Reason, ed.Reject.Description)
		ev.BounceClass = bounce.Other
	case "complained":
		ev.Type = providers.EventComplaint
		ev.Diagnostic = "Spam complaint via Mailgun"
	case "unsubscribed":
		ev.Type = providers.EventUnsubscribe
	case "opened":
		ev.Type, ev.UserAgent = providers.EventOpen, ed.ClientInfo.UserAgent
	case "clicked":
		ev.Type, ev.URL, ev.UserAgent = providers.EventClick, ed.URL, ed.ClientInfo.UserAgent
	default:
		// accepted, stored, list_member_uploaded…
		return ev, false
	}
	return ev, true
}

// deliveryCode joins the SMTP reply code and its enhanced code ("550
// 5.1.1"), taking the enhanced code from the message when Mailgun leaves
// the field empty.
func deliveryCode(raw json.RawMessage, enhanced *string, message string) string {
	code := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	if code == "null" || code == "0" {
		code = ""
	}
	enh := ""
	if enhanced != nil {
		enh = strings.TrimSpace(*enhanced)
	}
	if enh == "" {
		if c, _ := bounce.Split(message); bounce.Enhanced(c) != "" {
			enh = bounce.Enhanced(c)
		}
	}
	return joinNonEmpty(" ", code, enh)
}

func diagnostic(message string, description *string) string {
	_, rest := bounce.Split(message)
	if rest == "" {
		rest = strings.TrimSpace(message)
	}
	if description != nil {
		return joinNonEmpty(" · ", rest, *description)
	}
	return rest
}

// userVariable reads a custom variable; user-variables is an object, or
// an empty array when there are none.
func userVariable(raw json.RawMessage, name string) string {
	var vars map[string]any
	if json.Unmarshal(raw, &vars) != nil {
		return ""
	}
	switch v := vars[name].(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func floatTime(f float64) time.Time {
	if f <= 0 || math.IsInf(f, 0) || math.IsNaN(f) {
		return time.Now().UTC()
	}
	sec := math.Floor(f)
	return time.Unix(int64(sec), int64((f-sec)*1e9)).UTC()
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
