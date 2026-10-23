package vonage

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

func TestManifestReadsInTheReadersLanguage(t *testing.T) {
	m := Descriptor().Manifest
	if m.Summary != "Pay as you go · strong sender ID coverage in Europe" {
		t.Errorf("English: %q", m.Summary)
	}
	if got := m.In(i18n.For(i18n.Vietnamese)).Summary; got != "Trả theo mức dùng · hỗ trợ ID người gửi rộng khắp châu Âu" {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := m.In(i18n.Default).Summary; got != m.Summary {
		t.Errorf("English reader: %q", got)
	}
}
