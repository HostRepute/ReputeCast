package azureacs

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

func TestTextsReadInTheReadersLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	d := Descriptor()
	if got := d.In(vi).DomainSetup; got != "DNS thủ công" {
		t.Errorf("descriptor in Vietnamese: %q", got)
	}
	english, text := say(mAccount, i18n.Args{"minute": "yes", "perMinute": "30", "hour": "no"})
	if english != "Communication Services · reachable · 30 per minute" {
		t.Errorf("English: %q", english)
	}
	if got := providers.TextIn(english, text, vi); got != "Communication Services · truy cập được · 30 thư mỗi phút" {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := providers.TextIn(english, text, i18n.Default); got != english {
		t.Errorf("English reader: %q", got)
	}
}
