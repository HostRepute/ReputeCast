package applang_test

import (
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// Picking a language the request can't have reads in the request's
// language. (Values outside the lists are refused by the schema first.)
func TestLanguageProblemsInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.CreateUser(apptest.UserOpts{Email: "linh@acme.io", Name: "Linh Nguyen"})
	c := env.Client()
	c.Header = http.Header{"Accept-Language": {"vi"}}
	c.SignIn("linh@acme.io", "")

	if p := c.Do("PUT", "/me/language", choose("en-XA", "auto")).MustStatus(t, 422).Problem(t); p["detail"] != "Ngôn ngữ giả chỉ có trong môi trường phát triển, hoặc khi máy chủ chạy với i18n.pseudo_locale bật." {
		t.Errorf("pseudo-locale: %v", p)
	}
	en := env.Client()
	en.SignIn("linh@acme.io", "")
	if p := en.Do("PUT", "/me/language", choose("en-XA", "auto")).MustStatus(t, 422).Problem(t); p["detail"] != "The pseudo-locale is only available in development, or when the server runs with i18n.pseudo_locale on." {
		t.Errorf("English: %v", p)
	}
}
