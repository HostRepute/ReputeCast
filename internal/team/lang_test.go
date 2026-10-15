package team

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

func TestTeamWordsInTheReadersLanguage(t *testing.T) {
	en, vi := i18n.Default, i18n.For(i18n.Vietnamese)
	helpers := []string{"Linh Nguyen", "Marcus Chen"}
	cases := []struct{ got, want string }{
		{readOnlyReason(en, false, helpers), "You can view the team. Inviting needs the team:invite permission and changing roles needs team:manage. Ask Linh Nguyen or Marcus Chen."},
		{readOnlyReason(en, true, nil), "You can invite people with roles you hold. Changing roles needs team:manage."},
		{readOnlyReason(vi, false, helpers), "Bạn có thể xem nhóm. Mời thành viên cần quyền team:invite và đổi vai trò cần team:manage. Hãy nhờ Linh Nguyen hoặc Marcus Chen."},
		{mNoteInstanceAdmin.Text(en, nil), "instance admin"},
		{mNoteBreakGlass.Text(vi, nil), "tài khoản khẩn cấp"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}
