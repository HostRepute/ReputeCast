package pages_test

// Custom pages through the API (design expansion phase 7, X19), on a clone
// of the seeded demo: Acme Newsletter's six pages (the Field Guide's sales
// page with draft changes, the DNS checklist, the Launch Kit, the DMARC
// live session, About and Daniel Park's draft) and its team (Linh Nguyen
// Owner, Marcus Chen Admin, Sofia Rossi Editor, Daniel Park Content Writer,
// Aisha Bello Viewer).
//
// Nothing leaves the process: the link check answers from linkProbe, the
// System mail goes to a fake.

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/pages"
	"github.com/hostrepute/reputecast/internal/store"
)

const pagesAPI = "/workspaces/acme-newsletter/pages"

// fixture is one test's demo clone with Linh Nguyen (Owner) signed in.
type fixture struct {
	t     *testing.T
	env   *apptest.Env
	ws    store.Workspace
	owner *apptest.Client
	links *linkProbe
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	f := &fixture{t: t, env: env, links: &linkProbe{seen: map[string]int{}}}
	env.App.Mail = &nullMail{}
	svc := env.App.Pages()
	svc.Probe = f.links.probe
	t.Cleanup(func() { svc.Now = nil })
	ws, err := env.App.Q.GetWorkspaceBySlug(context.Background(), "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	f.ws = ws
	f.owner = f.as("linh@acme.io")
	return f
}

// linkProbe answers the publish check's link probe without the network:
// the demo's addresses as demo.PagesLinkStatus says, hosts with "broken"
// in them 404, anything else 200. It counts the probes per address.
type linkProbe struct {
	mu   sync.Mutex
	seen map[string]int
}

func (p *linkProbe) probe(_ context.Context, u string) (int, error) {
	p.mu.Lock()
	p.seen[u]++
	p.mu.Unlock()
	if st, ok := demo.PagesLinkStatus(u); ok {
		return st, nil
	}
	if strings.Contains(u, "broken") {
		return http.StatusNotFound, nil
	}
	return http.StatusOK, nil
}

func (p *linkProbe) count(u string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.seen[u]
}

// nullMail swallows System-stream mail (double opt-in confirmations).
type nullMail struct{}

func (nullMail) SendSystem(context.Context, mailer.Message) (string, error) { return "test", nil }
func (nullMail) SystemAvailable(context.Context, *uuid.UUID) bool           { return true }

// as signs a client in as a demo user.
func (f *fixture) as(email string) *apptest.Client {
	f.t.Helper()
	c := f.env.Client()
	c.SignInDemo(email)
	return c
}

// clock sets the pages service's clock.
func (f *fixture) clock(at time.Time) {
	f.env.App.Pages().Now = func() time.Time { return at }
}

func (f *fixture) exec(sql string, args ...any) {
	f.t.Helper()
	if _, err := f.env.App.Pool.Exec(context.Background(), sql, args...); err != nil {
		f.t.Fatalf("%s: %v", sql, err)
	}
}

func scan[T any](f *fixture, sql string, args ...any) T {
	f.t.Helper()
	var v T
	if err := f.env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		f.t.Fatalf("%s: %v", sql, err)
	}
	return v
}

// pageID is the id of the page at a draft address.
func (f *fixture) pageID(slug string) string {
	f.t.Helper()
	return scan[uuid.UUID](f, `SELECT id FROM pages WHERE workspace_id = $1 AND slug = $2 AND deleted_at IS NULL`, f.ws.ID, slug).String()
}

// liveHTML is what readers see of a page (the demo anchors its dates to
// today: the webinar 14 days ahead, the Launch Kit's opening 9 days ago).
func (f *fixture) liveHTML(slug string) string {
	f.t.Helper()
	return scan[string](f, `SELECT v.html FROM pages p JOIN page_versions v ON v.page_id = p.id AND v.version = p.live_version
		WHERE p.workspace_id = $1 AND p.slug = $2 AND p.deleted_at IS NULL`, f.ws.ID, slug)
}

// formID is a demo form by its public id ("frm_e7b302").
func (f *fixture) formID(public string) string {
	f.t.Helper()
	return scan[uuid.UUID](f, `SELECT id FROM forms WHERE workspace_id = $1 AND public_id = $2`, f.ws.ID, public).String()
}

// productID is a demo product by its slug.
func (f *fixture) productID(slug string) string {
	f.t.Helper()
	return scan[uuid.UUID](f, `SELECT id FROM products WHERE workspace_id = $1 AND slug = $2 AND deleted_at IS NULL`, f.ws.ID, slug).String()
}

// audits counts a page's audit entries of an action.
func (f *fixture) audits(action, target string) int {
	f.t.Helper()
	return int(scan[int64](f, `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = $2 AND target_id = $3`, f.ws.ID, action, target))
}

func (f *fixture) detail(c *apptest.Client, id string) pages.PageDetail {
	f.t.Helper()
	var d pages.PageDetail
	c.Do(http.MethodGet, pagesAPI+"/"+id, nil).MustStatus(f.t, http.StatusOK).JSON(f.t, &d)
	return d
}

func (f *fixture) create(c *apptest.Client, template, name, slug string) pages.PageDetail {
	f.t.Helper()
	body := map[string]any{"template": template, "name": name}
	if slug != "" {
		body["slug"] = slug
	}
	var d pages.PageDetail
	c.Do(http.MethodPost, pagesAPI, body).MustStatus(f.t, http.StatusCreated).JSON(f.t, &d)
	return d
}

// save saves a draft's blocks (Save, not the autosave).
func (f *fixture) save(c *apptest.Client, id string, base int32, html string) pages.PageSaved {
	f.t.Helper()
	var s pages.PageSaved
	c.Do(http.MethodPut, pagesAPI+"/"+id+"/content", map[string]any{"html": html, "baseVersion": base, "manual": true}).MustStatus(f.t, http.StatusOK).JSON(f.t, &s)
	return s
}

// page is a blank page whose draft is html.
func (f *fixture) page(name, html string) pages.PageDetail {
	f.t.Helper()
	d := f.create(f.owner, pages.TemplateBlank, name, "")
	f.save(f.owner, d.ID, d.Draft.Version, html)
	return f.detail(f.owner, d.ID)
}

// settingsOf is the Page tab as it is.
func settingsOf(d pages.PageDetail) pages.PageSettingsInput {
	return pages.PageSettingsInput{Name: d.Name, Slug: d.Slug, SEOTitle: d.Settings.SEOTitle, SEODescription: d.Settings.SEODescription,
		SocialImage: d.Settings.SocialImage, Visibility: d.Settings.Visibility, Tiers: d.Settings.Tiers, HideHeader: d.Settings.HideHeader,
		ShowInNav: d.Settings.ShowInNav, NavLabel: d.Settings.NavLabel}
}

func (f *fixture) settings(c *apptest.Client, id string, in pages.PageSettingsInput) *apptest.Response {
	f.t.Helper()
	return c.Do(http.MethodPut, pagesAPI+"/"+id+"/settings", in)
}

func (f *fixture) publishNow(c *apptest.Client, id string) pages.PageDetail {
	f.t.Helper()
	var d pages.PageDetail
	c.Do(http.MethodPost, pagesAPI+"/"+id+"/publish", map[string]any{"when": "now"}).MustStatus(f.t, http.StatusOK).JSON(f.t, &d)
	return d
}

// resolve is what the public site finds at an address.
func (f *fixture) resolve(slug string) pages.Resolved {
	f.t.Helper()
	res, err := f.env.App.Pages().Resolve(context.Background(), &f.ws, slug)
	if err != nil {
		f.t.Fatalf("resolve /%s: %v", slug, err)
	}
	return res
}

// problem is an RFC 9457 answer.
type problem struct {
	Status     int            `json:"status"`
	Code       string         `json:"code"`
	Detail     string         `json:"detail"`
	Permission string         `json:"permission"`
	Helpers    []string       `json:"helpers"`
	Meta       map[string]any `json:"meta"`
	Errors     []struct {
		Location string `json:"location"`
		Message  string `json:"message"`
	} `json:"errors"`
}

// mustProblem fails unless r is an error with status and code ("" for any).
func mustProblem(t *testing.T, r *apptest.Response, status int, code string) problem {
	t.Helper()
	if r.Status != status {
		t.Fatalf("status %d, want %d: %s", r.Status, status, r.Body)
	}
	var p problem
	r.JSON(t, &p)
	if code != "" && p.Code != code {
		t.Fatalf("code %q, want %q: %s", p.Code, code, r.Body)
	}
	return p
}

// denied fails unless r is a 403 for perm.
func denied(t *testing.T, r *apptest.Response, perm string) problem {
	t.Helper()
	p := mustProblem(t, r, http.StatusForbidden, "permission_denied")
	if p.Permission != perm {
		t.Fatalf("denied for %q, want %q: %+v", p.Permission, perm, p)
	}
	return p
}

// checksOf are a page's checks of one key.
func checksOf(cs []pages.PageCheck, key string) []pages.PageCheck {
	var out []pages.PageCheck
	for _, c := range cs {
		if c.Key == key {
			out = append(out, c)
		}
	}
	return out
}

// check is the only check of a key.
func check(t *testing.T, cs []pages.PageCheck, key string) pages.PageCheck {
	t.Helper()
	got := checksOf(cs, key)
	if len(got) != 1 {
		t.Fatalf("%d %q checks in %+v", len(got), key, cs)
	}
	return got[0]
}

// blocking are the keys of the blocking checks.
func blocking(cs []pages.PageCheck) []string {
	out := []string{}
	for _, c := range cs {
		if c.Status == pages.CheckBlock && !slices.Contains(out, c.Key) {
			out = append(out, c.Key)
		}
	}
	return out
}

// section is a top-level text block.
func section(id, inner string) string {
	return `<section data-pb="text" data-pb-id="` + id + `" data-pb-name="Text" class="sp-sec"><div class="sp-in nar">` + inner + `</div></section>`
}

// plain is a page with a heading and a line.
func plain(heading, line string) string {
	return section("b-text", `<div class="prose"><h1>`+heading+`</h1><p>`+line+`</p></div>`)
}
