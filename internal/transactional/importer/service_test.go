package importer_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/transactional/importer"
)

const ws = "/workspaces/acme-newsletter"

type importView struct {
	ID, Status, Account string
	KeySealed           bool
	Available           []struct{ Ref, Alias, Name string }
	Picked              []string
	Items               []struct {
		Alias, Name, Key, Status, Summary string
		SummaryLine                       int
		Failed                            bool
	}
	Converted, WithNotes, Manual int
	Created                      []struct{ ID, Key, Name, Status, Error string }
}

func audits(t *testing.T, env *apptest.Env, action string) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1`, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Postmark, end to end against the recorded account: a refused token, the
// pick list with the active templates picked, the report (an ID that's
// taken arrives with -2), and the drafts, unpublished, with their
// variables. The key is sealed and never returned, and dropped at the end.
func TestImportFromPostmark(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh, aisha := env.Client(), env.Client()
	linh.SignInDemo("linh@acme.io")
	aisha.SignInDemo("aisha@acme.io")
	key := importer.RecordedKeys["postmark"]

	linh.Do("POST", ws+"/transactional/templates", map[string]any{"name": "Password reset", "starter": "password-reset"}).MustStatus(t, 201)
	aisha.Do("POST", ws+"/transactional/imports", map[string]any{"source": "postmark", "apiKey": key}).MustStatus(t, 403)

	p := linh.Do("POST", ws+"/transactional/imports", map[string]any{"source": "postmark", "apiKey": "pm-not-this-one"}).MustStatus(t, 422).Problem(t)
	if p["code"] != "source_auth_failed" || p["detail"] != "Postmark answered 401: this token is for a different server or was revoked." {
		t.Fatalf("refused token: %v", p)
	}

	var imp importView
	r := linh.Do("POST", ws+"/transactional/imports", map[string]any{"source": "postmark", "apiKey": key}).MustStatus(t, 201)
	r.JSON(t, &imp)
	if imp.Status != "pick" || imp.Account != "Acme Cloud production" || len(imp.Available) != 9 || len(imp.Picked) != 6 || !imp.KeySealed {
		t.Fatalf("pick: %+v", imp)
	}
	base := ws + "/transactional/imports/" + imp.ID
	r = linh.Do("POST", base+"/convert", map[string]any{"refs": imp.Picked}).MustStatus(t, 200)
	if strings.Contains(string(r.Body), key) {
		t.Fatal("the key came back")
	}
	r.JSON(t, &imp)
	if imp.Status != "review" || imp.Converted != 4 || imp.WithNotes != 1 || imp.Manual != 1 {
		t.Fatalf("report: %+v", imp)
	}
	keys := map[string]string{}
	for _, it := range imp.Items {
		keys[it.Alias] = it.Key
		switch it.Alias {
		case "password-reset":
			if it.Key != "password-reset-2" || it.Summary != "The ID `password-reset` exists, so it arrives as `password-reset-2`." {
				t.Fatalf("taken ID: %+v", it)
			}
		case "receipt":
			if it.Status != "notes" || it.SummaryLine != 18 || !strings.Contains(it.Summary, "`{% for item in items %}`") {
				t.Fatalf("receipt: %+v", it)
			}
		case "comment-notification":
			if it.Status != "manual" || it.SummaryLine != 31 {
				t.Fatalf("partial: %+v", it)
			}
		case "verify-email":
			if it.Summary != "2 variables declared: name, action_url" {
				t.Fatalf("verify: %+v", it)
			}
		}
	}

	linh.Do("POST", base+"/import", nil).MustStatus(t, 200).JSON(t, &imp)
	if imp.Status != "done" || len(imp.Created) != 6 || imp.KeySealed {
		t.Fatalf("done: %+v", imp)
	}
	for _, c := range imp.Created {
		if c.Error != "" || c.ID == "" {
			t.Fatalf("draft: %+v", c)
		}
	}
	var tpl struct {
		Key         string
		LiveVersion *int
		Variables   []struct{ Name string }
		MJML, HTML  string
		Origin      map[string]any
	}
	linh.Do("GET", ws+"/transactional/templates/verify-email", nil).MustStatus(t, 200).JSON(t, &tpl)
	if tpl.LiveVersion != nil || len(tpl.Variables) != 2 || tpl.Variables[0].Name != "name" || tpl.Origin["source"] != "postmark" {
		t.Fatalf("verify-email draft: %+v", tpl)
	}
	linh.Do("GET", ws+"/transactional/templates/"+keys["receipt"], nil).MustStatus(t, 200).JSON(t, &tpl)
	if !strings.Contains(tpl.HTML+tpl.MJML, "{% for item in items %}") {
		t.Fatalf("receipt draft: %s", tpl.HTML)
	}
	if audits(t, env, "transactional.templates_imported") != 1 || audits(t, env, "transactional.import_started") != 1 || audits(t, env, "transactional.import_converted") != 1 {
		t.Fatal("audit entries are missing")
	}
	// Finished: converting again is refused.
	linh.Do("POST", base+"/convert", map[string]any{"refs": imp.Picked}).MustStatus(t, 409)
}

// Files convert as they are uploaded: an MJML file in Liquid and an HTML
// file written in Handlebars.
func TestImportFiles(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	files := []map[string]any{
		{"name": "welcome.mjml", "content": `<mjml><mj-head><mj-title>Welcome, {{ first_name }}</mj-title></mj-head><mj-body><mj-section><mj-column><mj-text>Hi {{ first_name | default: "there" }}</mj-text><mj-button href="{{ dashboard_url }}">Open</mj-button></mj-column></mj-section></mj-body></mjml>`},
		{"name": "invoice.html", "content": `<html><head><title>Invoice {{invoice_number}}</title></head><body>{{#each items}}<p>{{name}}: {{formatCurrency amount}}</p>{{/each}}</body></html>`},
	}
	var imp importView
	linh.Do("POST", ws+"/transactional/imports", map[string]any{"source": "files", "files": files}).MustStatus(t, 201).JSON(t, &imp)
	if imp.Status != "review" || len(imp.Items) != 2 || imp.Converted != 1 || imp.Manual != 1 {
		t.Fatalf("files: %+v", imp)
	}
	if imp.Items[1].Status != "manual" || !strings.Contains(imp.Items[1].Summary, "`formatCurrency`") {
		t.Fatalf("invoice: %+v", imp.Items[1])
	}
	linh.Do("POST", ws+"/transactional/imports/"+imp.ID+"/import", nil).MustStatus(t, 200).JSON(t, &imp)
	if imp.Status != "done" || len(imp.Created) != 2 || imp.Created[0].Error != "" || imp.Created[1].Error != "" {
		t.Fatalf("drafts: %+v", imp)
	}
	var tpl struct {
		Subject    string
		EditorMode string
		Variables  []struct{ Name, Type string }
	}
	linh.Do("GET", ws+"/transactional/templates/welcome", nil).MustStatus(t, 200).JSON(t, &tpl)
	if tpl.Subject != "Welcome, {{ first_name }}" || tpl.EditorMode != "visual" || len(tpl.Variables) != 2 {
		t.Fatalf("welcome: %+v", tpl)
	}
	linh.Do("GET", ws+"/transactional/templates/invoice", nil).MustStatus(t, 200).JSON(t, &tpl)
	if tpl.EditorMode != "html" {
		t.Fatalf("invoice: %+v", tpl)
	}
}

// Notifuse needs its instance's address; a cancelled import drops the key.
func TestImportFromNotifuseAndCancel(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	linh.Do("POST", ws+"/transactional/imports", map[string]any{"source": "notifuse", "apiKey": importer.RecordedKeys["notifuse"], "baseUrl": "notifuse.acme.io"}).MustStatus(t, 422)
	var imp importView
	linh.Do("POST", ws+"/transactional/imports", map[string]any{"source": "notifuse", "apiKey": importer.RecordedKeys["notifuse"], "baseUrl": importer.RecordedNotifuse}).MustStatus(t, 201).JSON(t, &imp)
	if imp.Status != "pick" || imp.Account != "Acme Cloud" || len(imp.Picked) == 0 || len(imp.Picked) >= len(imp.Available) {
		t.Fatalf("notifuse: %+v", imp)
	}
	linh.Do("POST", ws+"/transactional/imports/"+imp.ID+"/cancel", nil).MustStatus(t, 200).JSON(t, &imp)
	if imp.Status != "cancelled" || imp.KeySealed {
		t.Fatalf("cancelled: %+v", imp)
	}
	linh.Do("POST", ws+"/transactional/imports/"+imp.ID+"/import", nil).MustStatus(t, 409)
	if audits(t, env, "transactional.import_cancelled") != 1 {
		t.Fatal("the cancel isn't in the audit log")
	}
}
