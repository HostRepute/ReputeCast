package notify_test

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/notify"
)

func TestDemoNotificationCenter(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	c := env.Client()
	c.SignInDemo("linh@acme.io")

	var list map[string]any
	c.Do("GET", "/workspaces/acme-newsletter/notifications", nil).MustStatus(t, 200).JSON(t, &list)
	items := list["items"].([]any)
	// The Notifications board, with staging.acme.io's "Integration needs
	// attention" (integrations hub) after the unread ones, and magic-link's
	// bounce alert of 3 Oct (transactional) last.
	if len(items) != 10 || list["unread"].(float64) != 3 || list["retentionDays"].(float64) != 90 {
		t.Fatalf("center: %d items, %v unread", len(items), list["unread"])
	}
	want := []string{"Approval requested", "Send auto-paused", "Provider down", "Integration needs attention", "Approval given", "Webhooks silent", "Approval invalidated",
		"Invite accepted", "HostRepute alert", "Transactional delivery alert"}
	for i, w := range want {
		it := items[i].(map[string]any)
		if it["title"] != w {
			t.Fatalf("item %d: %v, want %s", i, it["title"], w)
		}
		if (i < 3) == it["read"].(bool) {
			t.Fatalf("item %d read state: %v", i, it)
		}
	}
	linhs := map[string]bool{}
	for _, it := range items {
		linhs[it.(map[string]any)["id"].(string)] = true
	}
	first := items[0].(map[string]any)
	if first["category"] != "approvals" || first["categoryLabel"] != "Approvals" || first["severity"] != "warn" ||
		first["summary"] == first["body"] {
		t.Fatalf("approval item: %v", first)
	}

	// Tabs filter by category; Unread only shows unread.
	c.Do("GET", "/workspaces/acme-newsletter/notifications?filter=providers", nil).MustStatus(t, 200).JSON(t, &list)
	if n := len(list["items"].([]any)); n != 2 || list["total"].(float64) != 2 || list["unread"].(float64) != 3 {
		t.Fatalf("providers tab: %d %v", n, list)
	}
	c.Do("GET", "/workspaces/acme-newsletter/notifications?filter=unread&limit=2", nil).MustStatus(t, 200).JSON(t, &list)
	if n := len(list["items"].([]any)); n != 2 || list["nextCursor"] != "2" {
		t.Fatalf("unread page: %d %v", n, list["nextCursor"])
	}

	// Opening one marks it read; Mark all as read clears the rest.
	c.Do("POST", "/workspaces/acme-newsletter/notifications/"+first["id"].(string)+"/read", nil).MustStatus(t, 204)
	var wme map[string]any
	c.Do("GET", "/workspaces/acme-newsletter/me", nil).MustStatus(t, 200).JSON(t, &wme)
	if wme["unread"].(float64) != 2 {
		t.Fatalf("bell count after reading one: %v", wme["unread"])
	}
	var marked map[string]any
	c.Do("POST", "/workspaces/acme-newsletter/notifications/read-all", nil).MustStatus(t, 200).JSON(t, &marked)
	if marked["marked"].(float64) != 2 {
		t.Fatalf("marked: %v", marked)
	}
	// Someone else's notification can't be touched.
	a := env.Client()
	a.SignInDemo("aisha@acme.io")
	a.Do("GET", "/workspaces/acme-newsletter/notifications", nil).MustStatus(t, 200).JSON(t, &list)
	// Her own: the 3 Oct transactional delivery alert (she holds transactional:view).
	for _, it := range list["items"].([]any) {
		m := it.(map[string]any)
		if linhs[m["id"].(string)] || m["kind"] != "transactional_alert" {
			t.Fatalf("Aisha sees Linh's notifications: %v", list)
		}
	}
}

func TestPreferencesAndDelivery(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	u := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	ws := env.CreateWorkspace("Acme Newsletter", u.ID)
	c := env.Client()
	c.SignIn("linh@acme.io", "")

	var prefs map[string]any
	c.Do("GET", "/me/notification-preferences?workspace="+ws.Slug, nil).MustStatus(t, 200).JSON(t, &prefs)
	events := prefs["events"].([]any)
	if len(events) != len(notify.Events) || prefs["digest"] != "immediate" || prefs["quietHours"] != nil {
		t.Fatalf("defaults: %v", prefs)
	}
	if e := events[0].(map[string]any); e["type"] != "approval_requested" || e["inApp"] != true || e["email"] != true || e["audience"] != "Only for approvers" {
		t.Fatalf("first event: %v", e)
	}
	c.Do("PUT", "/me/notification-preferences", map[string]any{"events": []any{map[string]any{"type": "nope", "inApp": true, "email": true}}, "digest": "immediate"}).MustStatus(t, 422)
	c.Do("PUT", "/me/notification-preferences", map[string]any{
		"events": []any{
			map[string]any{"type": "webhooks_silent", "inApp": false, "email": false},
			map[string]any{"type": "invite_accepted", "inApp": false, "email": true},
			map[string]any{"type": "send_paused", "inApp": true, "email": true},
			map[string]any{"type": "approval_requested", "inApp": true, "email": true},
		},
		"quietHours": map[string]any{"start": "00:00", "end": "23:59"},
		"digest":     "immediate",
	}).MustStatus(t, 204)
	c.Do("GET", "/me/notification-preferences", nil).MustStatus(t, 200).JSON(t, &prefs)
	if q := prefs["quietHours"].(map[string]any); q["start"] != "00:00" || q["end"] != "23:59" {
		t.Fatalf("quiet hours: %v", prefs)
	}

	wsID := ws.ID
	send := func(kind, title string) {
		if err := notify.ToUser(ctx, env.App.Pool, u.ID, notify.Notification{WorkspaceID: &wsID, Kind: kind, Title: title, Body: title + "."}); err != nil {
			t.Fatal(err)
		}
	}
	send(notify.KindWebhooksSilent, "Webhooks silent")       // both off: not stored
	send(notify.KindInviteAccepted, "Invite accepted")       // email only: hidden in the center
	send(notify.KindSendPaused, "Send auto-paused")          // urgent: emailed even in quiet hours
	send(notify.KindApprovalRequested, "Approval requested") // held by quiet hours
	var stored, visible int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE in_app) FROM notifications WHERE user_id = $1`, u.ID).Scan(&stored, &visible)
	if stored != 3 || visible != 2 {
		t.Fatalf("stored %d, visible %d", stored, visible)
	}

	mail := &fakeMail{available: true}
	w := &notify.EmailWorker{Pool: env.App.Pool, Mail: func() mailer.Sender { return mail }, PublicURL: func(context.Context) string { return "https://reputecast.acme.io" },
		Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := w.Work(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if len(mail.sent) != 1 || mail.sent[0].Subject != "[Acme Newsletter] Send auto-paused" {
		t.Fatalf("quiet hours let only the urgent email through: %+v", mail.sent)
	}
	// After quiet hours (none now), the held emails go out; nothing twice.
	c.Do("PUT", "/me/notification-preferences", map[string]any{"events": []any{}, "digest": "immediate"}).MustStatus(t, 204)
	if err := w.Work(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if len(mail.sent) != 3 {
		t.Fatalf("emails after quiet hours: %d", len(mail.sent))
	}
	if err := w.Work(ctx, nil); err != nil || len(mail.sent) != 3 {
		t.Fatalf("emails must not repeat: %d %v", len(mail.sent), err)
	}

	// Without a System provider, email is skipped silently.
	send(notify.KindSendPaused, "Paused again")
	none := &notify.EmailWorker{Pool: env.App.Pool, Mail: func() mailer.Sender { return mailer.Unavailable{} }, Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: time.Now}
	if err := none.Work(ctx, nil); err != nil {
		t.Fatal(err)
	}
	var skipped int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND email_status = 'skipped'`, u.ID).Scan(&skipped)
	if skipped != 1 {
		t.Fatalf("skipped: %d", skipped)
	}
	var audited int
	_ = env.App.Pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action = 'account.notification_preferences_update'`).Scan(&audited)
	if audited != 2 {
		t.Fatalf("audit entries: %d", audited)
	}
}

func TestSummarize(t *testing.T) {
	p := notify.NotificationPreferences{Timezone: "America/New_York", QuietHours: &notify.NotificationQuietHours{Start: "22:00", End: "07:00"}}
	for _, e := range notify.Events {
		p.Events = append(p.Events, notify.NotificationEventPreference{Type: e.Key, InApp: true, Email: e.DefaultEmail})
	}
	s := notify.Summarize(p)
	if s.InApp != "All events" || s.Email != "Approvals, auto-pauses, provider down, integration problems, post drafts, transactional delivery alerts, SMTP relay outages, held digest issues, paused digests, failing digest feeds, digest drafts, AI usage notices, rejected AI keys, payment problems, verification jobs, low verification credits, seed sign-in failures, website domain problems, held referrals, reward codes running low, failing reward webhooks, partner link problems" ||
		s.QuietHours != "22:00–07:00 America/New_York, email only" {
		t.Fatalf("summary: %+v", s)
	}
	// (The AI assistant's events aren't translated yet: they're off here.)
	for i := range p.Events {
		switch p.Events[i].Type {
		case notify.KindAIUsage, notify.KindAIKeyRejected:
			p.Events[i].Email = false
		}
	}

	// In the member's app language, starting with a capital in any script.
	// (The recurring digests', verification, placement, website and Grow's events aren't translated yet: they're off here.)
	for i := range p.Events {
		switch p.Events[i].Type {
		case notify.KindDigestIssueHeld, notify.KindDigestPaused, notify.KindDigestFeedFailing, notify.KindDigestDraftReady,
			notify.KindVerificationEnded, notify.KindVerificationCreditsLow, notify.KindPlacementFinished, notify.KindSeedSignInFailed,
			notify.KindWebsiteDomain, notify.KindReferralsHeld, notify.KindRewardCodesLow, notify.KindRewardWebhookFailing, notify.KindPartnerLinkProblem:
			p.Events[i].Email = false
		}
	}
	vi := i18n.For(i18n.Vietnamese)
	s = notify.SummarizeIn(vi, p)
	if s.InApp != "Tất cả sự kiện" || s.Email != "Phê duyệt, tự tạm dừng, nhà cung cấp gặp sự cố, sự cố tích hợp, bản nháp bài viết, cảnh báo gửi email giao dịch, sự cố SMTP relay, payment problems" ||
		s.QuietHours != "22:00–07:00 America/New_York, chỉ email" {
		t.Fatalf("Vietnamese summary: %+v", s)
	}
	p.QuietHours = nil
	for i := range p.Events {
		p.Events[i].Email = p.Events[i].Type == notify.KindWebhooksSilent
	}
	if s = notify.SummarizeIn(vi, p); s.Email != "Webhook im lặng" || s.QuietHours != "Tắt" {
		t.Fatalf("Vietnamese summary, one event: %+v", s)
	}
}

// Every event type's English short name is the catalog's short.<key>, so
// the summary reads the same whichever one it takes.
func TestEventShortNamesMatchTheCatalog(t *testing.T) {
	for _, e := range notify.Events {
		p := notify.NotificationPreferences{}
		for _, x := range notify.Events {
			p.Events = append(p.Events, notify.NotificationEventPreference{Type: x.Key, Email: x.Key == e.Key})
		}
		got := notify.SummarizeIn(i18n.Default, p).Email
		want := strings.ToUpper(e.Short[:1]) + e.Short[1:]
		if got != want {
			t.Errorf("%s: the summary says %q, the event type %q", e.Key, got, want)
		}
	}
}

type fakeMail struct {
	mu        sync.Mutex
	available bool
	sent      []mailer.Message
}

func (f *fakeMail) SendSystem(_ context.Context, m mailer.Message) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.available {
		return "", mailer.ErrNoSystemProvider
	}
	f.sent = append(f.sent, m)
	return "Postmark", nil
}

func (f *fakeMail) SystemAvailable(context.Context, *uuid.UUID) bool { return f.available }
