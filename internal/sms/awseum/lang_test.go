package awseum

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

func TestManifestReadsInTheReadersLanguage(t *testing.T) {
	m := Descriptor().Manifest
	if m.Summary != "Pay as you go · in your AWS account, with SES next door" {
		t.Errorf("English: %q", m.Summary)
	}
	if got := m.In(i18n.For(i18n.Vietnamese)).Summary; got != "Trả theo mức dùng · ngay trong tài khoản AWS của bạn, cạnh SES" {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := m.In(i18n.Default).Summary; got != m.Summary {
		t.Errorf("English reader: %q", got)
	}
}
