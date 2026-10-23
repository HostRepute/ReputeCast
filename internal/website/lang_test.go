package website_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/website"
)

// The website's server text reads in the reader's language: problems in
// the member's, and ReputeCast's words the workspace hasn't rewritten (the
// paywall card) in the language of the page that shows them.
func TestWebsiteInVietnamese(t *testing.T) {
	env, c := demoSite(t)
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	if p := vi.Do("GET", ws+"/website/posts/"+uuid.NewString(), nil).MustStatus(t, 404).Problem(t); p["detail"] != "Không tìm thấy bài viết." {
		t.Fatalf("not found (vi): %v", p)
	}
	if p := c.Do("GET", ws+"/website/posts/"+uuid.NewString(), nil).MustStatus(t, 404).Problem(t); p["detail"] != "Post not found." {
		t.Fatalf("not found (en): %v", p)
	}

	// The paywall card's words before anyone writes them, in each language.
	if p := website.DefaultPaywall(); p.Heading != "This post is for paid members" || p.Upgrade != "Upgrade" || p.Signup != "Sign up free" ||
		p.Free != "Not ready? Sign up free and read 1 paid post a month." {
		t.Fatalf("English paywall: %+v", p)
	}
	if p := website.DefaultPaywallIn(i18n.For(i18n.Vietnamese)); p.Heading != "Bài viết này dành cho hội viên trả phí" || p.Upgrade != "Nâng cấp" {
		t.Fatalf("Vietnamese paywall: %+v", p)
	}
}
