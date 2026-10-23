package resend

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

func TestTextsReadInTheReadersLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	d := Descriptor()
	if got := d.In(vi).SuppressionMode; got != "Hai chiều" {
		t.Errorf("descriptor in Vietnamese: %q", got)
	}
	english, text := say(mAccount, i18n.Args{"verified": 2, "plan": "yes", "limit": int64(50000)})
	if english != "2 verified domains · 50,000 per month plan" {
		t.Errorf("English: %q", english)
	}
	if got := providers.TextIn(english, text, vi); got != "2 tên miền đã xác minh · gói 50.000 email mỗi tháng" {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := providers.TextIn(english, text, i18n.Default); got != english {
		t.Errorf("English reader: %q", got)
	}
}
