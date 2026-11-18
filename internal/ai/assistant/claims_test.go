package assistant_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/ai/assistant"
)

// gatedWorker is a turn worker whose claims stop at their stages until
// released: hold[turn+stage] is closed to let a claim go on, and
// reached[turn+stage] is closed once it got there.
type gatedWorker struct {
	*assistant.Service
	hold, reached map[string]chan struct{}
}

func (f *fixture) gatedWorker(stops ...string) *gatedWorker {
	w := &gatedWorker{Service: f.worker(), hold: map[string]chan struct{}{}, reached: map[string]chan struct{}{}}
	for _, s := range stops {
		w.hold[s], w.reached[s] = make(chan struct{}), make(chan struct{})
	}
	// A failed test lets its held claims go, so they don't hold their
	// connections while the database is torn down.
	f.t.Cleanup(func() {
		for _, h := range w.hold {
			select {
			case <-h:
			default:
				close(h)
			}
		}
	})
	assistant.SetClaimHook(w.Service, func(turn uuid.UUID, stage string) {
		key := turn.String() + "/" + stage
		if r, ok := w.reached[key]; ok {
			close(r)
		}
		if h, ok := w.hold[key]; ok {
			<-h
		}
	})
	return w
}

// run executes a turn's job in the background; the result arrives on the
// channel.
func (w *gatedWorker) run(args assistant.TurnArgs) <-chan error {
	done := make(chan error, 1)
	go func() { done <- w.ExecuteTurn(context.Background(), args) }()
	return done
}

func waitClosed(t *testing.T, c <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(20 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func stillOpen(t *testing.T, c <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-c:
		t.Fatalf("%s happened", what)
	case <-time.After(300 * time.Millisecond):
	}
}

func (f *fixture) turnOf(thread uuid.UUID) assistant.AIThreadTurn {
	f.t.Helper()
	return *f.get(f.mc, thread).Turn
}

// A workspace's turns claim at once: they share the AI configuration lock,
// and only their reservations (under the budget's lock) and slots go one at
// a time. A claim held after taking its locks doesn't hold up another
// one's reservation and slot.
func TestAssistantClaimsShareTheConfigLock(t *testing.T) {
	f := newFixture(t)
	f.streams.then(says("First."), says("Second."))
	a, b := f.thread(f.mc, map[string]any{}), f.thread(f.mc, map[string]any{})
	ta, tb := f.send(f.mc, a.ID, "One").Turn.ID, f.send(f.mc, b.ID, "Two").Turn.ID
	w := f.gatedWorker(ta.String()+"/locked", tb.String()+"/slot")
	da := w.run(f.job(ta))
	waitClosed(t, w.reached[ta.String()+"/locked"], "the first claim's locks")
	db := w.run(f.job(tb))
	waitClosed(t, w.reached[tb.String()+"/slot"], "the second claim's reservation and slot while the first holds its locks")
	close(w.hold[tb.String()+"/slot"])
	close(w.hold[ta.String()+"/locked"])
	for _, d := range []<-chan error{da, db} {
		if err := <-d; err != nil {
			t.Fatal(err)
		}
	}
	if ga, gb := f.turnOf(a.ID), f.turnOf(b.ID); ga.State != "done" || gb.State != "done" {
		t.Fatalf("the turns: %+v %+v", ga, gb)
	}
}

// A change to the AI configuration waits for the claims holding the lock,
// and a claim that asks after a waiting change waits for it, then sees it:
// the assistant switched off, or the thread's model removed, refuses that
// turn and sends nothing. The turn that held the lock before the change
// was claimed and sent; its answer is checked again when it is saved, so
// it ends done, or failed for the change when the change was first.
func TestAssistantConfigChangesWaitForClaims(t *testing.T) {
	cases := []struct {
		name, reason string
		change       func(f *fixture, model uuid.UUID)
	}{
		{name: "assistant off", reason: "assistant_off", change: func(f *fixture, _ uuid.UUID) {
			f.setAssistant(ai.AIAssistantSettings{Enabled: false, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}, PerPerson: 2, PerWorkspace: 10})
		}},
		{name: "model removed", reason: "model_gone", change: func(f *fixture, model uuid.UUID) {
			f.oc.Do(http.MethodDelete, f.base+"/accounts/"+f.accountID.String()+"/models/"+model.String(), nil).MustStatus(f.t, 200)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			var s ai.AISettings
			f.oc.Do(http.MethodPost, f.base+"/accounts/"+f.accountID.String()+"/models", map[string]any{"model": "model-b", "inputPrice": 1, "outputPrice": 2}).
				MustStatus(t, 200).JSON(t, &s)
			modelB := s.Accounts[0].Models[1].ID
			f.passTools(f.oc, "/accounts/"+f.accountID.String()+"/models/"+modelB.String())
			f.streams.then(says("Before the change."))
			before := f.streams.requests.Load()
			a, b := f.thread(f.mc, map[string]any{"modelId": modelB}), f.thread(f.mc, map[string]any{"modelId": modelB})
			ta, tb := f.send(f.mc, a.ID, "One").Turn.ID, f.send(f.mc, b.ID, "Two").Turn.ID
			w := f.gatedWorker(ta.String()+"/locked", tb.String()+"/locked")
			close(w.hold[tb.String()+"/locked"])
			da := w.run(f.job(ta))
			waitClosed(t, w.reached[ta.String()+"/locked"], "the first claim's locks")
			changed := make(chan struct{})
			go func() { c.change(f, modelB); close(changed) }()
			stillOpen(t, changed, "the change while a claim held the lock")
			db := w.run(f.job(tb))
			stillOpen(t, w.reached[tb.String()+"/locked"], "a claim asking after a waiting change getting the lock first")
			close(w.hold[ta.String()+"/locked"])
			waitClosed(t, changed, "the change")
			for _, d := range []<-chan error{da, db} {
				if err := <-d; err != nil {
					t.Fatal(err)
				}
			}
			if c.name == "assistant off" {
				f.setAssistant(ai.AIAssistantSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}, PerPerson: 2, PerWorkspace: 10})
			}
			if got := f.turnOf(a.ID); got.StartedAt == nil || (got.State != "done" && (got.State != "failed" || got.Reason != c.reason)) {
				t.Fatalf("the turn that held the lock: %+v", got)
			}
			if got := f.turnOf(b.ID); got.State != "failed" || got.Reason != c.reason {
				t.Fatalf("the turn after the change: %+v, want failed %s", got, c.reason)
			}
			if n := f.streams.requests.Load() - before; n != 1 {
				t.Fatalf("%d requests sent, want the first turn's only", n)
			}
		})
	}
}

// A workspace key's turn and a personal key's take different budget locks,
// so they can claim at once and pick the same free slot: the later one
// waits for the earlier's commit, then takes the next slot instead of
// waiting its turn again.
func TestAssistantSlotsClaimedAtOnce(t *testing.T) {
	f := newFixture(t)
	f.oc.Do(http.MethodPut, f.base+"/personal-keys", map[string]any{"enabled": true, "capUsd": 10}).MustStatus(t, 200)
	var own ai.AIPersonalSettings
	f.mc.Do(http.MethodPost, f.base+"/personal/accounts", map[string]any{"provider": "compatible", "baseUrl": f.model.URL, "model": "own-model",
		"inputPrice": 1, "outputPrice": 2}).MustStatus(t, 200).JSON(t, &own)
	account, model := own.Accounts[0].ID, own.Accounts[0].Models[0].ID
	f.passTools(f.mc, "/personal/accounts/"+account.String()+"/models/"+model.String())
	f.streams.then(says("Workspace key."), says("Own key."))
	shared, personal := f.thread(f.mc, map[string]any{"modelId": f.modelID}), f.thread(f.mc, map[string]any{"modelId": model})
	ta, tb := f.send(f.mc, shared.ID, "One").Turn.ID, f.send(f.mc, personal.ID, "Two").Turn.ID
	w := f.gatedWorker(ta.String() + "/slot")
	da := w.run(f.job(ta))
	waitClosed(t, w.reached[ta.String()+"/slot"], "the first claim's slot")
	db := w.run(f.job(tb))
	time.Sleep(300 * time.Millisecond)
	close(w.hold[ta.String()+"/slot"])
	for _, d := range []<-chan error{da, db} {
		if err := <-d; err != nil {
			t.Fatalf("a claim: %v", err)
		}
	}
	if ga, gb := f.turnOf(shared.ID), f.turnOf(personal.ID); ga.State != "done" || gb.State != "done" {
		t.Fatalf("the turns: %+v %+v", ga, gb)
	}
}

// Removing a model (or its account) while a turn writes its thread: the
// turn has numbered an event (its own version of the thread's row) and
// then updates the row again, which checks the row's model reference and
// so waits for a model row being deleted. The removal locks the model's
// threads before the model, so it waits for the turn instead of
// deadlocking with it.
func TestAssistantModelRemovalWaitsForTurns(t *testing.T) {
	for _, removing := range []string{"model", "account"} {
		t.Run(removing, func(t *testing.T) {
			f := newFixture(t)
			ctx := context.Background()
			var s ai.AISettings
			account := f.accountID
			if removing == "account" {
				f.oc.Do(http.MethodPost, f.base+"/accounts", map[string]any{"provider": "compatible", "baseUrl": f.model.URL, "model": "model-b", "inputPrice": 1, "outputPrice": 2}).
					MustStatus(t, 200).JSON(t, &s)
				for _, a := range s.Accounts {
					if a.ID != f.accountID {
						account = a.ID
					}
				}
			} else {
				f.oc.Do(http.MethodPost, f.base+"/accounts/"+account.String()+"/models", map[string]any{"model": "model-b", "inputPrice": 1, "outputPrice": 2}).
					MustStatus(t, 200).JSON(t, &s)
			}
			var modelB uuid.UUID
			for _, a := range s.Accounts {
				for _, m := range a.Models {
					if a.ID == account && m.Model == "model-b" {
						modelB = m.ID
					}
				}
			}
			f.passTools(f.oc, "/accounts/"+account.String()+"/models/"+modelB.String())
			th := f.thread(f.mc, map[string]any{"modelId": modelB})
			tx, err := f.env.App.Pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(ctx) }()
			// As a turn's checkpoint does: the thread locked and its event
			// numbered.
			if _, err := tx.Exec(ctx, `SELECT 1 FROM ai_assistant_threads WHERE id = $1 FOR UPDATE`, th.ID); err != nil {
				t.Fatal(err)
			}
			if _, err := tx.Exec(ctx, `UPDATE ai_assistant_threads SET event_seq = event_seq + 1 WHERE id = $1`, th.ID); err != nil {
				t.Fatal(err)
			}
			path := f.base + "/accounts/" + account.String()
			if removing == "model" {
				path += "/models/" + modelB.String()
			}
			removed := make(chan int, 1)
			go func() { removed <- f.oc.Do(http.MethodDelete, path, nil).Status }()
			waitFor(t, "the removal waiting for a lock", func() bool {
				return f.count(`SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'`) > 0
			})
			// The turn writes its thread again, then commits.
			if _, err := tx.Exec(ctx, `UPDATE ai_assistant_threads SET last_activity_at = now() WHERE id = $1`, th.ID); err != nil {
				t.Fatalf("the turn's write: %v", err)
			}
			if err := tx.Commit(ctx); err != nil {
				t.Fatalf("the turn's commit: %v", err)
			}
			select {
			case status := <-removed:
				if status != 200 {
					t.Fatalf("the removal: %d", status)
				}
			case <-time.After(20 * time.Second):
				t.Fatal("the removal didn't finish")
			}
			if n := f.count(`SELECT count(*) FROM ai_assistant_threads WHERE id = $1 AND model_id IS NULL`, th.ID); n != 1 {
				t.Fatal("the thread still names the removed model")
			}
		})
	}
}
