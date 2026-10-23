package suppressions_test

import (
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
)

// The suppression sync card reads in the request's language: which list a
// provider syncs with, why a provider without a suppression API needs
// none, an import's row. Provider names stay as they are.
func TestSyncCardInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	en := env.Client()
	en.SignInDemo("linh@acme.io")
	type cardT struct {
		Rows []struct{ Provider, Scope, Direction, Detail string }
	}
	var vc, ec cardT
	vi.Do("GET", "/workspaces/acme-newsletter/suppression-sync", nil).MustStatus(t, 200).JSON(t, &vc)
	en.Do("GET", "/workspaces/acme-newsletter/suppression-sync", nil).MustStatus(t, 200).JSON(t, &ec)
	if len(vc.Rows) == 0 || len(vc.Rows) != len(ec.Rows) {
		t.Fatalf("rows: %+v", vc.Rows)
	}
	words := map[string]string{"account-level list": "danh sách cấp tài khoản", "per stream": "theo từng luồng", "enforced before sending": "áp dụng trước khi gửi"}
	seen := 0
	for i, r := range vc.Rows {
		e := ec.Rows[i]
		if want, ok := words[e.Scope]; ok {
			seen++
			if r.Scope != want {
				t.Fatalf("scope %q: %q", e.Scope, r.Scope)
			}
		}
		if e.Direction == "local" && r.Detail != "ReputeCast kiểm tra danh sách dùng chung trước khi chuyển thư cho "+e.Provider+"." {
			t.Fatalf("local detail: %q", r.Detail)
		}
		if e.Direction == "import" && (r.Provider == e.Provider || r.Scope == "") {
			t.Fatalf("import row: %+v / %+v", r, e)
		}
	}
	if seen == 0 {
		t.Fatalf("no scope words: %+v", ec.Rows)
	}
}
