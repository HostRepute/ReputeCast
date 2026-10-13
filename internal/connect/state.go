package connect

import (
	"encoding/json"
	"time"
)

// State is provider_setups.state: what each step of the setup found and
// chose, so a reload (or a teammate) resumes exactly where it was. Secrets
// never go here; they are sealed on the provider row once accepted.
type State struct {
	Connection Connection    `json:"connection"`
	Domain     *DomainStep   `json:"domain,omitempty"`
	Test       *TestStep     `json:"test,omitempty"`
	Streams    *StreamsStep  `json:"streams,omitempty"`
	Contacts   *ContactsStep `json:"contacts,omitempty"`
}

// Connection is step 2.
type Connection struct {
	// Status: "" (nothing yet), connected, failed.
	Status string     `json:"status,omitempty"`
	At     *time.Time `json:"at,omitempty"`
	// Settings are the non-secret values entered (to refill the form).
	Settings map[string]string `json:"settings,omitempty"`
	// Account is the provider's account standing ("Production · no enforcement").
	Account   string `json:"account,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	// Summary is the stepper's line once connected ("us-east-1 · production").
	Summary string     `json:"summary,omitempty"`
	Error   *StepError `json:"error,omitempty"`

	// Amazon SES
	Access           string            `json:"access,omitempty"` // role, keys
	ExternalID       string            `json:"external_id,omitempty"`
	RoleARN          string            `json:"role_arn,omitempty"`
	Region           string            `json:"region,omitempty"`
	Regions          []RegionState     `json:"regions,omitempty"`
	Provisioning     []ProvisionState  `json:"provisioning,omitempty"`
	ProvisionSuffix  string            `json:"provision_suffix,omitempty"`
	ProductionAccess *ProductionAccess `json:"production_access,omitempty"`

	// SMTP
	SMTP    *ProbeState `json:"smtp,omitempty"`
	Mailbox *ProbeState `json:"mailbox,omitempty"`

	// Events
	Webhook *WebhookState `json:"webhook,omitempty"`
}

// StepError is why a step failed, for the person.
type StepError struct {
	// Field is the form field to mark ("api_key"), empty for the step.
	Field   string `json:"field,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	// Detail is the provider's own answer ("401 Unauthorized · Forbidden").
	Detail string `json:"detail,omitempty"`
}

// RegionState is the account in one region.
type RegionState struct {
	Region     string  `json:"region"`
	Name       string  `json:"name"`
	Production bool    `json:"production"`
	Max24h     int64   `json:"max_24h,omitempty"`
	MaxRate    float64 `json:"max_rate,omitempty"`
	Err        string  `json:"err,omitempty"`
}

// ProvisionState is one provisioning row.
type ProvisionState struct {
	Key string `json:"key"`
	// Label is what the step does; Text what was done (`backticks` = code).
	Label  string     `json:"label"`
	Text   string     `json:"text,omitempty"`
	Status string     `json:"status"` // waiting, running, done, failed
	At     *time.Time `json:"at,omitempty"`
	// Error is the provider's error ("AccessDenied: ses:CreateConfigurationSet"),
	// Hint the likely cause.
	Error string `json:"error,omitempty"`
	Hint  string `json:"hint,omitempty"`
	// Attempts counts runs of a step that waits (SNS confirmation).
	Attempts int `json:"attempts,omitempty"`
}

// ProductionAccess is a filed request to leave the sandbox.
type ProductionAccess struct {
	Region      string    `json:"region"`
	RequestedAt time.Time `json:"requested_at"`
	Website     string    `json:"website"`
	MailType    string    `json:"mail_type"`
	UseCase     string    `json:"use_case"`
	Language    string    `json:"language"`
	DailyVolume int64     `json:"daily_volume"`
}

// ProbeState is the result of a connection or mailbox test.
type ProbeState struct {
	OK         bool      `json:"ok"`
	At         time.Time `json:"at"`
	Summary    string    `json:"summary,omitempty"`
	Transcript []string  `json:"transcript,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// WebhookState is how events come back.
type WebhookState struct {
	// Mode: webhook, polling (the provider's events API), mailbox (SMTP
	// bounce mailbox), none.
	Mode string `json:"mode"`
	URL  string `json:"url,omitempty"`
	// Token is the webhook token reserved for the provider the setup
	// connects, so the URL is known before the key works.
	Token string `json:"token,omitempty"`
	// NeedsSecret: the webhook is set up by hand in the provider and
	// authenticated with the secret ReputeCast generated at connect time,
	// sent in SecretHeader (or used to sign when that is empty).
	NeedsSecret  bool   `json:"needs_secret,omitempty"`
	SecretHeader string `json:"secret_header,omitempty"`
	// PublicURL overrides the instance's public URL for this provider (a
	// tunnel while testing).
	PublicURL    string         `json:"public_url,omitempty"`
	SelfTest     *SelfTestState `json:"self_test,omitempty"`
	Registration *Registration  `json:"registration,omitempty"`
}

// SelfTestState is the endpoint self-test.
type SelfTestState struct {
	OK bool      `json:"ok"`
	At time.Time `json:"at"`
	// Code is the error class ("ECONNREFUSED", "HTTP 404"); Reason explains
	// it ("DNS resolves to 10.0.4.18, a private address").
	Code   string `json:"code,omitempty"`
	Reason string `json:"reason,omitempty"`
	// Private is true when the public URL can't be reached from the
	// internet at all (localhost, a private address).
	Private bool `json:"private,omitempty"`
}

// Registration is the webhook registration at the provider.
type Registration struct {
	// Status: done, skipped, failed, manual (set up in the provider's dashboard).
	Status string    `json:"status"`
	At     time.Time `json:"at"`
	Detail string    `json:"detail,omitempty"`
}

// DomainStep is step 3.
type DomainStep struct {
	DomainID string     `json:"domain_id,omitempty"`
	Name     string     `json:"name"`
	Skipped  bool       `json:"skipped,omitempty"`
	Verified int        `json:"verified"`
	Total    int        `json:"total"`
	At       *time.Time `json:"at,omitempty"`
}

// TestStep is step 4.
type TestStep struct {
	MessageID string    `json:"message_id"`
	SenderID  string    `json:"sender_id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	QueuedAt  time.Time `json:"queued_at"`
	// RoundTripMs is queue to delivery event, once delivered.
	RoundTripMs int64 `json:"round_trip_ms,omitempty"`
	// SizeBytes is the message's approximate size (Gmail clips at 102 KB).
	SizeBytes int `json:"size_bytes,omitempty"`
	// Seeds are the seed-list addresses of the last seed test.
	Seeds []string `json:"seeds,omitempty"`
	// FirstEvent is the provider's first webhook about the message, kept
	// once seen: raw webhook payloads are swept after 90 days.
	FirstEvent *TestEvent `json:"first_event,omitempty"`
}

// TestEvent is a provider event the test timeline shows.
type TestEvent struct {
	At time.Time `json:"at"`
	// Name is the provider's event name ("Send"), Via how it came ("SNS").
	Name string `json:"name,omitempty"`
	Via  string `json:"via"`
}

// StreamChoice is one stream of step 5.
type StreamChoice struct {
	Enabled bool   `json:"enabled"`
	Role    string `json:"role,omitempty"` // primary, split, fallback
	Weight  int    `json:"weight,omitempty"`
}

// StreamsStep is step 5.
type StreamsStep struct {
	Saved   bool                    `json:"saved"`
	At      *time.Time              `json:"at,omitempty"`
	Streams map[string]StreamChoice `json:"streams"`
}

// ContactsStep is step 6.
type ContactsStep struct {
	Suppressions *SuppressionPreview `json:"suppressions,omitempty"`
	// Choice: import (an importer source), empty (start empty).
	Choice string `json:"choice,omitempty"`
	Source string `json:"source,omitempty"`
	Pull   bool   `json:"pull"`
	// ListsCreated are lists created by "Start empty".
	ListsCreated []string `json:"lists_created,omitempty"`
	// Imported is how many suppressions the pull added.
	Imported *int64     `json:"imported,omitempty"`
	At       *time.Time `json:"at,omitempty"`
}

// SuppressionPreview is the provider's own suppression list.
type SuppressionPreview struct {
	Available  bool      `json:"available"`
	Total      int       `json:"total"`
	Bounces    int       `json:"bounces"`
	Complaints int       `json:"complaints"`
	Error      string    `json:"error,omitempty"`
	At         time.Time `json:"at"`
}

func decodeState(raw json.RawMessage) State {
	var s State
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &s)
	}
	return s
}

func (s State) encode() json.RawMessage {
	b, _ := json.Marshal(s)
	return b
}
