package ai_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/campaigns"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/segments"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/testdb"
	"github.com/hostrepute/reputecast/internal/website"
)

// The real API, migrations, Postgres checkpoints and HTTP provider adapter
// run here; only the external model is a deterministic local server.
func TestAgentCheckpoints(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@agent.test", Name: "Agent Owner"})
	other := env.CreateUser(apptest.UserOpts{Email: "editor@agent.test"})
	ws := env.CreateWorkspace("Agent", owner.ID)
	env.AddMember(ws, other.ID, "editor")
	a, b := env.Client(), env.Client()
	a.SignInDemo(owner.Email)
	b.SignInDemo(other.Email)
	base := "/workspaces/" + ws.Slug + "/ai"
	plan := ai.AIAgentPlan{Summary: "Announce the improved export workflow.", Questions: []string{}, Items: []ai.AIAgentPlanItem{{Handle: "announcement", Kind: "campaign", Name: "Export improvements", Purpose: "Explain faster exports.", Bundle: "launch", DependsOn: []string{}}}}
	var phase atomic.Int32
	var requests atomic.Int32
	entered, release := make(chan struct{}, 1), make(chan struct{})
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		message := map[string]any{"role": "assistant", "content": "Ready"}
		stop := "stop"
		tools, _ := req["tools"].([]any)
		if len(tools) > 0 {
			requests.Add(1)
			if phase.Load() == 4 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":{"message":"key rejected"}}`))
				return
			}
			if phase.Load() == 3 {
				entered <- struct{}{}
				<-release
			}
			name := "propose_plan"
			var args any = plan
			if phase.Load() == 1 {
				name = "stage_campaign"
				args = map[string]any{"handle": "announcement", "email": ai.AIDraftEmail{Subject: "Exports, simplified", Preheader: "A shorter path to your data", MJML: `<mjml><mj-body><mj-section><mj-column><mj-text>Exports now take fewer steps.</mj-text></mj-column></mj-section></mj-body></mjml>`}}
			}
			if phase.Load() != 2 {
				raw, _ := json.Marshal(args)
				message["content"] = nil
				message["tool_calls"] = []any{map[string]any{"id": "call1", "type": "function", "function": map[string]any{"name": name, "arguments": string(raw)}}}
				stop = "tool_calls"
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}}, "usage": map[string]int{"prompt_tokens": 137, "completion_tokens": 43}})
	}))
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
		model.Close()
	}()
	a.Do("PUT", base+"/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "agent-fixture", "capKind": "none", "inputPrice": 3, "outputPrice": 7}).MustStatus(t, 200)
	settings := ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}}
	a.Do("PUT", base+"/agent/settings", settings).MustStatus(t, 204)
	s := &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets,
		AllowPrivate: func() bool { return true }, Jobs: func() *jobs.Client { return env.App.Jobs },
		BrandKit: &brandkit.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}}
	ctx := context.Background()
	var run ai.AIAgentRunView
	var queuedBefore int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE kind='ai.agent'`).Scan(&queuedBefore); err != nil {
		t.Fatal(err)
	}
	a.Do("POST", base+"/agent/runs", map[string]any{"brief": "Announce easier exports.", "language": "en"}).MustStatus(t, 422)
	a.Do("POST", base+"/agent/runs", ai.AIAgentBrief{Brief: "Announce easier exports.", Language: "en"}).MustStatus(t, 422)
	var runs, queuedAfter, reserved int
	if err := env.App.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM ai_agent_runs WHERE workspace_id=$1), (SELECT count(*) FROM river_job WHERE kind='ai.agent'), (SELECT count(*) FROM ai_budget_reservations WHERE workspace_id=$1 AND feature='agent')`, ws.ID).Scan(&runs, &queuedAfter, &reserved); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 0 || runs != 0 || reserved != 0 || queuedBefore != queuedAfter {
		t.Fatal("unreviewed brief created work or reached the model")
	}
	brief := ai.AIAgentBrief{SharingReviewed: true, Brief: "Announce easier exports.", Language: "en", RequestKey: "brief-retry-1"}
	a.Do("POST", base+"/agent/runs", brief).MustStatus(t, 200).JSON(t, &run)
	// Retrying after a lost response returns the same run without a second
	// plan; the same key with another brief is refused.
	var retried ai.AIAgentRunView
	a.Do("POST", base+"/agent/runs", brief).MustStatus(t, 200).JSON(t, &retried)
	changed := brief
	changed.Brief = "Announce faster imports."
	if p := a.Do("POST", base+"/agent/runs", changed).MustStatus(t, 409).Problem(t); p["code"] != "ai_agent_request_reused" {
		t.Fatalf("reused request key: %v", p)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM ai_agent_runs WHERE workspace_id=$1), (SELECT count(*) FROM river_job WHERE kind='ai.agent')`, ws.ID).Scan(&runs, &queuedAfter); err != nil {
		t.Fatal(err)
	}
	if retried.ID != run.ID || runs != 1 || queuedAfter != queuedBefore+1 || requests.Load() != 0 {
		t.Fatalf("a retried brief created another run: %s %s runs=%d jobs=%d", run.ID, retried.ID, runs, queuedAfter-queuedBefore)
	}
	path := base + "/agent/runs/" + run.ID.String()
	b.Do("GET", path, nil).MustStatus(t, 404)
	var list ai.AIAgentRunList
	b.Do("GET", base+"/agent/runs", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 0 {
		t.Fatal("another member's run leaked through the list")
	}
	a.Do("GET", base+"/agent/runs?cursor=invalid", nil).MustStatus(t, 409)
	a.Do("GET", base+"/agent/runs", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].ID != run.ID || list.Items[0].OwnerName != owner.Name || list.Items[0].Title != "Announce easier exports." || list.Items[0].Model != "agent-fixture" {
		t.Fatalf("run list lost metadata: %+v", list)
	}
	execute := func(version int64) {
		t.Helper()
		if err := s.ExecuteAgent(ctx, ai.AgentArgs{WorkspaceID: ws.ID, RunID: run.ID, Version: version}); err != nil {
			t.Fatal(err)
		}
	}
	execute(run.Version)
	a.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	if run.State != "ready" || run.PlanRevision != 1 || len(run.Plan.Items) != 1 || run.Plan.Items[0].Name != plan.Items[0].Name || run.ToolCalls != 1 {
		t.Fatalf("persisted plan: %+v", run)
	}
	if len(run.Steps) != 1 || run.Steps[0].State != "done" || len(run.Steps[0].Operations) != 1 || run.Steps[0].Operations[0].Tool != "propose_plan" || run.Steps[0].Operations[0].Failed {
		t.Fatalf("plan step summary: %+v", run.Steps)
	}
	for _, action := range []string{"answer", "revise", "replan", "propose", "refresh"} {
		for _, reviewed := range []any{nil, false} {
			input := map[string]any{"action": action, "version": run.Version, "message": "Shorten the export announcement."}
			if reviewed != nil {
				input["sharingReviewed"] = reviewed
			}
			a.Do("POST", path+"/actions", input).MustStatus(t, 422)
		}
	}
	var unchanged ai.AIAgentRunView
	a.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &unchanged)
	if unchanged.Version != run.Version || unchanged.State != run.State || unchanged.Tokens != run.Tokens || len(unchanged.Conversation) != len(run.Conversation) || requests.Load() != 1 {
		t.Fatal("unreviewed action changed the run or reached the model")
	}
	// Output-only pricing gives independent exact expectations: a 2k email
	// and 200-token recap, versus four full 8k requests including repairs.
	// Quotes neither spend money nor substitute zero for unknown pricing.
	for _, price := range []struct {
		output float64
		known  bool
		model  string
		quote  bool
	}{{1, true, "agent-fixture", true}, {0, true, "agent-fixture", true}, {0, false, "agent-fixture", false}, {1, true, "different-model", false}} {
		if _, err := env.App.Pool.Exec(ctx, `UPDATE ai_models SET input_price=0,output_price=$2,price_known=$3,model=$4 WHERE workspace_id=$1`, ws.ID, price.output, price.known, price.model); err != nil {
			t.Fatal(err)
		}
		var quoted ai.AIAgentRunView
		a.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &quoted)
		if (quoted.Estimate != nil) != price.quote {
			t.Fatalf("quote availability: %+v", quoted.Estimate)
		}
		if price.quote && (quoted.Estimate.LowMicros != int64(2200*price.output) || quoted.Estimate.HighMicros != int64(32000*price.output) || quoted.Estimate.PricingAt.IsZero()) {
			t.Fatalf("remaining build estimate: %+v", quoted.Estimate)
		}
		if quoted.Tokens != run.Tokens || quoted.CostMicros != run.CostMicros || requests.Load() != 1 {
			t.Fatal("reading an estimate spent budget or called the model")
		}
	}
	// A prior conversation is resent, not free. Adding exactly 4,000 ASCII
	// bytes adds 1,000 estimated input tokens to both low-scenario requests,
	// and 4,000 to each of the four conservative-scenario requests.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE ai_models SET input_price=1,output_price=0,price_known=true,model='agent-fixture' WHERE workspace_id=$1`, ws.ID); err != nil {
		t.Fatal(err)
	}
	var beforeQuote, afterQuote ai.AIAgentRunView
	a.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &beforeQuote)
	var messages json.RawMessage
	if err := env.App.Pool.QueryRow(ctx, `SELECT messages FROM ai_agent_contexts WHERE run_id=$1`, run.ID).Scan(&messages); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE ai_agent_contexts SET messages=jsonb_set(messages,'{0,Parts,0,Text}',to_jsonb((messages #>> '{0,Parts,0,Text}') || repeat('x',4000))) WHERE run_id=$1`, run.ID); err != nil {
		t.Fatal(err)
	}
	a.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &afterQuote)
	if beforeQuote.Estimate == nil || afterQuote.Estimate == nil || afterQuote.Estimate.LowMicros-beforeQuote.Estimate.LowMicros != 2000 || afterQuote.Estimate.HighMicros-beforeQuote.Estimate.HighMicros != 16000 {
		t.Fatalf("resent context not priced: before=%+v after=%+v", beforeQuote.Estimate, afterQuote.Estimate)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE ai_agent_contexts SET messages=$2 WHERE run_id=$1`, run.ID, messages); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE ai_models SET input_price=3,output_price=7,price_known=true,model='agent-fixture' WHERE workspace_id=$1`, ws.ID); err != nil {
		t.Fatal(err)
	}
	countCampaigns := func() int {
		t.Helper()
		var n int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM campaigns WHERE workspace_id = $1`, ws.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if countCampaigns() != 0 {
		t.Fatal("planning wrote a campaign")
	}
	// Confirm the exact version. A stale plan cannot be built.
	a.Do("POST", path+"/actions", ai.AIAgentAction{Action: "build", Version: run.Version, PlanRevision: 0}).MustStatus(t, 409)
	a.Do("POST", path+"/actions", ai.AIAgentAction{Action: "build", Version: run.Version, PlanRevision: run.PlanRevision}).MustStatus(t, 204)
	a.Do("POST", path+"/actions", ai.AIAgentAction{Action: "build", Version: run.Version, PlanRevision: run.PlanRevision}).MustStatus(t, 204)
	a.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	phase.Store(1)
	version := run.Version
	execute(version)
	execute(version) // A job replay after commit must not spend or write again.
	a.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	if len(run.Artifacts) != 1 || run.Artifacts[0].Revision != 1 || run.ToolCalls != 2 || requests.Load() != 2 || countCampaigns() != 0 {
		t.Fatalf("staging replay: calls=%d %+v", requests.Load(), run)
	}
	if run.Artifacts[0].Email.Subject != "Exports, simplified" || !strings.Contains(run.Artifacts[0].Email.MJML, "Exports now take fewer steps.") || !strings.Contains(run.Artifacts[0].HTML, "Exports now take fewer steps.") {
		t.Fatal("staged content differs")
	}
	if len(run.Steps) != 2 || run.Steps[1].Operations[0].Handle != "announcement" || run.Steps[1].Operations[0].Code != "ok" {
		t.Fatalf("draft step summary: %+v", run.Steps)
	}
	// Public polling never serializes model text, arguments or provider
	// signatures. Those are checkpoint data, not the visible action log.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE ai_agent_contexts SET messages=messages || '[{"role":"assistant","parts":[{"text":"PRIVATE_CHECKPOINT_SENTINEL"}]}]'::jsonb WHERE run_id=$1`, run.ID); err != nil {
		t.Fatal(err)
	}
	public := a.Do("GET", path, nil).MustStatus(t, 200)
	var snapshot map[string]any
	public.JSON(t, &snapshot)
	if strings.Contains(mustJSON(t, snapshot), "PRIVATE_CHECKPOINT_SENTINEL") {
		t.Fatal("model checkpoint leaked into the public run")
	}
	phase.Store(2)
	execute(run.Version)
	execute(run.Version) // Retried acknowledgment must not duplicate a notice.
	a.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	if run.State != "review" || run.Tokens != 540 || run.CostMicros != 2136 {
		t.Fatalf("review accounting: %+v", run)
	}
	var usage ai.AISettings
	a.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &usage)
	var agentUsage ai.AIUsageFeature
	for _, feature := range usage.ByFeature {
		if feature.Feature == "agent" {
			agentUsage = feature
		}
	}
	if agentUsage.Tokens != 540 || agentUsage.CostMicros != 2136 || agentUsage.Requests != 3 {
		t.Fatalf("agent missing from feature usage: %+v", agentUsage)
	}
	var notices int
	var onlyOwner, contentFree bool
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*),bool_and(user_id=$2),bool_and(title='Your AI task needs review' AND body NOT LIKE '%export%' AND link=$3) FROM notifications WHERE data->>'run_id'=$1`, run.ID.String(), owner.ID, "/w/"+ws.Slug+"/agent/"+run.ID.String()).Scan(&notices, &onlyOwner, &contentFree); err != nil {
		t.Fatal(err)
	}
	if notices != 2 || !onlyOwner || !contentFree {
		t.Fatalf("task notifications repeated, leaked content or reached other members: %d %v %v", notices, onlyOwner, contentFree)
	}
	decision := ai.AIAgentDecision{Version: run.Version, Decision: "kept", Artifacts: []ai.AIAgentArtifactRevision{{ID: run.Artifacts[0].ID, Revision: 1}}}
	a.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
	a.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
	a.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	if run.State != "completed" || run.Artifacts[0].ResultID == nil || countCampaigns() != 1 {
		t.Fatalf("keep was not atomic/idempotent: %+v", run)
	}
	keptRun := run.ID
	kept := *run.Artifacts[0].ResultID
	var status, body, rendered, via, actor string
	var provenance int
	if err := env.App.Pool.QueryRow(ctx, `SELECT status,mjml,html FROM campaigns WHERE id=$1`, kept).Scan(&status, &body, &rendered); err != nil {
		t.Fatal(err)
	}
	if status != "draft" || !strings.Contains(rendered, "Exports now take fewer steps.") {
		t.Fatal("keep changed publication state or lost compiled HTML")
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*),min(actor_type),min(metadata->>'via') FROM audit_events WHERE target_id=$1 AND action='campaign.create' AND metadata->>'run_id'=$2`, kept.String(), run.ID.String()).Scan(&provenance, &actor, &via); err != nil {
		t.Fatal(err)
	}
	if provenance != 1 || actor != "user" || via != "ai_agent" {
		t.Fatalf("domain audit: %d %s %s", provenance, actor, via)
	}
	a.Do("PUT", "/workspaces/"+ws.Slug+"/campaigns/"+kept.String()+"/content", map[string]any{"baseVersion": 1, "subject": "A human edited this", "preheader": "Reviewed locally", "mjml": body}).MustStatus(t, 200)
	a.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
	decision.Decision, decision.Version = "discarded", run.Version
	a.Do("POST", path+"/decisions", decision).MustStatus(t, 409)
	var subject string
	if err := env.App.Pool.QueryRow(ctx, `SELECT subject FROM campaigns WHERE id=$1`, kept).Scan(&subject); err != nil || subject != "A human edited this" {
		t.Fatalf("repeated decision overwrote human edit: %q, %v", subject, err)
	}
	// Integration managers can inspect another author's run, but its
	// kept resources remain subject to their current audience scope.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE workspace_members SET role_id=(SELECT id FROM roles WHERE workspace_id=$1 AND key='admin') WHERE workspace_id=$1 AND user_id=$2`, ws.ID, other.ID); err != nil {
		t.Fatal(err)
	}
	b.Do("GET", path, nil).MustStatus(t, 200)
	b.Do("GET", base+"/agent/runs", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 1 || list.Items[0].ID != keptRun {
		t.Fatalf("manager cannot inspect permitted run: %+v", list)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET audience=jsonb_build_object('lists',jsonb_build_array($1::text)) WHERE id=$2`, uuid.NewString(), kept); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE workspace_members SET scope=jsonb_build_object('lists',jsonb_build_array($1::text)) WHERE workspace_id=$2 AND user_id=$3`, uuid.NewString(), ws.ID, other.ID); err != nil {
		t.Fatal(err)
	}
	b.Do("GET", path, nil).MustStatus(t, 404)
	b.Do("GET", base+"/agent/runs", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 0 {
		t.Fatal("integration management bypassed run resource scope")
	}
	// Cancellation during HTTP keeps billed usage but fences the response.
	phase.Store(3)
	a.Do("POST", base+"/agent/runs", ai.AIAgentBrief{SharingReviewed: true, Brief: "Draft an export update.", Language: "en"}).MustStatus(t, 200).JSON(t, &run)
	path = base + "/agent/runs/" + run.ID.String()
	finished := make(chan error, 1)
	go func() {
		finished <- s.ExecuteAgent(ctx, ai.AgentArgs{WorkspaceID: ws.ID, RunID: run.ID, Version: run.Version})
	}()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("request did not reach model")
	}
	a.Do("POST", path+"/actions", ai.AIAgentAction{Action: "cancel", Version: run.Version}).MustStatus(t, 204)
	a.Do("POST", path+"/actions", ai.AIAgentAction{Action: "cancel", Version: run.Version}).MustStatus(t, 204)
	close(release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	a.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	if run.State != "cancelled" || run.PlanRevision != 0 || run.Tokens != 180 || len(run.Artifacts) != 0 {
		t.Fatalf("late cancellation: %+v", run)
	}
	// Closure purges messages after seven days, while the brief, review
	// artifacts and ordinary kept resources survive until their own expiry.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE ai_agent_runs SET closed_at=now()-interval '8 days' WHERE id=$1`, keptRun); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Purge(ctx); err != nil {
		t.Fatal(err)
	}
	var messagesEmpty, contentPresent, stepsEmpty bool
	if err := env.App.Pool.QueryRow(ctx, `SELECT messages='[]'::jsonb AND NOT (bindings ? 'Conversation'), brief<>'{}'::jsonb FROM ai_agent_contexts WHERE run_id=$1`, keptRun).Scan(&messagesEmpty, &contentPresent); err != nil {
		t.Fatal(err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT bool_and(request IS NULL AND response IS NULL AND results='[]'::jsonb) FROM ai_agent_steps WHERE run_id=$1`, keptRun).Scan(&stepsEmpty); err != nil {
		t.Fatal(err)
	}
	if !messagesEmpty || !contentPresent || !stepsEmpty {
		t.Fatal("closure retention boundary failed")
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE ai_agent_runs SET expires_at=now()-interval '1 second' WHERE workspace_id=$1`, ws.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Purge(ctx); err != nil {
		t.Fatal(err)
	}
	run = ai.AIAgentRunView{}
	a.Do("GET", base+"/agent/runs/"+keptRun.String(), nil).MustStatus(t, 200).JSON(t, &run)
	if run.State != "expired" || len(run.Artifacts) != 0 || run.Brief != nil || run.Plan != nil || run.Model != "agent-fixture" || run.Tokens != 540 || countCampaigns() != 1 {
		t.Fatalf("expiry deleted kept resources or exposed content: %+v", run)
	}
	b.Do("GET", base+"/agent/runs/"+keptRun.String(), nil).MustStatus(t, 404)
	var temporary int
	if err := env.App.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM ai_agent_contexts WHERE run_id=$1)+(SELECT count(*) FROM ai_agent_artifacts WHERE run_id=$1 AND (content IS NOT NULL OR context IS NOT NULL))`, keptRun).Scan(&temporary); err != nil || temporary != 0 {
		t.Fatalf("expired context survived: %d %v", temporary, err)
	}
	// Agent accounting must retain the settings alerts of ordinary AI
	// requests, including a rejected key that produces no usable content.
	phase.Store(4)
	a.Do("PUT", base+"/cap", map[string]any{"kind": "usd", "usd": 1}).MustStatus(t, 200)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE ai_requests SET cost_micros=810000 WHERE id=(SELECT id FROM ai_requests WHERE workspace_id=$1 ORDER BY created_at LIMIT 1)`, ws.ID); err != nil {
		t.Fatal(err)
	}
	a.Do("POST", base+"/agent/runs", ai.AIAgentBrief{SharingReviewed: true, Brief: "Draft a release announcement.", Language: "en"}).MustStatus(t, 200).JSON(t, &run)
	execute(run.Version)
	var connection ai.AISettings
	a.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &connection)
	if connection.Connection == nil || connection.Connection.Status != "key_rejected" {
		t.Fatalf("Agent did not report its rejected key: %+v", connection.Connection)
	}
	var keyNotices, budgetNotices int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE kind='ai_key_rejected'),count(*) FILTER (WHERE kind='ai_usage') FROM notifications WHERE workspace_id=$1`, ws.ID).Scan(&keyNotices, &budgetNotices); err != nil || keyNotices != 2 || budgetNotices != 2 {
		t.Fatalf("Agent manager alerts: key=%d budget=%d err=%v", keyNotices, budgetNotices, err)
	}
}

func TestAgentResumption(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@resume.test"})
	editor := env.CreateUser(apptest.UserOpts{Email: "editor@resume.test"})
	ws := env.CreateWorkspace("Resume", owner.ID)
	env.AddMember(ws, editor.ID, "editor")
	manager, author := env.Client(), env.Client()
	manager.SignInDemo(owner.Email)
	author.SignInDemo(editor.Email)
	base := "/workspaces/" + ws.Slug + "/ai"
	plan := ai.AIAgentPlan{Summary: "Two export announcements.", Questions: []string{}, Items: []ai.AIAgentPlanItem{
		{Handle: "intro", Kind: "campaign", Name: "An introduction", Purpose: "Explain exports.", Bundle: "exports", DependsOn: []string{}},
		{Handle: "details", Kind: "campaign", Name: "More detail", Purpose: "Explain filters.", Bundle: "exports", DependsOn: []string{"intro"}},
	}}
	call := func(id, name string, args any) map[string]any {
		return map[string]any{"id": id, "type": "function", "function": map[string]any{"name": name, "arguments": mustJSON(t, args)}}
	}
	planCalls := []any{call("p1", "propose_plan", plan)}
	buildCalls := []any{}
	for _, item := range plan.Items {
		buildCalls = append(buildCalls, call(item.Handle, "stage_campaign", map[string]any{"handle": item.Handle,
			"email": ai.AIDraftEmail{Subject: item.Name, Preheader: item.Purpose, MJML: `<mjml><mj-body><mj-section><mj-column><mj-text>` + item.Purpose + `</mj-text></mj-column></mj-section></mj-body></mjml>`}}))
	}
	var mu sync.Mutex
	tools := planCalls
	unknown := false
	var hook func()
	var requests atomic.Int32
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		mu.Lock()
		calls, missing, before := tools, unknown, hook
		mu.Unlock()
		message, stop := map[string]any{"role": "assistant", "content": "Drafts are ready."}, "stop"
		if req["tools"] != nil {
			requests.Add(1)
			if before != nil {
				before()
			}
			if len(calls) > 0 {
				message["content"], message["tool_calls"], stop = nil, calls, "tool_calls"
			}
		}
		answer := map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}}}
		if !missing {
			answer["usage"] = map[string]int{"prompt_tokens": 123, "completion_tokens": 47}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(answer)
	}))
	defer model.Close()
	manager.Do("PUT", base+"/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "resume-fixture", "capKind": "none", "inputPrice": 3, "outputPrice": 7}).MustStatus(t, 200)
	settings := ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 2, Tokens: 300000, CostMicros: 2000000}}
	manager.Do("PUT", base+"/agent/settings", settings).MustStatus(t, 204)
	s := &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets,
		AllowPrivate: func() bool { return true }, Jobs: func() *jobs.Client { return env.App.Jobs },
		BrandKit: &brandkit.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}}
	ctx := context.Background()
	var run ai.AIAgentRunView
	path := ""
	start := func() {
		t.Helper()
		run = ai.AIAgentRunView{}
		author.Do("POST", base+"/agent/runs", ai.AIAgentBrief{SharingReviewed: true, Brief: "Explain the new exports.", Language: "en"}).MustStatus(t, 200).JSON(t, &run)
		path = base + "/agent/runs/" + run.ID.String()
	}
	execute := func() {
		t.Helper()
		if err := s.ExecuteAgent(ctx, ai.AgentArgs{WorkspaceID: ws.ID, RunID: run.ID, Version: run.Version}); err != nil {
			t.Fatal(err)
		}
		author.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	}
	build := func() {
		t.Helper()
		author.Do("POST", path+"/actions", ai.AIAgentAction{Action: "build", Version: run.Version, PlanRevision: run.PlanRevision}).MustStatus(t, 204)
		author.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	}
	start()
	execute()
	build()
	mu.Lock()
	tools = buildCalls
	mu.Unlock()
	execute()
	if run.State != "paused" || run.PauseReason != "run_cap" || run.ToolCalls != 2 || len(run.Artifacts) != 1 || requests.Load() != 2 {
		t.Fatalf("partial response did not pause: %+v requests=%d", run, requests.Load())
	}
	decision := ai.AIAgentDecision{Version: run.Version, Decision: "kept", Artifacts: []ai.AIAgentArtifactRevision{{ID: run.Artifacts[0].ID, Revision: 1}}}
	author.Do("POST", path+"/decisions", decision).MustStatus(t, 409) // Missing the dependent draft.
	author.Do("POST", path+"/actions", ai.AIAgentAction{Action: "continue", Version: run.Version}).MustStatus(t, 409)
	raised := ai.AIAgentLimitChange{Version: run.Version, Limits: ai.AIAgentLimits{ToolCalls: 4, Tokens: 300000, CostMicros: 2000000}}
	author.Do("PUT", path+"/limits", raised).MustStatus(t, 403)
	manager.Do("PUT", path+"/limits", raised).MustStatus(t, 204)
	author.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	manager.Do("POST", path+"/actions", ai.AIAgentAction{Action: "continue", Version: run.Version}).MustStatus(t, 404)
	author.Do("POST", path+"/actions", ai.AIAgentAction{Action: "continue", Version: run.Version}).MustStatus(t, 204)
	author.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	execute()
	if run.ToolCalls != 3 || len(run.Artifacts) != 2 || run.Artifacts[0].Revision != 1 || run.Artifacts[1].Revision != 1 || requests.Load() != 2 || run.Tokens != 340 || run.CostMicros != 1396 {
		t.Fatalf("continuation resent or replayed part of the response: %+v requests=%d", run, requests.Load())
	}
	mu.Lock()
	tools = nil
	mu.Unlock()
	execute()
	decision.Version, decision.Artifacts = run.Version, []ai.AIAgentArtifactRevision{}
	for _, a := range run.Artifacts {
		decision.Artifacts = append(decision.Artifacts, ai.AIAgentArtifactRevision{ID: a.ID, Revision: a.Revision})
	}
	author.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
	author.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
	var n int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM campaigns WHERE workspace_id=$1`, ws.ID).Scan(&n); err != nil || n != 2 {
		t.Fatalf("bundle keep: %d %v", n, err)
	}
	// Discarding the staged half of a paused bundle must also prevent its
	// not-yet-executed dependent call from resurrecting it on continuation.
	mu.Lock()
	tools = planCalls
	mu.Unlock()
	start()
	execute()
	build()
	mu.Lock()
	tools = buildCalls
	mu.Unlock()
	execute()
	decision = ai.AIAgentDecision{Version: run.Version, Decision: "discarded", Artifacts: []ai.AIAgentArtifactRevision{{ID: run.Artifacts[0].ID, Revision: 1}}}
	discardedName := run.Artifacts[0].Name
	author.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
	author.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
	run = ai.AIAgentRunView{}
	author.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	if len(run.Artifacts) != 1 || run.Artifacts[0].Email != nil || run.Artifacts[0].Decision != "discarded" || run.Artifacts[0].Name != discardedName {
		t.Fatalf("discarded content survived: %+v", run)
	}
	raised.Version = run.Version
	manager.Do("PUT", path+"/limits", raised).MustStatus(t, 204)
	author.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	author.Do("POST", path+"/actions", ai.AIAgentAction{Action: "continue", Version: run.Version}).MustStatus(t, 204)
	author.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	beforeDiscardResume := requests.Load()
	execute()
	if requests.Load() != beforeDiscardResume || len(run.Artifacts) != 1 || run.Artifacts[0].Email != nil {
		t.Fatalf("continuation recreated discarded bundle: %+v", run)
	}
	mu.Lock()
	tools = nil
	mu.Unlock()
	execute()
	if run.State != "completed" || len(run.Artifacts) != 1 {
		t.Fatalf("discarded plan never finished: %+v", run)
	}

	// An intact response without usage waits for reconciliation and then
	// resumes locally. It must not purchase an identical planning request.
	mu.Lock()
	tools, unknown = planCalls, true
	mu.Unlock()
	start()
	execute()
	if run.State != "paused" || !run.Uncertain || run.PlanRevision != 0 {
		t.Fatalf("unknown usage was treated as free: %+v", run)
	}
	author.Do("POST", path+"/actions", ai.AIAgentAction{Action: "continue", Version: run.Version}).MustStatus(t, 409)
	var requestID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `UPDATE ai_budget_reservations SET created_at=now()-interval '11 minutes' WHERE run_id=$1 RETURNING id`, run.ID).Scan(&requestID); err != nil {
		t.Fatal(err)
	}
	manager.Do("POST", base+"/budget-holds/"+requestID.String()+"/reconcile", ai.AIBudgetReconcileInput{Confirmed: true, InputTokens: 123, OutputTokens: 47}).MustStatus(t, 204)
	author.Do("POST", path+"/actions", ai.AIAgentAction{Action: "continue", Version: run.Version}).MustStatus(t, 204)
	author.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	before := requests.Load()
	execute()
	if run.State != "ready" || run.Tokens != 170 || run.CostMicros != 698 || requests.Load() != before {
		t.Fatalf("reconciled response was repeated: %+v", run)
	}

	// Even a model that invents three different privileged tools cannot
	// escape the denial/repair ceiling or mutate resources.
	settings.Revision, settings.Limits.ToolCalls = 1, 40
	manager.Do("PUT", base+"/agent/settings", settings).MustStatus(t, 204)
	mu.Lock()
	unknown = false
	tools = []any{call("bad1", "send_campaign", map[string]any{}), call("bad2", "export_contacts", map[string]any{}), call("bad3", "set_settings", map[string]any{})}
	mu.Unlock()
	start()
	execute()
	if run.State != "paused" || run.PauseReason != "validation_exhausted" || run.ToolCalls != 3 || len(run.Artifacts) != 0 {
		t.Fatalf("unclassified tools bypassed the registry: %+v", run)
	}
	author.Do("POST", path+"/actions", ai.AIAgentAction{Action: "continue", Version: run.Version}).MustStatus(t, 409)
	for _, op := range run.Steps[0].Operations {
		if op.Tool != "denied" || !op.Failed || op.Code != "tool_not_allowed" {
			t.Fatalf("unsafe operation summary: %+v", op)
		}
	}
	// Three invalid attempts exhaust just that output. A fourth valid
	// attempt cannot bypass the ceiling, but another output is still saved.
	mu.Lock()
	tools = planCalls
	mu.Unlock()
	start()
	execute()
	build()
	mu.Lock()
	tools = []any{}
	for _, id := range []string{"invalid1", "repair1", "repair2"} {
		tools = append(tools, call(id, "stage_campaign", map[string]any{"handle": plan.Items[0].Handle,
			"email": ai.AIDraftEmail{Subject: "Bad markup", MJML: "<mjml></mjml>"}}))
	}
	tools = append(tools, buildCalls...)
	mu.Unlock()
	execute()
	if run.State != "paused" || run.PauseReason != "validation_exhausted" || len(run.Artifacts) != 1 || run.Artifacts[0].Handle != plan.Items[1].Handle || run.ToolCalls != 6 {
		t.Fatalf("repair ceiling lost independent output or permitted a third repair: %+v", run)
	}
	// The person's replan starts the repair counts afresh: the new plan is
	// ready to build instead of pausing again on the old output's ceiling.
	author.Do("POST", path+"/actions", ai.AIAgentAction{SharingReviewed: true, Action: "replan", Version: run.Version, PlanRevision: run.PlanRevision, Message: "Try the introduction again."}).MustStatus(t, 204)
	author.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	mu.Lock()
	tools = planCalls
	mu.Unlock()
	execute()
	if run.State != "ready" {
		t.Fatalf("an earlier repair ceiling paused the person's new plan: %+v", run)
	}
	// Revoke the initiating user's edit permission while HTTP is in flight.
	mu.Lock()
	tools = planCalls
	hook = func() {
		_, err := env.App.Pool.Exec(ctx, `UPDATE workspace_members SET role_id=(SELECT id FROM roles WHERE workspace_id=$1 AND key='viewer') WHERE workspace_id=$1 AND user_id=$2`, ws.ID, editor.ID)
		if err != nil {
			t.Error(err)
		}
	}
	mu.Unlock()
	start()
	execute()
	if run.State != "paused" || run.PlanRevision != 0 || run.Tokens != 170 || len(run.Artifacts) != 0 {
		t.Fatalf("late response bypassed revoked access: %+v", run)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM campaigns WHERE workspace_id=$1`, ws.ID).Scan(&n); err != nil || n != 2 {
		t.Fatalf("denied calls changed resources: %d %v", n, err)
	}
}

func TestAgentProposal(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@proposal.test"})
	ws := env.CreateWorkspace("Proposals", owner.ID)
	c := env.Client()
	c.SignInDemo(owner.Email)
	ctx := context.Background()
	base := "/workspaces/" + ws.Slug
	var campaign struct{ ID uuid.UUID }
	c.Do("POST", base+"/campaigns", map[string]any{"name": "Export update"}).MustStatus(t, 201).JSON(t, &campaign)
	source := `<mjml><mj-body><mj-section><mj-column><mj-text>Original description.</mj-text></mj-column></mj-section></mj-body></mjml>`
	c.Do("PUT", base+"/campaigns/"+campaign.ID.String()+"/content", map[string]any{"baseVersion": 1, "subject": "Original subject", "mjml": source}).MustStatus(t, 200)
	var phase atomic.Int32
	var premature atomic.Bool
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		message, stop := map[string]any{"role": "assistant", "content": "Ready"}, "stop"
		messages := req["messages"].([]any)
		// Like a provider that stopped early, do not invent another user
		// instruction when a continuation replays only our terminal answer.
		canBuild := phase.Load() == 1 && !premature.Swap(false) && messages[len(messages)-1].(map[string]any)["role"] == "user"
		if req["tools"] != nil && (phase.Load() == 0 || canBuild) {
			name := "propose_plan"
			var args any = ai.AIAgentPlan{Summary: "Shorten the export draft.", Questions: []string{}, Items: []ai.AIAgentPlanItem{
				{Handle: "shorter", Kind: "campaign", Name: "Export update", Purpose: "Shorter explanation.", Bundle: "update", Target: "selected_campaign", DependsOn: []string{}},
			}}
			if phase.Load() == 1 {
				name = "stage_campaign"
				args = map[string]any{"handle": "shorter", "email": ai.AIDraftEmail{Subject: "A shorter subject", MJML: source}}
			}
			message["tool_calls"] = []any{map[string]any{"id": "one", "type": "function", "function": map[string]any{"name": name, "arguments": mustJSON(t, args)}}}
			stop = "tool_calls"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}}, "usage": map[string]int{"prompt_tokens": 101, "completion_tokens": 31}})
	}))
	defer model.Close()
	c.Do("PUT", base+"/ai/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "proposal-fixture", "capKind": "none", "inputPrice": 3, "outputPrice": 7}).MustStatus(t, 200)
	c.Do("PUT", base+"/ai/agent/settings", ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}}).MustStatus(t, 204)
	s := &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets,
		AllowPrivate: func() bool { return true }, Jobs: func() *jobs.Client { return env.App.Jobs },
		BrandKit: &brandkit.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}}
	var run ai.AIAgentRunView
	path := ""
	execute := func() {
		t.Helper()
		if err := s.ExecuteAgent(ctx, ai.AgentArgs{WorkspaceID: ws.ID, RunID: run.ID, Version: run.Version}); err != nil {
			t.Fatal(err)
		}
		c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	}
	for _, stale := range []bool{true, false} {
		var version int64
		if err := env.App.Pool.QueryRow(ctx, `SELECT version FROM campaigns WHERE id=$1`, campaign.ID).Scan(&version); err != nil {
			t.Fatal(err)
		}
		phase.Store(0)
		run = ai.AIAgentRunView{}
		c.Do("POST", base+"/ai/agent/runs", ai.AIAgentBrief{SharingReviewed: true, Brief: "Shorten the export description.", Language: "en", CampaignID: &campaign.ID, BaseRevision: version}).MustStatus(t, 200).JSON(t, &run)
		path = base + "/ai/agent/runs/" + run.ID.String()
		execute()
		c.Do("POST", path+"/actions", ai.AIAgentAction{Action: "build", Version: run.Version, PlanRevision: run.PlanRevision}).MustStatus(t, 204)
		c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
		phase.Store(1)
		premature.Store(true)
		execute()
		if run.State != "paused" || run.PauseReason != "incomplete_plan" || len(run.Artifacts) != 0 {
			t.Fatalf("premature completion was not paused: %+v", run)
		}
		used := run.Tokens
		c.Do("POST", path+"/actions", ai.AIAgentAction{Action: "continue", Version: run.Version}).MustStatus(t, 204)
		c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
		execute()
		phase.Store(2)
		execute()
		if run.State != "review" || run.Tokens <= used || len(run.Artifacts) != 1 || run.Artifacts[0].TargetID == nil || run.Artifacts[0].Before == nil || run.Artifacts[0].Email.Subject != "A shorter subject" {
			t.Fatalf("proposal not reviewable: %+v", run)
		}
		decision := ai.AIAgentDecision{Version: run.Version, Decision: "accepted", Artifacts: []ai.AIAgentArtifactRevision{{ID: run.Artifacts[0].ID, Revision: 1}}}
		if stale {
			c.Do("PUT", base+"/campaigns/"+campaign.ID.String()+"/content", map[string]any{"baseVersion": version, "subject": "Human revision", "mjml": source}).MustStatus(t, 200)
			c.Do("POST", path+"/decisions", decision).MustStatus(t, 409)
			decision.Decision = "rejected"
		} else {
			wrong := decision
			wrong.Decision = "kept"
			c.Do("POST", path+"/decisions", wrong).MustStatus(t, 409)
		}
		c.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
		c.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
		var got struct {
			Subject, Status string
			Version         int64
		}
		if err := env.App.Pool.QueryRow(ctx, `SELECT subject,status,version FROM campaigns WHERE id=$1`, campaign.ID).Scan(&got.Subject, &got.Status, &got.Version); err != nil {
			t.Fatal(err)
		}
		want := "A shorter subject"
		if stale {
			want = "Human revision"
		}
		if got.Subject != want || got.Status != "draft" || got.Version != version+1 {
			t.Fatalf("proposal applied twice or overwrote human revision: %+v", got)
		}
	}
}

func TestAgentConversation(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@conversation.test"})
	ws := env.CreateWorkspace("Conversation", owner.ID)
	c := env.Client()
	c.SignInDemo(owner.Email)
	base := "/workspaces/" + ws.Slug + "/ai"
	first := ai.AIAgentPlanItem{Handle: "first", Kind: "campaign", Name: "Export guide", Purpose: "Explain the export workflow.", Bundle: "first", DependsOn: []string{}}
	second := ai.AIAgentPlanItem{Handle: "second", Kind: "campaign", Name: "Export tips", Purpose: "Explain column selection.", Bundle: "second", DependsOn: []string{}}
	plan := ai.AIAgentPlan{Summary: "Draft export guides.", Items: []ai.AIAgentPlanItem{first, second}, Questions: []string{}}
	var phase atomic.Int32
	var requests atomic.Int32
	var proposalHandle atomic.Value
	const markup = `<mjml><mj-body><mj-section><mj-column><mj-text>Choose the columns you need.</mj-text></mj-column></mj-section></mj-body></mjml>`
	call := func(id, name string, args any) any {
		return map[string]any{"id": id, "type": "function", "function": map[string]any{"name": name, "arguments": mustJSON(t, args)}}
	}
	email := func(handle, subject string) any {
		return map[string]any{"handle": handle, "email": ai.AIDraftEmail{Subject: subject, MJML: markup}}
	}
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		message, stop := map[string]any{"role": "assistant", "content": "The drafts are ready for review."}, "stop"
		var calls []any
		if req["tools"] != nil {
			requests.Add(1)
			switch phase.Load() {
			case 0:
				calls = []any{call("question", "propose_plan", ai.AIAgentPlan{Summary: "Clarify the writing style.", Questions: []string{"Should the guides be concise?"}, Items: []ai.AIAgentPlanItem{}})}
			case 1:
				calls = []any{call("plan", "propose_plan", plan)}
			case 2:
				calls = []any{call("first", "stage_campaign", email("first", "Export guide")), call("second", "stage_campaign", email("second", "Export tips"))}
			case 4:
				calls = []any{call("neighbor", "stage_campaign", email("second", "Unrequested change")), call("revision", "stage_campaign", email("first", "Short export guide"))}
			case 5:
				replanned := plan
				replanned.Items = append(append([]ai.AIAgentPlanItem(nil), plan.Items...), ai.AIAgentPlanItem{Handle: "third", Kind: "campaign", Name: "Export checklist", Purpose: "Summarize the steps.", Bundle: "third", DependsOn: []string{}})
				calls = []any{call("scope", "propose_plan", replanned), call("premature", "stage_campaign", email("third", "Not yet allowed"))}
			case 6:
				calls = []any{call("third", "stage_campaign", email("third", "Export checklist"))}
			case 7:
				calls = []any{call("kept_revision", "stage_campaign", email(proposalHandle.Load().(string), "Shortened kept draft"))}
			}
		}
		if len(calls) > 0 {
			message["content"], message["tool_calls"], stop = nil, calls, "tool_calls"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}}, "usage": map[string]int{"prompt_tokens": 109, "completion_tokens": 37}})
	}))
	defer model.Close()
	c.Do("PUT", base+"/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "conversation-fixture", "capKind": "none", "inputPrice": 3, "outputPrice": 7}).MustStatus(t, 200)
	c.Do("PUT", base+"/agent/settings", ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}}).MustStatus(t, 204)
	s := &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets,
		AllowPrivate: func() bool { return true }, Jobs: func() *jobs.Client { return env.App.Jobs },
		BrandKit: &brandkit.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}}
	var run ai.AIAgentRunView
	c.Do("POST", base+"/agent/runs", ai.AIAgentBrief{SharingReviewed: true, Brief: "Draft two export guides.", Language: "en"}).MustStatus(t, 200).JSON(t, &run)
	path := base + "/agent/runs/" + run.ID.String()
	get := func() { t.Helper(); c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run) }
	execute := func(p int32) {
		t.Helper()
		phase.Store(p)
		if err := s.ExecuteAgent(context.Background(), ai.AgentArgs{WorkspaceID: ws.ID, RunID: run.ID, Version: run.Version}); err != nil {
			t.Fatal(err)
		}
		get()
	}
	action := func(name, message, handle string) {
		t.Helper()
		in := ai.AIAgentAction{SharingReviewed: true, Action: name, Message: message, Handle: handle, Version: run.Version, PlanRevision: run.PlanRevision}
		c.Do("POST", path+"/actions", in).MustStatus(t, 204)
		c.Do("POST", path+"/actions", in).MustStatus(t, 204)
		get()
	}
	execute(0)
	if run.State != "needs_answers" || len(run.Plan.Items) != 0 || len(run.Plan.Questions) != 1 {
		t.Fatalf("clarification required a fabricated plan: %+v", run)
	}
	c.Do("POST", path+"/actions", ai.AIAgentAction{Action: "build", Version: run.Version, PlanRevision: run.PlanRevision}).MustStatus(t, 409)
	action("answer", "Keep both guides concise.", "")
	execute(1)
	action("build", "", "")
	execute(2)
	execute(3)
	if run.State != "review" || len(run.Artifacts) != 2 {
		t.Fatalf("drafts unavailable: %+v", run)
	}
	action("revise", "Make the first guide shorter.", "first")
	execute(4)
	execute(3)
	if run.Artifacts[0].Revision != 2 || run.Artifacts[0].Email.Subject != "Short export guide" || run.Artifacts[1].Revision != 1 || run.Artifacts[1].Email.Subject != "Export tips" {
		t.Fatal("copy revision changed a neighboring draft or failed to revise its target")
	}
	if len(run.Conversation) != 5 || run.Conversation[1].Text != "Keep both guides concise." || run.Conversation[3].Handle != "first" {
		t.Fatalf("conversation duplicated or lost user messages: %+v", run.Conversation)
	}
	decision := ai.AIAgentDecision{Version: run.Version, Decision: "kept", Artifacts: []ai.AIAgentArtifactRevision{{ID: run.Artifacts[1].ID, Revision: 1}}}
	c.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
	get()
	c.Do("POST", path+"/actions", ai.AIAgentAction{SharingReviewed: true, Action: "revise", Version: run.Version, Handle: "second", Message: "Change this kept draft."}).MustStatus(t, 409)
	beforeTokens, beforeCalls := run.Tokens, run.ToolCalls
	action("replan", "Also create an export checklist.", "")
	if run.AcceptedRevision != 0 {
		t.Fatal("a scope change retained permission to build")
	}
	execute(5)
	if run.State != "ready" || len(run.Artifacts) != 2 || len(run.Plan.Items) != 3 || run.Artifacts[1].Decision != "kept" || run.Tokens <= beforeTokens || run.ToolCalls <= beforeCalls {
		t.Fatalf("replanning wrote early, reset accounting or lost a decision: %+v", run)
	}
	action("build", "", "")
	execute(6)
	execute(3)
	if run.State != "review" || len(run.Artifacts) != 3 || run.Artifacts[2].Email.Subject != "Export checklist" || requests.Load() != 9 {
		t.Fatalf("confirmed scope did not resume once: requests=%d run=%+v", requests.Load(), run)
	}
	kept := *run.Artifacts[1].ResultID
	campaignPath := "/workspaces/" + ws.Slug + "/campaigns/" + kept.String() + "/content"
	c.Do("PUT", campaignPath, map[string]any{"baseVersion": 1, "subject": "Human edit before revision", "mjml": markup}).MustStatus(t, 200)
	beforeRequests := requests.Load()
	action("propose", "Shorten the kept guide.", "second")
	if run.State != "ready" || run.AcceptedRevision == run.PlanRevision || requests.Load() != beforeRequests || run.Artifacts[1].Decision != "kept" {
		t.Fatal("kept edit did not preserve decision or required no confirmation")
	}
	handle := run.Plan.Items[len(run.Plan.Items)-1].Handle
	proposalHandle.Store(handle)
	action("build", "", "")
	execute(7)
	execute(3)
	var proposal ai.AIAgentArtifact
	for _, artifact := range run.Artifacts {
		if artifact.Handle == handle {
			proposal = artifact
		}
	}
	if proposal.TargetID == nil || *proposal.TargetID != kept || proposal.Before == nil || proposal.Before.Subject != "Human edit before revision" || proposal.Email.Subject != "Shortened kept draft" {
		t.Fatalf("kept draft not represented by a fresh diff: %+v; state=%s reason=%s steps=%+v", proposal, run.State, run.PauseReason, run.Steps)
	}
	decision = ai.AIAgentDecision{Version: run.Version, Decision: "accepted", Artifacts: []ai.AIAgentArtifactRevision{{ID: proposal.ID, Revision: proposal.Revision}}}
	c.Do("PUT", campaignPath, map[string]any{"baseVersion": 2, "subject": "Human edit after proposal", "mjml": markup}).MustStatus(t, 200)
	c.Do("POST", path+"/decisions", decision).MustStatus(t, 409)
	beforeRequests = requests.Load()
	action("refresh", "Shorten the latest version instead.", handle)
	if requests.Load() != beforeRequests || run.State != "ready" || run.AcceptedRevision == run.PlanRevision {
		t.Fatal("refresh spent before confirmation or kept the old permission to build")
	}
	c.Do("POST", path+"/decisions", decision).MustStatus(t, 409)
	for _, artifact := range run.Artifacts {
		if artifact.ID == proposal.ID && artifact.Decision != "rejected" {
			t.Fatal("refresh left the superseded proposal open")
		}
	}
	handle = run.Plan.Items[len(run.Plan.Items)-1].Handle
	proposalHandle.Store(handle)
	action("build", "", "")
	execute(7)
	execute(3)
	for _, artifact := range run.Artifacts {
		if artifact.Handle == handle {
			proposal = artifact
		}
	}
	if proposal.Before == nil || proposal.Before.Subject != "Human edit after proposal" || proposal.Email.Subject != "Shortened kept draft" {
		t.Fatalf("refresh reused the stale base: %+v", proposal)
	}
	decision = ai.AIAgentDecision{Version: run.Version, Decision: "accepted", Artifacts: []ai.AIAgentArtifactRevision{{ID: proposal.ID, Revision: proposal.Revision}}}
	c.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
	c.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
	var subject string
	var revision int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT subject,version FROM campaigns WHERE id=$1`, kept).Scan(&subject, &revision); err != nil || subject != "Shortened kept draft" || revision != 4 {
		t.Fatalf("kept proposal accepted incorrectly: %q revision=%d error=%v", subject, revision, err)
	}
}

func TestAgentReplan(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@replan.test"})
	ws := env.CreateWorkspace("Replanning", owner.ID)
	c := env.Client()
	c.SignInDemo(owner.Email)
	ctx := context.Background()
	base := "/workspaces/" + ws.Slug + "/ai"
	first := ai.AIAgentPlanItem{Handle: "first", Kind: "campaign", Name: "First guide", Purpose: "Explain exports.", Bundle: "guides", DependsOn: []string{}}
	second := ai.AIAgentPlanItem{Handle: "second", Kind: "campaign", Name: "Follow-up guide", Purpose: "Summarize exports.", Bundle: "guides", DependsOn: []string{"first"}}
	kept := ai.AIAgentPlanItem{Handle: "kept", Kind: "campaign", Name: "Independent guide", Purpose: "Explain import.", Bundle: "independent", DependsOn: []string{}}
	replacement := ai.AIAgentPlanItem{Handle: "replacement", Kind: "campaign", Name: "Combined guide", Purpose: "Combine the export guides.", Bundle: "combined", DependsOn: []string{}}
	plan := ai.AIAgentPlan{Summary: "Prepare the guides.", Items: []ai.AIAgentPlanItem{first, second, kept}, Questions: []string{}}
	var phase atomic.Int32
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		message, stop := map[string]any{"role": "assistant", "content": "Ready for review."}, "stop"
		calls := []any{}
		call := func(name string, args any) {
			calls = append(calls, map[string]any{"id": fmt.Sprintf("call_%d", len(calls)), "type": "function", "function": map[string]any{"name": name, "arguments": mustJSON(t, args)}})
		}
		stage := func(handle string) {
			call("stage_campaign", map[string]any{"handle": handle, "email": ai.AIDraftEmail{Subject: "Guide", MJML: `<mjml><mj-body><mj-section><mj-column><mj-text>Choose export columns.</mj-text></mj-column></mj-section></mj-body></mjml>`}})
		}
		if req["tools"] != nil {
			next := plan
			switch phase.Load() {
			case 0:
				call("propose_plan", plan)
			case 1:
				for _, item := range plan.Items {
					stage(item.Handle)
				}
			case 3:
				next.Items = []ai.AIAgentPlanItem{first, kept} // Silently drops a dependent.
				call("propose_plan", next)
			case 4:
				next.Items = []ai.AIAgentPlanItem{replacement} // Silently drops a kept resource.
				call("propose_plan", next)
			case 5:
				next.Items = []ai.AIAgentPlanItem{replacement, kept}
				call("propose_plan", next)
				stage("replacement") // Cannot stage in the planning response.
			case 6:
				stage("replacement")
			}
		}
		if len(calls) != 0 {
			message["content"], message["tool_calls"], stop = nil, calls, "tool_calls"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}}, "usage": map[string]int{"prompt_tokens": 113, "completion_tokens": 41}})
	}))
	defer model.Close()
	c.Do("PUT", base+"/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "replan-fixture", "capKind": "none", "inputPrice": 3, "outputPrice": 7}).MustStatus(t, 200)
	c.Do("PUT", base+"/agent/settings", ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}}).MustStatus(t, 204)
	s := &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, AllowPrivate: func() bool { return true }, Jobs: func() *jobs.Client { return env.App.Jobs },
		BrandKit: &brandkit.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}}
	for _, cancel := range []bool{false, true} {
		t.Run(fmt.Sprintf("cancel_%t", cancel), func(t *testing.T) {
			var run ai.AIAgentRunView
			c.Do("POST", base+"/agent/runs", ai.AIAgentBrief{SharingReviewed: true, Brief: "Prepare export and import guides.", Language: "en"}).MustStatus(t, 200).JSON(t, &run)
			path := base + "/agent/runs/" + run.ID.String()
			get := func() {
				t.Helper()
				var fresh ai.AIAgentRunView
				c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &fresh)
				run = fresh
			}
			execute := func(p int32) {
				t.Helper()
				phase.Store(p)
				if err := s.ExecuteAgent(ctx, ai.AgentArgs{WorkspaceID: ws.ID, RunID: run.ID, Version: run.Version}); err != nil {
					t.Fatal(err)
				}
				get()
			}
			action := func(name string) {
				t.Helper()
				c.Do("POST", path+"/actions", ai.AIAgentAction{SharingReviewed: true, Action: name, Version: run.Version, PlanRevision: run.PlanRevision, Message: "Combine the export guides."}).MustStatus(t, 204)
				get()
			}
			execute(0)
			action("build")
			execute(1)
			execute(2)
			var keptID uuid.UUID
			for _, a := range run.Artifacts {
				if a.Handle == "kept" {
					keptID = a.ID
				}
			}
			c.Do("POST", path+"/decisions", ai.AIAgentDecision{Version: run.Version, Decision: "kept", Artifacts: []ai.AIAgentArtifactRevision{{ID: keptID, Revision: 1}}}).MustStatus(t, 204)
			get()
			beforeTokens := run.Tokens
			action("replan")
			for _, invalid := range []int32{3, 4} {
				execute(invalid)
				if run.State != "planning" || len(run.Plan.Items) != 3 || len(run.PlanDiscards) != 0 {
					t.Fatalf("unsafe replan accepted: state=%s plan=%+v", run.State, run.Plan)
				}
			}
			execute(5)
			if run.State != "ready" || len(run.Plan.Items) != 2 || len(run.PlanDiscards) != 2 || len(run.Artifacts) != 3 || run.Tokens <= beforeTokens {
				t.Fatalf("replacement not proposed for review: %+v", run)
			}
			for _, a := range run.Artifacts {
				if a.Handle != "kept" && (a.Decision != "" || a.Email == nil) {
					t.Fatal("model removed staging content before confirmation")
				}
			}
			build := ai.AIAgentAction{Action: "build", Version: run.Version, PlanRevision: run.PlanRevision}
			for _, acknowledgments := range [][]ai.AIAgentArtifactRevision{nil, run.PlanDiscards[:1], {run.PlanDiscards[0], run.PlanDiscards[0]}, {{ID: run.PlanDiscards[0].ID, Revision: 2}, run.PlanDiscards[1]}} {
				build.PlanDiscards = acknowledgments
				c.Do("POST", path+"/actions", build).MustStatus(t, 409)
			}
			if cancel {
				action("cancel")
				if run.State != "cancelled" || len(run.PlanDiscards) != 0 || len(run.Plan.Items) != 3 {
					t.Fatal("cancel did not restore the original review scope")
				}
				// Deleting the kept draft leaves the rest of the run reviewable.
				for _, a := range run.Artifacts {
					if a.Handle == "kept" {
						c.Do("DELETE", "/workspaces/"+ws.Slug+"/campaigns/"+a.ResultID.String(), nil).MustStatus(t, 204)
					}
				}
				get()
				original := []ai.AIAgentArtifactRevision{}
				for _, a := range run.Artifacts {
					if a.Handle != "kept" {
						original = append(original, ai.AIAgentArtifactRevision{ID: a.ID, Revision: a.Revision})
					}
				}
				c.Do("POST", path+"/decisions", ai.AIAgentDecision{Version: run.Version, Decision: "kept", Artifacts: original}).MustStatus(t, 204)
				return
			}
			build.PlanDiscards = run.PlanDiscards
			c.Do("POST", path+"/actions", build).MustStatus(t, 204)
			c.Do("POST", path+"/actions", build).MustStatus(t, 204)
			get()
			if len(run.PlanDiscards) != 0 || run.Artifacts[0].Decision != "discarded" || run.Artifacts[1].Decision != "discarded" || run.Artifacts[0].Email != nil || run.Artifacts[1].Email != nil {
				t.Fatal("confirmed removal was incomplete")
			}
			execute(6)
			execute(2)
			if run.State != "review" || len(run.Artifacts) != 4 || run.Artifacts[0].Handle != "replacement" || run.Artifacts[0].Email == nil {
				t.Fatal("replacement did not reach review")
			}
			var resources, removals int
			if err := env.App.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM campaigns WHERE workspace_id=$1), (SELECT count(*) FROM audit_events WHERE action='ai.agent.discarded' AND metadata->>'run_id'=$2)`, ws.ID, run.ID.String()).Scan(&resources, &removals); err != nil || resources != 1 || removals != 2 {
				t.Fatalf("unexpected resources or repeated removal audit: %d %d %v", resources, removals, err)
			}
		})
	}
}

func TestAgentAudience(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@audience.test"})
	ws := env.CreateWorkspace("Agent audience", owner.ID)
	c := env.Client()
	c.SignInDemo(owner.Email)
	ctx := context.Background()
	base := "/workspaces/" + ws.Slug + "/ai"
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO custom_fields (id,workspace_id,key,label,type) VALUES ($1,$2,'contract_level','Private contact label','text')`, uuid.Must(uuid.NewV7()), ws.ID)
	for i := range 81 {
		value := "Other"
		if i < 31 {
			value = "Selected tier"
		}
		exec(`INSERT INTO contacts (id,workspace_id,email,first_name,attributes) VALUES ($1,$2,$3,'PRIVATE_PERSON',jsonb_build_object('contract_level',$4::text))`, uuid.Must(uuid.NewV7()), ws.ID, fmt.Sprintf("reader%d@private.test", i), value)
	}
	definition := segments.SegmentDefinition{Groups: []segments.SegmentGroup{{Match: "all", Conditions: []segments.SegmentCondition{{Field: "custom:contract_level", Op: "equals", Value: "Selected tier"}}}}}
	c.Do("PUT", "/workspaces/"+ws.Slug+"/brand-kit/voice", brandkit.BrandVoiceSaveInput{Content: brandkit.BrandVoiceContent{
		Tone: "Explain Selected tier clearly.", Examples: "Choose Ｓｅｌｅｃｔｅｄ ｔｉｅｒ using contract_level.",
	}}).MustStatus(t, 200)
	var source struct{ ID uuid.UUID }
	c.Do("POST", "/workspaces/"+ws.Slug+"/campaigns", map[string]any{"name": "Local source"}).MustStatus(t, 201).JSON(t, &source)
	c.Do("PUT", "/workspaces/"+ws.Slug+"/campaigns/"+source.ID.String()+"/content", map[string]any{
		"baseVersion": 1, "subject": "Selected tier guide", "preheader": "Learn about contract_level",
		"mjml": `<mjml><mj-body><mj-section><mj-column><mj-text>Use selected TIER and Ｓｅｌｅｃｔｅｄ ｔｉｅｒ.</mj-text></mj-column></mj-section></mj-body></mjml>`,
	}).MustStatus(t, 200)
	plan := ai.AIAgentPlan{Summary: "Prepare the selected audience.", Items: []ai.AIAgentPlanItem{{Handle: "audience", Kind: "segment", Name: "Selected audience", Purpose: "Use the exact selected conditions.", Bundle: "audience", DependsOn: []string{}}}, Questions: []string{}}
	var phase, requests atomic.Int32
	var mu sync.Mutex
	var payloads []string
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req map[string]any
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Error(err)
			return
		}
		message, stop := map[string]any{"role": "assistant", "content": "Ready to review."}, "stop"
		if req["tools"] != nil {
			requests.Add(1)
			mu.Lock()
			payloads = append(payloads, string(raw))
			mu.Unlock()
			call := func(id, name string, args any) any {
				return map[string]any{"id": id, "type": "function", "function": map[string]any{"name": name, "arguments": mustJSON(t, args)}}
			}
			var calls []any
			switch phase.Load() {
			case 0:
				calls = []any{call("inspect", "inspect_audience", struct{}{}), call("plan", "propose_plan", plan)}
			case 1:
				calls = []any{call("injected", "stage_segment", map[string]any{"handle": "audience", "definition": map[string]any{"groups": []any{}}}), call("stage", "stage_segment", map[string]any{"handle": "audience"})}
			case 3:
				unsafe := plan
				unsafe.Summary = "Selected&#32;tier is the audience."
				calls = []any{call("unsafe", "propose_plan", unsafe)}
			case 5:
				// The selected audience's field is deleted while this
				// request is in flight.
				if _, err := env.App.Pool.Exec(ctx, `DELETE FROM custom_fields WHERE workspace_id=$1 AND key='contract_level'`, ws.ID); err != nil {
					t.Error(err)
				}
				calls = []any{call("plan", "propose_plan", plan)}
			}
			if len(calls) > 0 {
				message["content"], message["tool_calls"], stop = nil, calls, "tool_calls"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}}, "usage": map[string]int{"prompt_tokens": 103, "completion_tokens": 29}})
	}))
	defer model.Close()
	c.Do("PUT", base+"/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "audience-fixture", "capKind": "none", "inputPrice": 3, "outputPrice": 7}).MustStatus(t, 200)
	c.Do("PUT", base+"/agent/settings", ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}}).MustStatus(t, 204)
	s := &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets,
		AllowPrivate: func() bool { return true }, Jobs: func() *jobs.Client { return env.App.Jobs },
		Segments: &segments.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}}
	var run ai.AIAgentRunView
	start := func() string {
		t.Helper()
		run = ai.AIAgentRunView{}
		c.Do("POST", base+"/agent/runs", ai.AIAgentBrief{SharingReviewed: true, Brief: "Prepare a segment for Selected tier readers.", Language: "en", Audience: &definition, CampaignID: &source.ID, BaseRevision: 2}).MustStatus(t, 200).JSON(t, &run)
		return base + "/agent/runs/" + run.ID.String()
	}
	path := start()
	execute := func(p int32) {
		t.Helper()
		phase.Store(p)
		if err := s.ExecuteAgent(ctx, ai.AgentArgs{WorkspaceID: ws.ID, RunID: run.ID, Version: run.Version}); err != nil {
			t.Fatal(err)
		}
		c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	}
	execute(0)
	if run.State != "ready" || len(run.Steps[0].Operations) != 2 {
		t.Fatalf("audience plan unavailable: %+v", run)
	}
	if run.Audience == nil || run.Audience.Count == nil || *run.Audience.Count != 30 || run.Audience.Total == nil || *run.Audience.Total != 80 {
		t.Fatalf("human review lost its rounded count: %+v", run.Audience)
	}
	c.Do("POST", path+"/actions", ai.AIAgentAction{SharingReviewed: true, Action: "answer", Version: run.Version, Message: "Keep Selected tier and Ｓｅｌｅｃｔｅｄ ｔｉｅｒ in contract_level."}).MustStatus(t, 204)
	c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	execute(0)
	c.Do("POST", path+"/actions", ai.AIAgentAction{Action: "build", Version: run.Version, PlanRevision: run.PlanRevision}).MustStatus(t, 204)
	c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	execute(1)
	execute(2)
	if run.State != "review" || len(run.Artifacts) != 1 || run.Artifacts[0].Segment == nil || mustJSON(t, *run.Artifacts[0].Segment) != mustJSON(t, definition) || run.Artifacts[0].Revision != 1 || !run.Steps[2].Operations[0].Failed {
		t.Fatalf("model changed or duplicated the selected audience: %+v", run)
	}
	var n int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM segments WHERE workspace_id=$1`, ws.ID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("staging created a live segment: %d %v", n, err)
	}
	decision := ai.AIAgentDecision{Version: run.Version, Decision: "kept", Artifacts: []ai.AIAgentArtifactRevision{{ID: run.Artifacts[0].ID, Revision: 1}}}
	exec(`UPDATE workspace_members SET scope=jsonb_build_object('lists',jsonb_build_array($1::text)) WHERE workspace_id=$2 AND user_id=$3`, uuid.NewString(), ws.ID, owner.ID)
	c.Do("GET", path, nil).MustStatus(t, 403)
	c.Do("POST", path+"/decisions", decision).MustStatus(t, 403)
	exec(`UPDATE workspace_members SET scope='{}' WHERE workspace_id=$1 AND user_id=$2`, ws.ID, owner.ID)
	c.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
	c.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
	var saved []byte
	if err := env.App.Pool.QueryRow(ctx, `SELECT definition FROM segments WHERE workspace_id=$1`, ws.ID).Scan(&saved); err != nil {
		t.Fatal(err)
	}
	var actual segments.SegmentDefinition
	if err := json.Unmarshal(saved, &actual); err != nil || mustJSON(t, actual) != mustJSON(t, definition) {
		t.Fatalf("Keep changed literals: %s %v", saved, err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM segment_members`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("Keep materialized members: %d %v", n, err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM segment_member_events`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("Keep emitted enrollment events: %d %v", n, err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='segment.create' AND metadata->>'via'='ai_agent' AND metadata->>'run_id'=$1`, run.ID.String()).Scan(&n); err != nil || n != 1 {
		t.Fatalf("missing or duplicate segment provenance: %d %v", n, err)
	}
	// A small complement must be suppressed as well as a small cohort.
	for _, selected := range []int{19, 62} {
		exec(`UPDATE contacts SET attributes=jsonb_build_object('contract_level','Other') WHERE workspace_id=$1`, ws.ID)
		exec(`UPDATE contacts SET attributes=jsonb_build_object('contract_level','Selected tier') WHERE id IN (SELECT id FROM contacts WHERE workspace_id=$1 ORDER BY id LIMIT $2)`, ws.ID, selected)
		path = start()
		execute(0)
		if run.Audience == nil || run.Audience.Status != "withheld" || run.Audience.Count != nil || run.Audience.Total != nil {
			t.Fatalf("small cohort/complement was exposed: %+v", run.Audience)
		}
	}
	// Deleting a source field blocks execution/Keep, not recovery/Discard.
	c.Do("POST", path+"/actions", ai.AIAgentAction{Action: "build", Version: run.Version, PlanRevision: run.PlanRevision}).MustStatus(t, 204)
	c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	execute(1)
	execute(2)
	exec(`DELETE FROM custom_fields WHERE workspace_id=$1 AND key='contract_level'`, ws.ID)
	c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	decision = ai.AIAgentDecision{Version: run.Version, Decision: "kept", Artifacts: []ai.AIAgentArtifactRevision{{ID: run.Artifacts[0].ID, Revision: 1}}}
	c.Do("POST", path+"/decisions", decision).MustStatus(t, 422)
	decision.Decision = "discarded"
	c.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
	c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	if run.State != "completed" || run.Artifacts[0].Decision != "discarded" {
		t.Fatalf("deleted-field recovery failed: %+v", run)
	}
	exec(`INSERT INTO custom_fields (id,workspace_id,key,label,type) VALUES ($1,$2,'contract_level','Private contact label','text')`, uuid.Must(uuid.NewV7()), ws.ID)
	path = start()
	execute(3)
	if run.State != "paused" || run.PauseReason != "refused" || run.Plan != nil || run.Tokens != 132 || run.CostMicros == 0 {
		t.Fatalf("private model output was applied or lost accounting: %+v", run)
	}
	// Access lost during the request pauses the run and keeps its known usage.
	path = start()
	execute(5)
	if run.State != "paused" || run.PauseReason != "access_changed" || run.Tokens != 132 || run.Uncertain {
		t.Fatalf("an in-flight access change lost accounting or left the run running: %+v", run)
	}
	mu.Lock()
	defer mu.Unlock()
	if requests.Load() != 10 || len(payloads) != 10 {
		t.Fatalf("unexpected model requests: %d payloads=%d", requests.Load(), len(payloads))
	}
	for _, payload := range payloads {
		for _, secret := range []string{"Selected tier", "selected TIER", "Ｓｅｌｅｃｔｅｄ ｔｉｅｒ", "contract_level", "Private contact label", "PRIVATE_PERSON", "@private.test", "Other"} {
			if strings.Contains(payload, secret) {
				t.Fatalf("private audience data leaked: %s", secret)
			}
		}
		if !strings.Contains(payload, "local_review_only") || !strings.Contains(payload, "bound_") || strings.Contains(payload, `\"count\":`) || strings.Contains(payload, `\"total\":`) {
			t.Fatal("model received a count oracle instead of the selected schema")
		}
	}
}

func TestAgentAutomationBundle(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@agent-flow.test"})
	ws := env.CreateWorkspace("Agent flow", owner.ID)
	c := env.Client()
	c.SignInDemo(owner.Email)
	ctx := context.Background()
	base := "/workspaces/" + ws.Slug + "/ai"
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	list, segment := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	exec(`INSERT INTO lists(id,workspace_id,name) VALUES($1,$2,'PRIVATE_LIST')`, list, ws.ID)
	exec(`INSERT INTO segments(id,workspace_id,name,mode,refresh,definition) VALUES($1,$2,'PRIVATE_SEGMENT','builder','dynamic','{"groups":[]}')`, segment, ws.ID)
	counts := func() [4]int {
		t.Helper()
		var n [4]int
		if err := env.App.Pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM segments WHERE workspace_id=$1),
		(SELECT count(*) FROM automations WHERE workspace_id=$1),
		(SELECT count(*) FROM automation_emails WHERE workspace_id=$1),
		(SELECT count(*) FROM automation_runs r JOIN automations a ON a.id=r.automation_id WHERE a.workspace_id=$1)`, ws.ID).Scan(&n[0], &n[1], &n[2], &n[3]); err != nil {
			t.Fatal(err)
		}
		return n
	}
	var phase atomic.Int32
	var mu sync.Mutex
	var plan ai.AIAgentPlan
	var flow ai.AIAutomationContent
	var payloads []string
	var proposalFlow ai.AIAutomationContent
	var proposalHandle string
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req map[string]any
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Error(err)
			return
		}
		message, stop := map[string]any{"role": "assistant", "content": "Ready for review."}, "stop"
		if req["tools"] != nil {
			mu.Lock()
			defer mu.Unlock()
			payloads = append(payloads, string(raw))
			call := func(id, name string, args any) any {
				return map[string]any{"id": id, "type": "function", "function": map[string]any{"name": name, "arguments": mustJSON(t, args)}}
			}
			var calls []any
			switch phase.Load() {
			case 0:
				calls = []any{call("plan", "propose_plan", plan)}
			case 1:
				calls = []any{call("segment", "stage_segment", map[string]any{"handle": "z_audience"})}
			case 2:
				bad := strings.Replace(mustJSON(t, flow), `"reentry":"once"`, `"reentry":"always"`, 1)
				calls = []any{call("wrong_audience", "stage_automation", map[string]any{"handle": "a_flow", "content": json.RawMessage(bad)}),
					call("flow", "stage_automation", map[string]any{"handle": "a_flow", "content": flow})}
			case 4:
				calls = []any{call("revision", "stage_automation", map[string]any{"handle": proposalHandle, "content": proposalFlow})}
			}
			if len(calls) > 0 {
				message["content"], message["tool_calls"], stop = nil, calls, "tool_calls"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}}, "usage": map[string]int{"prompt_tokens": 103, "completion_tokens": 79}})
	}))
	defer model.Close()
	c.Do("PUT", base+"/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "flow-bundle-fixture", "capKind": "none", "inputPrice": 3, "outputPrice": 7}).MustStatus(t, 200)
	c.Do("PUT", base+"/agent/settings", ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}}).MustStatus(t, 204)
	// An automation-only editor needs no campaign access. Losing ai:use
	// later still permits reading and keeping their authorized drafts.
	editor := env.CreateUser(apptest.UserOpts{Email: "editor@agent-flow.test"})
	env.AddMember(ws, editor.ID, "editor")
	role := uuid.Must(uuid.NewV7())
	permissions := []string{"ai:use", "automations:view", "automations:edit", "subscribers:view", "segments:edit"}
	exec(`INSERT INTO roles(id,workspace_id,key,name,permissions) VALUES($1,$2,'agent-flow-editor','Flow editor',$3)`, role, ws.ID, permissions)
	exec(`UPDATE workspace_members SET role_id=$1 WHERE workspace_id=$2 AND user_id=$3`, role, ws.ID, editor.ID)
	c = env.Client()
	c.SignInDemo(editor.Email)
	s := &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets,
		AllowPrivate: func() bool { return true }, Jobs: func() *jobs.Client { return env.App.Jobs },
		BrandKit:    &brandkit.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit},
		Automations: &automations.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit},
		Segments:    &segments.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}}
	for _, selection := range []ai.AIAgentAutomationSelection{
		{Trigger: "segment_entered", FromAudience: true},
		{Trigger: "joined_list", ListID: &list},
		{Trigger: "segment_left", SegmentID: &segment},
	} {
		t.Run(selection.Trigger, func(t *testing.T) {
			exec(`UPDATE roles SET permissions=$2 WHERE id=$1`, role, permissions)
			before := counts()
			in := ai.AIAgentBrief{SharingReviewed: true, Brief: "Draft a two-email export sequence for future events.", Language: "en", Automation: &selection}
			item := ai.AIAgentPlanItem{Handle: "a_flow", Kind: "automation", Name: "Export sequence", Purpose: "Help readers use exports.", Bundle: "sequence", DependsOn: []string{}}
			trigger := automations.AutomationTrigger{Kind: selection.Trigger, Reentry: "once"}
			items := []ai.AIAgentPlanItem{}
			switch {
			case selection.FromAudience:
				in.Audience = &segments.SegmentDefinition{Groups: []segments.SegmentGroup{{Match: "all", Conditions: []segments.SegmentCondition{{Field: "consent", Op: "is", Value: "double"}}}}}
				items = append(items, ai.AIAgentPlanItem{Handle: "z_audience", Kind: "segment", Name: "Selected readers", Purpose: "Use the selected conditions.", Bundle: "sequence", DependsOn: []string{}})
				item.DependsOn, trigger.SegmentID = []string{"z_audience"}, "z_audience"
			case selection.ListID != nil:
				trigger.ListID, trigger.ConfirmedOnly, trigger.ExitOnLeave = "selected_list", true, true
			default:
				trigger.SegmentID = "selected_segment"
			}
			const markup = `<mjml><mj-body><mj-section><mj-column><mj-text>Select the columns you need.</mj-text></mj-column></mj-section></mj-body></mjml>`
			mu.Lock()
			plan = ai.AIAgentPlan{Summary: "Prepare an export sequence.", Items: append(items, item), Questions: []string{}}
			flow = ai.AIAutomationContent{Graph: automations.AutomationGraph{
				Nodes: []automations.AutomationNode{
					{ID: "start", Type: "trigger", Data: automations.AutomationNodeData{Trigger: &trigger}},
					{ID: "first", Type: "email", Data: automations.AutomationNodeData{Email: &automations.AutomationEmailStep{EmailID: "intro", EmailVersion: 1, Subject: "Export guide"}}},
					{ID: "wait", Type: "delay", Data: automations.AutomationNodeData{Delay: &automations.AutomationDuration{Amount: 5, Unit: "days"}}},
					{ID: "second", Type: "email", Data: automations.AutomationNodeData{Email: &automations.AutomationEmailStep{EmailID: "followup", EmailVersion: 1, Subject: "Export shortcuts"}}},
					{ID: "end", Type: "exit"},
				},
				Edges: []automations.AutomationEdge{{ID: "a", Source: "start", Target: "first"}, {ID: "b", Source: "first", Target: "wait"}, {ID: "c", Source: "wait", Target: "second"}, {ID: "d", Source: "second", Target: "end"}},
			}, Emails: []automations.ProposedEmail{{Key: "intro", Name: "Introduction", Subject: "Export guide", MJML: markup}, {Key: "followup", Name: "Follow-up", Subject: "Export shortcuts", MJML: markup}}}
			mu.Unlock()
			var run ai.AIAgentRunView
			c.Do("POST", base+"/agent/runs", in).MustStatus(t, 200).JSON(t, &run)
			path := base + "/agent/runs/" + run.ID.String()
			execute := func(p int32) {
				t.Helper()
				phase.Store(p)
				if err := s.ExecuteAgent(ctx, ai.AgentArgs{WorkspaceID: ws.ID, RunID: run.ID, Version: run.Version}); err != nil {
					t.Fatal(err)
				}
				c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
			}
			execute(0)
			c.Do("POST", path+"/actions", ai.AIAgentAction{Action: "build", Version: run.Version, PlanRevision: run.PlanRevision}).MustStatus(t, 204)
			c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
			if selection.FromAudience {
				execute(1)
			}
			execute(2)
			execute(3)
			if run.State != "review" || len(run.Artifacts) != len(plan.Items) || run.Artifacts[0].Automation == nil || len(run.Artifacts[0].Previews) != 2 || run.Artifacts[0].Revision != 1 || counts() != before {
				t.Fatalf("flow not staged safely: state=%s reason=%s artifacts=%+v steps=%+v", run.State, run.PauseReason, run.Artifacts, run.Steps)
			}
			triggerName := "PRIVATE_SEGMENT"
			if selection.FromAudience {
				triggerName = "Selected readers"
			} else if selection.ListID != nil {
				triggerName = "PRIVATE_LIST"
			}
			if run.Artifacts[0].TriggerName != triggerName {
				t.Fatalf("human review lost the selected trigger: %q", run.Artifacts[0].TriggerName)
			}
			decision := ai.AIAgentDecision{Version: run.Version, Decision: "kept", Artifacts: []ai.AIAgentArtifactRevision{}}
			for _, a := range run.Artifacts {
				decision.Artifacts = append(decision.Artifacts, ai.AIAgentArtifactRevision{ID: a.ID, Revision: a.Revision})
			}
			exec(`UPDATE roles SET permissions=array_remove(permissions,'automations:edit') WHERE id=$1`, role)
			c.Do("GET", path, nil).MustStatus(t, 200)
			c.Do("POST", path+"/decisions", decision).MustStatus(t, 403)
			exec(`UPDATE roles SET permissions=$2 WHERE id=$1`, role, permissions[1:])
			c.Do("POST", path+"/actions", ai.AIAgentAction{SharingReviewed: true, Action: "revise", Version: run.Version, Handle: "a_flow", Message: "Shorten the emails."}).MustStatus(t, 409)
			var listed ai.AIAgentRunList
			c.Do("GET", base+"/agent/runs", nil).MustStatus(t, 200).JSON(t, &listed)
			if len(listed.Items) == 0 {
				t.Fatal("automation-only editor lost run list access")
			}
			if selection.FromAudience {
				partial := decision
				partial.Artifacts = partial.Artifacts[:1]
				c.Do("POST", path+"/decisions", partial).MustStatus(t, 409)
				// A damaged downstream artifact must roll back the already
				// inserted segment, its audit and every review decision.
				var saved []byte
				if err := env.App.Pool.QueryRow(ctx, `SELECT content FROM ai_agent_artifacts WHERE id=$1`, run.Artifacts[0].ID).Scan(&saved); err != nil {
					t.Fatal(err)
				}
				exec(`UPDATE ai_agent_artifacts SET content=jsonb_set(content,'{graph,nodes,0,data,trigger,reentry}','"always"') WHERE id=$1`, run.Artifacts[0].ID)
				c.Do("POST", path+"/decisions", decision).MustStatus(t, 422)
				if counts() != before {
					t.Fatal("failed bundle left partial resources")
				}
				exec(`UPDATE ai_agent_artifacts SET content=$2 WHERE id=$1`, run.Artifacts[0].ID, saved)
			}
			responses := make(chan *apptest.Response, 2)
			for range 2 {
				go func() { responses <- c.Do("POST", path+"/decisions", decision) }()
			}
			(<-responses).MustStatus(t, 204)
			(<-responses).MustStatus(t, 204)
			c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
			want := [4]int{before[0], before[1] + 1, before[2] + 2, 0}
			if selection.FromAudience {
				want[0]++
			}
			if counts() != want || run.State != "completed" || run.Artifacts[0].ResultID == nil {
				t.Fatalf("Keep not atomic/idempotent: counts=%v want=%v state=%s", counts(), want, run.State)
			}
			id := *run.Artifacts[0].ResultID
			var status string
			var live *int
			var raw []byte
			if err := env.App.Pool.QueryRow(ctx, `SELECT a.status,a.live_version,v.graph FROM automations a JOIN automation_versions v ON v.automation_id=a.id AND v.version=a.draft_version WHERE a.id=$1`, id).Scan(&status, &live, &raw); err != nil {
				t.Fatal(err)
			}
			graph, err := automations.ParseGraph(raw)
			if err != nil || status != "draft" || live != nil {
				t.Fatalf("kept flow became live: %s %v %v", status, live, err)
			}
			actual := graph.Trigger().Data.Trigger
			if actual.Kind != selection.Trigger || actual.Reentry != "once" || (selection.ListID != nil && actual.ListID != list.String()) || (selection.SegmentID != nil && actual.SegmentID != segment.String()) || (selection.FromAudience && actual.SegmentID != run.Artifacts[1].ResultID.String()) {
				t.Fatalf("trigger changed at Keep: %+v", actual)
			}
			for _, node := range graph.Nodes {
				if node.Type != "email" {
					continue
				}
				var subject, html string
				if err := env.App.Pool.QueryRow(ctx, `SELECT v.subject,v.html FROM automation_emails e JOIN automation_email_versions v ON v.email_id=e.id AND v.version=1 WHERE e.id=$1 AND e.automation_id=$2`, node.Data.Email.EmailID, id).Scan(&subject, &html); err != nil || subject != node.Data.Email.Subject || !strings.Contains(html, "Select the columns") {
					t.Fatalf("flow-owned email missing or mismatched: %q %v", subject, err)
				}
			}
			var n int
			if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action IN ('segment.create','automation.create','automation_email.create') AND metadata->>'run_id'=$1 AND metadata->>'via'='ai_agent' AND actor_type='user'`, run.ID.String()).Scan(&n); err != nil || n != 3+want[0]-before[0] {
				t.Fatalf("provenance missing or duplicated: %d %v", n, err)
			}
			decision.Decision = "discarded"
			c.Do("POST", path+"/decisions", decision).MustStatus(t, 409)
			if counts() != want {
				t.Fatal("Discard changed kept resources")
			}
			// Revising this kept flow stages a diff, never overwrites it.
			exec(`UPDATE roles SET permissions=$2 WHERE id=$1`, role, permissions)
			c.Do("POST", path+"/actions", ai.AIAgentAction{SharingReviewed: true, Action: "propose", Version: run.Version, Handle: "a_flow", Message: "Shorten the first email subject."}).MustStatus(t, 204)
			c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
			for round := range 2 {
				if run.State != "ready" || run.AcceptedRevision == run.PlanRevision {
					t.Fatal("flow proposal skipped confirmation")
				}
				mu.Lock()
				proposalHandle = run.Plan.Items[len(run.Plan.Items)-1].Handle
				proposalFlow = ai.AIAutomationContent{}
				if err := json.Unmarshal([]byte(mustJSON(t, flow)), &proposalFlow); err != nil {
					t.Fatal(err)
				}
				if selection.Trigger != "joined_list" {
					proposalFlow.Graph.Trigger().Data.Trigger.SegmentID = "selected_segment"
				}
				for i := range proposalFlow.Emails {
					proposalFlow.Emails[i].Key = fmt.Sprintf("source%d", i+1)
				}
				proposalFlow.Emails[0].Subject = "Export tips"
				for i := range proposalFlow.Graph.Nodes {
					if email := proposalFlow.Graph.Nodes[i].Data.Email; email != nil {
						if email.EmailID == "intro" {
							email.EmailID, email.Subject = "source1", "Export tips"
						} else {
							email.EmailID = "source2"
						}
					}
				}
				mu.Unlock()
				c.Do("POST", path+"/actions", ai.AIAgentAction{Action: "build", Version: run.Version, PlanRevision: run.PlanRevision}).MustStatus(t, 204)
				c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
				execute(4)
				execute(3)
				var artifact ai.AIAgentArtifact
				for _, a := range run.Artifacts {
					if a.Handle == proposalHandle {
						artifact = a
					}
				}
				if run.State != "review" || artifact.TargetID == nil || *artifact.TargetID != id || artifact.BeforeAutomation == nil || len(artifact.BeforeEmails) != 2 || artifact.BeforeEmails[0].Content.Subject != "Export guide" || artifact.Automation == nil || artifact.Automation.Emails[0].Subject != "Export tips" || counts() != want {
					t.Fatalf("flow diff not isolated: state=%s reason=%s artifact=%+v counts=%v", run.State, run.PauseReason, artifact, counts())
				}
				if artifact.TriggerName != triggerName {
					t.Fatalf("proposal lost the selected trigger: %q", artifact.TriggerName)
				}
				accept := ai.AIAgentDecision{Version: run.Version, Decision: "accepted", Artifacts: []ai.AIAgentArtifactRevision{{ID: artifact.ID, Revision: artifact.Revision}}}
				if round == 0 {
					exec(`UPDATE automations SET description='Human description' WHERE id=$1`, id)
					c.Do("POST", path+"/decisions", accept).MustStatus(t, 409)
					c.Do("POST", path+"/actions", ai.AIAgentAction{SharingReviewed: true, Action: "refresh", Version: run.Version, Handle: proposalHandle, Message: "Keep the current settings and shorten the first subject."}).MustStatus(t, 204)
					c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
					continue
				}
				exec(`UPDATE roles SET permissions=array_remove(permissions,'automations:edit') WHERE id=$1`, role)
				c.Do("POST", path+"/decisions", accept).MustStatus(t, 403)
				exec(`UPDATE roles SET permissions=$2 WHERE id=$1`, role, permissions[1:])
				c.Do("POST", path+"/decisions", accept).MustStatus(t, 204)
				c.Do("POST", path+"/decisions", accept).MustStatus(t, 204)
				want[2] += 2
				var revision int64
				var description string
				if err := env.App.Pool.QueryRow(ctx, `SELECT a.status,a.live_version,a.draft_revision,a.description,v.graph FROM automations a JOIN automation_versions v ON v.automation_id=a.id AND v.version=a.draft_version WHERE a.id=$1`, id).Scan(&status, &live, &revision, &description, &raw); err != nil {
					t.Fatal(err)
				}
				updated, err := automations.ParseGraph(raw)
				if err != nil || status != "draft" || live != nil || revision != 2 || description != "Human description" || counts() != want || mustJSON(t, updated.Trigger().Data.Trigger) != mustJSON(t, actual) {
					t.Fatalf("Accept changed enrollment or metadata: %s %v %d %s %v", status, live, revision, description, counts())
				}
				for _, node := range graph.Nodes {
					if node.Type != "email" {
						continue
					}
					var subject string
					if err := env.App.Pool.QueryRow(ctx, `SELECT subject FROM automation_email_versions WHERE email_id=$1 AND version=1`, node.Data.Email.EmailID).Scan(&subject); err != nil || subject != node.Data.Email.Subject {
						t.Fatal("Accept overwrote a source email version")
					}
				}
			}
		})
	}
	mu.Lock()
	defer mu.Unlock()
	for _, payload := range payloads {
		for _, excluded := range []string{list.String(), segment.String(), ws.ID.String(), "PRIVATE_LIST", "PRIVATE_SEGMENT", owner.Email} {
			if strings.Contains(payload, excluded) {
				t.Fatalf("private source leaked: %s", excluded)
			}
		}
	}
}

func TestAgentPost(t *testing.T) {
	for _, settings := range []bool{false, true} {
		t.Run(fmt.Sprintf("site_settings_%t", settings), func(t *testing.T) {
			env := apptest.New(t)
			owner := env.CreateUser(apptest.UserOpts{Email: "owner@agent-post.test"})
			ws := env.CreateWorkspace("Agent post", owner.ID)
			c := env.Client()
			c.SignInDemo(owner.Email)
			ctx := context.Background()
			base := "/workspaces/" + ws.Slug + "/ai"
			exec := func(query string, args ...any) {
				t.Helper()
				if _, err := env.App.Pool.Exec(ctx, query, args...); err != nil {
					t.Fatal(err)
				}
			}
			if settings {
				exec(`INSERT INTO website_settings(workspace_id,status,default_visibility) VALUES($1,'on','paid')`, ws.ID)
			}
			counts := func() [3]int {
				t.Helper()
				var n [3]int
				if err := env.App.Pool.QueryRow(ctx, `SELECT
				(SELECT count(*) FROM website_posts WHERE workspace_id=$1),
				(SELECT count(*) FROM campaigns WHERE workspace_id=$1),
				(SELECT count(*) FROM website_settings WHERE workspace_id=$1)`, ws.ID).Scan(&n[0], &n[1], &n[2]); err != nil {
					t.Fatal(err)
				}
				return n
			}
			before := counts()
			plan := ai.AIAgentPlan{Summary: "Two web-only guides.", Questions: []string{}, Items: []ai.AIAgentPlanItem{
				{Handle: "guide", Kind: "post", Name: "Export guide", Purpose: "Explain exports.", Bundle: "guide", DependsOn: []string{}},
				{Handle: "optional", Kind: "post", Name: "Extra tips", Purpose: "Optional reading.", Bundle: "optional", DependsOn: []string{}},
			}}
			post := ai.AIAgentPost{Title: "Export guide", Excerpt: "Choose your columns", MJML: `<mjml><mj-body><mj-section><mj-column><mj-text>Choose the columns before exporting.</mj-text></mj-column></mj-section></mj-body></mjml>`}
			var phase atomic.Int32
			var proposalHandle string
			model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req map[string]any
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				message, stop := map[string]any{"role": "assistant", "content": "Ready for review."}, "stop"
				if req["tools"] != nil {
					if !strings.Contains(mustJSON(t, req["messages"]), "selected a new web-only post") {
						t.Error("post selection missing from model context")
					}
					call := func(id, name string, args any) any {
						return map[string]any{"id": id, "type": "function", "function": map[string]any{"name": name, "arguments": mustJSON(t, args)}}
					}
					var calls []any
					switch phase.Load() {
					case 0:
						calls = []any{call("plan", "propose_plan", plan)}
					case 1:
						bad := post
						bad.MJML = strings.Replace(post.MJML, "Choose the columns before exporting.", `<script>alert(1)</script>`, 1)
						calls = []any{call("unsafe", "stage_post", map[string]any{"handle": "guide", "post": bad})}
					case 2:
						calls = []any{call("repaired", "stage_post", map[string]any{"handle": "guide", "post": post}), call("optional", "stage_post", map[string]any{"handle": "optional", "post": post})}
					case 4:
						revision := post
						revision.Title, revision.Excerpt = "Export reference", "A concise guide"
						calls = []any{call("revision", "stage_post", map[string]any{"handle": proposalHandle, "post": revision})}
					}
					if len(calls) > 0 {
						message["content"], message["tool_calls"], stop = nil, calls, "tool_calls"
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}}, "usage": map[string]int{"prompt_tokens": 109, "completion_tokens": 47}})
			}))
			defer model.Close()
			c.Do("PUT", base+"/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "post-fixture", "capKind": "none", "inputPrice": 3, "outputPrice": 7}).MustStatus(t, 200)
			c.Do("PUT", base+"/agent/settings", ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}}).MustStatus(t, 204)
			// A website-only editor can draft posts, but cannot publish them
			// or acquire campaign access through the agent.
			editor := env.CreateUser(apptest.UserOpts{Email: "editor@agent-post.test"})
			env.AddMember(ws, editor.ID, "editor")
			role := uuid.Must(uuid.NewV7())
			permissions := []string{"ai:use", "website:view", "website:edit"}
			exec(`INSERT INTO roles(id,workspace_id,key,name,permissions) VALUES($1,$2,'post-editor','Post editor',$3)`, role, ws.ID, permissions)
			exec(`UPDATE workspace_members SET role_id=$1 WHERE workspace_id=$2 AND user_id=$3`, role, ws.ID, editor.ID)
			c = env.Client()
			c.SignInDemo(editor.Email)
			site := &website.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}
			s := &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets,
				AllowPrivate: func() bool { return true }, Jobs: func() *jobs.Client { return env.App.Jobs }, Website: site,
				BrandKit: &brandkit.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}}
			var run ai.AIAgentRunView
			c.Do("POST", base+"/agent/runs", ai.AIAgentBrief{SharingReviewed: true, Brief: "Draft two export guides.", Language: "en", Kind: "post"}).MustStatus(t, 200).JSON(t, &run)
			path := base + "/agent/runs/" + run.ID.String()
			execute := func(p int32) {
				t.Helper()
				phase.Store(p)
				if err := s.ExecuteAgent(ctx, ai.AgentArgs{WorkspaceID: ws.ID, RunID: run.ID, Version: run.Version}); err != nil {
					t.Fatal(err)
				}
				c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
			}
			execute(0)
			c.Do("POST", path+"/actions", ai.AIAgentAction{Action: "build", Version: run.Version, PlanRevision: run.PlanRevision}).MustStatus(t, 204)
			c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
			execute(1)
			if len(run.Artifacts) != 0 || run.ToolCalls != 2 || run.Tokens != 312 || counts() != before {
				t.Fatalf("invalid output was saved or uncharged: %+v", run)
			}
			execute(2)
			execute(3)
			if run.State != "review" || len(run.Artifacts) != 2 || counts() != before || run.Artifacts[0].Post == nil || run.Artifacts[0].Email != nil || run.Artifacts[0].Revision != 1 || !strings.Contains(run.Artifacts[0].HTML, "Choose the columns") || strings.Contains(run.Artifacts[0].HTML, "<script") {
				t.Fatalf("web staging failed: %+v", run)
			}
			keep := ai.AIAgentDecision{Version: run.Version, Decision: "kept", Artifacts: []ai.AIAgentArtifactRevision{{ID: run.Artifacts[0].ID, Revision: 1}}}
			stale := keep
			stale.Artifacts = []ai.AIAgentArtifactRevision{{ID: run.Artifacts[0].ID, Revision: 2}}
			c.Do("POST", path+"/decisions", stale).MustStatus(t, 409)
			exec(`UPDATE roles SET permissions=array_remove(permissions,'website:edit') WHERE id=$1`, role)
			c.Do("GET", path, nil).MustStatus(t, 200)
			c.Do("POST", path+"/decisions", keep).MustStatus(t, 403)
			// Recovery does not need AI access, but still needs website edit.
			exec(`UPDATE roles SET permissions=$2 WHERE id=$1`, role, permissions[1:])
			responses := make(chan *apptest.Response, 2)
			for range 2 {
				go func() { responses <- c.Do("POST", path+"/decisions", keep) }()
			}
			(<-responses).MustStatus(t, 204)
			(<-responses).MustStatus(t, 204)
			c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
			want := before
			want[0]++
			if run.Artifacts[0].ResultID == nil || counts() != want {
				t.Fatal("Keep duplicated the post or changed site settings/campaigns")
			}
			id := *run.Artifacts[0].ResultID
			row, err := env.App.Q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: id})
			if err != nil {
				t.Fatal(err)
			}
			visibility := "public"
			if settings {
				visibility = "paid"
			}
			if row.Status != "draft" || row.Kind != "web" || row.CampaignID != nil || row.PublishAt != nil || row.PublishedAt != nil || row.BodyHtml != "" || row.PublishedMjml != "" || row.Visibility != visibility || row.Version != 1 || row.Title != post.Title || row.Excerpt != post.Excerpt || row.Mjml != run.Artifacts[0].Post.MJML || row.BylineUserID == nil || *row.BylineUserID != editor.ID {
				t.Fatalf("kept post differs or became public: %+v", row)
			}
			v, err := env.App.Q.WebsitePostVersion(ctx, store.WebsitePostVersionParams{PostID: id, Version: 1})
			if err != nil || v.Mjml != row.Mjml || v.SavedBy == nil || *v.SavedBy != editor.ID {
				t.Fatalf("version history lost attribution: %+v %v", v, err)
			}
			feed, err := site.Feed(ctx, &ws, nil, 20)
			if err != nil || len(feed) != 0 {
				t.Fatalf("unpublished draft in feed: %+v %v", feed, err)
			}
			index, err := site.Sitemap(ctx, &ws)
			if err != nil || len(index) != 0 {
				t.Fatalf("unpublished draft indexed: %+v %v", index, err)
			}
			llms, err := site.LLMsPosts(ctx, &ws)
			if err != nil || len(llms) != 0 {
				t.Fatalf("unpublished draft in llms.txt: %+v %v", llms, err)
			}
			c.Do("GET", "/p/"+ws.Slug+"/posts/"+row.Slug, nil).MustStatus(t, 404)
			var n int
			if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='post.created' AND target_id=$1 AND actor_id=$2 AND actor_type='user' AND metadata->>'via'='ai_agent' AND metadata->>'run_id'=$3`, id.String(), editor.ID.String(), run.ID.String()).Scan(&n); err != nil || n != 1 {
				t.Fatalf("provenance missing or duplicated: %d %v", n, err)
			}
			postPath := "/workspaces/" + ws.Slug + "/website/posts/" + id.String()
			c.Do("POST", postPath+"/publish", map[string]any{}).MustStatus(t, 403)
			// The ordinary public-site checks above lazily initialize site
			// settings. Proposal actions must not change that new baseline.
			want = counts()
			// After Keep, even the same run needs a new confirmed plan and
			// an exact-revision Accept. A settings-only edit is stale too.
			exec(`UPDATE roles SET permissions=$2 WHERE id=$1`, role, permissions)
			propose := ai.AIAgentAction{SharingReviewed: true, Action: "propose", Version: run.Version, Handle: "guide", Message: "Make the title and excerpt shorter."}
			c.Do("POST", path+"/actions", propose).MustStatus(t, 204)
			c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
			for round := range 2 {
				if run.State != "ready" || run.PlanRevision == run.AcceptedRevision {
					t.Fatal("proposal skipped plan confirmation")
				}
				proposalHandle = run.Plan.Items[len(run.Plan.Items)-1].Handle
				c.Do("POST", path+"/actions", ai.AIAgentAction{Action: "build", Version: run.Version, PlanRevision: run.PlanRevision}).MustStatus(t, 204)
				c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
				execute(4)
				execute(3)
				var artifact ai.AIAgentArtifact
				for _, a := range run.Artifacts {
					if a.Handle == proposalHandle {
						artifact = a
					}
				}
				if run.State != "review" || artifact.TargetID == nil || *artifact.TargetID != id || artifact.Before == nil || artifact.Post == nil || artifact.Before.Subject != post.Title || artifact.Post.Title != "Export reference" {
					t.Fatalf("post diff unavailable: state=%s reason=%s artifact=%+v", run.State, run.PauseReason, artifact)
				}
				unchanged, err := env.App.Q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: id})
				if err != nil || unchanged.Version != 1 || unchanged.Title != post.Title {
					t.Fatal("building overwrote kept post")
				}
				accept := ai.AIAgentDecision{Version: run.Version, Decision: "accepted", Artifacts: []ai.AIAgentArtifactRevision{{ID: artifact.ID, Revision: artifact.Revision}}}
				if round == 0 {
					exec(`UPDATE website_posts SET seo_title='Human SEO' WHERE id=$1`, id)
					c.Do("POST", path+"/decisions", accept).MustStatus(t, 409)
					c.Do("POST", path+"/actions", ai.AIAgentAction{SharingReviewed: true, Action: "refresh", Version: run.Version, Handle: proposalHandle, Message: "Keep the latest settings and shorten the title."}).MustStatus(t, 204)
					c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
					continue
				}
				c.Do("POST", path+"/decisions", accept).MustStatus(t, 204)
				c.Do("POST", path+"/decisions", accept).MustStatus(t, 204)
				updated, err := env.App.Q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: id})
				if err != nil || updated.Version != 2 || updated.Title != "Export reference" || updated.Excerpt != "A concise guide" || updated.SeoTitle != "Human SEO" || updated.Slug != row.Slug || updated.Visibility != visibility || updated.Status != "draft" || updated.PublishedAt != nil || counts() != want {
					t.Fatalf("post acceptance changed more than the reviewed content: %+v %v", updated, err)
				}
				if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='post.content.save' AND target_id=$1 AND metadata->>'run_id'=$2`, id.String(), run.ID.String()).Scan(&n); err != nil || n != 1 {
					t.Fatalf("duplicate/missing post accept audit: %d %v", n, err)
				}
			}
			c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
			// Human edits survive decision replay, independent Discard and
			// retention of the run that originally created the post.
			exec(`UPDATE website_posts SET title='Human revision',version=2 WHERE id=$1`, id)
			c.Do("POST", path+"/decisions", keep).MustStatus(t, 204)
			var optional uuid.UUID
			for _, a := range run.Artifacts {
				if a.Handle == "optional" {
					optional = a.ID
				}
			}
			c.Do("POST", path+"/decisions", ai.AIAgentDecision{Version: run.Version, Decision: "discarded", Artifacts: []ai.AIAgentArtifactRevision{{ID: optional, Revision: 1}}}).MustStatus(t, 204)
			exec(`UPDATE ai_agent_runs SET expires_at=now()-interval '1 second' WHERE id=$1`, run.ID)
			if _, err := s.Purge(ctx); err != nil {
				t.Fatal(err)
			}
			row, err = env.App.Q.WebsitePostLock(ctx, store.WebsitePostLockParams{WorkspaceID: ws.ID, ID: id})
			if err != nil || row.Title != "Human revision" || row.Version != 2 || row.Status != "draft" {
				t.Fatalf("human revision lost: %+v %v", row, err)
			}
		})
	}
}

func TestAgentExecutionSlots(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@slots.test"})
	editor := env.CreateUser(apptest.UserOpts{Email: "editor@slots.test"})
	ws := env.CreateWorkspace("Execution slots", owner.ID)
	other := env.CreateWorkspace("Another workspace", owner.ID)
	env.AddMember(ws, editor.ID, "editor")
	a, b := env.Client(), env.Client()
	a.SignInDemo(owner.Email)
	b.SignInDemo(editor.Email)
	ctx := context.Background()
	var calls atomic.Int32
	entered, release := make(chan struct{}, 4), make(chan struct{})
	plan := ai.AIAgentPlan{Summary: "Export update.", Questions: []string{}, Items: []ai.AIAgentPlanItem{
		{Handle: "update", Kind: "campaign", Name: "Exports", Purpose: "Explain exports.", Bundle: "update", DependsOn: []string{}},
	}}
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		message, stop := map[string]any{"role": "assistant", "content": "Ready"}, "stop"
		if req["tools"] != nil {
			calls.Add(1)
			entered <- struct{}{}
			<-release
			raw, _ := json.Marshal(plan)
			message["tool_calls"] = []any{map[string]any{"id": "plan", "type": "function", "function": map[string]any{"name": "propose_plan", "arguments": string(raw)}}}
			stop = "tool_calls"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}}, "usage": map[string]int{"prompt_tokens": 117, "completion_tokens": 39}})
	}))
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
		model.Close()
	}()
	for _, slug := range []string{ws.Slug, other.Slug} {
		base := "/workspaces/" + slug + "/ai"
		a.Do("PUT", base+"/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "slot-fixture", "capKind": "none", "inputPrice": 3, "outputPrice": 7}).MustStatus(t, 200)
		a.Do("PUT", base+"/agent/settings", ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}}).MustStatus(t, 204)
	}
	for range 2 {
		workers := river.NewWorkers()
		s := &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, AllowPrivate: func() bool { return true }, Jobs: func() *jobs.Client { return env.App.Jobs }}
		river.AddWorker(workers, &ai.AgentWorker{S: s})
		client, err := river.NewClient(riverpgxv5.New(env.App.Pool), &river.Config{Workers: workers, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			Queues: map[string]river.QueueConfig{jobs.QueueDefault: {MaxWorkers: 3}}, FetchCooldown: 5 * time.Millisecond, FetchPollInterval: 20 * time.Millisecond})
		if err != nil {
			t.Fatal(err)
		}
		if err := client.Start(ctx); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			stop, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			if err := client.StopAndCancel(stop); err != nil {
				t.Error(err)
			}
		})
	}
	start := func(c *apptest.Client, slug string) ai.AIAgentRunView {
		t.Helper()
		var run ai.AIAgentRunView
		c.Do("POST", "/workspaces/"+slug+"/ai/agent/runs", ai.AIAgentBrief{SharingReviewed: true, Brief: "Explain the export workflow.", Language: "en"}).MustStatus(t, 200).JSON(t, &run)
		return run
	}
	waitForCall := func() {
		t.Helper()
		select {
		case <-entered:
		case <-time.After(10 * time.Second):
			t.Fatal("worker did not acquire an execution slot")
		}
	}
	first := start(a, ws.Slug)
	waitForCall()
	second := start(b, ws.Slug)
	waitForCall()
	// Both a third run in this workspace and the same user in another
	// workspace must snooze without reserving tokens or sending HTTP.
	third, fourth := start(a, ws.Slug), start(a, other.Slug)
	for _, check := range []struct {
		slug string
		id   uuid.UUID
		want int64
	}{{ws.Slug, first.ID, 0}, {ws.Slug, second.ID, 0}, {ws.Slug, third.ID, 1}, {other.Slug, fourth.ID, 1}} {
		var view ai.AIAgentRunView
		a.Do("GET", "/workspaces/"+check.slug+"/ai/agent/runs/"+check.id.String(), nil).MustStatus(t, 200).JSON(t, &view)
		if view.QueuePosition != check.want {
			t.Fatalf("queue position %d, want %d", view.QueuePosition, check.want)
		}
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		var attempted int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE kind='ai.agent' AND args->>'runId'=ANY($1::text[]) AND (metadata->>'snoozes')::int>0`, []string{third.ID.String(), fourth.ID.String()}).Scan(&attempted); err != nil {
			t.Fatal(err)
		}
		if attempted == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("concurrent clients did not snooze blocked runs")
		}
		time.Sleep(20 * time.Millisecond)
	}
	var slots, reservations int
	if err := env.App.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM ai_agent_slots), (SELECT count(*) FROM ai_budget_reservations WHERE run_id IS NOT NULL)`).Scan(&slots, &reservations); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 || slots != 2 || reservations != 2 {
		t.Fatalf("execution ceilings bypassed: requests=%d slots=%d reservations=%d", calls.Load(), slots, reservations)
	}
	a.Do("POST", "/workspaces/"+ws.Slug+"/ai/agent/runs/"+third.ID.String()+"/actions", ai.AIAgentAction{Action: "cancel", Version: third.Version}).MustStatus(t, 204)
	close(release)
	deadline = time.Now().Add(15 * time.Second)
	for {
		var ready int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM ai_agent_runs WHERE id=ANY($1::uuid[]) AND state='ready'`, []uuid.UUID{first.ID, second.ID, fourth.ID}).Scan(&ready); err != nil {
			t.Fatal(err)
		}
		if ready == 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("queued runs never resumed after slots were released")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if calls.Load() != 3 {
		t.Fatalf("cancelled queued run spent budget: requests=%d", calls.Load())
	}
	t.Log("two River clients: two active workspace slots, one request per user across workspaces; cancelled queue request never sent")
}

// This middleware terminates a real worker process after a committed staging
// write, before River can acknowledge that job. It is never used in the app.
type agentCrashAfterWrite struct {
	river.MiddlewareDefaults
	pool *pgxpool.Pool
}

func (m *agentCrashAfterWrite) Work(ctx context.Context, job *rivertype.JobRow, next func(context.Context) error) error {
	err := next(ctx)
	var staged bool
	if err == nil && m.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM ai_agent_artifacts)`).Scan(&staged) == nil && staged {
		os.Exit(71)
	}
	return err
}

func TestAgentWorkerProcess(t *testing.T) {
	url := os.Getenv("REPUTECAST_AGENT_WORKER_TEST_DB")
	if url == "" {
		t.Skip("subprocess helper")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	q := store.New(pool)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	aud := audit.New(q, logger)
	var client *jobs.Client
	s := &ai.Service{Pool: pool, Q: q, Audit: aud, AllowPrivate: func() bool { return true },
		Jobs: func() *jobs.Client { return client }, BrandKit: &brandkit.Service{Pool: pool, Q: q, Audit: aud}}
	workers := river.NewWorkers()
	river.AddWorker(workers, &ai.AgentWorker{S: s})
	cfg := &river.Config{Workers: workers, Logger: logger, Queues: map[string]river.QueueConfig{jobs.QueueDefault: {MaxWorkers: 1}},
		JobTimeout: 3 * time.Minute, RescueStuckJobsAfter: 4 * time.Minute, FetchCooldown: 5 * time.Millisecond, FetchPollInterval: 20 * time.Millisecond}
	if os.Getenv("REPUTECAST_AGENT_WORKER_CRASH") == "after_write" {
		cfg.Middleware = []rivertype.Middleware{&agentCrashAfterWrite{pool: pool}}
	}
	client, err = river.NewClient(riverpgxv5.New(pool), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	<-client.Stopped()
}

func TestAgentWorkerRecovery(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@worker.test"})
	ws := env.CreateWorkspace("Worker recovery", owner.ID)
	c := env.Client()
	c.SignInDemo(owner.Email)
	base := "/workspaces/" + ws.Slug + "/ai"
	plan := ai.AIAgentPlan{Summary: "Explain faster exports.", Questions: []string{}, Items: []ai.AIAgentPlanItem{
		{Handle: "update", Kind: "campaign", Name: "Faster exports", Purpose: "Introduce the workflow.", Bundle: "export", DependsOn: []string{}},
	}}
	var requests atomic.Int32
	var stall atomic.Bool
	entered, release := make(chan struct{}, 1), make(chan struct{})
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		message, stop := map[string]any{"role": "assistant", "content": "Draft ready."}, "stop"
		if tools, ok := req["tools"].([]any); ok {
			requests.Add(1)
			if stall.Load() {
				entered <- struct{}{}
				<-release
			}
			tool := tools[0].(map[string]any)["function"].(map[string]any)["name"].(string)
			var args any = plan
			if tool == "stage_campaign" {
				args = map[string]any{"handle": "update", "email": ai.AIDraftEmail{Subject: "A faster export", Preheader: "Fewer steps",
					MJML: `<mjml><mj-body><mj-section><mj-column><mj-text>Export in fewer steps.</mj-text></mj-column></mj-section></mj-body></mjml>`}}
			}
			alreadyStaged := false
			for _, msg := range req["messages"].([]any) {
				msg := msg.(map[string]any)
				if msg["role"] == "tool" {
					var result struct {
						Staged bool `json:"staged"`
					}
					if err := json.Unmarshal([]byte(msg["content"].(string)), &result); err != nil {
						t.Error(err)
					}
					alreadyStaged = alreadyStaged || result.Staged
				}
			}
			if !alreadyStaged {
				raw, _ := json.Marshal(args)
				message["tool_calls"] = []any{map[string]any{"id": "call", "type": "function", "function": map[string]any{"name": tool, "arguments": string(raw)}}}
				stop = "tool_calls"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}}, "usage": map[string]int{"prompt_tokens": 127, "completion_tokens": 31}})
	}))
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
		model.Close()
	}()
	c.Do("PUT", base+"/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "worker-fixture", "capKind": "none", "inputPrice": 3, "outputPrice": 7}).MustStatus(t, 200)
	c.Do("PUT", base+"/agent/settings", ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}}).MustStatus(t, 204)
	ctx := context.Background()
	startWorker := func(mode string) (*exec.Cmd, <-chan error) {
		t.Helper()
		cmd := exec.Command(os.Args[0], "-test.run=^TestAgentWorkerProcess$", "-test.timeout=3m")
		cmd.Env = append(os.Environ(), "REPUTECAST_AGENT_WORKER_TEST_DB="+testdb.URL(env.App.Pool), "REPUTECAST_AGENT_WORKER_CRASH="+mode)
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		t.Cleanup(func() { _ = cmd.Process.Kill() })
		return cmd, done
	}
	var run ai.AIAgentRunView
	path := ""
	startRun := func() {
		t.Helper()
		run = ai.AIAgentRunView{}
		c.Do("POST", base+"/agent/runs", ai.AIAgentBrief{SharingReviewed: true, Brief: "Announce faster exports.", Language: "en"}).MustStatus(t, 200).JSON(t, &run)
		path = base + "/agent/runs/" + run.ID.String()
	}
	waitFor := func(state string) {
		t.Helper()
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
			if run.State == state {
				return
			}
			time.Sleep(30 * time.Millisecond)
		}
		t.Fatalf("worker did not reach %s: %+v", state, run)
	}
	_, dead := startWorker("after_write")
	startRun()
	waitFor("ready")
	c.Do("POST", path+"/actions", ai.AIAgentAction{Action: "build", Version: run.Version, PlanRevision: run.PlanRevision}).MustStatus(t, 204)
	select {
	case err := <-dead:
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 71 {
			t.Fatalf("worker did not die at the write/ack boundary: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("worker did not stage a draft")
	}
	// Age the abandoned River attempt so the replacement's real rescuer
	// handles it now, rather than waiting the production rescue interval.
	var abandoned int64
	if err := env.App.Pool.QueryRow(ctx, `UPDATE river_job SET attempted_at=now()-interval '1 hour' WHERE kind='ai.agent' AND state='running' RETURNING id`).Scan(&abandoned); err != nil {
		t.Fatal(err)
	}
	worker, stopped := startWorker("")
	waitFor("review")
	if len(run.Artifacts) != 1 || run.Artifacts[0].Revision != 1 || run.ToolCalls != 2 || requests.Load() != 3 || run.Tokens != 474 || run.CostMicros != 1794 {
		t.Fatalf("process restart replayed a paid call or draft: %+v, requests=%d", run, requests.Load())
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		var state string
		if err := env.App.Pool.QueryRow(ctx, `SELECT state FROM river_job WHERE id=$1`, abandoned).Scan(&state); err != nil {
			t.Fatal(err)
		}
		if state == "completed" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("rescuer left abandoned job in %s", state)
		}
		time.Sleep(30 * time.Millisecond)
	}
	if requests.Load() != 3 {
		t.Fatal("rescued old version called the model")
	}
	// A process killed after sending a request cannot infer whether it was
	// billed. Its replacement must pause, not silently resend that request.
	stall.Store(true)
	startRun()
	select {
	case <-entered:
	case <-time.After(30 * time.Second):
		t.Fatal("second run never requested its plan")
	}
	_ = worker.Process.Kill()
	<-stopped
	close(release)
	stall.Store(false)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE river_job SET attempted_at=now()-interval '1 hour' WHERE kind='ai.agent' AND state='running'`); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE ai_agent_slots SET expires_at=now()-interval '1 second' WHERE run_id=$1`, run.ID); err != nil {
		t.Fatal(err)
	}
	worker, stopped = startWorker("")
	waitFor("paused")
	if run.PauseReason != "usage_unknown" || !run.Uncertain || run.PlanRevision != 0 || requests.Load() != 4 {
		t.Fatalf("lost response was repeated or treated as free: %+v requests=%d", run, requests.Load())
	}
	c.Do("POST", path+"/actions", ai.AIAgentAction{Action: "continue", Version: run.Version}).MustStatus(t, 409)
	_ = worker.Process.Kill()
	<-stopped
	t.Log("real worker deaths: committed draft revision=1, requests=3, tokens=474, cost_micros=1794; lost response paused with hold, requests=4")
}

func TestAgentToolCapability(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@tools.test"})
	manager := env.CreateUser(apptest.UserOpts{Email: "manager@tools.test"})
	ws := env.CreateWorkspace("Tools", owner.ID)
	env.AddMember(ws, manager.ID, "viewer")
	ctx := context.Background()
	role := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO roles(id,workspace_id,key,name,permissions) VALUES($1,$2,'probe-manager','Probe manager',$3)`, role, ws.ID, []string{"integrations:view", "integrations:manage"}); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE workspace_members SET role_id=$1 WHERE workspace_id=$2 AND user_id=$3`, role, ws.ID, manager.ID); err != nil {
		t.Fatal(err)
	}
	c, m := env.Client(), env.Client()
	c.SignInDemo(owner.Email)
	m.SignInDemo(manager.Email)
	base := "/workspaces/" + ws.Slug + "/ai"
	var mode, calls atomic.Int32
	entered, release := make(chan struct{}, 1), make(chan struct{})
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Tools    []struct{ Function struct{ Name string } }
			Messages []struct{ Role, Content string }
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		calls.Add(1)
		message := map[string]any{"role": "assistant", "content": "Ready"}
		stop := "stop"
		if len(req.Tools) > 0 {
			if len(req.Tools) != 1 || req.Tools[0].Function.Name != "check_tools" || len(req.Messages) != 1 || req.Messages[0].Content != "Call check_tools exactly once with value ready. Do not answer in plain text." {
				t.Errorf("capability probe accepted custom context or tools: %+v", req)
			}
			if mode.Load() == 4 {
				entered <- struct{}{}
				<-release
			}
			if mode.Load() != 1 {
				name, args := "check_tools", `{"value":"ready"}`
				if mode.Load() == 2 {
					name = "send_campaign"
				}
				if mode.Load() == 3 {
					args = `{"value":"wrong"}`
				}
				message["content"] = nil
				message["tool_calls"] = []any{map[string]any{"id": "probe", "type": "function", "function": map[string]any{"name": name, "arguments": args}}}
				stop = "tool_calls"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}}, "usage": map[string]int{"prompt_tokens": 17, "completion_tokens": 11}})
	}))
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		model.Close()
	})
	c.Do("PUT", base+"/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "tool-fixture", "capKind": "none", "inputPrice": 3, "outputPrice": 7}).MustStatus(t, 200)
	c.Do("PUT", base+"/agent/settings", ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}}).MustStatus(t, 204)
	check := func(state string) {
		t.Helper()
		var settings ai.AIAgentSettings
		c.Do("GET", base+"/agent/settings", nil).MustStatus(t, 200).JSON(t, &settings)
		if settings.Capability == nil || settings.Capability.State != state || (state != "untested" && settings.Capability.CheckedAt == nil) {
			t.Fatalf("capability: %+v", settings.Capability)
		}
	}
	check("untested")
	for _, v := range []int32{1, 2, 3, 0} {
		mode.Store(v)
		var got ai.AIAgentCapability
		m.Do("POST", base+"/agent/test", nil).MustStatus(t, 200).JSON(t, &got)
		expected := "unsupported"
		if v == 0 {
			expected = "supported"
		}
		if got.State != expected {
			t.Fatalf("mode %d: %+v", v, got)
		}
		check(expected)
		if v != 0 {
			c.Do("POST", base+"/agent/runs", ai.AIAgentBrief{SharingReviewed: true, Brief: "Draft an export announcement", Language: "en"}).MustStatus(t, 409)
		}
	}
	// A harmless fixed test does not grant this manager paid task access.
	m.Do("POST", base+"/agent/runs", ai.AIAgentBrief{SharingReviewed: true, Brief: "Draft an export announcement", Language: "en"}).MustStatus(t, 403)
	var n, tokens, cost int64
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*), sum(input_tokens+output_tokens), sum(cost_micros) FROM ai_requests WHERE workspace_id=$1 AND action='tools'`, ws.ID).Scan(&n, &tokens, &cost); err != nil || n != 4 || tokens != 112 || cost != 512 {
		t.Fatalf("probe billing: %d %d %d %v", n, tokens, cost, err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE workspace_id=$1 AND action='ai.agent.tools_tested'`, ws.ID).Scan(&n); err != nil || n != 4 {
		t.Fatalf("probe audit: %d %v", n, err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM ai_agent_runs WHERE workspace_id=$1`, ws.ID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("probe created a task: %d %v", n, err)
	}
	// Prices/caps do not change model capability. A model change does, and
	// an old in-flight answer may not certify the replacement model.
	c.Do("PUT", base+"/cap", map[string]any{"kind": "usd", "usd": 25}).MustStatus(t, 200)
	check("supported")
	mode.Store(4)
	done := make(chan *apptest.Response, 1)
	go func() { done <- m.Do("POST", base+"/agent/test", nil) }()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("probe did not start")
	}
	c.Do("PUT", base+"/model", map[string]any{"model": "replacement", "inputPrice": 3, "outputPrice": 7}).MustStatus(t, 200)
	close(release)
	(<-done).MustStatus(t, 409)
	check("untested")
	if calls.Load() != 6 {
		t.Fatalf("unexpected automatic probe/retry: %d", calls.Load())
	}
}

func TestAgentReportContext(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@report.test"})
	ws := env.CreateWorkspace("Report context", owner.ID)
	c := env.Client()
	c.SignInDemo(owner.Email)
	base := "/workspaces/" + ws.Slug + "/ai"
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(t.Context(), query, args...); err != nil {
			t.Fatal(err)
		}
	}
	report := uuid.Must(uuid.NewV7())
	exec(`INSERT INTO campaigns(id,workspace_id,name,subject,mjml,status,started_at,completed_at)
VALUES($1,$2,'PERSON_SENTINEL','PRIVATE_SENTINEL@example.test','SECRET_SENTINEL','sent',now()-interval '81 hours',now()-interval '80 hours')`, report, ws.ID)
	exec(`INSERT INTO campaign_counters(campaign_id,provider_id,mailbox_provider,sent,delivered,opened_human,clicked)
VALUES($1,$2,'MAILBOX_SENTINEL',237,201,79,19)`, report, uuid.New())
	var phase, calls atomic.Int32
	var mu sync.Mutex
	var payloads []string
	var totals map[string]any
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req map[string]any
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Error(err)
			return
		}
		message := map[string]any{"role": "assistant", "content": "Ready."}
		stop := "stop"
		if req["tools"] != nil {
			calls.Add(1)
			mu.Lock()
			payloads = append(payloads, string(raw))
			for _, value := range req["messages"].([]any) {
				m := value.(map[string]any)
				if m["role"] == "tool" && m["tool_call_id"] == "fixed" {
					if err := json.Unmarshal([]byte(m["content"].(string)), &totals); err != nil {
						t.Error(err)
					}
				}
			}
			mu.Unlock()
			if phase.Load() == 0 {
				// An attempted filter is denied; the valid read is counted too.
				message["content"] = nil
				message["tool_calls"] = []any{
					map[string]any{"id": "filter", "type": "function", "function": map[string]any{"name": "read_report", "arguments": `{"filter":"unknown"}`}},
					map[string]any{"id": "fixed", "type": "function", "function": map[string]any{"name": "read_report", "arguments": `{}`}},
				}
				stop = "tool_calls"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}}, "usage": map[string]int{"prompt_tokens": 137, "completion_tokens": 43}})
	}))
	defer model.Close()
	c.Do("PUT", base+"/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "report-fixture", "capKind": "none", "inputPrice": 3, "outputPrice": 7}).MustStatus(t, 200)
	c.Do("PUT", base+"/agent/settings", ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}}).MustStatus(t, 204)
	var run ai.AIAgentRunView
	input := ai.AIAgentBrief{SharingReviewed: true, Brief: "Use the previous campaign totals to inform a new draft.", Language: "en", ReportID: &report}
	c.Do("POST", base+"/agent/runs", input).MustStatus(t, 200).JSON(t, &run)
	if run.Report == nil || run.Report.Sent == nil || *run.Report.Sent != 230 || run.Report.Metrics["humanOpens"] != 70 || run.Report.Metrics["delivered"] != 200 || len(run.Report.Metrics) != 2 {
		t.Fatalf("incorrect fixed totals: %+v", run.Report)
	}
	s := &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, AllowPrivate: func() bool { return true }, Jobs: func() *jobs.Client { return env.App.Jobs }}
	args := ai.AgentArgs{WorkspaceID: ws.ID, RunID: run.ID, Version: run.Version}
	if err := s.ExecuteAgent(t.Context(), args); err != nil {
		t.Fatal(err)
	}
	exec(`UPDATE campaign_counters SET sent=1000,opened_human=999 WHERE campaign_id=$1`, report)
	phase.Store(1)
	c.Do("GET", base+"/agent/runs/"+run.ID.String(), nil).MustStatus(t, 200).JSON(t, &run)
	args.Version = run.Version
	if err := s.ExecuteAgent(t.Context(), args); err != nil {
		t.Fatal(err)
	}
	c.Do("GET", base+"/agent/runs/"+run.ID.String(), nil).MustStatus(t, 200).JSON(t, &run)
	if run.ToolCalls != 2 || len(run.Steps[0].Operations) != 2 || !run.Steps[0].Operations[0].Failed || run.Steps[0].Operations[1].Failed {
		t.Fatal("report tool attempt accounting or filter rejection failed")
	}
	mu.Lock()
	requests := strings.Join(payloads, "\n")
	projected := totals
	mu.Unlock()
	for _, sentinel := range []string{"SENTINEL", report.String()} {
		if strings.Contains(requests, sentinel) {
			t.Fatalf("report payload disclosed %s", sentinel)
		}
	}
	if projected["sent"] != float64(230) || calls.Load() != 2 {
		t.Fatal("model did not receive the immutable projection")
	}
	metrics := projected["metrics"].(map[string]any)
	if len(metrics) != 2 || metrics["humanOpens"] != float64(70) || metrics["delivered"] != float64(200) {
		t.Fatalf("model received incorrect or unselected metrics: %v", metrics)
	}
	var second ai.AIAgentRunView
	c.Do("POST", base+"/agent/runs", input).MustStatus(t, 200).JSON(t, &second)
	if *second.Report.Sent != 230 || !second.Report.AsOf.Equal(run.Report.AsOf) || second.ExpiresAt.Sub(second.CreatedAt) != 30*24*time.Hour {
		t.Fatal("another run refreshed the report or changed the run retention")
	}
	hidden, allowed := uuid.New(), uuid.New()
	exec(`UPDATE campaigns SET audience=jsonb_build_object('lists',jsonb_build_array($1::text)) WHERE id=$2`, hidden.String(), report)
	exec(`UPDATE workspace_members SET scope=jsonb_build_object('lists',jsonb_build_array($1::text)) WHERE workspace_id=$2 AND user_id=$3`, allowed.String(), ws.ID, owner.ID)
	if err := s.ExecuteAgent(t.Context(), ai.AgentArgs{WorkspaceID: ws.ID, RunID: second.ID, Version: second.Version}); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatal("revoked report scope reached the model")
	}
	c.Do("GET", base+"/agent/runs/"+second.ID.String(), nil).MustStatus(t, 403)
	exec(`UPDATE workspace_members SET scope='{}' WHERE workspace_id=$1 AND user_id=$2`, ws.ID, owner.ID)
	c.Do("GET", base+"/agent/runs/"+second.ID.String(), nil).MustStatus(t, 200).JSON(t, &second)
	if second.State != "paused" || second.PauseReason != "access_changed" {
		t.Fatalf("revoked scope was not persisted: %+v", second)
	}
	exec(`UPDATE report_model_snapshots SET created_at=now()-interval '31 days' WHERE campaign_id=$1`, report)
	c.Do("POST", base+"/agent/runs", input).MustStatus(t, 409)
	c.Do("POST", base+"/agent/runs/"+run.ID.String()+"/actions", ai.AIAgentAction{Action: "cancel", Version: run.Version}).MustStatus(t, 204)
	exec(`UPDATE ai_agent_runs SET closed_at=now()-interval '8 days' WHERE id=$1`, run.ID)
	if _, err := s.Purge(t.Context()); err != nil {
		t.Fatal(err)
	}
	var closed ai.AIAgentRunView
	c.Do("GET", base+"/agent/runs/"+run.ID.String(), nil).MustStatus(t, 200).JSON(t, &closed)
	if closed.Report != nil || closed.Brief == nil {
		t.Fatal("closure retained report content or purged the brief early")
	}
	exec(`UPDATE ai_agent_runs SET expires_at=now()-interval '1 second' WHERE workspace_id=$1`, ws.ID)
	if _, err := s.Purge(t.Context()); err != nil {
		t.Fatal(err)
	}
	var expired ai.AIAgentRunView
	c.Do("GET", base+"/agent/runs/"+run.ID.String(), nil).MustStatus(t, 200).JSON(t, &expired)
	if expired.State != "expired" || expired.Report != nil || expired.Brief != nil || len(expired.Steps) != 0 {
		t.Fatal("expired report context survived")
	}
}

func TestAgentDiscountPool(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@pool.test"})
	ws := env.CreateWorkspace("Discount context", owner.ID)
	c := env.Client()
	c.SignInDemo(owner.Email)
	base := "/workspaces/" + ws.Slug + "/ai"
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(t.Context(), query, args...); err != nil {
			t.Fatal(err)
		}
	}
	pool := uuid.Must(uuid.NewV7())
	exec(`INSERT INTO coupon_pools(id,workspace_id,name,slug,source) VALUES($1,$2,'PRIVATE_POOL_LABEL','private_pool','upload')`, pool, ws.ID)
	var phase atomic.Int32
	var mu sync.Mutex
	var payloads []string
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req struct {
			Tools    []any
			Messages []struct{ Content any }
		}
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Error(err)
			return
		}
		message := map[string]any{"role": "assistant", "content": "Ready."}
		stop := "stop"
		if len(req.Tools) > 0 {
			mu.Lock()
			payloads = append(payloads, string(raw))
			mu.Unlock()
			name := "propose_plan"
			var args any = ai.AIAgentPlan{Summary: "Draft the selected offer.", Questions: []string{}, Items: []ai.AIAgentPlanItem{{Handle: "offer", Kind: "campaign", Name: "Offer draft", Purpose: "Explain the offer.", Bundle: "offer", DependsOn: []string{}}}}
			if phase.Load() == 1 {
				var source struct {
					Source string
					Links  []string
					Tokens []string
				}
				for _, item := range req.Messages {
					text, _ := item.Content.(string)
					start, end := strings.Index(text, "<editorial_context>\n"), strings.Index(text, "\n</editorial_context>")
					if start >= 0 && end > start {
						if err := json.Unmarshal([]byte(text[start+len("<editorial_context>\n"):end]), &source); err != nil {
							t.Error(err)
						}
					}
				}
				if len(source.Links) != 1 || len(source.Tokens) != 1 || !strings.Contains(source.Source, source.Tokens[0]) {
					t.Errorf("selected code and redemption link must remain separate capabilities: %+v", source)
					return
				}
				name = "stage_campaign"
				args = map[string]any{"handle": "offer", "email": ai.AIDraftEmail{Subject: "Your next order", Preheader: "An offer to review", MJML: `<mjml><mj-body><mj-section><mj-column><mj-text>` + strings.Join(source.Tokens, " ") + `</mj-text><mj-button href="` + source.Links[0] + `">Use the offer</mj-button></mj-column></mj-section></mj-body></mjml>`}}
			}
			if phase.Load() != 2 {
				encoded, _ := json.Marshal(args)
				message["content"] = nil
				message["tool_calls"] = []any{map[string]any{"id": "selected", "type": "function", "function": map[string]any{"name": name, "arguments": string(encoded)}}}
				stop = "tool_calls"
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}}, "usage": map[string]int{"prompt_tokens": 137, "completion_tokens": 43}})
	}))
	defer model.Close()
	c.Do("PUT", base+"/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "pool-fixture", "capKind": "none", "inputPrice": 1, "outputPrice": 2}).MustStatus(t, 200)
	c.Do("PUT", base+"/agent/settings", ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}}).MustStatus(t, 204)
	input := ai.AIAgentBrief{SharingReviewed: true, Brief: "Use PRIVATE_POOL_LABEL for a new email.", Language: "en", PoolID: &pool}
	var run ai.AIAgentRunView
	c.Do("POST", base+"/agent/runs", input).MustStatus(t, 200).JSON(t, &run)
	if run.Pool == nil || run.Pool.Free != 0 || run.Pool.Name != "PRIVATE_POOL_LABEL" {
		t.Fatalf("human snapshot: %+v", run.Pool)
	}
	// Inventory is advisory, not reserved by planning or Keep. The encrypted
	// payload must remain unread even when there is a code to allocate.
	exec(`INSERT INTO coupon_codes(id,workspace_id,pool_id,code_sealed,code_hash) VALUES($1,$2,$3,$4,$5)`, uuid.Must(uuid.NewV7()), ws.ID, pool, []byte("PRIVATE_CODE_PAYLOAD"), []byte("private-hash"))
	s := &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, AllowPrivate: func() bool { return true }, Jobs: func() *jobs.Client { return env.App.Jobs }, BrandKit: &brandkit.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}, Campaigns: &campaigns.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}}
	path := base + "/agent/runs/" + run.ID.String()
	step := func() {
		t.Helper()
		if err := s.ExecuteAgent(t.Context(), ai.AgentArgs{WorkspaceID: ws.ID, RunID: run.ID, Version: run.Version}); err != nil {
			t.Fatal(err)
		}
		c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	}
	step()
	c.Do("POST", path+"/actions", ai.AIAgentAction{Action: "build", Version: run.Version, PlanRevision: run.PlanRevision}).MustStatus(t, 204)
	c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &run)
	phase.Store(1)
	step()
	phase.Store(2)
	step()
	if run.State != "review" || len(run.Artifacts) != 1 || run.Artifacts[0].Email == nil {
		t.Fatalf("staging failed: state=%s reason=%s steps=%+v", run.State, run.PauseReason, run.Steps)
	}
	for _, token := range []string{"{{ coupons.private_pool.code }}", "{{ coupons.private_pool.url }}"} {
		if strings.Count(run.Artifacts[0].Email.MJML, token) != 1 {
			t.Fatalf("token not preserved once: %s", token)
		}
	}
	mu.Lock()
	requests := strings.Join(payloads, "\n")
	mu.Unlock()
	for _, sentinel := range []string{"PRIVATE_POOL_LABEL", "private_pool", "PRIVATE_CODE_PAYLOAD", pool.String(), owner.Email} {
		if strings.Contains(requests, sentinel) {
			t.Fatalf("private pool context reached model: %s", sentinel)
		}
	}
	if run.Pool.Free != 0 {
		t.Fatal("planning silently refreshed the selected inventory snapshot")
	}
	decision := ai.AIAgentDecision{Version: run.Version, Decision: "kept", Artifacts: []ai.AIAgentArtifactRevision{{ID: run.Artifacts[0].ID, Revision: run.Artifacts[0].Revision}}}
	exec(`UPDATE coupon_pools SET status='archived' WHERE id=$1`, pool)
	c.Do("POST", path+"/decisions", decision).MustStatus(t, 409)
	c.Do("GET", path, nil).MustStatus(t, 200) // Recovery remains readable.
	exec(`UPDATE coupon_pools SET status='active' WHERE id=$1`, pool)
	exec(`UPDATE coupon_pools SET updated_at=updated_at+interval '1 second' WHERE id=$1`, pool)
	c.Do("POST", path+"/decisions", decision).MustStatus(t, 409)
	exec(`UPDATE coupon_pools SET updated_at=$2 WHERE id=$1`, pool, run.Pool.UpdatedAt)
	c.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
	c.Do("POST", path+"/decisions", decision).MustStatus(t, 204)
	var products, codes, assigned, jobs int
	if err := env.App.Pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM campaigns WHERE workspace_id=$1 AND status='draft'), (SELECT count(*) FROM coupon_codes WHERE pool_id=$2), (SELECT count(*) FROM coupon_codes WHERE pool_id=$2 AND assigned_at IS NOT NULL), (SELECT count(*) FROM coupon_batches WHERE pool_id=$2)`, ws.ID, pool).Scan(&products, &codes, &assigned, &jobs); err != nil {
		t.Fatal(err)
	}
	if products != 1 || codes != 1 || assigned != 0 || jobs != 0 {
		t.Fatalf("keep effects: drafts=%d codes=%d assigned=%d batches=%d", products, codes, assigned, jobs)
	}
	other := env.CreateWorkspace("Other discount", owner.ID)
	c.Do("POST", "/workspaces/"+other.Slug+"/ai/agent/runs", input).MustStatus(t, 404)
}
