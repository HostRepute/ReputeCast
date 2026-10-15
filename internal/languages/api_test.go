package languages_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Every content-languages endpoint, over HTTP as the composer and Settings
// call them: the paths bind, the permissions hold and each change is
// audited.

const ws = "/workspaces/acme-newsletter"

type workspaceLanguages struct {
	Default, Fallback string
	CanManage         bool
	Helpers           []string
	TotalContacts     int64
	NoLanguage        int64
	NotOffered        int64
	Languages         []struct {
		Code, Name           string
		IsDefault, IsFallback bool
		Contacts             int64
		PageCopy             struct{ Status string }
	}
}

func (w workspaceLanguages) codes() []string {
	out := []string{}
	for _, l := range w.Languages {
		out = append(out, l.Code)
	}
	return out
}

type emailLanguages struct {
	Source          string
	SourceVersion   int
	CanChangeSource bool
	Blocks          int
	CanEdit         bool
	Versions        []emailLanguage
	SourceChange    *struct {
		Version  int
		Blocks   []string
		Outdated []string
	}
}

type emailLanguage struct {
	Language, Status, Gets string
	Offered, Held          bool
	Attention, Translated  int
	Version                int
	Subject                string
	ChangedBlocks          []string
}

func (e emailLanguages) version(t *testing.T, lang string) emailLanguage {
	t.Helper()
	for _, v := range e.Versions {
		if v.Language == lang {
			return v
		}
	}
	t.Fatalf("no %s in %+v", lang, e.Versions)
	return emailLanguage{}
}

type compare struct {
	Language, Status string
	Attention        int
	Version          int
	Rows             []struct {
		Key, State string
		Version    *struct{ Index int }
	}
}

// draft is a campaign whose MJML has a headline, an intro, a button and a
// footer; extra adds a section before the footer.
func draft(intro, extra string) string {
	return `<mjml><mj-head><mj-title>Hello</mj-title></mj-head><mj-body>
<mj-section><mj-column><mj-text><h1>Scheduled rollouts are here</h1></mj-text></mj-column></mj-section>
<mj-section><mj-column><mj-text>` + intro + `</mj-text></mj-column></mj-section>
<mj-section><mj-column><mj-button href="https://acme.io/start">Get started</mj-button></mj-column></mj-section>` + extra + `
<mj-section><mj-column><mj-text>Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105 · <a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section>
</mj-body></mjml>`
}

func audited(t *testing.T, env *apptest.Env, action string) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1`, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestLanguagesAPI(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh, daniel, aisha := env.Client(), env.Client(), env.Client()
	linh.SignInDemo("linh@acme.io")
	daniel.SignInDemo("daniel@acme.io")
	aisha.SignInDemo("aisha@acme.io")
	// The demo's own history may hold these actions already.
	before := map[string]int{}
	for _, a := range []string{"language.add", "language.fallback_change", "language.remove", "campaign.language.add", "campaign.language.mark_up_to_date",
		"campaign.language.source_change"} {
		before[a] = audited(t, env, a)
	}
	logged := func(action string) int { return audited(t, env, action) - before[action] }

	// --- Workspace languages (Settings › General) ---
	var list workspaceLanguages
	linh.Do("GET", ws+"/languages", nil).MustStatus(t, 200).JSON(t, &list)
	if list.Default != "en" || list.Fallback != "en" || !list.CanManage || list.TotalContacts == 0 || list.Languages[0].Code != "en" || !list.Languages[0].IsDefault {
		t.Fatalf("languages: %+v", list)
	}
	var readOnly workspaceLanguages
	daniel.Do("GET", ws+"/languages", nil).MustStatus(t, 200).JSON(t, &readOnly)
	if readOnly.CanManage || len(readOnly.Helpers) == 0 {
		t.Fatalf("Daniel (Content Writer) reads the languages without managing them: %+v", readOnly)
	}
	var catalog struct {
		Items []struct {
			Code     string
			RTL      bool
			Contacts int64
			Added    bool
		}
	}
	linh.Do("GET", ws+"/languages/catalog?q=fran", nil).MustStatus(t, 200).JSON(t, &catalog)
	if len(catalog.Items) == 0 || catalog.Items[0].Code != "fr" {
		t.Fatalf("catalog for “fran”: %+v", catalog.Items)
	}
	linh.Do("GET", ws+"/languages/catalog?q=ar", nil).MustStatus(t, 200).JSON(t, &catalog)
	if !slices.ContainsFunc(catalog.Items, func(i struct {
		Code     string
		RTL      bool
		Contacts int64
		Added    bool
	}) bool {
		return i.Code == "ar" && i.RTL
	}) {
		t.Fatalf("Arabic is listed as right to left: %+v", catalog.Items)
	}

	// Adding needs workspace:manage; right-to-left languages are refused.
	daniel.Do("POST", ws+"/languages", map[string]any{"code": "es"}).MustStatus(t, 403)
	p := linh.Do("POST", ws+"/languages", map[string]any{"code": "ar"}).MustStatus(t, 422).Problem(t)
	if !strings.Contains(fmt.Sprint(p), "Right-to-left languages aren't supported yet") {
		t.Fatalf("RTL refusal: %v", p)
	}
	linh.Do("POST", ws+"/languages", map[string]any{"code": "es-MX"}).MustStatus(t, 422)
	for _, code := range []string{"es", "de"} {
		linh.Do("POST", ws+"/languages", map[string]any{"code": code}).MustStatus(t, 200).JSON(t, &list)
	}
	if !slices.Equal(list.codes(), []string{"en", "es", "de"}) || logged("language.add") != 2 {
		t.Fatalf("added Spanish and German: %v, %d audited", list.codes(), logged("language.add"))
	}
	linh.Do("POST", ws+"/languages", map[string]any{"code": "es"}).MustStatus(t, 409)

	// The fallback: its impact first, then the change (audited) and back.
	var impact struct {
		Fallback                    string
		Moved, NoLanguage, NotOffered int64
	}
	linh.Do("GET", ws+"/languages/fallback-impact?code=es", nil).MustStatus(t, 200).JSON(t, &impact)
	if impact.Fallback != "es" || impact.Moved != impact.NoLanguage+impact.NotOffered {
		t.Fatalf("impact: %+v", impact)
	}
	daniel.Do("PUT", ws+"/languages/defaults", map[string]any{"default": "en", "fallback": "es"}).MustStatus(t, 403)
	linh.Do("PUT", ws+"/languages/defaults", map[string]any{"default": "en", "fallback": "es"}).MustStatus(t, 200).JSON(t, &list)
	if list.Fallback != "es" || logged("language.fallback_change") != 1 {
		t.Fatalf("fallback: %+v", list)
	}
	linh.Do("PUT", ws+"/languages/defaults", map[string]any{"default": "en", "fallback": "en"}).MustStatus(t, 200)

	// --- A campaign's language versions (the composer) ---
	var created struct{ ID string }
	linh.Do("POST", ws+"/campaigns", map[string]any{"name": "Languages over HTTP"}).MustStatus(t, 201).JSON(t, &created)
	camp := ws + "/campaigns/" + created.ID
	base := camp + "/languages"
	var saved struct{ Version int }
	linh.Do("PUT", camp+"/content", map[string]any{"baseVersion": 1, "subject": "Scheduled rollouts", "mjml": draft("Hi, you can now schedule flag changes.", "")}).
		MustStatus(t, 200).JSON(t, &saved)

	var langs emailLanguages
	aisha.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &langs)
	if langs.Source != "en" || !langs.CanChangeSource || langs.CanEdit || langs.Blocks != 4 || langs.version(t, "es").Status != "missing" {
		t.Fatalf("versions before adding one: %+v", langs)
	}
	aisha.Do("POST", base, map[string]any{"language": "es"}).MustStatus(t, 403)
	linh.Do("POST", base, map[string]any{"language": "fr"}).MustStatus(t, 422)
	daniel.Do("POST", base, map[string]any{"language": "es", "start": "copy"}).MustStatus(t, 201).JSON(t, &langs)
	es := langs.version(t, "es")
	if es.Status != "draft" || es.Gets != "en" || langs.CanChangeSource || logged("campaign.language.add") != 1 {
		t.Fatalf("Spanish added as a draft: %+v", es)
	}
	linh.Do("POST", base, map[string]any{"language": "es"}).MustStatus(t, 409)

	// The composer reads and saves the Spanish version through the
	// campaign's own endpoints.
	var spanish struct {
		Language, SourceLanguage, Subject, MJML string
		Version                                 int
	}
	linh.Do("GET", camp+"?language=es", nil).MustStatus(t, 200).JSON(t, &spanish)
	if spanish.Language != "es" || spanish.SourceLanguage != "en" || !strings.Contains(spanish.MJML, "schedule flag changes") {
		t.Fatalf("Spanish content: %+v", spanish)
	}
	translated := strings.NewReplacer("Scheduled rollouts are here", "Ya están aquí los despliegues programados", "Hi, you can now schedule flag changes.",
		"Hola, ahora puedes programar cambios de flags.", "Get started", "Empezar", "Unsubscribe", "Darse de baja").Replace(spanish.MJML)
	daniel.Do("PUT", camp+"/content?language=es", map[string]any{"baseVersion": spanish.Version, "subject": "Despliegues programados", "mjml": translated}).MustStatus(t, 200)
	var cmp compare
	linh.Do("GET", base+"/es/compare", nil).MustStatus(t, 200).JSON(t, &cmp)
	if cmp.Language != "es" || cmp.Status != "draft" || len(cmp.Rows) != 4 {
		t.Fatalf("compare a draft: %+v", cmp)
	}
	aisha.Do("POST", base+"/es/mark-translated", nil).MustStatus(t, 403)
	daniel.Do("POST", base+"/es/mark-translated", nil).MustStatus(t, 200).JSON(t, &langs)
	if es = langs.version(t, "es"); es.Status != "up_to_date" || es.Gets != "es" {
		t.Fatalf("marked translated: %+v", es)
	}
	var preview struct {
		Language, Subject, HTML string
	}
	linh.Do("POST", camp+"/preview", map[string]any{"language": "es"}).MustStatus(t, 200).JSON(t, &preview)
	if preview.Language != "es" || preview.Subject != "Despliegues programados" || !strings.Contains(preview.HTML, "Hola") {
		t.Fatalf("preview in Spanish: %s %s", preview.Language, preview.Subject)
	}

	// The source's intro changes: Spanish is outdated by one block.
	linh.Do("PUT", camp+"/content", map[string]any{"baseVersion": saved.Version, "subject": "Scheduled rollouts",
		"mjml": draft("Hi, you can now schedule flag changes ahead of time, in any timezone.", "")}).MustStatus(t, 200).JSON(t, &saved)
	linh.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &langs)
	if es = langs.version(t, "es"); es.Status != "outdated" || es.Attention != 1 || len(es.ChangedBlocks) != 1 {
		t.Fatalf("outdated per block: %+v", es)
	}
	linh.Do("GET", base+"/es/compare", nil).MustStatus(t, 200).JSON(t, &cmp)
	changed := ""
	for _, r := range cmp.Rows {
		if r.State == "changed" {
			changed = r.Key
		}
	}
	if cmp.Status != "outdated" || changed == "" {
		t.Fatalf("compare an outdated version: %+v", cmp)
	}
	daniel.Do("POST", base+"/es/mark-up-to-date", map[string]any{"blocks": []string{changed}}).MustStatus(t, 200).JSON(t, &langs)
	if es = langs.version(t, "es"); es.Status != "up_to_date" || logged("campaign.language.mark_up_to_date") != 1 {
		t.Fatalf("marked up to date: %+v", es)
	}

	// Hold: Spanish readers get English this time, then Spanish again.
	daniel.Do("PUT", base+"/es/hold", map[string]any{"held": true}).MustStatus(t, 200).JSON(t, &langs)
	if es = langs.version(t, "es"); !es.Held || es.Gets != "en" {
		t.Fatalf("held: %+v", es)
	}
	daniel.Do("PUT", base+"/es/hold", map[string]any{"held": false}).MustStatus(t, 200).JSON(t, &langs)
	if es = langs.version(t, "es"); es.Held || es.Gets != "es" {
		t.Fatalf("released: %+v", es)
	}

	// A typo in the source needs no translating: keep the versions up to date.
	linh.Do("PUT", camp+"/content", map[string]any{"baseVersion": saved.Version, "subject": "Scheduled rollouts",
		"mjml": draft("Hi, you can now schedule flag changes ahead of time, in any time zone.", "")}).MustStatus(t, 200).JSON(t, &saved)
	linh.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &langs)
	if es = langs.version(t, "es"); es.Status != "outdated" || langs.SourceChange == nil || langs.SourceChange.Version != saved.Version ||
		!slices.Equal(langs.SourceChange.Outdated, []string{"es"}) {
		t.Fatalf("after the typo: %+v %+v", es, langs.SourceChange)
	}
	daniel.Do("POST", base+"/keep-up-to-date", map[string]any{"blocks": langs.SourceChange.Blocks}).MustStatus(t, 200).JSON(t, &langs)
	if es = langs.version(t, "es"); es.Status != "up_to_date" {
		t.Fatalf("kept up to date: %+v", es)
	}

	// A link changed in the source, applied to every version.
	var applied struct {
		Languages []string
		Replaced  int
	}
	daniel.Do("POST", base+"/apply", map[string]any{"kind": "link", "from": "https://acme.io/start", "to": "https://acme.io/begin"}).MustStatus(t, 200).JSON(t, &applied)
	if !slices.Equal(applied.Languages, []string{"es"}) || applied.Replaced != 1 {
		t.Fatalf("applied: %+v", applied)
	}

	// A block new in the source is copied into Spanish; then the source
	// drops it and Spanish removes its copy.
	linh.Do("PUT", camp+"/content", map[string]any{"baseVersion": saved.Version, "subject": "Scheduled rollouts",
		"mjml": draft("Hi, you can now schedule flag changes ahead of time, in any time zone.", `<mj-section><mj-column><mj-text>The v1 API stops answering on 30 November.</mj-text></mj-column></mj-section>`)}).
		MustStatus(t, 200).JSON(t, &saved)
	linh.Do("GET", base+"/es/compare", nil).MustStatus(t, 200).JSON(t, &cmp)
	added := ""
	for _, r := range cmp.Rows {
		if r.State == "new" {
			added = r.Key
		}
	}
	if added == "" {
		t.Fatalf("a block new in the source: %+v", cmp.Rows)
	}
	daniel.Do("POST", base+"/es/copy-block", map[string]any{"key": added}).MustStatus(t, 200).JSON(t, &cmp)
	if len(cmp.Rows) != 5 || slices.ContainsFunc(cmp.Rows, func(r struct {
		Key, State string
		Version    *struct{ Index int }
	}) bool {
		return r.State == "new"
	}) {
		t.Fatalf("copied into Spanish: %+v", cmp.Rows)
	}
	linh.Do("PUT", camp+"/content", map[string]any{"baseVersion": saved.Version, "subject": "Scheduled rollouts",
		"mjml": draft("Hi, you can now schedule flag changes ahead of time, in any time zone.", "")}).MustStatus(t, 200).JSON(t, &saved)
	linh.Do("GET", base+"/es/compare", nil).MustStatus(t, 200).JSON(t, &cmp)
	removed := -1
	for _, r := range cmp.Rows {
		if r.State == "removed" && r.Version != nil {
			removed = r.Version.Index
		}
	}
	if removed < 0 {
		t.Fatalf("a block the source dropped: %+v", cmp.Rows)
	}
	daniel.Do("POST", base+"/es/remove-block", map[string]any{"index": removed}).MustStatus(t, 200).JSON(t, &cmp)
	if len(cmp.Rows) != 4 {
		t.Fatalf("removed from Spanish: %+v", cmp.Rows)
	}

	// The version history names the Spanish saves.
	var history struct {
		Items []struct{ Version int }
	}
	aisha.Do("GET", base+"/es/versions", nil).MustStatus(t, 200).JSON(t, &history)
	if len(history.Items) < 3 {
		t.Fatalf("Spanish history: %+v", history)
	}

	// The source language can't change while there are versions.
	linh.Do("PUT", base+"/source", map[string]any{"language": "de"}).MustStatus(t, 409)

	// --- Removing a language from the workspace ---
	var removal struct {
		Code, Fallback, Refusal string
		Readers                 int64
		Has                     []struct{ Kind, ID, State, Status string }
	}
	linh.Do("GET", ws+"/languages/en/removal", nil).MustStatus(t, 200).JSON(t, &removal)
	if removal.Refusal == "" {
		t.Fatalf("the default can't be removed: %+v", removal)
	}
	linh.Do("DELETE", ws+"/languages/en", nil).MustStatus(t, 409)
	removal.Refusal = ""
	linh.Do("GET", ws+"/languages/es/removal", nil).MustStatus(t, 200).JSON(t, &removal)
	if removal.Refusal != "" || removal.Fallback != "en" || len(removal.Has) != 1 || removal.Has[0].ID != created.ID || removal.Has[0].State != "draft" ||
		removal.Has[0].Status != "up_to_date" {
		t.Fatalf("removing Spanish affects the campaign: %+v", removal)
	}
	daniel.Do("DELETE", ws+"/languages/es", nil).MustStatus(t, 403)
	linh.Do("DELETE", ws+"/languages/es", nil).MustStatus(t, 200).JSON(t, &list)
	if slices.Contains(list.codes(), "es") || logged("language.remove") != 1 {
		t.Fatalf("Spanish removed: %v", list.codes())
	}
	linh.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &langs)
	if es = langs.version(t, "es"); es.Offered || es.Gets != "en" {
		t.Fatalf("a removed language's version is kept but not sent: %+v", es)
	}
	linh.Do("POST", ws+"/languages", map[string]any{"code": "es"}).MustStatus(t, 200)
	linh.Do("GET", base, nil).MustStatus(t, 200).JSON(t, &langs)
	if es = langs.version(t, "es"); !es.Offered || es.Gets != "es" {
		t.Fatalf("adding Spanish again brings its version back: %+v", es)
	}

	// Removing the campaign's Spanish version; then the source can change.
	aisha.Do("DELETE", base+"/es", nil).MustStatus(t, 403)
	daniel.Do("DELETE", base+"/es", nil).MustStatus(t, 200).JSON(t, &langs)
	if es = langs.version(t, "es"); es.Status != "missing" || !langs.CanChangeSource {
		t.Fatalf("Spanish version removed: %+v", langs)
	}
	linh.Do("PUT", base+"/source", map[string]any{"language": "de"}).MustStatus(t, 200).JSON(t, &langs)
	if langs.Source != "de" || logged("campaign.language.source_change") != 1 {
		t.Fatalf("written in German: %+v", langs)
	}
}
