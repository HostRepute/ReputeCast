package postmark

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

func TestTextsReadInTheReadersLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	d := Descriptor()
	if got := d.In(vi).Terms[0].Variant; got != "Luồng Transactional" {
		t.Errorf("descriptor in Vietnamese: %q", got)
	}
	english, text := say(mAccount, i18n.Args{"plan": "yes", "n": int64(10000)})
	if english != "Approved · 10,000 per month plan" {
		t.Errorf("English: %q", english)
	}
	if got := providers.TextIn(english, text, vi); got != "Đã duyệt · gói 10.000 email mỗi tháng" {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := providers.TextIn(english, text, i18n.Default); got != english {
		t.Errorf("English reader: %q", got)
	}
}
