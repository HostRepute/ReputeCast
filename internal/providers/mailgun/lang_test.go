package mailgun

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

func TestTextsReadInTheReadersLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	d := Descriptor()
	if got := d.In(vi).Fields[2].Label; got != "Tên miền gửi thư" {
		t.Errorf("descriptor in Vietnamese: %q", got)
	}
	english, text := say(mAccount, i18n.Args{"sandbox": "no", "domain": "mg.acme.io", "state": "active", "limit": "yes", "n": int64(100000)})
	if english != "Domain mg.acme.io · active · custom limit 100,000 per month" {
		t.Errorf("English: %q", english)
	}
	if got := providers.TextIn(english, text, vi); got != "Tên miền mg.acme.io · active · giới hạn tùy chỉnh 100.000 email mỗi tháng" {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := providers.TextIn(english, text, i18n.Default); got != english {
		t.Errorf("English reader: %q", got)
	}
}
