package digests_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// TestDigestTextInVietnamese checks that a member who reads ReputeCast in
// Vietnamese gets a digest's server text in Vietnamese: a problem made for
// the request, why the feed failed (kept when an English request read it),
// and a notification, written in its recipient's language.
func TestDigestTextInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Sending = &fakeStarter{}
	n := counter{t, env}
	feed := newFeedServer(t, basePosts(time.Now())...)
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	d := setUp(t, linh, n, feed, "Bilingual roundup")
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	// A problem made for the request.
	bad := settings("Bilingual roundup", feed.srv.URL+"/feed/", n.listID("Product Updates"), 2)
	bad["schedule"] = map[string]any{"frequency": "items", "count": 3, "weekday": 5, "time": "08:00", "windowFrom": "18:00", "windowUntil": "08:00", "timezone": "America/New_York"}
	if body := string(vi.Do("PUT", ws+"/digests/"+d.ID, bad).MustStatus(t, 422).Body); !strings.Contains(body, "Khung giờ phải kết thúc sau khi bắt đầu: hãy chọn giờ kết thúc muộn hơn 18:00.") {
		t.Fatalf("Vietnamese problem: %s", body)
	}
	if body := string(linh.Do("PUT", ws+"/digests/"+d.ID, bad).MustStatus(t, 422).Body); !strings.Contains(body, "The window ends after it starts: pick an end time later than 18:00.") {
		t.Fatalf("English problem: %s", body)
	}

	// Why the feed failed, read in an English request, reads in each
	// reader's language.
	feed.setFail(true)
	linh.Do("POST", ws+"/digests/"+d.ID+"/read", map[string]any{}).MustStatus(t, 200)
	var en, vn struct {
		Feed struct {
			Error *struct{ Kind, Detail string }
		}
	}
	linh.Do("GET", ws+"/digests/"+d.ID, nil).MustStatus(t, 200).JSON(t, &en)
	vi.Do("GET", ws+"/digests/"+d.ID, nil).MustStatus(t, 200).JSON(t, &vn)
	if en.Feed.Error == nil || !strings.HasSuffix(en.Feed.Error.Detail, " answered 503 Service Unavailable") {
		t.Fatalf("English feed error: %+v", en.Feed.Error)
	}
	if vn.Feed.Error == nil || !strings.HasSuffix(vn.Feed.Error.Detail, " trả về 503 Service Unavailable") {
		t.Fatalf("Vietnamese feed error: %+v", vn.Feed.Error)
	}

	// Linh reads ReputeCast in Vietnamese: the feed failing at preparation
	// tells her in Vietnamese.
	n.exec(`UPDATE users SET language = 'vi' WHERE email = 'linh@acme.io'`)
	if err := env.App.Digests().Prepare(context.Background(), uuid.MustParse(d.Next.ID)); err != nil {
		t.Fatal(err)
	}
	var title, body string
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT n.title, n.body FROM notifications n JOIN users u ON u.id = n.user_id
		WHERE u.email = 'linh@acme.io' AND n.kind = 'digest_feed_failing' ORDER BY n.created_at DESC LIMIT 1`).Scan(&title, &body); err != nil {
		t.Fatal(err)
	}
	if title != "Nguồn cấp của bản tin định kỳ đang lỗi" || !strings.HasPrefix(body, "Nguồn cấp của Bilingual roundup bị lỗi lúc chuẩn bị: ") || !strings.Contains(body, " trả về 503 Service Unavailable") {
		t.Fatalf("notification: %q %q", title, body)
	}
}
