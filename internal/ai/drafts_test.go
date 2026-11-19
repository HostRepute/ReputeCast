package ai_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/brandkit"
)

func TestCampaignDraftAcceptance(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@draft.test", Name: "Owner"})
	viewer := env.CreateUser(apptest.UserOpts{Email: "viewer@draft.test", Name: "Viewer"})
	other := env.CreateUser(apptest.UserOpts{Email: "editor@draft.test", Name: "Editor"})
	w := env.CreateWorkspace("Drafts", owner.ID)
	env.AddMember(w, viewer.ID, "viewer")
	env.AddMember(w, other.ID, "editor")
	c, read, editor := env.Client(), env.Client(), env.Client()
	c.SignInDemo(owner.Email)
	read.SignInDemo(viewer.Email)
	editor.SignInDemo(other.Email)
	base := "/workspaces/" + w.Slug
	endpoint := base + "/ai/drafts/campaigns"
	ctx := context.Background()
	var mu sync.Mutex
	var override string
	var answers []string
	var beforeAnswer func()
	var payloads []string
	const generated = `<mjml><mj-body><mj-section><mj-column><mj-text>A useful improvement.</mj-text><mj-button href="https://reputecast.invalid/link/1/end">Read more</mj-button></mj-column></mj-section></mj-body></mjml>`
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		payloads = append(payloads, string(body))
		custom, hook := override, beforeAnswer
		if len(answers) > 0 {
			custom, answers = answers[0], answers[1:]
		}
		mu.Unlock()
		if hook != nil {
			hook()
		}
		answer, _ := json.Marshal(ai.AIDraftEmail{Subject: "A useful update", Preheader: "Here is what changed", MJML: generated})
		if custom != "" {
			answer = []byte(custom)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": string(answer)}, "finish_reason": "stop"}},
			"usage":   map[string]any{"prompt_tokens": 100, "completion_tokens": 80},
		})
	}))
	defer model.Close()
	c.Do("PUT", base+"/ai/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL + "/v1", "model": "draft-fixture", "capKind": "usd", "capUsd": 10, "inputPrice": 1, "outputPrice": 2}).MustStatus(t, 200)
	count := func(table string) int {
		t.Helper()
		var n int
		if err := env.App.Pool.QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE workspace_id = $1", w.ID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	in := ai.AICampaignBriefInput{SharingReviewed: true, Brief: "Draft an update about easier exports.", Name: "Export update", Language: "en", Links: []string{"https://example.test/read?private=local-only"}}
	mu.Lock()
	beforeReview := len(payloads)
	mu.Unlock()
	unreviewed := in
	unreviewed.SharingReviewed = false
	beforeReservations := count("ai_budget_reservations")
	c.Do("POST", endpoint, map[string]any{"brief": in.Brief, "name": in.Name, "language": in.Language}).MustStatus(t, 422)
	c.Do("POST", endpoint, unreviewed).MustStatus(t, 422)
	mu.Lock()
	afterReview := len(payloads)
	mu.Unlock()
	if beforeReview != afterReview || count("ai_draft_proposals") != 0 || count("ai_budget_reservations") != beforeReservations {
		t.Fatal("unreviewed content reached the model or created a proposal")
	}
	read.Do("POST", endpoint, in).MustStatus(t, 403)
	var proposal ai.AICampaignProposal
	c.Do("POST", endpoint, in).MustStatus(t, 200).JSON(t, &proposal)
	if count("campaigns") != 0 || proposal.BaseRevision != 0 || proposal.SubjectCheck.Subject != "A useful update" || !strings.Contains(proposal.Content.MJML, in.Links[0]) || !strings.Contains(proposal.HTML, "A useful improvement.") {
		t.Fatalf("proposal changed resources or lost content: %+v", proposal)
	}
	var pending ai.AIPendingDrafts
	c.Do("GET", endpoint, nil).MustStatus(t, 200).JSON(t, &pending)
	if len(pending.Items) != 1 || pending.Items[0].ID != proposal.ID || pending.Items[0].Name != in.Name {
		t.Fatal("generated proposal cannot be recovered")
	}
	exec(`UPDATE ai_draft_proposals SET expires_at=now()-interval '1 second' WHERE id=$1`, proposal.ID)
	c.Do("GET", endpoint+"?all=true", nil).MustStatus(t, 200).JSON(t, &pending)
	if len(pending.Items) != 0 {
		t.Fatal("recovery listed an expired proposal before cleanup")
	}
	exec(`UPDATE ai_draft_proposals SET expires_at=$2 WHERE id=$1`, proposal.ID, proposal.ExpiresAt)
	for _, client := range []*apptest.Client{read, editor} {
		client.Do("GET", endpoint, nil).MustStatus(t, 200).JSON(t, &pending)
		if len(pending.Items) != 0 {
			t.Fatal("recovery disclosed another person's proposal")
		}
	}
	c.Do("GET", endpoint+"?before="+proposal.ID.String(), nil).MustStatus(t, 200).JSON(t, &pending)
	if len(pending.Items) != 0 {
		t.Fatal("recovery cursor repeated the previous item")
	}
	mu.Lock()
	sent := payloads[len(payloads)-1]
	mu.Unlock()
	for _, excluded := range []string{"local-only", owner.Email, "Your company", "Street and number", w.ID.String()} {
		if strings.Contains(sent, excluded) {
			t.Errorf("excluded context reached model: %s", excluded)
		}
	}
	decision := func(id uuid.UUID) string { return endpoint + "/" + id.String() + "/decision" }
	read.Do("POST", decision(proposal.ID), ai.AIDraftDecision{Decision: "accepted"}).MustStatus(t, 403)
	editor.Do("GET", endpoint+"/"+proposal.ID.String(), nil).MustStatus(t, 404)
	c.Do("POST", base+"/ai/requests/"+proposal.ID.String()+"/outcome", ai.AIOutcomeInput{Outcome: "accepted"}).MustStatus(t, 409)
	var accepted, retried ai.AIDraftDecided
	c.Do("POST", decision(proposal.ID), ai.AIDraftDecision{Decision: "accepted"}).MustStatus(t, 200).JSON(t, &accepted)
	c.Do("POST", decision(proposal.ID), ai.AIDraftDecision{Decision: "accepted"}).MustStatus(t, 200).JSON(t, &retried)
	if accepted.ID == nil || retried.ID == nil || *accepted.ID != *retried.ID || count("campaigns") != 1 {
		t.Fatal("acceptance not idempotent")
	}
	c.Do("GET", endpoint, nil).MustStatus(t, 200).JSON(t, &pending)
	if len(pending.Items) != 0 {
		t.Fatal("accepted proposal still listed as pending")
	}
	var status, mj, storedHTML string
	var sender, scheduled *string
	if err := env.App.Pool.QueryRow(ctx, `SELECT status,mjml,html,sender_id::text,scheduled_at::text FROM campaigns WHERE id=$1`, accepted.ID).Scan(&status, &mj, &storedHTML, &sender, &scheduled); err != nil {
		t.Fatal(err)
	}
	if status != "draft" || sender != nil || scheduled != nil || !strings.Contains(mj, "unsubscribe_url") || !strings.Contains(storedHTML, "A useful improvement.") {
		t.Fatal("accepted content or draft-only boundary broken")
	}
	var actor, via string
	if err := env.App.Pool.QueryRow(ctx, `SELECT actor_type,metadata->>'via' FROM audit_events WHERE target_id=$1 AND action='campaign.create'`, accepted.ID.String()).Scan(&actor, &via); err != nil {
		t.Fatal(err)
	}
	if actor != "user" || via != "ai_assistant" {
		t.Fatalf("provenance: %s / %s", actor, via)
	}
	c.Do("POST", decision(proposal.ID), ai.AIDraftDecision{Decision: "rejected"}).MustStatus(t, 409)

	// A stale proposal must not auto-merge or overwrite a newer manual save.
	in.CampaignID, in.BaseRevision = accepted.ID, 1
	c.Do("POST", endpoint, in).MustStatus(t, 200).JSON(t, &proposal)
	c.Do("GET", endpoint, nil).MustStatus(t, 200).JSON(t, &pending)
	if len(pending.Items) != 0 {
		t.Fatal("existing-draft proposal offered while creating a new campaign")
	}
	c.Do("GET", endpoint+"?target="+accepted.ID.String(), nil).MustStatus(t, 200).JSON(t, &pending)
	if len(pending.Items) != 1 || pending.Items[0].ID != proposal.ID {
		t.Fatal("existing-draft proposal missing from its recovery list")
	}
	c.Do("GET", endpoint+"?all=true", nil).MustStatus(t, 200).JSON(t, &pending)
	if len(pending.Items) != 1 || pending.Items[0].ID != proposal.ID {
		t.Fatal("proposal missing from the cross-target recovery list")
	}
	c.Do("PUT", base+"/campaigns/"+accepted.ID.String()+"/content", map[string]any{"baseVersion": 1, "subject": "Human subject", "preheader": "Human preview", "mjml": mj}).MustStatus(t, 200)
	c.Do("POST", decision(proposal.ID), ai.AIDraftDecision{Decision: "accepted"}).MustStatus(t, 409)
	c.Do("POST", decision(proposal.ID), ai.AIDraftDecision{Decision: "rejected"}).MustStatus(t, 200)
	var subject string
	_ = env.App.Pool.QueryRow(ctx, `SELECT subject FROM campaigns WHERE id=$1`, accepted.ID).Scan(&subject)
	if subject != "Human subject" {
		t.Fatal("proposal overwrote manual revision")
	}

	in.CampaignID, in.BaseRevision = nil, 0
	// A real built-in starter keeps its local header, images, links and
	// button styling; none of those addresses become model context.
	in.TemplateID = "starter:weekly-digest"
	withAssets := strings.Replace(generated, "</mj-column>", `<mj-text>Hi RC_DRAFT_VALUE_2_END</mj-text><mj-image src="https://reputecast.invalid/image/3/end" alt="Hero"/><mj-image src="https://reputecast.invalid/image/4/end" alt="Card"/><mj-button inner-padding="9px 14px" href="https://reputecast.invalid/link/5/end">First</mj-button><mj-button href="https://reputecast.invalid/link/6/end">Second</mj-button></mj-column>`, 1)
	mu.Lock()
	override = mustJSON(t, ai.AIDraftEmail{Subject: "A template-based update", MJML: withAssets})
	mu.Unlock()
	c.Do("POST", endpoint, in).MustStatus(t, 200).JSON(t, &proposal)
	for _, expected := range []string{"/media/starters/logo.png", "/media/starters/hero.png", "https://example.com/article-1", "first_name", "unsubscribe_url", "9px 14px"} {
		if !strings.Contains(proposal.Content.MJML, expected) {
			t.Errorf("starter lost %s", expected)
		}
	}
	c.Do("POST", decision(proposal.ID), ai.AIDraftDecision{Decision: "rejected"}).MustStatus(t, 200)
	in.TemplateID = ""
	// Both repairs count toward the proposal's usage. The model receives
	// the original voice and opaque bindings, not any rejected answer.
	c.Do("PUT", base+"/brand-kit/voice", brandkit.BrandVoiceSaveInput{Content: brandkit.BrandVoiceContent{Tone: "Calm and clear"}}).MustStatus(t, 200)
	mu.Lock()
	repairStart := len(payloads)
	answers = []string{`not JSON rejected-person@example.test`, `{"subject":"Oops","preheader":"","mjml":"bad"}`, mustJSON(t, ai.AIDraftEmail{Subject: "Repaired update", Preheader: "Exports are easier", MJML: generated})}
	beforeAnswer = func() {
		mu.Lock()
		beforeAnswer = nil
		mu.Unlock()
		c.Do("PUT", base+"/brand-kit/voice", brandkit.BrandVoiceSaveInput{BaseRevision: 1, Content: brandkit.BrandVoiceContent{Tone: "Playful and surprising"}}).MustStatus(t, 200)
	}
	mu.Unlock()
	c.Do("POST", endpoint, in).MustStatus(t, 200).JSON(t, &proposal)
	if proposal.Sent.InputTokens != 300 || proposal.Sent.OutputTokens != 240 || proposal.Sent.CostMicros != 780 || proposal.Sent.VoiceRevision != 1 || proposal.Content.Subject != "Repaired update" {
		t.Fatalf("repair usage or pinned voice: %+v", proposal.Sent)
	}
	mu.Lock()
	repairPayloads := append([]string(nil), payloads[repairStart:]...)
	mu.Unlock()
	if len(repairPayloads) != 3 {
		t.Fatalf("got %d attempts, want 3", len(repairPayloads))
	}
	for _, payload := range repairPayloads {
		if !strings.Contains(payload, "Calm and clear") || strings.Contains(payload, "Playful and surprising") || strings.Contains(payload, "local-only") || strings.Contains(payload, "rejected-person") {
			t.Fatal("repair changed pinned voice or disclosed rejected/local content")
		}
	}
	c.Do("POST", decision(proposal.ID), ai.AIDraftDecision{Decision: "rejected"}).MustStatus(t, 200)
	baseline := count("ai_draft_proposals")
	// Bad markup, dropped links, new tokens and silent schema coercion all fail.
	for _, bad := range []string{
		strings.Replace(generated, "A useful improvement.", "<script>alert(1)</script>", 1),
		strings.Replace(generated, "https://reputecast.invalid/link/1/end", "https://other.test", 1),
		strings.Replace(generated, "A useful improvement.", "{{ email }}", 1),
		strings.Replace(generated, "A useful improvement.", "RC_DRAFT_VALUE_99_END", 1),
		`<mjml><mj-body><mj-section><mj-column><mj-text>No link</mj-text></mj-column></mj-section></mj-body></mjml>`,
	} {
		b, _ := json.Marshal(ai.AIDraftEmail{Subject: "Update", MJML: bad})
		mu.Lock()
		override = string(b)
		before := len(payloads)
		mu.Unlock()
		c.Do("POST", endpoint, in).MustStatus(t, 502)
		mu.Lock()
		after := len(payloads)
		mu.Unlock()
		if after-before != 3 {
			t.Fatalf("invalid output took %d attempts, want exactly 3", after-before)
		}
	}
	for _, bad := range []string{
		`{"subject":"Update","mjml":` + mustJSON(t, generated) + `}`,
		`{"subject":"Update","preheader":null,"mjml":` + mustJSON(t, generated) + `}`,
		`{"subject":"Update","preheader":"","mjml":` + mustJSON(t, generated) + `} {}`,
	} {
		mu.Lock()
		override = bad
		mu.Unlock()
		c.Do("POST", endpoint, in).MustStatus(t, 502)
	}
	mu.Lock()
	override = ""
	mu.Unlock()
	if count("campaigns") != 1 || count("ai_draft_proposals") != baseline {
		t.Fatal("invalid output persisted")
	}
	// A source edit between attempts prevents a second paid call.
	in.CampaignID, in.BaseRevision = accepted.ID, 2
	mu.Lock()
	sourceStart := len(payloads)
	override = `not JSON`
	beforeAnswer = func() {
		c.Do("PUT", base+"/campaigns/"+accepted.ID.String()+"/content", map[string]any{"baseVersion": 2, "subject": "New human subject", "preheader": "Human preview", "mjml": mj}).MustStatus(t, 200)
	}
	mu.Unlock()
	c.Do("POST", endpoint, in).MustStatus(t, 409)
	mu.Lock()
	sourceCalls := len(payloads) - sourceStart
	beforeAnswer, override = nil, ""
	mu.Unlock()
	if sourceCalls != 1 || count("ai_draft_proposals") != baseline {
		t.Fatal("repair spent or saved after the source changed")
	}
	in.CampaignID, in.BaseRevision = nil, 0
	// Permissions and the feature switch are rechecked after the model returns.
	mu.Lock()
	beforeAnswer = func() {
		exec(`UPDATE ai_workspace_settings SET features_off=array_append(features_off,'draft') WHERE workspace_id=$1`, w.ID)
	}
	mu.Unlock()
	c.Do("POST", endpoint, in).MustStatus(t, 409)
	mu.Lock()
	beforeAnswer = nil
	mu.Unlock()
	if count("ai_draft_proposals") != baseline {
		t.Fatal("late result committed after switch off")
	}
	exec(`UPDATE ai_workspace_settings SET features_off=array_remove(features_off,'draft') WHERE workspace_id=$1`, w.ID)
	mu.Lock()
	calls := len(payloads)
	mu.Unlock()
	in.Brief = "Email: secret-person@example.test"
	c.Do("POST", endpoint, in).MustStatus(t, 422)
	mu.Lock()
	defer mu.Unlock()
	if len(payloads) != calls {
		t.Fatal("personal brief reached model")
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestAutomationDraftAcceptance(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@flow.test", Name: "Owner"})
	other := env.CreateUser(apptest.UserOpts{Email: "other@flow.test", Name: "Other"})
	w := env.CreateWorkspace("Flow drafts", owner.ID)
	env.AddMember(w, other.ID, "editor")
	c, editor := env.Client(), env.Client()
	c.SignInDemo(owner.Email)
	editor.SignInDemo(other.Email)
	ctx := context.Background()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	list := uuid.Must(uuid.NewV7())
	exec(`INSERT INTO lists (id,workspace_id,name) VALUES ($1,$2,'PRIVATE-LIST-NAME')`, list, w.ID)
	base := "/workspaces/" + w.Slug
	endpoint := base + "/ai/drafts/automations"
	const source = `<mjml><mj-body><mj-section><mj-column><mj-text>Try the export panel.</mj-text></mj-column></mj-section></mj-body></mjml>`
	fixture := ai.AIAutomationContent{Graph: automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			{ID: "start", Type: "trigger", Data: automations.AutomationNodeData{Trigger: &automations.AutomationTrigger{Kind: "joined_list", ListID: "selected_list", ConfirmedOnly: true, ExitOnLeave: true, Reentry: "once"}}},
			{ID: "first", Type: "email", Data: automations.AutomationNodeData{Email: &automations.AutomationEmailStep{EmailID: "intro", EmailVersion: 1, Subject: "Welcome to exports"}}},
			{ID: "wait", Type: "delay", Data: automations.AutomationNodeData{Delay: &automations.AutomationDuration{Amount: 3, Unit: "days"}}},
			{ID: "second", Type: "email", Data: automations.AutomationNodeData{Email: &automations.AutomationEmailStep{EmailID: "followup", EmailVersion: 1, Subject: "One more export tip"}}},
			{ID: "end", Type: "exit"},
		},
		Edges: []automations.AutomationEdge{{ID: "a", Source: "start", Target: "first"}, {ID: "b", Source: "first", Target: "wait"}, {ID: "c", Source: "wait", Target: "second"}, {ID: "d", Source: "second", Target: "end"}},
	}, Emails: []automations.ProposedEmail{
		{Key: "intro", Name: "Introduction", Subject: "Welcome to exports", Preheader: "Start here", MJML: source},
		{Key: "followup", Name: "Follow-up", Subject: "One more export tip", Preheader: "A useful shortcut", MJML: strings.Replace(source, "Try the export panel.", "Select columns before exporting.", 1)},
	}}
	valid := mustJSON(t, fixture)
	var mu sync.Mutex
	answer := valid
	var answers []string
	var hook func()
	var payloads []string
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		payloads = append(payloads, string(body))
		text, before := answer, hook
		if len(answers) > 0 {
			text, answers = answers[0], answers[1:]
		}
		mu.Unlock()
		if before != nil {
			before()
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": text}, "finish_reason": "stop"}}, "usage": map[string]any{"prompt_tokens": 150, "completion_tokens": 300}})
	}))
	defer model.Close()
	c.Do("PUT", base+"/ai/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL + "/v1", "model": "flow-fixture", "capKind": "usd", "capUsd": 10, "inputPrice": 1, "outputPrice": 2}).MustStatus(t, 200)
	counts := func() [3]int {
		t.Helper()
		var out [3]int
		if err := env.App.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM automations WHERE workspace_id=$1), (SELECT count(*) FROM automation_emails WHERE workspace_id=$1), (SELECT count(*) FROM automation_runs r JOIN automations a ON a.id=r.automation_id WHERE a.workspace_id=$1)`, w.ID).Scan(&out[0], &out[1], &out[2]); err != nil {
			t.Fatal(err)
		}
		return out
	}
	in := ai.AIAutomationBriefInput{SharingReviewed: true, Brief: "Welcome future confirmed subscribers with two export tips, three days apart.", Name: "Export welcome", Language: "en", ListID: list}
	mu.Lock()
	beforeReview := len(payloads)
	mu.Unlock()
	unreviewed := in
	unreviewed.SharingReviewed = false
	c.Do("POST", endpoint, unreviewed).MustStatus(t, 422)
	c.Do("POST", endpoint, map[string]any{"brief": in.Brief, "name": in.Name, "language": in.Language, "listId": list}).MustStatus(t, 422)
	mu.Lock()
	afterReview := len(payloads)
	mu.Unlock()
	if beforeReview != afterReview || counts() != [3]int{} {
		t.Fatal("unreviewed automation reached the model or changed resources")
	}
	var proposal ai.AIAutomationProposal
	c.Do("POST", endpoint, in).MustStatus(t, 200).JSON(t, &proposal)
	if counts() != [3]int{} || proposal.Content.Graph.Trigger().Data.Trigger.ListID != list.String() {
		t.Fatal("generation wrote resources or lost selected list")
	}
	mu.Lock()
	payload := payloads[len(payloads)-1]
	mu.Unlock()
	for _, excluded := range []string{list.String(), w.ID.String(), "PRIVATE-LIST-NAME", owner.Email} {
		if strings.Contains(payload, excluded) {
			t.Errorf("private context reached model: %s", excluded)
		}
	}
	decide := func(id uuid.UUID) string { return endpoint + "/" + id.String() + "/decision" }
	editor.Do("GET", endpoint+"/"+proposal.ID.String(), nil).MustStatus(t, 404)
	editor.Do("POST", decide(proposal.ID), ai.AIDraftDecision{Decision: "accepted"}).MustStatus(t, 404)
	responses := make(chan *apptest.Response, 2)
	for range 2 {
		go func() { responses <- c.Do("POST", decide(proposal.ID), ai.AIDraftDecision{Decision: "accepted"}) }()
	}
	var first, second ai.AIDraftDecided
	(<-responses).MustStatus(t, 200).JSON(t, &first)
	(<-responses).MustStatus(t, 200).JSON(t, &second)
	if first.ID == nil || second.ID == nil || *first.ID != *second.ID || counts() != [3]int{1, 2, 0} {
		t.Fatal("concurrent acceptance duplicated bundle or enrolled contacts")
	}
	var status string
	var live *int
	var graphJSON []byte
	if err := env.App.Pool.QueryRow(ctx, `SELECT a.status,a.live_version,v.graph FROM automations a JOIN automation_versions v ON v.automation_id=a.id AND v.version=a.draft_version WHERE a.id=$1`, first.ID).Scan(&status, &live, &graphJSON); err != nil {
		t.Fatal(err)
	}
	if status != "draft" || live != nil {
		t.Fatal("accepted flow is live")
	}
	g, err := automations.ParseGraph(graphJSON)
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range g.Nodes {
		if node.Type != "email" {
			continue
		}
		var subject, html, mj string
		if err := env.App.Pool.QueryRow(ctx, `SELECT v.subject,v.html,v.mjml FROM automation_emails e JOIN automation_email_versions v ON v.email_id=e.id AND v.version=1 WHERE e.id=$1 AND e.automation_id=$2`, node.Data.Email.EmailID, first.ID).Scan(&subject, &html, &mj); err != nil {
			t.Fatal(err)
		}
		if subject != node.Data.Email.Subject || !strings.Contains(html, "export") || !strings.Contains(mj, "unsubscribe_url") {
			t.Fatal("email graph reference or compiled content lost")
		}
	}
	var events int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE metadata->>'proposal_id'=$1 AND metadata->>'via'='ai_assistant' AND actor_type='user'`, proposal.ID.String()).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != 3 {
		t.Fatalf("bundle audit: got %d, want one flow and two emails", events)
	}
	c.Do("POST", decide(proposal.ID), ai.AIDraftDecision{Decision: "rejected"}).MustStatus(t, 409)

	// A manual save and its emails survive both stale acceptance and rejection.
	in.AutomationID, in.BaseRevision = first.ID, 1
	mu.Lock()
	answer = strings.ReplaceAll(strings.ReplaceAll(valid, `"intro"`, `"source1"`), `"followup"`, `"source2"`)
	mu.Unlock()
	c.Do("POST", endpoint, in).MustStatus(t, 200).JSON(t, &proposal)
	if len(proposal.BeforeEmails) != 2 || proposal.BeforeEmails[0].Content.Subject != "Welcome to exports" {
		t.Fatal("existing email context missing from review")
	}
	exec(`UPDATE automations SET draft_revision=draft_revision+1 WHERE id=$1`, first.ID)
	c.Do("POST", decide(proposal.ID), ai.AIDraftDecision{Decision: "accepted"}).MustStatus(t, 409)
	c.Do("POST", decide(proposal.ID), ai.AIDraftDecision{Decision: "rejected"}).MustStatus(t, 200)
	if counts() != [3]int{1, 2, 0} {
		t.Fatal("stale or rejected proposal changed resources")
	}

	// Distinct destinations and exact Liquid remain attached to their own
	// email; neither address nor any database identifier reaches the model.
	firstEmail := g.Node("first").Data.Email.EmailID
	secondEmail := g.Node("second").Data.Email.EmailID
	personalized := strings.Replace(source, "Try the export panel.", `Hi {{ first_name | default: 'reader' }}. Try exports.`, 1)
	personalized = strings.Replace(personalized, "</mj-column>", `<mj-button href="https://example.test/first?private=source-one">Read</mj-button></mj-column>`, 1)
	otherSource := strings.Replace(source, "</mj-column>", `<mj-button href="https://example.test/second?private=source-two">Read</mj-button></mj-column>`, 1)
	exec(`UPDATE automation_email_versions SET mjml=$2 WHERE email_id=$1 AND version=1`, firstEmail, personalized)
	exec(`UPDATE automation_email_versions SET mjml=$2 WHERE email_id=$1 AND version=1`, secondEmail, otherSource)
	var revision ai.AIAutomationContent
	if err := json.Unmarshal([]byte(valid), &revision); err != nil {
		t.Fatal(err)
	}
	revision.Emails[0].Key, revision.Emails[1].Key = "source1", "source2"
	revision.Graph.Node("first").Data.Email.EmailID = "source1"
	revision.Graph.Node("second").Data.Email.EmailID = "source2"
	revision.Emails[0].MJML = strings.ReplaceAll(strings.ReplaceAll(personalized, `{{ first_name | default: 'reader' }}`, "RC_DRAFT_VALUE_source1_1_END"), "https://example.test/first?private=source-one", "https://reputecast.invalid/link/source1_2/end")
	revision.Emails[1].MJML = strings.ReplaceAll(otherSource, "https://example.test/second?private=source-two", "https://reputecast.invalid/link/source2_1/end")
	mu.Lock()
	answer = mustJSON(t, revision)
	mu.Unlock()
	in.BaseRevision = 2
	c.Do("POST", endpoint, in).MustStatus(t, 200).JSON(t, &proposal)
	for _, keep := range []string{`{{ first_name | default: 'reader' }}`, "https://example.test/first?private=source-one"} {
		if !strings.Contains(proposal.Content.Emails[0].MJML, keep) {
			t.Errorf("source lost %s", keep)
		}
	}
	if !strings.Contains(proposal.Content.Emails[1].MJML, "https://example.test/second?private=source-two") || strings.Contains(proposal.Content.Emails[1].MJML, "source-one") {
		t.Fatal("second email lost or borrowed a destination")
	}
	mu.Lock()
	payload = payloads[len(payloads)-1]
	mu.Unlock()
	for _, private := range []string{"source-one", "source-two", firstEmail, secondEmail, list.String(), "PRIVATE-LIST-NAME"} {
		if strings.Contains(payload, private) {
			t.Errorf("source leaked %s", private)
		}
	}
	if !strings.Contains(payload, "Try exports") || !strings.Contains(payload, "source1_2/end") {
		t.Fatal("source not projected")
	}
	c.Do("POST", decide(proposal.ID), ai.AIDraftDecision{Decision: "rejected"}).MustStatus(t, 200)
	mu.Lock()
	answer = strings.ReplaceAll(mustJSON(t, revision), "link/source1_2/end", "link/source2_1/end")
	mu.Unlock()
	c.Do("POST", endpoint, in).MustStatus(t, 502)
	if counts() != [3]int{1, 2, 0} {
		t.Fatal("rejected replacement changed the flow or emails")
	}

	in.AutomationID, in.BaseRevision = nil, 0
	// A 70/30 fork may merge before a branch only when both paths include
	// the email being tested. Testing the first email on the bypass is not
	// equivalent to "did not click" after receiving that email.
	var branched ai.AIAutomationContent
	if err := json.Unmarshal([]byte(valid), &branched); err != nil {
		t.Fatal(err)
	}
	branched.Graph.Nodes[2] = automations.AutomationNode{ID: "wait", Type: "split", Data: automations.AutomationNodeData{Variants: []automations.AutomationSplitVariant{{Key: "A", Weight: 70}, {Key: "B", Weight: 30}}}}
	branched.Graph.Nodes = append(branched.Graph.Nodes, automations.AutomationNode{ID: "clicked", Type: "branch", Data: automations.AutomationNodeData{Condition: &automations.AutomationCondition{Kind: "clicked", Step: "first", Within: &automations.AutomationDuration{Amount: 3, Unit: "days"}, HumanOnly: true}}})
	branched.Graph.Edges = []automations.AutomationEdge{
		{ID: "a", Source: "start", Target: "first"}, {ID: "b", Source: "first", Target: "wait"},
		{ID: "c", Source: "wait", Target: "second", Data: automations.AutomationEdgeData{Path: "A"}},
		{ID: "d", Source: "wait", Target: "clicked", Data: automations.AutomationEdgeData{Path: "B"}},
		{ID: "e", Source: "second", Target: "clicked"},
		{ID: "f", Source: "clicked", Target: "end", Data: automations.AutomationEdgeData{Path: "yes"}},
		{ID: "g", Source: "clicked", Target: "end", Data: automations.AutomationEdgeData{Path: "no"}},
	}
	mu.Lock()
	answer = mustJSON(t, branched)
	mu.Unlock()
	c.Do("POST", endpoint, in).MustStatus(t, 200).JSON(t, &proposal)
	c.Do("POST", decide(proposal.ID), ai.AIDraftDecision{Decision: "rejected"}).MustStatus(t, 200)
	branched.Graph.Node("clicked").Data.Condition.Step = "second"
	mu.Lock()
	answer = mustJSON(t, branched)
	mu.Unlock()
	c.Do("POST", endpoint, in).MustStatus(t, 502)
	if counts() != [3]int{1, 2, 0} {
		t.Fatal("branched proposal changed resources before acceptance")
	}

	// Restore the first email locally, then fail on the second. Repair
	// must not replay the restored footer or any other partial bundle.
	var repair ai.AIAutomationContent
	if err := json.Unmarshal([]byte(valid), &repair); err != nil {
		t.Fatal(err)
	}
	repair.Emails[1].MJML = strings.Replace(source, "Try the export panel.", "contact-private@example.test", 1)
	mu.Lock()
	repairStart := len(payloads)
	answers = []string{mustJSON(t, repair), valid}
	mu.Unlock()
	c.Do("POST", endpoint, in).MustStatus(t, 200).JSON(t, &proposal)
	if proposal.Sent.InputTokens != 300 || proposal.Sent.OutputTokens != 600 || proposal.Sent.CostMicros != 1500 || len(proposal.Content.Emails) != 2 || counts() != [3]int{1, 2, 0} {
		t.Fatal("repair lost email content or accounting, or created product resources")
	}
	mu.Lock()
	repairPayloads := append([]string(nil), payloads[repairStart:]...)
	mu.Unlock()
	if len(repairPayloads) != 2 || strings.Contains(repairPayloads[1], "contact-private") || strings.Contains(repairPayloads[1], "Street and number") {
		t.Fatal("repair replayed invalid output or local footer")
	}
	c.Do("POST", decide(proposal.ID), ai.AIDraftDecision{Decision: "rejected"}).MustStatus(t, 200)

	for name, mutate := range map[string]func(*ai.AIAutomationContent){
		"audience":          func(v *ai.AIAutomationContent) { v.Graph.Trigger().Data.Trigger.ConfirmedOnly = false },
		"cycle":             func(v *ai.AIAutomationContent) { v.Graph.Edges[2].Target = "first" },
		"unknown reference": func(v *ai.AIAutomationContent) { v.Graph.Nodes[1].Data.Email.EmailID = uuid.NewString() },
		"hidden action": func(v *ai.AIAutomationContent) {
			v.Graph.Nodes[1].Data.Action = &automations.AutomationListAction{Kind: "unsubscribe_all"}
		},
		"webhook":      func(v *ai.AIAutomationContent) { v.Graph.Nodes[2].Type = "webhook" },
		"missing path": func(v *ai.AIAutomationContent) { v.Graph.Edges = v.Graph.Edges[:3] },
		"unused email": func(v *ai.AIAutomationContent) {
			v.Emails = append(v.Emails, automations.ProposedEmail{Key: "orphan", Name: "Unused", Subject: "Unused", MJML: source})
		},
		"script": func(v *ai.AIAutomationContent) {
			v.Emails[0].MJML = strings.Replace(source, "Try the export panel.", "<script>alert(1)</script>", 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			var bad ai.AIAutomationContent
			if err := json.Unmarshal([]byte(valid), &bad); err != nil {
				t.Fatal(err)
			}
			mutate(&bad)
			mu.Lock()
			answer = mustJSON(t, bad)
			before := len(payloads)
			mu.Unlock()
			c.Do("POST", endpoint, in).MustStatus(t, 502)
			mu.Lock()
			after := len(payloads)
			mu.Unlock()
			if after-before != 3 {
				t.Fatalf("got %d attempts, want exactly 3", after-before)
			}
			if counts() != [3]int{1, 2, 0} {
				t.Fatal("bad output changed a resource")
			}
		})
	}
	mu.Lock()
	scopeStart := len(payloads)
	answer = `not JSON`
	hook = func() {
		exec(`UPDATE workspace_members SET scope=jsonb_build_object('lists',jsonb_build_array($1::text)) WHERE workspace_id=$2 AND user_id=$3`, uuid.NewString(), w.ID, owner.ID)
	}
	mu.Unlock()
	c.Do("POST", endpoint, in).MustStatus(t, 404)
	mu.Lock()
	scopeCalls := len(payloads) - scopeStart
	hook = nil
	mu.Unlock()
	if scopeCalls != 1 {
		t.Fatal("repair spent after scope revocation")
	}
	exec(`UPDATE workspace_members SET scope='{}' WHERE workspace_id=$1 AND user_id=$2`, w.ID, owner.ID)
	mu.Lock()
	answer = valid
	hook = func() {
		exec(`UPDATE ai_workspace_settings SET features_off=array_append(features_off,'draft') WHERE workspace_id=$1`, w.ID)
	}
	mu.Unlock()
	c.Do("POST", endpoint, in).MustStatus(t, 409)
	mu.Lock()
	hook = nil
	mu.Unlock()
	exec(`UPDATE ai_workspace_settings SET features_off=array_remove(features_off,'draft') WHERE workspace_id=$1`, w.ID)
	c.Do("POST", endpoint, in).MustStatus(t, 200).JSON(t, &proposal)
	exec(`UPDATE workspace_members SET scope=jsonb_build_object('lists',jsonb_build_array($1::text)) WHERE workspace_id=$2 AND user_id=$3`, uuid.NewString(), w.ID, owner.ID)
	var pending ai.AIPendingDrafts
	c.Do("GET", endpoint, nil).MustStatus(t, 200).JSON(t, &pending)
	if len(pending.Items) != 0 {
		t.Fatal("recovery listed a proposal after its audience scope was revoked")
	}
	c.Do("GET", endpoint+"/"+proposal.ID.String(), nil).MustStatus(t, 404)
	c.Do("POST", decide(proposal.ID), ai.AIDraftDecision{Decision: "accepted"}).MustStatus(t, 404)
	if counts() != [3]int{1, 2, 0} {
		t.Fatal("revoked scope allowed mutation")
	}
}
