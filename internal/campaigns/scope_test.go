package campaigns_test

import (
	"context"
	"encoding/json"
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
