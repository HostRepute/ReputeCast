package assistant_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/ai/assistant"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/testdb"
)

// A message queues a turn whose job carries ids only; the worker streams
// the answer into events, saves it, accounts it in the thread's ledger and
// the activity log (under the person, without content), and running the
// job again sends nothing more. One turn answers at a time; a retried
// message returns its turn, a reused key for another text is refused.
func TestAssistantTurnAnswers(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	// The word looked for below isn't hexadecimal: a hex word can turn up by
	// chance in a row's ids (51b2 once matched a UUID in ai_requests).
	const sentinel = "Sentinel question kumquat about subject lines"
	th := f.thread(f.mc, map[string]any{"screen": "campaign.report"})
	path := f.base + "/assistant/threads/" + th.ID.String()
	f.vc.Do(http.MethodPost, path+"/messages", map[string]any{"text": "Hi", "requestKey": "viewer-1"}).MustStatus(t, 403)
	f.cc.Do(http.MethodPost, path+"/messages", map[string]any{"text": "Hi", "requestKey": "colleague-1"}).MustStatus(t, 404)
	var sent assistant.AIThread
	f.mc.Do(http.MethodPost, path+"/messages", map[string]any{"text": sentinel, "requestKey": "turn-1"}).MustStatus(t, 200).JSON(t, &sent)
	if sent.Turn == nil || sent.Turn.State != "queued" || sent.Turn.ModelID == nil || *sent.Turn.ModelID != f.modelID || len(sent.Messages) != 1 ||
		sent.Messages[0].Role != "user" || sent.Messages[0].Text != sentinel || *sent.Messages[0].TurnID != sent.Turn.ID {
		t.Fatalf("after sending: %+v", sent)
	}
	turn := sent.Turn.ID
	var again assistant.AIThread
	f.mc.Do(http.MethodPost, path+"/messages", map[string]any{"text": sentinel, "requestKey": "turn-1"}).MustStatus(t, 200).JSON(t, &again)
	if again.Turn.ID != turn || len(again.Messages) != 1 {
		t.Fatalf("a retried message: %+v", again)
	}
	if p := f.mc.Do(http.MethodPost, path+"/messages", map[string]any{"text": "Other", "requestKey": "turn-1"}).MustStatus(t, 409).Problem(t); p["code"] != "ai_assistant_turn_key" {
		t.Fatalf("a reused key: %v", p)
	}
	if p := f.mc.Do(http.MethodPost, path+"/messages", map[string]any{"text": "And another", "requestKey": "turn-2"}).MustStatus(t, 409).Problem(t); p["code"] != "ai_assistant_turn_active" {
		t.Fatalf("a message while a turn answers: %v", p)
	}
	// The job: ids only.
	var raw string
	if err := f.env.App.Pool.QueryRow(ctx, `SELECT args::text FROM river_job WHERE kind = 'ai.assistant_turn'`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var keys map[string]any
	if err := json.Unmarshal([]byte(raw), &keys); err != nil || len(keys) != 4 || keys["turnId"] != turn.String() || keys["threadId"] != th.ID.String() ||
		strings.Contains(raw, "kumquat") {
		t.Fatalf("the job's args: %s", raw)
	}
	f.streams.set(answer("Shorter ", "subjects ", "work."))
	w := f.worker()
	if err := w.ExecuteTurn(ctx, f.job(turn)); err != nil {
		t.Fatal(err)
	}
	after := f.get(f.mc, th.ID)
	if after.Turn == nil || after.Turn.State != "done" || after.Turn.Reason != "" || after.Turn.EndedAt == nil || len(after.Messages) != 2 ||
		after.Messages[1].Role != "assistant" || after.Messages[1].Text != "Shorter subjects work." || after.Messages[1].Partial ||
		after.Usage.Requests != 1 || after.Usage.Tokens != 150 || after.Usage.CostMicros != 180 || after.Usage.UncertainRequests != 0 {
		t.Fatalf("after the turn: %+v", after)
	}
	// Again (a retry after a lost acknowledgment): nothing is sent.
	if err := w.ExecuteTurn(ctx, f.job(turn)); err != nil {
		t.Fatal(err)
	}
	if n := f.streams.requests.Load(); n != 1 {
		t.Fatalf("%d requests, want 1", n)
	}
	// The events tell it in order.
	kinds := []string{}
	for _, e := range f.events(th.ID) {
		if len(kinds) == 0 || kinds[len(kinds)-1] != e.Kind || e.Kind != "delta" {
			kinds = append(kinds, e.Kind)
		}
	}
	if !slices.Equal(kinds, []string{"message", "turn", "turn", "delta", "message", "usage", "turn"}) {
		t.Fatalf("events %v", kinds)
	}
	if after.EventSeq != int64(len(f.events(th.ID))) {
		t.Fatalf("event seq %d, %d events", after.EventSeq, len(f.events(th.ID)))
	}
	// The activity log and the usage per person, never the content.
	if n := f.count(`SELECT count(*) FROM ai_requests WHERE workspace_id = $1 AND user_id = $2 AND feature = 'assistant' AND outcome = 'answered'
		AND target_kind = 'assistant_thread' AND target_id = $3 AND usage_known AND cost_micros = 180 AND model_id = $4 AND target_label = ''`, f.ws.ID, f.member, th.ID, f.modelID); n != 1 {
		t.Fatalf("%d activity rows", n)
	}
	var s ai.AISettings
	f.oc.Do(http.MethodGet, f.base, nil).MustStatus(t, 200).JSON(t, &s)
	if !slices.ContainsFunc(s.ByMember, func(m ai.AIUsageMember) bool {
		return m.UserID == f.member.String() && m.Requests == 1 && m.CostMicros == 180
	}) {
		t.Fatalf("by member: %+v", s.ByMember)
	}
	if n := f.count(`SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'ai_assistant_turn.created' AND target_id = $2
		AND metadata->>'turn_id' = $3 AND metadata->>'via' = 'ai_assistant'`, f.ws.ID, th.ID.String(), turn.String()); n != 1 {
		t.Fatalf("%d audit entries of the message", n)
	}
	for _, q := range []string{`SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND (row_to_json(audit_events.*)::text LIKE '%kumquat%' OR row_to_json(audit_events.*)::text LIKE '%Shorter%')`,
		`SELECT count(*) FROM ai_requests WHERE workspace_id = $1 AND (row_to_json(ai_requests.*)::text LIKE '%kumquat%' OR row_to_json(ai_requests.*)::text LIKE '%Shorter%')`,
		`SELECT count(*) FROM river_job WHERE $1::uuid IS NOT NULL AND (args::text LIKE '%kumquat%' OR args::text LIKE '%Shorter%')`,
		`SELECT count(*) FROM ai_assistant_events e JOIN ai_assistant_threads t ON t.id = e.thread_id WHERE t.workspace_id = $1
			AND (position(convert_to('kumquat', 'UTF8') in e.payload_sealed) > 0 OR position(convert_to('Shorter', 'UTF8') in e.payload_sealed) > 0)`,
		`SELECT count(*) FROM ai_assistant_messages m JOIN ai_assistant_threads t ON t.id = m.thread_id WHERE t.workspace_id = $1
			AND (position(convert_to('kumquat', 'UTF8') in m.content_sealed) > 0 OR position(convert_to('Shorter', 'UTF8') in m.content_sealed) > 0)`} {
		if n := f.count(q, f.ws.ID); n != 0 {
			t.Fatalf("the content in plain text: %d rows of %s", n, q)
		}
	}
}

// A question about reputation, blocklists or placement gets the HostRepute
// card at once: no job, no model request, no cost.
func TestAssistantReputationCard(t *testing.T) {
	f := newFixture(t)
	th := f.thread(f.mc, map[string]any{})
	after := f.send(f.mc, th.ID, "Why is our domain on a blocklist?")
	if after.Turn == nil || after.Turn.State != "done" || after.Turn.Reason != "reputation" || len(after.Messages) != 2 || after.Messages[1].Card != "hostrepute" ||
		!strings.Contains(after.Messages[1].Text, "HostRepute") {
		t.Fatalf("a reputation question: %+v", after)
	}
	if n := f.count(`SELECT count(*) FROM river_job WHERE kind = 'ai.assistant_turn'`) + f.count(`SELECT count(*) FROM ai_requests WHERE workspace_id = $1 AND feature = 'assistant'`, f.ws.ID); n != 0 {
		t.Fatalf("%d jobs and requests", n)
	}
	// The next message is the model's again.
	if next := f.send(f.mc, th.ID, "Thanks. Now a subject line?"); next.Turn.State != "queued" {
		t.Fatalf("the next turn: %+v", next.Turn)
	}
}

// Two attempts of a turn's job at once make one provider request: the
// second finds the turn taken and waits, and later attempts find it done.
func TestAssistantTwoWorkersOneRequest(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	th := f.thread(f.mc, map[string]any{})
	turn := f.send(f.mc, th.ID, "Write me an intro").Turn.ID
	args := f.job(turn)
	release := make(chan struct{})
	f.streams.set(held(release, "Intro ", "text."))
	w := f.worker()
	errs := make(chan error, 2)
	for range 2 {
		go func() { errs <- w.ExecuteTurn(ctx, args) }()
	}
	f.streams.waitStarted(t, 1)
	if err := <-errs; !isSnooze(err) {
		t.Fatalf("the second attempt: %v, want a snooze", err)
	}
	close(release)
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
	if err := w.ExecuteTurn(ctx, args); err != nil {
		t.Fatal(err)
	}
	if n := f.streams.requests.Load(); n != 1 {
		t.Fatalf("%d provider requests, want 1", n)
	}
	if th := f.get(f.mc, th.ID); th.Turn.State != "done" || len(th.Messages) != 2 || th.Messages[1].Text != "Intro text." {
		t.Fatalf("the turn: %+v", th)
	}
}

// Two turns answer at once per person by default: a third waits queued
// (its job snoozes) until one ends.
func TestAssistantSlotsPerPerson(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	release := make(chan struct{})
	f.streams.set(held(release, "Working ", "on it."))
	w := f.worker()
	var turns []uuid.UUID
	for i := range 3 {
		th := f.thread(f.mc, map[string]any{})
		turns = append(turns, f.send(f.mc, th.ID, "Question "+string(rune('A'+i))).Turn.ID)
	}
	errs := make(chan error, 2)
	for _, turn := range turns[:2] {
		args := f.job(turn)
		go func() { errs <- w.ExecuteTurn(ctx, args) }()
	}
	f.streams.waitStarted(t, 2)
	if err := w.ExecuteTurn(ctx, f.job(turns[2])); !isSnooze(err) {
		t.Fatalf("the third turn: %v, want a snooze", err)
	}
	if n := f.count(`SELECT count(*) FROM ai_assistant_turns WHERE id = $1 AND state = 'queued'`, turns[2]); n != 1 {
		t.Fatal("the third turn isn't queued")
	}
	close(release)
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	if err := w.ExecuteTurn(ctx, f.job(turns[2])); err != nil {
		t.Fatal(err)
	}
	if n := f.count(`SELECT count(*) FROM ai_assistant_turns WHERE id = ANY($1) AND state = 'done'`, turns); n != 3 || f.streams.requests.Load() != 3 {
		t.Fatalf("%d turns done, %d requests", n, f.streams.requests.Load())
	}
}

// Two threads racing for the last of the monthly cap: one request is
// reserved and sent, the other turn fails on the cap without a request.
func TestAssistantLastDollar(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	// $100 per million tokens: one request reserves most of a $1 cap.
	f.oc.Do(http.MethodPut, f.base+"/accounts/"+f.accountID.String()+"/models/"+f.modelID.String()+"/prices", map[string]any{"input": 100, "output": 100}).MustStatus(t, 200)
	f.oc.Do(http.MethodPut, f.base+"/cap", map[string]any{"kind": "usd", "usd": 1}).MustStatus(t, 200)
	release := make(chan struct{})
	defer close(release)
	f.streams.set(held(release, "Spent."))
	w := f.worker()
	var threads, turns []uuid.UUID
	for _, c := range []*apptest.Client{f.mc, f.cc} {
		th := f.thread(c, map[string]any{})
		threads, turns = append(threads, th.ID), append(turns, f.send(c, th.ID, "Draft an intro").Turn.ID)
	}
	errs := make(chan error, 2)
	for _, turn := range turns {
		args := f.job(turn)
		go func() {
			// As River runs a job again after a snooze.
			for {
				err := w.ExecuteTurn(ctx, args)
				if !isSnooze(err) {
					errs <- err
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		}()
	}
	refused := `SELECT count(*) FROM ai_assistant_turns WHERE id = ANY($1) AND state = 'failed' AND reason IN ('reservation_cap', 'cap_reached')`
	waitFor(t, "a turn refused, or a second request", func() bool { return f.streams.requests.Load() == 2 || f.count(refused, turns) == 1 })
	time.Sleep(200 * time.Millisecond)
	if n := f.streams.requests.Load(); n != 1 {
		t.Fatalf("%d provider requests, want 1", n)
	}
	if err := <-errs; err != nil {
		t.Fatalf("the turn refused: %v", err)
	}
	if n := f.count(refused, turns); n != 1 {
		t.Fatalf("%d turns refused on the cap", n)
	}
	if n := f.count(`SELECT count(*) FROM ai_assistant_turns WHERE id = ANY($1) AND state = 'answering'`, turns); n != 1 {
		t.Fatalf("%d turns answering", n)
	}
	// The refused turn never started: its claim saved the start and the
	// step before reserving, and the refusal undid them. Its turn events
	// are its queued and its failed state.
	if n := f.count(`SELECT count(*) FROM ai_assistant_turns WHERE id = ANY($1) AND state = 'failed' AND started_at IS NULL
		AND NOT EXISTS (SELECT 1 FROM ai_assistant_steps s WHERE s.turn_id = ai_assistant_turns.id)
		AND (SELECT count(*) FROM ai_assistant_events e WHERE e.turn_id = ai_assistant_turns.id AND e.kind = 'turn') = 2`, turns); n != 1 {
		t.Fatal("the refused turn kept its start, its step or its started event")
	}
	for _, th := range threads {
		if n := f.count(`SELECT count(*) FROM ai_assistant_threads t WHERE id = $1
			AND event_seq = (SELECT count(*) FROM ai_assistant_events e WHERE e.thread_id = t.id)`, th); n != 1 {
			t.Fatal("a thread's events aren't numbered without gaps")
		}
	}
}

// Stop while the answer streams: the turn is stopped at once, nothing of
// the answer is written after the stopped event, the stream is cut and
// its request is still accounted.
func TestAssistantStopMidStream(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	th := f.thread(f.mc, map[string]any{})
	turn := f.send(f.mc, th.ID, "Tell me a long story").Turn.ID
	f.streams.set(drip("word ", 50*time.Millisecond, 200))
	done := make(chan error, 1)
	args := f.job(turn)
	go func() { done <- f.worker().ExecuteTurn(ctx, args) }()
	f.waitKind(th.ID, "delta")
	f.vc.Do(http.MethodPost, f.base+"/assistant/threads/"+th.ID.String()+"/turns/"+turn.String()+"/stop", nil).MustStatus(t, 403)
	f.cc.Do(http.MethodPost, f.base+"/assistant/threads/"+th.ID.String()+"/turns/"+turn.String()+"/stop", nil).MustStatus(t, 404)
	var stopped assistant.AIThread
	f.mc.Do(http.MethodPost, f.base+"/assistant/threads/"+th.ID.String()+"/turns/"+turn.String()+"/stop", nil).MustStatus(t, 200).JSON(t, &stopped)
	if stopped.Turn.State != "stopped" || stopped.Turn.Reason != "stopped" {
		t.Fatalf("after Stop: %+v", stopped.Turn)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the worker kept streaming after Stop")
	}
	evs := f.events(th.ID)
	if last := evs[len(evs)-1]; last.Seq != stopped.EventSeq || last.Kind != "turn" {
		t.Fatalf("events after the stopped one: %v (stopped at %d)", evs, stopped.EventSeq)
	}
	// After a reload, the answer stays as far as it was written before
	// Stop: exactly the batches flushed until then, nothing after.
	after := f.get(f.mc, th.ID)
	written := f.deltaText(th.ID)
	if after.Turn.State != "stopped" || len(after.Messages) != 2 || after.Messages[1].Role != "assistant" || !after.Messages[1].Partial ||
		written == "" || after.Messages[1].Text != written || strings.Count(written, "word ") >= 200 || after.Usage.Requests != 1 {
		t.Fatalf("after the worker: %+v (written before Stop: %q)", after, written)
	}
	if n := f.count(`SELECT count(*) FROM ai_requests r JOIN ai_assistant_steps s ON s.id = r.id WHERE s.turn_id = $1 AND r.outcome = 'cancelled'`, turn) +
		f.count(`SELECT count(*) FROM ai_budget_reservations b JOIN ai_assistant_steps s ON s.id = b.id WHERE s.turn_id = $1 AND b.state = 'reserved'`, turn)*10 +
		f.count(`SELECT count(*) FROM ai_assistant_slots`)*100; n != 1 {
		t.Fatalf("accounting %d: want one cancelled request, no open reservation, no slot", n)
	}
	if n := f.count(`SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'ai_assistant_turn.cancelled' AND metadata->>'turn_id' = $2`, f.ws.ID, turn.String()); n != 1 {
		t.Fatalf("%d audit entries of Stop", n)
	}
}

// Stop after the provider answered and before the checkpoint: the answer
// is accounted, but neither saved nor told as done.
func TestAssistantStopAfterAnswer(t *testing.T) {
	f := newFixture(t)
	th := f.thread(f.mc, map[string]any{})
	turn := f.send(f.mc, th.ID, "Quick question").Turn.ID
	w := f.worker()
	var stopped assistant.AIThread
	assistant.SetAfterAnswer(w, func(uuid.UUID) {
		f.mc.Do(http.MethodPost, f.base+"/assistant/threads/"+th.ID.String()+"/turns/"+turn.String()+"/stop", nil).MustStatus(t, 200).JSON(t, &stopped)
	})
	if err := w.ExecuteTurn(context.Background(), f.job(turn)); err != nil {
		t.Fatal(err)
	}
	// Stop keeps what was written before it (a partial message, if a batch
	// was flushed); the checkpoint saves nothing.
	after := f.get(f.mc, th.ID)
	if after.Turn.State != "stopped" || after.EventSeq != stopped.EventSeq || len(after.Messages) > 2 ||
		(len(after.Messages) == 2 && (!after.Messages[1].Partial || after.Messages[1].Text != f.deltaText(th.ID))) {
		t.Fatalf("after Stop: %+v (stopped at %d)", after, stopped.EventSeq)
	}
	if n := f.count(`SELECT count(*) FROM ai_requests r JOIN ai_budget_reservations b ON b.id = r.id JOIN ai_assistant_steps s ON s.id = r.id
		WHERE s.turn_id = $1 AND r.outcome = 'cancelled' AND r.usage_known AND b.state = 'settled' AND b.cost_micros = 180`, turn); n != 1 {
		t.Fatalf("%d settled, cancelled requests", n)
	}
}

// workerOn is a turn worker of another process (the worker role) on its
// own pool.
func (f *fixture) workerOn(pool *pgxpool.Pool) *assistant.Service {
	q := store.New(pool)
	log := f.env.App.Log
	a := audit.New(q, log)
	return &assistant.Service{Pool: pool, Q: q, Audit: a, Box: f.env.App.Secrets, Log: log,
		AI: &ai.Service{Pool: pool, Q: q, Audit: a, Box: f.env.App.Secrets, Log: log, AllowPrivate: func() bool { return true },
			BrandKit: &brandkit.Service{Pool: pool, Q: q, Audit: a}}}
}

// otherPool is a second connection pool to the test's database, as a
// separate worker process has.
func (f *fixture) otherPool() *pgxpool.Pool {
	f.t.Helper()
	cfg, err := pgxpool.ParseConfig(testdb.URL(f.env.App.Pool))
	if err != nil {
		f.t.Fatal(err)
	}
	cfg.MaxConns = 4
	db.Configure(cfg)
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		f.t.Fatal(err)
	}
	f.t.Cleanup(pool.Close)
	return pool
}

// A worker that dies while it streams (its process and connections gone)
// leaves its lease to lapse: the sweep fails the turn as interrupted,
// keeps what it wrote and leaves the request's hold uncertain.
func TestAssistantWorkerDeath(t *testing.T) {
	f := newFixture(t)
	th := f.thread(f.mc, map[string]any{})
	turn := f.send(f.mc, th.ID, "Explain the welcome series").Turn.ID
	release := make(chan struct{})
	defer close(release)
	f.streams.set(held(release, "Partial answer "))
	pool := f.otherPool()
	dead := f.workerOn(pool)
	ctx, kill := context.WithCancel(context.Background())
	done := make(chan error, 1)
	args := f.job(turn)
	go func() { done <- dead.ExecuteTurn(ctx, args) }()
	f.waitKind(th.ID, "delta")
	// The process dies: its connections go, then its goroutines.
	pool.Close()
	kill()
	if err := <-done; err == nil {
		t.Fatal("the dead worker saved its turn")
	}
	// Before the lease lapses, the sweep leaves the turn alone.
	if n, err := f.worker().Sweep(context.Background()); err != nil || n != 0 {
		t.Fatalf("an early sweep: %d, %v", n, err)
	}
	sweeper := f.worker()
	sweeper.Now = func() time.Time { return time.Now().Add(2 * time.Minute) }
	if n, err := sweeper.Sweep(context.Background()); err != nil || n != 1 {
		t.Fatalf("the sweep: %d, %v", n, err)
	}
	after := f.get(f.mc, th.ID)
	if after.Turn.State != "failed" || after.Turn.Reason != "interrupted" || len(after.Messages) != 2 || after.Messages[1].Text != "Partial answer " ||
		!after.Messages[1].Partial || after.Usage.UncertainRequests != 1 {
		t.Fatalf("after the sweep: %+v", after)
	}
	if n := f.count(`SELECT count(*) FROM ai_budget_reservations b JOIN ai_assistant_steps s ON s.id = b.id JOIN ai_requests r ON r.id = b.id
		WHERE s.turn_id = $1 AND b.state = 'uncertain' AND s.state = 'uncertain' AND NOT r.usage_known`, turn) + f.count(`SELECT count(*) FROM ai_assistant_slots`)*10; n != 1 {
		t.Fatalf("the hold: %d", n)
	}
}

// A provider error after the answer began: the turn fails, keeps what
// arrived, and its hold stays uncertain (the provider may bill for it).
func TestAssistantProviderErrorAfterDelta(t *testing.T) {
	f := newFixture(t)
	th := f.thread(f.mc, map[string]any{})
	turn := f.send(f.mc, th.ID, "Summarize the report").Turn.ID
	f.streams.set(func(_ context.Context, c *chunks) {
		c.piece("Half an ans")
		c.data(map[string]any{"error": map[string]any{"message": "The server had an error.", "type": "server_error"}})
	})
	if err := f.worker().ExecuteTurn(context.Background(), f.job(turn)); err != nil {
		t.Fatal(err)
	}
	after := f.get(f.mc, th.ID)
	if after.Turn.State != "failed" || after.Turn.Reason != "provider_error" || len(after.Messages) != 2 || after.Messages[1].Text != "Half an ans" ||
		!after.Messages[1].Partial || after.Usage.UncertainRequests != 1 {
		t.Fatalf("after the error: %+v", after)
	}
	if n := f.count(`SELECT count(*) FROM ai_budget_reservations b JOIN ai_assistant_steps s ON s.id = b.id WHERE s.turn_id = $1 AND b.state = 'uncertain' AND s.state = 'uncertain'`, turn); n != 1 {
		t.Fatal("the hold isn't uncertain")
	}
}

// A person moved to Viewer while their turn answers loses it within about
// a second: the stream is cut and the turn fails on access.
func TestAssistantAccessLostMidStream(t *testing.T) {
	f := newFixture(t)
	th := f.thread(f.mc, map[string]any{})
	turn := f.send(f.mc, th.ID, "Write a long piece").Turn.ID
	f.streams.set(drip("word ", 100*time.Millisecond, 100))
	done := make(chan error, 1)
	args := f.job(turn)
	go func() { done <- f.worker().ExecuteTurn(context.Background(), args) }()
	f.waitKind(th.ID, "delta")
	if _, err := f.env.App.Pool.Exec(context.Background(), `UPDATE workspace_members SET role_id = (SELECT id FROM roles WHERE workspace_id = $1 AND key = 'viewer')
		WHERE workspace_id = $1 AND user_id = $2`, f.ws.ID, f.member); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the turn kept answering")
	}
	if d := time.Since(start); d > 2500*time.Millisecond {
		t.Fatalf("cut after %s", d)
	}
	if n := f.count(`SELECT count(*) FROM ai_assistant_turns WHERE id = $1 AND state = 'failed' AND reason = 'access_lost'`, turn); n != 1 {
		t.Fatal("the turn didn't fail on access")
	}
}

// A stopped request's usage is unknown: it counts at its reserved maximum
// against the thread's ceilings, and the thread goes on. The next request
// is bounded by what that leaves; a thread the maxima fill refuses with
// thread_cap; reconciling the hold frees the difference.
func TestAssistantUncertainHoldCounts(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	th := f.thread(f.mc, map[string]any{})
	first := f.send(f.mc, th.ID, "Tell me a long story").Turn.ID
	f.streams.set(drip("word ", 50*time.Millisecond, 200))
	done := make(chan error, 1)
	args := f.job(first)
	go func() { done <- f.worker().ExecuteTurn(ctx, args) }()
	f.waitKind(th.ID, "delta")
	f.mc.Do(http.MethodPost, f.base+"/assistant/threads/"+th.ID.String()+"/turns/"+first.String()+"/stop", nil).MustStatus(t, 200)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	var hold uuid.UUID
	var heldCost, heldTokens, input int64
	if err := f.env.App.Pool.QueryRow(ctx, `SELECT b.id, b.cost_micros, b.input_tokens + b.output_tokens, b.input_tokens FROM ai_budget_reservations b
		JOIN ai_assistant_steps s ON s.id = b.id WHERE s.turn_id = $1 AND b.state = 'uncertain'`, first).Scan(&hold, &heldCost, &heldTokens, &input); err != nil {
		t.Fatalf("the stopped request's hold: %v", err)
	}
	if u := f.get(f.mc, th.ID).Usage; u.UncertainRequests != 1 || u.UncertainCostMicros != heldCost || u.UncertainTokens != heldTokens || u.CostMicros != heldCost {
		t.Fatalf("the usage: %+v, held %d µ$", u, heldCost)
	}
	limit := func(micros int64) {
		if _, err := f.env.App.Pool.Exec(ctx, `UPDATE ai_assistant_threads SET cost_limit_micros = $2 WHERE id = $1`, th.ID, micros); err != nil {
			t.Fatal(err)
		}
	}
	run := func(text string) uuid.UUID {
		turn := f.send(f.mc, th.ID, text).Turn.ID
		if err := f.worker().ExecuteTurn(ctx, f.job(turn)); err != nil {
			t.Fatal(err)
		}
		return turn
	}
	// Room for a short answer only, once the hold counts at its maximum.
	ceiling := heldCost + input + 3000
	limit(ceiling)
	release := make(chan struct{})
	f.streams.set(held(release, "Short."))
	second := f.send(f.mc, th.ID, "A short one").Turn.ID
	args = f.job(second)
	go func() { done <- f.worker().ExecuteTurn(ctx, args) }()
	// The reservation as it was made (settling replaces it with the usage).
	var in, out int64
	waitFor(t, "the next request's reservation", func() bool {
		return f.env.App.Pool.QueryRow(ctx, `SELECT b.input_tokens, b.output_tokens FROM ai_budget_reservations b JOIN ai_assistant_steps s ON s.id = b.id
			WHERE s.turn_id = $1 AND b.state = 'reserved'`, second).Scan(&in, &out) == nil
	})
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if want := (ceiling - heldCost - in) / 2; want >= 4000 || out != want {
		t.Fatalf("the next request reserved %d output tokens, want %d (the thread's room with the hold at its maximum)", out, want)
	}
	if n := f.count(`SELECT count(*) FROM ai_assistant_turns WHERE id = $1 AND state = 'done'`, second); n != 1 {
		t.Fatal("the next turn didn't answer")
	}
	// The maxima fill the ceiling: refused.
	limit(heldCost + 180 + 100)
	if third := run("Another"); f.count(`SELECT count(*) FROM ai_assistant_turns WHERE id = $1 AND state = 'failed' AND reason = 'thread_cap'`, third) != 1 {
		t.Fatal("a thread full of uncertain maxima took another request")
	}
	// Reconciled with what the provider billed: the difference is free.
	if _, err := f.env.App.Pool.Exec(ctx, `UPDATE ai_budget_reservations SET created_at = now() - interval '11 minutes' WHERE id = $1`, hold); err != nil {
		t.Fatal(err)
	}
	f.oc.Do(http.MethodPost, f.base+"/budget-holds/"+hold.String()+"/reconcile", map[string]any{"inputTokens": 100, "outputTokens": 10, "confirmed": true}).MustStatus(t, 204)
	if u := f.get(f.mc, th.ID).Usage; u.UncertainRequests != 0 || u.CostMicros != 120+180 {
		t.Fatalf("after reconciling: %+v", u)
	}
	if fourth := run("Now?"); f.count(`SELECT count(*) FROM ai_assistant_turns WHERE id = $1 AND state = 'done'`, fourth) != 1 {
		t.Fatal("the reconciled thread refused")
	}
}

// Many turns starting at once don't exhaust the database pool: a claim
// holds one connection (and the workspace's configuration lock) and never
// waits for a second one, so turns more than the pool's connections all
// answer (B6's load test found claims waiting on each other for minutes).
func TestAssistantTurnsShareThePool(t *testing.T) {
	f := newFixture(t)
	f.setAssistant(ai.AIAssistantSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}, PerPerson: 10, PerWorkspace: 10})
	n := int(f.env.App.Cfg.Database.MaxConns) + 2
	jobs := make([]assistant.TurnArgs, n)
	for i := range jobs {
		th := f.thread(f.mc, map[string]any{})
		jobs[i] = f.job(f.send(f.mc, th.ID, "Hello").Turn.ID)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	errs := make(chan error, n)
	for _, args := range jobs {
		go func() {
			for {
				err := f.worker().ExecuteTurn(ctx, args)
				if !isSnooze(err) {
					errs <- err
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		}()
	}
	for range n {
		if err := <-errs; err != nil {
			t.Fatalf("a turn of %d at once: %v", n, err)
		}
	}
	if got := f.count(`SELECT count(*) FROM ai_assistant_turns t JOIN ai_assistant_threads h ON h.id = t.thread_id WHERE h.workspace_id = $1 AND t.state = 'done'`, f.ws.ID); got != n {
		t.Fatalf("%d of %d turns done", got, n)
	}
}
