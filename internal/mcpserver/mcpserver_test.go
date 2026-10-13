package mcpserver_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/mcpserver"
)

const ws = "/workspaces/acme-newsletter"

type bearer struct {
	token string
	next  http.RoundTripper
}

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	r.Header.Set("User-Agent", "Claude-Desktop/1.4")
	return b.next.RoundTrip(r)
}

func connect(t *testing.T, env *apptest.Env, token string) (*mcp.ClientSession, error) {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	return client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:             env.Server.URL + mcpserver.Path,
		HTTPClient:           &http.Client{Transport: bearer{token, http.DefaultTransport}},
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, nil)
}

func text(r *mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range r.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func TestMCPServer(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var agent struct {
		Key    struct{ ID string }
		Secret string
	}
	c.Do("POST", ws+"/mcp-agents", map[string]any{"name": "Analyst", "permissions": []string{"reports:view", "campaigns:view", "audit:view"}}).MustStatus(t, 201).JSON(t, &agent)

	cs, err := connect(t, env, agent.Secret)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer cs.Close()
	ctx := context.Background()
	tools, err := cs.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != len(mcpserver.Tools) {
		t.Fatalf("tools: %d %v", len(tools.Tools), err)
	}

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "get_campaign_report", Arguments: map[string]any{"campaign": "October Product Update"}})
	if err != nil || res.IsError {
		t.Fatalf("report: %v %s", err, text(res))
	}
	var report struct {
		Sent              int64
		DeliveredPct      float64 `json:"delivered_pct"`
		BySendingProvider []map[string]any `json:"by_sending_provider"`
	}
	if err := json.Unmarshal([]byte(text(res)), &report); err != nil || report.Sent != 47980 || report.DeliveredPct != 99.1 || len(report.BySendingProvider) != 2 {
		t.Fatalf("report body: %+v %v %s", report, err, text(res))
	}

	// A tool the agent lacks the permission for is denied, audited and proposed.
	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "resume_campaign", Arguments: map[string]any{"campaign": "October Product Update"}})
	if err != nil || !res.IsError || !strings.Contains(text(res), "needs the campaigns:send permission") || !strings.Contains(text(res), "Linh Nguyen") {
		t.Fatalf("denied: %v %s", err, text(res))
	}
	var result, reason, actorType string
	if err := env.App.Pool.QueryRow(ctx, `SELECT result, reason, actor_type FROM audit_events WHERE actor_name = 'Analyst' AND action = 'campaign.resumed'`).Scan(&result, &reason, &actorType); err != nil ||
		result != "denied" || reason != "missing campaigns:send" || actorType != string(authz.ActorMCPAgent) {
		t.Fatalf("denied audit: %s %s %s %v", result, reason, actorType, err)
	}
	var proposals int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE kind = 'agent_proposal'`).Scan(&proposals)
	if proposals != 1 {
		t.Fatalf("proposal notifications: %d", proposals)
	}

	// The agent's last call shows in Keys & MCP.
	var agents struct {
		Items []struct{ Name, LastCall, LastUsedClient string }
	}
	c.Do("GET", ws+"/mcp-agents", nil).MustStatus(t, 200).JSON(t, &agents)
	found := false
	for _, a := range agents.Items {
		if a.Name == "Analyst" && a.LastCall == "resume_campaign" && a.LastUsedClient == "Claude Desktop" {
			found = true
		}
	}
	if !found {
		t.Fatalf("agent last call: %+v", agents.Items)
	}
	var actions struct {
		Items []struct{ Agent, Tool, Result, Note string }
	}
	c.Do("GET", ws+"/mcp-agents/actions", nil).MustStatus(t, 200).JSON(t, &actions)
	if len(actions.Items) == 0 || actions.Items[0].Agent != "Analyst" || actions.Items[0].Result != "denied" || !strings.HasPrefix(actions.Items[0].Note, "needs campaigns:send") {
		t.Fatalf("actions: %+v", actions.Items[:min(2, len(actions.Items))])
	}

	// Resources check permissions too.
	rr, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "reputecast://audit/recent.json"})
	if err != nil || len(rr.Contents) != 1 || !strings.Contains(rr.Contents[0].Text, "campaign.resumed") {
		t.Fatalf("audit resource: %v", err)
	}

	// Settings: overview with the capability matrix; turning MCP off stops agents.
	var ov struct {
		Endpoint string
		Agents   []struct {
			Name  string
			Cells []struct{ Level, Label string }
		}
	}
	c.Do("GET", ws+"/mcp", nil).MustStatus(t, 200).JSON(t, &ov)
	for _, a := range ov.Agents {
		if a.Name == "Ops agent" && a.Cells[3].Label != "Pause only · never start" {
			t.Fatalf("ops agent cells: %+v", a.Cells)
		}
	}
	if !strings.HasSuffix(ov.Endpoint, "/mcp") {
		t.Fatalf("endpoint: %q", ov.Endpoint)
	}
	c.Do("PUT", ws+"/mcp", map[string]any{"enabled": false, "rateLimitPerMinute": 120, "approvalExpiryHours": 2}).MustStatus(t, 200)
	if _, err := connect(t, env, agent.Secret); err == nil {
		t.Fatal("a disabled MCP server refuses agents")
	}
	c.Do("PUT", ws+"/mcp", map[string]any{"enabled": true, "rateLimitPerMinute": 120, "approvalExpiryHours": 2}).MustStatus(t, 200)

	// A revoked agent can't connect; its rejected calls are counted.
	c.Do("POST", ws+"/mcp-agents/"+agent.Key.ID+"/revoke", nil).MustStatus(t, 204)
	if _, err := connect(t, env, agent.Secret); err == nil {
		t.Fatal("revoked agents are refused")
	}
	var rejected int
	_ = env.App.Pool.QueryRow(ctx, `SELECT coalesce(sum(calls), 0) FROM api_key_rejections WHERE api_key_id = $1`, agent.Key.ID).Scan(&rejected)
	if rejected == 0 {
		t.Fatal("rejections are recorded")
	}
	// API keys aren't agent tokens.
	var key struct{ Secret string }
	c.Do("POST", ws+"/api-keys", map[string]any{"name": "k", "permissions": []string{"campaigns:view"}}).MustStatus(t, 201).JSON(t, &key)
	if _, err := connect(t, env, key.Secret); err == nil {
		t.Fatal("API keys can't use the MCP endpoint")
	}
}

func TestCells(t *testing.T) {
	cells := mcpserver.Cells(authz.NewSet(authz.ReportsView, authz.DeliverabilityView))
	want := []string{"Allow", "Deny", "Deny", "Deny", "Deny"}
	for i, c := range cells {
		if c.Label != want[i] {
			t.Fatalf("cell %d: %+v", i, c)
		}
	}
	cells = mcpserver.Cells(authz.NewSet(authz.CampaignsPause, authz.ProvidersView))
	if cells[0].Label != "Allow · provider health" || cells[0].Level != "allow" || cells[3].Label != "Pause only · never start" || cells[3].Level != "partial" {
		t.Fatalf("ops cells: %+v", cells)
	}
}
