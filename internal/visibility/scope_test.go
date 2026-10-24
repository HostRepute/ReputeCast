package visibility_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/visibility"
)

// A member limited to some lists only previews versions as contacts on them.
func TestPreviewContactsStayInScope(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")

	var all, two struct{ ID string }
	linh.Do("POST", ws+"/lists", map[string]any{"name": "Scope test"}).MustStatus(t, 201).JSON(t, &all)
	linh.Do("POST", ws+"/lists", map[string]any{"name": "Scope test, two"}).MustStatus(t, 201).JSON(t, &two)
	join := func(list string, emails ...string) {
		t.Helper()
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO list_memberships (list_id, contact_id, status, subscribed_at)
			SELECT $1, c.id, 'subscribed', now() FROM contacts c JOIN workspaces w ON w.id = c.workspace_id
			WHERE w.slug = 'acme-newsletter' AND c.email = ANY($2)`, list, emails); err != nil {
			t.Fatal(err)
		}
	}
	join(all.ID, "aisha@rahmanlabs.io", "sam.rivera@outlook.com", "hannah@bergconsulting.de", "kenji.w@icloud.com")
	join(two.ID, "aisha@rahmanlabs.io", "sam.rivera@outlook.com")

	var draft struct{ ID string }
	linh.Do("POST", ws+"/campaigns", map[string]any{"name": "Scope test draft"}).MustStatus(t, 201).JSON(t, &draft)
	linh.Do("PATCH", ws+"/campaigns/"+draft.ID, map[string]any{"audience": map[string]any{"lists": []string{all.ID}}}).MustStatus(t, 200)
	rule := visibility.VisibilityRule{ID: "two001", Show: visibility.ShowOnly, Conditions: []visibility.VisibilityCondition{{Kind: visibility.KindList, Op: "includes", Value: two.ID}}}
	src := `<mjml><mj-body>
<mj-section><mj-column><mj-text>Acme</mj-text></mj-column></mj-section>
<mj-section rc-name="For the two" rc-visible="` + attr(rule) + `"><mj-column><mj-text>Just for you.</mj-text></mj-column></mj-section>
<!-- rc:footer -->
<mj-section><mj-column><mj-text><a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section>
</mj-body></mjml>`
	res := mjml.Compile(src)
	if len(res.Errors) > 0 {
		t.Fatal(res.Errors)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE campaigns SET mjml = $2, html = $3 WHERE id = $1`, draft.ID, src, res.HTML); err != nil {
		t.Fatal(err)
	}

	var wsID, editor uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT w.id, r.id FROM workspaces w JOIN roles r ON r.workspace_id = w.id AND r.key = 'editor' WHERE w.slug = 'acme-newsletter'`).Scan(&wsID, &editor); err != nil {
		t.Fatal(err)
	}
	u := env.CreateUser(apptest.UserOpts{Email: "kim@contractor.dev", Name: "Kim Contractor"})
	scope, _ := json.Marshal(map[string]any{"lists": []string{two.ID}})
	if _, err := env.App.Q.CreateMember(ctx, store.CreateMemberParams{ID: uuid.Must(uuid.NewV7()), WorkspaceID: wsID, UserID: u.ID, RoleID: editor, Scope: scope, Source: "invite"}); err != nil {
		t.Fatal(err)
	}
	kim := env.Client()
	kim.SignInDemo("kim@contractor.dev")

	emails := func(c *apptest.Client) map[string]bool {
		t.Helper()
		var who struct{ Items []struct{ Email string } }
		c.Do("GET", ws+"/campaigns/"+draft.ID+"/visibility/preview-contacts", nil).MustStatus(t, 200).JSON(t, &who)
		out := map[string]bool{}
		for _, x := range who.Items {
			out[x.Email] = true
		}
		return out
	}
	if got := emails(linh); !got["hannah@bergconsulting.de"] && !got["kenji.w@icloud.com"] {
		t.Fatalf("Linh previews the version without the block as one of the other two: %v", got)
	}
	got := emails(kim)
	if len(got) == 0 || got["hannah@bergconsulting.de"] || got["kenji.w@icloud.com"] {
		t.Fatalf("Kim previews only as Aisha or Sam: %v", got)
	}
}
