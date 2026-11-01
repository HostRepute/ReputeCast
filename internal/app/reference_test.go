package app_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/version"
)

// TestAPIReference: the instance serves its own API reference, of the
// version it runs, with every operation and the permission it needs, and
// nothing loaded from elsewhere (review 2026-11 IN-4).
func TestAPIReference(t *testing.T) {
	env := apptest.New(t)
	res, err := http.Get(env.Server.URL + "/api/reference")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	page := string(b)
	if res.StatusCode != 200 || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("status %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	for _, want := range []string{"Version " + version.Get().Version, "/workspaces/{workspace}/transactional/messages/{messageId}/cancel", "transactional:send", `href="/api/openapi.json"`} {
		if !strings.Contains(page, want) {
			t.Errorf("the reference lacks %q", want)
		}
	}
	if strings.Contains(page, "<script") || strings.Contains(page, "https://") && strings.Contains(page, "src=") {
		t.Error("the reference loads nothing from elsewhere")
	}
	if csp := res.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'none'") {
		t.Errorf("csp: %q", csp)
	}
}
