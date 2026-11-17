package ai_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audit"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/brandkit"
)

// The account, model and feature-model operations (B1): what each refuses,
// what it audits, and that a key never comes back.

// Only integration managers add, change or remove accounts and models, or
// pick a feature's model: a Content Writer and a Viewer can't.
func TestAIAccountPermissions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var s ai.AISettings
	linh.Do(http.MethodGet, ws+"/ai", nil).MustStatus(t, 200).JSON(t, &s)
	if len(s.Accounts) != 1 || len(s.Accounts[0].Models) != 1 || !s.Accounts[0].Primary || len(s.Defaults) != 11 {
		t.Fatalf("the demo's accounts: %+v, defaults %+v", s.Accounts, s.Defaults)
	}
	account, model := s.Accounts[0].ID.String(), s.Accounts[0].Models[0].ID.String()
	base := ws + "/ai/accounts"
	ops := []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, base, map[string]any{"provider": "openai", "apiKey": "sk-not-sent", "model": "gpt-5-mini"}},
		{http.MethodDelete, base + "/" + account, nil},
		{http.MethodPut, base + "/" + account + "/key", map[string]any{"apiKey": "sk-not-sent"}},
		{http.MethodPost, base + "/" + account + "/models", map[string]any{"model": "claude-haiku-4-5"}},
		{http.MethodPut, base + "/" + account + "/models/" + model + "/prices", map[string]any{"input": 1, "output": 2}},
		{http.MethodDelete, base + "/" + account + "/models/" + model, nil},
		{http.MethodPost, base + "/" + account + "/models/" + model + "/test-tools", nil},
		{http.MethodPut, ws + "/ai/features/rewrite/model", map[string]any{"modelId": model}},
	}
	for _, who := range []string{"daniel@acme.io", "aisha@acme.io"} {
		c := env.Client()
		c.SignInDemo(who)
		for _, op := range ops {
			c.Do(op.method, op.path, op.body).MustStatus(t, 403)
		}
	}
	var after ai.AISettings
	linh.Do(http.MethodGet, ws+"/ai", nil).MustStatus(t, 200).JSON(t, &after)
	if len(after.Accounts) != 1 || len(after.Accounts[0].Models) != 1 {
		t.Fatalf("a refused operation changed the accounts: %+v", after.Accounts)
	}
}

// Accounts and models through the API: a second account's key reaches only
// its own server and never comes back; what a feature uses can't be
// removed; a feature's model must suit it; every change is audited.
func TestAIAccountOperations(t *testing.T) {
	first, second := newModelServer(t), newModelServer(t)
	r := newRegistry(t, first, map[string]any{"apiKey": "sk-first-0001"})
	const sentinel, replaced = "sk-sentinel-5f3a9c-never-shown", "sk-sentinel-replaced-77d1"
	ctx := context.Background()
	var bodies []string
	do := func(method, path string, body any, status int) *apptest.Response {
		t.Helper()
		res := r.c.Do(method, r.base+path, body).MustStatus(t, status)
		bodies = append(bodies, string(res.Body))
		return res
	}
	var s ai.AISettings
	do(http.MethodPost, "/accounts", map[string]any{"provider": "compatible", "label": "Second", "baseUrl": second.srv.URL, "apiKey": sentinel,
		"model": "model-b", "inputPrice": 3, "outputPrice": 7}, 200).JSON(t, &s)
	if len(s.Accounts) != 2 || !s.Accounts[0].Primary || s.Accounts[1].Primary || !s.Accounts[1].Key.Set || s.Accounts[1].Status != "connected" ||
		len(s.Accounts[1].Models) != 1 || s.Accounts[1].Models[0].Model != "model-b" || len(s.Accounts[1].Models[0].Features) != 0 {
		t.Fatalf("after adding an account: %+v", s.Accounts)
	}
	account, modelB := s.Accounts[1].ID, s.Accounts[1].Models[0].ID
	path := "/accounts/" + account.String()

	// Another model on that account; the same one twice is a conflict.
	do(http.MethodPost, path+"/models", map[string]any{"model": "model-c", "inputPrice": 1, "outputPrice": 2}, 200).JSON(t, &s)
	if p := do(http.MethodPost, path+"/models", map[string]any{"model": "model-c"}, 409).Problem(t); p["code"] != "ai_model_exists" {
		t.Fatalf("the same model twice: %v", p)
	}
	modelC := s.Accounts[1].Models[1].ID
	// Change model on the first account to a name it already has: 409, not 500.
	do(http.MethodPost, "/accounts/"+r.account.String()+"/models", map[string]any{"model": "model-x", "inputPrice": 1, "outputPrice": 2}, 200)
	if p := do(http.MethodPut, "/model", map[string]any{"model": "model-x"}, 409).Problem(t); p["code"] != "ai_model_exists" {
		t.Fatalf("change model to one the account has: %v", p)
	}

	// Drafts move to model-c: they reach the second server with its key.
	do(http.MethodPut, "/features/draft/model", map[string]any{"modelId": modelC}, 200).JSON(t, &s)
	if !slices.Contains(s.Defaults, ai.AIFeatureDefault{Feature: "draft", ModelID: modelC}) || featureModel(s.Features, "draft") != modelC {
		t.Fatalf("draft's model: %+v %+v", s.Defaults, s.Features)
	}
	do(http.MethodPost, "/drafts/campaigns", ai.AICampaignBriefInput{SharingReviewed: true, Brief: "Announce easier exports.", Name: "Exports", Language: "en"}, 200)
	if keys, models := second.got(); keys[len(keys)-1] != sentinel || models[len(models)-1] != "model-c" {
		t.Fatalf("the second server got %v for %v", keys, models)
	}

	// What a feature uses stays.
	p := do(http.MethodDelete, path+"/models/"+modelC.String(), nil, 409).Problem(t)
	if p["code"] != "ai_model_in_use" || !strings.Contains(p["detail"].(string), "Draft from a brief") {
		t.Fatalf("removing a feature's model: %v", p)
	}
	if p := do(http.MethodDelete, path, nil, 409).Problem(t); p["code"] != "ai_model_in_use" {
		t.Fatalf("removing an account a feature uses: %v", p)
	}
	// Alt text needs a model that reads images; model-c doesn't.
	if p := do(http.MethodPut, "/features/alt_text/model", map[string]any{"modelId": modelC}, 409).Problem(t); p["code"] != "ai_no_image_input" {
		t.Fatalf("alt text on a text-only model: %v", p)
	}
	// The agent needs a model that passed the tool-calling check.
	if p := do(http.MethodPut, "/features/agent/model", map[string]any{"modelId": modelB}, 409).Problem(t); p["code"] != "ai_tools_unchecked" {
		t.Fatalf("the agent on an unchecked model: %v", p)
	}
	var tools ai.AIAgentCapability
	do(http.MethodPost, path+"/models/"+modelB.String()+"/test-tools", nil, 200).JSON(t, &tools)
	if tools.State != "supported" {
		t.Fatalf("tool check: %+v", tools)
	}
	do(http.MethodPut, "/features/agent/model", map[string]any{"modelId": modelB}, 200)
	// A model of another workspace, or none, isn't found.
	otherOwner := r.env.CreateUser(apptest.UserOpts{Email: "owner@other.test"})
	other := r.env.CreateWorkspace("Other", otherOwner.ID)
	oc := r.env.Client()
	oc.SignInDemo(otherOwner.Email)
	oc.Do(http.MethodPut, "/workspaces/"+other.Slug+"/ai/connection", map[string]any{"provider": "compatible", "baseUrl": first.srv.URL, "model": "model-o", "capKind": "none", "inputPrice": 1, "outputPrice": 2}).MustStatus(t, 200)
	var otherModel uuid.UUID
	if err := r.env.App.Pool.QueryRow(ctx, `SELECT id FROM ai_models WHERE workspace_id = $1`, other.ID).Scan(&otherModel); err != nil {
		t.Fatal(err)
	}
	for _, id := range []uuid.UUID{otherModel, uuid.Must(uuid.NewV7())} {
		do(http.MethodPut, "/features/draft/model", map[string]any{"modelId": id}, 404)
	}
	do(http.MethodPut, "/accounts/"+r.account.String()+"/models/"+otherModel.String()+"/prices", map[string]any{"input": 1, "output": 1}, 404)

	// Prices and the key.
	do(http.MethodPut, path+"/models/"+modelC.String()+"/prices", map[string]any{"input": 5, "output": 9}, 200).JSON(t, &s)
	if pr := s.Accounts[1].Models[1].Prices; pr.Input != 5 || pr.Output != 9 || !pr.Known {
		t.Fatalf("prices: %+v", pr)
	}
	do(http.MethodPut, path+"/key", map[string]any{"apiKey": replaced}, 200)
	if keys, _ := second.got(); keys[len(keys)-1] != replaced {
		t.Fatalf("the replaced key wasn't tested: %v", keys)
	}
	if keys, _ := first.got(); slices.Contains(keys, sentinel) || slices.Contains(keys, replaced) {
		t.Fatal("the second account's key reached the first account's server")
	}

	// Once nothing uses them, the model and the account go.
	do(http.MethodPost, "/accounts/"+r.account.String()+"/models/"+r.model.String()+"/test-tools", nil, 200)
	do(http.MethodPut, "/features/agent/model", map[string]any{"modelId": r.model}, 200)
	do(http.MethodPut, "/features/draft/model", map[string]any{"modelId": r.model}, 200)
	do(http.MethodDelete, path+"/models/"+modelC.String(), nil, 200)
	if p := do(http.MethodDelete, path+"/models/"+modelB.String(), nil, 409).Problem(t); p["code"] != "ai_last_model" {
		t.Fatalf("removing an account's only model: %v", p)
	}
	do(http.MethodDelete, path, nil, 200).JSON(t, &s)
	if len(s.Accounts) != 1 {
		t.Fatalf("after removing the account: %+v", s.Accounts)
	}

	var actions []string
	rows, err := r.env.App.Pool.Query(ctx, `SELECT action FROM audit_events WHERE workspace_id = $1 AND (action LIKE 'ai\_account.%' OR action LIKE 'ai\_model.%' OR action LIKE 'ai\_feature.%')`, r.ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			t.Fatal(err)
		}
		actions = append(actions, a)
	}
	rows.Close()
	for _, want := range []string{"ai_account.add", "ai_model.add", "ai_feature.model_changed", "ai_model.prices_updated", "ai_account.key_replaced", "ai_model.remove", "ai_account.remove"} {
		if !slices.Contains(actions, want) {
			t.Errorf("no %s audit entry: %v", want, actions)
		}
	}
	assertNoKey(t, r, bodies, sentinel, replaced)
}

// assertNoKey checks the keys appear in no response, audit entry, activity
// log row, notification or job.
func assertNoKey(t *testing.T, r *registry, bodies []string, keys ...string) {
	t.Helper()
	for _, key := range keys {
		for _, b := range bodies {
			if strings.Contains(b, key) {
				t.Fatalf("a response has the key: %s", b)
			}
		}
		var n int
		if err := r.env.App.Pool.QueryRow(context.Background(), `SELECT
			(SELECT count(*) FROM audit_events WHERE concat(before::text, after::text, metadata::text, target_name) LIKE '%' || $1 || '%')
			+ (SELECT count(*) FROM ai_requests WHERE concat(outcome_detail, target_label) LIKE '%' || $1 || '%')
			+ (SELECT count(*) FROM ai_accounts WHERE status_detail LIKE '%' || $1 || '%')
			+ (SELECT count(*) FROM notifications WHERE row_to_json(notifications)::text LIKE '%' || $1 || '%')
			+ (SELECT count(*) FROM river_job WHERE args::text LIKE '%' || $1 || '%')`, key).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("the key is stored readable in %d rows", n)
		}
	}
}

func featureModel(features []ai.AIFeatureState, key string) uuid.UUID {
	for _, f := range features {
		if f.Key == key && f.ModelID != nil {
			return *f.ModelID
		}
	}
	return uuid.Nil
}

// What the account operations log never has the key: added, tested,
// replaced, or refused by the provider.
func TestAIAccountKeyNotLogged(t *testing.T) {
	first, second := newModelServer(t), newKeyServer(t)
	r := newRegistry(t, first, nil)
	var logs bytes.Buffer
	env := r.env
	s := &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, AllowPrivate: func() bool { return true },
		Log: slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})), BrandKit: &brandkit.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}}
	p := &authz.Principal{Type: authz.ActorUser, ID: r.owner, UserID: r.owner, WorkspaceID: r.ws.ID}
	ctx := audit.WithPrincipal(context.Background(), p)
	const key, next = "sk-sentinel-logged-31c0", "sk-sentinel-logged-next-9e2b"
	out, err := s.AddAccount(ctx, &r.ws, p, ai.AIAccountInput{Provider: "compatible", BaseURL: second.srv.URL + "/v1", APIKey: key, Model: "gemma3:27b", InputPrice: ptr(1.0), OutputPrice: ptr(2.0)})
	if err != nil {
		t.Fatal(err)
	}
	account := out.Accounts[1].ID
	model := out.Accounts[1].Models[0].ID
	if _, _, err := s.ReplaceAccountKey(ctx, &r.ws, p, account, ai.AIKeyInput{APIKey: next}); err != nil {
		t.Fatal(err)
	}
	if res, err := s.Test(ctx, &r.ws, p, ai.AITestInput{ModelID: &model}); err != nil || !res.OK {
		t.Fatalf("test of the saved model: %+v %v", res, err)
	}
	if got := second.got(); !slices.Equal(got, []string{key, next, next}) {
		t.Fatalf("keys at the server: %v", got)
	}
	for _, k := range []string{key, next} {
		if strings.Contains(logs.String(), k) {
			t.Fatalf("the log has the key: %s", logs.String())
		}
	}
	assertNoKey(t, r, nil, key, next)
}

func ptr[T any](v T) *T { return &v }

// A request prepared with a feature's model isn't sent once a manager picks
// another model for the feature.
func TestAIFeatureModelChangeRefusesPrepared(t *testing.T) {
	first, second := newModelServer(t), newModelServer(t)
	r := newRegistry(t, first, nil)
	model := r.addModel(r.addAccount(second.srv.URL, ""), "model-b", 1, 2)
	ctx := context.Background()
	tx, err := r.env.App.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT 1 FROM ai_workspace_settings WHERE workspace_id = $1 FOR UPDATE`, r.ws.ID); err != nil {
		t.Fatal(err)
	}
	before, _ := first.got()
	done := make(chan *apptest.Response, 1)
	go func() { done <- r.draft() }()
	r.waitForLock(1)
	// The manager's change, under the lock the request waits for.
	if _, err := tx.Exec(ctx, `UPDATE ai_feature_models SET model_id = $2 WHERE workspace_id = $1 AND feature = 'draft'`, r.ws.ID, model); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if p := (<-done).MustStatus(t, 409).Problem(t); p["code"] != "ai_connection_changed" {
		t.Fatalf("a request prepared with the previous model: %v", p)
	}
	after, _ := first.got()
	if keys, _ := second.got(); len(after) != len(before) || len(keys) != 0 {
		t.Fatal("a request prepared with the previous model was sent")
	}
	r.draft().MustStatus(t, 200)
	if _, models := second.got(); !slices.Equal(models, []string{"model-b"}) {
		t.Fatalf("the next request: %v", models)
	}
}

// Each feature follows its own model: with the first account disconnected
// and rewrite on a connected one, the assistant is ready for rewrites only,
// and the integrations catalog shows the connected account.
func TestAIStatusFollowsFeatureModels(t *testing.T) {
	first, second := newModelServer(t), newModelServer(t)
	r := newRegistry(t, first, nil)
	var s ai.AISettings
	r.c.Do(http.MethodPost, r.base+"/accounts", map[string]any{"provider": "compatible", "baseUrl": second.srv.URL, "model": "model-b", "inputPrice": 1, "outputPrice": 2}).MustStatus(t, 200).JSON(t, &s)
	model := s.Accounts[1].Models[0].ID
	r.c.Do(http.MethodPut, r.base+"/features/rewrite/model", map[string]any{"modelId": model}).MustStatus(t, 200)
	r.c.Do(http.MethodDelete, r.base+"/connection", nil).MustStatus(t, 200).JSON(t, &s)
	if s.Connected || s.Accounts[0].Status != "disconnected" || s.Accounts[1].Status != "connected" || featureModel(s.Features, "rewrite") != model {
		t.Fatalf("settings: %+v %+v", s.Accounts, s.Features)
	}
	var st ai.AIStatus
	r.c.Do(http.MethodGet, r.base+"/status", nil).MustStatus(t, 200).JSON(t, &st)
	if st.State != "ready" || st.Model != "model-b" || len(st.Accounts) != 2 || len(st.Defaults) != 11 {
		t.Fatalf("status: %+v", st)
	}
	for _, f := range st.Features {
		if (f.Key == "rewrite") != f.On {
			t.Fatalf("status features: %+v", st.Features)
		}
	}
	var catalog struct {
		Groups []struct {
			Key     string
			Entries []struct {
				Key, Status string
				Facts       struct {
					AI *struct{ Connected, ModelName string }
				}
			}
		}
	}
	r.c.Do(http.MethodGet, "/workspaces/"+r.ws.Slug+"/integrations/catalog", nil).MustStatus(t, 200).JSON(t, &catalog)
	found := false
	for _, g := range catalog.Groups {
		for _, e := range g.Entries {
			switch {
			case g.Key == "ai" && e.Key == "compatible":
				found = e.Status == "connected" && e.Facts.AI != nil && e.Facts.AI.ModelName == "model-b"
			case g.Key == "ai" && (e.Facts.AI == nil || e.Facts.AI.Connected != "OpenAI-compatible"):
				t.Fatalf("%s doesn't name the connected provider: %+v", e.Key, e.Facts.AI)
			}
		}
	}
	if !found {
		t.Fatalf("the catalog's connected account: %+v", catalog)
	}
}
