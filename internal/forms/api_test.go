package forms_test

// The Forms & public pages API end to end on the demo data: the builder,
// the page settings, Turnstile keys, the logo, the custom domain, the
// archive tab and the previews, with permissions and audit entries.

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/forms"
)

const api = "/workspaces/acme-newsletter"

type formJSON struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Kind           string   `json:"kind"`
	Status         string   `json:"status"`
	Slug           string   `json:"slug"`
	PublicID       string   `json:"publicId"`
	HostedPage     bool     `json:"hostedPage"`
	HostedURL      string   `json:"hostedUrl"`
	ListIDs        []string `json:"listIds"`
	DefaultListIDs []string `json:"defaultListIds"`
	ListChoice     bool     `json:"listChoice"`
	DoubleOptIn    bool     `json:"doubleOptIn"`
	ConsentText    string   `json:"consentText"`
	ConsentVersion int      `json:"consentVersion"`
	AllowedOrigins []string `json:"allowedOrigins"`
	AfterSubmit    string   `json:"afterSubmit"`
	RedirectURL    string   `json:"redirectUrl"`
	PlainHTML      string   `json:"plainHtml"`
	Fields         []struct {
		Key      string `json:"key"`
		Label    string `json:"label"`
		Required bool   `json:"required"`
	} `json:"fields"`
	Spam    map[string]any `json:"spam"`
	Content map[string]any `json:"content"`
	URLs    struct {
		Hosted   string `json:"hosted"`
		Script   string `json:"script"`
		Submit   string `json:"submit"`
		ScriptKB int    `json:"scriptKb"`
	} `json:"urls"`
	Stats struct {
		Days           int     `json:"days"`
		Views          int64   `json:"views"`
		Submissions    int64   `json:"submissions"`
		SubmissionRate float64 `json:"submissionRate"`
		Confirmed      int64   `json:"confirmed"`
		ConfirmedRate  float64 `json:"confirmedRate"`
		Blocked        int64   `json:"blocked"`
		Pending        int64   `json:"pending"`
	} `json:"stats"`
	BlockedOrigins []struct {
		Origin string `json:"origin"`
		Count  int64  `json:"count"`
	} `json:"blockedOrigins"`
}

type overviewJSON struct {
	Forms []struct {
		ID, Name, Kind, Status, PublicID, HostedURL string
		HostedPage                                  bool
	} `json:"forms"`
	Lists []struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Subscribed int64  `json:"subscribed"`
	} `json:"lists"`
	HostedURL      string `json:"hostedUrl"`
	HostedFormID   string `json:"hostedFormId"`
	SystemProvider string `json:"systemProvider"`
	Branding       struct {
		PublisherName string `json:"publisherName"`
		Accent        string `json:"accent"`
		Logo          *struct {
			Name, URL string
		} `json:"logo"`
	} `json:"branding"`
	Domain struct {
		Domain, Status string
	} `json:"domain"`
	Turnstile struct {
		SiteKey   string `json:"siteKey"`
		HasSecret bool   `json:"hasSecret"`
		Status    string `json:"status"`
	} `json:"turnstile"`
}

func signedIn(env *apptest.Env, email string) *apptest.Client {
	c := env.Client()
	c.SignInDemo(email)
	return c
}

func overview(t *testing.T, c *apptest.Client) overviewJSON {
	t.Helper()
	var o overviewJSON
	c.Do("GET", api+"/forms", nil).MustStatus(t, 200).JSON(t, &o)
	return o
}

func formByName(t *testing.T, o overviewJSON, name string) string {
	t.Helper()
	for _, f := range o.Forms {
		if f.Name == name {
			return f.ID
		}
	}
	t.Fatalf("no form %q in %+v", name, o.Forms)
	return ""
}

func getForm(t *testing.T, c *apptest.Client, id string) formJSON {
	t.Helper()
	var f formJSON
	c.Do("GET", api+"/forms/"+id, nil).MustStatus(t, 200).JSON(t, &f)
	return f
}

// inputOf is the update body that saves f unchanged.
func inputOf(f formJSON) map[string]any {
	fields := []map[string]any{}
	for _, x := range f.Fields {
		fields = append(fields, map[string]any{"key": x.Key, "label": x.Label, "required": x.Required})
	}
	return map[string]any{
		"name": f.Name, "kind": f.Kind, "fields": fields, "listIds": f.ListIDs, "defaultListIds": f.DefaultListIDs, "listChoice": f.ListChoice,
		"doubleOptIn": f.DoubleOptIn, "consentText": f.ConsentText, "spam": f.Spam, "afterSubmit": f.AfterSubmit, "redirectUrl": f.RedirectURL,
		"allowedOrigins": f.AllowedOrigins, "content": f.Content, "publish": true,
	}
}

func audits(t *testing.T, env *apptest.Env, action string) int {
	t.Helper()
	var n int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE action = $1`, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestFormsBuilderAPI(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := signedIn(env, "linh@acme.io")

	o := overview(t, c)
	if len(o.Forms) != 3 || o.Forms[0].Name != "Newsletter signup" || o.Forms[1].Name != "Footer mini form" || o.Forms[2].Name != "Beta apply" {
		t.Fatalf("forms: %+v", o.Forms)
	}
	if o.Branding.PublisherName != "Acme" || o.Branding.Accent != "#4338CA" || o.Branding.Logo == nil || o.Branding.Logo.Name != "acme-mark.svg" {
		t.Fatalf("branding: %+v", o.Branding)
	}
	if o.Domain.Domain != "news.acme.io" || o.Domain.Status != "verified" || !strings.HasSuffix(o.HostedURL, "news.acme.io/subscribe") {
		t.Fatalf("domain %+v, hosted %s", o.Domain, o.HostedURL)
	}
	if !o.Turnstile.HasSecret || o.Turnstile.Status != "ok" || o.Turnstile.SiteKey == "" {
		t.Fatalf("turnstile: %+v", o.Turnstile)
	}
	signup := formByName(t, o, "Newsletter signup")
	if o.HostedFormID != signup {
		t.Fatalf("the subscribe page is backed by Newsletter signup, got %s", o.HostedFormID)
	}

	// The builder's numbers (board FormBuilder: 14,208 · 812 · 5.7% · 731 · 90% · 1,106 · 81 pending).
	f := getForm(t, c, signup)
	s := f.Stats
	if s.Days != 30 || s.Views != 14208 || s.Submissions != 812 || s.Confirmed != 731 || s.Blocked != 1106 || s.Pending != 81 {
		t.Fatalf("stats: %+v", s)
	}
	if s.SubmissionRate != 5.7 || s.ConfirmedRate != 90 {
		t.Fatalf("rates: %v %v", s.SubmissionRate, s.ConfirmedRate)
	}
	if f.PublicID != "frm_8c2e1a" || !strings.HasSuffix(f.URLs.Script, "/p/acme-newsletter/forms/frm_8c2e1a.js") || f.URLs.ScriptKB < 2 || f.URLs.ScriptKB > 12 {
		t.Fatalf("embed: %s %+v", f.PublicID, f.URLs)
	}
	if !strings.Contains(f.PlainHTML, `/p/acme-newsletter/subscribe/newsletter-signup" method="post"`) || len(f.BlockedOrigins) != 0 {
		t.Fatalf("plain HTML or blocked origins: %s %+v", f.PlainHTML, f.BlockedOrigins)
	}
	if !f.HostedPage || f.ConsentVersion != 3 || len(f.AllowedOrigins) != 3 {
		t.Fatalf("form: %+v", f)
	}

	// Saving checks the lists and starts a new consent version when the text changes.
	in := inputOf(f)
	in["listIds"] = []string{}
	p := c.Do("PUT", api+"/forms/"+signup, in).MustStatus(t, 422).Problem(t)
	if !strings.Contains(string(mustJSON(t, p)), "body.listIds") {
		t.Fatalf("missing lists: %v", p)
	}
	in = inputOf(f)
	in["consentText"] = "I agree to receive emails from Acme. Unsubscribe with one click from any email. See the privacy policy."
	in["allowedOrigins"] = []string{"https://Docs.Acme.io/path", "acme.io"}
	var saved formJSON
	c.Do("PUT", api+"/forms/"+signup, in).MustStatus(t, 200).JSON(t, &saved)
	if saved.ConsentVersion != 4 || saved.Status != "live" || strings.Join(saved.AllowedOrigins, ",") != "docs.acme.io,acme.io" {
		t.Fatalf("saved: version %d status %s origins %v", saved.ConsentVersion, saved.Status, saved.AllowedOrigins)
	}
	if audits(t, env, "form.update") != 1 {
		t.Fatal("form.update audit entry missing")
	}
	in["redirectUrl"], in["afterSubmit"] = "javascript:alert(1)", "redirect"
	c.Do("PUT", api+"/forms/"+signup, in).MustStatus(t, 422)

	// Create, rename, pause, delete.
	var created formJSON
	c.Do("POST", api+"/forms", map[string]any{"name": "Pricing page", "kind": "popup"}).MustStatus(t, 201).JSON(t, &created)
	if created.Status != "draft" || created.Slug != "pricing-page" || !strings.HasPrefix(created.PublicID, "frm_") || created.ConsentVersion != 1 || !created.DoubleOptIn {
		t.Fatalf("created: %+v", created)
	}
	c.Do("POST", api+"/forms/"+created.ID+"/rename", map[string]any{"name": "Pricing popup"}).MustStatus(t, 200).JSON(t, &created)
	if created.Slug != "pricing-popup" {
		t.Fatalf("renamed slug %s", created.Slug)
	}
	footer := formByName(t, o, "Footer mini form")
	var paused formJSON
	c.Do("POST", api+"/forms/"+footer+"/status", map[string]any{"status": "paused"}).MustStatus(t, 200).JSON(t, &paused)
	if paused.Status != "paused" {
		t.Fatalf("paused: %s", paused.Status)
	}
	if r := c.Do("GET", "/p/acme-newsletter/subscribe/footer-mini-form", nil); r.Status != 410 || !strings.Contains(string(r.Body), "This form is closed") {
		t.Fatalf("paused form's page: %d", r.Status)
	}
	c.Do("DELETE", api+"/forms/"+created.ID, nil).MustStatus(t, 204)
	c.Do("GET", api+"/forms/"+created.ID, nil).MustStatus(t, 404)
	for _, a := range []string{"form.create", "form.rename", "form.pause", "form.delete"} {
		if audits(t, env, a) != 1 {
			t.Errorf("audit entry %s missing", a)
		}
	}

	// A site rejected today: allow it, or keep it blocked.
	ctx := context.Background()
	day := time.Now().In(mustLoc(t, "America/New_York"))
	today := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	for _, origin := range []string{"staging.acme.io", "evil.example"} {
		if _, err := env.App.Pool.Exec(ctx, `INSERT INTO form_rejections (form_id, day, reason, origin, count, last_at) VALUES ($1,$2,'origin',$3,41,now())`, signup, today, origin); err != nil {
			t.Fatal(err)
		}
	}
	f = getForm(t, c, signup)
	if len(f.BlockedOrigins) != 2 || f.BlockedOrigins[0].Count != 41 {
		t.Fatalf("blocked origins: %+v", f.BlockedOrigins)
	}
	c.Do("POST", api+"/forms/"+signup+"/allowed-origins", map[string]any{"origin": "staging.acme.io"}).MustStatus(t, 200).JSON(t, &f)
	if !contains(f.AllowedOrigins, "staging.acme.io") || len(f.BlockedOrigins) != 1 {
		t.Fatalf("allowed: %v, blocked %+v", f.AllowedOrigins, f.BlockedOrigins)
	}
	c.Do("POST", api+"/forms/"+signup+"/blocked-origins/keep", map[string]any{"origin": "evil.example"}).MustStatus(t, 200).JSON(t, &f)
	if len(f.BlockedOrigins) != 0 || contains(f.AllowedOrigins, "evil.example") {
		t.Fatalf("kept blocked: %v, alerts %+v", f.AllowedOrigins, f.BlockedOrigins)
	}
	if audits(t, env, "form.allow_origin") != 1 || audits(t, env, "form.keep_blocked") != 1 {
		t.Fatal("origin audit entries missing")
	}
}

func TestFormsPermissions(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	signup := formByName(t, overview(t, signedIn(env, "linh@acme.io")), "Newsletter signup")
	for _, email := range []string{"aisha@acme.io", "daniel@acme.io"} {
		c := signedIn(env, email)
		// Viewers and Content Writers see the forms and pages...
		o := overview(t, c)
		f := getForm(t, c, signup)
		c.Do("GET", api+"/public-pages", nil).MustStatus(t, 200)
		c.Do("GET", api+"/public-pages/preview/subscribe", nil).MustStatus(t, 200)
		// ...but can't change them.
		c.Do("PUT", api+"/forms/"+signup, inputOf(f)).MustStatus(t, 403)
		c.Do("POST", api+"/forms", map[string]any{"name": "Nope", "kind": "embedded"}).MustStatus(t, 403)
		c.Do("DELETE", api+"/forms/"+o.Forms[1].ID, nil).MustStatus(t, 403)
		c.Do("PUT", api+"/public-pages/branding", map[string]any{"publisherName": "X", "accent": "#4338CA", "background": "#F5F3FF", "font": "plex", "poweredBy": true}).MustStatus(t, 403)
		c.Do("PUT", api+"/public-pages/pages", map[string]any{"subscribe": map[string]any{"heading": "X", "description": "", "footnote": "", "listText": map[string]string{}, "index": true}}).MustStatus(t, 403)
		c.Do("PUT", api+"/public-pages/turnstile", map[string]any{"siteKey": "0x4AAAAAAAx"}).MustStatus(t, 403)
	}
	// People outside the workspace see nothing.
	outsider := env.CreateUser(apptest.UserOpts{Email: "outsider@example.com", Name: "Outsider"})
	o := env.Client()
	o.SignIn(outsider.Email, "")
	if r := o.Do("GET", api+"/forms", nil); r.Status != 403 && r.Status != 404 {
		t.Fatalf("outsider: %d", r.Status)
	}
}

func TestPublicPagesSettingsAPI(t *testing.T) {
	// Turnstile's siteverify answers from a fake: the secret below is
	// rejected, every other secret accepted.
	verify := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		w.Header().Set("Content-Type", "application/json")
		if r.PostForm.Get("secret") == "0xBAD" {
			_, _ = w.Write([]byte(`{"success":false,"error-codes":["invalid-input-secret"]}`))
			return
		}
		_, _ = w.Write([]byte(`{"success":false,"error-codes":["invalid-input-response"]}`))
	}))
	defer verify.Close()
	prev := forms.SiteverifyURL
	forms.SiteverifyURL = verify.URL
	defer func() { forms.SiteverifyURL = prev }()

	env := apptest.New(t)
	env.SeedDemo()
	c := signedIn(env, "linh@acme.io")

	// The Turnstile secret is write-only.
	r := c.Do("GET", api+"/public-pages", nil).MustStatus(t, 200)
	if strings.Contains(string(r.Body), "demo-secret-not-real") || strings.Contains(strings.ToLower(string(r.Body)), `"secret"`) {
		t.Fatalf("the Turnstile secret leaked: %s", r.Body)
	}
	var tsOut struct {
		Turnstile struct {
			Status    string `json:"status"`
			Error     string `json:"error"`
			HasSecret bool   `json:"hasSecret"`
		} `json:"turnstile"`
	}
	c.Do("PUT", api+"/public-pages/turnstile", map[string]any{"siteKey": "0x4AAAAAAAnewkey", "secret": "0xBAD"}).MustStatus(t, 200).JSON(t, &tsOut)
	if tsOut.Turnstile.Status != "error" || !strings.Contains(tsOut.Turnstile.Error, "invalid-input-secret") {
		t.Fatalf("rejected secret: %+v", tsOut.Turnstile)
	}
	r = c.Do("PUT", api+"/public-pages/turnstile", map[string]any{"siteKey": "0x4AAAAAAAnewkey", "secret": "0xGOOD-secret"}).MustStatus(t, 200)
	r.JSON(t, &tsOut)
	if tsOut.Turnstile.Status != "ok" || !tsOut.Turnstile.HasSecret || strings.Contains(string(r.Body), "0xGOOD-secret") {
		t.Fatalf("accepted secret: %+v", tsOut.Turnstile)
	}
	// An empty secret keeps the stored one.
	c.Do("PUT", api+"/public-pages/turnstile", map[string]any{"siteKey": "0x4AAAAAAAother"}).MustStatus(t, 200).JSON(t, &tsOut)
	if tsOut.Turnstile.Status != "ok" {
		t.Fatalf("kept secret: %+v", tsOut.Turnstile)
	}
	var stored []byte
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT turnstile_secret_enc FROM public_page_settings ps JOIN workspaces w ON w.id = ps.workspace_id WHERE w.slug = 'acme-newsletter'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if len(stored) == 0 || bytes.Contains(stored, []byte("0xGOOD-secret")) {
		t.Fatal("the secret must be stored sealed")
	}
	if audits(t, env, "public_pages.turnstile") != 3 {
		t.Fatalf("turnstile audit entries: %d", audits(t, env, "public_pages.turnstile"))
	}

	// Branding: colours are checked, contrast is kept.
	c.Do("PUT", api+"/public-pages/branding", map[string]any{"publisherName": "Acme", "accent": "#43ZZCA", "background": "#F5F3FF", "font": "plex", "poweredBy": true}).MustStatus(t, 422)
	c.Do("PUT", api+"/public-pages/branding", map[string]any{"publisherName": "Acme", "accent": "#4338CA", "background": "#111111", "font": "plex", "poweredBy": true}).MustStatus(t, 422)
	var pages struct {
		Branding struct {
			Accent, OnAccent, AccentText, Background string
			PoweredBy                                bool
			Notes                                    []string
		} `json:"branding"`
		Pages struct {
			Subscribe struct {
				Heading string `json:"heading"`
			} `json:"subscribe"`
		} `json:"pages"`
		URLs struct {
			Subscribe, Archive, Feed string
		} `json:"urls"`
	}
	c.Do("PUT", api+"/public-pages/branding", map[string]any{"publisherName": "Acme Labs", "accent": "#FDE047", "background": "#FFFFFF", "font": "system", "poweredBy": false,
		"privacyUrl": "https://acme.io/privacy"}).MustStatus(t, 200).JSON(t, &pages)
	if pages.Branding.OnAccent == "#FFFFFF" || pages.Branding.AccentText == "#FDE047" || pages.Branding.PoweredBy || len(pages.Branding.Notes) == 0 {
		t.Fatalf("a light accent gets dark button text and darker links: %+v", pages.Branding)
	}
	if audits(t, env, "public_pages.branding") != 1 {
		t.Fatal("branding audit entry missing")
	}
	// The public page follows: no "Sent with ReputeCast", the new name.
	page := c.Do("GET", "/p/acme-newsletter/subscribe", nil).MustStatus(t, 200)
	if strings.Contains(string(page.Body), "Sent with ReputeCast") || !strings.Contains(string(page.Body), "Acme Labs") {
		t.Fatal("the subscribe page didn't follow the branding")
	}

	// Page options.
	c.Do("PUT", api+"/public-pages/pages", map[string]any{"subscribe": map[string]any{"heading": "  The Acme Labs letter ", "description": "", "footnote": "", "listText": map[string]string{"not-a-list": "x"}, "index": false}}).
		MustStatus(t, 200).JSON(t, &pages)
	if pages.Pages.Subscribe.Heading != "The Acme Labs letter" {
		t.Fatalf("heading: %q", pages.Pages.Subscribe.Heading)
	}
	if audits(t, env, "public_pages.update") != 1 {
		t.Fatal("pages audit entry missing")
	}
	c.Do("PUT", api+"/public-pages/pages", map[string]any{"previewEmail": "not an address"}).MustStatus(t, 422)

	// Logo: images only, never scripts.
	upload := func(name, data string) *apptest.Response {
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		part, _ := mw.CreateFormFile("file", name)
		_, _ = part.Write([]byte(data))
		_ = mw.Close()
		return c.DoRaw("PUT", api+"/public-pages/logo", mw.FormDataContentType(), &body)
	}
	upload("mark.svg", `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10"/></svg>`).MustStatus(t, 200)
	upload("evil.svg", `<svg xmlns="http://www.w3.org/2000/svg"><rect onclick = "alert(1)"/></svg>`).MustStatus(t, 422)
	upload("notes.txt", `hello`).MustStatus(t, 422)
	logo := c.Do("GET", "/p/acme-newsletter/logo?v=1", nil).MustStatus(t, 200)
	if !strings.Contains(logo.Header.Get("Content-Security-Policy"), "sandbox") || logo.Header.Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("logo headers: %v", logo.Header)
	}
	c.Do("DELETE", api+"/public-pages/logo", nil).MustStatus(t, 200)

	// Custom domain: not this server's own host, not another workspace's.
	c.Do("PUT", api+"/public-pages/domain", map[string]any{"domain": "localhost"}).MustStatus(t, 422)
	c.Do("PUT", api+"/public-pages/domain", map[string]any{"domain": "news acme io"}).MustStatus(t, 422)
	// A pasted URL is read as its host.
	var dom struct {
		Domain struct{ Domain string } `json:"domain"`
	}
	c.Do("PUT", api+"/public-pages/domain", map[string]any{"domain": "https://News.Acme.io/path"}).MustStatus(t, 200).JSON(t, &dom)
	if dom.Domain.Domain != "news.acme.io" {
		t.Fatalf("normalized domain: %q", dom.Domain.Domain)
	}
	eng := env.Client()
	eng.SignInDemo("linh@acme.io")
	eng.Do("PUT", "/workspaces/acme-engineering-blog/public-pages/domain", map[string]any{"domain": "news.acme.io"}).MustStatus(t, 409)

	// The unsubscribe numbers.
	var st struct {
		Days         int   `json:"days"`
		Unsubscribed int64 `json:"unsubscribed"`
	}
	c.Do("GET", api+"/public-pages/unsubscribe/stats", nil).MustStatus(t, 200).JSON(t, &st)
	if st.Days != 30 {
		t.Fatalf("unsubscribe stats: %+v", st)
	}
}

func TestPublicPagePreviewAPI(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := signedIn(env, "linh@acme.io")
	type preview struct {
		HTML    string `json:"html"`
		URL     string `json:"url"`
		Contact string `json:"contact"`
	}
	get := func(page string, q url.Values) preview {
		t.Helper()
		var p preview
		path := api + "/public-pages/preview/" + page
		if len(q) > 0 {
			path += "?" + q.Encode()
		}
		c.Do("GET", path, nil).MustStatus(t, 200).JSON(t, &p)
		if !strings.Contains(p.HTML, `class="preview" inert`) {
			t.Errorf("%s preview isn't inert", page)
		}
		return p
	}
	p := get("subscribe", nil)
	if !strings.Contains(p.HTML, "The Acme newsletter") || !strings.Contains(p.HTML, "31,988 subscribers") || !strings.HasSuffix(p.URL, "news.acme.io/subscribe") {
		t.Fatalf("subscribe preview: %s", p.URL)
	}
	draft, _ := json.Marshal(map[string]any{"subscribe": map[string]any{"heading": "Unsaved heading", "description": "", "footnote": "", "listText": map[string]string{}, "index": true}})
	if p = get("subscribe", url.Values{"draft": {string(draft)}}); !strings.Contains(p.HTML, "Unsaved heading") {
		t.Fatal("the draft heading isn't previewed")
	}
	// Nothing was saved.
	var pages struct {
		Pages struct {
			Subscribe struct{ Heading string } `json:"subscribe"`
		} `json:"pages"`
	}
	c.Do("GET", api+"/public-pages", nil).MustStatus(t, 200).JSON(t, &pages)
	if pages.Pages.Subscribe.Heading != "The Acme newsletter" {
		t.Fatalf("a preview saved the draft: %q", pages.Pages.Subscribe.Heading)
	}
	// The token pages are shown for the "preview as" subscriber, Sam Rivera,
	// without a working link: Viewers see previews too, and a signed link
	// would let them change Sam's lists.
	signed := regexp.MustCompile(`/(?:preferences|unsubscribe|confirm|view)/([A-Za-z0-9_-]+)`)
	for _, page := range []string{"preferences", "unsubscribe", "confirm"} {
		if p = get(page, nil); p.Contact != "sam.rivera@outlook.com" || !strings.Contains(p.HTML, "sam.rivera@outlook.com") {
			t.Errorf("%s preview for %s", page, p.Contact)
		}
		links := signed.FindAllStringSubmatch(p.HTML, -1)
		if len(links) == 0 {
			t.Errorf("%s preview has no preference or unsubscribe link", page)
		}
		for _, m := range links {
			if m[1] != "preview" {
				t.Errorf("%s preview carries a working link: %s", page, m[0])
			}
		}
	}
	if p = get("archive", nil); !strings.Contains(p.HTML, "44 issues") {
		t.Fatal("archive preview")
	}
	c.Do("GET", api+"/public-pages/preview/preferences?contact=nobody@example.com", nil).MustStatus(t, 422)
	c.Do("GET", api+"/public-pages/preview/nope", nil).MustStatus(t, 422)
}

func TestArchiveIssuesAPI(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := signedIn(env, "linh@acme.io")
	type issue struct {
		CampaignID, Slug, Title, ListID, Label string
		Hidden                                 bool
	}
	var page struct {
		Issues    []issue `json:"issues"`
		Total     int64   `json:"total"`
		Published int64   `json:"published"`
		PageSize  int     `json:"pageSize"`
	}
	c.Do("GET", api+"/public-pages/archive/issues", nil).MustStatus(t, 200).JSON(t, &page)
	if page.Total != 44 || page.Published != 44 || page.PageSize != 10 || len(page.Issues) != 10 {
		t.Fatalf("archive: total %d published %d page %d (%d)", page.Total, page.Published, page.PageSize, len(page.Issues))
	}
	first := page.Issues[0]
	if first.Title != "What shipped in October, and what we're retiring" {
		t.Fatalf("newest first: %+v", first)
	}
	c.Do("GET", api+"/public-pages/archive/issues?q=boring+infrastructure", nil).MustStatus(t, 200).JSON(t, &page)
	if page.Total != 1 || page.Issues[0].Label != "#141" {
		t.Fatalf("search: %+v", page.Issues)
	}
	digest := page.Issues[0]
	body := map[string]any{"hidden": true, "title": "", "excerpt": "", "label": digest.Label, "listId": digest.ListID}
	signedIn(env, "aisha@acme.io").Do("PUT", api+"/public-pages/archive/issues/"+digest.CampaignID, body).MustStatus(t, 403)
	var hidden issue
	c.Do("PUT", api+"/public-pages/archive/issues/"+digest.CampaignID, body).MustStatus(t, 200).JSON(t, &hidden)
	if !hidden.Hidden || hidden.Title == "" {
		t.Fatalf("hidden: %+v", hidden)
	}
	if audits(t, env, "archive.hide") != 1 {
		t.Fatal("archive.hide audit entry missing")
	}
	// Readers no longer find it; the tab still lists it.
	if r := c.Do("GET", "/p/acme-newsletter/archive/"+digest.Slug, nil); r.Status != 404 {
		t.Fatalf("hidden issue page: %d", r.Status)
	}
	if r := c.Do("GET", "/p/acme-newsletter/archive", nil).MustStatus(t, 200); !strings.Contains(string(r.Body), "43 issues") {
		t.Fatal("the archive still counts the hidden issue")
	}
	c.Do("GET", api+"/public-pages/archive/issues", nil).MustStatus(t, 200).JSON(t, &page)
	if page.Total != 44 || page.Published != 43 {
		t.Fatalf("tab: total %d published %d", page.Total, page.Published)
	}
	body["listId"] = uuid.NewString()
	c.Do("PUT", api+"/public-pages/archive/issues/"+digest.CampaignID, body).MustStatus(t, 422)
	c.Do("PUT", api+"/public-pages/archive/issues/"+uuid.NewString(), map[string]any{"hidden": false, "title": "", "excerpt": "", "label": "", "listId": digest.ListID}).MustStatus(t, 404)
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
