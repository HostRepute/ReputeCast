// Package providers defines the pluggable sending-provider contract. Every
// adapter lives in internal/providers/<name> and implements Provider plus any
// optional interfaces it supports; core code only talks to these interfaces
// (never imports an adapter) and finds adapters through the Registry.
//
// Every adapter must pass the shared contract-test suite in
// internal/providers/contracttest (Send, ParseWebhook with signature
// verification and event normalization, Capabilities).
package providers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"time"

	"github.com/google/uuid"
)

// Stream is the kind of mail.
type Stream string

// Streams.
const (
	StreamBroadcast  Stream = "broadcast"
	StreamAutomation Stream = "automation"
	StreamSystem     Stream = "system"
)

// BulkPolicy says whether a provider's terms allow bulk marketing mail.
type BulkPolicy string

// Bulk policies shown as capability badges and checked by Send Readiness.
const (
	BulkAllowed          BulkPolicy = "allowed"
	BulkBroadcastStreams BulkPolicy = "broadcast_streams_only" // e.g. Postmark: only on Broadcast message streams
	BulkForbidden        BulkPolicy = "forbidden"
	BulkYourPolicy       BulkPolicy = "your_policy" // self-hosted (Postal, SMTP)
)

// Capabilities are the badges shown on each provider (spec 5.2).
type Capabilities struct {
	Bulk            BulkPolicy `json:"bulk"`
	Webhooks        string     `json:"webhooks"`  // "SNS", "Registered", "Bounce mailbox", ""
	DomainAPI       string     `json:"domainApi"` // "full", "dkim_only", ""
	SuppressionSync bool       `json:"suppressionSync"`
	BatchSize       int        `json:"batchSize"` // messages per API call; 1 = no batching
	DedicatedIPs    string     `json:"dedicatedIps"`
	Sandbox         string     `json:"sandbox"` // "detection", "test token", "mode", ""
	QuotaDiscovery  bool       `json:"quotaDiscovery"`
	Pricing         string     `json:"pricing"` // human summary: "$0.10 per 1,000"
	// RewritesLinks / AddsOpenPixel are true when provider-side tracking is
	// on by default and should be turned off to avoid double tracking.
	RewritesLinks bool `json:"rewritesLinks"`
	AddsOpenPixel bool `json:"addsOpenPixel"`
}

// Message is one email to one recipient, already rendered and personalized.
type Message struct {
	ID      uuid.UUID // ReputeCast message id; also used in Message-ID and tracking
	Stream  Stream
	From    mail.Address
	ReplyTo *mail.Address
	To      mail.Address
	Subject string
	HTML    string
	Text    string
	// Headers include List-Unsubscribe, List-Unsubscribe-Post (RFC 8058),
	// Feedback-ID and X-ReputeCast-Message.
	Headers map[string]string
	// Tags are passed through where the provider supports them (SES tags,
	// Postmark metadata) and come back on events.
	Tags map[string]string
}

// SendResult is the outcome for one message of a batch.
type SendResult struct {
	MessageID         uuid.UUID
	ProviderMessageID string
	Err               *SendError
}

// SendError explains a refused message.
type SendError struct {
	// Temporary errors (throttling, 5xx, timeouts) are retried; permanent
	// ones (invalid recipient, suppressed at provider) are not.
	Temporary bool
	Throttled bool
	Code      string // provider's code, kept for the event log
	Message   string
}

func (e *SendError) Error() string { return e.Code + ": " + e.Message }

// ErrUnavailable means the provider can't be reached at all (outage).
var ErrUnavailable = errors.New("provider unavailable")

// EventType is a normalized delivery or engagement event.
type EventType string

// Normalized events; the provider's original type and code are kept.
const (
	EventDelivered   EventType = "delivered"
	EventDeferred    EventType = "deferred"
	EventHardBounce  EventType = "hard_bounce"
	EventSoftBounce  EventType = "soft_bounce"
	EventComplaint   EventType = "complaint"
	EventRejected    EventType = "rejected"
	EventOpen        EventType = "open"
	EventClick       EventType = "click"
	EventUnsubscribe EventType = "unsubscribe"
)

// Event is a provider event mapped to ReputeCast's model.
type Event struct {
	Type              EventType
	ProviderMessageID string
	MessageID         *uuid.UUID // when the provider echoes our tag/header
	Recipient         string
	OccurredAt        time.Time
	OriginalType      string // "Bounce/Permanent/General", "HardBounce"
	OriginalCode      string // "550 5.1.1"
	Diagnostic        string
	// BounceClass is the shared taxonomy: mailbox_unknown, mailbox_full,
	// domain_invalid, policy_block, spam_block, reputation, temporary, other.
	BounceClass string
	URL         string // clicks
	UserAgent   string
	Raw         json.RawMessage
}

// Provider is the contract every adapter implements.
type Provider interface {
	// Adapter is the registry key ("ses", "postmark", "smtp"…).
	Adapter() string
	Capabilities() Capabilities
	// Send delivers a batch (at most Capabilities().BatchSize messages).
	// It returns one result per message; an error means the whole call
	// failed (wrap ErrUnavailable for outages).
	Send(ctx context.Context, msgs []Message) ([]SendResult, error)
	// ParseWebhook verifies the request's signature and maps its payload to
	// events. It must not do network I/O except what verification requires
	// (for example fetching the SNS signing certificate, cached).
	ParseWebhook(ctx context.Context, r *http.Request, body []byte) ([]Event, error)
}

// ErrBadSignature is returned by ParseWebhook for unverifiable payloads.
var ErrBadSignature = errors.New("webhook signature verification failed")

// DNSRecord is one record a provider requires for a sending domain.
type DNSRecord struct {
	Purpose string `json:"purpose"` // dkim, spf, return_path, dmarc, verification
	Type    string `json:"type"`    // CNAME, TXT, MX
	Name    string `json:"name"`
	Value   string `json:"value"`
	// Status is filled by DNS checks: pending, verified, wrong_value, missing.
	Status string `json:"status,omitempty"`
}

// DomainManager is implemented by providers that set up domains over their API.
type DomainManager interface {
	AddDomain(ctx context.Context, domain string) ([]DNSRecord, error)
	DomainStatus(ctx context.Context, domain string) ([]DNSRecord, error)
}

// WebhookRegistrar is implemented by providers whose webhooks ReputeCast
// registers automatically.
type WebhookRegistrar interface {
	RegisterWebhook(ctx context.Context, url string) error
}

// Quota is a provider's sending allowance.
type Quota struct {
	Kind    string // rolling_24h, monthly_plan, warmup_daily
	Max     int64  // 0 = unknown/unlimited
	Used    int64
	MaxRate float64 // per second, 0 = unknown
	Sandbox bool
	Account string // human status: "Production · no enforcement"
}

// QuotaReporter is implemented by providers that expose quota and rate.
type QuotaReporter interface {
	Quota(ctx context.Context) (Quota, error)
}

// Suppression is one address suppressed at a provider.
type Suppression struct {
	Email     string
	Reason    string // hard_bounce, complaint, unsubscribe, manual
	Code      string
	CreatedAt time.Time
}

// SuppressionSyncer is implemented by providers with a suppression list API.
type SuppressionSyncer interface {
	ListSuppressions(ctx context.Context, since time.Time) ([]Suppression, error)
	AddSuppressions(ctx context.Context, emails []string) error
}

// Verifier is implemented by adapters that can check credentials at connect time.
type Verifier interface {
	// Verify checks the credentials and returns account details for the UI.
	Verify(ctx context.Context) (Quota, error)
}

// Config is the non-secret connection settings and the decrypted
// credentials of one connected provider.
type Config struct {
	ProviderID  uuid.UUID
	WorkspaceID uuid.UUID
	Settings    json.RawMessage // providers.config column
	Credentials json.RawMessage // decrypted credentials_enc
	// WebhookSecret signs or authenticates incoming webhooks where the
	// provider supports a shared secret.
	WebhookSecret string
	HTTPClient    *http.Client
}

// Descriptor describes an adapter in the catalog (setup wizard, Adapters tab).
type Descriptor struct {
	Adapter      string       `json:"adapter"`
	Name         string       `json:"name"`
	Kind         string       `json:"kind"` // api, smtp, self_hosted, community
	Capabilities Capabilities `json:"capabilities"`
	Version      string       `json:"version"`
	Maintainer   string       `json:"maintainer"`
	// Fields lists what the connection step asks for.
	Fields []Field `json:"fields"`
	New    func(Config) (Provider, error)
}

// Field is one input of a provider's connection form.
type Field struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Secret   bool     `json:"secret"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
	Help     string   `json:"help,omitempty"`
}

var registry = map[string]Descriptor{}

// Register adds an adapter to the catalog. Adapters call it from init.
func Register(d Descriptor) {
	if _, dup := registry[d.Adapter]; dup {
		panic("providers: duplicate adapter " + d.Adapter)
	}
	registry[d.Adapter] = d
}

// Lookup returns a registered adapter.
func Lookup(adapter string) (Descriptor, bool) {
	d, ok := registry[adapter]
	return d, ok
}

// Catalog returns every registered adapter.
func Catalog() []Descriptor {
	out := make([]Descriptor, 0, len(registry))
	for _, d := range registry {
		out = append(out, d)
	}
	return out
}
