package assistant_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/ai/assistant"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/templates"
)

// demoFixture is the Acme demo with the assistant on the fake model: Linh
// (Owner) manages it, Sofia (Editor) uses it; draft is a draft campaign,
// sent a sent one with its report.
type demoFixture struct {
	*fixture
	draft, sent  uuid.UUID
	draftVersion int64
	draftSubject string
}

func newDemoFixture(t *testing.T) *demoFixture {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	f := &fixture{t: t, env: env, streams: newFakeStreams(), base: demoAI}
	f.model = fakeModel(t, f.streams)
	f.oc, f.mc = f.client("linh@acme.io"), f.client("sofia@acme.io")
	if err := env.App.Pool.QueryRow(ctx, `SELECT w.id, u.id FROM workspaces w, users u WHERE w.slug = 'acme-newsletter' AND u.email = 'sofia@acme.io'`).Scan(&f.ws.ID, &f.member); err != nil {
		t.Fatal(err)
	}
	ws, err := env.App.Q.GetWorkspace(ctx, f.ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.ws = ws
	f.oc.Do(http.MethodDelete, demoAI+"/connection", nil).MustStatus(t, 200)
	var s ai.AISettings
	f.oc.Do(http.MethodPut, demoAI+"/connection", map[string]any{"provider": "compatible", "baseUrl": f.model.URL, "model": "model-a", "capKind": "none",
		"inputPrice": 1, "outputPrice": 2}).MustStatus(t, 200).JSON(t, &s)
	f.accountID, f.modelID = s.Accounts[0].ID, s.Accounts[0].Models[0].ID
	f.passTools(f.oc, "/accounts/"+f.accountID.String()+"/models/"+f.modelID.String())
	f.settings = s.Assistant
	f.setAssistant(ai.AIAssistantSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 40, Tokens: 300000, CostMicros: 2000000}, PerPerson: 2, PerWorkspace: 10})
	d := &demoFixture{fixture: f}
	if err := env.App.Pool.QueryRow(ctx, `SELECT c.id, c.version, c.subject FROM campaigns c WHERE c.workspace_id = $1 AND c.channel = 'email' AND c.status = 'draft'
		AND c.deleted_at IS NULL AND c.subject <> '' ORDER BY c.id LIMIT 1`, f.ws.ID).Scan(&d.draft, &d.draftVersion, &d.draftSubject); err != nil {
		t.Fatal(err)
	}
	// Its content as a new campaign starts: what the editorial projection
	// takes (the demo's designed drafts use blocks it keeps to itself).
	starter, _ := templates.StarterMJML("blank", "")
	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET mjml = $2 WHERE id = $1`, d.draft, starter); err != nil {
		t.Fatal(err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT c.id FROM campaigns c WHERE c.workspace_id = $1 AND c.channel = 'email' AND c.status = 'sent'
		AND c.completed_at < now() - interval '4 days' AND c.deleted_at IS NULL ORDER BY c.completed_at DESC LIMIT 1`, f.ws.ID).Scan(&d.sent); err != nil {
		t.Fatal(err)
	}
	return d
}

// attach attaches resources to a thread and returns their handles.
func (f *fixture) attach(c *apptest.Client, thread uuid.UUID, kind string, id uuid.UUID) uuid.UUID {
	f.t.Helper()
	var th assistant.AIThread
	c.Do(http.MethodPost, f.base+"/assistant/threads/"+thread.String()+"/contexts", map[string]any{"kind": kind, "id": id, "sharingReviewed": true}).MustStatus(f.t, 200).JSON(f.t, &th)
	for _, x := range th.Contexts {
		if x.Kind == kind && x.TargetID == id {
			return x.ID
		}
	}
	f.t.Fatalf("%s %s isn't attached", kind, id)
	return uuid.Nil
}

// toolNames are a request's tools, in order.
func toolNames(body map[string]any) []string {
	out := []string{}
	tools, _ := body["tools"].([]any)
	for _, t := range tools {
		fn, _ := t.(map[string]any)["function"].(map[string]any)
		out = append(out, fn["name"].(string))
	}
	return out
}

// toolResults are the tool messages' contents of a request, by call id.
func toolResults(t *testing.T, body map[string]any) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	msgs, _ := body["messages"].([]any)
	for _, m := range msgs {
		mm := m.(map[string]any)
		if mm["role"] != "tool" {
			continue
		}
		var v map[string]any
		if err := json.Unmarshal([]byte(mm["content"].(string)), &v); err != nil {
			t.Fatalf("a tool result isn't JSON: %v", mm["content"])
		}
		out[mm["tool_call_id"].(string)] = v
	}
	return out
}

// A turn whose model reads what was attached: the tools run as the owner,
// their results go back as data from their source, sealed in the thread;
// each request of the turn has its own step and reservation; each call is
// a tool event and counts against the thread's ceiling.
func TestAssistantToolRound(t *testing.T) {
	d := newDemoFixture(t)
	ctx := context.Background()
	th := d.thread(d.mc, map[string]any{"screen": "campaign.report"})
	campaign := d.attach(d.mc, th.ID, "campaign", d.draft)
	report := d.attach(d.mc, th.ID, "report", d.sent)
	d.streams.then(calls("read_attached", `{"handle":"`+campaign.String()+`"}`, "read_report", `{"handle":"`+report.String()+`"}`), says("Opens held."))
	turn := d.send(d.mc, th.ID, "How did it do?").Turn.ID
	if err := d.worker().ExecuteTurn(ctx, d.job(turn)); err != nil {
		t.Fatal(err)
	}
	bodies := d.streams.streamed(t)
	if len(bodies) != 2 {
		t.Fatalf("%d requests, want 2", len(bodies))
	}
	names := toolNames(bodies[0])
	if !slices.Equal(names, []string{"search_help", "describe_screen", "suggest_link", "read_attached", "read_report", "propose_edit", "draft_agent_task", "subject_ideas", "rewrite"}) {
		t.Fatalf("tools offered: %v", names)
	}
	results := toolResults(t, bodies[1])
	read, rep := results["call_0"], results["call_1"]
	if read["ok"] != true || read["source"] != "read_attached" || !strings.Contains(mustJSON(t, read["data"]), d.draftSubject) ||
		rep["ok"] != true || rep["source"] != "read_report" || !strings.Contains(mustJSON(t, rep["data"]), "metrics") {
		t.Fatalf("the results the model read: %v", results)
	}
	after := d.get(d.mc, th.ID)
	roles := []string{}
	for _, m := range after.Messages {
		roles = append(roles, m.Role)
	}
	if !slices.Equal(roles, []string{"user", "tool", "assistant"}) || len(after.Messages[1].Tools) != 2 || !after.Messages[1].Tools[0].OK || !after.Messages[1].Tools[1].OK ||
		after.Messages[2].Text != "Opens held." || after.Usage.ToolCalls != 2 || after.Usage.Requests != 2 || after.Turn.State != "done" {
		t.Fatalf("after the turn: %+v", after)
	}
	if n := d.count(`SELECT count(*) FROM ai_assistant_steps s JOIN ai_budget_reservations b ON b.id = s.id JOIN ai_requests r ON r.id = s.id
		WHERE s.turn_id = $1 AND b.thread_id = $2 AND b.state = 'settled'`, turn, th.ID); n != 2 {
		t.Fatalf("%d steps with their own settled reservation, want 2", n)
	}
	if n := d.count(`SELECT count(*) FROM ai_assistant_events WHERE thread_id = $1 AND kind = 'tool'`, th.ID); n != 2 {
		t.Fatalf("%d tool events", n)
	}
	if n := d.count(`SELECT count(*) FROM ai_assistant_messages WHERE thread_id = $1 AND position(convert_to($2, 'UTF8') in content_sealed) > 0`, th.ID, d.draftSubject); n != 0 {
		t.Fatal("a tool result in plain text")
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// A feature a manager switched off is neither offered nor run: a model
// that calls it anyway reads not_available.
func TestAssistantToolsFollowTheSwitches(t *testing.T) {
	d := newDemoFixture(t)
	ctx := context.Background()
	th := d.thread(d.mc, map[string]any{})
	campaign := d.attach(d.mc, th.ID, "campaign", d.draft)
	if _, err := d.env.App.Pool.Exec(ctx, `UPDATE ai_workspace_settings SET features_off = ARRAY['draft', 'subject_ideas'] WHERE workspace_id = $1`, d.ws.ID); err != nil {
		t.Fatal(err)
	}
	d.streams.then(calls("propose_edit", `{"handle":"`+campaign.String()+`","baseRevision":1,"subject":"x","preheader":"","source":"<mjml></mjml>"}`), says("Done."))
	turn := d.send(d.mc, th.ID, "Fix the subject").Turn.ID
	if err := d.worker().ExecuteTurn(ctx, d.job(turn)); err != nil {
		t.Fatal(err)
	}
	bodies := d.streams.streamed(t)
	if names := toolNames(bodies[0]); slices.Contains(names, "propose_edit") || slices.Contains(names, "subject_ideas") || !slices.Contains(names, "rewrite") {
		t.Fatalf("tools offered with draft and subject ideas off: %v", names)
	}
	if r := toolResults(t, bodies[1])["call_0"]; r["ok"] != false || r["code"] != "not_available" {
		t.Fatalf("a switched-off tool called anyway: %v", r)
	}
	if n := d.count(`SELECT count(*) FROM ai_draft_proposals WHERE workspace_id = $1`, d.ws.ID); n != 0 {
		t.Fatalf("%d proposals", n)
	}
}

// propose_edit saves the model's edit through the drafts' review: the
// person accepts it in the composer at its exact revision, audited with
// the thread; the edit keeps the campaign's handles.
func TestAssistantProposeEdit(t *testing.T) {
	d := newDemoFixture(t)
	ctx := context.Background()
	th := d.thread(d.mc, map[string]any{})
	campaign := d.attach(d.mc, th.ID, "campaign", d.draft)
	var source string
	d.streams.then(calls("read_attached", `{"handle":"`+campaign.String()+`"}`),
		func(ctx context.Context, c *chunks, body []byte) {
			var b map[string]any
			_ = json.Unmarshal(body, &b)
			var read struct {
				Data struct {
					Campaign ai.AssistantCampaign `json:"campaign"`
				} `json:"data"`
			}
			raw, _ := json.Marshal(toolResults(t, b)["call_0"])
			_ = json.Unmarshal(raw, &read)
			source = `<mjml><mj-body><mj-section><mj-column><mj-text>Spring is here.</mj-text></mj-column></mj-section></mj-body></mjml>`
			args, _ := json.Marshal(map[string]any{"handle": campaign, "baseRevision": read.Data.Campaign.Revision, "subject": "A sharper subject",
				"preheader": read.Data.Campaign.Preheader, "source": source})
			calls("propose_edit", string(args))(ctx, c, body)
		}, says("I proposed a sharper subject."))
	turn := d.send(d.mc, th.ID, "Make the subject sharper").Turn.ID
	if err := d.worker().ExecuteTurn(ctx, d.job(turn)); err != nil {
		t.Fatal(err)
	}
	after := d.get(d.mc, th.ID)
	var row assistant.AIThreadToolRow
	for _, m := range after.Messages {
		for _, r := range m.Tools {
			if r.Name == "propose_edit" {
				row = r
			}
		}
	}
	if !row.OK || row.Card == nil || row.Card.Kind != "proposal" || row.Card.ProposalID == nil || row.Card.Path != "/w/acme-newsletter/campaigns/"+d.draft.String()+"/edit" {
		t.Fatalf("the proposal's row: %+v (source %d bytes)", row, len(source))
	}
	proposal := *row.Card.ProposalID
	if n := d.count(`SELECT count(*) FROM ai_draft_proposals WHERE id = $1 AND user_id = $2 AND target_id = $3 AND base_revision = $4 AND context->>'ThreadID' = $5`,
		proposal, d.member, d.draft, d.draftVersion, th.ID.String()); n != 1 {
		t.Fatal("the proposal isn't the drafts' own, on the campaign's revision, from the thread")
	}
	// Nothing changed yet; accepting it is the person's, in the drafts' review.
	if n := d.count(`SELECT count(*) FROM campaigns WHERE id = $1 AND subject = 'A sharper subject'`, d.draft); n != 0 {
		t.Fatal("the proposal changed the campaign")
	}
	d.oc.Do(http.MethodPost, demoAI+"/drafts/campaigns/"+proposal.String()+"/decision", map[string]any{"decision": "accepted"}).MustStatus(t, 404)
	d.mc.Do(http.MethodPost, demoAI+"/drafts/campaigns/"+proposal.String()+"/decision", map[string]any{"decision": "accepted"}).MustStatus(t, 200)
	if n := d.count(`SELECT count(*) FROM campaigns WHERE id = $1 AND subject = 'A sharper subject' AND version = $2`, d.draft, d.draftVersion+1); n != 1 {
		t.Fatal("accepting didn't save the proposal at the next revision")
	}
	if n := d.count(`SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'campaign.content.save' AND target_id = $2
		AND metadata->>'via' = 'ai_assistant' AND metadata->>'thread_id' = $3 AND metadata->>'proposal_id' = $4`, d.ws.ID, d.draft.String(), th.ID.String(), proposal.String()); n != 1 {
		t.Fatal("the acceptance isn't audited with the thread")
	}
	if n := d.count(`SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'ai.draft.propose' AND target_id = $2 AND metadata->>'thread_id' = $3`,
		d.ws.ID, proposal.String(), th.ID.String()); n != 1 {
		t.Fatal("the proposal isn't audited with the thread")
	}
}

// The one-shot suggestions run through their own services: their own
// request, reservation and activity row; the agent task is a card only.
func TestAssistantOneShotsAndHandoffs(t *testing.T) {
	d := newDemoFixture(t)
	ctx := context.Background()
	th := d.thread(d.mc, map[string]any{})
	campaign := d.attach(d.mc, th.ID, "campaign", d.draft)
	d.streams.then(calls("subject_ideas", `{"handle":"`+campaign.String()+`"}`, "draft_agent_task", `{"title":"Spring series","brief":"Draft three spring emails"}`,
		"suggest_link", `{"screen":"campaign.report","id":"`+d.sent.String()+`"}`, "suggest_link", `{"screen":"billing"}`), says("Here you go."))
	turn := d.send(d.mc, th.ID, "Ideas?").Turn.ID
	runs := d.count(`SELECT count(*) FROM ai_agent_runs WHERE workspace_id = $1`, d.ws.ID)
	if err := d.worker().ExecuteTurn(ctx, d.job(turn)); err != nil {
		t.Fatal(err)
	}
	results := toolResults(t, d.streams.streamed(t)[1])
	if r := results["call_0"]; r["ok"] != true || !strings.Contains(mustJSON(t, r["data"]), "Fresh ideas for spring") {
		t.Fatalf("subject ideas: %v", r)
	}
	if n := d.count(`SELECT count(*) FROM ai_requests r JOIN ai_budget_reservations b ON b.id = r.id WHERE r.workspace_id = $1 AND r.user_id = $2
		AND r.feature = 'subject_ideas' AND b.thread_id IS NULL`, d.ws.ID, d.member); n != 1 {
		t.Fatalf("%d subject ideas requests of their own", n)
	}
	if r := results["call_3"]; r["ok"] != false || r["code"] != "unknown_screen" {
		t.Fatalf("a screen the app doesn't know: %v", r)
	}
	rows := map[string]*assistant.AIThreadCard{}
	for _, m := range d.get(d.mc, th.ID).Messages {
		for _, r := range m.Tools {
			if r.Card != nil {
				rows[r.Card.Kind] = r.Card
			}
		}
	}
	if c := rows["agent_task"]; c == nil || c.Brief != "Draft three spring emails" || c.Path != "/w/acme-newsletter/agent/new" {
		t.Fatalf("the agent task card: %+v", c)
	}
	if c := rows["link"]; c == nil || c.Path != "/w/acme-newsletter/campaigns/"+d.sent.String()+"/report" {
		t.Fatalf("the link card: %+v", c)
	}
	if n := d.count(`SELECT count(*) FROM ai_agent_runs WHERE workspace_id = $1`, d.ws.ID); n != runs {
		t.Fatal("drafting an agent task started a run")
	}
}

// The tool-call ceiling holds before each call and each request: a call
// past it doesn't run, and the next request offers no tools.
func TestAssistantToolCeiling(t *testing.T) {
	f := newFixture(t)
	f.setAssistant(ai.AIAssistantSettings{Enabled: true, Limits: ai.AIAgentLimits{ToolCalls: 1, Tokens: 300000, CostMicros: 2000000}, PerPerson: 2, PerWorkspace: 10})
	th := f.thread(f.mc, map[string]any{})
	f.streams.then(calls("search_help", `{"query":"dkim"}`, "search_help", `{"query":"segments"}`), says("Done."))
	turn := f.send(f.mc, th.ID, "Explain sending domains").Turn.ID
	if err := f.worker().ExecuteTurn(context.Background(), f.job(turn)); err != nil {
		t.Fatal(err)
	}
	bodies := f.streams.streamed(t)
	results := toolResults(t, bodies[1])
	if results["call_0"]["ok"] != true || !strings.Contains(mustJSON(t, results["call_0"]["data"]), "DKIM") || results["call_1"]["code"] != "tool_limit" {
		t.Fatalf("results: %v", results)
	}
	if names := toolNames(bodies[1]); len(names) != 0 {
		t.Fatalf("tools offered past the ceiling: %v", names)
	}
	if after := f.get(f.mc, th.ID); after.Usage.ToolCalls != 1 || after.Turn.State != "done" {
		t.Fatalf("after: %+v", after)
	}
}
