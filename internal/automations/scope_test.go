package automations_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/automations"
	"github.com/hostrepute/reputecast/internal/store"
)

// scopedMember adds kim@contractor.dev to the demo workspace with a role
// and a scope, and signs her in.
func scopedMember(t *testing.T, env *apptest.Env, roleKey string, scope authz.Scope) *apptest.Client {
	t.Helper()
	ctx := context.Background()
	var wsID, role uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT w.id, r.id FROM workspaces w JOIN roles r ON r.workspace_id = w.id AND r.key = $1
		WHERE w.slug = 'acme-newsletter'`, roleKey).Scan(&wsID, &role); err != nil {
		t.Fatal(err)
	}
	u := env.CreateUser(apptest.UserOpts{Email: "kim@contractor.dev", Name: "Kim Contractor"})
	raw, _ := json.Marshal(scope)
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, UserID: u.ID, RoleID: role, Scope: raw, Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	c := env.Client()
	c.SignInDemo("kim@contractor.dev")
	return c
}

// A member limited to some lists sees only the contacts on them at a
// flow's step, and a contact's flows only for those contacts.
func TestScopedMemberSeesOnlyTheirContactsInAFlow(t *testing.T) {
	env, linh := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	svc := engine(env, &clock{t: time.Now()})
	product, beta := listID(t, env, "Product Updates"), listID(t, env, "Beta Program")
	flow := publishFlow(t, linh, "Scoped step", automations.AutomationGraph{
		Nodes: []automations.AutomationNode{
			node("t", automations.NodeTrigger, trig(automations.AutomationTrigger{Kind: automations.TriggerJoinedList, ListID: product})),
			node("d", automations.NodeDelay, delayStep(2, "days")),
			node("m", automations.NodeEmail, emailData(newEmail(t, linh, "Test · scoped step"))),
		},
		Edges: []automations.AutomationEdge{link("t", "d", ""), link("d", "m", "")},
	})
	inside, outside := addContact(t, env, "scoped-in@example.com", nil), addContact(t, env, "scoped-out@example.com", nil)
	joinList(t, env, beta, inside)
	joinList(t, env, product, inside)
	joinList(t, env, product, outside)
	if _, err := svc.Dispatch(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	b, _ := uuid.Parse(beta)
	kim := scopedMember(t, env, authz.RoleEditor, authz.Scope{Lists: []uuid.UUID{b}})

	type page struct {
		Total int64
		Items []struct{ ContactID, Email string }
	}
	var all, mine page
	linh.Do("GET", wsPath+"/"+flow+"/steps/d/contacts", nil).MustStatus(t, 200).JSON(t, &all)
	if all.Total != 2 || len(all.Items) != 2 {
		t.Fatalf("Linh sees both: %+v", all)
	}
	kim.Do("GET", wsPath+"/"+flow+"/steps/d/contacts", nil).MustStatus(t, 200).JSON(t, &mine)
	if mine.Total != 1 || len(mine.Items) != 1 || mine.Items[0].ContactID != inside.String() {
		t.Fatalf("Kim sees only the contact on Beta Program: %+v", mine)
	}

	type runs struct {
		Items []struct{ AutomationID string }
	}
	inFlow := func(r runs) bool {
		for _, it := range r.Items {
			if it.AutomationID == flow {
				return true
			}
		}
		return false
	}
	var theirs, others runs
	kim.Do("GET", "/workspaces/acme-newsletter/contacts/"+outside.String()+"/automations", nil).MustStatus(t, 404)
	kim.Do("GET", "/workspaces/acme-newsletter/contacts/"+inside.String()+"/automations", nil).MustStatus(t, 200).JSON(t, &theirs)
	if !inFlow(theirs) {
		t.Fatalf("Kim sees the flows of a contact on Beta Program: %+v", theirs.Items)
	}
	linh.Do("GET", "/workspaces/acme-newsletter/contacts/"+outside.String()+"/automations", nil).MustStatus(t, 200).JSON(t, &others)
	if !inFlow(others) {
		t.Fatalf("Linh sees every contact's flows: %+v", others.Items)
	}
}
