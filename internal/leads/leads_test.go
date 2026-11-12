package leads_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/growth"
	"github.com/hostrepute/reputecast/internal/leads"
	"github.com/hostrepute/reputecast/internal/leads/meta/metatest"
	"github.com/hostrepute/reputecast/internal/store"
)

type fixture struct {
	env   *apptest.Env
	owner *apptest.Client
	ws    store.Workspace
	path  string
	fake  *metatest.Fake
	list  uuid.UUID
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	env := apptest.New(t)
	u := env.CreateUser(apptest.UserOpts{Email: "owner@m66leads.test", Name: "Pat Owner"})
	ws := env.CreateWorkspace("M66 Lead Ads Test", u.ID)
	c := env.Client()
	c.SignIn(u.Email, "")
	list := uuid.Must(uuid.NewV7())
	if _, err := env.App.Pool.Exec(context.Background(), `INSERT INTO lists (id, workspace_id, name, description, created_at) VALUES ($1,$2,'Weekly Digest','',now())`, list, ws.ID); err != nil {
		t.Fatal(err)
	}
	fake := env.App.LeadAdsFake()
	// Like Meta, subscribing the app calls the callback's verification.
	fake.Callback = env.App.Handler()
	return &fixture{env: env, owner: c, ws: ws, path: "/workspaces/" + ws.Slug + "/lead-ads", fake: fake, list: list}
}

func one[T any](t *testing.T, env *apptest.Env, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

func (f *fixture) do(t *testing.T, method, path string, body []byte, header map[string]string) (int, string) {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), method, f.env.Server.URL+path, bytes.NewReader(body))
	for k, v := range header {
		req.Header.Set(k, v)
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Body.Close() }()
	b, _ := io.ReadAll(r.Body)
	return r.StatusCode, string(b)
}

func pathOf(t *testing.T, raw string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u.Path
}

// connect connects the demo's Meta app, chooses Acme Inc and maps Spring
// webinar signup (double opt-in).
func (f *fixture) connect(t *testing.T) leads.LeadAdsConnection {
	t.Helper()
	var conn leads.LeadAdsConnection
	f.owner.Do(http.MethodPost, f.path+"/connections", map[string]any{"name": "Acme Marketing", "appId": demo.LeadAdsAppID,
		"appSecret": demo.LeadAdsAppSecret, "token": demo.LeadAdsToken}).MustStatus(t, 200).JSON(t, &conn)
	if conn.Receiving.SubscribedAt == nil || conn.Webhook == nil {
		t.Fatalf("connect: Meta's verification of the callback should pass: %+v", conn.Receiving)
	}
	f.owner.Do(http.MethodPut, f.path+"/connections/"+conn.ID+"/pages", map[string]any{"pageIds": []string{"102030405060001"}}).MustStatus(t, 200).JSON(t, &conn)
	if len(conn.Pages) != 1 || len(conn.Pages[0].Forms) != 2 {
		t.Fatalf("pages: %+v", conn.Pages)
	}
	return conn
}

func formID(t *testing.T, conn leads.LeadAdsConnection, name string) string {
	t.Helper()
	for _, p := range conn.Pages {
		for _, fm := range p.Forms {
			if fm.Name == name {
				return fm.ID
			}
		}
	}
	t.Fatalf("no form %s", name)
	return ""
}

// deliver posts a signed leadgen webhook for a new lead and applies it.
func (f *fixture) deliver(t *testing.T, cb, formMetaID string, answers map[string]string, checked ...string) (string, int) {
	t.Helper()
	lead := f.fake.AddLead("102030405060001", formMetaID, time.Now().Add(-time.Minute), answers, checked...)
	body, sig := metatest.LeadWebhook(demo.LeadAdsAppSecret, "102030405060001", formMetaID, lead, time.Now())
	st, _ := f.do(t, http.MethodPost, cb, body, map[string]string{"X-Hub-Signature-256": sig, "Content-Type": "application/json"})
	f.applyPending(t)
	return lead, st
}

func (f *fixture) applyPending(t *testing.T) {
	t.Helper()
	rows, err := f.env.App.Pool.Query(context.Background(), `SELECT id FROM lead_events WHERE workspace_id = $1 AND status = 'pending'`, f.ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if err := f.env.App.LeadAds().ApplyEvent(context.Background(), id); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLeadAdsWebhook(t *testing.T) {
	f := newFixture(t)
	conn := f.connect(t)
	cb := pathOf(t, conn.Webhook.CallbackURL)
	webinar := formID(t, conn, "Spring webinar signup")
	f.owner.Do(http.MethodPut, f.path+"/forms/"+webinar+"/mapping", map[string]any{"mapping": map[string]string{"email": "email", "full_name": "full_name"},
		"listIds": []string{f.list.String()}, "tags": []string{"Webinar-2026"}, "consentMode": "double"}).MustStatus(t, 200)

	// Meta's verification request: the verify token decides.
	q := url.Values{"hub.mode": {"subscribe"}, "hub.verify_token": {"wrong"}, "hub.challenge": {"42"}}
	if st, _ := f.do(t, http.MethodGet, cb+"?"+q.Encode(), nil, nil); st != http.StatusForbidden {
		t.Fatalf("wrong verify token: %d", st)
	}
	q.Set("hub.verify_token", conn.Webhook.VerifyToken)
	if st, body := f.do(t, http.MethodGet, cb+"?"+q.Encode(), nil, nil); st != http.StatusOK || body != "42" {
		t.Fatalf("verification: %d %q", st, body)
	}
	if st, _ := f.do(t, http.MethodGet, "/hooks/meta-leads/"+uuid.NewString()+uuid.NewString(), nil, nil); st != http.StatusNotFound {
		t.Fatalf("unknown token: %d", st)
	}

	// A bad signature: 401, kept for diagnosis, never applied.
	lead := f.fake.AddLead("102030405060001", "1180000000001", time.Now(), map[string]string{"email": "nope@example.com"})
	body, _ := metatest.LeadWebhook(demo.LeadAdsAppSecret, "102030405060001", "1180000000001", lead, time.Now())
	if st, _ := f.do(t, http.MethodPost, cb, body, map[string]string{"X-Hub-Signature-256": "sha256=" + string(bytes.Repeat([]byte("0"), 64))}); st != http.StatusUnauthorized {
		t.Fatalf("bad signature: %d", st)
	}
	if n := one[int64](t, f.env, `SELECT count(*) FROM lead_events WHERE workspace_id = $1 AND NOT signature_ok AND status = 'rejected'`, f.ws.ID); n != 1 {
		t.Fatalf("rejected deliveries kept: %d", n)
	}

	// A valid lead: a pending contact (double opt-in) with the form's consent
	// source, its tag, and the growth source's kind.
	lead, st := f.deliver(t, cb, "1180000000001", map[string]string{"email": "Priya.Shah@example.com", "full_name": "Priya Shah"})
	if st != http.StatusOK {
		t.Fatalf("delivery: %d", st)
	}
	outcome := one[string](t, f.env, `SELECT outcome FROM lead_entries WHERE leadgen_id = $1`, lead)
	if outcome != leads.OutcomePending {
		t.Fatalf("outcome %s", outcome)
	}
	if n := one[int64](t, f.env, `SELECT count(*) FROM consent_events e JOIN contacts c ON c.id = e.contact_id
		WHERE c.workspace_id = $1 AND c.email = 'priya.shah@example.com' AND e.source = $2`, f.ws.ID, "lead_ad:"+webinar); n == 0 {
		t.Fatal("no consent record with the lead form's source")
	}
	if tags := one[[]string](t, f.env, `SELECT tags FROM contacts WHERE workspace_id = $1 AND email = 'priya.shah@example.com'`, f.ws.ID); len(tags) != 1 || tags[0] != "webinar-2026" {
		t.Fatalf("tags %v", tags)
	}
	if first := one[string](t, f.env, `SELECT first_name FROM contacts WHERE workspace_id = $1 AND email = 'priya.shah@example.com'`, f.ws.ID); first != "Priya" {
		t.Fatalf("first name %q", first)
	}

	// The same lead again (Meta retries): one entry.
	body, sig := metatest.LeadWebhook(demo.LeadAdsAppSecret, "102030405060001", "1180000000001", lead, time.Now())
	f.do(t, http.MethodPost, cb, body, map[string]string{"X-Hub-Signature-256": sig})
	f.applyPending(t)
	if n := one[int64](t, f.env, `SELECT count(*) FROM lead_entries WHERE leadgen_id = $1`, lead); n != 1 {
		t.Fatalf("entries for one lead: %d", n)
	}

	// Meta's test lead creates nobody.
	test, _ := f.deliver(t, cb, "1180000000001", map[string]string{"email": "test@meta.com", "full_name": "<test lead: dummy data for full_name>"})
	if o := one[string](t, f.env, `SELECT outcome FROM lead_entries WHERE leadgen_id = $1`, test); o != leads.OutcomeTest {
		t.Fatalf("test lead outcome %s", o)
	}

	// An unmapped form's lead is counted, not stored.
	unmapped, _ := f.deliver(t, cb, "1180000000002", map[string]string{"email": "tom@example.com"})
	if n := one[int64](t, f.env, `SELECT count(*) FROM lead_entries WHERE leadgen_id = $1`, unmapped); n != 0 {
		t.Fatal("an unmapped form's lead was stored")
	}
	if w := one[int32](t, f.env, `SELECT waiting FROM lead_forms WHERE form_id = '1180000000002' AND workspace_id = $1`, f.ws.ID); w != 1 {
		t.Fatalf("waiting %d", w)
	}
	// Every change by a person is audited.
	for _, action := range []string{"lead_connection.created", "lead_connection.updated", "lead_form.mapped"} {
		if n := one[int64](t, f.env, `SELECT count(*) FROM audit_events WHERE workspace_id = $1 AND action = $2`, f.ws.ID, action); n == 0 {
			t.Fatalf("no audit entry %s", action)
		}
	}
}

func TestLeadAdsSingleOptInAndHealth(t *testing.T) {
	f := newFixture(t)
	conn := f.connect(t)
	cb := pathOf(t, conn.Webhook.CallbackURL)
	guide := formID(t, conn, "Field Guide download")
	// Single opt-in needs the form's consent checkbox.
	f.owner.Do(http.MethodPut, f.path+"/forms/"+guide+"/mapping", map[string]any{"mapping": map[string]string{"email": "email", "first_name": "first_name"},
		"listIds": []string{f.list.String()}, "consentMode": "single", "consentKey": "marketing_optin"}).MustStatus(t, 200)
	webinar := formID(t, conn, "Spring webinar signup")
	f.owner.Do(http.MethodPut, f.path+"/forms/"+webinar+"/mapping", map[string]any{"mapping": map[string]string{"email": "email"},
		"listIds": []string{f.list.String()}, "consentMode": "single"}).MustStatus(t, 422)

	ticked, _ := f.deliver(t, cb, "1180000000002", map[string]string{"email": "tom.becker@example.com", "first_name": "Tom"}, "marketing_optin")
	if o := one[string](t, f.env, `SELECT outcome FROM lead_entries WHERE leadgen_id = $1`, ticked); o != leads.OutcomeCreated {
		t.Fatalf("ticked: %s", o)
	}
	unticked, _ := f.deliver(t, cb, "1180000000002", map[string]string{"email": "no.consent@example.com"})
	if o := one[string](t, f.env, `SELECT outcome FROM lead_entries WHERE leadgen_id = $1`, unticked); o != leads.OutcomeDeclined {
		t.Fatalf("unticked: %s", o)
	}
	if n := one[int64](t, f.env, `SELECT count(*) FROM automation_events WHERE workspace_id = $1 AND kind = 'lead_ad_submitted'`, f.ws.ID); n != 1 {
		t.Fatalf("trigger events: %d", n)
	}

	// Tom joined at once: Growth counts him under "Lead ads · Field Guide
	// download" (the lead_ad kind, named after the form).
	ctx := context.Background()
	g := &growth.Service{Pool: f.env.App.Pool, Q: f.env.App.Q, Audit: f.env.App.Audit}
	if err := g.Attribute(ctx, &f.ws); err != nil {
		t.Fatal(err)
	}
	if src := one[string](t, f.env, `SELECT g.source_kind || ':' || g.source_id FROM contact_growth g JOIN contacts c ON c.id = g.contact_id
		WHERE c.workspace_id = $1 AND c.email = 'tom.becker@example.com'`, f.ws.ID); src != "lead_ad:"+guide {
		t.Fatalf("growth source %s", src)
	}
	var sources growth.GrowthSourceList
	f.owner.Do(http.MethodGet, "/workspaces/"+f.ws.Slug+"/growth/sources", nil).MustStatus(t, 200).JSON(t, &sources)
	named := false
	for _, it := range sources.Items {
		if it.Source.Kind == "lead_ad" && it.Source.Name == "Field Guide download" && it.Contacts == 1 {
			named = true
		}
	}
	if !named {
		t.Fatalf("growth sources: %+v", sources.Items)
	}

	// No delivery for 24 hours while forms are active: the health check
	// polls each active form.
	if _, err := f.env.App.Pool.Exec(ctx, `UPDATE lead_connections SET subscribed_at = now() - interval '2 days', last_event_at = now() - interval '25 hours' WHERE workspace_id = $1`, f.ws.ID); err != nil {
		t.Fatal(err)
	}
	polled := f.fake.AddLead("102030405060001", "1180000000002", time.Now().Add(-time.Hour), map[string]string{"email": "polled@example.com"}, "marketing_optin")
	c := one[uuid.UUID](t, f.env, `SELECT id FROM lead_connections WHERE workspace_id = $1`, f.ws.ID)
	row, err := f.env.App.Q.LeadConnectionAny(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.env.App.LeadAds().Check(ctx, row, true); err != nil {
		t.Fatal(err)
	}
	if src := one[string](t, f.env, `SELECT source FROM lead_entries WHERE leadgen_id = $1`, polled); src != leads.SourcePoll {
		t.Fatalf("polled lead source %s", src)
	}
	if r := one[string](t, f.env, `SELECT polling_reason FROM lead_connections WHERE id = $1`, c); r != leads.PollingQuiet {
		t.Fatalf("polling reason %q", r)
	}

	// A refused token: attention and one notification.
	f.fake.ExpireToken()
	t.Cleanup(func() { f.fake.Renew(demo.LeadAdsToken) })
	row, _ = f.env.App.Q.LeadConnectionAny(ctx, c)
	if err := f.env.App.LeadAds().Check(ctx, row, true); err != nil {
		t.Fatal(err)
	}
	if st := one[string](t, f.env, `SELECT status || ':' || status_detail FROM lead_connections WHERE id = $1`, c); st != "attention:token_expired" {
		t.Fatalf("status %s", st)
	}
	if n := one[int64](t, f.env, `SELECT count(*) FROM notifications WHERE workspace_id = $1 AND kind = 'lead_ads_attention'`, f.ws.ID); n == 0 {
		t.Fatal("no notification")
	}
}

// TestLeadAdsAccess: a Viewer reads the page without the webhook's tokens
// and can't change anything; no response ever carries the app secret or a
// token.
func TestLeadAdsAccess(t *testing.T) {
	f := newFixture(t)
	conn := f.connect(t)
	viewer := f.env.CreateUser(apptest.UserOpts{Email: "viewer@m66leads.test", Name: "Vic Viewer"})
	f.env.AddMember(f.ws, viewer.ID, "viewer")
	v := f.env.Client()
	v.SignIn(viewer.Email, "")

	var seen leads.LeadAdsConnection
	v.Do(http.MethodGet, f.path+"/connections/"+conn.ID, nil).MustStatus(t, 200).JSON(t, &seen)
	if seen.CanManage || seen.Webhook != nil || len(seen.Helpers) == 0 {
		t.Fatalf("viewer's page: canManage=%v webhook=%v helpers=%v", seen.CanManage, seen.Webhook, seen.Helpers)
	}
	webinar := formID(t, conn, "Spring webinar signup")
	for _, w := range []struct{ method, path string }{
		{http.MethodPost, f.path + "/connections"},
		{http.MethodGet, f.path + "/connections/" + conn.ID + "/pages"},
		{http.MethodPut, f.path + "/connections/" + conn.ID + "/pages"},
		{http.MethodPost, f.path + "/connections/" + conn.ID + "/refresh"},
		{http.MethodPut, f.path + "/connections/" + conn.ID + "/credentials"},
		{http.MethodPost, f.path + "/connections/" + conn.ID + "/disconnect"},
		{http.MethodPut, f.path + "/forms/" + webinar + "/mapping"},
		{http.MethodPost, f.path + "/forms/" + webinar + "/pause"},
		{http.MethodPost, f.path + "/forms/" + webinar + "/import"},
	} {
		v.Do(w.method, w.path, map[string]any{}).MustStatus(t, 403)
	}

	// The owner's answers never carry the credentials.
	for _, path := range []string{f.path + "/connections/" + conn.ID, f.path + "/connections", f.path + "/setup", f.path + "/forms/" + webinar} {
		body := f.owner.Do(http.MethodGet, path, nil).MustStatus(t, 200).Body
		for _, secret := range []string{demo.LeadAdsAppSecret, demo.LeadAdsToken, "demo-page-token"} {
			if bytes.Contains(body, []byte(secret)) {
				t.Fatalf("%s returned a secret", path)
			}
		}
	}

	// Another workspace can't see the connection.
	other := f.env.CreateUser(apptest.UserOpts{Email: "other@m66leads.test", Name: "Oz Other"})
	ows := f.env.CreateWorkspace("M66 Other", other.ID)
	o := f.env.Client()
	o.SignIn(other.Email, "")
	o.Do(http.MethodGet, "/workspaces/"+ows.Slug+"/lead-ads/connections/"+conn.ID, nil).MustStatus(t, 404)
}
