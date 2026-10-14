package mailjet

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

// VerifyWebhook implements providers.WebhookVerifier: Mailjet doesn't sign
// event callbacks, so the callback URL ReputeCast registers carries basic
// auth credentials.
func (p *Provider) VerifyWebhook(_ context.Context, r *http.Request, _ []byte) error {
	return webhookauth.Basic(r, WebhookUser, p.cfg.WebhookSecret)
}

// event is one Mailjet event (a callback carries one, or an array when
// grouped).
type event struct {
	Event          string      `json:"event"`
	Time           int64       `json:"time"`
	MessageID      json.Number `json:"MessageID"`
	Email          string      `json:"email"`
	CustomID       string      `json:"CustomID"`
	SMTPReply      string      `json:"smtp_reply"`
	HardBounce     bool        `json:"hard_bounce"`
	ErrorRelatedTo string      `json:"error_related_to"`
	Error          string      `json:"error"`
	Comment        string      `json:"comment"`
	Source         string      `json:"source"`
	URL            string      `json:"url"`
	Agent          string      `json:"agent"`
}

// ParseWebhook implements providers.Provider. Mailjet's "Send test" posts
// an empty payload, which has no events.
func (p *Provider) ParseWebhook(ctx context.Context, r *http.Request, body []byte) ([]providers.Event, error) {
	if !providers.SignatureVerified(ctx) {
		if err := p.VerifyWebhook(ctx, r, body); err != nil {
			return nil, err
		}
	}
	trimmed := bytes.TrimSpace(body)
	var items []json.RawMessage
	switch {
	case len(trimmed) == 0:
		return []providers.Event{}, nil
	case trimmed[0] == '[':
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return nil, fmt.Errorf("mailjet webhook: %w", err)
		}
	default:
		items = []json.RawMessage{trimmed}
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

// errorClass maps Mailjet's error values to the shared taxonomy.
var errorClass = map[string]string{
	"user unknown": bounce.MailboxUnknown, "mailbox inactive": bounce.MailboxUnknown, "quota exceeded": bounce.MailboxFull,
	"invalid domain": bounce.DomainInvalid, "no mail host": bounce.DomainInvalid, "typofix": bounce.DomainInvalid, "domain error": bounce.DomainInvalid,
	"relay/access denied": bounce.PolicyBlock, "policy issue": bounce.PolicyBlock, "blacklisted": bounce.PolicyBlock, "spam reporter": bounce.PolicyBlock,
	"preblocked": bounce.PolicyBlock, "sender blocked": bounce.SpamBlock, "content blocked": bounce.SpamBlock,
	"greylisted": bounce.Temporary, "system issue": bounce.Temporary, "protocol issue": bounce.Temporary, "connection issue": bounce.Temporary,
}

// senderSide are error categories about the sender or Mailjet, not the
// recipient's address.
var senderSide = map[string]bool{"spam": true, "system": true, "mailjet": true, "content": true}

func normalize(e event) (providers.Event, bool) {
	ev := providers.Event{ProviderMessageID: e.MessageID.String(), Recipient: strings.ToLower(strings.TrimSpace(e.Email)), OriginalType: e.Event}
	if ev.ProviderMessageID == "0" {
		ev.ProviderMessageID = ""
	}
	if id, err := uuid.Parse(e.CustomID); err == nil {
		ev.MessageID = &id
	}
	ev.OccurredAt = time.Now().UTC()
	if e.Time > 0 {
		ev.OccurredAt = time.Unix(e.Time, 0).UTC()
	}
	code, rest := bounce.Split(e.Comment)
	if code == "" {
		code = e.Error
	}
	class := errorClass[strings.ToLower(e.Error)]
	if class == "" && e.Error != "" {
		class = bounce.Classify(code, e.Comment+" "+e.Error)
	}
	switch e.Event {
	case "sent":
		ev.Type = providers.EventDelivered
		reply := strings.TrimSpace(e.SMTPReply)
		if i := strings.Index(reply, "("); i >= 0 && strings.HasSuffix(reply, ")") {
			reply = reply[i+1 : len(reply)-1]
		}
		ev.OriginalCode, ev.Diagnostic = bounce.Split(reply)
	case "bounce":
		hardness := "soft"
		ev.Type = providers.EventSoftBounce
		if e.HardBounce && !senderSide[e.ErrorRelatedTo] {
			hardness, ev.Type = "hard", providers.EventHardBounce
		}
		ev.OriginalType = joinNonEmpty(" · ", e.Event, hardness, e.ErrorRelatedTo, e.Error)
		ev.OriginalCode, ev.Diagnostic, ev.BounceClass = code, firstNonEmpty(rest, e.Comment), class
		if ev.BounceClass == "" {
			ev.BounceClass = bounce.Other
		}
	case "blocked":
		ev.Type = providers.EventRejected
		ev.OriginalType = joinNonEmpty(" · ", e.Event, e.ErrorRelatedTo, e.Error)
		ev.OriginalCode, ev.Diagnostic, ev.BounceClass = code, firstNonEmpty(rest, e.Comment), class
		if ev.BounceClass == "" {
			ev.BounceClass = bounce.Other
		}
	case "spam":
		ev.Type = providers.EventComplaint
		ev.OriginalType = joinNonEmpty(" · ", e.Event, e.Source)
		ev.Diagnostic = "Spam complaint via Mailjet"
	case "unsub":
		ev.Type = providers.EventUnsubscribe
	case "open":
		ev.Type, ev.UserAgent = providers.EventOpen, e.Agent
	case "click":
		ev.Type, ev.URL, ev.UserAgent = providers.EventClick, e.URL, e.Agent
	default:
		return ev, false
	}
	return ev, true
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
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
