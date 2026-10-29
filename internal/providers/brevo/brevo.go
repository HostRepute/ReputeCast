// Package brevo is the Brevo (formerly Sendinblue) adapter, through the
// transactional email API v3.
//
// Terms (checked 2026-10-13): the Terms of Use
// (https://www.brevo.com/legal/termsofuse/) and Acceptable Use Policy
// (https://www.brevo.com/legal/antispampolicy/) allow marketing email to
// contacts with active, explicit, provable consent, with an unsubscribe link
// and no bought or scraped lists. Marketing isn't limited to Brevo's
// campaign tool: promotional mail through the API needs the unsubscribe link
// too (https://help.brevo.com/hc/en-us/articles/37113920427922), so bulk is
// allowed on every stream.
//
// Headers: the API only takes non-standard headers, and Brevo adds its own
// List-Unsubscribe to transactional mail with no API to change it
// (https://help.brevo.com/hc/en-us/articles/19100260472850). One-click
// unsubscribes therefore go to Brevo, which reports them through the
// unsubscribed webhook.
package brevo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/providers/flex"
	"github.com/hostrepute/reputecast/internal/providers/mimemsg"
	"github.com/hostrepute/reputecast/internal/version"
)

// Adapter is the registry key.
const Adapter = "brevo"

// DefaultBaseURL is Brevo's API.
const DefaultBaseURL = "https://api.brevo.com"

// CustomHeader carries ReputeCast's message id; Brevo echoes it on every
// webhook event.
const CustomHeader = "X-Mailin-custom"

// parallel is how many calls one Send makes at a time when it is handed
// more than one message.
const parallel = 8

// Settings are the non-secret connection settings.
type Settings struct {
	// MonthlyLimit is the plan's monthly allowance, entered at connect time.
	MonthlyLimit flex.Int `json:"monthly_limit,omitempty"`
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
		Name:    "Brevo",
		Kind:    "api",
		Capabilities: providers.Capabilities{
			Bulk:            providers.BulkAllowed,
			Webhooks:        "Registered",
			RemovesWebhook:  true,
			DomainAPI:       "full",
			SuppressionSync: true,
			BatchSize:       1,
			Sandbox:         "mode",
			QuotaDiscovery:  true,
			ReportsIPs:      true,
			// Brevo sets its own List-Unsubscribe (see Headers above).
			OneClick: providers.OneClickReplaced,
			// Pricing, DedicatedIPs and every other text people read (the
			// terms' note, the Adapters tab's words, the fields' labels and
			// hints) are in messages/en.json: WithMessages fills them.
			// Brevo tracks opens and clicks in transactional mail and has no
			// API or per-message switch to stop it (only anonymous tracking
			// in its settings), so Send Readiness can warn about it.
			RewritesLinks: true,
			AddsOpenPixel: true,
		},
		Version:    version.Version,
		Maintainer: "HostRepute",
		Terms:      []providers.Terms{{Bulk: providers.BulkAllowed, Streams: []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem}}},
		Fields: []providers.Field{
			{Key: "api_key", Secret: true, Required: true},
			{Key: "monthly_limit"},
		},
		New: New,
	}.WithMessages(msgs)
}

// Provider talks to one Brevo account.
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
			return nil, fmt.Errorf("brevo settings: %w", err)
		}
	}
	if len(cfg.Credentials) > 0 {
		if err := json.Unmarshal(cfg.Credentials, &p.c); err != nil {
			return nil, fmt.Errorf("brevo credentials: %w", err)
		}
	}
	base := p.s.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	p.api = apiclient.New(base, cfg.HTTPClient)
	p.api.Header.Set("api-key", strings.TrimSpace(p.c.APIKey))
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
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &e) == nil && (e.Code != "" || e.Message != "") {
		return e.Code, strings.TrimSpace(e.Message)
	}
	return "", strings.TrimSpace(string(body))
}

// classify adjusts apiclient's status mapping to Brevo: running out of
// credits or an account under validation stops the whole account, shown as
// an outage until it is solved.
func classify(err error) error {
	var ce *providers.CallError
	if !errors.As(err, &ce) {
		return err
	}
	switch {
	case ce.Status == http.StatusPaymentRequired, ce.Code == "not_enough_credits", ce.Code == "account_under_validation":
		ce.Auth, ce.Temporary = false, true
	}
	if ce.Code == "" && ce.Status != 0 {
		ce.Code = fmt.Sprintf("HTTP %d", ce.Status)
	}
	return ce
}

// duplicate reports Brevo refusing a request whose IdempotencyKey it saw in
// the last 30 minutes, which means an earlier attempt was accepted. Brevo
// answers duplicate_parameter, a code other duplicated parameters share, so
// the message must be about the key or a request already processed.
func duplicate(ce *providers.CallError) bool {
	msg := strings.ToLower(ce.Message)
	return ce.Code == "duplicate_parameter" && (strings.Contains(msg, "idempotency") || strings.Contains(msg, "already"))
}

type address struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type email struct {
	Sender      address           `json:"sender"`
	To          []address         `json:"to"`
	ReplyTo     *address          `json:"replyTo,omitempty"`
	Subject     string            `json:"subject"`
	HTMLContent string            `json:"htmlContent,omitempty"`
	TextContent string            `json:"textContent,omitempty"`
	Headers     map[string]string `json:"headers"`
	Tags        []string          `json:"tags,omitempty"`
}

// standard headers Brevo doesn't take; it sets its own List-Unsubscribe.
var standard = map[string]bool{"list-unsubscribe": true, "list-unsubscribe-post": true, "list-id": true, "from": true, "to": true, "cc": true,
	"bcc": true, "subject": true, "reply-to": true, "message-id": true, "date": true, "mime-version": true, "content-type": true,
	"content-transfer-encoding": true, "return-path": true, "sender": true}

func message(m providers.Message) email {
	e := email{Sender: address{Email: m.From.Address, Name: name(m.From.Name)}, To: []address{{Email: m.To.Address, Name: name(m.To.Name)}},
		Subject: m.Subject, HTMLContent: m.HTML, TextContent: m.Text,
		Headers: map[string]string{
			CustomHeader:          "rc_message:" + m.ID.String() + "|rc_stream:" + string(m.Stream),
			mimemsg.HeaderMessage: m.ID.String(),
			// Brevo refuses a second request with the same key for 30
			// minutes, so a retry after a timeout never sends twice.
			"IdempotencyKey": m.ID.String(),
		},
		Tags: []string{"reputecast", string(m.Stream)}}
	if m.ReplyTo != nil {
		e.ReplyTo = &address{Email: m.ReplyTo.Address, Name: name(m.ReplyTo.Name)}
	}
	for k, v := range m.Headers {
		if !standard[strings.ToLower(k)] {
			e.Headers[k] = v
		}
	}
	return e
}

// name keeps a display name within Brevo's 70 characters.
func name(s string) string {
	if r := []rune(s); len(r) > 70 {
		return strings.TrimSpace(string(r[:70]))
	}
	return s
}

// Send implements providers.Provider: one POST /v3/smtp/email per message
// (message versions can't carry their own headers).
func (p *Provider) Send(ctx context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	return apiclient.Fanout(ctx, msgs, parallel, p.sendOne)
}

func (p *Provider) sendOne(ctx context.Context, m providers.Message) (providers.SendResult, error) {
	var out struct {
		MessageID string `json:"messageId"`
	}
	err := p.api.Do(ctx, "POST /v3/smtp/email", http.MethodPost, "/v3/smtp/email", message(m), &out)
	if err != nil {
		err = classify(err)
		var ce *providers.CallError
		if errors.As(err, &ce) && duplicate(ce) {
			// Accepted by an earlier attempt; events find the message by
			// the rc_message in X-Mailin-custom.
			return providers.SendResult{MessageID: m.ID, ProviderMessageID: m.ID.String()}, nil
		}
		return apiclient.Failed(m, err)
	}
	id := strings.Trim(strings.TrimSpace(out.MessageID), "<>")
	if id == "" {
		id = m.ID.String()
	}
	return providers.SendResult{MessageID: m.ID, ProviderMessageID: id}, nil
}
