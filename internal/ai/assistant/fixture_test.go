package assistant_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/ai/assistant"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/store"
)

// fakeModel is an OpenAI-compatible server: it passes the tool-calling
// check, streams answers as st's script says, and answers anything else
// with a sentence.
func fakeModel(t *testing.T, st *fakeStreams) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Stream bool `json:"stream"`
			Tools  []struct {
				Function struct{ Name string } `json:"function"`
			} `json:"tools"`
			Messages []struct {
				Role    string `json:"role"`
				Content any    `json:"content"`
			} `json:"messages"`
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &req)
		if req.Stream {
			st.serve(w, r, raw)
			return
		}
		st.record(raw)
		message := map[string]any{"role": "assistant", "content": "Ready."}
		if strings.Contains(string(raw), "subjects") {
			// The subject ideas' one-shot request.
			message["content"] = `{"subjects":["Fresh ideas for spring","Your spring update"]}`
		}
		stop := "stop"
		if len(req.Tools) > 0 && req.Tools[0].Function.Name == "check_tools" {
			message["content"] = nil
			message["tool_calls"] = []any{map[string]any{"id": "call1", "type": "function", "function": map[string]any{"name": "check_tools", "arguments": `{"value":"ready"}`}}}
			stop = "tool_calls"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": message, "finish_reason": stop}},
			"usage": map[string]int{"prompt_tokens": 9, "completion_tokens": 2}})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// fixture is a workspace with a model that passed its tool check and the
// assistant on: its owner manages integrations; member and colleague are
// Editors (ai:use); viewer has no ai:use.
type fixture struct {
	t                        *testing.T
	env                      *apptest.Env
	ws                       store.Workspace
	base                     string
	model                    *httptest.Server
	streams                  *fakeStreams
	owner, member, colleague uuid.UUID
	oc, mc, cc, vc           *apptest.Client
	accountID, modelID       uuid.UUID
	settings                 ai.AIAssistantSettings
}

func newFixture(t *testing.T) *fixture {
	env := apptest.New(t)
	f := &fixture{t: t, env: env, streams: newFakeStreams()}
	f.model = fakeModel(t, f.streams)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@assistant.test", Name: "Assistant Owner"})
	f.ws = env.CreateWorkspace("Assistant", owner.ID)
	f.base = "/workspaces/" + f.ws.Slug + "/ai"
	f.owner = owner.ID
	member := env.CreateUser(apptest.UserOpts{Email: "member@assistant.test", Name: "Thread Owner"})
	colleague := env.CreateUser(apptest.UserOpts{Email: "colleague@assistant.test", Name: "Colleague"})
	viewer := env.CreateUser(apptest.UserOpts{Email: "viewer@assistant.test", Name: "Viewer"})
	env.AddMember(f.ws, member.ID, authz.RoleEditor)
	env.AddMember(f.ws, colleague.ID, authz.RoleEditor)
	env.AddMember(f.ws, viewer.ID, authz.RoleViewer)
	f.member, f.colleague = member.ID, colleague.ID
	f.oc, f.mc, f.cc, f.vc = f.client(owner.Email), f.client(member.Email), f.client(colleague.Email), f.client(viewer.Email)
	f.oc.Do(http.MethodPut, f.base+"/connection", map[string]any{"provider": "compatible", "baseUrl": f.model.URL, "model": "model-a", "capKind": "none",
		"inputPrice": 1, "outputPrice": 2}).MustStatus(t, 200)
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT a.id, m.id FROM ai_accounts a JOIN ai_models m ON m.account_id = a.id WHERE a.workspace_id = $1`, f.ws.ID).
		Scan(&f.accountID, &f.modelID); err != nil {
		t.Fatal(err)
	}
	f.passTools(f.oc, "/accounts/"+f.accountID.String()+"/models/"+f.modelID.String())
	f.setAssistant(ai.AIAssistantSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}, PerPerson: 2, PerWorkspace: 10})
	return f
}

func (f *fixture) client(email string) *apptest.Client {
	c := f.env.Client()
	c.SignInDemo(email)
	return c
}

// passTools runs a model's tool-calling check (path is the model's, under
// /ai or /ai/personal).
func (f *fixture) passTools(c *apptest.Client, path string) {
	f.t.Helper()
	var tools ai.AIAgentCapability
	c.Do(http.MethodPost, f.base+path+"/test-tools", nil).MustStatus(f.t, 200).JSON(f.t, &tools)
	if tools.State != "supported" {
		f.t.Fatalf("the tool check: %+v", tools)
	}
}

// setAssistant saves the assistant's settings at their current revision.
func (f *fixture) setAssistant(in ai.AIAssistantSettings) {
	f.t.Helper()
	in.Revision = f.settings.Revision
	var s ai.AISettings
	f.oc.Do(http.MethodPut, f.base+"/assistant/settings", in).MustStatus(f.t, 200).JSON(f.t, &s)
	f.settings = s.Assistant
}

// thread starts a thread as c.
func (f *fixture) thread(c *apptest.Client, in map[string]any) assistant.AIThread {
	f.t.Helper()
	if in["requestKey"] == nil {
		in["requestKey"] = strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	var th assistant.AIThread
	c.Do(http.MethodPost, f.base+"/assistant/threads", in).MustStatus(f.t, 200).JSON(f.t, &th)
	return th
}

// aiService is the AI service as the app builds it, for calls the API has
// no operation for yet.
func (f *fixture) aiService() *ai.Service {
	env := f.env
	return &ai.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, AllowPrivate: func() bool { return true },
		BrandKit: &brandkit.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}}
}

func person(ws store.Workspace, user uuid.UUID) *authz.Principal {
	return &authz.Principal{Type: authz.ActorUser, ID: user, UserID: user, WorkspaceID: ws.ID}
}

// count runs a count query.
func (f *fixture) count(query string, args ...any) int {
	f.t.Helper()
	var n int
	if err := f.env.App.Pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		f.t.Fatal(err)
	}
	return n
}
