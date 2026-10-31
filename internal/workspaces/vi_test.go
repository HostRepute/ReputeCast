package workspaces_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/notify"
)

// Settings › General reads in the member's language: the approval rules,
// how members sign in, the transfer's notes and the problems. The new
// Owner's notification and the audit log's summary, written while Linh
// read English, read in Vietnamese for a Vietnamese reader.
func TestGeneralSettingsInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignInDemo("linh@acme.io")

	var s settings
	vi.Do("GET", ws+"/settings", nil).MustStatus(t, 200).JSON(t, &s)
	if got := strings.Join(s.Approvals.Rules, " · "); got != "người nhận ≥ 10.000 · tuyến có một nhà cung cấp đang khởi động · tác giả chưa được gửi" {
		t.Errorf("rules: %q", got)
	}
	if !strings.HasPrefix(s.Members.SignIn, "SSO Google Workspace cho ") || !strings.HasSuffix(s.Members.SignIn, " · vẫn cho phép mật khẩu và khóa truy cập") {
		t.Errorf("sign-in: %q", s.Members.SignIn)
	}
	var marcus, aisha string
	for _, c := range s.Candidates {
		switch c.Name {
		case "Marcus Chen":
			marcus = c.MemberID
			if c.Note != "Marcus Chen đăng nhập qua Google Workspace với xác thực hai lớp do IdP bắt buộc, nên đáp ứng yêu cầu về tài khoản khẩn cấp của Chủ sở hữu." {
				t.Errorf("marcus: %q", c.Note)
			}
		case "Aisha Bello":
			aisha = c.MemberID
		}
	}
	if p := vi.Do("PUT", ws+"/settings", map[string]any{"name": "Acme Weekly", "timezone": "Mars/Olympus"}).MustStatus(t, 422).Problem(t); p["detail"] != "Múi giờ không xác định." {
		t.Errorf("timezone: %v", p)
	}
	if p := vi.Do("POST", ws+"/transfer", map[string]any{"memberId": aisha, "password": "reputecast-demo"}).MustStatus(t, 409).Problem(t); p["detail"] != "Aisha Bello chưa bật xác thực hai lớp. Chỉ thành viên đang hoạt động có xác thực hai lớp mới có thể trở thành Chủ sở hữu." {
		t.Errorf("not eligible: %v", p)
	}
	viewer := env.Client()
	viewer.Header = http.Header{"Accept-Language": {"vi"}}
	viewer.SignInDemo("aisha@acme.io")
	var vs settings
	viewer.Do("GET", ws+"/settings", nil).MustStatus(t, 200).JSON(t, &vs)
	if !strings.HasPrefix(vs.Reason, "Vai trò của bạn không thể thay đổi cài đặt không gian làm việc. Hãy nhờ ") {
		t.Errorf("viewer: %q", vs.Reason)
	}

	// Linh hands the workspace to Marcus while reading English.
	en := env.Client()
	en.SignInDemo("linh@acme.io")
	en.Do("POST", ws+"/transfer", map[string]any{"memberId": marcus, "password": "reputecast-demo"}).MustStatus(t, 204)
	var title, summary, body string
	var data []byte
	if err := env.App.Pool.QueryRow(context.Background(), `SELECT n.title, n.summary, n.body, n.data FROM notifications n JOIN users u ON u.id = n.user_id
		WHERE u.email = 'marcus@acme.io' AND n.kind = 'ownership_transferred'`).Scan(&title, &summary, &body, &data); err != nil {
		t.Fatal(err)
	}
	if title != "Linh Nguyen made you the Owner of Acme Newsletter" {
		t.Errorf("English: %q", title)
	}
	if vt, _, _ := notify.Localize(i18n.For(i18n.Vietnamese), title, summary, body, data); vt != "Linh Nguyen đã chuyển cho bạn quyền Chủ sở hữu của Acme Newsletter" {
		t.Errorf("Vietnamese: %q", vt)
	}
	var log struct {
		Items []struct{ Action, Summary string }
	}
	vi.Do("GET", ws+"/audit?limit=20", nil).MustStatus(t, 200).JSON(t, &log)
	found := false
	for _, e := range log.Items {
		if e.Action == "workspace.transferred" {
			found = e.Summary == "Linh Nguyen → Marcus Chen · Linh Nguyen giờ là Quản trị viên"
			if !found {
				t.Errorf("audit summary: %q", e.Summary)
			}
		}
	}
	if !found {
		t.Errorf("no transfer in %+v", log.Items)
	}
}
