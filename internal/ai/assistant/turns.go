package assistant

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/editorial"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// AIThreadMessageInput is a person's message: it starts a turn.
type AIThreadMessageInput struct {
	Text string `json:"text" minLength:"1" maxLength:"20000"`
	// RequestKey makes a retry after a lost response return the same turn.
	RequestKey string `json:"requestKey" pattern:"^[A-Za-z0-9_-]{1,80}$"`
}

// SendMessage adds the person's message to one of their threads and
// queues its turn on the thread's model; the answer streams on the
// thread's events. One turn answers at a time in a thread. A question
// about reputation, blocklists or placement gets the HostRepute card at
// once, without a model call. A retry with the same request key returns
// the thread; the key reused for another message is a conflict.
func (s *Service) SendMessage(ctx context.Context, ws *store.Workspace, p *authz.Principal, thread uuid.UUID, in AIThreadMessageInput) (AIThread, error) {
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return AIThread{}, api.Invalid("body.text", mNothingToSend.Text(i18n.LangFrom(ctx), nil))
	}
	// The person writes it and the panel says the model receives it: no
	// sharing review, but no contact's record or credential either.
	if kind := editorial.Chat(text); kind != "" {
		return AIThread{}, problem(ctx, http.StatusUnprocessableEntity, codePrivateData, mPrivateData[kind], nil).WithMeta("kind", kind)
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, _, err := s.actor(ctx, q, ws, p, true)
		if err != nil {
			return err
		}
		t, err := lockThread(ctx, q, ws.ID, fresh.UserID, thread)
		if err != nil {
			return err
		}
		prior, err := q.AIAssistantTurnByRequest(ctx, store.AIAssistantTurnByRequestParams{ThreadID: thread, RequestKey: in.RequestKey})
		if err == nil {
			return s.sameMessage(ctx, q, prior, text)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if _, err := q.AIAssistantThreadActiveTurn(ctx, thread); err == nil {
			return problem(ctx, http.StatusConflict, codeTurnActive, mTurnActive, nil)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if t.ModelID == nil {
			return problem(ctx, http.StatusConflict, codeModelGone, mModelGone, nil)
		}
		m, err := s.AI.ResolveAssistantModel(ctx, q, ws.ID, fresh.UserID, t.ModelID)
		if err != nil {
			return s.modelProblem(ctx, q, ws.ID, fresh.UserID, t.ModelID, err)
		}
		reputation := ai.AsksAboutReputation(text)
		if !reputation {
			if err := s.AI.ReadyAssistant(ctx, q, ws, m); err != nil {
				return s.modelProblem(ctx, q, ws.ID, fresh.UserID, t.ModelID, err)
			}
		}
		now := s.now()
		modelID := m.ID()
		turnIn := store.AIAssistantTurnInsertParams{ID: uuid.Must(uuid.NewV7()), ThreadID: thread, ModelID: &modelID, State: "queued", RequestKey: in.RequestKey, CreatedAt: now}
		if reputation {
			// Answered here: HostRepute's boundary, no model call, no cost.
			turnIn.State, turnIn.Reason, turnIn.EndedAt = "done", "reputation", &now
		}
		turn, err := q.AIAssistantTurnInsert(ctx, turnIn)
		if err != nil {
			return err
		}
		msg, err := s.addMessage(ctx, q, thread, &turn.ID, "user", nil, messageContent{Text: text})
		if err != nil {
			return err
		}
		if err := s.appendEvent(ctx, q, thread, &turn.ID, kindMessage, AIThreadMessageEvent{Message: msg}); err != nil {
			return err
		}
		if reputation {
			card, err := s.addMessage(ctx, q, thread, &turn.ID, "assistant", nil, messageContent{Text: mReputation.Text(i18n.LangFrom(ctx), nil), Card: "hostrepute"})
			if err != nil {
				return err
			}
			if err := s.appendEvent(ctx, q, thread, &turn.ID, kindMessage, AIThreadMessageEvent{Message: card}); err != nil {
				return err
			}
		}
		if err := s.turnEvent(ctx, q, turn); err != nil {
			return err
		}
		if err := s.touch(ctx, q, t); err != nil {
			return err
		}
		if !reputation {
			if err := s.enqueue(ctx, tx, ws.ID, turn); err != nil {
				return err
			}
		}
		return s.Audit.LogWith(ctx, q, threadEvent(ws.ID, thread, "ai_assistant_turn.created",
			map[string]any{"turn_id": turn.ID, "model_id": modelID, "personal": m.Personal(), "reputation": reputation}))
	})
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" && (pg.ConstraintName == "ai_assistant_turns_thread_id_request_key_key" || pg.ConstraintName == "ai_assistant_turns_active_idx") {
		// The same request raced itself (it waited for the thread's lock,
		// so this is rare): read what the other one made.
		return s.SendMessage(ctx, ws, p, thread, in)
	}
	if err != nil {
		return AIThread{}, err
	}
	return s.Thread(ctx, ws, p, thread)
}

// sameMessage accepts a retried message: the turn its key made was for
// this text. It compares with the sealed message, never with a digest.
func (s *Service) sameMessage(ctx context.Context, q *store.Queries, turn store.AiAssistantTurn, text string) error {
	m, err := q.AIAssistantMessageOfTurn(ctx, &turn.ID)
	if err != nil {
		return err
	}
	c, err := s.messageContent(m)
	if err != nil {
		return err
	}
	if c.Text != text {
		return problem(ctx, http.StatusConflict, codeTurnKey, mTurnKey, nil)
	}
	return nil
}

// StopTurn stops one of the person's turns: no model request starts for
// it any more, and what an answering request writes after now is dropped
// (it is still accounted: the provider may charge for it). Stopping a
// turn that already ended changes nothing. The person may stop a turn with
// the assistant switched off.
func (s *Service) StopTurn(ctx context.Context, ws *store.Workspace, p *authz.Principal, thread, turn uuid.UUID) (AIThread, error) {
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, _, err := s.actor(ctx, q, ws, p, false)
		if err != nil {
			return err
		}
		th, err := lockThread(ctx, q, ws.ID, fresh.UserID, thread)
		if err != nil {
			return err
		}
		t, err := q.AIAssistantTurnLock(ctx, store.AIAssistantTurnLockParams{ThreadID: thread, ID: turn})
		if errors.Is(err, pgx.ErrNoRows) {
			return api.NotFound("Turn")
		}
		if err != nil {
			return err
		}
		if !turnActive(t.State) {
			return nil
		}
		was := t.State
		// A call waiting for the person, or allowed and not sent yet, never
		// runs.
		if err := q.AIAssistantConfirmationsClose(ctx, t.ID); err != nil {
			return err
		}
		if err := noticesGone(ctx, q, th); err != nil {
			return err
		}
		// What it wrote stays, up to the last batch before Stop: batches
		// after it are fenced, and so is the worker's checkpoint.
		text, err := s.answerSoFar(ctx, q, th.ID, t.ID, th.EventSeq)
		if err != nil {
			return err
		}
		if strings.TrimSpace(text) != "" {
			msg, err := s.addMessage(ctx, q, th.ID, &t.ID, "assistant", t.ModelID, messageContent{Text: text, Partial: true})
			if err != nil {
				return err
			}
			if err := s.appendEvent(ctx, q, th.ID, &t.ID, kindMessage, AIThreadMessageEvent{Message: msg}); err != nil {
				return err
			}
		}
		if err := s.endTurn(ctx, q, &t, "stopped", "stopped"); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, threadEvent(ws.ID, thread, "ai_assistant_turn.cancelled", map[string]any{"turn_id": t.ID, "was": was}))
	})
	if err != nil {
		return AIThread{}, err
	}
	// As the person may read it now, switched off or not.
	return s.thread(ctx, ws, p, thread, false)
}

// endTurn ends a locked turn and tells its streams. Its version moves on,
// so its worker writes nothing more of it.
func (s *Service) endTurn(ctx context.Context, q *store.Queries, t *store.AiAssistantTurn, state, reason string) error {
	now := s.now()
	t.State, t.Reason, t.EndedAt = state, reason, &now
	t.Version++
	if err := q.AIAssistantTurnSave(ctx, store.AIAssistantTurnSaveParams{ID: t.ID, State: t.State, Reason: t.Reason, Version: t.Version, StartedAt: t.StartedAt,
		EndedAt: t.EndedAt}); err != nil {
		return err
	}
	return s.turnEvent(ctx, q, *t)
}
