package eventqueue_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/providers"
	"github.com/hostrepute/reputecast/internal/providers/eventqueue"
	"github.com/hostrepute/reputecast/internal/providers/manage"
	"github.com/hostrepute/reputecast/internal/store"
)

// fakeQueue is an adapter whose events come through a queue: it hands out
// what's queued and records what was acknowledged.
type fakeQueue struct{}

var (
	qmu     sync.Mutex
	queued  []providers.QueuedEvent
	acked   []string
	failing error
)

func init() {
	providers.Register(providers.Descriptor{Adapter: "fakequeue", Name: "Fake Queue", Kind: "api",
		Capabilities: providers.Capabilities{Bulk: providers.BulkAllowed, Webhooks: "SNS", BatchSize: 1},
		New:          func(providers.Config) (providers.Provider, error) { return &fakeQueue{}, nil }})
}

func (f *fakeQueue) Adapter() string { return "fakequeue" }
func (f *fakeQueue) Capabilities() providers.Capabilities {
	return providers.Capabilities{Webhooks: "SNS"}
}
func (f *fakeQueue) Send(context.Context, []providers.Message) ([]providers.SendResult, error) {
	return nil, errors.New("not used")
}
func (f *fakeQueue) ParseWebhook(context.Context, *http.Request, []byte) ([]providers.Event, error) {
	return []providers.Event{}, nil
}
func (f *fakeQueue) QueueEnabled() bool { return true }
func (f *fakeQueue) ReceiveEvents(_ context.Context, maxEvents int, _ time.Duration) ([]providers.QueuedEvent, error) {
	qmu.Lock()
	defer qmu.Unlock()
	if failing != nil {
		return nil, failing
	}
	n := min(maxEvents, len(queued))
	out := append([]providers.QueuedEvent(nil), queued[:n]...)
	queued = queued[n:]
	return out, nil
}
func (f *fakeQueue) AckEvents(_ context.Context, handles []string) error {
	qmu.Lock()
	defer qmu.Unlock()
	acked = append(acked, handles...)
	return nil
}

// PR-4: events polled from the queue are stored like webhook deliveries
// (source "queue") and leave the queue only once stored; a queue that
// can't be read is told to the provider's health.
func TestPollStoresEventsThenAcks(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Owner"})
	ws := env.CreateWorkspace("Queue test", owner.ID)
	cfg, _ := json.Marshal(map[string]any{"events": providers.EventsQueue})
	p, err := env.App.Q.ConnectInsertProvider(ctx, store.ConnectInsertProviderParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: ws.ID, Adapter: "fakequeue",
		Name: "Fake Queue", Config: cfg, WebhookToken: "token-0123456789", Status: "healthy", IpMode: "shared", Ips: []string{},
		Capabilities: json.RawMessage(`{}`), Limits: json.RawMessage(`{}`), Health: json.RawMessage(`{}`), Pricing: json.RawMessage(`{}`), ConnectedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	qmu.Lock()
	queued, acked, failing = nil, nil, nil
	for i := range 12 {
		queued = append(queued, providers.QueuedEvent{Body: []byte(`{"Type":"Notification","Message":"{}"}`), ContentType: "application/json",
			Handle: "handle-" + string(rune('a'+i))})
	}
	qmu.Unlock()
	var polled []error
	w := &eventqueue.Worker{Pool: env.App.Pool, Q: env.App.Q, Factory: &manage.Factory{Box: env.App.Secrets}, For: 50 * time.Millisecond,
		OnPoll: func(_ context.Context, _ store.Provider, err error) { polled = append(polled, err) }}
	n, err := w.Poll(ctx, p)
	if err != nil || n != 12 {
		t.Fatalf("polled %d events: %v", n, err)
	}
	var stored int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM provider_raw_events WHERE provider_id = $1 AND source = 'queue' AND signature_ok`, p.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	qmu.Lock()
	gotAcked := len(acked)
	qmu.Unlock()
	if stored != 12 || gotAcked != 12 {
		t.Fatalf("stored %d, acknowledged %d", stored, gotAcked)
	}
	if len(polled) != 1 || polled[0] != nil {
		t.Fatalf("health told %v", polled)
	}

	// A queue that can't be read: nothing stored, nothing acknowledged,
	// and the provider's health hears about it.
	qmu.Lock()
	failing = errors.New("AccessDenied: sqs:ReceiveMessage")
	qmu.Unlock()
	if _, err := w.Poll(ctx, p); err == nil {
		t.Fatal("a failing queue polled without an error")
	}
	if len(polled) != 2 || polled[1] == nil {
		t.Fatalf("health told %v", polled)
	}
}
