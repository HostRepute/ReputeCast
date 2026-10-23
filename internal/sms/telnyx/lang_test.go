package telnyx

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

func TestManifestReadsInTheReadersLanguage(t *testing.T) {
	m := Descriptor().Manifest
	if m.Summary != "Pay as you go · its own carrier network in the US" {
		t.Errorf("English: %q", m.Summary)
	}
	if got := m.In(i18n.For(i18n.Vietnamese)).Summary; got != "Trả theo mức dùng · mạng lưới nhà mạng riêng tại Mỹ" {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := m.In(i18n.Default).Summary; got != m.Summary {
		t.Errorf("English reader: %q", got)
	}
}
