package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/httprate"
)

// RateLimits protects the API from abuse: sign-in endpoints are limited per
// client IP (on top of the per-email lockout), and every API request is
// limited per API key (or per IP for browser sessions). Limits answer 429
// with Retry-After and an RFC 9457 body.
type RateLimits struct {
	AuthPerMinute int // per IP on /api/v1/auth/*
	APIPerMinute  int // per API key, or per IP without one
}

func (rl RateLimits) Middleware() func(http.Handler) http.Handler {
	auth := httprate.LimitBy(rl.AuthPerMinute, time.Minute, keyByIP,
		httprate.WithLimitHandler(limited("Too many sign-in attempts from your network. Wait a minute and try again.")),
	)
	api := httprate.LimitBy(rl.APIPerMinute, time.Minute, keyByCredential,
		httprate.WithLimitHandler(limited("Rate limit reached. Requests resume within a minute; batch endpoints accept up to 1,000 items per call.")),
	)
	return func(next http.Handler) http.Handler {
		authed := auth(next)
		limitedAPI := api(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.HasPrefix(r.URL.Path, "/api/v1/auth/"):
				authed.ServeHTTP(w, r)
			case strings.HasPrefix(r.URL.Path, "/api/"):
				limitedAPI.ServeHTTP(w, r)
			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}

// keyByCredential buckets requests by API key (hashed) when one is sent,
// otherwise by client IP.
func keyByCredential(r *http.Request) (string, error) {
	if h := r.Header.Get("Authorization"); len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		sum := sha256.Sum256([]byte(strings.TrimSpace(h[7:])))
		return "key:" + hex.EncodeToString(sum[:8]), nil
	}
	ip, err := keyByIP(r)
	return "ip:" + ip, err
}

// keyByIP keys on the client address. RealIP, earlier in the chain, has
// already replaced RemoteAddr with the client's address when the request came
// through a trusted proxy, so a forwarded header is never trusted here.
func keyByIP(r *http.Request) (string, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return httprate.CanonicalizeIP(host), nil
}

func limited(detail string) http.HandlerFunc {
	body, _ := json.Marshal(map[string]any{"status": http.StatusTooManyRequests, "title": "Too Many Requests", "detail": detail, "code": "rate_limited"})
	return func(w http.ResponseWriter, r *http.Request) {
		if w.Header().Get("Retry-After") == "" {
			w.Header().Set("Retry-After", strconv.Itoa(60))
		}
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write(body)
	}
}
