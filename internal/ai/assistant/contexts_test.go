package assistant_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/ai/assistant"
	"github.com/hostrepute/reputecast/internal/apptest"
)

const demoAI = "/workspaces/acme-newsletter/ai"

// Attaching needs the person's sharing review and their access to the
// resource; a report is its fixed snapshot, sealed; what they can no
// longer read stays attached but unavailable; another person can't attach
// to their thread.
func TestAssistantContexts(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	model := fakeModel(t, newFakeStreams())
	ctx := context.Background()
	client := func(email string) *apptest.Client {
		c := env.Client()
		c.SignInDemo(email)
		return c
	}
	linh, sofia, daniel := client("linh@acme.io"), client("sofia@acme.io"), client("daniel@acme.io")
	linh.Do(http.MethodDelete, demoAI+"/connection", nil).MustStatus(t, 200)
	var s ai.AISettings
	linh.Do(http.MethodPut, demoAI+"/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL, "model": "model-a", "capKind": "none",
		"inputPrice": 1, "outputPrice": 2}).MustStatus(t, 200).JSON(t, &s)
	var tools ai.AIAgentCapability
	linh.Do(http.MethodPost, demoAI+"/accounts/"+s.Accounts[0].ID.String()+"/models/"+s.Accounts[0].Models[0].ID.String()+"/test-tools", nil).MustStatus(t, 200).JSON(t, &tools)
	linh.Do(http.MethodPut, demoAI+"/assistant/settings", ai.AIAssistantSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000},
		PerPerson: 2, PerWorkspace: 10, Revision: s.Assistant.Revision}).MustStatus(t, 200)
	var th assistant.AIThread
	sofia.Do(http.MethodPost, demoAI+"/assistant/threads", map[string]any{"requestKey": "contexts-1"}).MustStatus(t, 200).JSON(t, &th)
	path := demoAI + "/assistant/threads/" + th.ID.String() + "/contexts"

	var draft, sent uuid.UUID
	var draftName, sentName string
	var draftVersion int64
	if err := env.App.Pool.QueryRow(ctx, `SELECT c.id, c.name, c.version FROM campaigns c JOIN workspaces w ON w.id = c.workspace_id
		WHERE w.slug = 'acme-newsletter' AND c.channel = 'email' AND c.status = 'draft' AND c.deleted_at IS NULL ORDER BY c.id LIMIT 1`).Scan(&draft, &draftName, &draftVersion); err != nil {
		t.Fatal(err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT c.id, c.name FROM campaigns c JOIN workspaces w ON w.id = c.workspace_id
		WHERE w.slug = 'acme-newsletter' AND c.channel = 'email' AND c.status = 'sent' AND c.completed_at < now() - interval '4 days' AND c.deleted_at IS NULL
		ORDER BY c.completed_at DESC LIMIT 1`).Scan(&sent, &sentName); err != nil {
		t.Fatal(err)
	}

	// Nothing is shared before the person reviewed it.
	if p := sofia.Do(http.MethodPost, path, map[string]any{"kind": "campaign", "id": draft, "sharingReviewed": false}).MustStatus(t, 422).Problem(t); p["code"] != "ai_assistant_sharing_review" {
		t.Fatalf("attaching without the sharing review: %v", p)
	}
	sofia.Do(http.MethodPost, path, map[string]any{"kind": "campaign", "id": draft, "sharingReviewed": true}).MustStatus(t, 200).JSON(t, &th)
	sofia.Do(http.MethodPost, path, map[string]any{"kind": "campaign", "id": draft, "sharingReviewed": true}).MustStatus(t, 200).JSON(t, &th)
	if len(th.Contexts) != 1 || th.Contexts[0].Title != draftName || !th.Contexts[0].Available || th.Contexts[0].Revision != draftVersion {
		t.Fatalf("after attaching the draft (twice): %+v", th.Contexts)
	}
	// A report is its fixed snapshot, sealed for its attachment.
	sofia.Do(http.MethodPost, path, map[string]any{"kind": "report", "id": sent, "sharingReviewed": true}).MustStatus(t, 200).JSON(t, &th)
	if len(th.Contexts) != 2 || th.Contexts[1].Kind != "report" || th.Contexts[1].Title != sentName {
		t.Fatalf("after attaching the report: %+v", th.Contexts)
	}
	var sealed []byte
	if err := env.App.Pool.QueryRow(ctx, `SELECT snapshot_sealed FROM ai_assistant_contexts WHERE id = $1`, th.Contexts[1].ID).Scan(&sealed); err != nil {
		t.Fatal(err)
	}
	raw, err := env.App.Secrets.Open(sealed, "ai_assistant_context:"+th.Contexts[1].ID.String())
	var snap struct {
		Status  string           `json:"status"`
		Metrics map[string]int64 `json:"metrics"`
	}
	if err != nil || json.Unmarshal(raw, &snap) != nil || snap.Status == "" || snap.Metrics == nil || bytes.Contains(sealed, []byte(`"status"`)) {
		t.Fatalf("the report's snapshot at rest: %v %s", err, raw)
	}
	// What the person can't read isn't attached: a draft isn't a report,
	// another workspace's or an unknown id isn't there.
	sofia.Do(http.MethodPost, path, map[string]any{"kind": "report", "id": draft, "sharingReviewed": true}).MustStatus(t, 404)
	sofia.Do(http.MethodPost, path, map[string]any{"kind": "campaign", "id": uuid.New(), "sharingReviewed": true}).MustStatus(t, 404)
	// Someone else's thread isn't there.
	daniel.Do(http.MethodPost, path, map[string]any{"kind": "campaign", "id": draft, "sharingReviewed": true}).MustStatus(t, 404)
	daniel.Do(http.MethodDelete, path+"/"+th.Contexts[0].ID.String(), nil).MustStatus(t, 404)
	if n := countIn(t, env, `SELECT count(*) FROM audit_events WHERE target_id = $1 AND action = 'ai_assistant_context.add' AND metadata->>'sharing_policy' <> ''`, th.ID.String()); n != 2 {
		t.Fatalf("%d audit entries of attaching", n)
	}

	// Once the person can no longer read it, it is shown unavailable, without its name.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET deleted_at = now() WHERE id = $1`, draft); err != nil {
		t.Fatal(err)
	}
	sofia.Do(http.MethodGet, demoAI+"/assistant/threads/"+th.ID.String(), nil).MustStatus(t, 200).JSON(t, &th)
	if th.Contexts[0].Available || th.Contexts[0].Title != "" || !th.Contexts[1].Available {
		t.Fatalf("after the draft was deleted: %+v", th.Contexts)
	}
	sofia.Do(http.MethodDelete, path+"/"+th.Contexts[0].ID.String(), nil).MustStatus(t, 200).JSON(t, &th)
	if len(th.Contexts) != 1 || th.Contexts[0].Kind != "report" {
		t.Fatalf("after detaching: %+v", th.Contexts)
	}
	// A thread holds up to ten.
	for range 9 {
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO ai_assistant_contexts (id, thread_id, kind, target_id, shared_at) VALUES ($1, $2, 'campaign', $3, now())`,
			uuid.Must(uuid.NewV7()), th.ID, uuid.New()); err != nil {
			t.Fatal(err)
		}
	}
	if p := sofia.Do(http.MethodPost, path, map[string]any{"kind": "campaign", "id": sent, "sharingReviewed": true}).MustStatus(t, 409).Problem(t); p["code"] != "ai_assistant_contexts_full" {
		t.Fatalf("an eleventh resource: %v", p)
	}
}

func countIn(t *testing.T, env *apptest.Env, query string, args ...any) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
