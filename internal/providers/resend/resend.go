// Package resend is the Resend adapter (resend.com). It sends through the
// batch email API (100 distinct messages per call, idempotent), receives
// webhooks signed with Standard Webhooks / Svix, registers them over the
// API, sets up domains, syncs the suppression list both ways, reads and
// turns off per-domain open and click tracking, and backfills message
// status.
//
// Terms (checked 2026-10-13): Resend's Acceptable Use Policy
// (https://resend.com/legal/acceptable-use) allows mail to recipients who
// "explicitly opted in", requires bulk senders to include an address,
// explain why they write and offer a frictionless unsubscribe, and lets
// Resend close accounts above 0.08% complaints or 4% bounces. Its pricing
// FAQ (https://resend.com/pricing) answers "Can I send my newsletter
// through the /emails API instead of Broadcasts?" with "Yes", provided the
// mail carries a visible unsubscribe link and List-Unsubscribe and
// List-Unsubscribe-Post headers, which ReputeCast always sets. Bulk
// marketing is therefore allowed on every stream.
package resend

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/providers/mimemsg"
	"github.com/hostrepute/reputecast/internal/version"
)

// Adapter is the registry key.
const Adapter = "resend"

// DefaultBaseURL is Resend's API.
const DefaultBaseURL = "https://api.resend.com"

// BatchSize is Resend's limit for POST /emails/batch.
const BatchSize = 100

// Tags Resend echoes back on webhook events.
const (
	TagMessage = "rc_message"
	TagStream  = "rc_stream"
)

// Settings are the non-secret connection settings.
type Settings struct {
	// Region is where domains ReputeCast adds send from (us-east-1,
	// eu-west-1, sa-east-1, ap-northeast-1).
	Region string `json:"region,omitempty"`
	// MonthlyLimit is the plan's monthly allowance, entered at connect time.
	MonthlyLimit int64 `json:"monthly_limit,omitempty"`
	// BaseURL overrides the API (tests).
	BaseURL string `json:"base_url,omitempty"`
}

// Credentials are write-only.
type Credentials struct {
	APIKey string `json:"api_key"`
	// WebhookSigningSecret is the whsec_ secret of a webhook created by hand
	// in Resend; webhooks ReputeCast registers are read from the API.
	WebhookSigningSecret string `json:"webhook_signing_secret,omitempty"`
}

func init() { providers.Register(Descriptor()) }

// Descriptor describes the adapter for the catalog.
func Descriptor() providers.Descriptor {
	return providers.Descriptor{
		Adapter: Adapter,
		Name:    "Resend",
		Kind:    "api",
		Capabilities: providers.Capabilities{
			Bulk:            providers.BulkAllowed,
			Webhooks:        "Registered",
			DomainAPI:       "full",
			SuppressionSync: true,
			BatchSize:       BatchSize,
			DedicatedIPs:    "Scale plan, $30 per month",
			Sandbox:         "test addresses",
			QuotaDiscovery:  true,
			Pricing:         "plan, free up to 3,000 per month, then from $20 per month",
		},
		Version:    version.Version,
		Maintainer: "HostRepute",
		Terms: []providers.Terms{{Bulk: providers.BulkAllowed,
			Note:    "Opted-in recipients only. Bulk mail needs a visible unsubscribe link and one-click List-Unsubscribe headers, which ReputeCast adds; Resend may close accounts above 0.08% complaints or 4% bounces.",
			Streams: []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem}}},
		Events:          "Webhooks",
		DomainSetup:     "API",
		SuppressionMode: "Two-way",
		Fields: []providers.Field{
			{Key: "api_key", Label: "API key", Secret: true, Required: true,
				Help: "Resend › API Keys. A Full access key lets ReputeCast register webhooks, set up domains and sync suppressions; a Sending access key can only send."},
			{Key: "region", Label: "Region", Options: []string{"us-east-1", "eu-west-1", "sa-east-1", "ap-northeast-1"}, Help: "Where sending domains ReputeCast adds are hosted."},
			{Key: "monthly_limit", Label: "Plan · emails per month", Help: "Used for the quota bar and cost per 1,000."},
			{Key: "webhook_signing_secret", Label: "Webhook signing secret", Secret: true,
				Help: "Only for a webhook you created yourself (Resend › Webhooks › Signing secret). ReputeCast reads it for the webhooks it registers."},
		},
		New: New,
	}
}

// Provider talks to one Resend team.
type Provider struct {
	cfg providers.Config
	s   Settings
	c   Credentials
	api *apiclient.Client
}

var _ providers.Provider = (*Provider)(nil)

// New builds the adapter from a connected provider's configuration.
func New(cfg providers.Config) (providers.Provider, error) {
	p := &Provider{cfg: cfg}
	if len(cfg.Settings) > 0 {
		if err := json.Unmarshal(cfg.Settings, &p.s); err != nil {
			return nil, fmt.Errorf("resend settings: %w", err)
		}
	}
	if len(cfg.Credentials) > 0 {
		if err := json.Unmarshal(cfg.Credentials, &p.c); err != nil {
			return nil, fmt.Errorf("resend credentials: %w", err)
		}
	}
	base := p.s.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	p.api = apiclient.New(base, cfg.HTTPClient)
	p.api.Header.Set("Authorization", "Bearer "+strings.TrimSpace(p.c.APIKey))
	// Resend refuses requests without a User-Agent.
	p.api.Header.Set("User-Agent", "ReputeCast/"+version.Version)
	p.api.Decode = decodeError
	return p, nil
}

// Adapter implements providers.Provider.
func (p *Provider) Adapter() string { return Adapter }

// Capabilities implements providers.Provider.
func (p *Provider) Capabilities() providers.Capabilities {
	c := Descriptor().Capabilities
	if p.s.MonthlyLimit > 0 {
		c.Pricing = "plan, " + count(p.s.MonthlyLimit) + " per month"
	}
	return c
}

// apiError is Resend's error body.
type apiError struct {
	StatusCode int    `json:"statusCode"`
	Name       string `json:"name"`
	Message    string `json:"message"`
}

func decodeError(_ int, body []byte) (string, string) {
	var e apiError
	if json.Unmarshal(body, &e) == nil && (e.Name != "" || e.Message != "") {
		return e.Name, e.Message
	}
	return "", strings.TrimSpace(string(body))
}

// classify adjusts apiclient's status mapping to Resend's error names: a
// 403 validation_error is about the message (unverified domain, testing
// sender) and quota errors are throttling, not rejected credentials.
func classify(err error) error {
	var ce *providers.CallError
	if !errors.As(err, &ce) {
		return err
	}
	switch ce.Code {
	case "validation_error", "invalid_parameter", "missing_required_field", "invalid_attachment", "invalid_idempotent_request":
		ce.Auth, ce.Temporary, ce.Throttled = false, false, false
	case "daily_quota_exceeded", "monthly_quota_exceeded", "rate_limit_exceeded", "concurrent_idempotent_requests":
		ce.Auth, ce.Temporary, ce.Throttled = false, true, true
	}
	return ce
}

// messageLevel reports whether a failed call concerns the message rather
// than the account or the service.
func messageLevel(ce *providers.CallError) bool {
	return !ce.Auth && !ce.Temporary && ce.Status >= 400 && ce.Status < 500
}

type tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type outEmail struct {
	From    string            `json:"from"`
	To      []string          `json:"to"`
	Subject string            `json:"subject"`
	HTML    string            `json:"html,omitempty"`
	Text    string            `json:"text,omitempty"`
	ReplyTo []string          `json:"reply_to,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Tags    []tag             `json:"tags,omitempty"`
}

func email(m providers.Message) outEmail {
	e := outEmail{From: mimemsg.Address(m.From), To: []string{m.To.Address}, Subject: m.Subject, HTML: m.HTML, Text: m.Text,
		Headers: map[string]string{mimemsg.HeaderMessage: m.ID.String()},
		Tags:    []tag{{Name: TagMessage, Value: m.ID.String()}, {Name: TagStream, Value: string(m.Stream)}}}
	if m.ReplyTo != nil {
		// Resend documents display names for from only.
		e.ReplyTo = []string{m.ReplyTo.Address}
	}
	for k, v := range m.Headers {
		e.Headers[k] = v
	}
	for k, v := range m.Tags {
		if len(e.Tags) < 50 && validTag(k) && validTag(v) && k != TagMessage && k != TagStream {
			e.Tags = append(e.Tags, tag{Name: k, Value: v})
		}
	}
	return e
}

// validTag reports whether s may be a Resend tag name or value.
func validTag(s string) bool {
	if s == "" || len(s) > 256 {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

// Send implements providers.Provider: one message through POST /emails,
// more through POST /emails/batch. Both carry an Idempotency-Key derived
// from the ReputeCast message ids, so a retry after a timeout never sends
// twice. Resend validates a batch as a whole: when one message is invalid,
// nothing is sent and the messages are sent one by one instead, so the
// invalid one fails alone.
func (p *Provider) Send(ctx context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	switch {
	case len(msgs) == 0:
		return []providers.SendResult{}, nil
	case len(msgs) > BatchSize:
		return nil, fmt.Errorf("resend: batch of %d exceeds %d", len(msgs), BatchSize)
	case len(msgs) == 1:
		res, err := p.sendOne(ctx, msgs[0])
		if err != nil {
			return nil, err
		}
		return []providers.SendResult{res}, nil
	}
	in := make([]outEmail, len(msgs))
	for i, m := range msgs {
		in[i] = email(m)
	}
	body, _ := json.Marshal(in)
	res, err := p.api.Exec(ctx, apiclient.Request{Op: "POST /emails/batch", Method: http.MethodPost, Path: "/emails/batch", ContentType: "application/json", Body: body,
		Header: http.Header{"Idempotency-Key": {batchKey(msgs)}, "X-Batch-Validation": {"strict"}}})
	if err != nil {
		err = classify(err)
		var ce *providers.CallError
		if errors.As(err, &ce) && messageLevel(ce) {
			// The resend.dev testing sender only reaches the account
			// owner: that refusal is per recipient, so it falls through to
			// sending one by one.
			if ce.Status == http.StatusForbidden && sameSender(msgs) && !strings.Contains(ce.Message, "testing emails") {
				// The sender itself is refused (unverified domain): every
				// message of the batch fails the same way.
				out := make([]providers.SendResult, len(msgs))
				for i, m := range msgs {
					out[i] = providers.SendResult{MessageID: m.ID, Err: &providers.SendError{Code: ce.Code, Message: ce.Message}}
				}
				return out, nil
			}
			return apiclient.Fanout(ctx, msgs, 2, p.sendOne)
		}
		return nil, err
	}
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &out); err != nil || len(out.Data) != len(msgs) {
		return nil, &providers.CallError{Op: "POST /emails/batch", Status: res.Status, Temporary: true,
			Message: fmt.Sprintf("Resend answered %d ids for %d messages", len(out.Data), len(msgs))}
	}
	results := make([]providers.SendResult, len(msgs))
	for i, m := range msgs {
		results[i] = providers.SendResult{MessageID: m.ID, ProviderMessageID: out.Data[i].ID}
	}
	return results, nil
}

// sendOne sends one message through POST /emails. Message-level refusals
// are returned on the result; account and service failures as the error.
func (p *Provider) sendOne(ctx context.Context, m providers.Message) (providers.SendResult, error) {
	body, _ := json.Marshal(email(m))
	res, err := p.api.Exec(ctx, apiclient.Request{Op: "POST /emails", Method: http.MethodPost, Path: "/emails", ContentType: "application/json", Body: body,
		Header: http.Header{"Idempotency-Key": {"rc-" + m.ID.String()}}})
	if err != nil {
		err = classify(err)
		var ce *providers.CallError
		if errors.As(err, &ce) && messageLevel(ce) {
			return providers.SendResult{MessageID: m.ID, Err: &providers.SendError{Code: ce.Code, Message: ce.Message}}, nil
		}
		return providers.SendResult{}, err
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(res.Body, &out); err != nil || out.ID == "" {
		return providers.SendResult{}, &providers.CallError{Op: "POST /emails", Status: res.Status, Temporary: true, Message: "Resend answered without an email id"}
	}
	return providers.SendResult{MessageID: m.ID, ProviderMessageID: out.ID}, nil
}

// batchKey is the idempotency key of a batch: the same messages in the
// same order give the same key.
func batchKey(msgs []providers.Message) string {
	h := sha256.New()
	for _, m := range msgs {
		h.Write(m.ID[:])
	}
	return "rc-batch-" + hex.EncodeToString(h.Sum(nil))
}

func sameSender(msgs []providers.Message) bool {
	for _, m := range msgs[1:] {
		if !strings.EqualFold(m.From.Address, msgs[0].From.Address) {
			return false
		}
	}
	return true
}

func count(n int64) string {
	s := fmt.Sprint(n)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}
