// Package sendgrid is the Twilio SendGrid adapter (v3 API).
//
// Terms (checked 2026-10-13): Twilio's Acceptable Use Policy
// (https://www.twilio.com/en-us/legal/aup) and Email Policy
// (https://www.twilio.com/en-us/legal/service-country-specific-terms/email)
// allow marketing email with affirmative, provable consent, renewed after
// long inactivity; every marketing email needs a physical address, an
// unsubscribe link and a privacy-policy link, and bought or rented lists are
// forbidden. Mail Send takes marketing as well as transactional email
// (https://www.twilio.com/docs/sendgrid/for-developers/sending-email/v3-mail-send-faq),
// so bulk marketing is allowed on every stream.
//
// Batching: Mail Send's personalizations share one body, but ReputeCast
// renders every message for its recipient, so each message is its own call.
package sendgrid

import (
	"context"
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
const Adapter = "sendgrid"

// API hosts.
const (
	DefaultBaseURL = "https://api.sendgrid.com"
	EUBaseURL      = "https://api.eu.sendgrid.com"
)

// Custom args SendGrid echoes back on every event.
const (
	ArgMessage = "rc_message"
	ArgStream  = "rc_stream"
)

// parallel is how many Mail Send calls one Send call makes at a time when
// it is handed more than one message.
const parallel = 8

// Settings are the non-secret connection settings.
type Settings struct {
	// Region is "global" (default) or "eu" (EU subusers on Pro plans and up).
	Region string `json:"region,omitempty"`
	// IPPool sends through a named IP pool (dedicated IPs).
	IPPool string `json:"ip_pool,omitempty"`
	// WebhookVerificationKey is the signed Event Webhook's public key
	// (base64 DER). It is read from SendGrid for webhooks ReputeCast
	// registers; enter it only for a webhook set up by hand.
	WebhookVerificationKey string `json:"webhook_verification_key,omitempty"`
	// BaseURL overrides the API (tests).
	BaseURL string `json:"base_url,omitempty"`
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
		Name:    "SendGrid",
		Kind:    "api",
		Capabilities: providers.Capabilities{
			Bulk:            providers.BulkAllowed,
			Webhooks:        "Registered",
			RemovesWebhook:  true,
			DomainAPI:       "full",
			SuppressionSync: true,
			BatchSize:       1,
			DedicatedIPs:    "Pro plan and up, one included",
			Sandbox:         "mode",
			QuotaDiscovery:  true,
			Pricing:         "plan, free trial 100 per day, then from $19.95 per month",
		},
		Version:    version.Version,
		Maintainer: "HostRepute",
		Terms: []providers.Terms{{Bulk: providers.BulkAllowed,
			Note:    "Opted-in recipients with provable consent only, renewed after long inactivity. Every marketing email needs a postal address, an unsubscribe link and a privacy-policy link; no purchased lists.",
			Streams: []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem}}},
		Events:          "Event webhook",
		DomainSetup:     "API",
		SuppressionMode: "Two-way",
		Fields: []providers.Field{
			{Key: "api_key", Label: "API key", Secret: true, Required: true,
				Help: "SendGrid › Settings › API Keys. Full Access lets ReputeCast register the signed Event Webhook, authenticate domains and sync suppressions."},
			{Key: "region", Label: "Region", Options: []string{"global", "eu"}, Help: "EU only for an EU subuser (Pro plans and up)."},
			{Key: "ip_pool", Label: "IP pool", Help: "Optional. Sends through this pool of dedicated IPs."},
			{Key: "webhook_verification_key", Label: "Event Webhook verification key",
				Help: "Read from SendGrid for the webhook ReputeCast registers. Paste it only for a webhook you set up yourself (Mail Settings › Event Webhook › Signature verification)."},
		},
		New: New,
	}
}

// Provider talks to one SendGrid account (or subuser).
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
			return nil, fmt.Errorf("sendgrid settings: %w", err)
		}
	}
	if len(cfg.Credentials) > 0 {
		if err := json.Unmarshal(cfg.Credentials, &p.c); err != nil {
			return nil, fmt.Errorf("sendgrid credentials: %w", err)
		}
	}
	base := p.s.BaseURL
	switch {
	case base != "":
	case p.s.Region == "eu":
		base = EUBaseURL
	default:
		base = DefaultBaseURL
	}
	p.api = apiclient.New(base, cfg.HTTPClient)
	p.api.Header.Set("Authorization", "Bearer "+strings.TrimSpace(p.c.APIKey))
	p.api.Header.Set("User-Agent", "ReputeCast/"+version.Version)
	p.api.Decode = decodeError
	return p, nil
}

// Adapter implements providers.Provider.
func (p *Provider) Adapter() string { return Adapter }

// Capabilities implements providers.Provider.
func (p *Provider) Capabilities() providers.Capabilities { return Descriptor().Capabilities }

func decodeError(_ int, body []byte) (string, string) {
	var e struct {
		Errors []struct {
			Message string `json:"message"`
			Field   any    `json:"field"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &e) == nil && len(e.Errors) > 0 {
		msgs := make([]string, 0, len(e.Errors))
		for _, x := range e.Errors {
			m := strings.TrimSpace(x.Message)
			if f, ok := x.Field.(string); ok && f != "" && f != "null" {
				m = f + ": " + m
			}
			msgs = append(msgs, m)
		}
		return "", strings.Join(msgs, "; ")
	}
	return "", strings.TrimSpace(string(body))
}

// classify adjusts apiclient's status mapping to SendGrid: running out of
// credits answers 401 but is a quota, not a bad key; a 403 about the sender
// identity concerns the message; a temporary block is throttling.
func classify(err error) error {
	var ce *providers.CallError
	if !errors.As(err, &ce) {
		return err
	}
	msg := strings.ToLower(ce.Message)
	switch {
	case strings.Contains(msg, "maximum credits exceeded"):
		ce.Code = "credits_exceeded"
		ce.Auth, ce.Temporary, ce.Throttled = false, true, true
	case strings.Contains(msg, "temporarily blocked"):
		ce.Code = "temporarily_blocked"
		ce.Auth, ce.Temporary, ce.Throttled = false, true, true
	case ce.Status == http.StatusForbidden && (strings.Contains(msg, "sender identity") || strings.Contains(msg, "from address")):
		ce.Code = "sender_not_verified"
		ce.Auth = false
	case ce.Status == http.StatusUnavailableForLegalReasons:
		ce.Code = "data_residency"
	}
	if ce.Code == "" && ce.Status != 0 {
		ce.Code = fmt.Sprintf("HTTP %d", ce.Status)
	}
	return ce
}

type address struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type content struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type toggle struct {
	Enable bool `json:"enable"`
}

type mail struct {
	Personalizations []personalization `json:"personalizations"`
	From             address           `json:"from"`
	ReplyTo          *address          `json:"reply_to,omitempty"`
	Subject          string            `json:"subject"`
	Content          []content         `json:"content"`
	Headers          map[string]string `json:"headers,omitempty"`
	IPPoolName       string            `json:"ip_pool_name,omitempty"`
	TrackingSettings map[string]any    `json:"tracking_settings"`
}

type personalization struct {
	To         []address         `json:"to"`
	CustomArgs map[string]string `json:"custom_args"`
}

// reserved headers SendGrid sets itself.
var reserved = map[string]bool{"x-sg-id": true, "x-sg-eid": true, "received": true, "dkim-signature": true, "content-type": true,
	"content-transfer-encoding": true, "to": true, "from": true, "subject": true, "reply-to": true, "cc": true, "bcc": true}

func (p *Provider) mail(m providers.Message) mail {
	args := map[string]string{ArgMessage: m.ID.String(), ArgStream: string(m.Stream)}
	size := 0
	for k, v := range m.Tags {
		if size += len(k) + len(v); size > 4000 || k == ArgMessage || k == ArgStream {
			continue
		}
		args[k] = v
	}
	out := mail{
		Personalizations: []personalization{{To: []address{{Email: m.To.Address, Name: m.To.Name}}, CustomArgs: args}},
		From:             address{Email: m.From.Address, Name: m.From.Name},
		Subject:          m.Subject,
		Headers:          map[string]string{mimemsg.HeaderMessage: m.ID.String()},
		IPPoolName:       p.s.IPPool,
		// ReputeCast tracks opens and clicks and handles unsubscribes
		// itself: SendGrid's own tracking stays off on every message,
		// whatever the account settings say.
		TrackingSettings: map[string]any{"click_tracking": map[string]bool{"enable": false, "enable_text": false},
			"open_tracking": toggle{}, "subscription_tracking": toggle{}},
	}
	if m.ReplyTo != nil {
		out.ReplyTo = &address{Email: m.ReplyTo.Address, Name: m.ReplyTo.Name}
	}
	// Mail Send wants text/plain before text/html.
	if m.Text != "" {
		out.Content = append(out.Content, content{Type: "text/plain", Value: m.Text})
	}
	if m.HTML != "" {
		out.Content = append(out.Content, content{Type: "text/html", Value: m.HTML})
	}
	for k, v := range m.Headers {
		if !reserved[strings.ToLower(k)] {
			out.Headers[k] = v
		}
	}
	return out
}

// Send implements providers.Provider: one Mail Send call per message.
func (p *Provider) Send(ctx context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	return apiclient.Fanout(ctx, msgs, parallel, p.sendOne)
}

func (p *Provider) sendOne(ctx context.Context, m providers.Message) (providers.SendResult, error) {
	body, err := json.Marshal(p.mail(m))
	if err != nil {
		// A message that can't be encoded fails on its own.
		return providers.SendResult{MessageID: m.ID, Err: &providers.SendError{Code: "Render", Message: err.Error()}}, nil //nolint:nilerr // reported on the result

	}
	res, err := p.api.Exec(ctx, apiclient.Request{Op: "POST /v3/mail/send", Method: http.MethodPost, Path: "/v3/mail/send", ContentType: "application/json", Body: body})
	if err != nil {
		err = classify(err)
		return apiclient.Failed(m, err)
	}
	id := strings.TrimSpace(res.Header.Get("X-Message-Id"))
	if id == "" {
		// Accepted all the same (sending again would duplicate it); events
		// find the message through the rc_message custom arg.
		id = m.ID.String()
	}
	return providers.SendResult{MessageID: m.ID, ProviderMessageID: id}, nil
}
