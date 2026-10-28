package mcpserver_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// R4.L3, CO-3: an agent's draft is stored as the composer stores one. The
// server compiles and checks the MJML, so the HTML a send uses is what the
// MJML says (a block meant for some readers keeps its conditional markers),
// each change is a version in the history, and the tool call is the audit
// entry, with the agent as its actor.
func TestAgentDraftsAreCompiledAndVersioned(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	var agent struct{ Secret string }
	c.Do("POST", ws+"/mcp-agents", map[string]any{"name": "Drafts", "permissions": []string{"campaigns:view", "campaigns:edit"}}).MustStatus(t, 201).JSON(t, &agent)
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
	type stored struct {
		Subject, MJML, HTML, PlainText, Hash, Source, Label, SavedBy string
		Version, Versions                                            int
		HasSender                                                    bool
	}
	load := func(id string) stored {
		t.Helper()
		var s stored
		if err := env.App.Pool.QueryRow(ctx, `SELECT c.subject, c.mjml, c.html, c.plain_text, c.content_hash, c.version, c.sender_id IS NOT NULL,
				v.source, v.label, coalesce(u.email, ''), (SELECT count(*) FROM campaign_versions WHERE campaign_id = c.id)
			FROM campaigns c JOIN campaign_versions v ON v.campaign_id = c.id AND v.version = c.version LEFT JOIN users u ON u.id = v.saved_by
			WHERE c.id = $1`, id).Scan(&s.Subject, &s.MJML, &s.HTML, &s.PlainText, &s.Hash, &s.Version, &s.HasSender, &s.Source, &s.Label, &s.SavedBy, &s.Versions); err != nil {
			t.Fatalf("load %s: %v", id, err)
		}
		return s
	}

	// A new draft is compiled on the server, with its first version and the
	// workspace's default sender, as the composer makes one.
	var d struct {
		ID      string
		Version int
	}
	call("create_campaign_draft", map[string]any{"name": "Agent's digest", "subject": "What shipped",
		"mjml": `<mjml><mj-body><mj-section><mj-column><mj-text>First draft from the agent.</mj-text></mj-column></mj-section></mj-body></mjml>`}, &d)
	s := load(d.ID)
	if s.Version != 1 || s.Versions != 1 || s.Source != "api" || s.Label != "created by an agent" || s.SavedBy != "linh@acme.io" || !s.HasSender {
		t.Fatalf("created: %+v", s)
	}
	if !strings.Contains(s.HTML, "First draft from the agent.") || !strings.Contains(s.HTML, "<title>What shipped</title>") || !strings.Contains(s.PlainText, "First draft from the agent.") || s.Hash == "" {
		t.Fatalf("created draft isn't compiled: %+v", s)
	}

	// New MJML: the HTML is the server's compile, with the markers that keep
	// a block for some readers out of everyone else's email.
	mjmlSrc := `<mjml><mj-body><mj-section><mj-column><mj-text>Second draft.</mj-text></mj-column></mj-section>` +
		`<mj-section rc-visible='{"id":"beta01","show":"only","match":"all","conditions":[]}'><mj-column><mj-text>Beta testers only.</mj-text></mj-column></mj-section></mj-body></mjml>`
	var u struct {
		Version   int
		SizeBytes int `json:"size_bytes"`
	}
	call("update_campaign_content", map[string]any{"campaign": d.ID, "mjml": mjmlSrc}, &u)
	s = load(d.ID)
	if u.Version != 2 || s.Version != 2 || s.Versions != 2 || s.Source != "api" || s.Label != "changed by an agent" || s.SavedBy != "linh@acme.io" || u.SizeBytes != len(s.HTML) {
		t.Fatalf("updated: %+v %+v", u, s)
	}
	if strings.Contains(s.HTML, "First draft") || !strings.Contains(s.HTML, "Second draft.") || !strings.Contains(s.HTML, "<!--rc:if beta01-->") || !strings.Contains(s.HTML, "<!--rc:fi beta01-->") ||
		strings.Contains(s.HTML, "rc-visible") || strings.Contains(s.PlainText, "First draft") {
		t.Fatalf("the stored HTML isn't the new MJML's compile:\n%s", s.HTML)
	}
	hash := s.Hash

	// A new subject recompiles the email: its title follows.
	call("update_campaign_content", map[string]any{"campaign": "Agent's digest", "subject": "What shipped this week"}, &u)
	s = load(d.ID)
	if s.Version != 3 || s.Subject != "What shipped this week" || !strings.Contains(s.HTML, "<title>What shipped this week</title>") || !strings.Contains(s.HTML, "Second draft.") || s.Hash == hash {
		t.Fatalf("subject: %+v", s)
	}
	// The same content again stores nothing.
	call("update_campaign_content", map[string]any{"campaign": d.ID, "subject": "What shipped this week"}, &u)
	if s = load(d.ID); u.Version != 3 || s.Versions != 3 {
		t.Fatalf("unchanged: %+v %+v", u, s)
	}

	// MJML that doesn't compile is refused, and nothing changes.
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "update_campaign_content", Arguments: map[string]any{"campaign": d.ID, "mjml": `<mjml><mj-body><mj-section><mj-column><mj-text>Unclosed</mj-section></mj-body></mjml>`}})
	if err != nil || !res.IsError || !strings.Contains(text(res), "MJML") {
		t.Fatalf("broken MJML: %v %s", err, text(res))
	}
	if s = load(d.ID); s.Version != 3 || !strings.Contains(s.HTML, "Second draft.") {
		t.Fatalf("after a refused save: %+v", s)
	}

	// Each call is one audit entry, the agent's: the save doesn't add its own.
	var calls, saves int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE action = 'campaign.updated' AND actor_name = 'Drafts' AND result = 'allowed'),
			count(*) FILTER (WHERE action = 'campaign.content.save')
		FROM audit_events WHERE target_id = $1`, d.ID).Scan(&calls, &saves); err != nil || calls != 3 || saves != 0 {
		t.Fatalf("audit: %d tool calls, %d content saves, %v", calls, saves, err)
	}
	// The composer's history names the agent's versions in the reader's language.
	var history struct {
		Items []struct {
			Version       int
			Label, Source string
		}
	}
	c.Do("GET", ws+"/campaigns/"+d.ID+"/versions", nil).MustStatus(t, 200).JSON(t, &history)
	if len(history.Items) != 3 || history.Items[0].Label != "changed by an agent" || history.Items[2].Label != "created by an agent" || history.Items[0].Source != "api" {
		t.Fatalf("history: %+v", history.Items)
	}
}
