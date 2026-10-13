package brevo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/bounce"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

// VerifyWebhook implements providers.WebhookVerifier: Brevo doesn't sign
// webhooks, so ReputeCast registers them with bearer authentication — the
// provider's webhook secret, which Brevo sends as "Authorization: Bearer"
// on every call.
func (p *Provider) VerifyWebhook(_ context.Context, r *http.Request, _ []byte) error {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || !webhookauth.Equal(strings.TrimSpace(token), p.cfg.WebhookSecret) {
		return providers.ErrBadSignature
	}
	return nil
}

// event is one Brevo transactional webhook event.
type event struct {
	Event     string          `json:"event"`
	Email     string          `json:"email"`
	Date      string          `json:"date"`
	TSEvent   json.Number     `json:"ts_event"`
	TSEpoch   json.Number     `json:"ts_epoch"`
	MessageID string          `json:"message-id"`
	Reason    string          `json:"reason"`
	Custom    string          `json:"X-Mailin-custom"`
	Link      string          `json:"link"`
	UserAgent string          `json:"user_agent"`
	Tags      json.RawMessage `json:"tags"`
}

// ParseWebhook implements providers.Provider: one event per call, or an
// array when the webhook is batched.
func (p *Provider) ParseWebhook(ctx context.Context, r *http.Request, body []byte) ([]providers.Event, error) {
	if !providers.SignatureVerified(ctx) {
		if err := p.VerifyWebhook(ctx, r, body); err != nil {
			return nil, err
		}
	}
	trimmed := bytes.TrimSpace(body)
	items := []json.RawMessage{trimmed}
	if len(trimmed) > 0 && trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return nil, fmt.Errorf("brevo webhook: %w", err)
		}
	}
	out := []providers.Event{}
	for _, raw := range items {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var e event
		if dec.Decode(&e) != nil || e.Event == "" {
			continue
		}
		if ev, ok := normalize(e); ok {
			ev.Raw = append(json.RawMessage(nil), raw...)
			out = append(out, ev)
		}
	}
	return out, nil
}

// customValue reads key from X-Mailin-custom ("k1:v1|k2:v2").
func customValue(custom, key string) string {
	for _, pair := range strings.Split(custom, "|") {
		if k, v, ok := strings.Cut(pair, ":"); ok && strings.TrimSpace(k) == key {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func normalize(e event) (providers.Event, bool) {
	ev := providers.Event{ProviderMessageID: strings.Trim(strings.TrimSpace(e.MessageID), "<>"), Recipient: strings.ToLower(strings.TrimSpace(e.Email)),
		OriginalType: e.Event, OccurredAt: eventTime(e)}
	if id, err := uuid.Parse(customValue(e.Custom, "rc_message")); err == nil {
		ev.MessageID = &id
	}
	code, rest := bounce.Split(e.Reason)
	switch e.Event {
	case "delivered":
		ev.Type = providers.EventDelivered
	case "hard_bounce", "hardBounce":
		ev.Type = providers.EventHardBounce
		ev.OriginalCode, ev.Diagnostic = code, rest
		ev.BounceClass = bounce.Classify(code, e.Reason)
		switch ev.BounceClass {
		case bounce.SpamBlock, bounce.Reputation, bounce.PolicyBlock:
			// Refused because of the sender, not the address.
			ev.Type = providers.EventSoftBounce
		}
	case "soft_bounce", "softBounce":
		ev.Type = providers.EventSoftBounce
		ev.OriginalCode, ev.Diagnostic = code, rest
		ev.BounceClass = bounce.Classify(code, e.Reason)
	case "deferred":
		ev.Type = providers.EventDeferred
		ev.OriginalCode, ev.Diagnostic = code, rest
		ev.BounceClass = bounce.Classify(code, e.Reason)
		if ev.BounceClass == bounce.Other {
			ev.BounceClass = bounce.Temporary
		}
	case "invalid_email", "invalid":
		// Brevo found the address invalid: it will never take mail.
		ev.Type = providers.EventHardBounce
		ev.Diagnostic = firstNonEmpty(e.Reason, "Invalid email address")
		ev.BounceClass = bounce.MailboxUnknown
	case "blocked":
		ev.Type = providers.EventRejected
		ev.Diagnostic = firstNonEmpty(e.Reason, "Brevo blocked the message: the contact is on its blocklist")
		ev.BounceClass = bounce.PolicyBlock
	case "error":
		ev.Type = providers.EventRejected
		ev.Diagnostic = firstNonEmpty(e.Reason, "Brevo couldn't send the message")
		ev.BounceClass = bounce.Other
	case "spam":
		ev.Type = providers.EventComplaint
		ev.Diagnostic = "Spam complaint via Brevo"
	case "unsubscribed":
		ev.Type = providers.EventUnsubscribe
	case "opened", "unique_opened":
		ev.Type, ev.UserAgent = providers.EventOpen, e.UserAgent
	case "click":
		ev.Type, ev.URL, ev.UserAgent = providers.EventClick, e.Link, e.UserAgent
	default:
		// request (sent) and proxy opens (machine opens).
		return ev, false
	}
	return ev, true
}

// eventTime reads ts_epoch (milliseconds, sometimes seconds), else
// ts_event (seconds, UTC), else date (Brevo's CET/CEST wall time).
func eventTime(e event) time.Time {
	if n, err := e.TSEpoch.Int64(); err == nil && n > 0 {
		if n > 1e12 {
			return time.UnixMilli(n).UTC()
		}
		return time.Unix(n, 0).UTC()
	}
	if n, err := e.TSEvent.Int64(); err == nil && n > 0 {
		return time.Unix(n, 0).UTC()
	}
	if loc, err := time.LoadLocation("Europe/Paris"); err == nil {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", e.Date, loc); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
