package server

import "testing"

func TestRedactPath(t *testing.T) {
	for in, want := range map[string]string{
		"/t/o/AbC.def":                                   "/t/o/{token}",
		"/t/c/AbC.def":                                   "/t/c/{token}",
		"/t/h/AbC.def":                                   "/t/h/{token}",
		"/p/acme-newsletter/unsubscribe/tok":             "/p/acme-newsletter/unsubscribe/{token}",
		"/p/acme-newsletter/preferences/tok":             "/p/acme-newsletter/preferences/{token}",
		"/p/acme-newsletter/confirm/tok":                 "/p/acme-newsletter/confirm/{token}",
		"/p/acme-newsletter/view/tok":                    "/p/acme-newsletter/view/{token}",
		"/r/8f2c1a9d":                                    "/r/{token}",
		"/hooks/postmark/whtok":                          "/hooks/postmark/{token}",
		"/invite/tok":                                    "/invite/{token}",
		"/reset-password/tok":                            "/reset-password/{token}",
		"/api/v1/auth/password-reset/tok":                "/api/v1/auth/password-reset/{token}",
		"/api/v1/invitations/tok/accept":                 "/api/v1/invitations/{token}/accept",
		"/api/v1/shared-reports/tok":                     "/api/v1/shared-reports/{token}",
		"/review/tok":                                    "/review/{token}",
		"/api/v1/review/tok":                             "/api/v1/review/{token}",
		"/api/v1/review/tok/comments":                    "/api/v1/review/{token}/comments",
		"/p/acme-newsletter/archive/the-case-for-boring": "/p/acme-newsletter/archive/the-case-for-boring",
		"/api/v1/workspaces/acme-newsletter/campaigns":   "/api/v1/workspaces/acme-newsletter/campaigns",
		"/reset-password":                                "/reset-password",
		// A workspace's custom domain serves the email links at the root.
		"/unsubscribe/tok":             "/unsubscribe/{token}",
		"/preferences/tok":             "/preferences/{token}",
		"/confirm/tok":                 "/confirm/{token}",
		"/view/tok":                    "/view/{token}",
		"/preferences":                 "/preferences",
		"/archive/the-case-for-boring": "/archive/the-case-for-boring",
		// Grow: the hub, downloads, the recommendations page, partner codes.
		"/p/acme-newsletter/referrals/tok":  "/p/acme-newsletter/referrals/{token}",
		"/p/acme-newsletter/rewards/tok":    "/p/acme-newsletter/rewards/{token}",
		"/p/acme-newsletter/recommends/tok": "/p/acme-newsletter/recommends/{token}",
		"/referrals/tok":                    "/referrals/{token}",
		"/p/H3RT-6QWN-8ZKB":                 "/p/{token}",
		"/p/acme-newsletter/join/K7Q2MX":    "/p/acme-newsletter/join/K7Q2MX",
		"/p/acme-newsletter":                "/p/acme-newsletter",
		// Sign-in links, downloads, billing and cancel links, gifts, polls.
		"/p/acme-newsletter/signin/link/tok":   "/p/acme-newsletter/signin/link/{token}",
		"/signin/link/tok":                     "/signin/link/{token}",
		"/p/acme-newsletter/d/tok/guide.pdf":   "/p/acme-newsletter/d/{token}/guide.pdf",
		"/d/tok/new-link":                      "/d/{token}/new-link",
		"/l/tok":                               "/l/{token}",
		"/p/acme-newsletter/billing/tok":       "/p/acme-newsletter/billing/{token}",
		"/cancel/tok":                          "/cancel/{token}",
		"/p/acme-newsletter/gift/redeem/tok":   "/p/acme-newsletter/gift/redeem/{token}",
		"/gift/redeem/tok":                     "/gift/redeem/{token}",
		"/p/acme-newsletter/poll/tok/2":        "/p/acme-newsletter/poll/{token}/2",
		"/poll/tok/2/comment":                  "/poll/{token}/2/comment",
		"/signin":                              "/signin",
		"/p/acme-newsletter/products/the-book": "/p/acme-newsletter/products/the-book",
		"/gift":                                "/gift",
	} {
		if got := RedactPath(in); got != want {
			t.Errorf("RedactPath(%q) = %q, want %q", in, got, want)
		}
	}
}
