package mcpconnect_test

import (
	"encoding/json"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/mcpconnect"
	"github.com/hostrepute/reputecast/internal/reports"
)

func TestExternalReportSnapshots(t *testing.T) {
	f := newFixture(t)
	client := f.client("none")
	connect := func(scopes ...string) (uuid.UUID, *mcp.ClientSession) {
		t.Helper()
		values, verifier := f.authorization(client)
		handle := f.pending(values)
		f.human.Do("GET", "/mcp-connect/consent?request="+handle, nil).MustStatus(t, 200)
		body := f.consentBody(handle)
		body.Scopes = scopes
		var consent mcpconnect.MCPConsentResult
		f.human.Do("POST", f.base+"/consent", body).MustStatus(t, 200).JSON(t, &consent)
		redirect, _ := url.Parse(consent.Redirect)
		tokens := f.exchange(client, redirect.Query().Get("code"), verifier)
		var grant uuid.UUID
		if err := f.env.App.Pool.QueryRow(t.Context(), `SELECT grant_id FROM mcp_connect_tokens WHERE hash=$1`, authn.HashToken(tokens.AccessToken)).Scan(&grant); err != nil {
			t.Fatal(err)
		}
		return grant, f.mcp(tokens.AccessToken)
	}
	grant, cs := connect("reports:read")
	create := func(sent int, age time.Duration) uuid.UUID {
		t.Helper()
		id := uuid.Must(uuid.NewV7())
		end := time.Now().Add(-age)
		if _, err := f.env.App.Pool.Exec(t.Context(), `INSERT INTO campaigns(id,workspace_id,name,subject,mjml,status,started_at,completed_at)
VALUES($1,$2,'CONTACT_NAME_SENTINEL','CONTACT_ADDRESS_SENTINEL@example.test','<mjml>https://private.example/SECRET_SENTINEL</mjml>','sent',$3,$4)`, id, f.ws.ID, end.Add(-time.Hour), end); err != nil {
			t.Fatal(err)
		}
		if _, err := f.env.App.Pool.Exec(t.Context(), `INSERT INTO campaign_counters(campaign_id,provider_id,mailbox_provider,sent,delivered,opened_human,clicked,unsubscribed,hard_bounced)
VALUES($1,$2,'PRIVATE_MAILBOX_SENTINEL',$3::bigint,$3::bigint-19,79,20,0,19)`, id, uuid.New(), sent); err != nil {
			t.Fatal(err)
		}
		return id
	}
	deny := func(session *mcp.ClientSession, tool string, in any) {
		t.Helper()
		out, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: tool, Arguments: in})
		if err != nil || !out.IsError {
			t.Fatalf("%s accepted forbidden context: %v %+v", tool, err, out)
		}
		raw, _ := json.Marshal(out)
		if strings.Contains(string(raw), "SENTINEL") {
			t.Fatal("error disclosed content")
		}
	}

	for _, tc := range []struct {
		sent    int
		status  string
		metrics map[string]int64
	}{
		{99, "withheld", map[string]int64{}},
		{100, "rounded", map[string]int64{"humanOpens": 70, "humanClicks": 20}},
		{137, "rounded", map[string]int64{"humanOpens": 70, "humanClicks": 20}},
	} {
		id := create(tc.sent, 72*time.Hour)
		handle := f.selectSource(grant, "report", id)
		out := toolResult[reports.ModelSnapshot](t, cs, "read_report", map[string]any{"source": handle})
		if out.Status != tc.status || !reflect.DeepEqual(out.Metrics, tc.metrics) || out.Rounding != 10 || out.AsOf.IsZero() {
			t.Fatalf("cohort %d: %+v", tc.sent, out)
		}
		if (tc.sent < 100 && out.Sent != nil) || (tc.sent >= 100 && (out.Sent == nil || *out.Sent != int64(tc.sent/10*10))) {
			t.Fatal("incorrect rounded or suppressed sent count")
		}
		raw, _ := json.Marshal(out)
		if strings.Contains(string(raw), "SENTINEL") || strings.Contains(string(raw), id.String()) || strings.Contains(string(raw), "example") {
			t.Fatal("report disclosed content, dimensions or local identifiers")
		}
		deny(cs, "read_source", map[string]any{"source": handle})
		deny(cs, "read_report", map[string]any{"source": id.String()})
		deny(cs, "read_report", map[string]any{"source": handle, "filter": "CONTACT_ADDRESS_SENTINEL@example.test"})
	}
	recent := create(400, 71*time.Hour)
	f.human.Do("POST", f.base+"/connections/"+grant.String()+"/sources", mcpconnect.MCPSourceSelection{SharingReviewed: true, Kind: "report", ID: recent}).MustStatus(t, 409)
	if f.scalar(`SELECT count(*) FROM report_model_snapshots WHERE campaign_id=$1`, recent) != 0 {
		t.Fatal("live report persisted a snapshot")
	}
	id := create(237, 80*time.Hour)
	handle := f.selectSource(grant, "report", id)
	first := toolResult[reports.ModelSnapshot](t, cs, "read_report", map[string]any{"source": handle})
	if _, err := f.env.App.Pool.Exec(t.Context(), `UPDATE campaign_counters SET sent=900,delivered=840,opened_human=700,clicked=200 WHERE campaign_id=$1`, id); err != nil {
		t.Fatal(err)
	}
	grant2, other := connect("reports:read")
	deny(other, "read_report", map[string]any{"source": handle})
	otherHandle := f.selectSource(grant2, "report", id)
	second := toolResult[reports.ModelSnapshot](t, other, "read_report", map[string]any{"source": otherHandle})
	if !reflect.DeepEqual(first, second) || f.scalar(`SELECT count(*) FROM report_model_snapshots WHERE campaign_id=$1`, id) != 1 {
		t.Fatal("reconnect disclosed a changed snapshot")
	}
	list := toolResult[struct {
		Sources []mcpconnect.MCPSourceHandle `json:"sources"`
	}](t, other, "list_sources", map[string]any{})
	if len(list.Sources) != 1 || list.Sources[0].Kind != "report" {
		t.Fatal("reports-only discovery failed")
	}
	// Editorial consent alone cannot authorize report reads or selection.
	editorialGrant, editorial := connect("editorial:read")
	f.human.Do("POST", f.base+"/connections/"+editorialGrant.String()+"/sources", mcpconnect.MCPSourceSelection{SharingReviewed: true, Kind: "report", ID: id}).MustStatus(t, 409)
	deny(editorial, "read_report", map[string]any{"source": handle})
	f.human.Do("POST", f.base+"/connections/"+grant.String()+"/sources", mcpconnect.MCPSourceSelection{SharingReviewed: true, Kind: "campaign", ID: id}).MustStatus(t, 409)

	// An old disclosure cannot be reconstructed, even before its cleanup
	// worker runs. Cleanup retains a content-free tombstone, not its counts.
	expired := create(500, 80*time.Hour)
	f.selectSource(grant, "report", expired)
	if _, err := f.env.App.Pool.Exec(t.Context(), `UPDATE report_model_snapshots SET created_at=now()-interval '30 days' WHERE campaign_id=$1`, expired); err != nil {
		t.Fatal(err)
	}
	f.human.Do("POST", f.base+"/connections/"+grant2.String()+"/sources", mcpconnect.MCPSourceSelection{SharingReviewed: true, Kind: "report", ID: expired}).MustStatus(t, 409)
	worker := reports.ModelRetentionWorker{S: &reports.Service{Q: f.env.App.Q}}
	if err := worker.Work(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if f.scalar(`SELECT count(*) FROM report_model_snapshots WHERE campaign_id=$1 AND content IS NULL`, expired) != 1 ||
		f.scalar(`SELECT count(*) FROM report_model_snapshots WHERE campaign_id=$1 AND content IS NOT NULL`, id) != 1 {
		t.Fatal("retention removed a live snapshot or retained expired counts")
	}
	f.human.Do("POST", f.base+"/connections/"+grant2.String()+"/sources", mcpconnect.MCPSourceSelection{SharingReviewed: true, Kind: "report", ID: expired}).MustStatus(t, 409)

	allowed, hidden := uuid.New(), uuid.New()
	for _, listID := range []uuid.UUID{allowed, hidden} {
		if _, err := f.env.App.Pool.Exec(t.Context(), `INSERT INTO lists(id,workspace_id,name) VALUES($1,$2,'List')`, listID, f.ws.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.env.App.Pool.Exec(t.Context(), `UPDATE campaigns SET audience=jsonb_build_object('lists',jsonb_build_array($1::text)) WHERE id=$2`, hidden.String(), id); err != nil {
		t.Fatal(err)
	}
	if _, err := f.env.App.Pool.Exec(t.Context(), `UPDATE workspace_members SET scope=jsonb_build_object('lists',jsonb_build_array($1::text)) WHERE workspace_id=$2 AND user_id=$3`, allowed.String(), f.ws.ID, f.owner.ID); err != nil {
		t.Fatal(err)
	}
	deny(other, "read_report", map[string]any{"source": otherHandle})
	list = toolResult[struct {
		Sources []mcpconnect.MCPSourceHandle `json:"sources"`
	}](t, other, "list_sources", map[string]any{})
	if len(list.Sources) != 0 {
		t.Fatal("cached snapshot bypassed current campaign scope")
	}
	for _, table := range []string{"ai_requests", "ai_budget_reservations", "ai_agent_runs", "messages", "report_shares"} {
		if f.scalar(`SELECT count(*) FROM `+table) != 0 {
			t.Fatalf("external report touched %s", table)
		}
	}
}
