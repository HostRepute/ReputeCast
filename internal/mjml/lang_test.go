package mjml

import (
	"errors"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// What people read from this package comes in their language, while Lint,
// Compile and ParseError.Error keep writing English.
func TestProblemsInVietnamese(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)

	en, inVi := Lint(doc), LintIn(vi, doc)
	if len(en) != 1 || en[0].Message != `Line 18: mj-image width 600px exceeds the 50% column (276px); it will be scaled down. Set width="226px" or remove it.` {
		t.Fatalf("English lint: %+v", en)
	}
	if len(inVi) != 1 || inVi[0].Line != 18 || inVi[0].Message != `Dòng 18: mj-image rộng 600px, vượt quá cột 50% (276px); ảnh sẽ bị thu nhỏ. Hãy đặt width="226px" hoặc bỏ thuộc tính này.` {
		t.Fatalf("Vietnamese lint: %+v", inVi)
	}
	wide := `<mjml><mj-body><mj-section><mj-column width="70%"><mj-text>a</mj-text></mj-column><mj-column width="50%"><mj-text>b</mj-text></mj-column></mj-section></mj-body></mjml>`
	if p := LintIn(vi, wide); len(p) != 1 || p[0].Message != "Dòng 1: các cột của phần này cộng lại thành 120%; MJML sẽ ép chúng lại và bố cục bị xô lệch. Hãy chỉnh để tổng là 100%." {
		t.Fatalf("columns: %+v", p)
	}
	if p := Lint(wide); len(p) != 1 || p[0].Message != "Line 1: the columns of this section add up to 120%; MJML squeezes them and the layout shifts. Make them add up to 100%." {
		t.Fatalf("columns in English: %+v", p)
	}

	_, err := Parse("<mjml>\n<mj-body>\n<mj-section>\n<mj-column>\n<mj-text>x</mj-text>\n</mj-section>\n</mj-body>\n</mjml>")
	var pe *ParseError
	if !errors.As(err, &pe) || pe.Error() != "Line 4: unclosed <mj-column>" || pe.In(vi) != "Dòng 4: <mj-column> chưa được đóng" {
		t.Fatalf("parse error: %v", err)
	}
	if _, err := Parse("<mj-body></mj-body>"); !errors.As(err, &pe) || pe.Error() != "Line 1: the document must start with <mjml>" || pe.In(vi) != "Dòng 1: tài liệu phải bắt đầu bằng <mjml>" {
		t.Fatalf("no root: %v", err)
	}
	if _, err := Parse("</mj-section>"); !errors.As(err, &pe) || pe.Error() != "Line 1: </mj-section> has no opening tag" || pe.In(vi) != "Dòng 1: </mj-section> không có thẻ mở" {
		t.Fatalf("no opening tag: %v", err)
	}
	bad := "<mjml><mj-body><mj-section><mj-column><mj-text>x</mj-section></mj-body></mjml>"
	if res := CompileIn(vi, bad); len(res.Errors) != 1 || res.Errors[0].Message != "Dòng 1: <mj-text> chưa được đóng" {
		t.Fatalf("compile in Vietnamese: %+v", res.Errors)
	}
	if res := Compile(bad); len(res.Errors) != 1 || res.Errors[0].Message != "Line 1: unclosed <mj-text>" || res.Errors[0].In(vi) != "Dòng 1: <mj-text> chưa được đóng" {
		t.Fatalf("compile: %+v", res.Errors)
	}

	for label, want := range map[string]string{"footer": "chân thư", "2 columns": "2 cột", "raw HTML": "HTML thô", "hero": "hero", "Beta corner": "Beta corner"} {
		if got := SectionWord(vi, label); got != want {
			t.Errorf("%s: %q, want %q", label, got, want)
		}
		if got := SectionWord(i18n.Default, label); got != label {
			t.Errorf("%s in English: %q", label, got)
		}
	}
}
