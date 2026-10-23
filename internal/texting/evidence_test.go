package texting

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The evidence ReputeCast records for a change made by text reads in the
// reader's language, rows written before included; evidence people or
// forms gave stays as it was given.
func TestEvidenceInVietnamese(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	for in, want := range map[string]string{
		"Opted out by text":         "Đã tắt nhận qua tin nhắn",
		"Opted back in by text":     "Đã bật nhận lại qua tin nhắn",
		"Confirmed by replying YES": "Đã xác nhận bằng cách trả lời YES",
		"Text message":              "Tin nhắn",
		"Opt-in confirmation":       "Tin nhắn xác nhận đăng ký",
		"acme.io/sms-signup":        "acme.io/sms-signup",
		"Front desk sign-up sheet":  "Front desk sign-up sheet",
	} {
		if got := evidenceIn(vi, in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
		if got := evidenceIn(i18n.Default, in); got != in {
			t.Errorf("English %q: %q", in, got)
		}
	}
}
