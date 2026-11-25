package assistant_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/ai/assistant"
	"github.com/hostrepute/reputecast/internal/apptest"
)

// fakeStreams is the fake model's streamed answers: each streaming request
// runs the script set last (by default, "Hello there.").
type fakeStreams struct {
	mu     sync.Mutex
	script func(ctx context.Context, w *chunks)
	// queued scripts answer the next streaming requests, in order, before
	// script; each sees the request's body.
	queued   []func(ctx context.Context, w *chunks, body []byte)
	bodies   [][]byte
	requests atomic.Int32
	// started gets a value as each streaming request arrives.
	started chan struct{}
}

// then queues scripts for the next streaming requests.
func (st *fakeStreams) then(scripts ...func(ctx context.Context, w *chunks, body []byte)) {
	st.mu.Lock()
	st.queued = append(st.queued, scripts...)
	st.mu.Unlock()
}

// record keeps a request's body (streamed or not), in order.
func (st *fakeStreams) record(body []byte) {
	st.mu.Lock()
	st.bodies = append(st.bodies, body)
	st.mu.Unlock()
}

// streamed are the streaming requests' bodies, decoded.
func (st *fakeStreams) streamed(t *testing.T) []map[string]any {
	t.Helper()
	st.mu.Lock()
	defer st.mu.Unlock()
	out := []map[string]any{}
	for _, b := range st.bodies {
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		if m["stream"] == true {
			out = append(out, m)
		}
	}
	return out
}

func newFakeStreams() *fakeStreams { return &fakeStreams{started: make(chan struct{}, 64)} }

func (st *fakeStreams) set(script func(ctx context.Context, w *chunks)) {
	st.mu.Lock()
	st.script = script
	st.mu.Unlock()
}

func (st *fakeStreams) serve(w http.ResponseWriter, r *http.Request, body []byte) {
	st.requests.Add(1)
	select {
	case st.started <- struct{}{}:
	default:
	}
	st.mu.Lock()
	st.bodies = append(st.bodies, body)
	script := st.script
	var next func(context.Context, *chunks, []byte)
	if len(st.queued) > 0 {
		next, st.queued = st.queued[0], st.queued[1:]
	}
	st.mu.Unlock()
	if script == nil {
		script = answer("Hello", " there.")
	}
	w.Header().Set("Content-Type", "text/event-stream")
	c := &chunks{w: w, f: w.(http.Flusher)}
	if next != nil {
		next(r.Context(), c, body)
		return
	}
	script(r.Context(), c)
}

// calls streams tool calls (name and JSON arguments, in pairs) and ends
// with finish_reason tool_calls.
func calls(nameArgs ...string) func(context.Context, *chunks, []byte) {
	return func(_ context.Context, c *chunks, _ []byte) {
		for i := 0; i+1 < len(nameArgs); i += 2 {
			c.data(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"tool_calls": []any{map[string]any{
				"index": i / 2, "id": fmt.Sprintf("call_%d", i/2), "type": "function", "function": map[string]any{"name": nameArgs[i], "arguments": nameArgs[i+1]}}}}}}})
		}
		c.data(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": "tool_calls"}}})
		c.data(map[string]any{"choices": []any{}, "usage": map[string]int{"prompt_tokens": 100, "completion_tokens": 20}})
		_, _ = io.WriteString(c.w, "data: [DONE]\n\n")
		c.f.Flush()
	}
}

// says streams a plain answer.
func says(text string) func(context.Context, *chunks, []byte) {
	return func(ctx context.Context, c *chunks, _ []byte) { answer(text)(ctx, c) }
}

// waitStarted waits for n more streaming requests.
func (st *fakeStreams) waitStarted(t *testing.T, n int) {
	t.Helper()
	for range n {
		select {
		case <-st.started:
		case <-time.After(20 * time.Second):
			t.Fatal("no streaming request arrived")
		}
	}
}

// chunks writes an OpenAI-compatible stream.
type chunks struct {
	w http.ResponseWriter
	f http.Flusher
}

func (c *chunks) data(v any) {
	b, _ := json.Marshal(v)
	_, _ = fmt.Fprintf(c.w, "data: %s\n\n", b)
	c.f.Flush()
}

func (c *chunks) piece(text string) {
	c.data(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"content": text}}}})
}

func (c *chunks) end(in, out int) {
	c.data(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": "stop"}}})
	c.data(map[string]any{"choices": []any{}, "usage": map[string]int{"prompt_tokens": in, "completion_tokens": out}})
	_, _ = io.WriteString(c.w, "data: [DONE]\n\n")
	c.f.Flush()
}

// answer streams its pieces and ends.
func answer(pieces ...string) func(context.Context, *chunks) {
	return func(_ context.Context, c *chunks) {
		for _, p := range pieces {
			c.piece(p)
		}
		c.end(120, 30)
	}
}

// held streams first, waits for release (or the request's end), then
// streams rest and ends.
func held(release <-chan struct{}, first string, rest ...string) func(context.Context, *chunks) {
	return func(ctx context.Context, c *chunks) {
		c.piece(first)
		select {
		case <-release:
		case <-ctx.Done():
			return
		}
		answer(rest...)(ctx, c)
	}
}

// drip streams n pieces, one every interval, until the request ends.
func drip(piece string, every time.Duration, n int) func(context.Context, *chunks) {
	return func(ctx context.Context, c *chunks) {
		for range n {
			c.piece(piece)
			select {
			case <-ctx.Done():
				return
			case <-time.After(every):
			}
		}
		c.end(120, 30)
	}
}

// worker is a turn worker as the app builds it: its own copy, so a test
// can tune it.
func (f *fixture) worker() *assistant.Service {
	w := *f.env.App.Assistant()
	w.Hub = nil
	return &w
}

// send sends a message as c and returns the thread.
func (f *fixture) send(c *apptest.Client, thread uuid.UUID, text string) assistant.AIThread {
	f.t.Helper()
	var th assistant.AIThread
	c.Do(http.MethodPost, f.base+"/assistant/threads/"+thread.String()+"/messages", map[string]any{"text": text, "requestKey": strings.ReplaceAll(uuid.NewString(), "-", "")}).
		MustStatus(f.t, 200).JSON(f.t, &th)
	return th
}

// job is a turn's job as it was queued.
func (f *fixture) job(turn uuid.UUID) assistant.TurnArgs {
	f.t.Helper()
	var raw []byte
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT args FROM river_job WHERE kind = 'ai.assistant_turn' AND args->>'turnId' = $1`, turn.String()).Scan(&raw); err != nil {
		f.t.Fatalf("the turn's job: %v", err)
	}
	var args assistant.TurnArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		f.t.Fatal(err)
	}
	return args
}

// get reads a thread as c.
func (f *fixture) get(c *apptest.Client, thread uuid.UUID) assistant.AIThread {
	f.t.Helper()
	var th assistant.AIThread
	c.Do(http.MethodGet, f.base+"/assistant/threads/"+thread.String(), nil).MustStatus(f.t, 200).JSON(f.t, &th)
	return th
}

// event is a row of the events table.
type event struct {
	Seq  int64
	Kind string
}

func (f *fixture) events(thread uuid.UUID) []event {
	f.t.Helper()
	rows, err := f.env.App.Pool.Query(context.Background(), `SELECT seq, kind FROM ai_assistant_events WHERE thread_id = $1 ORDER BY seq`, thread)
	if err != nil {
		f.t.Fatal(err)
	}
	defer rows.Close()
	out := []event{}
	for rows.Next() {
		var e event
		if err := rows.Scan(&e.Seq, &e.Kind); err != nil {
			f.t.Fatal(err)
		}
		out = append(out, e)
	}
	return out
}

// deltaText is what a thread's delta events wrote, in order.
func (f *fixture) deltaText(thread uuid.UUID) string {
	f.t.Helper()
	rows, err := f.env.App.Pool.Query(context.Background(), `SELECT seq, payload_sealed FROM ai_assistant_events WHERE thread_id = $1 AND kind = 'delta' ORDER BY seq`, thread)
	if err != nil {
		f.t.Fatal(err)
	}
	defer rows.Close()
	var b strings.Builder
	for rows.Next() {
		var seq int64
		var sealed []byte
		if err := rows.Scan(&seq, &sealed); err != nil {
			f.t.Fatal(err)
		}
		raw, err := f.env.App.Secrets.Open(sealed, fmt.Sprintf("ai_assistant_event:%s:%d", thread, seq))
		if err != nil {
			f.t.Fatal(err)
		}
		var d assistant.AIThreadDeltaEvent
		if err := json.Unmarshal(raw, &d); err != nil {
			f.t.Fatal(err)
		}
		b.WriteString(d.Text)
	}
	return b.String()
}

// waitFor polls cond until it holds.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// waitKind waits until the thread has an event of kind.
func (f *fixture) waitKind(thread uuid.UUID, kind string) {
	f.t.Helper()
	waitFor(f.t, "a "+kind+" event", func() bool {
		for _, e := range f.events(thread) {
			if e.Kind == kind {
				return true
			}
		}
		return false
	})
}

// isSnooze reports whether a job asked to run again later.
func isSnooze(err error) bool {
	var s *rivertype.JobSnoozeError
	return errors.As(err, &s)
}

// sseEvent is one event a stream sent.
type sseEvent struct {
	ID, Event string
	Data      []byte
}

// readEvent reads the next event (comments and retry lines skipped).
func readEvent(r *bufio.Reader) (sseEvent, error) {
	var ev sseEvent
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return ev, err
		}
		line = strings.TrimRight(line, "\r\n")
		switch {
		case line == "":
			if ev.Event != "" || len(ev.Data) > 0 {
				if ev.Event == "" {
					// SSE's default event, which Huma doesn't name.
					ev.Event = "message"
				}
				return ev, nil
			}
		case strings.HasPrefix(line, "id: "):
			ev.ID = line[4:]
		case strings.HasPrefix(line, "event: "):
			ev.Event = line[7:]
		case strings.HasPrefix(line, "data: "):
			ev.Data = append(ev.Data, line[6:]...)
		}
	}
}

// stream is an open SSE stream.
type stream struct {
	resp *http.Response
	r    *bufio.Reader
	stop context.CancelFunc
}

// openStream opens a thread's stream as c, after cursor ("" for none).
func (f *fixture) openStream(c *apptest.Client, thread uuid.UUID, cursor string) *stream {
	f.t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	path := f.base + "/assistant/threads/" + thread.String() + "/stream"
	if cursor != "" {
		c.Header = http.Header{"Last-Event-Id": {cursor}}
	}
	resp, err := c.Stream(ctx, path) //nolint:bodyclose // s.close closes it at cleanup
	c.Header = nil
	if err != nil {
		cancel()
		f.t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		cancel()
		_ = resp.Body.Close()
		f.t.Fatalf("stream: status %d", resp.StatusCode)
	}
	s := &stream{resp: resp, r: newReader(resp), stop: cancel}
	f.t.Cleanup(s.close)
	return s
}

func newReader(resp *http.Response) *bufio.Reader { return bufio.NewReaderSize(resp.Body, 1<<20) }

func (s *stream) close() {
	s.stop()
	_ = s.resp.Body.Close()
}

// next reads the next event other than a heartbeat, within d.
func (s *stream) next(t *testing.T, d time.Duration) sseEvent {
	t.Helper()
	ev, err := s.read(t, d)
	if err != nil {
		t.Fatalf("stream ended: %v", err)
	}
	return ev
}

// nextOrEnd is next, except that the server ending the stream is false
// rather than a failure, for a reader that resumes from the last event it
// read, as the panel does.
func (s *stream) nextOrEnd(t *testing.T, d time.Duration) (sseEvent, bool) {
	t.Helper()
	ev, err := s.read(t, d)
	if errors.Is(err, io.EOF) {
		return sseEvent{}, false
	}
	if err != nil {
		t.Fatalf("stream ended: %v", err)
	}
	return ev, true
}

// read reads the next event other than a heartbeat, failing the test when
// none comes within d.
func (s *stream) read(t *testing.T, d time.Duration) (sseEvent, error) {
	t.Helper()
	type result struct {
		ev  sseEvent
		err error
	}
	ch := make(chan result, 1)
	go func() {
		for {
			ev, err := readEvent(s.r)
			if err == nil && ev.Event == "heartbeat" {
				continue
			}
			ch <- result{ev, err}
			return
		}
	}()
	select {
	case r := <-ch:
		return r.ev, r.err
	case <-time.After(d):
		s.stop()
		<-ch
		t.Fatalf("no event within %s", d)
	}
	return sseEvent{}, nil
}

// until reads events up to the turn event with state (included).
func (s *stream) until(t *testing.T, state string) []sseEvent {
	t.Helper()
	out := []sseEvent{}
	for {
		ev := s.next(t, 20*time.Second)
		out = append(out, ev)
		if ev.Event == "turn" && bytes.Contains(ev.Data, []byte(`"state":"`+state+`"`)) {
			return out
		}
	}
}
