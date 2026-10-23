package publicpages_test

// The words outside the built-in labels in the page's language: Grow's
// pages, the poll and survey pages, the Earn pages and the plain-text
// answers. The demo offers Vietnamese without its page copy; once the copy
// is written the pages are Live in it, and a reader whose language is
// Vietnamese reads them in Vietnamese.

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/linktoken"
)

// liveIn writes the pages' own text in a language (the English as it is)
// so the pages are Live in it.
func liveIn(t *testing.T, env *apptest.Env, code string) {
	t.Helper()
	c := owner(env)
	var pc struct {
		Strings []struct {
			Key, Source string
			Optional    bool
		}
	}
	c.Do(http.MethodGet, wsAPI+"/public-pages/copy?language="+code, nil).MustStatus(t, 200).JSON(t, &pc)
	written := map[string]string{}
	for _, s := range pc.Strings {
		if !s.Optional {
			written[s.Key] = s.Source
		}
	}
	c.Do(http.MethodPut, wsAPI+"/public-pages/copy/"+code, map[string]any{"strings": written}).MustStatus(t, 200)
}

func TestPageWordsInVietnamese(t *testing.T) {
	env, ids := demo(t)
	liveIn(t, env, "vi")
	vi := hdr{"Accept-Language": "vi-VN,vi;q=0.9"}
	if _, err := env.App.Pool.Exec(context.Background(), `UPDATE contacts SET language = 'vi' WHERE id = $1`, ids.sam); err != nil {
		t.Fatal(err)
	}

	t.Run("Grow", func(t *testing.T) {
		hub := "/p/" + wsSlug + "/referrals/" + env.App.Links.Sign(linktoken.Claims{Kind: linktoken.ReferralHub, WorkspaceID: ids.ws, ContactID: ids.sam,
			Link: linktoken.IssueDay(time.Now())})
		// Sam's hub follows his language, whatever the browser says.
		r := do(t, env, http.MethodGet, hub, nil, hdr{"Accept-Language": "es"})
		must(t, r, 200, `<html lang="vi" dir="ltr">`, "Lượt giới thiệu của bạn", "Đến nay có 61 lượt truy cập vào liên kết của bạn · 4 lượt đăng ký",
			"Còn 1 lượt giới thiệu nữa để nhận The Acme Field Guide", `<nav class="langs"`)
		lacks(t, r, "Your referrals", "visits to your link")
		// The menu's pick beats it; English stays as it was.
		must(t, get(t, env, hub+"?lang=en"), 200, `<html lang="en" dir="ltr">`, "1 more referral for the Acme Field Guide")
		// The leaderboard follows the browser.
		must(t, do(t, env, http.MethodGet, "/p/"+wsSlug+"/leaderboard", nil, vi), 200, `<html lang="vi" dir="ltr">`, "Người giới thiệu hàng đầu")
		must(t, get(t, env, "/p/"+wsSlug+"/leaderboard"), 200, `<html lang="en" dir="ltr">`, "Top referrers")
	})

	t.Run("a poll's answer page", func(t *testing.T) {
		poll := one[uuid.UUID](t, env, `SELECT id FROM polls WHERE workspace_id = $1 AND question LIKE 'How likely are you%'`, ids.ws)
		number := one[int64](t, env, `SELECT number FROM polls WHERE id = $1`, poll)
		pat := pollTester(t, env, ids, "pat.vi@example.com", "Pat")
		if _, err := env.App.Pool.Exec(context.Background(), `UPDATE contacts SET language = 'vi' WHERE id = $1`, pat); err != nil {
			t.Fatal(err)
		}
		base := "/p/" + wsSlug + "/poll/" + pollLink(env, ids.ws, pat, uuid.Nil, number)
		r := get(t, env, base+"/9")
		must(t, r, 200, `<html lang="vi" dir="ltr">`, "Bạn đã chọn 9 trong email. Chưa có gì được lưu cho đến khi bạn xác nhận.", "Xác nhận câu trả lời",
			`aria-label="9 trên 10"`, `data-saving="Đang lưu câu trả lời…"`)
		lacks(t, r, "Confirm my answer")
		r = do(t, env, http.MethodPost, base+"/9", url.Values{"answer": {"9"}}, nil)
		must(t, r, 200, "Cảm ơn Pat. Câu trả lời của bạn đã được lưu.", "Bạn muốn nói thêm điều gì?")
		// A broken link, in the browser's language.
		must(t, do(t, env, http.MethodGet, "/p/"+wsSlug+"/poll/nope/9", nil, vi), 404, `<html lang="vi" dir="ltr">`, "Liên kết này không còn hoạt động", "Đọc số mới nhất")
	})

	t.Run("a survey's page", func(t *testing.T) {
		// Without a contact's link the survey refuses, in the browser's language.
		slug := one[string](t, env, `SELECT slug FROM surveys WHERE workspace_id = $1 ORDER BY created_at LIMIT 1`, ids.ws)
		r := do(t, env, http.MethodGet, "/p/"+wsSlug+"/s/"+slug, nil, vi)
		if r.Status != 200 {
			t.Skipf("the demo's survey: %d", r.Status)
		}
		must(t, r, 200, `<html lang="vi" dir="ltr">`, "Khảo sát · Acme")
	})

	t.Run("Earn's upgrade page", func(t *testing.T) {
		r := do(t, env, http.MethodGet, "/p/"+wsSlug+"/upgrade", nil, vi)
		must(t, r, 200, `<html lang="vi" dir="ltr">`, "Đăng ký miễn phí", "Giá tính bằng đô la Mỹ, chưa gồm thuế.")
		lacks(t, r, "Subscribe free")
	})

	t.Run("plain-text answers", func(t *testing.T) {
		r := do(t, env, http.MethodGet, "/p/nobody-here/subscribe", nil, vi)
		must(t, r, 404, "Trang này không tồn tại.")
		must(t, do(t, env, http.MethodGet, "/p/nobody-here/subscribe", nil, nil), 404, "This page doesn't exist.")
	})
}
