package assistant_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/ai/assistant"
	"github.com/hostrepute/reputecast/internal/apptest"
)

// The demo works without a provider (B4's panel, in development): the
// assistant is on, the sample model passed its tool check (and passes it
// again when a manager runs it), and a turn reads one thing, shows its tool
// row and streams the demo model's recorded answer in pieces.
func TestAssistantDemoModel(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	base := "/workspaces/acme-newsletter/ai"
	var models ai.AIAssistantModelList
	c.Do(http.MethodGet, base+"/assistant/models", nil).MustStatus(t, 200).JSON(t, &models)
	if len(models.Models) != 1 || !models.Models[0].Usable || models.Models[0].Tools.State != "supported" || models.DefaultID == nil || *models.DefaultID != models.Models[0].ID {
		t.Fatalf("the demo's models: %+v", models)
	}
	m := models.Models[0]
	var tools ai.AIAgentCapability
	c.Do(http.MethodPost, base+"/accounts/"+m.AccountID.String()+"/models/"+m.ID.String()+"/test-tools", nil).MustStatus(t, 200).JSON(t, &tools)
	if tools.State != "supported" {
		t.Fatalf("the demo model's tool check: %+v", tools)
	}

	var th assistant.AIThread
	c.Do(http.MethodPost, base+"/assistant/threads", map[string]any{"screen": "campaign.report", "requestKey": "demo-thread-1"}).MustStatus(t, 200).JSON(t, &th)
	c.Do(http.MethodPost, base+"/assistant/threads/"+th.ID.String()+"/messages", map[string]any{"text": "Why did opens fall?", "requestKey": "demo-turn-1"}).
		MustStatus(t, 200).JSON(t, &th)
	if th.Turn == nil {
		t.Fatalf("no turn: %+v", th)
	}
	var args assistant.TurnArgs
	var raw string
	if err := env.App.Pool.QueryRow(ctx, `SELECT args::text FROM river_job WHERE kind = 'ai.assistant_turn' AND args->>'turnId' = $1`, th.Turn.ID.String()).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		t.Fatal(err)
	}
	w := *env.App.Assistant()
	w.Hub = nil
	if err := w.ExecuteTurn(ctx, args); err != nil {
		t.Fatal(err)
	}

	var after assistant.AIThread
	c.Do(http.MethodGet, base+"/assistant/threads/"+th.ID.String(), nil).MustStatus(t, 200).JSON(t, &after)
	roles := []string{}
	for _, msg := range after.Messages {
		roles = append(roles, msg.Role)
	}
	if after.Turn == nil || after.Turn.State != "done" || strings.Join(roles, ",") != "user,tool,assistant" {
		t.Fatalf("after the turn: %+v", after)
	}
	if rows := after.Messages[1].Tools; len(rows) != 1 || rows[0].Name != "describe_screen" || !rows[0].OK || rows[0].Label != "Campaign report" {
		t.Fatalf("the tool row: %+v", rows)
	}
	if text := after.Messages[2].Text; !strings.Contains(text, "demo model's recorded answer") || !strings.Contains(text, "I read what this screen is for first.") ||
		!strings.Contains(text, "{{ subscriber.first_name }}") || after.Messages[2].Partial {
		t.Fatalf("the answer: %q", text)
	}
	var deltas int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM ai_assistant_events WHERE thread_id = $1 AND kind = 'delta'`, th.ID).Scan(&deltas); err != nil {
		t.Fatal(err)
	}
	if deltas < 2 {
		t.Fatalf("%d delta events: the answer didn't stream", deltas)
	}
	if after.Usage.Requests != 2 || after.Usage.ToolCalls != 1 || after.Usage.UncertainRequests != 0 {
		t.Fatalf("usage: %+v", after.Usage)
	}

	// A tagged question shows a provider's refusal (development only).
	c.Do(http.MethodPost, base+"/assistant/threads/"+th.ID.String()+"/messages", map[string]any{"text": "Write it anyway #refuse", "requestKey": "demo-turn-2"}).
		MustStatus(t, 200).JSON(t, &th)
	if err := env.App.Pool.QueryRow(ctx, `SELECT args::text FROM river_job WHERE kind = 'ai.assistant_turn' AND args->>'turnId' = $1`, th.Turn.ID.String()).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		t.Fatal(err)
	}
	if err := w.ExecuteTurn(ctx, args); err != nil {
		t.Fatal(err)
	}
	c.Do(http.MethodGet, base+"/assistant/threads/"+th.ID.String(), nil).MustStatus(t, 200).JSON(t, &after)
	if after.Turn == nil || after.Turn.State != "failed" || after.Turn.Reason != "refused" {
		t.Fatalf("a refused turn: %+v", after.Turn)
	}
}
