package assistant

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/store"
)

// A process writes the delta batches of all its answering turns together:
// each turn's writer hands the group a batch once it is due (deltaWriter's
// rules), and the group commits every batch due by then in one
// transaction, with one NOTIFY (or a few) naming their threads. Before, a
// batch was a transaction and a NOTIFY of its own; 200 answering turns made
// some 750 of each a second (B6's load test), and transactions that notify
// commit one at a time.
const (
	// groupWait is how long the group gathers batches after the first one
	// due, unless it is an answer's first piece (written at once). In B6's
	// load test, 200 answering turns had some 900 batches due a second; the
	// groups carried about 20 each, some 45 transactions a second instead of
	// 900. The 20 ms a batch may wait is a fifth of a turn's own batch
	// window, and comes before the event's insert (the event lag metric
	// starts at the insert).
	groupWait = 20 * time.Millisecond
	// groupMax batches (each under batchBytes and a seal) start a group at
	// once and are the most one transaction writes.
	groupMax = 256
	// notifyMax bounds a NOTIFY's payload (Postgres's limit is 8,000
	// bytes); a group with more threads notifies more than once.
	notifyMax = 7000
)

// deltaGroup is a pool's group writer. Its loop runs while batches are due
// and stops when none is.
type deltaGroup struct {
	pool *pgxpool.Pool

	mu      sync.Mutex
	due     map[*deltaWriter]time.Time // when each became due
	order   []*deltaWriter             // in that order (may hold forgotten ones)
	urgent  bool                       // an answer's first piece is due
	running bool
	kick    chan struct{}
}

var groups sync.Map // *pgxpool.Pool → *deltaGroup

// groupWaitSet replaces groupWait (tests; 0: groupWait).
var groupWaitSet atomic.Int64

func gatherWait() time.Duration {
	if d := groupWaitSet.Load(); d > 0 {
		return time.Duration(d)
	}
	return groupWait
}

// groupOf is pool's group writer: one a process for each database.
func groupOf(pool *pgxpool.Pool) *deltaGroup {
	if g, ok := groups.Load(pool); ok {
		return g.(*deltaGroup)
	}
	g, _ := groups.LoadOrStore(pool, &deltaGroup{pool: pool, due: map[*deltaWriter]time.Time{}, kick: make(chan struct{}, 1)})
	return g.(*deltaGroup)
}

// ready says w has a batch due; urgent: its answer's first piece.
func (g *deltaGroup) ready(w *deltaWriter, urgent bool) {
	g.mu.Lock()
	if _, ok := g.due[w]; !ok {
		g.due[w] = time.Now()
		g.order = append(g.order, w)
	}
	g.urgent = g.urgent || urgent
	start, kick := !g.running, urgent || len(g.due) >= groupMax
	g.running = true
	g.mu.Unlock()
	switch {
	case start:
		go g.run()
	case kick:
		select {
		case g.kick <- struct{}{}:
		default:
		}
	}
}

// forget drops w's due batch (its turn took what was left).
func (g *deltaGroup) forget(w *deltaWriter) {
	g.mu.Lock()
	delete(g.due, w)
	g.mu.Unlock()
}

// run writes groups until no batch is due: each once groupWait passed
// since its first batch was due, at once for a first piece or groupMax
// batches. Groups are written one at a time, so each writer's batches
// commit in order.
func (g *deltaGroup) run() {
	for {
		g.mu.Lock()
		var oldest time.Time
		for _, w := range g.order {
			if at, ok := g.due[w]; ok {
				oldest = at
				break
			}
		}
		if oldest.IsZero() {
			g.running, g.urgent, g.order = false, false, nil
			g.mu.Unlock()
			return
		}
		wait := time.Until(oldest.Add(gatherWait()))
		if g.urgent || len(g.due) >= groupMax {
			wait = 0
		}
		g.mu.Unlock()
		if wait > 0 {
			t := time.NewTimer(wait)
			select {
			case <-t.C:
			case <-g.kick:
			}
			t.Stop()
		}
		g.flush(g.take())
	}
}

// take removes up to groupMax due writers, oldest first.
func (g *deltaGroup) take() []*deltaWriter {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out, rest []*deltaWriter
	for _, w := range g.order {
		if _, ok := g.due[w]; !ok {
			continue
		}
		if len(out) < groupMax {
			out = append(out, w)
			delete(g.due, w)
		} else {
			rest = append(rest, w)
		}
	}
	g.order = rest
	if len(rest) == 0 {
		g.urgent = false
	}
	select {
	case <-g.kick:
	default:
	}
	return out
}

// batch is a writer's text in a group, and what became of it.
type batch struct {
	w      *deltaWriter
	text   string
	status batchStatus
	err    error
	seq    int64
}

type batchStatus int

const (
	batchWritten batchStatus = iota + 1
	// batchFenced: its turn is no longer the writer's (Stop, the sweep,
	// retention, its thread deleted); nothing more of it is written.
	batchFenced
	// batchSkipped: another transaction held its thread; it goes in a
	// later group, before what arrived since.
	batchSkipped
	// batchFailed: it couldn't be written (err) and its stream is cut, as
	// one batch's error was before groups.
	batchFailed
)

// flush writes ws's batches as one group and tells each writer the
// outcome. If the group's transaction fails, each batch is tried on its
// own, so one batch's error loses no other's.
func (g *deltaGroup) flush(ws []*deltaWriter) []*batch {
	var bs []*batch
	for _, w := range ws {
		// Held until the outcome is told: the turn's take waits for it.
		w.writing.Lock()
		w.mu.Lock()
		if w.timer != nil {
			w.timer.Stop()
			w.timer = nil
		}
		text, err := w.buf.String(), w.err
		w.buf.Reset()
		w.mu.Unlock()
		if err != nil || text == "" {
			w.writing.Unlock()
			continue
		}
		bs = append(bs, &batch{w: w, text: text})
	}
	if len(bs) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := g.write(ctx, bs); err != nil {
		var unknown commitError
		for _, b := range bs {
			// A commit that failed may have committed: trying again could
			// write a batch twice.
			if len(bs) == 1 || errors.As(err, &unknown) {
				b.status, b.err = batchFailed, err
			} else if err := g.write(ctx, []*batch{b}); err != nil {
				b.status, b.err = batchFailed, err
			}
		}
	}
	for _, b := range bs {
		w := b.w
		switch b.status {
		case batchWritten:
			w.mu.Lock()
			if w.committedAt.IsZero() {
				w.committedAt = time.Now()
			}
			w.mu.Unlock()
			w.settle()
		case batchSkipped:
			w.mu.Lock()
			later := w.buf.String()
			w.buf.Reset()
			w.buf.WriteString(b.text)
			w.buf.WriteString(later)
			w.mu.Unlock()
			g.ready(w, false)
		case batchFenced, batchFailed:
			err := b.err
			if b.status == batchFenced {
				err = errFenced
			}
			w.mu.Lock()
			if w.err == nil {
				w.err = err
			}
			w.mu.Unlock()
			w.settle()
			w.stop(err)
		}
		w.writing.Unlock()
	}
	return bs
}

// write commits a group: its threads locked in order (one another
// transaction holds is skipped), each batch's turn fence read under that
// lock, the threads' next numbers, the sealed events and the NOTIFY.
func (g *deltaGroup) write(ctx context.Context, bs []*batch) error {
	for _, b := range bs {
		b.status, b.seq = 0, 0
	}
	threads, turns := make([]uuid.UUID, 0, len(bs)), make([]uuid.UUID, 0, len(bs))
	for _, b := range bs {
		threads, turns = append(threads, b.w.thread), append(turns, b.w.turn)
	}
	slices.SortFunc(threads, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
	threads, turns = slices.Compact(threads), slices.Compact(turns)
	// A group's only lasting change is its threads' event numbers (the
	// events table is unlogged): its commit needn't wait for the WAL to
	// reach the disk. A crash may lose the last groups' numbers with the
	// events themselves; streams then resume from a snapshot.
	tx, err := g.pool.BeginTx(ctx, pgx.TxOptions{BeginQuery: "BEGIN; SET LOCAL synchronous_commit TO OFF"})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	q := store.New(tx)
	locked, err := q.AIAssistantDeltaThreadsLock(ctx, threads)
	if err != nil {
		return err
	}
	held := map[uuid.UUID]uuid.UUID{} // thread → its workspace
	for _, r := range locked {
		held[r.ID] = r.WorkspaceID
	}
	fences, err := q.AIAssistantTurnFences(ctx, turns)
	if err != nil {
		return err
	}
	fence := map[uuid.UUID]store.AIAssistantTurnFencesRow{}
	for _, f := range fences {
		fence[f.ID] = f
	}
	n := map[uuid.UUID]int64{}
	for _, b := range bs {
		f, ok := fence[b.w.turn]
		ws, mine := held[b.w.thread]
		switch {
		// A fence that fails without the thread's lock fails for good: a
		// turn's version only grows, and its writer is gone once it stops
		// answering.
		case !ok || f.Version != b.w.version || f.State != "answering" || (mine && ws != b.w.ws):
			b.status = batchFenced
		case !mine:
			b.status = batchSkipped
		default:
			b.status = batchWritten
			n[b.w.thread]++
		}
	}
	if len(n) > 0 {
		ids := make([]uuid.UUID, 0, len(n))
		for id := range n {
			ids = append(ids, id)
		}
		slices.SortFunc(ids, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
		ns := make([]int64, len(ids))
		for i, id := range ids {
			ns[i] = n[id]
		}
		last, err := q.AIAssistantDeltaSeqs(ctx, store.AIAssistantDeltaSeqsParams{Ids: ids, Ns: ns})
		if err != nil {
			return err
		}
		next := map[uuid.UUID]int64{}
		for _, r := range last {
			next[r.ID] = r.EventSeq - n[r.ID] + 1
		}
		var in store.AIAssistantDeltasInsertParams
		for _, b := range bs {
			if b.status != batchWritten {
				continue
			}
			w := b.w
			b.seq = next[w.thread]
			next[w.thread]++
			raw, err := json.Marshal(AIThreadDeltaEvent{TurnID: w.turn, Text: b.text})
			if err != nil {
				return err
			}
			sealed, err := w.s.Box.Seal(raw, eventAD(w.thread, b.seq))
			if err != nil {
				return err
			}
			in.ThreadIds, in.Seqs, in.TurnIds = append(in.ThreadIds, w.thread), append(in.Seqs, b.seq), append(in.TurnIds, w.turn)
			in.Payloads, in.CreatedAts = append(in.Payloads, sealed), append(in.CreatedAts, w.s.now())
		}
		if err := q.AIAssistantDeltasInsert(ctx, in); err != nil {
			return err
		}
		for _, payload := range notifications(last) {
			if err := q.AIAssistantNotify(ctx, payload); err != nil {
				return err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return commitError{err}
	}
	return nil
}

// commitError: a group's commit failed, so whether it committed is
// unknown.
type commitError struct{ error }

func (e commitError) Unwrap() error { return e.error }

// notifications are a group's NOTIFY payloads: each thread's last event
// as "thread:seq", comma-separated, notifyMax bytes at most each.
func notifications(last []store.AIAssistantDeltaSeqsRow) []string {
	slices.SortFunc(last, func(a, b store.AIAssistantDeltaSeqsRow) int { return cmp.Compare(a.ID.String(), b.ID.String()) })
	var out []string
	var p []byte
	for _, r := range last {
		entry := r.ID.String() + ":" + strconv.FormatInt(r.EventSeq, 10)
		if len(p) > 0 && len(p)+1+len(entry) > notifyMax {
			out, p = append(out, string(p)), p[:0]
		}
		if len(p) > 0 {
			p = append(p, ',')
		}
		p = append(p, entry...)
	}
	if len(p) > 0 {
		out = append(out, string(p))
	}
	return out
}
