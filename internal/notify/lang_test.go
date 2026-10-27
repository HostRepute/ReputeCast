package notify_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mailer"
	"github.com/hostrepute/reputecast/internal/notify"
)

// A catalog standing in for a package that sends a notification.
var sender = i18n.MustLoad(fstest.MapFS{
	"messages/en.json": {Data: []byte(`{
		"title": "{site} answered {status}",
		"body": "Notifications to {count, plural, one {# subscriber} other {# subscribers}} are held."
	}`)},
	"messages/vi.json": {Data: []byte(`{
		"title": "{site} trả về {status}",
		"body": "Thông báo đến {count, plural, other {# người đăng ký}} đang bị tạm giữ."
	}`)},
}, "notifytest")

// A notification is written once and read later by every member it
// reaches: its kept text reads in each reader's language, in the center and
// in email, while the row keeps its English.
func TestNotificationsReadInEachReadersLanguage(t *testing.T) {
	env := apptest.New(t)
	ctx := context.Background()
	u := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	ws := env.CreateWorkspace("Acme Newsletter", u.ID)
	wsID := ws.ID
	args := i18n.Args{"site": "help.acme.io", "status": 404, "count": 1204}
	if err := notify.ToUser(ctx, env.App.Pool, u.ID, notify.Notification{
		WorkspaceID: &wsID, Kind: notify.KindWhatsAppAttention, Severity: "bad",
		TitleText: sender.M("title").Keep(args), BodyText: sender.M("body").Keep(args), Link: "/w/" + ws.Slug + "/providers/push",
	}); err != nil {
		t.Fatal(err)
	}
	// The row keeps the English, for readers that don't know the message.
	var title, body string
	if err := env.App.Pool.QueryRow(ctx, `SELECT title, body FROM notifications WHERE user_id = $1`, u.ID).Scan(&title, &body); err != nil {
		t.Fatal(err)
	}
	if title != "help.acme.io answered 404" || body != "Notifications to 1,204 subscribers are held." {
		t.Fatalf("stored: %q, %q", title, body)
	}

	item := func(c *apptest.Client) map[string]any {
		var list map[string]any
		c.Do("GET", "/workspaces/"+ws.Slug+"/notifications", nil).MustStatus(t, 200).JSON(t, &list)
		return list["items"].([]any)[0].(map[string]any)
	}
	c := env.Client()
	c.SignIn("linh@acme.io", "")
	if it := item(c); it["title"] != "help.acme.io answered 404" || it["summary"] != "Notifications to 1,204 subscribers are held." || it["categoryLabel"] != "Providers" {
		t.Fatalf("English: %v", it)
	}
	c.Header = http.Header{"Accept-Language": {"vi"}}
	it := item(c)
	if it["title"] != "help.acme.io trả về 404" || it["body"] != "Thông báo đến 1.204 người đăng ký đang bị tạm giữ." ||
		it["summary"] != it["body"] || it["categoryLabel"] != "Nhà cung cấp" {
		t.Fatalf("Vietnamese: %v", it)
	}

	// A notification passed as English only (a package that hasn't kept its
	// text yet) reads in English to everyone.
	if err := notify.ToUser(ctx, env.App.Pool, u.ID, notify.Notification{WorkspaceID: &wsID, Kind: notify.KindWebhooksSilent, Title: "Webhooks silent", Body: "No events for 6 hours."}); err != nil {
		t.Fatal(err)
	}
	var list map[string]any
	c.Do("GET", "/workspaces/"+ws.Slug+"/notifications", nil).MustStatus(t, 200).JSON(t, &list)
	if first := list["items"].([]any)[0].(map[string]any); first["title"] != "Webhooks silent" || first["categoryLabel"] != "Nhà cung cấp" {
		t.Fatalf("English-only notification: %v", first)
	}

	// The email goes out in the recipient's own language.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE users SET language = 'vi' WHERE id = $1`, u.ID); err != nil {
		t.Fatal(err)
	}
	mail := &fakeMail{available: true}
	w := &notify.EmailWorker{Pool: env.App.Pool, Mail: func() mailer.Sender { return mail }, PublicURL: func(context.Context) string { return "https://reputecast.acme.io" },
		Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := w.Work(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if len(mail.sent) != 1 || !strings.Contains(mail.sent[0].Subject, "help.acme.io trả về 404") ||
		!strings.Contains(mail.sent[0].Text, "Thông báo đến 1.204 người đăng ký đang bị tạm giữ.") {
		t.Fatalf("email in Vietnamese: %+v", mail.sent)
	}
}

// Notification preferences and their problems read in the request's language.
func TestPreferencesInVietnamese(t *testing.T) {
	env := apptest.New(t)
	u := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	env.CreateWorkspace("Acme Newsletter", u.ID)
	c := env.Client()
	c.SignIn("linh@acme.io", "")
	c.Header = http.Header{"Accept-Language": {"vi"}}
	var prefs map[string]any
	c.Do("GET", "/me/notification-preferences", nil).MustStatus(t, 200).JSON(t, &prefs)
	events := prefs["events"].([]any)
	byType := map[string]map[string]any{}
	for _, e := range events {
		m := e.(map[string]any)
		byType[m["type"].(string)] = m
	}
	if e := byType["send_paused"]; e["label"] != "Đợt gửi tự tạm dừng" || e["audience"] != "Những người có thể gửi chiến dịch" {
		t.Fatalf("send_paused: %v", e)
	}
	if e := byType["webhooks_silent"]; e["label"] != "Webhook im lặng" || e["audience"] != "Những người có thể quản lý nhà cung cấp gửi thư" {
		t.Fatalf("webhooks_silent: %v", e)
	}
	var p map[string]any
	c.Do("PUT", "/me/notification-preferences", map[string]any{"events": []any{map[string]any{"type": "nope", "inApp": true, "email": true}}, "digest": "immediate"}).MustStatus(t, 422).JSON(t, &p)
	if p["detail"] != "Không có loại sự kiện nope." {
		t.Fatalf("problem: %v", p)
	}
	c.Do("PUT", "/me/notification-preferences", map[string]any{"events": []any{}, "quietHours": map[string]any{"start": "22:00", "end": "22:00"}, "digest": "immediate"}).MustStatus(t, 422).JSON(t, &p)
	if p["detail"] != "Giờ yên lặng cần giờ bắt đầu và giờ kết thúc khác nhau." {
		t.Fatalf("problem: %v", p)
	}
}

// Alert preferences say who gets each alert in people's words, never a
// permission key (review GS-9).
func TestAudiencesNameWho(t *testing.T) {
	key := regexp.MustCompile(`\b[a-z]+:[a-z]+\b`)
	for _, e := range notify.Events {
		if key.MatchString(e.Audience) {
			t.Errorf("%s: %q names a permission key", e.Key, e.Audience)
		}
	}
}

// Every event type and tab has its English in the catalog, the same as the
// code's, and its Vietnamese.
func TestEventTypesMatchTheCatalog(t *testing.T) {
	var c *i18n.Catalog
	for _, x := range i18n.Catalogs() {
		if x.Namespace == "notify" {
			c = x
		}
	}
	if c == nil {
		t.Fatal("no notify catalog")
	}
	source := func(key string) string {
		m, err := c.Lookup(key)
		if err != nil {
			return ""
		}
		return m.Source()
	}
	for _, e := range notify.Events {
		if got := source("event." + e.Key); got != e.Label {
			t.Errorf("event.%s: %q, the code says %q", e.Key, got, e.Label)
		}
		if got := source("audience." + e.Key); got != e.Audience {
			t.Errorf("audience.%s: %q, the code says %q", e.Key, got, e.Audience)
		}
	}
	for _, tab := range []string{"approvals", "sending", "providers", "team", "hostrepute", "other", "integrations", "transactional", "ai", "earn", "website", "grow"} {
		if got := source("category." + tab); got != notify.CategoryLabel(tab) {
			t.Errorf("category.%s: %q, the code says %q", tab, got, notify.CategoryLabel(tab))
		}
	}
	if missing := c.Missing(i18n.Vietnamese); len(missing) > 0 {
		t.Errorf("not in Vietnamese: %v", missing)
	}
}
