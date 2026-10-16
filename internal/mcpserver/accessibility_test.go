package mcpserver_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/apptest"
)

func TestAccessibilityTool(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	agent := func(perms ...string) string {
		var a struct{ Secret string }
		c.Do("POST", ws+"/mcp-agents", map[string]any{"name": "Reviewer", "permissions": perms}).MustStatus(t, 201).JSON(t, &a)
		return a.Secret
	}
	ctx := context.Background()

	cs, err := connect(t, env, agent("campaigns:view"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = cs.Close() }()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "get_accessibility_issues", Arguments: map[string]any{"campaign": "Weekly Digest #143"}})
	if err != nil || res.IsError {
		t.Fatalf("call: %v %s", err, text(res))
	}
	var out struct {
		Summary string
		Serious int
		Issues  []struct {
			Severity, Issue, Element string
			Languages                []string
		}
	}
	if err := json.Unmarshal([]byte(text(res)), &out); err != nil {
		t.Fatalf("result: %v %s", err, text(res))
	}
	if out.Summary != "6 issues: 2 serious, 4 moderate" || len(out.Issues) != 6 || out.Issues[0].Element != "mjml › mj-body › mj-section · hero › mj-image" {
		t.Fatalf("issues: %+v", out)
	}
	var n int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'campaign.accessibility_read' AND actor_type = 'mcp_agent'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("audited reads: %d %v", n, err)
	}

	// Without campaigns:view the tool is denied.
	cs2, err := connect(t, env, agent("reports:view"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = cs2.Close() }()
	res, err = cs2.CallTool(ctx, &mcp.CallToolParams{Name: "get_accessibility_issues", Arguments: map[string]any{"campaign": "Weekly Digest #143"}})
	if err == nil && !res.IsError {
		t.Fatalf("denied call answered: %s", text(res))
	}
}
