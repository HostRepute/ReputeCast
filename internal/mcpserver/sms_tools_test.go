package mcpserver_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
)

// The SMS tools (X21): an agent reads the providers' overview and a text
// campaign's report, and drafts a text campaign; it never sends one, and
// without providers:view it's told so.
func TestSMSTools(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var agent struct{ Secret string }
	c.Do("POST", ws+"/mcp-agents", map[string]any{"name": "Texts", "permissions": []string{"reports:view", "providers:view", "campaigns:view", "campaigns:edit"}}).
		MustStatus(t, 201).JSON(t, &agent)
	cs, err := connect(t, env, agent.Secret)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = cs.Close() }()
	ctx := context.Background()
	call := func(name string, args map[string]any, out any) {
		t.Helper()
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil || res.IsError {
			t.Fatalf("%s: %v %s", name, err, text(res))
		}
		if err := json.Unmarshal([]byte(text(res)), out); err != nil {
			t.Fatalf("%s: %v %s", name, err, text(res))
		}
	}

	var ov struct {
		Providers []struct{ Name string }
		Routes    []struct{ Countries []string }
		Month     struct {
			Spend  float64 `json:"spend_usd"`
			Budget float64 `json:"budget_usd"`
			Sent   int64   `json:"texts_sent"`
		}
	}
	call("get_sms_overview", map[string]any{}, &ov)
	if len(ov.Providers) != 2 || ov.Providers[0].Name != "Twilio" || len(ov.Routes) != 3 || ov.Month.Spend != 39.51 || ov.Month.Budget != 100 || ov.Month.Sent != 1731 {
		t.Fatalf("overview: %+v", ov)
	}

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "get_sms_campaign_report", Arguments: map[string]any{"campaign": "Summit: 8 days to go"}})
	if err != nil || res.IsError {
		t.Fatalf("report: %v %s", err, text(res))
	}
	body := text(res)
	var r struct {
		Sent      int64
		Delivered int64
		OptOuts   int64 `json:"opt_outs"`
		Cost      float64
		Countries []struct{ Country string }
		Replies   map[string]int64
	}
	if err := json.Unmarshal([]byte(body), &r); err != nil || r.Sent != 854 || r.Delivered != 840 || r.OptOuts != 6 || len(r.Countries) != 6 || r.Replies["help"] != 2 {
		t.Fatalf("report body: %+v %v", r, err)
	}
	// Counts only: no contact's number or reply reaches the agent (Sam Rivera's
	// number, a reply's words).
	if strings.Contains(body, "5550123") || strings.Contains(body, "555 0123") || strings.Contains(body, "Aufzeichnung") {
		t.Fatalf("the report names numbers: %s", body)
	}

	var d struct{ ID, Name, Status string }
	call("create_sms_draft", map[string]any{"name": "Summit: last call", "body": "Last call for Acme Summit: doors close at 18:00.", "lists": []string{"Acme Summit 2026"}}, &d)
	if d.Name != "Summit: last call" || d.Status != "draft" {
		t.Fatalf("draft: %+v", d)
	}
	var channel, actor, lists string
	if err := env.App.Pool.QueryRow(ctx, `SELECT c.channel, a.actor_type, a.after->>'lists' FROM campaigns c JOIN audit_events a ON a.target_id = c.id::text AND a.action = 'campaign.create'
		WHERE c.id = $1`, d.ID).Scan(&channel, &actor, &lists); err != nil || channel != "sms" || actor != string(authz.ActorMCPAgent) || !strings.Contains(lists, "-") {
		t.Fatalf("draft row and audit: %s %s %s %v", channel, actor, lists, err)
	}
	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "create_sms_draft", Arguments: map[string]any{"name": "Nowhere", "body": "Hi", "lists": []string{"No such list"}}})
	if err != nil || !res.IsError || !strings.Contains(text(res), "no list named") {
		t.Fatalf("unknown list: %v %s", err, text(res))
	}

	// Without providers:view the overview is denied, like any tool.
	var reader struct{ Secret string }
	c.Do("POST", ws+"/mcp-agents", map[string]any{"name": "Reader", "permissions": []string{"reports:view"}}).MustStatus(t, 201).JSON(t, &reader)
	rs, err := connect(t, env, reader.Secret)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = rs.Close() }()
	res, err = rs.CallTool(ctx, &mcp.CallToolParams{Name: "get_sms_overview", Arguments: map[string]any{}})
	if err != nil || !res.IsError || !strings.Contains(text(res), "needs the providers:view permission") {
		t.Fatalf("denied: %v %s", err, text(res))
	}
}
