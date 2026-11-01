package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

// Signed-in browsers behind one address share a cap ten times a session's
// limit, not one session's (review 2026-11 OP-6).
func TestRateLimitsBrowsersShareAHigherCap(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	h := RateLimits{AuthPerMinute: 100, APIPerMinute: 2, IPPerMinute: 5, SessionCookie: "rc_session"}.Middleware()(ok)
	do := func(cookie string) int {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/x/campaigns", nil)
		r.RemoteAddr = "203.0.113.10:1"
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: "rc_session", Value: cookie})
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	// Five browsers' requests from the office pass this layer; the sixth hits the cap.
	for i := range 5 {
		if c := do("session-" + string(rune('a'+i))); c != http.StatusOK {
			t.Fatalf("browser request %d: %d", i, c)
		}
	}
	if c := do("session-f"); c != http.StatusTooManyRequests {
		t.Fatalf("over the address's cap: %d", c)
	}
	// Anonymous calls from the same address have their own, smaller budget.
	if do("") != http.StatusOK || do("") != http.StatusOK || do("") != http.StatusTooManyRequests {
		t.Fatal("anonymous calls keep the per-address limit")
	}
}

func TestSessionLimiter(t *testing.T) {
	now := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	l := NewSessionLimiter(3)
	l.now = func() time.Time { return now }
	for i := range 3 {
		if ok, _ := l.Allow("s1"); !ok {
			t.Fatalf("request %d refused", i)
		}
	}
	if ok, wait := l.Allow("s1"); ok || wait != time.Minute {
		t.Fatalf("fourth: %v %v", ok, wait)
	}
	if ok, _ := l.Allow("s2"); !ok {
		t.Fatal("another session has its own budget")
	}
	// Halfway through the next minute, half of the last minute still counts.
	now = now.Add(90 * time.Second)
	if ok, _ := l.Allow("s1"); !ok {
		t.Fatal("1.5 of 3 used")
	}
	if ok, _ := l.Allow("s1"); ok {
		t.Fatal("2.5 + 1 is over 3")
	}
	now = now.Add(3 * time.Minute)
	if ok, _ := l.Allow("s1"); !ok {
		t.Fatal("a quiet session starts over")
	}
}
