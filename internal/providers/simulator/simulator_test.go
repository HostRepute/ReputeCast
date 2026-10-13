package simulator_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/contracttest"
	"github.com/hostrepute/reputecast/internal/providers/simulator"
)

// The demo transport stays out of the public catalog (it can't be
// connected); the contract suite needs it registered, so its test binary
// does that.
func init() { providers.Register(simulator.Descriptor()) }

const secret = "simulator-test-secret"

func newSim(t *testing.T, settings map[string]any) *simulator.Provider {
	t.Helper()
	b, _ := json.Marshal(settings)
	p, err := simulator.New(providers.Config{ProviderID: uuid.New(), WorkspaceID: uuid.New(), WebhookSecret: secret, Settings: b})
	if err != nil {
		t.Fatal(err)
	}
	return p.(*simulator.Provider)
}

func TestContract(t *testing.T) {
	contracttest.Run(t, contracttest.Suite{
		Adapter: simulator.Adapter,
		New: func(t *testing.T, _ *contracttest.Env) providers.Provider {
			return newSim(t, map[string]any{"demo": true, "mimic": "ses"})
		},
		Webhooks: []contracttest.Webhook{{
			Name: "delivery and bounce",
			Request: func(t *testing.T, _ *contracttest.Env) (*http.Request, []byte) {
				body := []byte(`[{"type":"delivered","recipient":"ana@example.com","provider_message_id":"sim-1","at":"2026-10-14T09:21:44Z","code":"250 2.0.0 OK"},` +
					`{"type":"hard_bounce","recipient":"rob@example.com","provider_message_id":"sim-2","at":"2026-10-14T09:21:45Z","code":"550 5.1.1","class":"mailbox_unknown"}]`)
				req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://ingest.test/hooks/simulator/token", bytes.NewReader(body))
				req.Header.Set("X-Simulator-Signature", simulator.Sign(secret, body))
				return req, body
			},
			Want: []providers.Event{
				{Type: providers.EventDelivered, Recipient: "ana@example.com", ProviderMessageID: "sim-1", OriginalType: "Simulated/delivered", OriginalCode: "250 2.0.0 OK"},
				{Type: providers.EventHardBounce, Recipient: "rob@example.com", ProviderMessageID: "sim-2", OriginalType: "Simulated/hard_bounce", OriginalCode: "550 5.1.1", BounceClass: "mailbox_unknown"},
			},
		}},
	})
}

func TestPostalOutageWindow(t *testing.T) {
	start := time.Date(2026, 10, 14, 9, 0, 4, 0, time.UTC)
	postal := newSim(t, map[string]any{"demo": true, "mimic": "postal", "scenario": simulator.ScenarioPostalOutage, "started": start, "server_url": "https://mta.acme.io"})
	ses := newSim(t, map[string]any{"demo": true, "mimic": "ses", "scenario": simulator.ScenarioPostalOutage, "started": start})
	for _, tc := range []struct {
		after time.Duration
		down  bool
	}{{0, false}, {13 * time.Minute, false}, {simulator.OutageAfter, true}, {20 * time.Minute, true}, {simulator.OutageAfter + simulator.OutageLasts, false}} {
		if got := postal.Outage(start.Add(tc.after)); got != tc.down {
			t.Errorf("Postal down %v after the start = %v, want %v", tc.after, got, tc.down)
		}
		if ses.Outage(start.Add(tc.after)) {
			t.Errorf("SES must never be down in the Postal outage")
		}
	}
	msg := contracttest.Message(0)
	_, err := postal.Send(simulator.WithNow(context.Background(), start.Add(15*time.Minute)), []providers.Message{msg})
	var ce *providers.CallError
	if !errors.As(err, &ce) || !ce.Temporary || ce.Throttled {
		t.Fatalf("an outage must fail the call temporarily (not as throttling): %v", err)
	}
	if res, err := postal.Send(simulator.WithNow(context.Background(), start.Add(5*time.Minute)), []providers.Message{msg}); err != nil && !errors.As(err, &ce) {
		t.Fatalf("before the outage: %v %v", res, err)
	}
}

// plan runs n messages through Plan for one mailbox provider and counts
// what happened to them.
func plan(t *testing.T, p *simulator.Provider, n int, tags map[string]string) map[string]float64 {
	t.Helper()
	out := map[string]float64{}
	for range n {
		m := contracttest.Message(0)
		m.Tags = tags
		seen := map[string]bool{}
		for _, ev := range p.Plan(m) {
			kind := ev.Kind
			if kind == "open" && ev.UserAgent == "Mozilla/5.0" {
				kind = "machine"
			}
			if !seen[kind] {
				seen[kind] = true
				out[kind]++
			}
		}
	}
	for k := range out {
		out[k] = out[k] / float64(n) * 100
	}
	return out
}

func TestPlanFollowsTheBoardsRates(t *testing.T) {
	p := newSim(t, map[string]any{"demo": true, "mimic": "ses"})
	const n = 40000
	for _, tc := range []struct {
		mailbox                string
		bounce, human, machine float64
	}{
		{"gmail", 0.4, 18.0, 5.7},
		{"outlook", 1.1, 13.4, 3.3},
		{"icloud", 0.4, 7.5, 5.6},
	} {
		got := plan(t, p, n, map[string]string{simulator.TagMailbox: tc.mailbox, simulator.TagEngaged: "true", simulator.TagTier: "active"})
		bounced := got["hard_bounce"] + got["soft_bounce"]
		near := func(what string, v, want, tol float64) {
			if math.Abs(v-want) > tol {
				t.Errorf("%s %s = %.2f%%, want about %.2f%%", tc.mailbox, what, v, want)
			}
		}
		near("bounces", bounced, tc.bounce, 0.25)
		// Bounced messages are never opened.
		near("human opens", got["open"], tc.human*(1-tc.bounce/100), 0.8)
		near("machine opens", got["machine"], tc.machine*(1-tc.bounce/100), 0.6)
		if got["delivered"] < 98 {
			t.Errorf("%s delivered only %.1f%%", tc.mailbox, got["delivered"])
		}
	}
}

func TestPlanIsDeterministic(t *testing.T) {
	p := newSim(t, map[string]any{"demo": true, "mimic": "postal"})
	m := contracttest.Message(0)
	m.Tags = map[string]string{simulator.TagMailbox: "yahoo", simulator.TagEngaged: "true"}
	a, b := p.Plan(m), p.Plan(m)
	if len(a) != len(b) {
		t.Fatalf("plans differ: %d and %d events", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("event %d differs: %+v and %+v", i, a[i], b[i])
		}
	}
}

func TestOutlookComplaintsScenario(t *testing.T) {
	p := newSim(t, map[string]any{"demo": true, "mimic": "ses", "scenario": simulator.ScenarioOutlookComplaints})
	const n = 40000
	lapsed := plan(t, p, n, map[string]string{simulator.TagMailbox: "outlook", simulator.TagEngaged: "true", simulator.TagTier: "sunset"})
	active := plan(t, p, n, map[string]string{simulator.TagMailbox: "outlook", simulator.TagEngaged: "true", simulator.TagTier: "active"})
	gmail := plan(t, p, n, map[string]string{simulator.TagMailbox: "gmail", simulator.TagEngaged: "false", simulator.TagTier: "sunset"})
	if lapsed["complaint"] < 0.6 || lapsed["complaint"] > 1.2 {
		t.Errorf("lapsed Outlook recipients complain %.2f%%, want about 0.9%%", lapsed["complaint"])
	}
	if active["complaint"] > 0.1 {
		t.Errorf("active Outlook recipients complain %.2f%%, want about 0.03%%", active["complaint"])
	}
	if gmail["complaint"] > 0.1 {
		t.Errorf("the scenario only affects Outlook: Gmail complaints %.2f%%", gmail["complaint"])
	}
}
