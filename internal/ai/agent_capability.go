package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

type AIAgentCapability struct {
	State     string     `json:"state" enum:"untested,supported,unsupported"`
	CheckedAt *time.Time `json:"checkedAt,omitempty"`
}

// Tool support is a model's. Pricing, labels and feature switches do not
// change it. An endpoint, model or credential change does; never reuse its
// previous test. The inputs are those of the single connection before B1, so
// a test made then still applies to the model it became.
func agentConnectionHash(c store.AIBindingRow) string {
	raw, _ := json.Marshal([]any{agentModelOf(c), c.KeySealed, c.ConnectedAt, c.DisconnectedAt, c.Demo})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func agentCapability(ctx context.Context, q *store.Queries, c *connection) (AIAgentCapability, error) {
	out := AIAgentCapability{State: "untested"}
	if !c.connected() {
		return out, nil
	}
	r, err := q.AIAgentCapability(ctx, store.AIAgentCapabilityParams{WorkspaceID: c.WorkspaceID, ModelID: c.ModelID})
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if r.ConnectionHash == agentConnectionHash(c.AIBindingRow) {
		out.State, out.CheckedAt = "unsupported", &r.CheckedAt
		if r.Supported {
			out.State = "supported"
		}
	}
	return out, nil
}

// TestAgentTools has one harmless schema, no domain dispatcher and no user
// content. Managers without ai:use can test, but cannot supply a prompt or
// start a task. The ordinary reservation ledger accounts this fixed probe.
// It tests the agent's model.
func (s *Service) TestAgentTools(ctx context.Context, ws *store.Workspace, p *authz.Principal) (AIAgentCapability, error) {
	c, err := s.loadFor(ctx, s.Q, ws.ID, nil, FeatureAgent)
	if err != nil {
		return AIAgentCapability{}, err
	}
	return s.testTools(ctx, ws, p, c)
}

// testTools runs the tool-calling check on a model: a workspace model's by
// a manager, a personal key's by its owner.
func (s *Service) testTools(ctx context.Context, ws *store.Workspace, p *authz.Principal, c *connection) (AIAgentCapability, error) {
	permission := authz.IntegrationsManage
	if c.personal() {
		permission = authz.AIUse
	}
	p, err := draftActor(ctx, s.Q, ws, p, false, permission)
	if err != nil {
		return AIAgentCapability{}, err
	}
	if c, err = s.ready(ctx, ws, c, FeatureConnectionTest); err != nil {
		return AIAgentCapability{}, err
	}
	t := task{feature: FeatureConnectionTest, action: "tools", req: llm.Request{MaxTokens: 128,
		Messages: []llm.Message{{Role: "user", Parts: []llm.Part{{Text: "Call check_tools exactly once with value ready. Do not answer in plain text."}}}},
		Tools: []llm.Tool{{Name: "check_tools", Description: "A fixed compatibility check. No action is performed.", Parameters: map[string]any{
			"type": "object", "properties": map[string]any{"value": map[string]any{"type": "string", "enum": []string{"ready"}}}, "required": []string{"value"}, "additionalProperties": false,
		}}},
	}, checkSource: func(ctx context.Context, q *store.Queries, fresh *authz.Principal) error {
		return checkTestConnection(ctx, q, ws, fresh, c)
	}}
	res, err := s.send(ctx, ws, p, c, t)
	if err != nil {
		return AIAgentCapability{}, err
	}
	valid, calls := !res.Response.Refused && res.Response.StopReason == llm.StopTools, 0
	for _, part := range res.Response.Message.Parts {
		if call := part.ToolCall; call != nil {
			calls++
			var args struct {
				Value string `json:"value"`
			}
			valid = valid && call.Name == "check_tools" && decodeDraft(call.Arguments, &args) == nil && args.Value == "ready"
		}
	}
	valid = valid && calls == 1
	out := AIAgentCapability{State: "unsupported", CheckedAt: ptr(s.now())}
	if valid {
		out.State = "supported"
	}
	err = db.Tx(ctx, s.Pool, func(tx pgx.Tx) error {
		q := store.New(tx)
		if err := checkTestConnection(ctx, q, ws, p, c); err != nil {
			return err
		}
		if err := q.AIAgentCapabilitySave(ctx, store.AIAgentCapabilitySaveParams{WorkspaceID: ws.ID, ModelID: c.ModelID, ConnectionHash: agentConnectionHash(c.AIBindingRow), Supported: valid, CheckedAt: *out.CheckedAt}); err != nil {
			return err
		}
		return s.Audit.LogWith(audit.WithPrincipal(ctx, p), q, audit.Event{WorkspaceID: &ws.ID, Action: "ai.agent.tools_tested", TargetType: "ai_assistant", TargetID: ws.ID.String(),
			Metadata: personalMeta(c.AccountUserID, map[string]any{"supported": valid, "request_id": res.RequestID})})
	})
	if err != nil {
		return AIAgentCapability{}, err
	}
	s.finish(ctx, ws, p, res.RequestID, OutcomeNothingDrafted, "tool capability test")
	return out, nil
}
