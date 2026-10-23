package mailjet

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

func TestTextsReadInTheReadersLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	d := Descriptor()
	if got := d.In(vi).Capabilities.Pricing; got != "theo gói, miễn phí 6.000 email mỗi tháng (200 email mỗi ngày), sau đó từ 9 US$ mỗi tháng" {
		t.Errorf("descriptor in Vietnamese: %q", got)
	}
	english, text := say(mAccount, i18n.Args{"state": "quarantine", "n": "100"})
	if english != "API key in quarantine · 100 per hour" {
		t.Errorf("English: %q", english)
	}
	if got := providers.TextIn(english, text, vi); got != "Khóa API đang bị cách ly · 100 thư mỗi giờ" {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := providers.TextIn(english, text, i18n.Default); got != english {
		t.Errorf("English reader: %q", got)
	}
}
