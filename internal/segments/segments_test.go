package segments_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

const ws = "/workspaces/acme-newsletter"

func audits(t *testing.T, env *apptest.Env, action string) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1 AND result = 'allowed'`, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

type segment struct {
	ID, Name string
	Count    int64
}

func TestSegmentsDemo(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")

	var list struct{ Items []segment }
	c.Do(http.MethodGet, ws+"/segments", nil).MustStatus(t, 200).JSON(t, &list)

	t.Run("seven saved segments with the boards' counts", func(t *testing.T) {
		want := map[string]int64{
			"Engaged Outlook users on paid plans": 2118,
			"Active in last 30 days":              29412,
			"Unengaged 180 days, for sunset":      3615,
			"Mailchimp import, consent pending":   2118,
			"Beta Program, opened a beta email":   1402,
			"Gmail only, clickers":                4916,
			"Customers with mrr over $50":         1087,
			// Contact scoring (X13): Sofia's segment of Hot contacts on paid plans.
			"Hot on paid plans": 1532,
		}
		if len(list.Items) != len(want) {
			t.Fatalf("%d segments, want %d", len(list.Items), len(want))
		}
		for _, s := range list.Items {
			if s.Count != want[s.Name] {
				t.Errorf("%s: stored count %d, want %d", s.Name, s.Count, want[s.Name])
			}
		}
	})

	t.Run("live counts equal the stored ones", func(t *testing.T) {
		for _, s := range list.Items {
			var r struct {
				Count int64
				Tiers struct{ Active, Cooling, AtRisk, Sunset int64 }
			}
			c.Do(http.MethodPost, ws+"/segments/"+s.ID+"/count", nil).MustStatus(t, 200).JSON(t, &r)
			if r.Count != s.Count {
				t.Errorf("%s: live %d, stored %d", s.Name, r.Count, s.Count)
			}
			if sum := r.Tiers.Active + r.Tiers.Cooling + r.Tiers.AtRisk + r.Tiers.Sunset; sum != r.Count {
				t.Errorf("%s: tier mix adds up to %d of %d", s.Name, sum, r.Count)
			}
		}
	})

	t.Run("preview, create, update and delete", func(t *testing.T) {
		def := map[string]any{"groups": []map[string]any{{"match": "all", "conditions": []map[string]any{
			{"field": "tier", "op": "is", "value": "sunset"},
		}}}}
		var p struct {
			Count        int64
			GeneratedSQL string
		}
		c.Do(http.MethodPost, ws+"/segments/preview", map[string]any{"mode": "builder", "definition": def}).MustStatus(t, 200).JSON(t, &p)
		if p.Count != 3615 || !strings.Contains(p.GeneratedSQL, "sunset") {
			t.Fatalf("preview: %d %q", p.Count, p.GeneratedSQL)
		}
		bad := map[string]any{"groups": []map[string]any{{"match": "all", "conditions": []map[string]any{{"field": "no_such_field", "op": "is", "value": "x"}}}}}
		c.Do(http.MethodPost, ws+"/segments/preview", map[string]any{"mode": "builder", "definition": bad}).MustStatus(t, 422)

		// "Language is Spanish" counts Spanish contacts, regional codes
		// included (4,180 on SettingsLanguages).
		spanish := map[string]any{"groups": []map[string]any{{"match": "all", "conditions": []map[string]any{{"field": "language", "op": "is", "value": "es"}}}}}
		c.Do(http.MethodPost, ws+"/segments/preview", map[string]any{"mode": "builder", "definition": spanish}).MustStatus(t, 200).JSON(t, &p)
		if p.Count != 4180 {
			t.Fatalf("Language is Spanish: %d", p.Count)
		}
		var cat struct {
			Fields []struct {
				Key     string
				Label   string
				Options []struct{ Value, Label string }
			}
		}
		c.Do(http.MethodGet, ws+"/segments/catalog", nil).MustStatus(t, 200).JSON(t, &cat)
		found := false
		for _, f := range cat.Fields {
			if f.Key == "language" {
				found = len(f.Options) == 5 && f.Options[1].Value == "es" && f.Options[1].Label == "Spanish" && f.Options[4].Label == "No language"
			}
		}
		if !found {
			t.Fatalf("the builder names the languages: %+v", cat.Fields)
		}

		in := map[string]any{"name": "Sunset, for the win-back", "mode": "builder", "definition": def}
		viewer.Do(http.MethodPost, ws+"/segments", in).MustStatus(t, 403)
		var s segment
		created := audits(t, env, "segment.create")
		c.Do(http.MethodPost, ws+"/segments", in).MustStatus(t, 201).JSON(t, &s)
		if s.Count != 3615 || audits(t, env, "segment.create") != created+1 {
			t.Fatalf("create: %+v, audit %d", s, audits(t, env, "segment.create"))
		}
		in["name"] = "Sunset, renamed"
		c.Do(http.MethodPut, ws+"/segments/"+s.ID, in).MustStatus(t, 200)
		c.Do(http.MethodGet, ws+"/segments/"+s.ID, nil).MustStatus(t, 200).JSON(t, &s)
		if s.Name != "Sunset, renamed" || audits(t, env, "segment.update") != 1 {
			t.Fatalf("update: %+v", s)
		}
		viewer.Do(http.MethodDelete, ws+"/segments/"+s.ID, nil).MustStatus(t, 403)
		c.Do(http.MethodDelete, ws+"/segments/"+s.ID, nil).MustStatus(t, 204)
		c.Do(http.MethodGet, ws+"/segments/"+s.ID, nil).MustStatus(t, 404)
		if audits(t, env, "segment.delete") != 1 {
			t.Fatal("delete wrote no audit entry")
		}
	})

	t.Run("user SQL is one read-only SELECT over the views", func(t *testing.T) {
		r := c.Do(http.MethodPost, ws+"/segments/preview", map[string]any{"mode": "sql", "sql": "SELECT c.id FROM contacts c WHERE c.tier = 'sunset'"})
		if r.Status != 200 && strings.Contains(string(r.Body), "turned off") {
			t.Skip("SQL segments are turned off on this database")
		}
		var p struct{ Count int64 }
		r.MustStatus(t, 200).JSON(t, &p)
		if p.Count != 3615 {
			t.Fatalf("sql count %d, want 3,615", p.Count)
		}
		for _, sql := range []string{
			"DELETE FROM contacts",
			"SELECT c.id FROM contacts c; DELETE FROM contacts",
			"SELECT id FROM users",
			"WITH d AS (DELETE FROM contacts RETURNING id) SELECT id FROM d",
			"SELECT id FROM public.contacts",
		} {
			r := c.Do(http.MethodPost, ws+"/segments/preview", map[string]any{"mode": "sql", "sql": sql})
			if r.Status != 422 {
				t.Errorf("%q: status %d, want 422: %s", sql, r.Status, r.Body)
			}
		}
		var n int64
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM contacts c JOIN workspaces w ON w.id = c.workspace_id WHERE w.slug = 'acme-newsletter'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 48213 {
			t.Fatalf("contacts changed to %d", n)
		}
	})

	t.Run("a workspace sees only its own contacts", func(t *testing.T) {
		var p struct{ Count, Total int64 }
		def := map[string]any{"groups": []map[string]any{{"match": "all", "conditions": []map[string]any{{"field": "tier", "op": "is", "value": "active"}}}}}
		c.Do(http.MethodPost, "/workspaces/acme-engineering-blog/segments/preview", map[string]any{"mode": "builder", "definition": def}).MustStatus(t, 200).JSON(t, &p)
		if p.Total != 6420 {
			t.Fatalf("engineering blog total %d, want 6,420", p.Total)
		}
	})
}
