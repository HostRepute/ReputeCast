package publicpages_test

// A Turnstile widget error that a challenge page reports is kept for the
// publisher to see and never turns Turnstile off: any browser can send one,
// with the challenge id of its own suspicious signup. Only Cloudflare
// refusing the stored secret, asked from the server, turns it off.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/forms"
)

// turnstileHold submits the hosted page too quickly, as a script would: the
// submission is held for the Turnstile check, and the page names its id.
func turnstileHold(t *testing.T, env *apptest.Env, ids demoIDs, email string) string {
	t.Helper()
	f := url.Values{"_t": {formToken(env, "frm_8c2e1a", 0)}, "email": {email}, "list": {ids.digest.String()}, "consent": {"1"}}
	r := do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe", f, hdr{"Origin": env.Server.URL})
	m := regexp.MustCompile(`name="challenge" value="([^"]+)"`).FindStringSubmatch(r.Body)
	if r.Status != 200 || m == nil {
		t.Fatalf("no Turnstile check for the signup: %d\n%s", r.Status, clip(r.Body))
	}
	return m[1]
}

func turnstileReport(t *testing.T, env *apptest.Env, challenge, code string) {
	t.Helper()
	if r := do(t, env, http.MethodPost, "/p/acme-newsletter/turnstile-error", url.Values{"challenge": {challenge}, "code": {code}}, nil); r.Status != 204 {
		t.Fatalf("report: %d", r.Status)
	}
}

func TestTurnstileReportKeepsTurnstileOn(t *testing.T) {
	fake := &siteverify{}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	prev := forms.SiteverifyURL
	forms.SiteverifyURL = srv.URL
	defer func() { forms.SiteverifyURL = prev }()

	env, ids := demo(t)
	keyChecks := func() int {
		fake.mu.Lock()
		defer fake.mu.Unlock()
		n := 0
		for _, c := range fake.calls {
			if c.Get("response") == "reputecast-key-check" {
				n++
			}
		}
		return n
	}

	// Mallory's own signup gets her a challenge id to report errors with.
	id := turnstileHold(t, env, ids, "mallory@fastmail.com")
	turnstileReport(t, env, id, "110200")
	turnstileReport(t, env, id, "110200")
	var status, code string
	var reports int
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT turnstile_status, turnstile_reported_code, turnstile_reported_count FROM public_page_settings
		WHERE workspace_id = $1`, ids.ws).Scan(&status, &code, &reports); err != nil {
		t.Fatal(err)
	}
	if status != "ok" || code != "110200" || reports != 2 {
		t.Fatalf("after two reports: status %q, reported %q × %d", status, code, reports)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'public_pages.turnstile_report'`, ids.ws); n != 1 {
		t.Errorf("%d report entries in the audit log, want one per new code", n)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'public_pages.turnstile_error'`, ids.ws); n != 0 {
		t.Errorf("%d Turnstile error entries: a report marked the keys failing", n)
	}
	// The server asked Cloudflare about the secret once, which still works.
	if n := keyChecks(); n != 1 {
		t.Errorf("%d key checks with Cloudflare, want one in 15 minutes", n)
	}
	// Suspicious signups still get the check.
	turnstileHold(t, env, ids, "speedy@fastmail.com")

	// The publisher sees the report beside the keys, in their language.
	type turnstileJSON struct {
		Turnstile struct {
			Status   string `json:"status"`
			Reported *struct {
				Code, Error string
				Count       int
				LastAt      time.Time
			} `json:"reported"`
		} `json:"turnstile"`
	}
	var o turnstileJSON
	c := owner(env)
	c.Do(http.MethodGet, wsAPI+"/public-pages", nil).MustStatus(t, 200).JSON(t, &o)
	r := o.Turnstile.Reported
	if o.Turnstile.Status != "ok" || r == nil || r.Code != "110200" || r.Error != "Cloudflare rejected the site key (error 110200)" || r.Count != 2 ||
		time.Since(r.LastAt) > time.Hour {
		t.Fatalf("the settings' Turnstile: %+v %+v", o.Turnstile, r)
	}

	// A person passing a check shows the widget works: the report is over.
	held := turnstileHold(t, env, ids, "person.passes@fastmail.com")
	must(t, do(t, env, http.MethodPost, "/p/acme-newsletter/subscribe/challenge", url.Values{"challenge": {held}, "cf-turnstile-response": {"pass"}},
		hdr{"Origin": env.Server.URL}), 200, "Check your inbox")
	var after turnstileJSON
	c.Do(http.MethodGet, wsAPI+"/public-pages", nil).MustStatus(t, 200).JSON(t, &after)
	if after.Turnstile.Reported != nil {
		t.Errorf("the report outlived an accepted answer: %+v", after.Turnstile.Reported)
	}
	if n := count(t, env, `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = 'public_pages.turnstile_report_cleared'`, ids.ws); n != 1 {
		t.Errorf("%d cleared entries in the audit log, want 1", n)
	}
}

// A secret Cloudflare refuses, checked from the server after a report,
// turns Turnstile off as before.
func TestTurnstileReportFindsRefusedSecret(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"error-codes":["invalid-input-secret"]}`))
	}))
	defer srv.Close()
	prev := forms.SiteverifyURL
	forms.SiteverifyURL = srv.URL
	defer func() { forms.SiteverifyURL = prev }()

	env, ids := demo(t)
	turnstileReport(t, env, turnstileHold(t, env, ids, "widget.gone@fastmail.com"), "110100")
	var status, problem, actor, meta string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT turnstile_status, turnstile_error FROM public_page_settings WHERE workspace_id = $1`, ids.ws).
		Scan(&status, &problem); err != nil {
		t.Fatal(err)
	}
	if status != "error" || problem != "Cloudflare rejected the secret key (invalid-input-secret)" {
		t.Fatalf("status %q, error %q", status, problem)
	}
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT actor_type, metadata::text FROM audit_events WHERE workspace_id = $1 AND action = 'public_pages.turnstile_error'`,
		ids.ws).Scan(&actor, &meta); err != nil {
		t.Fatal(err)
	}
	if actor != "system" || !regexp.MustCompile(`"widgetError": ?"110100"`).MatchString(meta) {
		t.Errorf("audit entry by %s: %s", actor, meta)
	}
}
