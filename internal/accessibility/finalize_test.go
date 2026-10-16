package accessibility

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/mjml"
)

func TestFinalizeMJMLOutput(t *testing.T) {
	src := digest(t)
	res := mjml.Compile(src)
	if len(res.Errors) > 0 {
		t.Fatal(res.Errors)
	}
	out, add := Finalize(res.HTML, "es")
	if !add.LangAdded || add.Lang != "es" || add.Dir != "ltr" {
		t.Fatalf("additions: %+v", add)
	}
	if !strings.Contains(out, `<html lang="es" dir="ltr"`) {
		t.Errorf("html tag: %s", out[:120])
	}
	// MJML's article wrapper carries lang too (clients that drop <html>).
	if !strings.Contains(out, `role="article" lang="es" dir="ltr"`) {
		i := strings.Index(out, `role="article"`)
		t.Errorf("wrapper: %s", out[i-60:i+60])
	}
	// The Raw HTML block's table is the one MJML missed.
	if add.RoleAdded != 1 || !strings.Contains(out, `style="margin:0 auto;background:#ffffff;width:600px" role="presentation"><tr>`) {
		t.Errorf("raw table: %+v", add)
	}
	// Nothing else changes, and running it again changes nothing.
	again, add2 := Finalize(out, "es")
	if again != out || add2.RoleAdded != 0 || add2.LangAdded {
		t.Errorf("not idempotent: %+v", add2)
	}
	if strings.Count(out, "<table") != strings.Count(res.HTML, "<table") || len(out)-len(res.HTML) > 80 {
		t.Errorf("finalize changed more than attributes: %d → %d bytes", len(res.HTML), len(out))
	}
}

func TestFinalizeTables(t *testing.T) {
	src := `<!doctype html><html lang="en"><head><title>x</title></head><body>` +
		`<div style="display:none">preview</div>` +
		`<table><tr><td>layout</td></tr></table>` +
		`<table><tr><th>Plan</th><th>Price</th></tr><tr><td>Team</td><td>$29</td></tr></table>` +
		`<table cellpadding="0" style="table-layout: auto; width:100%"><tr><td>Table block</td></tr></table>` +
		`<table role="grid"><tr><td>grid</td></tr></table>` +
		`<table role="presentation"><tr><td><table><tr><td>nested</td></tr></table></td></tr></table>` +
		`<!--[if mso]><table><tr><td>outlook</td></tr></table><![endif]-->` +
		`</body></html>`
	out, add := Finalize(src, "en")
	if add.RoleAdded != 2 || add.LayoutTables != 3 || add.DataTables != 3 {
		t.Fatalf("tables: %+v", add)
	}
	if !strings.Contains(out, `<table role="presentation"><tr><td>layout`) || !strings.Contains(out, `<table role="presentation"><tr><td>nested`) {
		t.Errorf("layout tables: %s", out)
	}
	if strings.Contains(out, `<table role="presentation"><tr><th>`) || strings.Contains(out, `role="presentation" cellpadding="0" style="table-layout`) {
		t.Errorf("data tables keep their meaning: %s", out)
	}
	if !strings.Contains(out, `<!--[if mso]><table><tr><td>outlook`) {
		t.Errorf("comments stay as they are")
	}
	// lang="en" already set: dir is added; the hidden preview isn't the wrapper.
	if !strings.Contains(out, `<html lang="en" dir="ltr">`) || !strings.Contains(out, `<body lang="en" dir="ltr">`) || strings.Contains(out, `<div lang=`) {
		t.Errorf("lang: %s", out[:200])
	}
}

func TestFinalizeEmpty(t *testing.T) {
	if out, add := Finalize("", "de"); out != "" || add.Lang != "de" {
		t.Fatalf("empty: %q %+v", out, add)
	}
	if _, add := Finalize("<p>fragment</p>", ""); add.Lang != "en" {
		t.Fatalf("default language: %+v", add)
	}
}
