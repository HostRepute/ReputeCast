package sendtest

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/providers"
)

// The fault provider is a sending provider whose answers a test chooses: it
// takes the mail and answers, or takes it and never answers (the call times
// out), or holds the call until the test lets it go, or refuses it. It
// counts what it took per message, so a test can tell whether anything was
// delivered twice. "faultesp-idem" dedupes by message id like a provider
// with idempotency keys (providers.IdempotentSender).
//
// Only tests import sendtest, so the adapters exist only in test binaries.

// Fault modes.
const (
	// FaultOK takes every message and answers.
	FaultOK = "ok"
	// FaultTakeNoAnswer takes the messages, then never answers: the call
	// ends with the engine's call timeout.
	FaultTakeNoAnswer = "take_no_answer"
	// FaultHold takes the messages once the test closes Release, then
	// answers; Started is signalled when a call arrives.
	FaultHold = "hold"
	// FaultRefuse refuses the call with a 503 the provider answered.
	FaultRefuse = "refuse"
	// FaultFlaky refuses every other call with a 503 and takes the rest.
	FaultFlaky = "flaky"
	// FaultThrottle answers 429: the account sends too fast.
	FaultThrottle = "throttle"
	// FaultQuota answers that the account's daily quota is used up.
	FaultQuota = "quota"
	// FaultIdentity refuses every message: the sender isn't verified.
	FaultIdentity = "identity"
)

// Fault is the fault providers' shared state.
var Fault = &FaultState{}

// FaultState is what the fault providers do and what they took.
type FaultState struct {
	mu      sync.Mutex
	mode    string
	took    map[uuid.UUID]int
	to      map[string]int
	calls   int
	started chan struct{}
	release chan struct{}
}

// Reset sets the mode and forgets what was taken.
func (f *FaultState) Reset(mode string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.mode, f.took, f.to, f.calls = mode, map[uuid.UUID]int{}, map[string]int{}, 0
	f.started, f.release = make(chan struct{}, 64), make(chan struct{})
}

// SetMode changes what the providers do, keeping what they took.
func (f *FaultState) SetMode(mode string) {
	f.mu.Lock()
	f.mode = mode
	f.mu.Unlock()
}

// Started is signalled when a call arrives in FaultHold.
func (f *FaultState) Started() <-chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.started
}

// Release lets the calls held in FaultHold go.
func (f *FaultState) Release() {
	f.mu.Lock()
	defer f.mu.Unlock()
	select {
	case <-f.release:
	default:
		close(f.release)
	}
}

// Took is how many times the provider took each message (a delivery).
func (f *FaultState) Took() map[uuid.UUID]int {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[uuid.UUID]int, len(f.took))
	for k, v := range f.took {
		out[k] = v
	}
	return out
}

// TookFor is how many messages the provider took for an address.
func (f *FaultState) TookFor(address string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.to[address]
}

// Calls is how many Send calls arrived.
func (f *FaultState) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

type faultESP struct{ dedupe bool }

func init() {
	for _, d := range []struct {
		key    string
		dedupe bool
	}{{"faultesp", false}, {"faultesp-idem", true}} {
		dedupe := d.dedupe
		providers.Register(providers.Descriptor{Adapter: d.key, Name: "Fault ESP", Kind: "api",
			Capabilities: providers.Capabilities{Bulk: providers.BulkAllowed, Webhooks: "Registered", BatchSize: 10},
			New:          func(providers.Config) (providers.Provider, error) { return &faultESP{dedupe: dedupe}, nil }})
	}
}

func (p *faultESP) Adapter() string {
	if p.dedupe {
		return "faultesp-idem"
	}
	return "faultesp"
}

func (p *faultESP) Capabilities() providers.Capabilities {
	return providers.Capabilities{Bulk: providers.BulkAllowed, Webhooks: "Registered", BatchSize: 10}
}

func (p *faultESP) ParseWebhook(context.Context, *http.Request, []byte) ([]providers.Event, error) {
	return []providers.Event{}, nil
}

func (p *faultESP) take(msgs []providers.Message) []providers.SendResult {
	Fault.mu.Lock()
	defer Fault.mu.Unlock()
	out := make([]providers.SendResult, len(msgs))
	for i, m := range msgs {
		if !p.dedupe || Fault.took[m.ID] == 0 {
			Fault.took[m.ID]++
			Fault.to[m.To.Address]++
		}
		out[i] = providers.SendResult{MessageID: m.ID, ProviderMessageID: "fault-" + m.ID.String()}
	}
	return out
}

func (p *faultESP) Send(ctx context.Context, msgs []providers.Message) ([]providers.SendResult, error) {
	Fault.mu.Lock()
	Fault.calls++
	mode, started, release := Fault.mode, Fault.started, Fault.release
	Fault.mu.Unlock()
	switch mode {
	case FaultTakeNoAnswer:
		p.take(msgs)
		<-ctx.Done()
		return nil, &providers.CallError{Op: "POST /send", Message: ctx.Err().Error(), Temporary: true, Cause: ctx.Err()}
	case FaultHold:
		started <- struct{}{}
		select {
		case <-release:
		case <-time.After(time.Minute):
		}
		return p.take(msgs), nil
	case FaultRefuse:
		return nil, &providers.CallError{Op: "POST /send", Status: 503, Code: "ServiceUnavailable", Message: "Service Unavailable", Temporary: true}
	case FaultFlaky:
		Fault.mu.Lock()
		odd := Fault.calls%2 == 1
		Fault.mu.Unlock()
		if odd {
			return nil, &providers.CallError{Op: "POST /send", Status: 503, Code: "ServiceUnavailable", Message: "Service Unavailable", Temporary: true}
		}
	case FaultThrottle:
		return nil, &providers.CallError{Op: "POST /send", Status: 429, Code: "Throttling", Message: "Maximum sending rate exceeded.", Temporary: true, Throttled: true}
	case FaultQuota:
		return nil, &providers.CallError{Op: "POST /send", Status: 429, Code: "Throttling", Message: "Daily message quota exceeded.", Temporary: true, Throttled: true}
	case FaultIdentity:
		out := make([]providers.SendResult, len(msgs))
		for i, m := range msgs {
			out[i] = providers.SendResult{MessageID: m.ID, Err: &providers.SendError{Code: "MessageRejected",
				Message: "Email address is not verified. The following identities failed the check: " + m.From.Address}}
		}
		return out, nil
	}
	return p.take(msgs), nil
}

// IdempotencyWindow implements providers.IdempotentSender for faultesp-idem.
func (p *faultESP) IdempotencyWindow() time.Duration {
	if p.dedupe {
		return 30 * time.Minute
	}
	return 0
}

// UseFaultProvider connects a fault provider to the workspace and sends the
// fixture's campaign through it alone. It returns the provider's id.
func (f *Fixture) UseFaultProvider(dedupe bool) uuid.UUID {
	f.T.Helper()
	adapter := "faultesp"
	if dedupe {
		adapter = "faultesp-idem"
	}
	id := uuid.Must(uuid.NewV7())
	f.Exec(`INSERT INTO providers (id, workspace_id, adapter, name, label, config, credentials_enc, webhook_token, status, status_detail,
		ip_mode, ips, capabilities, limits, health, pricing, tracking, connected_at)
		VALUES ($1, $2, $3, $4, 'test', '{}', NULL, $5, 'healthy', '', 'shared', '{}',
		'{"bulk":"allowed","webhooks":"Registered","batchSize":10}', '{}', '{}', '{}', '{}', now())`,
		id, f.WS, adapter, "Fault ESP "+id.String()[:8], "fault-"+id.String())
	f.Exec(`UPDATE campaigns SET route_override = $2 WHERE id = $1`, f.Campaign, `{"provider_id":"`+id.String()+`"}`)
	return id
}
