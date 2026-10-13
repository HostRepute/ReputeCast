package publicpages_test

// Flows across the public pages, the builder's API and the jobs: double
// opt-in while nothing can send System mail (the copy-link fallback), the
// confirmations job and its reminder, Cloudflare Turnstile checks answered
// by a fake siteverify, view online, and RFC 8058 one-click at the exact
// address the sending engine puts in List-Unsubscribe.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/forms"
	"github.com/hostrepute/reputecast/internal/linktoken"
	"github.com/hostrepute/reputecast/internal/mailer"
)

// formsService is the forms service of env's app, for running its jobs.
func formsService(env *apptest.Env) *forms.Service {
	a := env.App
	return &forms.Service{
		Pool: a.Pool, Q: a.Q, Audit: a.Audit, Box: a.Secrets, Tokens: a.Links, URLs: a.URLs,
		Audience: &audience.Service{Pool: a.Pool, Q: a.Q, Audit: a.Audit, Mail: a.Mail, Links: a.Links, URLs: a.URLs},
		Mail:     func() mailer.Sender { return a.Mail },
	}
}

// signUp submits the hosted subscribe page (Newsletter signup) for email.
func signUp(t *testing.T, env *apptest.Env, ids demoIDs, email string) resp {
	t.Helper()
	return do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe", url.Values{
		"_t": {formToken(env, "frm_8c2e1a", time.Minute)}, "email": {email}, "first_name": {"Lee"}, "list": {ids.digest.String()}, "consent": {"1"},
	}, hdr{"Origin": env.Server.URL})
}

// While nothing can send System mail (App.Mail is mailer.Unavailable until a
// provider handles the System stream), double opt-in signups wait and the
// builder hands out their links to send by hand, each one logged.
func TestConfirmationLinksWithoutSystemMail(t *testing.T) {
	env, ids := demo(t)
	if _, ok := env.App.Mail.(mailer.Unavailable); !ok {
		t.Fatalf("App.Mail is %T, want mailer.Unavailable", env.App.Mail)
	}
	must(t, signUp(t, env, ids, "lee.wong@fastmail.com"), 200, "Check your inbox", "lee.wong@fastmail.com")
	lee, ok := contactID(t, env, ids.ws, "lee.wong@fastmail.com")
	if !ok {
		t.Fatal("no contact for the signup")
	}
	if n := count(t, env, `SELECT count(*) FROM consent_events WHERE contact_id = $1 AND action = 'optin_sent'`, lee); n != 0 {
		t.Errorf("%d optin_sent events while nothing could send", n)
	}

	c := owner(env)
	formPath := wsAPI + "/forms/" + ids.signup.String()
	var o struct {
		SystemProvider string `json:"systemProvider"`
		SystemMail     bool   `json:"systemMail"`
	}
	c.Do(http.MethodGet, wsAPI+"/forms", nil).MustStatus(t, 200).JSON(t, &o)
	if o.SystemMail || o.SystemProvider != "Postmark" {
		t.Errorf("overview: System mail %v via %q, want false via the routed Postmark", o.SystemMail, o.SystemProvider)
	}
	var f struct {
		Waiting int64 `json:"waiting"`
	}
	c.Do(http.MethodGet, formPath, nil).MustStatus(t, 200).JSON(t, &f)
	if f.Waiting != 1 {
		t.Fatalf("waiting signups: %d, want 1", f.Waiting)
	}
	type waiting struct {
		Items []struct {
			ID, Email, FirstName string
			Lists                []string
		} `json:"items"`
		Total      int64 `json:"total"`
		SystemMail bool  `json:"systemMail"`
	}
	var w waiting
	c.Do(http.MethodGet, formPath+"/waiting", nil).MustStatus(t, 200).JSON(t, &w)
	if w.Total != 1 || len(w.Items) != 1 || w.SystemMail {
		t.Fatalf("waiting: %+v", w)
	}
	it := w.Items[0]
	if it.Email != "lee.wong@fastmail.com" || it.FirstName != "Lee" || strings.Join(it.Lists, ",") != "Weekly Digest" {
		t.Errorf("waiting signup: %+v", it)
	}

	// Viewers see who waits but can't take a link; nobody can for another form.
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	viewer.Do(http.MethodGet, formPath+"/waiting", nil).MustStatus(t, 200)
	viewer.Do(http.MethodPost, formPath+"/waiting/"+it.ID+"/link", nil).MustStatus(t, 403)
	c.Do(http.MethodPost, wsAPI+"/forms/"+ids.footer.String()+"/waiting/"+it.ID+"/link", nil).MustStatus(t, 404)
	c.Do(http.MethodPost, formPath+"/waiting/"+uuid.NewString()+"/link", nil).MustStatus(t, 404)

	var link struct{ URL, Email string }
	c.Do(http.MethodPost, formPath+"/waiting/"+it.ID+"/link", nil).MustStatus(t, 200).JSON(t, &link)
	if link.Email != "lee.wong@fastmail.com" || !strings.HasPrefix(link.URL, env.Server.URL+"/p/acme-newsletter/confirm/") {
		t.Fatalf("link: %+v", link)
	}
	var target, meta string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT target_id, metadata::text FROM audit_events WHERE action = 'form.confirm_link'`).Scan(&target, &meta); err != nil {
		t.Fatalf("no audit entry for the copied link: %v", err)
	}
	if target != lee.String() || !strings.Contains(meta, "Newsletter signup") || strings.Contains(meta, link.URL) {
		t.Errorf("audit entry: target %s metadata %s", target, meta)
	}

	// The link subscribes the person once they open it.
	u, _ := url.Parse(link.URL)
	must(t, get(t, env, u.Path), 200, "You're in, Lee")
	if st := membership(t, env, lee, ids.digest); st != "subscribed" {
		t.Fatalf("Weekly Digest is %s after the click", st)
	}
	c.Do(http.MethodGet, formPath+"/waiting", nil).MustStatus(t, 200).JSON(t, &w)
	if w.Total != 0 || len(w.Items) != 0 {
		t.Errorf("still waiting after the confirmation: %+v", w)
	}
	c.Do(http.MethodPost, formPath+"/waiting/"+it.ID+"/link", nil).MustStatus(t, 409)
}

// The confirmations job sends what waited once System mail works, then one
// reminder to people who didn't click within 24 hours.
func TestConfirmationsJob(t *testing.T) {
	env, ids := demo(t)
	must(t, signUp(t, env, ids, "late.mail@fastmail.com"), 200, "Check your inbox")
	late, _ := contactID(t, env, ids.ws, "late.mail@fastmail.com")
	svc := formsService(env)
	ctx := context.Background()

	// Nothing can send yet: the run leaves the signup waiting.
	if err := svc.RunConfirmations(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, env, `SELECT count(*) FROM form_submissions WHERE contact_id = $1 AND confirm_sent_at IS NULL`, late); n != 1 {
		t.Fatalf("%d unsent submissions, want 1", n)
	}

	mail := &fakeMail{}
	env.App.Mail = mail
	if err := svc.RunConfirmations(ctx); err != nil {
		t.Fatal(err)
	}
	sent := mail.to("late.mail@fastmail.com")
	if len(sent) != 1 || sent[0].Kind != "double_opt_in" || strings.HasPrefix(sent[0].Subject, "Reminder") || !linkIn.MatchString(sent[0].Text) {
		t.Fatalf("the waiting confirmation: %+v", sent)
	}
	if n := count(t, env, `SELECT count(*) FROM consent_events WHERE contact_id = $1 AND action = 'optin_sent' AND note LIKE 'confirmation sent via Postmark%'`, late); n != 1 {
		t.Errorf("%d optin_sent events naming the provider, want 1", n)
	}

	// No click in 24 hours: one reminder, and only one.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE form_submissions SET confirm_sent_at = now() - interval '25 hours', created_at = now() - interval '25 hours' WHERE contact_id = $1`, late); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := svc.RunConfirmations(ctx); err != nil {
			t.Fatal(err)
		}
	}
	sent = mail.to("late.mail@fastmail.com")
	if len(sent) != 2 || !strings.HasPrefix(sent[1].Subject, "Reminder: confirm your subscription to Acme") || !strings.Contains(sent[1].Text, "Yesterday you asked for Weekly Digest") {
		t.Fatalf("reminder: %d emails, %+v", len(sent), sent)
	}

	// A contact who confirmed some other way gets nothing more.
	must(t, signUp(t, env, ids, "other.way@fastmail.com"), 200, "Check your inbox")
	other, _ := contactID(t, env, ids.ws, "other.way@fastmail.com")
	if _, err := env.App.Pool.Exec(ctx, `UPDATE list_memberships SET status = 'subscribed' WHERE contact_id = $1`, other); err != nil {
		t.Fatal(err)
	}
	if _, err := env.App.Pool.Exec(ctx, `UPDATE form_submissions SET confirm_sent_at = NULL WHERE contact_id = $1`, other); err != nil {
		t.Fatal(err)
	}
	before := len(mail.to("other.way@fastmail.com"))
	if err := svc.RunConfirmations(ctx); err != nil {
		t.Fatal(err)
	}
	if got := len(mail.to("other.way@fastmail.com")); got != before {
		t.Errorf("a confirmed contact got %d more emails", got-before)
	}
	if n := count(t, env, `SELECT count(*) FROM form_submissions WHERE contact_id = $1 AND confirmed_at IS NOT NULL`, other); n != 1 {
		t.Errorf("the submission wasn't settled")
	}
}

// siteverify is a fake Cloudflare siteverify: "pass" succeeds, anything else
// fails as a bad answer would.
type siteverify struct {
	mu    sync.Mutex
	calls []url.Values
}

func (s *siteverify) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	s.mu.Lock()
	s.calls = append(s.calls, r.PostForm)
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if r.PostForm.Get("response") == "pass" {
		_, _ = w.Write([]byte(`{"success":true,"hostname":"news.acme.io"}`))
		return
	}
	_, _ = w.Write([]byte(`{"success":false,"error-codes":["invalid-input-response"]}`))
}

// A submission that looks automated waits for Cloudflare Turnstile; the
// answer is checked server-side with the stored secret before anything is
// saved.
func TestTurnstileChallenge(t *testing.T) {
	fake := &siteverify{}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	prev := forms.SiteverifyURL
	forms.SiteverifyURL = srv.URL
	defer func() { forms.SiteverifyURL = prev }()

	env, ids := demo(t)
	challengeID := regexp.MustCompile(`name="challenge" value="([^"]+)"`)
	hold := func(email string) string {
		t.Helper()
		f := url.Values{"_t": {formToken(env, "frm_8c2e1a", 0)}, "email": {email}, "list": {ids.digest.String()}, "consent": {"1"}}
		r := do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe", f, hdr{"Origin": env.Server.URL})
		must(t, r, 200, "One more step")
		m := challengeID.FindStringSubmatch(r.Body)
		if m == nil {
			t.Fatalf("no challenge on the page:\n%s", clip(r.Body))
		}
		return m[1]
	}
	answer := func(id, response string) resp {
		return do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe/challenge", url.Values{"challenge": {id}, "cf-turnstile-response": {response}},
			hdr{"Origin": env.Server.URL})
	}

	t.Run("a person passes and is signed up", func(t *testing.T) {
		id := hold("human.one@fastmail.com")
		must(t, answer(id, "pass"), 200, "Check your inbox", "human.one@fastmail.com")
		who, ok := contactID(t, env, ids.ws, "human.one@fastmail.com")
		if !ok || membership(t, env, who, ids.digest) != "pending" {
			t.Fatal("the checked signup wasn't saved")
		}
		fake.mu.Lock()
		last := fake.calls[len(fake.calls)-1]
		fake.mu.Unlock()
		if last.Get("secret") != "0x4AAAAAAAdemo-secret-not-real" || last.Get("remoteip") != "127.0.0.1" || last.Get("idempotency_key") != id {
			t.Errorf("siteverify call: %v", last)
		}
		// The hold is used up: the same answer can't sign up twice.
		must(t, answer(id, "pass"), 410, "This check expired")
	})
	t.Run("a failed check saves nothing", func(t *testing.T) {
		id := hold("robot.one@fastmail.com")
		must(t, answer(id, "fail"), 403, "We couldn't confirm you're a person.")
		if _, ok := contactID(t, env, ids.ws, "robot.one@fastmail.com"); ok {
			t.Error("a failed check created a contact")
		}
		if n := count(t, env, `SELECT coalesce(sum(count), 0)::bigint FROM form_rejections WHERE form_id = $1 AND reason = 'turnstile'`, ids.signup); n != 1 {
			t.Errorf("%d Turnstile rejections recorded, want 1", n)
		}
	})
	t.Run("another workspace's hold isn't taken", func(t *testing.T) {
		id := hold("cross.ws@fastmail.com")
		r := do(t, env, http.MethodPost, "/p/acme-engineering-blog/subscribe/challenge", url.Values{"challenge": {id}, "cf-turnstile-response": {"pass"}}, nil)
		if r.Status == 200 && strings.Contains(r.Text(), "Check your inbox") {
			t.Fatal("a hold was completed through another workspace")
		}
		if _, ok := contactID(t, env, ids.ws, "cross.ws@fastmail.com"); ok {
			t.Error("the hold signed up through another workspace")
		}
	})
}

// View online renders the email the link was sent with, for its contact.
func TestViewOnline(t *testing.T) {
	env, ids := demo(t)
	var msg uuid.UUID
	var subject string
	err := env.App.Pool.QueryRow(context.Background(), `SELECT m.id, c.subject FROM messages m JOIN campaigns c ON c.id = m.campaign_id
		WHERE m.contact_id = $1 AND c.name = 'October Product Update' LIMIT 1`, ids.sam).Scan(&msg, &subject)
	if err != nil {
		t.Fatalf("Sam's October Product Update: %v", err)
	}
	r := get(t, env, "/p/acme-newsletter/view/"+sign(env, linktoken.ViewOnline, ids.ws, ids.sam, msg))
	if r.Status != 200 || !strings.Contains(r.Body, "<title>") || !strings.Contains(r.Body, `<meta name="robots" content="noindex, nofollow">`) {
		t.Fatalf("view online: %d\n%s", r.Status, clip(r.Body))
	}
	if strings.Contains(r.Body, "{{") {
		t.Error("Liquid tokens were left in the email")
	}
	if !strings.Contains(r.Body, "/p/acme-newsletter/unsubscribe/") && !strings.Contains(r.Body, "/p/acme-newsletter/preferences/") {
		t.Error("the email's footer links weren't filled in for Sam")
	}
	// The publisher's HTML never runs script, and runs in an opaque origin.
	csp := r.Header.Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'none'") || !strings.Contains(csp, "sandbox allow-popups") || strings.Contains(csp, "allow-scripts") || strings.Contains(csp, "allow-same-origin") {
		t.Errorf("view online CSP: %s", csp)
	}
	if r.Header.Get("Cache-Control") != "no-store" || !strings.Contains(r.Header.Get("X-Robots-Tag"), "noindex") {
		t.Errorf("view online headers: %v", r.Header)
	}
	for name, tok := range map[string]string{
		"no message":        sign(env, linktoken.ViewOnline, ids.ws, ids.sam, uuid.Nil),
		"another workspace": sign(env, linktoken.ViewOnline, ids.eng, ids.sam, msg),
		"another kind":      sign(env, linktoken.Preferences, ids.ws, ids.sam, msg),
	} {
		if r := get(t, env, "/p/acme-newsletter/view/"+tok); r.Status != 404 {
			t.Errorf("%s: %d", name, r.Status)
		}
	}
}

// Mail apps POST to the List-Unsubscribe address the sending engine builds
// with App.URLs.Unsubscribe, with no cookie and no CSRF token; a webmail's
// browser may add cross-site fetch metadata.
func TestOneClickAtTheListUnsubscribeAddress(t *testing.T) {
	env, ids := demo(t)
	var msg uuid.UUID
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT id FROM messages WHERE contact_id = $1 AND campaign_id IS NOT NULL ORDER BY queued_at DESC LIMIT 1`, ids.sam).Scan(&msg); err != nil {
		t.Fatal(err)
	}
	address := env.App.URLs.Unsubscribe(wsSlug, sign(env, linktoken.Unsubscribe, ids.ws, ids.sam, msg))
	u, err := url.Parse(address)
	if err != nil || !strings.HasPrefix(address, env.Server.URL+"/p/acme-newsletter/unsubscribe/") {
		t.Fatalf("List-Unsubscribe address %q", address)
	}
	r := do(t, env, http.MethodPost, u.Path, url.Values{"List-Unsubscribe": {"One-Click"}},
		hdr{"Sec-Fetch-Site": "cross-site", "Sec-Fetch-Mode": "no-cors", "Origin": "null", "User-Agent": "GoogleOneClick/1.0"})
	if r.Status != 200 {
		t.Fatalf("one-click: %d %s", r.Status, r.Body)
	}
	if st := one[string](t, env, `SELECT status FROM contacts WHERE id = $1`, ids.sam); st != "unsubscribed" {
		t.Errorf("Sam is %s", st)
	}
	var source string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT source FROM delivery_events WHERE message_id = $1 AND type = 'unsubscribe'`, msg).Scan(&source); err != nil || source != "List-Unsubscribe one-click" {
		t.Errorf("the unsubscribe on Sam's message: %q %v", source, err)
	}
}
