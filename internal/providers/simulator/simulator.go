// Package simulator is the demo transport: a sending provider that touches
// no network. It accepts messages like the provider it stands in for (SES,
// Postal, Postmark, SMTP), answers with that provider's codes, and plans the
// delivery and engagement events the real one would report over time, at
// the per-mailbox-provider rates of the design boards: delivered, deferred
// (Yahoo TSS04), hard and soft bounces, complaints, Apple Mail Privacy
// Protection prefetches, human opens and clicks. Scenarios reproduce the
// guarded-sending states: "outlook-complaints" raises complaints among
// never-engaged Outlook recipients until the auto-pause trips, and
// "postal-outage" makes a Postal server refuse connections 14 minutes into
// the send.
//
// The sending engine selects it only for providers whose providers.config
// holds "demo": true (set by the demo seeder; the API never sets it). It is
// not in the public adapter catalog, so it can't be connected.
package simulator

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
)

// Adapter is the transport's key.
const Adapter = "simulator"

// Scenarios.
const (
	ScenarioNormal            = "normal"
	ScenarioOutlookComplaints = "outlook-complaints"
	ScenarioPostalOutage      = "postal-outage"
)

// Scenarios lists the valid scenario names.
var Scenarios = []string{ScenarioNormal, ScenarioOutlookComplaints, ScenarioPostalOutage}

// Outage timing of the postal-outage scenario, measured from the send's start.
const (
	OutageAfter = 14 * time.Minute
	OutageLasts = 25 * time.Minute
)

// Hint tags the engine adds to messages for the simulator only (never sent
// to a real provider): the recipient's engagement tier, whether they ever
// opened, how they joined and the mailbox provider.
const (
	TagTier    = "rc_sim_tier"
	TagEngaged = "rc_sim_engaged"
	TagSource  = "rc_sim_source"
	TagMailbox = "rc_sim_mailbox"
)

// Settings are the simulator's options, read from the provider's config
// plus what the engine adds for one send.
type Settings struct {
	Demo bool `json:"demo"`
	// Mimic is the adapter whose answers the simulator gives.
	Mimic    string `json:"mimic"`
	Scenario string `json:"scenario"`
	// Started is when the campaign send started (scenario timing).
	Started time.Time `json:"started"`
	// Host names the server in error messages ("mta.acme.io").
	Host string `json:"server_url"`
	IP   string `json:"ip"`
}

// Provider is the simulated transport.
type Provider struct {
	s      Settings
	secret string
	now    func() time.Time
}

// New builds the simulator from a provider's configuration.
func New(cfg providers.Config) (providers.Provider, error) {
	var s Settings
	if len(cfg.Settings) > 0 {
		if err := json.Unmarshal(cfg.Settings, &s); err != nil {
			return nil, fmt.Errorf("simulator settings: %w", err)
		}
	}
	if s.Mimic == "" {
		s.Mimic = "ses"
	}
	if s.Scenario == "" {
		s.Scenario = ScenarioNormal
	}
	s.Host = strings.TrimPrefix(strings.TrimPrefix(s.Host, "https://"), "http://")
	return &Provider{s: s, secret: cfg.WebhookSecret, now: time.Now}, nil
}

// WithClock replaces the clock (tests and fast-forwarded demo sends).
func (p *Provider) WithClock(now func() time.Time) *Provider {
	p.now = now
	return p
}

type nowKey struct{}

// WithNow carries the sending engine's clock to Send, so a send driven on a
// virtual clock (tests, a fast-forwarded demo send) sees scenario timing on
// that clock.
func WithNow(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, nowKey{}, t)
}

func (p *Provider) clock(ctx context.Context) time.Time {
	if t, ok := ctx.Value(nowKey{}).(time.Time); ok {
		return t
	}
	return p.now()
}

// Descriptor describes the simulator for tests that need it registered.
func Descriptor() providers.Descriptor {
	return providers.Descriptor{
		Adapter: Adapter, Name: "Demo simulator", Kind: "simulator", Version: "1.0.0", Maintainer: "ReputeCast",
		Capabilities: (&Provider{}).Capabilities(),
		New:          New,
	}
}

// Adapter implements providers.Provider.
func (p *Provider) Adapter() string { return Adapter }

// Capabilities implements providers.Provider.
func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{Bulk: providers.BulkYourPolicy, Webhooks: "Simulated", BatchSize: 50, Pricing: "simulated, free"}
}

// Outage reports whether the scenario has the provider down at t.
func (p *Provider) Outage(t time.Time) bool {
	if p.s.Scenario != ScenarioPostalOutage || p.s.Mimic != "postal" || p.s.Started.IsZero() {
		return false
	}
	from := p.s.Started.Add(OutageAfter)
	return !t.Before(from) && t.Before(from.Add(OutageLasts))
}

// Send implements providers.Provider: every message is accepted unless the
// scenario has the provider down, or (rarely) the provider throttles.
func (p *Provider) Send(ctx context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	now := p.clock(ctx)
	if p.Outage(now) {
		return nil, p.outageError(now)
	}
	if len(msgs) > 0 && p.throttled(msgs[0].ID, now) {
		return nil, &providers.CallError{Op: p.op(), Status: 429, Code: "Throttling", Message: "Maximum sending rate exceeded.", Temporary: true, Throttled: true}
	}
	out := make([]providers.SendResult, len(msgs))
	for i, m := range msgs {
		out[i] = providers.SendResult{MessageID: m.ID, ProviderMessageID: p.providerID(m.ID)}
	}
	return out, nil
}

func (p *Provider) op() string {
	switch p.s.Mimic {
	case "postal":
		return "POST /api/v1/send/message"
	case "postmark":
		return "POST /email/batch"
	case "smtp":
		return "SMTP DATA"
	}
	return "SendEmail"
}

func (p *Provider) host() string {
	if p.s.Host != "" {
		return p.s.Host
	}
	return "mta.acme.io"
}

// outageError alternates the two failures a Postal server behind a proxy
// gives when its web process is down: refused connections and 502s.
func (p *Provider) outageError(now time.Time) error {
	if (now.Unix()/30)%3 == 1 {
		return &providers.CallError{Op: p.op(), Status: 502, Code: "Bad Gateway", Message: "HTTP 502 Bad Gateway · /api/v1/send/message", Temporary: true}
	}
	return &providers.CallError{Op: p.op(), Code: "ECONNREFUSED", Message: "ECONNREFUSED " + p.host() + ":443", Temporary: true}
}

// throttled is true for about one call in a thousand (SES) or 250 (Postal),
// the API error rates on the boards. The draw depends on the call's time
// too, so a retry of the same message isn't throttled again for sure.
func (p *Provider) throttled(id uuid.UUID, now time.Time) bool {
	h := binary.BigEndian.Uint32(id[12:16]) ^ uint32(now.UnixNano()/int64(time.Millisecond))*2654435761
	r := float64(h) / float64(math.MaxUint32)
	limit := 0.001
	if p.s.Mimic == "postal" {
		limit = 0.004
	}
	return r < limit
}

func (p *Provider) providerID(id uuid.UUID) string {
	switch p.s.Mimic {
	case "ses":
		return "0100018f" + strings.ReplaceAll(id.String(), "-", "")[8:] + "-000000"
	case "postal":
		return "postal-" + id.String()
	}
	return id.String()
}

// ParseWebhook implements providers.Provider. The simulator never receives
// webhooks in the app (its events are applied directly); the format exists
// so it passes the shared contract suite: a JSON array of events signed
// with HMAC-SHA256 of the body in X-Simulator-Signature.
func (p *Provider) ParseWebhook(ctx context.Context, r *http.Request, body []byte) ([]providers.Event, error) {
	if !providers.SignatureVerified(ctx) {
		mac := hmac.New(sha256.New, []byte(p.secret))
		mac.Write(body)
		want := hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(want), []byte(r.Header.Get("X-Simulator-Signature"))) {
			return nil, providers.ErrBadSignature
		}
	}
	var in []struct {
		Type      string    `json:"type"`
		Recipient string    `json:"recipient"`
		MessageID string    `json:"provider_message_id"`
		At        time.Time `json:"at"`
		Code      string    `json:"code"`
		Class     string    `json:"class"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, fmt.Errorf("simulator payload: %w", err)
	}
	out := make([]providers.Event, 0, len(in))
	for _, e := range in {
		out = append(out, providers.Event{Type: providers.EventType(e.Type), Recipient: e.Recipient, ProviderMessageID: e.MessageID,
			OccurredAt: e.At, OriginalType: "Simulated/" + e.Type, OriginalCode: e.Code, BounceClass: e.Class, Raw: json.RawMessage(body)})
	}
	return out, nil
}

// Sign signs a webhook body (contract tests).
func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Event is something the simulated provider (or the recipient) will do
// after a message was accepted.
type Event struct {
	// Kind: delivered, deferred, hard_bounce, soft_bounce, complaint,
	// open, click.
	Kind string
	// After is the delay from acceptance.
	After        time.Duration
	OriginalType string
	Code         string
	Diagnostic   string
	BounceClass  string
	// Opens and clicks: who fetched the pixel or followed the link.
	UserAgent string
	IP        string
	// Link picks the clicked link (0–1, weighted towards the first links).
	Link float64
}

// mailboxRates are the per-mailbox-provider outcomes of the LiveMonitor
// board, in percent: delivered share is what remains.
type mailboxRates struct {
	bounce, complaint, deferral, opens, machine, clicks float64
}

var rates = map[string]mailboxRates{
	"gmail":   {bounce: 0.4, complaint: 0.02, deferral: 0.2, opens: 15.0, machine: 1.0, clicks: 2.4},
	"outlook": {bounce: 1.1, complaint: 0.04, deferral: 0.4, opens: 10.6, machine: 0.6, clicks: 1.6},
	"yahoo":   {bounce: 0.7, complaint: 0.03, deferral: 2.0, opens: 13.0, machine: 0.8, clicks: 1.9},
	"icloud":  {bounce: 0.4, complaint: 0.0, deferral: 0.2, opens: 4.0, machine: 52.0, clicks: 1.2},
	"other":   {bounce: 0.8, complaint: 0.03, deferral: 0.4, opens: 11.0, machine: 1.0, clicks: 2.0},
}

// rng is a deterministic generator per message, so a re-run of the same
// send reports the same outcomes.
func rng(id uuid.UUID, salt uint64) *rand.Rand {
	return rand.New(rand.NewPCG(binary.BigEndian.Uint64(id[:8])^salt, binary.BigEndian.Uint64(id[8:]))) //nolint:gosec // simulation, not security
}

// Plan returns what will happen to an accepted message.
func (p *Provider) Plan(m providers.Message) []Event {
	mbp := m.Tags[TagMailbox]
	if mbp == "" {
		mbp = "other"
	}
	rt, ok := rates[mbp]
	if !ok {
		rt = rates["other"]
	}
	r := rng(m.ID, 0x9e3779b97f4a7c15)
	pct := func() float64 { return r.Float64() * 100 }
	sec := func(lo, hi float64) time.Duration {
		return time.Duration((lo + r.Float64()*(hi-lo)) * float64(time.Second))
	}
	engaged := m.Tags[TagEngaged] == "true"
	tier := m.Tags[TagTier]
	complaint := rt.complaint
	if p.s.Scenario == ScenarioOutlookComplaints && mbp == "outlook" {
		// Never-engaged or long-unengaged Outlook recipients mark the
		// email as spam far more often.
		if !engaged || tier == "at_risk" || tier == "sunset" {
			complaint = 0.9
		} else {
			complaint = 0.03
		}
	}
	var out []Event
	deliveredAt := sec(0.6, 6)
	switch roll := pct(); {
	case roll < rt.bounce:
		if r.Float64() < 0.8 {
			out = append(out, p.hardBounce(mbp, sec(1, 9), r))
			return out
		}
		out = append(out, p.softBounce(mbp, sec(20, 240)))
		return out
	case roll < rt.bounce+rt.deferral:
		def := sec(1, 8)
		out = append(out, p.deferral(mbp, def))
		deliveredAt = def + 5*time.Minute + sec(0, 40)
	}
	out = append(out, Event{Kind: "delivered", After: deliveredAt, OriginalType: p.deliveredType(), Code: p.deliveredCode()})
	if pct() < complaint {
		out = append(out, Event{Kind: "complaint", After: deliveredAt + sec(60, 360), OriginalType: p.complaintType(), Code: "abuse", Diagnostic: "Feedback loop report"})
	}
	machine := rt.machine
	if mbp == "icloud" && !engaged {
		machine *= 0.8
	}
	if pct() < machine {
		out = append(out, Event{Kind: "open", After: deliveredAt + sec(2, 50), UserAgent: "Mozilla/5.0", IP: fmt.Sprintf("17.58.%d.%d", r.IntN(250)+1, r.IntN(250)+1)})
	}
	opens := rt.opens
	if !engaged {
		opens *= 0.25
	}
	if pct() < opens {
		openAt := deliveredAt + humanDelay(r)
		ua := humanAgent(mbp, r)
		out = append(out, Event{Kind: "open", After: openAt, UserAgent: ua, IP: fmt.Sprintf("%d.%d.%d.%d", 24+r.IntN(170), r.IntN(250), r.IntN(250), r.IntN(250)+1)})
		if pct() < rt.clicks/rt.opens*100 {
			out = append(out, Event{Kind: "click", After: openAt + sec(12, 240), UserAgent: ua, IP: fmt.Sprintf("%d.%d.%d.%d", 24+r.IntN(170), r.IntN(250), r.IntN(250), r.IntN(250)+1), Link: r.Float64()})
		}
	}
	return out
}

// humanDelay: most people who open do so within the first half hour.
func humanDelay(r *rand.Rand) time.Duration {
	m := math.Min(-math.Log(1-r.Float64())*9, 180) // exponential, mean 9 min
	return time.Duration((0.4 + m) * float64(time.Minute))
}

func humanAgent(mbp string, r *rand.Rand) string {
	pick := func(xs ...string) string { return xs[r.IntN(len(xs))] }
	const (
		chromeMac = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"
		chromeWin = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36"
		iosMail   = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148"
		macMail   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko)"
		outlookW  = "Mozilla/4.0 (compatible; ms-office; MSOffice 16)"
		gmailPx   = "Mozilla/5.0 (Windows NT 5.1; rv:11.0) Gecko Firefox/11.0 (via ggpht.com GoogleImageProxy)"
		yahooPx   = "YahooMailProxy; https://help.yahoo.com/kb/yahoo-mail-proxy-SLN28749.html"
		android   = "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36"
		firefoxW  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:131.0) Gecko/20100101 Firefox/131.0"
	)
	switch mbp {
	case "gmail":
		return pick(chromeMac, chromeMac, chromeWin, gmailPx, gmailPx, android)
	case "outlook":
		return pick(outlookW, outlookW, chromeWin, firefoxW)
	case "yahoo":
		return pick(yahooPx, yahooPx, chromeWin, iosMail)
	case "icloud":
		return pick(iosMail, iosMail, macMail)
	}
	return pick(chromeMac, chromeWin, outlookW, iosMail, firefoxW)
}

func (p *Provider) deliveredType() string {
	switch p.s.Mimic {
	case "postal":
		return "MessageSent"
	case "postmark":
		return "Delivery"
	}
	return "Delivery"
}

func (p *Provider) deliveredCode() string {
	if p.s.Mimic == "postal" {
		return "250 OK"
	}
	return "250 2.0.0 OK"
}

func (p *Provider) complaintType() string {
	switch p.s.Mimic {
	case "postal":
		return "MessageComplaint"
	case "postmark":
		return "SpamComplaint"
	}
	return "Complaint/abuse"
}

func (p *Provider) hardBounce(mbp string, after time.Duration, r *rand.Rand) Event {
	typ := "Bounce/Permanent/General"
	switch p.s.Mimic {
	case "postal":
		typ = "MessageBounced"
	case "postmark":
		typ = "HardBounce"
	}
	if r.Float64() < 0.85 {
		return Event{Kind: "hard_bounce", After: after, OriginalType: typ, Code: "550 5.1.1", Diagnostic: "user unknown", BounceClass: "mailbox_unknown"}
	}
	if mbp == "outlook" {
		return Event{Kind: "hard_bounce", After: after, OriginalType: typ, Code: "550 5.7.1", Diagnostic: "Service unavailable, client host blocked using Spamhaus", BounceClass: "reputation"}
	}
	return Event{Kind: "hard_bounce", After: after, OriginalType: typ, Code: "550 5.1.2", Diagnostic: "domain not found", BounceClass: "domain_invalid"}
}

func (p *Provider) softBounce(mbp string, after time.Duration) Event {
	typ := "Bounce/Transient/MailboxFull"
	if p.s.Mimic == "postal" {
		typ = "MessageDelayed"
	}
	if mbp == "yahoo" {
		return Event{Kind: "soft_bounce", After: after, OriginalType: typ, Code: "452 4.2.2", Diagnostic: "mailbox over quota", BounceClass: "mailbox_full"}
	}
	return Event{Kind: "soft_bounce", After: after, OriginalType: typ, Code: "452 4.2.2", Diagnostic: "mailbox full", BounceClass: "mailbox_full"}
}

func (p *Provider) deferral(mbp string, after time.Duration) Event {
	typ := "DeliveryDelay"
	if p.s.Mimic == "postal" {
		typ = "MessageDelayed"
	}
	if mbp == "yahoo" {
		return Event{Kind: "deferred", After: after, OriginalType: typ, Code: "421 4.7.0 TSS04", Diagnostic: "retry in 5 min", BounceClass: "temporary"}
	}
	return Event{Kind: "deferred", After: after, OriginalType: typ, Code: "451 4.7.1", Diagnostic: "temporarily deferred, retry in 5 min", BounceClass: "temporary"}
}
