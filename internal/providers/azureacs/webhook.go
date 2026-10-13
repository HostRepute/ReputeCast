package azureacs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/bounce"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

func asCallError(err error, target **providers.CallError) bool { return errors.As(err, target) }

// now is the clock, replaced in tests.
var now = time.Now

// gridEvent is an Event Grid event in the Event Grid schema or as a
// CloudEvent.
type gridEvent struct {
	ID        string          `json:"id"`
	EventType string          `json:"eventType"`
	Type      string          `json:"type"`
	EventTime string          `json:"eventTime"`
	Time      string          `json:"time"`
	Data      json.RawMessage `json:"data"`
}

func (e gridEvent) kind() string {
	if e.EventType != "" {
		return e.EventType
	}
	return e.Type
}

func decodeEvents(body []byte) ([]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var items []json.RawMessage
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return nil, err
		}
		return items, nil
	}
	if !json.Valid(trimmed) {
		return nil, errors.New("not JSON")
	}
	return []json.RawMessage{trimmed}, nil
}

// Handshake implements providers.WebhookHandshaker: Event Grid validates a
// new subscription by posting a SubscriptionValidationEvent and expects its
// validationCode back. The answer proves nothing about deliveries, which
// still need the secret, so it needs none itself.
func (p *Provider) Handshake(_ context.Context, r *http.Request, body []byte) (bool, []byte, error) {
	switch r.Header.Get("aeg-event-type") {
	case "SubscriptionDeletion":
		return true, nil, nil
	case "Notification":
		return false, nil, nil
	}
	items, err := decodeEvents(body)
	if err != nil {
		// Not a handshake; ParseWebhook reports the malformed body.
		return false, nil, nil //nolint:nilerr // the body is judged by ParseWebhook
	}
	for _, raw := range items {
		var e gridEvent
		if json.Unmarshal(raw, &e) != nil || e.kind() != "Microsoft.EventGrid.SubscriptionValidationEvent" {
			continue
		}
		var data struct {
			ValidationCode string `json:"validationCode"`
		}
		if err := json.Unmarshal(e.Data, &data); err != nil || data.ValidationCode == "" {
			return true, nil, fmt.Errorf("%w: a validation event without a code", providers.ErrBadSignature)
		}
		resp, _ := json.Marshal(map[string]string{"validationResponse": data.ValidationCode})
		return true, resp, nil
	}
	return false, nil, nil
}

// VerifyWebhook implements providers.WebhookVerifier. Event Grid doesn't
// sign deliveries. A subscription protected with Microsoft Entra ID sends a
// bearer token, checked against the webhook application (when
// event_grid_audience is set); otherwise the subscription sends the
// provider's webhook secret as a delivery header (X-ReputeCast-Secret, or
// Authorization: Bearer) or in the endpoint URL's query (secret=).
func (p *Provider) VerifyWebhook(ctx context.Context, r *http.Request, _ []byte) error {
	if p.s.EventGridAudience != "" {
		if token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok && strings.Count(token, ".") == 2 {
			return p.verifyEntraToken(ctx, strings.TrimSpace(token))
		}
	}
	got := strings.TrimSpace(r.Header.Get(SecretHeader))
	if got == "" {
		got, _ = strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	if got == "" && r.URL != nil {
		got = r.URL.Query().Get("secret")
	}
	if !webhookauth.Equal(strings.TrimSpace(got), p.cfg.WebhookSecret) {
		return providers.ErrBadSignature
	}
	return nil
}

type deliveryReport struct {
	Recipient             string `json:"recipient"`
	MessageID             string `json:"messageId"`
	Status                string `json:"status"`
	DeliveryStatusDetails struct {
		StatusMessage string `json:"statusMessage"`
		Host          string `json:"recipientMailServerHostName"`
	} `json:"deliveryStatusDetails"`
	DeliveryAttemptTimestamp string `json:"deliveryAttemptTimestamp"`
}

type engagementReport struct {
	Recipient           string `json:"recipient"`
	MessageID           string `json:"messageId"`
	UserActionTimestamp string `json:"userActionTimestamp"`
	EngagementContext   string `json:"engagementContext"`
	UserAgent           string `json:"userAgent"`
	EngagementType      string `json:"engagementType"`
}

// ParseWebhook implements providers.Provider for Event Grid deliveries (an
// array of events, or one CloudEvent).
func (p *Provider) ParseWebhook(ctx context.Context, r *http.Request, body []byte) ([]providers.Event, error) {
	if !providers.SignatureVerified(ctx) {
		if err := p.VerifyWebhook(ctx, r, body); err != nil {
			return nil, err
		}
	}
	items, err := decodeEvents(body)
	if err != nil {
		return nil, fmt.Errorf("azureacs webhook: %w", err)
	}
	out := []providers.Event{}
	for _, raw := range items {
		var e gridEvent
		if json.Unmarshal(raw, &e) != nil {
			continue
		}
		if ev, ok := normalize(e); ok {
			ev.Raw = append(json.RawMessage(nil), raw...)
			out = append(out, ev)
		}
	}
	return out, nil
}

func normalize(e gridEvent) (providers.Event, bool) {
	at := parseTime(e.EventTime, e.Time)
	switch e.kind() {
	case "Microsoft.Communication.EmailDeliveryReportReceived":
		var d deliveryReport
		if json.Unmarshal(e.Data, &d) != nil {
			return providers.Event{}, false
		}
		ev := providers.Event{ProviderMessageID: d.MessageID, Recipient: strings.ToLower(strings.TrimSpace(d.Recipient)),
			OriginalType: "EmailDeliveryReportReceived · " + d.Status, OccurredAt: parseTime(d.DeliveryAttemptTimestamp, "")}
		if ev.OccurredAt.IsZero() {
			ev.OccurredAt = at
		}
		if id, err := uuid.Parse(d.MessageID); err == nil {
			ev.MessageID = &id
		}
		msg := strings.TrimSpace(d.DeliveryStatusDetails.StatusMessage)
		code, rest := bounce.Split(msg)
		ev.OriginalCode = code
		ev.Diagnostic = rest
		if host := d.DeliveryStatusDetails.Host; host != "" {
			if ev.Diagnostic != "" {
				ev.Diagnostic += " · "
			}
			ev.Diagnostic += host
		}
		class := bounce.Classify(code, msg)
		switch strings.ToLower(d.Status) {
		case "delivered":
			ev.Type = providers.EventDelivered
		case "expanded":
			// A distribution group was expanded: not an outcome.
			return ev, false
		case "bounced":
			ev.Type, ev.BounceClass = providers.EventHardBounce, class
			switch class {
			case bounce.SpamBlock, bounce.Reputation, bounce.PolicyBlock:
				// Refused because of the sender, not the address.
				ev.Type = providers.EventSoftBounce
			}
		case "suppressed":
			// ACS didn't try: the address hard-bounced before.
			ev.Type, ev.BounceClass = providers.EventRejected, bounce.MailboxUnknown
		case "filteredspam", "quarantined":
			ev.Type, ev.BounceClass = providers.EventSoftBounce, bounce.SpamBlock
		default: // Failed and statuses added later
			ev.Type, ev.BounceClass = providers.EventSoftBounce, class
		}
		return ev, true
	case "Microsoft.Communication.EmailEngagementTrackingReportReceived":
		var d engagementReport
		if json.Unmarshal(e.Data, &d) != nil {
			return providers.Event{}, false
		}
		ev := providers.Event{ProviderMessageID: d.MessageID, Recipient: strings.ToLower(strings.TrimSpace(d.Recipient)),
			OriginalType: "EmailEngagementTrackingReportReceived · " + d.EngagementType, OccurredAt: parseTime(d.UserActionTimestamp, ""), UserAgent: d.UserAgent}
		if ev.OccurredAt.IsZero() {
			ev.OccurredAt = at
		}
		if id, err := uuid.Parse(d.MessageID); err == nil {
			ev.MessageID = &id
		}
		switch strings.ToLower(d.EngagementType) {
		case "view":
			ev.Type = providers.EventOpen
		case "click":
			ev.Type, ev.URL = providers.EventClick, d.EngagementContext
		default:
			return ev, false
		}
		return ev, true
	}
	return providers.Event{}, false
}

// parseTime reads Azure's timestamps (seven fractional digits, an offset);
// the second value is a fallback; zero when neither parses.
func parseTime(vs ...string) time.Time {
	for _, s := range vs {
		if t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(s)); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
