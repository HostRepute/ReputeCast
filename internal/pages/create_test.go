package pages_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/pages"
)

// New page: each template starts a draft at an address taken from its
// name, with the blocks of the template filled in for the workspace.
func TestCreatePageFromTemplates(t *testing.T) {
	f := newFixture(t)
	now := time.Now().Truncate(time.Second)
	f.clock(now)
	ny, _ := time.LoadLocation("America/New_York")
	inTwoWeeks := now.In(ny).AddDate(0, 0, 14).Format("2006-01-02")
	firstForm := scan[uuid.UUID](f, `SELECT id FROM forms WHERE workspace_id = $1 AND status = 'live' ORDER BY created_at LIMIT 1`, f.ws.ID).String()

	cases := []struct {
		template, name, slug string
		has                  []string
		blocks               []string
		products             int // empty product blocks
	}{
		{pages.TemplateAbout, "About the Digest", "about-the-digest", []string{`<h1>About the Digest</h1>`, `<rc-form`, `form="` + firstForm + `"`}, nil, 0},
		{pages.TemplateLeadMagnet, "Free SPF primer", "free-spf-primer", []string{`<h1>Free SPF primer</h1>`, `form="` + firstForm + `"`}, nil, 0},
		{pages.TemplateLaunch, "Warm-up planner launch", "warm-up-planner-launch",
			[]string{`<rc-countdown`, `at="` + inTwoWeeks + `T09:00"`, `tz="America/New_York"`, `end="reveal"`, `reveal="buy"`, `hide="waitlist"`}, []string{pages.CheckProduct}, 1},
		{pages.TemplateWebinar, "Live: BIMI in practice", "live-bimi-in-practice",
			[]string{`<rc-event`, `date="` + inTwoWeeks + `"`, `start="17:00"`, `form="` + firstForm + `"`}, nil, 0},
		{pages.TemplateSales, "Bounce codes, explained", "bounce-codes-explained", []string{`<rc-product`, `layout="final"`}, []string{pages.CheckProduct}, 2},
		{pages.TemplateBlank, "Sponsor kit 2027", "sponsor-kit-2027", []string{`<h1>Sponsor kit 2027</h1>`}, nil, 0},
	}
	for _, c := range cases {
		t.Run(c.template, func(t *testing.T) {
			d := f.create(f.owner, c.template, c.name, "")
			if d.Slug != c.slug || d.Path != "/"+c.slug || !strings.HasSuffix(d.URL, "/"+c.slug) || d.Name != c.name || d.Template != c.template {
				t.Fatalf("address: slug %q path %q url %q name %q template %q", d.Slug, d.Path, d.URL, d.Name, d.Template)
			}
			if d.Status != pages.StatusDraft || d.Live != nil || d.Draft.Version != 1 || d.Versions != 1 || d.Changes {
				t.Fatalf("a new page is a draft at version 1: %+v", d)
			}
			if d.Draft.SavedBy == nil || d.Draft.SavedBy.Name != "Linh Nguyen" {
				t.Errorf("saved by %+v", d.Draft.SavedBy)
			}
			for _, h := range c.has {
				if !strings.Contains(d.Draft.HTML, h) {
					t.Errorf("the %s template lacks %s:\n%s", c.template, h, d.Draft.HTML)
				}
			}
			if !strings.Contains(d.Draft.HTML, `data-pb-id="b-hero"`) && c.template != pages.TemplateBlank {
				t.Errorf("blocks have ids: %s", d.Draft.HTML)
			}
			if got := blocking(d.Checks); strings.Join(got, ",") != strings.Join(append([]string{}, c.blocks...), ",") {
				t.Errorf("blocking checks %v, want %v: %+v", got, c.blocks, d.Checks)
			}
			if c.products > 0 {
				p := check(t, d.Checks, pages.CheckProduct)
				if p.Problem != "missing" || p.Count != c.products {
					t.Errorf("product check %+v, want %d missing", p, c.products)
				}
			}
			if a := check(t, d.Checks, pages.CheckAddress); a.Status != pages.CheckPass || a.Detail != "free" || a.Path != "/"+c.slug {
				t.Errorf("address check %+v", a)
			}
			if d.Publish.CanPublish != (len(c.blocks) == 0) || !d.Publish.CanEdit || !d.Publish.CanManage {
				t.Errorf("publish state %+v with blocking %v", d.Publish, c.blocks)
			}
			if n := f.audits("page.created", d.ID); n != 1 {
				t.Errorf("%d page.created audit entries", n)
			}
			var vs pages.PageVersionList
			f.owner.Do(http.MethodGet, pagesAPI+"/"+d.ID+"/versions", nil).MustStatus(t, 200).JSON(t, &vs)
			if len(vs.Items) != 1 || vs.Items[0].Label != "created" || !vs.Items[0].Current || vs.Items[0].Live {
				t.Errorf("versions %+v", vs.Items)
			}
		})
	}

	t.Run("the template's name when none is given", func(t *testing.T) {
		d := f.create(f.owner, pages.TemplateBlank, "", "")
		if d.Name != "New page" || d.Slug != "new-page" {
			t.Fatalf("default name %q at /%s", d.Name, d.Slug)
		}
		w := f.create(f.owner, pages.TemplateWebinar, "", "")
		if w.Name != "Live session" || w.Slug != "live-session" {
			t.Fatalf("webinar default %q at /%s", w.Name, w.Slug)
		}
		// A second one at the same address gets a free one suggested.
		p := mustProblem(t, f.owner.Do(http.MethodPost, pagesAPI, map[string]any{"template": "blank", "name": ""}), http.StatusConflict, "slug_taken")
		if p.Meta["suggestion"] != "new-page-2" {
			t.Fatalf("suggestion %v", p.Meta)
		}
		if d := f.create(f.owner, pages.TemplateBlank, "", "new-page-2"); d.Slug != "new-page-2" || d.Name != "New page" {
			t.Fatalf("with the suggested address: %q /%s", d.Name, d.Slug)
		}
	})
	t.Run("an unknown template", func(t *testing.T) {
		mustProblem(t, f.owner.Do(http.MethodPost, pagesAPI, map[string]any{"template": "landing", "name": "X"}), http.StatusUnprocessableEntity, "validation_failed")
	})
}

// Addresses: the shape, the site's own addresses, another page's, and a
// published page that moves (its old address redirects and stays taken).
func TestPageAddresses(t *testing.T) {
	f := newFixture(t)
	create := func(name, slug string) *apptest.Response {
		body := map[string]any{"template": "blank", "name": name}
		if slug != "" {
			body["slug"] = slug
		}
		return f.owner.Do(http.MethodPost, pagesAPI, body)
	}

	t.Run("invalid", func(t *testing.T) {
		for _, slug := range []string{"Bad Slug!", "-lead", "trail-", "two--hyphens", "under_score", "émile"} {
			p := mustProblem(t, create("Something", slug), http.StatusUnprocessableEntity, "validation_failed")
			if len(p.Errors) != 1 || p.Errors[0].Location != "body.slug" {
				t.Errorf("%q: %+v", slug, p)
			}
		}
		// Upper case is lowered, not refused.
		if r := create("Something", "Spring-Sale"); r.Status != http.StatusCreated || !strings.Contains(string(r.Body), `"slug":"spring-sale"`) {
			t.Fatalf("upper case: %d %s", r.Status, r.Body)
		}
	})
	t.Run("the site's own", func(t *testing.T) {
		for _, slug := range []string{"products", "signin", "account", "posts", "p", "api", "buy", "d", "subscribe", "feed", "checkout"} {
			p := mustProblem(t, create("Something", slug), http.StatusUnprocessableEntity, "validation_failed")
			if !strings.Contains(p.Detail, "/"+slug+" belongs to the site itself") {
				t.Errorf("%s: %s", slug, p.Detail)
			}
		}
		// A name that makes one.
		p := mustProblem(t, create("Products", ""), http.StatusUnprocessableEntity, "validation_failed")
		if !strings.Contains(p.Detail, "/products belongs to the site") {
			t.Errorf("from the name: %s", p.Detail)
		}
	})
	t.Run("another page's", func(t *testing.T) {
		for slug, suggestion := range map[string]string{"field-guide": "field-guide-2", "sponsor": "sponsor-2"} {
			p := mustProblem(t, create("Something", slug), http.StatusConflict, "slug_taken")
			if p.Meta["suggestion"] != suggestion || !strings.Contains(p.Detail, "/"+slug) {
				t.Errorf("%s: %+v", slug, p)
			}
		}
		// From the name: About's address.
		p := mustProblem(t, create("About", ""), http.StatusConflict, "slug_taken")
		if p.Meta["suggestion"] != "about-2" {
			t.Errorf("about: %+v", p.Meta)
		}
	})

	// A page that moves.
	d := f.create(f.owner, pages.TemplateBlank, "Autumn sale", "")
	t.Run("settings refuse the same addresses", func(t *testing.T) {
		in := settingsOf(d)
		in.Slug = "about"
		mustProblem(t, f.settings(f.owner, d.ID, in), http.StatusConflict, "slug_taken")
		in.Slug = "signin"
		mustProblem(t, f.settings(f.owner, d.ID, in), http.StatusUnprocessableEntity, "validation_failed")
		in.Slug = "not valid"
		mustProblem(t, f.settings(f.owner, d.ID, in), http.StatusUnprocessableEntity, "validation_failed")
		if got := f.detail(f.owner, d.ID); got.Slug != "autumn-sale" || got.Draft.Version != 1 {
			t.Fatalf("refused settings changed the page: /%s v%d", got.Slug, got.Draft.Version)
		}
	})
	f.publishNow(f.owner, d.ID)
	in := settingsOf(d)
	in.Slug = "fall-sale"
	var moved pages.PageDetail
	f.settings(f.owner, d.ID, in).MustStatus(t, http.StatusOK).JSON(t, &moved)
	if moved.Slug != "fall-sale" || moved.Path != "/fall-sale" || moved.Live == nil || moved.Live.Slug != "autumn-sale" || !moved.Changes {
		t.Fatalf("moved draft: slug %q path %q live %+v changes %v", moved.Slug, moved.Path, moved.Live, moved.Changes)
	}
	// Until it's published the page answers at its old address.
	if res := f.resolve("autumn-sale"); res.Page == nil || res.Page.ID.String() != d.ID {
		t.Fatalf("the live page before publishing the move: %+v", res)
	}
	if res := f.resolve("fall-sale"); res.Page != nil || res.MovedTo != "" || res.Gone {
		t.Fatalf("the new address before publishing: %+v", res)
	}
	moved = f.publishNow(f.owner, d.ID)
	if moved.Live == nil || moved.Live.Slug != "fall-sale" || moved.Changes {
		t.Fatalf("published the move: %+v", moved.Live)
	}
	if n := scan[int64](f, `SELECT count(*) FROM page_redirects WHERE workspace_id = $1 AND slug = 'autumn-sale' AND page_id = $2`, f.ws.ID, d.ID); n != 1 {
		t.Fatalf("redirects from the old address: %d", n)
	}
	if meta := scan[string](f, `SELECT metadata->>'redirect_from' FROM audit_events WHERE action = 'page.changes_published' AND target_id = $1`, d.ID); meta != "/autumn-sale" {
		t.Errorf("the audit entry names the redirect: %q", meta)
	}
	if res := f.resolve("autumn-sale"); res.MovedTo != "fall-sale" {
		t.Fatalf("the old address: %+v", res)
	}
	if res := f.resolve("fall-sale"); res.Page == nil || res.Page.Slug != "fall-sale" {
		t.Fatalf("the new address: %+v", res)
	}
	t.Run("the old address stays taken", func(t *testing.T) {
		p := mustProblem(t, create("Something", "autumn-sale"), http.StatusConflict, "slug_taken")
		if p.Meta["suggestion"] != "autumn-sale-2" {
			t.Errorf("suggestion %v", p.Meta)
		}
	})
	t.Run("moving back to it", func(t *testing.T) {
		in.Slug = "autumn-sale"
		f.settings(f.owner, d.ID, in).MustStatus(t, http.StatusOK)
		f.publishNow(f.owner, d.ID)
		if res := f.resolve("autumn-sale"); res.Page == nil || res.Page.ID.String() != d.ID {
			t.Fatalf("back at the old address: %+v", res)
		}
		if res := f.resolve("fall-sale"); res.MovedTo != "autumn-sale" {
			t.Fatalf("the second address redirects: %+v", res)
		}
		if n := scan[int64](f, `SELECT count(*) FROM page_redirects WHERE workspace_id = $1 AND slug = 'autumn-sale'`, f.ws.ID); n != 0 {
			t.Fatalf("the page's own address is no redirect: %d", n)
		}
	})
}

// Forms & pages › Pages: the demo's six pages with their numbers, the
// template chips and the shop's row.
func TestPagesOverview(t *testing.T) {
	f := newFixture(t)
	var ov pages.PagesOverview
	f.owner.Do(http.MethodGet, pagesAPI, nil).MustStatus(t, http.StatusOK).JSON(t, &ov)
	if len(ov.Pages) != 6 || ov.Period != "30d" || !ov.WebsiteOn || !ov.CanEdit || !ov.CanPublish {
		t.Fatalf("overview: %d pages, period %s, website %v, can %v/%v", len(ov.Pages), ov.Period, ov.WebsiteOn, ov.CanEdit, ov.CanPublish)
	}
	bySlug := map[string]pages.PageListItem{}
	for _, p := range ov.Pages {
		bySlug[p.Slug] = p
	}
	for slug, want := range map[string]struct{ template, status, result string }{
		"field-guide":   {pages.TemplateSales, pages.StatusPublished, pages.ResultSales},
		"dns-checklist": {pages.TemplateLeadMagnet, pages.StatusPublished, pages.ResultSignups},
		"launch-kit":    {pages.TemplateLaunch, pages.StatusPublished, pages.ResultSales},
		"dmarc-live":    {pages.TemplateWebinar, pages.StatusPublished, pages.ResultRegistrations},
		"about":         {pages.TemplateAbout, pages.StatusPublished, pages.ResultSignups},
		"sponsor":       {pages.TemplateBlank, pages.StatusDraft, pages.ResultNone},
	} {
		p, ok := bySlug[slug]
		if !ok {
			t.Fatalf("no /%s in %+v", slug, ov.Pages)
		}
		if p.Template != want.template || p.Status != want.status || p.Result.Kind != want.result || p.Path != "/"+slug || p.Visibility != pages.VisibilityPublic {
			t.Errorf("/%s: %+v", slug, p)
		}
	}
	if fg := bySlug["field-guide"]; !fg.Changes || fg.Views == 0 || fg.FirstPublishedAt == nil {
		t.Errorf("the Field Guide has draft changes and views: %+v", fg)
	}
	if dns := bySlug["dns-checklist"]; dns.Changes || dns.Result.Count == 0 || dns.Result.Conversion <= 0 {
		t.Errorf("the checklist's signups: %+v", dns.Result)
	}
	if sp := bySlug["sponsor"]; sp.EditedBy == nil || sp.EditedBy.Name != "Daniel Park" || sp.Views != 0 || sp.FirstPublishedAt != nil {
		t.Errorf("Daniel's draft: %+v", sp)
	}
	lk := bySlug["launch-kit"]
	cds := pages.Analyze(f.liveHTML("launch-kit")).Countdowns
	if len(cds) != 1 || !cds[0].At.Before(time.Now()) {
		t.Fatalf("the Launch Kit's countdown: %+v", cds)
	}
	if lk.CountdownEnded == nil || !lk.CountdownEnded.Equal(cds[0].At) {
		t.Errorf("the Launch Kit's countdown ended at %v, the list says %v", cds[0].At, lk.CountdownEnded)
	}
	if len(ov.Templates) != 6 {
		t.Errorf("template chips %+v", ov.Templates)
	}
	for _, c := range ov.Templates {
		if c.Count != 1 {
			t.Errorf("chip %+v", c)
		}
	}
	if !ov.Shop.On || ov.Shop.Products != 2 || ov.Shop.Path != "/products" || !strings.HasSuffix(ov.Shop.URL, "/products") || !ov.Shop.CanToggle {
		t.Errorf("shop %+v", ov.Shop)
	}
	if ov.Stats.Views == 0 || !ov.Stats.ShopViewed || ov.Stats.Signups == 0 || ov.Stats.Registrations == 0 {
		t.Errorf("stats %+v", ov.Stats)
	}
	f.owner.Do(http.MethodGet, pagesAPI+"?period=7d", nil).MustStatus(t, http.StatusOK).JSON(t, &ov)
	if ov.Period != "7d" {
		t.Errorf("period %s", ov.Period)
	}
	mustProblem(t, f.owner.Do(http.MethodGet, pagesAPI+"?period=1y", nil), http.StatusUnprocessableEntity, "")
	mustProblem(t, f.owner.Do(http.MethodGet, pagesAPI+"/"+uuid.Must(uuid.NewV7()).String(), nil), http.StatusNotFound, "not_found")
}
