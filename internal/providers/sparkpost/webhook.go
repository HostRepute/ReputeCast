package sparkpost

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/bounce"
	"github.com/hostrepute/reputecast/internal/providers/webhookauth"
)

// VerifyWebhook implements providers.WebhookVerifier: SparkPost doesn't sign
// payloads, so webhooks are registered with HTTP basic auth, over the
// webhook's unguessable URL.
func (p *Provider) VerifyWebhook(_ context.Context, r *http.Request, _ []byte) error {
	return webhookauth.Basic(r, WebhookUser, p.cfg.WebhookSecret)
}

// event is the inner object of {"msys": {"message_event": {…}}}; every
// SparkPost event type has these fields or a subset.
type event struct {
	Type           string          `json:"type"`
	EventID        string          `json:"event_id"`
	MessageID      string          `json:"message_id"`
	TransmissionID string          `json:"transmission_id"`
	RcptTo         string          `json:"rcpt_to"`
	RawRcptTo      string          `json:"raw_rcpt_to"`
	Timestamp      json.RawMessage `json:"timestamp"`
	RawReason      string          `json:"raw_reason"`
	Reason         string          `json:"reason"`
	ErrorCode      string          `json:"error_code"`
	BounceClass    string          `json:"bounce_class"`
	FBType         string          `json:"fbtype"`
	RcptMeta       map[string]any  `json:"rcpt_meta"`
	TargetLinkURL  string          `json:"target_link_url"`
	UserAgent      string          `json:"user_agent"`
}

// ParseWebhook implements providers.Provider: the body is a JSON array of
// {"msys": {"<kind>_event": {…}}}; the empty test batch SparkPost posts when
// a webhook is created has no events.
func (p *Provider) ParseWebhook(ctx context.Context, r *http.Request, body []byte) ([]providers.Event, error) {
	if !providers.SignatureVerified(ctx) {
		if err := p.VerifyWebhook(ctx, r, body); err != nil {
			return nil, err
		}
	}
	var batch []struct {
		Msys map[string]json.RawMessage `json:"msys"`
	}
	if err := json.Unmarshal(body, &batch); err != nil {
		return nil, fmt.Errorf("sparkpost webhook: %w", err)
	}
	out := []providers.Event{}
	for _, item := range batch {
		for _, raw := range item.Msys {
			var e event
			if json.Unmarshal(raw, &e) != nil {
				continue
			}
			if ev, ok := normalize(e); ok {
				ev.Raw = append(json.RawMessage(nil), raw...)
				out = append(out, ev)
			}
		}
	}
	return out, nil
}

// hard bounce classes: invalid recipient (10) and no RCPT (30).
var hardClasses = map[string]bool{"10": true, "30": true}

// ignored bounce classes are not delivery outcomes: auto-replies (60),
// subscribe requests (80) and challenge-response (100).
var ignoredClasses = map[string]bool{"60": true, "80": true, "100": true}

func normalize(e event) (providers.Event, bool) {
	rcpt := e.RcptTo
	if rcpt == "" {
		rcpt = e.RawRcptTo
	}
	ev := providers.Event{ProviderMessageID: e.TransmissionID, Recipient: strings.ToLower(strings.TrimSpace(rcpt)),
		OriginalType: e.Type, OccurredAt: parseTime(e.Timestamp)}
	if s, ok := e.RcptMeta[MetaMessage].(string); ok {
		if id, err := uuid.Parse(s); err == nil {
			ev.MessageID = &id
		}
	}
	reply := firstNonEmpty(e.RawReason, e.Reason)
	code, rest := bounce.Split(reply)
	if code == "" {
		code = e.ErrorCode
	}
	switch e.Type {
	case "delivery":
		ev.Type = providers.EventDelivered
	case "delay":
		ev.Type = providers.EventDeferred
		ev.OriginalType = joinNonEmpty(" · ", e.Type, className(e.BounceClass))
		ev.OriginalCode, ev.Diagnostic = code, rest
		ev.BounceClass = classOf(e.BounceClass, code, reply)
	case "bounce", "out_of_band":
		if ignoredClasses[e.BounceClass] {
			return ev, false
		}
		ev.OriginalType = joinNonEmpty(" · ", e.Type, className(e.BounceClass))
		ev.OriginalCode, ev.Diagnostic = code, rest
		ev.BounceClass = classOf(e.BounceClass, code, reply)
		switch {
		case e.BounceClass == "90":
			// An unsubscribe request sent as a reply.
			ev.Type = providers.EventUnsubscribe
		case e.BounceClass == "25":
			// Refused by SparkPost's own policy, such as a suppressed address.
			ev.Type = providers.EventRejected
		case hardClasses[e.BounceClass]:
			ev.Type = providers.EventHardBounce
		default:
			// Soft, block (50–54) and undetermined classes: not the address.
			ev.Type = providers.EventSoftBounce
		}
	case "policy_rejection", "generation_failure", "generation_rejection":
		ev.Type = providers.EventRejected
		ev.OriginalCode, ev.Diagnostic = code, firstNonEmpty(rest, reply)
		ev.BounceClass = classOf(e.BounceClass, code, reply)
		if ev.BounceClass == "" {
			ev.BounceClass = bounce.Other
		}
	case "spam_complaint":
		if strings.EqualFold(e.FBType, "not-spam") {
			return ev, false
		}
		ev.Type = providers.EventComplaint
		ev.OriginalType = joinNonEmpty(" · ", e.Type, e.FBType)
		ev.Diagnostic = "Feedback report via SparkPost"
	case "list_unsubscribe", "link_unsubscribe":
		ev.Type = providers.EventUnsubscribe
	case "open", "initial_open":
		ev.Type, ev.UserAgent = providers.EventOpen, e.UserAgent
	case "click":
		ev.Type, ev.URL, ev.UserAgent = providers.EventClick, e.TargetLinkURL, e.UserAgent
	default:
		// injection, relay, A/B test and ingest events.
		return ev, false
	}
	return ev, true
}

// classNames are SparkPost's bounce classification codes.
var classNames = map[string]string{"1": "Undetermined", "10": "Invalid Recipient", "20": "Soft Bounce", "21": "DNS Failure", "22": "Mailbox Full",
	"23": "Too Large", "24": "Timeout", "25": "Admin Failure", "30": "Generic Bounce: No RCPT", "40": "Generic Bounce", "50": "Mail Block",
	"51": "Spam Block", "52": "Spam Content", "53": "Prohibited Attachment", "54": "Relaying Denied", "60": "Auto-Reply", "70": "Transient Failure",
	"80": "Subscribe", "90": "Unsubscribe", "100": "Challenge-Response"}

// className names a bounce class for the event log ("class 10 Invalid Recipient").
func className(class string) string {
	if class == "" {
		return ""
	}
	return strings.TrimSpace("class " + class + " " + classNames[class])
}

// classOf maps a SparkPost bounce class to the shared taxonomy.
func classOf(class, code, reply string) string {
	switch class {
	case "10", "30":
		return bounce.MailboxUnknown
	case "21":
		return bounce.DomainInvalid
	case "22":
		return bounce.MailboxFull
	case "20", "24", "70":
		return bounce.Temporary
	case "23", "25", "50", "54":
		return bounce.PolicyBlock
	case "51", "52", "53":
		return bounce.SpamBlock
	case "1", "40", "":
		if class == "" && code == "" && reply == "" {
			return ""
		}
		return bounce.Classify(code, reply)
	}
	return bounce.Classify(code, reply)
}

// parseTime reads Unix seconds (a string or a number) or an ISO time (the
// events API).
func parseTime(raw json.RawMessage) time.Time {
	s := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
		return time.Unix(n, 0).UTC()
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000-07:00"} {
		if t, err := time.Parse(layout, s); err == nil {
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

func joinNonEmpty(sep string, parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, strings.TrimSpace(p))
		}
	}
	return strings.Join(out, sep)
}
