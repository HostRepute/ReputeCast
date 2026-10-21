package pages_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/pages"
)

// Saving the draft: every save is a version; a version saved meanwhile by
// someone else is a conflict until "Keep mine"; the same person's other
// tab wins; an earlier version comes back as a new one.
func TestPageContentVersions(t *testing.T) {
	f := newFixture(t)
	sofia := f.as("sofia@acme.io")
	d := f.create(f.owner, pages.TemplateBlank, "Release notes", "")
	content := pagesAPI + "/" + d.ID + "/content"

	s := f.save(sofia, d.ID, 1, plain("Release notes", "Sofia wrote this line."))
	if s.Version != 2 || s.SavedBy == nil || s.SavedBy.Name != "Sofia Rossi" || s.Changes || len(s.Checks) == 0 || s.Counts.Blocks != 1 {
		t.Fatalf("Sofia's save: %+v", s)
	}

	// Linh's editor still has version 1: a conflict naming Sofia.
	mine := plain("Release notes", "Linh wrote this line.")
	p := mustProblem(t, f.owner.Do(http.MethodPut, content, map[string]any{"html": mine, "baseVersion": 1, "manual": true}), http.StatusConflict, "version_conflict")
	if p.Meta["version"] != float64(2) || p.Meta["savedBy"] != "Sofia Rossi" || !strings.Contains(p.Detail, "Sofia Rossi saved version 2") {
		t.Fatalf("conflict: %+v", p)
	}
	if got := f.detail(f.owner, d.ID); got.Draft.Version != 2 || !strings.Contains(got.Draft.HTML, "Sofia wrote") {
		t.Fatalf("the conflict saved something: v%d", got.Draft.Version)
	}
	// "Keep mine" puts Linh's version on top.
	var kept pages.PageSaved
	f.owner.Do(http.MethodPut, content, map[string]any{"html": mine, "baseVersion": 1, "manual": true, "force": true}).MustStatus(t, http.StatusOK).JSON(t, &kept)
	if kept.Version != 3 || kept.SavedBy == nil || kept.SavedBy.Name != "Linh Nguyen" {
		t.Fatalf("kept: %+v", kept)
	}
	// Linh's other tab, on version 2: her own newer save isn't a conflict.
	if s := f.save(f.owner, d.ID, 2, plain("Release notes", "From the other tab.")); s.Version != 4 {
		t.Fatalf("the other tab: %+v", s)
	}
	// The autosave.
	auto := plain("Release notes", "Autosaved line.")
	var as pages.PageSaved
	f.owner.Do(http.MethodPut, content, map[string]any{"html": auto, "baseVersion": 4}).MustStatus(t, http.StatusOK).JSON(t, &as)
	if as.Version != 5 {
		t.Fatalf("autosave: %+v", as)
	}
	if got := f.detail(f.owner, d.ID); !got.Draft.Autosaved || got.Draft.Version != 5 || got.Versions != 5 {
		t.Fatalf("after the autosave: %+v", got.Draft)
	}
	// The same blocks again make no version.
	f.owner.Do(http.MethodPut, content, map[string]any{"html": auto, "baseVersion": 5}).MustStatus(t, http.StatusOK).JSON(t, &as)
	if as.Version != 5 {
		t.Fatalf("an unchanged save made version %d", as.Version)
	}
	mustProblem(t, f.owner.Do(http.MethodPut, content, map[string]any{"html": auto, "baseVersion": 0}), http.StatusUnprocessableEntity, "validation_failed")

	var vs pages.PageVersionList
	f.owner.Do(http.MethodGet, pagesAPI+"/"+d.ID+"/versions", nil).MustStatus(t, http.StatusOK).JSON(t, &vs)
	var labels []string
	for _, v := range vs.Items {
		labels = append(labels, v.Label)
	}
	if strings.Join(labels, ",") != "autosave,saved,kept,saved,created" || !vs.Items[0].Current || vs.Items[0].Version != 5 {
		t.Fatalf("history %v: %+v", labels, vs.Items)
	}
	if vs.Items[3].SavedBy == nil || vs.Items[3].SavedBy.Name != "Sofia Rossi" {
		t.Errorf("version 2 is Sofia's: %+v", vs.Items[3])
	}

	var v2 pages.PageVersionDetail
	f.owner.Do(http.MethodGet, pagesAPI+"/"+d.ID+"/versions/2", nil).MustStatus(t, http.StatusOK).JSON(t, &v2)
	if !strings.Contains(v2.HTML, "Sofia wrote this line.") || v2.Slug != "release-notes" || v2.Settings.Visibility != pages.VisibilityPublic || v2.Current || v2.Live {
		t.Fatalf("version 2: %+v", v2.PageVersionItem)
	}
	mustProblem(t, f.owner.Do(http.MethodGet, pagesAPI+"/"+d.ID+"/versions/99", nil), http.StatusNotFound, "not_found")

	// Restore version 2: it's the draft again, as version 6.
	var restored pages.PageDetail
	f.owner.Do(http.MethodPost, pagesAPI+"/"+d.ID+"/versions/2/restore", nil).MustStatus(t, http.StatusOK).JSON(t, &restored)
	if restored.Draft.Version != 6 || !strings.Contains(restored.Draft.HTML, "Sofia wrote this line.") || restored.Draft.Autosaved {
		t.Fatalf("restored: %+v", restored.Draft)
	}
	f.owner.Do(http.MethodGet, pagesAPI+"/"+d.ID+"/versions", nil).MustStatus(t, http.StatusOK).JSON(t, &vs)
	if vs.Items[0].Label != "restored" || vs.Items[0].Version != 6 {
		t.Fatalf("history after restoring: %+v", vs.Items[0])
	}
	if n := f.audits("page.version_restored", d.ID); n != 1 {
		t.Errorf("%d restore audit entries", n)
	}
	if from := scan[string](f, `SELECT after->>'from' FROM audit_events WHERE action = 'page.version_restored' AND target_id = $1`, d.ID); from != "2" {
		t.Errorf("the audit entry names the version restored: %q", from)
	}
	mustProblem(t, f.owner.Do(http.MethodPost, pagesAPI+"/"+d.ID+"/versions/99/restore", nil), http.StatusNotFound, "not_found")
}

// Saved HTML keeps only what the blocks may hold.
func TestPageContentSanitized(t *testing.T) {
	f := newFixture(t)
	d := f.create(f.owner, pages.TemplateBlank, "Clean", "")
	f.save(f.owner, d.ID, 1, `<section data-pb="text"><h1 onclick="steal()">Hi</h1><script>alert(1)</script><p style="color:red">Text</p>`+
		`<iframe src="https://evil.example"></iframe><a href="javascript:alert(1)">x</a><form action="/x"><input name="a"></form></section>`+
		`<section data-pb="text"><p>Second</p></section>`)
	got := f.detail(f.owner, d.ID).Draft.HTML
	for _, bad := range []string{"<script", "onclick", "style=", "<iframe", "javascript:", "<form", "<input"} {
		if strings.Contains(got, bad) {
			t.Errorf("saved HTML keeps %s: %s", bad, got)
		}
	}
	for _, want := range []string{"<h1>Hi</h1>", "<p>Text</p>", `data-pb-id="b-text"`, `data-pb-id="b-text-2"`} {
		if !strings.Contains(got, want) {
			t.Errorf("saved HTML lacks %s: %s", want, got)
		}
	}
}

// A published page's draft changes stay off the site until they're
// published; the publish dialog lists them.
func TestDraftChangesOfAPublishedPage(t *testing.T) {
	f := newFixture(t)
	about := f.pageID("about")
	before := f.detail(f.owner, about)
	if before.Status != pages.StatusPublished || before.Live == nil || before.Changes || before.Live.Version != before.Draft.Version {
		t.Fatalf("the demo's About: %+v", before)
	}
	s := f.save(f.owner, about, before.Draft.Version, plain("About Acme", "A new line about the team."))
	if !s.Changes {
		t.Fatalf("saved changes of a published page: %+v", s)
	}
	after := f.detail(f.owner, about)
	if !after.Changes || after.Live.Version != before.Live.Version || after.Status != pages.StatusPublished {
		t.Fatalf("after saving: %+v", after.Live)
	}
	if res := f.resolve("about"); res.Page == nil || strings.Contains(res.Page.HTML, "A new line about the team.") || !strings.Contains(res.Page.HTML, "The Acme newsletter") {
		t.Fatalf("the live About changed before publishing")
	}

	// The Field Guide's draft (version 9) against what went live (8).
	var diff pages.PageDiff
	f.owner.Do(http.MethodGet, pagesAPI+"/"+f.pageID("field-guide")+"/diff", nil).MustStatus(t, http.StatusOK).JSON(t, &diff)
	if diff.First || diff.Since != 8 || diff.Version != 9 || diff.PublishedBy != "Sofia Rossi" || diff.PublishedAt == nil {
		t.Fatalf("diff: %+v", diff)
	}
	var changed []string
	for _, b := range diff.Blocks {
		changed = append(changed, b.ID+":"+b.Kind)
	}
	for _, want := range []string{"b-hero:changed", "b-quotes:changed", "b-faq:changed"} {
		if !slices.Contains(changed, want) {
			t.Errorf("diff blocks %v lack %s", changed, want)
		}
	}
	var settings []string
	for _, s := range diff.Settings {
		settings = append(settings, s.Key)
	}
	if !slices.Equal(settings, []string{"seoDescription"}) {
		t.Errorf("diff settings %v", settings)
	}
	// A page never published has nothing to compare.
	f.owner.Do(http.MethodGet, pagesAPI+"/"+f.pageID("sponsor")+"/diff", nil).MustStatus(t, http.StatusOK).JSON(t, &diff)
	if !diff.First {
		t.Errorf("the sponsor draft's diff: %+v", diff)
	}
}
