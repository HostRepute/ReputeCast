package demoworkspace_test

import (
	"context"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// PR-13: "Explore with demo data" loads the sample into a separate
// workspace the person owns, once; its connections are demo connections.
func TestExploreWithDemoData(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "new.owner@example.com", Name: "New Owner"})
	env.CreateWorkspace("Acme Newsletter", owner.ID)
	c := env.Client()
	c.SignIn("new.owner@example.com", "correct horse battery staple")
	var first struct {
		Slug    string
		Name    string
		Created bool
	}
	c.Do("POST", "/api/v1/demo-workspace", nil).MustStatus(t, 200).JSON(t, &first)
	if !first.Created || first.Slug == "" || first.Name != "Acme Newsletter (demo)" {
		t.Fatalf("demo workspace %+v", first)
	}
	var lists, contacts, providers, demoProviders, campaigns int
	if err := env.App.Pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM lists l JOIN workspaces w ON w.id = l.workspace_id WHERE w.slug = $1),
		(SELECT count(*) FROM contacts c JOIN workspaces w ON w.id = c.workspace_id WHERE w.slug = $1),
		(SELECT count(*) FROM providers p JOIN workspaces w ON w.id = p.workspace_id WHERE w.slug = $1),
		(SELECT count(*) FROM providers p JOIN workspaces w ON w.id = p.workspace_id WHERE w.slug = $1 AND (p.config->>'demo')::boolean),
		(SELECT count(*) FROM campaigns c JOIN workspaces w ON w.id = c.workspace_id WHERE w.slug = $1)`, first.Slug).Scan(&lists, &contacts, &providers, &demoProviders, &campaigns); err != nil {
		t.Fatal(err)
	}
	if lists != 3 || contacts < 200 || providers != 3 || demoProviders != 3 || campaigns != 1 {
		t.Fatalf("sample: %d lists, %d contacts, %d providers (%d demo), %d campaigns", lists, contacts, providers, demoProviders, campaigns)
	}
	// The overview of the sample shows its routing.
	c.Do("GET", "/api/v1/workspaces/"+first.Slug+"/routing", nil).MustStatus(t, 200)

	// Asking again opens the same workspace.
	var again struct {
		Slug    string
		Created bool
	}
	c.Do("POST", "/api/v1/demo-workspace", nil).MustStatus(t, 200).JSON(t, &again)
	if again.Created || again.Slug != first.Slug {
		t.Fatalf("second demo workspace %+v", again)
	}
}
