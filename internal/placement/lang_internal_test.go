package placement

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Why a provider is down reads in the reader's words in the run dialog,
// whatever English its status keeps.
func TestDownReasonInVietnamese(t *testing.T) {
	vi, en := i18n.For(i18n.Vietnamese), i18n.For(i18n.English)
	cases := []struct{ name, got, want string }{
		{"an outage", downReason(vi, "outage", "Down"), "API của nhà cung cấp ngừng phản hồi."},
		{"refused credentials", downReason(vi, "auth", "Down · auth failed"), "nhà cung cấp từ chối thông tin xác thực của ReputeCast."},
		{"an outage in English", downReason(en, "outage", "Down"), "its API stopped answering."},
		{"a reason of its own", downReason(vi, "", "Paused by Postmark"), "Paused by Postmark"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: %q, want %q", c.name, c.got, c.want)
		}
	}
}
