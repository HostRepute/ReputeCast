package ai

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/authz"
)

type agentSettingsInput struct {
	api.WorkspaceInput
	Body AIAgentSettings
}

type agentBriefInput struct {
	api.WorkspaceInput
	Body AIAgentBrief
}

type agentListInput struct {
	api.WorkspaceInput
	Cursor string `query:"cursor" maxLength:"200"`
}

type agentRunPath struct {
	api.WorkspaceInput
	ID uuid.UUID `path:"id"`
}

type agentActionInput struct {
	api.WorkspaceInput
	ID   uuid.UUID `path:"id"`
	Body AIAgentAction
}

type agentLimitInput struct {
	api.WorkspaceInput
	ID   uuid.UUID `path:"id"`
	Body AIAgentLimitChange
}

type agentDecisionInput struct {
	api.WorkspaceInput
	ID   uuid.UUID `path:"id"`
	Body AIAgentDecision
}

func registerAgent(h huma.API, s *Service) {
	base := "/workspaces/{workspace}/ai/agent"
	tags := []string{"AI agent"}
	api.Get(h, api.Op{ID: "get-ai-agent-settings", Path: base + "/settings", Summary: "Agent availability and default cumulative run ceilings", Tags: tags, Access: api.WorkspaceHuman(authz.IntegrationsView)},
		func(ctx context.Context, in *settingsInput) (*api.Body[AIAgentSettings], error) {
			out, err := s.AgentSettings(ctx, api.WorkspaceFrom(ctx))
			return api.OK(out), err
		})
	api.Put(h, api.Op{ID: "set-ai-agent-settings", Path: base + "/settings", Summary: "Configure Agent and defaults without changing existing runs", Tags: tags, Access: api.WorkspaceHuman(authz.IntegrationsManage)},
		func(ctx context.Context, in *agentSettingsInput) (*api.Empty, error) {
			return api.NoContent, s.SetAgentSettings(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
		})
	api.Post(h, api.Op{ID: "test-ai-agent-tools", Path: base + "/test", Summary: "Test tool calling with fixed data, using the monthly AI budget", Tags: tags, Access: api.WorkspaceHuman(authz.IntegrationsManage)},
		func(ctx context.Context, in *settingsInput) (*api.Body[AIAgentCapability], error) {
			out, err := s.TestAgentTools(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx))
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "start-ai-agent", Path: base + "/runs", Summary: "Start a task and queue its first planning request", Tags: tags, Access: api.WorkspaceHuman(authz.AIUse)},
		func(ctx context.Context, in *agentBriefInput) (*api.Body[AIAgentRunView], error) {
			out, err := s.StartAgent(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Body)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "list-ai-agent-runs", Path: base + "/runs", Summary: "List only runs permitted by current membership and resource scope", Tags: tags, Access: api.WorkspaceHuman("")},
		func(ctx context.Context, in *agentListInput) (*api.Body[AIAgentRunList], error) {
			out, err := s.AgentRuns(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.Cursor)
			return api.OK(out), err
		})
	api.Get(h, api.Op{ID: "get-ai-agent-run", Path: base + "/runs/{id}", Summary: "Read a permitted task and its reviewable drafts without hidden reasoning", Tags: tags, Access: api.WorkspaceHuman("")},
		func(ctx context.Context, in *agentRunPath) (*api.Body[AIAgentRunView], error) {
			out, err := s.AgentRun(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID)
			return api.OK(out), err
		})
	api.Post(h, api.Op{ID: "act-on-ai-agent-run", Path: base + "/runs/{id}/actions", Summary: "Confirm a plan, answer, stop or continue without resetting usage", Tags: tags, Access: api.WorkspaceHuman("")},
		func(ctx context.Context, in *agentActionInput) (*api.Empty, error) {
			return api.NoContent, s.AgentAction(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
		})
	api.Put(h, api.Op{ID: "set-ai-agent-run-limits", Path: base + "/runs/{id}/limits", Summary: "Raise cumulative limits on a paused run without continuing it", Tags: tags, Access: api.WorkspaceHuman(authz.IntegrationsManage)},
		func(ctx context.Context, in *agentLimitInput) (*api.Empty, error) {
			return api.NoContent, s.SetAgentRunLimits(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
		})
	api.Post(h, api.Op{ID: "decide-ai-agent-drafts", Path: base + "/runs/{id}/decisions", Summary: "Keep or discard new drafts; accept or reject exact existing-resource proposals", Tags: tags, Access: api.WorkspaceHuman("")},
		func(ctx context.Context, in *agentDecisionInput) (*api.Empty, error) {
			return api.NoContent, s.DecideAgent(ctx, api.WorkspaceFrom(ctx), api.PrincipalFrom(ctx), in.ID, in.Body)
		})
}
