package resend

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/providers/bounce"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

// Events ReputeCast subscribes to. Opens and clicks are tracked by
// ReputeCast itself.
var webhookEvents = []string{"email.delivered", "email.delivery_delayed", "email.bounced", "email.complained", "email.failed", "email.suppressed"}

// now is the clock, replaced in tests.
var now = time.Now

// VerifyWebhook implements providers.WebhookVerifier: Resend signs webhooks
// with Standard Webhooks (svix-id, svix-timestamp, svix-signature) and the
// webhook's whsec_ signing secret. The secret is the one entered at connect
// time, or read from Resend's API for the webhook pointing at this URL.
func (p *Provider) VerifyWebhook(ctx context.Context, r *http.Request, body []byte) error {
	secret, cached, err := p.signingSecret(ctx, r.URL.Path, false)
	if err != nil {
		return fmt.Errorf("%w: %w", providers.ErrBadSignature, err)
	}
	err = webhookauth.Standard(secret, r.Header, body, now())
	if err == nil || !cached || errors.Is(err, webhookauth.ErrStale) {
		return err
	}
	// The secret may have been rotated: read it again (at most once a minute).
	fresh, _, ferr := p.signingSecret(ctx, r.URL.Path, true)
	if ferr != nil || fresh == secret {
		return err
	}
	return webhookauth.Standard(fresh, r.Header, body, now())
}

// secretEntry caches one webhook's signing secret, shared by every adapter
// instance (the factory builds one per request).
type secretEntry struct {
	mu      sync.Mutex
	secret  string
	fetched time.Time
}

var (
	secretsMu sync.Mutex
	secrets   = map[string]*secretEntry{}
)

// signingSecret returns the secret webhooks posted to path are signed with
// and whether it came from the API (and may be stale).
func (p *Provider) signingSecret(ctx context.Context, path string, refresh bool) (string, bool, error) {
	if s := strings.TrimSpace(p.c.WebhookSigningSecret); s != "" {
		return s, false, nil
	}
	if path == "" || path == "/" {
		return "", false, errors.New("no signing secret is configured")
	}
	key := sha256.Sum256([]byte(p.api.BaseURL + "\x00" + p.c.APIKey + "\x00" + path))
	secretsMu.Lock()
	e, ok := secrets[hex.EncodeToString(key[:])]
	if !ok {
		e = &secretEntry{}
		secrets[hex.EncodeToString(key[:])] = e
	}
	secretsMu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.secret != "" && (!refresh || time.Since(e.fetched) < time.Minute) {
		return e.secret, true, nil
	}
	hook, err := p.findWebhook(ctx, func(endpoint string) bool {
		return pathOf(endpoint) == path || strings.HasSuffix(pathOf(endpoint), path)
	})
	if err != nil {
		return "", false, err
	}
	if hook == nil {
		return "", false, errors.New("no Resend webhook points at this URL; register it again or enter its signing secret")
	}
	var full struct {
		SigningSecret string `json:"signing_secret"`
	}
	if err := p.api.Do(ctx, "GET /webhooks/{id}", http.MethodGet, apiclient.Path("/webhooks/%s", hook.ID), nil, &full); err != nil {
		return "", false, err
	}
	if full.SigningSecret == "" {
		return "", false, errors.New("the webhook's signing secret was missing from Resend's answer")
	}
	e.secret, e.fetched = full.SigningSecret, time.Now()
	return e.secret, true, nil
}

func pathOf(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	return u.Path
}

// event is a Resend webhook payload.
type event struct {
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
	Data      struct {
		EmailID   string          `json:"email_id"`
		CreatedAt string          `json:"created_at"`
		To        []string        `json:"to"`
		Tags      json.RawMessage `json:"tags"`
		Bounce    *struct {
			Message        string          `json:"message"`
			SubType        string          `json:"subType"`
			Type           string          `json:"type"`
			DiagnosticCode json.RawMessage `json:"diagnosticCode"`
		} `json:"bounce"`
		Failed *struct {
			Reason string `json:"reason"`
		} `json:"failed"`
		Suppressed *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Reason  string `json:"reason"`
		} `json:"suppressed"`
		Click *struct {
			Link      string `json:"link"`
			Timestamp string `json:"timestamp"`
			UserAgent string `json:"userAgent"`
		} `json:"click"`
		Open *struct {
			Timestamp string `json:"timestamp"`
			UserAgent string `json:"userAgent"`
		} `json:"open"`
	} `json:"data"`
}

// ParseWebhook implements providers.Provider.
func (p *Provider) ParseWebhook(ctx context.Context, r *http.Request, body []byte) ([]providers.Event, error) {
	if !providers.SignatureVerified(ctx) {
		if err := p.VerifyWebhook(ctx, r, body); err != nil {
			return nil, err
		}
	}
	var ev event
	if err := json.Unmarshal(body, &ev); err != nil {
		return nil, fmt.Errorf("resend webhook: %w", err)
	}
	evs := normalize(ev)
	for i := range evs {
		evs[i].Raw = append(json.RawMessage(nil), body...)
	}
	return evs, nil
}

func normalize(ev event) []providers.Event {
	base := providers.Event{ProviderMessageID: ev.Data.EmailID, OriginalType: ev.Type, OccurredAt: parseTime(ev.CreatedAt)}
	if base.OccurredAt.IsZero() {
		base.OccurredAt = parseTime(ev.Data.CreatedAt)
	}
	if id, err := uuid.Parse(tagValue(ev.Data.Tags, TagMessage)); err == nil {
		base.MessageID = &id
	}
	switch ev.Type {
	case "email.delivered":
		base.Type = providers.EventDelivered
	case "email.delivery_delayed":
		base.Type = providers.EventDeferred
		base.BounceClass = bounce.Temporary
	case "email.complained":
		base.Type = providers.EventComplaint
		base.Diagnostic = "Spam complaint reported to Resend"
	case "email.bounced":
		b := ev.Data.Bounce
		if b == nil {
			return []providers.Event{}
		}
		base.Type = providers.EventSoftBounce
		if strings.EqualFold(b.Type, "Permanent") {
			base.Type = providers.EventHardBounce
		}
		base.OriginalType = joinNonEmpty(" · ", ev.Type, b.Type, b.SubType)
		reply := firstDiagnostic(b.DiagnosticCode)
		code, rest := bounce.Split(reply)
		base.OriginalCode = code
		base.Diagnostic = joinNonEmpty(" · ", rest, b.Message)
		base.BounceClass = bounceClass(b.SubType, code, reply+" "+b.Message)
	case "email.failed":
		base.Type = providers.EventRejected
		if f := ev.Data.Failed; f != nil {
			base.OriginalCode = f.Reason
			base.Diagnostic = "Resend didn't send the message: " + strings.ReplaceAll(f.Reason, "_", " ")
		}
		base.BounceClass = bounce.Other
	case "email.suppressed":
		base.Type = providers.EventRejected
		base.BounceClass = bounce.Other
		if s := ev.Data.Suppressed; s != nil {
			base.OriginalType = joinNonEmpty(" · ", ev.Type, s.Type)
			base.OriginalCode = s.Reason
			base.Diagnostic = s.Message
			switch s.Reason {
			case "previous_bounce":
				base.BounceClass = bounce.MailboxUnknown
			case "previous_complaint":
				base.BounceClass = bounce.PolicyBlock
			}
		}
	case "email.opened":
		base.Type = providers.EventOpen
		if o := ev.Data.Open; o != nil {
			base.UserAgent = o.UserAgent
			if t := parseTime(o.Timestamp); !t.IsZero() {
				base.OccurredAt = t
			}
		}
	case "email.clicked":
		base.Type = providers.EventClick
		if c := ev.Data.Click; c != nil {
			base.URL, base.UserAgent = c.Link, c.UserAgent
			if t := parseTime(c.Timestamp); !t.IsZero() {
				base.OccurredAt = t
			}
		}
	default:
		// email.sent, email.scheduled, email.received, domain and contact
		// events carry no delivery outcome.
		return []providers.Event{}
	}
	if base.OccurredAt.IsZero() {
		base.OccurredAt = now().UTC()
	}
	out := make([]providers.Event, 0, len(ev.Data.To))
	for _, to := range ev.Data.To {
		e := base
		e.Recipient = strings.ToLower(strings.TrimSpace(addressOf(to)))
		out = append(out, e)
	}
	if len(out) == 0 {
		out = append(out, base)
	}
	return out
}

// bounceClass prefers the SMTP reply's meaning and falls back to Resend's
// bounce subtype.
func bounceClass(subType, code, text string) string {
	if c := bounce.Classify(code, text); c != bounce.Other && c != bounce.Temporary {
		return c
	}
	switch subType {
	case "NoEmail", "Suppressed":
		return bounce.MailboxUnknown
	case "MailboxFull":
		return bounce.MailboxFull
	case "ContentRejected", "AttachmentRejected":
		return bounce.SpamBlock
	case "MessageTooLarge":
		return bounce.PolicyBlock
	}
	return bounce.Classify(code, text)
}

// firstDiagnostic reads bounce.diagnosticCode, a list of SMTP replies (or a
// single one).
func firstDiagnostic(raw json.RawMessage) string {
	var list []string
	if json.Unmarshal(raw, &list) == nil && len(list) > 0 {
		return strings.TrimSpace(list[0])
	}
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return strings.TrimSpace(one)
	}
	return ""
}

// tagValue reads a tag from webhook tags ({"name": "value"}) or from the
// API's [{"name", "value"}] form.
func tagValue(raw json.RawMessage, name string) string {
	var obj map[string]string
	if json.Unmarshal(raw, &obj) == nil {
		return obj[name]
	}
	var list []tag
	if json.Unmarshal(raw, &list) == nil {
		for _, t := range list {
			if t.Name == name {
				return t.Value
			}
		}
	}
	return ""
}

// addressOf extracts the address from "Name <addr>" or a bare address.
func addressOf(s string) string {
	if i := strings.LastIndexByte(s, '<'); i >= 0 {
		if j := strings.IndexByte(s[i:], '>'); j > 0 {
			return s[i+1 : i+j]
		}
	}
	return s
}

// parseTime reads RFC 3339 (webhooks) and Postgres-style times (the REST
// API: "2026-04-03 22:13:42.674981+00").
func parseTime(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07", "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05-07"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
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
