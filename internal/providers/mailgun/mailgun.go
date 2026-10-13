// Package mailgun is the Mailgun adapter (US and EU regions). It sends one
// personalized message per call with Mailgun's open and click tracking off,
// receives webhooks signed with the account's HTTP webhook signing key
// (HMAC-SHA256 over timestamp and token), registers them per domain, sets
// up and verifies the sending domain, syncs suppressions both ways, reads
// the account's custom sending limit, usage and reputation, and backfills
// message status from the events API.
//
// Terms (checked 2026-10-13): Mailgun's Acceptable Use Policy
// (https://www.mailgun.com/legal/aup/) allows marketing email to recipients
// who gave "clear, explicit and provable consent", ideally through a
// confirmed opt-in; it forbids bought, rented or scraped lists, requires an
// unsubscribe link in every marketing email and a privacy-policy link, and
// sets thresholds of 5% bounces, 0.08% spam complaints and 1.4%
// unsubscribes. Bulk marketing is therefore allowed on every stream.
//
// Batching: Mailgun's batch sending (up to 1,000 recipients per call) shares
// one body between recipients through recipient variables. ReputeCast
// renders every message for its recipient, so each message is its own call
// and the adapter declares a batch size of 1.
package mailgun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"mime/multipart"
	"net/http"
	"slices"
	"strings"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/providers/mimemsg"
	"github.com/hostrepute/reputecast/internal/version"
)

// Adapter is the registry key.
const Adapter = "mailgun"

// API hosts per region.
const (
	USBaseURL = "https://api.mailgun.net"
	EUBaseURL = "https://api.eu.mailgun.net"
)

// Custom variables Mailgun echoes back on events (user-variables).
const (
	VarMessage = "rc_message"
	VarStream  = "rc_stream"
)

// parallel is how many calls one Send makes at a time when it is handed
// more than one message.
const parallel = 8

// Settings are the non-secret connection settings.
type Settings struct {
	// Domain is the Mailgun sending domain (mg.acme.io).
	Domain string `json:"domain"`
	// Region is "us" (default) or "eu".
	Region string `json:"region,omitempty"`
	// MonthlyLimit is the plan's monthly allowance, used when the account
	// has no custom sending limit.
	MonthlyLimit int64 `json:"monthly_limit,omitempty"`
	// BaseURL overrides the API (tests).
	BaseURL string `json:"base_url,omitempty"`
}

// Credentials are write-only.
type Credentials struct {
	APIKey string `json:"api_key"`
	// WebhookSigningKey is the account's HTTP webhook signing key; when
	// empty it is read from the API.
	WebhookSigningKey string `json:"webhook_signing_key,omitempty"`
}

func init() { providers.Register(Descriptor()) }

// Descriptor describes the adapter for the catalog.
func Descriptor() providers.Descriptor {
	return providers.Descriptor{
		Adapter: Adapter,
		Name:    "Mailgun",
		Kind:    "api",
		Capabilities: providers.Capabilities{
			Bulk:            providers.BulkAllowed,
			Webhooks:        "Registered",
			DomainAPI:       "full",
			SuppressionSync: true,
			BatchSize:       1,
			DedicatedIPs:    "Scale plan, one included; $59 per extra IP per month",
			Sandbox:         "detection",
			QuotaDiscovery:  true,
			Pricing:         "plan, free 100 per day, then from $15 per month for 10,000",
		},
		Version:    version.Version,
		Maintainer: "HostRepute",
		Terms: []providers.Terms{{Bulk: providers.BulkAllowed,
			Note:    "Explicit, provable opt-in only; no bought, rented or scraped lists. Every marketing email needs an unsubscribe link and a privacy-policy link. Mailgun's limits: 5% bounces, 0.08% complaints.",
			Streams: []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem}}},
		Events:          "Webhooks",
		DomainSetup:     "API",
		SuppressionMode: "Two-way",
		Fields: []providers.Field{
			{Key: "api_key", Label: "Private API key", Secret: true, Required: true,
				Help: "Mailgun › API Security. Use a Private API key (or a Developer key), not a Domain Sending key or the HTTP webhook signing key."},
			{Key: "region", Label: "Region", Required: true, Options: []string{"us", "eu"}, Help: "US · api.mailgun.net or EU · api.eu.mailgun.net, where the domain was created."},
			{Key: "domain", Label: "Sending domain", Required: true, Help: "The Mailgun domain mail is sent from, e.g. mg.acme.io. ReputeCast can add it for you."},
			{Key: "monthly_limit", Label: "Plan · emails per month", Help: "Used for the quota bar when the account has no custom sending limit."},
			{Key: "webhook_signing_key", Label: "HTTP webhook signing key", Secret: true,
				Help: "Optional. ReputeCast reads it from Mailgun; enter it only if your key can't (API Security › HTTP webhook signing key)."},
		},
		New: New,
	}
}

// Provider talks to one Mailgun domain.
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
			return nil, fmt.Errorf("mailgun settings: %w", err)
		}
	}
	if len(cfg.Credentials) > 0 {
		if err := json.Unmarshal(cfg.Credentials, &p.c); err != nil {
			return nil, fmt.Errorf("mailgun credentials: %w", err)
		}
	}
	p.s.Domain = strings.ToLower(strings.TrimSpace(p.s.Domain))
	base := p.s.BaseURL
	switch {
	case base != "":
	case strings.EqualFold(p.s.Region, "eu"):
		base = EUBaseURL
	default:
		base = USBaseURL
	}
	p.api = apiclient.New(base, cfg.HTTPClient)
	p.api.Header.Set("Authorization", "Basic "+basic("api", strings.TrimSpace(p.c.APIKey)))
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
		Message     string `json:"message"`
		Reason      string `json:"Reason"`
		Description string `json:"Description"`
	}
	if json.Unmarshal(body, &e) == nil {
		for _, m := range []string{e.Message, e.Reason, e.Description} {
			if strings.TrimSpace(m) != "" {
				return "", strings.TrimSpace(m)
			}
		}
	}
	var s string
	if json.Unmarshal(body, &s) == nil {
		return "", s
	}
	return "", strings.TrimSpace(string(body))
}

// classify adjusts apiclient's status mapping to Mailgun: an unknown domain
// is a settings problem (shown like rejected credentials), a payment problem
// stops the account like an outage, and refusals about one recipient
// (sandbox authorized recipients) concern the message.
func (p *Provider) classify(err error) error {
	var ce *providers.CallError
	if !errors.As(err, &ce) {
		return err
	}
	msg := strings.ToLower(ce.Message)
	switch {
	case ce.Status == http.StatusNotFound && strings.Contains(msg, "domain"):
		ce.Code, ce.Auth = "domain_not_found", true
		ce.Message = "Mailgun doesn't know the domain " + p.s.Domain + " in this region. Check the domain and the region."
	case ce.Status == http.StatusPaymentRequired:
		ce.Code, ce.Temporary = "payment_required", true
	case ce.Status == http.StatusForbidden && (strings.Contains(msg, "authorized recipients") || strings.Contains(msg, "sandbox")):
		ce.Code, ce.Auth = "unauthorized_recipient", false
	}
	if ce.Code == "" && ce.Status != 0 {
		ce.Code = fmt.Sprintf("HTTP %d", ce.Status)
	}
	return ce
}

func messageLevel(ce *providers.CallError) bool {
	return !ce.Auth && !ce.Temporary && ce.Status >= 400 && ce.Status < 500
}

// form encodes values as multipart/form-data, which every Mailgun form
// endpoint accepts; repeated keys keep their order.
func form(fields [][2]string) (string, []byte) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, f := range fields {
		_ = w.WriteField(f[0], f[1])
	}
	_ = w.Close()
	return w.FormDataContentType(), buf.Bytes()
}

// reserved headers are set from the message's own fields.
var reserved = map[string]bool{"from": true, "to": true, "cc": true, "bcc": true, "subject": true, "reply-to": true,
	"message-id": true, "date": true, "mime-version": true, "content-type": true, "content-transfer-encoding": true}

func (p *Provider) fields(m providers.Message) [][2]string {
	f := [][2]string{{"from", mimemsg.Address(m.From)}, {"to", mimemsg.Address(m.To)}, {"subject", m.Subject}}
	if m.Text != "" {
		f = append(f, [2]string{"text", m.Text})
	}
	if m.HTML != "" {
		f = append(f, [2]string{"html", m.HTML})
	}
	if m.ReplyTo != nil {
		f = append(f, [2]string{"h:Reply-To", mimemsg.Address(*m.ReplyTo)})
	}
	f = append(f, [2]string{"h:" + mimemsg.HeaderMessage, m.ID.String()})
	for _, k := range slices.Sorted(maps.Keys(m.Headers)) {
		if !reserved[strings.ToLower(k)] && validHeaderName(k) && !strings.EqualFold(k, mimemsg.HeaderMessage) {
			f = append(f, [2]string{"h:" + k, strings.NewReplacer("\r", " ", "\n", " ").Replace(m.Headers[k])})
		}
	}
	f = append(f, [2]string{"v:" + VarMessage, m.ID.String()}, [2]string{"v:" + VarStream, string(m.Stream)})
	size := 0
	for _, k := range slices.Sorted(maps.Keys(m.Tags)) {
		v := m.Tags[k]
		if size += len(k) + len(v); size > 4000 || k == VarMessage || k == VarStream || !validHeaderName(k) {
			continue
		}
		f = append(f, [2]string{"v:" + k, v})
	}
	// ReputeCast tracks opens and clicks itself.
	return append(f, [2]string{"o:tracking", "no"}, [2]string{"o:tracking-clicks", "no"}, [2]string{"o:tracking-opens", "no"})
}

// Send implements providers.Provider: one POST /v3/{domain}/messages per
// message.
func (p *Provider) Send(ctx context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	return apiclient.Fanout(ctx, msgs, parallel, p.sendOne)
}

func (p *Provider) sendOne(ctx context.Context, m providers.Message) (providers.SendResult, error) {
	ct, body := form(p.fields(m))
	res, err := p.api.Exec(ctx, apiclient.Request{Op: "POST /v3/{domain}/messages", Method: http.MethodPost,
		Path: apiclient.Path("/v3/%s/messages", p.s.Domain), ContentType: ct, Body: body})
	if err != nil {
		err = p.classify(err)
		var ce *providers.CallError
		if errors.As(err, &ce) && messageLevel(ce) {
			return providers.SendResult{MessageID: m.ID, Err: &providers.SendError{Code: ce.Code, Message: ce.Message}}, nil
		}
		return providers.SendResult{}, err
	}
	var out struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(res.Body, &out)
	id := strings.Trim(strings.TrimSpace(out.ID), "<>")
	if id == "" {
		// Queued all the same (sending again would duplicate it); events
		// find the message through the rc_message variable.
		id = m.ID.String()
	}
	return providers.SendResult{MessageID: m.ID, ProviderMessageID: id}, nil
}

func validHeaderName(k string) bool {
	if k == "" {
		return false
	}
	for _, r := range k {
		if r <= 32 || r >= 127 || r == ':' {
			return false
		}
	}
	return true
}
