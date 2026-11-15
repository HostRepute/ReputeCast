package mcpconnect_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authn"
	"github.com/hostrepute/reputecast/internal/brandkit"
	"github.com/hostrepute/reputecast/internal/mcpconnect"
)

const proposedMJML = `<mjml><mj-body><mj-section><mj-column><mj-text>A clearer editorial update.</mj-text></mj-column></mj-section></mj-body></mjml>`

func toolResult[T any](t *testing.T, s *mcp.ClientSession, name string, input any) T {
	t.Helper()
	out, err := s.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: input})
	if err != nil || out.IsError {
		t.Fatalf("%s failed: %v %+v", name, err, out)
	}
	raw, err := json.Marshal(out.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var result T
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func (f *fixture) proposal(id uuid.UUID) mcpconnect.MCPProposalView {
	f.t.Helper()
	var out mcpconnect.MCPProposalView
	f.human.Do("GET", f.base+"/proposals/"+id.String(), nil).MustStatus(f.t, 200).JSON(f.t, &out)
	return out
}

func (f *fixture) decide(id uuid.UUID, decision string) *apptest.Response {
	f.t.Helper()
	return f.human.Do("POST", f.base+"/proposals/"+id.String()+"/decision", mcpconnect.MCPProposalDecision{Decision: decision})
}

func (f *fixture) scalar(sql string, args ...any) int {
	f.t.Helper()
	var out int
	if err := f.env.App.Pool.QueryRow(f.t.Context(), sql, args...).Scan(&out); err != nil {
		f.t.Fatal(err)
	}
	return out
}

func (f *fixture) selectSource(grant uuid.UUID, kind string, id uuid.UUID) string {
	f.t.Helper()
	var out mcpconnect.MCPSourceHandle
	f.human.Do("POST", f.base+"/connections/"+grant.String()+"/sources", mcpconnect.MCPSourceSelection{SharingReviewed: true, Kind: kind, ID: id}).MustStatus(f.t, 200).JSON(f.t, &out)
	if out.Handle == "" || out.Kind != kind {
		f.t.Fatal("missing source handle")
	}
	return out.Handle
}

func TestExternalProposalsLifecycle(t *testing.T) {
	for _, kind := range []string{"campaign", "post"} {
		t.Run(kind, func(t *testing.T) {
			f := newFixture(t)
			client := f.client("none")
			code, verifier := f.allow(client)
			tokens := f.exchange(client, code, verifier)
			cs := f.mcp(tokens.AccessToken)
			in := mcpconnect.MCPProposalInput{Key: "first-draft", Kind: kind, Name: "Editorial draft", Email: mcpconnect.MCPProposedEmail{Subject: "A clearer update", Preheader: "Reviewed before use", MJML: proposedMJML}}
			first := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_email", in)
			retry := toolResult[mcpconnect.MCPProposalReceipt](t, f.mcp(tokens.AccessToken), "propose_email", in)
			if first.ID == uuid.Nil || retry.ID != first.ID || retry.Decision != "" {
				t.Fatal("retry did not return the same staged proposal")
			}
			if f.scalar(`SELECT count(*) FROM campaigns`) != 0 || f.scalar(`SELECT count(*) FROM website_posts`) != 0 {
				t.Fatal("external proposal changed product resources")
			}
			for _, edit := range []string{
				`content=jsonb_set(content, '{subject}', '"Unreviewed replacement"')`,
				`preview='<p>Unreviewed replacement</p>'`,
				`context=jsonb_set(context, '{Draft,Name}', '"Replacement"')`,
				`expires_at=expires_at+interval '1 hour'`,
				`purge_at=purge_at+interval '1 day'`,
			} {
				_, err := f.env.App.Pool.Exec(t.Context(), `UPDATE mcp_connect_proposals SET `+edit+` WHERE id=$1`, first.ID)
				var pg *pgconn.PgError
				if !errors.As(err, &pg) || pg.Code != "23514" {
					t.Fatalf("stored review was mutable: %s: %v", edit, err)
				}
			}
			in.Email.Subject = "Different payload"
			out, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: "propose_email", Arguments: in})
			if err != nil || !out.IsError || out.Content[0].(*mcp.TextContent).Text != "idempotency_conflict" {
				t.Fatalf("changed retry: %v %+v", err, out)
			}
			view := f.proposal(first.ID)
			if view.Email.Subject != "A clearer update" || view.HTML == "" || view.Blocked != "" || view.Before != nil || view.TargetID != nil {
				t.Fatalf("review does not match staged content: %+v", view)
			}
			f.decide(first.ID, "accepted").MustStatus(t, 409)
			f.decide(first.ID, "kept").MustStatus(t, 204)
			f.decide(first.ID, "kept").MustStatus(t, 204)
			f.decide(first.ID, "discarded").MustStatus(t, 409)
			view = f.proposal(first.ID)
			if view.Decision != "kept" || view.ResultID == nil {
				t.Fatal("Keep did not persist its decision and resource")
			}
			if kind == "campaign" {
				if f.scalar(`SELECT count(*) FROM campaigns WHERE status='draft' AND subject='A clearer update' AND sender_id IS NULL`) != 1 || f.scalar(`SELECT count(*) FROM campaign_versions`) != 1 {
					t.Fatal("Keep duplicated or changed the campaign")
				}
			} else if f.scalar(`SELECT count(*) FROM website_posts WHERE status='draft' AND title='A clearer update' AND first_published_at IS NULL`) != 1 || f.scalar(`SELECT count(*) FROM website_post_versions`) != 1 {
				t.Fatal("Keep published or duplicated the post")
			}
			status := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "proposal_status", map[string]any{"id": first.ID})
			if status.Decision != "kept" {
				t.Fatal("external receipt did not reflect the decision")
			}
			if f.scalar(`SELECT count(*) FROM audit_events WHERE action='mcp_connect.proposed' AND actor_type='mcp_agent' AND metadata->>'via'='external_mcp'`) != 1 ||
				f.scalar(`SELECT count(*) FROM audit_events WHERE action='mcp_connect.kept' AND actor_type='user' AND actor_id=$1 AND metadata->>'proposal_id'=$2`, f.owner.ID, first.ID.String()) != 1 {
				t.Fatal("proposal/human audit attribution was lost or duplicated")
			}
			if f.scalar(`SELECT count(*) FROM notifications WHERE kind='external_proposal' AND user_id=$1 AND email_status='pending' AND link=$2`,
				f.owner.ID, "/w/"+f.ws.Slug+"/proposals/"+first.ID.String()) != 1 {
				t.Fatal("proposal notification was lost, duplicated or sent to another person")
			}
			var notices struct {
				Items []struct{ Title, Body string }
			}
			f.human.Header = http.Header{"Accept-Language": {"vi"}}
			f.human.Do("GET", "/workspaces/"+f.ws.Slug+"/notifications?filter=integrations", nil).MustStatus(t, 200).JSON(t, &notices)
			if len(notices.Items) != 1 || notices.Items[0].Title != "Đề xuất bên ngoài sẵn sàng để duyệt" || strings.Contains(notices.Items[0].Body, in.Email.Subject) || strings.Contains(notices.Items[0].Body, in.Name) {
				t.Fatal("notification did not localize, or disclosed proposal content")
			}
			f.human.Header = nil
			for _, table := range []string{"messages", "automation_runs", "ai_requests", "ai_budget_reservations", "ai_agent_runs"} {
				if f.scalar(`SELECT count(*) FROM `+table) != 0 {
					t.Fatalf("external proposal touched %s", table)
				}
			}
			// Closed content purges, but a client retry cannot create another
			// draft using the same key even after the source has disappeared.
			s := &mcpconnect.Service{Pool: f.env.App.Pool, Q: f.env.App.Q, Now: func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }}
			if err := s.Purge(t.Context()); err != nil {
				t.Fatal(err)
			}
			view = f.proposal(first.ID)
			if view.Blocked != "purged" || view.HTML != "" || view.Email.MJML != "" {
				t.Fatal("purged proposal still exposes content")
			}
			var refreshed mcpconnect.TokenResponse
			f.exchangeAt(f.env.Server.URL, client, f.refreshValues(tokens.RefreshToken)).MustStatus(t, 200).JSON(t, &refreshed)
			in.Email.Subject = "A clearer update"
			retry = toolResult[mcpconnect.MCPProposalReceipt](t, f.mcp(refreshed.AccessToken), "propose_email", in)
			if retry.ID != first.ID || retry.Decision != "kept" {
				t.Fatal("retry after purge did not return the original receipt")
			}
			if f.scalar(`SELECT count(*) FROM mcp_connect_proposals WHERE request_key='first-draft' AND content='{}' AND context='{"Source":null}' AND preview=''`) != 1 {
				t.Fatal("purge lost the idempotency receipt")
			}
			if _, err := f.env.App.Pool.Exec(t.Context(), `INSERT INTO notification_preferences (user_id, event_type, in_app, email) VALUES ($1, 'external_proposal', false, false)`, f.owner.ID); err != nil {
				t.Fatal(err)
			}
			in.Key = "notifications-off"
			toolResult[mcpconnect.MCPProposalReceipt](t, f.mcp(refreshed.AccessToken), "propose_email", in)
			if f.scalar(`SELECT count(*) FROM notifications WHERE kind='external_proposal'`) != 1 {
				t.Fatal("proposal ignored notification preferences")
			}
		})
	}
}

func TestExternalSelectedSourcesAndExactDecision(t *testing.T) {
	f := newFixture(t)
	client := f.client("none")
	code, verifier := f.allow(client)
	tokens := f.exchange(client, code, verifier)
	cs := f.mcp(tokens.AccessToken)
	var grant uuid.UUID
	if err := f.env.App.Pool.QueryRow(t.Context(), `SELECT id FROM mcp_connect_grants WHERE client_id=$1`, client.ID).Scan(&grant); err != nil {
		t.Fatal(err)
	}
	var campaign struct{ ID uuid.UUID }
	base := "/workspaces/" + f.ws.Slug + "/campaigns"
	f.human.Do("POST", base, map[string]any{"name": "Contact-derived hidden label"}).MustStatus(t, 201).JSON(t, &campaign)
	const localLink = "https://editorial.example/article?signature=LOCAL_LINK_SENTINEL"
	src := strings.Replace(proposedMJML, "</mj-column>", `<mj-button href="`+localLink+`">Read more</mj-button></mj-column>`, 1)
	f.human.Do("PUT", base+"/"+campaign.ID.String()+"/content", map[string]any{"baseVersion": 1, "subject": "Original subject", "mjml": src}).MustStatus(t, 200)
	path := f.base + "/connections/" + grant.String() + "/sources"
	f.human.Do("POST", path, map[string]any{"kind": "campaign", "id": campaign.ID}).MustStatus(t, 422)
	f.human.Do("POST", path, mcpconnect.MCPSourceSelection{Kind: "campaign", ID: campaign.ID}).MustStatus(t, 422)
	if f.scalar(`SELECT count(*) FROM mcp_connect_sources`) != 0 || f.scalar(`SELECT count(*) FROM audit_events WHERE action='mcp_connect.source_selected'`) != 0 {
		t.Fatal("unreviewed source created a sharing capability")
	}
	handle := f.selectSource(grant, "campaign", campaign.ID)
	if f.scalar(`SELECT count(*) FROM audit_events WHERE action='mcp_connect.source_selected' AND metadata->>'sharing_policy'='reviewed-editorial-v1'`) != 1 {
		t.Fatal("source sharing lost its reviewed-policy receipt")
	}
	if repeated := f.selectSource(grant, "campaign", campaign.ID); repeated != handle || f.scalar(`SELECT count(*) FROM mcp_connect_sources`) != 1 {
		t.Fatal("repeated source selection duplicated access")
	}
	var selections mcpconnect.MCPSelectedSources
	f.human.Do("GET", f.base+"/connections/"+grant.String()+"/sources", nil).MustStatus(t, 200).JSON(t, &selections)
	if len(selections.Items) != 1 || selections.Items[0].ID != campaign.ID || selections.Items[0].Handle != handle || selections.Items[0].Name != "Contact-derived hidden label" {
		t.Fatal("human selection receipt lost its local identity")
	}
	read := toolResult[struct {
		Text  string   `json:"text"`
		Links []string `json:"links"`
	}](t, cs, "read_source", map[string]any{"source": handle})
	if len(read.Links) != 1 || !strings.Contains(read.Text, "editorial update") {
		t.Fatalf("missing projected source: %+v", read)
	}
	raw, _ := json.Marshal(read)
	if strings.Contains(string(raw), "LOCAL_LINK_SENTINEL") || strings.Contains(string(raw), "Contact-derived") || strings.Contains(string(raw), campaign.ID.String()) {
		t.Fatal("source exposed local identifiers or links")
	}
	email := strings.Replace(proposedMJML, "</mj-column>", `<mj-button href="`+read.Links[0]+`">Read more</mj-button></mj-column>`, 1)
	in := mcpconnect.MCPProposalInput{Key: "change-1", Kind: "campaign", Name: "Edit", Source: handle, Change: true,
		Email: mcpconnect.MCPProposedEmail{Subject: "Shorter subject", MJML: email}}
	first := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_email", in)
	view := f.proposal(first.ID)
	if view.Before == nil || view.Before.Subject != "Original subject" || !strings.Contains(view.Email.MJML, localLink) {
		t.Fatal("human review lost its original content or local bindings")
	}
	// Source selection expiry does not erase a proposal's saved bindings.
	if _, err := f.env.App.Pool.Exec(t.Context(), `DELETE FROM mcp_connect_sources WHERE handle=$1`, handle); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	responses := make(chan *apptest.Response, 2)
	for range 2 {
		wg.Go(func() { responses <- f.decide(first.ID, "accepted") })
	}
	wg.Wait()
	for range 2 {
		(<-responses).MustStatus(t, 204)
	}
	if f.scalar(`SELECT version FROM campaigns WHERE id=$1`, campaign.ID) != 3 || f.scalar(`SELECT count(*) FROM campaign_versions WHERE campaign_id=$1`, campaign.ID) != 3 {
		t.Fatal("double Accept created multiple versions")
	}
	in.Key, in.Source = "change-2", f.selectSource(grant, "campaign", campaign.ID)
	read = toolResult[struct {
		Text  string   `json:"text"`
		Links []string `json:"links"`
	}](t, cs, "read_source", map[string]any{"source": in.Source})
	in.Email.MJML = strings.Replace(proposedMJML, "</mj-column>", `<mj-button href="`+read.Links[0]+`">Read more</mj-button></mj-column>`, 1)
	stale := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_email", in)
	f.human.Do("PUT", base+"/"+campaign.ID.String()+"/content", map[string]any{"baseVersion": 3, "subject": "Human wins", "mjml": src}).MustStatus(t, 200)
	f.human.Do("GET", f.base+"/connections/"+grant.String()+"/sources", nil).MustStatus(t, 200).JSON(t, &selections)
	if len(selections.Items) != 0 {
		t.Fatal("stale source is still shown as selected")
	}
	if f.proposal(stale.ID).Blocked != "changed" {
		t.Fatal("stale revision did not show a conflict")
	}
	f.decide(stale.ID, "accepted").MustStatus(t, 409)
	f.decide(stale.ID, "rejected").MustStatus(t, 204)
	if f.scalar(`SELECT version FROM campaigns WHERE id=$1`, campaign.ID) != 4 {
		t.Fatal("stale proposal overwrote the person's version")
	}
}

func TestExternalProposalRaceAcrossProcesses(t *testing.T) {
	f := newFixture(t)
	client := f.client("none")
	code, verifier := f.allow(client)
	tokens := f.exchange(client, code, verifier)
	second := f.replica()
	in := mcpconnect.MCPProposalInput{Key: "same-key", Kind: "campaign", Name: "Draft", Email: mcpconnect.MCPProposedEmail{Subject: "An update", MJML: proposedMJML}}
	args, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "propose_email", "arguments": in}})
	responses := make(chan *apptest.Response, 2)
	var wg sync.WaitGroup
	for _, host := range []string{f.env.Server.URL, second} {
		wg.Go(func() {
			responses <- request(t, host, "POST", mcpconnect.Path, "application/json", string(args), http.Header{"Authorization": {"Bearer " + tokens.AccessToken}, "Accept": {"application/json, text/event-stream"}})
		})
	}
	wg.Wait()
	var ids []uuid.UUID
	for range 2 {
		resp := <-responses
		if resp.Status == 429 { // Bounded admission may ask the loser to retry.
			resp = request(t, second, "POST", mcpconnect.Path, "application/json", string(args), http.Header{"Authorization": {"Bearer " + tokens.AccessToken}, "Accept": {"application/json, text/event-stream"}})
		}
		var out struct {
			Result struct {
				IsError    bool                          `json:"isError"`
				Structured mcpconnect.MCPProposalReceipt `json:"structuredContent"`
			} `json:"result"`
		}
		resp.MustStatus(t, 200).JSON(t, &out)
		if out.Result.IsError || out.Result.Structured.ID == uuid.Nil {
			t.Fatalf("proposal failed: %s", resp.Body)
		}
		ids = append(ids, out.Result.Structured.ID)
	}
	if ids[0] != ids[1] || f.scalar(`SELECT count(*) FROM mcp_connect_proposals`) != 1 || f.scalar(`SELECT count(*) FROM campaigns`) != 0 {
		t.Fatal("same-key race created duplicate proposals or product drafts")
	}
}

func TestExternalProposalRecoveryAndAuthority(t *testing.T) {
	for _, blocked := range []string{"expired", "revoked", "permission", "disabled"} {
		t.Run(blocked, func(t *testing.T) {
			f := newFixture(t)
			client := f.client("none")
			code, verifier := f.allow(client)
			tokens := f.exchange(client, code, verifier)
			cs := f.mcp(tokens.AccessToken)
			in := mcpconnect.MCPProposalInput{Key: "pending", Kind: "campaign", Name: "Recovery draft", Email: mcpconnect.MCPProposedEmail{Subject: "Recover this locally", MJML: proposedMJML}}
			receipt := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_email", in)
			var grant uuid.UUID
			if err := f.env.App.Pool.QueryRow(t.Context(), `SELECT grant_id FROM mcp_connect_proposals WHERE id=$1`, receipt.ID).Scan(&grant); err != nil {
				t.Fatal(err)
			}
			viewer := f.env.CreateUser(apptest.UserOpts{Email: "viewer@proposal.test"})
			f.env.AddMember(f.ws, viewer.ID, "viewer")
			read := f.env.Client()
			read.SignIn(viewer.Email, "")
			read.Do("GET", f.base+"/proposals/"+receipt.ID.String(), nil).MustStatus(t, 404)
			read.Do("POST", f.base+"/proposals/"+receipt.ID.String()+"/decision", mcpconnect.MCPProposalDecision{Decision: "kept"}).MustStatus(t, 404)
			var list mcpconnect.MCPProposalList
			read.Do("GET", f.base+"/proposals", nil).MustStatus(t, 200).JSON(t, &list)
			if len(list.Items) != 0 {
				t.Fatal("unrelated person can enumerate a proposal")
			}
			f.human.Do("GET", f.base+"/proposals", nil).MustStatus(t, 200).JSON(t, &list)
			if len(list.Items) != 1 || list.Items[0].Name != in.Name || list.Items[0].ID != receipt.ID {
				t.Fatal("owner cannot find their pending proposal")
			}
			var err error
			switch blocked {
			case "expired":
				f.env.App.Now = func() time.Time { return receipt.ExpiresAt }
			case "revoked":
				f.human.Do("DELETE", f.base+"/connections/"+grant.String(), nil).MustStatus(t, 204)
			case "permission":
				_, err = f.env.App.Pool.Exec(t.Context(), `UPDATE mcp_connect_grants SET permissions=array_remove(permissions, 'campaigns:edit') WHERE id=$1`, grant)
			case "disabled":
				_, err = f.env.App.Pool.Exec(t.Context(), `UPDATE workspaces SET mcp_settings=jsonb_set(mcp_settings, '{enabled}', 'false') WHERE id=$1`, f.ws.ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			want := blocked
			if want == "disabled" {
				want = "revoked"
			}
			view := f.proposal(receipt.ID)
			if view.Blocked != want || view.Email.Subject != in.Email.Subject {
				t.Fatalf("blocked recovery: got %q, want %q", view.Blocked, want)
			}
			f.decide(receipt.ID, "kept").MustStatus(t, 409)
			f.decide(receipt.ID, "discarded").MustStatus(t, 204)
			f.decide(receipt.ID, "discarded").MustStatus(t, 204)
			if f.scalar(`SELECT count(*) FROM campaigns`) != 0 {
				t.Fatal("blocked proposal created a draft")
			}
		})
	}
}

func TestExternalProposalPrivacyAndScope(t *testing.T) {
	f := newFixture(t)
	client := f.client("none")
	code, verifier := f.allow(client)
	tokens := f.exchange(client, code, verifier)
	cs := f.mcp(tokens.AccessToken)
	var grant uuid.UUID
	if err := f.env.App.Pool.QueryRow(t.Context(), `SELECT id FROM mcp_connect_grants WHERE client_id=$1`, client.ID).Scan(&grant); err != nil {
		t.Fatal(err)
	}
	// Invalid typed arguments must not echo values from SDK errors.
	for _, input := range []any{
		map[string]any{"key": "invalid", "kind": "campaign", "name": "Draft", "change": "PRIVATE_ARGUMENT_SENTINEL", "email": nil},
		mcpconnect.MCPProposalInput{Key: "url", Kind: "campaign", Name: "Draft", Email: mcpconnect.MCPProposedEmail{Subject: "Subject", MJML: strings.Replace(proposedMJML, "</mj-column>", `<mj-button href="https://evil.example/PRIVATE_ARGUMENT_SENTINEL">Read</mj-button></mj-column>`, 1)}},
		mcpconnect.MCPProposalInput{Key: "token", Kind: "campaign", Name: "Draft", Email: mcpconnect.MCPProposedEmail{Subject: "{{ first_name }}", MJML: proposedMJML}},
	} {
		out, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: "propose_email", Arguments: input})
		if err != nil || !out.IsError {
			t.Fatalf("unsafe proposal accepted: %v %+v", err, out)
		}
		raw, _ := json.Marshal(out)
		if strings.Contains(string(raw), "PRIVATE_ARGUMENT_SENTINEL") {
			t.Fatal("validation echoed private arguments")
		}
	}
	if f.scalar(`SELECT count(*) FROM mcp_connect_proposals`) != 0 {
		t.Fatal("invalid output was staged")
	}
	var campaign struct{ ID uuid.UUID }
	base := "/workspaces/" + f.ws.Slug + "/campaigns"
	f.human.Do("POST", base, map[string]any{"name": "Selected source"}).MustStatus(t, 201).JSON(t, &campaign)
	private := strings.Replace(proposedMJML, "A clearer editorial update.", "PRIVATE_ADDRESS_SENTINEL@example.test", 1)
	f.human.Do("PUT", base+"/"+campaign.ID.String()+"/content", map[string]any{"baseVersion": 1, "subject": "Subject", "mjml": private}).MustStatus(t, 200)
	r := f.human.Do("POST", f.base+"/connections/"+grant.String()+"/sources", mcpconnect.MCPSourceSelection{SharingReviewed: true, Kind: "campaign", ID: campaign.ID}).MustStatus(t, 422)
	if strings.Contains(string(r.Body), "PRIVATE_ADDRESS_SENTINEL") {
		t.Fatal("source refusal exposed personal content")
	}
	f.human.Do("PUT", base+"/"+campaign.ID.String()+"/content", map[string]any{"baseVersion": 2, "subject": "Subject", "mjml": proposedMJML}).MustStatus(t, 200)
	handle := f.selectSource(grant, "campaign", campaign.ID)
	code2, verifier2 := f.allow(client)
	other := f.exchange(client, code2, verifier2)
	foreign := f.mcp(other.AccessToken)
	for _, name := range []string{"read_source", "propose_email"} {
		input := any(map[string]any{"source": handle})
		if name == "propose_email" {
			input = mcpconnect.MCPProposalInput{Key: "foreign", Kind: "campaign", Name: "Copy", Source: handle, Email: mcpconnect.MCPProposedEmail{Subject: "Subject", MJML: proposedMJML}}
		}
		out, err := foreign.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: input})
		if err != nil || !out.IsError {
			t.Fatalf("foreign handle accepted by %s", name)
		}
	}
	in := mcpconnect.MCPProposalInput{Key: "local", Kind: "campaign", Name: "Copied draft", Source: handle, Email: mcpconnect.MCPProposedEmail{Subject: "Subject", MJML: proposedMJML}}
	proposal := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_email", in)
	allowed, hidden := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	for _, id := range []uuid.UUID{allowed, hidden} {
		if _, err := f.env.App.Pool.Exec(t.Context(), `INSERT INTO lists(id,workspace_id,name) VALUES($1,$2,'List')`, id, f.ws.ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.env.App.Pool.Exec(t.Context(), `UPDATE campaigns SET audience=jsonb_build_object('lists', jsonb_build_array($1::text)) WHERE id=$2`, hidden.String(), campaign.ID); err != nil {
		t.Fatal(err)
	}
	scope, _ := json.Marshal(map[string]any{"lists": []uuid.UUID{allowed}})
	if _, err := f.env.App.Pool.Exec(t.Context(), `UPDATE workspace_members SET scope=$1 WHERE workspace_id=$2 AND user_id=$3`, scope, f.ws.ID, f.owner.ID); err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		name  string
		input any
	}{
		{"read_source", map[string]any{"source": handle}},
		{"proposal_status", map[string]any{"id": proposal.ID}},
		{"propose_email", in},
	} {
		out, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: check.name, Arguments: check.input})
		if err != nil || !out.IsError {
			t.Fatalf("current scope escaped through %s", check.name)
		}
	}
	f.human.Do("GET", f.base+"/proposals/"+proposal.ID.String(), nil).MustStatus(t, 404)
	f.decide(proposal.ID, "kept").MustStatus(t, 404)
	var list mcpconnect.MCPProposalList
	f.human.Do("GET", f.base+"/proposals", nil).MustStatus(t, 200).JSON(t, &list)
	if len(list.Items) != 0 {
		t.Fatal("list leaked an out-of-scope source")
	}
	s := &mcpconnect.Service{Q: f.env.App.Q, Now: func() time.Time { return time.Now().Add(8 * 24 * time.Hour) }}
	if err := s.Purge(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.human.Do("GET", f.base+"/proposals/"+proposal.ID.String(), nil).MustStatus(t, 404)
	if f.scalar(`SELECT count(*) FROM mcp_connect_proposals WHERE id=$1 AND context->'Source'->>'TargetID'=$2 AND NOT context ? 'Draft'`, proposal.ID, campaign.ID.String()) != 1 {
		t.Fatal("purge lost the source-scope receipt or kept content")
	}
}

func TestExternalTemplateAndVoiceReferences(t *testing.T) {
	f := newFixture(t)
	client := f.client("none")
	code, verifier := f.allow(client)
	tokens := f.exchange(client, code, verifier)
	cs := f.mcp(tokens.AccessToken)
	var grant uuid.UUID
	if err := f.env.App.Pool.QueryRow(t.Context(), `SELECT id FROM mcp_connect_grants WHERE client_id=$1`, client.ID).Scan(&grant); err != nil {
		t.Fatal(err)
	}
	base := "/workspaces/" + f.ws.Slug
	const link = "https://example.test/article?signature=LOCAL_TEMPLATE_LINK"
	src := strings.Replace(proposedMJML, "</mj-column>", `<mj-button href="`+link+`">Read more</mj-button></mj-column>`, 1)
	var template struct{ ID string }
	f.human.Do("POST", base+"/templates", map[string]any{"name": "LOCAL_TEMPLATE_NAME", "kind": "template", "mjml": src}).MustStatus(t, 201).JSON(t, &template)
	voice := brandkit.BrandVoiceSaveInput{Content: brandkit.BrandVoiceContent{Tone: "Calm and direct", Address: "Use the second person", Examples: "A clear update. Read more at https://example.test/LOCAL_VOICE_LINK"}}
	f.human.Do("PUT", base+"/brand-kit/voice", voice).MustStatus(t, 200)
	selected := toolResult[struct{ Sources []mcpconnect.MCPSourceHandle }](t, cs, "list_sources", map[string]any{})
	if len(selected.Sources) != 0 {
		t.Fatal("unselected templates or voice were discoverable")
	}
	templateHandle := f.selectSource(grant, "template", uuid.MustParse(template.ID))
	voiceHandle := f.selectSource(grant, "voice", f.ws.ID)
	for _, handle := range []string{templateHandle, voiceHandle} {
		out := toolResult[map[string]any](t, cs, "read_source", map[string]any{"source": handle})
		raw, _ := json.Marshal(out)
		for _, sentinel := range []string{"LOCAL_", template.ID, f.ws.ID.String(), f.owner.Name, f.owner.Email} {
			if strings.Contains(string(raw), sentinel) {
				t.Fatalf("external source leaked %s", sentinel)
			}
		}
		if handle == voiceHandle {
			v := out["voice"].(map[string]any)
			if v["tone"] != "Calm and direct" || v["examples"] != "A clear update. Read more at [link omitted]" {
				t.Fatalf("wrong voice projection: %s", raw)
			}
		} else if len(out["links"].([]any)) != 1 {
			t.Fatal("template lost its link binding")
		}
	}
	read := toolResult[struct{ Links []string }](t, cs, "read_source", map[string]any{"source": templateHandle})
	in := mcpconnect.MCPProposalInput{Key: "template-copy", Kind: "campaign", Name: "New issue", Source: templateHandle,
		Email: mcpconnect.MCPProposedEmail{Subject: "A new issue", MJML: strings.Replace(src, link, read.Links[0], 1)}}
	proposal := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_email", in)
	if view := f.proposal(proposal.ID); view.TargetID != nil || !strings.Contains(view.Email.MJML, link) {
		t.Fatal("template copy lost local bindings or became a template change")
	}
	f.decide(proposal.ID, "kept").MustStatus(t, 204)
	if f.scalar(`SELECT count(*) FROM campaigns WHERE status='draft'`) != 1 || f.scalar(`SELECT count(*) FROM templates WHERE mjml=$1`, src) != 1 {
		t.Fatal("Keep failed or mutated the source template")
	}
	in.Key, in.Kind = "template-post", "post"
	post := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_email", in)
	f.decide(post.ID, "kept").MustStatus(t, 204)
	for _, kind := range []string{"campaign", "template", "voice"} {
		in.Key, in.Kind, in.Change = "bad-change-"+kind, kind, true
		if kind == "voice" {
			in.Source = voiceHandle
		}
		out, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: "propose_email", Arguments: in})
		if err != nil || !out.IsError {
			t.Fatal("read-only reference accepted as edit target")
		}
	}
	// Updated references invalidate saved handles and pending copies, not
	// previously kept drafts. An authority downgrade also blocks reads.
	in.Key, in.Kind, in.Change, in.Source = "before-template-change", "campaign", false, templateHandle
	pending := toolResult[mcpconnect.MCPProposalReceipt](t, cs, "propose_email", in)
	f.human.Do("PUT", base+"/templates/"+template.ID, map[string]any{"name": "LOCAL_TEMPLATE_NAME", "kind": "template", "mjml": proposedMJML}).MustStatus(t, 200)
	voice.BaseRevision, voice.Content.Tone = 1, "Warm and concise"
	f.human.Do("PUT", base+"/brand-kit/voice", voice).MustStatus(t, 200)
	for _, handle := range []string{templateHandle, voiceHandle} {
		out, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: "read_source", Arguments: map[string]any{"source": handle}})
		if err != nil || !out.IsError || out.Content[0].(*mcp.TextContent).Text != "source_changed" {
			t.Fatal("stale reference remained readable")
		}
	}
	if f.proposal(pending.ID).Blocked != "changed" {
		t.Fatal("template replacement did not invalidate the pending copy")
	}
	f.decide(pending.ID, "kept").MustStatus(t, 409)
	voiceHandle = f.selectSource(grant, "voice", f.ws.ID)
	if _, err := f.env.App.Pool.Exec(t.Context(), `UPDATE mcp_connect_grants SET permissions=array_remove(array_remove(permissions, 'campaigns:edit'), 'campaigns:view') WHERE id=$1`, grant); err != nil {
		t.Fatal(err)
	}
	out, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: "read_source", Arguments: map[string]any{"source": voiceHandle}})
	if err != nil || !out.IsError {
		t.Fatalf("voice revocation: %v %+v", err, out)
	}
	voice.BaseRevision, voice.Content.Tone = 2, "Contact name: Private Person"
	f.human.Do("PUT", base+"/brand-kit/voice", voice).MustStatus(t, 200)
	// A new grant has permission, but the private source is still refused.
	code, verifier = f.allow(client)
	tokens = f.exchange(client, code, verifier)
	if err := f.env.App.Pool.QueryRow(t.Context(), `SELECT grant_id FROM mcp_connect_tokens WHERE hash=$1`, authn.HashToken(tokens.AccessToken)).Scan(&grant); err != nil {
		t.Fatal(err)
	}
	f.human.Do("POST", f.base+"/connections/"+grant.String()+"/sources", mcpconnect.MCPSourceSelection{SharingReviewed: true, Kind: "voice", ID: f.ws.ID}).MustStatus(t, 422)
	for _, table := range []string{"ai_requests", "ai_budget_reservations", "messages", "automation_runs"} {
		if f.scalar(`SELECT count(*) FROM `+table) != 0 {
			t.Fatalf("references touched %s", table)
		}
	}
}
