package auditlog_test

import (
	"net/http"
	"testing"

	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/auditlog"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// Both audit logs read in the reader's language: the action filter, the
// actors' labels, an entry's What line, its link and client, the sign-in
// summaries, and the summaries a change kept while its author read
// English. Action names stay as they are.
func TestAuditLogsInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	vi := env.Client()
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	vi.SignInDemo("linh@acme.io")

	var l struct {
		eventList
		Categories []struct{ Key, Label string }
	}
	vi.Do("GET", ws+"/audit", nil).MustStatus(t, 200).JSON(t, &l)
	if l.Categories[0].Key != "sending" || l.Categories[0].Label != "Gửi thư" {
		t.Errorf("categories: %+v", l.Categories[:2])
	}
	routing := ""
	for _, it := range l.Items {
		if it.Action == "routing.updated" && it.RequestID == "req_8c1f2a9d" {
			routing = it.ID
		}
		if it.Action == "report.read" && it.Actor.Label != "tác tử MCP" {
			t.Errorf("agent row: %+v", it.Actor)
		}
		if it.Actor.Type == "user" && it.Actor.Label != "thành viên" {
			t.Errorf("member row: %+v", it.Actor)
		}
	}
	var d struct{ What, Client, LinkLabel, Workspace string }
	vi.Do("GET", ws+"/audit/events/"+routing, nil).MustStatus(t, 200).JSON(t, &d)
	// The demo's entry keeps the What line it was seeded with, in English.
	if d.Client != "Chrome 141 trên macOS 26" || d.LinkLabel != "Mở định tuyến" {
		t.Errorf("detail: %+v", d)
	}

	// The instance log: sign-in summaries and the instance level.
	var il instanceList
	vi.Do("GET", "/admin/audit?workspace=instance&limit=20", nil).MustStatus(t, 200).JSON(t, &il)
	byAction := map[string]int{}
	for i, it := range il.Items {
		byAction[it.Action] = i
	}
	// The demo seeds this rejection's summary in English.
	rej := il.Items[byAction["signin.rejected"]]
	if rej.Actor.Name != "Hệ thống" || rej.Actor.Label != "tự động" {
		t.Errorf("rejection: %+v", rej.Actor)
	}
	vi.Do("GET", "/admin/audit/events/"+il.Items[byAction["user.deactivated"]].ID, nil).MustStatus(t, 200).JSON(t, &d)
	if d.Workspace != "Cấp hệ thống" || d.What != "Đã vô hiệu hóa người dùng Tom Becker · tom@acme.io" {
		t.Errorf("instance detail: %+v", d)
	}

	// Retention changed in English reads in Vietnamese.
	en := env.Client()
	en.SignInDemo("linh@acme.io")
	en.Do("PUT", "/admin/audit/settings", map[string]any{"retentionDays": 180}).MustStatus(t, 200)
	vi.Do("GET", "/admin/audit?workspace=instance&limit=5", nil).MustStatus(t, 200).JSON(t, &il)
	if s := il.Items[0]; s.Action != "audit.settings_updated" || s.Summary != "thời gian lưu giữ 180 ngày (trước đây 365)" || s.Target != "Nhật ký kiểm tra cấp hệ thống" {
		t.Errorf("kept summary: %s %q %q", s.Action, s.Summary, s.Target)
	}
	en.Do("GET", "/admin/audit?workspace=instance&limit=5", nil).MustStatus(t, 200).JSON(t, &il)
	if s := il.Items[0]; s.Summary != "retention 180 days (was 365)" || s.Target != "Instance audit log" {
		t.Errorf("English summary: %q %q", s.Summary, s.Target)
	}
}

// The What line keeps its English, and has a whole sentence per verb in
// Vietnamese, with what was acted on in Vietnamese.
func TestDescribe(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	for _, c := range []struct{ action, typ, name, en, vi string }{
		{"routing.updated", "routing_rule", "Broadcasts", "Changed the Broadcasts routing rule", "Đã thay đổi quy tắc định tuyến Broadcasts"},
		{"approval.requested", "campaign", "November Update", "Requested approval for campaign “November Update”", "Đã yêu cầu phê duyệt cho chiến dịch “November Update”"},
		{"apikey.created", "api_key", "CI", "Created API key “CI”", "Đã tạo khóa API “CI”"},
		{"member.role_changed", "member", "Daniel Park", "Changed the role of member “Daniel Park”", "Đã đổi vai trò của thành viên “Daniel Park”"},
		{"workspace.update", "workspace", "", "Changed workspace", "Đã thay đổi không gian làm việc"},
		{"sms.ab_winner_picked", "campaign", "Flash sale", "Ab winner picked campaign “Flash sale”", "Ab winner picked campaign “Flash sale”"},
	} {
		if got := auditlog.Describe(c.action, c.typ, c.name); got != c.en {
			t.Errorf("%s: %q, want %q", c.action, got, c.en)
		}
		if got := auditlog.DescribeIn(vi, c.action, c.typ, c.name); got != c.vi {
			t.Errorf("%s in Vietnamese: %q, want %q", c.action, got, c.vi)
		}
	}
	if got := auditlog.SignInSummary(vi, "signin.failed", "wrong password or unknown email", auditlog.SignInMeta{Method: "password", Failures: 3, MaxAttempts: 10}); got != "sai mật khẩu · lần thử 3/10" {
		t.Errorf("failed sign-in: %q", got)
	}
	if got := auditlog.SignInSummary(vi, "signin.rejected", "email domain not allowed for Google Workspace SSO", auditlog.SignInMeta{Method: "sso", Provider: "Google Workspace", Code: "domain_not_allowed", Reference: "auth_7f3k9dqe"}); got != "SSO Google Workspace · tên miền không được phép · mã tham chiếu auth_7f3k9dqe" {
		t.Errorf("rejected sign-in: %q", got)
	}
	if got := auditlog.SignInSummary(i18n.Default, "signin.succeeded", "", auditlog.SignInMeta{Method: "password+totp", BreakGlass: true, FailedBefore: 2}); got != "Password + TOTP · break-glass Owner · after 2 failed attempts" {
		t.Errorf("English sign-in: %q", got)
	}
}
