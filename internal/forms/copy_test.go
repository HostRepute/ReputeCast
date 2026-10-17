package forms_test

// Page and form copy per language (design expansion M17) on the demo data:
// Spanish and German are Live (Sofia Rossi, 8 Oct), Vietnamese is Not set
// up with ReputeCast's own labels ready; the Newsletter signup form's
// consent text has its own version in each language.

import (
	"context"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/pagelabels"
)

type copyLangJSON struct {
	Code     string `json:"code"`
	Native   string `json:"native"`
	IsSource bool   `json:"isSource"`
	Status   string `json:"status"`
	Missing  int    `json:"missing"`
	Own      int    `json:"own"`
	Written  int    `json:"written"`
	By       string `json:"by"`
	Labels   struct {
		Shipped bool   `json:"shipped"`
		Written int    `json:"written"`
		Total   int    `json:"total"`
		By      string `json:"by"`
	} `json:"labels"`
}

type pageCopyJSON struct {
	CopyLanguage string         `json:"copyLanguage"`
	Languages    []copyLangJSON `json:"languages"`
	Language     string         `json:"language"`
	Strings      []struct {
		Key      string `json:"key"`
		Page     string `json:"page"`
		Source   string `json:"source"`
		Text     string `json:"text"`
		Optional bool   `json:"optional"`
	} `json:"strings"`
}

func (p pageCopyJSON) lang(t *testing.T, code string) copyLangJSON {
	t.Helper()
	for _, l := range p.Languages {
		if l.Code == code {
			return l
		}
	}
	t.Fatalf("no %s in %+v", code, p.Languages)
	return copyLangJSON{}
}

type formCopyLangJSON struct {
	Code           string `json:"code"`
	IsSource       bool   `json:"isSource"`
	Status         string `json:"status"`
	Missing        int    `json:"missing"`
	Written        int    `json:"written"`
	Total          int    `json:"total"`
	By             string `json:"by"`
	ConsentVersion int    `json:"consentVersion"`
	ConsentFor     int    `json:"consentFor"`
	ConsentReview  bool   `json:"consentReview"`
	Copy           struct {
		Heading     string `json:"heading"`
		Button      string `json:"button"`
		ConsentText string `json:"consentText"`
		AfterSubmit string `json:"afterSubmit"`
	} `json:"copy"`
	Evidence *struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"evidence"`
}

type formCopyJSON struct {
	LanguageMode     string             `json:"languageMode"`
	FixedLanguage    string             `json:"fixedLanguage"`
	CopyLanguage     string             `json:"copyLanguage"`
	Fallback         string             `json:"fallback"`
	ConsentVersion   int                `json:"consentVersion"`
	ConsentChangedBy string             `json:"consentChangedBy"`
	Languages        []formCopyLangJSON `json:"languages"`
}

func (f formCopyJSON) lang(t *testing.T, code string) formCopyLangJSON {
	t.Helper()
	for _, l := range f.Languages {
		if l.Code == code {
			return l
		}
	}
	t.Fatalf("no %s in %+v", code, f.Languages)
	return formCopyLangJSON{}
}

func storedPageCopy(t *testing.T, env *apptest.Env, code string) (string, int) {
	t.Helper()
	var st string
	var missing int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT wl.page_copy_status, wl.page_copy_missing FROM workspace_languages wl
		JOIN workspaces w ON w.id = wl.workspace_id WHERE w.slug = 'acme-newsletter' AND wl.code = $1`, code).Scan(&st, &missing); err != nil {
		t.Fatal(err)
	}
	return st, missing
}

func TestPageCopyPerLanguage(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := signedIn(env, "linh@acme.io")

	var pc pageCopyJSON
	c.Do("GET", api+"/public-pages/copy", nil).MustStatus(t, 200).JSON(t, &pc)
	if pc.CopyLanguage != "en" || len(pc.Languages) != 4 {
		t.Fatalf("copy: %+v", pc)
	}
	en, es, vi := pc.lang(t, "en"), pc.lang(t, "es"), pc.lang(t, "vi")
	if !en.IsSource || en.Status != "live" || !en.Labels.Shipped {
		t.Fatalf("English: %+v", en)
	}
	if es.Status != "live" || es.Missing != 0 || es.Native != "Español" || es.By != "Sofia Rossi" || es.Labels.Shipped || es.Labels.Written != pagelabels.Count() || es.Labels.Written != es.Labels.Total {
		t.Fatalf("Spanish: %+v", es)
	}
	if pc.lang(t, "de").Status != "live" {
		t.Fatalf("German: %+v", pc.lang(t, "de"))
	}
	// Vietnamese: the labels ship with ReputeCast, Acme's own text isn't written.
	if vi.Status != "not_set_up" || !vi.Labels.Shipped || vi.Written != 0 || vi.Missing != vi.Own || vi.Own == 0 {
		t.Fatalf("Vietnamese: %+v", vi)
	}
	// Reading keeps Settings › General's statuses as the sample has them.
	if st, _ := storedPageCopy(t, env, "es"); st != "live" {
		t.Fatalf("stored Spanish status %s", st)
	}
	if st, missing := storedPageCopy(t, env, "vi"); st != "not_set_up" || missing != vi.Missing {
		t.Fatalf("stored Vietnamese: %s, %d missing", st, missing)
	}

	// One language's strings, next to the pages' own text.
	c.Do("GET", api+"/public-pages/copy?language=es-MX", nil).MustStatus(t, 200).JSON(t, &pc)
	found := false
	for _, s := range pc.Strings {
		if s.Key == "subscribe.heading" {
			found = s.Source == "The Acme newsletter" && s.Text == "El boletín de Acme" && s.Page == "subscribe" && !s.Optional
		}
	}
	if pc.Language != "es" || !found {
		t.Fatalf("Spanish strings: %+v", pc.Strings)
	}
	c.Do("GET", api+"/public-pages/copy?language=pt", nil).MustStatus(t, 422)

	// Clearing a string makes Spanish Incomplete, for the pages and Settings › General.
	before := audits(t, env, "page_copy.update")
	c.Do("PUT", api+"/public-pages/copy/es", map[string]any{"strings": map[string]string{"subscribe.heading": ""}}).MustStatus(t, 200).JSON(t, &pc)
	if es = pc.lang(t, "es"); es.Status != "incomplete" || es.Missing != 1 {
		t.Fatalf("after clearing: %+v", es)
	}
	if st, missing := storedPageCopy(t, env, "es"); st != "incomplete" || missing != 1 {
		t.Fatalf("stored after clearing: %s %d", st, missing)
	}
	c.Do("PUT", api+"/public-pages/copy/es", map[string]any{"strings": map[string]string{"subscribe.heading": "El boletín de Acme"}}).MustStatus(t, 200).JSON(t, &pc)
	if es = pc.lang(t, "es"); es.Status != "live" || es.By != "Linh Nguyen" {
		t.Fatalf("after writing it again: %+v", es)
	}
	if audits(t, env, "page_copy.update") != before+2 {
		t.Fatal("page_copy.update audit entries missing")
	}
	c.Do("PUT", api+"/public-pages/copy/es", map[string]any{"strings": map[string]string{"nope": "x"}}).MustStatus(t, 422)
	c.Do("PUT", api+"/public-pages/copy/es", map[string]any{"strings": map[string]string{"privacyUrl": "javascript:alert(1)"}}).MustStatus(t, 422)
	c.Do("PUT", api+"/public-pages/copy/en", map[string]any{"strings": map[string]string{"subscribe.heading": "x"}}).MustStatus(t, 422)

	// Changing the pages' own text leaves the translations to catch up.
	var pages struct {
		Pages struct {
			Subscribe map[string]any `json:"subscribe"`
		} `json:"pages"`
	}
	c.Do("GET", api+"/public-pages", nil).MustStatus(t, 200).JSON(t, &pages)
	sub := pages.Pages.Subscribe
	sub["footnote"] = ""
	sub["description"] = "A new description."
	c.Do("PUT", api+"/public-pages/pages", map[string]any{"subscribe": sub}).MustStatus(t, 200)
	c.Do("GET", api+"/public-pages/copy", nil).MustStatus(t, 200).JSON(t, &pc)
	if es = pc.lang(t, "es"); es.Status != "live" {
		// The description's translation still stands for the new text: the
		// key is the same, so nothing goes missing (the footnote went away).
		t.Fatalf("after the source changed: %+v", es)
	}
}

func TestPageLabels(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := signedIn(env, "linh@acme.io")

	var vi struct {
		Shipped bool `json:"shipped"`
		Written int  `json:"written"`
		Total   int  `json:"total"`
		Labels  []struct {
			Key, Source, Text string
			Arguments         []string
		} `json:"labels"`
	}
	c.Do("GET", api+"/public-pages/labels/vi", nil).MustStatus(t, 200).JSON(t, &vi)
	if !vi.Shipped || vi.Written != vi.Total || len(vi.Labels) != pagelabels.Count() {
		t.Fatalf("Vietnamese labels: shipped %v %d/%d, %d labels", vi.Shipped, vi.Written, vi.Total, len(vi.Labels))
	}
	for _, l := range vi.Labels {
		if l.Key == "form.button" && l.Text != "Đăng ký" {
			t.Fatalf("form.button in Vietnamese: %q", l.Text)
		}
	}
	p := c.Do("PUT", api+"/public-pages/labels/vi", map[string]any{"labels": map[string]string{"form.button": "x"}}).MustStatus(t, 409).Problem(t)
	if !strings.Contains(string(mustJSON(t, p)), "ships the Vietnamese labels") {
		t.Fatalf("shipped: %v", p)
	}

	// Spanish labels are the workspace's; each keeps to the English placeholders.
	p = c.Do("PUT", api+"/public-pages/labels/es", map[string]any{"labels": map[string]string{"form.closed.text": "{name} no acepta suscripciones."}}).MustStatus(t, 422).Problem(t)
	if !strings.Contains(string(mustJSON(t, p)), "{name}") || !strings.Contains(string(mustJSON(t, p)), "{publisher}") {
		t.Fatalf("argument check: %v", p)
	}
	c.Do("PUT", api+"/public-pages/labels/es", map[string]any{"labels": map[string]string{"form.closed.text": "{publisher no"}}).MustStatus(t, 422)
	var es struct {
		Written int    `json:"written"`
		By      string `json:"by"`
	}
	c.Do("PUT", api+"/public-pages/labels/es", map[string]any{"labels": map[string]string{"form.button": ""}}).MustStatus(t, 200).JSON(t, &es)
	if es.Written != pagelabels.Count()-1 || es.By != "Linh Nguyen" {
		t.Fatalf("after removing one: %+v", es)
	}
	if st, missing := storedPageCopy(t, env, "es"); st != "incomplete" || missing != 1 {
		t.Fatalf("a missing label makes Spanish Incomplete: %s %d", st, missing)
	}
	if audits(t, env, "page_copy.labels") != 1 {
		t.Fatal("page_copy.labels audit entry missing")
	}
}

func TestFormCopyPerLanguage(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := signedIn(env, "linh@acme.io")
	signup := formByName(t, overview(t, c), "Newsletter signup")
	path := api + "/forms/" + signup + "/copy"

	var fc formCopyJSON
	c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &fc)
	if fc.LanguageMode != "page" || fc.CopyLanguage != "en" || fc.Fallback != "en" || fc.ConsentVersion != 3 || len(fc.Languages) != 4 {
		t.Fatalf("form copy: %+v", fc)
	}
	en, es, vi := fc.lang(t, "en"), fc.lang(t, "es"), fc.lang(t, "vi")
	if !en.IsSource || en.Status != "live" || en.ConsentVersion != 3 || en.Copy.Button != "Subscribe" {
		t.Fatalf("English: %+v", en)
	}
	if es.Status != "live" || es.ConsentVersion != 1 || es.ConsentFor != 3 || es.ConsentReview || es.By != "Sofia Rossi" || es.Copy.Button != "Suscribirme" {
		t.Fatalf("Spanish: %+v", es)
	}
	if es.Evidence == nil || es.Evidence.Name != "Lucía Fernández" || es.Evidence.Version != "v1" {
		t.Fatalf("Spanish consent evidence: %+v", es.Evidence)
	}
	if vi.Status != "not_set_up" || vi.Written != 0 || vi.Missing != vi.Total {
		t.Fatalf("Vietnamese: %+v", vi)
	}

	// The English consent text changes: Spanish and German are flagged.
	f := getForm(t, c, signup)
	in := inputOf(f)
	in["consentText"] = "I agree to receive emails from Acme. Unsubscribe with one click from any email. See the privacy policy."
	c.Do("PUT", api+"/forms/"+signup, in).MustStatus(t, 200)
	c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &fc)
	if fc.ConsentVersion != 4 || fc.ConsentChangedBy != "Linh Nguyen" || !fc.lang(t, "es").ConsentReview || !fc.lang(t, "de").ConsentReview {
		t.Fatalf("after the English consent changed: %+v", fc)
	}
	// Flagged texts keep being shown: Spanish stays Live.
	if fc.lang(t, "es").Status != "live" {
		t.Fatal("a flagged consent text shouldn't take Spanish off Live")
	}
	// "Spanish is still right"…
	c.Do("POST", path+"/es/consent-reviewed", nil).MustStatus(t, 200).JSON(t, &fc)
	if es = fc.lang(t, "es"); es.ConsentReview || es.ConsentVersion != 1 || es.ConsentFor != 4 {
		t.Fatalf("after confirming: %+v", es)
	}
	c.Do("POST", path+"/es/consent-reviewed", nil).MustStatus(t, 409)
	if audits(t, env, "form.consent_reviewed") != 1 {
		t.Fatal("form.consent_reviewed audit entry missing")
	}
	// …and "Update German": a new German text is its version 2, for English v4.
	de := fc.lang(t, "de")
	c.Do("PUT", path+"/de", map[string]any{"heading": de.Copy.Heading, "description": "Eine E-Mail pro Woche.", "button": de.Copy.Button,
		"consentText": "Ich möchte E-Mails von Acme erhalten. Abmelden geht mit einem Klick aus jeder E-Mail. Siehe Datenschutzerklärung.",
		"afterSubmit": de.Copy.AfterSubmit, "fields": map[string]string{}}).MustStatus(t, 200).JSON(t, &fc)
	if de = fc.lang(t, "de"); de.ConsentVersion != 2 || de.ConsentFor != 4 || de.ConsentReview || de.Status != "live" {
		t.Fatalf("after updating German: %+v", de)
	}

	// Writing Vietnamese from scratch: Incomplete until every string is there.
	c.Do("PUT", path+"/vi", map[string]any{"heading": "Bản tin Acme", "description": "", "button": "Đăng ký", "consentText": "", "afterSubmit": "", "fields": map[string]string{}}).
		MustStatus(t, 200).JSON(t, &fc)
	if vi = fc.lang(t, "vi"); vi.Status != "incomplete" || vi.Written != 2 || vi.Missing != vi.Total-2 {
		t.Fatalf("Vietnamese, two strings in: %+v", vi)
	}
	c.Do("PUT", path+"/pt", map[string]any{"heading": "x", "description": "", "button": "x", "consentText": "", "afterSubmit": "", "fields": map[string]string{}}).MustStatus(t, 422)
	// The form's own language edits the form: a new consent version.
	c.Do("PUT", path+"/en", map[string]any{"heading": "The Acme newsletter", "description": f.Content["description"], "button": "Subscribe",
		"consentText": "I agree to receive emails from Acme. See the privacy policy.", "afterSubmit": "", "fields": map[string]string{}}).MustStatus(t, 200).JSON(t, &fc)
	if fc.ConsentVersion != 5 || !fc.lang(t, "de").ConsentReview {
		t.Fatalf("after changing English from the language view: %+v", fc)
	}
	if audits(t, env, "form.copy_update") != 3 {
		t.Fatalf("form.copy_update audit entries: %d", audits(t, env, "form.copy_update"))
	}

	// Form language.
	c.Do("PUT", api+"/forms/"+signup+"/language", map[string]any{"mode": "fixed", "fixed": "de-AT"}).MustStatus(t, 200).JSON(t, &fc)
	if fc.LanguageMode != "fixed" || fc.FixedLanguage != "de" {
		t.Fatalf("fixed: %+v", fc)
	}
	c.Do("PUT", api+"/forms/"+signup+"/language", map[string]any{"mode": "fixed", "fixed": "pt"}).MustStatus(t, 422)
	c.Do("PUT", api+"/forms/"+signup+"/language", map[string]any{"mode": "browser"}).MustStatus(t, 200).JSON(t, &fc)
	if fc.LanguageMode != "browser" || fc.FixedLanguage != "" || audits(t, env, "form.language") != 2 {
		t.Fatalf("browser: %+v", fc)
	}
}

func TestCopyPermissions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	signup := formByName(t, overview(t, signedIn(env, "linh@acme.io")), "Newsletter signup")
	for _, email := range []string{"aisha@acme.io", "daniel@acme.io"} {
		c := signedIn(env, email)
		c.Do("GET", api+"/public-pages/copy?language=es", nil).MustStatus(t, 200)
		c.Do("GET", api+"/public-pages/labels/es", nil).MustStatus(t, 200)
		c.Do("GET", api+"/forms/"+signup+"/copy", nil).MustStatus(t, 200)
		c.Do("PUT", api+"/public-pages/copy/es", map[string]any{"strings": map[string]string{}}).MustStatus(t, 403)
		c.Do("PUT", api+"/public-pages/labels/es", map[string]any{"labels": map[string]string{}}).MustStatus(t, 403)
		c.Do("PUT", api+"/forms/"+signup+"/copy/es", map[string]any{"heading": "x", "description": "", "button": "x", "consentText": "", "afterSubmit": "", "fields": map[string]string{}}).MustStatus(t, 403)
		c.Do("POST", api+"/forms/"+signup+"/copy/es/consent-reviewed", nil).MustStatus(t, 403)
		c.Do("PUT", api+"/forms/"+signup+"/language", map[string]any{"mode": "browser"}).MustStatus(t, 403)
	}
}
