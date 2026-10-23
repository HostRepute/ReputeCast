package liquid

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The token menu reads in the member's language: the contact fields and
// system links, and the fallback a first name gets in the email.
func TestCatalogInVietnamese(t *testing.T) {
	custom := []CustomField{{Key: "plan", Label: "Plan", Type: "select"}}
	vi := CatalogIn(i18n.For(i18n.Vietnamese), custom)
	en := Catalog(custom)
	if len(vi) != len(en) {
		t.Fatalf("vi has %d tokens, en %d", len(vi), len(en))
	}
	want := map[string]string{"first_name": "Tên", "last_name": "Họ", "email": "Địa chỉ email", "unsubscribe_url": "Liên kết hủy đăng ký",
		"preferences_url": "Liên kết tùy chọn email", "view_in_browser_url": "Liên kết xem trên trình duyệt", "plan": "Plan"}
	for _, tok := range vi {
		if w, ok := want[tok.Key]; ok && tok.Label != w {
			t.Errorf("%s: %q, want %q", tok.Key, tok.Label, w)
		}
		switch tok.Key {
		case "first_name":
			if tok.Fallback != "bạn" {
				t.Errorf("first-name fallback: %q", tok.Fallback)
			}
		case "plan":
			if tok.Fallback != "free" {
				t.Errorf("a custom field keeps its fallback: %q", tok.Fallback)
			}
		}
	}
	// English reads as it always has.
	for i, tok := range CatalogIn(i18n.Default, custom) {
		if !reflect.DeepEqual(tok, en[i]) {
			t.Errorf("English %d: %+v, want %+v", i, tok, en[i])
		}
	}
	if FirstNameFallback(i18n.Default) != "there" || FirstNameFallback(i18n.For(i18n.Vietnamese)) != "bạn" {
		t.Error("first-name fallback")
	}
}

func TestValidateInVietnamese(t *testing.T) {
	src := "Hi\n{% if x %}unclosed"
	en, vi := Validate(src), ValidateIn(i18n.For(i18n.Vietnamese), src)
	if len(en) != 1 || len(vi) != 1 || en[0].Line != vi[0].Line {
		t.Fatalf("problems: %+v / %+v", en, vi)
	}
	if !strings.HasPrefix(en[0].Message, "Line 2: ") || !strings.HasPrefix(vi[0].Message, "Dòng 2: ") {
		t.Fatalf("line prefix: %q / %q", en[0].Message, vi[0].Message)
	}
	loop := "{% for i in (1..5000) %}{{ i }}{% endfor %}"
	if p := Validate(loop); len(p) == 0 || p[0].Message != "Loops are limited to 1000 items; (1..5000) has 5000." {
		t.Fatalf("loop limit, English: %+v", p)
	}
	if p := ValidateIn(i18n.For(i18n.Vietnamese), loop); len(p) == 0 || p[0].Message != "Vòng lặp chỉ được tối đa 1000 mục; (1..5000) có 5000 mục." {
		t.Fatalf("loop limit, Vietnamese: %+v", p)
	}
}
