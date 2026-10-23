package smtp

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/providers"
)

func TestTextsReadInTheReadersLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	d := Descriptor()
	if got := d.In(vi).Fields[0].Label; got != "Máy chủ SMTP" {
		t.Errorf("descriptor in Vietnamese: %q", got)
	}
	english, text := say(mSummary, i18n.Args{"tls": "starttls", "version": "TLS 1.3", "auth": "accepted", "mechanism": "PLAIN"})
	if english != "STARTTLS, TLS 1.3 · AUTH PLAIN accepted" {
		t.Errorf("English: %q", english)
	}
	if got := providers.TextIn(english, text, vi); got != "STARTTLS, TLS 1.3 · đã chấp nhận AUTH PLAIN" {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := providers.TextIn(english, text, i18n.Default); got != english {
		t.Errorf("English reader: %q", got)
	}
}
