package campaigns_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/store"
)

// A member limited to some lists ("Editor, Beta Program list only") only
// previews as contacts on them: the composer's contact search, its random
// sample and a contact picked by id, the per-language contacts and the SMS
// recipients. An unscoped member finds the others the same way.
func TestPreviewContactsStayInScope(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	var wsID, beta, editor uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT w.id, l.id, r.id FROM workspaces w
		JOIN lists l ON l.workspace_id = w.id AND l.name = 'Beta Program'
		JOIN roles r ON r.workspace_id = w.id AND r.key = 'editor'
		WHERE w.slug = 'acme-newsletter'`).Scan(&wsID, &beta, &editor); err != nil {
		t.Fatal(err)
	}
	u := env.CreateUser(apptest.UserOpts{Email: "kim@contractor.dev", Name: "Kim Contractor"})
	scope, _ := json.Marshal(map[string]any{"lists": []uuid.UUID{beta}})
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, UserID: u.ID, RoleID: editor, Scope: scope, Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	kim := env.Client()
	kim.SignInDemo("kim@contractor.dev")
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")

	onBeta := func(id string) bool {
		t.Helper()
		var ok bool
		if err := env.App.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM list_memberships WHERE contact_id = $1 AND list_id = $2)`, id, beta).Scan(&ok); err != nil {
			t.Fatal(err)
		}
		return ok
	}
	type items struct{ Items []struct{ ID, Email string } }
	// check reads a list of contacts as Kim (all on Beta Program) and as
	// Linh (at least one who isn't).
	check := func(what, path string) {
		t.Helper()
		var mine, all items
		kim.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &mine)
		for _, it := range mine.Items {
			if !onBeta(it.ID) {
				t.Fatalf("%s: Kim got %s, who isn't on Beta Program", what, it.Email)
			}
		}
		linh.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &all)
		off := 0
		for _, it := range all.Items {
			if !onBeta(it.ID) {
				off++
			}
		}
		if off == 0 {
			t.Fatalf("%s: Linh got no one off Beta Program: %+v", what, all.Items)
		}
	}

	// A new draft has no audience: the search used to cover the workspace.
	var draft struct{ ID string }
	kim.Do("POST", ws+"/campaigns", map[string]any{}).MustStatus(t, 201).JSON(t, &draft)
	for _, q := range []string{"", "a", "%40"} {
		check("search "+q, ws+"/campaigns/"+draft.ID+"/preview-contacts?q="+q)
	}
	var mine items
	kim.Do("GET", ws+"/campaigns/"+draft.ID+"/preview-contacts", nil).MustStatus(t, 200).JSON(t, &mine)
	if len(mine.Items) == 0 {
		t.Fatal("Kim finds the contacts on Beta Program")
	}

	var pv struct {
		Contact *struct{ ID, Email string }
	}
	for range 5 {
		kim.Do("POST", ws+"/campaigns/"+draft.ID+"/preview", map[string]any{"sample": "random"}).MustStatus(t, 200).JSON(t, &pv)
		if pv.Contact == nil || pv.Contact.ID == "" || !onBeta(pv.Contact.ID) {
			t.Fatalf("random sample: %+v", pv.Contact)
		}
	}
	var off string
	if err := env.App.Pool.QueryRow(ctx, `SELECT c.id::text FROM contacts c WHERE c.workspace_id = $1 AND c.erased_at IS NULL
		AND NOT EXISTS (SELECT 1 FROM list_memberships m WHERE m.contact_id = c.id AND m.list_id = $2) LIMIT 1`, wsID, beta).Scan(&off); err != nil {
		t.Fatal(err)
	}
	kim.Do("POST", ws+"/campaigns/"+draft.ID+"/preview", map[string]any{"contactId": off}).MustStatus(t, 404)
	linh.Do("POST", ws+"/campaigns/"+draft.ID+"/preview", map[string]any{"contactId": off}).MustStatus(t, 200)

	check("languages", ws+"/campaigns/"+draft.ID+"/languages/preview-contacts")

	summit := campaignID(t, env, "Summit: doors open")
	check("SMS", ws+"/sms-campaigns/"+summit+"/preview-contacts")
	kim.Do("POST", ws+"/sms-campaigns/"+summit+"/preview", map[string]any{"contactId": off}).MustStatus(t, 404)
}

// A member limited to some lists sends only to those lists, by every
// channel: no segment (which can reach anyone), no other list, whoever
// built the audience; a copy of a campaign they couldn't target starts
// with no audience.
func TestScopedMemberSendsToTheirListsOnly(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	var wsID, beta, editor, seg uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT w.id, l.id, r.id, s.id FROM workspaces w
		JOIN lists l ON l.workspace_id = w.id AND l.name = 'Beta Program'
		JOIN roles r ON r.workspace_id = w.id AND r.key = 'editor'
		JOIN LATERAL (SELECT id FROM segments WHERE workspace_id = w.id ORDER BY created_at LIMIT 1) s ON true
		WHERE w.slug = 'acme-newsletter'`).Scan(&wsID, &beta, &editor, &seg); err != nil {
		t.Fatal(err)
	}
	u := env.CreateUser(apptest.UserOpts{Email: "kim@contractor.dev", Name: "Kim Contractor"})
	scope, _ := json.Marshal(map[string]any{"lists": []uuid.UUID{beta}})
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, UserID: u.ID, RoleID: editor, Scope: scope, Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	kim := env.Client()
	kim.SignInDemo("kim@contractor.dev")
	refused := func(what string, r *apptest.Response, status int) {
		t.Helper()
		// The reason is the problem's detail (403) or its field error (422).
		b, _ := json.Marshal(r.MustStatus(t, status).Problem(t))
		if !strings.Contains(string(b), "Your access is limited to some lists: send only to those lists, not to other lists or a segment.") {
			t.Fatalf("%s: %s", what, b)
		}
	}

	// The composer offers no segment, and refuses one.
	var cx struct{ Segments []struct{ ID string } }
	kim.Do("GET", ws+"/composer", nil).MustStatus(t, 200).JSON(t, &cx)
	if len(cx.Segments) != 0 {
		t.Fatalf("segments offered: %+v", cx.Segments)
	}
	var draft struct{ ID string }
	kim.Do("POST", ws+"/campaigns", map[string]any{}).MustStatus(t, 201).JSON(t, &draft)
	refused("segment audience", kim.Do("PATCH", ws+"/campaigns/"+draft.ID, map[string]any{"audience": map[string]any{"lists": []string{beta.String()}, "segments": []string{seg.String()}}}), 422)
	kim.Do("PATCH", ws+"/campaigns/"+draft.ID, map[string]any{"audience": map[string]any{"lists": []string{beta.String()}, "excludeSegments": []string{seg.String()}}}).MustStatus(t, 200)

	// A campaign Linh built for Weekly Digest: Kim can't send it, and a copy
	// starts with no audience.
	weekly := campaignID(t, env, "Weekly Digest #143")
	refused("send", kim.Do("POST", ws+"/campaigns/"+weekly+"/send", map[string]any{"runId": uuid.NewString()}), 403)
	var cp detail
	kim.Do("POST", ws+"/campaigns", map[string]any{"fromCampaignId": weekly}).MustStatus(t, 201).JSON(t, &cp)
	if len(cp.Audience.Lists) != 0 {
		t.Fatalf("the copy kept Weekly Digest: %+v", cp.Audience)
	}
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	var mine detail
	linh.Do("POST", ws+"/campaigns", map[string]any{"fromCampaignId": weekly}).MustStatus(t, 201).JSON(t, &mine)
	if len(mine.Audience.Lists) != 1 {
		t.Fatalf("Linh's copy keeps its audience: %+v", mine.Audience)
	}

	// SMS: no segment, no other list, and no send of the Summit list's text.
	summit := campaignID(t, env, "Summit: doors open")
	var sms struct{ Version int }
	kim.Do("GET", ws+"/sms-campaigns/"+summit, nil).MustStatus(t, 200).JSON(t, &sms)
	refused("SMS segment", kim.Do("PATCH", ws+"/sms-campaigns/"+summit, map[string]any{"version": sms.Version, "segments": []string{seg.String()}}), 422)
	refused("SMS create", kim.Do("POST", ws+"/sms-campaigns", map[string]any{"name": "Beta texts", "lists": []string{uuid.NewString()}}), 422)
	refused("SMS send", kim.Do("POST", ws+"/sms-campaigns/"+summit+"/send", map[string]any{"fingerprint": "reviewed"}), 403)

	// WhatsApp and push.
	refused("WhatsApp send", kim.Do("POST", ws+"/whatsapp-campaigns/"+campaignID(t, env, "Summit: see you tomorrow")+"/send", map[string]any{"version": 1}), 403)
	var push struct{ ID string }
	linh.Do("POST", ws+"/push-campaigns", map[string]any{"name": "Every subscriber"}).MustStatus(t, 201).JSON(t, &push)
	refused("push send", kim.Do("POST", ws+"/push-campaigns/"+push.ID+"/send", map[string]any{"version": 1}), 403)
}
