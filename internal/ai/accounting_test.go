package ai_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/apptest"
)

// Accounting settles a request within its reservation without the
// budget's lock, which every reservation of the budget takes (reserving
// reads what the budget used and holds in one snapshot, so a settle moving
// a request from one to the other, for less, can't be missed). A request
// the provider reports above its reservation settles under the lock, as a
// reservation would: it adds to the budget.
func TestAIAccountingLocks(t *testing.T) {
	var prompt atomic.Int64
	var hold atomic.Bool
	sent, answer := make(chan struct{}, 1), make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		if hold.Load() {
			sent <- struct{}{}
			<-answer
		}
		draft, _ := json.Marshal(ai.AIDraftEmail{Subject: "Export update", Preheader: "What changed",
			MJML: `<mjml><mj-body><mj-section><mj-column><mj-text>Exports are easier.</mj-text></mj-column></mj-section></mj-body></mjml>`})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": string(draft)}, "finish_reason": "stop"}},
			"usage": map[string]int64{"prompt_tokens": prompt.Load(), "completion_tokens": 29}})
	}))
	defer srv.Close()
	prompt.Store(113)
	r := newRegistry(t, &modelServer{srv: srv}, map[string]any{"capKind": "usd", "capUsd": 1000})
	ctx := context.Background()
	// The prompt the provider reports, from the request's reservation.
	for _, c := range []struct {
		name   string
		prompt func(input, output int64) int64
		waits  bool
	}{
		{"within its reservation", func(int64, int64) int64 { return 113 }, false},
		{"above its reservation", func(int64, int64) int64 { return 50_000_000 }, true},
		// More tokens than reserved for less than the reserved cost: a
		// token cap would be passed.
		{"above its reservation in tokens", func(input, output int64) int64 { return input + output + 100 }, true},
	} {
		hold.Store(true)
		done := make(chan *apptest.Response, 1)
		go func() { done <- draftBy(r.c, r.base) }()
		select {
		case <-sent:
		case <-time.After(20 * time.Second):
			t.Fatalf("%s: the request wasn't sent", c.name)
		}
		var input, output, reserved int64
		if err := r.env.App.Pool.QueryRow(ctx, `SELECT input_tokens, output_tokens, cost_micros FROM ai_budget_reservations WHERE workspace_id = $1 AND state = 'reserved'`, r.ws.ID).
			Scan(&input, &output, &reserved); err != nil {
			t.Fatal(err)
		}
		p := c.prompt(input, output)
		if c.waits && p+29 <= input+output {
			t.Fatalf("%s: the fixture reports %d tokens of %d reserved", c.name, p+29, input+output)
		}
		prompt.Store(p)
		// Reserved and sent: now a reservation of the budget holds its lock.
		tx, err := r.env.App.Pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = tx.Rollback(ctx) })
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('ai_budget:' || $1::text, 0))`, r.ws.ID); err != nil {
			t.Fatal(err)
		}
		answer <- struct{}{}
		var res *apptest.Response
		select {
		case res = <-done:
			if c.waits {
				t.Fatalf("%s: settled without the budget's lock", c.name)
			}
		case <-time.After(1500 * time.Millisecond):
			if !c.waits {
				t.Fatalf("%s: waited for the budget's lock", c.name)
			}
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
		if res == nil {
			select {
			case res = <-done:
			case <-time.After(20 * time.Second):
				t.Fatalf("%s: didn't settle once the lock was free", c.name)
			}
		}
		hold.Store(false)
		res.MustStatus(t, 200)
		var cost int64
		if err := r.env.App.Pool.QueryRow(ctx, `SELECT cost_micros FROM ai_requests WHERE workspace_id = $1 AND feature = 'draft' ORDER BY created_at DESC, id DESC LIMIT 1`, r.ws.ID).
			Scan(&cost); err != nil {
			t.Fatal(err)
		}
		if want := p*1 + 29*2; cost != want {
			t.Fatalf("%s: logged at %d micros, want %d", c.name, cost, want)
		}
		if c.name == "above its reservation in tokens" && cost > reserved {
			t.Fatalf("%s: %d micros of %d reserved", c.name, cost, reserved)
		}
	}
}
