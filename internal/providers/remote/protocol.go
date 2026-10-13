package remote

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/bounce"
)

// Operations of the adapter protocol (POST /v1/<operation>).
//
//nolint:gosec // G101: operation names, not credentials
const (
	OpHealth           = "health"
	OpVerify           = "verify"
	OpSend             = "send"
	OpWebhookVerify    = "webhooks/verify"
	OpWebhookParse     = "webhooks/parse"
	OpWebhookRegister  = "webhooks/register"
	OpDomainAdd        = "domains/add"
	OpDomainStatus     = "domains/status"
	OpQuota            = "quota"
	OpSuppressionsList = "suppressions/list"
	OpSuppressionsAdd  = "suppressions/add"
)

// Limits of the protocol.
const (
	// MaxSendRequest is the largest send request ReputeCast builds; larger
	// batches are split into several calls.
	MaxSendRequest = 24 << 20
	// MaxSuppressionPages bounds one pull of the provider's suppressions.
	MaxSuppressionPages = 100
	// SuppressionPushBatch is the most addresses per suppressions/add call.
	SuppressionPushBatch = 500
	// maxIdentifier is the longest id, address or record name accepted.
	maxIdentifier = 512
	// maxText is where longer texts are cut.
	maxText = 2000
	// maxHandshake is the largest handshake answer passed to a provider.
	maxHandshake = 16 << 10
)

// opSpec is what ReputeCast allows for one operation.
type opSpec struct {
	Timeout     time.Duration
	MaxResponse int64
	// Allowed reports whether the manifest declares the capability the
	// operation needs.
	Allowed func(Capabilities) bool
}

func always(Capabilities) bool { return true }

var ops = map[string]opSpec{
	OpHealth:           {5 * time.Second, 64 << 10, always},
	OpVerify:           {20 * time.Second, 64 << 10, always},
	OpSend:             {30 * time.Second, 4 << 20, always},
	OpWebhookVerify:    {5 * time.Second, 64 << 10, func(c Capabilities) bool { return c.Webhooks }},
	OpWebhookParse:     {15 * time.Second, 16 << 20, func(c Capabilities) bool { return c.Webhooks }},
	OpWebhookRegister:  {30 * time.Second, 64 << 10, func(c Capabilities) bool { return c.RegisterWebhooks }},
	OpDomainAdd:        {30 * time.Second, 1 << 20, func(c Capabilities) bool { return c.DomainSetup != "" }},
	OpDomainStatus:     {30 * time.Second, 1 << 20, func(c Capabilities) bool { return c.DomainSetup != "" }},
	OpQuota:            {20 * time.Second, 64 << 10, func(c Capabilities) bool { return c.QuotaDiscovery != "" }},
	OpSuppressionsList: {60 * time.Second, 8 << 20, Capabilities.Pull},
	OpSuppressionsAdd:  {60 * time.Second, 64 << 10, Capabilities.Push},
}

// Operations returns the operations a manifest lets ReputeCast call, in
// protocol order.
func Operations(c Capabilities) []string {
	out := []string{}
	for _, op := range []string{OpHealth, OpVerify, OpSend, OpWebhookVerify, OpWebhookParse, OpWebhookRegister, OpDomainAdd, OpDomainStatus, OpQuota, OpSuppressionsList, OpSuppressionsAdd} {
		if ops[op].Allowed(c) {
			out = append(out, op)
		}
	}
	return out
}

// Envelope is the body of every request ReputeCast sends.
type Envelope struct {
	Protocol       int         `json:"protocol"`
	Operation      string      `json:"operation"`
	Adapter        string      `json:"adapter"`
	IdempotencyKey string      `json:"idempotencyKey"`
	Deadline       string      `json:"deadline"`
	Connection     *Connection `json:"connection,omitempty"`
	Input          any         `json:"input"`
}

// Connection is the connected provider a call is for.
type Connection struct {
	ProviderID    string         `json:"providerId"`
	WorkspaceID   string         `json:"workspaceId"`
	Settings      map[string]any `json:"settings"`
	Credentials   map[string]any `json:"credentials"`
	WebhookSecret string         `json:"webhookSecret,omitempty"`
}

// Response is the body of every answer.
type Response struct {
	Protocol       int             `json:"protocol"`
	IdempotencyKey string          `json:"idempotencyKey"`
	Result         json.RawMessage `json:"result,omitempty"`
	Error          *WireError      `json:"error,omitempty"`
}

// WireError is a failed operation.
type WireError struct {
	Code         string  `json:"code"`
	Message      string  `json:"message"`
	Status       int     `json:"status,omitempty"`
	ProviderCode string  `json:"providerCode,omitempty"`
	RetryAfter   float64 `json:"retryAfter,omitempty"`
}

// Error codes.
const (
	CodeInvalidRequest      = "invalid_request"
	CodeUnauthenticated     = "unauthenticated"
	CodeTooLarge            = "too_large"
	CodeBadSignature        = "bad_signature"
	CodeProviderThrottled   = "provider_throttled"
	CodeProviderAuth        = "provider_auth"
	CodeProviderError       = "provider_error"
	CodeProviderUnavailable = "provider_unavailable"
	CodeNotSupported        = "not_supported"
	CodeInternal            = "internal"
)

// Address is a mailbox.
type Address struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

// WireMessage is one message to send.
type WireMessage struct {
	ID      string            `json:"id"`
	Stream  string            `json:"stream"`
	From    Address           `json:"from"`
	ReplyTo *Address          `json:"replyTo,omitempty"`
	To      Address           `json:"to"`
	Subject string            `json:"subject"`
	HTML    string            `json:"html,omitempty"`
	Text    string            `json:"text,omitempty"`
	Headers map[string]string `json:"headers"`
	Tags    map[string]string `json:"tags,omitempty"`
}

// MessageHeader carries the ReputeCast message id on every message.
const MessageHeader = "X-ReputeCast-Message"

func wireMessage(m providers.Message) WireMessage {
	w := WireMessage{ID: m.ID.String(), Stream: string(m.Stream), From: Address{m.From.Name, m.From.Address}, To: Address{m.To.Name, m.To.Address},
		Subject: m.Subject, HTML: m.HTML, Text: m.Text, Headers: map[string]string{}, Tags: m.Tags}
	if m.ReplyTo != nil {
		w.ReplyTo = &Address{m.ReplyTo.Name, m.ReplyTo.Address}
	}
	for k, v := range m.Headers {
		w.Headers[k] = v
	}
	w.Headers[MessageHeader] = m.ID.String()
	return w
}

// SendInput is the send operation's input.
type SendInput struct {
	Messages []WireMessage `json:"messages"`
}

// SendOutput is the send operation's result.
type SendOutput struct {
	Results []WireSendResult `json:"results"`
}

// WireSendResult is the outcome for one message.
type WireSendResult struct {
	MessageID         string         `json:"messageId"`
	ProviderMessageID string         `json:"providerMessageId,omitempty"`
	Error             *WireSendError `json:"error,omitempty"`
}

// WireSendError is a per-message refusal.
type WireSendError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Temporary bool   `json:"temporary"`
	Throttled bool   `json:"throttled"`
}

// WebhookRequest is a provider webhook forwarded to the adapter.
type WebhookRequest struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	// Body is the raw body (base64 in JSON).
	Body []byte `json:"body"`
}

// WebhookVerifyInput is the webhooks/verify input.
type WebhookVerifyInput struct {
	Request WebhookRequest `json:"request"`
}

// WebhookVerifyOutput is the webhooks/verify result.
type WebhookVerifyOutput struct {
	Handshake *struct {
		Body string `json:"body"`
	} `json:"handshake,omitempty"`
}

// WebhookParseInput is the webhooks/parse input.
type WebhookParseInput struct {
	Request  WebhookRequest `json:"request"`
	Verified bool           `json:"verified"`
}

// WebhookParseOutput is the webhooks/parse result.
type WebhookParseOutput struct {
	Events []WireEvent `json:"events"`
}

// WireEvent is a normalized provider event.
type WireEvent struct {
	Type              string          `json:"type"`
	Recipient         string          `json:"recipient"`
	ProviderMessageID string          `json:"providerMessageId,omitempty"`
	MessageID         string          `json:"messageId,omitempty"`
	OccurredAt        string          `json:"occurredAt"`
	OriginalType      string          `json:"originalType,omitempty"`
	OriginalCode      string          `json:"originalCode,omitempty"`
	Diagnostic        string          `json:"diagnostic,omitempty"`
	BounceClass       string          `json:"bounceClass,omitempty"`
	URL               string          `json:"url,omitempty"`
	UserAgent         string          `json:"userAgent,omitempty"`
	Raw               json.RawMessage `json:"raw"`
}

// RegisterInput is the webhooks/register input.
type RegisterInput struct {
	URL string `json:"url"`
}

// DomainInput is the domains/add and domains/status input.
type DomainInput struct {
	Domain string `json:"domain"`
}

// DomainOutput is their result.
type DomainOutput struct {
	Records []providers.DNSRecord `json:"records"`
}

// WireQuota is the quota result (and the verify result).
type WireQuota struct {
	Kind    string  `json:"kind,omitempty"`
	Max     int64   `json:"max,omitempty"`
	Used    int64   `json:"used,omitempty"`
	MaxRate float64 `json:"maxRate,omitempty"`
	Sandbox bool    `json:"sandbox,omitempty"`
	Account string  `json:"account,omitempty"`
}

// SuppressionsListInput is the suppressions/list input.
type SuppressionsListInput struct {
	Since  *string `json:"since"`
	Cursor *string `json:"cursor"`
}

// SuppressionsListOutput is its result.
type SuppressionsListOutput struct {
	Suppressions []WireSuppression `json:"suppressions"`
	NextCursor   *string           `json:"nextCursor"`
}

// WireSuppression is one suppressed address.
type WireSuppression struct {
	Email     string `json:"email"`
	Reason    string `json:"reason"`
	Code      string `json:"code,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

// SuppressionsAddInput is the suppressions/add input.
type SuppressionsAddInput struct {
	Emails []string `json:"emails"`
}

// HealthOutput is the health result.
type HealthOutput struct {
	Adapter      string          `json:"adapter"`
	Version      string          `json:"version"`
	Protocol     int             `json:"protocol"`
	SDK          string          `json:"sdk,omitempty"`
	Capabilities json.RawMessage `json:"capabilities,omitempty"`
}

// invalid is a response that doesn't follow the protocol.
type invalid string

func (e invalid) Error() string { return string(e) }

func invalidf(format string, a ...any) error { return invalid(fmt.Sprintf(format, a...)) }

// cut shortens a text to the protocol's limit.
func cut(s string) string {
	s = strings.ToValidUTF8(strings.TrimSpace(s), "")
	if utf8.RuneCountInString(s) <= maxText {
		return s
	}
	r := []rune(s)
	return string(r[:maxText]) + "…"
}

func identifier(field, s string) error {
	if len(s) > maxIdentifier {
		return invalidf("%s is longer than %d characters", field, maxIdentifier)
	}
	return nil
}

// sendResults validates a send result against the messages it answers.
func sendResults(msgs []providers.Message, out SendOutput) ([]providers.SendResult, error) {
	if len(out.Results) != len(msgs) {
		return nil, invalidf("send answered %d results for %d messages", len(out.Results), len(msgs))
	}
	res := make([]providers.SendResult, len(msgs))
	for i, r := range out.Results {
		if r.MessageID != msgs[i].ID.String() {
			return nil, invalidf("send result %d is for message %q, not %s (results must keep the order of the messages)", i, r.MessageID, msgs[i].ID)
		}
		if err := identifier("providerMessageId", r.ProviderMessageID); err != nil {
			return nil, err
		}
		res[i] = providers.SendResult{MessageID: msgs[i].ID, ProviderMessageID: r.ProviderMessageID}
		if r.Error != nil {
			res[i].ProviderMessageID = ""
			res[i].Err = &providers.SendError{Temporary: r.Error.Temporary || r.Error.Throttled, Throttled: r.Error.Throttled, Code: cut(r.Error.Code), Message: cut(r.Error.Message)}
			if res[i].Err.Code == "" && res[i].Err.Message == "" {
				res[i].Err.Message = "refused by the provider"
			}
			continue
		}
		if r.ProviderMessageID == "" {
			return nil, invalidf("send result %d has neither a providerMessageId nor an error", i)
		}
	}
	return res, nil
}

var eventTypes = map[string]providers.EventType{
	"delivered": providers.EventDelivered, "deferred": providers.EventDeferred, "hard_bounce": providers.EventHardBounce,
	"soft_bounce": providers.EventSoftBounce, "complaint": providers.EventComplaint, "rejected": providers.EventRejected,
	"open": providers.EventOpen, "click": providers.EventClick, "unsubscribe": providers.EventUnsubscribe,
}

var bounceClasses = map[string]bool{
	bounce.MailboxUnknown: true, bounce.MailboxFull: true, bounce.DomainInvalid: true, bounce.PolicyBlock: true,
	bounce.SpamBlock: true, bounce.Reputation: true, bounce.Temporary: true, bounce.Other: true,
}

// events validates and converts normalized events.
func events(out WebhookParseOutput) ([]providers.Event, error) {
	res := make([]providers.Event, 0, len(out.Events))
	for i, e := range out.Events {
		t, ok := eventTypes[e.Type]
		if !ok {
			return nil, invalidf("event %d has the unknown type %q", i, e.Type)
		}
		rcpt := strings.ToLower(strings.TrimSpace(e.Recipient))
		if rcpt == "" {
			return nil, invalidf("event %d has no recipient", i)
		}
		for f, v := range map[string]string{"recipient": rcpt, "providerMessageId": e.ProviderMessageID, "url": e.URL} {
			if err := identifier(fmt.Sprintf("event %d %s", i, f), v); err != nil {
				return nil, err
			}
		}
		at, err := time.Parse(time.RFC3339Nano, e.OccurredAt)
		if err != nil {
			return nil, invalidf("event %d occurredAt %q is not an RFC 3339 time", i, e.OccurredAt)
		}
		raw := json.RawMessage(strings.TrimSpace(string(e.Raw)))
		if len(raw) == 0 || string(raw) == "null" || !json.Valid(raw) {
			return nil, invalidf("event %d does not keep the provider's raw event", i)
		}
		ev := providers.Event{Type: t, ProviderMessageID: e.ProviderMessageID, Recipient: rcpt, OccurredAt: at.UTC(),
			OriginalType: cut(e.OriginalType), OriginalCode: cut(e.OriginalCode), Diagnostic: cut(e.Diagnostic),
			URL: e.URL, UserAgent: cut(e.UserAgent), Raw: raw}
		if ev.OriginalType == "" {
			ev.OriginalType = e.Type
		}
		if e.MessageID != "" {
			id, err := uuid.Parse(e.MessageID)
			if err != nil {
				return nil, invalidf("event %d messageId %q is not a ReputeCast message id", i, e.MessageID)
			}
			ev.MessageID = &id
		}
		switch {
		case e.BounceClass != "":
			if !bounceClasses[e.BounceClass] {
				return nil, invalidf("event %d has the unknown bounceClass %q", i, e.BounceClass)
			}
			ev.BounceClass = e.BounceClass
		case t == providers.EventHardBounce || t == providers.EventSoftBounce:
			ev.BounceClass = bounce.Classify(ev.OriginalCode, ev.Diagnostic)
		}
		res = append(res, ev)
	}
	return res, nil
}

var (
	recordPurposes = map[string]bool{"dkim": true, "spf": true, "return_path": true, "dmarc": true, "verification": true}
	recordTypes    = map[string]bool{"CNAME": true, "TXT": true, "MX": true}
	recordStatuses = map[string]bool{"": true, "pending": true, "verified": true, "wrong_value": true, "missing": true}
)

func records(out DomainOutput) ([]providers.DNSRecord, error) {
	res := make([]providers.DNSRecord, 0, len(out.Records))
	for i, r := range out.Records {
		r.Type = strings.ToUpper(r.Type)
		switch {
		case !recordPurposes[r.Purpose]:
			return nil, invalidf("record %d has the unknown purpose %q", i, r.Purpose)
		case !recordTypes[r.Type]:
			return nil, invalidf("record %d has the unknown type %q", i, r.Type)
		case !recordStatuses[r.Status]:
			return nil, invalidf("record %d has the unknown status %q", i, r.Status)
		case r.Name == "" || r.Value == "":
			return nil, invalidf("record %d needs a name and a value", i)
		case len(r.Name) > maxIdentifier || len(r.Value) > 4096:
			return nil, invalidf("record %d is too long", i)
		}
		res = append(res, r)
	}
	return res, nil
}

func quota(q WireQuota) (providers.Quota, error) {
	switch q.Kind {
	case "", "rolling_24h", "monthly_plan":
	default:
		return providers.Quota{}, invalidf("quota kind %q is not rolling_24h or monthly_plan", q.Kind)
	}
	if q.Max < 0 || q.Used < 0 || q.MaxRate < 0 {
		return providers.Quota{}, invalid("quota values can't be negative")
	}
	return providers.Quota{Kind: q.Kind, Max: q.Max, Used: q.Used, MaxRate: q.MaxRate, Sandbox: q.Sandbox, Account: cut(q.Account)}, nil
}

var suppressionReasons = map[string]bool{"hard_bounce": true, "complaint": true, "unsubscribe": true, "manual": true}

func suppressions(out SuppressionsListOutput) ([]providers.Suppression, error) {
	if len(out.Suppressions) > 1000 {
		return nil, invalidf("suppressions/list answered %d addresses; pages hold at most 1,000", len(out.Suppressions))
	}
	res := make([]providers.Suppression, 0, len(out.Suppressions))
	for i, s := range out.Suppressions {
		email := strings.ToLower(strings.TrimSpace(s.Email))
		if _, err := mail.ParseAddress(email); err != nil || len(email) > maxIdentifier {
			return nil, invalidf("suppression %d %q is not an email address", i, s.Email)
		}
		if !suppressionReasons[s.Reason] {
			return nil, invalidf("suppression %d has the unknown reason %q", i, s.Reason)
		}
		sp := providers.Suppression{Email: email, Reason: s.Reason, Code: cut(s.Code)}
		if s.CreatedAt != "" {
			t, err := time.Parse(time.RFC3339Nano, s.CreatedAt)
			if err != nil {
				return nil, invalidf("suppression %d createdAt %q is not an RFC 3339 time", i, s.CreatedAt)
			}
			sp.CreatedAt = t.UTC()
		}
		res = append(res, sp)
	}
	return res, nil
}
