package brevo

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

func TestTextsReadInTheReadersLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	d := Descriptor()
	if got := d.In(vi).SuppressionMode; got != "Chỉ kéo về" {
		t.Errorf("descriptor in Vietnamese: %q", got)
	}
	english, text := say(mAccount, i18n.Args{"company": "yes", "name": "Acme", "plan": "free", "n": "212", "today": "yes"})
	if english != "Acme · free plan · 212 emails left today" {
		t.Errorf("English: %q", english)
	}
	if got := providers.TextIn(english, text, vi); got != "Acme · gói miễn phí · còn 212 email hôm nay" {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := providers.TextIn(english, text, i18n.Default); got != english {
		t.Errorf("English reader: %q", got)
	}
}
