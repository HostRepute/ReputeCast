package pages_test

import (
	"context"
	"net/http"
	"slices"
	"testing"

	"github.com/hostrepute/reputecast/internal/pages"
)

// Who may do what: website:view reads, website:edit writes drafts,
// website:publish puts pages on the site and takes them off,
// website:manage decides visibility and navigation; the shop's switch is
// products:publish; the reminder automation automations:edit.
func TestPagePermissions(t *testing.T) {
	f := newFixture(t)
	aisha := f.as("aisha@acme.io")   // Viewer
	daniel := f.as("daniel@acme.io") // Content Writer: website:edit, products:edit
	sofia := f.as("sofia@acme.io")   // Editor
	about := f.pageID("about")
	dmarc := f.pageID("dmarc-live")
	publishers := []string{"Sofia Rossi", "Marcus Chen", "Linh Nguyen"}
	managers := []string{"Marcus Chen", "Linh Nguyen"}

	t.Run("a Viewer reads", func(t *testing.T) {
		var ov pages.PagesOverview
		aisha.Do(http.MethodGet, pagesAPI, nil).MustStatus(t, http.StatusOK).JSON(t, &ov)
		if ov.CanEdit || ov.CanPublish || ov.Shop.CanToggle || !slices.Equal(ov.Editors, managers) || !slices.Equal(ov.Publishers, publishers) {
			t.Errorf("a Viewer's list: canEdit %v canPublish %v shop %v editors %v publishers %v", ov.CanEdit, ov.CanPublish, ov.Shop.CanToggle, ov.Editors, ov.Publishers)
		}
		d := f.detail(aisha, about)
		if d.Publish.CanEdit || d.Publish.CanPublish || d.Publish.CanManage || !slices.Equal(d.Publish.Publishers, publishers) {
			t.Errorf("a Viewer's page: %+v", d.Publish)
		}
		for _, p := range []string{"/versions", "/versions/1", "/diff", "/reminder-automation"} {
			aisha.Do(http.MethodGet, pagesAPI+"/"+about+p, nil).MustStatus(t, http.StatusOK)
		}
		for _, c := range []struct {
			method, path string
			body         any
			perm         string
		}{
			{http.MethodPost, pagesAPI, map[string]any{"template": "blank", "name": "Viewer's page"}, "website:edit"},
			{http.MethodPut, pagesAPI + "/" + about + "/content", map[string]any{"html": plain("A", "B"), "baseVersion": 2}, "website:edit"},
			{http.MethodPut, pagesAPI + "/" + about + "/settings", settingsOf(d), "website:edit"},
			{http.MethodPost, pagesAPI + "/" + about + "/versions/1/restore", nil, "website:edit"},
			{http.MethodDelete, pagesAPI + "/" + about, nil, "website:edit"},
			{http.MethodPost, pagesAPI + "/" + about + "/publish", map[string]any{"when": "now"}, "website:publish"},
			{http.MethodPost, pagesAPI + "/" + about + "/unpublish", map[string]any{}, "website:publish"},
			{http.MethodPost, pagesAPI + "/" + about + "/undo-publish", nil, "website:publish"},
			{http.MethodPut, pagesAPI + "/shop", map[string]any{"on": false}, "products:publish"},
			{http.MethodPost, pagesAPI + "/" + dmarc + "/reminder-automation", nil, "automations:edit"},
		} {
			denied(t, aisha.Do(c.method, c.path, c.body), c.perm)
		}
		if got := f.detail(f.owner, about); got.Status != pages.StatusPublished || got.Draft.Version != 2 {
			t.Fatalf("a Viewer changed About: %s v%d", got.Status, got.Draft.Version)
		}
	})

	t.Run("a Content Writer edits but doesn't publish or manage", func(t *testing.T) {
		d := f.create(daniel, pages.TemplateBlank, "Sponsor rates", "")
		f.save(daniel, d.ID, 1, plain("Sponsor rates", "One slot per issue."))
		d = f.detail(daniel, d.ID)
		if !d.Publish.CanEdit || d.Publish.CanPublish || d.Publish.CanManage || !slices.Equal(d.Publish.Publishers, publishers) {
			t.Errorf("Daniel's page: %+v", d.Publish)
		}
		// The title, address and SEO are his to change.
		in := settingsOf(d)
		in.SEOTitle, in.SEODescription, in.Slug, in.NavLabel = "Sponsor rates · Acme", "One slot per issue.", "rates", "Rates"
		f.settings(daniel, d.ID, in).MustStatus(t, http.StatusOK)
		// Visibility and navigation aren't.
		vis := in
		vis.Visibility = pages.VisibilityMembers
		if p := denied(t, f.settings(daniel, d.ID, vis), "website:manage"); !slices.Equal(p.Helpers, managers) {
			t.Errorf("who manages: %v", p.Helpers)
		}
		nav := in
		nav.ShowInNav = true
		denied(t, f.settings(daniel, d.ID, nav), "website:manage")
		got := f.detail(f.owner, d.ID)
		if got.Slug != "rates" || got.Settings.SEOTitle != "Sponsor rates · Acme" || got.Settings.Visibility != pages.VisibilityPublic || got.Settings.ShowInNav {
			t.Errorf("after Daniel's settings: /%s %+v", got.Slug, got.Settings)
		}
		denied(t, daniel.Do(http.MethodPost, pagesAPI+"/"+d.ID+"/publish", map[string]any{"when": "now"}), "website:publish")
		denied(t, daniel.Do(http.MethodPost, pagesAPI+"/"+about+"/unpublish", map[string]any{}), "website:publish")
		denied(t, daniel.Do(http.MethodPost, pagesAPI+"/"+about+"/undo-publish", nil), "website:publish")
		denied(t, daniel.Do(http.MethodPut, pagesAPI+"/shop", map[string]any{"on": false}), "products:publish")
		denied(t, daniel.Do(http.MethodPost, pagesAPI+"/"+dmarc+"/reminder-automation", nil), "automations:edit")
		// Refused changes leave no audit entries.
		if n := f.audits("page.updated", d.ID); n != 1 {
			t.Errorf("%d page.updated entries, want Daniel's one", n)
		}
	})

	t.Run("an Editor publishes and manages", func(t *testing.T) {
		sponsor := f.pageID("sponsor")
		in := settingsOf(f.detail(sofia, sponsor))
		in.Visibility, in.ShowInNav, in.NavLabel = pages.VisibilityMembers, true, "Sponsors"
		var d pages.PageDetail
		f.settings(sofia, sponsor, in).MustStatus(t, http.StatusOK).JSON(t, &d)
		if d.Settings.Visibility != pages.VisibilityMembers || !d.Settings.ShowInNav || d.Settings.NavLabel != "Sponsors" {
			t.Fatalf("Sofia's settings: %+v", d.Settings)
		}
		if diff := scan[string](f, `SELECT after->>'visibility' FROM audit_events WHERE action = 'page.updated' AND target_id = $1`, sponsor); diff != pages.VisibilityMembers {
			t.Errorf("the audit entry has the change: %q", diff)
		}
		d = f.publishNow(sofia, sponsor)
		if d.Live == nil || d.Live.PublishedBy == nil || d.Live.PublishedBy.Name != "Sofia Rossi" {
			t.Fatalf("Sofia's publish: %+v", d.Live)
		}
		if st, _ := f.env.App.Website().PageLink(context.Background(), f.env.App.Q, &f.ws, "sponsor"); !st.Linked {
			t.Error("Show in navigation didn't add the header link")
		}
		sofia.Do(http.MethodPut, pagesAPI+"/shop", map[string]any{"on": false}).MustStatus(t, http.StatusOK)
	})
}

// The shop's switch: off takes /products off the site; each change is audited once.
func TestShopSwitch(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	svc := f.env.App.Pages()
	if on, err := svc.ShopOn(ctx, &f.ws); err != nil || !on {
		t.Fatalf("the demo's shop is on: %v %v", on, err)
	}
	var shop pages.PagesShop
	f.owner.Do(http.MethodPut, pagesAPI+"/shop", map[string]any{"on": false}).MustStatus(t, http.StatusOK).JSON(t, &shop)
	if shop.On || shop.Products != 2 || !shop.CanToggle || shop.Path != "/products" {
		t.Fatalf("off: %+v", shop)
	}
	if on, _ := svc.ShopOn(ctx, &f.ws); on {
		t.Fatal("still on")
	}
	f.owner.Do(http.MethodPut, pagesAPI+"/shop", map[string]any{"on": false}).MustStatus(t, http.StatusOK)
	if n := f.audits("page.shop_updated", "shop"); n != 1 {
		t.Fatalf("%d audit entries for one change", n)
	}
	if before := scan[string](f, `SELECT before->>'on' FROM audit_events WHERE action = 'page.shop_updated'`); before != "true" {
		t.Errorf("audited before %q", before)
	}
	f.owner.Do(http.MethodPut, pagesAPI+"/shop", map[string]any{"on": true}).MustStatus(t, http.StatusOK).JSON(t, &shop)
	if !shop.On || f.audits("page.shop_updated", "shop") != 2 {
		t.Fatalf("on again: %+v", shop)
	}
	var ov pages.PagesOverview
	f.owner.Do(http.MethodGet, pagesAPI, nil).MustStatus(t, http.StatusOK).JSON(t, &ov)
	if !ov.Shop.On {
		t.Error("the list's shop row")
	}
}
