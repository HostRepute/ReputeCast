// Package sms is the pluggable SMS-provider contract (design expansion phase
// 8, M31), on the sending-provider pattern: a catalog of adapters with
// capability badges from each adapter's manifest, one connection pattern
// with write-only credentials, health, and the shared contract-test suite in
// internal/sms/contracttest that every adapter passes.
//
// Texts go through the workspace's own account: an adapter talks to that
// account with the workspace's own credentials, and ReputeCast adds nothing
// to what the provider charges. Adapters live in internal/sms/<name>; the
// SMS domain (internal/texting) talks to these interfaces only and finds
// adapters through the registry.
//
// Required of every adapter: Send, delivery receipts and inbound messages
// (ParseWebhook) and their signatures (VerifyWebhook). Optional, declared
// in the manifest and checked by the contract suite: the account's numbers
// (SenderLister), webhook registration (WebhookRegistrar), opt-out sync
// (OptOutReader, OptOutSyncer), registration status (RegistrationReader)
// and pricing (PricingReader).
package sms

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Capability is a badge on a provider's catalog card and provider card.
type Capability string

// Capabilities, in the order the cards show them.
const (
	CapTollFree     Capability = "toll_free"
	Cap10DLC        Capability = "10dlc"
	CapSenderIDs    Capability = "sender_ids"
	CapReceipts     Capability = "receipts"
	CapInbound      Capability = "inbound"
	CapOptOutSync   Capability = "opt_out_sync"
	CapRegistration Capability = "registration"
	CapPricing      Capability = "pricing"
)

// Capabilities lists every capability in display order.
var Capabilities = []Capability{CapTollFree, Cap10DLC, CapSenderIDs, CapReceipts, CapInbound, CapOptOutSync, CapRegistration, CapPricing}

// Support is how far an adapter offers a capability.
type Support string

// Supports: yes (green), partial (amber: "read only", "set by hand") and no
// (struck through).
const (
	Yes     Support = "yes"
	Partial Support = "partial"
	No      Support = "no"
)

// Badge is one capability as the manifest declares it.
type Badge struct {
	Capability Capability `json:"capability" enum:"toll_free,10dlc,sender_ids,receipts,inbound,opt_out_sync,registration,pricing"`
	Support    Support    `json:"support" enum:"yes,partial,no"`
	// Note qualifies it: "outside US and Canada", "read only", "set by hand".
	Note string `json:"note,omitempty"`
	// NoteText is the message Note comes from (text.go).
	NoteText i18n.Text `json:"-"`
}

// Field is one input of the connection form.
type Field struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	// Secret fields are sealed and never shown again.
	Secret   bool     `json:"secret"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
	Help     string   `json:"help,omitempty"`
	Default  string   `json:"default,omitempty"`
	// LabelText and HelpText are the messages Label and Help come from.
	LabelText i18n.Text `json:"-"`
	HelpText  i18n.Text `json:"-"`
}

// Manifest describes an adapter for the catalog, the connection step and
// the provider cards.
type Manifest struct {
	Adapter string
	Name    string
	// Summary is the catalog card's line ("Pay as you go · numbers in 190
	// countries").
	Summary string
	Badges  []Badge
	Fields  []Field
	// Webhooks is how receipts and inbound texts reach ReputeCast:
	// "registered" (through the provider's API) or "manual" (an address and
	// a secret to paste in the provider's console).
	Webhooks string
	// Console links to the provider's console for a kind of page
	// ("registration", "numbers", "webhooks", "opt_outs") and an object.
	Console    func(kind, id string) string
	Version    string
	Maintainer string
	// SummaryText is the message Summary comes from, and Messages the
	// adapter's catalog its texts come from (text.go).
	SummaryText i18n.Text
	Messages    *i18n.Catalog
}

// Has reports the support the manifest declares for c.
func (m Manifest) Has(c Capability) Support {
	for _, b := range m.Badges {
		if b.Capability == c {
			return b.Support
		}
	}
	return No
}

// Descriptor is a registered adapter.
type Descriptor struct {
	Manifest
	// New opens a provider on an account.
	New func(cfg Config) (Provider, error)
}

// Config opens a provider.
type Config struct {
	ProviderID  uuid.UUID
	WorkspaceID uuid.UUID
	// Settings are the non-secret fields (account id, region, live or test).
	Settings map[string]string
	// Credentials are the secret fields (write-only: never logged or returned).
	Credentials map[string]string
	// WebhookSecret checks the provider's webhook signatures where the
	// provider signs with a secret ReputeCast holds.
	WebhookSecret string
	// BaseURL overrides the provider's API address (tests and the demo).
	BaseURL string
	// HTTP overrides the client (the SSRF guard, the demo's fakes).
	HTTP *http.Client
	// Now overrides the clock (tests).
	Now func() time.Time
}

// Value returns a setting or credential by key.
func (c Config) Value(key string) string {
	if v, ok := c.Credentials[key]; ok {
		return v
	}
	return c.Settings[key]
}

// Clock returns the configured clock.
func (c Config) Clock() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// Account is what checking the credentials reads about the account.
type Account struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Hint is the part of the account id or key that's safe to show
	// ("AC…3f9", "…7c21").
	Hint string `json:"hint"`
	// Live is false for a provider's test credentials.
	Live bool `json:"live"`
}

// Message is one text to one number, already rendered.
type Message struct {
	// ID is ReputeCast's message id; adapters that can carry a reference
	// pass it so receipts can be matched without the provider's id.
	ID uuid.UUID
	// To is the recipient in E.164.
	To string
	// From is the sender: an E.164 number or an alphanumeric sender ID.
	From string
	Body string
	// Encoding is how the text travels; adapters that must be told (Vonage)
	// use it.
	Encoding Encoding
	// StatusURL is where the provider should send this text's receipts,
	// for APIs that take it per message.
	StatusURL string
}

// Reason is what a failure means for ReputeCast: whether to retry, to
// suppress the number, or to count it toward the filtered threshold.
type Reason string

// Reasons.
const (
	// ReasonUnreachable: the phone is off or out of coverage. Temporary,
	// not suppressed.
	ReasonUnreachable Reason = "unreachable"
	// ReasonFiltered: carriers blocked the text (spam filtering).
	ReasonFiltered Reason = "filtered"
	// ReasonNotInService: the number doesn't exist. Suppressed.
	ReasonNotInService Reason = "not_in_service"
	// ReasonLandline: the number can't receive texts. Suppressed.
	ReasonLandline Reason = "landline"
	// ReasonInvalid: not a valid number. Suppressed.
	ReasonInvalid Reason = "invalid"
	// ReasonOptedOut: the provider's own opt-out list blocked it.
	ReasonOptedOut Reason = "opted_out"
	// ReasonRejected: the provider refused it (sender not registered,
	// content, a limit).
	ReasonRejected Reason = "rejected"
	ReasonOther    Reason = "other"
)

// Suppresses reports whether a failure for this reason puts the number on
// the SMS suppression list.
func (r Reason) Suppresses() bool {
	switch r {
	case ReasonNotInService, ReasonLandline, ReasonInvalid, ReasonOptedOut:
		return true
	}
	return false
}

// SendError is a text the provider refused or couldn't take.
type SendError struct {
	// Code is the provider's own code ("21211", "29").
	Code    string
	Message string
	Reason  Reason
	// Temporary errors are retried with back-off.
	Temporary bool
	// Throttled means the provider asked to slow down.
	Throttled bool
}

func (e *SendError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}

// SendResult is the provider's answer to one text.
type SendResult struct {
	ProviderMessageID string
	// Segments as the provider counted them (0 when it doesn't say).
	Segments int
	// Price per text when the provider reports it right away.
	Price *float64
}

// EventKind is a normalized provider event.
type EventKind string

// Event kinds, as the Events tab shows them.
const (
	EventQueued      EventKind = "queued"
	EventSent        EventKind = "sent"
	EventDelivered   EventKind = "delivered"
	EventUndelivered EventKind = "undelivered"
	EventFailed      EventKind = "failed"
	EventInbound     EventKind = "inbound"
)

// Event is one thing a provider reported: a receipt or an inbound text.
type Event struct {
	Kind              EventKind
	ProviderMessageID string
	// MessageID is ReputeCast's id when the provider echoed it.
	MessageID *uuid.UUID
	// To and From in E.164 (From is the sender for receipts, the person
	// for inbound texts).
	To   string
	From string
	// Body of an inbound text.
	Body string
	// Code is the provider's own code ("30007", "absent subscriber").
	Code   string
	Reason Reason
	// OptOut is the keyword the provider itself acted on ("STOP"), when it
	// handles keywords.
	OptOut string
	// Price is the provider's price for the text, when it reports one.
	Price      *float64
	Segments   int
	OccurredAt time.Time
	// Summary is the raw event as the Events tab shows it:
	// MessageStatus=undelivered · ErrorCode=30007.
	Summary string
}

// Provider is an SMS account ReputeCast sends through.
type Provider interface {
	Adapter() string
	// Check validates the credentials and reads the account.
	Check(ctx context.Context) (Account, error)
	// Send hands one text to the provider.
	Send(ctx context.Context, m Message) (SendResult, error)
	// VerifyWebhook checks a webhook as it arrives: its signature, and the
	// request it was signed for. It returns ErrBadSignature when it fails.
	VerifyWebhook(ctx context.Context, r *http.Request, body []byte) error
	// ParseWebhook reads a verified webhook into events. It runs later, in
	// a job, from the stored request.
	ParseWebhook(ctx context.Context, contentType string, body []byte) ([]Event, error)
}

// Sender is a number or sender ID found in the account.
type Sender struct {
	Kind  string `json:"kind" enum:"toll_free,10dlc,local,mobile,sender_id"`
	Value string `json:"value"`
	// Countries the provider says it can text.
	Countries []string `json:"countries"`
	// Throughput in texts per second, as the provider reports it.
	Throughput  float64 `json:"throughput"`
	ProviderRef string  `json:"providerRef"`
}

// SenderLister lists the numbers already in the account (the wizard's
// "Import your senders").
type SenderLister interface {
	Senders(ctx context.Context) ([]Sender, error)
}

// Webhooks is where a provider sends receipts and inbound texts.
type Webhooks struct {
	StatusURL  string
	InboundURL string
}

// WebhookSetup is what registering webhooks produced.
type WebhookSetup struct {
	// Secret signs the provider's webhooks from now on, when the provider
	// created one for ReputeCast (Twilio's shared keys); sealed by the caller.
	Secret string
	// Settings are non-secret values the adapter needs from now on (the
	// SNS topic and configuration set it created); the caller keeps them
	// with the provider's settings.
	Settings map[string]string
}

// WebhookRegistrar sets the receipt and inbound webhooks through the
// provider's API, on the account and on the given senders.
type WebhookRegistrar interface {
	RegisterWebhooks(ctx context.Context, hooks Webhooks, senders []Sender) (WebhookSetup, error)
}

// WebhookHandshaker answers a provider's subscription handshake (SNS).
type WebhookHandshaker interface {
	Handshake(ctx context.Context, r *http.Request, body []byte) (handled bool, err error)
}

// OptOut is a number on the provider's own opt-out list.
type OptOut struct {
	Phone   string
	Keyword string
	At      time.Time
}

// OptOutReader reads the provider's opt-out list ("Opt-out sync · read only").
type OptOutReader interface {
	OptOuts(ctx context.Context) ([]OptOut, error)
}

// OptOutSyncer also adds numbers to it (two-way sync).
type OptOutSyncer interface {
	OptOutReader
	AddOptOuts(ctx context.Context, phones []string) error
	RemoveOptOut(ctx context.Context, phone string) error
}

// Registration is a sender's registration as the provider reports it.
type Registration struct {
	// Sender is the number or ID it belongs to.
	Sender  string
	Kind    string `json:"kind" enum:"toll_free,brand,campaign,sender_id"`
	Country string
	Name    string
	Status  string `json:"status" enum:"not_registered,pending,verified,approved,rejected,allowed,not_supported"`
	Reason  string
	Since   *time.Time
	// Details are what was submitted (use case, volume, opt-in, terms).
	Details map[string]string
}

// RegistrationReader reads registration statuses (hourly, from the provider
// health job).
type RegistrationReader interface {
	Registrations(ctx context.Context, senders []Sender) ([]Registration, error)
}

// PricingReader reads the price of one outbound segment per country.
type PricingReader interface {
	Prices(ctx context.Context, countries []string) (map[string]float64, error)
}

// Errors adapters return.
var (
	// ErrCredentialsRejected: the provider refused the credentials.
	ErrCredentialsRejected = errors.New("sms: the provider rejected the credentials")
	// ErrUnavailable: the provider can't be reached or answers 5xx.
	ErrUnavailable = errors.New("sms: provider unavailable")
	// ErrBadSignature: a webhook whose signature doesn't verify.
	ErrBadSignature = errors.New("sms: webhook signature verification failed")
	// ErrNotSupported: the adapter doesn't offer this.
	ErrNotSupported = errors.New("sms: not supported by this provider")
)

// CallError is a failed API call.
type CallError struct {
	Op      string
	Status  int
	Code    string
	Message string
}

func (e *CallError) Error() string {
	return fmt.Sprintf("%s: HTTP %d %s %s", e.Op, e.Status, e.Code, e.Message)
}

// Is maps HTTP answers to the package's errors.
func (e *CallError) Is(target error) bool {
	switch target {
	case ErrCredentialsRejected:
		return e.Status == http.StatusUnauthorized || e.Status == http.StatusForbidden
	case ErrUnavailable:
		return e.Status >= 500 || e.Status == http.StatusTooManyRequests || e.Status == 0
	}
	return false
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Descriptor{}
	order      []string
)

// Register adds an adapter; adapters call it from their package's init.
func Register(d Descriptor) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, ok := registry[d.Adapter]; !ok {
		order = append(order, d.Adapter)
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

// catalogOrder is the brief's order: Twilio, Vonage, Plivo, Telnyx, AWS.
var catalogOrder = []string{"twilio", "vonage", "plivo", "telnyx", "aws_eum"}

// Catalog lists the adapters in the catalog's order.
func Catalog() []Descriptor {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Descriptor, 0, len(order))
	for _, k := range order {
		out = append(out, registry[k])
	}
	slices.SortStableFunc(out, func(a, b Descriptor) int {
		ia, ib := slices.Index(catalogOrder, a.Adapter), slices.Index(catalogOrder, b.Adapter)
		if ia < 0 {
			ia = len(catalogOrder)
		}
		if ib < 0 {
			ib = len(catalogOrder)
		}
		return ia - ib
	})
	return out
}

// Hint keeps the start and the end of an id and hides the rest:
// "AC8e2c…e8d3f9" → "AC…3f9".
func Hint(id string, prefix, suffix int) string {
	r := []rune(id)
	if len(r) <= prefix+suffix {
		return "…" + string(r[max(0, len(r)-suffix):])
	}
	return string(r[:prefix]) + "…" + string(r[len(r)-suffix:])
}

// CanSend reports whether a registration status lets a sender text a
// country.
func CanSend(status string) bool {
	switch status {
	case "verified", "approved", "allowed":
		return true
	}
	return false
}

// E164Digits is a number without its +, as some APIs take it.
func E164Digits(e164 string) string { return strings.TrimPrefix(e164, "+") }
