package whatsapp_test

// WhatsApp over the API on a workspace of its own (never the demo's),
// against the in-process Graph API: connecting (credentials sealed and
// never returned), Meta's verification request, the template sync and its
// statuses, webhook signatures (X-Hub-Signature-256), WhatsApp consent with
// its evidence, a campaign through review, the dispatcher and Meta's
// statuses, the 24-hour customer service window and a reply, the keywords,
// the messaging limit, and an expired token holding the sends until
// Reconnect. Nothing reaches Meta.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/whatsapp"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta"
	"github.com/hostrepute/reputecast/internal/whatsapp/meta/metatest"
)

func one[T any](t *testing.T, env *apptest.Env, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

type fixture struct {
	env   *apptest.Env
	owner *apptest.Client
	ws    store.Workspace
	path  string
	fake  *metatest.Fake
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	env := apptest.New(t)
	u := env.CreateUser(apptest.UserOpts{Email: "owner@x22wa.test", Name: "Pat Owner"})
	ws := env.CreateWorkspace("X22 WhatsApp Test", u.ID)
	c := env.Client()
	c.SignIn(u.Email, "")
	return &fixture{env: env, owner: c, ws: ws, path: "/workspaces/" + ws.Slug, fake: env.App.WhatsAppFake()}
}

var creds = map[string]any{"businessAccountId": demo.WhatsAppBusinessID, "phoneNumberId": demo.WhatsAppPhoneID, "accessToken": demo.WhatsAppToken,
	"appSecret": demo.WhatsAppAppSecret}

// connect connects the account and answers Meta's verification request.
func (f *fixture) connect(t *testing.T) whatsapp.WhatsAppOverview {
	t.Helper()
	var ov whatsapp.WhatsAppOverview
	f.owner.Do(http.MethodPost, f.path+"/whatsapp/account", creds).MustStatus(t, 201).JSON(t, &ov)
	if ov.Account == nil || ov.Account.VerifyToken == "" {
		t.Fatalf("connect: %+v", ov.Account)
	}
	q := url.Values{"hub.mode": {"subscribe"}, "hub.verify_token": {ov.Account.VerifyToken}, "hub.challenge": {"1158201444"}}
	if code, _ := f.get(t, "/hooks/whatsapp/"+f.ws.Slug+"?"+q.Encode()); code != http.StatusOK {
		t.Fatalf("verification: %d", code)
	}
	f.owner.Do(http.MethodGet, f.path+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &ov)
	return ov
}

func (f *fixture) get(t *testing.T, path string) (int, string) {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, f.env.Server.URL+path, nil)
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Body.Close() }()
	b, _ := io.ReadAll(r.Body)
	return r.StatusCode, string(b)
}

// hook posts a webhook as Meta does and normalizes it as the job would.
func (f *fixture) hook(t *testing.T, body []byte, sig string) int {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, f.env.Server.URL+"/hooks/whatsapp/"+f.ws.Slug, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if sig != "" {
		req.Header.Set("X-Hub-Signature-256", sig)
	}
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Body.Close() }()
	if r.StatusCode == http.StatusOK {
		id := one[uuid.UUID](t, f.env, `SELECT id FROM whatsapp_raw_events WHERE workspace_id = $1 ORDER BY received_at DESC, id DESC LIMIT 1`, f.ws.ID)
		if err := f.env.App.WhatsApp().Normalize(context.Background(), id); err != nil {
			t.Fatal(err)
		}
	}
	return r.StatusCode
}

func (f *fixture) run(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	s := f.env.App.WhatsApp()
	if err := s.StartDue(ctx); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Dispatch(ctx, 10*time.Second); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) actions(t *testing.T) map[string]int {
	t.Helper()
	rows, err := f.env.App.Pool.Query(context.Background(), `SELECT action FROM audit_events WHERE workspace_id = $1`, f.ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			t.Fatal(err)
		}
		out[a]++
	}
	return out
}

func TestWhatsAppConnectTemplatesAndVerification(t *testing.T) {
	f := newFixture(t)
	env, c, p := f.env, f.owner, f.path

	var ov whatsapp.WhatsAppOverview
	c.Do(http.MethodGet, p+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &ov)
	if ov.Account != nil || !strings.HasSuffix(ov.WebhookURL, "/hooks/whatsapp/"+f.ws.Slug) || !ov.CanManage {
		t.Fatalf("before: %+v", ov)
	}
	// Meta refuses a wrong token; ids are numbers.
	bad := map[string]any{"businessAccountId": demo.WhatsAppBusinessID, "phoneNumberId": demo.WhatsAppPhoneID, "accessToken": "wrong", "appSecret": "x"}
	if pr := c.Do(http.MethodPost, p+"/whatsapp/account", bad).MustStatus(t, 422).Problem(t); !strings.Contains(pr["detail"].(string)+toString(pr["errors"]), "rejected the access token") {
		t.Fatalf("wrong token: %v", pr)
	}
	badID := map[string]any{"businessAccountId": "acme", "phoneNumberId": demo.WhatsAppPhoneID, "accessToken": demo.WhatsAppToken, "appSecret": "x"}
	c.Do(http.MethodPost, p+"/whatsapp/account", badID).MustStatus(t, 422)

	r := c.Do(http.MethodPost, p+"/whatsapp/account", creds).MustStatus(t, 201)
	r.JSON(t, &ov)
	a := ov.Account
	if a == nil || a.Status != "waiting" || a.Number != "+1 312 555 0110" || a.DisplayName != "Acme" || a.NameStatus != "approved" || a.Quality != "high" ||
		a.MessagingLimit != 1000 || a.BusinessName != "Acme Inc" || a.BusinessHint != "…4410" || !strings.HasPrefix(a.VerifyToken, "rc-wa-") {
		t.Fatalf("account: %+v", a)
	}
	// Credentials are write-only.
	for _, b := range [][]byte{r.Body, c.Do(http.MethodGet, p+"/whatsapp", nil).MustStatus(t, 200).Body} {
		for _, secret := range []string{demo.WhatsAppToken, demo.WhatsAppAppSecret, demo.WhatsAppBusinessID, demo.WhatsAppPhoneID} {
			if bytes.Contains(b, []byte(secret)) {
				t.Fatalf("a credential (%s) is in an answer", secret)
			}
		}
	}
	if !f.fake.Subscribed {
		t.Fatal("the app didn't subscribe to the account's webhooks")
	}
	// The templates came with the connection.
	if len(ov.Templates) != 4 || ov.TemplateCounts.Approved != 2 || ov.TemplateCounts.Pending != 1 || ov.TemplateCounts.Rejected != 1 {
		t.Fatalf("templates: %+v %+v", ov.TemplateCounts, ov.Templates)
	}
	var rejected, day whatsapp.WhatsAppTemplate
	for _, tp := range ov.Templates {
		switch tp.Name {
		case "digest_highlights":
			rejected = tp
		case "summit_day_before":
			day = tp
		}
	}
	if rejected.Status != "rejected" || rejected.Category != "marketing" || rejected.Reason == "" || len(rejected.Buttons) != 1 || rejected.Buttons[0].Type != "quick_reply" {
		t.Fatalf("rejected: %+v", rejected)
	}
	if day.Status != "approved" || day.LanguageLabel != "English (US)" || len(day.Variables) != 2 || day.Variables[0].Example != "Sam" || day.Buttons[0].Text != "View the agenda" {
		t.Fatalf("summit_day_before: %+v", day)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM notifications WHERE workspace_id = $1 AND kind = 'whatsapp_attention'`, f.ws.ID); n != 1 {
		t.Fatalf("rejected-template notifications: %d", n)
	}
	// Meta's verification request: the wrong token gets nothing.
	q := url.Values{"hub.mode": {"subscribe"}, "hub.verify_token": {"rc-wa-wrong00000"}, "hub.challenge": {"42"}}
	if code, _ := f.get(t, "/hooks/whatsapp/"+f.ws.Slug+"?"+q.Encode()); code != http.StatusForbidden {
		t.Fatalf("wrong verify token: %d", code)
	}
	q.Set("hub.verify_token", a.VerifyToken)
	if code, body := f.get(t, "/hooks/whatsapp/"+f.ws.Slug+"?"+q.Encode()); code != http.StatusOK || body != "42" {
		t.Fatalf("verification: %d %q", code, body)
	}
	c.Do(http.MethodGet, p+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &ov)
	if ov.Account.Status != "connected" || ov.Account.VerifiedAt == nil {
		t.Fatalf("after verification: %+v", ov.Account)
	}
	c.Do(http.MethodPost, p+"/whatsapp/account", creds).MustStatus(t, 409)

	// A template approved in WhatsApp Manager comes in at the next sync.
	ts := demo.WhatsAppTemplates()
	ts[2].Status = "APPROVED"
	f.fake.SetTemplates(ts)
	var sync whatsapp.WhatsAppSyncResult
	c.Do(http.MethodPost, p+"/whatsapp/templates/sync", nil).MustStatus(t, 200).JSON(t, &sync)
	if sync.Templates != 4 || len(sync.Changed) != 1 || !strings.HasPrefix(sync.Changed[0], "summit_venue_change") {
		t.Fatalf("sync: %+v", sync)
	}

	// A viewer sees the account but no verify token, and changes nothing.
	viewer := env.CreateUser(apptest.UserOpts{Email: "viewer@x22wa.test", Name: "Vic Viewer"})
	env.AddMember(f.ws, viewer.ID, "viewer")
	vc := env.Client()
	vc.SignInDemo(viewer.Email)
	ov = whatsapp.WhatsAppOverview{}
	vc.Do(http.MethodGet, p+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &ov)
	if ov.Account == nil || ov.Account.VerifyToken != "" || ov.CanManage || len(ov.Helpers) == 0 {
		t.Fatalf("viewer: %+v", ov)
	}
	vc.Do(http.MethodPost, p+"/whatsapp/templates/sync", nil).MustStatus(t, 403)
	vc.Do(http.MethodDelete, p+"/whatsapp/account", nil).MustStatus(t, 403)

	acts := f.actions(t)
	for _, want := range []string{"whatsapp.connected", "whatsapp_templates.synced"} {
		if acts[want] == 0 {
			t.Fatalf("audit: no %s in %v", want, acts)
		}
	}
	c.Do(http.MethodDelete, p+"/whatsapp/account", nil).MustStatus(t, 204)
	if acts := f.actions(t); acts["whatsapp.disconnected"] != 1 {
		t.Fatalf("audit after disconnect: %v", acts)
	}
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func TestWhatsAppWebhooks(t *testing.T) {
	f := newFixture(t)
	env := f.env
	f.connect(t)
	ctx := context.Background()

	// A body signed with another secret is refused and kept for diagnosis.
	body, _ := metatest.TemplateWebhook("not-the-secret", demo.WhatsAppBusinessID, "1290033512244412", "summit_venue_change", "en_US", "APPROVED", "NONE", "")
	if st := f.hook(t, body, meta.Sign("not-the-secret", body)); st != 401 {
		t.Fatalf("bad signature: %d", st)
	}
	if st := f.hook(t, body, ""); st != 401 {
		t.Fatalf("no signature: %d", st)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM whatsapp_raw_events WHERE workspace_id = $1 AND NOT signature_ok`, f.ws.ID); n != 2 {
		t.Fatalf("refused events kept: %d", n)
	}
	if st := f.hook(t, body, "sha256=00"); st != 401 {
		t.Fatalf("malformed signature: %d", st)
	}
	// Signed with the app secret: stored, 200, normalized.
	body, sig := metatest.TemplateWebhook(demo.WhatsAppAppSecret, demo.WhatsAppBusinessID, "1290033512244412", "summit_venue_change", "en_US", "APPROVED", "NONE", "")
	if st := f.hook(t, body, sig); st != 200 {
		t.Fatalf("signed: %d", st)
	}
	if s := one[string](t, env, `SELECT status FROM whatsapp_templates WHERE workspace_id = $1 AND name = 'summit_venue_change'`, f.ws.ID); s != "approved" {
		t.Fatalf("template status: %s", s)
	}
	// Meta's words for a rejection.
	body, sig = metatest.TemplateWebhook(demo.WhatsAppAppSecret, demo.WhatsAppBusinessID, "1290033512244411", "summit_livestream", "en_US", "REJECTED",
		"INVALID_FORMAT", "Add sample values for every variable.")
	f.hook(t, body, sig)
	if r := one[string](t, env, `SELECT reason FROM whatsapp_templates WHERE workspace_id = $1 AND name = 'summit_livestream'`, f.ws.ID); r != "Add sample values for every variable." {
		t.Fatalf("reason: %q", r)
	}
	var evs whatsapp.WhatsAppEventList
	f.owner.Do(http.MethodGet, f.path+"/whatsapp/events", nil).MustStatus(t, 200).JSON(t, &evs)
	if len(evs.Items) != 5 || evs.Items[0].Kind != "template" || !evs.Items[0].SignatureOK || evs.Items[0].NormalizedAt == nil || evs.Items[4].SignatureOK {
		t.Fatalf("events: %+v", evs.Items)
	}

	// The quality rating drops: the card follows and the managers hear.
	f.fake.SetPhone(meta.PhoneNumber{DisplayPhoneNumber: "+1 312-555-0110", VerifiedName: "Acme", NameStatus: "APPROVED", QualityRating: "YELLOW",
		MessagingLimitTier: "TIER_250", Status: "CONNECTED"})
	body, sig = metatest.QualityWebhook(demo.WhatsAppAppSecret, demo.WhatsAppBusinessID, "DOWNGRADE", "TIER_250", "TIER_1K")
	f.hook(t, body, sig)
	var ov whatsapp.WhatsAppOverview
	f.owner.Do(http.MethodGet, f.path+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &ov)
	if ov.Account.Quality != "medium" || ov.Account.PreviousQuality != "high" || ov.Account.MessagingLimit != 250 || ov.Account.PreviousLimit != 1000 ||
		ov.Account.QualityChangedAt == nil {
		t.Fatalf("quality: %+v", ov.Account)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM notifications WHERE workspace_id = $1 AND kind = 'whatsapp_attention' AND title LIKE '%quality%'`, f.ws.ID); n != 1 {
		t.Fatalf("quality notifications: %d", n)
	}
	// An unknown address.
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, env.Server.URL+"/hooks/whatsapp/no-such-workspace", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", sig)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown workspace: %d", resp.StatusCode)
	}
}

// people adds a list and contacts with phones: Sam and Ana in Chicago,
// Lee without a first name, Kim without WhatsApp consent.
func (f *fixture) people(t *testing.T) (list uuid.UUID, ids map[string]uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	list = uuid.Must(uuid.NewV7())
	if _, err := f.env.App.Pool.Exec(ctx, `INSERT INTO lists (id, workspace_id, name) VALUES ($1, $2, 'X22 Summit')`, list, f.ws.ID); err != nil {
		t.Fatal(err)
	}
	ids = map[string]uuid.UUID{}
	for _, p := range []struct{ key, first, email, phone string }{
		{"sam", "Sam", "sam@x22wa.test", "+13125550123"},
		{"ana", "Ana", "ana@x22wa.test", "+13125550124"},
		{"lee", "", "lee@x22wa.test", "+13125550125"},
		{"kim", "Kim", "kim@x22wa.test", "+13125550126"},
	} {
		id := uuid.Must(uuid.NewV7())
		if _, err := f.env.App.Pool.Exec(ctx, `INSERT INTO contacts (id, workspace_id, email, first_name, timezone, phone, phone_country) VALUES ($1, $2, $3, $4, 'America/Chicago', $5, 'US')`,
			id, f.ws.ID, p.email, p.first, p.phone); err != nil {
			t.Fatal(err)
		}
		if _, err := f.env.App.Pool.Exec(ctx, `INSERT INTO list_memberships (list_id, contact_id, status, subscribed_at) VALUES ($1, $2, 'subscribed', now())`, list, id); err != nil {
			t.Fatal(err)
		}
		ids[p.key] = id
	}
	return list, ids
}

func evidence() map[string]any {
	return map[string]any{"consentText": "Send me reminders about X22 Summit on WhatsApp.", "consentVersion": "1", "source": "form:x22-summit",
		"page": "https://x22wa.test/summit", "ip": "198.51.100.7", "at": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)}
}

func TestWhatsAppConsentCampaignWindowAndKeywords(t *testing.T) {
	f := newFixture(t)
	env, c, p := f.env, f.owner, f.path
	ctx := context.Background()
	f.connect(t)
	list, ids := f.people(t)
	c.Do(http.MethodPut, p+"/whatsapp/settings", map[string]any{"quietHours": false}).MustStatus(t, 200)

	// Consent needs its evidence.
	c.Do(http.MethodPut, p+"/contacts/"+ids["sam"].String()+"/whatsapp/consent", map[string]any{"listId": list, "status": "subscribed"}).MustStatus(t, 422)
	for _, k := range []string{"sam", "ana", "lee"} {
		c.Do(http.MethodPut, p+"/contacts/"+ids[k].String()+"/whatsapp/consent", map[string]any{"listId": list, "status": "subscribed", "evidence": evidence()}).
			MustStatus(t, 200)
	}
	var card whatsapp.WhatsAppContactCard
	c.Do(http.MethodGet, p+"/contacts/"+ids["sam"].String()+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &card)
	if !card.Connected || len(card.Lists) != 1 || card.Lists[0].Status != "subscribed" || card.Lists[0].Version != "1" || card.Window.Open || card.CanReply {
		t.Fatalf("card: %+v", card)
	}

	// A campaign: summit_day_before, {{1}} the first name or "there",
	// {{2}} a fixed "8:30".
	var tpl string
	var ov whatsapp.WhatsAppOverview
	c.Do(http.MethodGet, p+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &ov)
	for _, tp := range ov.Templates {
		if tp.Name == "summit_day_before" {
			tpl = tp.ID
		}
		if tp.Name == "summit_venue_change" {
			// Only approved templates can be picked.
			c.Do(http.MethodPost, p+"/whatsapp-campaigns", map[string]any{"name": "Nope", "templateId": tp.ID}).MustStatus(t, 422)
		}
	}
	var camp whatsapp.WhatsAppCampaign
	c.Do(http.MethodPost, p+"/whatsapp-campaigns", map[string]any{"name": "X22 see you tomorrow", "templateId": tpl, "lists": []string{list.String()}}).
		MustStatus(t, 201).JSON(t, &camp)
	upd := map[string]any{"name": camp.Name, "templateId": tpl, "lists": []string{list.String()}, "mode": "now", "version": camp.Version,
		"variables": []map[string]any{{"key": "1", "source": "field", "field": "first_name", "fallback": "there"}, {"key": "2", "source": "text", "text": "8:30"}}}
	c.Do(http.MethodPatch, p+"/whatsapp-campaigns/"+camp.ID, upd).MustStatus(t, 200).JSON(t, &camp)
	if camp.Audience.Final != 3 || camp.Audience.Without != 1 || camp.Preview == nil || camp.Cost.Messages != 3 || camp.Cost.Category != "utility" {
		t.Fatalf("composer: audience %+v preview %+v cost %+v", camp.Audience, camp.Preview, camp.Cost)
	}
	if !strings.HasPrefix(camp.Preview.Text, "Hi ") || !strings.Contains(camp.Preview.Text, "Doors open at 8:30 at Lakeside Hall") {
		t.Fatalf("preview: %q", camp.Preview.Text)
	}
	upd["variables"] = []map[string]any{{"key": "1", "source": "field", "field": "no_such_field"}}
	upd["version"] = camp.Version
	c.Do(http.MethodPatch, p+"/whatsapp-campaigns/"+camp.ID, upd).MustStatus(t, 422)
	var rv whatsapp.WhatsAppReview
	c.Do(http.MethodGet, p+"/whatsapp-campaigns/"+camp.ID+"/review", nil).MustStatus(t, 200).JSON(t, &rv)
	if rv.Verdict != "ready" || len(rv.Checks) != 7 {
		t.Fatalf("review: %s %+v", rv.Verdict, rv.Checks)
	}
	// Lee's number isn't on WhatsApp.
	f.fake.Fail("+13125550125", 131026)
	var res whatsapp.WhatsAppSendResult
	c.Do(http.MethodPost, p+"/whatsapp-campaigns/"+camp.ID+"/send", map[string]any{"version": rv.Version}).MustStatus(t, 200).JSON(t, &res)
	if res.Status != "sending" || res.Recipients != 3 {
		t.Fatalf("send: %+v", res)
	}
	f.run(t)
	sent := f.fake.SentMessages()
	if len(sent) != 2 {
		t.Fatalf("sent %d", len(sent))
	}
	for _, m := range sent {
		if m.Template != "summit_day_before" || m.Language != "en_US" || len(m.Params) != 2 || m.Params[1] != "8:30" {
			t.Fatalf("message: %+v", m)
		}
	}
	if n := one[int64](t, env, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1 AND status = 'failed' AND reason = 'undeliverable' AND provider_code = '131026'`,
		uuid.MustParse(camp.ID)); n != 1 {
		t.Fatalf("undeliverable: %d", n)
	}
	// Meta's statuses: delivered (billable) and read.
	samMsg := ""
	for _, m := range sent {
		if m.To == "+13125550123" {
			samMsg = m.ID
		}
	}
	for _, st := range []string{"delivered", "read"} {
		body, sig := metatest.StatusWebhook(demo.WhatsAppAppSecret, demo.WhatsAppBusinessID, demo.WhatsAppPhoneID, samMsg, "+13125550123", st, time.Now(), 0)
		if code := f.hook(t, body, sig); code != 200 {
			t.Fatalf("status %s: %d", st, code)
		}
	}
	var rep whatsapp.WhatsAppReport
	c.Do(http.MethodGet, p+"/whatsapp-campaigns/"+camp.ID+"/report", nil).MustStatus(t, 200).JSON(t, &rep)
	if rep.Status != "sent" || rep.Sent != 2 || rep.Delivered != 1 || rep.Read != 1 || rep.Failed != 1 || rep.Failures[0].Reason != "undeliverable" || rep.Cost != 0.0 &&
		rep.Cost != 0.01 {
		t.Fatalf("report: %+v", rep)
	}
	if cost := one[float64](t, env, `SELECT cost FROM channel_messages WHERE provider_message_id = $1`, samMsg); cost != 0.004 {
		t.Fatalf("Sam's message cost %v", cost)
	}

	// Sam writes back: the window opens for 24 hours and a reply goes.
	body, sig := metatest.MessageWebhook(demo.WhatsAppAppSecret, demo.WhatsAppBusinessID, demo.WhatsAppPhoneID, "+13125550123", "Sam Rivera",
		"Is there parking at the venue?", time.Now())
	f.hook(t, body, sig)
	c.Do(http.MethodGet, p+"/contacts/"+ids["sam"].String()+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &card)
	if !card.Window.Open || card.Window.Until == nil || card.Window.LastText != "Is there parking at the venue?" || !card.CanReply || len(card.Thread) != 2 ||
		card.Thread[0].Direction == card.Thread[1].Direction {
		t.Fatalf("window: %+v", card)
	}
	var item whatsapp.WhatsAppThreadItem
	c.Do(http.MethodPost, p+"/contacts/"+ids["sam"].String()+"/whatsapp/reply", map[string]any{"text": "Yes: the Lakeside Hall garage on Harbor Drive."}).
		MustStatus(t, 200).JSON(t, &item)
	f.run(t)
	last := f.fake.SentMessages()[len(f.fake.SentMessages())-1]
	if last.Text != "Yes: the Lakeside Hall garage on Harbor Drive." || last.To != "+13125550123" {
		t.Fatalf("reply: %+v", last)
	}
	// Ana never wrote: her window is closed.
	c.Do(http.MethodPost, p+"/contacts/"+ids["ana"].String()+"/whatsapp/reply", map[string]any{"text": "Hello"}).MustStatus(t, 409)

	// Ana writes STOP: WhatsApp consent ends on every list and her number
	// is suppressed; START brings it back.
	anaMsg := ""
	for _, m := range sent {
		if m.To == "+13125550124" {
			anaMsg = m.ID
		}
	}
	body, sig = metatest.MessageWebhook(demo.WhatsAppAppSecret, demo.WhatsAppBusinessID, demo.WhatsAppPhoneID, "+13125550124", "Ana", "stop", time.Now())
	f.hook(t, body, sig)
	if st := one[string](t, env, `SELECT status FROM channel_consents WHERE contact_id = $1 AND channel = 'whatsapp'`, ids["ana"]); st != "opted_out" {
		t.Fatalf("after STOP: %s", st)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM channel_suppressions WHERE workspace_id = $1 AND channel = 'whatsapp' AND phone = '+13125550124' AND removed_at IS NULL`,
		f.ws.ID); n != 1 {
		t.Fatalf("suppressions: %d", n)
	}
	// A person can't override it.
	c.Do(http.MethodPut, p+"/contacts/"+ids["ana"].String()+"/whatsapp/consent", map[string]any{"listId": list, "status": "subscribed", "evidence": evidence()}).
		MustStatus(t, 409)
	body, sig = metatest.MessageWebhook(demo.WhatsAppAppSecret, demo.WhatsAppBusinessID, demo.WhatsAppPhoneID, "+13125550124", "Ana", "START", time.Now())
	f.hook(t, body, sig)
	if st := one[string](t, env, `SELECT status FROM channel_consents WHERE contact_id = $1 AND channel = 'whatsapp'`, ids["ana"]); st != "subscribed" {
		t.Fatalf("after START: %s", st)
	}
	// The "Stop promotions" quick reply works as STOP.
	body, sig = metatest.ButtonWebhook(demo.WhatsAppAppSecret, demo.WhatsAppBusinessID, demo.WhatsAppPhoneID, "+13125550124", anaMsg, "Stop promotions", time.Now())
	f.hook(t, body, sig)
	if st := one[string](t, env, `SELECT status FROM channel_consents WHERE contact_id = $1 AND channel = 'whatsapp'`, ids["ana"]); st != "opted_out" {
		t.Fatalf("after the quick reply: %s", st)
	}
	c.Do(http.MethodGet, p+"/whatsapp-campaigns/"+camp.ID+"/report", nil).MustStatus(t, 200).JSON(t, &rep)
	if rep.Replies != 1 || rep.OptOuts != 2 {
		t.Fatalf("report replies %d, opt-outs %d", rep.Replies, rep.OptOuts)
	}
	acts := f.actions(t)
	for _, want := range []string{"contact.whatsapp_consent_changed", "campaign.create", "campaign.scheduled", "campaign.send_started", "campaign.completed",
		"whatsapp_settings.updated"} {
		if acts[want] == 0 {
			t.Fatalf("audit: no %s in %v", want, acts)
		}
	}
	_ = ctx
}

func TestWhatsAppLimitAndExpiredToken(t *testing.T) {
	f := newFixture(t)
	env, c, p := f.env, f.owner, f.path
	f.connect(t)
	list, ids := f.people(t)
	c.Do(http.MethodPut, p+"/whatsapp/settings", map[string]any{"quietHours": false}).MustStatus(t, 200)
	for _, k := range []string{"sam", "ana", "lee"} {
		c.Do(http.MethodPut, p+"/contacts/"+ids[k].String()+"/whatsapp/consent", map[string]any{"listId": list, "status": "subscribed", "evidence": evidence()}).
			MustStatus(t, 200)
	}
	var ov whatsapp.WhatsAppOverview
	c.Do(http.MethodGet, p+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &ov)
	tpl := ""
	for _, tp := range ov.Templates {
		if tp.Name == "summit_livestream" {
			tpl = tp.ID
		}
	}
	newCampaign := func(name string) (whatsapp.WhatsAppCampaign, whatsapp.WhatsAppReview) {
		var camp whatsapp.WhatsAppCampaign
		c.Do(http.MethodPost, p+"/whatsapp-campaigns", map[string]any{"name": name, "templateId": tpl, "lists": []string{list.String()}}).MustStatus(t, 201).JSON(t, &camp)
		c.Do(http.MethodPatch, p+"/whatsapp-campaigns/"+camp.ID, map[string]any{"name": name, "templateId": tpl, "lists": []string{list.String()}, "mode": "now",
			"version": camp.Version, "variables": []map[string]any{{"key": "1", "source": "text", "text": "go.acme.io/k3P9xq"}}}).MustStatus(t, 200).JSON(t, &camp)
		var rv whatsapp.WhatsAppReview
		c.Do(http.MethodGet, p+"/whatsapp-campaigns/"+camp.ID+"/review", nil).MustStatus(t, 200).JSON(t, &rv)
		return camp, rv
	}
	first, rv := newCampaign("X22 livestream")
	c.Do(http.MethodPost, p+"/whatsapp-campaigns/"+first.ID+"/send", map[string]any{"version": rv.Version}).MustStatus(t, 200)
	f.run(t)
	if n := len(f.fake.SentMessages()); n != 3 {
		t.Fatalf("first send: %d", n)
	}

	// Meta lowered the limit to 2 people a day: three people got a
	// message in the last 24 hours, so a second send waits for tomorrow.
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE whatsapp_accounts SET messaging_limit = 2 WHERE workspace_id = $1`, f.ws.ID); err != nil {
		t.Fatal(err)
	}
	second, rv := newCampaign("X22 livestream again")
	var lim *whatsapp.WhatsAppCheck
	for i := range rv.Checks {
		if rv.Checks[i].Key == "limit" {
			lim = &rv.Checks[i]
		}
	}
	if rv.Verdict != "warnings" || lim == nil || lim.Status != "warn" || lim.Split.Used != 3 || lim.Split.Left != 0 || lim.Split.Now != 0 || lim.Split.Later != 3 ||
		lim.Split.LaterAt == nil || lim.Split.UsedBy != "X22 livestream" {
		t.Fatalf("limit check: %s %+v", rv.Verdict, lim)
	}
	var res whatsapp.WhatsAppSendResult
	c.Do(http.MethodPost, p+"/whatsapp-campaigns/"+second.ID+"/send", map[string]any{"version": rv.Version, "overLimit": "skip"}).MustStatus(t, 200).JSON(t, &res)
	f.run(t)
	if n := one[int64](t, env, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1 AND status = 'skipped' AND reason = 'messaging_limit'`,
		uuid.MustParse(second.ID)); n != 3 {
		t.Fatalf("skipped over the limit: %d", n)
	}

	// The token expires: sends are held, the managers hear, Reconnect
	// with a new token sends them.
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE whatsapp_accounts SET messaging_limit = 1000 WHERE workspace_id = $1`, f.ws.ID); err != nil {
		t.Fatal(err)
	}
	f.fake.ExpireToken()
	third, rv := newCampaign("X22 doors open")
	c.Do(http.MethodPost, p+"/whatsapp-campaigns/"+third.ID+"/send", map[string]any{"version": rv.Version}).MustStatus(t, 200)
	before := len(f.fake.SentMessages())
	f.run(t)
	if n := one[int64](t, env, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1 AND status = 'held' AND reason = 'token_rejected'`,
		uuid.MustParse(third.ID)); n != 3 {
		t.Fatalf("held: %d", n)
	}
	c.Do(http.MethodGet, p+"/whatsapp", nil).MustStatus(t, 200).JSON(t, &ov)
	if ov.Account.Status != "token_rejected" || ov.Account.StatusDetail != "expired" || ov.Account.HeldCount != 3 || len(ov.Account.Held) != 1 {
		t.Fatalf("expired: %+v", ov.Account)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM notifications WHERE workspace_id = $1 AND kind = 'whatsapp_token'`, f.ws.ID); n != 1 {
		t.Fatalf("token notifications: %d", n)
	}
	// A new campaign can't go while the token is refused.
	fourth, rv := newCampaign("X22 blocked")
	if rv.Verdict != "blocked" {
		t.Fatalf("review with a refused token: %s", rv.Verdict)
	}
	c.Do(http.MethodPost, p+"/whatsapp-campaigns/"+fourth.ID+"/send", map[string]any{"version": rv.Version}).MustStatus(t, 409)
	// A token Meta refuses isn't saved.
	c.Do(http.MethodPut, p+"/whatsapp/account/credentials", map[string]any{"accessToken": "still-wrong"}).MustStatus(t, 422)
	f.fake.Renew("renewed-token")
	c.Do(http.MethodPut, p+"/whatsapp/account/credentials", map[string]any{"accessToken": "renewed-token"}).MustStatus(t, 200).JSON(t, &ov)
	if ov.Account.Status != "connected" {
		t.Fatalf("after reconnect: %+v", ov.Account)
	}
	f.run(t)
	if n := len(f.fake.SentMessages()) - before; n != 3 {
		t.Fatalf("released and sent: %d", n)
	}
	if acts := f.actions(t); acts["whatsapp.credentials_replaced"] != 1 {
		t.Fatalf("audit: %v", acts)
	}
}
