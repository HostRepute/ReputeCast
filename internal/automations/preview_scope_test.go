package automations_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
)

// Previewing or test-sending an automation email as a real contact shows
// their name, address and fields: it needs subscribers:view, and a contact
// on the caller's lists when they are limited to some, as the campaign
// preview does. Without a contact anyone who sees the flow previews it.
func TestPreviewAsAContactNeedsTheAudience(t *testing.T) {
	env, linh := signedIn(t, "linh@acme.io")
	ctx := context.Background()
	mail := &systemMail{}
	env.App.Mail = mail
	emailID := newEmail(t, linh, "Test · preview as a contact")
	preview := "/workspaces/acme-newsletter/automation-emails/" + emailID + "/preview"
	test := "/workspaces/acme-newsletter/automation-emails/" + emailID + "/test"
	beta := listID(t, env, "Beta Program")
	inside, outside := addContact(t, env, "preview-in@example.com", nil), addContact(t, env, "preview-out@example.com", nil)
	joinList(t, env, beta, inside)

	// A role that builds flows but doesn't see contacts.
	var wsID uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&wsID); err != nil {
		t.Fatal(err)
	}
	role, err := env.App.Q.CreateRole(ctx, store.CreateRoleParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, Key: "flow-builder", Name: "Flow builder",
		Permissions: authz.NewSet(authz.AutomationsView, authz.AutomationsEdit).Strings()})
	if err != nil {
		t.Fatal(err)
	}
	u := env.CreateUser(apptest.UserOpts{Email: "rio@acme.io", Name: "Rio Flows"})
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, UserID: u.ID, RoleID: role.ID, Scope: json.RawMessage(`{}`), Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	rio := env.Client()
	rio.SignInDemo("rio@acme.io")
	p := rio.Do("POST", preview, map[string]any{"contactId": inside.String()}).MustStatus(t, 403).Problem(t)
	if p["permission"] != "subscribers:view" {
		t.Fatalf("the problem names the permission: %v", p)
	}
	var pv struct{ Contact string }
	rio.Do("POST", preview, map[string]any{}).MustStatus(t, 200).JSON(t, &pv)
	if pv.Contact != "" {
		t.Fatalf("a preview without a contact: %+v", pv)
	}
	rio.Do("POST", test, map[string]any{"to": []string{"rio@acme.io"}, "contactId": inside.String()}).MustStatus(t, 403)
	if len(mail.sent) != 0 {
		t.Fatalf("a refused test sent %d emails", len(mail.sent))
	}

	// The Viewer and the Content Writer see contacts, so they preview as one.
	for _, who := range []string{"aisha@acme.io", "daniel@acme.io"} {
		c := env.Client()
		c.SignInDemo(who)
		c.Do("POST", preview, map[string]any{"contactId": outside.String()}).MustStatus(t, 200).JSON(t, &pv)
		if pv.Contact == "" {
			t.Fatalf("%s previews as the contact: %+v", who, pv)
		}
	}

	// A member limited to Beta Program previews only as its contacts.
	b, _ := uuid.Parse(beta)
	kim := scopedMember(t, env, authz.RoleEditor, authz.Scope{Lists: []uuid.UUID{b}})
	kim.Do("POST", preview, map[string]any{"contactId": outside.String()}).MustStatus(t, 404)
	kim.Do("POST", test, map[string]any{"to": []string{"kim@contractor.dev"}, "contactId": outside.String()}).MustStatus(t, 404)
	kim.Do("POST", preview, map[string]any{"contactId": inside.String()}).MustStatus(t, 200).JSON(t, &pv)
	if pv.Contact == "" {
		t.Fatalf("Kim previews as a contact on Beta Program: %+v", pv)
	}
	if len(mail.sent) != 0 {
		t.Fatalf("refused tests sent %d emails", len(mail.sent))
	}
}
