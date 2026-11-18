package assistant

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// AIThread is a thread as its owner sees it.
type AIThread struct {
	ID uuid.UUID `json:"id"`
	// Title is the person's name for the thread ("" until they give one).
	Title  string `json:"title"`
	State  string `json:"state" enum:"open,expired"`
	Screen string `json:"screen,omitempty"`
	// Model is the model the thread's turns use, absent once it was removed.
	Model    *ai.AIAssistantModel `json:"model,omitempty"`
	Limits   ai.AIAgentLimits     `json:"limits"`
	Usage    AIThreadUsage        `json:"usage"`
	Contexts []AIThreadContext    `json:"contexts"`
	Messages []AIThreadMessage    `json:"messages"`
	// Turn is the thread's latest turn: the one answering now, or how the
	// last one ended. Absent before the first message.
	Turn *AIThreadTurn `json:"turn,omitempty"`
	// Confirmation is the call of a writing MCP tool its turn waits on
	// (the turn is "waiting"): its owner allows or denies it.
	Confirmation *AIThreadConfirmation `json:"confirmation,omitempty"`
	// MCPServerIDs are the MCP servers the thread may use.
	MCPServerIDs []uuid.UUID `json:"mcpServerIds"`
	// EventSeq is the thread's last stream event: a stream resumes after it.
	EventSeq       int64     `json:"eventSeq"`
	CreatedAt      time.Time `json:"createdAt"`
	LastActivityAt time.Time `json:"lastActivityAt"`
	// ExpiresAt is when the content goes, 30 days after the last activity.
	ExpiresAt time.Time `json:"expiresAt"`
}

// AIThreadUsage is what a thread used of its ceilings, reserved included.
// A request whose usage the provider never reported (one stopped, or cut
// short) is uncertain: it counts at its reserved maximum, in Tokens and
// CostMicros too, until a person reconciles it.
type AIThreadUsage struct {
	ToolCalls           int32 `json:"toolCalls"`
	Tokens              int64 `json:"tokens"`
	CostMicros          int64 `json:"costMicros"`
	Requests            int64 `json:"requests"`
	UncertainRequests   int64 `json:"uncertainRequests"`
	UncertainTokens     int64 `json:"uncertainTokens"`
	UncertainCostMicros int64 `json:"uncertainCostMicros"`
}

func usageOf(t store.AiAssistantThread, u store.AIBudgetThreadRow) AIThreadUsage {
	return AIThreadUsage{ToolCalls: t.ToolCalls, Tokens: u.Tokens, CostMicros: u.CostMicros, Requests: u.Requests, UncertainRequests: u.Uncertain,
		UncertainTokens: u.UncertainTokens, UncertainCostMicros: u.UncertainCostMicros}
}

// AIThreadContext is a resource attached to a thread. Available is false
// when the person can no longer read it: it is then never read.
type AIThreadContext struct {
	ID        uuid.UUID `json:"id"`
	Kind      string    `json:"kind" enum:"campaign,post,report,segment,automation"`
	TargetID  uuid.UUID `json:"targetId"`
	Title     string    `json:"title"`
	Available bool      `json:"available"`
	Revision  int64     `json:"revision"`
	SharedAt  time.Time `json:"sharedAt"`
}

// AIThreadMessage is one message of a thread: the person's, the
// assistant's, the tools a turn called (their rows), or the marker where
// earlier messages were summarized (summary).
type AIThreadMessage struct {
	ID     uuid.UUID  `json:"id"`
	Role   string     `json:"role" enum:"user,assistant,tool,summary"`
	Text   string     `json:"text"`
	TurnID *uuid.UUID `json:"turnId,omitempty"`
	// Partial: the answer stopped before it ended (its turn failed); it is
	// what arrived.
	Partial bool `json:"partial,omitempty"`
	// Card is a fixed answer the app shows as a card, without a model call:
	// hostrepute, for questions about reputation, blocklists or placement.
	Card string `json:"card,omitempty" enum:"hostrepute"`
	// Tools are a tool message's calls.
	Tools     []AIThreadToolRow `json:"tools,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
}

// AIThreadTurn is a turn's state: Queued, Answering, Waiting for you,
// Stopped, Failed or Done. Reason says why it stopped, failed or ended
// short: stopped; interrupted (the worker was lost, so its usage is
// unknown); not_started (no worker took it within half an hour);
// confirmation_expired (a call waited a day for its owner: nothing was
// sent); access_lost, assistant_off; the thread's model: model_gone (it
// was removed, with its account or alone), account_disconnected (its
// account was disconnected), key_rejected, tools_unchecked (its tool
// check isn't passed, for example after its key was replaced),
// personal_keys_off (it is on the owner's own key and personal keys were
// turned off), not_configured; a budget: cap_reached, reservation_cap,
// personal_cap_reached, thread_cap…; the provider: rate_limited, timeout,
// provider_error, too_large; refused; length (done, cut at its length);
// reputation (done, the HostRepute card); expired (retention removed the
// thread's content while it answered).
type AIThreadTurn struct {
	ID       uuid.UUID `json:"id"`
	Sequence int32     `json:"sequence"`
	State    string    `json:"state" enum:"queued,answering,waiting,stopped,failed,done"`
	Reason   string    `json:"reason,omitempty"`
	// ModelID is the model the turn uses, fixed when it was sent.
	ModelID *uuid.UUID `json:"modelId,omitempty"`
	// Text is the answer so far while it answers.
	Text      string     `json:"text,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	StartedAt *time.Time `json:"startedAt,omitempty"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
}

// AIThreadListItem is a thread in the thread list.
type AIThreadListItem struct {
	ID             uuid.UUID  `json:"id"`
	Title          string     `json:"title"`
	State          string     `json:"state" enum:"open,expired"`
	ModelID        *uuid.UUID `json:"modelId,omitempty"`
	LastActivityAt time.Time  `json:"lastActivityAt"`
	ExpiresAt      time.Time  `json:"expiresAt"`
}

// AIThreadList is a page of the person's threads, the most recently active
// first.
type AIThreadList struct {
	Items []AIThreadListItem `json:"items"`
	Next  string             `json:"next,omitempty"`
}

// AIThreadInput starts a thread.
type AIThreadInput struct {
	Title string `json:"title,omitempty" maxLength:"120"`
	// ModelID picks the model; empty takes the person's own pick for the
	// assistant, or else the workspace's.
	ModelID *uuid.UUID `json:"modelId,omitempty" format:"uuid"`
	// Screen is the route key of the screen it was opened on.
	Screen string `json:"screen,omitempty" maxLength:"80" pattern:"^[a-z0-9_.-]*$"`
	// RequestKey makes a retry after a lost response return the same thread.
	RequestKey string `json:"requestKey" pattern:"^[A-Za-z0-9_-]{1,80}$"`
}

// AIThreadUpdate renames a thread or picks another model for its next turns.
type AIThreadUpdate struct {
	Title   *string    `json:"title,omitempty" maxLength:"120"`
	ModelID *uuid.UUID `json:"modelId,omitempty" format:"uuid"`
}

// titleAD is the associated data a thread's title is sealed with.
func titleAD(thread uuid.UUID) string { return "ai_assistant_thread:" + thread.String() + ":title" }

// contextAD is an attached resource's snapshot's.
func contextAD(context uuid.UUID) string { return "ai_assistant_context:" + context.String() }

// sealTitle seals a title; an empty one is none.
func (s *Service) sealTitle(thread uuid.UUID, title string) ([]byte, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, nil
	}
	return s.Box.SealString(title, titleAD(thread))
}

func (s *Service) title(t store.AiAssistantThread) (string, error) {
	if len(t.TitleSealed) == 0 {
		return "", nil
	}
	return s.Box.OpenString(t.TitleSealed, titleAD(t.ID))
}

func threadNotFound() error { return api.NotFound("Thread") }

// CreateThread starts a thread on a model the person may use. A retry with
// the same request key returns the same thread; the key reused for another
// request is a conflict.
func (s *Service) CreateThread(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIThreadInput) (AIThread, error) {
	var id uuid.UUID
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, settings, err := s.actor(ctx, q, ws, p, true)
		if err != nil {
			return err
		}
		prior, err := q.AIAssistantThreadByRequest(ctx, store.AIAssistantThreadByRequestParams{WorkspaceID: ws.ID, UserID: fresh.UserID, RequestKey: &in.RequestKey})
		if err == nil {
			same, err := s.sameRequest(prior, in)
			if err != nil {
				return err
			}
			if !same {
				return problem(ctx, http.StatusConflict, codeRequestKey, mRequestKey, nil)
			}
			id = prior.ID
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		model, err := s.AI.ResolveAssistantModel(ctx, q, ws.ID, fresh.UserID, in.ModelID)
		if err != nil {
			return err
		}
		id = uuid.Must(uuid.NewV7())
		title, err := s.sealTitle(id, in.Title)
		if err != nil {
			return err
		}
		now := s.now()
		modelID := model.ID()
		if err := q.AIAssistantThreadInsert(ctx, store.AIAssistantThreadInsertParams{ID: id, WorkspaceID: ws.ID, UserID: fresh.UserID, ModelID: &modelID,
			TitleSealed: title, Screen: in.Screen, ToolLimit: settings.Limits.ToolCalls, TokenLimit: settings.Limits.Tokens, CostLimitMicros: settings.Limits.CostMicros,
			RequestKey: &in.RequestKey, Now: now, ExpiresAt: now.Add(Retention)}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, threadEvent(ws.ID, id, "ai_assistant_thread.created",
			map[string]any{"model_id": modelID, "personal": model.Personal(), "limits": settings.Limits}))
	})
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" && pg.ConstraintName == "ai_assistant_threads_request_key_idx" {
		// The same request raced itself: the other one made the thread.
		return s.CreateThread(ctx, ws, p, in)
	}
	if err != nil {
		return AIThread{}, err
	}
	return s.Thread(ctx, ws, p, id)
}

// sameRequest reports whether a thread is the one in asked for. It compares
// with the sealed title: a digest of the request kept beside it would
// confirm guesses about the title. A thread made on the default model is
// the same request whatever the default is now.
func (s *Service) sameRequest(t store.AiAssistantThread, in AIThreadInput) (bool, error) {
	title, err := s.title(t)
	if err != nil {
		return false, err
	}
	if title != strings.TrimSpace(in.Title) || t.Screen != in.Screen {
		return false, nil
	}
	return in.ModelID == nil || (t.ModelID != nil && *t.ModelID == *in.ModelID), nil
}

// Thread returns one of the person's threads.
func (s *Service) Thread(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) (AIThread, error) {
	fresh, _, err := s.actor(ctx, s.Q, ws, p, true)
	if err != nil {
		return AIThread{}, err
	}
	t, err := s.Q.AIAssistantThread(ctx, store.AIAssistantThreadParams{WorkspaceID: ws.ID, UserID: fresh.UserID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return AIThread{}, threadNotFound()
	}
	if err != nil {
		return AIThread{}, err
	}
	return s.view(ctx, s.Q, ws, fresh, t)
}

// view is a thread with its usage, its attached resources as the person
// may read them now, its messages and its latest turn, as q reads them.
func (s *Service) view(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, t store.AiAssistantThread) (AIThread, error) {
	title, err := s.title(t)
	if err != nil {
		return AIThread{}, err
	}
	out := AIThread{ID: t.ID, Title: title, State: t.State, Screen: t.Screen, EventSeq: t.EventSeq, CreatedAt: t.CreatedAt, LastActivityAt: t.LastActivityAt,
		ExpiresAt: t.ExpiresAt, Limits: ai.AIAgentLimits{ToolCalls: t.ToolLimit, Tokens: t.TokenLimit, CostMicros: t.CostLimitMicros},
		Contexts: []AIThreadContext{}, Messages: []AIThreadMessage{}, MCPServerIDs: []uuid.UUID{}}
	if out.MCPServerIDs, err = q.MCPThreadServers(ctx, t.ID); err != nil {
		return AIThread{}, err
	}
	if out.MCPServerIDs == nil {
		out.MCPServerIDs = []uuid.UUID{}
	}
	if out.Model, err = s.AI.AssistantModelView(ctx, q, ws.ID, t.UserID, t.ModelID); err != nil {
		return AIThread{}, err
	}
	used, err := q.AIBudgetThread(ctx, &t.ID)
	if err != nil {
		return AIThread{}, err
	}
	out.Usage = usageOf(t, used)
	contexts, err := q.AIAssistantContexts(ctx, t.ID)
	if err != nil {
		return AIThread{}, err
	}
	for _, c := range contexts {
		v := AIThreadContext{ID: c.ID, Kind: c.Kind, TargetID: c.TargetID, Revision: c.Revision, SharedAt: c.SharedAt}
		a, err := s.AI.AssistantAttachment(ctx, q, ws, p, c.Kind, c.TargetID, false)
		var pr *api.Problem
		switch {
		case err == nil:
			v.Title, v.Available = a.Title, true
		case !errors.As(err, &pr):
			return AIThread{}, err
		}
		out.Contexts = append(out.Contexts, v)
	}
	messages, err := q.AIAssistantMessages(ctx, t.ID)
	if err != nil {
		return AIThread{}, err
	}
	for _, m := range messages {
		c, err := s.messageContent(m)
		if err != nil {
			return AIThread{}, err
		}
		if m.Role == "assistant" && strings.TrimSpace(c.Text) == "" {
			continue // only tool calls: the tool message shows them
		}
		out.Messages = append(out.Messages, c.view(m))
	}
	turn, err := q.AIAssistantThreadLastTurn(ctx, t.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return AIThread{}, err
	}
	v := turnView(turn)
	if turnActive(turn.State) {
		if v.Text, err = s.answerSoFar(ctx, q, t.ID, turn.ID, t.EventSeq); err != nil {
			return AIThread{}, err
		}
	}
	out.Turn = &v
	if turn.State == "waiting" {
		c, err := q.AIAssistantConfirmationPending(ctx, t.ID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return AIThread{}, err
		}
		if err == nil {
			cv, err := s.confirmationView(c)
			if err != nil {
				return AIThread{}, err
			}
			out.Confirmation = &cv
		}
	}
	return out, nil
}

func turnView(t store.AiAssistantTurn) AIThreadTurn {
	return AIThreadTurn{ID: t.ID, Sequence: t.Sequence, State: t.State, Reason: t.Reason, ModelID: t.ModelID, CreatedAt: t.CreatedAt, StartedAt: t.StartedAt, EndedAt: t.EndedAt}
}

// turnActive: the turn is queued, answering or waiting for the person.
func turnActive(state string) bool {
	return state == "queued" || state == "answering" || state == "waiting"
}

// messageAD is the associated data a message is sealed with.
func messageAD(message uuid.UUID) string { return "ai_assistant_message:" + message.String() }

// messageContent is a message as it is sealed. Replay is the model's own
// message (its tool calls, Gemini's thought parts), sent back to the same
// model; Results a tool message's results as the model read them; Upto the
// last message a summary covers.
type messageContent struct {
	Text    string            `json:"text"`
	Partial bool              `json:"partial,omitempty"`
	Card    string            `json:"card,omitempty"`
	Replay  *llm.Message      `json:"replay,omitempty"`
	Results []llm.Part        `json:"results,omitempty"`
	Tools   []AIThreadToolRow `json:"tools,omitempty"`
	Upto    int32             `json:"upto,omitempty"`
}

func (c messageContent) view(m store.AiAssistantMessage) AIThreadMessage {
	return AIThreadMessage{ID: m.ID, Role: m.Role, Text: c.Text, TurnID: m.TurnID, Partial: c.Partial, Card: c.Card, Tools: c.Tools, CreatedAt: m.CreatedAt}
}

func (s *Service) messageContent(m store.AiAssistantMessage) (messageContent, error) {
	var c messageContent
	raw, err := s.Box.Open(m.ContentSealed, messageAD(m.ID))
	if err != nil {
		return c, err
	}
	return c, json.Unmarshal(raw, &c)
}

// addMessage seals and saves a message of a locked thread.
func (s *Service) addMessage(ctx context.Context, q *store.Queries, thread uuid.UUID, turn *uuid.UUID, role string, model *uuid.UUID, c messageContent) (AIThreadMessage, error) {
	id := uuid.Must(uuid.NewV7())
	raw, err := json.Marshal(c)
	if err != nil {
		return AIThreadMessage{}, err
	}
	sealed, err := s.Box.Seal(raw, messageAD(id))
	if err != nil {
		return AIThreadMessage{}, err
	}
	now := s.now()
	if err := q.AIAssistantMessageInsert(ctx, store.AIAssistantMessageInsertParams{ID: id, ThreadID: thread, TurnID: turn, Role: role, ModelID: model,
		ContentSealed: sealed, Bytes: int32(len(raw)), CreatedAt: now}); err != nil {
		return AIThreadMessage{}, err
	}
	return c.view(store.AiAssistantMessage{ID: id, Role: role, TurnID: turn, CreatedAt: now}), nil
}

// maxSearched bounds the threads a title search decrypts.
const maxSearched = 500

// Threads lists the person's threads; with search, the ones whose title has
// it (titles are sealed, so they are searched here, not in the database).
func (s *Service) Threads(ctx context.Context, ws *store.Workspace, p *authz.Principal, search, cursor string, limit int) (AIThreadList, error) {
	out := AIThreadList{Items: []AIThreadListItem{}}
	fresh, _, err := s.actor(ctx, s.Q, ws, p, true)
	if err != nil {
		return out, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	search = strings.ToLower(strings.TrimSpace(search))
	params := store.AIAssistantThreadsParams{WorkspaceID: ws.ID, UserID: fresh.UserID, Lim: int32(limit + 1)}
	if search != "" {
		params.Lim = maxSearched
	} else if cursor != "" {
		before, beforeID, ok := parseCursor(cursor)
		if !ok {
			return out, api.Invalid("query.cursor", mCursor.Text(i18n.LangFrom(ctx), nil))
		}
		params.Before, params.BeforeID = &before, &beforeID
	}
	rows, err := s.Q.AIAssistantThreads(ctx, params)
	if err != nil {
		return out, err
	}
	for _, t := range rows {
		if len(out.Items) == limit {
			if search == "" {
				last := out.Items[len(out.Items)-1]
				out.Next = cursorOf(last.LastActivityAt, last.ID)
			}
			break
		}
		title, err := s.title(t)
		if err != nil {
			return out, err
		}
		if search != "" && !strings.Contains(strings.ToLower(title), search) {
			continue
		}
		out.Items = append(out.Items, AIThreadListItem{ID: t.ID, Title: title, State: t.State, ModelID: t.ModelID, LastActivityAt: t.LastActivityAt, ExpiresAt: t.ExpiresAt})
	}
	return out, nil
}

func cursorOf(at time.Time, id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString([]byte(at.UTC().Format(time.RFC3339Nano) + "|" + id.String()))
}

func parseCursor(c string) (time.Time, uuid.UUID, bool) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, uuid.Nil, false
	}
	at, id, ok := strings.Cut(string(b), "|")
	t, err1 := time.Parse(time.RFC3339Nano, at)
	u, err2 := uuid.Parse(id)
	return t, u, ok && err1 == nil && err2 == nil
}

// lockThread locks one of the person's open threads.
func lockThread(ctx context.Context, q *store.Queries, ws, user, id uuid.UUID) (store.AiAssistantThread, error) {
	t, err := q.AIAssistantThreadLock(ctx, store.AIAssistantThreadLockParams{WorkspaceID: ws, UserID: user, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return t, threadNotFound()
	}
	if err != nil {
		return t, err
	}
	if t.State != "open" {
		return t, problem(ctx, http.StatusConflict, codeExpired, mExpired, nil)
	}
	return t, nil
}

// UpdateThread renames a thread or picks another model for its next turns.
// Turns pin their model when they start, so a turn answering keeps its own.
func (s *Service) UpdateThread(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID, in AIThreadUpdate) (AIThread, error) {
	if in.Title == nil && in.ModelID == nil {
		return AIThread{}, problem(ctx, http.StatusUnprocessableEntity, codeNothingToSave, mNothingToSave, nil)
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, _, err := s.actor(ctx, q, ws, p, true)
		if err != nil {
			return err
		}
		t, err := lockThread(ctx, q, ws.ID, fresh.UserID, id)
		if err != nil {
			return err
		}
		changed := []string{}
		meta := map[string]any{}
		title, model := t.TitleSealed, t.ModelID
		if in.Title != nil {
			if title, err = s.sealTitle(id, *in.Title); err != nil {
				return err
			}
			changed = append(changed, "title")
		}
		if in.ModelID != nil {
			m, err := s.AI.ResolveAssistantModel(ctx, q, ws.ID, fresh.UserID, in.ModelID)
			if err != nil {
				return err
			}
			modelID := m.ID()
			model = &modelID
			changed = append(changed, "model")
			meta["model_id"], meta["personal"] = modelID, m.Personal()
		}
		now := s.now()
		if err := q.AIAssistantThreadSave(ctx, store.AIAssistantThreadSaveParams{TitleSealed: title, ModelID: model, Now: now, ExpiresAt: now.Add(Retention),
			WorkspaceID: ws.ID, ID: id}); err != nil {
			return err
		}
		slices.Sort(changed)
		meta["changed"] = changed
		return s.Audit.LogWith(ctx, q, threadEvent(ws.ID, id, "ai_assistant_thread.updated", meta))
	})
	if err != nil {
		return AIThread{}, err
	}
	return s.Thread(ctx, ws, p, id)
}

// DeleteThread deletes one of the person's threads and all its content at
// once: messages, attached resources and stream events. The activity log
// and the budget ledger keep its usage, without content. A person may
// delete their thread with the assistant switched off.
func (s *Service) DeleteThread(ctx context.Context, ws *store.Workspace, p *authz.Principal, id uuid.UUID) error {
	return db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, _, err := s.actor(ctx, q, ws, p, false)
		if err != nil {
			return err
		}
		n, err := q.AIAssistantThreadDelete(ctx, store.AIAssistantThreadDeleteParams{WorkspaceID: ws.ID, UserID: fresh.UserID, ID: id})
		if err != nil {
			return err
		}
		if n == 0 {
			return threadNotFound()
		}
		if err := noticesGone(ctx, q, store.AiAssistantThread{ID: id, UserID: fresh.UserID}); err != nil {
			return err
		}
		return s.Audit.LogWith(ctx, q, threadEvent(ws.ID, id, "ai_assistant_thread.deleted", nil))
	})
}
