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
		// Webhooks: SMS adapters name themselves before the token.
		"/hooks/sms/twilio/whtok":   "/hooks/sms/twilio/{token}",
		"/hooks/stripe/whtok":       "/hooks/stripe/{token}",
		"/hooks/shopify/whtok":      "/hooks/shopify/{token}",
		"/hooks/woocommerce/whtok":  "/hooks/woocommerce/{token}",
		"/hooks/whatsapp/acme-news": "/hooks/whatsapp/{token}",
		// WhatsApp's callback URL names the workspace before the token.
		"/hooks/whatsapp/acme-news/whtok": "/hooks/whatsapp/acme-news/{token}",
		// Push notification clicks and SMS short links, on the instance's
		// host and at the root of a short-link domain.
		"/t/p/AbC.def":   "/t/p/{token}",
		"/t/p/AbC.def/2": "/t/p/{token}/2",
		"/t/s/Xk7Pq2":    "/t/s/{token}",
		"/Xk7Pq2":        "/{token}",
		"/xk7pq2":        "/{token}",
		"/readyz":        "/readyz",
		"/healthz":       "/healthz",
		"/sign-in":       "/sign-in",
		"/setup":         "/setup",
		"/Xk7Pq2/x":      "/Xk7Pq2/x",
		// The rest of the links with a token in them.
		"/p/acme-newsletter/l/tok":                     "/p/acme-newsletter/l/{token}",
		"/p/acme-newsletter/cancel/tok":                "/p/acme-newsletter/cancel/{token}",
		"/p/acme-newsletter/referrals/tok/address/rw1": "/p/acme-newsletter/referrals/{token}/address/rw1",
		"/billing/tok":                                 "/billing/{token}",
		"/rewards/tok":                                 "/rewards/{token}",
		"/recommends/tok":                              "/recommends/{token}",
		"/api/v1/invitations/tok/request":              "/api/v1/invitations/{token}/request",
		"/api/v1/auth/password-reset":                  "/api/v1/auth/password-reset",
	} {
		if got := RedactPath(in); got != want {
			t.Errorf("RedactPath(%q) = %q, want %q", in, got, want)
		}
	}
}
