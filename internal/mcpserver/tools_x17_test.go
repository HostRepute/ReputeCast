package mcpserver_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Agents read Earn's numbers and the sponsor bookings, and propose a
// booking that a team member confirms; without sponsors:manage they can't.
func TestEarnAgentTools(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	agent := func(name string, perms ...string) *mcp.ClientSession {
		var a struct{ Secret string }
		c.Do("POST", ws+"/mcp-agents", map[string]any{"name": name, "permissions": perms}).MustStatus(t, 201).JSON(t, &a)
		cs, err := connect(t, env, a.Secret)
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		t.Cleanup(func() { _ = cs.Close() })
		return cs
	}
	ctx := context.Background()
	cs := agent("Revenue analyst", "earn:view", "sponsors:manage")

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "get_revenue_overview", Arguments: map[string]any{}})
	if err != nil || res.IsError {
		t.Fatalf("revenue: %v %s", err, text(res))
	}
	var rev struct {
		MRR     int64 `json:"mrr_cents"`
		Members int64 `json:"paid_members"`
		Gifts   int64 `json:"gift_income_cents"`
		Period  string
	}
	if err := json.Unmarshal([]byte(text(res)), &rev); err != nil || rev.MRR <= 0 || rev.Members <= 0 || rev.Gifts != 65600 || rev.Period != "30d" {
		t.Fatalf("revenue body: %+v %v %s", rev, err, text(res))
	}
	if strings.Contains(text(res), "@") {
		t.Errorf("revenue numbers carry an address: %s", text(res))
	}

	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "list_sponsor_bookings", Arguments: map[string]any{"sponsor": "northwind"}})
	if err != nil || res.IsError {
		t.Fatalf("bookings: %v %s", err, text(res))
	}
	var list struct {
		Bookings []struct {
			Issue   int
			Sponsor string
			Status  string
		}
	}
	if err := json.Unmarshal([]byte(text(res)), &list); err != nil || len(list.Bookings) < 2 {
		t.Fatalf("bookings body: %+v %v %s", list, err, text(res))
	}
	for _, b := range list.Bookings {
		if b.Sponsor != "Northwind Analytics" {
			t.Errorf("another sponsor's booking: %+v", b)
		}
	}
	if strings.Contains(text(res), "maya@") {
		t.Errorf("bookings carry the sponsor's contact: %s", text(res))
	}

	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "propose_sponsor_booking", Arguments: map[string]any{"slot": "Main sponsor", "issue": 150, "sponsor": "Contoso Cloud",
		"note": "They asked for a December issue."}})
	if err != nil || res.IsError || !strings.Contains(text(res), `"status":"proposed"`) {
		t.Fatalf("propose: %v %s", err, text(res))
	}
	var status, proposedBy string
	if err := env.App.Pool.QueryRow(ctx, `SELECT status, proposed_name FROM sponsor_bookings WHERE issue_number = 150`).Scan(&status, &proposedBy); err != nil ||
		status != "proposed" || proposedBy != "Revenue analyst" {
		t.Fatalf("proposed booking: %s %s %v", status, proposedBy, err)
	}
	var audited int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'sponsors.booking_proposed' AND actor_name = 'Revenue analyst'`).Scan(&audited)
	if audited == 0 {
		t.Error("the proposal isn't audited")
	}

	// Reading Earn doesn't let an agent propose.
	reader := agent("Earn reader", "earn:view")
	res, err = reader.CallTool(ctx, &mcp.CallToolParams{Name: "propose_sponsor_booking", Arguments: map[string]any{"slot": "Main sponsor", "issue": 151, "sponsor": "Contoso Cloud"}})
	if err != nil || !res.IsError || !strings.Contains(text(res), "sponsors:manage") {
		t.Fatalf("denied: %v %s", err, text(res))
	}
}
