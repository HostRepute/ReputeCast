package ai_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// TestAITextInVietnamese checks that the assistant's problems follow the
// request's language, with amounts in the reader's region format.
func TestAITextInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	en := env.Client()
	en.SignInDemo("linh@acme.io")
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}

	low := map[string]any{"kind": "usd", "usd": 0.5}
	// The region format writes a no-break space before the currency.
	if body := strings.ReplaceAll(string(vi.Do(http.MethodPut, ws+"/ai/cap", low).MustStatus(t, 422).Body), " ", " "); !strings.Contains(body, "Hãy đặt hạn mức hằng tháng từ 1,00 US$ đến 100.000,00 US$, hoặc chọn không đặt hạn mức.") {
		t.Fatalf("Vietnamese problem: %s", body)
	}
	if body := string(en.Do(http.MethodPut, ws+"/ai/cap", low).MustStatus(t, 422).Body); !strings.Contains(body, "Set a monthly cap between $1.00 and $100,000.00, or choose no cap.") {
		t.Fatalf("English problem: %s", body)
	}
}
