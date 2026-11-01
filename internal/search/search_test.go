package search_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

type results struct {
	Groups []struct {
		Kind  string
		Items []struct {
			ID, Title, Status, Detail string
			Count                     *int64
		}
	}
}

func (r results) kinds() map[string]int {
	out := map[string]int{}
	for _, g := range r.Groups {
		out[g.Kind] = len(g.Items)
	}
	return out
}

// One call searches every kind the role can open, and nothing else
// (review 2026-11 FE-1).
func TestSearch(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	search := func(c *apptest.Client, q string) results {
		t.Helper()
		var r results
		c.Do(http.MethodGet, "/workspaces/acme-newsletter/search?q="+url.QueryEscape(q), nil).MustStatus(t, 200).JSON(t, &r)
		return r
	}

	r := search(linh, "digest")
	if r.kinds()["campaigns"] == 0 {
		t.Fatalf("campaigns: %+v", r)
	}
	for _, g := range r.Groups {
		if len(g.Items) > 5 {
			t.Fatalf("%s: %d results, at most 5", g.Kind, len(g.Items))
		}
	}
	// A contact by part of the address, and by the whole address.
	var email string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT email FROM contacts c JOIN workspaces w ON w.id = c.workspace_id
		WHERE w.slug = 'acme-newsletter' AND c.erased_at IS NULL AND c.first_name <> '' ORDER BY c.email LIMIT 1`).Scan(&email); err != nil {
		t.Fatal(err)
	}
	if r := search(linh, email); r.kinds()["contacts"] != 1 || r.Groups[0].Items[0].Title != email {
		t.Fatalf("exact address %s: %+v", email, r)
	}
	if r := search(linh, email[:4]); r.kinds()["contacts"] == 0 {
		t.Fatalf("part of an address: %+v", r)
	}
	// Too short, or a LIKE wildcard, finds nothing by accident.
	if r := search(linh, "a"); len(r.Groups) != 0 {
		t.Fatalf("one letter: %+v", r)
	}
	if r := search(linh, "%%"); len(r.Groups) != 0 {
		t.Fatalf("wildcards: %+v", r)
	}

	// A custom role with automations:view only finds automations.
	ctx := context.Background()
	if _, err := env.App.Pool.Exec(ctx, `
		WITH r AS (
			INSERT INTO roles (id, workspace_id, key, name, builtin, permissions)
			SELECT gen_random_uuid(), id, 'automations-only', 'Automations only', false, ARRAY['automations:view']
			FROM workspaces WHERE slug = 'acme-newsletter' RETURNING id, workspace_id)
		UPDATE workspace_members m SET role_id = r.id FROM r, users u
		WHERE m.workspace_id = r.workspace_id AND m.user_id = u.id AND u.email = 'aisha@acme.io'`); err != nil {
		t.Fatal(err)
	}
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	for _, q := range []string{"digest", "welcome", email} {
		for kind := range search(aisha, q).kinds() {
			if kind != "automations" {
				t.Fatalf("%q found %s, which the role can't open", q, kind)
			}
		}
	}
	if search(aisha, "welcome").kinds()["automations"] == 0 {
		t.Fatal("automations the role can open")
	}
}
