package providers

import (
	"context"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Optional interfaces of the connect flow (internal/connect): detailed
// connection tests, provider-side provisioning with per-step progress and
// retry, region discovery, production-access requests and access templates.
// Adapters that implement none of them get the generic flow.
//
// Texts returned to the flow may mark names and values with backticks
// ("Created SNS topic `reputecast-events`"): the UI shows them in monospace.
// Each text may carry the message it comes from beside it (SummaryText,
// LabelText, DoneText), so the flow shows it in the reader's language
// (text.go).

// ConnectionReport is the detailed result of a connection test.
type ConnectionReport struct {
	// Summary is one line for a status badge: "STARTTLS, TLS 1.3 · AUTH PLAIN accepted".
	Summary string
	// Transcript is the protocol exchange without secrets, one entry per
	// line: "220 smtp.mta.acme.io ESMTP Postal", "EHLO reputecast.acme.io"…
	Transcript []string
	// Auth is the login method the server accepted ("plain", "login"), so
	// sends can use it rather than discover it again; empty without login.
	Auth string
	// SummaryText is the message Summary comes from.
	SummaryText i18n.Text
}

// ConnectionTester is implemented by adapters that can describe a connection
// test in detail (SMTP: greeting, extensions, TLS version, authentication).
type ConnectionTester interface {
	TestConnection(ctx context.Context) (ConnectionReport, error)
}

// MailboxReport describes the bounce mailbox an adapter reads.
type MailboxReport struct {
	Protocol string // IMAP, POP3
	Folder   string
	// Messages waiting to be processed, and how many of them look like
	// bounce (DSN) or complaint (ARF) reports.
	Messages int
	Reports  int
}

// BounceMailboxTester is implemented by adapters that read bounces and
// complaints from a mailbox (SMTP). The test never marks or deletes mail.
type BounceMailboxTester interface {
	TestBounceMailbox(ctx context.Context) (MailboxReport, error)
}

// AccountIdentifier reports which provider account the credentials belong
// to (the AWS account id), so people can tell connections apart.
type AccountIdentifier interface {
	AccountID(ctx context.Context) (string, error)
}

// RegionStatus is the account's standing in one region.
type RegionStatus struct {
	Region string // "us-east-1"
	Name   string // "N. Virginia"
	// Production is false while the region is in the provider's sandbox.
	Production bool
	Max24h     int64
	MaxRate    float64
	// Err is set when the region couldn't be read (not enabled, no access).
	Err string
}

// RegionScanner is implemented by adapters whose account standing differs
// per region (SES), so the region step can show where production access
// already exists.
type RegionScanner interface {
	ScanRegions(ctx context.Context) ([]RegionStatus, error)
}

// ProductionAccessRequest asks a provider to take a region out of its
// sandbox (SES PutAccountDetails).
type ProductionAccessRequest struct {
	Region      string
	WebsiteURL  string
	MailType    string // "marketing" or "transactional"
	UseCase     string
	Language    string // "en" or "ja"
	DailyVolume int64
	// ContactEmails receive the provider's answer.
	ContactEmails []string
}

// ProductionAccessRequester is implemented by adapters with a sandbox that a
// request can lift.
type ProductionAccessRequester interface {
	RequestProductionAccess(ctx context.Context, r ProductionAccessRequest) error
}

// ProvisionStep is one provider-side resource the connect flow creates.
type ProvisionStep struct {
	Key string
	// Label says what the step will do ("Create configuration set `rc-system`").
	Label string
	// LabelText is the message Label comes from.
	LabelText i18n.Text
}

// ProvisionInput is what provisioning needs from ReputeCast.
type ProvisionInput struct {
	// WebhookURL is where the provider posts events.
	WebhookURL string
	// Suffix, when set, is appended to the names of created resources
	// because another connection of this instance already uses the plain
	// names in the same account and region.
	Suffix string
	// Events is how events come back: "" or "webhook" (the provider calls
	// WebhookURL) or "queue" (a queue ReputeCast polls, PR-4).
	Events string
	// Tenant names the workspace's own tenant at the provider when it has
	// them (SES tenants, PR-11); empty for none.
	Tenant string
}

// EventsQueue is ProvisionInput.Events for events that come through a
// queue ReputeCast polls.
const EventsQueue = "queue"

// ProvisionResult is the outcome of one step.
type ProvisionResult struct {
	// Done describes what exists now ("Created SNS topic `reputecast-events`
	// and subscribed `https://…`").
	Done string
	// DoneText is the message Done comes from.
	DoneText i18n.Text
	// Settings are merged into the provider's settings (topic ARNs,
	// configuration sets per stream).
	Settings map[string]any
	// Pending is true when the step waits for something outside ReputeCast
	// (the SNS subscription handshake); call Provision again a bit later.
	Pending bool
	// Note qualifies what was done, under it ("This workspace's own
	// suppression list…"); NoteText is its message.
	Note     string
	NoteText i18n.Text
}

// EventQueue is implemented by adapters whose events can come through a
// queue ReputeCast polls instead of webhooks (SES through SQS, PR-4):
// nothing has to reach ReputeCast from the internet.
type EventQueue interface {
	// QueueEnabled reports whether this connection gets its events that way.
	QueueEnabled() bool
	// ReceiveEvents returns up to max queued events, waiting up to wait for
	// the first one (a long poll).
	ReceiveEvents(ctx context.Context, max int, wait time.Duration) ([]QueuedEvent, error)
	// AckEvents removes stored events from the queue.
	AckEvents(ctx context.Context, handles []string) error
}

// QueuedEvent is one event pulled from an event queue, in the format the
// adapter's ParseWebhook reads.
type QueuedEvent struct {
	Body        []byte
	ContentType string
	// Handle acknowledges the event once it is stored.
	Handle string
}

// EventChecker is implemented by adapters with an address that always
// accepts mail and makes the provider report a delivery (SES's mailbox
// simulator): the setup sends there and waits for that event before it
// calls events verified (PR-3).
type EventChecker interface {
	EventCheckAddress() string
	// EventCheckFrom picks an address the provider may send the check from:
	// one of candidates (the workspace's senders) whose address or domain
	// it has verified, else any address of an identity it has verified; ""
	// while none is (the check then waits for domain verification).
	EventCheckFrom(ctx context.Context, candidates []string) (string, error)
}

// Provisioner creates the provider-side resources ReputeCast needs, one
// retryable step at a time (SES: SNS topic and subscription, a
// configuration set per stream).
type Provisioner interface {
	ProvisionSteps(ctx context.Context, in ProvisionInput) []ProvisionStep
	Provision(ctx context.Context, step string, in ProvisionInput) (ProvisionResult, error)
}

// AccessTemplateInput parameterizes an access template.
type AccessTemplateInput struct {
	Region     string
	ExternalID string
	// TemplateURL is where the published template lives; the launch link
	// points the provider's console at it.
	TemplateURL string
}

// AccessTemplate grants ReputeCast access through the provider's own
// console, without long-lived keys (SES: a CloudFormation stack creating an
// IAM role ReputeCast assumes with an external id).
type AccessTemplate struct {
	Format string // "cloudformation"
	// Body is the template itself, for download or review.
	Body string
	// LaunchURL opens the provider's console with the template ready to run.
	LaunchURL string
	// Policy is the least-privilege policy the template grants, for display.
	Policy string
	// Principal is the identity of this server that may assume the role,
	// when it could be detected ("" otherwise: the person fills it in).
	Principal string
	// RoleName is the role the stack creates; its ARN is the stack output.
	RoleName string
}

// AccessTemplater is implemented by adapters whose access can be granted
// with a template in the provider's console.
type AccessTemplater interface {
	AccessTemplate(ctx context.Context, in AccessTemplateInput) (AccessTemplate, error)
}
