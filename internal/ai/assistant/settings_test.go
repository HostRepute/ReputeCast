package assistant_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/ai/assistant"
	"github.com/hostrepute/reputecast/internal/ai/llm"
	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/db"
	"github.com/hostrepute/reputecast/internal/store"
)

// Only integration managers change the assistant's settings, at the
// revision they read; a thread keeps the limits it started with.
func TestAssistantSettings(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	if d, err := f.aiService().AssistantSettings(ctx, f.env.App.Q, uuid.New()); err != nil || d.Enabled || d.Limits != (ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}) ||
		d.PerPerson != 2 || d.PerWorkspace != 10 || d.Revision != 0 {
		t.Fatalf("a workspace's defaults: %+v %v", d, err)
	}
	path := f.base + "/assistant/settings"
	in := ai.AIAssistantSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 7, Tokens: 50000, CostMicros: 500000}, PerPerson: 3, PerWorkspace: 20, Revision: f.settings.Revision}
	f.mc.Do(http.MethodPut, path, in).MustStatus(t, 403)
	f.vc.Do(http.MethodPut, path, in).MustStatus(t, 403)
	stale := in
	stale.Revision--
	if p := f.oc.Do(http.MethodPut, path, stale).MustStatus(t, 409).Problem(t); p["code"] != "ai_assistant_changed" {
		t.Fatalf("a change from a stale revision: %v", p)
	}
	var s ai.AISettings
	f.oc.Do(http.MethodPut, path, in).MustStatus(t, 200).JSON(t, &s)
	if a := s.Assistant; !a.Enabled || a.Limits != in.Limits || a.PerPerson != 3 || a.PerWorkspace != 20 || a.Revision != in.Revision+1 {
		t.Fatalf("after saving: %+v", a)
	}
	f.settings = s.Assistant
	bad := in
	bad.Revision, bad.PerPerson = f.settings.Revision, 11
	f.oc.Do(http.MethodPut, path, bad).MustStatus(t, 422)
	if n := f.count(`SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'ai_assistant.settings_updated' AND after->>'perPerson' = '3'
		AND (before->>'perPerson') = '2'`, f.ws.ID); n != 1 {
		t.Fatalf("%d audit entries of the change", n)
	}
	// A thread starts with the limits of the moment and keeps them.
	th := f.thread(f.mc, map[string]any{})
	f.setAssistant(ai.AIAssistantSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 9, Tokens: 60000, CostMicros: 600000}, PerPerson: 3, PerWorkspace: 20})
	var after assistant.AIThread
	f.mc.Do(http.MethodGet, f.base+"/assistant/threads/"+th.ID.String(), nil).MustStatus(t, 200).JSON(t, &after)
	if th.Limits != in.Limits || after.Limits != in.Limits {
		t.Fatalf("the thread's limits: %+v, then %+v", th.Limits, after.Limits)
	}
}

// A thread's requests count against its own cumulative ceilings: the
// first reservation is cut to what the thread has left, the next is
// refused, another thread is unaffected, and settling gives back what the
// provider didn't use. The activity log names the thread, never its title.
func TestAssistantThreadLedger(t *testing.T) {
	f := newFixture(t)
	f.setAssistant(ai.AIAssistantSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 11000, CostMicros: 2000000}, PerPerson: 2, PerWorkspace: 10})
	one, two := f.thread(f.mc, map[string]any{}), f.thread(f.mc, map[string]any{})
	s := f.aiService()
	ctx := context.Background()
	reserve := func(th assistant.AIThread) (uuid.UUID, error) {
		id := uuid.Must(uuid.NewV7())
		// As a claim reserves: under the configuration lock, with the
		// person and the model read under it.
		return id, db.Tx(ctx, f.env.App.Pool, func(tx pgx.Tx) error {
			q := store.New(tx)
			if err := s.LockAssistantConfig(ctx, q, f.ws.ID); err != nil {
				return err
			}
			fresh, err := brandkit.VoiceActor(ctx, q, &f.ws, person(f.ws, f.member))
			if err != nil {
				return err
			}
			m, err := s.ResolveAssistantModel(ctx, q, f.ws.ID, f.member, &th.Model.ID)
			if err != nil {
				return err
			}
			req := llm.Request{MaxTokens: 8000, Messages: []llm.Message{{Role: "user", Parts: []llm.Part{{Text: "Why did opens fall?"}}}}}
			return s.ReserveAssistant(ctx, q, &f.ws, fresh, m, &req,
				ai.AssistantLedger{ThreadID: th.ID, TokenLimit: th.Limits.Tokens, CostLimitMicros: th.Limits.CostMicros}, id)
		})
	}
	codeOf := func(err error) string {
		var p *api.Problem
		if errors.As(err, &p) {
			return p.Code
		}
		return ""
	}
	first, err := reserve(one)
	if err != nil {
		t.Fatal(err)
	}
	var input, output int64
	var thread *uuid.UUID
	if err := f.env.App.Pool.QueryRow(ctx, `SELECT input_tokens, output_tokens, thread_id FROM ai_budget_reservations WHERE id = $1`, first).Scan(&input, &output, &thread); err != nil {
		t.Fatal(err)
	}
	if thread == nil || *thread != one.ID || output != 11000-input || output >= 8000 {
		t.Fatalf("the first reservation: %d in, %d out, thread %v", input, output, thread)
	}
	if _, err := reserve(one); codeOf(err) != "ai_thread_cap" {
		t.Fatalf("a request past the thread's ceiling: %v", err)
	}
	if _, err := reserve(two); err != nil {
		t.Fatalf("another thread's request: %v", err)
	}
	var view assistant.AIThread
	f.mc.Do(http.MethodGet, f.base+"/assistant/threads/"+one.ID.String(), nil).MustStatus(t, 200).JSON(t, &view)
	if view.Usage.Tokens != input+output || view.Usage.Requests != 1 {
		t.Fatalf("the thread's usage: %+v", view.Usage)
	}
	// Settled at what the provider reported: the thread has room again.
	if err := db.Tx(ctx, f.env.App.Pool, func(tx pgx.Tx) error {
		known, err := s.AccountAssistant(ctx, store.New(tx), ai.AssistantRequest{ID: first, WorkspaceID: f.ws.ID, UserID: f.member, ThreadID: one.ID, Action: "answer",
			Outcome: ai.OutcomeAnswered, CreatedAt: time.Now()}, llm.Response{InputTokens: 100, OutputTokens: 20, UsageKnown: true}, nil)
		if err == nil && !known {
			err = errors.New("usage unknown")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if n := f.count(`SELECT count(*) FROM ai_requests WHERE id = $1 AND feature = 'assistant' AND target_kind = 'assistant_thread' AND target_id = $2
		AND target_label = '' AND outcome = 'answered' AND input_tokens = 100`, first, one.ID); n != 1 {
		t.Fatal("the activity log's row of the settled request")
	}
	if _, err := reserve(one); err != nil {
		t.Fatalf("a request after settling: %v", err)
	}
}
