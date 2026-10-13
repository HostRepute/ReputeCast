// Package azureacs is the Azure Communication Services Email adapter, on
// the data-plane REST API (api-version 2025-09-01) without the Azure SDK.
// It sends one message per call, authenticated with the resource's access
// key (HMAC-SHA256 request signing) or a Microsoft Entra ID application
// (client credentials), and receives delivery reports from an Event Grid
// subscription: the validation handshake is answered, and deliveries are
// authenticated with a shared secret Event Grid sends as a delivery header
// (or in the URL's query).
//
// What the data-plane API can't do stays manual and is not claimed:
// domains, sender usernames, Event Grid subscriptions and suppression lists
// are Azure Resource Manager resources, set up in the Azure portal. ACS
// keeps its own suppression list of hard-bounced addresses (not exposed);
// its Suppressed events reach the shared list here.
//
// Terms (checked 2026-10-13): the Email overview says ACS "facilitates
// high-volume transactional, bulk, and marketing emails"
// (https://learn.microsoft.com/en-us/azure/communication-services/concepts/email/email-overview),
// under the Messaging Policy, which requires written consent for
// promotional messages and an unsubscribe link in every email
// (https://learn.microsoft.com/en-us/azure/communication-services/concepts/sms/messaging-policy),
// and the Online Services Acceptable Use Policy, which forbids spam
// (https://www.microsoft.com/licensing/terms/product/ForOnlineServices/all).
// Bulk marketing is allowed, with two caveats in the terms note: ACS's DKIM
// signature doesn't cover List-Unsubscribe, so Gmail and Yahoo one-click
// unsubscribe can't be relied on (a gap Microsoft acknowledged in 2026),
// and Microsoft is retiring ACS as a standalone offering on 30 September
// 2028, limiting new onboarding from 23 October 2026
// (https://learn.microsoft.com/en-us/azure/communication-services/acs-retirement-and-breaking-changes-guide).
package azureacs

import (
	"cmp"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/apiclient"
	"github.com/hostrepute/reputecast/internal/version"
)

// Adapter is the registry key.
const Adapter = "azureacs"

// APIVersion is the latest GA version of the Email data-plane API.
const APIVersion = "2025-09-01"

// DefaultAuthority is Microsoft Entra ID's token endpoint host.
const DefaultAuthority = "https://login.microsoftonline.com"

// SecretHeader carries the webhook secret on Event Grid deliveries (a
// delivery property marked secret in the subscription).
const SecretHeader = "X-ReputeCast-Secret" //nolint:gosec // a header name, not a credential

// parallel is how many calls one Send makes at a time when it is handed
// more than one message (ACS's default limits are low).
const parallel = 2

// Settings are the non-secret connection settings.
type Settings struct {
	// Endpoint is the resource's endpoint
	// (https://<resource>.communication.azure.com); taken from the
	// connection string when that is used.
	Endpoint string `json:"endpoint,omitempty"`
	// TenantID and ClientID select Microsoft Entra ID authentication
	// (with client_secret) instead of the access key.
	TenantID string `json:"tenant_id,omitempty"`
	ClientID string `json:"client_id,omitempty"`
	// PerMinute and PerHour are the sending limits Azure granted (30 and
	// 100 for a custom domain until raised), shown as the account's rate.
	PerMinute int64 `json:"per_minute,omitempty"`
	PerHour   int64 `json:"per_hour,omitempty"`
	// AuthorityURL overrides Entra ID's token host (tests, sovereign clouds).
	AuthorityURL string `json:"authority_url,omitempty"`
}

// Credentials are write-only.
type Credentials struct {
	// ConnectionString is "endpoint=https://…/;accesskey=…".
	ConnectionString string `json:"connection_string,omitempty"`
	// AccessKey is the key alone, with the endpoint in the settings.
	AccessKey    string `json:"access_key,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
}

func init() { providers.Register(Descriptor()) }

// Descriptor describes the adapter for the catalog.
func Descriptor() providers.Descriptor {
	return providers.Descriptor{
		Adapter: Adapter,
		Name:    "Azure Communication Services",
		Kind:    "api",
		Capabilities: providers.Capabilities{
			Bulk:      providers.BulkAllowed,
			Webhooks:  "Event Grid",
			BatchSize: 1,
			Pricing:   "pay as you go, per email and per MB",
		},
		Version:    version.Version,
		Maintainer: "HostRepute",
		Terms: []providers.Terms{{Bulk: providers.BulkAllowed,
			Note:    "Bulk and marketing mail allowed with written consent and an unsubscribe link. ACS doesn't DKIM-sign List-Unsubscribe, so Gmail and Yahoo one-click unsubscribe can't be relied on; Microsoft retires ACS as a standalone service on 30 Sep 2028.",
			Streams: []providers.Stream{providers.StreamBroadcast, providers.StreamAutomation, providers.StreamSystem}}},
		Events:          "Event Grid",
		DomainSetup:     "Manual DNS",
		SuppressionMode: "Local only",
		Fields: []providers.Field{
			{Key: "connection_string", Label: "Connection string", Secret: true,
				Help: "Azure portal › your Communication Services resource › Keys. Or use an Entra ID application below."},
			{Key: "endpoint", Label: "Endpoint", Help: "https://<resource>.communication.azure.com, only with Entra ID."},
			{Key: "tenant_id", Label: "Entra ID tenant ID", Help: "For Microsoft Entra ID authentication instead of the access key."},
			{Key: "client_id", Label: "Application (client) ID", Help: "An app registration with the Communication and Email Service Owner role on the resource."},
			{Key: "client_secret", Label: "Client secret", Secret: true},
			{Key: "per_minute", Label: "Sending limit · per minute", Help: "30 for a custom domain until Azure raises it."},
			{Key: "per_hour", Label: "Sending limit · per hour", Help: "100 for a custom domain until Azure raises it."},
		},
		New: New,
	}
}

// Provider talks to one Communication Services resource.
type Provider struct {
	cfg      providers.Config
	s        Settings
	c        Credentials
	endpoint string
	key      []byte
	api      *apiclient.Client
	http     *http.Client
}

var _ providers.Provider = (*Provider)(nil)

// New builds the adapter from a connected provider's configuration.
func New(cfg providers.Config) (providers.Provider, error) {
	p := &Provider{cfg: cfg, http: cfg.HTTPClient}
	if len(cfg.Settings) > 0 {
		if err := json.Unmarshal(cfg.Settings, &p.s); err != nil {
			return nil, fmt.Errorf("azureacs settings: %w", err)
		}
	}
	if len(cfg.Credentials) > 0 {
		if err := json.Unmarshal(cfg.Credentials, &p.c); err != nil {
			return nil, fmt.Errorf("azureacs credentials: %w", err)
		}
	}
	if p.http == nil {
		p.http = &http.Client{Timeout: apiclient.DefaultTimeout}
	}
	endpoint, key, err := parseConnectionString(p.c.ConnectionString)
	if err != nil {
		return nil, err
	}
	if endpoint == "" {
		endpoint = p.s.Endpoint
	}
	if key == "" {
		key = strings.TrimSpace(p.c.AccessKey)
	}
	if key != "" {
		if p.key, err = base64.StdEncoding.DecodeString(key); err != nil {
			return nil, errors.New("azureacs: the access key is not base64")
		}
	}
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if endpoint != "" && (err != nil || u.Host == "" || (u.Scheme != "https" && !strings.HasPrefix(u.Host, "127.0.0.1"))) {
		return nil, errors.New("azureacs: the endpoint must be an https:// URL")
	}
	p.endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	p.api = apiclient.New(p.endpoint, p.http)
	p.api.Header.Set("User-Agent", "ReputeCast/"+version.Version)
	p.api.Decode = decodeError
	return p, nil
}

// parseConnectionString reads "endpoint=…;accesskey=…".
func parseConnectionString(cs string) (endpoint, key string, err error) {
	cs = strings.TrimSpace(cs)
	if cs == "" {
		return "", "", nil
	}
	for _, part := range strings.Split(cs, ";") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch strings.ToLower(k) {
		case "endpoint":
			endpoint = v
		case "accesskey":
			key = v
		}
	}
	if endpoint == "" || key == "" {
		return "", "", errors.New("azureacs: the connection string needs endpoint= and accesskey=")
	}
	return endpoint, key, nil
}

// Adapter implements providers.Provider.
func (p *Provider) Adapter() string { return Adapter }

// Capabilities implements providers.Provider.
func (p *Provider) Capabilities() providers.Capabilities { return Descriptor().Capabilities }

func decodeError(_ int, body []byte) (string, string) {
	var e struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && (e.Error.Code != "" || e.Error.Message != "") {
		return e.Error.Code, strings.TrimSpace(e.Error.Message)
	}
	return "", strings.TrimSpace(string(body))
}

// request signs and sends one data-plane call; extra headers (Operation-Id)
// are sent as they are.
func (p *Provider) request(ctx context.Context, op, method, path string, body []byte, extra http.Header) (*apiclient.Response, error) {
	if p.endpoint == "" {
		return nil, &providers.CallError{Op: op, Auth: true, Message: "no endpoint: enter the connection string or the endpoint"}
	}
	header := extra.Clone()
	if header == nil {
		header = http.Header{}
	}
	switch {
	case p.s.TenantID != "" && p.s.ClientID != "":
		token, err := p.token(ctx)
		if err != nil {
			return nil, err
		}
		header.Set("Authorization", "Bearer "+token)
	case len(p.key) > 0:
		u, _ := url.Parse(p.endpoint + path)
		date := time.Now().UTC().Format(http.TimeFormat)
		sum := sha256.Sum256(body)
		hash := base64.StdEncoding.EncodeToString(sum[:])
		header.Set("x-ms-date", date)
		header.Set("x-ms-content-sha256", hash)
		header.Set("Authorization", "HMAC-SHA256 SignedHeaders=x-ms-date;host;x-ms-content-sha256&Signature="+Sign(p.key, method, u.RequestURI(), date, u.Host, hash))
	default:
		return nil, &providers.CallError{Op: op, Auth: true, Message: "no credentials: enter the connection string or an Entra ID application"}
	}
	ct := ""
	if body != nil {
		ct = "application/json"
	}
	return p.api.Exec(ctx, apiclient.Request{Op: op, Method: method, Path: path, ContentType: ct, Body: body, Header: header})
}

// Sign returns the HMAC-SHA256 signature of an ACS request: the method,
// the path and query, and "date;host;content hash", keyed with the
// decoded access key.
func Sign(key []byte, method, pathAndQuery, date, host, contentHash string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(method + "\n" + pathAndQuery + "\n" + date + ";" + host + ";" + contentHash))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

type address struct {
	Address     string `json:"address"`
	DisplayName string `json:"displayName,omitempty"`
}

type sendRequest struct {
	SenderAddress string `json:"senderAddress"`
	Content       struct {
		Subject   string `json:"subject"`
		PlainText string `json:"plainText,omitempty"`
		HTML      string `json:"html,omitempty"`
	} `json:"content"`
	Recipients struct {
		To []address `json:"to"`
	} `json:"recipients"`
	ReplyTo                        []address         `json:"replyTo,omitempty"`
	Headers                        map[string]string `json:"headers,omitempty"`
	UserEngagementTrackingDisabled bool              `json:"userEngagementTrackingDisabled"`
}

// reserved headers ACS sets from the message's own fields.
var reserved = map[string]bool{"from": true, "to": true, "cc": true, "bcc": true, "subject": true, "reply-to": true, "date": true,
	"message-id": true, "mime-version": true, "content-type": true, "content-transfer-encoding": true, "sender": true, "return-path": true}

// Send implements providers.Provider: one POST /emails:send per message.
// The ReputeCast message id is the operation id, so Event Grid reports
// carry it as their messageId.
func (p *Provider) Send(ctx context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	return apiclient.Fanout(ctx, msgs, parallel, p.sendOne)
}

func (p *Provider) sendOne(ctx context.Context, m providers.Message) (providers.SendResult, error) {
	var in sendRequest
	// The sender's display name comes from its sender username in Azure;
	// the API takes the address only.
	in.SenderAddress = m.From.Address
	in.Content.Subject, in.Content.PlainText, in.Content.HTML = m.Subject, m.Text, m.HTML
	in.Recipients.To = []address{{Address: m.To.Address, DisplayName: m.To.Name}}
	if m.ReplyTo != nil {
		in.ReplyTo = []address{{Address: m.ReplyTo.Address, DisplayName: m.ReplyTo.Name}}
	}
	in.Headers = map[string]string{"X-ReputeCast-Message": m.ID.String()}
	for k, v := range m.Headers {
		if !reserved[strings.ToLower(k)] {
			in.Headers[k] = v
		}
	}
	// ReputeCast tracks opens and clicks itself.
	in.UserEngagementTrackingDisabled = true
	body, _ := json.Marshal(in)
	res, err := p.request(ctx, "POST /emails:send", http.MethodPost, "/emails:send?api-version="+APIVersion, body, http.Header{"Operation-Id": {m.ID.String()}})
	if err != nil {
		var ce *providers.CallError
		if errors.As(err, &ce) && !ce.Auth && !ce.Temporary && ce.Status >= 400 && ce.Status < 500 {
			return providers.SendResult{MessageID: m.ID, Err: &providers.SendError{Code: cmp.Or(ce.Code, fmt.Sprintf("HTTP %d", ce.Status)), Message: ce.Message}}, nil
		}
		return providers.SendResult{}, err
	}
	var out struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(res.Body, &out)
	id := strings.TrimSpace(out.ID)
	if id == "" {
		id = m.ID.String()
	}
	return providers.SendResult{MessageID: m.ID, ProviderMessageID: id}, nil
}
