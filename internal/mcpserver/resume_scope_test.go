package mcpserver_test

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// An agent limited to some lists resumes only a send to them: resuming
// lets the mail go out, as starting it would.
func TestAgentResumesOnlyWithinItsScope(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var beta string
	if err := env.App.Pool.QueryRow(ctx, `SELECT l.id::text FROM lists l JOIN workspaces w ON w.id = l.workspace_id
		WHERE w.slug = 'acme-newsletter' AND l.name = 'Beta Program'`).Scan(&beta); err != nil {
		t.Fatal(err)
	}
	var a struct{ Secret string }
	linh.Do("POST", ws+"/mcp-agents", map[string]any{"name": "Beta sender", "permissions": []string{"campaigns:send"}, "lists": []string{beta}}).MustStatus(t, 201).JSON(t, &a)
	cs, err := connect(t, env, a.Secret)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = cs.Close() }()

	// Weekly Digest #143 goes to the Weekly Digest list; paused here.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET status = 'paused' WHERE name = 'Weekly Digest #143'`); err != nil {
		t.Fatal(err)
	}
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "resume_campaign", Arguments: map[string]any{"campaign": "Weekly Digest #143"}})
	if err != nil || !res.IsError || !strings.Contains(text(res), "Your access is limited to some lists") {
		t.Fatalf("resume outside the agent's lists: %v %s", err, text(res))
	}
	var status, result string
	if err := env.App.Pool.QueryRow(ctx, `SELECT status FROM campaigns WHERE name = 'Weekly Digest #143'`).Scan(&status); err != nil || status != "paused" {
		t.Fatalf("still paused: %s %v", status, err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT result FROM audit_events WHERE actor_name = 'Beta sender' AND action = 'campaign.resumed'`).Scan(&result); err != nil || result != "error" {
		t.Fatalf("the refused resume is audited: %s %v", result, err)
	}
}
