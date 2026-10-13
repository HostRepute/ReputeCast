package providers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// This file extends the contract in provider.go with optional interfaces and
// helpers. Everything here is additive: adapters that don't implement an
// interface keep working, and core code type-asserts before using one.

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
