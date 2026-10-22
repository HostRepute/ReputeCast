package templates_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/mjml"
	"github.com/hostrepute/reputecast/internal/templates"
)

// A starter opened in Vietnamese is written for the email's readers in
// Vietnamese: its sample texts, the first-name fallback, the footer's links
// and the placeholder address. It still compiles cleanly.
func TestStartersInVietnamese(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	for _, key := range []string{"blank", "weekly-digest", "product-update", "plain-text", "announcement", "product-grid", "blog-digest"} {
		src, ok := templates.StarterMJMLIn(vi, key, "")
		if !ok {
			t.Fatalf("starter %s missing", key)
		}
		if strings.Contains(src, "%%") || strings.Contains(src, "Unsubscribe") || strings.Contains(src, `"there"`) {
			t.Errorf("starter %s keeps English or a placeholder:\n%s", key, src)
		}
		if !strings.Contains(src, "Hủy đăng ký") || !strings.Contains(src, templates.DefaultAddressIn(vi)) {
			t.Errorf("starter %s footer:\n%s", key, src)
		}
		if res := mjml.Compile(src); len(res.Errors) > 0 {
			t.Errorf("starter %s: %v", key, res.Errors)
		}
	}
	src, _ := templates.StarterMJMLIn(vi, "plain-text", "")
	if !strings.Contains(src, `Chào {{ first_name | default: "bạn" }},`) {
		t.Errorf("greeting: %s", src)
	}
	src, _ = templates.StarterMJMLIn(vi, "blog-digest", "")
	if !strings.Contains(src, `"buttonText":"Đọc bài viết"`) {
		t.Errorf("feed button: %s", src)
	}
	src, _ = templates.StarterMJMLIn(vi, "product-grid", "")
	if !strings.Contains(src, `alt="Mô tả sản phẩm thứ tư"`) || !strings.Contains(src, "$49 · một dòng giới thiệu") {
		t.Errorf("product cards: %s", src)
	}
}

// The gallery names its starters in the member's language, and saving a
// template explains what's wrong in it; API clients get English.
func TestGalleryInVietnamese(t *testing.T) {
	env := apptest.New(t)
	linh := env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	env.CreateWorkspace("Acme Newsletter", linh.ID)
	c := env.Client()
	c.SignInDemo("linh@acme.io")
	c.Header = http.Header{"Accept-Language": {"vi"}}

	var g struct{ Starters []summary }
	c.Do("GET", ws+"/templates", nil).MustStatus(t, 200).JSON(t, &g)
	if len(g.Starters) == 0 || g.Starters[1].Name != "Bản tin hằng tuần" || g.Starters[1].Description != "Hero, hai thẻ bài viết, chân thư" {
		t.Fatalf("starters in Vietnamese: %+v", g.Starters)
	}
	var d struct{ Name, MJML string }
	c.Do("GET", ws+"/templates/starter:product-update", nil).MustStatus(t, 200).JSON(t, &d)
	if d.Name != "Cập nhật sản phẩm" || !strings.Contains(d.MJML, "Xem có gì mới") {
		t.Fatalf("starter in Vietnamese: %s", d.Name)
	}
	p := c.Do("POST", ws+"/templates", map[string]any{"name": " ", "mjml": "<mjml><mj-body></mj-body></mjml>"}).MustStatus(t, 422).Problem(t)
	if p["detail"] != "Hãy đặt tên cho mẫu." {
		t.Fatalf("no name, in Vietnamese: %v", p)
	}
	c.Do("POST", ws+"/templates", map[string]any{"name": "Monthly", "mjml": "<mjml><mj-body></mj-body></mjml>"}).MustStatus(t, 201)
	p = c.Do("POST", ws+"/templates", map[string]any{"name": "Monthly", "mjml": "<mjml><mj-body></mj-body></mjml>"}).MustStatus(t, 422).Problem(t)
	if p["detail"] != "Đã có mẫu tên “Monthly”. Hãy chọn tên khác." {
		t.Fatalf("name taken, in Vietnamese: %v", p)
	}
	p = c.Do("GET", ws+"/templates/starter:nope", nil).MustStatus(t, 404).Problem(t)
	if p["detail"] != "Không tìm thấy mẫu." {
		t.Fatalf("not found, in Vietnamese: %v", p)
	}

	en := env.Client()
	en.SignInDemo("linh@acme.io")
	en.Do("GET", ws+"/templates", nil).MustStatus(t, 200).JSON(t, &g)
	if g.Starters[1].Name != "Weekly digest" || g.Starters[1].Description != "Hero, two article cards, footer" {
		t.Fatalf("starters in English: %+v", g.Starters)
	}
	p = en.Do("POST", ws+"/templates", map[string]any{"name": "Monthly", "mjml": "<mjml><mj-body></mj-body></mjml>"}).MustStatus(t, 422).Problem(t)
	if p["detail"] != "A template called “Monthly” already exists. Pick another name." {
		t.Fatalf("name taken, in English: %v", p)
	}
}
