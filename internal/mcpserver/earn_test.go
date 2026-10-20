package mcpserver_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// get_paid_member_counts: counts and MRR only, never a name or an email,
// for an agent with earn:view.
func TestPaidMemberCountsTool(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	agent := func(perms ...string) string {
		var a struct{ Secret string }
		c.Do("POST", ws+"/mcp-agents", map[string]any{"name": "Revenue reader", "permissions": perms}).MustStatus(t, 201).JSON(t, &a)
		return a.Secret
	}
	ctx := context.Background()
	cs, err := connect(t, env, agent("earn:view"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = cs.Close() }()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "get_paid_member_counts", Arguments: map[string]any{}})
	if err != nil || res.IsError {
		t.Fatalf("call: %v %s", err, text(res))
	}
	var out struct {
		Counts struct {
			WithAccess, Paying, Trialing, PastDue int64
		}
		MRRCents int64 `json:"mrr_cents"`
		Currency string
		Mode     string
	}
	if err := json.Unmarshal([]byte(text(res)), &out); err != nil {
		t.Fatalf("result: %v %s", err, text(res))
	}
	if out.Counts.WithAccess != 1252 || out.Counts.Paying != 1184 || out.Counts.Trialing != 37 || out.Counts.PastDue != 14 || out.MRRCents != 896000 || out.Mode != "live" {
		t.Fatalf("counts: %s", text(res))
	}
	if body := text(res); strings.Contains(body, "@") {
		t.Fatalf("an address in the counts: %s", body)
	}
	var n int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'earn.paid_member_counts_read' AND actor_type = 'mcp_agent'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("audited reads: %d %v", n, err)
	}
	// Without earn:view the tool is denied.
	cs2, err := connect(t, env, agent("campaigns:view"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = cs2.Close() }()
	res, err = cs2.CallTool(ctx, &mcp.CallToolParams{Name: "get_paid_member_counts", Arguments: map[string]any{}})
	if err == nil && !res.IsError {
		t.Fatalf("denied call answered: %s", text(res))
	}
}
