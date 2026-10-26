// Package providers defines the pluggable sending-provider contract. Every
// adapter lives in internal/providers/<name> and implements Provider plus any
// optional interfaces it supports; core code only talks to these interfaces
// (never imports an adapter) and finds adapters through the registry.
//
// Every adapter must pass the shared contract-test suite in
// internal/providers/contracttest.
package providers

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/mail"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
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
	// DedicatedIPs is the dedicated-IP option as badges show it ("optional,
	// $24.95 per IP per month"); "" when the provider offers none.
	DedicatedIPs   string `json:"dedicatedIps"`
	Sandbox        string `json:"sandbox"` // "detection", "test token", "mode", ""
	QuotaDiscovery bool   `json:"quotaDiscovery"`
	// RateDiscovery is true when the API also reports the maximum send rate
	// (SES: "Quota & rate discovery"; Postmark only reports quota).
	RateDiscovery bool   `json:"rateDiscovery"`
	Pricing       string `json:"pricing"` // human summary: "$0.10 per 1,000"
	// RewritesLinks / AddsOpenPixel are true when provider-side tracking is
	// on by default and should be turned off to avoid double tracking.
	RewritesLinks bool `json:"rewritesLinks"`
	AddsOpenPixel bool `json:"addsOpenPixel"`
	// RemovesWebhook is true when the adapter removes the webhook it
	// registered once the provider is removed (WebhookUnregistrar).
	RemovesWebhook bool `json:"removesWebhook"`
	// OneClick is how the provider carries the List-Unsubscribe and
	// List-Unsubscribe-Post headers ReputeCast adds: "" as given, signed
	// with the domain's DKIM key; OneClickReplaced when it sets its own;
	// OneClickUnsigned when its DKIM signature doesn't cover them, so
	// Gmail and Yahoo may not offer one-click unsubscribe.
	OneClick string `json:"oneClick,omitempty"`
	// PricingText and DedicatedIPsText are the messages Pricing and
	// DedicatedIPs come from (text.go). They are kept with the capability
	// snapshot of a connected provider, so it reads in each reader's
	// language later.
	PricingText      i18n.Text `json:"pricingText,omitzero"`
	DedicatedIPsText i18n.Text `json:"dedicatedIpsText,omitzero"`
}

// How a provider carries the one-click unsubscribe headers
// (Capabilities.OneClick).
const (
	OneClickReplaced = "replaced"
	OneClickUnsigned = "unsigned"
)

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

// WebhookUnregistrar removes the webhook RegisterWebhook added, when the
// provider is removed. A webhook that isn't there is not an error.
type WebhookUnregistrar interface {
	UnregisterWebhook(ctx context.Context, url string) error
}

// Quota is a provider's sending allowance.
type Quota struct {
	Kind    string // rolling_24h, monthly_plan, warmup_daily
	Max     int64  // 0 = unknown/unlimited
	Used    int64
	MaxRate float64 // per second, 0 = unknown
	Sandbox bool
	Account string // human status: "Production · no enforcement"
	// AccountText is the message Account comes from (text.go).
	AccountText i18n.Text
	// VDM is true when Amazon SES's Virtual Deliverability Manager is on for
	// the account; other providers leave it false.
	VDM bool
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
	// Dialer opens raw connections (SMTP, IMAP, POP3) through the same
	// address guard as HTTPClient; nil dials directly.
	Dialer *net.Dialer
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
	// Terms summarizes what the provider's terms allow, per account variant
	// (Postmark: Transactional and Broadcast streams). Shown on the routing
	// screen and used by Send Readiness.
	Terms []Terms `json:"terms,omitempty"`
	// Events, DomainSetup and SuppressionMode describe the adapter on the
	// Adapters tab ("SNS webhooks", "API", "Two-way").
	Events          string `json:"events,omitempty"`
	DomainSetup     string `json:"domainSetup,omitempty"`
	SuppressionMode string `json:"suppressionMode,omitempty"`
	// EventsText, DomainSetupText and SuppressionModeText are the messages
	// the three words come from (text.go).
	EventsText          i18n.Text `json:"-"`
	DomainSetupText     i18n.Text `json:"-"`
	SuppressionModeText i18n.Text `json:"-"`
	// Messages is the adapter's own catalog, which its texts come from
	// (WithMessages). Built-in adapters have one; community adapters write
	// English only.
	Messages *i18n.Catalog `json:"-"`
	New      func(Config) (Provider, error)
}

// Terms is what one variant of a provider account may be used for.
type Terms struct {
	// Variant names the account kind ("Transactional stream"); empty for the
	// provider as a whole.
	Variant string     `json:"variant,omitempty"`
	Bulk    BulkPolicy `json:"bulk"`
	Note    string     `json:"note"`
	Streams []Stream   `json:"streams"`
	// VariantText and NoteText are the messages Variant and Note come from.
	VariantText i18n.Text `json:"-"`
	NoteText    i18n.Text `json:"-"`
}

// Field is one input of a provider's connection form.
type Field struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Secret   bool     `json:"secret"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
	Help     string   `json:"help,omitempty"`
	// LabelText and HelpText are the messages Label and Help come from.
	LabelText i18n.Text `json:"-"`
	HelpText  i18n.Text `json:"-"`
}

// The registry holds built-in adapters (registered from init) and runtime
// adapters (community adapters, see SetRuntime), so reads take the lock.
var (
	registryMu sync.RWMutex
	registry   = map[string]Descriptor{}
)

// Register adds an adapter to the catalog. Adapters call it from init.
func Register(d Descriptor) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry[d.Adapter]; dup {
		panic("providers: duplicate adapter " + d.Adapter)
	}
	registry[d.Adapter] = d
}

// Lookup returns a registered adapter.
func Lookup(adapter string) (Descriptor, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	d, ok := registry[adapter]
	return d, ok
}

// Catalog returns every registered adapter.
func Catalog() []Descriptor {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Descriptor, 0, len(registry))
	for _, d := range registry {
		out = append(out, d)
	}
	return out
}
