package assistant

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Stream timing defaults.
const (
	defaultHeartbeat  = 15 * time.Second
	defaultSlowClient = 5 * time.Second
	// streamAccessEvery is how often a stream checks that its reader may
	// still read the thread.
	streamAccessEvery = 5 * time.Second
	// streamPage bounds the events read at once.
	streamPage = 200
)

func (s *Service) heartbeat() time.Duration {
	if s.Heartbeat > 0 {
		return s.Heartbeat
	}
	return defaultHeartbeat
}

func (s *Service) slowClient() time.Duration {
	if s.SlowClient > 0 {
		return s.SlowClient
	}
	return defaultSlowClient
}

// writerKey holds the stream's http.ResponseWriter in its context: a slow
// client's write is cut through it.
type writerKey struct{}

// keepWriter is the stream operation's middleware that keeps its writer.
func keepWriter(ctx huma.Context, next func(huma.Context)) {
	if w, ok := ctx.BodyWriter().(http.ResponseWriter); ok {
		ctx = huma.WithValue(ctx, writerKey{}, w)
	}
	next(ctx)
}

// cutter sends an event, cutting the connection when the client doesn't
// take it in time: its writes would otherwise wait on the network while the
// thread goes on. The client resumes with the last event it read.
type cutter struct {
	send  sse.Sender
	rc    *http.ResponseController
	limit time.Duration
}

func (c cutter) do(m sse.Message) error {
	if c.rc != nil {
		t := time.AfterFunc(c.limit, func() { _ = c.rc.SetWriteDeadline(time.Now()) })
		defer t.Stop()
	}
	return c.send(m)
}

// streamProblem is the event that ends a stream the caller may not read.
func streamProblem(err error) AIThreadStreamProblem {
	var p *api.Problem
	if errors.As(err, &p) {
		return AIThreadStreamProblem{Status: p.Status, Code: p.Code, Detail: p.Detail}
	}
	return AIThreadStreamProblem{Status: http.StatusInternalServerError, Code: "internal", Detail: "The stream failed. Reconnect to resume."}
}

// Stream sends a thread's events to its owner after the cursor (the last
// event id they read): a snapshot first when there is no cursor, or when
// the events after it are gone (24 hours after their turn, or lost in a
// crash), then each event as it commits, and a heartbeat every 15 s.
func (s *Service) Stream(ctx context.Context, ws *store.Workspace, p *authz.Principal, thread uuid.UUID, cursor string, send sse.Sender) {
	c := cutter{send: send, limit: s.slowClient()}
	if w, ok := ctx.Value(writerKey{}).(http.ResponseWriter); ok {
		c.rc = http.NewResponseController(w)
	}
	if _, err := s.Thread(ctx, ws, p, thread); err != nil {
		_ = c.do(sse.Message{Data: streamProblem(err)})
		return
	}
	var wake <-chan struct{}
	if s.Hub != nil {
		ch, cancel := s.Hub.Subscribe(thread)
		defer cancel()
		wake = ch
	}
	after, err := strconv.ParseInt(cursor, 10, 64)
	snapshot := err != nil || after < 0
	beat := time.NewTicker(s.heartbeat())
	defer beat.Stop()
	check := time.NewTicker(streamAccessEvery)
	defer check.Stop()
	_ = c.do(sse.Message{Retry: 2000, Comment: "connected"})
	for {
		var err error
		if after, snapshot, err = s.sendEvents(ctx, ws, p, thread, after, snapshot, c); err != nil {
			var pr *api.Problem
			if errors.As(err, &pr) {
				_ = c.do(sse.Message{Data: streamProblem(err)})
			}
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-wake:
		case <-beat.C:
			if err := c.do(sse.Message{Data: AIThreadHeartbeat{At: time.Now().UTC()}}); err != nil {
				return
			}
		case <-check.C:
			if _, _, err := s.actor(ctx, s.Q, ws, p, true); err != nil {
				_ = c.do(sse.Message{Data: streamProblem(err)})
				return
			}
		}
	}
}

// sendEvents sends what the thread has after the cursor and returns the
// new cursor. A send that fails (the client left, or was cut) ends the
// stream.
func (s *Service) sendEvents(ctx context.Context, ws *store.Workspace, p *authz.Principal, thread uuid.UUID, after int64, snapshot bool, c cutter) (int64, bool, error) {
	for {
		if snapshot {
			th, err := s.snapshot(ctx, ws, p, thread)
			if err != nil {
				return after, true, err
			}
			if err := c.do(sse.Message{IDString: strconv.FormatInt(th.EventSeq, 10), Data: AIThreadSnapshotEvent{Thread: th}}); err != nil {
				return after, false, err
			}
			after, snapshot = th.EventSeq, false
		}
		t, err := s.Q.AIAssistantThread(ctx, store.AIAssistantThreadParams{WorkspaceID: ws.ID, UserID: p.UserID, ID: thread})
		if errors.Is(err, pgx.ErrNoRows) {
			return after, false, threadNotFound()
		}
		if err != nil {
			return after, false, err
		}
		if after == t.EventSeq {
			return after, false, nil
		}
		rows, err := s.Q.AIAssistantEvents(ctx, store.AIAssistantEventsParams{ThreadID: thread, After: after, Lim: streamPage})
		if err != nil {
			return after, false, err
		}
		// Events this cursor needs are gone, or it is from elsewhere.
		if after > t.EventSeq || len(rows) == 0 || rows[0].Seq != after+1 {
			snapshot = true
			continue
		}
		for _, r := range rows {
			data, err := s.eventData(thread, r)
			if err != nil {
				return after, false, err
			}
			if err := c.do(sse.Message{IDString: strconv.FormatInt(r.Seq, 10), Data: data}); err != nil {
				return after, false, err
			}
			after = r.Seq
		}
	}
}

// eventData is an event as its SSE type.
func (s *Service) eventData(thread uuid.UUID, r store.AIAssistantEventsRow) (any, error) {
	var v any
	switch r.Kind {
	case kindTurn:
		v = &AIThreadTurnEvent{}
	case kindDelta:
		v = &AIThreadDeltaEvent{}
	case kindMessage:
		v = &AIThreadMessageEvent{}
	case kindUsage:
		v = &AIThreadUsageEvent{}
	case kindTool:
		v = &AIThreadToolEvent{}
	case kindConfirmation:
		v = &AIThreadConfirmationEvent{}
	default:
		return nil, errors.New("assistant: unknown event kind " + r.Kind)
	}
	if err := s.openEvent(thread, r.Seq, r.PayloadSealed, v); err != nil {
		return nil, err
	}
	return v, nil
}

// snapshot is the thread as of its last event, read in one snapshot of the
// database: the events after it are exactly the ones it doesn't include.
func (s *Service) snapshot(ctx context.Context, ws *store.Workspace, p *authz.Principal, thread uuid.UUID) (AIThread, error) {
	// Not read-only: reading the person's access locks their membership
	// for share.
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return AIThread{}, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	q := store.New(tx)
	fresh, _, err := s.actor(ctx, q, ws, p, true)
	if err != nil {
		return AIThread{}, err
	}
	t, err := q.AIAssistantThread(ctx, store.AIAssistantThreadParams{WorkspaceID: ws.ID, UserID: fresh.UserID, ID: thread})
	if errors.Is(err, pgx.ErrNoRows) {
		return AIThread{}, threadNotFound()
	}
	if err != nil {
		return AIThread{}, err
	}
	return s.view(ctx, q, ws, fresh, t)
}
