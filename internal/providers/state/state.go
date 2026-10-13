// Package state describes what ReputeCast stores about a connected provider
// in the providers table's JSON columns (capability snapshot, health,
// pricing, limits, provider-side tracking) and the display helpers every
// screen shares (short names, chart colours). It has no database access, so
// routing, warm-up, ingest and the providers service can all use it.
//
// These types are a contract other workstreams read (Deliverability,
// Overview, send readiness, the sending engine): fields keep their JSON
// names and meanings and are only ever added. Percentages are plain
// percent values (0.1 means 0.1%), money is US dollars, times are UTC.
//
// A provider whose providers.config holds "demo": true is sample data:
// background jobs never call its API and the sending engine simulates it.
package state

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/store"
)

// Provider statuses (providers.status).
const (
	Healthy   = "healthy"
	Degraded  = "degraded"
	Down      = "down"
	Sandbox   = "sandbox"
	WarmingUp = "warming_up"
	Paused    = "paused"
)

// Snapshot is the capability snapshot stored when a provider is connected:
// the adapter's capabilities plus what this connection adds.
type Snapshot struct {
	providers.Capabilities
	// BroadcastStream is true when the account has a stream on which its
	// terms allow bulk mail (Postmark Broadcast stream).
	BroadcastStream bool `json:"broadcastStream,omitempty"`
}

// BulkAllowed reports whether Broadcasts may go through the provider under
// its terms.
func (s Snapshot) BulkAllowed() bool {
	switch s.Bulk {
	case providers.BulkAllowed, providers.BulkYourPolicy:
		return true
	case providers.BulkBroadcastStreams:
		return s.BroadcastStream
	}
	return false
}

// Caps decodes a provider's capability snapshot, falling back to the
// adapter's declared capabilities.
func Caps(p store.Provider) Snapshot {
	var s Snapshot
	if d, ok := providers.Lookup(p.Adapter); ok {
		s.Capabilities = d.Capabilities
	}
	if len(p.Capabilities) > 0 {
		_ = json.Unmarshal(p.Capabilities, &s)
	}
	if s.Bulk == "" {
		s.Bulk = providers.BulkYourPolicy
	}
	if s.BatchSize == 0 {
		s.BatchSize = 1
	}
	return s
}

// CallRecord is one API call result kept for the "Recent errors" list.
type CallRecord struct {
	At      time.Time `json:"at"`
	Op      string    `json:"op"`
	Status  int       `json:"status,omitempty"`
	Code    string    `json:"code,omitempty"`
	Message string    `json:"message,omitempty"`
}

// Quota is the provider's sending allowance.
type Quota struct {
	// Kind is rolling_24h (SES), monthly_plan (Postmark) or warmup_daily
	// (today's cap of a warm-up plan).
	Kind string `json:"kind"`
	// Max is the allowance in messages; Used is what was sent against it.
	Max  int64 `json:"max"`
	Used int64 `json:"used"`
}

// Rate is the send rate in messages per second: the limit (0 when the
// provider publishes none) and what was used.
type Rate struct {
	Max       float64 `json:"max,omitempty"`
	Now       float64 `json:"now"`
	PeakToday float64 `json:"peakToday"`
	// Label names where the limit comes from ("ramp limit").
	Label string `json:"label,omitempty"`
}

// DownState describes an outage or rejected credentials.
type DownState struct {
	Since  time.Time `json:"since"`
	Reason string    `json:"reason"` // outage, auth
	// NotifiedAt is when members were told (once per outage).
	NotifiedAt *time.Time `json:"notifiedAt,omitempty"`
	// ProviderStatus is what the provider's public status page said.
	ProviderStatus *ProviderStatus `json:"providerStatus,omitempty"`
	// RecoveringSince is when calls started succeeding again; the provider
	// is healthy again after ten healthy minutes.
	RecoveringSince *time.Time `json:"recoveringSince,omitempty"`
}

// ProviderStatus is a public status page reading.
type ProviderStatus struct {
	Incident  bool      `json:"incident"`
	Summary   string    `json:"summary,omitempty"`
	CheckedAt time.Time `json:"checkedAt"`
}

// WebhookCheck is one finding of the webhooks-silent diagnosis.
type WebhookCheck struct {
	Key    string     `json:"key"` // api, registered, endpoint, provider_log
	OK     *bool      `json:"ok,omitempty"`
	At     *time.Time `json:"at,omitempty"`
	Detail string     `json:"detail,omitempty"`
	ID     string     `json:"id,omitempty"`
	URL    string     `json:"url,omitempty"`
	Count  *int64     `json:"count,omitempty"`
	Queued *int64     `json:"queued,omitempty"`
}

// WebhookState is the webhook freshness of a provider.
type WebhookState struct {
	// SilentSince is the last event before a silence of six hours or more.
	SilentSince *time.Time     `json:"silentSince,omitempty"`
	NotifiedAt  *time.Time     `json:"notifiedAt,omitempty"`
	Checks      []WebhookCheck `json:"checks,omitempty"`
	// Causes are likely causes, most likely first (worker_stuck, firewall,
	// secret_rotated, webhook_removed, endpoint_unreachable).
	Causes    []string   `json:"causes,omitempty"`
	CheckedAt *time.Time `json:"checkedAt,omitempty"`
}

// Health is the snapshot the health job writes into providers.health.
type Health struct {
	Quota *Quota `json:"quota,omitempty"`
	Rate  *Rate  `json:"rate,omitempty"`
	// Account is the account standing as the provider reports it
	// ("Production · no enforcement", "Approved · 10,000 per month plan").
	Account string `json:"account,omitempty"`
	// Sandbox is true while the account can only send to verified addresses.
	Sandbox bool `json:"sandbox,omitempty"`
	// ConsecutiveFailures counts failed health checks in a row.
	ConsecutiveFailures int         `json:"consecutiveFailures,omitempty"`
	LastSuccessAt       *time.Time  `json:"lastSuccessAt,omitempty"`
	LastSuccessOp       string      `json:"lastSuccessOp,omitempty"`
	LastError           *CallRecord `json:"lastError,omitempty"`
	// RecentErrors are the last API failures, newest first (at most 10).
	RecentErrors []CallRecord `json:"recentErrors,omitempty"`
	// Down is set while the provider is down (outage or rejected credentials).
	Down *DownState `json:"down,omitempty"`
	// Webhooks is set once the health job has checked webhook freshness.
	Webhooks *WebhookState `json:"webhooks,omitempty"`
	// Version is the provider software version when it reports one
	// (Postal's webhook User-Agent).
	Version string `json:"version,omitempty"`
	// VDM is true when Amazon SES's Virtual Deliverability Manager is on.
	VDM bool `json:"vdm,omitempty"`
	// Reputation is the bounce and complaint rate the provider itself
	// reports for the account, when its API offers one. Screens prefer it
	// to ReputeCast's own count while it is fresh (see ReputationOf).
	Reputation *Reputation `json:"reputation,omitempty"`
}

// Reputation is a provider-reported bounce and complaint rate.
type Reputation struct {
	// Bounce and Complaint are percentages (0.2 means 0.2%).
	Bounce    float64 `json:"bounce"`
	Complaint float64 `json:"complaint"`
	// Window is the period the provider measured ("7 d").
	Window    string    `json:"window,omitempty"`
	CheckedAt time.Time `json:"checkedAt"`
}

// ReputationFresh is how long a provider-reported reputation is used.
const ReputationFresh = 48 * time.Hour

// ReputationOf returns the provider-reported reputation when it is recent
// enough to show instead of ReputeCast's own 7-day count.
func ReputationOf(h Health, now time.Time) *Reputation {
	if h.Reputation == nil || now.Sub(h.Reputation.CheckedAt) > ReputationFresh {
		return nil
	}
	return h.Reputation
}

// HealthOf decodes a provider's health snapshot.
func HealthOf(p store.Provider) Health {
	var h Health
	if len(p.Health) > 0 {
		_ = json.Unmarshal(p.Health, &h)
	}
	return h
}

// Pricing is how a provider is paid for.
type Pricing struct {
	Model string `json:"model"` // metered, plan, self_hosted
	// Per1000 is the metered price per 1,000 emails.
	Per1000 float64 `json:"per1000,omitempty"`
	// Monthly is the plan or server price per month.
	Monthly float64 `json:"monthly,omitempty"`
	// Source explains the per-1,000 figure for metered pricing ("AWS pricing").
	Source string `json:"source,omitempty"`
	// NextTier is the next plan step ("adds 50,000 for $115").
	NextTier *NextTier `json:"nextTier,omitempty"`
}

// NextTier is a provider plan's next step.
type NextTier struct {
	Adds  int64   `json:"adds"`
	Price float64 `json:"price"`
}

// PricingOf decodes a provider's pricing.
func PricingOf(p store.Provider) Pricing {
	var pr Pricing
	if len(p.Pricing) > 0 {
		_ = json.Unmarshal(p.Pricing, &pr)
	}
	if pr.Model == "" {
		pr.Model = "metered"
	}
	return pr
}

// Limit is a rate limit shown against a provider's bounce or complaint rate.
type Limit struct {
	// Limit is the line the bar is drawn against, in percent.
	Limit float64 `json:"limit"`
	// Review and Pause are the provider's own enforcement levels (SES).
	Review float64 `json:"review,omitempty"`
	Pause  float64 `json:"pause,omitempty"`
	// Label names whose limit it is ("SES", "Postmark", "warm-up guard").
	Label string `json:"label,omitempty"`
}

// Limits are a provider's bounce and complaint limits.
type Limits struct {
	Bounce    Limit `json:"bounce"`
	Complaint Limit `json:"complaint"`
}

// LimitsOf decodes a provider's limits with sensible defaults.
func LimitsOf(p store.Provider) Limits {
	l := Limits{Bounce: Limit{Limit: 5}, Complaint: Limit{Limit: 0.1}}
	if len(p.Limits) > 0 {
		_ = json.Unmarshal(p.Limits, &l)
	}
	if l.Bounce.Limit <= 0 {
		l.Bounce.Limit = 5
	}
	if l.Complaint.Limit <= 0 {
		l.Complaint.Limit = 0.1
	}
	return l
}

// Tracking is the provider-side tracking found by the tracking check.
type Tracking struct {
	// OpenPixel and LinkRewrite are true when the provider adds its own open
	// pixel or rewrites links (double tracking with ReputeCast's).
	OpenPixel   bool `json:"openPixel"`
	LinkRewrite bool `json:"linkRewrite"`
	// Detail is what was found ("Open tracking on the default configuration
	// set"); Scope is where (a configuration set, a message stream).
	Detail string `json:"detail,omitempty"`
	Scope  string `json:"scope,omitempty"`
	// Mode is "reputecast" (ours, the default) or "provider" when a person
	// chose to keep the provider's tracking and ignore ReputeCast's.
	Mode      string     `json:"mode,omitempty"`
	CheckedAt *time.Time `json:"checkedAt,omitempty"`
	Error     string     `json:"error,omitempty"`
}

// TrackingOf decodes a provider's tracking state.
func TrackingOf(p store.Provider) Tracking {
	var t Tracking
	if len(p.Tracking) > 0 {
		_ = json.Unmarshal(p.Tracking, &t)
	}
	if t.Mode == "" {
		t.Mode = "reputecast"
	}
	return t
}

// Short is the label used in badges and legends: "SES", "Postal", "Postmark".
func Short(name, adapter string) string {
	switch adapter {
	case "ses":
		return "SES"
	case "azureacs":
		return "Azure"
	}
	if i := strings.Index(name, " ("); i > 0 {
		return name[:i]
	}
	return name
}

// Color is the sending-provider chart colour key ("ses", "postal",
// "postmark") from the design system; other adapters share "other".
func Color(adapter string) string {
	switch adapter {
	case "ses", "postal", "postmark":
		return adapter
	}
	return "other"
}

// WarmupDay is the day number (1-based) of a warm-up plan that started on
// start, on the calendar of loc.
func WarmupDay(start, now time.Time, loc *time.Location) int {
	if loc == nil {
		loc = time.UTC
	}
	s := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	n := now.In(loc)
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, loc)
	return int(today.Sub(s).Hours()/24+0.5) + 1
}

// Location loads a workspace timezone, UTC when unknown.
func Location(tz string) *time.Location {
	if loc, err := time.LoadLocation(tz); err == nil && tz != "" {
		return loc
	}
	return time.UTC
}

// Available reports whether new mail may be routed to the provider.
func Available(p store.Provider) bool {
	switch p.Status {
	case Down, Paused:
		return false
	case Degraded:
		// Webhooks silent: sending works but nothing protects the route.
		h := HealthOf(p)
		return h.Webhooks == nil || h.Webhooks.SilentSince == nil
	}
	return p.DisconnectedAt == nil
}
