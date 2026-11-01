package mcpserver_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// The read tools agents asked for and send proposals for texts, push and
// WhatsApp (review 2026-11 IN-3): each needs its REST operation's
// permission, and proposing never sends.
func TestM46Tools(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var agent struct{ Secret string }
	c.Do("POST", ws+"/mcp-agents", map[string]any{"name": "Assistant", "permissions": []string{"subscribers:view", "automations:view", "campaigns:view", "campaigns:send", "earn:view"}}).
		MustStatus(t, 201).JSON(t, &agent)
	cs, err := connect(t, env, agent.Secret)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = cs.Close() }()
	ctx := context.Background()
	call := func(name string, args map[string]any, out any) string {
		t.Helper()
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil || res.IsError {
			t.Fatalf("%s: %v %s", name, err, text(res))
		}
		if out != nil {
			if err := json.Unmarshal([]byte(text(res)), out); err != nil {
				t.Fatalf("%s: %v %s", name, err, text(res))
			}
		}
		return text(res)
	}

	var lists struct {
		Lists []struct {
			Name       string
			Subscribed int64
		}
	}
	call("list_lists", map[string]any{}, &lists)
	if len(lists.Lists) == 0 || lists.Lists[0].Subscribed == 0 {
		t.Fatalf("lists: %+v", lists)
	}
	var contact struct {
		Email string
		Lists []string
	}
	var someone string
	if err := env.App.Pool.QueryRow(ctx, `SELECT c.email FROM contacts c JOIN list_memberships m ON m.contact_id = c.id AND m.status = 'subscribed'
		JOIN workspaces w ON w.id = c.workspace_id WHERE w.slug = 'acme-newsletter' AND c.erased_at IS NULL ORDER BY c.email LIMIT 1`).Scan(&someone); err != nil {
		t.Fatal(err)
	}
	call("get_contact", map[string]any{"contact": strings.ToUpper(someone)}, &contact)
	if contact.Email != someone || len(contact.Lists) == 0 {
		t.Fatalf("contact: %+v", contact)
	}
	var forms struct{ Forms []struct{ Name string } }
	call("list_forms", map[string]any{}, &forms)
	var flows struct{ Automations []struct{ Name string } }
	call("list_automations", map[string]any{}, &flows)
	if len(flows.Automations) == 0 {
		t.Fatal("the demo has flows")
	}
	var flow struct {
		Name  string
		Stats *struct{ Days int }
	}
	call("get_automation", map[string]any{"automation": flows.Automations[0].Name}, &flow)
	if flow.Name != flows.Automations[0].Name || flow.Stats == nil || flow.Stats.Days != 30 {
		t.Fatalf("automation: %+v", flow)
	}
	var approvals struct {
		Approvals []struct{ Name, Status string }
		Note      string
	}
	call("list_approvals", map[string]any{}, &approvals)
	if len(approvals.Approvals) == 0 || approvals.Approvals[0].Status != "pending" {
		t.Fatalf("approvals: %+v", approvals)
	}
	var offers struct{ Offers []struct{ Name string } }
	call("list_offers", map[string]any{}, &offers)

	// Proposing a text's send: a proposal, not a send.
	var proposed struct {
		Proposed bool
		Channel  string
	}
	call("propose_send", map[string]any{"campaign": "Summit: doors open"}, &proposed)
	if !proposed.Proposed || proposed.Channel != "sms" {
		t.Fatalf("proposal: %+v", proposed)
	}
	var status string
	if err := env.App.Pool.QueryRow(ctx, `SELECT status FROM campaigns WHERE name = 'Summit: doors open'`).Scan(&status); err != nil || status != "draft" {
		t.Fatalf("a proposal sends nothing: %s %v", status, err)
	}
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "propose_send", Arguments: map[string]any{"campaign": "Flash sale: 48 hours only"}})
	if err != nil || !res.IsError || !strings.Contains(text(res), "schedule_campaign") {
		t.Fatalf("an email campaign is scheduled with schedule_campaign: %v %s", err, text(res))
	}
}
