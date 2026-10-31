package team

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

func TestTeamWordsInTheReadersLanguage(t *testing.T) {
	en, vi := i18n.Default, i18n.For(i18n.Vietnamese)
	helpers := []string{"Linh Nguyen", "Marcus Chen"}
	cases := []struct{ got, want string }{
		{readOnlyReason(en, false, helpers), "You can view the team. Your role can't invite people or change members' roles. Ask Linh Nguyen or Marcus Chen."},
		{readOnlyReason(en, true, nil), "You can invite people with roles you hold. Your role can't change members' roles."},
		{readOnlyReason(vi, false, helpers), "Bạn có thể xem nhóm. Vai trò của bạn không thể mời người khác hay đổi vai trò của thành viên. Hãy nhờ Linh Nguyen hoặc Marcus Chen."},
		{mNoteInstanceAdmin.Text(en, nil), "instance admin"},
		{mNoteBreakGlass.Text(vi, nil), "tài khoản khẩn cấp"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}

// The invite dialog's texts from the server, and the problems inviting
// answers, read in the inviter's language.
func TestInviteWordsInTheReadersLanguage(t *testing.T) {
	en, vi := i18n.Default, i18n.For(i18n.Vietnamese)
	pol := store.InstanceSetting{TwofaEnforcement: "sensitive", TwofaGraceDays: 7, SocialProviders: json.RawMessage(`{"google":{"enabled":true},"github":{"enabled":true}}`)}
	editor := authz.Effective(authz.RoleEditor, true, nil)
	viewer := authz.Effective(authz.RoleViewer, true, nil)
	_, missing := canGrantRole(vi, viewer, authz.RoleEditor, true, editor)
	_, owner := canGrantRole(en, editor, authz.RoleOwner, true, editor)
	if _, err := normalizeEmails(vi, []string{"jo@contractor"}); err == nil || !strings.Contains(err.Error(), "“jo@contractor” không phải là địa chỉ email.") {
		t.Errorf("not an email, in Vietnamese: %v", err)
	}
	cases := []struct{ got, want string }{
		{roleHint(en, "Editor", editor, pol), "Editors can send; two-factor becomes required (7-day grace)."},
		{roleHint(en, "Viewer", viewer, store.InstanceSetting{TwofaEnforcement: "off"}), "Viewers can view everything, never secrets."},
		{roleHint(vi, "Biên tập viên", editor, pol), "Biên tập viên gửi được chiến dịch; bắt buộc xác thực hai lớp (gia hạn 7 ngày)."},
		{signInSummary(en, pol), "Invitees pick a password or continue with Google or GitHub."},
		{signInSummary(vi, store.InstanceSetting{}), "Người được mời tự đặt mật khẩu và có thể thêm khóa truy cập."},
		{owner, "Only a transfer of ownership makes someone the Owner."},
		{missing[:len("Bạn không cấp được vai trò này vì vai trò này có “")], "Bạn không cấp được vai trò này vì vai trò này có “"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
	if !strings.Contains(missing, "“Gửi hoặc lên lịch chiến dịch”") || !strings.HasSuffix(missing, "”, còn vai trò của bạn thì không.") {
		t.Errorf("can't grant, in Vietnamese: %q", missing)
	}
}

// What you can't grant, or what someone's role has that yours doesn't, is
// named by the permissions' labels in the reader's language, never their
// keys (review X-1); the problem's permission field keeps a key for API
// clients.
func TestMissingPermissionsByTheirLabels(t *testing.T) {
	en, vi := i18n.Default, i18n.For(i18n.Vietnamese)
	_, some := canGrantRole(en, authz.NewSet(authz.CampaignsView), "custom", false, authz.NewSet(authz.CampaignsSend, authz.SubscribersExport))
	outrank := outrankError(i18n.WithLang(context.Background(), vi), "Marcus Chen", []authz.Permission{authz.WorkspaceDelete})
	cases := []struct{ got, want string }{
		{some, "You can't grant this role because it includes “Send or schedule campaigns”, “Export contacts” and “View contacts”, which your role doesn't."},
		{outrankError(context.Background(), "Marcus Chen", []authz.Permission{authz.TeamManage, authz.WorkspaceDelete}).Detail,
			"You can't change Marcus Chen's access: their role includes “Manage roles and remove members” and “Delete the workspace”, which yours doesn't."},
		{outrank.Detail, "Bạn không thay đổi được quyền truy cập của Marcus Chen: vai trò của họ có “Xóa không gian làm việc”, còn vai trò của bạn thì không."},
		{outrank.Permission, "workspace:delete"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}
