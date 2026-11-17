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
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// A thread's stream events. Each is numbered (its SSE id) without gaps in
// the order it committed: under the thread's lock, in the transaction of
// the change it tells. Their data is sealed at rest; NOTIFY carries only
// "thread:seq", to wake the streams that then read the events themselves.

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
	sealed, err := s.Box.Seal(raw, eventAD(thread, seq))
	if err != nil {
		return err
	}
	if err := q.AIAssistantEventInsert(ctx, store.AIAssistantEventInsertParams{ThreadID: thread, Seq: seq, TurnID: turn, Kind: kind, PayloadSealed: sealed,
		CreatedAt: s.now()}); err != nil {
		return err
	}
	return q.AIAssistantNotify(ctx, thread.String()+":"+strconv.FormatInt(seq, 10))
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
	return s.appendEvent(ctx, q, t.ID, &turn, kindUsage, AIThreadUsageEvent{Usage: AIThreadUsage{ToolCalls: t.ToolCalls, Tokens: used.Tokens,
		CostMicros: used.CostMicros, Requests: used.Requests, UncertainRequests: used.Uncertain}})
}

// openEvent reads an event's data.
func (s *Service) openEvent(thread uuid.UUID, seq int64, sealed []byte, v any) error {
	raw, err := s.Box.Open(sealed, eventAD(thread, seq))
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

// answerSoFar is what a turn's deltas wrote up to an event.
func (s *Service) answerSoFar(ctx context.Context, q *store.Queries, thread, turn uuid.UUID, upto int64) (string, error) {
	rows, err := q.AIAssistantTurnDeltas(ctx, store.AIAssistantTurnDeltasParams{ThreadID: thread, TurnID: &turn, Upto: upto})
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, r := range rows {
		var d AIThreadDeltaEvent
		if err := s.openEvent(thread, r.Seq, r.PayloadSealed, &d); err != nil {
			return "", err
		}
		b.WriteString(d.Text)
	}
	return b.String(), nil
}

// Deltas are written in batches: once 2 KB arrived, or 100 ms after the
// first piece of a batch.
const (
	batchBytes = 2 << 10
	batchWait  = 100 * time.Millisecond
)

// errFenced: the turn was stopped, failed by the sweep, or its thread
// deleted since the worker took it; nothing more of it is written.
var errFenced = errors.New("assistant: the turn changed")

// deltaWriter writes a turn's answer as it arrives, in batches, each in its
// own transaction under the thread's lock and the turn's fence.
type deltaWriter struct {
	s                *Service
	ws, thread, turn uuid.UUID
	version          int64
	// stop cancels the stream when a batch can't be written.
	stop func(error)

	mu    sync.Mutex
	buf   strings.Builder // not written yet
	all   strings.Builder // everything that arrived
	timer *time.Timer
	err   error
	// flushing serializes batches, so they commit in order.
	flushing sync.Mutex
}

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
	full := w.buf.Len() >= batchBytes
	if !full && w.timer == nil {
		w.timer = time.AfterFunc(batchWait, func() { _ = w.flush() })
	}
	w.mu.Unlock()
	if full {
		return w.flush()
	}
	return nil
}

// take returns what wasn't written yet and stops batching: the checkpoint
// writes it.
func (w *deltaWriter) take() string {
	w.flushing.Lock()
	defer w.flushing.Unlock()
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	if w.err == nil {
		w.err = errFenced
	}
	rest := w.buf.String()
	w.buf.Reset()
	return rest
}

// text is everything that arrived.
func (w *deltaWriter) text() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.all.String()
}

func (w *deltaWriter) flush() error {
	w.flushing.Lock()
	defer w.flushing.Unlock()
	w.mu.Lock()
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	text, err := w.buf.String(), w.err
	w.buf.Reset()
	w.mu.Unlock()
	if err != nil || text == "" {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = db.Tx(ctx, w.s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if _, err := q.AIAssistantThreadLockByID(ctx, store.AIAssistantThreadLockByIDParams{WorkspaceID: w.ws, ID: w.thread}); errors.Is(err, pgx.ErrNoRows) {
			return errFenced
		} else if err != nil {
			return err
		}
		f, err := q.AIAssistantTurnFence(ctx, w.turn)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && (f.Version != w.version || f.State != "answering")) {
			return errFenced
		}
		if err != nil {
			return err
		}
		return w.s.appendEvent(ctx, q, w.thread, &w.turn, kindDelta, AIThreadDeltaEvent{TurnID: w.turn, Text: text})
	})
	if err != nil {
		w.mu.Lock()
		if w.err == nil {
			w.err = err
		}
		w.mu.Unlock()
		w.stop(err)
	}
	return err
}
