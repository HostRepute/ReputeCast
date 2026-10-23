package welcomeprompt_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

type promptResult struct {
	Prompt *struct {
		ListID, ListName, Starter, DraftID string
		CoveredLists                       []string
	}
}

// The prompt names the newest list nothing welcomes yet, leaves out lists a
// live automation starts on and lists dismissed with "Not now", and points
// at an automation that isn't live yet instead of making another.
func TestWelcomePrompts(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Olive Owner"})
	ws := env.CreateWorkspace("Prompts", owner.ID)
	c := env.Client()
	c.SignIn("owner@example.com", "")
	viewerUser := env.CreateUser(apptest.UserOpts{Email: "viewer@example.com", Name: "Vic Viewer"})
	env.AddMember(ws, viewerUser.ID, "viewer")
	viewer := env.Client()
	viewer.SignIn("viewer@example.com", "")
	base := "/workspaces/" + ws.Slug

	lists := map[string]string{}
	for _, name := range []string{"Weekly Digest", "Product Updates", "Beta Program"} {
		var l struct{ ID string }
		c.Do(http.MethodPost, base+"/lists", map[string]any{"name": name}).MustStatus(t, 201).JSON(t, &l)
		lists[name] = l.ID
	}
	health := func() promptResult {
		t.Helper()
		var r promptResult
		c.Do(http.MethodGet, base+"/welcome-prompt?context=health", nil).MustStatus(t, 200).JSON(t, &r)
		return r
	}

	// The newest list first: nothing welcomes anyone yet.
	r := health()
	if r.Prompt == nil || r.Prompt.ListName != "Beta Program" || r.Prompt.Starter != "welcome-series" || len(r.Prompt.CoveredLists) != 0 {
		t.Fatalf("first prompt: %+v", r.Prompt)
	}
	// Viewers can't start an automation: no prompt for them, and no dismissing.
	viewer.Do(http.MethodGet, base+"/welcome-prompt?context=health", nil).MustStatus(t, 403)
	viewer.Do(http.MethodPost, base+"/welcome-prompt/dismiss", map[string]any{"listId": lists["Beta Program"]}).MustStatus(t, 403)

	// The welcome series on Weekly Digest, started from the prompt, goes live.
	var auto struct{ ID string }
	c.Do(http.MethodPost, base+"/automations", map[string]any{"template": "welcome-series", "list": lists["Weekly Digest"]}).MustStatus(t, 201).JSON(t, &auto)
	var detail struct {
		Name string
	}
	c.Do(http.MethodGet, base+"/automations/"+auto.ID, nil).MustStatus(t, 200).JSON(t, &detail)
	if detail.Name != "Welcome series · Weekly Digest" {
		t.Fatalf("automation name: %q", detail.Name)
	}
	var trig json.RawMessage
	if err := env.App.Pool.QueryRow(ctx, `SELECT trigger FROM automations WHERE id = $1`, auto.ID).Scan(&trig); err != nil {
		t.Fatal(err)
	}
	var tr struct{ Kind, ListID string }
	_ = json.Unmarshal(trig, &tr)
	if tr.Kind != "joined_list" || tr.ListID != lists["Weekly Digest"] {
		t.Fatalf("trigger: %s", trig)
	}
	c.Do(http.MethodPost, base+"/automations", map[string]any{"template": "welcome-series", "list": uuid.NewString()}).MustStatus(t, 422)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE automations SET status = 'live', live_version = 1 WHERE id = $1`, auto.ID); err != nil {
		t.Fatal(err)
	}
	r = health()
	if r.Prompt == nil || r.Prompt.ListName != "Beta Program" || len(r.Prompt.CoveredLists) != 1 || r.Prompt.CoveredLists[0] != "Weekly Digest" {
		t.Fatalf("with a live welcome: %+v", r.Prompt)
	}

	// Not now on Beta Program: Product Updates is next.
	c.Do(http.MethodPost, base+"/welcome-prompt/dismiss", map[string]any{"listId": lists["Beta Program"]}).MustStatus(t, 204)
	c.Do(http.MethodPost, base+"/welcome-prompt/dismiss", map[string]any{"listId": uuid.NewString()}).MustStatus(t, 404)
	var audits int
	if err := env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'automation.welcome_prompt_dismissed' AND target_name = 'Beta Program'`).Scan(&audits); err != nil || audits != 1 {
		t.Fatalf("audit: %d %v", audits, err)
	}
	r = health()
	if r.Prompt == nil || r.Prompt.ListName != "Product Updates" || r.Prompt.DraftID != "" {
		t.Fatalf("after dismissing: %+v", r.Prompt)
	}
	// A draft that starts on Product Updates is opened instead of a new one.
	var draft struct{ ID string }
	c.Do(http.MethodPost, base+"/automations", map[string]any{"template": "welcome-series", "list": lists["Product Updates"]}).MustStatus(t, 201).JSON(t, &draft)
	if r = health(); r.Prompt == nil || r.Prompt.DraftID != draft.ID {
		t.Fatalf("draft: %+v", r.Prompt)
	}

	// A live form offers its lists in order; one a live automation starts on
	// shows nothing.
	form := uuid.New()
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO forms (id, workspace_id, name, slug, list_ids, status) VALUES ($1, $2, 'Newsletter signup', 'newsletter', $3, 'live')`,
		form, ws.ID, []uuid.UUID{uuid.MustParse(lists["Weekly Digest"]), uuid.MustParse(lists["Product Updates"])}); err != nil {
		t.Fatal(err)
	}
	var fr promptResult
	c.Do(http.MethodGet, base+"/welcome-prompt?context=form&id="+form.String(), nil).MustStatus(t, 200).JSON(t, &fr)
	if fr.Prompt == nil || fr.Prompt.ListName != "Product Updates" {
		t.Fatalf("form: %+v", fr.Prompt)
	}
	formAuto := uuid.New()
	trigger, _ := json.Marshal(map[string]string{"kind": "form_submitted", "formId": form.String()})
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO automations (id, workspace_id, name, status, trigger, live_version) VALUES ($1, $2, 'Form welcome', 'live', $3, 1)`, formAuto, ws.ID, trigger); err != nil {
		t.Fatal(err)
	}
	fr = promptResult{}
	c.Do(http.MethodGet, base+"/welcome-prompt?context=form&id="+form.String(), nil).MustStatus(t, 200).JSON(t, &fr)
	if fr.Prompt != nil {
		t.Fatalf("a live automation starts on the form: %+v", fr.Prompt)
	}
	// A paused automation doesn't welcome anyone; a draft form isn't live yet.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE automations SET status = 'paused' WHERE id = $1`, formAuto); err != nil {
		t.Fatal(err)
	}
	fr = promptResult{}
	c.Do(http.MethodGet, base+"/welcome-prompt?context=form&id="+form.String(), nil).MustStatus(t, 200).JSON(t, &fr)
	if fr.Prompt == nil {
		t.Fatal("a paused automation doesn't cover the form")
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE forms SET status = 'draft' WHERE id = $1`, form); err != nil {
		t.Fatal(err)
	}
	fr = promptResult{}
	c.Do(http.MethodGet, base+"/welcome-prompt?context=form&id="+form.String(), nil).MustStatus(t, 200).JSON(t, &fr)
	if fr.Prompt != nil {
		t.Fatalf("draft form: %+v", fr.Prompt)
	}
	c.Do(http.MethodGet, base+"/welcome-prompt?context=form&id="+uuid.NewString(), nil).MustStatus(t, 404)
	c.Do(http.MethodGet, base+"/welcome-prompt?context=form", nil).MustStatus(t, 422)

	// A finished import offers its target lists, new ones by name.
	imp := uuid.New()
	opts, _ := json.Marshal(map[string]any{"lists": []map[string]string{{"sourceId": "a", "target": lists["Weekly Digest"]}, {"sourceId": "b", "target": "new:Product Updates"}}})
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO imports (id, workspace_id, source, status, options) VALUES ($1, $2, 'csv', 'done', $3)`, imp, ws.ID, opts); err != nil {
		t.Fatal(err)
	}
	var ir promptResult
	c.Do(http.MethodGet, base+"/welcome-prompt?context=import&id="+imp.String(), nil).MustStatus(t, 200).JSON(t, &ir)
	if ir.Prompt == nil || ir.Prompt.ListName != "Product Updates" {
		t.Fatalf("import: %+v", ir.Prompt)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE imports SET status = 'running' WHERE id = $1`, imp); err != nil {
		t.Fatal(err)
	}
	ir = promptResult{}
	c.Do(http.MethodGet, base+"/welcome-prompt?context=import&id="+imp.String(), nil).MustStatus(t, 200).JSON(t, &ir)
	if ir.Prompt != nil {
		t.Fatalf("a running import: %+v", ir.Prompt)
	}
}

// The prompt's problems read in the app's language; API clients that don't
// ask for one get English.
func TestWelcomePromptProblemsInVietnamese(t *testing.T) {
	env := apptest.New(t)
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Olive Owner"})
	ws := env.CreateWorkspace("Prompts", owner.ID)
	base := "/workspaces/" + ws.Slug

	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignIn("owner@example.com", "")
	if p := vi.Do(http.MethodGet, base+"/welcome-prompt?context=form", nil).MustStatus(t, 422).Problem(t); p["detail"] != "Hãy cho biết biểu mẫu hoặc lần nhập." {
		t.Errorf("form without id: %v", p["detail"])
	}

	en := env.Client()
	en.SignIn("owner@example.com", "")
	if p := en.Do(http.MethodGet, base+"/welcome-prompt?context=import", nil).MustStatus(t, 422).Problem(t); p["detail"] != "Name the form or the import." {
		t.Errorf("English: %v", p["detail"])
	}
}
