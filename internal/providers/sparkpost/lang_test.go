package sparkpost

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

func TestTextsReadInTheReadersLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	d := Descriptor()
	if got := d.In(vi).Fields[1].Label; got != "Khu vực" {
		t.Errorf("descriptor in Vietnamese: %q", got)
	}
	english, text := say(mTrackingOff, nil)
	if english != "Open and click tracking are off by default and on every transmission ReputeCast sends." {
		t.Errorf("English: %q", english)
	}
	if got := providers.TextIn(english, text, vi); got != "Theo dõi lượt mở và lượt nhấp mặc định đang tắt, và tắt trên mọi lượt gửi ReputeCast thực hiện." {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := providers.TextIn(english, text, i18n.Default); got != english {
		t.Errorf("English reader: %q", got)
	}
}
