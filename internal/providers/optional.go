package providers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Optional interfaces: core code type-asserts before using one, so adapters
// that don't implement them keep working.

// ErrUnauthorized means the provider rejected the credentials (HTTP 401/403,
// SMTP 535). Health checks report it as "Down · auth failed" and ask for new
// credentials instead of retrying.
var ErrUnauthorized = errors.New("provider rejected the credentials")

// ErrNotSupported is returned by optional operations an account or plan can't do.
var ErrNotSupported = errors.New("not supported by this provider")

// CallError describes a failed API call for the event log and health
// ("503 ServiceUnavailable · SendEmail"). It unwraps to ErrUnavailable for
// outages and to ErrUnauthorized for rejected credentials.
type CallError struct {
	Op        string // provider operation: "SendEmail", "POST /email/batch", "GET /server"
	Status    int    // HTTP status or SMTP reply code; 0 for network errors
	Code      string // provider error code: "ServiceUnavailable", "ErrorCode 10"
	Message   string
	Temporary bool
	Throttled bool
	Auth      bool
	Cause     error
	// Text is the message Message comes from, when the adapter wrote the
	// sentence itself (a connection test's finding); a provider's own answer
	// has none.
	Text i18n.Text
}

func (e *CallError) Error() string {
	s := e.Op
	if e.Status != 0 {
		s += fmt.Sprintf(": %d", e.Status)
	}
	if e.Code != "" {
		s += " " + e.Code
	}
	if e.Message != "" {
		s += ": " + e.Message
	}
	if e.Cause != nil && e.Message == "" {
		s += ": " + e.Cause.Error()
	}
	return s
}

// Unwrap lets errors.Is match ErrUnavailable / ErrUnauthorized and the cause.
func (e *CallError) Unwrap() []error {
	var out []error
	if e.Auth {
		out = append(out, ErrUnauthorized)
	} else if e.Temporary && !e.Throttled {
		out = append(out, ErrUnavailable)
	}
	if e.Cause != nil {
		out = append(out, e.Cause)
	}
	return out
}

// WebhookVerifier checks a webhook request's signature without parsing it,
// so the ingest endpoint can answer immediately and leave normalization to
// a background job. Adapters without it are verified through ParseWebhook.
type WebhookVerifier interface {
	VerifyWebhook(ctx context.Context, r *http.Request, body []byte) error
}

// WebhookHandshaker completes subscription handshakes that are not events,
// such as an SNS SubscriptionConfirmation. handled is true when the request
// was a handshake; response, when set, is written back with 200.
type WebhookHandshaker interface {
	Handshake(ctx context.Context, r *http.Request, body []byte) (handled bool, response []byte, err error)
}

// WebhookSecrets names the request headers that carry a shared webhook
// secret in plain text, beyond Authorization (which is never stored). The
// ingest endpoint leaves them out of the stored raw event; normalization
// runs on verified events and never needs them.
type WebhookSecrets interface {
	WebhookSecretHeaders() []string
}

// WebhookStatus is the provider-side view of the webhook pointing at us.
type WebhookStatus struct {
	Registered bool
	ID         string
	URL        string
	// Queued and Sent come from the provider's own webhook delivery log when
	// its API exposes one; -1 when unknown.
	Queued int64
	Sent   int64
}

// WebhookInspector reports whether the provider has a webhook pointing at url.
type WebhookInspector interface {
	InspectWebhook(ctx context.Context, url string) (WebhookStatus, error)
}

// WebhookTester asks the provider to send a test event to its webhook.
type WebhookTester interface {
	SendTestEvent(ctx context.Context, url string) error
}

// EventBackfiller fetches the current delivery status of messages from the
// provider's API, for when webhooks went silent.
type EventBackfiller interface {
	Backfill(ctx context.Context, providerMessageIDs []string) ([]Event, error)
}

// Tracking is the provider-side open and click tracking configuration.
type Tracking struct {
	OpenPixel   bool
	LinkRewrite bool
	// Detail explains the setting in the provider's words ("Server setting
	// TrackOpens is on, and TrackLinks is HtmlAndText").
	Detail string
	// Scope names where it applies ("Configuration set rc-broadcasts",
	// "Transactional stream").
	Scope string
	// DetailText and ScopeText are the messages Detail and Scope come from.
	DetailText i18n.Text
	ScopeText  i18n.Text
}

// TrackingManager reads and turns off provider-side tracking, so opens and
// clicks are not counted twice and links are not rewritten twice.
type TrackingManager interface {
	Tracking(ctx context.Context) (Tracking, error)
	DisableTracking(ctx context.Context) error
}

// BroadcastStreamCreator is implemented by providers whose terms allow bulk
// mail only on a dedicated stream (Postmark Broadcast message streams).
type BroadcastStreamCreator interface {
	CreateBroadcastStream(ctx context.Context, name string) (streamID string, err error)
}

// ServiceStatus is what a provider's public status page reports.
type ServiceStatus struct {
	Incident bool
	Summary  string
	URL      string
	Updated  time.Time
	// SummaryText is the message Summary comes from.
	SummaryText i18n.Text
}

// StatusReporter reads the provider's public status page during an outage.
type StatusReporter interface {
	ServiceStatus(ctx context.Context) (ServiceStatus, error)
}

// Reputation is the bounce and complaint rate a provider measures for the
// account itself: what its own review and suspension thresholds apply to.
type Reputation struct {
	// Bounce and Complaint are percentages (0.2 means 0.2%).
	Bounce    float64
	Complaint float64
	// Window describes the period measured ("7 d").
	Window string
}

// ReputationReporter is implemented by adapters whose API reports the
// account's own bounce and complaint rates (Postmark outbound stats). Without
// it, ReputeCast counts the rates from the events it received.
type ReputationReporter interface {
	Reputation(ctx context.Context) (Reputation, error)
}

// StatusPage is the public status page of a provider, shown as a button
// during outages ("AWS status").
type StatusPage interface {
	StatusPageURL() (label, url string)
}

type verifiedKey struct{}

// WithSignatureVerified marks ctx for a payload whose signature was already
// verified at ingest, so ParseWebhook can skip verification (and time-window
// checks) when the normalization job parses it later.
func WithSignatureVerified(ctx context.Context) context.Context {
	return context.WithValue(ctx, verifiedKey{}, true)
}

// SignatureVerified reports whether ctx carries WithSignatureVerified.
func SignatureVerified(ctx context.Context) bool {
	v, _ := ctx.Value(verifiedKey{}).(bool)
	return v
}

// Registered reports whether an adapter is in the registry (and therefore
// connectable from this build).
func Registered(adapter string) bool {
	_, ok := Lookup(adapter)
	return ok
}

// Receiving mail (M58). A workspace receives mail through one inbound
// route: a connected provider's receiving, or a mailbox ReputeCast polls.
// The provider posts what it receives to ReputeCast's inbound hook
// (POST /hooks/inbound/{token}); the hook checks the request at once
// (VerifyInbound) and stores it, and a job parses it later (ParseInbound).
// Capabilities.Inbound says which of the interfaces below an adapter has.

// Values of Capabilities.Inbound.
const (
	// InboundAutomatic: ReputeCast points the provider's receiving at its
	// hook itself (InboundRegistrar); people add the MX record.
	InboundAutomatic = "automatic"
	// InboundManual: people follow the adapter's steps in the provider's
	// console (InboundSetup.Steps) and add the MX record.
	InboundManual = "manual"
)

// InboundRoute is the route a workspace receives through, as an adapter
// sees it.
type InboundRoute struct {
	ID uuid.UUID
	// Domain is the domain mail is received for (reply.news.acme.io).
	Domain string
	// Settings are the values of InboundSetup.Settings people entered
	// (an SNS topic ARN).
	Settings map[string]string
	// Secret is the route's own secret: the password of the hook's basic
	// authentication, or the key of its signature, for providers that
	// authenticate inbound requests with a shared secret.
	Secret string
	// URL is the hook's address without credentials
	// (https://reputecast.example/hooks/inbound/<token>), for setup steps.
	URL string
}

// InboundUser is the user name of the hook's basic authentication: the
// password is InboundRoute.Secret.
const InboundUser = "reputecast"

// InboundAttachment is one attachment of a received message. Only its name,
// type and size are kept; Data is read for reports (DMARC aggregate
// reports arrive as attachments) and dropped.
type InboundAttachment struct {
	Name        string
	ContentType string
	Size        int64
	Data        []byte
}

// InboundMessage is one received message. Adapters whose provider passes
// the whole message set Raw (RFC 5322) and may leave the header and body
// fields empty: ReputeCast parses Raw itself. Others fill the fields from
// the provider's parsed payload. Envelope facts the message itself doesn't
// carry (the recipients, the provider's verdicts) come in their fields
// either way.
type InboundMessage struct {
	Raw []byte
	// From, Subject, MessageID, InReplyTo, To and References are the
	// message's headers (addresses as "Name <addr>" or the address).
	From, Subject, MessageID, InReplyTo string
	To, References                      []string
	// Headers are the message's headers when Raw is empty.
	Headers     map[string][]string
	Text, HTML  string
	Attachments []InboundAttachment
	// Recipients are the envelope's RCPT TO addresses, when the provider
	// reports them.
	Recipients        []string
	ProviderMessageID string
	// Spam is the provider's spam verdict: "", "pass" or "spam".
	Spam string
	// SPF, DKIM and DMARC are the provider's authentication verdicts:
	// "pass", "fail", "none" or "".
	SPF, DKIM, DMARC string
	ReceivedAt       time.Time
}

// InboundSetup is what setting up a route through the adapter takes.
type InboundSetup struct {
	// Automatic is true when ReputeCast registers the hook itself.
	Automatic bool
	// MX are the records people add for route.Domain.
	MX []DNSRecord
	// Steps are what people do in the provider's console, in order, from
	// the adapter's catalog (with the route's domain and URL as arguments).
	Steps []i18n.Text
	// Settings are the values people enter for the route (an SNS topic
	// ARN); they come back in InboundRoute.Settings.
	Settings []Field
}

// InboundReceiver is implemented by adapters whose provider posts received
// mail to ReputeCast (Capabilities.Inbound is set).
type InboundReceiver interface {
	InboundSetup(route InboundRoute) InboundSetup
	// VerifyInbound checks the request when it arrives, without parsing
	// it; a request that fails is ErrBadSignature.
	VerifyInbound(ctx context.Context, route InboundRoute, r *http.Request, body []byte) error
	// ParseInbound reads a verified request in a job, from the stored body
	// and headers (credentials are never stored). It may call the
	// provider's API (Resend sends only metadata).
	ParseInbound(ctx context.Context, route InboundRoute, r *http.Request, body []byte) ([]InboundMessage, error)
}

// InboundRegistrar is implemented by adapters that point the provider's
// receiving for route.Domain at url themselves
// (Capabilities.Inbound == InboundAutomatic).
type InboundRegistrar interface {
	RegisterInbound(ctx context.Context, route InboundRoute, url string) error
	// UnregisterInbound removes what RegisterInbound added; a registration
	// that isn't there is not an error.
	UnregisterInbound(ctx context.Context, route InboundRoute, url string) error
}

// InboundHandshaker answers subscription handshakes on the inbound hook
// (an SNS SubscriptionConfirmation), checked against the route's settings
// rather than the provider's event topic. handled is true when the request
// was a handshake; response, when set, is written back with 200.
type InboundHandshaker interface {
	HandshakeInbound(ctx context.Context, route InboundRoute, r *http.Request, body []byte) (handled bool, response []byte, err error)
}
