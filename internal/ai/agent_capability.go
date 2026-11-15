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

// Pricing, labels and feature switches do not change tool support. An
// endpoint, model or credential change does; never reuse its previous test.
func agentConnectionHash(c store.AiConnection) string {
	raw, _ := json.Marshal([]any{agentModelOf(c), c.KeySealed, c.ConnectedAt, c.DisconnectedAt, c.Demo})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func agentCapability(ctx context.Context, q *store.Queries, c *connection) (AIAgentCapability, error) {
	out := AIAgentCapability{State: "untested"}
	if !c.connected() {
		return out, nil
	}
	r, err := q.AIAgentCapability(ctx, c.WorkspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	if r.ConnectionHash == agentConnectionHash(c.AiConnection) {
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
func (s *Service) TestAgentTools(ctx context.Context, ws *store.Workspace, p *authz.Principal) (AIAgentCapability, error) {
	p, err := draftActor(ctx, s.Q, ws, p, false, authz.IntegrationsManage)
	if err != nil {
		return AIAgentCapability{}, err
	}
	c, err := s.usable(ctx, ws, FeatureConnectionTest)
	if err != nil {
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
		if err := q.AIAgentCapabilitySave(ctx, store.AIAgentCapabilitySaveParams{WorkspaceID: ws.ID, ConnectionHash: agentConnectionHash(c.AiConnection), Supported: valid, CheckedAt: *out.CheckedAt}); err != nil {
			return err
		}
		return s.Audit.LogWith(audit.WithPrincipal(ctx, p), q, audit.Event{WorkspaceID: &ws.ID, Action: "ai.agent.tools_tested", TargetType: "ai_assistant", TargetID: ws.ID.String(), Metadata: map[string]any{"supported": valid, "request_id": res.RequestID}})
	})
	if err != nil {
		return AIAgentCapability{}, err
	}
	s.finish(ctx, ws, p, res.RequestID, OutcomeNothingDrafted, "tool capability test")
	return out, nil
}
