package placement_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// TestPlacementTextInVietnamese checks that a problem made for the request
// follows its language, and names the provider's password as the reader
// says it.
func TestPlacementTextInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	base := "/workspaces/acme-newsletter/placement"
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	blank := map[string]any{"provider": "icloud", "email": "seed@seeds.test", "password": "   "}
	if body := string(vi.Do("POST", base+"/seeds", blank).MustStatus(t, 422).Body); !strings.Contains(body, "Hãy dán mật khẩu dành riêng cho ứng dụng.") {
		t.Fatalf("Vietnamese problem: %s", body)
	}
	if body := string(linh.Do("POST", base+"/seeds", blank).MustStatus(t, 422).Body); !strings.Contains(body, "Paste the app-specific password.") {
		t.Fatalf("English problem: %s", body)
	}
}
