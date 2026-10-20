package texting_test

// SMS over the API against the demo (brief §4): the providers page, the
// sent campaign's report, Daniel's draft in the composer and Review & send,
// Routing & rules with the route tester, the suppression list; then the
// live paths through the in-process fakes: provider webhooks (verified,
// stored raw, normalized by the job), a STOP and START by text, a campaign
// sent by the dispatcher inside quiet hours, the budget and permissions.

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/demo"
	"github.com/hostrepute/reputecast/internal/jobs"
	"github.com/hostrepute/reputecast/internal/sms"
	"github.com/hostrepute/reputecast/internal/sms/twilio/twiliotest"
	"github.com/hostrepute/reputecast/internal/sms/vonage/vonagetest"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/texting"
)

const ws = "/workspaces/acme-newsletter"

func one[T any](t *testing.T, env *apptest.Env, sql string, args ...any) T {
	t.Helper()
	var v T
	if err := env.App.Pool.QueryRow(context.Background(), sql, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return v
}

// service is the SMS domain on the test app's database, its demo
// providers answered by fakes, its clock at now.
func service(env *apptest.Env, now func() time.Time) (*texting.Service, *twiliotest.Fake, *vonagetest.Fake) {
	tw, vo := twiliotest.New(), vonagetest.New()
	s := &texting.Service{Pool: env.App.Pool, Q: env.App.Q, Audit: env.App.Audit, Box: env.App.Secrets, URLs: env.App.URLs,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Jobs: func() *jobs.Client { return nil }, Now: now,
		Demo: func(adapter string) (*http.Client, string) {
			if adapter == "vonage" {
				return vo.Client(), "https://vonage.test"
			}
			return tw.Client(), "https://twilio.test"
		}}
	return s, tw, vo
}

func workspace(t *testing.T, env *apptest.Env) *store.Workspace {
	t.Helper()
	w, err := env.App.Q.GetWorkspaceBySlug(context.Background(), "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	return &w
}

func TestSMSDemo(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")

	// Sending providers › SMS › Providers.
	var ov texting.SmsOverview
	linh.Do(http.MethodGet, ws+"/sms/overview", nil).MustStatus(t, 200).JSON(t, &ov)
	tl := ov.Tiles
	if tl.Sent != 1731 || tl.ByKind["confirmation"] != 871 || tl.ByKind["campaign"] != 854 || tl.ByKind["test"] != 6 || tl.Delivered != 1710 || tl.Failed != 21 {
		t.Fatalf("texts: %+v", tl)
	}
	if tl.Spend != 39.51 || tl.Budget != 100 || tl.AlertPct != 80 || tl.Suppressed != 25 || tl.SuppressedBy["opted_out"] != 16 || tl.SuppressedBy["landline"] != 5 {
		t.Fatalf("spend and suppressions: %+v", tl)
	}
	if len(ov.Providers) != 2 || ov.Providers[0].Name != "Twilio" || ov.Providers[0].Spend != 14.71 || ov.Providers[1].Spend != 24.80 || ov.Providers[0].AccountHint != "AC…3f9" {
		t.Fatalf("providers: %+v", ov.Providers)
	}
	if len(ov.NotConnected) != 3 || len(ov.Senders) != 4 || len(ov.Routes) != 3 || !ov.Routes[2].Skipped || ov.Routes[2].Countries[0] != "IE" {
		t.Fatalf("catalog %d, senders %d, routes %+v", len(ov.NotConnected), len(ov.Senders), ov.Routes)
	}
	for _, p := range ov.Providers {
		if len(p.SecretLabels) == 0 {
			t.Fatalf("%s names no secrets", p.Name)
		}
	}
	if tl.BudgetHeld != 0 {
		t.Fatalf("held for the budget: %d", tl.BudgetHeld)
	}
	// Import from providers: the three numbers in Twilio's account, two of
	// them senders already.
	var nums texting.SmsFoundSenderList
	linh.Do(http.MethodGet, ws+"/sms/providers/"+ov.Providers[0].ID+"/numbers", nil).MustStatus(t, 200).JSON(t, &nums)
	imported := 0
	for _, n := range nums.Items {
		if n.Imported {
			imported++
		}
	}
	if len(nums.Items) != 3 || imported != 2 {
		t.Fatalf("account numbers: %+v", nums.Items)
	}

	// The public text message terms page (PublicSmsTerms), in the demo's
	// published version; a workspace that doesn't text has none.
	page := string(linh.Do(http.MethodGet, "/p/acme-newsletter/sms-terms", nil).MustStatus(t, 200).Body)
	since := one[time.Time](t, env, `SELECT published_at FROM sms_terms_versions WHERE version = 1`).In(time.UTC).Format("2 Jan 2006")
	for _, want := range []string{"Text message terms", "Version 1 · in effect since " + since, "<dt>Who texts you</dt>",
		`<a href="mailto:support@acme.io">support@acme.io</a>`, `<span class="nw">+1 888 555 0142</span>`,
		`<a href="https://news.acme.io/privacy">privacy policy</a>`, `href="https://news.acme.io/summit">Back to registration</a>`,
		"Acme Inc · 500 Market St, Suite 1200 · San Francisco, CA 94105. When these terms change"} {
		if !strings.Contains(page, want) {
			t.Fatalf("terms page lacks %q:\n%s", want, page)
		}
	}
	linh.Do(http.MethodGet, "/p/acme-engineering-blog/sms-terms", nil).MustStatus(t, 404)
	// The channel switch: email and SMS are set up.
	var ch texting.SmsChannels
	linh.Do(http.MethodGet, ws+"/channels", nil).MustStatus(t, 200).JSON(t, &ch)
	if !ch.SwitchShown {
		t.Fatalf("channels: %+v", ch)
	}

	// The sent campaign's report.
	campaign := one[string](t, env, `SELECT id::text FROM campaigns WHERE name = 'Summit: 8 days to go'`)
	var r texting.SmsReport
	linh.Do(http.MethodGet, ws+"/sms-campaigns/"+campaign+"/report", nil).MustStatus(t, 200).JSON(t, &r)
	if r.Tiles.Sent != 854 || r.Tiles.Delivered != 840 || r.Tiles.Failed != 14 || r.Tiles.Clicks != 232 || r.Tiles.Machine != 41 || r.Tiles.OptOuts != 6 ||
		r.Tiles.Cost != 19.54 || r.Tiles.Skipped != 20 {
		t.Fatalf("report tiles: %+v", r.Tiles)
	}
	if r.ReplyCounts["opt_out"] != 6 || r.ReplyCounts["help"] != 2 || r.ReplyCounts["reply"] != 11 || len(r.Replies) != 6 {
		t.Fatalf("replies: %v, %d shown", r.ReplyCounts, len(r.Replies))
	}
	if len(r.Steps) != 7 || r.Steps[0].Total != 77 || r.Steps[0].Delivered != 76 || r.Steps[1].Total != 118 {
		t.Fatalf("time zones: %+v", r.Steps)
	}
	countries := map[string]texting.SmsCountryReport{}
	for _, c := range r.Countries {
		countries[c.Country] = c
	}
	if c := countries["US"]; c.Sent != 611 || c.Delivered != 601 || c.Clicks != 158 || c.OptOuts != 4 || c.Cost != 6.72 {
		t.Fatalf("US: %+v", c)
	}
	if c := countries["IE"]; c.Skipped != 20 || c.Sent != 0 || c.Sender != "ACME" {
		t.Fatalf("Ireland: %+v", c)
	}
	reasons := map[string]int64{}
	for _, f := range r.Failures {
		reasons[f.Reason] = f.Count
	}
	if reasons["unreachable"] != 9 || reasons["filtered"] != 3 || reasons["not_in_service"] != 2 {
		t.Fatalf("failures: %v", reasons)
	}

	// Daniel's draft: a curly apostrophe makes it UCS-2.
	draft := one[string](t, env, `SELECT id::text FROM campaigns WHERE name = 'Summit: doors open'`)
	daniel := env.Client()
	daniel.SignInDemo("daniel@acme.io")
	var c texting.SmsCampaign
	daniel.Do(http.MethodGet, ws+"/sms-campaigns/"+draft, nil).MustStatus(t, 200).JSON(t, &c)
	if c.Counter.Characters != 98 || c.Counter.Segments != 2 || c.Counter.Encoding != sms.UCS2 || len(c.Counter.Outside) != 1 {
		t.Fatalf("counter: %+v", c.Counter.Count)
	}
	// After the campaign: its 6 STOPs have no consent now, its 2 numbers
	// not in service are suppressed.
	if f := c.Funnel; f.Base != 1240 || f.NoConsent != 365 || f.Suppressed != 9 || f.CanReceive != 866 || f.ToSend != 846 || f.Skipped != 20 || f.Assumed != 12 {
		t.Fatalf("funnel: %+v", f)
	}
	if c.Access.CanSend || len(c.Access.Missing) != 2 {
		t.Fatalf("Daniel's access: %+v", c.Access)
	}
	var rv texting.SmsReview
	daniel.Do(http.MethodGet, ws+"/sms-campaigns/"+draft+"/review", nil).MustStatus(t, 200).JSON(t, &rv)
	if rv.Verdict != texting.VerdictWarnings || rv.Warnings != 2 || rv.Blocks != 0 {
		t.Fatalf("review: %s, %d warnings, %d blocks: %+v", rv.Verdict, rv.Warnings, rv.Blocks, rv.Checks)
	}
	tz := rv.Checks[1]
	if tz.Key != "time_zones" || tz.Count != 12 || tz.ByArea != 10 || tz.ByCountry != 2 {
		t.Fatalf("time zones: %+v", tz)
	}
	// Daniel can't send it: he lacks campaigns:send and sms:send.
	daniel.Do(http.MethodPost, ws+"/sms-campaigns/"+draft+"/send", map[string]any{"fingerprint": rv.Fingerprint}).MustStatus(t, 403)

	// Routing & rules, the route tester.
	var rules texting.SmsRules
	linh.Do(http.MethodGet, ws+"/sms/rules", nil).MustStatus(t, 200).JSON(t, &rules)
	if rules.QuietFrom != "09:00" || rules.QuietTo != "20:00" || len(rules.TestNumbers) != 2 || rules.TestsSent != 6 || rules.TestsCost != 0.07 ||
		rules.Assumed != 12 || len(rules.Gaps) != 1 || rules.Gaps[0].Held != 20 || rules.ShortDomain.Status != "verified" {
		t.Fatalf("rules: %+v", rules)
	}
	var rt texting.SmsRouteTest
	linh.Do(http.MethodPost, ws+"/sms/route-test", map[string]any{"phone": "+44 7700 900456"}).MustStatus(t, 200).JSON(t, &rt)
	if rt.Provider != "Vonage" || rt.Sender != "ACME" || rt.Price != 0.045 || rt.Registration.Status != "allowed" || rt.CanReply {
		t.Fatalf("route test: %+v", rt)
	}

	// The suppression list: a text opt-out can't be removed by hand.
	var sl texting.SmsSuppressionList
	linh.Do(http.MethodGet, ws+"/sms/suppressions?reason=opted_out", nil).MustStatus(t, 200).JSON(t, &sl)
	if sl.Total != 25 || sl.Matching != 16 || !strings.Contains(sl.Items[0].Number, "•••") {
		t.Fatalf("suppressions: %d of %d, %+v", sl.Matching, sl.Total, sl.Items[0])
	}
	linh.Do(http.MethodPost, ws+"/sms/suppressions/"+sl.Items[0].ID+"/remove", map[string]any{"reason": "Asked us by email"}).MustStatus(t, 409)

	// A Viewer sees costs and reports, never credentials, and can't change rules.
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	var seen texting.SmsOverview
	aisha.Do(http.MethodGet, ws+"/sms/overview", nil).MustStatus(t, 200).JSON(t, &seen)
	if seen.CanManage || seen.Providers[0].AccountHint != "" || seen.Providers[0].WebhookURL != "" || seen.Providers[0].Spend != 14.71 {
		t.Fatalf("a viewer's providers: %+v", seen.Providers[0])
	}
	aisha.Do(http.MethodGet, ws+"/sms/providers/"+ov.Providers[0].ID+"/numbers", nil).MustStatus(t, 403)
	aisha.Do(http.MethodPut, ws+"/sms/short-domain", map[string]any{"domain": "go.example.com"}).MustStatus(t, 403)
}

func TestSMSLive(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := workspace(t, env)
	// 15:30 in Chicago, the next one: inside the window in the US, outside
	// it in Europe, and after whatever the API scheduled with the real clock.
	loc, _ := time.LoadLocation("America/Chicago")
	n := time.Now().In(loc)
	clock := time.Date(n.Year(), n.Month(), n.Day(), 15, 30, 0, 0, loc)
	if clock.Before(n) {
		clock = clock.AddDate(0, 0, 1)
	}
	now := func() time.Time { return clock }
	s, tw, _ := service(env, now)

	// A receipt for a test text: verified, stored raw, answered 200, and
	// normalized by the job.
	twilio := one[string](t, env, `SELECT id::text FROM sms_providers WHERE adapter = 'twilio'`)
	token := one[string](t, env, `SELECT webhook_token FROM sms_providers WHERE adapter = 'twilio'`)
	hook := env.Server.URL + "/hooks/sms/twilio/" + token
	sofia := env.Client()
	sofia.SignInDemo("sofia@acme.io")
	var test texting.SmsMessageView
	sofia.Do(http.MethodPost, ws+"/sms/providers/"+twilio+"/test", map[string]any{"phone": "+1 312 555 0161"}).MustStatus(t, 403)
	marcus := env.Client()
	marcus.SignInDemo("marcus@acme.io")
	marcus.Do(http.MethodPost, ws+"/sms/providers/"+twilio+"/test", map[string]any{"phone": "+1 312 555 0161"}).MustStatus(t, 200).JSON(t, &test)
	if test.Status != "sent" || test.ProviderMessageID == "" || test.Sender != "+1 888 555 0142" {
		t.Fatalf("test text: %+v", test)
	}
	req, body := twiliotest.Receipt(demo.TwilioDemoWebhookSecret, hook, test.ProviderMessageID, "+13125550161", "+18885550142", "delivered", "")
	resp := post(t, hook, req, body)
	if resp != 200 {
		t.Fatalf("receipt answered %d", resp)
	}
	forged, fbody := twiliotest.Receipt("not-the-secret", hook, test.ProviderMessageID, "+13125550161", "+18885550142", "failed", "30003")
	if code := post(t, hook, forged, fbody); code != 401 {
		t.Fatalf("a forged receipt answered %d", code)
	}
	normalizeAll(t, env, s)
	if st := one[string](t, env, `SELECT status FROM channel_messages WHERE provider_message_id = $1`, test.ProviderMessageID); st != "delivered" {
		t.Fatalf("after the receipt: %s", st)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM sms_raw_events WHERE NOT signature_ok`); n != 1 {
		t.Fatalf("unverified raw events: %d", n)
	}

	// STOP from Sam Rivera: SMS consent withdrawn on every list, the number
	// suppressed as "Opted out by text", one reply; email untouched.
	sam := one[string](t, env, `SELECT id::text FROM contacts WHERE email = 'sam.rivera@outlook.com'`)
	emailBefore := one[int64](t, env, `SELECT count(*) FROM list_memberships WHERE contact_id = $1 AND status = 'subscribed'`, sam)
	req, body = twiliotest.Inbound(demo.TwilioDemoWebhookSecret, hook, "SMstop0001", "+13125550123", "+18885550142", "stop", "")
	if code := post(t, hook, req, body); code != 200 {
		t.Fatalf("inbound answered %d", code)
	}
	normalizeAll(t, env, s)
	if st := one[string](t, env, `SELECT status FROM channel_consents WHERE contact_id = $1 AND channel = 'sms'`, sam); st != "opted_out" {
		t.Fatalf("Sam's SMS consent: %s", st)
	}
	if r := one[string](t, env, `SELECT reason FROM channel_suppressions WHERE phone = '+13125550123' AND removed_at IS NULL`); r != "opted_out" {
		t.Fatalf("suppression: %s", r)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM list_memberships WHERE contact_id = $1 AND status = 'subscribed'`, sam); n != emailBefore {
		t.Fatalf("email memberships changed: %d → %d", emailBefore, n)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM channel_messages WHERE recipient = '+13125550123' AND kind = 'reply'`); n != 1 {
		t.Fatalf("opt-out replies: %d", n)
	}
	// START brings back exactly what STOP removed.
	req, body = twiliotest.Inbound(demo.TwilioDemoWebhookSecret, hook, "SMstart0001", "+13125550123", "+18885550142", "START", "")
	post(t, hook, req, body)
	normalizeAll(t, env, s)
	if st := one[string](t, env, `SELECT status FROM channel_consents WHERE contact_id = $1 AND channel = 'sms'`, sam); st != "subscribed" {
		t.Fatalf("after START: %s", st)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM channel_suppressions WHERE phone = '+13125550123' AND removed_at IS NULL`); n != 0 {
		t.Fatalf("still suppressed after START")
	}

	// A campaign to the summit list, sent now: the US and Canada go at
	// once, Europe waits for 09:00 local, Ireland is skipped.
	var c texting.SmsCampaign
	list := one[string](t, env, `SELECT id::text FROM lists WHERE name = 'Acme Summit 2026'`)
	sofia.Do(http.MethodPost, ws+"/sms-campaigns", map[string]any{"name": "Summit: parking", "lists": []string{list},
		"body": "Parking at Lakeside Hall is free with your badge."}).MustStatus(t, 201).JSON(t, &c)
	var rv texting.SmsReview
	sofia.Do(http.MethodPut, ws+"/sms-campaigns/"+c.ID+"/send-settings", map[string]any{"mode": "now"}).MustStatus(t, 200).JSON(t, &rv)
	acks := []string{}
	for _, ch := range rv.Checks {
		if ch.Status == "warn" {
			acks = append(acks, ch.Key)
		}
	}
	// Above 1,000 recipients or $100 the count is typed; this one isn't.
	if rv.Large.Required {
		t.Fatalf("a send of %d texts for $%.2f asks for the typed count", rv.Funnel.ToSend, rv.Cost)
	}
	var res texting.SmsSendResult
	sofia.Do(http.MethodPost, ws+"/sms-campaigns/"+c.ID+"/send", map[string]any{"fingerprint": rv.Fingerprint, "acknowledged": acks}).MustStatus(t, 200).JSON(t, &res)
	if res.Recipients != 846 { // the 866 who can receive texts now, Ireland skipped; Sam is back after START
		t.Fatalf("recipients: %d", res.Recipients)
	}
	if _, _, err := s.Dispatch(ctx, 30*time.Second); err != nil {
		t.Fatal(err)
	}
	sent := one[int64](t, env, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1 AND sent_at IS NOT NULL`, c.ID)
	held := one[int64](t, env, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1 AND status = 'queued'`, c.ID)
	if sent == 0 || held == 0 {
		t.Fatalf("sent %d, waiting %d", sent, held)
	}
	if n := one[int64](t, env, `SELECT count(*) FROM channel_messages WHERE campaign_id = $1 AND sent_at IS NOT NULL AND country IN ('GB','DE','NL')`, c.ID); n != 0 {
		t.Fatalf("%d texts reached Europe at night", n)
	}
	if len(tw.Sent()) == 0 {
		t.Fatalf("Twilio's fake got nothing")
	}
	if a := one[int64](t, env, `SELECT count(*) FROM audit_events WHERE action = 'campaign.send_started' AND target_id = $1`, c.ID); a != 1 {
		t.Fatalf("audit entries: %d", a)
	}
	_ = w
	_ = uuid.Nil
	_ = url.Values{}
	_ = fmt.Sprint
}

func post(t *testing.T, hook string, r *http.Request, body []byte) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, hook, strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range r.Header {
		req.Header[k] = v
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

// normalizeAll runs the normalize job on every stored webhook.
func normalizeAll(t *testing.T, env *apptest.Env, s *texting.Service) {
	t.Helper()
	ctx := context.Background()
	rows, err := env.App.Pool.Query(ctx, `SELECT id FROM sms_raw_events WHERE normalized_at IS NULL AND signature_ok ORDER BY received_at`)
	if err != nil {
		t.Fatal(err)
	}
	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if err := s.Normalize(ctx, id); err != nil {
			t.Fatalf("normalize %s: %v", id, err)
		}
	}
}
