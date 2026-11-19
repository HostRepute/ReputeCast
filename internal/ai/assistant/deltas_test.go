package assistant_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/ai/assistant"
)

// answering starts n threads of the member, each with a turn marked
// answering as a worker's claim does, for the group writer.
func (f *fixture) answering(n int) []assistant.DeltaBatch {
	f.t.Helper()
	out := []assistant.DeltaBatch{}
	for range n {
		th := f.thread(f.mc, map[string]any{})
		turn := f.send(f.mc, th.ID, "Hello").Turn.ID
		var version int64
		if err := f.env.App.Pool.QueryRow(context.Background(), `UPDATE ai_assistant_turns SET state = 'answering', version = version + 1, started_at = now()
			WHERE id = $1 RETURNING version`, turn).Scan(&version); err != nil {
			f.t.Fatal(err)
		}
		out = append(out, assistant.DeltaBatch{Workspace: f.ws.ID, Thread: th.ID, Turn: turn, Version: version})
	}
	return out
}

// texts are bs with a text each.
func texts(bs []assistant.DeltaBatch, text ...string) []assistant.DeltaBatch {
	out := slices.Clone(bs)
	for i := range out {
		out[i].Text = text[i]
	}
	return out
}

// gapFree fails unless a thread's events are numbered 1 to its last
// number without a gap.
func (f *fixture) gapFree(thread uuid.UUID) {
	f.t.Helper()
	var last int64
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT event_seq FROM ai_assistant_threads WHERE id = $1`, thread).Scan(&last); err != nil {
		f.t.Fatal(err)
	}
	evs := f.events(thread)
	for i, e := range evs {
		if e.Seq != int64(i)+1 {
			f.t.Fatalf("thread %s: events %v", thread, evs)
		}
	}
	if int64(len(evs)) != last {
		f.t.Fatalf("thread %s: %d events, numbered up to %d", thread, len(evs), last)
	}
}

// A group in which one thread's turn was stopped since its last batch:
// that batch isn't written and its turn is told (fenced), the others' are,
// in the same transaction, and every thread stays numbered without gaps.
func TestAssistantDeltaGroupFencesOne(t *testing.T) {
	f := newFixture(t)
	bs := f.answering(3)
	w := f.worker()
	if got := assistant.FlushDeltas(w, texts(bs, "a1 ", "b1 ", "c1 ")); !slices.Equal(got, []string{"written", "written", "written"}) {
		t.Fatalf("the first group: %v", got)
	}
	var stopped assistant.AIThread
	f.mc.Do(http.MethodPost, f.base+"/assistant/threads/"+bs[1].Thread.String()+"/turns/"+bs[1].Turn.String()+"/stop", nil).MustStatus(t, 200).JSON(t, &stopped)
	if got := assistant.FlushDeltas(w, texts(bs, "a2", "b2", "c2")); !slices.Equal(got, []string{"written", "fenced", "written"}) {
		t.Fatalf("the group after Stop: %v", got)
	}
	if a, b, c := f.deltaText(bs[0].Thread), f.deltaText(bs[1].Thread), f.deltaText(bs[2].Thread); a != "a1 a2" || b != "b1 " || c != "c1 c2" {
		t.Fatalf("written: %q %q %q", a, b, c)
	}
	evs := f.events(bs[1].Thread)
	if last := evs[len(evs)-1]; last.Seq != stopped.EventSeq || last.Kind != "turn" {
		t.Fatalf("the stopped thread's events %v (stopped at %d)", evs, stopped.EventSeq)
	}
	if after := f.get(f.mc, bs[1].Thread); len(after.Messages) != 2 || after.Messages[1].Text != "b1 " || !after.Messages[1].Partial {
		t.Fatalf("the stopped answer: %+v", after.Messages)
	}
	for _, b := range bs {
		f.gapFree(b.Thread)
	}
}

// A thread another transaction holds (Stop, the sweep, a model's removal,
// another process) doesn't hold up the group: its batch is skipped and
// written by a later group once the lock is gone; the others' are written
// at once.
func TestAssistantDeltaGroupSkipsLockedThread(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	bs := f.answering(3)
	tx, err := f.env.App.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT 1 FROM ai_assistant_threads WHERE id = $1 FOR UPDATE`, bs[1].Thread); err != nil {
		t.Fatal(err)
	}
	done := make(chan []string, 1)
	go func() { done <- assistant.FlushDeltas(f.worker(), texts(bs, "a", "b", "c")) }()
	select {
	case got := <-done:
		if !slices.Equal(got, []string{"written", "skipped", "written"}) {
			t.Fatalf("the group: %v", got)
		}
	case <-time.After(3 * time.Second):
		_ = tx.Rollback(ctx)
		<-done
		t.Fatal("the group waited for a thread another transaction holds")
	}
	if a, b, c := f.deltaText(bs[0].Thread), f.deltaText(bs[1].Thread), f.deltaText(bs[2].Thread); a != "a" || b != "" || c != "c" {
		t.Fatalf("written while one thread is held: %q %q %q", a, b, c)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the skipped batch", func() bool { return f.deltaText(bs[1].Thread) == "b" })
	for _, b := range bs {
		f.gapFree(b.Thread)
	}
}

// One batch that can't be written (its thread's next number is taken)
// fails the group's transaction; the others are then written one by one,
// and only that turn's stream is cut.
func TestAssistantDeltaGroupKeepsOthersOnError(t *testing.T) {
	f := newFixture(t)
	bs := f.answering(3)
	if _, err := f.env.App.Pool.Exec(context.Background(), `INSERT INTO ai_assistant_events (thread_id, seq, kind, payload_sealed, created_at)
		SELECT id, event_seq + 1, 'delta', '\x00', now() FROM ai_assistant_threads WHERE id = $1`, bs[1].Thread); err != nil {
		t.Fatal(err)
	}
	if got := assistant.FlushDeltas(f.worker(), texts(bs, "a", "b", "c")); !slices.Equal(got, []string{"written", "failed", "written"}) {
		t.Fatalf("the group: %v", got)
	}
	if a, c := f.deltaText(bs[0].Thread), f.deltaText(bs[2].Thread); a != "a" || c != "c" {
		t.Fatalf("the others: %q %q", a, c)
	}
	f.gapFree(bs[0].Thread)
	f.gapFree(bs[2].Thread)
}

// Concurrent turns keep their order through the group writer, also while
// one thread is held after its first piece (its batches skipped, then
// written before what came after): each thread's deltas and what was left
// at the end are its pieces in order, numbered without gaps.
func TestAssistantDeltaGroupKeepsOrder(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	bs := f.answering(8)
	w := f.worker()
	const pieces = 400
	rest := make([]string, len(bs))
	var wg sync.WaitGroup
	for i, b := range bs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d := assistant.NewDeltaWriter(w, b)
			for n := range pieces {
				if err := d.Emit(fmt.Sprintf("%d,", n)); err != nil {
					t.Error(err)
					return
				}
				time.Sleep(time.Duration(1+n%3) * time.Millisecond)
			}
			rest[i] = d.Take()
		}()
	}
	held := bs[3].Thread
	waitFor(t, "the held thread's first piece", func() bool { return f.deltaText(held) != "" })
	tx, err := f.env.App.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `SELECT 1 FROM ai_assistant_threads WHERE id = $1 FOR UPDATE`, held); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	var want strings.Builder
	for n := range pieces {
		fmt.Fprintf(&want, "%d,", n)
	}
	for i, b := range bs {
		if got := f.deltaText(b.Thread) + rest[i]; got != want.String() {
			t.Fatalf("thread %d: %q", i, got)
		}
		f.gapFree(b.Thread)
	}
}

// An answer's first piece is written at once; a later batch waits for the
// group's window, so it is written with the others due by then.
func TestAssistantDeltaGroupFirstPieceAtOnce(t *testing.T) {
	f := newFixture(t)
	defer assistant.SetDeltaGroupWait(2 * time.Second)()
	b := f.answering(1)[0]
	d := assistant.NewDeltaWriter(f.worker(), b)
	start := time.Now()
	if err := d.Emit("First "); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the first piece", func() bool { return f.deltaText(b.Thread) == "First " })
	if took := time.Since(start); took > time.Second {
		t.Fatalf("the first piece took %s", took)
	}
	// A full batch: due at once, written after the group's window.
	full := strings.Repeat("x", 2100)
	if err := d.Emit(full); err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	if got := f.deltaText(b.Thread); got != "First " {
		t.Fatalf("a later batch didn't wait for the group: %d bytes", len(got))
	}
	waitFor(t, "the later batch", func() bool { return f.deltaText(b.Thread) == "First "+full })
	if rest := d.Take(); rest != "" {
		t.Fatalf("left: %q", rest)
	}
	f.gapFree(b.Thread)
}

// One group's NOTIFY names several threads: each thread's open stream
// receives its delta.
func TestAssistantDeltaGroupWakesEveryStream(t *testing.T) {
	f := newFixture(t)
	bs := f.answering(3)
	streams := make([]*stream, len(bs))
	for i, b := range bs {
		streams[i] = f.openStream(f.mc, b.Thread, "")
		if ev := streams[i].next(t, 10*time.Second); ev.Event != "snapshot" {
			t.Fatalf("first event %s", ev.Event)
		}
	}
	if got := assistant.FlushDeltas(f.worker(), texts(bs, "a", "b", "c")); !slices.Equal(got, []string{"written", "written", "written"}) {
		t.Fatalf("the group: %v", got)
	}
	for i, s := range streams {
		if ev := s.next(t, 3*time.Second); ev.Event != "delta" || !strings.Contains(string(ev.Data), `"text":"`+string(rune('a'+i))+`"`) {
			t.Fatalf("stream %d: %s %s", i, ev.Event, ev.Data)
		}
	}
}

// Stop and a model's removal while turns stream through the group writer:
// nothing deadlocks, every worker finishes, and every thread stays
// numbered without gaps.
func TestAssistantDeltaGroupWithStopAndRemoval(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	f.setAssistant(ai.AIAssistantSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}, PerPerson: 10, PerWorkspace: 10})
	var s struct {
		Accounts []struct {
			ID     uuid.UUID
			Models []struct {
				ID    uuid.UUID
				Model string
			}
		}
	}
	f.oc.Do(http.MethodPost, f.base+"/accounts/"+f.accountID.String()+"/models", map[string]any{"model": "model-b", "inputPrice": 1, "outputPrice": 2}).
		MustStatus(t, 200).JSON(t, &s)
	var modelB uuid.UUID
	for _, a := range s.Accounts {
		for _, m := range a.Models {
			if m.Model == "model-b" {
				modelB = m.ID
			}
		}
	}
	f.passTools(f.oc, "/accounts/"+f.accountID.String()+"/models/"+modelB.String())
	f.streams.set(drip("word ", 5*time.Millisecond, 400))
	type running struct {
		thread, turn uuid.UUID
		done         chan error
	}
	var turns []running
	for i := range 6 {
		in := map[string]any{}
		if i%2 == 0 {
			in["modelId"] = modelB
		}
		th := f.thread(f.mc, in)
		turn := f.send(f.mc, th.ID, "Tell me a long story").Turn.ID
		r := running{thread: th.ID, turn: turn, done: make(chan error, 1)}
		args := f.job(turn)
		go func() { r.done <- f.worker().ExecuteTurn(ctx, args) }()
		turns = append(turns, r)
	}
	for _, r := range turns {
		f.waitKind(r.thread, "delta")
	}
	var wg sync.WaitGroup
	statuses := make([]int, 3)
	for i, r := range turns[:2] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses[i] = f.mc.Do(http.MethodPost, f.base+"/assistant/threads/"+r.thread.String()+"/turns/"+r.turn.String()+"/stop", nil).Status
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		statuses[2] = f.oc.Do(http.MethodDelete, f.base+"/accounts/"+f.accountID.String()+"/models/"+modelB.String(), nil).Status
	}()
	wg.Wait()
	if !slices.Equal(statuses, []int{200, 200, 200}) {
		t.Fatalf("Stop, Stop and the removal: %v", statuses)
	}
	for i, r := range turns {
		select {
		case err := <-r.done:
			if err != nil && !isSnooze(err) {
				t.Fatalf("turn %d: %v", i, err)
			}
		case <-time.After(30 * time.Second):
			t.Fatalf("turn %d didn't finish", i)
		}
		f.gapFree(r.thread)
	}
}
