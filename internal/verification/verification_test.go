package verification_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/api"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/audience"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/verification"
)

// fakeKickbox answers like Kickbox's v2 API by the address's local part
// (valid…, invalid…, catchall…, role…, disposable…, unknown…), and can fail
// after a number of checks.
type fakeKickbox struct {
	mu         sync.Mutex
	checks     int
	failAfter  int // fail every check after this many (0: never)
	failStatus int
	failBody   string
	delay      time.Duration
	balance    int64
}

func (f *fakeKickbox) set(fn func(f *fakeKickbox)) {
	f.mu.Lock()
	fn(f)
	f.mu.Unlock()
}

func (f *fakeKickbox) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	f.mu.Lock()
	delay, balance := f.delay, f.balance
	f.mu.Unlock()
	if r.URL.Path == "/v2/balance" {
		_, _ = fmt.Fprintf(w, `{"balance":%d,"success":true}`, balance)
		return
	}
	if delay > 0 {
		time.Sleep(delay)
	}
	f.mu.Lock()
	f.checks++
	n, after, status, body := f.checks, f.failAfter, f.failStatus, f.failBody
	if after == 0 || n <= after {
		f.balance--
	}
	f.mu.Unlock()
	if after > 0 && n > after {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
		return
	}
	email := r.URL.Query().Get("email")
	local := strings.SplitN(email, "@", 2)[0]
	a := map[string]any{"result": "deliverable", "reason": "accepted_email", "email": email, "success": true}
	switch {
	case strings.HasPrefix(local, "invalid"):
		a["result"], a["reason"] = "undeliverable", "rejected_email"
	case strings.HasPrefix(local, "catchall"):
		a["result"], a["reason"], a["accept_all"] = "risky", "low_deliverability", true
	case strings.HasPrefix(local, "role"):
		a["role"] = true
	case strings.HasPrefix(local, "disposable"):
		a["result"], a["reason"], a["disposable"] = "risky", "low_quality", true
	case strings.HasPrefix(local, "unknown"):
		a["result"], a["reason"] = "unknown", "no_connect"
	}
	_ = json.NewEncoder(w).Encode(a)
}

// fakeZeroBounce answers ZeroBounce's batch endpoint, with a spam trap.
func fakeZeroBounce() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/getcredits":
			_, _ = w.Write([]byte(`{"Credits":52115}`))
		case "/v2/validatebatch":
			var in struct {
				Batch []struct {
					Email string `json:"email_address"`
				} `json:"email_batch"`
			}
			_ = json.NewDecoder(r.Body).Decode(&in)
			out := []map[string]string{}
			for _, e := range in.Batch {
				status, sub := "valid", ""
				local := strings.SplitN(e.Email, "@", 2)[0]
				switch {
				case strings.HasPrefix(local, "invalid"):
					status, sub = "invalid", "mailbox_not_found"
				case strings.HasPrefix(local, "trap"):
					status = "spamtrap"
				case strings.HasPrefix(local, "catchall"):
					status = "catch-all"
				case strings.HasPrefix(local, "role"):
					status, sub = "do_not_mail", "role_based"
				case strings.HasPrefix(local, "unknown"):
					status, sub = "unknown", "greylisted"
				}
				out = append(out, map[string]string{"address": e.Email, "status": status, "sub_status": sub})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"email_batch": out, "errors": []any{}})
		default:
			http.NotFound(w, r)
		}
	})
}

type audienceAdapter struct{ a *audience.Service }

func (x audienceAdapter) SelectionIDs(ctx context.Context, ws *store.Workspace, sel verification.Selection) ([]uuid.UUID, error) {
	if len(sel.Tiers) > 0 {
		var out []uuid.UUID
		for _, t := range sel.Tiers {
			ids, err := x.a.SelectionIDs(ctx, ws, audience.Selection{AllMatching: true, Filter: audience.ContactFilter{Tier: t}})
			if err != nil {
				return nil, err
			}
			out = append(out, ids...)
		}
		return out, nil
	}
	ids := make([]string, len(sel.ContactIDs))
	for i, id := range sel.ContactIDs {
		ids[i] = id.String()
	}
	return x.a.SelectionIDs(ctx, ws, audience.Selection{IDs: ids})
}

type fakeImports struct{ emails []string }

func (f fakeImports) NewAddresses(context.Context, *store.Workspace, uuid.UUID) (string, []string, error) {
	return `Mailchimp “Acme Customers”`, f.emails, nil
}

type fixture struct {
	env  *apptest.Env
	ws   store.Workspace
	user store.User
	ctx  context.Context
	svc  *verification.Service
}

func setup(t *testing.T, bases map[string]string) *fixture {
	t.Helper()
	env := apptest.New(t)
	u := env.CreateUser(apptest.UserOpts{Email: "owner@verify.test", Name: "Vera Owner"})
	ws := env.CreateWorkspace("Verify Test", u.ID)
	p := &authz.Principal{Type: authz.ActorUser, ID: u.ID, UserID: u.ID, Name: u.Name, WorkspaceID: ws.ID, RoleKey: authz.RoleOwner,
		Permissions: authz.NewSet(authz.SubscribersVerify, authz.IntegrationsManage, authz.SubscribersExport, authz.FormsEdit, authz.SubscribersView)}
	svc := &verification.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, Inline: true,
		BaseURLs: bases, HTTP: &http.Client{Timeout: 10 * time.Second},
		Audience: audienceAdapter{a: &audience.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit}}}
	return &fixture{env: env, ws: ws, user: u, ctx: api.WithPrincipal(context.Background(), p), svc: svc}
}

// contacts adds contacts in a tier and returns their ids by address.
func (f *fixture) contacts(t *testing.T, tier string, emails ...string) map[string]uuid.UUID {
	t.Helper()
	out := map[string]uuid.UUID{}
	for _, e := range emails {
		id := uuid.Must(uuid.NewV7())
		if _, err := f.env.App.Pool.Exec(context.Background(), `INSERT INTO contacts (id, workspace_id, email, tier) VALUES ($1, $2, $3, $4)`, id, f.ws.ID, e, tier); err != nil {
			t.Fatal(err)
		}
		out[e] = id
	}
	return out
}

func (f *fixture) connect(t *testing.T, service string, price float64, bulk, forms bool) verification.VerificationServiceView {
	t.Helper()
	v, err := f.svc.Connect(f.ctx, &f.ws, verification.VerificationConnectInput{
		VerificationKeyInput: verification.VerificationKeyInput{Service: service, APIKey: "test-key-0f3c8a21-" + service},
		Price:                price, DefaultBulk: bulk, DefaultForms: forms})
	if err != nil {
		t.Fatalf("connect %s: %v", service, err)
	}
	return v
}

func (f *fixture) audits(t *testing.T, action string) int {
	t.Helper()
	var n int
	if err := f.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = $2`, f.ws.ID, action).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestConnectEstimateAndNoSecret(t *testing.T) {
	kb := &fakeKickbox{balance: 36656}
	srv := httptest.NewServer(kb)
	defer srv.Close()
	f := setup(t, map[string]string{"kickbox": srv.URL})
	f.contacts(t, "sunset", "valid1@x.test", "valid2@x.test", "invalid1@x.test")

	v := f.connect(t, "kickbox", 0.008, true, false)
	if !v.Connected || v.Balance == nil || *v.Balance != 36656 || v.Price != 0.008 || !v.DefaultBulk {
		t.Fatalf("connected view = %+v", v)
	}
	if f.audits(t, "verification.service_connect") != 1 {
		t.Fatal("connecting isn't audited")
	}
	o, err := f.svc.Overview(f.ctx, &f.ws)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(o)
	if strings.Contains(string(b), "test-key-0f3c8a21") {
		t.Fatal("the overview returns the API key")
	}

	e, err := f.svc.Estimate(f.ctx, &f.ws, verification.VerificationEstimateInput{Target: verification.VerificationTarget{Kind: "tier", Tiers: []string{"sunset"}}})
	if err != nil {
		t.Fatal(err)
	}
	if e.Label != "Sunset" || e.Contacts != 3 || e.ToCheck != 3 || e.Cost != 0.02 || e.After == nil || *e.After != 36653 || e.Service != "kickbox" {
		t.Fatalf("estimate = %+v", e)
	}

	// The import preview's card: the new addresses at the plan's price.
	emails := make([]string, 11844)
	for i := range emails {
		emails[i] = fmt.Sprintf("new%05d@x.test", i)
	}
	f.svc.Imports = fakeImports{emails: emails}
	e, err = f.svc.Estimate(f.ctx, &f.ws, verification.VerificationEstimateInput{Target: verification.VerificationTarget{Kind: "import", ImportID: uuid.NewString()}})
	if err != nil {
		t.Fatal(err)
	}
	if e.ToCheck != 11844 || e.Cost != 94.75 || e.Label != `Mailchimp “Acme Customers”` {
		t.Fatalf("import estimate = %+v (want 11,844 × $0.008 = $94.75)", e)
	}

	// A rejected key is refused with the service's status and nothing is saved.
	kb.set(func(f *fakeKickbox) {})
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"success":false,"message":"Unknown API key"}`))
	}))
	defer bad.Close()
	f.svc.BaseURLs["neverbounce"] = bad.URL
	f.svc.BaseURLs["bouncer"] = bad.URL
	if _, err := f.svc.Connect(f.ctx, &f.ws, verification.VerificationConnectInput{VerificationKeyInput: verification.VerificationKeyInput{Service: "bouncer", APIKey: "rejected-key-1234"}}); err == nil || !strings.Contains(err.Error(), "rejected this key (403)") {
		t.Fatalf("a rejected key = %v", err)
	}
}

func TestBulkJobSuppressesInvalidAndTagsRisky(t *testing.T) {
	f := setup(t, map[string]string{"zerobounce": httptest.NewServer(fakeZeroBounce()).URL})
	ids := f.contacts(t, "sunset", "valid1@x.test", "invalid1@x.test", "trap1@x.test", "catchall1@x.test", "role1@x.test", "unknown1@x.test")
	f.connect(t, "zerobounce", 0.008, true, false)

	j, err := f.svc.Start(f.ctx, &f.ws, verification.VerificationStartInput{Target: verification.VerificationTarget{Kind: "tier", Tiers: []string{"sunset"}}, Origin: "list_health"})
	if err != nil {
		t.Fatal(err)
	}
	j, _ = f.svc.Job(f.ctx, &f.ws, uuid.MustParse(j.ID))
	if j.Status != "finished" || j.Checked != 6 || j.Valid != 1 || j.Invalid != 2 || j.Risky != 2 || j.Unknown != 1 || j.Suppressed != 2 {
		t.Fatalf("job = %+v", j)
	}
	if j.Reasons["spam_trap"] != 1 || j.Reasons["invalid_address"] != 1 || j.Reasons["catch_all"] != 1 || j.Reasons["role"] != 1 {
		t.Fatalf("reasons = %v", j.Reasons)
	}
	if j.RecheckFrom == nil {
		t.Error("unknown results say when they can be checked again")
	}
	// The jobs tray shows it for two hours after it ended.
	tray, err := f.svc.TrayJobs(f.ctx, &f.ws)
	if err != nil || len(tray) != 1 || tray[0].Status != "done" || tray[0].Title != "Verified the Sunset tier with ZeroBounce" || tray[0].Detail != "2 suppressed, 2 tagged risky" {
		t.Fatalf("tray = %+v, %v", tray, err)
	}

	type sup struct{ reason, source, code, scope string }
	rows, err := f.env.App.Pool.Query(context.Background(), `SELECT email, reason, source, original_code, scope FROM suppressions WHERE workspace_id = $1`, f.ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]sup{}
	for rows.Next() {
		var e string
		var s sup
		if err := rows.Scan(&e, &s.reason, &s.source, &s.code, &s.scope); err != nil {
			t.Fatal(err)
		}
		got[e] = s
	}
	rows.Close()
	if s := got["invalid1@x.test"]; s.reason != "invalid" || s.source != "verify:zerobounce" || s.code != "ZeroBounce: invalid / mailbox_not_found" || s.scope != "all" {
		t.Errorf("invalid address suppression = %+v", s)
	}
	if s := got["trap1@x.test"]; s.reason != "spam_trap" || s.code != "ZeroBounce: spamtrap" {
		t.Errorf("spam trap suppression = %+v", s)
	}
	if len(got) != 2 {
		t.Errorf("suppressed %d addresses, want only the 2 invalid ones", len(got))
	}
	var status string
	_ = f.env.App.Pool.QueryRow(context.Background(), `SELECT status FROM contacts WHERE id = $1`, ids["invalid1@x.test"]).Scan(&status)
	if status != "suppressed" {
		t.Errorf("an invalid address's contact is %s, want suppressed", status)
	}
	tags := func(e string) []string {
		var ts []string
		_ = f.env.App.Pool.QueryRow(context.Background(), `SELECT tags FROM contacts WHERE id = $1`, ids[e]).Scan(&ts)
		return ts
	}
	if ts := tags("catchall1@x.test"); len(ts) != 1 || ts[0] != "verify:catch-all" {
		t.Errorf("catch-all tags = %v", ts)
	}
	if ts := tags("unknown1@x.test"); len(ts) != 1 || ts[0] != "verify:unknown" {
		t.Errorf("unknown tags = %v", ts)
	}
	if ts := tags("valid1@x.test"); len(ts) != 0 {
		t.Errorf("a valid address got tags %v", ts)
	}

	c, err := f.svc.ContactCheck(f.ctx, &f.ws, ids["invalid1@x.test"])
	if err != nil || !c.Checked || c.Result != "invalid" || !c.Suppressed || c.ServiceName != "ZeroBounce" {
		t.Errorf("profile row = %+v, %v", c, err)
	}
	if f.audits(t, "verification.job_start") != 1 || f.audits(t, "verification.job_finish") != 1 {
		t.Error("the job's start and results aren't audited")
	}

	// Checked in the last 90 days: skipped the next time.
	e, err := f.svc.Estimate(f.ctx, &f.ws, verification.VerificationEstimateInput{Target: verification.VerificationTarget{Kind: "tier", Tiers: []string{"sunset"}}})
	if err != nil || e.Skipped != 6 || e.ToCheck != 0 || e.SkippedService != "ZeroBounce" {
		t.Fatalf("estimate after the check = %+v, %v", e, err)
	}

	// The results CSV holds addresses: an export, audited.
	csv, _, err := f.svc.ResultsCSV(f.ctx, &f.ws, uuid.MustParse(j.ID))
	if err != nil || !strings.Contains(string(csv), "invalid1@x.test,invalid,invalid_address,ZeroBounce: invalid / mailbox_not_found") {
		t.Fatalf("csv = %s, %v", csv, err)
	}
	if f.audits(t, "verification.results_export") != 1 {
		t.Error("the results download isn't audited")
	}
}

func TestPartialResultsOutageAndCredits(t *testing.T) {
	kb := &fakeKickbox{balance: 9214, failAfter: 30, failStatus: http.StatusServiceUnavailable, failBody: `<html>503</html>`}
	srv := httptest.NewServer(kb)
	defer srv.Close()
	f := setup(t, map[string]string{"kickbox": srv.URL})
	emails := []string{}
	for i := 0; i < 60; i++ {
		emails = append(emails, fmt.Sprintf("valid%02d@x.test", i))
	}
	f.contacts(t, "at_risk", emails...)
	f.connect(t, "kickbox", 0.010, true, true)

	j, err := f.svc.Start(f.ctx, &f.ws, verification.VerificationStartInput{Target: verification.VerificationTarget{Kind: "tier", Tiers: []string{"at_risk"}}})
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.MustParse(j.ID)
	j, _ = f.svc.Job(f.ctx, &f.ws, id)
	if j.Status != "paused" || j.Checked == 0 || j.Checked >= 60 || j.RetryUntil == nil || j.NextTryAt == nil {
		t.Fatalf("after an outage mid-way the job pauses with its partial results: %+v", j)
	}
	partial := j.Checked
	o, _ := f.svc.Overview(f.ctx, &f.ws)
	for _, s := range o.Services {
		if s.Key == "kickbox" && (!s.Down || s.LastErrorSince == nil) {
			t.Errorf("the service shows its error: %+v", s)
		}
	}

	// Cancelling keeps the results so far.
	if j, err = f.svc.Cancel(f.ctx, &f.ws, id); err != nil || j.Status != "cancelled" || j.Checked != partial {
		t.Fatalf("cancel = %+v, %v", j, err)
	}
	if f.audits(t, "verification.job_cancel") != 1 {
		t.Error("cancelling isn't audited")
	}

	// Out of credits: the job stops, keeps its results and resumes later.
	kb.set(func(k *fakeKickbox) {
		k.checks, k.failAfter, k.failStatus, k.failBody = 0, 10, http.StatusForbidden, `{"success":false,"message":"Insufficient balance"}`
	})
	j, err = f.svc.Start(f.ctx, &f.ws, verification.VerificationStartInput{Target: verification.VerificationTarget{Kind: "tier", Tiers: []string{"at_risk"}}})
	if err != nil {
		t.Fatal(err)
	}
	id = uuid.MustParse(j.ID)
	j, _ = f.svc.Job(f.ctx, &f.ws, id)
	if j.Status != "stopped" || j.StopReason != "no_credits" || j.Checked == 0 || j.Checked >= j.Total {
		t.Fatalf("out of credits = %+v", j)
	}
	kb.set(func(k *fakeKickbox) { k.failAfter, k.balance = 0, 5000 })
	if j, err = f.svc.Resume(f.ctx, &f.ws, id); err != nil {
		t.Fatal(err)
	}
	if j.Status != "finished" || j.Checked != j.Total {
		t.Fatalf("resumed job = %+v", j)
	}
}

func TestRealtimeCheckFailsOpen(t *testing.T) {
	kb := &fakeKickbox{balance: 9214, delay: 4 * time.Second}
	srv := httptest.NewServer(kb)
	defer srv.Close()
	f := setup(t, map[string]string{"kickbox": srv.URL})
	f.connect(t, "kickbox", 0.010, false, true)
	form := uuid.NewString()
	if _, err := f.svc.SetFormCheck(f.ctx, &f.ws, form, "Newsletter signup", verification.VerificationFormInput{Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if f.audits(t, "verification.realtime_on") != 1 {
		t.Fatal("turning real-time checks on isn't audited")
	}

	// The service takes longer than 3 seconds: the signup goes through.
	start := time.Now()
	c := f.svc.CheckSignup(context.Background(), &f.ws, form, "valid@x.test")
	if !c.NoAnswer || c.Checked() || time.Since(start) > 3500*time.Millisecond {
		t.Fatalf("a slow service = %+v after %v", c, time.Since(start))
	}
	var waiting int
	_ = f.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM verification_realtime_pending WHERE workspace_id = $1`, f.ws.ID).Scan(&waiting)
	if waiting != 1 {
		t.Fatalf("%d signups wait for a check, want 1", waiting)
	}

	// Answering again: an invalid address is asked to fix, a risky one tagged.
	kb.set(func(k *fakeKickbox) { k.delay = 0 })
	if c := f.svc.CheckSignup(context.Background(), &f.ws, form, "invalid@x.test"); !c.Checked() || c.Answer.Result != "invalid" {
		t.Fatalf("an invalid address = %+v", c)
	}
	ids := f.contacts(t, "active", "catchall@x.test", "valid@x.test")
	c = f.svc.CheckSignup(context.Background(), &f.ws, form, "catchall@x.test")
	f.svc.SignedUp(context.Background(), &f.ws, ids["catchall@x.test"], c)
	var tags []string
	_ = f.env.App.Pool.QueryRow(context.Background(), `SELECT tags FROM contacts WHERE id = $1`, ids["catchall@x.test"]).Scan(&tags)
	if len(tags) != 1 || tags[0] != "verify:catch-all" {
		t.Fatalf("risky signup tags = %v", tags)
	}

	// The waiting signup is checked once the service answers; counts per day and form.
	if err := f.svc.CheckPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	_ = f.env.App.Pool.QueryRow(context.Background(), `SELECT count(*) FROM verification_realtime_pending WHERE workspace_id = $1`, f.ws.ID).Scan(&waiting)
	if waiting != 0 {
		t.Errorf("%d signups still wait", waiting)
	}
	rt, err := f.svc.Realtime(f.ctx, &f.ws, "kickbox")
	if err != nil {
		t.Fatal(err)
	}
	if rt.Checks != 3 || rt.Total.NoAnswer != 1 || rt.Total.Invalid != 1 || rt.Total.Risky != 1 || rt.AskedToFix != 1 {
		t.Fatalf("real-time counts = %+v", rt)
	}
	if f.audits(t, "verification.job_start") != 0 {
		t.Error("real-time checks are counted, not audited one by one")
	}
}

func TestPermissionsOverHTTP(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	viewer := env.Client()
	viewer.SignInDemo("aisha@acme.io")
	body := map[string]any{"target": map[string]any{"kind": "tier", "tiers": []string{"sunset"}}}
	viewer.Do("POST", "/workspaces/acme-newsletter/verification/jobs", body).MustStatus(t, http.StatusForbidden)
	viewer.Do("POST", "/workspaces/acme-newsletter/verification/services", map[string]any{"service": "neverbounce", "apiKey": "nb-test-key-123456"}).MustStatus(t, http.StatusForbidden)
	viewer.Do("GET", "/workspaces/acme-newsletter/verification/services", nil).MustStatus(t, http.StatusOK)

	// Path parameters reach the handlers (the demo's Kickbox checks the forms).
	owner := env.Client()
	owner.SignInDemo("linh@acme.io")
	var form verification.VerificationFormSetting
	owner.Do("PUT", "/workspaces/acme-newsletter/verification/forms/subscribe_page?name=Subscribe%20page", map[string]any{"enabled": false}).
		MustStatus(t, http.StatusOK).JSON(t, &form)
	if form.FormKey != "subscribe_page" || form.Enabled || form.Service != "kickbox" {
		t.Fatalf("form setting = %+v", form)
	}
	var svc verification.VerificationServiceView
	owner.Do("GET", "/workspaces/acme-newsletter/verification/services/zerobounce", nil).MustStatus(t, http.StatusOK).JSON(t, &svc)
	if svc.Key != "zerobounce" || !svc.Connected || svc.Balance == nil || *svc.Balance != 36656 {
		t.Fatalf("ZeroBounce = %+v", svc)
	}
}
