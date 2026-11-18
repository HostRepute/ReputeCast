package ai_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/migrations"
)

// Model accounts, models and per-feature defaults (assistant brief D3,
// migration 03630). B1's operations to add accounts and models come later;
// these tests add them with SQL, as those operations will store them.

// modelServer is an OpenAI-compatible server that answers every request with
// a campaign draft (and, for the agent, a plan), keeping the key and the
// model of each. hold makes requests wait for release.
type modelServer struct {
	srv     *httptest.Server
	mu      sync.Mutex
	keys    []string
	models  []string
	hold    atomic.Bool
	release chan struct{}
	once    sync.Once
}

// free lets held requests answer, once.
func (m *modelServer) free() { m.once.Do(func() { close(m.release) }) }

func newModelServer(t *testing.T) *modelServer {
	m := &modelServer{release: make(chan struct{})}
	plan := ai.AIAgentPlan{Summary: "Announce the improved export workflow.", Questions: []string{}, Items: []ai.AIAgentPlanItem{{Handle: "announcement", Kind: "campaign",
		Name: "Export improvements", Purpose: "Explain faster exports.", Bundle: "launch", DependsOn: []string{}}}}
	m.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model string `json:"model"`
			Tools []struct {
				Function struct{ Name string } `json:"function"`
			} `json:"tools"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		m.mu.Lock()
		m.keys = append(m.keys, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		m.models = append(m.models, req.Model)
		m.mu.Unlock()
		if m.hold.Load() {
			<-m.release
		}
		answer, _ := json.Marshal(ai.AIDraftEmail{Subject: "Export update", Preheader: "What changed", MJML: `<mjml><mj-body><mj-section><mj-column><mj-text>Exports are easier.</mj-text></mj-column></mj-section></mj-body></mjml>`})
		message := map[string]any{"role": "assistant", "content": string(answer)}
		stop := "stop"
		if len(req.Tools) > 0 {
			// The tool-calling check, or a plan for the agent.
			name, args := "propose_plan", any(plan)
			if req.Tools[0].Function.Name == "check_tools" {
				name, args = "check_tools", map[string]string{"value": "ready"}
			}
			raw, _ := json.Marshal(args)
			message["content"] = nil
			message["tool_calls"] = []any{map[string]any{"id": "call1", "type": "function", "function": map[string]any{"name": name, "arguments": string(raw)}}}
			stop = "tool_calls"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}},
			"usage": map[string]int{"prompt_tokens": 113, "completion_tokens": 29}})
	}))
	t.Cleanup(func() {
		m.free()
		m.srv.Close()
	})
	return m
}

func (m *modelServer) got() (keys, models []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.keys), slices.Clone(m.models)
}

// registry is a workspace whose first account (on first) was connected
// through the API, with SQL helpers for the accounts and models B1 adds.
type registry struct {
	t       *testing.T
	env     *apptest.Env
	ws      store.Workspace
	c       *apptest.Client
	base    string
	owner   uuid.UUID
	account uuid.UUID // the first account
	model   uuid.UUID // its model, every feature's default
}

func newRegistry(t *testing.T, first *modelServer, connect map[string]any) *registry {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@registry.test", Name: "Registry Owner"})
	ws := env.CreateWorkspace("Registry", owner.ID)
	r := &registry{t: t, env: env, ws: ws, c: env.Client(), base: "/workspaces/" + ws.Slug + "/ai", owner: owner.ID}
	r.c.SignInDemo(owner.Email)
	in := map[string]any{"provider": "compatible", "baseUrl": first.srv.URL, "model": "model-a", "capKind": "none", "inputPrice": 1, "outputPrice": 2}
	for k, v := range connect {
		in[k] = v
	}
	r.c.Do("PUT", r.base+"/connection", in).MustStatus(t, 200)
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT a.id, m.id FROM ai_accounts a JOIN ai_models m ON m.account_id = a.id WHERE a.workspace_id = $1`, ws.ID).Scan(&r.account, &r.model); err != nil {
		t.Fatal(err)
	}
	return r
}

func (r *registry) exec(query string, args ...any) {
	r.t.Helper()
	if _, err := r.env.App.Pool.Exec(context.Background(), query, args...); err != nil {
		r.t.Fatal(err)
	}
}

// addAccount adds an OpenAI-compatible account, its key sealed for it.
func (r *registry) addAccount(baseURL, key string) uuid.UUID {
	r.t.Helper()
	id := uuid.Must(uuid.NewV7())
	var sealed []byte
	if key != "" {
		var err error
		if sealed, err = r.env.App.Secrets.SealString(key, "ai:"+id.String()+":key"); err != nil {
			r.t.Fatal(err)
		}
	}
	r.exec(`INSERT INTO ai_accounts (id, workspace_id, provider, base_url, key_sealed, connected_at, created_at, updated_at)
		VALUES ($1, $2, 'compatible', $3, $4, now(), now(), now())`, id, r.ws.ID, baseURL, sealed)
	return id
}

func (r *registry) addModel(account uuid.UUID, model string, input, output float64) uuid.UUID {
	r.t.Helper()
	id := uuid.Must(uuid.NewV7())
	r.exec(`INSERT INTO ai_models (id, workspace_id, account_id, model, input_price, output_price, price_known, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, true, now(), now())`, id, r.ws.ID, account, model, input, output)
	return id
}

func (r *registry) setDefault(feature string, model uuid.UUID) {
	r.t.Helper()
	r.exec(`UPDATE ai_feature_models SET model_id = $3 WHERE workspace_id = $1 AND feature = $2`, r.ws.ID, feature, model)
}

// waitForLock waits until n requests wait for a lock the test holds.
func (r *registry) waitForLock(n int) {
	r.t.Helper()
	for range 200 {
		var got int
		if err := r.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&got); err != nil {
			r.t.Fatal(err)
		}
		if got >= n {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	r.t.Fatalf("%d requests aren't waiting for the AI configuration lock", n)
}

func (r *registry) draft() *apptest.Response {
	return r.c.Do("POST", r.base+"/drafts/campaigns", ai.AICampaignBriefInput{SharingReviewed: true, Brief: "Announce easier exports.", Name: "Exports", Language: "en"})
}

// A request is priced with its feature's model, not the workspace's first
// model or the account's: reserved, logged and shown at that model's prices.
func TestAIFeatureModelPrices(t *testing.T) {
	srv := newModelServer(t)
	r := newRegistry(t, srv, nil)
	other := r.addModel(r.account, "model-b", 3, 7)
	check := func(model string, id uuid.UUID, input, output float64) {
		t.Helper()
		var p ai.AICampaignProposal
		r.draft().MustStatus(t, 200).JSON(t, &p)
		if _, models := srv.got(); models[len(models)-1] != model {
			t.Fatalf("the request went to %q, want %q", models[len(models)-1], model)
		}
		var reserved, logged string
		var modelID, logModelID *uuid.UUID
		var in, out float64
		var cost int64
		if err := r.env.App.Pool.QueryRow(context.Background(), `SELECT b.model, b.model_id, b.input_price, b.output_price, q.model, q.model_id, q.cost_micros
			FROM ai_budget_reservations b JOIN ai_requests q ON q.id = b.id WHERE b.id = $1`, p.ID).Scan(&reserved, &modelID, &in, &out, &logged, &logModelID, &cost); err != nil {
			t.Fatal(err)
		}
		want := int64(113*input + 29*output)
		if reserved != model || logged != model || modelID == nil || *modelID != id || logModelID == nil || *logModelID != id || in != input || out != output || cost != want {
			t.Fatalf("bound %s %v at %v/%v, logged %s %v costing %d; want %s %s at %v/%v costing %d", reserved, modelID, in, out, logged, logModelID, cost, model, id, input, output, want)
		}
		if p.Sent.Model != model || p.Sent.CostMicros != want {
			t.Fatalf("the proposal says %s cost %d, want %s %d", p.Sent.Model, p.Sent.CostMicros, model, want)
		}
	}
	check("model-a", r.model, 1, 2)
	r.setDefault("draft", other)
	check("model-b", other, 3, 7)
}

// Reservations on all the workspace's accounts take the same lock and share
// its monthly cap: of two requests racing for the last of it on two accounts,
// one reserves and the other is refused before reaching its server. (Two
// requests of one feature always go to the same model: they race the saved
// models' tests, each on its own account.)
func TestAIReservationsSpanAccounts(t *testing.T) {
	first, second := newModelServer(t), newModelServer(t)
	defer first.free()
	defer second.free()
	r := newRegistry(t, first, map[string]any{"capKind": "tokens", "capTokens": 10000, "inputPrice": 3, "outputPrice": 7})
	model := r.addModel(r.addAccount(second.srv.URL, ""), "model-b", 3, 7)
	// The connection's test used 142 tokens. A test reserves about 4,300
	// (its estimated input and 16 out): 6,000 left cover one, not two.
	r.exec(`UPDATE ai_workspace_settings SET cap_tokens = 6142 WHERE workspace_id = $1`, r.ws.ID)
	first.hold.Store(true)
	second.hold.Store(true)
	ctx := context.Background()
	// Hold the AI configuration lock until both requests wait for it.
	tx, err := r.env.App.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT 1 FROM ai_workspace_settings WHERE workspace_id = $1 FOR UPDATE`, r.ws.ID); err != nil {
		t.Fatal(err)
	}
	done := make(chan *apptest.Response, 2)
	for _, id := range []uuid.UUID{r.model, model} {
		go func() { done <- r.c.Do("POST", r.base+"/test", map[string]any{"modelId": id}) }()
	}
	r.waitForLock(2)
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	// One reserves and waits at its server; the other is refused meanwhile.
	var refused *apptest.Response
	select {
	case refused = <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("neither request was refused")
	}
	if p := refused.MustStatus(t, 409).Problem(t); p["code"] != "ai_reservation_cap" {
		t.Fatalf("the losing request: %v", p)
	}
	first.free()
	second.free()
	(<-done).MustStatus(t, 200)
	firstKeys, _ := first.got()
	secondKeys, _ := second.got()
	// The first server also answered the connection's test.
	if len(firstKeys)+len(secondKeys) != 2 {
		t.Fatalf("requests reached the servers: %d and %d", len(firstKeys)-1, len(secondKeys))
	}
	var tests int64
	if err := r.env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM ai_budget_reservations WHERE workspace_id = $1 AND model_id IS NOT NULL`, r.ws.ID).Scan(&tests); err != nil || tests != 1 {
		t.Fatalf("%d reservations of the saved models (%v), want 1", tests, err)
	}
}

// Each account's key is sealed for that account ("ai:<account>:key") and
// goes only to that account's server.
func TestAIAccountKeys(t *testing.T) {
	first, second := newModelServer(t), newModelServer(t)
	r := newRegistry(t, first, map[string]any{"apiKey": "sk-first-0001"})
	account := r.addAccount(second.srv.URL, "sk-second-0002")
	model := r.addModel(account, "model-b", 1, 2)
	r.setDefault("draft", model)
	r.draft().MustStatus(t, 200)
	r.setDefault("draft", r.model)
	r.draft().MustStatus(t, 200)
	firstKeys, _ := first.got()
	secondKeys, _ := second.got()
	if !slices.Equal(secondKeys, []string{"sk-second-0002"}) {
		t.Fatalf("the second account's server got keys %v", secondKeys)
	}
	if len(firstKeys) != 2 || slices.ContainsFunc(firstKeys, func(k string) bool { return k != "sk-first-0001" }) {
		t.Fatalf("the first account's server got keys %v", firstKeys)
	}
	if r.account == r.ws.ID {
		t.Fatal("a new workspace's first account has the workspace's id")
	}
	var sealed []byte
	if err := r.env.App.Pool.QueryRow(context.Background(), `SELECT key_sealed FROM ai_accounts WHERE id = $1`, r.account).Scan(&sealed); err != nil {
		t.Fatal(err)
	}
	if key, err := r.env.App.Secrets.OpenString(sealed, "ai:"+r.account.String()+":key"); err != nil || key != "sk-first-0001" {
		t.Fatalf("the first key doesn't open as its account's: %v", err)
	}
	if _, err := r.env.App.Secrets.OpenString(sealed, "ai:"+r.ws.ID.String()+":key"); err == nil {
		t.Fatal("the first key opens as the workspace's")
	}
}

// A feature whose model's account is disconnected waits for it: no other
// model answers instead.
func TestAIDisconnectedDefault(t *testing.T) {
	first, second := newModelServer(t), newModelServer(t)
	r := newRegistry(t, first, nil)
	account := r.addAccount(second.srv.URL, "")
	r.setDefault("draft", r.addModel(account, "model-b", 1, 2))
	r.exec(`UPDATE ai_accounts SET disconnected_at = now() WHERE id = $1`, account)
	before, _ := first.got()
	if p := r.draft().MustStatus(t, 409).Problem(t); p["code"] != "ai_not_configured" {
		t.Fatalf("a disconnected default: %v", p)
	}
	after, _ := first.got()
	if keys, _ := second.got(); len(keys) != 0 || len(after) != len(before) {
		t.Fatal("a model answered for a feature whose model is disconnected")
	}
	r.setDefault("draft", r.model)
	r.draft().MustStatus(t, 200)
}

// A run keeps the model it started with when the agent's default changes;
// a new run takes the new default.
func TestAgentRunKeepsItsModel(t *testing.T) {
	first, second := newModelServer(t), newModelServer(t)
	r := newRegistry(t, first, map[string]any{"model": "agent-a", "inputPrice": 3, "outputPrice": 7})
	r.c.Do("PUT", r.base+"/agent/settings", ai.AIAgentSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}}).MustStatus(t, 204)
	env := r.env
	s := &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets,
		AllowPrivate: func() bool { return true }, Jobs: func() *jobs.Client { return env.App.Jobs },
		BrandKit: &brandkit.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}}
	ctx := context.Background()
	start := func(key string) ai.AIAgentRunView {
		t.Helper()
		var run ai.AIAgentRunView
		r.c.Do("POST", r.base+"/agent/runs", ai.AIAgentBrief{SharingReviewed: true, Brief: "Announce easier exports.", Language: "en", RequestKey: key}).MustStatus(t, 200).JSON(t, &run)
		return run
	}
	check := func(run ai.AIAgentRunView, model uuid.UUID, name string) {
		t.Helper()
		if err := s.ExecuteAgent(ctx, ai.AgentArgs{WorkspaceID: r.ws.ID, RunID: run.ID, Version: run.Version}); err != nil {
			t.Fatal(err)
		}
		var runModel, reserved *uuid.UUID
		var named string
		if err := env.App.Pool.QueryRow(ctx, `SELECT r.model_id, r.model, (SELECT b.model_id FROM ai_budget_reservations b WHERE b.run_id = r.id ORDER BY b.created_at DESC LIMIT 1)
			FROM ai_agent_runs r WHERE r.id = $1`, run.ID).Scan(&runModel, &named, &reserved); err != nil {
			t.Fatal(err)
		}
		if runModel == nil || *runModel != model || named != name || reserved == nil || *reserved != model {
			t.Fatalf("run on %v (%s), reserved on %v; want %s (%s)", runModel, named, reserved, model, name)
		}
	}
	run := start("pinned-1")
	account := r.addAccount(second.srv.URL, "")
	other := r.addModel(account, "agent-b", 3, 7)
	r.setDefault("agent", other)
	check(run, r.model, "agent-a")
	if _, models := second.got(); len(models) != 0 {
		t.Fatal("a running task moved to the agent's new default model")
	}
	if _, models := first.got(); models[len(models)-1] != "agent-a" {
		t.Fatalf("the run's request went to %v", models)
	}
	check(start("pinned-2"), other, "agent-b")
	if _, models := second.got(); !slices.Equal(models, []string{"agent-b"}) {
		t.Fatalf("the new run's requests: %v", models)
	}
}

// Migration 03630 makes a workspace's one connection its first account and
// model, every feature's default, and keeps its key, cap, prices, switches,
// usage, agent runs and tool test.
func TestAIRegistryMigration(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@legacy.test", Name: "Legacy Owner"})
	ws := env.CreateWorkspace("Legacy", owner.ID)
	c := env.Client()
	c.SignInDemo(owner.Email)
	base := "/workspaces/" + ws.Slug + "/ai"
	keys := newKeyServer(t)
	ctx := context.Background()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	p, err := goose.NewProvider(goose.DialectPostgres, stdlib.OpenDBFromPool(env.App.Pool), migrations.FS, goose.WithAllowOutofOrder(true))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
	// 03631 (personal keys), 03640 (the assistant), 03650 (its MCP
	// servers), 03651 (their hardening) and 03652 (the hourly usage) build
	// on 03630's tables: they go down first and up after.
	const version, personal, assistant, mcp, hardening, hours = 3630, 3631, 3640, 3650, 3651, 3652
	down := func() {
		t.Helper()
		for _, v := range []int64{hours, hardening, mcp, assistant, personal, version} {
			if _, err := p.ApplyVersion(ctx, v, false); err != nil {
				t.Fatal(err)
			}
		}
	}
	up := func() {
		t.Helper()
		for _, v := range []int64{version, personal, assistant, mcp, hardening, hours} {
			if _, err := p.ApplyVersion(ctx, v, true); err != nil {
				t.Fatal(err)
			}
		}
	}
	down()
	// The connection as the app kept it before.
	sealed, err := env.App.Secrets.SealString("sk-legacy-0042", "ai:"+ws.ID.String()+":key")
	if err != nil {
		t.Fatal(err)
	}
	exec(`INSERT INTO ai_connections (workspace_id, provider, label, base_url, model, model_name, image_input, key_sealed, key_added_by, key_added_at,
		connected_by, connected_at, status, last_test_at, last_test_ok, last_test_ms, cap_kind, cap_usd, cap_tokens, cap_set_by, cap_set_at,
		input_price, output_price, price_known, features_off, notice_month, cap_notice_month, updated_by, updated_at)
		VALUES ($1, 'compatible', 'Office model', $2, 'gemma3:27b', 'Gemma 3 27B', false, $3, $4, now() - interval '2 days',
		$4, now() - interval '2 days', 'ok', now() - interval '2 days', true, 840, 'tokens', 25, 500000, $4, now() - interval '1 day',
		3, 7, true, '{translate}', '2026-10', '2026-09', $4, now() - interval '1 day')`, ws.ID, keys.srv.URL+"/v1", sealed, owner.ID)
	exec(`INSERT INTO ai_requests (id, workspace_id, user_id, feature, provider, model, input_tokens, output_tokens, cost_micros, outcome, created_at)
		VALUES ($1, $2, $3, 'rewrite', 'compatible', 'gemma3:27b', 1000, 200, 4400, 'accepted', now())`, uuid.Must(uuid.NewV7()), ws.ID, owner.ID)
	run := uuid.Must(uuid.NewV7())
	exec(`INSERT INTO ai_agent_runs (id, workspace_id, user_id, model, state, tool_limit, token_limit, cost_limit_micros, created_at, updated_at, expires_at)
		VALUES ($1, $2, $3, 'gemma3:27b', 'review', 40, 300000, 2000000, now(), now(), now() + interval '30 days')`, run, ws.ID, owner.ID)
	// The tool test's hash, as the agent computed it from the connection.
	var model struct{ Provider, Model, BaseURL, Deployment, APIVersion string }
	var key []byte
	var connectedAt time.Time
	var disconnectedAt *time.Time
	var demo bool
	if err := env.App.Pool.QueryRow(ctx, `SELECT provider, model, base_url, deployment, api_version, key_sealed, connected_at, disconnected_at, demo
		FROM ai_connections WHERE workspace_id = $1`, ws.ID).Scan(&model.Provider, &model.Model, &model.BaseURL, &model.Deployment, &model.APIVersion,
		&key, &connectedAt, &disconnectedAt, &demo); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal([]any{model, key, connectedAt, disconnectedAt, demo})
	hash := sha256.Sum256(raw)
	exec(`INSERT INTO ai_agent_capabilities (workspace_id, connection_hash, supported, checked_at) VALUES ($1, $2, true, now())`, ws.ID, hex.EncodeToString(hash[:]))

	up()
	var account, modelID uuid.UUID
	var defaults int
	var runModel, capability *uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT a.id, m.id,
			(SELECT count(*) FROM ai_feature_models f WHERE f.workspace_id = a.workspace_id AND f.model_id = m.id),
			(SELECT r.model_id FROM ai_agent_runs r WHERE r.id = $2),
			(SELECT c.model_id FROM ai_agent_capabilities c WHERE c.workspace_id = a.workspace_id)
		FROM ai_accounts a JOIN ai_models m ON m.account_id = a.id WHERE a.workspace_id = $1`, ws.ID, run).Scan(&account, &modelID, &defaults, &runModel, &capability); err != nil {
		t.Fatal(err)
	}
	if account != ws.ID || defaults != 11 || runModel == nil || *runModel != modelID || capability == nil || *capability != modelID {
		t.Fatalf("account %s (workspace %s), %d defaults, run on %v, tool test for %v; model %s", account, ws.ID, defaults, runModel, capability, modelID)
	}
	var s ai.AISettings
	c.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &s)
	v := s.Connection
	if !s.Connected || v == nil || v.Provider != "compatible" || v.Label != "Office model" || v.Model != "gemma3:27b" || v.ModelName != "Gemma 3 27B" || v.ImageInput || !v.Key.Set ||
		v.Cap.Kind != "tokens" || v.Cap.Tokens != 500000 || v.Cap.SetBy != "Legacy Owner" || v.Prices.Input != 3 || v.Prices.Output != 7 || !v.Prices.Known ||
		v.LastTest == nil || v.LastTest.Ms != 840 {
		t.Fatalf("the connection after the migration: %+v", v)
	}
	if s.Usage.Requests != 1 || s.Usage.Tokens != 1200 || s.Usage.CostMicros != 4400 {
		t.Fatalf("usage after the migration: %+v", s.Usage)
	}
	for _, f := range s.Features {
		if (f.Key == "translate" || f.Key == "alt_text") == f.On {
			t.Fatalf("features after the migration: %+v", s.Features)
		}
	}
	var agent ai.AIAgentSettings
	c.Do("GET", base+"/agent/settings", nil).MustStatus(t, 200).JSON(t, &agent)
	if agent.Capability == nil || agent.Capability.State != "supported" {
		t.Fatalf("the tool test after the migration: %+v", agent.Capability)
	}
	// The key sealed as the workspace's opens as the account's.
	var test ai.AITestResult
	c.Do("POST", base+"/test", map[string]any{}).MustStatus(t, 200).JSON(t, &test)
	if got := keys.got(); !test.OK || len(got) != 1 || got[0] != "sk-legacy-0042" {
		t.Fatalf("test after the migration: %+v, keys %v", test, got)
	}

	// Down brings the connection back as it was.
	down()
	var restored []byte
	var capTokens int64
	var off []string
	var input, output float64
	if err := env.App.Pool.QueryRow(ctx, `SELECT key_sealed, cap_tokens, features_off, input_price, output_price FROM ai_connections WHERE workspace_id = $1`, ws.ID).
		Scan(&restored, &capTokens, &off, &input, &output); err != nil {
		t.Fatal(err)
	}
	if string(restored) != string(sealed) || capTokens != 500000 || !slices.Equal(off, []string{"translate"}) || input != 3 || output != 7 {
		t.Fatalf("down: cap %d, off %v, prices %v/%v, key kept %v", capTokens, off, input, output, string(restored) == string(sealed))
	}
	up()
}
