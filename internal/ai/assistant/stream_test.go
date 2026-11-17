package assistant_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/ai/assistant"
)

// ids are the events' ids, as numbers.
func ids(t *testing.T, evs []sseEvent) []int64 {
	t.Helper()
	out := []int64{}
	for _, ev := range evs {
		n, err := strconv.ParseInt(ev.ID, 10, 64)
		if err != nil {
			t.Fatalf("event %s without an id: %s", ev.Event, ev.Data)
		}
		out = append(out, n)
	}
	return out
}

// consecutive fails unless the ids run from first without a gap.
func consecutive(t *testing.T, got []int64, first int64) {
	t.Helper()
	for i, id := range got {
		if id != first+int64(i) {
			t.Fatalf("ids %v, want consecutive from %d", got, first)
		}
	}
}

// A stream starts with the thread's snapshot, then sends every event once,
// in order; resuming after an event (Last-Event-ID or lastEventId) sends
// exactly the ones after it, and only the owner reads it.
func TestAssistantStreamResumes(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	th := f.thread(f.mc, map[string]any{})
	if resp, err := f.vc.Stream(ctx, f.base+"/assistant/threads/"+th.ID.String()+"/stream"); err != nil || resp.StatusCode != 403 {
		t.Fatalf("a viewer's stream: %v %v", resp.StatusCode, err)
	}
	other := f.openStream(f.cc, th.ID, "")
	if ev := other.next(t, 5*time.Second); ev.Event != "problem" || !json.Valid(ev.Data) {
		t.Fatalf("a colleague's stream: %s %s", ev.Event, ev.Data)
	}
	s := f.openStream(f.mc, th.ID, "")
	first := s.next(t, 5*time.Second)
	var snap assistant.AIThreadSnapshotEvent
	if err := json.Unmarshal(first.Data, &snap); err != nil || first.Event != "snapshot" || first.ID != "0" || snap.Thread.ID != th.ID {
		t.Fatalf("the first event: %s %s %s", first.Event, first.ID, first.Data)
	}
	turn := f.send(f.mc, th.ID, "Three words please").Turn.ID
	f.streams.set(answer("One ", "two ", "three."))
	if err := f.worker().ExecuteTurn(ctx, f.job(turn)); err != nil {
		t.Fatal(err)
	}
	evs := s.until(t, "done")
	consecutive(t, ids(t, evs), 1)
	s.close()
	// Resume after the third event.
	r := f.openStream(f.mc, th.ID, evs[2].ID)
	rest := r.until(t, "done")
	if len(rest) != len(evs)-3 {
		t.Fatalf("resumed with %d events, want %d", len(rest), len(evs)-3)
	}
	for i, ev := range rest {
		if want := evs[3+i]; ev.ID != want.ID || ev.Event != want.Event || string(ev.Data) != string(want.Data) {
			t.Fatalf("resumed event %d: %s %s, want %s %s", i, ev.ID, ev.Data, want.ID, want.Data)
		}
	}
	// From the last event (lastEventId), the next message's first.
	resp, err := f.mc.Stream(ctx, f.base+"/assistant/threads/"+th.ID.String()+"/stream?lastEventId="+evs[len(evs)-1].ID)
	if err != nil {
		t.Fatal(err)
	}
	q := &stream{resp: resp, r: newReader(resp), stop: func() {}}
	defer q.close()
	f.send(f.mc, th.ID, "And one more")
	if ev := q.next(t, 5*time.Second); ev.Event != "message" || ev.ID != strconv.Itoa(len(evs)+1) {
		t.Fatalf("after the last event: %s %s", ev.Event, ev.ID)
	}
}

// A cursor whose events are gone (lost in a crash, or past their 24 hours)
// gets the thread's snapshot instead, with the answers saved; a deleted
// thread's stream ends with a problem.
func TestAssistantStreamSnapshot(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	th := f.thread(f.mc, map[string]any{})
	turn := f.send(f.mc, th.ID, "Hello?").Turn.ID
	if err := f.worker().ExecuteTurn(ctx, f.job(turn)); err != nil {
		t.Fatal(err)
	}
	last := f.get(f.mc, th.ID).EventSeq
	if _, err := f.env.App.Pool.Exec(ctx, `TRUNCATE ai_assistant_events`); err != nil {
		t.Fatal(err)
	}
	for _, cursor := range []string{"1", "999"} {
		s := f.openStream(f.mc, th.ID, cursor)
		ev := s.next(t, 5*time.Second)
		var snap assistant.AIThreadSnapshotEvent
		if err := json.Unmarshal(ev.Data, &snap); err != nil || ev.Event != "snapshot" || ev.ID != strconv.FormatInt(last, 10) || snap.Thread.EventSeq != last ||
			len(snap.Thread.Messages) != 2 || snap.Thread.Messages[1].Text != "Hello there." || snap.Thread.Turn.State != "done" {
			t.Fatalf("after cursor %s: %s %s %s", cursor, ev.Event, ev.ID, ev.Data)
		}
		s.close()
	}
	s := f.openStream(f.mc, th.ID, strconv.FormatInt(last, 10))
	f.mc.Do(http.MethodDelete, f.base+"/assistant/threads/"+th.ID.String(), nil).MustStatus(t, 204)
	if ev := s.next(t, 10*time.Second); ev.Event != "problem" {
		t.Fatalf("a deleted thread's stream: %s %s", ev.Event, ev.Data)
	}
}

// The hub's connection cut by Postgres: streams still get the events,
// polled until the hub listens again.
func TestAssistantStreamListenerLost(t *testing.T) {
	f := newFixture(t)
	th := f.thread(f.mc, map[string]any{})
	s := f.openStream(f.mc, th.ID, "")
	if ev := s.next(t, 5*time.Second); ev.Event != "snapshot" {
		t.Fatalf("first %s", ev.Event)
	}
	hub := f.env.App.Assistant().Hub
	waitFor(t, "the hub to listen", hub.Listening)
	listeners := `SELECT count(*) FROM pg_stat_activity WHERE application_name = 'reputecast-assistant' AND datname = current_database()`
	if n := f.count(`SELECT count(pg_terminate_backend(pid)) FROM pg_stat_activity WHERE application_name = 'reputecast-assistant' AND datname = current_database()`); n != 1 {
		t.Fatalf("%d listeners", n)
	}
	waitFor(t, "the listener to go", func() bool { return f.count(listeners) == 0 })
	f.send(f.mc, th.ID, "Are you there?")
	// Sooner than the stream's own access check (every 5 s) and heartbeat
	// (15 s), which read the events too.
	if ev := s.next(t, 2*time.Second); ev.Event != "message" {
		t.Fatalf("after the listener was cut: %s", ev.Event)
	}
	waitFor(t, "the hub to listen again", func() bool { return hub.Listening() && f.count(listeners) == 1 })
}

// The web role streams what a worker of another process writes: NOTIFY
// reaches it across processes.
func TestAssistantStreamSplitRoles(t *testing.T) {
	f := newFixture(t)
	th := f.thread(f.mc, map[string]any{})
	s := f.openStream(f.mc, th.ID, "")
	if ev := s.next(t, 5*time.Second); ev.Event != "snapshot" {
		t.Fatalf("first %s", ev.Event)
	}
	turn := f.send(f.mc, th.ID, "Split roles?").Turn.ID
	// Each event sooner than the stream's own access check (every 5 s) and
	// heartbeat (15 s), which read the events too.
	for _, want := range []string{"message", "turn"} {
		if ev := s.next(t, 2*time.Second); ev.Event != want {
			t.Fatalf("got %s, want %s", ev.Event, want)
		}
	}
	worker := f.workerOn(f.otherPool())
	if err := worker.ExecuteTurn(context.Background(), f.job(turn)); err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for {
		ev := s.next(t, 2*time.Second)
		got = append(got, ev.Event)
		if ev.Event == "turn" && len(got) > 1 {
			break
		}
	}
	if len(got) < 5 || got[0] != "turn" || got[len(got)-3] != "message" || got[len(got)-2] != "usage" {
		t.Fatalf("events %v", got)
	}
}
