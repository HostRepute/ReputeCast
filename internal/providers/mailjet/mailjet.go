// Package mailjet is the Mailjet adapter (Send API v3.1).
//
// Terms (checked 2026-10-13): the Acceptable Use Policy
// (https://www.mailjet.com/legal/sending-policy/) allows marketing email
// only with clear, explicit and provable consent; it forbids third-party
// lists, requires unsubscribe and privacy-policy links, and sets thresholds
// of 5% bounces, 0.08% spam complaints and 1.4% unsubscribes. Bulk marketing
// is therefore allowed on every stream.
//
// Mailjet's exclusion list only applies to its campaigns, not the Send API
// (https://dev.mailjet.com/docs/email-api/contact-management/manage-exclusion-list).
// ReputeCast enforces the shared suppression list before sending; the sync
// keeps Mailjet's own campaigns in line.
package mailjet

import (
	"context"
	"encoding/base64"
	"encoding/json"
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
const Adapter = "mailjet"

// DefaultBaseURL is Mailjet's API.
const DefaultBaseURL = "https://api.mailjet.com"

// BatchSize is Mailjet's limit of messages per Send API call.
const BatchSize = 50

// WebhookUser is the basic-auth user put in the event callback URL; the
// password is the provider's webhook secret.
const WebhookUser = "reputecast"

// Settings are the non-secret connection settings.
type Settings struct {
	// MonthlyLimit is the plan's monthly allowance, entered at connect time.
	MonthlyLimit flex.Int `json:"monthly_limit,omitempty"`
	// BaseURL overrides the API (tests, Mailjet's US infrastructure).
	BaseURL string `json:"base_url,omitempty"`
}

// Credentials are write-only: the API key and its secret key.
type Credentials struct {
	APIKey    string `json:"api_key"`
	SecretKey string `json:"secret_key"`
}

func init() { providers.Register(Descriptor()) }

// Descriptor describes the adapter for the catalog.
func Descriptor() providers.Descriptor {
	return providers.Descriptor{
		Adapter: Adapter,
		Name:    "Mailjet",
		Kind:    "api",
		Capabilities: providers.Capabilities{
			Bulk:            providers.BulkAllowed,
			Webhooks:        "Registered",
			RemovesWebhook:  true,
			DomainAPI:       "full",
			SuppressionSync: true,
			BatchSize:       BatchSize,
			DedicatedIPs:    "Premium plans from 100,000 per month, on request",
			Sandbox:         "mode",
			QuotaDiscovery:  true,
			Pricing:         "plan, free 6,000 per month (200 per day), then from $9 per month",
		},
		Version:    version.Version,
		Maintainer: "HostRepute",
		Terms: []providers.Terms{{Bulk: providers.BulkAllowed,
			Note:    "Clear, explicit and provable opt-in only; no third-party lists. Every marketing email needs an unsubscribe link and a privacy-policy link. Mailjet's limits: 5% bounces, 0.08% complaints.",
			Streams: []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem}}},
		Events:          "Webhooks",
		DomainSetup:     "API",
		SuppressionMode: "Two-way",
		Fields: []providers.Field{
			{Key: "api_key", Label: "API key", Secret: true, Required: true,
				Help: "Mailjet › Account settings › API key management. A key used only by ReputeCast keeps its event callbacks separate from other tools."},
			{Key: "secret_key", Label: "Secret key", Secret: true, Required: true, Help: "Shown next to the API key."},
			{Key: "monthly_limit", Label: "Plan · emails per month", Help: "Used for the quota bar and cost per 1,000."},
		},
		New: New,
	}
}

// Provider talks to one Mailjet API key.
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
			return nil, fmt.Errorf("mailjet settings: %w", err)
		}
	}
	if len(cfg.Credentials) > 0 {
		if err := json.Unmarshal(cfg.Credentials, &p.c); err != nil {
			return nil, fmt.Errorf("mailjet credentials: %w", err)
		}
	}
	base := p.s.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	p.api = apiclient.New(base, cfg.HTTPClient)
	auth := strings.TrimSpace(p.c.APIKey) + ":" + strings.TrimSpace(p.c.SecretKey)
	p.api.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	p.api.Header.Set("User-Agent", "ReputeCast/"+version.Version)
	p.api.Decode = decodeError
	return p, nil
}

// Adapter implements providers.Provider.
func (p *Provider) Adapter() string { return Adapter }

// Capabilities implements providers.Provider.
func (p *Provider) Capabilities() providers.Capabilities { return Descriptor().Capabilities }

// apiError is a Send API error (global or per message) and the REST API's
// {ErrorMessage, StatusCode} body.
type apiError struct {
	ErrorIdentifier string          `json:"ErrorIdentifier"`
	ErrorCode       string          `json:"ErrorCode"`
	StatusCode      int             `json:"StatusCode"`
	ErrorMessage    string          `json:"ErrorMessage"`
	ErrorInfo       string          `json:"ErrorInfo"`
	ErrorRelatedTo  json.RawMessage `json:"ErrorRelatedTo"`
}

func decodeError(_ int, body []byte) (string, string) {
	var e apiError
	if json.Unmarshal(body, &e) == nil && (e.ErrorMessage != "" || e.ErrorCode != "") {
		msg := strings.TrimSpace(e.ErrorMessage)
		if e.ErrorInfo != "" {
			msg = strings.TrimSpace(msg + " " + e.ErrorInfo)
		}
		return e.ErrorCode, msg
	}
	return "", strings.TrimSpace(string(body))
}

type address struct {
	Email string `json:"Email"`
	Name  string `json:"Name,omitempty"`
}

type outMessage struct {
	From        address           `json:"From"`
	To          []address         `json:"To"`
	ReplyTo     *address          `json:"ReplyTo,omitempty"`
	Subject     string            `json:"Subject"`
	TextPart    string            `json:"TextPart,omitempty"`
	HTMLPart    string            `json:"HTMLPart,omitempty"`
	Headers     map[string]string `json:"Headers,omitempty"`
	CustomID    string            `json:"CustomID"`
	TrackOpens  string            `json:"TrackOpens"`
	TrackClicks string            `json:"TrackClicks"`
	Priority    int               `json:"Priority"`
}

// forbidden headers Mailjet sets itself or reserves.
var forbidden = map[string]bool{}

func init() {
	for _, h := range []string{"From", "Sender", "Subject", "To", "Cc", "Bcc", "Return-Path", "Delivered-To", "DKIM-Signature", "DomainKey-Status",
		"Received-SPF", "Authentication-Results", "Received", "X-Mailjet-Prio", "X-Mailjet-Debug", "User-Agent", "X-Mailer", "X-MJ-CustomID",
		"X-MJ-EventPayload", "X-MJ-Vars", "X-MJ-TemplateErrorDeliver", "X-MJ-TemplateErrorReporting", "X-MJ-TemplateLanguage", "X-Mailjet-TrackOpen",
		"X-Mailjet-TrackClick", "X-MJ-TemplateID", "X-MJ-WorkflowID", "X-Feedback-Id", "X-Mailjet-Segmentation", "List-Id", "X-MJ-MID",
		"X-MJ-ErrorMessage", "Date", "X-CSA-Complaints", "Message-Id", "X-Mailjet-Campaign", "X-MJ-StatisticsContactsListID", "Reply-To",
		"Content-Type", "Content-Transfer-Encoding", "MIME-Version"} {
		forbidden[strings.ToLower(h)] = true
	}
}

func message(m providers.Message) outMessage {
	out := outMessage{From: address{Email: m.From.Address, Name: m.From.Name}, To: []address{{Email: m.To.Address, Name: m.To.Name}},
		Subject: m.Subject, TextPart: m.Text, HTMLPart: m.HTML, CustomID: m.ID.String(),
		// ReputeCast tracks opens and clicks itself.
		TrackOpens: "disabled", TrackClicks: "disabled",
		Headers: map[string]string{mimemsg.HeaderMessage: m.ID.String()}, Priority: 2}
	if m.Stream == providers.StreamBroadcast {
		// Mailjet's advice: 0 for marketing, 2 for transactional mail.
		out.Priority = 0
	}
	if m.ReplyTo != nil {
		out.ReplyTo = &address{Email: m.ReplyTo.Address, Name: m.ReplyTo.Name}
	}
	for k, v := range m.Headers {
		if !forbidden[strings.ToLower(k)] {
			out.Headers[k] = v
		}
	}
	return out
}

type sendResponse struct {
	Messages []struct {
		Status string `json:"Status"`
		To     []struct {
			Email     string      `json:"Email"`
			MessageID json.Number `json:"MessageID"`
		} `json:"To"`
		Errors []apiError `json:"Errors"`
	} `json:"Messages"`
}

// Send implements providers.Provider with POST /v3.1/send: up to 50
// messages, each with its own result.
func (p *Provider) Send(ctx context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	if len(msgs) == 0 {
		return []providers.SendResult{}, nil
	}
	if len(msgs) > BatchSize {
		return nil, fmt.Errorf("mailjet: batch of %d exceeds %d", len(msgs), BatchSize)
	}
	in := struct {
		Messages             []outMessage `json:"Messages"`
		AdvanceErrorHandling bool         `json:"AdvanceErrorHandling"`
	}{AdvanceErrorHandling: true}
	for _, m := range msgs {
		in.Messages = append(in.Messages, message(m))
	}
	body, _ := json.Marshal(in)
	res, err := p.api.Exec(ctx, apiclient.Request{Op: "POST /v3.1/send", Method: http.MethodPost, Path: "/v3.1/send", ContentType: "application/json", Body: body,
		Accept: func(status int, body []byte) bool {
			// Per-message results ride on a 400 or 403 when some failed;
			// any other answer is about the whole request.
			if status != http.StatusBadRequest && status != http.StatusForbidden {
				return false
			}
			var r sendResponse
			return json.Unmarshal(body, &r) == nil && len(r.Messages) == len(msgs)
		}})
	if err != nil {
		if se := apiclient.MessageError(err); se != nil {
			// A request-level error (malformed payload): every message fails
			// the same way.
			out := make([]providers.SendResult, len(msgs))
			for i, m := range msgs {
				out[i] = providers.SendResult{MessageID: m.ID, Err: se}
			}
			return out, nil
		}
		return nil, err
	}
	results := make([]providers.SendResult, len(msgs))
	var out sendResponse
	if err := json.Unmarshal(res.Body, &out); err != nil || len(out.Messages) != len(msgs) {
		// Accepted (a 2xx), but the answer can't be matched to the
		// messages: they count as sent, since sending them again could
		// deliver them twice, and their events find them by CustomID.
		for i, m := range msgs {
			results[i] = providers.SendResult{MessageID: m.ID, ProviderMessageID: m.ID.String()}
		}
		return results, nil
	}
	for i, m := range msgs {
		r := out.Messages[i]
		results[i] = providers.SendResult{MessageID: m.ID}
		if r.Status == "success" && len(r.To) > 0 {
			id := r.To[0].MessageID.String()
			if id == "" || id == "0" {
				// Sandbox mode answers without ids.
				id = m.ID.String()
			}
			results[i].ProviderMessageID = id
			continue
		}
		se := &providers.SendError{Code: "error", Message: "Mailjet refused the message"}
		if len(r.Errors) > 0 {
			e := r.Errors[0]
			se.Code, se.Message = e.ErrorCode, strings.TrimSpace(e.ErrorMessage)
			if e.StatusCode >= 500 || e.StatusCode == http.StatusTooManyRequests {
				se.Temporary, se.Throttled = true, e.StatusCode == http.StatusTooManyRequests
			}
		}
		results[i].Err = se
	}
	return results, nil
}
