package app_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/server"
)

// A signed-in browser is limited per session, not per address: two people
// in one office each get the full budget (review 2026-11 OP-6).
func TestAPILimitsEachSession(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	env.App.Auth.SessionLimit = server.NewSessionLimiter(3).Allow
	env.App.Auth.SessionLimitPerMinute = 3
	linh, sofia := env.Client(), env.Client()
	linh.SignInDemo("linh@acme.io")
	sofia.SignInDemo("sofia@acme.io")
	for range 3 {
		linh.Do("GET", "/workspaces/acme-newsletter/lists", nil).MustStatus(t, 200)
	}
	r := linh.Do("GET", "/workspaces/acme-newsletter/lists", nil).MustStatus(t, http.StatusTooManyRequests)
	p := r.Problem(t)
	if p["code"] != "rate_limited" || r.Header.Get("Retry-After") == "" || !strings.Contains(p["detail"].(string), "at most 3 a minute") {
		t.Fatalf("limited: %+v %v", p, r.Header)
	}
	// Another session from the same address isn't.
	sofia.Do("GET", "/workspaces/acme-newsletter/lists", nil).MustStatus(t, 200)
}
