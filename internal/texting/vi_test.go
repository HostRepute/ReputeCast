package texting_test

import (
	"context"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
	"github.com/hostrepute/reputecast/internal/store"
	"github.com/hostrepute/reputecast/internal/texting"
)

// SMS's own words read in the reader's language: a problem, the Send SMS
// step's publish checks, what ReputeCast stored in English (the short-link
// domain's check, a suppression's context, a provider's answer), the
// notifications, kept as messages while an English request or a job made
// them, and the short link's expiry page. API clients that don't ask, and
// English readers, read English as before.
func TestSMSInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	ctx := context.Background()
	w := workspace(t, env)
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	vi := env.Client()
	vi.SignInDemo("marcus@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	marcus := env.Client()
	marcus.SignInDemo("marcus@acme.io")

	// A problem, in the request's language.
	blank := map[string]any{"name": "  "}
	if pr := vi.Do(http.MethodPost, ws+"/sms-campaigns", blank).MustStatus(t, 422).Problem(t); pr["detail"] != "Hãy đặt tên cho chiến dịch." {
		t.Fatalf("Vietnamese problem: %v", pr)
	}
	if pr := marcus.Do(http.MethodPost, ws+"/sms-campaigns", blank).MustStatus(t, 422).Problem(t); pr["detail"] != "Give the campaign a name." {
		t.Fatalf("English problem: %v", pr)
	}
	if pr := vi.Do(http.MethodPost, ws+"/sms/route-test", map[string]any{"phone": "abc"}).MustStatus(t, 422).Problem(t); pr["detail"] != "Đây không phải là số điện thoại. Hãy dùng định dạng quốc tế, ví dụ +44 7700 900456." {
		t.Fatalf("Vietnamese route test: %v", pr)
	}

	// The reminders flow's publish checks: quiet hours, and Ireland, where
	// ACME isn't approved yet.
	flow := one[string](t, env, `SELECT id::text FROM automations WHERE name = 'Acme Summit reminders'`)
	var pub struct {
		Checks []struct{ Status, Label string }
	}
	vi.Do(http.MethodGet, ws+"/automations/"+flow+"/publish-preview", nil).MustStatus(t, 200).JSON(t, &pub)
	labels := map[string]string{}
	for _, c := range pub.Checks {
		labels[c.Label] = c.Status
	}
	if labels["Giờ yên lặng: tin nhắn được gửi trong khung 09:00–20:00 theo múi giờ của từng liên hệ; ngoài khung giờ đó, tin nhắn phải chờ"] != "ok" ||
		labels["ACME chưa được phê duyệt ở Ireland: 20 liên hệ ở đó bị bỏ qua cho đến khi được phê duyệt"] != "warn" {
		t.Fatalf("Vietnamese publish checks: %+v", pub.Checks)
	}

	// What ReputeCast stored in English reads in Vietnamese: the short-link
	// domain's check, a suppression's context, a viewer's reason.
	var rules texting.SmsRules
	vi.Do(http.MethodGet, ws+"/sms/rules", nil).MustStatus(t, 200).JSON(t, &rules)
	if rules.ShortDomain.Detail != "CNAME trỏ đến reputecast.acme.io." {
		t.Fatalf("Vietnamese DNS check: %q", rules.ShortDomain.Detail)
	}
	marcus.Do(http.MethodGet, ws+"/sms/rules", nil).MustStatus(t, 200).JSON(t, &rules)
	if rules.ShortDomain.Detail != "CNAME points at reputecast.acme.io." {
		t.Fatalf("English DNS check: %q", rules.ShortDomain.Detail)
	}
	var sl texting.SmsSuppressionList
	vi.Do(http.MethodGet, ws+"/sms/suppressions", nil).MustStatus(t, 200).JSON(t, &sl)
	contexts := 0
	for _, it := range sl.Items {
		switch it.Context {
		case "Tin xác nhận đăng ký":
			contexts++
		case "Opt-in confirmation":
			t.Fatalf("a context in English: %+v", it)
		}
	}
	if contexts == 0 {
		t.Fatalf("no confirmation context: %+v", sl.Items)
	}
	aisha := env.Client()
	aisha.SignInDemo("aisha@acme.io")
	aisha.Header = http.Header{"Accept-Language": {"vi"}}
	aisha.Do(http.MethodGet, ws+"/sms/suppressions", nil).MustStatus(t, 200).JSON(t, &sl)
	if want := authz.DeniedText(i18n.For(i18n.Vietnamese), authz.SubscribersEdit, nil); sl.Reason != want || strings.Contains(sl.Reason, "needs") {
		t.Fatalf("a viewer's reason: %q, want %q", sl.Reason, want)
	}

	// An English request brings a provider back: each member reads the
	// notification in their own language, the row keeps the English.
	twilio := one[string](t, env, `SELECT id::text FROM sms_providers WHERE adapter = 'twilio'`)
	if _, err := env.App.Pool.Exec(ctx, `UPDATE sms_providers SET status = 'down', health = health - 'silentSince' WHERE id = $1`, twilio); err != nil {
		t.Fatal(err)
	}
	linh.Do(http.MethodPost, ws+"/sms/providers/"+twilio+"/check", nil).MustStatus(t, 200)
	if title := one[string](t, env, `SELECT title FROM notifications WHERE kind = $1 ORDER BY created_at DESC LIMIT 1`, notify.KindSMSProviderDown); title != "Twilio is back" {
		t.Fatalf("stored title: %q", title)
	}
	if n := latest(t, vi, notify.KindSMSProviderDown); n.Title != "Twilio đã hoạt động lại" ||
		n.Body != "Twilio đã phản hồi lại. Tin nhắn bị tạm giữ được gửi đi trong khung giờ gửi của từng người nhận." {
		t.Fatalf("Vietnamese notification: %+v", n)
	}
	if n := latest(t, marcus, notify.KindSMSProviderDown); n.Title != "Twilio is back" {
		t.Fatalf("English notification: %+v", n)
	}

	// The health job finds the credentials rejected: the provider's answer
	// keeps its status, ReputeCast's words read in the reader's language.
	s, tw, _ := service(env, time.Now)
	tw.Rejected = true
	prov, err := env.App.Q.SmsProvider(ctx, store.SmsProviderParams{ID: uuid.MustParse(twilio), WorkspaceID: w.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CheckProvider(ctx, prov); err == nil {
		t.Fatal("rejected credentials checked out")
	}
	if n := latest(t, vi, notify.KindSMSProviderDown); n.Title != "Twilio đã từ chối thông tin xác thực" {
		t.Fatalf("Vietnamese notification: %+v", n)
	}
	var ov texting.SmsOverview
	vi.Do(http.MethodGet, ws+"/sms/overview", nil).MustStatus(t, 200).JSON(t, &ov)
	if ov.Providers[0].Name != "Twilio" || ov.Providers[0].Answer != "401 Unauthorized · thông tin xác thực không hợp lệ" {
		t.Fatalf("Vietnamese answer: %+v", ov.Providers[0])
	}
	marcus.Do(http.MethodGet, ws+"/sms/overview", nil).MustStatus(t, 200).JSON(t, &ov)
	if ov.Providers[0].Answer != "401 Unauthorized · invalid credentials" {
		t.Fatalf("English answer: %q", ov.Providers[0].Answer)
	}

	// The short link's expiry page, in the visitor's language.
	for lang, want := range map[string]string{"vi": "Liên kết này đã hết hạn. Liên kết rút gọn trong tin nhắn dùng được trong 12 tháng.\n",
		"": "This link has expired. Short links in texts work for 12 months.\n"} {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, env.Server.URL+"/t/s/zzzzzz", nil)
		if lang != "" {
			req.Header.Set("Accept-Language", lang)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != 404 || string(body) != want {
			t.Fatalf("expired link (%q): %d %q", lang, resp.StatusCode, body)
		}
	}

	// The budget at its alert level: Linh reads Vietnamese (her own choice),
	// the amounts in her region format; Marcus reads English, as the row
	// keeps it.
	linh.Do(http.MethodPut, "/me/language", map[string]any{"language": "vi", "regionFormat": "auto", "hourCycle": "auto", "weekStart": "auto"}).MustStatus(t, 200)
	marcus.Do(http.MethodGet, ws+"/sms/overview", nil).MustStatus(t, 200).JSON(t, &ov)
	spent := ov.Tiles.Spend
	budget := math.Ceil(spent/0.9*100) / 100
	if _, err := env.App.Pool.Exec(ctx, `UPDATE sms_settings SET budget = $1 WHERE workspace_id = $2`, budget, w.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.BudgetAlerts(ctx, w); err != nil {
		t.Fatal(err)
	}
	en, vn := i18n.Default.Region, i18n.For(i18n.Vietnamese).Region
	if n := latest(t, linh, notify.KindSMSBudget); n.Title != "Ngân sách SMS đạt 80%" ||
		n.Body != "Đã chi "+vn.Money(spent, 2)+" trên "+vn.Money(budget, 2)+" cho tin nhắn trong tháng này." {
		t.Fatalf("Linh's budget notice: %+v", n)
	}
	if n := latest(t, marcus, notify.KindSMSBudget); n.Title != "SMS budget at 80%" ||
		n.Body != en.Money(spent, 2)+" of "+en.Money(budget, 2)+" spent on texts this month." {
		t.Fatalf("Marcus's budget notice: %+v", n)
	}
	if body := one[string](t, env, `SELECT n.body FROM notifications n JOIN users u ON u.id = n.user_id WHERE n.kind = $1 AND u.email = 'linh@acme.io'`,
		notify.KindSMSBudget); body != en.Money(spent, 2)+" of "+en.Money(budget, 2)+" spent on texts this month." {
		t.Fatalf("Linh's stored body: %q", body)
	}
}

// latest is the newest notification of a kind, as a member reads it.
func latest(t *testing.T, c *apptest.Client, kind string) notify.NotificationItem {
	t.Helper()
	var list notify.NotificationList
	c.Do(http.MethodGet, ws+"/notifications", nil).MustStatus(t, 200).JSON(t, &list)
	var out *notify.NotificationItem
	for i := range list.Items {
		if it := list.Items[i]; it.Kind == kind && (out == nil || it.CreatedAt.After(out.CreatedAt)) {
			out = &list.Items[i]
		}
	}
	if out == nil {
		t.Fatalf("no %s notification: %+v", kind, list.Items)
	}
	return *out
}
