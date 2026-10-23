package authn

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The password rules read in the person's language; the CLI's stay English.
func TestPasswordProblemInVietnamese(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	if p := PasswordProblemIn(vi, "short", 12); p != "Hãy dùng ít nhất 12 ký tự. Dùng cụm mật khẩu cũng được." {
		t.Fatalf("short: %q", p)
	}
	if p := PasswordProblemIn(vi, "            ", 12); p != "Mật khẩu không được chỉ gồm khoảng trắng." {
		t.Fatalf("spaces: %q", p)
	}
	if p := PasswordProblem("short", 12); p != "Use at least 12 characters. Passphrases are fine." {
		t.Fatalf("English: %q", p)
	}
}
