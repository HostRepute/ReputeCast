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
	"github.com/hostrepute/reputecast/internal/brandkit"
)

func TestBrandVoice(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@voice.test", Name: "Owner"})
	other := env.CreateUser(apptest.UserOpts{Email: "editor@voice.test", Name: "Editor"})
	viewer := env.CreateUser(apptest.UserOpts{Email: "viewer@voice.test", Name: "Viewer"})
	w := env.CreateWorkspace("Voice", owner.ID)
	env.AddMember(w, other.ID, "editor")
	env.AddMember(w, viewer.ID, "viewer")
	c, admin, read := env.Client(), env.Client(), env.Client()
	c.SignInDemo(owner.Email)
	admin.SignInDemo(other.Email)
	read.SignInDemo(viewer.Email)
	base := "/workspaces/" + w.Slug
	voicePath := base + "/brand-kit/voice"
	ctx := context.Background()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	var mu sync.Mutex
	var payloads []string
	var beforeAnswer func()
	var override, stop string
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		payloads = append(payloads, string(body))
		f := beforeAnswer
		custom, finish := override, stop
		mu.Unlock()
		if f != nil {
			f()
		}
		answer := `{"tone":"Warm and specific","address":"you","sentenceLength":"Short sentences","preferredWords":"build","avoidedWords":"hype","examples":"Here is what changed."}`
		if strings.Contains(string(body), "five different subject lines") {
			answer = `{"subjects":["A useful improvement"]}`
		}
		if custom != "" {
			answer = custom
		}
		if finish == "" {
			finish = "stop"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": answer}, "finish_reason": finish}},
			"usage":   map[string]any{"prompt_tokens": 120, "completion_tokens": 80},
		})
	}))
	defer model.Close()
	c.Do("PUT", base+"/ai/connection", map[string]any{"provider": "compatible", "baseUrl": model.URL + "/v1", "model": "voice-fixture", "capKind": "usd", "capUsd": 10, "inputPrice": 1, "outputPrice": 2}).MustStatus(t, 200)
	mu.Lock()
	payloads = nil // Connecting probes the model; manual edits must not.
	mu.Unlock()
	countCalls := func() int { mu.Lock(); defer mu.Unlock(); return len(payloads) }
	manual := brandkit.BrandVoiceContent{Tone: "Direct and calm", Examples: "Try one useful change."}
	var saved brandkit.BrandVoice
	read.Do("PUT", voicePath, brandkit.BrandVoiceSaveInput{Content: manual}).MustStatus(t, 403)
	c.Do("PUT", voicePath, brandkit.BrandVoiceSaveInput{Content: manual}).MustStatus(t, 200).JSON(t, &saved)
	if saved.Revision != 1 || saved.Learned || saved.Content != manual || countCalls() != 0 {
		t.Fatalf("manual save must be local: %+v, calls %d", saved, countCalls())
	}
	c.Do("PUT", voicePath, brandkit.BrandVoiceSaveInput{Content: manual}).MustStatus(t, 409)
	read.Do("GET", voicePath, nil).MustStatus(t, 200)

	makeCampaign := func(text string) uuid.UUID {
		t.Helper()
		var result struct{ ID uuid.UUID }
		c.Do("POST", base+"/campaigns", map[string]any{"name": "Selected campaign"}).MustStatus(t, 201).JSON(t, &result)
		c.Do("PUT", base+"/campaigns/"+result.ID.String()+"/content", map[string]any{
			"baseVersion": 1, "subject": "A useful update", "mjml": `<mjml><mj-body><mj-section><mj-column><mj-text>` + text + `</mj-text></mj-column></mj-section></mj-body></mjml>`,
		}).MustStatus(t, 200)
		exec(`UPDATE campaigns SET status = 'sent' WHERE id = $1`, result.ID)
		return result.ID
	}
	token := `{{ first_name | default: "there &amp; friend" }}`
	selected := makeCampaign(`Hello ` + token + `. ONE-SELECTED-SAMPLE <a href="https://tracking.example/private?recipient=hidden@example.test">Read more</a><!-- SECRET-COMMENT --></mj-text></mj-column></mj-section><mj-section rc-synced="footer"><mj-column><mj-text>500 Market St · FOOTER-ONLY <a href="{{ unsubscribe_url }}">Unsubscribe</a>`)
	_ = makeCampaign("UNSELECTED-SAMPLE")
	learn := ai.AILearnVoiceInput{SharingReviewed: true, CampaignIDs: []uuid.UUID{selected}}
	nBeforeReview := countCalls()
	c.Do("POST", base+"/ai/brand-voice", map[string]any{"campaignIds": []uuid.UUID{selected}}).MustStatus(t, 422)
	c.Do("POST", base+"/ai/brand-voice", ai.AILearnVoiceInput{CampaignIDs: []uuid.UUID{selected}}).MustStatus(t, 422)
	if countCalls() != nBeforeReview {
		t.Fatal("unreviewed voice samples reached the model")
	}
	var suggestion ai.AILearnedVoice
	mu.Lock()
	beforeAnswer = func() {
		var receipts int
		if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE workspace_id=$1 AND action='ai.sharing.reviewed' AND metadata->>'sharing_policy'='reviewed-editorial-v1'`, w.ID).Scan(&receipts); err != nil || receipts != 1 {
			t.Errorf("sharing receipt was not committed before the model call: %d, %v", receipts, err)
		}
	}
	mu.Unlock()
	c.Do("POST", base+"/ai/brand-voice", learn).MustStatus(t, 200).JSON(t, &suggestion)
	mu.Lock()
	beforeAnswer = nil
	mu.Unlock()
	if suggestion.BaseRevision != 1 || suggestion.RequestID == "" || suggestion.Content.Tone != "Warm and specific" {
		t.Fatalf("suggestion: %+v", suggestion)
	}
	mu.Lock()
	var request struct{ Messages []struct{ Content string } }
	if err := json.Unmarshal([]byte(payloads[len(payloads)-1]), &request); err != nil {
		t.Fatal(err)
	}
	mu.Unlock()
	var samples []string
	message := request.Messages[len(request.Messages)-1].Content
	start, end := strings.Index(message, "<samples>\n"), strings.Index(message, "\n</samples>")
	if start < 0 || end < start || json.Unmarshal([]byte(message[start+len("<samples>\n"):end]), &samples) != nil || len(samples) != 1 {
		t.Fatalf("selected sample missing: %s", message)
	}
	if !strings.Contains(samples[0], token) || !strings.Contains(samples[0], "ONE-SELECTED-SAMPLE") {
		t.Fatalf("Liquid or selected text changed: %s", samples[0])
	}
	for _, secret := range []string{"UNSELECTED-SAMPLE", "SECRET-COMMENT", "tracking.example", "hidden@example.test", "500 Market St", "FOOTER-ONLY", owner.Email, selected.String()} {
		if strings.Contains(message, secret) {
			t.Errorf("model received excluded context %q", secret)
		}
	}
	c.Do("GET", voicePath, nil).MustStatus(t, 200).JSON(t, &saved)
	if saved.Revision != 1 {
		t.Fatal("learning wrote a voice without acceptance")
	}
	accept := brandkit.BrandVoiceSaveInput{BaseRevision: 1, Content: suggestion.Content, SuggestionID: suggestion.RequestID}
	admin.Do("PUT", voicePath, accept).MustStatus(t, 409)
	accept.Content.Tone = "Warm, specific and concise"
	c.Do("PUT", voicePath, accept).MustStatus(t, 200).JSON(t, &saved)
	if saved.Revision != 2 || !saved.Learned || saved.Content != accept.Content {
		t.Fatalf("acceptance: %+v", saved)
	}
	c.Do("PUT", voicePath, accept).MustStatus(t, 200).JSON(t, &saved)
	if saved.Revision != 2 {
		t.Fatal("retry created a second revision")
	}
	accept.Content.Tone = "A different retry"
	c.Do("PUT", voicePath, accept).MustStatus(t, 409)
	c.Do("POST", base+"/ai/requests/"+suggestion.RequestID+"/outcome", map[string]any{"outcome": "rejected"}).MustStatus(t, 409)
	var outcome string
	if err := env.App.Pool.QueryRow(ctx, `SELECT outcome FROM ai_requests WHERE id = $1`, suggestion.RequestID).Scan(&outcome); err != nil || outcome != "accepted_edits" {
		t.Fatalf("accepted edit outcome: %s, %v", outcome, err)
	}

	// Turning learning off doesn't disable a saved voice. The selected
	// historical revision remains pinned, and zero explicitly opts out.
	c.Do("PUT", base+"/ai/features/brand_voice", map[string]any{"on": false}).MustStatus(t, 200)
	c.Do("POST", base+"/ai/brand-voice", learn).MustStatus(t, 409)
	for _, revision := range []int32{1, 2, 0} {
		var ideas ai.AISubjectIdeas
		c.Do("POST", base+"/ai/subject-ideas", ai.AISubjectIdeasInput{CampaignID: selected.String(), Subject: "News", MJML: "", VoiceRevision: &revision}).MustStatus(t, 200).JSON(t, &ideas)
		if ideas.Sent.VoiceRevision != revision {
			t.Fatalf("selected voice %d, got %d", revision, ideas.Sent.VoiceRevision)
		}
		mu.Lock()
		payload := payloads[len(payloads)-1]
		mu.Unlock()
		want := map[int32]string{1: manual.Tone, 2: "Warm, specific and concise", 0: ""}[revision]
		if (revision == 0 && strings.Contains(payload, "<brand_voice>")) || (revision > 0 && !strings.Contains(payload, want)) {
			t.Fatalf("request did not use selected voice %d", revision)
		}
		var recorded int32
		if err := env.App.Pool.QueryRow(ctx, `SELECT voice_revision FROM ai_requests WHERE id = $1`, ideas.RequestID).Scan(&recorded); err != nil || recorded != revision {
			t.Fatalf("voice provenance: %d, %v", recorded, err)
		}
	}
	c.Do("PUT", base+"/ai/features/brand_voice", map[string]any{"on": true}).MustStatus(t, 200)
	// Truncated or invalid structured responses must not become suggestions.
	for _, result := range []struct{ body, stop string }{
		{`{"tone":"reader\u0040example.test"}`, "stop"},
		{`{"tone":"Valid JSON but truncated output"}`, "length"},
		{`{"tone":""}`, "stop"},
	} {
		mu.Lock()
		override, stop = result.body, result.stop
		mu.Unlock()
		c.Do("POST", base+"/ai/brand-voice", learn).MustStatus(t, 502)
	}
	mu.Lock()
	override, stop = "", ""
	mu.Unlock()

	// Privacy failures, missing/duplicate selections, non-sent/deleted and
	// foreign resources never reach the model.
	for _, text := range []string{"reader@example.test", "reader&#64;example.test", "Hi Alice", "phone: +1 202 555 0198", "name: Alice Jones", "500 Market St", "reader%40example.test", "rea\u200bder@example.test", "ｒｅａｄｅｒ＠ｅｘａｍｐｌｅ．ｔｅｓｔ", "-----BEGIN PRIVATE KEY-----", `{{ first_name | default: "reader&#64;example.test" }}`, `{{ first_name | default: "https://private.example/link" }}`, `<script>steal()</script>`} {
		t.Run(text, func(t *testing.T) {
			exec(`UPDATE campaigns SET mjml = $2 WHERE id = $1`, selected, `<mjml><mj-body><mj-section><mj-column><mj-text>`+text+`</mj-text></mj-column></mj-section></mj-body></mjml>`)
			n := countCalls()
			c.Do("POST", base+"/ai/brand-voice", learn).MustStatus(t, 422)
			if countCalls() != n {
				t.Fatal("private text reached the model")
			}
		})
	}
	exec(`UPDATE campaigns SET mjml = '<mjml><mj-body><mj-section><mj-column><mj-text>Safe content.</mj-text></mj-column></mj-section></mj-body></mjml>' WHERE id = $1`, selected)
	for _, ids := range [][]uuid.UUID{nil, {selected, selected}, {uuid.New()}, {selected, uuid.New()}} {
		n := countCalls()
		c.Do("POST", base+"/ai/brand-voice", ai.AILearnVoiceInput{SharingReviewed: true, CampaignIDs: ids}).MustStatus(t, 422)
		if countCalls() != n {
			t.Fatal("invalid selection reached the model")
		}
	}
	exec(`UPDATE campaigns SET status = 'draft' WHERE id = $1`, selected)
	c.Do("POST", base+"/ai/brand-voice", learn).MustStatus(t, 422)
	exec(`UPDATE campaigns SET status = 'sent', deleted_at = now() WHERE id = $1`, selected)
	c.Do("POST", base+"/ai/brand-voice", learn).MustStatus(t, 422)
	exec(`UPDATE campaigns SET deleted_at = NULL WHERE id = $1`, selected)

	// A pending suggestion conflicts instead of overwriting a later manual save.
	c.Do("POST", base+"/ai/brand-voice", learn).MustStatus(t, 200).JSON(t, &suggestion)
	c.Do("PUT", voicePath, brandkit.BrandVoiceSaveInput{BaseRevision: 2, Content: manual}).MustStatus(t, 200)
	c.Do("PUT", voicePath, brandkit.BrandVoiceSaveInput{BaseRevision: 2, Content: suggestion.Content, SuggestionID: suggestion.RequestID}).MustStatus(t, 409)
	c.Do("DELETE", voicePath+"/suggestions/"+suggestion.RequestID, nil).MustStatus(t, 204)
	c.Do("DELETE", voicePath+"/suggestions/"+suggestion.RequestID, nil).MustStatus(t, 204)
	c.Do("POST", base+"/ai/brand-voice", learn).MustStatus(t, 200).JSON(t, &suggestion)
	exec(`UPDATE brand_voice_suggestions SET expires_at = now() - interval '1 second' WHERE id = $1`, suggestion.RequestID)
	c.Do("PUT", voicePath, brandkit.BrandVoiceSaveInput{BaseRevision: 3, Content: suggestion.Content, SuggestionID: suggestion.RequestID}).MustStatus(t, 409)
	if _, err := aiService(env).Purge(ctx); err != nil {
		t.Fatal(err)
	}

	// Current role, scope and the feature switch are rechecked when the
	// model responds; a revoked caller gets neither the text nor a suggestion.
	for _, revoke := range []string{
		`UPDATE workspace_members SET role_id = (SELECT id FROM roles WHERE workspace_id = $1 AND key = 'viewer') WHERE workspace_id = $1 AND user_id = $2`,
		`UPDATE workspace_members SET scope = '{"senderDomains":["outside.test"]}' WHERE workspace_id = $1 AND user_id = $2`,
		`UPDATE ai_workspace_settings SET features_off = '{brand_voice}' WHERE workspace_id = $1 AND $2::uuid IS NOT NULL`,
	} {
		for _, refusal := range []bool{false, true} {
			mu.Lock()
			beforeAnswer = func() { exec(revoke, w.ID, other.ID) }
			if refusal {
				override = `{"refusal":"Refused with source text"}`
			}
			mu.Unlock()
			n := countCalls()
			r := admin.Do("POST", base+"/ai/brand-voice", learn)
			if countCalls() != n+1 {
				t.Fatal("revocation must happen during the model request")
			}
			if r.Status != 403 && r.Status != 422 && r.Status != 409 {
				t.Fatalf("revoked access: %d %s", r.Status, r.Body)
			}
			if strings.Contains(string(r.Body), "Warm and specific") || strings.Contains(string(r.Body), "Refused with source text") {
				t.Fatal("revoked caller received suggestion text")
			}
			mu.Lock()
			beforeAnswer = nil
			override = ""
			mu.Unlock()
			exec(`UPDATE workspace_members SET role_id = (SELECT id FROM roles WHERE workspace_id = $1 AND key = 'editor'), scope = '{}' WHERE workspace_id = $1 AND user_id = $2`, w.ID, other.ID)
			exec(`UPDATE ai_workspace_settings SET features_off = '{}' WHERE workspace_id = $1`, w.ID)
		}
	}

	// Two saves against one base produce exactly one revision and one conflict.
	responses := make(chan *apptest.Response, 2)
	for _, client := range []*apptest.Client{c, admin} {
		go func() {
			responses <- client.Do("PUT", voicePath, brandkit.BrandVoiceSaveInput{BaseRevision: 3, Content: manual})
		}()
	}
	a, b := <-responses, <-responses
	if min(a.Status, b.Status) != 200 || max(a.Status, b.Status) != 409 {
		t.Fatalf("concurrent saves: %d %d", a.Status, b.Status)
	}
	var history brandkit.BrandVoiceHistory
	c.Do("GET", voicePath+"/history", nil).MustStatus(t, 200).JSON(t, &history)
	if len(history.Items) != 4 || history.Items[0].Revision != 4 || history.Items[3].Content != manual {
		t.Fatalf("immutable history: %+v", history)
	}
	var auditJSON string
	if err := env.App.Pool.QueryRow(ctx, `SELECT coalesce(jsonb_agg(to_jsonb(a))::text, '') FROM audit_events a WHERE action LIKE 'brand_voice.%' OR action='ai.sharing.reviewed'`).Scan(&auditJSON); err != nil {
		t.Fatal(err)
	}
	for _, content := range []string{manual.Tone, "Warm and specific", "ONE-SELECTED-SAMPLE"} {
		if strings.Contains(auditJSON, content) {
			t.Fatalf("audit contains voice content %q", content)
		}
	}
}
