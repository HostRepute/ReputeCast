package ai

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// The AI assistant's switch, the limits each thread starts with and how
// many turns answer at once (assistant brief §3.4, B3). Managers change
// them; the threads themselves are internal/ai/assistant's.

// AIAssistantSettings are the assistant's settings. Limits are each new
// thread's cumulative ceilings.
type AIAssistantSettings struct {
	Enabled bool          `json:"enabled"`
	Limits  AIAgentLimits `json:"limits"`
	// PerPerson and PerWorkspace bound the turns answering at once (one per
	// thread), apart from agent runs.
	PerPerson    int32 `json:"perPerson" minimum:"1" maximum:"10"`
	PerWorkspace int32 `json:"perWorkspace" minimum:"1" maximum:"100"`
	// Revision is the settings' version: a change names the one it read.
	Revision int64 `json:"revision" minimum:"0"`
}

// assistantDefaults are the settings of a workspace that never set them
// (the brief's defaults, the agent's limits).
var assistantDefaults = AIAssistantSettings{Limits: AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}, PerPerson: 2, PerWorkspace: 10}

func assistantSettingsOf(r store.AiAssistantSetting) AIAssistantSettings {
	return AIAssistantSettings{Enabled: r.Enabled, Limits: AIAgentLimits{ToolCalls: r.ToolLimit, Tokens: r.TokenLimit, CostMicros: r.CostLimitMicros},
		PerPerson: r.PerPerson, PerWorkspace: r.PerWorkspace, Revision: r.Revision}
}

// AssistantSettings returns the assistant's settings, its defaults when
// they were never set.
func (s *Service) AssistantSettings(ctx context.Context, q *store.Queries, ws uuid.UUID) (AIAssistantSettings, error) {
	r, err := q.AIAssistantSettings(ctx, ws)
	if errors.Is(err, pgx.ErrNoRows) {
		return assistantDefaults, nil
	}
	if err != nil {
		return AIAssistantSettings{}, err
	}
	return assistantSettingsOf(r), nil
}

// SetAssistantSettings turns the assistant on or off and sets each new
// thread's limits and the turns at once. It takes the AI configuration
// lock, like every change a request is bound by.
func (s *Service) SetAssistantSettings(ctx context.Context, ws *store.Workspace, p *authz.Principal, in AIAssistantSettings) (AISettings, error) {
	if in.Limits.ToolCalls < 1 || in.Limits.ToolCalls > 200 || in.Limits.Tokens < 1000 || in.Limits.Tokens > 2000000 ||
		in.Limits.CostMicros < 1 || in.Limits.CostMicros > 100000000 || in.PerPerson < 1 || in.PerPerson > 10 || in.PerWorkspace < 1 || in.PerWorkspace > 100 {
		return AISettings{}, problem(ctx, http.StatusUnprocessableEntity, codeAssistantLimits, mAssistantLimits, nil)
	}
	err := db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		fresh, err := draftActor(ctx, q, ws, p, false, authz.IntegrationsManage)
		if err != nil {
			return err
		}
		if _, err := lockSettings(ctx, q, ws.ID); err != nil {
			return err
		}
		cur, err := s.AssistantSettings(ctx, q, ws.ID)
		if err != nil {
			return err
		}
		if cur.Revision != in.Revision {
			return problem(ctx, http.StatusConflict, codeAssistantChanged, mAssistantChanged, nil)
		}
		if err := q.AIAssistantSettingsSave(ctx, store.AIAssistantSettingsSaveParams{WorkspaceID: ws.ID, Enabled: in.Enabled, ToolLimit: in.Limits.ToolCalls,
			TokenLimit: in.Limits.Tokens, CostLimitMicros: in.Limits.CostMicros, PerPerson: in.PerPerson, PerWorkspace: in.PerWorkspace,
			UpdatedBy: actorID(fresh), UpdatedAt: s.now()}); err != nil {
			return err
		}
		view := func(v AIAssistantSettings) map[string]any {
			return map[string]any{"enabled": v.Enabled, "limits": v.Limits, "perPerson": v.PerPerson, "perWorkspace": v.PerWorkspace}
		}
		return s.Audit.LogWith(audit.WithPrincipal(ctx, fresh), q, audit.Event{WorkspaceID: &ws.ID, Action: "ai_assistant.settings_updated", TargetType: "ai_assistant",
			TargetID: ws.ID.String(), TargetName: "AI assistant", Before: view(cur), After: view(in), Metadata: map[string]any{"via": "ai_assistant"}})
	})
	if err != nil {
		return AISettings{}, err
	}
	return s.Settings(ctx, ws, p)
}
