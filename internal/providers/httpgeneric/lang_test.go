package httpgeneric

import (
	"errors"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

func TestTextsReadInTheReadersLanguage(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	if got := Descriptor().In(vi).Fields[0].Help; got != "Các mẫu của ánh xạ đọc khóa này dưới dạng {{ secrets.api_key }}." {
		t.Errorf("descriptor in Vietnamese: %q", got)
	}
	err := Mapping{Method: "GET", URL: "https://api.example-esp.com/v1/messages", Body: "{}"}.Validate()
	var fe *FieldError
	if !errors.As(err, &fe) || fe.Field != "method" || fe.Message != "Use POST, PUT or PATCH." {
		t.Fatalf("validate: %#v", err)
	}
	if got := fe.MessageIn(vi); got != "Hãy dùng POST, PUT hoặc PATCH." {
		t.Errorf("Vietnamese: %q", got)
	}
	if got := fe.MessageIn(i18n.Default); got != "Use POST, PUT or PATCH." {
		t.Errorf("English: %q", got)
	}
}
