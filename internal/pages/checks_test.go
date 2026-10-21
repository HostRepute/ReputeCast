package pages_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/pages"
)

// Blocking checks stop publishing (409 checks_failed with their keys);
// warnings don't.
func TestPublishChecksBlock(t *testing.T) {
	f := newFixture(t)
	publish := func(id string) *problem {
		r := f.owner.Do(http.MethodPost, pagesAPI+"/"+id+"/publish", map[string]any{"when": "now"})
		if r.Status == http.StatusOK {
			return nil
		}
		p := mustProblem(t, r, http.StatusConflict, "checks_failed")
		return &p
	}
	// blocked asserts the page's blocking checks and that publishing names them.
	blocked := func(t *testing.T, d pages.PageDetail, keys ...string) {
		t.Helper()
		if got := blocking(d.Checks); !slices.Equal(got, keys) {
			t.Fatalf("blocking %v, want %v: %+v", got, keys, d.Checks)
		}
		if d.Publish.CanPublish {
			t.Errorf("canPublish with blocking checks")
		}
		p := publish(d.ID)
		if p == nil {
			t.Fatalf("published with blocking checks %v", keys)
		}
		// One key per blocking check: two product blocks name "product" twice.
		var names []string
		for _, k := range p.Meta["checks"].([]any) {
			if !slices.Contains(names, k.(string)) {
				names = append(names, k.(string))
			}
		}
		if !slices.Equal(names, keys) {
			t.Errorf("checks_failed names %v, want %v", names, keys)
		}
		if got := f.detail(f.owner, d.ID); got.Status != pages.StatusDraft || got.Live != nil {
			t.Errorf("a refused publish changed the page: %s", got.Status)
		}
	}
	formBlock := func(form string) string {
		return section("b-signup", `<rc-form data-pb-id="signup" form="`+form+`" heading="Join"></rc-form>`)
	}
	productBlock := func(product string) string {
		return section("b-buy", `<rc-product data-pb-id="buy" product="`+product+`" button="Buy for {price}" price="1" offer="1" layout="box"></rc-product>`)
	}
	top := plain("Checks", "A line.")

	t.Run("a signup block without a form", func(t *testing.T) {
		d := f.page("No form", top+section("b-signup", `<rc-form data-pb-id="signup" heading="Join"></rc-form>`))
		c := check(t, d.Checks, pages.CheckForm)
		if c.Problem != "missing" || c.Count != 1 {
			t.Errorf("form check %+v", c)
		}
		blocked(t, d, pages.CheckForm)
	})
	t.Run("a paused form", func(t *testing.T) {
		waitlist := f.formID("frm_a91f55")
		f.exec(`UPDATE forms SET status = 'paused' WHERE id = $1`, waitlist)
		d := f.page("Paused form", top+formBlock(waitlist))
		c := check(t, d.Checks, pages.CheckForm)
		if c.Problem != "paused" || c.Name != "Launch Kit waitlist" || c.Ref != waitlist || c.At == nil {
			t.Errorf("form check %+v", c)
		}
		blocked(t, d, pages.CheckForm)
		f.exec(`UPDATE forms SET status = 'live' WHERE id = $1`, waitlist)
	})
	t.Run("a deleted form", func(t *testing.T) {
		d := f.page("Deleted form", top+formBlock(uuid.Must(uuid.NewV7()).String()))
		if c := check(t, d.Checks, pages.CheckForm); c.Problem != "deleted" {
			t.Errorf("form check %+v", c)
		}
		blocked(t, d, pages.CheckForm)
	})
	t.Run("a live form passes", func(t *testing.T) {
		signup := f.formID("frm_8c2e1a")
		d := f.page("Live form", top+formBlock(signup))
		if c := check(t, d.Checks, pages.CheckForm); c.Status != pages.CheckPass || c.Ref != signup || c.Name == "" {
			t.Errorf("form check %+v", c)
		}
	})
	t.Run("a product that isn't published", func(t *testing.T) {
		d := f.page("Draft product", top+productBlock(f.productID("warm-up-planner")))
		c := check(t, d.Checks, pages.CheckProduct)
		if c.Problem != "draft" || c.Name != "Warm-up planner" {
			t.Errorf("product check %+v", c)
		}
		if s := check(t, d.Checks, pages.CheckStripe); s.Status != pages.CheckPass || s.Count != 1 || s.Account == "" {
			t.Errorf("stripe check %+v", s)
		}
		blocked(t, d, pages.CheckProduct)
	})
	t.Run("a product that's gone, and one not picked", func(t *testing.T) {
		d := f.page("Gone product", top+productBlock(uuid.Must(uuid.NewV7()).String())+
			section("b-buy-2", `<rc-product data-pb-id="buy-2" button="Buy"></rc-product>`))
		var problems []string
		for _, c := range checksOf(d.Checks, pages.CheckProduct) {
			problems = append(problems, c.Problem)
		}
		slices.Sort(problems)
		if !slices.Equal(problems, []string{"deleted", "missing"}) {
			t.Errorf("product checks %+v", d.Checks)
		}
		blocked(t, d, pages.CheckProduct)
	})
	t.Run("an archived product", func(t *testing.T) {
		kit := f.productID("newsletter-launch-kit")
		f.exec(`UPDATE products SET status = 'archived' WHERE id = $1`, kit)
		d := f.page("Archived product", top+productBlock(kit))
		if c := check(t, d.Checks, pages.CheckProduct); c.Problem != "archived" || c.Name != "Newsletter Launch Kit" {
			t.Errorf("product check %+v", c)
		}
		blocked(t, d, pages.CheckProduct)
		f.exec(`UPDATE products SET status = 'published' WHERE id = $1`, kit)
	})
	t.Run("a published product passes with its price", func(t *testing.T) {
		d := f.page("Published product", top+productBlock(f.productID("deliverability-field-guide")))
		c := check(t, d.Checks, pages.CheckProduct)
		if c.Status != pages.CheckPass || c.Name != "The Deliverability Field Guide" || c.Detail != "$29" || c.PWYW {
			t.Errorf("product check %+v", c)
		}
		k := f.page("PWYW product", top+productBlock(f.productID("newsletter-launch-kit")))
		if c := check(t, k.Checks, pages.CheckProduct); c.Detail != "$5" || !c.PWYW {
			t.Errorf("pay what you want: %+v", c)
		}
	})
	t.Run("a Buy button without a payment connection", func(t *testing.T) {
		f.exec(`UPDATE payment_connections SET status = 'disconnected' WHERE workspace_id = $1 AND status = 'connected'`, f.ws.ID)
		d := f.page("No Stripe", top+productBlock(f.productID("deliverability-field-guide")))
		if s := check(t, d.Checks, pages.CheckStripe); s.Status != pages.CheckBlock || s.Count != 1 {
			t.Errorf("stripe check %+v", s)
		}
		blocked(t, d, pages.CheckStripe)
	})
	t.Run("members-only without the website", func(t *testing.T) {
		d := f.page("Members notes", top)
		in := settingsOf(d)
		in.Visibility = pages.VisibilityMembers
		f.settings(f.owner, d.ID, in).MustStatus(t, http.StatusOK)
		f.exec(`UPDATE website_settings SET status = 'off' WHERE workspace_id = $1`, f.ws.ID)
		d = f.detail(f.owner, d.ID)
		if c := check(t, d.Checks, pages.CheckVisibility); c.Status != pages.CheckBlock || c.Detail != pages.VisibilityMembers {
			t.Errorf("visibility check %+v", c)
		}
		blocked(t, d, pages.CheckVisibility)
		// The Page tab refuses Members and Paid while the website is off.
		p := mustProblem(t, f.settings(f.owner, f.pageID("sponsor"), func() pages.PageSettingsInput {
			s := settingsOf(f.detail(f.owner, f.pageID("sponsor")))
			s.Visibility = pages.VisibilityPaid
			return s
		}()), http.StatusUnprocessableEntity, "validation_failed")
		if !strings.Contains(p.Detail, "Turn the website on first") {
			t.Errorf("paid without the website: %s", p.Detail)
		}
		f.exec(`UPDATE website_settings SET status = 'on' WHERE workspace_id = $1`, f.ws.ID)
	})
}

// Warnings name what to look at; they don't stop publishing.
func TestPublishCheckWarnings(t *testing.T) {
	f := newFixture(t)
	// The link check's answers are cached for the process: addresses of
	// this test only.
	broken := "https://broken.example/" + uuid.Must(uuid.NewV7()).String()
	working := "https://acme.io/pricing?v=" + uuid.Must(uuid.NewV7()).String()
	html := section("b-intro", `<p>No heading opens this page.</p><img src="/media/no-alt.png"><img src="/media/cover.png" alt="The cover">`) +
		section("b-links", `<p><a href="`+broken+`">Old terms</a> <a href="`+working+`">Pricing</a> <a href="/products">Shop</a> <a href="mailto:hi@acme.io">Mail</a></p>`) +
		section("b-moments", `<rc-countdown data-pb-id="countdown" at="2026-01-05T09:00" tz="UTC" units="days,hours" end="hide" label="Opened"></rc-countdown>`+
			`<rc-event data-pb-id="event" date="2026-01-06" start="17:00" end="17:30" tz="UTC" title="Past session"></rc-event>`) +
		section("b-store", `<rc-store-product store="acme-merch" product="mug"></rc-store-product>`)
	d := f.page("Warnings", html)
	if len(blocking(d.Checks)) != 0 || !d.Publish.CanPublish {
		t.Fatalf("warnings block: %+v", d.Checks)
	}
	if c := check(t, d.Checks, pages.CheckHeading); c.Status != pages.CheckWarn {
		t.Errorf("heading %+v", c)
	}
	if c := check(t, d.Checks, pages.CheckImages); c.Status != pages.CheckWarn || c.Ref != "/media/no-alt.png" || c.Name != "no-alt.png" {
		t.Errorf("images %+v", c)
	}
	if c := check(t, d.Checks, pages.CheckLinks); c.Status != pages.CheckWarn || c.Ref != broken || c.Name != "Old terms" || c.HTTPStatus != http.StatusNotFound {
		t.Errorf("links %+v", c)
	}
	if c := check(t, d.Checks, pages.CheckMoments); c.Status != pages.CheckWarn || c.Detail != "countdown" || c.Name != "Opened" || c.At == nil {
		t.Errorf("moments %+v", c)
	}
	if c := check(t, d.Checks, pages.CheckStoreBlocks); c.Status != pages.CheckWarn || c.Count != 1 {
		t.Errorf("store blocks %+v", c)
	}
	if d.Counts.Blocks != 4 || d.Counts.Images != 2 || d.Counts.ImagesWithAlt != 1 || d.Counts.Links != 4 {
		t.Errorf("counts %+v", d.Counts)
	}
	// Only outside links are fetched, each once while the answer is fresh.
	f.detail(f.owner, d.ID)
	if n := f.links.count(broken); n != 1 {
		t.Errorf("the broken link was fetched %d times", n)
	}
	if n := f.links.count(working); n != 1 {
		t.Errorf("the working link was fetched %d times", n)
	}
	for u := range map[string]bool{"/products": true, "mailto:hi@acme.io": true} {
		if f.links.count(u) != 0 {
			t.Errorf("%s was fetched", u)
		}
	}
	// Blocking first, then warnings, then what passed.
	rank := map[string]int{pages.CheckBlock: 0, pages.CheckWarn: 1, pages.CheckPass: 2}
	if !slices.IsSortedFunc(d.Checks, func(a, b pages.PageCheck) int { return rank[a.Status] - rank[b.Status] }) {
		t.Errorf("checks out of order: %+v", d.Checks)
	}

	pub := f.publishNow(f.owner, d.ID)
	if pub.Status != pages.StatusPublished {
		t.Fatalf("status %s", pub.Status)
	}
	// Once published, a countdown that ended did its job; an event in the
	// past still warns.
	if c := check(t, pub.Checks, pages.CheckMoments); c.Status != pages.CheckWarn || c.Detail != "event" || c.Name != "Past session" {
		t.Errorf("moments after publishing %+v", c)
	}
	if c := check(t, pub.Checks, pages.CheckAddress); c.Status != pages.CheckPass || c.Detail != "yours" {
		t.Errorf("address of a published page %+v", c)
	}

	// Links that all answer pass, with their count.
	ok := f.page("Good links", plain("Links", `<a href="https://acme.io/docs">Docs</a> <a href="/products">Shop</a>`))
	if c := check(t, ok.Checks, pages.CheckLinks); c.Status != pages.CheckPass || c.Count != 2 {
		t.Errorf("links %+v", c)
	}
	// The demo's beta terms are gone (PageBuilderStates).
	beta := f.page("Beta terms", plain("Beta", `<a href="https://acme.io/beta-terms">Beta terms</a>`))
	if c := check(t, beta.Checks, pages.CheckLinks); c.Status != pages.CheckWarn || c.HTTPStatus != http.StatusNotFound {
		t.Errorf("beta terms %+v", c)
	}
}

// A signup block's form must be live: a draft form takes no signups and
// the site doesn't draw it.
func TestPublishCheckDraftForm(t *testing.T) {
	f := newFixture(t)
	beta := f.formID("frm_b7a412")
	f.exec(`UPDATE forms SET status = 'draft' WHERE id = $1`, beta)
	d := f.page("Draft form", plain("Draft form", "A line.")+section("b-signup", `<rc-form data-pb-id="signup" form="`+beta+`" heading="Join"></rc-form>`))
	if c := check(t, d.Checks, pages.CheckForm); c.Status != pages.CheckBlock {
		t.Fatalf("a draft form passes the check: %+v", c)
	}
}
