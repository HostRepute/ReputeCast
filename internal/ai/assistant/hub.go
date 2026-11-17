package assistant

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Hub wakes this process's thread streams when their thread has new
// events. It listens on one connection of its own, opened when the first
// stream starts and closed a while after the last one ends; NOTIFY only
// says which thread changed, and each stream reads its events from the
// table after its own cursor. While the connection is down (Postgres
// restarted it, a proxy cut it) the hub wakes every stream every
// PollEvery until it listens again, so nothing waits for a notification
// that was lost.
type Hub struct {
	Pool *pgxpool.Pool
	Log  *slog.Logger
	// PollEvery is how often streams are woken while the hub can't listen
	// (1 s); IdleClose how long it keeps listening without streams (1 min).
	PollEvery, IdleClose time.Duration

	mu        sync.Mutex
	subs      map[uuid.UUID]map[chan struct{}]struct{}
	running   bool
	listening bool
	idleSince time.Time
	closed    bool
	cancel    context.CancelFunc
}

// listenChannel is the NOTIFY channel of every thread's events.
const listenChannel = "ai_assistant"

// listenerName names the hub's connection in pg_stat_activity.
const listenerName = "reputecast-assistant"

func (h *Hub) pollEvery() time.Duration {
	if h.PollEvery > 0 {
		return h.PollEvery
	}
	return time.Second
}

func (h *Hub) idleClose() time.Duration {
	if h.IdleClose > 0 {
		return h.IdleClose
	}
	return time.Minute
}

// Subscribe wakes the returned channel (without blocking: a stream that is
// busy writing reads everything at once later) when thread may have new
// events. cancel ends it.
func (h *Hub) Subscribe(thread uuid.UUID) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	if h.subs == nil {
		h.subs = map[uuid.UUID]map[chan struct{}]struct{}{}
	}
	if h.subs[thread] == nil {
		h.subs[thread] = map[chan struct{}]struct{}{}
	}
	h.subs[thread][ch] = struct{}{}
	if !h.running && !h.closed {
		ctx, cancel := context.WithCancel(context.Background())
		h.running, h.cancel = true, cancel
		go h.run(ctx)
	}
	h.mu.Unlock()
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subs[thread], ch)
			if len(h.subs[thread]) == 0 {
				delete(h.subs, thread)
			}
			if len(h.subs) == 0 {
				h.idleSince = time.Now()
			}
			h.mu.Unlock()
		})
	}
}

// Listening reports whether the hub's connection listens now.
func (h *Hub) Listening() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.listening
}

// Close stops listening for good (the process stops).
func (h *Hub) Close() {
	h.mu.Lock()
	h.closed = true
	if h.cancel != nil {
		h.cancel()
	}
	h.mu.Unlock()
}

// wake wakes thread's streams, or every stream (uuid.Nil).
func (h *Hub) wake(thread uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, chans := range h.subs {
		if thread != uuid.Nil && id != thread {
			continue
		}
		for ch := range chans {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}
}

// idle reports, and then marks the hub stopped, when it has had no stream
// for IdleClose.
func (h *Hub) idle() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.subs) > 0 || h.idleSince.IsZero() || time.Since(h.idleSince) < h.idleClose() {
		return false
	}
	h.running, h.listening = false, false
	return true
}

func (h *Hub) setListening(on bool) {
	h.mu.Lock()
	h.listening = on
	h.mu.Unlock()
}

func (h *Hub) log() *slog.Logger {
	if h.Log != nil {
		return h.Log
	}
	return slog.Default()
}

// run listens until the hub is idle or closed, connecting again whenever
// the connection is lost.
func (h *Hub) run(ctx context.Context) {
	defer func() {
		h.mu.Lock()
		h.running, h.listening = false, false
		h.mu.Unlock()
	}()
	for ctx.Err() == nil {
		conn, err := h.connect(ctx)
		if err == nil {
			h.setListening(true)
			// Anything committed while it wasn't listening is read now.
			h.wake(uuid.Nil)
			err = h.listen(ctx, conn)
			_ = conn.Close(context.Background())
			h.setListening(false)
			if err == nil { // idle
				return
			}
		}
		if ctx.Err() != nil {
			return
		}
		h.log().Warn("assistant: listening for thread events failed; streams poll until it listens again", "err", err)
		// Poll until the next attempt.
		deadline := time.Now().Add(max(h.pollEvery(), time.Second))
		for time.Now().Before(deadline) {
			h.wake(uuid.Nil)
			select {
			case <-ctx.Done():
				return
			case <-time.After(h.pollEvery()):
			}
			if h.idle() {
				return
			}
		}
	}
}

func (h *Hub) connect(ctx context.Context) (*pgx.Conn, error) {
	if h.Pool == nil {
		return nil, errors.New("no database")
	}
	cfg := h.Pool.Config().ConnConfig.Copy()
	cfg.RuntimeParams["application_name"] = listenerName
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := pgx.ConnectConfig(cctx, cfg)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Exec(cctx, "LISTEN "+listenChannel); err != nil {
		_ = conn.Close(context.Background())
		return nil, err
	}
	return conn, nil
}

// listen wakes the streams of each notified thread. It returns nil once
// the hub is idle, or the error that cut the connection.
func (h *Hub) listen(ctx context.Context, conn *pgx.Conn) error {
	for {
		wctx, cancel := context.WithTimeout(ctx, time.Second)
		n, err := conn.WaitForNotification(wctx)
		cancel()
		switch {
		case err == nil:
			thread, _, _ := strings.Cut(n.Payload, ":")
			if id, err := uuid.Parse(thread); err == nil {
				h.wake(id)
			}
			continue
		case ctx.Err() != nil:
			return ctx.Err()
		case conn.IsClosed() || !errors.Is(err, context.DeadlineExceeded):
			if err == nil {
				err = errors.New("connection closed")
			}
			return err
		}
		// A second without notifications.
		if h.idle() {
			return nil
		}
	}
}
