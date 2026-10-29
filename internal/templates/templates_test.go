package templates_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/templates"
)

const ws = "/workspaces/acme-newsletter"

type summary struct {
	ID, Kind, Name, Description string
	Builtin                     bool
	Thumb                       []map[string]any
}

func TestStartersCompile(t *testing.T) {
	for _, key := range []string{"blank", "weekly-digest", "product-update", "plain-text", "announcement", "product-grid", "blog-digest"} {
		src, ok := templates.StarterMJML(key, "")
		if !ok {
			t.Fatalf("starter %s missing", key)
		}
		res := mjml.Compile(src)
		if len(res.Errors) > 0 {
			t.Fatalf("starter %s: %v", key, res.Errors)
		}
		if !strings.Contains(src, templates.DefaultAddress) || strings.Contains(src, "rc-asset:") {
			t.Fatalf("starter %s placeholders not filled", key)
		}
		if lint := mjml.Lint(src); len(lint) > 0 {
			t.Fatalf("starter %s lint: %v", key, lint)
		}
	}
}

func TestTemplateGallery(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	var g struct {
		Starters     []summary
		Templates    []summary
		Sections     []summary
		LastCampaign *struct {
			ID, Name string
			Thumb    []map[string]any
		}
	}
	c.Do("GET", ws+"/templates", nil).MustStatus(t, 200).JSON(t, &g)
	names := []string{}
	for _, s := range g.Starters {
		names = append(names, s.Name)
	}
	if strings.Join(names, ",") != "Blank,Weekly digest,Product update,Plain text,Announcement,Product grid" {
		t.Fatalf("starters: %v", names)
	}
	// Other seeders may add templates and sent campaigns: check the composer demo's own.
	digest := false
	for _, tp := range g.Templates {
		if tp.Name == "Acme digest v3" && len(tp.Thumb) == 3 {
			digest = true
		}
	}
	if !digest {
		t.Fatalf("templates: %+v", g.Templates)
	}
	if len(g.Sections) < 4 {
		t.Fatalf("sections: %+v", g.Sections)
	}
	if g.LastCampaign == nil || g.LastCampaign.Name == "" || len(g.LastCampaign.Thumb) == 0 {
		t.Fatalf("last campaign: %+v", g.LastCampaign)
	}

	var tpl struct {
		Name string
		MJML string
	}
	c.Do("GET", ws+"/templates/starter:product-update", nil).MustStatus(t, 200).JSON(t, &tpl)
	if tpl.Name != "Product update" || !strings.Contains(tpl.MJML, "/media/starters/hero.png") || !strings.Contains(tpl.MJML, "500 Market St") {
		t.Fatalf("starter detail: %s", tpl.MJML)
	}
	c.Do("GET", ws+"/templates/starter:nope", nil).MustStatus(t, 404)

	// Save a section, refuse a duplicate name, rename, delete.
	var created summary
	section := `<mj-section><mj-column><mj-text>Hello</mj-text></mj-column></mj-section>`
	c.Do("POST", ws+"/templates", map[string]any{"kind": "section", "name": "Greeting", "mjml": section}).MustStatus(t, 201).JSON(t, &created)
	if created.Kind != "section" || created.Name != "Greeting" {
		t.Fatalf("created: %+v", created)
	}
	r := c.Do("POST", ws+"/templates", map[string]any{"kind": "section", "name": "greeting", "mjml": section})
	r.MustStatus(t, 422)
	if !strings.Contains(string(r.Body), "already exists") {
		t.Fatalf("duplicate: %s", r.Body)
	}
	c.Do("POST", ws+"/templates", map[string]any{"name": "Broken", "mjml": "<mjml><mj-body><mj-section></mj-body></mjml>"}).MustStatus(t, 422)
	c.Do("PUT", ws+"/templates/"+created.ID, map[string]any{"name": "Greeting block", "mjml": section}).MustStatus(t, 200).JSON(t, &created)
	if created.Name != "Greeting block" {
		t.Fatalf("rename: %+v", created)
	}
	c.Do("DELETE", ws+"/templates/"+created.ID, nil).MustStatus(t, 204)
	c.Do("GET", ws+"/templates/"+created.ID, nil).MustStatus(t, 404)
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE target_id = $1 AND action LIKE 'template.%'`, created.ID).Scan(&n); err != nil || n != 3 {
		t.Fatalf("template audit entries: %d %v", n, err)
	}

	// Viewers browse but can't save.
	v := env.Client()
	v.SignInDemo("aisha@acme.io")
	v.Do("GET", ws+"/templates", nil).MustStatus(t, 200)
	v.Do("POST", ws+"/templates", map[string]any{"name": "x", "mjml": section}).MustStatus(t, 403)

	// R4.U4: the gallery shows each template's real email: a starter compiled
	// in the brand kit, a saved one as stored, with fallbacks and no links.
	var pv struct{ HTML string }
	c.Do("GET", ws+"/templates/starter:weekly-digest/preview", nil).MustStatus(t, 200).JSON(t, &pv)
	if !strings.Contains(pv.HTML, "Hi there, two or three sentences") || !strings.Contains(pv.HTML, "/media/starters/") || strings.Contains(pv.HTML, "{{") || strings.Contains(pv.HTML, "href=") {
		t.Fatalf("starter preview: %s", pv.HTML)
	}
	for _, tp := range g.Templates {
		if tp.Name != "Acme digest v3" {
			continue
		}
		v.Do("GET", ws+"/templates/"+tp.ID+"/preview", nil).MustStatus(t, 200).JSON(t, &pv)
		if !strings.Contains(pv.HTML, "<table") || strings.Contains(pv.HTML, "href=") {
			t.Fatalf("saved template preview: %s", pv.HTML)
		}
	}
	c.Do("GET", ws+"/templates/starter:nope/preview", nil).MustStatus(t, 404)
}

// A preview is a picture: nothing in it runs, links or takes focus.
func TestPicture(t *testing.T) {
	out := templates.Picture(`<html><head><base href="https://evil.example/"><link rel="stylesheet" href="https://fonts.example/x.css"><meta http-equiv="refresh" content="0;url=https://evil.example/"></head>` +
		`<body><p onclick="steal()">Hi {{ first_name | default: "there" }}, {{ last_name }}.</p><a href="{{ unsubscribe_url }}" tabindex="2">Unsubscribe</a>` +
		`<script>alert(1)</script><form action="/x"><input name="q"><button>Go</button></form><iframe src="https://evil.example/"></iframe>` +
		`<div contenteditable="true">Notes</div><img src="/media/starters/hero.png" alt="Hero"></body></html>`)
	for _, gone := range []string{"onclick", "href=", "tabindex", "<script", "alert(1)", "<form", "<input", "<button", "<iframe", "contenteditable", "<base", "<link", "http-equiv", "{{"} {
		if strings.Contains(out, gone) {
			t.Fatalf("%s left in: %s", gone, out)
		}
	}
	for _, kept := range []string{"Hi there, .", ">Unsubscribe</a>", `<img src="/media/starters/hero.png" alt="Hero"/>`, "Notes"} {
		if !strings.Contains(out, kept) {
			t.Fatalf("%s missing: %s", kept, out)
		}
	}

	// MJML's mj-font imports each web font in the stylesheet too: the picture
	// loads nothing from elsewhere, and keeps the rest of the styles.
	styled := templates.Picture(`<html><head><style type="text/css">@import url(https://fonts.googleapis.com/css2?family=Inter:wght@400;600&display=swap);` +
		`@import 'https://fonts.example/a.css' screen; @IMPORT url("https://fonts.example/b.css");` +
		`body { font-family: Inter, Arial, sans-serif; }</style></head><body><p>Hi</p></body></html>`)
	if strings.Contains(strings.ToLower(styled), "@import") || strings.Contains(styled, "fonts.") {
		t.Fatalf("a web font import left in: %s", styled)
	}
	if !strings.Contains(styled, "body { font-family: Inter, Arial, sans-serif; }") {
		t.Fatalf("styles lost: %s", styled)
	}
}
