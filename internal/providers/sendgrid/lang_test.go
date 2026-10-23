package sendgrid

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

func TestTextsReadInTheReadersLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	d := Descriptor()
	if got := d.In(vi).Fields[2].Label; got != "Nhóm IP" {
		t.Errorf("descriptor in Vietnamese: %q", got)
	}
	english, text := say(mAccount, i18n.Args{"kind": "paid", "reputation": 99.5})
	if english != "Paid account · reputation 99.5%" {
		t.Errorf("English: %q", english)
	}
	if got := providers.TextIn(english, text, vi); got != "Tài khoản trả phí · uy tín 99,5%" {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := providers.TextIn(english, text, i18n.Default); got != english {
		t.Errorf("English reader: %q", got)
	}
}
