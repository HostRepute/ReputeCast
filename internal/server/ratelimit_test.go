package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitsAuthPerIP(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := RateLimits{AuthPerMinute: 2, APIPerMinute: 100}.Middleware()(ok)
	do := func(path, ip, auth string) int {
		r := httptest.NewRequest(http.MethodPost, path, nil)
		r.RemoteAddr = ip + ":1234"
		if auth != "" {
			r.Header.Set("Authorization", "Bearer "+auth)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	for i := 0; i < 2; i++ {
		if c := do("/api/v1/auth/password", "198.51.100.7", ""); c != http.StatusNoContent {
			t.Fatalf("request %d: %d", i, c)
		}
	}
	if c := do("/api/v1/auth/password", "198.51.100.7", ""); c != http.StatusTooManyRequests {
		t.Fatalf("third sign-in should be limited, got %d", c)
	}
	if c := do("/api/v1/auth/password", "198.51.100.8", ""); c != http.StatusNoContent {
		t.Fatalf("another IP is not limited, got %d", c)
	}
	if c := do("/assets/app.js", "198.51.100.7", ""); c != http.StatusNoContent {
		t.Fatalf("static files are not limited, got %d", c)
	}
}

func TestRateLimitsAPIPerKey(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	h := RateLimits{AuthPerMinute: 100, APIPerMinute: 1}.Middleware()(ok)
	do := func(key string) int {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/x/campaigns", nil)
		r.RemoteAddr = "203.0.113.9:1"
		r.Header.Set("Authorization", "Bearer "+key)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	if do("rc_live_a") != http.StatusOK || do("rc_live_a") != http.StatusTooManyRequests {
		t.Fatal("second call with the same key should be limited")
	}
	if do("rc_live_b") != http.StatusOK {
		t.Fatal("a different key has its own budget")
	}
}
