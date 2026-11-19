package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/store"
)

// A thread's stream events. Each is numbered (its SSE id) without gaps in
// the order it committed: under the thread's lock, in the transaction of
// the change it tells. Their data is sealed at rest; NOTIFY carries only
// "thread:seq" (comma-separated pairs for a group of deltas), to wake the
// streams that then read the events themselves.

// AIThreadTurnEvent: a turn was queued, began answering, stopped, failed
// or was done.
type AIThreadTurnEvent struct {
	Turn AIThreadTurn `json:"turn"`
}

// AIThreadDeltaEvent is a piece of a turn's answer as it arrives.
type AIThreadDeltaEvent struct {
	TurnID uuid.UUID `json:"turnId"`
	Text   string    `json:"text"`
}

// AIThreadMessageEvent is a message saved in the thread: the person's, or
// the assistant's whole answer once it ended (it replaces the deltas).
type AIThreadMessageEvent struct {
	Message AIThreadMessage `json:"message"`
}

// AIThreadToolEvent is a tool call of a turn, once it ran.
type AIThreadToolEvent struct {
	TurnID uuid.UUID       `json:"turnId"`
	Tool   AIThreadToolRow `json:"tool"`
}

// AIThreadUsageEvent is the thread's usage after a request was accounted.
type AIThreadUsageEvent struct {
	Usage AIThreadUsage `json:"usage"`
}

// AIThreadSnapshotEvent is the whole thread, sent instead of events the
// stream no longer has (or first, without a cursor). Its id is the last
// event it includes.
type AIThreadSnapshotEvent struct {
	Thread AIThread `json:"thread"`
}

// AIThreadHeartbeat says the stream is alive while nothing happens.
type AIThreadHeartbeat struct {
	At time.Time `json:"at"`
}

// AIThreadStreamProblem ends a stream the caller may not or can no longer
// read: the thread isn't theirs or is gone, they lost ai:use, or the
// assistant was switched off. A client doesn't reconnect after it.
type AIThreadStreamProblem struct {
	Status int    `json:"status"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// Event kinds as the events table keeps them.
const (
	kindTurn    = "turn"
	kindDelta   = "delta"
	kindMessage = "message"
	kindUsage   = "usage"
	kindTool    = "tool"
)

// eventAD is the associated data an event's payload is sealed with.
func eventAD(thread uuid.UUID, seq int64) string {
	return "ai_assistant_event:" + thread.String() + ":" + strconv.FormatInt(seq, 10)
}

// appendEvent adds an event to a thread the caller has locked and wakes its
// streams once the transaction commits.
func (s *Service) appendEvent(ctx context.Context, q *store.Queries, thread uuid.UUID, turn *uuid.UUID, kind string, data any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	seq, err := q.AIAssistantEventSeq(ctx, thread)
	if err != nil {
		return err
	}
	return s.insertEvent(ctx, q, thread, seq, turn, kind, raw)
}

// insertEvent seals and adds event seq, which the caller numbered under the
// thread's lock; its streams wake once the transaction commits.
func (s *Service) insertEvent(ctx context.Context, q *store.Queries, thread uuid.UUID, seq int64, turn *uuid.UUID, kind string, raw []byte) error {
	sealed, err := s.Box.Seal(raw, eventAD(thread, seq))
	if err != nil {
		return err
	}
	return q.AIAssistantEventInsert(ctx, store.AIAssistantEventInsertParams{ThreadID: thread, Seq: seq, TurnID: turn, Kind: kind, PayloadSealed: sealed,
		CreatedAt: s.now()})
}

// turnEvent tells a turn's new state.
func (s *Service) turnEvent(ctx context.Context, q *store.Queries, t store.AiAssistantTurn) error {
	return s.appendEvent(ctx, q, t.ThreadID, &t.ID, kindTurn, AIThreadTurnEvent{Turn: turnView(t)})
}

// usageEvent tells the thread's usage now.
func (s *Service) usageEvent(ctx context.Context, q *store.Queries, t store.AiAssistantThread, turn uuid.UUID) error {
	used, err := q.AIBudgetThread(ctx, &t.ID)
	if err != nil {
		return err
	}
	return s.appendEvent(ctx, q, t.ID, &turn, kindUsage, AIThreadUsageEvent{Usage: usageOf(t, used)})
}

// openEvent reads an event's data.
func (s *Service) openEvent(thread uuid.UUID, seq int64, sealed []byte, v any) error {
	raw, err := s.Box.Open(sealed, eventAD(thread, seq))
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

// answerSoFar is what a turn's deltas wrote up to an event and no message
// saved yet: the answer of its current request (an earlier request's is in
// its message).
func (s *Service) answerSoFar(ctx context.Context, q *store.Queries, thread, turn uuid.UUID, upto int64) (string, error) {
	rows, err := q.AIAssistantTurnDeltas(ctx, store.AIAssistantTurnDeltasParams{ThreadID: thread, TurnID: &turn, Upto: upto})
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, r := range rows {
		if r.Kind == kindMessage {
			b.Reset()
			continue
		}
		var d AIThreadDeltaEvent
		if err := s.openEvent(thread, r.Seq, r.PayloadSealed, &d); err != nil {
			return "", err
		}
		b.WriteString(d.Text)
	}
	return b.String(), nil
}

// Deltas are written in batches: the answer's first piece at once (the
// person sees the answer begin), then once 2 KB arrived, or 100 ms after
// the first piece of a batch. A batch due goes to the process's group
// writer (deltaGroup), which commits the batches of every answering turn
// together.
const (
	batchBytes = 2 << 10
	batchWait  = 100 * time.Millisecond
)

// errFenced: the turn was stopped, failed by the sweep, or its thread
// deleted since the worker took it; nothing more of it is written.
var errFenced = errors.New("assistant: the turn changed")

// deltaWriter collects a turn's answer as it arrives and hands its batches
// to the group writer, which writes each under the thread's lock and the
// turn's fence.
type deltaWriter struct {
	s                *Service
	g                *deltaGroup
	ws, thread, turn uuid.UUID
	version          int64
	// stop cancels the stream when a batch can't be written.
	stop func(error)

	mu    sync.Mutex
	buf   strings.Builder // not written yet
	all   strings.Builder // everything that arrived
	timer *time.Timer
	err   error
	// firstAt is when the first piece arrived, committedAt when it was
	// committed (the metrics' time to first token).
	firstAt, committedAt time.Time
	// writing is held while a group writes a batch of this writer's: take
	// waits for its outcome.
	writing sync.Mutex
	// settled closes once the first batch was written or can't be: the
	// first piece's emit waits for it, as the person waits for it.
	settled   chan struct{}
	settledAt sync.Once
}

func (s *Service) deltaWriter(ws, thread, turn uuid.UUID, version int64, stop func(error)) *deltaWriter {
	return &deltaWriter{s: s, g: groupOf(s.Pool), ws: ws, thread: thread, turn: turn, version: version, stop: stop, settled: make(chan struct{})}
}

// settle says the first batch's outcome is known.
func (w *deltaWriter) settle() { w.settledAt.Do(func() { close(w.settled) }) }

// emit takes a piece of the answer. Tool calls aren't offered yet, so their
// pieces are ignored (the response says the model asked for one).
func (w *deltaWriter) emit(d llm.Delta) error {
	w.mu.Lock()
	if w.err != nil {
		err := w.err
		w.mu.Unlock()
		return err
	}
	if d.Text == "" {
		w.mu.Unlock()
		return nil
	}
	w.buf.WriteString(d.Text)
	w.all.WriteString(d.Text)
	first := w.firstAt.IsZero()
	if first {
		w.firstAt = time.Now()
	}
	due := first || w.buf.Len() >= batchBytes
	if !due && w.timer == nil {
		w.timer = time.AfterFunc(batchWait, func() { w.g.ready(w, false) })
	}
	w.mu.Unlock()
	if due {
		w.g.ready(w, first)
	}
	if !first {
		return nil
	}
	// The first piece is committed before the answer goes on, as when each
	// batch was its own transaction (the time to first token is measured
	// from it).
	t := time.NewTimer(10 * time.Second)
	defer t.Stop()
	select {
	case <-w.settled:
	case <-t.C:
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.err
}

// take returns what wasn't written yet and stops batching: the checkpoint
// writes it. A batch the group is writing is first written or not.
func (w *deltaWriter) take() string {
	w.writing.Lock()
	defer w.writing.Unlock()
	w.mu.Lock()
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	if w.err == nil {
		w.err = errFenced
	}
	rest := w.buf.String()
	w.buf.Reset()
	w.mu.Unlock()
	w.g.forget(w)
	w.settle()
	return rest
}

// first is when the answer's first piece arrived and was committed (zero
// when none was).
func (w *deltaWriter) first() (time.Time, time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.firstAt, w.committedAt
}

// text is everything that arrived.
func (w *deltaWriter) text() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.all.String()
}
