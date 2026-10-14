package publicpages_test

// The public pages over HTTP, as subscribers' browsers and mail apps reach
// them: no session cookie, the instance's own origin (App.URLs follows the
// test server's address, env.Server.URL) or a third-party site.
//
// Every submission of the demo forms counts against the per-address limit
// (5 a minute per workspace, all tests come from 127.0.0.1), so each test
// makes at most five and uses its own clone of the demo.

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mailer"
)

const (
	wsSlug    = "acme-newsletter"
	wsAPI     = "/workspaces/acme-newsletter"
	browserUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15"
	// The consent sentence of the demo forms (version 3 on Newsletter signup).
	consentText = "I agree to receive emails from Acme. Unsubscribe any time with one click. See the privacy policy."
)

// demoIDs are the demo rows the tests use.
type demoIDs struct {
	ws, eng                   uuid.UUID // Acme Newsletter, Acme Engineering Blog
	digest, updates, beta     uuid.UUID
	sam                       uuid.UUID
	signup, footer, betaApply uuid.UUID // the three forms
}

func demo(t *testing.T) (*apptest.Env, demoIDs) {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	var ids demoIDs
	ids.ws = one[uuid.UUID](t, env, `SELECT id FROM workspaces WHERE slug = 'acme-newsletter'`)
	ids.eng = one[uuid.UUID](t, env, `SELECT id FROM workspaces WHERE slug = 'acme-engineering-blog'`)
	list := func(name string) uuid.UUID {
		return one[uuid.UUID](t, env, `SELECT id FROM lists WHERE workspace_id = $1 AND name = $2 AND deleted_at IS NULL`, ids.ws, name)
	}
	ids.digest, ids.updates, ids.beta = list("Weekly Digest"), list("Product Updates"), list("Beta Program")
	ids.sam = one[uuid.UUID](t, env, `SELECT id FROM contacts WHERE workspace_id = $1 AND email = 'sam.rivera@outlook.com'`, ids.ws)
	form := func(publicID string) uuid.UUID {
		return one[uuid.UUID](t, env, `SELECT id FROM forms WHERE workspace_id = $1 AND public_id = $2`, ids.ws, publicID)
	}
	ids.signup, ids.footer, ids.betaApply = form("frm_8c2e1a"), form("frm_3f91d0"), form("frm_b7a412")
	return env, ids
}

// one runs a query that returns one value.
func one[T any](t *testing.T, env *apptest.Env, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

func count(t *testing.T, env *apptest.Env, sql string, args ...any) int {
	t.Helper()
	return int(one[int64](t, env, sql, args...))
}

// rawClient sends no cookies and doesn't follow redirects.
var rawClient = &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

type resp struct {
	Status int
	Header http.Header
	Body   string
}

// Text is the body with HTML entities decoded ("You&#39;re in" reads "You're in").
func (r resp) Text() string { return html.UnescapeString(r.Body) }

// hdr are request headers; "Host" sets the request's host.
type hdr map[string]string

// do sends a request to a public path. A non-nil form is sent url-encoded.
func do(t *testing.T, env *apptest.Env, method, path string, form url.Values, h hdr) resp {
	t.Helper()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(context.Background(), method, env.Server.URL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", browserUA)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for k, v := range h {
		if k == "Host" {
			req.Host = v
			continue
		}
		req.Header.Set(k, v)
	}
	res, err := rawClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = res.Body.Close() }()
	b, _ := io.ReadAll(res.Body)
	return resp{Status: res.StatusCode, Header: res.Header, Body: string(b)}
}

func get(t *testing.T, env *apptest.Env, path string) resp {
	t.Helper()
	return do(t, env, http.MethodGet, path, nil, nil)
}

// must fails unless r has status and every text in want.
func must(t *testing.T, r resp, status int, want ...string) {
	t.Helper()
	if r.Status != status {
		t.Fatalf("status %d, want %d:\n%s", r.Status, status, clip(r.Body))
	}
	text := r.Text()
	for _, w := range want {
		if !strings.Contains(text, w) {
			t.Errorf("page lacks %q:\n%s", w, clip(r.Body))
		}
	}
}

func clip(s string) string {
	if len(s) > 3000 {
		return s[:3000] + "…"
	}
	return s
}

func lacks(t *testing.T, r resp, texts ...string) {
	t.Helper()
	for _, s := range texts {
		if strings.Contains(r.Text(), s) {
			t.Errorf("page shows %q", s)
		}
	}
}

var hiddenToken = regexp.MustCompile(`name="_t" value="([^"]*)"`)

func pageToken(t *testing.T, r resp) string {
	t.Helper()
	m := hiddenToken.FindStringSubmatch(r.Body)
	if m == nil || m[1] == "" {
		t.Fatalf("no _t token on the page:\n%s", clip(r.Body))
	}
	return m[1]
}

// formToken is a timing token issued age ago for a form ("frm_8c2e1a").
func formToken(env *apptest.Env, key string, age time.Duration) string {
	return (&forms.Service{Box: env.App.Secrets}).FormToken(key, time.Now().Add(-age))
}

func sign(env *apptest.Env, kind linktoken.Kind, ws, contact, message uuid.UUID) string {
	return env.App.Links.Sign(linktoken.Claims{Kind: kind, WorkspaceID: ws, ContactID: contact, MessageID: message})
}

func jsonBody(t *testing.T, r resp) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(r.Body), &m); err != nil {
		t.Fatalf("not JSON (%d): %v\n%s", r.Status, err, clip(r.Body))
	}
	return m
}

// owner is a signed-in Owner for API reads.
func owner(env *apptest.Env) *apptest.Client {
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	return c
}

type formStats struct {
	Views, Submissions, Confirmed, Blocked, Pending int64
}

func stats(t *testing.T, env *apptest.Env, form uuid.UUID) formStats {
	t.Helper()
	var f struct{ Stats formStats }
	owner(env).Do(http.MethodGet, wsAPI+"/forms/"+form.String(), nil).MustStatus(t, 200).JSON(t, &f)
	return f.Stats
}

func membership(t *testing.T, env *apptest.Env, contact, list uuid.UUID) string {
	t.Helper()
	var st string
	err := env.App.Pool.QueryRow(context.Background(), `SELECT status FROM list_memberships WHERE contact_id = $1 AND list_id = $2`, contact, list).Scan(&st)
	if err != nil {
		return "none"
	}
	return st
}

func contactID(t *testing.T, env *apptest.Env, ws uuid.UUID, email string) (uuid.UUID, bool) {
	t.Helper()
	var id uuid.UUID
	err := env.App.Pool.QueryRow(context.Background(), `SELECT id FROM contacts WHERE workspace_id = $1 AND email = $2`, ws, email).Scan(&id)
	return id, err == nil
}

func TestHostedSubscribePage(t *testing.T) {
	env, ids := demo(t)
	before := stats(t, env, ids.signup)
	if before.Views != 14208 || before.Submissions != 812 || before.Confirmed != 731 || before.Blocked != 1106 || before.Pending != 81 {
		t.Fatalf("demo numbers %+v, want the board's 14,208 / 812 / 731 / 1,106 / 81", before)
	}

	page := get(t, env, "/p/acme-newsletter/subscribe")
	t.Run("page", func(t *testing.T) {
		must(t, page, 200, "The Acme newsletter", "Every Tuesday morning.", "Only when something ships. About twice a month.",
			"<title>Subscribe to the Acme newsletter</title>", `<meta name="robots" content="index, follow">`,
			`<link rel="canonical" href="http://news.acme.io/subscribe">`,
			`<form class="form" method="post" action="`+env.Server.URL+`/p/acme-newsletter/subscribe"`,
			`See the <a href="https://acme.io/privacy">privacy policy</a>.`,
			`name="website"`, `value="`+ids.digest.String()+`" checked>`, `value="`+ids.updates.String()+`">`)
		lacks(t, page, "{subscribers}", "Beta Program")
		if strings.Contains(page.Body, "<script") || strings.Contains(page.Body, " style=") || strings.Contains(page.Body, "<style") {
			t.Error("the page has inline script or style; its CSP allows neither")
		}
		if ct := page.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("content type %q", ct)
		}
		csp := page.Header.Get("Content-Security-Policy")
		for _, want := range []string{"default-src 'none'", "script-src 'self';", "style-src 'self';", "form-action 'self'",
			"frame-ancestors 'self' https://acme.io https://docs.acme.io https://blog.acme.io"} {
			if !strings.Contains(csp, want) {
				t.Errorf("CSP lacks %q: %s", want, csp)
			}
		}
		if strings.Contains(csp, "unsafe-inline") || strings.Contains(csp, "unsafe-eval") {
			t.Errorf("CSP allows inline code: %s", csp)
		}
		if page.Header.Get("X-Frame-Options") != "" {
			t.Error("X-Frame-Options DENY would stop the allowed sites from framing the page")
		}
		if page.Header.Get("Set-Cookie") != "" {
			t.Error("the public page sets a cookie")
		}
	})

	email := "jordan.lee@fastmail.com"
	var jordan uuid.UUID
	t.Run("submitting with the page's token after 3 seconds", func(t *testing.T) {
		tok := pageToken(t, page)
		time.Sleep(3200 * time.Millisecond) // the form's minimum time to fill it in
		r := do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe", url.Values{
			"_t": {tok}, "email": {"  Jordan.Lee@Fastmail.com "}, "first_name": {"Jordan"}, "list": {ids.digest.String()}, "consent": {"1"}, "website": {""},
		}, hdr{"Origin": env.Server.URL})
		must(t, r, 200, "Check your inbox", "We sent a confirmation link to <strong>"+email+"</strong>", "until then you won't hear from us.")
		if r.Header.Get("Cache-Control") != "no-store" || !strings.Contains(r.Header.Get("X-Robots-Tag"), "noindex") {
			t.Errorf("the result page is private: %v", r.Header)
		}
		var ok bool
		if jordan, ok = contactID(t, env, ids.ws, email); !ok {
			t.Fatal("no contact for the new address")
		}
		if st := membership(t, env, jordan, ids.digest); st != "pending" {
			t.Errorf("Weekly Digest is %s, want pending until the confirmation click", st)
		}
		if st := membership(t, env, jordan, ids.updates); st != "none" {
			t.Errorf("Product Updates wasn't picked but is %s", st)
		}
		if c := one[string](t, env, `SELECT consent FROM contacts WHERE id = $1`, jordan); c != "pending" {
			t.Errorf("contact consent %q, want pending", c)
		}
		var text, version, source, ip, ua, note string
		err := env.App.Pool.QueryRow(context.Background(), `SELECT consent_text, consent_version, source, coalesce(host(ip), ''), user_agent, note
			FROM consent_events WHERE contact_id = $1 AND action = 'subscribed' AND list_id = $2`, jordan, ids.digest).Scan(&text, &version, &source, &ip, &ua, &note)
		if err != nil {
			t.Fatalf("no consent event for the signup: %v", err)
		}
		if text != consentText || version != "v3" || source != "form:newsletter-signup" || ip != "127.0.0.1" || ua != browserUA {
			t.Errorf("consent evidence: text %q version %q source %q ip %q ua %q", text, version, source, ip, ua)
		}
		if note == "" {
			t.Error("the pending consent event says it waits for the click")
		}
		if n := count(t, env, `SELECT count(*) FROM form_submissions WHERE contact_id = $1 AND form_id = $2 AND double_opt_in AND confirmed_at IS NULL`, jordan, ids.signup); n != 1 {
			t.Errorf("%d waiting submissions, want 1", n)
		}
		after := stats(t, env, ids.signup)
		if after.Views != before.Views+1 || after.Submissions != before.Submissions+1 || after.Pending != before.Pending+1 || after.Blocked != before.Blocked {
			t.Errorf("numbers after one view and one signup: %+v (before %+v)", after, before)
		}
	})
	if jordan == uuid.Nil {
		t.Fatal("no signup to confirm")
	}

	confirmPath := "/p/acme-newsletter/confirm/" + sign(env, linktoken.Confirm, ids.ws, jordan, uuid.Nil)
	t.Run("link scanners' HEAD never confirms", func(t *testing.T) {
		r := do(t, env, http.MethodHead, confirmPath, nil, nil)
		if r.Status != 200 || r.Body != "" {
			t.Errorf("HEAD: %d %q", r.Status, r.Body)
		}
		if st := membership(t, env, jordan, ids.digest); st != "pending" {
			t.Fatalf("a HEAD request confirmed the subscription (%s)", st)
		}
	})
	t.Run("the confirmation link", func(t *testing.T) {
		r := get(t, env, confirmPath)
		must(t, r, 200, "You're in, Jordan", "Your subscription for <strong>"+email+"</strong> is confirmed",
			"Weekly Digest", "Tuesdays at 09:00 your time.", "Product Updates", "if you'd like release news too.",
			"from 127.0.0.1 · kept as proof of consent", "Email preferences")
		if r.Header.Get("Cache-Control") != "no-store" || r.Header.Get("Referrer-Policy") != "no-referrer" {
			t.Errorf("signed-link pages are private: %v", r.Header)
		}
		if st := membership(t, env, jordan, ids.digest); st != "subscribed" {
			t.Errorf("Weekly Digest is %s after the click", st)
		}
		if c := one[string](t, env, `SELECT consent FROM contacts WHERE id = $1`, jordan); c != "double" {
			t.Errorf("consent %q, want double", c)
		}
		var text, version, source string
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT consent_text, consent_version, source FROM consent_events
			WHERE contact_id = $1 AND action = 'confirmed' AND list_id = $2`, jordan, ids.digest).Scan(&text, &version, &source); err != nil {
			t.Fatalf("no confirmation event: %v", err)
		}
		if text != consentText || version != "v3" || source != "confirm_link" {
			t.Errorf("the confirmation carries the evidence: %q %q %q", text, version, source)
		}
		if n := count(t, env, `SELECT count(*) FROM form_submissions WHERE contact_id = $1 AND confirmed_at IS NOT NULL`, jordan); n != 1 {
			t.Errorf("%d confirmed submissions, want 1", n)
		}
		s := stats(t, env, ids.signup)
		if s.Confirmed != before.Confirmed+1 || s.Pending != before.Pending {
			t.Errorf("confirmed %d pending %d after the click (before %+v)", s.Confirmed, s.Pending, before)
		}
		// A second click shows the page again and changes nothing.
		must(t, get(t, env, confirmPath), 200, "You're in")
		if n := count(t, env, `SELECT count(*) FROM consent_events WHERE contact_id = $1 AND action = 'confirmed'`, jordan); n != 1 {
			t.Errorf("%d confirmation events after two clicks, want 1", n)
		}
	})
	t.Run("broken and foreign links", func(t *testing.T) {
		for name, tok := range map[string]string{
			"preferences token":   sign(env, linktoken.Preferences, ids.ws, jordan, uuid.Nil),
			"other workspace":     sign(env, linktoken.Confirm, ids.eng, jordan, uuid.Nil),
			"unknown contact":     sign(env, linktoken.Confirm, ids.ws, uuid.Must(uuid.NewV7()), uuid.Nil),
			"cut short":           strings.TrimPrefix(confirmPath, "/p/acme-newsletter/confirm/")[:20],
			"not a token at all":  "hello",
			"tampered with a bit": strings.TrimPrefix(confirmPath, "/p/acme-newsletter/confirm/")[:40] + "AAAA",
		} {
			r := get(t, env, "/p/acme-newsletter/confirm/"+tok)
			if r.Status != 404 || !strings.Contains(r.Text(), "This link doesn't work") || strings.Contains(r.Text(), email) {
				t.Errorf("%s: %d\n%s", name, r.Status, clip(r.Body))
			}
		}
		if r := get(t, env, "/p/no-such-workspace/subscribe"); r.Status != 404 {
			t.Errorf("unknown workspace: %d", r.Status)
		}
	})
}

func TestSubscribeSpamChecks(t *testing.T) {
	env, ids := demo(t)
	before := stats(t, env, ids.signup)
	valid := func(email string) url.Values {
		return url.Values{"_t": {formToken(env, "frm_8c2e1a", time.Minute)}, "email": {email}, "list": {ids.digest.String()}, "consent": {"1"}}
	}
	t.Run("a filled honeypot looks like success but saves nothing", func(t *testing.T) {
		f := valid("bot.one@fastmail.com")
		f.Set("website", "https://cheap-pills.example")
		must(t, do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe", f, hdr{"Origin": env.Server.URL}), 200, "Check your inbox")
		if _, ok := contactID(t, env, ids.ws, "bot.one@fastmail.com"); ok {
			t.Error("the honeypot submission created a contact")
		}
		if n := count(t, env, `SELECT coalesce(sum(count), 0)::bigint FROM form_rejections WHERE form_id = $1 AND reason = 'honeypot'`, ids.signup); n != 1 {
			t.Errorf("%d honeypot rejections recorded, want 1", n)
		}
	})
	t.Run("another site's origin is refused", func(t *testing.T) {
		r := do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe", valid("evil.one@fastmail.com"), hdr{"Origin": "https://evil.example"})
		must(t, r, 403, "This form can't be sent from here", "This form can't be used on this site.")
		if _, ok := contactID(t, env, ids.ws, "evil.one@fastmail.com"); ok {
			t.Error("a cross-site submission created a contact")
		}
		if n := count(t, env, `SELECT coalesce(sum(count), 0)::bigint FROM form_rejections WHERE form_id = $1 AND reason = 'origin' AND origin = 'evil.example'`, ids.signup); n != 1 {
			t.Errorf("%d origin rejections for evil.example, want 1", n)
		}
	})
	t.Run("a post without Origin or Referer is refused", func(t *testing.T) {
		must(t, do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe", valid("script.one@fastmail.com"), nil), 403)
	})
	t.Run("the Referer stands in for a missing Origin", func(t *testing.T) {
		r := do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe", valid("ref.one@fastmail.com"), hdr{"Referer": env.Server.URL + "/p/acme-newsletter/subscribe"})
		must(t, r, 200, "Check your inbox")
	})
	t.Run("too fast goes to a Turnstile check", func(t *testing.T) {
		f := valid("quick.one@fastmail.com")
		f.Set("_t", formToken(env, "frm_8c2e1a", 0))
		r := do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe", f, hdr{"Origin": env.Server.URL})
		must(t, r, 200, "One more step", `name="challenge"`, `data-sitekey="0x4AAAAAAADnPIDROrmt1Wwj"`,
			`<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" defer></script>`)
		csp := r.Header.Get("Content-Security-Policy")
		if !strings.Contains(csp, "script-src 'self' https://challenges.cloudflare.com") || !strings.Contains(csp, "frame-src https://challenges.cloudflare.com") {
			t.Errorf("the challenge page's CSP: %s", csp)
		}
		if _, ok := contactID(t, env, ids.ws, "quick.one@fastmail.com"); ok {
			t.Error("a held submission created a contact before the check")
		}
		held := one[[]byte](t, env, `SELECT payload_enc FROM form_challenges WHERE workspace_id = $1`, ids.ws)
		if strings.Contains(string(held), "quick.one") {
			t.Error("the held submission is stored in the clear")
		}
		// The check page without an answer asks again and calls nobody.
		id := regexp.MustCompile(`name="challenge" value="([^"]+)"`).FindStringSubmatch(r.Body)
		if id == nil {
			t.Fatal("no challenge id on the page")
		}
		must(t, do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe/challenge", url.Values{"challenge": {id[1]}}, nil), 200,
			"Finish the check above, then press Continue.")
	})
	t.Run("the sixth try in a minute is rate limited", func(t *testing.T) {
		r := do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe", valid("six@fastmail.com"), hdr{"Origin": env.Server.URL})
		must(t, r, 429, "Too many tries")
		if r.Header.Get("Retry-After") != "60" {
			t.Errorf("Retry-After %q", r.Header.Get("Retry-After"))
		}
	})
	after := stats(t, env, ids.signup)
	// Spam (honeypot, rate limit) counts as blocked; a refused origin is an alert, not spam.
	if after.Submissions != before.Submissions+1 || after.Blocked != before.Blocked+2 {
		t.Errorf("numbers %+v, want one more submission and two more blocked than %+v", after, before)
	}
}

func TestSubscribeValidation(t *testing.T) {
	env, ids := demo(t)
	post := func(f url.Values) resp {
		f.Set("_t", formToken(env, "frm_8c2e1a", time.Minute))
		return do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe", f, hdr{"Origin": env.Server.URL})
	}
	t.Run("missing list and consent keep what was typed", func(t *testing.T) {
		r := post(url.Values{"email": {"casey.kim@fastmail.com"}, "first_name": {"Casey <b>"}})
		must(t, r, 422, "Pick at least one newsletter.", "Tick the box to agree to receive emails.", `value="casey.kim@fastmail.com"`)
		if !strings.Contains(r.Body, `value="Casey &lt;b&gt;"`) {
			t.Error("the typed name isn't escaped back into the form")
		}
		// Each error is tied to its control.
		for _, want := range []string{`aria-invalid="true" aria-describedby="consent-err"`, `id="consent-err" role="alert"`,
			`<fieldset class="lists" aria-describedby="lists-err">`, `id="lists-err" role="alert"`} {
			if !strings.Contains(r.Body, want) {
				t.Errorf("the page lacks %s", want)
			}
		}
		if r.Header.Get("Cache-Control") != "no-store" {
			t.Error("a page with typed values is cached")
		}
	})
	t.Run("disposable and role addresses", func(t *testing.T) {
		must(t, post(url.Values{"email": {"temp@mailinator.com"}, "list": {ids.digest.String()}, "consent": {"1"}}), 422, "Use a permanent address")
		must(t, post(url.Values{"email": {"info@northwind.co"}, "list": {ids.digest.String()}, "consent": {"1"}}), 422, "Use a personal address")
	})
	t.Run("an address that isn't one", func(t *testing.T) {
		must(t, post(url.Values{"email": {"not-an-address"}, "list": {ids.digest.String()}, "consent": {"1"}}), 422,
			"That doesn't look like a complete email address.")
	})
	t.Run("the archive's box fills the address in", func(t *testing.T) {
		r := do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe", url.Values{"step": {"prefill"}, "email": {" riley@fastmail.com "}}, hdr{"Origin": env.Server.URL})
		must(t, r, 200, `value="riley@fastmail.com"`, `value="`+ids.digest.String()+`" checked>`)
		if _, ok := contactID(t, env, ids.ws, "riley@fastmail.com"); ok {
			t.Error("prefilling subscribed the address")
		}
	})
	if n := count(t, env, `SELECT count(*) FROM contacts WHERE workspace_id = $1 AND email IN ('casey.kim@fastmail.com', 'temp@mailinator.com', 'info@northwind.co')`, ids.ws); n != 0 {
		t.Errorf("%d invalid submissions saved", n)
	}
	t.Run("a draft form has no public page", func(t *testing.T) {
		if _, err := env.App.Pool.Exec(context.Background(), `UPDATE forms SET status = 'draft' WHERE id = $1`, ids.betaApply); err != nil {
			t.Fatal(err)
		}
		must(t, get(t, env, "/p/acme-newsletter/subscribe/beta-apply"), 404)
		must(t, get(t, env, "/p/acme-newsletter/forms/frm_b7a412.js"), 404, "console.warn")
	})
	t.Run("a paused form is closed", func(t *testing.T) {
		if _, err := env.App.Pool.Exec(context.Background(), `UPDATE forms SET status = 'paused' WHERE id = $1`, ids.signup); err != nil {
			t.Fatal(err)
		}
		must(t, get(t, env, "/p/acme-newsletter/subscribe"), 410, "This form is closed", "Read past issues")
		must(t, post(url.Values{"email": {"late@fastmail.com"}, "list": {ids.digest.String()}, "consent": {"1"}}), 410, "This form is closed")
	})
}

func TestEmbeddedForm(t *testing.T) {
	env, ids := demo(t)
	submit := "/p/acme-newsletter/forms/frm_8c2e1a/submit"
	jsonHdr := func(origin string) hdr { return hdr{"Origin": origin, "Accept": "application/json"} }
	fields := func(email, tok string) url.Values {
		return url.Values{"_t": {tok}, "email": {email}, "list": {ids.digest.String(), ids.updates.String()}, "consent": {"1"}, "_js": {"1"}}
	}
	t.Run("script", func(t *testing.T) {
		r := get(t, env, "/p/acme-newsletter/forms/frm_8c2e1a.js")
		must(t, r, 200, `"id":"frm_8c2e1a"`, `/p/acme-newsletter/forms/frm_8c2e1a/submit`, "The Acme newsletter")
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/javascript") {
			t.Errorf("content type %q", ct)
		}
		if r.Header.Get("X-Content-Type-Options") != "nosniff" || r.Header.Get("Cross-Origin-Resource-Policy") != "cross-origin" {
			t.Errorf("script headers: %v", r.Header)
		}
		if strings.Contains(r.Body, "sam.rivera") {
			t.Error("the script holds personal data")
		}
		must(t, get(t, env, "/p/acme-newsletter/forms/frm_000000.js"), 404, "console.warn")
		must(t, get(t, env, "/p/acme-newsletter/forms/frm_8c2e1a.css"), 404)
	})
	t.Run("preflight", func(t *testing.T) {
		r := do(t, env, http.MethodOptions, submit, nil, hdr{"Origin": "https://docs.acme.io", "Access-Control-Request-Method": "POST"})
		if r.Status != 204 || r.Header.Get("Access-Control-Allow-Origin") != "https://docs.acme.io" || !strings.Contains(r.Header.Get("Access-Control-Allow-Methods"), "POST") {
			t.Errorf("allowed site's preflight: %d %v", r.Status, r.Header)
		}
		r = do(t, env, http.MethodOptions, submit, nil, hdr{"Origin": "https://evil.example", "Access-Control-Request-Method": "POST"})
		if r.Status != 403 || r.Header.Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("other site's preflight: %d %v", r.Status, r.Header)
		}
	})
	var sessionToken string
	t.Run("session", func(t *testing.T) {
		r := do(t, env, http.MethodGet, "/p/acme-newsletter/forms/frm_8c2e1a/session", nil, hdr{"Origin": "https://acme.io"})
		if r.Status != 200 || r.Header.Get("Access-Control-Allow-Origin") != "https://acme.io" || !strings.Contains(r.Header.Get("Vary"), "Origin") {
			t.Fatalf("session: %d %v %s", r.Status, r.Header, r.Body)
		}
		sessionToken, _ = jsonBody(t, r)["t"].(string)
		if sessionToken == "" {
			t.Fatalf("no token: %s", r.Body)
		}
		r = do(t, env, http.MethodGet, "/p/acme-newsletter/forms/frm_8c2e1a/session", nil, hdr{"Origin": "https://evil.example"})
		if r.Status != 403 || r.Header.Get("Access-Control-Allow-Origin") != "" || jsonBody(t, r)["status"] != "rejected" {
			t.Errorf("session for another site: %d %v %s", r.Status, r.Header, r.Body)
		}
	})
	t.Run("an allowed site gets JSON it can read", func(t *testing.T) {
		r := do(t, env, http.MethodPost, submit, fields("morgan@fastmail.com", formToken(env, "frm_8c2e1a", time.Minute)), jsonHdr("https://acme.io"))
		if r.Status != 200 || jsonBody(t, r)["status"] != "confirm" {
			t.Fatalf("submit: %d %s", r.Status, r.Body)
		}
		if r.Header.Get("Access-Control-Allow-Origin") != "https://acme.io" || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			t.Errorf("headers: %v", r.Header)
		}
		id, ok := contactID(t, env, ids.ws, "morgan@fastmail.com")
		if !ok {
			t.Fatal("no contact")
		}
		if membership(t, env, id, ids.digest) != "pending" || membership(t, env, id, ids.updates) != "pending" {
			t.Error("both picked lists wait for the confirmation")
		}
		if src := one[string](t, env, `SELECT source FROM consent_events WHERE contact_id = $1 AND action = 'subscribed' LIMIT 1`, id); src != "form:newsletter-signup" {
			t.Errorf("consent source %q", src)
		}
	})
	t.Run("another site gets a 403 it can't read", func(t *testing.T) {
		r := do(t, env, http.MethodPost, submit, fields("evil.two@fastmail.com", formToken(env, "frm_8c2e1a", time.Minute)), jsonHdr("https://evil.example"))
		if r.Status != 403 || jsonBody(t, r)["status"] != "rejected" || r.Header.Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("submit from evil.example: %d %v %s", r.Status, r.Header, r.Body)
		}
		if _, ok := contactID(t, env, ids.ws, "evil.two@fastmail.com"); ok {
			t.Error("contact created")
		}
	})
	t.Run("a fresh session token asks for the Turnstile check", func(t *testing.T) {
		if sessionToken == "" {
			t.Skip("no session token")
		}
		// Ask for a new one: the check needs a token under three seconds old.
		s := do(t, env, http.MethodGet, "/p/acme-newsletter/forms/frm_8c2e1a/session", nil, hdr{"Origin": "https://blog.acme.io"})
		fresh, _ := jsonBody(t, s)["t"].(string)
		r := do(t, env, http.MethodPost, submit, fields("speedy@fastmail.com", fresh), jsonHdr("https://blog.acme.io"))
		m := jsonBody(t, r)
		u, _ := m["challengeUrl"].(string)
		if r.Status != 200 || m["status"] != "challenge" || !strings.HasPrefix(u, env.Server.URL+"/p/acme-newsletter/subscribe/challenge?c=") {
			t.Errorf("challenge: %d %s", r.Status, r.Body)
		}
	})
	t.Run("field errors come back as JSON", func(t *testing.T) {
		r := do(t, env, http.MethodPost, submit, url.Values{"_t": {formToken(env, "frm_8c2e1a", time.Minute)}, "email": {"nope"}, "_js": {"1"}}, jsonHdr("https://acme.io"))
		m := jsonBody(t, r)
		errs, _ := m["errors"].(map[string]any)
		if r.Status != 422 || m["status"] != "invalid" || errs["email"] == nil || errs["consent"] == nil || errs["lists"] == nil {
			t.Errorf("invalid: %d %s", r.Status, r.Body)
		}
	})
	t.Run("the plain HTML copy posts to the hosted page", func(t *testing.T) {
		f := url.Values{"_plain": {"1"}, "email": {"plain@fastmail.com"}, "list": {ids.digest.String()}, "consent": {"1"}, "website": {""}}
		r := do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe/newsletter-signup", f, hdr{"Origin": "https://blog.acme.io"})
		must(t, r, 200, "Check your inbox", "plain@fastmail.com")
	})
}

func TestPreferenceCenter(t *testing.T) {
	env, ids := demo(t)
	start := time.Now().Add(-time.Second)
	if membership(t, env, ids.sam, ids.digest) != "subscribed" || membership(t, env, ids.sam, ids.updates) != "subscribed" {
		t.Fatal("Sam Rivera is on Weekly Digest and Product Updates in the demo")
	}
	tok := sign(env, linktoken.Preferences, ids.ws, ids.sam, uuid.Nil)
	path := "/p/acme-newsletter/preferences/" + tok
	checked := func(r resp, id uuid.UUID) bool {
		return strings.Contains(r.Body, `name="list" value="`+id.String()+`" checked>`)
	}
	t.Run("page", func(t *testing.T) {
		r := get(t, env, path)
		must(t, r, 200, "Email preferences", "for sam.rivera@outlook.com", "Weekly Digest", "Product Updates", "Beta Program",
			`name="frequency" value="weekly" checked>`, "Don't pause", `name="first_name" value="Sam"`, "Unsubscribe from everything")
		if !checked(r, ids.digest) || !checked(r, ids.updates) || checked(r, ids.beta) {
			t.Error("Sam's lists are checked, the others not")
		}
		if r.Header.Get("Cache-Control") != "no-store" || !strings.Contains(r.Header.Get("X-Robots-Tag"), "noindex") || r.Header.Get("Referrer-Policy") != "no-referrer" {
			t.Errorf("private headers: %v", r.Header)
		}
	})
	t.Run("saving lists, frequency, a pause and the name", func(t *testing.T) {
		shown := strings.Join([]string{ids.digest.String(), ids.updates.String(), ids.beta.String()}, ",")
		r := do(t, env, http.MethodPost, path, url.Values{"shown": {shown}, "list": {ids.digest.String()}, "frequency": {"monthly"}, "pause": {"30"}, "first_name": {"Samuel"}}, nil)
		if r.Status != 303 || !strings.HasSuffix(r.Header.Get("Location"), "/p/acme-newsletter/preferences/"+tok+"?saved=1") {
			t.Fatalf("save: %d %v\n%s", r.Status, r.Header, clip(r.Body))
		}
		var freq string
		var until *time.Time
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT frequency, paused_until FROM contact_preferences WHERE contact_id = $1`, ids.sam).Scan(&freq, &until); err != nil {
			t.Fatal(err)
		}
		if freq != "monthly" || until == nil || until.Before(time.Now().Add(29*24*time.Hour)) || until.After(time.Now().Add(31*24*time.Hour)) {
			t.Errorf("preferences: %s until %v", freq, until)
		}
		if membership(t, env, ids.sam, ids.updates) != "unsubscribed" || membership(t, env, ids.sam, ids.digest) != "subscribed" {
			t.Error("unchecking Product Updates leaves it; Weekly Digest stays")
		}
		if name := one[string](t, env, `SELECT first_name FROM contacts WHERE id = $1`, ids.sam); name != "Samuel" {
			t.Errorf("first name %q", name)
		}
		for _, action := range []string{"frequency_changed", "paused", "unsubscribed"} {
			if n := count(t, env, `SELECT count(*) FROM consent_events WHERE contact_id = $1 AND action = $2 AND source = 'preference_center' AND host(ip) = '127.0.0.1'
				AND occurred_at >= $3`, ids.sam, action, start); n != 1 {
				t.Errorf("%d %s events from the preference center, want 1", n, action)
			}
		}
		r = get(t, env, path+"?saved=1")
		must(t, r, 200, "Saved. Changes apply to the next send.", `name="frequency" value="monthly" checked>`, "Paused until", "Resume now")
		if checked(r, ids.updates) || !checked(r, ids.digest) {
			t.Error("the page shows the saved lists")
		}
	})
	t.Run("resuming and joining again", func(t *testing.T) {
		shown := strings.Join([]string{ids.digest.String(), ids.updates.String(), ids.beta.String()}, ",")
		r := do(t, env, http.MethodPost, path, url.Values{"shown": {shown}, "list": {ids.digest.String(), ids.updates.String()}, "frequency": {"monthly"}, "pause": {"0"}}, nil)
		if r.Status != 303 {
			t.Fatalf("save: %d", r.Status)
		}
		if one[*time.Time](t, env, `SELECT paused_until FROM contact_preferences WHERE contact_id = $1`, ids.sam) != nil {
			t.Error("the pause isn't lifted")
		}
		if membership(t, env, ids.sam, ids.updates) != "subscribed" {
			t.Error("Product Updates isn't joined again")
		}
		if name := one[string](t, env, `SELECT first_name FROM contacts WHERE id = $1`, ids.sam); name != "Samuel" {
			t.Error("leaving the name field out of the post cleared the name")
		}
	})
	t.Run("an unknown frequency is refused", func(t *testing.T) {
		r := do(t, env, http.MethodPost, path, url.Values{"frequency": {"hourly"}}, nil)
		must(t, r, 422, "Pick how often you'd like emails.")
	})
	t.Run("links that don't belong here", func(t *testing.T) {
		for name, tok := range map[string]string{
			"another workspace's token": sign(env, linktoken.Preferences, ids.eng, ids.sam, uuid.Nil),
			"an unsubscribe token":      sign(env, linktoken.Unsubscribe, ids.ws, ids.sam, uuid.Nil),
			"a contact of nobody":       sign(env, linktoken.Preferences, ids.ws, uuid.Must(uuid.NewV7()), uuid.Nil),
		} {
			r := get(t, env, "/p/acme-newsletter/preferences/"+tok)
			if r.Status != 404 || strings.Contains(r.Body, "sam.rivera") {
				t.Errorf("%s: %d", name, r.Status)
			}
			if r := do(t, env, http.MethodPost, "/p/acme-newsletter/preferences/"+tok, url.Values{"frequency": {"all"}}, nil); r.Status != 404 {
				t.Errorf("%s (POST): %d", name, r.Status)
			}
		}
		if f := one[string](t, env, `SELECT frequency FROM contact_preferences WHERE contact_id = $1`, ids.sam); f != "monthly" {
			t.Errorf("a foreign link changed the preferences: %s", f)
		}
	})
}

// fakeMail is a System-stream sender that keeps what it sends.
type fakeMail struct {
	mu   sync.Mutex
	sent []mailer.Message
}

func (f *fakeMail) SendSystem(_ context.Context, m mailer.Message) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return "Postmark", nil
}

func (f *fakeMail) SystemAvailable(context.Context, *uuid.UUID) bool { return true }

func (f *fakeMail) to(addr string) []mailer.Message {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []mailer.Message
	for _, m := range f.sent {
		if m.To == addr {
			out = append(out, m)
		}
	}
	return out
}

var linkIn = regexp.MustCompile(`https?://\S+/p/acme-newsletter/(?:confirm|preferences)/[A-Za-z0-9_-]+`)

func TestPreferenceLinkRequest(t *testing.T) {
	env, _ := demo(t)
	must(t, get(t, env, "/p/acme-newsletter/preferences"), 200, "Email preferences")
	post := func(email string, extra ...string) resp {
		f := url.Values{"email": {email}}
		if len(extra) == 2 {
			f.Set(extra[0], extra[1])
		}
		return do(t, env, http.MethodPost, "/p/acme-newsletter/preferences", f, hdr{"Origin": env.Server.URL})
	}
	// No System provider yet: the same answer for every address, so it
	// says nothing about who subscribes.
	known, unknown := post("sam.rivera@outlook.com"), post("nobody.here@fastmail.com")
	if known.Status != 503 || unknown.Status != 503 || firstLine(known.Text(), "can't send") != firstLine(unknown.Text(), "can't send") {
		t.Fatalf("without System mail: %d / %d", known.Status, unknown.Status)
	}
	// With one, everyone reads "Check your inbox"; only subscribers get mail.
	mail := &fakeMail{}
	env.App.Mail = mail
	time.Sleep(61 * time.Second) // the address limit: 3 a minute
	known, unknown = post("sam.rivera@outlook.com"), post("nobody.here@fastmail.com")
	must(t, known, 200, "Check your inbox", "<strong>sam.rivera@outlook.com</strong> gets emails from Acme")
	must(t, unknown, 200, "Check your inbox", "<strong>nobody.here@fastmail.com</strong> gets emails from Acme")
	if got := mail.to("sam.rivera@outlook.com"); len(got) != 1 || !linkIn.MatchString(got[0].Text) || got[0].Kind != "preference_link" {
		t.Fatalf("Sam's link: %+v", got)
	}
	if got := mail.to("nobody.here@fastmail.com"); len(got) != 0 {
		t.Fatal("an unknown address got mail")
	}
	// A filled honeypot gets the same page and sends nothing; the fourth try in a minute is limited.
	must(t, post("sam.rivera@outlook.com", "website", "x"), 200, "Check your inbox")
	if got := mail.to("sam.rivera@outlook.com"); len(got) != 1 {
		t.Fatalf("the honeypot sent mail: %d", len(got))
	}
	must(t, post("again@fastmail.com"), 429, "Too many tries from your network.")
}

// Double opt-in end to end: the form's confirmation goes out on the System
// stream with a signed link; the link confirms the list.
func TestDoubleOptInEmail(t *testing.T) {
	env, ids := demo(t)
	mail := &fakeMail{}
	env.App.Mail = mail
	page := get(t, env, "/p/acme-newsletter/subscribe")
	tok := pageToken(t, page)
	time.Sleep(3200 * time.Millisecond)
	r := do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe", url.Values{
		"_t": {tok}, "email": {"mira.okafor@fastmail.com"}, "first_name": {"Mira"}, "list": {ids.digest.String(), ids.updates.String()}, "consent": {"1"},
	}, hdr{"Origin": env.Server.URL})
	must(t, r, 200, "Check your inbox", "<strong>mira.okafor@fastmail.com</strong>")
	sent := mail.to("mira.okafor@fastmail.com")
	if len(sent) != 1 || sent[0].Kind != "double_opt_in" || !strings.Contains(sent[0].Subject, "Confirm your subscription to Acme") {
		t.Fatalf("confirmation email: %+v", sent)
	}
	if !strings.Contains(sent[0].Text, "Weekly Digest and Product Updates") || !strings.Contains(sent[0].HTML, "Confirm my subscription") {
		t.Fatalf("confirmation copy:\n%s", sent[0].Text)
	}
	link := linkIn.FindString(sent[0].Text)
	if link == "" {
		t.Fatalf("no confirmation link in:\n%s", sent[0].Text)
	}
	mira, _ := contactID(t, env, ids.ws, "mira.okafor@fastmail.com")
	if n := count(t, env, `SELECT count(*) FROM consent_events WHERE contact_id = $1 AND action = 'optin_sent'`, mira); n != 2 {
		t.Errorf("optin_sent events: %d", n)
	}
	u, _ := url.Parse(link)
	// A link scanner's HEAD never confirms.
	if h := do(t, env, http.MethodHead, u.Path, nil, nil); h.Status != 200 || membership(t, env, mira, ids.digest) != "pending" {
		t.Fatalf("HEAD: %d, %s", h.Status, membership(t, env, mira, ids.digest))
	}
	must(t, get(t, env, u.Path), 200, "You're in, Mira", "mira.okafor@fastmail.com")
	if membership(t, env, mira, ids.digest) != "subscribed" || membership(t, env, mira, ids.updates) != "subscribed" {
		t.Fatal("the click didn't confirm both lists")
	}
	if c := one[string](t, env, `SELECT consent FROM contacts WHERE id = $1`, mira); c != "double" {
		t.Errorf("consent %q, want double", c)
	}
	var text, version string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT consent_text, consent_version FROM consent_events WHERE contact_id = $1 AND action = 'confirmed' LIMIT 1`, mira).Scan(&text, &version); err != nil || text != consentText || version != "v3" {
		t.Fatalf("the confirmation carries the consent evidence: %q %q %v", text, version, err)
	}
	if n := count(t, env, `SELECT count(*) FROM form_submissions WHERE contact_id = $1 AND confirmed_at IS NOT NULL`, mira); n != 1 {
		t.Errorf("confirmed submissions: %d", n)
	}
	// "Form submitted" automations start on the click, not on the submission.
	if n := count(t, env, `SELECT count(*) FROM automation_events WHERE contact_id = $1 AND kind = 'form_submitted' AND ref = $2`, mira, ids.signup.String()); n != 1 {
		t.Errorf("form_submitted events: %d, want 1", n)
	}
}

func firstLine(text, near string) string {
	i := strings.Index(text, near)
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(text[max(0, i-60):min(len(text), i+60)])
}

func TestUnsubscribe(t *testing.T) {
	env, ids := demo(t)
	start := time.Now().Add(-time.Second)
	tok := sign(env, linktoken.Unsubscribe, ids.ws, ids.sam, uuid.Nil)
	path := "/p/acme-newsletter/unsubscribe/" + tok
	oneClick := url.Values{"List-Unsubscribe": {"One-Click"}}

	t.Run("page", func(t *testing.T) {
		r := get(t, env, path)
		must(t, r, 200, "Unsubscribe sam.rivera@outlook.com?", "You're on Weekly Digest and Product Updates.",
			"Too many emails? Get fewer instead.", "Send me the monthly best-of only", "Why are you leaving?", "Not Sam?",
			"Unsubscribe from all Acme emails", "One-click unsubscribe from your mail app works too.")
		if r.Header.Get("Cache-Control") != "no-store" || r.Header.Get("Referrer-Policy") != "no-referrer" {
			t.Errorf("private headers: %v", r.Header)
		}
		if r := do(t, env, http.MethodHead, path, nil, nil); r.Status != 200 {
			t.Errorf("HEAD: %d", r.Status)
		}
		if membership(t, env, ids.sam, ids.digest) != "subscribed" {
			t.Fatal("viewing the page unsubscribed")
		}
	})
	t.Run("RFC 8058 one-click", func(t *testing.T) {
		r := do(t, env, http.MethodPost, path, oneClick, nil) // no cookie, no Origin, no CSRF token
		if r.Status != 200 || !strings.HasPrefix(r.Header.Get("Content-Type"), "text/plain") {
			t.Fatalf("one-click: %d %v %s", r.Status, r.Header, r.Body)
		}
		for _, l := range []uuid.UUID{ids.digest, ids.updates} {
			if st := membership(t, env, ids.sam, l); st != "unsubscribed" {
				t.Errorf("list %s is %s", l, st)
			}
		}
		if st := one[string](t, env, `SELECT status FROM contacts WHERE id = $1`, ids.sam); st != "unsubscribed" {
			t.Errorf("contact status %q", st)
		}
		var reason, scope, source string
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT reason, scope, source FROM suppressions WHERE workspace_id = $1 AND email = 'sam.rivera@outlook.com'`,
			ids.ws).Scan(&reason, &scope, &source); err != nil {
			t.Fatalf("no suppression: %v", err)
		}
		if reason != "unsubscribe" || scope != "marketing" || source != "one_click" {
			t.Errorf("suppression %s / %s / %s, want unsubscribe / marketing / one_click", reason, scope, source)
		}
		if n := count(t, env, `SELECT count(*) FROM consent_events WHERE contact_id = $1 AND action = 'unsubscribed' AND source = 'one_click' AND occurred_at >= $2`, ids.sam, start); n != 2 {
			t.Errorf("%d one-click consent events, want one per list (2)", n)
		}
	})
	t.Run("a repeated one-click is fine and changes nothing", func(t *testing.T) {
		events := count(t, env, `SELECT count(*) FROM consent_events WHERE contact_id = $1`, ids.sam)
		feedback := count(t, env, `SELECT count(*) FROM unsubscribe_feedback WHERE contact_id = $1`, ids.sam)
		if r := do(t, env, http.MethodPost, path, oneClick, nil); r.Status != 200 {
			t.Fatalf("second one-click: %d", r.Status)
		}
		if count(t, env, `SELECT count(*) FROM consent_events WHERE contact_id = $1`, ids.sam) != events ||
			count(t, env, `SELECT count(*) FROM unsubscribe_feedback WHERE contact_id = $1`, ids.sam) != feedback {
			t.Error("a repeated one-click wrote again")
		}
	})
	t.Run("the page then offers to resubscribe", func(t *testing.T) {
		r := get(t, env, path)
		must(t, r, 200, "You're unsubscribed", "won't get newsletters or automated emails from Acme any more", `name="action" value="resubscribe"`)
		r = do(t, env, http.MethodPost, path, url.Values{"action": {"resubscribe"}}, nil)
		must(t, r, 200, "Welcome back", "sam.rivera@outlook.com", "is subscribed to Acme again.")
		for _, l := range []uuid.UUID{ids.digest, ids.updates} {
			if st := membership(t, env, ids.sam, l); st != "subscribed" {
				t.Errorf("list %s is %s after resubscribing", l, st)
			}
		}
		if membership(t, env, ids.sam, ids.beta) != "none" {
			t.Error("resubscribing joined a list Sam was never on")
		}
		if n := count(t, env, `SELECT count(*) FROM suppressions WHERE workspace_id = $1 AND email = 'sam.rivera@outlook.com'`, ids.ws); n != 0 {
			t.Error("the unsubscribe suppression is still there")
		}
		if st := one[string](t, env, `SELECT status FROM contacts WHERE id = $1`, ids.sam); st != "active" {
			t.Errorf("status %q", st)
		}
	})
	t.Run("fewer emails instead", func(t *testing.T) {
		r := do(t, env, http.MethodPost, path, url.Values{"action": {"fewer"}}, nil)
		must(t, r, 200, "Done: one email a month", "gets the monthly best-of from Acme")
		if f := one[string](t, env, `SELECT frequency FROM contact_preferences WHERE contact_id = $1`, ids.sam); f != "monthly" {
			t.Errorf("frequency %q", f)
		}
		if n := count(t, env, `SELECT count(*) FROM unsubscribe_feedback WHERE contact_id = $1 AND chose_fewer AND reason = 'too_frequent' AND created_at >= $2`, ids.sam, start); n != 1 {
			t.Errorf("%d chose-fewer answers, want 1", n)
		}
		if membership(t, env, ids.sam, ids.digest) != "subscribed" {
			t.Error("choosing fewer emails unsubscribed")
		}
		lacks(t, get(t, env, path), "Too many emails? Get fewer instead.")
	})
	t.Run("leaving with a reason", func(t *testing.T) {
		r := do(t, env, http.MethodPost, path, url.Values{"action": {"unsubscribe"}, "reason": {"switching"}}, nil)
		if r.Status != 303 || !strings.HasSuffix(r.Header.Get("Location"), path) {
			t.Fatalf("unsubscribe: %d %v", r.Status, r.Header)
		}
		var reason, comment string
		if err := env.App.Pool.QueryRow(context.Background(), `SELECT reason, comment FROM unsubscribe_feedback WHERE contact_id = $1 AND NOT chose_fewer ORDER BY created_at DESC LIMIT 1`,
			ids.sam).Scan(&reason, &comment); err != nil {
			t.Fatal(err)
		}
		if reason != "other" || comment != "switching email addresses" {
			t.Errorf("feedback %q %q", reason, comment)
		}
		if n := count(t, env, `SELECT count(*) FROM consent_events WHERE contact_id = $1 AND action = 'unsubscribed' AND source = 'unsubscribe_page' AND occurred_at >= $2`, ids.sam, start); n != 2 {
			t.Errorf("%d unsubscribe-page events, want 2", n)
		}
		if src := one[string](t, env, `SELECT source FROM suppressions WHERE workspace_id = $1 AND email = 'sam.rivera@outlook.com'`, ids.ws); src != "unsubscribe_page" {
			t.Errorf("suppression source %q", src)
		}
	})
	t.Run("an email's link counts on its campaign", func(t *testing.T) {
		var msg, campaign uuid.UUID
		err := env.App.Pool.QueryRow(context.Background(), `SELECT id, campaign_id FROM messages WHERE contact_id = $1 AND campaign_id IS NOT NULL AND provider_id IS NOT NULL
			ORDER BY queued_at DESC LIMIT 1`, ids.sam).Scan(&msg, &campaign)
		if err != nil {
			t.Fatalf("Sam's latest campaign message: %v", err)
		}
		if r := do(t, env, http.MethodPost, path, url.Values{"action": {"resubscribe"}}, nil); r.Status != 200 {
			t.Fatalf("resubscribe: %d", r.Status)
		}
		before := count(t, env, `SELECT coalesce(sum(unsubscribed), 0)::bigint FROM campaign_counters WHERE campaign_id = $1`, campaign)
		mpath := "/p/acme-newsletter/unsubscribe/" + sign(env, linktoken.Unsubscribe, ids.ws, ids.sam, msg)
		if r := do(t, env, http.MethodPost, mpath, url.Values{"action": {"unsubscribe"}, "reason": {"not_relevant"}}, nil); r.Status != 303 {
			t.Fatalf("unsubscribe: %d", r.Status)
		}
		if after := count(t, env, `SELECT coalesce(sum(unsubscribed), 0)::bigint FROM campaign_counters WHERE campaign_id = $1`, campaign); after != before+1 {
			t.Errorf("campaign unsubscribes %d → %d", before, after)
		}
		if n := count(t, env, `SELECT count(*) FROM delivery_events WHERE message_id = $1 AND type = 'unsubscribe'`, msg); n < 1 {
			t.Error("no unsubscribe event on the message")
		}
		if n := count(t, env, `SELECT count(*) FROM unsubscribe_feedback WHERE contact_id = $1 AND campaign_id = $2 AND reason = 'not_relevant' AND created_at >= $3`, ids.sam, campaign, start); n != 1 {
			t.Errorf("%d answers filed under the campaign, want 1", n)
		}
	})
	t.Run("foreign tokens", func(t *testing.T) {
		for name, tok := range map[string]string{
			"another workspace": sign(env, linktoken.Unsubscribe, ids.eng, ids.sam, uuid.Nil),
			"preferences kind":  sign(env, linktoken.Preferences, ids.ws, ids.sam, uuid.Nil),
		} {
			if r := do(t, env, http.MethodPost, "/p/acme-newsletter/unsubscribe/"+tok, oneClick, nil); r.Status != 404 {
				t.Errorf("%s one-click: %d", name, r.Status)
			}
			if r := get(t, env, "/p/acme-newsletter/unsubscribe/"+tok); r.Status != 404 {
				t.Errorf("%s page: %d", name, r.Status)
			}
		}
	})
}

func TestArchive(t *testing.T) {
	env, _ := demo(t)
	t.Run("list", func(t *testing.T) {
		r := get(t, env, "/p/acme-newsletter/archive")
		must(t, r, 200, "Newsletter archive", "Every issue of the Weekly Digest and Product Updates since ", " 44 issues.",
			">All</a>", ">Weekly Digest</a>", ">Product Updates</a>", "?list=weekly-digest", "?list=product-updates",
			"1 to 10 of 44", `href="`+env.Server.URL+`/p/acme-newsletter/archive?page=2"`, "Older issues",
			"What shipped in October, and what we're retiring", `<meta name="robots" content="index, follow">`,
			`<link rel="canonical" href="http://news.acme.io/archive">`, `type="application/rss+xml"`, "Get the next one in your inbox")
		lacks(t, r, "Newer issues", "Beta Program")
		m := regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`).FindStringSubmatch(r.Body)
		if m == nil {
			t.Fatal("no JSON-LD")
		}
		var ld struct {
			Type     string `json:"@type"`
			BlogPost []struct {
				Type string `json:"@type"`
				URL  string `json:"url"`
			} `json:"blogPost"`
		}
		if err := json.Unmarshal([]byte(m[1]), &ld); err != nil || ld.Type != "Blog" || len(ld.BlogPost) != 10 || !strings.HasPrefix(ld.BlogPost[0].URL, "http://news.acme.io/archive/") {
			t.Errorf("JSON-LD %v %+v", err, ld)
		}
		if csp := r.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'none'") || strings.Contains(csp, "unsafe-inline") {
			t.Errorf("CSP %s", csp)
		}
	})
	t.Run("page 2 and past the end", func(t *testing.T) {
		r := get(t, env, "/p/acme-newsletter/archive?page=2")
		must(t, r, 200, "11 to 20 of 44", "Newer issues", "Older issues", "page 2</title>")
		r = get(t, env, "/p/acme-newsletter/archive?page=5")
		must(t, r, 200, "41 to 44 of 44")
		lacks(t, r, "Older issues")
		must(t, get(t, env, "/p/acme-newsletter/archive?page=99"), 404, "There are no older issues.")
	})
	t.Run("search", func(t *testing.T) {
		r := get(t, env, "/p/acme-newsletter/archive?q=boring")
		must(t, r, 200, "The case for boring infrastructure", `value="boring"`, `<meta name="robots" content="noindex, follow">`)
		lacks(t, r, "Shipping on Fridays, revisited", "What shipped in October")
		r = get(t, env, "/p/acme-newsletter/archive?q=%25") // a literal % matches nothing special
		must(t, r, 200)
		lacks(t, r, "1 to 10 of 44")
		r = get(t, env, "/p/acme-newsletter/archive?q=zzz-nothing")
		must(t, r, 200, "No issues match “zzz-nothing”.")
	})
	t.Run("a long search in Japanese", func(t *testing.T) {
		// 34 characters, 102 bytes: cut to 100 bytes it splits a character.
		r := get(t, env, "/p/acme-newsletter/archive?q="+url.QueryEscape(strings.Repeat("設計", 17)))
		if r.Status != 200 {
			t.Errorf("status %d, want 200 with no results (archive.go archiveParamsOf cuts ?q to 100 bytes mid-character; Postgres refuses the invalid UTF-8)", r.Status)
		}
	})
	t.Run("list filter", func(t *testing.T) {
		r := get(t, env, "/p/acme-newsletter/archive?list=product-updates")
		must(t, r, 200, "What shipped in October, and what we're retiring", `aria-current="true">Product Updates</a>`)
		lacks(t, r, "The case for boring infrastructure", "Shipping on Fridays")
		r = get(t, env, "/p/acme-newsletter/archive?list=no-such-list")
		must(t, r, 200, "1 to 10 of 44")
	})
	t.Run("issue page", func(t *testing.T) {
		r := get(t, env, "/p/acme-newsletter/archive/the-case-for-boring-infrastructure")
		must(t, r, 200, "The case for boring infrastructure", "Acme · Newsletter archive", "Weekly Digest · ", " · #141",
			`<link rel="canonical" href="http://news.acme.io/archive/the-case-for-boring-infrastructure">`,
			`<title>The case for boring infrastructure · Acme newsletter</title>`, `<meta property="og:type" content="article">`)
		if ct := r.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("content type %q", ct)
		}
		m := regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`).FindStringSubmatch(r.Body)
		if m == nil {
			t.Fatal("no JSON-LD")
		}
		var ld map[string]any
		if err := json.Unmarshal([]byte(m[1]), &ld); err != nil || ld["@type"] != "BlogPosting" || ld["headline"] != "The case for boring infrastructure" {
			t.Errorf("JSON-LD %v %v", err, ld)
		}
		csp := r.Header.Get("Content-Security-Policy")
		if !strings.Contains(csp, "script-src 'none'") || !strings.Contains(csp, "default-src 'none'") || strings.Contains(csp, "script-src 'self'") {
			t.Errorf("an issue page never runs scripts: %s", csp)
		}
		// The stored email HTML runs in an opaque origin, away from the app's cookies.
		if !strings.Contains(csp, "sandbox allow-popups") || strings.Contains(csp, "allow-scripts") || strings.Contains(csp, "allow-same-origin") {
			t.Errorf("an issue page isn't sandboxed: %s", csp)
		}
		if strings.Contains(r.Body, "{{") || strings.Contains(r.Body, "first_name |") {
			t.Error("Liquid tokens are left in the issue")
		}
		if strings.Contains(r.Body, `src="/media/`) {
			t.Error("relative media links are left in the issue")
		}
		if n := strings.Count(strings.ToLower(r.Body), "<title"); n != 1 {
			t.Errorf("%d <title> elements", n)
		}
	})
	t.Run("missing issue", func(t *testing.T) {
		must(t, get(t, env, "/p/acme-newsletter/archive/no-such-issue"), 404, "This issue isn't in the archive")
	})
	t.Run("feed", func(t *testing.T) {
		r := get(t, env, "/p/acme-newsletter/archive.rss")
		if r.Status != 200 || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/rss+xml") {
			t.Fatalf("feed: %d %v", r.Status, r.Header)
		}
		var feed struct {
			Channel struct {
				Title string `xml:"title"`
				Link  string `xml:"link"`
				Items []struct {
					Title string `xml:"title"`
					Link  string `xml:"link"`
					GUID  string `xml:"guid"`
				} `xml:"item"`
			} `xml:"channel"`
		}
		if err := xml.Unmarshal([]byte(r.Body), &feed); err != nil {
			t.Fatalf("feed isn't XML: %v", err)
		}
		if n := len(feed.Channel.Items); n == 0 || n > 44 {
			t.Errorf("%d items", n)
		}
		if feed.Channel.Title != "Acme newsletter" || feed.Channel.Link != "http://news.acme.io/archive" {
			t.Errorf("channel %q %q", feed.Channel.Title, feed.Channel.Link)
		}
		for _, it := range feed.Channel.Items {
			if !strings.HasPrefix(it.Link, "http://news.acme.io/archive/") || it.GUID != it.Link || it.Title == "" {
				t.Errorf("item %+v", it)
				break
			}
		}
	})
	t.Run("sitemap", func(t *testing.T) {
		r := get(t, env, "/p/acme-newsletter/sitemap.xml")
		var set struct {
			URLs []struct {
				Loc string `xml:"loc"`
			} `xml:"url"`
		}
		if err := xml.Unmarshal([]byte(r.Body), &set); err != nil || r.Status != 200 {
			t.Fatalf("sitemap: %d %v", r.Status, err)
		}
		locs := map[string]bool{}
		for _, u := range set.URLs {
			locs[u.Loc] = true
		}
		for _, want := range []string{"http://news.acme.io/subscribe", "http://news.acme.io/archive", "http://news.acme.io/archive/the-case-for-boring-infrastructure"} {
			if !locs[want] {
				t.Errorf("sitemap lacks %s", want)
			}
		}
		if len(set.URLs) != 46 {
			t.Errorf("%d URLs, want the subscribe page, the archive and 44 issues", len(set.URLs))
		}
	})
	t.Run("archive off", func(t *testing.T) {
		if _, err := env.App.Pool.Exec(context.Background(), `UPDATE public_page_settings SET pages = jsonb_set(pages, '{archive,enabled}', 'false')`); err != nil {
			t.Fatal(err)
		}
		must(t, get(t, env, "/p/acme-newsletter/archive"), 404)
		must(t, get(t, env, "/p/acme-newsletter/archive/the-case-for-boring-infrastructure"), 404)
		must(t, get(t, env, "/p/acme-newsletter/archive.rss"), 404)
	})
}

func TestCustomDomain(t *testing.T) {
	env, ids := demo(t)
	news := func(method, path string, form url.Values, h hdr) resp {
		if h == nil {
			h = hdr{}
		}
		h["Host"] = "news.acme.io"
		return do(t, env, method, path, form, h)
	}
	t.Run("subscribe page at the root", func(t *testing.T) {
		r := news(http.MethodGet, "/subscribe", nil, nil)
		must(t, r, 200, "The Acme newsletter", `action="http://news.acme.io/subscribe"`, `<link rel="canonical" href="http://news.acme.io/subscribe">`,
			`href="http://news.acme.io/p/_/public.css?v=`)
		if r := do(t, env, http.MethodGet, "/subscribe", nil, hdr{"Host": "NEWS.acme.io:443"}); r.Status != 200 || !strings.Contains(r.Text(), "The Acme newsletter") {
			t.Errorf("host with a port and capitals: %d", r.Status)
		}
	})
	t.Run("home redirects to the subscribe page", func(t *testing.T) {
		r := news(http.MethodGet, "/", nil, nil)
		if r.Status != 302 || r.Header.Get("Location") != "http://news.acme.io/subscribe" {
			t.Errorf("home: %d %q", r.Status, r.Header.Get("Location"))
		}
	})
	t.Run("archive at the root", func(t *testing.T) {
		r := news(http.MethodGet, "/archive", nil, nil)
		must(t, r, 200, "Newsletter archive", "44 issues.", `href="http://news.acme.io/archive/`)
		must(t, news(http.MethodGet, "/archive/the-case-for-boring-infrastructure", nil, nil), 200, "The case for boring infrastructure")
		must(t, news(http.MethodGet, "/archive.rss", nil, nil), 200, "<rss")
	})
	t.Run("robots", func(t *testing.T) {
		r := news(http.MethodGet, "/robots.txt", nil, nil)
		must(t, r, 200, "User-agent: *", "Disallow: /confirm/", "Disallow: /unsubscribe/", "Disallow: /preferences", "Sitemap: http://news.acme.io/sitemap.xml")
		if strings.Contains(r.Body, "Disallow: /archive") {
			t.Error("the indexable archive is disallowed")
		}
	})
	t.Run("signed links at the root", func(t *testing.T) {
		r := news(http.MethodGet, "/preferences/"+sign(env, linktoken.Preferences, ids.ws, ids.sam, uuid.Nil), nil, nil)
		must(t, r, 200, "Email preferences", "sam.rivera@outlook.com", `action="http://news.acme.io/preferences/`)
	})
	t.Run("signing up on the custom domain", func(t *testing.T) {
		f := url.Values{"_t": {formToken(env, "frm_8c2e1a", time.Minute)}, "email": {"avery@fastmail.com"}, "list": {ids.digest.String()}, "consent": {"1"}}
		r := news(http.MethodPost, "/subscribe", f, hdr{"Origin": "http://news.acme.io"})
		must(t, r, 200, "Check your inbox", "avery@fastmail.com")
	})
	t.Run("other hosts keep the web app at the root", func(t *testing.T) {
		r := do(t, env, http.MethodGet, "/subscribe", nil, hdr{"Host": "unknown.example"})
		if r.Status != 200 || strings.Contains(r.Text(), "The Acme newsletter") || strings.Contains(r.Header.Get("Content-Security-Policy"), "default-src 'none'") {
			t.Errorf("unknown host: %d %v\n%s", r.Status, r.Header, clip(r.Body))
		}
		r = do(t, env, http.MethodGet, "/robots.txt", nil, hdr{"Host": "unknown.example"})
		if strings.Contains(r.Body, "Disallow: /unsubscribe/") {
			t.Error("an unknown host answers with the custom domain's robots.txt")
		}
		// The instance itself: its own host never maps to a workspace.
		if r := get(t, env, "/subscribe"); strings.Contains(r.Text(), "The Acme newsletter") {
			t.Error("the instance's root /subscribe serves a workspace's page")
		}
	})
	t.Run("a pending domain isn't served", func(t *testing.T) {
		if _, err := env.App.Pool.Exec(context.Background(), `UPDATE public_page_settings SET custom_domain = 'pending.acme.io', domain_status = 'pending' WHERE workspace_id = $1`, ids.ws); err != nil {
			t.Fatal(err)
		}
		if r := do(t, env, http.MethodGet, "/subscribe", nil, hdr{"Host": "pending.acme.io"}); strings.Contains(r.Text(), "The Acme newsletter") {
			t.Error("a domain whose DNS isn't verified serves the pages")
		}
		if r := get(t, env, "/p/_/tls-check?domain=pending.acme.io"); r.Status != 404 {
			t.Errorf("on-demand TLS ask for a pending domain: %d", r.Status)
		}
	})
}

func TestAssets(t *testing.T) {
	env, _ := demo(t)
	t.Run("theme", func(t *testing.T) {
		r := get(t, env, "/p/acme-newsletter/theme.css?v=1")
		must(t, r, 200, "--pub-accent:#4338CA", "--pub-ground:#F5F3FF", "--pub-hover:#3730A3", "--pub-on:#FFFFFF")
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "text/css") || !strings.Contains(r.Header.Get("Cache-Control"), "immutable") {
			t.Errorf("theme headers: %v", r.Header)
		}
	})
	t.Run("logo", func(t *testing.T) {
		r := get(t, env, "/p/acme-newsletter/logo?v=1")
		if r.Status != 200 || r.Header.Get("Content-Type") != "image/svg+xml" || !strings.Contains(r.Body, "<svg") {
			t.Fatalf("logo: %d %v", r.Status, r.Header)
		}
		if csp := r.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "sandbox") || !strings.Contains(csp, "default-src 'none'") {
			t.Errorf("an SVG logo opened directly must not run anything: %s", csp)
		}
		if r.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Error("nosniff")
		}
	})
	t.Run("static files", func(t *testing.T) {
		for path, ctype := range map[string]string{"/p/_/public.css": "text/css", "/p/_/embed.css": "text/css", "/p/_/issue.css": "text/css", "/p/_/challenge.js": "text/javascript"} {
			r := get(t, env, path)
			if r.Status != 200 || !strings.HasPrefix(r.Header.Get("Content-Type"), ctype) || r.Body == "" {
				t.Errorf("%s: %d %q", path, r.Status, r.Header.Get("Content-Type"))
			}
		}
		for _, path := range []string{"/p/_/nope.css", "/p/_/fonts/nope.woff2", "/p/_/fonts/..%2F..%2Fpublicpages.go"} {
			if r := get(t, env, path); r.Status != 404 {
				t.Errorf("%s: %d", path, r.Status)
			}
		}
	})
	t.Run("unknown workspace", func(t *testing.T) {
		for _, path := range []string{"/p/nobody/subscribe", "/p/nobody/archive", "/p/nobody/theme.css", "/p/nobody/preferences"} {
			if r := get(t, env, path); r.Status != 404 {
				t.Errorf("%s: %d", path, r.Status)
			}
		}
	})
}

// A Turnstile error report counts only from the page of a live challenge.
// Otherwise anyone could set the workspace's Turnstile status to "error",
// which turns the challenge fallback off (suspicious signups are refused
// instead of checked) and shows a false configuration error in the app.
func TestTurnstileErrorReportNeedsAChallenge(t *testing.T) {
	env, ids := demo(t)
	r := do(t, env, http.MethodPost, "/p/acme-newsletter/turnstile-error", url.Values{"code": {"110200"}}, nil)
	if r.Status != 204 {
		t.Fatalf("report: %d", r.Status)
	}
	if st := one[string](t, env, `SELECT turnstile_status FROM public_page_settings WHERE workspace_id = $1`, ids.ws); st != "ok" {
		t.Errorf("an anonymous report without a challenge set Turnstile to %q (accept reports only for a live form_challenges id, and rate limit them)", st)
	}
}
