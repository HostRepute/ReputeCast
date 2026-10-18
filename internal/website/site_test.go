package website_test

import (
	"context"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/mailer"
)

const site = "/p/acme-newsletter"

// mailbox keeps the website's System-stream emails (an in-process fake:
// nothing leaves the test).
type mailbox struct {
	mu   sync.Mutex
	msgs []mailer.Message
}

func (m *mailbox) SendSystem(_ context.Context, msg mailer.Message) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
	return "test", nil
}

func (m *mailbox) SystemAvailable(context.Context, *uuid.UUID) bool { return true }

func (m *mailbox) to(addr string) []mailer.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []mailer.Message
	for _, msg := range m.msgs {
		if strings.EqualFold(msg.To, addr) {
			out = append(out, msg)
		}
	}
	return out
}

var linkRe = regexp.MustCompile(`/signin/link/([A-Za-z0-9_-]{20,100})`)
var codeRe = regexp.MustCompile(`\b(\d{3}) (\d{3})\b`)

// linkOf is the token and the code of an email.
func linkOf(t *testing.T, msg mailer.Message) (token, code string) {
	t.Helper()
	m := linkRe.FindStringSubmatch(msg.Text)
	if m == nil {
		t.Fatalf("no link in %q", msg.Text)
	}
	if c := codeRe.FindStringSubmatch(msg.Text); c != nil {
		code = c[1] + c[2]
	}
	return m[1], code
}

// demoOn is the seeded demo, its website on, with a mailbox for its emails.
func demoOn(t *testing.T) (*apptest.Env, *mailbox) {
	t.Helper()
	env := apptest.New(t)
	env.SeedDemo()
	box := &mailbox{}
	env.App.Website().Mail = func() mailer.Sender { return box }
	return env, box
}

// sessionOf signs a client in with one of the demo's site sessions.
func sessionOf(t *testing.T, env *apptest.Env, key string) *apptest.Client {
	t.Helper()
	c := env.Client()
	c.Header = http.Header{"Cookie": {"rc_member=demo-session-" + key + "-" + workspaceID(t, env).String()}}
	return c
}

func page(t *testing.T, c *apptest.Client, path string, want int) string {
	t.Helper()
	r := c.Do("GET", site+path, nil)
	if r.Status != want {
		t.Fatalf("GET %s: %d, want %d: %.300s", path, r.Status, want, r.Body)
	}
	return string(r.Body)
}

// has reports text in a page, as the page escapes it.
func has(body, text string) bool {
	return strings.Contains(body, text) || strings.Contains(body, html.EscapeString(text))
}

func form(c *apptest.Client, path string, v url.Values) *apptest.Response {
	return c.DoRaw("POST", site+path, "application/x-www-form-urlencoded", strings.NewReader(v.Encode()))
}

// Text a reader may not read never reaches the page, the feed, the
// sitemap or the search: the runbook's preview stops at its marker.
func TestSiteGating(t *testing.T) {
	env, _ := demoOn(t)
	ctx := context.Background()
	const preview, gated = "every migration guide we found stopped", "Size the target for the week after"

	visitor := env.Client()
	body := page(t, visitor, "/posts/postgres-migration-runbook", 200)
	if !strings.Contains(body, preview) || strings.Contains(body, gated) || !strings.Contains(body, "This post is for Insider members") {
		t.Fatalf("visitor's runbook: preview %v, gated %v", strings.Contains(body, preview), strings.Contains(body, gated))
	}
	if !strings.Contains(body, `"isAccessibleForFree":false`) || !strings.Contains(body, "Payments go to Acme through Stripe.") {
		t.Fatalf("structured data or trust line missing")
	}
	r := visitor.Do("GET", site+"/posts/postgres-migration-runbook", nil)
	if cc := r.Header.Get("Cache-Control"); !strings.Contains(cc, "no-store") {
		t.Fatalf("a gated post is cached: %q", cc)
	}
	// A public post may be cached.
	r = visitor.Do("GET", site+"/posts/the-case-for-boring-infrastructure", nil)
	if r.Status != 200 || !strings.Contains(r.Header.Get("Cache-Control"), "public") {
		t.Fatalf("public post: %d %q", r.Status, r.Header.Get("Cache-Control"))
	}

	// Sam holds Weekly Digest Insider: the whole post and a quiet label.
	sam := sessionOf(t, env, "sam")
	body = page(t, sam, "/posts/postgres-migration-runbook", 200)
	if !strings.Contains(body, gated) || !strings.Contains(body, "Included in your Insider membership") || strings.Contains(body, `class="st-pw"`) {
		t.Fatal("a paid member doesn't get the whole post")
	}
	r = sam.Do("GET", site+"/posts/the-case-for-boring-infrastructure", nil)
	if !strings.Contains(r.Header.Get("Cache-Control"), "no-store") {
		t.Fatalf("a signed-in page is cached: %q", r.Header.Get("Cache-Control"))
	}

	// Alex is a free member: one paid post a month, then the tiers.
	wsID := workspaceID(t, env)
	if _, err := env.App.Pool.Exec(ctx, `DELETE FROM website_meter_reads WHERE workspace_id = $1`, wsID); err != nil {
		t.Fatal(err)
	}
	alex := sessionOf(t, env, "alex")
	body = page(t, alex, "/posts/postgres-migration-runbook", 200)
	if !strings.Contains(body, gated) || !strings.Contains(body, "This is your free paid post for") {
		t.Fatal("the free member's first paid post isn't whole")
	}
	body = page(t, alex, "/posts/postgres-migration-runbook", 200)
	if !strings.Contains(body, gated) {
		t.Fatal("reading the same post again isn't free")
	}
	body = page(t, alex, "/posts/rate-limits-best-customers", 200)
	if strings.Contains(body, "Counting customers, not addresses") || !has(body, "You've read your free paid post for") || !strings.Contains(body, "free member") {
		t.Fatal("the meter didn't stop the second paid post")
	}

	// Feed, sitemap and search read only what visitors may.
	feed := page(t, visitor, "/feed", 200)
	if !strings.Contains(feed, "Inside our Postgres migration") || strings.Contains(feed, gated) || strings.Contains(feed, "Counting customers") {
		t.Fatal("the feed leaks gated text")
	}
	sitemap := page(t, visitor, "/sitemap.xml", 200)
	if !strings.Contains(sitemap, "/posts/postgres-migration-runbook") || !strings.Contains(sitemap, "/tag/engineering") || strings.Contains(sitemap, "q4-roadmap-deep-dive") {
		t.Fatalf("sitemap: %.400s", sitemap)
	}
	if body := page(t, visitor, "/archive?q=vCPUs", 200); strings.Contains(body, "/posts/postgres-migration-runbook") {
		t.Fatal("search finds gated text")
	}
	if body := page(t, visitor, "/archive?q=switching", 200); !strings.Contains(body, "Inside our") {
		t.Fatal("search doesn't find the preview's text")
	}
}

// The visitor meter: members-only posts a month per browser, then "Sign
// up free to keep reading".
func TestSiteVisitorMeter(t *testing.T) {
	env, _ := demoOn(t)
	ctx := context.Background()
	wsID := workspaceID(t, env)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE website_settings SET meter_visitors = 1 WHERE workspace_id = $1`, wsID); err != nil {
		t.Fatal(err)
	}
	// A second members-only post.
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	var p struct{ ID string }
	owner.Do("POST", ws+"/website/posts", map[string]any{"title": "Members' notes"}).MustStatus(t, 201).JSON(t, &p)
	owner.Do("PUT", ws+"/website/posts/"+p.ID+"/content", map[string]any{"mjml": `<mjml><mj-body><mj-section><mj-column><mj-text><p>Only for members, word zebrafish.</p></mj-text></mj-column></mj-section></mj-body></mjml>`,
		"baseVersion": 0, "source": "manual"}).MustStatus(t, 200)
	owner.Do("PUT", ws+"/website/posts/"+p.ID, map[string]any{"title": "Members' notes", "slug": "members-notes", "excerpt": "", "coverUrl": "", "coverAlt": "",
		"tags": []string{}, "visibility": "members", "tiers": []string{}, "neverFree": false, "featured": false, "webOn": true, "publishMode": "now",
		"seo": map[string]any{"title": "", "description": "", "socialImage": "", "canonicalUrl": "", "hideFromSearch": false}}).MustStatus(t, 200)
	owner.Do("POST", ws+"/website/posts/"+p.ID+"/publish", map[string]any{"when": "now"}).MustStatus(t, 200)

	v := env.Client()
	body := page(t, v, "/posts/ask-acme-deliverability-questions", 200)
	if !strings.Contains(body, "1 of 1") || !strings.Contains(body, "the Promotions tab") {
		t.Fatal("the first members-only post isn't whole for a visitor")
	}
	body = page(t, v, "/posts/members-notes", 200)
	if strings.Contains(body, "zebrafish") || !strings.Contains(body, "Sign up free to keep reading") {
		t.Fatal("the visitor meter didn't stop the second post")
	}
	// Another browser has its own month.
	if body := page(t, env.Client(), "/posts/members-notes", 200); !strings.Contains(body, "zebrafish") {
		t.Fatal("a new browser has no free read")
	}
	// Never free leaves a post out of the meter.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE website_posts SET never_free = true WHERE id = $1`, p.ID); err != nil {
		t.Fatal(err)
	}
	if body := page(t, env.Client(), "/posts/members-notes", 200); strings.Contains(body, "zebrafish") {
		t.Fatal("a never-free post opens with the meter")
	}
}

// Months are calendar months in the workspace's time zone.
func TestSiteMeterMonths(t *testing.T) {
	env, _ := demoOn(t)
	ctx := context.Background()
	svc := env.App.Website()
	wsID := workspaceID(t, env)
	if _, err := env.App.Pool.Exec(ctx, `DELETE FROM website_meter_reads WHERE workspace_id = $1`, wsID); err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("America/New_York")
	defer func() { svc.Now = nil }()
	alex := sessionOf(t, env, "alex")
	// 31 Oct, 23:30 in New York: October's free paid post.
	svc.Now = func() time.Time { return time.Date(2026, 10, 31, 23, 30, 0, 0, loc) }
	if body := page(t, alex, "/posts/postgres-migration-runbook", 200); !strings.Contains(body, "This is your free paid post for October") {
		t.Fatal("October's read")
	}
	if body := page(t, alex, "/posts/rate-limits-best-customers", 200); strings.Contains(body, "Counting customers") {
		t.Fatal("two paid posts in October")
	}
	// 1 Nov, 00:10 in New York (04:10 UTC): a new month.
	svc.Now = func() time.Time { return time.Date(2026, 11, 1, 0, 10, 0, 0, loc) }
	if body := page(t, alex, "/posts/rate-limits-best-customers", 200); !strings.Contains(body, "Counting customers") || !strings.Contains(body, "for November") {
		t.Fatal("November's free paid post")
	}
}

// Signing in: the same answer for every address, a link that scanners
// don't use up, single use, a newer link replacing older ones, the code
// in the browser that asked, its tries, expiry, limits and suppressed
// addresses.
func TestSiteSignin(t *testing.T) {
	env, box := demoOn(t)
	ctx := context.Background()
	svc := env.App.Website()
	wsID := workspaceID(t, env)
	back := site + "/posts/ask-acme-deliverability-questions"
	ask := func(c *apptest.Client, email string) *apptest.Response {
		return form(c, "/signin", url.Values{"email": {email}, "return": {back}})
	}
	c := env.Client()
	r := ask(c, "Sam.Rivera@outlook.com")
	if r.Status != 200 || !strings.Contains(string(r.Body), "a link and a code are on their way") {
		t.Fatalf("ask: %d %.300s", r.Status, r.Body)
	}
	if !strings.Contains(r.Header.Get("Referrer-Policy"), "no-referrer") {
		t.Fatalf("referrer policy: %q", r.Header.Get("Referrer-Policy"))
	}
	sent := box.to("sam.rivera@outlook.com")
	if len(sent) != 1 || !strings.Contains(sent[0].Subject, "sign-in link") {
		t.Fatalf("sign-in email: %+v", sent)
	}
	token, code := linkOf(t, sent[0])
	if len(code) != 6 {
		t.Fatalf("code %q", code)
	}
	// Unknown and suppressed addresses get the same answer and no email.
	if r := ask(env.Client(), "nobody-here@example.com"); r.Status != 200 || !strings.Contains(string(r.Body), "a link and a code are on their way") {
		t.Fatalf("unknown: %d", r.Status)
	}
	var supp string
	if err := env.App.Pool.QueryRow(ctx, `SELECT email FROM contacts WHERE workspace_id = $1 AND status = 'active' AND email <> 'sam.rivera@outlook.com' ORDER BY id LIMIT 1`, wsID).Scan(&supp); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `INSERT INTO suppressions (id, workspace_id, email, reason, created_at) VALUES ($1,$2,$3,'hard_bounce',now())`, uuid.Must(uuid.NewV7()), wsID, supp); err != nil {
		t.Fatal(err)
	}
	if r := ask(env.Client(), supp); r.Status != 200 || !strings.Contains(string(r.Body), "a link and a code are on their way") {
		t.Fatalf("suppressed: %d", r.Status)
	}
	if len(box.to("nobody-here@example.com")) != 0 || len(box.to(supp)) != 0 {
		t.Fatal("an email went to an unknown or suppressed address")
	}
	// A link scanner fetches the link: nothing is used.
	scanner := env.Client()
	if r := scanner.Do("GET", site+"/signin/link/"+token, nil); r.Status != 200 || !strings.Contains(string(r.Body), "Finish signing in") {
		t.Fatalf("link page: %d", r.Status)
	}
	if r := scanner.Do("HEAD", site+"/signin/link/"+token, nil); r.Status != 200 {
		t.Fatalf("HEAD: %d", r.Status)
	}
	// Continue signs in, once.
	r = form(c, "/signin/link/"+token, url.Values{})
	if r.Status != http.StatusSeeOther || !strings.HasSuffix(r.Header.Get("Location"), "/posts/ask-acme-deliverability-questions") {
		t.Fatalf("continue: %d %s", r.Status, r.Header.Get("Location"))
	}
	if body := page(t, c, "/posts/ask-acme-deliverability-questions", 200); !has(body, "You're signed in as sam.rivera@outlook.com") {
		t.Fatal("no toast after signing in")
	}
	if body := page(t, c, "/account", 200); !strings.Contains(body, "sam.rivera@outlook.com") || has(body, "You're signed in as") {
		t.Fatal("not signed in, or the toast shows twice")
	}
	if r := form(env.Client(), "/signin/link/"+token, url.Values{}); r.Status != http.StatusGone || !strings.Contains(string(r.Body), "already used") {
		t.Fatalf("second use: %d", r.Status)
	}

	// A newer link replaces older ones; the code works where it was asked.
	b := env.Client()
	ask(b, "sam.rivera@outlook.com")
	ask(b, "sam.rivera@outlook.com")
	sent = box.to("sam.rivera@outlook.com")
	old, _ := linkOf(t, sent[1])
	_, code = linkOf(t, sent[2])
	if r := b.Do("GET", site+"/signin/link/"+old, nil); r.Status != http.StatusGone || !strings.Contains(string(r.Body), "newer link") {
		t.Fatalf("replaced link: %d", r.Status)
	}
	other := env.Client()
	if r := form(other, "/signin/code", url.Values{"code": {code}, "return": {back}}); r.Status == http.StatusSeeOther {
		t.Fatal("a code works in another browser")
	}
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	if r := form(b, "/signin/code", url.Values{"code": {wrong}, "return": {back}}); r.Status != 422 || !strings.Contains(string(r.Body), "2 tries left") {
		t.Fatalf("wrong code: %d", r.Status)
	}
	if r := form(b, "/signin/code", url.Values{"d1": {code[:1]}, "d2": {code[1:2]}, "d3": {code[2:3]}, "d4": {code[3:4]}, "d5": {code[4:5]}, "d6": {code[5:]},
		"return": {back}}); r.Status != http.StatusSeeOther {
		t.Fatalf("right code: %d %.300s", r.Status, r.Body)
	}

	// Links work for 30 minutes.
	e := env.Client()
	ask(e, "sam.rivera@outlook.com")
	sent = box.to("sam.rivera@outlook.com")
	late, _ := linkOf(t, sent[len(sent)-1])
	svc.Now = func() time.Time { return time.Now().Add(31 * time.Minute) }
	r = e.Do("GET", site+"/signin/link/"+late, nil)
	svc.Now = nil
	if r.Status != http.StatusGone || !strings.Contains(string(r.Body), "has expired") {
		t.Fatalf("expired link: %d", r.Status)
	}
	// 5 links an hour per address.
	limited := false
	for i := 0; i < 6 && !limited; i++ {
		limited = ask(env.Client(), "sam.rivera@outlook.com").Status == http.StatusTooManyRequests
	}
	if !limited {
		t.Fatal("no limit per address")
	}
	// Sign out.
	r = form(c, "/signout", url.Values{})
	if r.Status != http.StatusSeeOther {
		t.Fatalf("sign out: %d", r.Status)
	}
	if r := c.Do("GET", site+"/account", nil); r.Status != http.StatusSeeOther {
		t.Fatalf("account after signing out: %d", r.Status)
	}
}

// Signing up from the paywall: one email confirms the newsletters ticked
// and signs the reader in, back on the post; the sign-up counts for it.
func TestSiteJoin(t *testing.T) {
	env, box := demoOn(t)
	ctx := context.Background()
	wsID := workspaceID(t, env)
	var digest uuid.UUID
	if err := env.App.Pool.QueryRow(ctx, `SELECT id FROM lists WHERE workspace_id = $1 AND name = 'Weekly Digest'`, wsID).Scan(&digest); err != nil {
		t.Fatal(err)
	}
	c := env.Client()
	page(t, c, "/posts/postgres-migration-runbook", 200)
	r := form(c, "/posts/postgres-migration-runbook", url.Values{"email": {"not an email"}, "list": {digest.String()}})
	if r.Status != 422 || !strings.Contains(string(r.Body), "pw-err") {
		t.Fatalf("bad address: %d", r.Status)
	}
	r = form(c, "/posts/postgres-migration-runbook", url.Values{"email": {"jordan.lee@example.com"}, "list": {digest.String()}, "return": {site + "/posts/postgres-migration-runbook"}})
	if r.Status != 200 || !strings.Contains(string(r.Body), "Check your email") {
		t.Fatalf("join: %d %.300s", r.Status, r.Body)
	}
	sent := box.to("jordan.lee@example.com")
	if len(sent) != 1 || sent[0].Subject != "Confirm your email and continue reading" || !strings.Contains(sent[0].Text, "Inside our Postgres migration") {
		t.Fatalf("join email: %+v", sent)
	}
	token, _ := linkOf(t, sent[0])
	r = form(c, "/signin/link/"+token, url.Values{})
	if r.Status != http.StatusSeeOther || !strings.HasSuffix(r.Header.Get("Location"), "/posts/postgres-migration-runbook") {
		t.Fatalf("confirm: %d %s", r.Status, r.Header.Get("Location"))
	}
	var status string
	if err := env.App.Pool.QueryRow(ctx, `SELECT lm.status FROM list_memberships lm JOIN contacts c ON c.id = lm.contact_id
		WHERE c.email = 'jordan.lee@example.com' AND lm.list_id = $1`, digest).Scan(&status); err != nil || status != "subscribed" {
		t.Fatalf("membership: %q %v", status, err)
	}
	var n int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM website_post_conversions v JOIN website_posts p ON p.id = v.post_id
		WHERE p.slug = 'postgres-migration-runbook' AND v.kind = 'signup' AND v.contact_id IS NOT NULL`).Scan(&n)
	if n != 1 {
		t.Fatalf("signup conversions: %d", n)
	}
	// The consent keeps the exact text.
	var text string
	_ = env.App.Pool.QueryRow(ctx, `SELECT coalesce(e.consent_text, '') FROM consent_events e JOIN contacts c ON c.id = e.contact_id
		WHERE c.email = 'jordan.lee@example.com' AND e.consent_text <> '' ORDER BY e.occurred_at LIMIT 1`).Scan(&text)
	if !strings.Contains(text, "Weekly Digest every Tuesday") {
		t.Fatalf("consent text: %q", text)
	}
}

// Unpublished posts answer 410, or their redirect; renamed posts and
// archive issues redirect; the website off brings the archive back.
func TestSiteAddresses(t *testing.T) {
	env, _ := demoOn(t)
	ctx := context.Background()
	v := env.Client()
	page(t, v, "/posts/feature-flags-preview", http.StatusGone)
	page(t, v, "/posts/no-such-post", http.StatusNotFound)
	r := v.Do("GET", site+"/archive/the-case-for-boring-infrastructure", nil)
	if r.Status != http.StatusMovedPermanently || !strings.HasSuffix(r.Header.Get("Location"), "/posts/the-case-for-boring-infrastructure") {
		t.Fatalf("archive issue: %d %s", r.Status, r.Header.Get("Location"))
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE website_posts SET redirect_to = '/archive' WHERE slug = 'feature-flags-preview'`); err != nil {
		t.Fatal(err)
	}
	r = v.Do("GET", site+"/posts/feature-flags-preview", nil)
	if r.Status != http.StatusMovedPermanently || !strings.HasSuffix(r.Header.Get("Location"), "/archive") {
		t.Fatalf("redirect: %d %s", r.Status, r.Header.Get("Location"))
	}
	// Turned off: the archive is back, posts are gone, home is the subscribe page.
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	var ov struct{ Address struct{ Domain, Host string } }
	owner.Do("GET", ws+"/website", nil).MustStatus(t, 200).JSON(t, &ov)
	confirm := ov.Address.Domain
	if confirm == "" {
		confirm = ov.Address.Host
	}
	owner.Do("POST", ws+"/website/turn-off", map[string]any{"confirm": confirm}).MustStatus(t, 200)
	page(t, v, "/posts/postgres-migration-runbook", http.StatusNotFound)
	page(t, v, "/archive/the-case-for-boring-infrastructure", http.StatusOK)
	if r := v.Do("GET", site+"/", nil); r.Status != http.StatusFound || !strings.Contains(r.Header.Get("Location"), "/subscribe") {
		t.Fatalf("home while off: %d %s", r.Status, r.Header.Get("Location"))
	}
	page(t, v, "/signin", http.StatusNotFound)
}

// Views are counted on the server without cookies: one per visitor, page
// and day; bots aren't counted.
func TestSiteViews(t *testing.T) {
	env, _ := demoOn(t)
	ctx := context.Background()
	count := func() int64 {
		var n int64
		_ = env.App.Pool.QueryRow(ctx, `SELECT coalesce(sum(d.views), 0) FROM website_view_days d JOIN website_posts p ON p.id = d.page_id
			WHERE p.slug = 'the-case-for-boring-infrastructure' AND d.page_kind = 'post'`).Scan(&n)
		return n
	}
	before := count()
	a := env.Client()
	a.Header = http.Header{"User-Agent": {"Mozilla/5.0 (Macintosh) Firefox/131.0"}}
	page(t, a, "/posts/the-case-for-boring-infrastructure", 200)
	page(t, a, "/posts/the-case-for-boring-infrastructure?utm_source=linkedin", 200)
	if got := count(); got != before+1 {
		t.Fatalf("one visitor twice: %d → %d", before, got)
	}
	b := env.Client()
	b.Header = http.Header{"User-Agent": {"Mozilla/5.0 (iPhone) Safari/605.1.15"}}
	page(t, b, "/posts/the-case-for-boring-infrastructure", 200)
	if got := count(); got != before+2 {
		t.Fatalf("a second visitor: %d → %d", before, got)
	}
	bot := env.Client()
	bot.Header = http.Header{"User-Agent": {"Googlebot/2.1 (+http://www.google.com/bot.html)"}}
	page(t, bot, "/posts/the-case-for-boring-infrastructure", 200)
	if got := count(); got != before+2 {
		t.Fatalf("a bot was counted: %d → %d", before, got)
	}
	// Nothing in the database identifies the visitor: hashes only.
	var cookies int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM website_view_seen WHERE length(hash) <> 32`).Scan(&cookies)
	if cookies != 0 {
		t.Fatal("unexpected view keys")
	}
}

// The account page: name, email change by a link to the new address,
// newsletters, sign out everywhere.
func TestSiteAccount(t *testing.T) {
	env, box := demoOn(t)
	ctx := context.Background()
	alex := sessionOf(t, env, "alex")
	body := page(t, alex, "/account", 200)
	if !strings.Contains(body, "a.murphy@icloud.com") || !strings.Contains(body, "Free member") {
		t.Fatalf("account: %.300s", body)
	}
	// Forms post with the session a sign-in puts in the client's jar.
	c := env.Client()
	r := form(c, "/signin", url.Values{"email": {"a.murphy@icloud.com"}})
	if r.Status != 200 {
		t.Fatalf("ask: %d", r.Status)
	}
	token, _ := linkOf(t, box.to("a.murphy@icloud.com")[0])
	form(c, "/signin/link/"+token, url.Values{}).MustStatus(t, http.StatusSeeOther)
	form(c, "/account/name", url.Values{"first_name": {"Alexandra"}, "last_name": {"Murphy"}}).MustStatus(t, http.StatusSeeOther)
	var first string
	_ = env.App.Pool.QueryRow(ctx, `SELECT first_name FROM contacts WHERE email = 'a.murphy@icloud.com'`).Scan(&first)
	if first != "Alexandra" {
		t.Fatalf("name: %q", first)
	}
	r = form(c, "/account/email", url.Values{"email": {"alex.new@example.com"}})
	if r.Status != 200 || !strings.Contains(string(r.Body), "alex.new@example.com") {
		t.Fatalf("email change: %d", r.Status)
	}
	msgs := box.to("alex.new@example.com")
	if len(msgs) != 1 {
		t.Fatalf("confirmation to the new address: %d", len(msgs))
	}
	token, _ = linkOf(t, msgs[0])
	form(c, "/signin/link/"+token, url.Values{}).MustStatus(t, http.StatusSeeOther)
	var email string
	_ = env.App.Pool.QueryRow(ctx, `SELECT email FROM contacts WHERE first_name = 'Alexandra' AND last_name = 'Murphy'`).Scan(&email)
	if email != "alex.new@example.com" {
		t.Fatalf("email after confirming: %q", email)
	}
	form(c, "/account/everywhere", url.Values{}).MustStatus(t, http.StatusSeeOther)
	var active int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM website_sessions s JOIN contacts c ON c.id = s.contact_id WHERE c.email = 'alex.new@example.com' AND s.ended_at IS NULL`).Scan(&active)
	if active != 0 {
		t.Fatalf("sessions after signing out everywhere: %d", active)
	}
}
