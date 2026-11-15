package mcpconnect_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/mcpconnect"
	"github.com/hostrepute/reputecast/internal/segments"
)

func externalFlow(trigger automations.AutomationTrigger) *mcpconnect.MCPProposedFlow {
	return &mcpconnect.MCPProposedFlow{
		Emails: []automations.ProposedEmail{{Key: "email1", Name: "First note", Subject: "A clearer update", MJML: proposedMJML}},
		Graph: automations.AutomationGraph{
			Nodes: []automations.AutomationNode{
				{ID: "start", Type: "trigger", Data: automations.AutomationNodeData{Trigger: &trigger}},
				{ID: "first", Type: "email", Data: automations.AutomationNodeData{Email: &automations.AutomationEmailStep{EmailID: "email1", EmailVersion: 1, Subject: "A clearer update"}}},
				{ID: "end", Type: "exit"},
			},
			Edges: []automations.AutomationEdge{{ID: "a", Source: "start", Target: "first"}, {ID: "b", Source: "first", Target: "end"}},
		},
	}
}

func TestExternalWorkflowBundle(t *testing.T) {
	f := newFixture(t)
	client := f.client("none")
	code, verifier := f.allow(client)
	tokens := f.exchange(client, code, verifier)
	cs := f.mcp(tokens.AccessToken)
	var grant uuid.UUID
	if err := f.env.App.Pool.QueryRow(t.Context(), `SELECT grant_id FROM mcp_connect_tokens WHERE hash=$1`, authn.HashToken(tokens.AccessToken)).Scan(&grant); err != nil {
		t.Fatal(err)
	}
	base := "/workspaces/" + f.ws.Slug
	definition := segments.SegmentDefinition{Groups: []segments.SegmentGroup{{Match: "all", Conditions: []segments.SegmentCondition{
		{Field: "tag", Op: "has", Value: "LOCAL-ONLY-VIP-COHORT"}, {Field: "consent", Op: "is", Value: "double"}, {Field: "clicked", Op: "not_in_last", Amount: 60, Unit: "days"},
	}}}}
	var segment struct{ ID uuid.UUID }
	f.human.Do("POST", base+"/segments", map[string]any{"name": "PRIVATE-SOURCE-NAME", "mode": "builder", "refresh": "dynamic", "definition": definition}).MustStatus(t, 201).JSON(t, &segment)
	handle := f.selectSource(grant, "segment", segment.ID)
	projected := toolResult[struct{ Segment segments.SegmentDefinition }](t, cs, "read_source", map[string]any{"source": handle})
	raw, _ := json.Marshal(projected)
	for _, secret := range []string{"LOCAL-ONLY-VIP-COHORT", "double", segment.ID.String(), "PRIVATE-SOURCE-NAME"} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("segment context disclosed a bound value or identity")
		}
	}
	if len(projected.Segment.Groups) != 1 || projected.Segment.Groups[0].Conditions[2].Amount != 60 {
		t.Fatal("projected segment lost its safe semantics")
	}
	in := mcpconnect.MCPWorkflowInput{Key: "bundle", Name: "Re-engagement", Source: handle, Trigger: "segment_entered", Segment: &projected.Segment, SegmentName: "Quiet readers",
		Flow: externalFlow(automations.AutomationTrigger{Kind: "segment_entered", SegmentID: "selected_segment", Reentry: "once"})}
	proposal := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_workflow", in)
	if retry := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_workflow", in); retry.ID != proposal.ID {
		t.Fatal("bundle retry duplicated proposal")
	}
	view := f.proposal(proposal.ID)
	if view.Kind != "automation" || view.Workflow == nil || view.Workflow.Segment.Groups[0].Conditions[0].Value != "LOCAL-ONLY-VIP-COHORT" || len(view.Workflow.Previews) != 1 {
		t.Fatal("human review did not restore audience and email preview")
	}
	if f.scalar(`SELECT count(*) FROM segments`) != 1 || f.scalar(`SELECT count(*) FROM automations`) != 0 || f.scalar(`SELECT count(*) FROM automation_emails`) != 0 {
		t.Fatal("staging changed product resources")
	}
	f.decide(proposal.ID, "kept").MustStatus(t, 204)
	f.decide(proposal.ID, "kept").MustStatus(t, 204)
	view = f.proposal(proposal.ID)
	if view.ResultID == nil || view.Decision != "kept" || f.scalar(`SELECT count(*) FROM segments`) != 2 || f.scalar(`SELECT count(*) FROM automations WHERE status='draft' AND live_version IS NULL`) != 1 || f.scalar(`SELECT count(*) FROM automation_emails`) != 1 {
		t.Fatal("Keep failed or duplicated dependency bundle")
	}
	var graphRaw []byte
	if err := f.env.App.Pool.QueryRow(t.Context(), `SELECT graph FROM automation_versions WHERE automation_id=$1`, *view.ResultID).Scan(&graphRaw); err != nil {
		t.Fatal(err)
	}
	graph, err := automations.ParseGraph(graphRaw)
	if err != nil || graph.Trigger().Data.Trigger.SegmentID == segment.ID.String() || graph.Trigger().Data.Trigger.SegmentID == "selected_segment" || graph.Node("first").Data.Email.EmailID == "email1" {
		t.Fatal("Keep did not bind the newly created segment and email")
	}
	for _, table := range []string{"messages", "automation_runs", "segment_members", "segment_member_events", "ai_requests", "ai_budget_reservations"} {
		if f.scalar(`SELECT count(*) FROM `+table) != 0 {
			t.Fatalf("Keep touched %s", table)
		}
	}
	if f.scalar(`SELECT count(*) FROM audit_events WHERE actor_type='user' AND metadata->>'via'='external_mcp' AND action IN ('segment.create','automation.create','automation_email.create')`) != 3 {
		t.Fatal("human domain provenance missing")
	}
	// The bundle needs both edit permissions, including after it was staged.
	in.Key = "permission"
	denied := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_workflow", in)
	if _, err := f.env.App.Pool.Exec(t.Context(), `UPDATE mcp_connect_grants SET permissions=array_remove(permissions,'segments:edit') WHERE id=$1`, grant); err != nil {
		t.Fatal(err)
	}
	if f.proposal(denied.ID).Blocked != "permission" {
		t.Fatal("bundle lost its segment authorization requirement")
	}
	f.decide(denied.ID, "kept").MustStatus(t, 409)
	if _, err := f.env.App.Pool.Exec(t.Context(), `UPDATE mcp_connect_grants SET permissions=array_append(permissions,'segments:edit') WHERE id=$1`, grant); err != nil {
		t.Fatal(err)
	}
	f.decide(denied.ID, "discarded").MustStatus(t, 204)
	// No partial Keep when the last domain write fails after segment creation.
	in.Key = "rollback"
	failed := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_workflow", in)
	if _, err := f.env.App.Pool.Exec(t.Context(), `CREATE FUNCTION fail_proposed_flow() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected domain failure'; END $$; CREATE TRIGGER fail_proposed_flow BEFORE INSERT ON automations FOR EACH ROW EXECUTE FUNCTION fail_proposed_flow()`); err != nil {
		t.Fatal(err)
	}
	f.decide(failed.ID, "kept").MustStatus(t, 500)
	if _, err := f.env.App.Pool.Exec(t.Context(), `DROP TRIGGER fail_proposed_flow ON automations; DROP FUNCTION fail_proposed_flow()`); err != nil {
		t.Fatal(err)
	}
	if f.scalar(`SELECT count(*) FROM segments`) != 2 || f.proposal(failed.ID).Decision != "" {
		t.Fatal("failed Keep left a segment or a decision behind")
	}
	f.decide(failed.ID, "discarded").MustStatus(t, 204)
	// A source edit fences both a standalone segment and a dependency bundle.
	in.Key, in.Flow, in.Trigger = "segment-copy", nil, ""
	stale := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_workflow", in)
	if _, err := f.env.App.Pool.Exec(t.Context(), `UPDATE segments SET definition=jsonb_set(definition,'{groups,0,conditions,2,amount}','90') WHERE id=$1`, segment.ID); err != nil {
		t.Fatal(err)
	}
	if f.proposal(stale.ID).Blocked != "changed" {
		t.Fatal("source change did not invalidate proposal")
	}
	f.decide(stale.ID, "kept").MustStatus(t, 409)
	f.decide(stale.ID, "discarded").MustStatus(t, 204)
	// Re-selection does not permit injecting raw values or subscription steps.
	in.Source = f.selectSource(grant, "segment", segment.ID)
	projected = toolResult[struct{ Segment segments.SegmentDefinition }](t, cs, "read_source", map[string]any{"source": in.Source})
	in.Segment = &projected.Segment
	in.Segment.Groups[0].Conditions[0].Value = "unselected-private-value"
	out, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: "propose_workflow", Arguments: in})
	if err != nil || !out.IsError {
		t.Fatal("accepted an unbound audience literal")
	}
}

func TestExternalFlowRevisionAndBoundary(t *testing.T) {
	f := newFixture(t)
	client := f.client("none")
	code, verifier := f.allow(client)
	tokens := f.exchange(client, code, verifier)
	cs := f.mcp(tokens.AccessToken)
	var grant uuid.UUID
	if err := f.env.App.Pool.QueryRow(t.Context(), `SELECT id FROM mcp_connect_grants WHERE client_id=$1`, client.ID).Scan(&grant); err != nil {
		t.Fatal(err)
	}
	list := uuid.Must(uuid.NewV7())
	if _, err := f.env.App.Pool.Exec(t.Context(), `INSERT INTO lists(id,workspace_id,name) VALUES($1,$2,'LOCAL-LIST-TITLE')`, list, f.ws.ID); err != nil {
		t.Fatal(err)
	}
	handle := f.selectSource(grant, "list", list)
	trigger := automations.AutomationTrigger{Kind: "joined_list", ListID: "selected_list", ConfirmedOnly: true, ExitOnLeave: true, Reentry: "once"}
	in := mcpconnect.MCPWorkflowInput{Key: "list-flow", Name: "Welcome", Source: handle, Trigger: "joined_list", Flow: externalFlow(trigger)}
	for _, wrong := range []string{"subscription", "wrong-trigger", "raw-id", "unknown-email"} {
		bad := in
		bad.Flow = externalFlow(trigger)
		switch wrong {
		case "subscription":
			bad.Flow.Graph.Nodes[1].Type = "list"
		case "wrong-trigger":
			bad.Flow.Graph.Trigger().Data.Trigger.IncludeImports = true
		case "raw-id":
			bad.Flow.Graph.Trigger().Data.Trigger.ListID = list.String()
		case "unknown-email":
			bad.Flow.Graph.Nodes[1].Data.Email.EmailID = uuid.Must(uuid.NewV7()).String()
		}
		out, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: "propose_workflow", Arguments: bad})
		if err != nil || !out.IsError {
			t.Fatalf("accepted %s", wrong)
		}
	}
	first := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_workflow", in)
	f.decide(first.ID, "kept").MustStatus(t, 204)
	view := f.proposal(first.ID)
	if view.TriggerName != "LOCAL-LIST-TITLE" || view.ResultID == nil {
		t.Fatal("human review lost the selected audience")
	}
	source := f.selectSource(grant, "automation", *view.ResultID)
	read := toolResult[struct{ Flow mcpconnect.MCPFlowSource }](t, cs, "read_source", map[string]any{"source": source})
	raw, _ := json.Marshal(read)
	if strings.Contains(string(raw), list.String()) || strings.Contains(string(raw), "LOCAL-LIST-TITLE") || strings.Contains(string(raw), view.ResultID.String()) {
		t.Fatal("flow projection exposed local identifiers")
	}
	if read.Flow.Graph.Trigger().Data.Trigger.ListID != "selected_list" || len(read.Flow.Emails) != 1 {
		t.Fatal("flow projection lost its trigger or email")
	}
	mail := read.Flow.Emails["email1"]
	change := mcpconnect.MCPWorkflowInput{Key: "change", Name: "Welcome", Source: source, Change: true, Flow: &mcpconnect.MCPProposedFlow{
		Graph: read.Flow.Graph, Emails: []automations.ProposedEmail{{Key: "email1", Name: "Revised welcome", Subject: mail.Subject, Preheader: mail.Preheader, MJML: proposedMJML}},
	}}
	accepted := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_workflow", change)
	change.Key = "stale"
	stale := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_workflow", change)
	before := f.proposal(accepted.ID)
	if before.BeforeFlow == nil || len(before.BeforeFlow.Emails) != 1 || before.BeforeFlow.Emails[0].Subject != "A clearer update" {
		t.Fatal("exact before revision unavailable for human comparison")
	}
	f.decide(accepted.ID, "accepted").MustStatus(t, 204)
	f.decide(accepted.ID, "accepted").MustStatus(t, 204)
	if f.scalar(`SELECT draft_revision FROM automations WHERE id=$1`, *view.ResultID) != 2 || f.scalar(`SELECT count(*) FROM automation_emails`) != 2 {
		t.Fatal("Accept duplicated a revision or did not create a separate email")
	}
	if f.proposal(stale.ID).Blocked != "changed" {
		t.Fatal("another accepted proposal did not stale the prior revision")
	}
	f.decide(stale.ID, "accepted").MustStatus(t, 409)
	f.decide(stale.ID, "rejected").MustStatus(t, 204)
	// A separate grant cannot resolve the source, even for the same person.
	other := f.client("none")
	otherCode, otherVerifier := f.allow(other)
	otherTokens := f.exchange(other, otherCode, otherVerifier)
	out, err := f.mcp(otherTokens.AccessToken).CallTool(t.Context(), &mcp.CallToolParams{Name: "read_source", Arguments: map[string]any{"source": handle}})
	if err != nil || !out.IsError {
		t.Fatal("source crossed grants")
	}
	in.Key = "revoked"
	revoked := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_workflow", in)
	f.human.Do("DELETE", f.base+"/connections/"+grant.String(), nil).MustStatus(t, 204)
	if f.proposal(revoked.ID).Blocked != "revoked" {
		t.Fatal("revoked workflow remained applicable")
	}
	f.decide(revoked.ID, "kept").MustStatus(t, 409)
	if f.scalar(`SELECT count(*) FROM automation_runs`) != 0 || f.scalar(`SELECT count(*) FROM messages`) != 0 || f.scalar(`SELECT count(*) FROM ai_requests`) != 0 {
		t.Fatal("workflow crossed sending, enrollment or billing boundary")
	}
}
