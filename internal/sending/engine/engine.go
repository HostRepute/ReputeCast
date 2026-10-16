// Package engine is ReputeCast's sending engine (spec 5.3, 7.6): it builds
// a campaign's recipient list, routes, renders and hands every message to
// its sending provider within rate limits and throttles, watches bounce,
// complaint and outage signals and pauses what they concern, sends
// automation emails, and serves the live monitor's snapshot and stream.
// Providers marked "demo": true go through the simulated transport.
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
	Segments  sending.Segments
	// Normalizer applies the demo transport's delivery events exactly as
	// webhook events are applied.
	Normalizer *ingest.Normalizer
	// Recorder records the demo transport's opens and clicks through the
	// same classification as real pixel fetches and clicks.
	Recorder sending.Recorder
	// Automations receives delivery outcomes of automation emails.
	Automations *automations.Service
	// Languages supplies the language versions each reader picks from;
	// nil sends the source to everyone.
	Languages LanguageVersions
	// Conditional decides each reader's conditional blocks and fills in
	// poll and survey links (conditional.go); nil sends every block.
	Conditional Conditional
	// QuietDemo makes the demo transport report no delivery or engagement
	// events (throughput measurements of the send path alone).
	QuietDemo bool

	mu       sync.Mutex
	calls    map[routeKey]*callLog
	hubs     map[uuid.UUID]*hub
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

// ParseThresholds resolves a send's thresholds: the defaults, then the
// workspace's (workspaces.send_thresholds, Settings › Providers), then the
// campaign's own (campaigns.thresholds, Review & send). Each level may set
// any limit; the spellings of both screens are read (hard_bounce or
// bounce; provider_error_rate, provider_errors or api_errors). The
// per-mailbox complaint limit follows the overall one unless set.
func ParseThresholds(campaign, workspace json.RawMessage) Thresholds {
	t := DefaultThresholds
	mailbox := false
	for _, raw := range []json.RawMessage{workspace, campaign} {
		var in struct {
			Bounce            *float64 `json:"bounce"`
			HardBounce        *float64 `json:"hard_bounce"`
			Complaint         *float64 `json:"complaint"`
			MailboxComplaint  *float64 `json:"mailbox_complaint"`
			Deferral          *float64 `json:"deferral"`
			APIErrors         *float64 `json:"api_errors"`
			ProviderErrors    *float64 `json:"provider_errors"`
			ProviderErrorRate *float64 `json:"provider_error_rate"`
			Scope             string   `json:"scope"`
		}
		if len(raw) == 0 || json.Unmarshal(raw, &in) != nil {
			continue
		}
		set := func(dst *float64, vs ...*float64) bool {
			ok := false
			for _, v := range vs {
				if v != nil && *v > 0 {
					*dst, ok = *v, true
				}
			}
			return ok
		}
		set(&t.Bounce, in.HardBounce, in.Bounce)
		set(&t.Complaint, in.Complaint)
		mailbox = set(&t.MailboxComplaint, in.MailboxComplaint) || mailbox
		set(&t.Deferral, in.Deferral)
		set(&t.APIErrors, in.ProviderErrorRate, in.ProviderErrors, in.APIErrors)
		if in.Scope == "campaign" || in.Scope == "route" {
			t.Scope = in.Scope
		}
	}
	if !mailbox {
		t.MailboxComplaint = t.Complaint
	}
	return t
}

// sendThrottles are the throttles of a send: the defaults, then the
// workspace's (sending_defaults.throttles, Settings › Providers), then what
// Send Readiness computed for it. 0 means no limit.
func sendThrottles(sendingDefaults json.RawMessage, readiness map[string]float64) map[string]float64 {
	out := map[string]float64{}
	for k, v := range DefaultThrottles {
		out[k] = v
	}
	var sd struct {
		Throttles map[string]float64 `json:"throttles"`
	}
	_ = json.Unmarshal(sendingDefaults, &sd)
	for _, layer := range []map[string]float64{sd.Throttles, readiness} {
		for k, v := range layer {
			if v > 0 {
				out[k] = v
			} else {
				delete(out, k)
			}
		}
	}
	return out
}

// DefaultThrottles are messages per second per mailbox provider when Send
// Readiness supplies none: Outlook, Yahoo and iCloud throttle new and bulk
// senders, Gmail and the rest take the provider's full rate.
var DefaultThrottles = map[string]float64{"outlook": 5, "yahoo": 4, "icloud": 3}

// Sample sizes before a rate can trip a threshold, and the least number of
// events, so a few complaints among the first few hundred never pause a
// send (below five events the normal approximation in over is too kind).
const (
	minSampleCampaign = 1000
	minSampleScoped   = 500
	minEvents         = 5
	minDeferrals      = 5
)

// zTrip is how sure a trip must be that the rate is really over its limit:
// about 90% (one-sided, Poisson approximation).
const zTrip = 1.28

// over reports whether events out of base put a rate over its limit (a
// percentage) with enough evidence: at least minEv events, the observed
// rate above the limit, and (events − expected) / √expected ≥ zTrip where
// expected is what the limit allows. Three complaints among 1,900 delivered
// (0.16%) don't pause a send, nor do 3 among the first 1,284; 7 among 3,980
// (0.18%, the LiveMonitorPaused board) do.
func over(events, base int64, limit float64, minEv int64) bool {
	if events < minEv || base <= 0 || limit <= 0 {
		return false
	}
	expected := float64(base) * limit / 100
	if float64(events) <= expected {
		return false
	}
	return (float64(events)-expected)/math.Sqrt(expected) >= zTrip
}

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
// three minutes (at least outageMinCalls calls), or the provider health job
// marked the provider down.
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
	PerSecond             float64
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
	lastFailure  time.Time
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
		l.lastSuccess, l.firstFailure, l.lastFailure, l.failures = at, time.Time{}, time.Time{}, 0
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
	l.lastFailure = at
	l.failures++
	// The log summarises: a line when the answer changes, and once more a
	// minute later when the provider keeps answering the same way.
	text := callText(err)
	n := len(l.lines)
	switch {
	case n == 0 || l.lines[n-1].Text != text:
	case at.Sub(l.lines[n-1].At) >= time.Minute && (n < 2 || l.lines[n-2].Text != text):
	default:
		return
	}
	l.lines = append(l.lines, IncidentLogLine{At: at, Text: text, Bad: true})
	if len(l.lines) > 12 {
		l.lines = l.lines[len(l.lines)-12:]
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
