package publicpages_test

// Landing and sales pages on the public site (design expansion phase 7,
// X19), as readers' browsers reach them: the demo's pages at
// /p/acme-newsletter/{slug}, the shop, the builder's previews. Pages are
// made and published through the API as Linh Nguyen (Owner).
//
// Signups through page forms count against the per-address limit (5 a
// minute): each test makes at most five, on its own demo clone. Signups go
// through forms the demo doesn't verify in real time (the DMARC live and
// DNS checklist forms), so nothing leaves the process.

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/pages"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/website"
)

const spSite = "/p/acme-newsletter/"

// spEnv is one test's demo clone for the sell pages: the link check never
// leaves the process, System mail goes to a fake, and api is Linh Nguyen.
type spEnv struct {
	*apptest.Env
	t    *testing.T
	ids  demoIDs
	ws   store.Workspace
	api  *apptest.Client
	mail *fakeMail
}

func spDemo(t *testing.T) *spEnv {
	t.Helper()
	env, ids := demo(t)
	e := &spEnv{Env: env, t: t, ids: ids, mail: &fakeMail{}}
	env.App.Mail = e.mail
	env.App.Pages().Probe = func(_ context.Context, u string) (int, error) {
		if strings.Contains(u, "broken") {
			return http.StatusNotFound, nil
		}
		return http.StatusOK, nil
	}
	ws, err := env.App.Q.GetWorkspaceBySlug(context.Background(), wsSlug)
	if err != nil {
		t.Fatal(err)
	}
	e.ws = ws
	e.api = owner(env)
	return e
}

func (e *spEnv) exec(sql string, args ...any) {
	e.t.Helper()
	if _, err := e.App.Pool.Exec(context.Background(), sql, args...); err != nil {
		e.t.Fatalf("%s: %v", sql, err)
	}
}

func (e *spEnv) pageID(slug string) string {
	e.t.Helper()
	return one[uuid.UUID](e.t, e.Env, `SELECT id FROM pages WHERE workspace_id = $1 AND slug = $2 AND deleted_at IS NULL`, e.ws.ID, slug).String()
}

func (e *spEnv) formID(public string) string {
	e.t.Helper()
	return one[uuid.UUID](e.t, e.Env, `SELECT id FROM forms WHERE workspace_id = $1 AND public_id = $2`, e.ws.ID, public).String()
}

func (e *spEnv) productID(slug string) string {
	e.t.Helper()
	return one[uuid.UUID](e.t, e.Env, `SELECT id FROM products WHERE workspace_id = $1 AND slug = $2 AND deleted_at IS NULL`, e.ws.ID, slug).String()
}

func (e *spEnv) detail(id string) pages.PageDetail {
	e.t.Helper()
	var d pages.PageDetail
	e.api.Do(http.MethodGet, wsAPI+"/pages/"+id, nil).MustStatus(e.t, http.StatusOK).JSON(e.t, &d)
	return d
}

// newPage is a blank page whose draft is html.
func (e *spEnv) newPage(name, html string) pages.PageDetail {
	e.t.Helper()
	var d pages.PageDetail
	e.api.Do(http.MethodPost, wsAPI+"/pages", map[string]any{"template": "blank", "name": name}).MustStatus(e.t, http.StatusCreated).JSON(e.t, &d)
	e.save(d.ID, html)
	return e.detail(d.ID)
}

// save saves html as the page's draft.
func (e *spEnv) save(id, html string) {
	e.t.Helper()
	d := e.detail(id)
	e.api.Do(http.MethodPut, wsAPI+"/pages/"+id+"/content", map[string]any{"html": html, "baseVersion": d.Draft.Version, "manual": true}).MustStatus(e.t, http.StatusOK)
}

func (e *spEnv) publish(id string) pages.PageDetail {
	e.t.Helper()
	var d pages.PageDetail
	e.api.Do(http.MethodPost, wsAPI+"/pages/"+id+"/publish", map[string]any{"when": "now"}).MustStatus(e.t, http.StatusOK).JSON(e.t, &d)
	return d
}

// settings changes the Page tab.
func (e *spEnv) settings(id string, change func(*pages.PageSettingsInput)) {
	e.t.Helper()
	d := e.detail(id)
	in := pages.PageSettingsInput{Name: d.Name, Slug: d.Slug, SEOTitle: d.Settings.SEOTitle, SEODescription: d.Settings.SEODescription,
		SocialImage: d.Settings.SocialImage, Visibility: d.Settings.Visibility, Tiers: d.Settings.Tiers, HideHeader: d.Settings.HideHeader,
		ShowInNav: d.Settings.ShowInNav, NavLabel: d.Settings.NavLabel}
	change(&in)
	e.api.Do(http.MethodPut, wsAPI+"/pages/"+id+"/settings", in).MustStatus(e.t, http.StatusOK)
}

// member is the cookie of one of the demo's site sessions: "sam" holds
// Weekly Digest Insider, "alex" is a free member.
func (e *spEnv) member(key string) hdr {
	return hdr{"Cookie": website.SessionCookie + "=demo-session-" + key + "-" + e.ws.ID.String()}
}

// views counts a page's views over every day (the shop is the nil page).
func (e *spEnv) views(page string) int64 {
	e.t.Helper()
	return one[int64](e.t, e.Env, `SELECT coalesce(sum(views), 0)::bigint FROM website_view_days WHERE workspace_id = $1 AND page_kind = 'page' AND page_id = $2`, e.ws.ID, page)
}

// event is the demo webinar's event as its live page has it: the demo
// anchors it to today (14 days ahead, 17:00–17:30 UTC).
func (e *spEnv) event() (pages.EventInfo, time.Time) {
	e.t.Helper()
	src := one[string](e.t, e.Env, `SELECT v.html FROM pages p JOIN page_versions v ON v.page_id = p.id AND v.version = p.live_version
		WHERE p.workspace_id = $1 AND p.slug = 'dmarc-live'`, e.ws.ID)
	ev, ok := pages.FirstEvent(src)
	if !ok {
		e.t.Fatal("the webinar page has no event")
	}
	start, ok := pages.Local(ev.Date+"T"+ev.Start, ev.TZ)
	if !ok || ev.Start != "17:00" || ev.End != "17:30" || ev.TZ != "UTC" || !start.After(time.Now()) {
		e.t.Fatalf("the webinar's event %+v", ev)
	}
	return ev, start
}

// spSection is a top-level block.
func spSection(id, inner string) string {
	return `<section data-pb="text" data-pb-id="` + id + `" data-pb-name="Text" class="sp-sec"><div class="sp-in nar">` + inner + `</div></section>`
}

func spPlain(heading, line string) string {
	return spSection("b-text", `<div class="prose"><h1>`+heading+`</h1><p>`+line+`</p></div>`)
}

// Published pages answer at their address; drafts and unknown addresses
// 404; a moved page redirects (301); an unpublished page answers 410 or
// sends readers where its publisher chose (302); a deleted one 410.
func TestCustomPageAddresses(t *testing.T) {
	e := spDemo(t)
	fg := get(t, e.Env, spSite+"field-guide")
	must(t, fg, http.StatusOK, "<h1>The Deliverability Field Guide</h1>", "Eight chapters on authentication, warm-up and list hygiene.", "Buy for $29",
		"Sofia Rossi", "<title>The Deliverability Field Guide · Acme</title>", `<link rel="canonical" href="http://news.acme.io/field-guide">`,
		`class="st-nav"`, `aria-current="page"`, `class="buybar"`, `action="`+e.Server.URL+spSite+`buy/`+e.productID("deliverability-field-guide")+`"`)
	// The draft's changes aren't live.
	lacks(t, fg, "Eight practical chapters", "Hannah Weiss")
	if !strings.HasPrefix(fg.Header.Get("Cache-Control"), "public") {
		t.Errorf("a public page's Cache-Control %q", fg.Header.Get("Cache-Control"))
	}
	if strings.Contains(fg.Body, "<script>") || strings.Contains(fg.Body, " style=") || strings.Contains(fg.Header.Get("Content-Security-Policy"), "unsafe-inline") {
		t.Error("the page has inline script or style; its CSP allows neither")
	}
	if h := do(t, e.Env, http.MethodHead, spSite+"field-guide", nil, nil); h.Status != http.StatusOK || h.Body != "" {
		t.Errorf("HEAD: %d %q", h.Status, h.Body)
	}
	// About's lead ("The Acme newsletter" is the site's title as well).
	const aboutLead = "Every Tuesday since 2021, for 48,000 readers."
	for slug, text := range map[string]string{
		"dns-checklist": "Every DNS record your newsletter needs, on two pages",
		"launch-kit":    "<h1>Newsletter Launch Kit</h1>",
		"dmarc-live":    "<h1>Live: Fix your DMARC setup in 30 minutes</h1>",
		"about":         aboutLead,
	} {
		must(t, get(t, e.Env, spSite+slug), http.StatusOK, text)
	}
	// Focused landing pages keep the header to the logo.
	dns := get(t, e.Env, spSite+"dns-checklist")
	lacks(t, dns, `class="st-nav"`)
	// The webinar: the event in its zone; the joining link never on the page.
	_, start := e.event()
	dmarc := get(t, e.Env, spSite+"dmarc-live")
	must(t, dmarc, http.StatusOK, start.Format("2 January 2006"), "17:00–17:30 UTC", "Online, on Zoom", "Joining link after you register",
		"Sent by email, with reminders 1 day and 1 hour before", `data-start="`+start.UTC().Format(time.RFC3339)+`"`)
	lacks(t, dmarc, "acme.zoom.us")

	must(t, get(t, e.Env, spSite+"sponsor"), http.StatusNotFound)
	must(t, get(t, e.Env, spSite+"no-such-page"), http.StatusNotFound)
	must(t, get(t, e.Env, "/p/no-such-workspace/field-guide"), http.StatusNotFound)

	// On the custom domain, at the root.
	cd := do(t, e.Env, http.MethodGet, "/field-guide", nil, hdr{"Host": "news.acme.io"})
	must(t, cd, http.StatusOK, "<h1>The Deliverability Field Guide</h1>", `action="http://news.acme.io/buy/`)

	// Moved: About to /about-us.
	about := e.pageID("about")
	e.settings(about, func(in *pages.PageSettingsInput) { in.Slug = "about-us" })
	must(t, get(t, e.Env, spSite+"about"), http.StatusOK, aboutLead)
	must(t, get(t, e.Env, spSite+"about-us"), http.StatusNotFound)
	e.publish(about)
	moved := get(t, e.Env, spSite+"about?utm_source=digest")
	if moved.Status != http.StatusMovedPermanently || !strings.HasSuffix(moved.Header.Get("Location"), spSite+"about-us?utm_source=digest") {
		t.Fatalf("the old address: %d %q", moved.Status, moved.Header.Get("Location"))
	}
	must(t, get(t, e.Env, spSite+"about-us"), http.StatusOK, aboutLead)

	// Unpublished: 410, or the path its publisher chose.
	dm := e.pageID("dmarc-live")
	e.api.Do(http.MethodPost, wsAPI+"/pages/"+dm+"/unpublish", map[string]any{"reason": "The session is full"}).MustStatus(t, http.StatusOK)
	gone := get(t, e.Env, spSite+"dmarc-live")
	must(t, gone, http.StatusGone)
	lacks(t, gone, "Save your seat")
	lk := e.pageID("launch-kit")
	e.api.Do(http.MethodPost, wsAPI+"/pages/"+lk+"/unpublish", map[string]any{"redirectTo": "/products"}).MustStatus(t, http.StatusOK)
	sent := get(t, e.Env, spSite+"launch-kit")
	if sent.Status != http.StatusFound || !strings.HasSuffix(sent.Header.Get("Location"), spSite+"products") {
		t.Fatalf("unpublished to /products: %d %q", sent.Status, sent.Header.Get("Location"))
	}
	// Deleted after being live: 410.
	e.api.Do(http.MethodDelete, wsAPI+"/pages/"+e.pageID("dns-checklist"), nil).MustStatus(t, http.StatusNoContent)
	must(t, get(t, e.Env, spSite+"dns-checklist"), http.StatusGone)
}

// Pages answer like posts when they're off the site, with the website
// off too: a page taken off the site is gone (410), not unknown (404).
func TestGonePageWithoutTheWebsite(t *testing.T) {
	e := spDemo(t)
	e.api.Do(http.MethodPost, wsAPI+"/pages/"+e.pageID("dmarc-live")+"/unpublish", map[string]any{}).MustStatus(t, http.StatusOK)
	e.api.Do(http.MethodDelete, wsAPI+"/pages/"+e.pageID("dns-checklist"), nil).MustStatus(t, http.StatusNoContent)
	websiteOff(t, e.Env, e.ws.ID)
	// Pages work without the website.
	must(t, get(t, e.Env, spSite+"field-guide"), http.StatusOK, "The Deliverability Field Guide")
	for _, slug := range []string{"dmarc-live", "dns-checklist"} {
		if r := get(t, e.Env, spSite+slug); r.Status != http.StatusGone {
			t.Errorf("/%s with the website off: %d, want 410", slug, r.Status)
		}
	}
}

// Members and Paid pages through the website's gate: a visitor gets the
// sign-in card, a free member reads Members pages, a paid member Paid ones
// (of the tiers the page names, or a tier ranked above).
func TestCustomPageVisibility(t *testing.T) {
	e := spDemo(t)
	d := e.newPage("Members briefing", spPlain("Members briefing", "The members-only line."))
	path := spSite + "members-briefing"
	setVis := func(vis string, tiers ...string) {
		e.settings(d.ID, func(in *pages.PageSettingsInput) { in.Visibility, in.Tiers = vis, tiers })
		e.publish(d.ID)
	}
	reader := func(key string) resp {
		if key == "" {
			return get(t, e.Env, path)
		}
		return do(t, e.Env, http.MethodGet, path, nil, e.member(key))
	}
	const line = "The members-only line."

	setVis(pages.VisibilityMembers)
	v := reader("")
	must(t, v, http.StatusOK, "This page is for members", "Sign in with the email you subscribe with to read it.", `<meta name="robots" content="noindex, follow">`, "Sign in")
	lacks(t, v, line)
	if cc := v.Header.Get("Cache-Control"); !strings.Contains(cc, "no-store") {
		t.Errorf("a gated page is cached: %q", cc)
	}
	must(t, reader("alex"), http.StatusOK, line)
	must(t, reader("sam"), http.StatusOK, line)

	setVis(pages.VisibilityPaid)
	v = reader("")
	must(t, v, http.StatusOK, "This page is for paid members", "Upgrade")
	lacks(t, v, line)
	alex := reader("alex")
	must(t, alex, http.StatusOK, "This page is for paid members")
	lacks(t, alex, line)
	must(t, reader("sam"), http.StatusOK, line)

	founding := one[uuid.UUID](t, e.Env, `SELECT id FROM earn_tiers WHERE workspace_id = $1 AND slug = 'founding-member'`, e.ws.ID).String()
	insider := one[uuid.UUID](t, e.Env, `SELECT id FROM earn_tiers WHERE workspace_id = $1 AND slug = 'insider'`, e.ws.ID).String()
	setVis(pages.VisibilityPaid, founding)
	sam := reader("sam")
	must(t, sam, http.StatusOK, "This page is for paid members")
	lacks(t, sam, line)
	setVis(pages.VisibilityPaid, insider)
	must(t, reader("sam"), http.StatusOK, line)

	// With sign-in off nobody reads them on the web.
	e.exec(`UPDATE website_settings SET signin_on = false WHERE workspace_id = $1`, e.ws.ID)
	sam = reader("sam")
	must(t, sam, http.StatusOK, "Sign-in is turned off on this site")
	lacks(t, sam, line)
}

// A countdown is drawn on the server: before its end it shows the time
// left and keeps back what it reveals; after it, what it hides goes and
// what it reveals shows (the Launch Kit's Buy button in place of its
// waitlist since 6 Oct).
func TestCustomPageCountdowns(t *testing.T) {
	e := spDemo(t)
	lk := get(t, e.Env, spSite+"launch-kit")
	must(t, lk, http.StatusOK, "Open now", "Twelve templates and a checklist to get your first issues out the door. Pay what you want.",
		"Pay what you want", `action="`+e.Server.URL+spSite+`buy/`+e.productID("newsletter-launch-kit")+`"`)
	// The "before" eyebrow ("Opens 6 October, 09:00 New York time" in the
	// sample, anchored to today) and lead, the waitlist and the clock are gone.
	lacks(t, lk, "New York time", "when it opens", "Join the waitlist", `role="timer"`)

	clock := func(at time.Time, end string) string {
		return spSection("before", `<p>Coming soon to the shop.</p>`) + spSection("after", `<p>On sale now in the shop.</p>`) +
			spSection("b-clock", `<rc-countdown data-pb-id="countdown" at="`+at.UTC().Format("2006-01-02T15:04")+`" tz="UTC" units="days,hours,minutes" end="`+end+
				`" reveal="after" hide="before" label="Opens" message="Doors closed"></rc-countdown>`)
	}
	d := e.newPage("Doors open", clock(time.Now().Add(72*time.Hour), "reveal"))
	e.publish(d.ID)
	before := get(t, e.Env, spSite+"doors-open")
	must(t, before, http.StatusOK, "Coming soon to the shop.", `role="timer"`, "Opens in 2 days")
	lacks(t, before, "On sale now in the shop.")

	e.save(d.ID, clock(time.Now().Add(-time.Hour), "reveal"))
	e.publish(d.ID)
	after := get(t, e.Env, spSite+"doors-open")
	must(t, after, http.StatusOK, "On sale now in the shop.")
	lacks(t, after, "Coming soon to the shop.", `role="timer"`, "Doors closed")

	e.save(d.ID, clock(time.Now().Add(-time.Hour), "message"))
	e.publish(d.ID)
	must(t, get(t, e.Env, spSite+"doors-open"), http.StatusOK, "Doors closed")
}

// Views are counted on the server once per visitor, page and day; bots,
// HEAD requests, prefetches and the builder's previews aren't views.
func TestCustomPageViews(t *testing.T) {
	e := spDemo(t)
	about := e.pageID("about")
	start := e.views(about)
	numbers := e.detail(about).Numbers.Views
	step := func(name string, want int64, r resp) {
		t.Helper()
		if r.Status != http.StatusOK {
			t.Fatalf("%s: %d", name, r.Status)
		}
		if got := e.views(about) - start; got != want {
			t.Fatalf("after %s: %d views, want %d", name, got, want)
		}
	}
	step("a reader", 1, get(t, e.Env, spSite+"about"))
	step("the same reader again", 1, get(t, e.Env, spSite+"about"))
	step("another browser", 2, do(t, e.Env, http.MethodGet, spSite+"about", nil, hdr{"User-Agent": "Mozilla/5.0 (X11; Linux x86_64; rv:131.0) Gecko/20100101 Firefox/131.0"}))
	step("a crawler", 2, do(t, e.Env, http.MethodGet, spSite+"about", nil, hdr{"User-Agent": "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"}))
	step("a link preview", 2, do(t, e.Env, http.MethodGet, spSite+"about", nil, hdr{"User-Agent": "Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)"}))
	step("HEAD", 2, do(t, e.Env, http.MethodHead, spSite+"about", nil, hdr{"User-Agent": "Mozilla/5.0 (Windows NT 10.0) Chrome/130.0 Safari/537.36"}))
	step("a prefetch", 2, do(t, e.Env, http.MethodGet, spSite+"about", nil, hdr{"User-Agent": "Mozilla/5.0 (Windows NT 10.0) Chrome/130.0 Safari/537.36", "Sec-Purpose": "prefetch"}))
	pv := e.api.Do(http.MethodPost, wsAPI+"/pages/"+about+"/preview", map[string]any{})
	pv.MustStatus(t, http.StatusOK)
	if got := e.views(about) - start; got != 2 {
		t.Fatalf("a preview counted: %d", got)
	}
	// The page's numbers show them.
	if d := e.detail(about); d.Numbers.Views != numbers+2 {
		t.Errorf("the page's views %d, want %d", d.Numbers.Views, numbers+2)
	}
}
