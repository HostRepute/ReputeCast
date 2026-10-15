package transactional_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

const base = "/workspaces/acme-newsletter/transactional"

type tvar struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Required  bool   `json:"required"`
	Sample    string `json:"sample,omitempty"`
	Default   string `json:"default,omitempty"`
	Sensitive bool   `json:"sensitive"`
}

type ttemplate struct {
	ID            string
	Key           string
	KeyLocked     bool
	Name          string
	Subject       string
	Preheader     string
	MJML          string
	HTML          string
	Variables     []tvar
	TestData      []struct{ Name string }
	DraftRevision int
	DraftChanges  bool
	LiveVersion   *int
	Category      *struct{ Name, Kind string }
	Access        struct {
		CanEdit, CanPublish bool
		PublishReason       string
		EditReason          string
	}
}

func auditActions(t *testing.T, env *apptest.Env, prefix string) []string {
	t.Helper()
	rows, err := env.App.Pool.Query(context.Background(), `SELECT action FROM audit_events WHERE action LIKE $1 ORDER BY occurred_at, id`, prefix+"%")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var a string
		_ = rows.Scan(&a)
		out = append(out, a)
	}
	return out
}

func TestTemplateLifecycle(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("sofia@acme.io") // Editor: view, edit, publish
	own := env.Client()
	own.SignInDemo("linh@acme.io") // Owner: also transactional:send (render)

	var starters struct {
		Items []struct {
			Key       string
			Name      string
			Variables []tvar
		}
	}
	c.Do("GET", base+"/starters", nil).MustStatus(t, 200).JSON(t, &starters)
	if len(starters.Items) != 8 || starters.Items[0].Key != "password-reset" || len(starters.Items[0].Variables) != 3 {
		t.Fatalf("starters: %+v", starters.Items)
	}

	var tpl ttemplate
	c.Do("POST", base+"/templates", map[string]any{"name": "Password reset", "starter": "password-reset"}).MustStatus(t, 201).JSON(t, &tpl)
	if tpl.Key != "password-reset" || tpl.LiveVersion != nil || !tpl.DraftChanges || tpl.KeyLocked || len(tpl.Variables) != 3 || len(tpl.TestData) != 2 {
		t.Fatalf("created: %+v", tpl)
	}
	if tpl.Category == nil || tpl.Category.Kind != "essential" || tpl.HTML == "" || !strings.Contains(tpl.MJML, "{{ reset_url }}") {
		t.Fatalf("starter content: %+v", tpl.Category)
	}
	// The same ID twice is refused.
	c.Do("POST", base+"/templates", map[string]any{"name": "Password reset"}).MustStatus(t, 422)
	c.Do("POST", base+"/templates", map[string]any{"name": "Odd", "key": "Not An ID"}).MustStatus(t, 422)

	// By key and by UUID.
	c.Do("GET", base+"/templates/password-reset", nil).MustStatus(t, 200)
	c.Do("GET", base+"/templates/"+tpl.ID, nil).MustStatus(t, 200)
	c.Do("GET", base+"/templates/nope", nil).MustStatus(t, 404)

	// Before the first publish the API has nothing to send.
	p := own.Do("POST", base+"/templates/password-reset/render", map[string]any{"data": map[string]any{}}).MustStatus(t, 422).Problem(t)
	if p["code"] != "template_not_published" || !strings.Contains(p["detail"].(string), "Publish it first") {
		t.Fatalf("not published: %v", p)
	}

	// Autosave, then a stale save conflicts.
	var saved struct {
		DraftRevision int
		DraftChanges  bool
	}
	c.Do("PUT", base+"/templates/password-reset/draft", map[string]any{"baseRevision": tpl.DraftRevision, "subject": "Reset your Acme Cloud password"}).MustStatus(t, 200).JSON(t, &saved)
	if saved.DraftRevision != tpl.DraftRevision+1 {
		t.Fatalf("saved: %+v", saved)
	}
	p = c.Do("PUT", base+"/templates/password-reset/draft", map[string]any{"baseRevision": tpl.DraftRevision, "subject": "Stale"}).MustStatus(t, 409).Problem(t)
	if p["code"] != "version_conflict" {
		t.Fatalf("conflict: %v", p)
	}
	// Variable names are checked.
	c.Do("PUT", base+"/templates/password-reset/draft", map[string]any{"baseRevision": saved.DraftRevision, "variables": []map[string]any{{"name": "bad name", "type": "text", "required": false, "sensitive": false}}}).MustStatus(t, 422)
	c.Do("PUT", base+"/templates/password-reset/draft", map[string]any{"baseRevision": saved.DraftRevision, "variables": []map[string]any{{"name": "to", "type": "text", "required": false, "sensitive": false}}}).MustStatus(t, 422)

	// Checks: an Essential template needs no unsubscribe link; its test data renders.
	var checks struct {
		Verdict  string
		Blocks   int
		Checks   []struct{ ID, Severity, Title string }
		TestData []struct {
			Name string
			OK   bool
		}
		Variables struct {
			Undeclared []struct{ Name string }
			Unused     []string
		}
	}
	c.Do("POST", base+"/templates/password-reset/checks", map[string]any{}).MustStatus(t, 200).JSON(t, &checks)
	if checks.Blocks != 0 || len(checks.TestData) != 2 || !checks.TestData[0].OK || !checks.TestData[1].OK {
		t.Fatalf("checks: %+v", checks)
	}
	// Unsaved content: an undeclared variable warns, an unclosed tag blocks.
	c.Do("POST", base+"/templates/password-reset/checks", map[string]any{"mjml": strings.Replace(tpl.MJML, "{{ expires_in }}", "{{ expires_in }} {{ company_name }}", 1)}).MustStatus(t, 200).JSON(t, &checks)
	if len(checks.Variables.Undeclared) != 1 || checks.Variables.Undeclared[0].Name != "company_name" {
		t.Fatalf("undeclared: %+v", checks.Variables)
	}
	c.Do("POST", base+"/templates/password-reset/checks", map[string]any{"subject": "{% if x %}Hi"}).MustStatus(t, 200).JSON(t, &checks)
	if checks.Verdict != "blocked" || checks.Checks[0].ID != "liquid.syntax" {
		t.Fatalf("liquid error: %+v", checks.Checks)
	}

	// Preview with a test data set: the optional first_name falls back.
	var prev struct {
		Subject  string
		HTML     string
		OK       bool
		Defaults []string
		Missing  []string
	}
	c.Do("POST", base+"/templates/password-reset/preview", map[string]any{"testData": "No first name"}).MustStatus(t, 200).JSON(t, &prev)
	if !prev.OK || prev.Subject != "Reset your Acme Cloud password" || !strings.Contains(prev.HTML, "Hi there") || len(prev.Defaults) != 1 {
		t.Fatalf("preview: %+v", prev.Defaults)
	}
	c.Do("POST", base+"/templates/password-reset/preview", map[string]any{"data": map[string]any{"first_name": "Sam"}}).MustStatus(t, 200).JSON(t, &prev)
	if prev.OK || len(prev.Missing) != 2 {
		t.Fatalf("preview missing: %+v", prev.Missing)
	}

	// Publish v1.
	var pv struct {
		NextVersion int
		CanPublish  bool
		Blocking    []string
		Diff        struct {
			ContentSummary string
			Variables      []struct{ Name, Change string }
		}
		Breaking struct {
			Calls    int
			Breaking []struct{ Name string }
		}
	}
	c.Do("POST", base+"/templates/password-reset/publish-preview", map[string]any{}).MustStatus(t, 200).JSON(t, &pv)
	if pv.NextVersion != 1 || !pv.CanPublish || len(pv.Blocking) != 0 || pv.Diff.ContentSummary != "First version" || len(pv.Diff.Variables) != 3 {
		t.Fatalf("publish preview: %+v", pv)
	}
	c.Do("POST", base+"/templates/password-reset/publish", map[string]any{"baseRevision": 1}).MustStatus(t, 409) // a newer save exists
	var pub struct {
		Version  int
		Previous *int
		Template ttemplate
	}
	c.Do("POST", base+"/templates/password-reset/publish", map[string]any{"baseRevision": saved.DraftRevision}).MustStatus(t, 200).JSON(t, &pub)
	if pub.Version != 1 || pub.Template.LiveVersion == nil || *pub.Template.LiveVersion != 1 || pub.Template.DraftChanges || !pub.Template.KeyLocked {
		t.Fatalf("published: %+v", pub)
	}
	// Nothing changed since v1: publishing again is refused.
	c.Do("POST", base+"/templates/password-reset/publish", map[string]any{"baseRevision": saved.DraftRevision}).MustStatus(t, 409)
	// The ID is fixed after the first publish.
	c.Do("PATCH", base+"/templates/password-reset", map[string]any{"key": "reset-password"}).MustStatus(t, 409)
	c.Do("PATCH", base+"/templates/password-reset", map[string]any{"name": "Password reset (v1)"}).MustStatus(t, 200)

	// The API renders the live version and lists missing variables.
	p = own.Do("POST", base+"/templates/password-reset/render", map[string]any{"data": map[string]any{"first_name": "Sam"}}).MustStatus(t, 422).Problem(t)
	missing, _ := p["missing"].([]any)
	if p["code"] != "missing_variables" || len(missing) != 2 || missing[0] != "reset_url" || p["title"] != "Missing variables" {
		t.Fatalf("missing: %v", p)
	}
	var rendered struct {
		Template string
		Version  int
		Subject  string
		HTML     string
		Text     string
	}
	own.Do("POST", base+"/templates/password-reset/render", map[string]any{"data": map[string]any{"reset_url": "https://app.acme.io/reset/abc", "expires_in": "30 minutes"},
		"to": map[string]any{"email": "sam.rivera@outlook.com", "name": "Sam Rivera"}}).MustStatus(t, 200).JSON(t, &rendered)
	if rendered.Version != 1 || !strings.Contains(rendered.HTML, "https://app.acme.io/reset/abc") || !strings.Contains(rendered.Text, "30 minutes") {
		t.Fatalf("render: %+v", rendered.Subject)
	}

	// v2 adds a required variable; with no calls recorded nothing breaks.
	var cur ttemplate
	c.Do("GET", base+"/templates/password-reset", nil).MustStatus(t, 200).JSON(t, &cur)
	vars := append(cur.Variables, tvar{Name: "ip_address", Type: "text", Required: true, Sample: "198.51.100.24"})
	c.Do("PUT", base+"/templates/password-reset/draft", map[string]any{"baseRevision": cur.DraftRevision, "variables": vars,
		"mjml": strings.Replace(cur.MJML, "works for {{ expires_in }}.", "works for {{ expires_in }}. Requested from {{ ip_address }}.", 1)}).MustStatus(t, 200).JSON(t, &saved)
	c.Do("POST", base+"/templates/password-reset/publish-preview", map[string]any{}).MustStatus(t, 200).JSON(t, &pv)
	if pv.NextVersion != 2 || len(pv.Diff.Variables) != 1 || pv.Diff.Variables[0].Name != "ip_address" || pv.Breaking.Calls != 0 {
		t.Fatalf("v2 preview: %+v", pv)
	}
	// The test data lacks the new required variable: it can't render, so
	// publishing waits until the data sets carry it.
	if pv.CanPublish || len(pv.Blocking) != 2 {
		t.Fatalf("test data blocks: %+v", pv.Blocking)
	}
	c.Do("PUT", base+"/templates/password-reset/draft", map[string]any{"baseRevision": saved.DraftRevision, "testData": []map[string]any{
		{"name": "New user", "data": map[string]any{"first_name": "Sam", "reset_url": "https://app.acme.io/reset/t", "expires_in": "30 minutes", "ip_address": "198.51.100.24"}},
	}}).MustStatus(t, 200).JSON(t, &saved)
	c.Do("POST", base+"/templates/password-reset/publish", map[string]any{"baseRevision": saved.DraftRevision, "note": "Shows the requesting address"}).MustStatus(t, 200).JSON(t, &pub)
	if pub.Version != 2 || pub.Template.LiveVersion == nil || *pub.Template.LiveVersion != 2 {
		t.Fatalf("v2: %+v", pub)
	}
	var versions struct {
		Items []struct {
			Version     int
			Live        bool
			Change      string
			PublishedBy string
		}
	}
	c.Do("GET", base+"/templates/password-reset/versions", nil).MustStatus(t, 200).JSON(t, &versions)
	if len(versions.Items) != 2 || !versions.Items[0].Live || versions.Items[0].Change != "Shows the requesting address" || versions.Items[1].Change != "First version" || versions.Items[1].PublishedBy != "Sofia Rossi" {
		t.Fatalf("versions: %+v", versions.Items)
	}
	// Roll back to v1 (Publish again).
	c.Do("POST", base+"/templates/password-reset/rollback", map[string]any{"version": 1}).MustStatus(t, 200).JSON(t, &pub)
	if pub.Version != 1 || pub.Previous == nil || *pub.Previous != 2 {
		t.Fatalf("rollback: %+v", pub)
	}
	c.Do("POST", base+"/templates/password-reset/rollback", map[string]any{"version": 1}).MustStatus(t, 409)
	own.Do("POST", base+"/templates/password-reset/render", map[string]any{"data": map[string]any{"reset_url": "https://x.example/r", "expires_in": "1 hour"}}).MustStatus(t, 200).JSON(t, &rendered)
	if rendered.Version != 1 {
		t.Fatalf("live after rollback: %d", rendered.Version)
	}
	var vd struct {
		Version   int
		Variables []tvar
	}
	c.Do("GET", base+"/templates/password-reset/versions/2", nil).MustStatus(t, 200).JSON(t, &vd)
	if vd.Version != 2 || len(vd.Variables) != 4 {
		t.Fatalf("version 2: %+v", vd)
	}

	// Archive: the API answers template_not_found; a live template can't be
	// deleted until it is archived.
	c.Do("DELETE", base+"/templates/password-reset", nil).MustStatus(t, 409)
	c.Do("POST", base+"/templates/password-reset/archive", nil).MustStatus(t, 200)
	p = own.Do("POST", base+"/templates/password-reset/render", map[string]any{"data": map[string]any{}}).MustStatus(t, 422).Problem(t)
	if p["code"] != "template_not_found" || !strings.Contains(p["detail"].(string), "archived") {
		t.Fatalf("archived: %v", p)
	}
	c.Do("POST", base+"/templates/password-reset/restore", nil).MustStatus(t, 200)
	c.Do("POST", base+"/templates/password-reset/archive", nil).MustStatus(t, 200)
	c.Do("DELETE", base+"/templates/password-reset", nil).MustStatus(t, 204)
	c.Do("GET", base+"/templates/password-reset", nil).MustStatus(t, 404)

	got := auditActions(t, env, "transactional.template_")
	want := []string{"transactional.template_created", "transactional.template_saved", "transactional.template_published", "transactional.template_updated",
		"transactional.template_saved", "transactional.template_saved", "transactional.template_published", "transactional.template_rolled_back", "transactional.template_archived",
		"transactional.template_restored", "transactional.template_archived", "transactional.template_deleted"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("audit: %v", got)
	}
}

func TestTemplatePermissions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	sofia := env.Client()
	sofia.SignInDemo("sofia@acme.io")
	var tpl ttemplate
	sofia.Do("POST", base+"/templates", map[string]any{"name": "Trial ends in 3 days", "key": "trial-ending", "starter": "trial-ending"}).MustStatus(t, 201).JSON(t, &tpl)
	if tpl.Category == nil || tpl.Category.Kind != "optional" {
		t.Fatalf("optional starter: %+v", tpl.Category)
	}

	// Daniel (Content Writer) edits drafts; publishing needs transactional:publish.
	daniel := env.Client()
	daniel.SignInDemo("daniel@acme.io")
	var dt ttemplate
	daniel.Do("GET", base+"/templates/trial-ending", nil).MustStatus(t, 200).JSON(t, &dt)
	if !dt.Access.CanEdit || dt.Access.CanPublish || !strings.Contains(dt.Access.PublishReason, "transactional:publish") || !strings.Contains(dt.Access.PublishReason, "Your draft is saved") {
		t.Fatalf("daniel access: %+v", dt.Access)
	}
	daniel.Do("PUT", base+"/templates/trial-ending/draft", map[string]any{"baseRevision": dt.DraftRevision, "subject": "Your trial ends soon"}).MustStatus(t, 200)
	p := daniel.Do("POST", base+"/templates/trial-ending/publish", map[string]any{"baseRevision": dt.DraftRevision + 1}).MustStatus(t, 403).Problem(t)
	if p["permission"] != "transactional:publish" {
		t.Fatalf("daniel publish: %v", p)
	}
	daniel.Do("PUT", base+"/settings", map[string]any{"linkContacts": true, "contentRetentionDays": 30, "alertBounceRate": 2, "alertFailed": 10, "alertMedianSeconds": 30}).MustStatus(t, 403)

	// Aisha (Viewer) reads everything and changes nothing.
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	var at ttemplate
	aisha.Do("GET", base+"/templates/trial-ending", nil).MustStatus(t, 200).JSON(t, &at)
	if at.Access.CanEdit || !strings.Contains(at.Access.EditReason, "transactional:edit") {
		t.Fatalf("aisha access: %+v", at.Access)
	}
	aisha.Do("POST", base+"/templates/trial-ending/preview", map[string]any{}).MustStatus(t, 200)
	aisha.Do("PUT", base+"/templates/trial-ending/draft", map[string]any{"baseRevision": 1, "subject": "x"}).MustStatus(t, 403)
	aisha.Do("POST", base+"/templates", map[string]any{"name": "Nope"}).MustStatus(t, 403)
	aisha.Do("POST", base+"/categories", map[string]any{"name": "Nope", "kind": "essential"}).MustStatus(t, 403)
	// Rendering for the API needs transactional:send.
	aisha.Do("POST", base+"/templates/trial-ending/render", map[string]any{}).MustStatus(t, 403)
}

func TestCategoriesAndSettings(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	var cat struct{ ID, Name, Kind string }
	c.Do("POST", base+"/categories", map[string]any{"name": "Reminders", "kind": "optional", "description": "Trial and renewal reminders"}).MustStatus(t, 201).JSON(t, &cat)
	c.Do("POST", base+"/categories", map[string]any{"name": "reminders", "kind": "optional", "description": "Again"}).MustStatus(t, 422)
	c.Do("POST", base+"/categories", map[string]any{"name": "No description", "kind": "optional"}).MustStatus(t, 422)
	c.Do("POST", base+"/templates", map[string]any{"name": "Trial ending", "starter": "trial-ending", "categoryId": cat.ID}).MustStatus(t, 201)
	// A category's kind is fixed while it has templates, and it can't be deleted.
	c.Do("PUT", base+"/categories/"+cat.ID, map[string]any{"name": "Reminders", "kind": "essential", "description": "x"}).MustStatus(t, 409)
	c.Do("DELETE", base+"/categories/"+cat.ID, nil).MustStatus(t, 409)
	c.Do("PUT", base+"/categories/"+cat.ID, map[string]any{"name": "Renewal reminders", "kind": "optional", "description": "Trial and renewal reminders"}).MustStatus(t, 200)
	var cats struct {
		Items []struct {
			Name      string
			Templates int64
		}
		CanEdit bool
	}
	c.Do("GET", base+"/categories", nil).MustStatus(t, 200).JSON(t, &cats)
	if len(cats.Items) != 1 || cats.Items[0].Name != "Renewal reminders" || cats.Items[0].Templates != 1 || !cats.CanEdit {
		t.Fatalf("categories: %+v", cats)
	}

	var st struct {
		LinkContacts         bool
		ContentRetentionDays int
		AlertBounceRate      float64
		AlertFailed          int
		AlertMedianSeconds   int
		CanEdit              bool
	}
	c.Do("GET", base+"/settings", nil).MustStatus(t, 200).JSON(t, &st)
	if !st.LinkContacts || st.ContentRetentionDays != 30 || st.AlertBounceRate != 2 || st.AlertFailed != 10 || st.AlertMedianSeconds != 30 || !st.CanEdit {
		t.Fatalf("defaults: %+v", st)
	}
	in := map[string]any{"linkContacts": true, "contentRetentionDays": 7, "alertBounceRate": 3, "alertFailed": 10, "alertMedianSeconds": 30, "defaultTrackOpens": false, "defaultTrackClicks": false}
	c.Do("PUT", base+"/settings", in).MustStatus(t, 200).JSON(t, &st)
	if st.ContentRetentionDays != 7 || st.AlertBounceRate != 3 {
		t.Fatalf("saved: %+v", st)
	}
	in["contentRetentionDays"] = 120
	c.Do("PUT", base+"/settings", in).MustStatus(t, 422)
	in["contentRetentionDays"] = 7
	in["defaultReplyTo"] = "not an address"
	c.Do("PUT", base+"/settings", in).MustStatus(t, 422)

	var meta json.RawMessage
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT metadata FROM audit_events WHERE action = 'transactional.settings_updated' ORDER BY occurred_at DESC LIMIT 1`).Scan(&meta); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(meta), "changed") {
		t.Fatalf("settings audit: %s", meta)
	}
	if got := auditActions(t, env, "transactional.category_"); strings.Join(got, ",") != "transactional.category_created,transactional.category_updated" {
		t.Fatalf("category audit: %v", got)
	}

	// Viewers read settings, read-only.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", base+"/settings", nil).MustStatus(t, 200).JSON(t, &st)
	if st.CanEdit {
		t.Fatal("viewer can edit settings")
	}
	v.Do("PUT", base+"/settings", in).MustStatus(t, 403)
}
