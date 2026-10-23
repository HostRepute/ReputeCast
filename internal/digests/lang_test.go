package digests_test

import (
	"context"
	"net/http"
	"regexp"
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

	// Who the issues go to, named in the reader's words.
	var enAud, viAud struct{ Audience struct{ Label string } }
	linh.Do("GET", ws+"/digests/"+d.ID, nil).MustStatus(t, 200).JSON(t, &enAud)
	vi.Do("GET", ws+"/digests/"+d.ID, nil).MustStatus(t, 200).JSON(t, &viAud)
	if enAud.Audience.Label != "Product Updates list" || viAud.Audience.Label != "Danh sách Product Updates" {
		t.Fatalf("audience: %q, %q", enAud.Audience.Label, viAud.Audience.Label)
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

// TestDigestEmailInItsLanguage checks that what a digest writes into its
// email follows the language the workspace writes in, whatever the reader
// reads the app in: a new digest's name, the starter's subject and
// preheader, and the posts' dates.
func TestDigestEmailInItsLanguage(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	n := counter{t, env}
	feed := newFeedServer(t, basePosts(time.Now())...)
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	host := strings.TrimPrefix(feed.srv.URL, "http://")
	type render struct{ Subject, Preheader, MJML, StarterSubject string }
	// A digest made from the type picker, with the latest posts in its first issue.
	create := func() (detail, render) {
		d := decodeDetail(t, linh.Do("POST", ws+"/digests", map[string]any{"source": "url", "feedUrl": feed.srv.URL + "/feed/"}).MustStatus(t, 201))
		linh.Do("PUT", ws+"/digests/"+d.ID, settings(d.Name, feed.srv.URL+"/feed/", n.listID("Product Updates"), 3)).MustStatus(t, 200)
		var r render
		linh.Do("POST", ws+"/digests/"+d.ID+"/render", map[string]any{"preview": "next"}).MustStatus(t, 200).JSON(t, &r)
		return d, r
	}
	englishDate := regexp.MustCompile(`(Mon|Tues|Wednes|Thurs|Fri|Satur|Sun)day \d{1,2} (January|February|March|April|May|June|July|August|September|October|November|December) \d{4}`)
	vietnameseDate := regexp.MustCompile(`(Thứ Hai|Thứ Ba|Thứ Tư|Thứ Năm|Thứ Sáu|Thứ Bảy|Chủ Nhật), \d{1,2} tháng \d{1,2}, \d{4}`)

	// The workspace writes in English: the starter as it always was.
	en, r := create()
	if en.Name != host+" digest" || r.Subject != "Feature flags at 2,000 teams: what broke first and 1 more from Test blog" || r.Preheader != "New on "+host+" · issue 1" ||
		!englishDate.MatchString(r.MJML) {
		t.Fatalf("English: %q %q %q", en.Name, r.Subject, r.Preheader)
	}

	// It writes in Vietnamese: so does the starter, for every reader.
	linh.Do("PUT", ws+"/languages/defaults", map[string]any{"default": "vi", "fallback": "en"}).MustStatus(t, 200)
	vi, r := create()
	if vi.Name != "Bản tin định kỳ "+host || r.Subject != "Feature flags at 2,000 teams: what broke first và 1 bài khác từ Test blog" || r.Preheader != "Bài mới trên "+host+" · số 1" ||
		!strings.Contains(r.StarterSubject, "{% if issue.items.size > 1 %} và {{ issue.items.size | minus: 1 }} bài khác{% endif %}") {
		t.Fatalf("Vietnamese: %q %q %q %q", vi.Name, r.Subject, r.Preheader, r.StarterSubject)
	}
	if !vietnameseDate.MatchString(r.MJML) || englishDate.MatchString(r.MJML) {
		t.Fatalf("Vietnamese dates: %s", r.MJML)
	}
}

// TestDigestHeldInVietnamese checks that a held issue's notification names
// the Send Readiness check that blocks it in its recipient's language, from
// the messages the run kept, while the audit keeps the check's English.
func TestDigestHeldInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Sending = &fakeStarter{}
	n := counter{t, env}
	ctx := context.Background()
	feed := newFeedServer(t, basePosts(time.Now())...)
	sofia := env.Client()
	sofia.SignInDemo("sofia@acme.io")
	d := decodeDetail(t, sofia.Do("POST", ws+"/digests", map[string]any{"name": "Held roundup", "source": "url", "feedUrl": feed.srv.URL + "/feed/"}).MustStatus(t, 201))
	sofia.Do("PUT", ws+"/digests/"+d.ID, settings("Held roundup", feed.srv.URL+"/feed/", n.listID("Product Updates"), 2)).MustStatus(t, 200)
	d = decodeDetail(t, sofia.Do("POST", ws+"/digests/"+d.ID+"/turn-on", map[string]any{}).MustStatus(t, 200))
	// The template loses its footer, so Send Readiness blocks the issue;
	// Sofia, its owner, reads ReputeCast in Vietnamese.
	n.exec(`UPDATE campaigns SET mjml = regexp_replace(mjml, '<!-- rc:footer -->.*?</mj-section>', '') WHERE id = $1`, uuid.MustParse(d.TemplateID))
	n.exec(`UPDATE users SET language = 'vi' WHERE email = 'sofia@acme.io'`)
	if err := env.App.Digests().Prepare(ctx, uuid.MustParse(d.Next.ID)); err != nil {
		t.Fatal(err)
	}
	var body, reason string
	if err := env.App.Pool.QueryRow(ctx, `SELECT n.body FROM notifications n JOIN users u ON u.id = n.user_id
		WHERE u.email = 'sofia@acme.io' AND n.kind = 'digest_issue_held' ORDER BY n.created_at DESC LIMIT 1`).Scan(&body); err != nil {
		t.Fatal(err)
	}
	if err := env.App.Pool.QueryRow(ctx, `SELECT reason FROM audit_events WHERE action = 'digest.issue_held' AND target_id = $1`, d.ID).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	english := strings.ToLower(strings.TrimSuffix(reason, "."))
	if reason == "" || !strings.Contains(body, " bị tạm giữ: ") || strings.Contains(strings.ToLower(body), english) {
		t.Fatalf("held: %q (the audit's reason: %q)", body, reason)
	}
}
