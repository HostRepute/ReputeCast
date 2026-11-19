package assistant_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/ai"
	"github.com/hostrepute/reputecast/internal/ai/assistant"
	"github.com/hostrepute/reputecast/internal/apptest"
)

// A thread is its owner's: a colleague and the workspace's owner (who
// manages integrations) get 404 on every operation and never see it
// listed. People without ai:use and API keys can't use the assistant. The
// title is sealed at rest and never audited.
func TestAssistantThreadsAreTheOwners(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	const sentinel, renamed = "Sentinel title 7f3a", "Renamed 9c1e"
	threads := f.base + "/assistant/threads"
	th := f.thread(f.mc, map[string]any{"title": sentinel, "requestKey": "thread-1", "screen": "campaign.report"})
	if th.Title != sentinel || th.State != "open" || th.Screen != "campaign.report" || th.Model == nil || th.Model.ID != f.modelID || th.EventSeq != 0 ||
		len(th.Messages) != 0 || len(th.Contexts) != 0 || th.Turn != nil || !th.ExpiresAt.After(th.LastActivityAt) {
		t.Fatalf("a new thread: %+v", th)
	}
	// A retry returns the same thread; the same key for another request is refused.
	if again := f.thread(f.mc, map[string]any{"title": sentinel, "requestKey": "thread-1", "screen": "campaign.report"}); again.ID != th.ID {
		t.Fatalf("the retry made thread %s, want %s", again.ID, th.ID)
	}
	if p := f.mc.Do(http.MethodPost, threads, map[string]any{"title": "Other", "requestKey": "thread-1"}).MustStatus(t, 409).Problem(t); p["code"] != "ai_assistant_request_key" {
		t.Fatalf("a reused request key: %v", p)
	}
	path := threads + "/" + th.ID.String()

	// Nobody else's, managers included.
	for name, c := range map[string]*apptest.Client{"a colleague": f.cc, "the owner": f.oc} {
		c.Do(http.MethodGet, path, nil).MustStatus(t, 404)
		c.Do(http.MethodPatch, path, map[string]any{"title": "Mine now"}).MustStatus(t, 404)
		c.Do(http.MethodPost, path+"/contexts", map[string]any{"kind": "campaign", "id": uuid.New(), "sharingReviewed": true}).MustStatus(t, 404)
		c.Do(http.MethodDelete, path+"/contexts/"+uuid.NewString(), nil).MustStatus(t, 404)
		c.Do(http.MethodDelete, path, nil).MustStatus(t, 404)
		var list assistant.AIThreadList
		c.Do(http.MethodGet, threads, nil).MustStatus(t, 200).JSON(t, &list)
		for _, item := range list.Items {
			if item.ID == th.ID {
				t.Fatalf("%s lists the member's thread", name)
			}
		}
	}
	// No ai:use, or an API key: no assistant.
	f.vc.Do(http.MethodGet, threads, nil).MustStatus(t, 403)
	f.vc.Do(http.MethodPost, threads, map[string]any{"requestKey": "viewer-1"}).MustStatus(t, 403)
	var key struct{ Secret string }
	f.oc.Do(http.MethodPost, "/workspaces/"+f.ws.Slug+"/api-keys", map[string]any{"name": "Reader", "permissions": []string{"integrations:view"}}).MustStatus(t, 201).JSON(t, &key)
	kc := f.env.Client()
	kc.Bearer = key.Secret
	if p := kc.Do(http.MethodGet, threads, nil).MustStatus(t, 403).Problem(t); p["code"] != "humans_only" {
		t.Fatalf("an API key: %v", p)
	}

	// Renamed, listed and found by its title; sealed at rest for this thread.
	f.mc.Do(http.MethodPatch, path, map[string]any{"title": renamed}).MustStatus(t, 200).JSON(t, &th)
	f.thread(f.mc, map[string]any{"title": "Another one"})
	var found assistant.AIThreadList
	f.mc.Do(http.MethodGet, threads+"?q=RENAMED", nil).MustStatus(t, 200).JSON(t, &found)
	if th.Title != renamed || len(found.Items) != 1 || found.Items[0].ID != th.ID {
		t.Fatalf("after renaming: %q, search found %+v", th.Title, found.Items)
	}
	var sealed []byte
	if err := f.env.App.Pool.QueryRow(ctx, `SELECT title_sealed FROM ai_assistant_threads WHERE id = $1`, th.ID).Scan(&sealed); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte(renamed)) {
		t.Fatal("the title is stored readable")
	}
	if title, err := f.env.App.Secrets.OpenString(sealed, "ai_assistant_thread:"+th.ID.String()+":title"); err != nil || title != renamed {
		t.Fatalf("the title doesn't open as this thread's: %q %v", title, err)
	}
	if n := f.count(`SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND target_id = $2 AND action IN ('ai_assistant_thread.created', 'ai_assistant_thread.updated')
		AND metadata->>'via' = 'ai_assistant'`, f.ws.ID, th.ID.String()); n != 2 {
		t.Fatalf("%d audit entries of creating and renaming", n)
	}

	// With the assistant off, the thread can't be read but can be deleted.
	f.setAssistant(ai.AIAssistantSettings{Enabled: false, Limits: f.settings.Limits, PerPerson: 2, PerWorkspace: 10})
	if p := f.mc.Do(http.MethodGet, path, nil).MustStatus(t, 409).Problem(t); p["code"] != "ai_assistant_off" {
		t.Fatalf("a thread while the assistant is off: %v", p)
	}
	f.mc.Do(http.MethodDelete, path, nil).MustStatus(t, 204)
	f.mc.Do(http.MethodDelete, path, nil).MustStatus(t, 404)
	if n := f.count(`SELECT count(*) FROM ai_assistant_threads WHERE id = $1`, th.ID); n != 0 {
		t.Fatal("the deleted thread is still there")
	}
	if n := f.count(`SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND target_id = $2 AND action = 'ai_assistant_thread.deleted'`, f.ws.ID, th.ID.String()); n != 1 {
		t.Fatalf("%d audit entries of the deletion", n)
	}
	for _, text := range []string{sentinel, renamed} {
		if n := f.count(`SELECT count(*) FROM audit_events WHERE concat(before::text, after::text, metadata::text, target_name) LIKE '%' || $1 || '%'`, text); n != 0 {
			t.Fatalf("the audit log has the title %q", text)
		}
	}
}
