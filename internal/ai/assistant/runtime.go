package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
)

// A turn's life. SendMessage commits the person's message and the queued
// turn with its job. The job claims the turn: under the thread's lock it
// checks the owner's access and the model, takes a slot (the thread's
// only one, one of the person's and one of the workspace's) with a lease,
// reserves the request's budget and saves the step: all committed before
// the request leaves. The answer streams in batches of events, each
// written only while the turn is still the worker's (its version and
// state: the fence). The checkpoint accounts the request whatever
// happened, then, only if the turn is still the worker's, saves the
// answer and ends the turn. Stop and the sweep move the version on, so a
// worker that comes back late writes nothing. A worker that dies leaves
// its slot to lapse; the sweep then fails the turn as interrupted, keeps
// what it wrote and leaves its budget hold uncertain.

// TurnArgs is a turn's job: ids only, never content.
type TurnArgs struct {
	WorkspaceID uuid.UUID `json:"workspaceId"`
	ThreadID    uuid.UUID `json:"threadId"`
	TurnID      uuid.UUID `json:"turnId"`
	Version     int64     `json:"version"`
}

// Kind implements river.JobArgs.
func (TurnArgs) Kind() string { return "ai.assistant_turn" }

// InsertOpts puts turns on their own queue: a long answer never holds up
// other jobs.
func (TurnArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueAssistant, MaxAttempts: 12}
}

// TurnWorker answers turns.
type TurnWorker struct {
	river.WorkerDefaults[TurnArgs]
	S *Service
}

const (
	// turnLease is how long a slot is a worker's without renewal; it is
	// renewed every renewEvery while the answer streams.
	turnLease  = time.Minute
	renewEvery = 15 * time.Second
	// turnSnooze is how soon a turn without a free slot tries again.
	turnSnooze = 2 * time.Second
	// queueTimeout fails a turn whose job never took it.
	queueTimeout = 30 * time.Minute
	// turnTimeout bounds the provider's answer (the assistant's timeout in
	// internal/ai).
	turnTimeout = 3 * time.Minute
)

// Timeout outlives the provider's own timeout and the checkpoint.
func (w *TurnWorker) Timeout(*river.Job[TurnArgs]) time.Duration { return turnTimeout + time.Minute }

// Work implements river.Worker.
func (w *TurnWorker) Work(ctx context.Context, job *river.Job[TurnArgs]) error {
	return w.S.ExecuteTurn(ctx, job.Args)
}

func (s *Service) enqueue(ctx context.Context, tx pgx.Tx, ws uuid.UUID, t store.AiAssistantTurn) error {
	if s.Jobs == nil || s.Jobs() == nil {
		return errors.New("assistant: job queue unavailable")
	}
	_, err := s.Jobs().InsertTx(ctx, tx, TurnArgs{WorkspaceID: ws, ThreadID: t.ThreadID, TurnID: t.ID, Version: t.Version}, nil)
	return err
}

func (s *Service) watchEvery() time.Duration {
	if s.WatchEvery > 0 {
		return s.WatchEvery
	}
	return time.Second
}

// turnRun is a claimed turn, ready to send.
type turnRun struct {
	thread  store.AiAssistantThread
	turn    store.AiAssistantTurn
	owner   *authz.Principal
	model   *ai.AssistantModel
	req     llm.Request
	step    uuid.UUID
	lease   uuid.UUID
	started time.Time
	// kind is the round's ("answer" or "summary"), upto the last message a
	// summary covers; rounds counts the turn's requests.
	kind       string
	upto       int32
	rounds     int
	summarized bool
	// resume: the turn goes on after its owner decided on a call.
	resume *resume
}

// Why a worker cut its own stream.
var (
	errLeaseLost   = errors.New("assistant: the turn's lease lapsed")
	errShutdown    = errors.New("assistant: the worker stopped")
	errInterrupted = errors.New("assistant: the worker answering the turn was lost")
)

// accessError: the owner lost access while the turn answered.
type accessError struct{ err error }

func (e *accessError) Error() string { return "assistant: access lost: " + e.err.Error() }
func (e *accessError) Unwrap() error { return e.err }

// ExecuteTurn answers a turn with at most one model request. Running it
// twice, at once or after a lost acknowledgment, sends nothing twice.
func (s *Service) ExecuteTurn(ctx context.Context, args TurnArgs) error {
	ws, err := s.Q.GetWorkspace(ctx, args.WorkspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var run *turnRun
	busy := false
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		var err error
		run, busy, err = s.claim(ctx, store.New(tx), &ws, args)
		return err
	})
	if err != nil {
		return err
	}
	if busy {
		return river.JobSnooze(turnSnooze)
	}
	if run == nil {
		return nil
	}
	return s.answer(ctx, &ws, args, run)
}

// claim takes a queued turn for this worker, or reports that it must wait
// (busy) or that there is nothing to do (nil).
func (s *Service) claim(ctx context.Context, q *store.Queries, ws *store.Workspace, args TurnArgs) (*turnRun, bool, error) {
	// The configuration first, as every request does: a model removed
	// meanwhile clears threads' models under it.
	if err := s.AI.LockAssistantConfig(ctx, q, ws.ID); err != nil {
		return nil, false, err
	}
	t, err := q.AIAssistantThreadLockByID(ctx, store.AIAssistantThreadLockByIDParams{WorkspaceID: ws.ID, ID: args.ThreadID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	turn, err := q.AIAssistantTurnLock(ctx, store.AIAssistantTurnLockParams{ThreadID: t.ID, ID: args.TurnID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if turn.Version != args.Version || t.State != "open" {
		return nil, false, nil
	}
	now := s.now()
	switch turn.State {
	case "queued":
	case "answering":
		// Another attempt of this job holds the turn, or held it and died.
		if _, err := q.AIAssistantSlotOfTurn(ctx, store.AIAssistantSlotOfTurnParams{TurnID: turn.ID, Now: now}); err == nil {
			return nil, true, nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return nil, false, err
		}
		return nil, false, s.interrupt(ctx, q, t, &turn)
	default:
		return nil, false, nil
	}
	owner := &authz.Principal{Type: authz.ActorUser, ID: t.UserID, UserID: t.UserID, WorkspaceID: ws.ID}
	fresh, settings, err := s.actor(ctx, q, ws, owner, true)
	if err != nil {
		return nil, false, s.failFor(ctx, q, &turn, err)
	}
	if turn.ModelID == nil {
		return nil, false, s.endTurn(ctx, q, &turn, "failed", "model_gone")
	}
	m, err := s.AI.ResolveAssistantModel(ctx, q, ws.ID, t.UserID, turn.ModelID)
	if err != nil {
		return nil, false, s.failFor(ctx, q, &turn, err)
	}
	if err := s.AI.ReadyAssistant(ctx, ws, m); err != nil {
		return nil, false, s.failFor(ctx, q, &turn, err)
	}
	if err := q.AIAssistantSlotsExpire(ctx, store.AIAssistantSlotsExpireParams{WorkspaceID: ws.ID, Now: now}); err != nil {
		return nil, false, err
	}
	lease := uuid.Must(uuid.NewV7())
	if _, err := q.AIAssistantSlotClaim(ctx, store.AIAssistantSlotClaimParams{ThreadID: t.ID, TurnID: turn.ID, WorkspaceID: ws.ID, UserID: t.UserID, LeaseID: lease,
		ExpiresAt: now.Add(turnLease), PerWorkspace: settings.PerWorkspace, PerPerson: settings.PerPerson}); errors.Is(err, pgx.ErrNoRows) {
		return nil, true, nil
	} else if err != nil {
		return nil, false, err
	}
	run := &turnRun{thread: t, turn: turn, owner: fresh, model: m, lease: lease}
	conf, err := q.AIAssistantConfirmationOpen(ctx, turn.ID)
	switch {
	case err == nil && conf.State != "pending":
		// The owner decided on a call: the round goes on from it, with no
		// new request until its calls ran.
		rd, err := s.openRound(conf)
		if err != nil {
			return nil, false, err
		}
		step, err := q.AIAssistantStepLast(ctx, turn.ID)
		if err != nil {
			return nil, false, err
		}
		run.resume, run.step, run.rounds = &resume{conf: conf, round: rd}, conf.StepID, int(step.Sequence)
	case err != nil && !errors.Is(err, pgx.ErrNoRows):
		return nil, false, err
	default:
		err = s.prepareRound(ctx, q, ws, run)
	}
	if err != nil {
		if err := q.AIAssistantSlotRelease(ctx, store.AIAssistantSlotReleaseParams{TurnID: turn.ID, LeaseID: lease}); err != nil {
			return nil, false, err
		}
		return nil, false, s.failFor(ctx, q, &turn, err)
	}
	turn.State, turn.StartedAt = "answering", &now
	if err := q.AIAssistantTurnSave(ctx, store.AIAssistantTurnSaveParams{ID: turn.ID, State: turn.State, Reason: "", Version: turn.Version, StartedAt: turn.StartedAt}); err != nil {
		return nil, false, err
	}
	if err := s.turnEvent(ctx, q, turn); err != nil {
		return nil, false, err
	}
	run.turn = turn
	return run, false, nil
}

// prepareRound assembles the turn's next request, reserves its budget in
// the thread's ledger and saves its step: committed before it is sent.
// The thread and the turn are locked.
func (s *Service) prepareRound(ctx context.Context, q *store.Queries, ws *store.Workspace, run *turnRun) error {
	req, kind, upto, err := s.request(ctx, q, ws, run)
	if err != nil {
		return err
	}
	step := uuid.Must(uuid.NewV7())
	t := run.thread
	if err := s.AI.ReserveAssistant(ctx, q, ws, run.owner, run.model, &req, ai.AssistantLedger{ThreadID: t.ID, TokenLimit: t.TokenLimit, CostLimitMicros: t.CostLimitMicros}, step); err != nil {
		return err
	}
	now := s.now()
	if err := q.AIAssistantStepInsert(ctx, store.AIAssistantStepInsertParams{ID: step, TurnID: run.turn.ID, TurnVersion: run.turn.Version, Kind: kind, Now: now}); err != nil {
		return err
	}
	run.req, run.kind, run.upto, run.step, run.started = req, kind, upto, step, now
	run.rounds++
	run.summarized = run.summarized || kind == "summary"
	return nil
}

// lockActive locks a running turn's thread and turn, and reports whether
// the turn is still this worker's: its version, answering, its thread
// open and its lease held.
func (s *Service) lockActive(ctx context.Context, q *store.Queries, ws *store.Workspace, args TurnArgs, run *turnRun) (store.AiAssistantThread, store.AiAssistantTurn, bool, error) {
	var turn store.AiAssistantTurn
	t, err := q.AIAssistantThreadLockByID(ctx, store.AIAssistantThreadLockByIDParams{WorkspaceID: ws.ID, ID: run.thread.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return t, turn, false, nil
	}
	if err != nil {
		return t, turn, false, err
	}
	turn, err = q.AIAssistantTurnLock(ctx, store.AIAssistantTurnLockParams{ThreadID: t.ID, ID: run.turn.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return t, turn, false, nil
	}
	if err != nil {
		return t, turn, false, err
	}
	if turn.Version != args.Version || turn.State != "answering" || t.State != "open" {
		return t, turn, false, nil
	}
	slot, err := q.AIAssistantSlotOfTurn(ctx, store.AIAssistantSlotOfTurnParams{TurnID: turn.ID, Now: s.now()})
	if errors.Is(err, pgx.ErrNoRows) {
		return t, turn, false, nil
	}
	return t, turn, err == nil && slot.LeaseID == run.lease, err
}

// runTools runs a response's tool calls in order, each as the owner is
// now, while the turn stays this worker's and the thread's tool-call
// ceiling allows; then saves their results for the next request. A call
// past the ceiling doesn't run: the model reads tool_limit, and its next
// request offers no tools. A call of a tool that writes outside
// ReputeCast stops the turn until its owner decides (it isn't this
// worker's any more: false); the turn's next job goes on from that call
// (conf), which runs at most once: it is marked calling, committed,
// before it is sent, and a call found calling is never sent again.
func (s *Service) runTools(ctx context.Context, ws *store.Workspace, args TurnArgs, run *turnRun, rd round, conf *store.AiAssistantConfirmation) (bool, error) {
	for i := len(rd.Results); i < len(rd.Calls); i++ {
		call := rd.Calls[i]
		decided := conf != nil && int(conf.CallIndex) == i
		// What happens to the call: run, paused, or a code the model reads.
		do, code := "", "tool_limit"
		c := &toolCall{ws: ws, owner: run.owner, callID: call.ID, args: call.Arguments}
		active := false
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			t, turn, ok, err := s.lockActive(ctx, q, ws, args, run)
			if err != nil || !ok {
				return err
			}
			active = true
			fresh, _, err := s.actor(ctx, q, ws, run.owner, true)
			if err != nil {
				return s.failRunning(ctx, q, run, err)
			}
			c.owner, c.thread = fresh, t
			if c.contexts, err = q.AIAssistantContexts(ctx, t.ID); err != nil {
				return err
			}
			if decided {
				cur, err := q.AIAssistantConfirmationLock(ctx, store.AIAssistantConfirmationLockParams{ThreadID: t.ID, ID: conf.ID})
				if err != nil {
					return err
				}
				*conf = cur
				switch {
				case cur.State == "denied":
					code = "denied"
					return nil
				case cur.State == "calling":
					// Sent before, by a worker that was lost: never again.
					code = "outcome_unknown"
					return nil
				case cur.State != "allowed":
					code = "not_available"
					return nil
				case t.ToolCalls >= t.ToolLimit:
					return nil
				}
				if err := q.AIAssistantConfirmationSet(ctx, store.AIAssistantConfirmationSetParams{ID: cur.ID, State: "calling"}); err != nil {
					return err
				}
				do = "run"
				return q.AIAssistantThreadToolCall(ctx, t.ID)
			}
			if t.ToolCalls >= t.ToolLimit {
				return nil
			}
			offered, err := s.offered(ctx, q, ws, fresh, t.ID, c.contexts)
			if err != nil {
				return err
			}
			if j := slices.IndexFunc(offered, func(x ToolSpec) bool { return x.Name == call.Name }); j >= 0 && offered[j].Confirm {
				do = "pause"
				return s.pause(ctx, q, run, turn, rd, i, offered[j])
			}
			do = "run"
			return q.AIAssistantThreadToolCall(ctx, t.ID)
		})
		if err != nil || !active || c.thread.ID == uuid.Nil || do == "pause" {
			return false, err
		}
		start := time.Now()
		r, content := failed(code), json.RawMessage(nil)
		if do == "run" {
			var confirmed *store.AiAssistantConfirmation
			if decided {
				confirmed = conf
			}
			r, content = s.runTool(ctx, s.Q, c, call.Name, confirmed)
		} else {
			if decided {
				// A denied or half-sent call is still named by its server and tool.
				r.label = conf.ServerName + " · " + conf.ToolName
			}
			content = envelope(call.Name, r)
		}
		row := AIThreadToolRow{CallID: call.ID, Name: call.Name, OK: r.code == "", Code: r.code, Ms: time.Since(start).Milliseconds(), Label: clip(r.label, 120), Card: r.card}
		rd.Results = append(rd.Results, llm.Part{ToolResult: &llm.ToolResult{ID: call.ID, Name: call.Name, Content: content, IsError: r.code != ""}})
		rd.Rows = append(rd.Rows, row)
		active = false
		err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if decided {
				// Done, whatever happened next: the call is never sent again.
				if err := q.AIAssistantConfirmationSet(ctx, store.AIAssistantConfirmationSetParams{ID: conf.ID, State: "done"}); err != nil {
					return err
				}
			}
			t, turn, ok, err := s.lockActive(ctx, q, ws, args, run)
			if err != nil || !ok {
				return err
			}
			active = true
			return s.appendEvent(ctx, q, t.ID, &turn.ID, kindTool, AIThreadToolEvent{TurnID: turn.ID, Tool: row})
		})
		if err != nil || !active {
			return false, err
		}
	}
	active := false
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		t, turn, ok, err := s.lockActive(ctx, q, ws, args, run)
		if err != nil || !ok {
			return err
		}
		active = true
		msg, err := s.addMessage(ctx, q, t.ID, &turn.ID, "tool", turn.ModelID, messageContent{Results: rd.Results, Tools: rd.Rows})
		if err != nil {
			return err
		}
		if err := s.appendEvent(ctx, q, t.ID, &turn.ID, kindMessage, AIThreadMessageEvent{Message: msg}); err != nil {
			return err
		}
		return q.AIAssistantStepSave(ctx, store.AIAssistantStepSaveParams{ID: run.step, State: "done", Now: s.now()})
	})
	return active, err
}

// failRunning fails a running turn for a domain problem: its slot goes.
func (s *Service) failRunning(ctx context.Context, q *store.Queries, run *turnRun, err error) error {
	var p *api.Problem
	if !errors.As(err, &p) || p.Status >= 500 {
		return err
	}
	if err := q.AIAssistantSlotRelease(ctx, store.AIAssistantSlotReleaseParams{TurnID: run.turn.ID, LeaseID: run.lease}); err != nil {
		return err
	}
	turn, err := q.AIAssistantTurnLock(ctx, store.AIAssistantTurnLockParams{ThreadID: run.thread.ID, ID: run.turn.ID})
	if err != nil {
		return err
	}
	return s.endTurn(ctx, q, &turn, "failed", reasonOf(p))
}

// nextRound prepares the turn's next request after its tools ran: the
// owner's access and the turn's model as they are now.
func (s *Service) nextRound(ctx context.Context, ws *store.Workspace, args TurnArgs, run *turnRun) (bool, error) {
	ok := false
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := s.AI.LockAssistantConfig(ctx, q, ws.ID); err != nil {
			return err
		}
		t, turn, active, err := s.lockActive(ctx, q, ws, args, run)
		if err != nil || !active {
			return err
		}
		run.thread, run.turn = t, turn
		fresh, _, err := s.actor(ctx, q, ws, run.owner, true)
		if err != nil {
			return s.failRunning(ctx, q, run, err)
		}
		if turn.ModelID == nil {
			return s.failRunning(ctx, q, run, api.NotFound("AI model"))
		}
		m, err := s.AI.ResolveAssistantModel(ctx, q, ws.ID, t.UserID, turn.ModelID)
		if err != nil {
			return s.failRunning(ctx, q, run, err)
		}
		run.owner, run.model = fresh, m
		if err := s.prepareRound(ctx, q, ws, run); err != nil {
			return s.failRunning(ctx, q, run, err)
		}
		ok = true
		return nil
	})
	return ok, err
}

// failFor fails a turn for a domain problem (its code is the reason); an
// operational error retries the job instead.
func (s *Service) failFor(ctx context.Context, q *store.Queries, t *store.AiAssistantTurn, err error) error {
	var p *api.Problem
	if !errors.As(err, &p) || p.Status >= 500 {
		return err
	}
	return s.endTurn(ctx, q, t, "failed", reasonOf(p))
}

// reasonOf is the turn's reason for a problem: its code without "ai_", a
// resource the owner can't see as model_gone, a refusal as access_lost.
func reasonOf(p *api.Problem) string {
	code, ok := strings.CutPrefix(p.Code, "ai_")
	switch {
	case ok && code != "":
	case p.Status == 404:
		return "model_gone"
	default:
		return "access_lost"
	}
	b := []byte(code)
	for i, c := range b {
		if (c < 'a' || c > 'z') && c != '_' {
			b[i] = '_'
		}
	}
	return string(b[:min(len(b), 40)])
}

// providerReason is a failed request's reason.
func providerReason(err error) string {
	e, ok := llm.AsError(err)
	if !ok {
		return "provider_error"
	}
	switch e.Kind {
	case llm.ErrAuth:
		return "key_rejected"
	case llm.ErrRateLimit:
		return "rate_limited"
	case llm.ErrTimeout:
		return "timeout"
	case llm.ErrTooLarge:
		return "too_large"
	}
	return "provider_error"
}

// answer sends the claimed turn's requests, one round at a time: each
// streams its answer and is saved; one that calls tools runs them, then
// the next request goes with their results.
func (s *Service) answer(ctx context.Context, ws *store.Workspace, args TurnArgs, run *turnRun) error {
	if r := run.resume; r != nil {
		run.resume = nil
		if ok, err := s.runTools(ctx, ws, args, run, r.round, &r.conf); err != nil || !ok {
			return err
		}
		if ok, err := s.nextRound(ctx, ws, args, run); err != nil || !ok {
			return err
		}
	}
	for {
		r := s.send(ctx, ws, run)
		calls, more, err := s.checkpoint(ctx, ws, args, run, r)
		if err != nil {
			return err
		}
		s.AI.AfterAssistant(ctx, ws, run.model, r.callErr)
		if !more {
			return nil
		}
		if len(calls) > 0 {
			if ok, err := s.runTools(ctx, ws, args, run, round{Calls: calls, Results: []llm.Part{}, Rows: []AIThreadToolRow{}}, nil); err != nil || !ok {
				return err
			}
		}
		if ok, err := s.nextRound(ctx, ws, args, run); err != nil || !ok {
			return err
		}
	}
}

// sent is a round's request as it went: the response, the call's error,
// why the worker cut it (cause), everything that arrived (text) and what
// wasn't written yet (rest).
type sent struct {
	res            llm.Response
	callErr, cause error
	text, rest     string
}

// send sends a round's request and streams its answer (a summary's isn't
// shown).
func (s *Service) send(ctx context.Context, ws *store.Workspace, run *turnRun) sent {
	sctx, stop := context.WithCancelCause(ctx)
	defer stop(nil)
	w := &deltaWriter{s: s, ws: ws.ID, thread: run.thread.ID, turn: run.turn.ID, version: run.turn.Version, stop: stop}
	emit := w.emit
	if run.kind == "summary" {
		emit = func(llm.Delta) error { return nil }
	}
	var res llm.Response
	var cause error
	prov, callErr := s.AI.AssistantProvider(run.model)
	if callErr != nil {
		// Built here, before any request: nothing was sent.
		res.UsageKnown = true
	} else {
		watched := make(chan struct{})
		go func() {
			defer close(watched)
			s.watch(sctx, ws, run, stop)
		}()
		res, callErr = llm.StreamOrGenerate(sctx, prov, run.req, emit)
		cause = context.Cause(sctx)
		if ctx.Err() != nil {
			// The worker is stopping (a deploy): the answer was cut short.
			cause = errShutdown
		}
		stop(nil)
		<-watched
	}
	rest := w.take()
	if s.afterAnswer != nil {
		s.afterAnswer(run.turn.ID)
	}
	return sent{res: res, callErr: callErr, cause: cause, text: w.text(), rest: rest}
}

// toolCalls are a response's calls, in order.
func toolCalls(m llm.Message) []llm.ToolCall {
	out := []llm.ToolCall{}
	for _, p := range m.Parts {
		if p.ToolCall != nil {
			out = append(out, *p.ToolCall)
		}
	}
	return out
}

// watch checks, while the answer streams, that the turn is still this
// worker's and its owner may still use the assistant, and renews the
// lease; it cuts the stream (stop) when one of them no longer holds.
func (s *Service) watch(ctx context.Context, ws *store.Workspace, run *turnRun, stop context.CancelCauseFunc) {
	tick := time.NewTicker(s.watchEvery())
	defer tick.Stop()
	renewed := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		f, err := s.Q.AIAssistantTurnFence(ctx, run.turn.ID)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && (f.Version != run.turn.Version || f.State != "answering")) {
			stop(errFenced)
			return
		}
		var p *api.Problem
		if _, _, err := s.actor(ctx, s.Q, ws, run.owner, true); errors.As(err, &p) {
			stop(&accessError{err: err})
			return
		}
		if time.Since(renewed) >= renewEvery {
			now := s.now()
			n, err := s.Q.AIAssistantSlotRenew(ctx, store.AIAssistantSlotRenewParams{ExpiresAt: now.Add(turnLease), TurnID: run.turn.ID, LeaseID: run.lease, Now: now})
			if err == nil && n == 0 {
				stop(errLeaseLost)
				return
			}
			if err == nil {
				renewed = time.Now()
			}
		}
	}
}

// checkpoint accounts the round's request and, while the turn is still
// this worker's, saves what it answered: a summary, or the model's
// message and its tool calls (the turn goes on: more), or the answer that
// ends the turn.
func (s *Service) checkpoint(ctx context.Context, ws *store.Workspace, args TurnArgs, run *turnRun, r sent) ([]llm.ToolCall, bool, error) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	res, callErr, cause, text, rest := r.res, r.callErr, r.cause, r.text, r.rest
	var calls []llm.ToolCall
	more := false
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		active := false
		t, err := q.AIAssistantThreadLockByID(ctx, store.AIAssistantThreadLockByIDParams{WorkspaceID: ws.ID, ID: run.thread.ID})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var turn store.AiAssistantTurn
		if err == nil {
			turn, err = q.AIAssistantTurnLock(ctx, store.AIAssistantTurnLockParams{ThreadID: t.ID, ID: run.turn.ID})
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
			active = err == nil && turn.Version == args.Version && turn.State == "answering" && t.State == "open"
		}
		// Access is checked again before anything of the answer is kept.
		var accessErr error
		var ae *accessError
		if errors.As(cause, &ae) {
			accessErr = ae.err
		} else if active {
			var p *api.Problem
			if _, _, err := s.actor(ctx, q, ws, run.owner, true); errors.As(err, &p) {
				accessErr = err
			} else if err != nil {
				return err
			}
		}
		outcome := ai.OutcomeAnswered
		switch {
		case !active:
			outcome = ai.OutcomeCancelled
		case callErr != nil || accessErr != nil || cause != nil:
			outcome = ai.OutcomeError
		case res.Refused:
			outcome = ai.OutcomeRefused
		}
		// Always, even when the turn was stopped meanwhile: the provider may
		// charge for the request.
		known, err := s.AI.AccountAssistant(ctx, q, ai.AssistantRequest{ID: run.step, WorkspaceID: ws.ID, UserID: run.thread.UserID, ThreadID: run.thread.ID,
			Action: run.kind, Outcome: outcome, CreatedAt: run.started}, res, callErr)
		if err != nil {
			return err
		}
		stepState := "done"
		if !known {
			stepState = "uncertain"
		}
		if err := q.AIAssistantStepSave(ctx, store.AIAssistantStepSaveParams{ID: run.step, State: stepState, Now: s.now()}); err != nil {
			return err
		}
		slot, slotErr := q.AIAssistantSlotOfTurn(ctx, store.AIAssistantSlotOfTurnParams{TurnID: run.turn.ID, Now: s.now()})
		if slotErr != nil && !errors.Is(slotErr, pgx.ErrNoRows) {
			return slotErr
		}
		leaseLost := slotErr != nil || slot.LeaseID != run.lease || errors.Is(cause, errLeaseLost) || errors.Is(cause, errShutdown)
		release := func() error {
			return q.AIAssistantSlotRelease(ctx, store.AIAssistantSlotReleaseParams{TurnID: run.turn.ID, LeaseID: run.lease})
		}
		if !active {
			return release()
		}
		if rest != "" {
			if err := s.appendEvent(ctx, q, t.ID, &turn.ID, kindDelta, AIThreadDeltaEvent{TurnID: turn.ID, Text: rest}); err != nil {
				return err
			}
		}
		state, reason := "done", ""
		switch {
		case accessErr != nil:
			var p *api.Problem
			errors.As(accessErr, &p)
			state, reason = "failed", reasonOf(p)
		case leaseLost:
			state, reason = "failed", "interrupted"
		case callErr != nil:
			state, reason = "failed", providerReason(callErr)
		case res.Refused:
			state, reason = "failed", "refused"
		case res.StopReason == llm.StopLength:
			reason = "length"
		}
		if callErr == nil && res.Text != "" {
			text = res.Text
		}
		if state == "done" && run.kind == "summary" {
			// Earlier messages are summarized: later requests send the
			// summary instead of them.
			if strings.TrimSpace(text) != "" {
				msg, err := s.addMessage(ctx, q, t.ID, &turn.ID, "summary", turn.ModelID, messageContent{Text: text, Upto: run.upto})
				if err != nil {
					return err
				}
				if err := s.appendEvent(ctx, q, t.ID, &turn.ID, kindMessage, AIThreadMessageEvent{Message: msg}); err != nil {
					return err
				}
			}
			more = true
			return s.usageEvent(ctx, q, t, turn.ID)
		}
		if state == "done" && res.StopReason == llm.StopTools && len(toolCalls(res.Message)) > 0 {
			// The model's message and its calls; the turn goes on.
			replay := res.Message
			msg, err := s.addMessage(ctx, q, t.ID, &turn.ID, "assistant", turn.ModelID, messageContent{Text: text, Replay: &replay})
			if err != nil {
				return err
			}
			if strings.TrimSpace(text) != "" {
				if err := s.appendEvent(ctx, q, t.ID, &turn.ID, kindMessage, AIThreadMessageEvent{Message: msg}); err != nil {
					return err
				}
			}
			calls, more = toolCalls(res.Message), true
			if known {
				if err := q.AIAssistantStepSave(ctx, store.AIAssistantStepSaveParams{ID: run.step, State: "responded", Now: s.now()}); err != nil {
					return err
				}
			}
			return s.usageEvent(ctx, q, t, turn.ID)
		}
		if err := release(); err != nil {
			return err
		}
		if strings.TrimSpace(text) != "" {
			msg, err := s.addMessage(ctx, q, t.ID, &turn.ID, "assistant", turn.ModelID, messageContent{Text: text, Partial: state == "failed" && reason != "refused"})
			if err != nil {
				return err
			}
			if err := s.appendEvent(ctx, q, t.ID, &turn.ID, kindMessage, AIThreadMessageEvent{Message: msg}); err != nil {
				return err
			}
		}
		if err := s.usageEvent(ctx, q, t, turn.ID); err != nil {
			return err
		}
		if err := s.endTurn(ctx, q, &turn, state, reason); err != nil {
			return err
		}
		return s.touch(ctx, q, t)
	})
	return calls, more, err
}

// interrupt fails a turn whose worker was lost while it answered: what it
// wrote is kept, and its request's usage is unknown, so its budget hold
// stays until a person reconciles it.
func (s *Service) interrupt(ctx context.Context, q *store.Queries, t store.AiAssistantThread, turn *store.AiAssistantTurn) error {
	step, err := q.AIAssistantStepLast(ctx, turn.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err == nil && step.State == "requested" {
		started := step.CreatedAt
		if _, err := s.AI.AccountAssistant(ctx, q, ai.AssistantRequest{ID: step.ID, WorkspaceID: t.WorkspaceID, UserID: t.UserID, ThreadID: t.ID, Action: "answer",
			Outcome: ai.OutcomeError, CreatedAt: started}, llm.Response{}, errInterrupted); err != nil {
			return err
		}
		if err := q.AIAssistantStepSave(ctx, store.AIAssistantStepSaveParams{ID: step.ID, State: "uncertain", Now: s.now()}); err != nil {
			return err
		}
	}
	text, err := s.answerSoFar(ctx, q, t.ID, turn.ID, t.EventSeq)
	if err != nil {
		return err
	}
	if strings.TrimSpace(text) != "" {
		msg, err := s.addMessage(ctx, q, t.ID, &turn.ID, "assistant", turn.ModelID, messageContent{Text: text, Partial: true})
		if err != nil {
			return err
		}
		if err := s.appendEvent(ctx, q, t.ID, &turn.ID, kindMessage, AIThreadMessageEvent{Message: msg}); err != nil {
			return err
		}
	}
	if err := q.AIAssistantSlotDrop(ctx, turn.ID); err != nil {
		return err
	}
	if err := s.usageEvent(ctx, q, t, turn.ID); err != nil {
		return err
	}
	return s.endTurn(ctx, q, turn, "failed", "interrupted")
}

// SweepArgs is the minute's sweep of turns no worker answers.
type SweepArgs struct{}

// Kind implements river.JobArgs.
func (SweepArgs) Kind() string { return "ai.assistant_sweep" }

// InsertOpts runs it once at a time.
func (SweepArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMaintenance, UniqueOpts: river.UniqueOpts{ByPeriod: time.Minute}}
}

// SweepWorker runs the sweep.
type SweepWorker struct {
	river.WorkerDefaults[SweepArgs]
	S *Service
}

// Work implements river.Worker.
func (w *SweepWorker) Work(ctx context.Context, _ *river.Job[SweepArgs]) error {
	_, err := w.S.Sweep(ctx)
	return err
}

// Sweep fails the turns no worker answers: answering without a live lease
// (interrupted) and queued for half an hour (not_started). It reports how
// many it looked at.
func (s *Service) Sweep(ctx context.Context) (int, error) {
	now := s.now()
	rows, err := s.Q.AIAssistantSweepTurns(ctx, store.AIAssistantSweepTurnsParams{Now: now, QueuedBefore: now.Add(-queueTimeout)})
	if err != nil {
		return 0, err
	}
	n := 0
	for _, r := range rows {
		err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			t, err := q.AIAssistantThreadLockByID(ctx, store.AIAssistantThreadLockByIDParams{WorkspaceID: r.WorkspaceID, ID: r.ThreadID})
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			turn, err := q.AIAssistantTurnLock(ctx, store.AIAssistantTurnLockParams{ThreadID: t.ID, ID: r.ID})
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			switch turn.State {
			case "answering":
				if _, err := q.AIAssistantSlotOfTurn(ctx, store.AIAssistantSlotOfTurnParams{TurnID: turn.ID, Now: now}); err == nil {
					return nil
				} else if !errors.Is(err, pgx.ErrNoRows) {
					return err
				}
				return s.interrupt(ctx, q, t, &turn)
			case "queued":
				queued := turn.CreatedAt
				if turn.QueuedAt != nil {
					queued = *turn.QueuedAt
				}
				if !queued.Before(now.Add(-queueTimeout)) {
					return nil
				}
				return s.endTurn(ctx, q, &turn, "failed", "not_started")
			}
			return nil
		})
		if err != nil {
			return n, fmt.Errorf("assistant sweep: %w", err)
		}
		n++
	}
	return n, nil
}
