// Package sparkpost is the SparkPost adapter (US and EU). SparkPost is now
// part of Bird and new projects are steered to Bird Email, a different API;
// this adapter connects existing SparkPost accounts, whose API
// (api.sparkpost.com, api.eu.sparkpost.com) is still operated and
// documented (checked 2026-10-13, https://bird.com/products/email/sparkpost).
//
// It sends one transmission per personalized message with an Idempotency-Key
// (a retry after a timeout or a 504 never sends twice), receives event
// webhooks authenticated with HTTP basic auth (SparkPost doesn't sign
// payloads), registers them, sets up sending and bounce domains, syncs the
// suppression list both ways, reads monthly usage and deliverability
// metrics, and backfills message status from the events API.
//
// Terms (checked 2026-10-13): SparkPost accounts are governed by Bird's
// terms since 22 December 2024 (https://bird.com/legal/legal-terms-updates);
// Bird's Acceptable Use Policy (https://bird.com/en-us/legal/acceptable-use-policy)
// requires "permission-based marketing lists", opt-in or another valid
// consent from each recipient and a working unsubscribe mechanism, and
// forbids unsolicited bulk messages and purchased or rented lists, as the
// archived SparkPost Messaging Policy did
// (https://bird.com/legal/sparkpost-messaging-policy). Bulk marketing to
// opted-in lists is therefore allowed on every stream.
package sparkpost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/providers/mimemsg"
	"github.com/hostrepute/reputecast/internal/version"
)

// Adapter is the registry key.
const Adapter = "sparkpost"

// API hosts per region.
const (
	USBaseURL = "https://api.sparkpost.com"
	EUBaseURL = "https://api.eu.sparkpost.com"
)

// Recipient metadata SparkPost echoes back on events (rcpt_meta).
const (
	MetaMessage = "rc_message"
	MetaStream  = "rc_stream"
)

// WebhookUser is the basic-auth user ReputeCast registers on webhooks; the
// password is the provider's webhook secret.
const WebhookUser = "reputecast"

// parallel is how many transmissions one Send makes at a time when it is
// handed more than one message (SparkPost advises at most 10 connections).
const parallel = 6

// Settings are the non-secret connection settings.
type Settings struct {
	// Region is "us" (default) or "eu".
	Region string `json:"region,omitempty"`
	// IPPool sends through a named IP pool (dedicated IPs).
	IPPool string `json:"ip_pool,omitempty"`
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
		Name:    "SparkPost",
		Kind:    "api",
		Capabilities: providers.Capabilities{
			Bulk:            providers.BulkAllowed,
			Webhooks:        "Registered",
			DomainAPI:       "full",
			SuppressionSync: true,
			BatchSize:       1,
			DedicatedIPs:    "paid plans, about $20 per IP per month",
			Sandbox:         "mode",
			QuotaDiscovery:  true,
			Pricing:         "plan, set by your SparkPost contract",
		},
		Version:    version.Version,
		Maintainer: "HostRepute",
		Terms: []providers.Terms{{Bulk: providers.BulkAllowed,
			Note:    "Permission-based lists only, with a working unsubscribe; no purchased or rented lists. Existing SparkPost accounts; new Bird accounts use Bird Email, a different API.",
			Streams: []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem}}},
		Events:          "Webhooks",
		DomainSetup:     "API",
		SuppressionMode: "Two-way",
		Fields: []providers.Field{
			{Key: "api_key", Label: "API key", Secret: true, Required: true,
				Help: "SparkPost › Configuration › API Keys, with Transmissions, Webhooks, Sending Domains, Suppression Lists, Account, Metrics and Message Events permissions."},
			{Key: "region", Label: "Region", Required: true, Options: []string{"us", "eu"}, Help: "US (app.sparkpost.com) or EU (app.eu.sparkpost.com) account."},
			{Key: "ip_pool", Label: "IP pool", Help: "Optional. Sends through this pool of dedicated IPs."},
		},
		New: New,
	}
}

// Provider talks to one SparkPost account.
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
			return nil, fmt.Errorf("sparkpost settings: %w", err)
		}
	}
	if len(cfg.Credentials) > 0 {
		if err := json.Unmarshal(cfg.Credentials, &p.c); err != nil {
			return nil, fmt.Errorf("sparkpost credentials: %w", err)
		}
	}
	base := p.s.BaseURL
	switch {
	case base != "":
	case strings.EqualFold(p.s.Region, "eu"):
		base = EUBaseURL
	default:
		base = USBaseURL
	}
	p.api = apiclient.New(base, cfg.HTTPClient)
	p.api.Header.Set("Authorization", strings.TrimSpace(p.c.APIKey))
	p.api.Header.Set("User-Agent", "ReputeCast/"+version.Version)
	p.api.Decode = decodeError
	return p, nil
}

// Adapter implements providers.Provider.
func (p *Provider) Adapter() string { return Adapter }

// Capabilities implements providers.Provider.
func (p *Provider) Capabilities() providers.Capabilities { return Descriptor().Capabilities }

// apiErrors is SparkPost's error body; code is a string or a number.
type apiErrors struct {
	Errors []struct {
		Message     string          `json:"message"`
		Description string          `json:"description"`
		Code        json.RawMessage `json:"code"`
	} `json:"errors"`
}

func decodeError(_ int, body []byte) (string, string) {
	var e apiErrors
	if json.Unmarshal(body, &e) == nil && len(e.Errors) > 0 {
		first := e.Errors[0]
		msg := strings.TrimSpace(first.Message)
		if first.Description != "" {
			msg += ": " + strings.TrimSpace(first.Description)
		}
		return strings.Trim(strings.TrimSpace(string(first.Code)), `"`), msg
	}
	return "", strings.TrimSpace(string(body))
}

// classify adjusts apiclient's status mapping: 420 is SparkPost's
// sending-limit status (nothing was sent), which is throttling.
func classify(err error) error {
	var ce *providers.CallError
	if !errors.As(err, &ce) {
		return err
	}
	if ce.Status == 420 {
		ce.Temporary, ce.Throttled = true, true
	}
	if ce.Code == "" && ce.Status != 0 {
		ce.Code = fmt.Sprintf("HTTP %d", ce.Status)
	}
	return ce
}

func messageLevel(ce *providers.CallError) bool {
	return !ce.Auth && !ce.Temporary && ce.Status >= 400 && ce.Status < 500
}

type address struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type transmission struct {
	Options    map[string]any    `json:"options"`
	Recipients []recipient       `json:"recipients"`
	Content    content           `json:"content"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type recipient struct {
	Address  address           `json:"address"`
	Metadata map[string]string `json:"metadata"`
}

type content struct {
	From    address           `json:"from"`
	Subject string            `json:"subject"`
	ReplyTo string            `json:"reply_to,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	HTML    string            `json:"html,omitempty"`
	Text    string            `json:"text,omitempty"`
}

// reserved headers SparkPost sets from the transmission's own fields.
var reserved = map[string]bool{"content-type": true, "content-transfer-encoding": true, "to": true, "from": true, "subject": true, "reply-to": true}

func (p *Provider) transmission(m providers.Message) transmission {
	meta := map[string]string{MetaMessage: m.ID.String(), MetaStream: string(m.Stream)}
	size := 0
	for _, k := range slices.Sorted(maps.Keys(m.Tags)) {
		if size += len(k) + len(m.Tags[k]); size > 4000 || k == MetaMessage || k == MetaStream {
			continue
		}
		meta[k] = m.Tags[k]
	}
	t := transmission{
		Options: map[string]any{
			// ReputeCast tracks opens and clicks itself.
			"open_tracking": false, "click_tracking": false, "initial_open": false,
			// SparkPost adds its own List-Unsubscribe header to every
			// non-transactional message. ReputeCast always sets its own
			// (with RFC 8058 one-click), so every message is sent as
			// transactional to keep exactly one; suppressions are pushed to
			// both of SparkPost's lists so they still apply.
			"transactional": true,
		},
		Recipients: []recipient{{Address: address{Email: m.To.Address, Name: m.To.Name}, Metadata: meta}},
		Content: content{From: address{Email: m.From.Address, Name: m.From.Name}, Subject: m.Subject, HTML: m.HTML, Text: m.Text,
			Headers: map[string]string{mimemsg.HeaderMessage: m.ID.String()}},
	}
	if p.s.IPPool != "" {
		t.Options["ip_pool"] = p.s.IPPool
	}
	if m.ReplyTo != nil {
		t.Content.ReplyTo = mimemsg.Address(*m.ReplyTo)
	}
	for k, v := range m.Headers {
		if !reserved[strings.ToLower(k)] {
			t.Content.Headers[k] = v
		}
	}
	return t
}

// Send implements providers.Provider: one transmission per message.
func (p *Provider) Send(ctx context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	return apiclient.Fanout(ctx, msgs, parallel, p.sendOne)
}

func (p *Provider) sendOne(ctx context.Context, m providers.Message) (providers.SendResult, error) {
	body, err := json.Marshal(p.transmission(m))
	if err != nil {
		// A message that can't be encoded fails on its own.
		return providers.SendResult{MessageID: m.ID, Err: &providers.SendError{Code: "Render", Message: err.Error()}}, nil //nolint:nilerr // reported on the result
	}
	res, err := p.api.Exec(ctx, apiclient.Request{Op: "POST /api/v1/transmissions", Method: http.MethodPost, Path: "/api/v1/transmissions",
		ContentType: "application/json", Body: body, Header: http.Header{"Idempotency-Key": {"rc-" + m.ID.String()}}})
	if err != nil {
		err = classify(err)
		var ce *providers.CallError
		if errors.As(err, &ce) && messageLevel(ce) {
			return providers.SendResult{MessageID: m.ID, Err: &providers.SendError{Code: ce.Code, Message: ce.Message}}, nil
		}
		return providers.SendResult{}, err
	}
	var out struct {
		apiErrors
		Results struct {
			Accepted int             `json:"total_accepted_recipients"`
			Rejected int             `json:"total_rejected_recipients"`
			ID       json.RawMessage `json:"id"`
			Errors   []struct {
				Message     string          `json:"message"`
				Description string          `json:"description"`
				Code        json.RawMessage `json:"code"`
			} `json:"rcpt_to_errors"`
		} `json:"results"`
	}
	_ = json.Unmarshal(res.Body, &out)
	if out.Results.Accepted == 0 && out.Results.Rejected > 0 {
		se := &providers.SendError{Code: "rejected", Message: "SparkPost rejected the recipient"}
		if len(out.Results.Errors) > 0 {
			e := out.Results.Errors[0]
			se.Code = strings.Trim(string(e.Code), `"`)
			se.Message = strings.TrimSpace(e.Message + ": " + e.Description)
		}
		return providers.SendResult{MessageID: m.ID, Err: se}, nil
	}
	id := strings.Trim(strings.TrimSpace(string(out.Results.ID)), `"`)
	if id == "" || id == "null" {
		// Accepted all the same; events find the message by rc_message.
		id = m.ID.String()
	}
	return providers.SendResult{MessageID: m.ID, ProviderMessageID: id}, nil
}
