package pages_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/pages"
)

// Draft → published → changes published → unpublished (410, or sent to
// another path) → published again → deleted (410). Every step is audited.
func TestPageLifecycle(t *testing.T) {
	f := newFixture(t)
	d := f.create(f.owner, pages.TemplateBlank, "Pricing FAQ", "")
	path := pagesAPI + "/" + d.ID
	if res := f.resolve("pricing-faq"); res.Page != nil || res.Gone || res.MovedTo != "" || res.RedirectTo != "" {
		t.Fatalf("a draft answers at its address: %+v", res)
	}

	pub := f.publishNow(f.owner, d.ID)
	if pub.Status != pages.StatusPublished || pub.Live == nil || pub.Live.Version != 1 || pub.Live.PublishedAt == nil ||
		pub.Live.PublishedBy == nil || pub.Live.PublishedBy.Name != "Linh Nguyen" || pub.Changes || pub.Live.Path != "/pricing-faq" {
		t.Fatalf("published: %+v %+v", pub.Status, pub.Live)
	}
	if n := f.audits("page.published", d.ID); n != 1 {
		t.Fatalf("%d page.published entries", n)
	}
	if p := scan[string](f, `SELECT after->>'path' FROM audit_events WHERE action = 'page.published' AND target_id = $1`, d.ID); p != "/pricing-faq" {
		t.Errorf("audited path %q", p)
	}
	if res := f.resolve("pricing-faq"); res.Page == nil || res.Page.Version != 1 || !strings.Contains(res.Page.HTML, "Pricing FAQ") {
		t.Fatalf("the live page: %+v", res)
	}

	// Changes go live when they're published.
	f.save(f.owner, d.ID, 1, plain("Pricing FAQ", "The second version."))
	if res := f.resolve("pricing-faq"); strings.Contains(res.Page.HTML, "The second version.") {
		t.Fatal("saved changes are live before publishing")
	}
	pub = f.publishNow(f.owner, d.ID)
	if pub.Live.Version != 2 || pub.Changes {
		t.Fatalf("changes published: %+v", pub.Live)
	}
	if n := f.audits("page.changes_published", d.ID); n != 1 {
		t.Errorf("%d page.changes_published entries", n)
	}
	if res := f.resolve("pricing-faq"); !strings.Contains(res.Page.HTML, "The second version.") {
		t.Fatal("the published changes aren't live")
	}

	// Unpublished: the address is gone (410).
	var un pages.PageDetail
	f.owner.Do(http.MethodPost, path+"/unpublish", map[string]any{"reason": "Prices changed"}).MustStatus(t, http.StatusOK).JSON(t, &un)
	if un.Status != pages.StatusUnpublished || un.UnpublishedAt == nil || un.UnpublishReason != "Prices changed" || un.RedirectTo != "" {
		t.Fatalf("unpublished: %+v", un)
	}
	if res := f.resolve("pricing-faq"); !res.Gone || res.Page != nil {
		t.Fatalf("an unpublished page's address: %+v", res)
	}
	if n := f.audits("page.unpublished", d.ID); n != 1 {
		t.Errorf("%d page.unpublished entries", n)
	}
	mustProblem(t, f.owner.Do(http.MethodPost, path+"/unpublish", map[string]any{}), http.StatusConflict, "conflict")

	// Published again: back, and the unpublishing forgotten.
	pub = f.publishNow(f.owner, d.ID)
	if pub.Status != pages.StatusPublished || pub.UnpublishedAt != nil || pub.UnpublishReason != "" {
		t.Fatalf("republished: %+v", pub)
	}
	if res := f.resolve("pricing-faq"); res.Page == nil {
		t.Fatal("the republished page doesn't answer")
	}

	// Unpublished to another path of the site.
	for _, bad := range []string{"//evil.example/x", "https://evil.example/x", "products"} {
		mustProblem(t, f.owner.Do(http.MethodPost, path+"/unpublish", map[string]any{"redirectTo": bad}), http.StatusUnprocessableEntity, "validation_failed")
	}
	f.owner.Do(http.MethodPost, path+"/unpublish", map[string]any{"redirectTo": "/products"}).MustStatus(t, http.StatusOK).JSON(t, &un)
	if un.RedirectTo != "/products" {
		t.Fatalf("redirect: %+v", un)
	}
	if res := f.resolve("pricing-faq"); res.RedirectTo != "/products" || res.Gone {
		t.Fatalf("an unpublished page sending readers on: %+v", res)
	}

	// Deleted after being live: 410 at its address.
	f.publishNow(f.owner, d.ID)
	f.owner.Do(http.MethodDelete, path, nil).MustStatus(t, http.StatusNoContent)
	mustProblem(t, f.owner.Do(http.MethodGet, path, nil), http.StatusNotFound, "not_found")
	if res := f.resolve("pricing-faq"); !res.Gone {
		t.Fatalf("a deleted page's address: %+v", res)
	}
	if n := f.audits("page.deleted", d.ID); n != 1 {
		t.Errorf("%d page.deleted entries", n)
	}
	var ov pages.PagesOverview
	f.owner.Do(http.MethodGet, pagesAPI, nil).MustStatus(t, http.StatusOK).JSON(t, &ov)
	for _, p := range ov.Pages {
		if p.ID == d.ID {
			t.Fatal("the list keeps a deleted page")
		}
	}
	// A draft deleted never answered anywhere.
	nd := f.create(f.owner, pages.TemplateBlank, "Never live", "")
	f.owner.Do(http.MethodDelete, pagesAPI+"/"+nd.ID, nil).MustStatus(t, http.StatusNoContent)
	if res := f.resolve("never-live"); res.Gone || res.Page != nil {
		t.Fatalf("a deleted draft's address: %+v", res)
	}
}

// Publish at a time: the time must be ahead and the zone real; the minute
// job (PublishDue) puts the page live when its time comes.
func TestScheduledPublish(t *testing.T) {
	f := newFixture(t)
	svc := f.env.App.Pages()
	ctx := context.Background()
	t0 := time.Now().Truncate(time.Second)
	f.clock(t0)
	d := f.create(f.owner, pages.TemplateBlank, "Black Friday", "")
	publish := func(id string, body map[string]any) *apptest.Response {
		return f.owner.Do(http.MethodPost, pagesAPI+"/"+id+"/publish", body)
	}
	p := mustProblem(t, publish(d.ID, map[string]any{"when": "at", "at": t0.Add(-time.Minute)}), http.StatusUnprocessableEntity, "validation_failed")
	if len(p.Errors) != 1 || p.Errors[0].Location != "body.at" {
		t.Errorf("a time in the past: %+v", p)
	}
	mustProblem(t, publish(d.ID, map[string]any{"when": "at"}), http.StatusUnprocessableEntity, "validation_failed")
	p = mustProblem(t, publish(d.ID, map[string]any{"when": "at", "at": t0.Add(time.Hour), "tz": "Mars/Olympus_Mons"}), http.StatusUnprocessableEntity, "validation_failed")
	if len(p.Errors) != 1 || p.Errors[0].Location != "body.tz" {
		t.Errorf("an unknown zone: %+v", p)
	}

	at := t0.Add(time.Hour)
	var s pages.PageDetail
	publish(d.ID, map[string]any{"when": "at", "at": at, "tz": "Europe/Berlin"}).MustStatus(t, http.StatusOK).JSON(t, &s)
	if s.Status != pages.StatusScheduled || s.PublishAt == nil || !s.PublishAt.Equal(at) || s.PublishTZ != "Europe/Berlin" || s.Live != nil {
		t.Fatalf("scheduled: %+v", s)
	}
	if n := f.audits("page.scheduled", d.ID); n != 1 {
		t.Errorf("%d page.scheduled entries", n)
	}
	if res := f.resolve("black-friday"); res.Page != nil {
		t.Fatal("a scheduled page is live before its time")
	}

	f.clock(at.Add(-time.Minute))
	if n, err := svc.PublishDue(ctx); err != nil || n != 0 {
		t.Fatalf("a minute early: %d, %v", n, err)
	}
	f.clock(at.Add(time.Minute))
	if n, err := svc.PublishDue(ctx); err != nil || n != 1 {
		t.Fatalf("at its time: %d, %v", n, err)
	}
	s = f.detail(f.owner, d.ID)
	if s.Status != pages.StatusPublished || s.Live == nil || s.Live.PublishedAt == nil || !s.Live.PublishedAt.Equal(at.Add(time.Minute)) || s.PublishAt != nil || s.Live.PublishedBy != nil {
		t.Fatalf("after its time: %+v %+v", s.Status, s.Live)
	}
	if res := f.resolve("black-friday"); res.Page == nil {
		t.Fatal("not live after its time")
	}
	var actorType, actorName string
	if err := f.env.App.Pool.QueryRow(ctx, `SELECT actor_type, actor_name FROM audit_events WHERE action = 'page.published' AND target_id = $1`, d.ID).Scan(&actorType, &actorName); err != nil {
		t.Fatal(err)
	}
	if actorType != "system" || actorName != "Schedule" {
		t.Errorf("the scheduled publish is the system's: %s %q", actorType, actorName)
	}
	if n, _ := svc.PublishDue(ctx); n != 0 {
		t.Errorf("published twice: %d", n)
	}

	// Without a zone, the workspace's; unscheduling a draft keeps it a draft.
	c := f.create(f.owner, pages.TemplateBlank, "Cyber Monday", "")
	publish(c.ID, map[string]any{"when": "at", "at": at.Add(time.Hour)}).MustStatus(t, http.StatusOK).JSON(t, &s)
	if s.PublishTZ != "America/New_York" {
		t.Errorf("default zone %q", s.PublishTZ)
	}
	var un pages.PageDetail // a fresh one: absent fields don't reset a decoded struct
	f.owner.Do(http.MethodPost, pagesAPI+"/"+c.ID+"/unpublish", map[string]any{}).MustStatus(t, http.StatusOK).JSON(t, &un)
	if un.Status != pages.StatusDraft || un.PublishAt != nil || un.PublishTZ != "" {
		t.Fatalf("unscheduled: %s at %v %q", un.Status, un.PublishAt, un.PublishTZ)
	}
	if n := f.audits("page.unscheduled", c.ID); n != 1 {
		t.Errorf("%d page.unscheduled entries", n)
	}
	f.clock(at.Add(3 * time.Hour))
	if n, _ := svc.PublishDue(ctx); n != 0 {
		t.Errorf("an unscheduled page went live: %d", n)
	}
}

// Undo within seconds of publishing: the version live before comes back,
// or a first publish goes back to its draft. Later, it's too late.
func TestUndoPublish(t *testing.T) {
	f := newFixture(t)
	t0 := time.Now().Truncate(time.Second)
	undo := func(id string) *apptest.Response {
		return f.owner.Do(http.MethodPost, pagesAPI+"/"+id+"/undo-publish", nil)
	}

	// The Field Guide: version 8 is live, 9 the draft.
	fg := f.pageID("field-guide")
	v8 := scan[time.Time](f, `SELECT published_at FROM page_versions WHERE page_id = $1 AND version = 8`, fg)
	f.clock(t0)
	if pub := f.publishNow(f.owner, fg); pub.Live.Version != 9 {
		t.Fatalf("published the draft: %+v", pub.Live)
	}
	if res := f.resolve("field-guide"); !strings.Contains(res.Page.HTML, "Eight practical chapters") {
		t.Fatal("version 9 isn't live")
	}
	f.clock(t0.Add(5 * time.Second))
	var d pages.PageDetail
	undo(fg).MustStatus(t, http.StatusOK).JSON(t, &d)
	if d.Status != pages.StatusPublished || d.Live == nil || d.Live.Version != 8 || !d.Changes || d.Live.PublishedAt == nil || !d.Live.PublishedAt.Equal(v8) {
		t.Fatalf("undone: %+v %+v", d.Status, d.Live)
	}
	if res := f.resolve("field-guide"); res.Page.Version != 8 || strings.Contains(res.Page.HTML, "Eight practical chapters") {
		t.Fatal("version 8 isn't back")
	}
	if n := f.audits("page.publish_undone", fg); n != 1 {
		t.Errorf("%d page.publish_undone entries", n)
	}

	// A first publish undone: a draft again, off the site.
	n := f.create(f.owner, pages.TemplateBlank, "Undo me", "")
	f.clock(t0.Add(time.Minute))
	f.publishNow(f.owner, n.ID)
	f.clock(t0.Add(time.Minute + 3*time.Second))
	var first pages.PageDetail // a fresh one: "live" is left out when there's none
	undo(n.ID).MustStatus(t, http.StatusOK).JSON(t, &first)
	if first.Status != pages.StatusDraft || first.Live != nil {
		t.Fatalf("a first publish undone: %+v %+v", first.Status, first.Live)
	}
	if res := f.resolve("undo-me"); res.Page != nil || res.Gone {
		t.Fatalf("its address after the undo: %+v", res)
	}

	// Too late, and never published.
	f.clock(t0.Add(2 * time.Minute))
	f.publishNow(f.owner, n.ID)
	f.clock(t0.Add(2*time.Minute + 20*time.Second))
	p := mustProblem(t, undo(n.ID), http.StatusConflict, "conflict")
	if !strings.Contains(p.Detail, "too late") {
		t.Errorf("late undo: %s", p.Detail)
	}
	mustProblem(t, undo(f.pageID("sponsor")), http.StatusConflict, "conflict")
}

// Publishing a page that has no changes, then Undo, leaves the site as it
// was (or the publish is refused).
func TestUndoOfAPublishWithoutChanges(t *testing.T) {
	f := newFixture(t)
	about := f.pageID("about")
	t0 := time.Now().Truncate(time.Second)
	f.clock(t0)
	r := f.owner.Do(http.MethodPost, pagesAPI+"/"+about+"/publish", map[string]any{"when": "now"})
	if r.Status == http.StatusConflict {
		return // refusing a publish without changes is fine too
	}
	r.MustStatus(t, http.StatusOK)
	f.clock(t0.Add(3 * time.Second))
	var d pages.PageDetail
	f.owner.Do(http.MethodPost, pagesAPI+"/"+about+"/undo-publish", nil).MustStatus(t, http.StatusOK).JSON(t, &d)
	if d.Status != pages.StatusPublished || d.Live == nil || d.Live.Version != 2 {
		t.Fatalf("undoing a publish that changed nothing took About off the site: %s, live %+v", d.Status, d.Live)
	}
	if res := f.resolve("about"); res.Page == nil {
		t.Fatal("About doesn't answer after the undo")
	}
}

// Deleting: a draft takes website:edit, a page on the site website:publish;
// a deleted page leaves the header, its address answers 410 and is free for
// another page.
func TestDeletePage(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	daniel := f.as("daniel@acme.io")
	draft := f.create(daniel, pages.TemplateBlank, "Scratch", "")
	daniel.Do(http.MethodDelete, pagesAPI+"/"+draft.ID, nil).MustStatus(t, http.StatusNoContent)

	about := f.pageID("about")
	if p := denied(t, daniel.Do(http.MethodDelete, pagesAPI+"/"+about, nil), "website:publish"); len(p.Helpers) == 0 {
		t.Errorf("no one named to ask: %+v", p)
	}
	site := f.env.App.Website()
	if st, err := site.PageLink(ctx, f.env.App.Q, &f.ws, "about"); err != nil || !st.Linked {
		t.Fatalf("the demo's header links About: %+v %v", st, err)
	}
	f.owner.Do(http.MethodDelete, pagesAPI+"/"+about, nil).MustStatus(t, http.StatusNoContent)
	if res := f.resolve("about"); !res.Gone {
		t.Fatalf("deleted About: %+v", res)
	}
	if st, _ := site.PageLink(ctx, f.env.App.Q, &f.ws, "about"); st.Linked {
		t.Error("the header still links the deleted page")
	}
	if before := scan[string](f, `SELECT before->>'status' FROM audit_events WHERE action = 'page.deleted' AND target_id = $1`, about); before != pages.StatusPublished {
		t.Errorf("audited status %q", before)
	}
	// Another page may take the address; it answers there once published.
	n := f.create(f.owner, pages.TemplateBlank, "About us", "about")
	if res := f.resolve("about"); !res.Gone {
		t.Fatalf("before the new page is published: %+v", res)
	}
	f.publishNow(f.owner, n.ID)
	if res := f.resolve("about"); res.Page == nil || res.Page.ID.String() != n.ID {
		t.Fatalf("the new page at the address: %+v", res)
	}
}

// A deleted page's earlier addresses are free again, like its own.
func TestDeletedPageFreesEarlierAddresses(t *testing.T) {
	f := newFixture(t)
	d := f.create(f.owner, pages.TemplateBlank, "Old offer", "")
	f.publishNow(f.owner, d.ID)
	in := settingsOf(d)
	in.Slug = "new-offer"
	f.settings(f.owner, d.ID, in).MustStatus(t, http.StatusOK)
	f.publishNow(f.owner, d.ID)
	f.owner.Do(http.MethodDelete, pagesAPI+"/"+d.ID, nil).MustStatus(t, http.StatusNoContent)
	if res := f.resolve("old-offer"); res.MovedTo != "" || res.Page != nil {
		t.Fatalf("the old address of a deleted page: %+v", res)
	}
	f.create(f.owner, pages.TemplateBlank, "Offer", "new-offer")
	r := f.owner.Do(http.MethodPost, pagesAPI, map[string]any{"template": "blank", "name": "Offer", "slug": "old-offer"})
	if r.Status != http.StatusCreated {
		t.Fatalf("the deleted page's earlier address stays taken: %d %s", r.Status, r.Body)
	}
}

// A published page whose changes are scheduled is still on the site:
// deleting it takes website:publish like any live page.
func TestScheduledChangesOfALivePage(t *testing.T) {
	f := newFixture(t)
	fg := f.pageID("field-guide")
	at := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	var d pages.PageDetail
	f.owner.Do(http.MethodPost, pagesAPI+"/"+fg+"/publish", map[string]any{"when": "at", "at": at}).MustStatus(t, http.StatusOK).JSON(t, &d)
	if d.Status != pages.StatusScheduled || d.Live == nil || d.Live.Version != 8 {
		t.Fatalf("scheduled changes: %s %+v", d.Status, d.Live)
	}
	if res := f.resolve("field-guide"); res.Page == nil || res.Page.Version != 8 {
		t.Fatalf("the live version while changes wait: %+v", res)
	}
	daniel := f.as("daniel@acme.io")
	r := daniel.Do(http.MethodDelete, pagesAPI+"/"+fg, nil)
	if r.Status != http.StatusForbidden {
		t.Fatalf("a Content Writer deleted a live page with scheduled changes: %d", r.Status)
	}
}

// A page's numbers count views by the workspace's days, like the
// website's: a view at 05:00 in Tokyo counts today in Tokyo.
func TestPageViewsByWorkspaceDay(t *testing.T) {
	f := newFixture(t)
	f.exec(`UPDATE workspaces SET timezone = 'Asia/Tokyo' WHERE id = $1`, f.ws.ID)
	// 20:00 UTC: already the next day in Tokyo.
	u := time.Now().UTC()
	at := time.Date(u.Year(), u.Month(), u.Day(), 20, 0, 0, 0, time.UTC)
	f.clock(at)
	site := f.env.App.Website()
	site.Now = func() time.Time { return at }
	t.Cleanup(func() { site.Now = nil })
	about := f.pageID("about")
	before := f.detail(f.owner, about).Numbers.Views
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, f.env.Server.URL+"/p/acme-newsletter/about", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /about: %d", res.StatusCode)
	}
	tokyo, _ := time.LoadLocation("Asia/Tokyo")
	l := at.In(tokyo)
	if n := scan[int64](f, `SELECT coalesce(sum(views), 0) FROM website_view_days WHERE page_id = $1 AND page_kind = 'page' AND day = $2`, about,
		time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)); n < 1 {
		t.Fatalf("the view isn't on Tokyo's day: %d", n)
	}
	if after := f.detail(f.owner, about).Numbers.Views; after != before+1 {
		t.Fatalf("views %d → %d: today's view in Tokyo isn't counted", before, after)
	}
}
