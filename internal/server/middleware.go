// Package server builds the HTTP stack: middleware, security headers, the
// embedded single-page app and graceful shutdown.
package server

import (
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/hostrepute/reputecast/internal/audit"
)

// RealIP sets r.RemoteAddr from X-Forwarded-For / X-Real-IP, but only when the
// direct peer is a trusted proxy. chi's RealIP trusts any client, which would
// let anyone spoof their address in the audit log and rate limits.
func RealIP(trusted []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(trusted) > 0 {
				if peer, ok := parseAddr(r.RemoteAddr); ok && isTrusted(peer, trusted) {
					if ip, ok := forwardedClient(r, trusted); ok {
						r.RemoteAddr = net.JoinHostPort(ip.String(), "0")
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// forwardedClient walks X-Forwarded-For right to left, skipping trusted hops.
// Every line of the header counts, in order: a proxy may add its hop as a
// line of its own (HAProxy's option forwardfor) instead of extending the
// line the client sent, and the client's lines, which come first, must
// never win over the hops our proxies added.
func forwardedClient(r *http.Request, trusted []netip.Prefix) (netip.Addr, bool) {
	if xff := strings.Join(r.Header.Values("X-Forwarded-For"), ","); xff != "" {
		hops := strings.Split(xff, ",")
		for i := len(hops) - 1; i >= 0; i-- {
			ip, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
			if err != nil {
				return netip.Addr{}, false
			}
			if !isTrusted(ip, trusted) || i == 0 {
				return ip, true
			}
		}
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		if ip, err := netip.ParseAddr(strings.TrimSpace(xr)); err == nil {
			return ip, true
		}
	}
	return netip.Addr{}, false
}

func isTrusted(ip netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(ip.Unmap()) {
			return true
		}
	}
	return false
}

func parseAddr(hostport string) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}
	return ip.Unmap(), true
}

// ClientIP returns the (already proxy-resolved) client address of r.
func ClientIP(r *http.Request) *netip.Addr {
	ip, ok := parseAddr(r.RemoteAddr)
	if !ok {
		return nil
	}
	return &ip
}

// AuditContext stores the client IP, user agent and request id in the
// request context so audit entries can include them.
func AuditContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := audit.WithRequest(r.Context(), audit.Request{
			IP:        ClientIP(r),
			UserAgent: truncate(r.UserAgent(), 512),
			RequestID: middleware.GetReqID(r.Context()),
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// SecurityHeaders sets defensive headers on every response. Public pages and
// the embeddable form endpoints relax frame-ancestors themselves.
func SecurityHeaders(httpsOnly bool) func(http.Handler) http.Handler {
	csp := strings.Join([]string{
		"default-src 'self'",
		"script-src 'self'",
		// React sets styles through the CSSOM, but the email composer and the
		// flow canvas inject <style> elements, so inline styles are allowed.
		"style-src 'self' 'unsafe-inline'",
		// Email content references images on any https host.
		"img-src 'self' data: blob: https:",
		"font-src 'self' data:",
		"connect-src 'self'",
		"frame-src 'self' blob:",
		"frame-ancestors 'none'",
		"form-action 'self'",
		"base-uri 'self'",
		"object-src 'none'",
	}, "; ")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Content-Security-Policy", csp)
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
			if httpsOnly {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// secretSegments match paths whose last part is a bearer secret: tracking
// and email links (opens, clicks and the hidden link), shared reports,
// provider webhooks, invitations, password resets and composer review
// links (API and app routes).
var secretSegments = []*regexp.Regexp{
	regexp.MustCompile(`^(/t/[och]/)[^/]+`),
	regexp.MustCompile(`^(/p/[^/]+/(?:unsubscribe|preferences|confirm|view)/)[^/]+`),
	regexp.MustCompile(`^(/r/)[^/]+`),
	regexp.MustCompile(`^(/hooks/[^/]+/)[^/]+`),
	regexp.MustCompile(`^(/(?:invite|reset-password|review)/)[^/]+`),
	regexp.MustCompile(`^(/api/v1/(?:auth/password-reset|invitations|shared-reports|review)/)[^/]+`),
	// The same email links on a workspace's custom domain, at the root
	// (news.acme.io/unsubscribe/{token}).
	regexp.MustCompile(`^(/(?:unsubscribe|preferences|confirm|view)/)[^/]+`),
	// Grow: a referrer's hub, reward downloads and the personal
	// recommendations page, and partner codes (/p/H3RT-6QWN-8ZKB).
	regexp.MustCompile(`^(/p/[^/]+/(?:referrals|rewards|recommends)/)[^/]+`),
	regexp.MustCompile(`^(/(?:referrals|rewards|recommends)/)[^/]+`),
	regexp.MustCompile(`^(/p/)[2-9A-Za-z]{4}-[2-9A-Za-z]{4}-[2-9A-Za-z]{4}$`),
	// The website's sign-in links, a buyer's downloads and lead magnets, a
	// paid member's billing and cancel links, gift codes and poll answers,
	// under /p/{workspace}/ and at a custom domain's root.
	regexp.MustCompile(`^(/p/[^/]+/(?:signin/link|d|l|billing|cancel|gift/redeem|poll)/)[^/]+`),
	regexp.MustCompile(`^(/(?:signin/link|d|l|billing|cancel|gift/redeem|poll)/)[^/]+`),
}

// RedactPath replaces the secret part of a path with {token}, so logs never
// hold a working link.
func RedactPath(p string) string {
	for _, re := range secretSegments {
		if m := re.FindStringSubmatchIndex(p); m != nil {
			return p[:m[3]] + "{token}" + p[m[1]:]
		}
	}
	return p
}

// RequestLogger logs one line per request at info (errors at warn/error).
func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			level := slog.LevelInfo
			switch {
			case status >= 500:
				level = slog.LevelError
			case status >= 400:
				level = slog.LevelWarn
			case strings.HasPrefix(r.URL.Path, "/assets/") || r.URL.Path == "/healthz" || r.URL.Path == "/readyz":
				level = slog.LevelDebug
			}
			log.Log(r.Context(), level, "http",
				"method", r.Method,
				"path", RedactPath(r.URL.Path),
				"status", status,
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
