package mcpserver_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// An agent limited to some lists finds only the contacts on them, by
// search or by id and address; one that isn't finds everyone.
func TestAgentListScopeOnContacts(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var beta, inside, outside string
	if err := env.App.Pool.QueryRow(ctx, `SELECT l.id::text FROM lists l JOIN workspaces w ON w.id = l.workspace_id
		WHERE w.slug = 'acme-newsletter' AND l.name = 'Beta Program'`).Scan(&beta); err != nil {
		t.Fatal(err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT c.email FROM contacts c JOIN list_memberships m ON m.contact_id = c.id AND m.list_id = $1
		WHERE c.erased_at IS NULL ORDER BY c.email LIMIT 1`, beta).Scan(&inside); err != nil {
		t.Fatal(err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT c.email FROM contacts c JOIN workspaces w ON w.id = c.workspace_id
		WHERE w.slug = 'acme-newsletter' AND c.erased_at IS NULL AND c.email LIKE '%@gmail.com'
		  AND NOT EXISTS (SELECT 1 FROM list_memberships m WHERE m.contact_id = c.id AND m.list_id = $1)
		ORDER BY c.email LIMIT 1`, beta).Scan(&outside); err != nil {
		t.Fatal(err)
	}
	agent := func(name string, lists []string) *mcp.ClientSession {
		t.Helper()
		var a struct{ Secret string }
		body := map[string]any{"name": name, "permissions": []string{"subscribers:view"}}
		if lists != nil {
			body["lists"] = lists
		}
		linh.Do("POST", ws+"/mcp-agents", body).MustStatus(t, 201).JSON(t, &a)
		cs, err := connect(t, env, a.Secret)
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		t.Cleanup(func() { _ = cs.Close() })
		return cs
	}
	call := func(cs *mcp.ClientSession, name string, args map[string]any) (string, bool) {
		t.Helper()
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return text(res), res.IsError
	}
	type found struct{ Contacts []struct{ Email string } }

	scoped := agent("Beta helper", []string{beta})
	if out, failed := call(scoped, "get_contact", map[string]any{"contact": outside}); !failed || !strings.Contains(out, "no contact with that id or address") {
		t.Fatalf("a contact off the agent's lists isn't found: %s", out)
	}
	if out, failed := call(scoped, "get_contact", map[string]any{"contact": inside}); failed || !strings.Contains(out, inside) {
		t.Fatalf("a contact on Beta Program: %s", out)
	}
	var mine found
	out, failed := call(scoped, "search_contacts", map[string]any{"query": "gmail.com", "limit": 100})
	if failed || json.Unmarshal([]byte(out), &mine) != nil {
		t.Fatalf("search: %s", out)
	}
	for _, c := range mine.Contacts {
		var on bool
		if err := env.App.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM list_memberships m JOIN contacts c ON c.id = m.contact_id
			WHERE c.email = $1 AND m.list_id = $2)`, c.Email, beta).Scan(&on); err != nil || !on {
			t.Fatalf("the scoped search found %s, who isn't on Beta Program", c.Email)
		}
	}

	// An agent with no list scope finds anyone, by search too.
	everyone := agent("Everyone", nil)
	if out, failed := call(everyone, "get_contact", map[string]any{"contact": outside}); failed || !strings.Contains(out, outside) {
		t.Fatalf("an unscoped agent finds the contact: %s", out)
	}
	var all found
	out, failed = call(everyone, "search_contacts", map[string]any{"query": outside})
	if failed || json.Unmarshal([]byte(out), &all) != nil || len(all.Contacts) != 1 || all.Contacts[0].Email != outside {
		t.Fatalf("an unscoped agent's search: %s", out)
	}
}
