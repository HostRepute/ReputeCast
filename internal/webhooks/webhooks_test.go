package webhooks_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/webhooks"
)

const ws = "/workspaces/acme-newsletter"

type receiver struct {
	mu     sync.Mutex
	bodies [][]byte
	sigs   []string
	ts     []string
	status int
}

func (r *receiver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	b, _ := io.ReadAll(req.Body)
	r.mu.Lock()
	r.bodies = append(r.bodies, b)
	r.sigs = append(r.sigs, req.Header.Get(webhooks.HeaderSignature))
	r.ts = append(r.ts, req.Header.Get(webhooks.HeaderTimestamp))
	status := r.status
	r.mu.Unlock()
	if status == 0 {
		status = 200
	}
	w.WriteHeader(status)
}

func service(env *apptest.Env) *webhooks.Service {
	return &webhooks.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, AllowPrivate: func() bool { return true }}
}

func TestWebhooks(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	var list struct {
		Items []struct {
			Name, Status string
			Success24h   *float64
			Total24h     int64
			Queued       int64
		}
		Events []struct{ Name string }
	}
	c.Do("GET", ws+"/webhooks", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 3 || list.Items[2].Status != "failing" || list.Items[2].Total24h != 18402 || list.Items[2].Success24h == nil || *list.Items[2].Success24h != 61 {
		t.Fatalf("seeded webhooks: %+v", list.Items)
	}

	rcv := &receiver{}
	srv := httptest.NewServer(rcv)
	defer srv.Close()

	// SSRF: private and metadata addresses are refused unless allowed.
	c.Do("POST", ws+"/webhooks", map[string]any{"url": srv.URL + "/hook", "events": []string{"email.bounced"}}).MustStatus(t, 422)
	c.Do("POST", ws+"/webhooks", map[string]any{"url": "http://169.254.169.254/latest/meta-data", "events": []string{"email.bounced"}}).MustStatus(t, 422)
	env.App.Cfg.Webhooks.AllowPrivateNetworks = true
	c.Do("POST", ws+"/webhooks", map[string]any{"url": "http://169.254.169.254/latest/meta-data", "events": []string{"email.bounced"}}).MustStatus(t, 422)
	c.Do("POST", ws+"/webhooks", map[string]any{"url": srv.URL + "/hook", "events": []string{"nope.event"}}).MustStatus(t, 422)

	var created struct {
		Webhook struct{ ID string }
		Secret  string
	}
	c.Do("POST", ws+"/webhooks", map[string]any{"name": "Test receiver", "url": srv.URL + "/hook", "events": []string{"email.*"}}).MustStatus(t, 201).JSON(t, &created)
	if len(created.Secret) < 20 {
		t.Fatalf("secret: %q", created.Secret)
	}
	r := c.Do("GET", ws+"/webhooks", nil).MustStatus(t, 200)
	if json.Valid(r.Body) && contains(r.Body, created.Secret) {
		t.Fatal("the secret is never listed")
	}

	// Test event: delivered and signed.
	var test struct {
		Delivery struct {
			State      string
			StatusCode *int32
		}
	}
	c.Do("POST", ws+"/webhooks/"+created.Webhook.ID+"/test", nil).MustStatus(t, 200).JSON(t, &test)
	if test.Delivery.State != "delivered" || len(rcv.bodies) != 1 {
		t.Fatalf("test delivery: %+v", test)
	}
	ts, _ := strconv.ParseInt(rcv.ts[0], 10, 64)
	if !webhooks.Verify(created.Secret, rcv.sigs[0], ts, rcv.bodies[0], time.Now(), 5*time.Minute) {
		t.Fatal("signature must verify with the secret")
	}
	if webhooks.Verify(created.Secret, rcv.sigs[0], ts-3600, rcv.bodies[0], time.Now(), 5*time.Minute) {
		t.Fatal("old timestamps are rejected")
	}

	// Emit from another domain, then a failing endpoint retries with backoff.
	ctx := context.Background()
	w, _ := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	n, err := webhooks.Emit(ctx, env.App.Pool, w.ID, w.Slug, "email.bounced", map[string]any{"subtype": "hard", "provider_code": "550 5.1.1"})
	if err != nil || n != 3 { // the test receiver, the product backend (email.bounced) and the warehouse (email.*)
		t.Fatalf("emit: %d %v", n, err)
	}
	var id uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT d.id FROM webhook_deliveries d JOIN webhooks h ON h.id = d.webhook_id WHERE h.url = $1 AND d.event = 'email.bounced'`, srv.URL+"/hook").Scan(&id); err != nil {
		t.Fatal(err)
	}
	rcv.mu.Lock()
	rcv.status = 502
	rcv.mu.Unlock()
	s := service(env)
	next, err := s.Attempt(ctx, id)
	if err != nil || next == nil || time.Until(*next) > 61*time.Second {
		t.Fatalf("first failure retries after a minute: %v %v", next, err)
	}
	var state string
	var attempt int
	_ = env.App.Pool.QueryRow(ctx, `SELECT state, attempt FROM webhook_deliveries WHERE id = $1`, id).Scan(&state, &attempt)
	if state != "retrying" || attempt != 1 {
		t.Fatalf("after failure: %s %d", state, attempt)
	}
	// Exhaust the attempts: the delivery goes to the dead-letter queue.
	for i := 0; i < 7; i++ {
		if _, err := env.App.Pool.Exec(ctx, `UPDATE webhook_deliveries SET next_attempt_at = now() WHERE id = $1`, id); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Attempt(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	_ = env.App.Pool.QueryRow(ctx, `SELECT state, attempt FROM webhook_deliveries WHERE id = $1`, id).Scan(&state, &attempt)
	if state != "dead" || attempt != 8 {
		t.Fatalf("dead letter: %s %d", state, attempt)
	}
	var retry struct{ Queued int }
	c.Do("POST", ws+"/webhooks/deliveries/"+id.String()+"/retry", nil).MustStatus(t, 200).JSON(t, &retry)
	if retry.Queued != 1 {
		t.Fatalf("replay: %+v", retry)
	}
	rcv.mu.Lock()
	rcv.status = 200
	rcv.mu.Unlock()
	if _, err := s.Attempt(ctx, id); err != nil {
		t.Fatal(err)
	}
	_ = env.App.Pool.QueryRow(ctx, `SELECT state FROM webhook_deliveries WHERE id = $1`, id).Scan(&state)
	if state != "delivered" {
		t.Fatalf("replayed: %s", state)
	}

	var deliveries struct {
		Items []struct{ Event, State, Endpoint string }
	}
	c.Do("GET", ws+"/webhooks/deliveries?webhookId="+created.Webhook.ID, nil).MustStatus(t, 200).JSON(t, &deliveries)
	if len(deliveries.Items) != 2 {
		t.Fatalf("deliveries: %+v", deliveries.Items)
	}
	var detail struct {
		Payload json.RawMessage
	}
	c.Do("GET", ws+"/webhooks/deliveries/"+id.String(), nil).MustStatus(t, 200).JSON(t, &detail)
	if !contains(detail.Payload, "550 5.1.1") {
		t.Fatalf("payload: %s", detail.Payload)
	}

	c.Do("POST", ws+"/webhooks/"+created.Webhook.ID+"/pause", nil).MustStatus(t, 204)
	c.Do("POST", ws+"/webhooks/"+created.Webhook.ID+"/resume", nil).MustStatus(t, 204)
	var sec struct{ Secret string }
	c.Do("POST", ws+"/webhooks/"+created.Webhook.ID+"/rotate-secret", nil).MustStatus(t, 200).JSON(t, &sec)
	if sec.Secret == created.Secret {
		t.Fatal("rotation issues a new secret")
	}
	c.Do("DELETE", ws+"/webhooks/"+created.Webhook.ID, nil).MustStatus(t, 204)

	// Viewers don't manage webhooks.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", ws+"/webhooks", nil).MustStatus(t, 403)
}

func contains(b []byte, s string) bool {
	return len(s) > 0 && len(b) >= len(s) && (string(b) == s || indexOf(string(b), s) >= 0)
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}
