package transactional_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// insertCalls records n calls of a template that sent the given variables.
func insertCalls(t *testing.T, env *apptest.Env, templateID string, n int, keys []string, data string) {
	t.Helper()
	ctx := context.Background()
	var ws uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		id := uuid.Must(uuid.NewV7())
		at := time.Now().Add(-time.Duration(n-i) * time.Hour)
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO messages (id, workspace_id, stream, email, status, queued_at, sent_at, delivered_at, kind)
			VALUES ($1, $2, 'transactional', $3, 'delivered', $4::timestamptz, $4::timestamptz, $4::timestamptz + interval '2 seconds', 'api')`, id, ws, "ops@contoso.example", at); err != nil {
			t.Fatal(err)
		}
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO transactional_messages (message_id, workspace_id, source, template_id, template_key, template_name, template_version,
			data_keys, data, accepted_at) VALUES ($1, $2, 'api', $3, 'trial-ending', 'Trial ends in 3 days', 1, $4, $5, $6)`, id, ws, templateID, keys, data, at); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBreakingChange(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("sofia@acme.io")
	var tpl ttemplate
	c.Do("POST", base+"/templates", map[string]any{"name": "Trial ends in 3 days", "key": "trial-ending", "starter": "trial-ending"}).MustStatus(t, 201).JSON(t, &tpl)
	c.Do("POST", base+"/templates/trial-ending/publish", map[string]any{"baseRevision": tpl.DraftRevision}).MustStatus(t, 200)
	insertCalls(t, env, tpl.ID, 71, []string{"days_left", "first_name", "upgrade_url"}, `{"first_name":"Ops","days_left":3,"upgrade_url":"https://app.acme.io/billing"}`)

	// v2 replaces days_left with a required trial_end_date.
	var cur ttemplate
	c.Do("GET", base+"/templates/trial-ending", nil).MustStatus(t, 200).JSON(t, &cur)
	vars := []tvar{}
	for _, v := range cur.Variables {
		if v.Name != "days_left" {
			vars = append(vars, v)
		}
	}
	vars = append(vars, tvar{Name: "trial_end_date", Type: "date", Required: true, Sample: "2026-10-18"})
	mjml := strings.ReplaceAll(cur.MJML, "{{ days_left }} days", "on {{ trial_end_date }}")
	var saved struct{ DraftRevision int }
	c.Do("PUT", base+"/templates/trial-ending/draft", map[string]any{"baseRevision": cur.DraftRevision, "variables": vars, "mjml": mjml,
		"subject":  "Your Acme Cloud trial ends on {{ trial_end_date }}",
		"testData": []map[string]any{{"name": "Trial user", "data": map[string]any{"first_name": "Sam", "trial_end_date": "2026-10-18", "upgrade_url": "https://app.acme.io/billing"}}}}).
		MustStatus(t, 200).JSON(t, &saved)

	var pv struct {
		NextVersion int
		CanPublish  bool
		Breaking    struct {
			Calls    int
			Breaking []struct {
				Name             string
				CallsWithout     int
				SuggestedDefault string
			}
			Unused  []string
			Summary string
		}
		Diff struct {
			Subject   *struct{ Before, After string }
			Variables []struct{ Name, Change string }
		}
		TestRenders []struct {
			Name, Source string
			OK           bool
			Missing      []string
			Defaults     []string
		}
	}
	c.Do("POST", base+"/templates/trial-ending/publish-preview", map[string]any{}).MustStatus(t, 200).JSON(t, &pv)
	if pv.Breaking.Calls != 71 || len(pv.Breaking.Breaking) != 1 || pv.Breaking.Breaking[0].Name != "trial_end_date" || pv.Breaking.Breaking[0].CallsWithout != 71 ||
		pv.Breaking.Breaking[0].SuggestedDefault != "soon" || len(pv.Breaking.Unused) != 1 || pv.Breaking.Unused[0] != "days_left" {
		t.Fatalf("breaking: %+v", pv.Breaking)
	}
	if !strings.HasPrefix(pv.Breaking.Summary, "trial_end_date is now required, and none of the 71 calls in the last 7 days sent it.") || !strings.Contains(pv.Breaking.Summary, "v2 no longer uses") {
		t.Fatalf("summary: %s", pv.Breaking.Summary)
	}
	if pv.Diff.Subject == nil || len(pv.Diff.Variables) != 2 {
		t.Fatalf("diff: %+v", pv.Diff)
	}
	last := pv.TestRenders[len(pv.TestRenders)-1]
	if last.Source != "last_call" || last.OK || len(last.Missing) != 1 {
		t.Fatalf("last call renders: %+v", last)
	}
	p := c.Do("POST", base+"/templates/trial-ending/publish", map[string]any{"baseRevision": saved.DraftRevision}).MustStatus(t, 409).Problem(t)
	if p["code"] != "breaking_change" {
		t.Fatalf("breaking publish: %v", p)
	}
	// The recommended fix: optional with the default "soon"; the last real
	// call now renders with the default.
	fix := []map[string]any{{"name": "trial_end_date", "default": "soon"}}
	c.Do("POST", base+"/templates/trial-ending/publish-preview", map[string]any{"makeOptional": fix}).MustStatus(t, 200).JSON(t, &pv)
	last = pv.TestRenders[len(pv.TestRenders)-1]
	if len(pv.Breaking.Breaking) != 0 || !last.OK || len(last.Defaults) != 1 || !pv.CanPublish {
		t.Fatalf("with the fix: %+v %+v", pv.Breaking, last)
	}
	var pub struct{ Version int }
	c.Do("POST", base+"/templates/trial-ending/publish", map[string]any{"baseRevision": saved.DraftRevision, "makeOptional": fix}).MustStatus(t, 200).JSON(t, &pub)
	var versions struct {
		Items []struct {
			Version int
			Change  string
		}
	}
	c.Do("GET", base+"/templates/trial-ending/versions", nil).MustStatus(t, 200).JSON(t, &versions)
	if versions.Items[0].Change != "trial_end_date replaces days_left · default “soon” · new subject" {
		t.Fatalf("change: %q", versions.Items[0].Change)
	}
	// The draft carries the fix, so it matches v2.
	c.Do("GET", base+"/templates/trial-ending", nil).MustStatus(t, 200).JSON(t, &cur)
	if cur.DraftChanges {
		t.Fatal("draft should match v2")
	}
	for _, v := range cur.Variables {
		if v.Name == "trial_end_date" && (v.Required || v.Default != "soon") {
			t.Fatalf("fixed variable: %+v", v)
		}
	}
	// Publishing anyway is possible too (a later breaking change).
	vars = append(cur.Variables, tvar{Name: "plan_name", Type: "text", Required: true, Sample: "Pro"})
	c.Do("PUT", base+"/templates/trial-ending/draft", map[string]any{"baseRevision": cur.DraftRevision, "variables": vars,
		"testData": []map[string]any{{"name": "Trial user", "data": map[string]any{"first_name": "Sam", "trial_end_date": "2026-10-18", "upgrade_url": "https://app.acme.io/billing", "plan_name": "Pro"}}},
		"mjml":     strings.Replace(cur.MJML, "Pick a plan", "Pick a plan ({{ plan_name }})", 1)}).MustStatus(t, 200).JSON(t, &saved)
	c.Do("POST", base+"/templates/trial-ending/publish", map[string]any{"baseRevision": saved.DraftRevision}).MustStatus(t, 409)
	c.Do("POST", base+"/templates/trial-ending/publish", map[string]any{"baseRevision": saved.DraftRevision, "publishAnyway": true}).MustStatus(t, 200).JSON(t, &pub)
	if pub.Version != 3 {
		t.Fatalf("published anyway: %d", pub.Version)
	}
}
