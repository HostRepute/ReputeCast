// Package assistant is the AI assistant's threads (assistant brief D1, D7;
// milestone B3): a person's conversations in a workspace, with the model
// each uses, what the person attached and confirmed sharing, and their
// cumulative ceilings. A thread is its owner's only: nobody else reads it,
// integration managers included; they see usage, never content.
//
// Content (titles, messages, attached snapshots, stream events) is sealed
// at rest with the instance's key. Models, budgets and projections come
// from internal/ai through Models; this package never opens a key.
package assistant

import (
	"context"
	"embed"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/secrets"
	"github.com/hostrepute/reputecast/internal/store"
)

// Models is what the assistant uses of internal/ai: the models a thread
// may use, the assistant's settings and what a person may attach.
// *ai.Service is one.
type Models interface {
	AssistantSettings(ctx context.Context, q *store.Queries, ws uuid.UUID) (ai.AIAssistantSettings, error)
	AssistantModels(ctx context.Context, ws *store.Workspace, p *authz.Principal) (ai.AIAssistantModelList, error)
	ResolveAssistantModel(ctx context.Context, q *store.Queries, ws, user uuid.UUID, model *uuid.UUID) (*ai.AssistantModel, error)
	AssistantModelView(ctx context.Context, q *store.Queries, ws, user uuid.UUID, model *uuid.UUID) (*ai.AIAssistantModel, error)
	AssistantAttachment(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, kind string, id uuid.UUID, snapshot bool) (ai.AssistantAttachment, error)
}

var _ Models = (*ai.Service)(nil)

// Service is the assistant's threads.
type Service struct {
	Pool  *pgxpool.Pool
	Q     *store.Queries
	Audit *audit.Logger
	Box   *secrets.Box
	AI    Models
	Log   *slog.Logger
	// Now is the clock (tests); nil is the wall clock.
	Now func() time.Time
}

// Retention is how long a thread keeps its content after its last activity.
const Retention = 30 * 24 * time.Hour

// maxContexts bounds what one thread has attached.
const maxContexts = 10

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

//go:embed messages/*.json
var messagesFS embed.FS

var msgs = i18n.MustLoad(messagesFS, "assistant")

var (
	mOff           = msgs.M("problem.off")
	mRequestKey    = msgs.M("problem.requestKey")
	mSharingReview = msgs.M("problem.sharingReview")
	mContextsFull  = msgs.M("problem.contextsFull")
	mExpired       = msgs.M("problem.expired")
	mNothingToSave = msgs.M("problem.nothingToSave")
	mCursor        = msgs.M("problem.cursor")
)

// Error codes of the assistant's problems.
const (
	codeOff           = "ai_assistant_off"
	codeRequestKey    = "ai_assistant_request_key"
	codeSharingReview = "ai_assistant_sharing_review"
	codeContextsFull  = "ai_assistant_contexts_full"
	codeExpired       = "ai_assistant_thread_expired"
	codeNothingToSave = "ai_assistant_nothing_to_save"
)

func problem(ctx context.Context, status int, code string, m i18n.Msg, args i18n.Args) *api.Problem {
	return api.Err(status, code, m.Text(i18n.LangFrom(ctx), args))
}

// actor is p as the workspace knows them now: a person with ai:use, and
// (on) with the assistant switched on. Membership and role are read again
// on every call, whatever the session says.
func (s *Service) actor(ctx context.Context, q *store.Queries, ws *store.Workspace, p *authz.Principal, on bool) (*authz.Principal, ai.AIAssistantSettings, error) {
	fresh, err := brandkit.VoiceActor(ctx, q, ws, p)
	if err != nil {
		return nil, ai.AIAssistantSettings{}, err
	}
	if !fresh.Can(authz.AIUse) {
		return nil, ai.AIAssistantSettings{}, api.Forbidden(authz.AIUse, nil)
	}
	settings, err := s.AI.AssistantSettings(ctx, q, ws.ID)
	if err != nil {
		return nil, settings, err
	}
	if on && !settings.Enabled {
		return nil, settings, problem(ctx, http.StatusConflict, codeOff, mOff, nil)
	}
	return fresh, settings, nil
}

// threadEvent is an audit entry about a thread: never its title or content.
func threadEvent(ws, thread uuid.UUID, action string, meta map[string]any) audit.Event {
	if meta == nil {
		meta = map[string]any{}
	}
	meta["via"] = "ai_assistant"
	return audit.Event{WorkspaceID: &ws, Action: action, TargetType: "ai_assistant_thread", TargetID: thread.String(), Metadata: meta}
}
