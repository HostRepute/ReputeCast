package accessrequests_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
)

// The 403 screen's suggestion and the problems read in the member's
// language; the request's notification, sent while Aisha read Vietnamese,
// reads in each manager's language, English included.
func TestAccessRequestInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignInDemo("aisha@acme.io")

	var s suggestion
	vi.Do("GET", ws+"/access-requests/suggestion?permission=providers:manage", nil).MustStatus(t, 200).JSON(t, &s)
	if s.Action != "Quản lý nhà cung cấp" || s.RoleName != "Admin" {
		t.Errorf("suggestion: %+v", s)
	}
	if p := vi.Do("GET", ws+"/access-requests/suggestion?permission=nope:nope", nil).MustStatus(t, 422).Problem(t); p["detail"] != "\"nope:nope\" không phải là một quyền." {
		t.Errorf("unknown permission: %v", p)
	}
	vi.Do("POST", ws+"/access-requests", map[string]any{"permission": "providers:manage"}).MustStatus(t, 202)
	p := vi.Do("POST", ws+"/access-requests", map[string]any{"permission": "providers:manage"}).MustStatus(t, 409).Problem(t)
	if d, _ := p["detail"].(string); !strings.HasPrefix(d, "Bạn đã xin quyền truy cập Quản trị viên trong ngày qua; ") || !strings.HasSuffix(d, " sẽ thấy yêu cầu này. Hãy hỏi trực tiếp họ nếu việc gấp.") {
		t.Errorf("asked twice: %v", p)
	}

	var title, summary, body string
	var data []byte
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT n.title, n.summary, n.body, n.data FROM notifications n JOIN users u ON u.id = n.user_id
		WHERE n.kind = 'access_requested' AND u.email = 'linh@acme.io'`).Scan(&title, &summary, &body, &data); err != nil {
		t.Fatal(err)
	}
	if title != "Aisha Bello asks for Admin access" || body != "No note." {
		t.Errorf("English: %q · %q", title, body)
	}
	if vt, _, vb := notify.Localize(i18n.For(i18n.Vietnamese), title, summary, body, data); vt != "Aisha Bello xin quyền truy cập Quản trị viên" || vb != "Không có ghi chú." {
		t.Errorf("Vietnamese: %q · %q", vt, vb)
	}
}
