package engine

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/ingest"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/providers/simulator"
	"github.com/hostrepute/reputecast/internal/providers/state"
	"github.com/hostrepute/reputecast/internal/publicurl"
	"github.com/hostrepute/reputecast/internal/sending"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/warmup"
)

// Engine runs campaign sends and automation emails. It implements Starter.
type Engine struct {
	Pool    *pgxpool.Pool
	Q       *store.Queries
	Audit   *audit.Logger
	Factory *manage.Factory
	Links   *linktoken.Signer
	URLs    publicurl.Builder
	Jobs    func() *jobs.Client
	Log     *slog.Logger
	Now     func() time.Time
	// Readiness reruns Send Readiness at send time.
	Readiness sending.Readiness
	// Segments resolves segment audiences.
	Segments sending.Segments
	// Normalizer applies the demo transport's delivery events exactly as
	// webhook events are applied.
	Normalizer *ingest.Normalizer
	// Recorder records the demo transport's opens and clicks through the
	// same classification as real pixel fetches and clicks.
	Recorder sending.Recorder
	// Automations receives delivery outcomes of automation emails.
	Automations *automations.Service

	mu       sync.Mutex
	calls    map[routeKey]*callLog
	hubs     map[uuid.UUID]*hub
	running  map[uuid.UUID]bool
	templMu  sync.Mutex
	template map[string]*cachedTemplate
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

func (e *Engine) log() *slog.Logger {
	if e.Log != nil {
		return e.Log
	}
	return slog.Default()
}

func (e *Engine) readiness() sending.Readiness {
	if e.Readiness != nil {
		return e.Readiness
	}
	return sending.ReadyAlways{}
}

func (e *Engine) jobs() *jobs.Client {
	if e.Jobs == nil {
		return nil
	}
	return e.Jobs()
}

// insert enqueues a job when a job client exists (tests drive the engine
// directly and have none).
func (e *Engine) insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) {
	c := e.jobs()
	if c == nil {
		return
	}
	if _, err := c.Insert(ctx, args, opts); err != nil {
		e.log().WarnContext(ctx, "enqueue send job", "kind", args.Kind(), "err", err)
	}
}

func (e *Engine) insertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) error {
	c := e.jobs()
	if c == nil {
		return nil
	}
	_, err := c.InsertTx(ctx, tx, args, opts)
	return err
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// Thresholds are the auto-pause limits of a send (campaigns.thresholds,
// edited on Review & send). Rates are percentages: bounce of sent,
// complaint of delivered, deferral of sent, API errors of calls in the
// last three minutes.
type Thresholds struct {
	Bounce float64 `json:"bounce"`
	// Complaint applies campaign-wide; MailboxComplaint to each mailbox provider.
	Complaint        float64 `json:"complaint"`
	MailboxComplaint float64 `json:"mailbox_complaint"`
	Deferral         float64 `json:"deferral"`
	APIErrors        float64 `json:"api_errors"`
	// Scope is "route" (pause only the affected mailbox provider or sending
	// provider) or "campaign" (pause everything).
	Scope string `json:"scope"`
}

// DefaultThresholds are used for limits a campaign doesn't set.
var DefaultThresholds = Thresholds{Bounce: 2.0, Complaint: 0.1, MailboxComplaint: 0.1, Deferral: 5.0, APIErrors: 5.0, Scope: "route"}

// ParseThresholds reads campaigns.thresholds with defaults.
func ParseThresholds(raw json.RawMessage) Thresholds {
	t := DefaultThresholds
	var in struct {
		Bounce           *float64 `json:"bounce"`
		Complaint        *float64 `json:"complaint"`
		MailboxComplaint *float64 `json:"mailbox_complaint"`
		Deferral         *float64 `json:"deferral"`
		APIErrors        *float64 `json:"api_errors"`
		Scope            string   `json:"scope"`
	}
	if len(raw) > 0 && json.Unmarshal(raw, &in) == nil {
		set := func(dst *float64, v *float64) {
			if v != nil && *v > 0 {
				*dst = *v
			}
		}
		set(&t.Bounce, in.Bounce)
		set(&t.Complaint, in.Complaint)
		t.MailboxComplaint = t.Complaint
		set(&t.MailboxComplaint, in.MailboxComplaint)
		set(&t.Deferral, in.Deferral)
		set(&t.APIErrors, in.APIErrors)
		if in.Scope == "campaign" || in.Scope == "route" {
			t.Scope = in.Scope
		}
	}
	return t
}

// DefaultThrottles are messages per second per mailbox provider when Send
// Readiness supplies none: Outlook, Yahoo and iCloud throttle new and bulk
// senders, Gmail and the rest take the provider's full rate.
var DefaultThrottles = map[string]float64{"outlook": 5, "yahoo": 4, "icloud": 3}

// Sample sizes before a rate can trip a threshold, and the least number of
// events, so one complaint among the first few hundred never pauses a send.
const (
	minSampleCampaign = 1000
	minSampleScoped   = 500
	minEvents         = 3
	minDeferrals      = 5
)

// Retry back-off for temporary errors: 5 s doubling to 60 s, at most
// maxAttempts provider calls per message.
const (
	backoffMin  = 5 * time.Second
	backoffMax  = 60 * time.Second
	maxAttempts = 10
	staleClaim  = 5 * time.Minute
)

func backoff(attempts int) time.Duration {
	d := backoffMin << max(attempts-1, 0)
	if d > backoffMax || d <= 0 {
		return backoffMax
	}
	return d
}

// Outage rule: a route is held after its provider failed every call for
// three minutes (at least outageMinCalls calls), or W01's health job marked
// the provider down.
const (
	outageAfter    = 3 * time.Minute
	outageMinCalls = 5
	retryEvery     = 60 * time.Second
	retryFor       = 15 * time.Minute
)

// defaultRate is a provider's messages per second when it publishes none.
func defaultRate(adapter string) float64 {
	switch adapter {
	case "ses":
		return 14
	case "postmark", "mailgun", "sendgrid", "sparkpost", "resend", "mailjet", "brevo":
		return 50
	case "smtp":
		return 10
	case "postal":
		return 20
	}
	return 20
}

// providerRate is how fast a provider may send right now and what limits
// it: the warm-up plan's ramp, the account's maximum rate, or a default.
type providerRate struct {
	PerSecond float64
	// Label names a warm-up limit ("warm-up day 9").
	WarmupDay, WarmupDays int
	Cap, SentToday        int64
	Tiers                 []string
	WarmingUp             bool
	Paused                bool
}

func (e *Engine) rateOf(ctx context.Context, q *store.Queries, p store.Provider, loc *time.Location, now time.Time) providerRate {
	r := providerRate{PerSecond: defaultRate(p.Adapter)}
	h := state.HealthOf(p)
	if h.Rate != nil && h.Rate.Max > 0 {
		r.PerSecond = h.Rate.Max
	}
	if a, ok, err := warmup.AllowanceFor(ctx, q, p.ID, now, loc); err == nil && ok {
		r.WarmingUp = true
		r.WarmupDay, r.WarmupDays, r.Cap, r.SentToday, r.Tiers, r.Paused = a.Day, a.Days, a.Cap, a.Sent, a.Tiers, a.Paused
		if a.Rate > 0 {
			r.PerSecond = math.Min(r.PerSecond, a.Rate)
		}
	}
	return r
}

// batchSize is how many messages one Send call carries for a provider; the
// demo transport uses the size of the adapter it stands in for.
func batchSize(p store.Provider) int {
	if d, ok := providers.Lookup(p.Adapter); ok && d.Capabilities.BatchSize > 0 {
		return d.Capabilities.BatchSize
	}
	return max(state.Caps(p).BatchSize, 1)
}

// transport opens the provider's adapter, or the demo transport for
// providers whose config holds "demo": true (never for any other).
func (e *Engine) transport(ctx context.Context, p store.Provider, scenario string, started time.Time) (providers.Provider, bool, error) {
	if manage.IsDemo(p) {
		var cfg map[string]any
		_ = json.Unmarshal(p.Config, &cfg)
		if cfg == nil {
			cfg = map[string]any{}
		}
		cfg["mimic"] = p.Adapter
		cfg["scenario"] = scenario
		if !started.IsZero() {
			cfg["started"] = started
		}
		if len(p.Ips) > 0 {
			cfg["ip"] = p.Ips[0]
		}
		b, _ := json.Marshal(cfg)
		sim, err := simulator.New(providers.Config{ProviderID: p.ID, WorkspaceID: p.WorkspaceID, Settings: b})
		if err != nil {
			return nil, true, err
		}
		if s, ok := sim.(*simulator.Provider); ok && e.Now != nil {
			s.WithClock(e.Now)
		}
		return sim, true, nil
	}
	if e.Factory == nil {
		return nil, false, sending.ErrUnavailable
	}
	t, err := e.Factory.Open(ctx, p)
	return t, false, err
}

// routeKey identifies a campaign's share on one provider (uuid.Nil campaign
// for the automation outbox).
type routeKey struct {
	campaign uuid.UUID
	provider uuid.UUID
}

// callLog remembers a route's recent provider calls for outage detection
// and the "What Postal returned" list.
type callLog struct {
	firstFailure time.Time
	lastSuccess  time.Time
	failures     int
	lines        []IncidentLogLine
}

type IncidentLogLine struct {
	At   time.Time `json:"at"`
	Text string    `json:"text"`
	Bad  bool      `json:"bad"`
}

func (e *Engine) callLogFor(k routeKey) *callLog {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.calls == nil {
		e.calls = map[routeKey]*callLog{}
	}
	l := e.calls[k]
	if l == nil {
		l = &callLog{}
		e.calls[k] = l
	}
	return l
}

func (e *Engine) recordCall(k routeKey, at time.Time, err error) {
	l := e.callLogFor(k)
	e.mu.Lock()
	defer e.mu.Unlock()
	if err == nil {
		l.lastSuccess, l.firstFailure, l.failures = at, time.Time{}, 0
		return
	}
	// Throttling is the provider pacing us, not an outage: the batch backs
	// off and the call doesn't count toward holding the route.
	var ce *providers.CallError
	if asCallError(err, &ce) && ce.Throttled {
		return
	}
	if l.firstFailure.IsZero() {
		l.firstFailure = at
	}
	l.failures++
	text := callText(err)
	if n := len(l.lines); n == 0 || l.lines[n-1].Text != text || at.Sub(l.lines[n-1].At) > 30*time.Second {
		l.lines = append(l.lines, IncidentLogLine{At: at, Text: text, Bad: true})
		if len(l.lines) > 12 {
			l.lines = l.lines[len(l.lines)-12:]
		}
	}
}

func (e *Engine) resetCalls(k routeKey) {
	e.mu.Lock()
	delete(e.calls, k)
	e.mu.Unlock()
}

func callText(err error) string {
	var ce *providers.CallError
	if ok := asCallError(err, &ce); ok {
		switch {
		case ce.Status > 0 && ce.Message != "" && strings.HasPrefix(ce.Message, "HTTP"):
			return ce.Message
		case ce.Status > 0:
			s := "HTTP " + itoa(ce.Status)
			if ce.Code != "" {
				s += " " + ce.Code
			}
			if ce.Op != "" {
				s += " · " + ce.Op
			}
			return s
		case ce.Message != "":
			return ce.Message
		}
		return ce.Error()
	}
	return err.Error()
}
