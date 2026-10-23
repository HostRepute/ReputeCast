package push_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/push"
	"github.com/hostrepute/reputecast/internal/push/webpush/webpushtest"
)

// Push's problems and labels read in the request's language; the
// worker-missing notification is kept as messages, so each member reads it
// in theirs while its row keeps the English; an expired link's page
// follows the reader's browser. API clients still get English.
func TestPushInVietnamese(t *testing.T) {
	f := newFixture(t)
	env, p := f.env, f.path
	vi := env.Client()
	vi.SignIn("owner@x22.test", "")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	// A problem, before and after push is on.
	var prob map[string]any
	vi.Do(http.MethodPut, p+"/push/contact", map[string]any{"contact": "ops@x22.test"}).MustStatus(t, 409).JSON(t, &prob)
	if prob["detail"] != "Hãy bật thông báo đẩy trước: địa chỉ liên hệ đi kèm với khóa." {
		t.Fatalf("problem: %v", prob)
	}
	site := f.pushOn(t, 5)
	vi.Do(http.MethodPost, p+"/push/enable", map[string]any{}).MustStatus(t, 409).JSON(t, &prob)
	if prob["detail"] != "Thông báo đẩy đã được bật cho không gian làm việc này." {
		t.Fatalf("problem: %v", prob)
	}
	f.owner.Do(http.MethodPost, p+"/push/enable", map[string]any{}).MustStatus(t, 409).JSON(t, &prob)
	if prob["detail"] != "Push is already on for this workspace." {
		t.Fatalf("English problem: %v", prob)
	}

	// Labels: a duplicate's name and a new prompt's words for its readers,
	// in the language of whoever makes them.
	var c, dup push.PushCampaign
	f.owner.Do(http.MethodPost, p+"/push-campaigns", map[string]any{"name": "Flags", "title": "Flags", "link": "help.x22.test/flags"}).MustStatus(t, 201).JSON(t, &c)
	vi.Do(http.MethodPost, p+"/push-campaigns/"+c.ID+"/duplicate", nil).MustStatus(t, 201).JSON(t, &dup)
	if dup.Name != "Bản sao của Flags" {
		t.Fatalf("duplicate: %q", dup.Name)
	}
	var pr push.PushPrompt
	vi.Do(http.MethodPost, p+"/push/prompts", map[string]any{"name": "Blog", "siteId": site.ID}).MustStatus(t, 201).JSON(t, &pr)
	if pr.Draft.Title != "Nhận bài viết mới qua thông báo" || pr.Draft.Allow != "Cho phép" || pr.Draft.NotNow != "Để sau" {
		t.Fatalf("prompt defaults: %+v", pr.Draft)
	}

	// The worker goes missing: members hear of it, each in their language.
	f.subscribe(t, "chrome", chromeUA, nil)
	f.subscribe(t, "firefox", firefoxUA, nil)
	env.App.PushFake().Serve(siteURL+"/"+push.WorkerFile, webpushtest.Page{Status: http.StatusNotFound})
	f.owner.Do(http.MethodPost, p+"/push/sites/"+site.ID+"/check", nil).MustStatus(t, 200)
	var title, body string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT title, body FROM notifications WHERE workspace_id = $1 AND kind = 'push_worker_missing'`, f.ws.ID).
		Scan(&title, &body); err != nil {
		t.Fatal(err)
	}
	if title != "Push service worker missing on help.x22.test" ||
		body != "https://help.x22.test/rc-push-sw.js answered 404. Notifications to help.x22.test's 2 subscribers are held, and new subscriptions there fail, until the file is back. Upload rc-push-sw.js to the site's root, then check again in Push setup." {
		t.Fatalf("stored: %q / %q", title, body)
	}
	var list map[string]any
	vi.Do(http.MethodGet, p+"/notifications", nil).MustStatus(t, 200).JSON(t, &list)
	n := list["items"].([]any)[0].(map[string]any)
	if n["title"] != "Thiếu service worker thông báo đẩy trên help.x22.test" ||
		n["summary"] != "https://help.x22.test/rc-push-sw.js trả về 404 · thông báo đến 2 người đăng ký đang bị tạm giữ" {
		t.Fatalf("Vietnamese notification: %v", n)
	}
	f.owner.Do(http.MethodGet, p+"/notifications", nil).MustStatus(t, 200).JSON(t, &list)
	n = list["items"].([]any)[0].(map[string]any)
	if n["title"] != "Push service worker missing on help.x22.test" ||
		n["summary"] != "https://help.x22.test/rc-push-sw.js answered 404 · notifications to 2 subscribers are held" {
		t.Fatalf("English notification: %v", n)
	}

	// A reader's expired link, in their browser's language.
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, env.Server.URL+"/t/p/nope", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	page, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusNotFound || string(page) != "Liên kết của thông báo này không còn hiệu lực.\n" {
		t.Fatalf("expired link: %d %q", resp.StatusCode, page)
	}
}
