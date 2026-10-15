package imports_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/imports"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/transactional/importer"
)

// fakeTemplates stands in for the transactional template importer.
type fakeTemplates struct {
	calls         int
	base, key     string
	workspaceSlug string
}

func (f *fakeTemplates) ImportNotifuseTemplates(_ context.Context, ws *store.Workspace, _ *uuid.UUID, base, key string) (imports.ImportTemplateStats, error) {
	f.calls++
	f.base, f.key, f.workspaceSlug = base, key, ws.Slug
	return imports.ImportTemplateStats{Transactional: 1, Saved: 1, Skipped: 1}, nil
}

// The Notifuse importer reads a self-hosted instance (recorded answers, no
// network): lists with counts per status, contacts by cursor with their
// status on each list, custom fields by their labels, and the templates.
func TestNotifuseImport(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	owner := env.CreateUser(apptest.UserOpts{Email: "owner@example.com", Name: "Olive Owner"})
	ws := env.CreateWorkspace("Importers", owner.ID)
	c := env.Client()
	c.SignIn("owner@example.com", "")
	base := "/workspaces/" + ws.Slug

	tpl := &fakeTemplates{}
	svc := service(env, nil)
	svc.HTTP = &http.Client{Transport: importer.RecordedTransport()}
	svc.Templates = tpl

	var d struct {
		ID         string
		SourceName string
	}
	c.Do(http.MethodPost, base+"/imports", map[string]any{"source": "notifuse"}).MustStatus(t, 201).JSON(t, &d)
	if d.SourceName != "Notifuse" {
		t.Fatalf("source name %q", d.SourceName)
	}
	id := uuid.MustParse(d.ID)

	// A refused key keeps the draft and says why.
	if _, err := svc.Connect(ctx, &ws, id, imports.ImportConnectInput{APIKey: "nf-wrong", BaseURL: importer.RecordedNotifuse}); err == nil {
		t.Fatal("a wrong key connected")
	}
	// The server address goes through the SSRF guard.
	if _, err := svc.Connect(ctx, &ws, id, imports.ImportConnectInput{APIKey: importer.RecordedKeys["notifuse"], BaseURL: "http://169.254.169.254/"}); err == nil {
		t.Fatal("a metadata address was accepted")
	}
	got, err := svc.Connect(ctx, &ws, id, imports.ImportConnectInput{APIKey: importer.RecordedKeys["notifuse"], BaseURL: importer.RecordedNotifuse + "/"})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	conn := got.Connection
	if conn == nil || conn.Account != "Acme Cloud" || conn.BaseURL != importer.RecordedNotifuse || len(conn.Lists) != 2 {
		t.Fatalf("connection: %+v", conn)
	}
	pu := conn.Lists[0]
	if pu.Name != "Product Updates" || pu.Subscribed != 3 || pu.Pending != 1 || pu.Unsubscribed != 1 || pu.Cleaned != 1 || !pu.DoubleOptIn || pu.Since != "2026" {
		t.Fatalf("Product Updates: %+v", pu)
	}
	if conn.Templates == nil || conn.Templates.Transactional != 1 || conn.Templates.Saved != 1 || conn.Templates.Skipped != 1 {
		t.Fatalf("templates: %+v", conn.Templates)
	}
	if !got.Options.ImportTemplates {
		t.Fatal("templates come along by default")
	}
	b, _ := json.Marshal(got)
	if bytes.Contains(b, []byte(importer.RecordedKeys["notifuse"])) {
		t.Fatal("the API key is returned")
	}

	var list struct{ ID string }
	c.Do(http.MethodPost, base+"/lists", map[string]any{"name": "Product Updates"}).MustStatus(t, 201).JSON(t, &list)
	opts := got.Options
	opts.Lists = []imports.ImportListChoice{{SourceID: "product_updates", Target: list.ID}}
	opts.PullSuppressions = false
	if _, err := svc.Update(ctx, &ws, id, imports.ImportUpdateInput{Options: &opts}); err != nil {
		t.Fatal(err)
	}
	c.Do(http.MethodPost, base+"/imports/"+d.ID+"/stage", nil).MustStatus(t, 200)
	if err := svc.Stage(ctx, id); err != nil {
		t.Fatalf("stage: %v", err)
	}
	var fields struct {
		Fields []struct{ Source, Label, Target string }
	}
	c.Do(http.MethodGet, base+"/imports/"+d.ID+"/fields", nil).MustStatus(t, 200).JSON(t, &fields)
	targets := map[string]string{}
	choices := []map[string]string{}
	for _, f := range fields.Fields {
		targets[f.Source] = f.Target
		choices = append(choices, map[string]string{"source": f.Source, "target": f.Target})
	}
	// Custom fields arrive under the labels the Notifuse workspace gave them.
	if targets["email"] != "email" || targets["first_name"] != "first_name" || targets["plan"] != "new:plan" || targets["trial_ends"] != "new:trial_ends" || targets["phone"] != "skip" {
		t.Fatalf("guesses: %+v", targets)
	}
	c.Do(http.MethodPut, base+"/imports/"+d.ID+"/fields", map[string]any{"fields": choices}).MustStatus(t, 200)
	var consent struct {
		Statuses []struct {
			Status   string
			Contacts int64
		}
	}
	c.Do(http.MethodGet, base+"/imports/"+d.ID+"/consent", nil).MustStatus(t, 200).JSON(t, &consent)
	byStatus := map[string]int64{}
	for _, s := range consent.Statuses {
		byStatus[s.Status] = s.Contacts
	}
	if byStatus["subscribed"] != 3 || byStatus["pending"] != 1 || byStatus["unsubscribed"] != 1 || byStatus["cleaned"] != 1 {
		t.Fatalf("statuses: %+v", consent.Statuses)
	}
	c.Do(http.MethodPut, base+"/imports/"+d.ID+"/consent", map[string]any{
		"map": map[string]string{"subscribed": "confirmed", "pending": "pending", "transactional": "skip"}, "evidence": "trust", "pullSuppressions": false, "recordHistory": true,
	}).MustStatus(t, 200)
	c.Do(http.MethodPost, base+"/imports/"+d.ID+"/preview", nil).MustStatus(t, 200)
	c.Do(http.MethodPost, base+"/imports/"+d.ID+"/start", nil).MustStatus(t, 200)
	if err := svc.Run(ctx, id); err != nil {
		t.Fatalf("run: %v", err)
	}
	done, err := svc.Get(ctx, &ws, id)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != "done" || done.Stats == nil || done.Stats.Suppressed != 2 {
		t.Fatalf("after the run: %s %+v", done.Status, done.Stats)
	}
	if done.Stats.Templates == nil || done.Stats.Templates.Transactional != 1 || tpl.calls != 1 || tpl.key != importer.RecordedKeys["notifuse"] || tpl.base != importer.RecordedNotifuse {
		t.Fatalf("templates step: %+v calls %d", done.Stats.Templates, tpl.calls)
	}
	nora, err := env.App.Q.GetContactByEmail(ctx, store.GetContactByEmailParams{WorkspaceID: ws.ID, Email: "nora@gmail.com"})
	if err != nil || nora.FirstName != "Nora" {
		t.Fatalf("nora: %+v %v", nora, err)
	}
	var plan string
	if err := env.App.Pool.QueryRow(ctx, `SELECT coalesce(attributes->>'plan', '') FROM contacts WHERE id = $1`, nora.ID).Scan(&plan); err != nil || plan != "pro" {
		t.Fatalf("custom field: %q %v", plan, err)
	}
	var supp int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM suppressions WHERE workspace_id = $1 AND email IN ('omar@outlook.com', 'rae@hotmail.com') AND source = 'import:notifuse'`, ws.ID).Scan(&supp)
	if supp != 2 {
		t.Fatalf("suppressions: %d", supp)
	}
	if audits(t, env, "import.connect") != 1 || audits(t, env, "import.finish") != 1 {
		t.Fatal("audit entries")
	}
	// Without the template importer, or with templates turned off, the step is skipped.
	opts.ImportTemplates = false
	if _, err := svc.Update(ctx, &ws, id, imports.ImportUpdateInput{Options: &opts}); err == nil {
		t.Fatal("a finished import can't change")
	}
}
