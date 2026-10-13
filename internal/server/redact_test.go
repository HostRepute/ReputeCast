package server

import "testing"

func TestRedactPath(t *testing.T) {
	for in, want := range map[string]string{
		"/t/o/AbC.def":                                   "/t/o/{token}",
		"/t/c/AbC.def":                                   "/t/c/{token}",
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
	} {
		if got := RedactPath(in); got != want {
			t.Errorf("RedactPath(%q) = %q, want %q", in, got, want)
		}
	}
}
