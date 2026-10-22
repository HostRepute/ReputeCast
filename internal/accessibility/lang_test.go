package accessibility

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Check writes English, as Send Readiness, API clients and agents read it;
// In writes a report in a member's language from the messages it kept.
func TestReportInVietnamese(t *testing.T) {
	src := digest(t)
	es := strings.Replace(src, ">Click here<", ">Haz clic aquí<", 1)
	de := strings.Replace(src, ">Click here<", ">Hier klicken<", 1)
	r := check(t, []Version{{Lang: "es", MJML: es}, {Lang: "en", MJML: src, Source: true}, {Lang: "de", MJML: de}},
		Options{Titles: map[string]PageTitle{"https://docs.acme.io/scheduled-rollouts": {Title: "Scheduled rollouts · Acme Docs", Lang: "en"}}})
	if en := r.In(i18n.For(i18n.English)); !reflect.DeepEqual(en, r) {
		t.Fatal("English readers get the report as Check wrote it")
	}
	vi := i18n.For(i18n.Vietnamese)
	v := r.In(vi)
	want := []struct{ title, tag, short string }{
		{"Hình hero chưa có văn bản thay thế", "Chưa có văn bản thay thế", "văn bản thay thế của hình hero"},
		{"“Plan your migration”: chữ trắng trên nền #5EBFB0 có độ tương phản 2,1:1. Cần tối thiểu 4,5:1.", "Độ tương phản 2,1:1", "độ tương phản của nút"},
		{"“Click here” không cho biết liên kết dẫn đến đâu", "“Click here” không nói lên điều gì", "“Click here”"},
		{"Chữ của liên kết là một URL trần: https://docs.acme.io/migrate-v2", "URL trần làm chữ liên kết", "một URL trần"},
		{"Cả hai tiêu đề thẻ đều là Đề mục 3, ngay sau Đề mục 1", "Đề mục 3 sau Đề mục 1", "một cấp đề mục bị bỏ qua"},
		{"Chữ ở phần chân thư cỡ 11 px. Cỡ tối thiểu là 12 px.", "Chữ 11 px", "chữ 11 px ở phần chân thư"},
	}
	for i, w := range want {
		is := v.Issues[i]
		if is.Title != w.title || is.Tag != w.tag || is.Short != w.short {
			t.Errorf("%s: %q · %q · %q", is.Kind, is.Title, is.Tag, is.Short)
		}
	}
	if is := v.Issues[0]; is.Name != "Hình hero" || is.Detail != "Trình đọc màn hình bỏ qua hình này hoặc đọc tên tệp. Văn bản thay thế được viết cho từng ngôn ngữ." {
		t.Errorf("alt text: %q · %q", is.Name, is.Detail)
	}
	if d := v.Issues[2].Detail; d != "Cả “Haz clic aquí” và “Hier klicken” cũng vậy. Trang được liên kết viết bằng tiếng Anh, nên tiếng Tây Ban Nha và tiếng Đức có ô để tự viết chữ." {
		t.Errorf("link text: %q", d)
	}
	if s := v.Issues[4].Fix.Subject; s != "Các tiêu đề thẻ" {
		t.Errorf("heading subject: %q", s)
	}
	if s := v.Issues[5].Fix.Subject; s != "Chữ ở phần chân thư" {
		t.Errorf("size subject: %q", s)
	}
	if loc := strings.Join(v.Issues[5].Location, " › "); !strings.Contains(loc, "mj-section · chân thư") {
		t.Errorf("location: %q", loc)
	}
	if v.Languages[0].Name != "Tiếng Anh" || v.Languages[1].Name != "Tiếng Tây Ban Nha" || r.Languages[1].Name != "Spanish" {
		t.Errorf("languages: %+v", v.Languages)
	}
	// The English words stay as they were.
	if r.Issues[0].Title != "The hero image has no alt text" || r.Issues[0].Name != "The hero image" || r.Issues[5].Fix.Subject != "Footer text" {
		t.Errorf("English: %q %q %q", r.Issues[0].Title, r.Issues[0].Name, r.Issues[5].Fix.Subject)
	}

	if got := r.SummaryIn(vi); got != "6 lỗi: 2 nghiêm trọng, 4 trung bình" {
		t.Errorf("summary: %q", got)
	}
	if got, want := GallerySummary(r.Issues), "Serious: the hero image has no alt text; “Plan your migration”: white text on #5EBFB0 is 2.1:1. It needs 4.5:1. Also a “Click here” link, a bare URL, a skipped heading level and 11 px text."; got != want {
		t.Errorf("gallery in English: %q", got)
	}
	if got := GallerySummaryIn(vi, r.Issues); got != "Nghiêm trọng: hình hero chưa có văn bản thay thế; “Plan your migration”: chữ trắng trên nền #5EBFB0 có độ tương phản 2,1:1. Cần tối thiểu 4,5:1. Ngoài ra còn một liên kết “Click here”, một URL trần, một cấp đề mục bị bỏ qua và chữ 11 px." {
		t.Errorf("gallery: %q", got)
	}
	if got := SeriousSummaryIn(vi, r.Issues, true); got != "văn bản thay thế của hình hero và độ tương phản của nút" {
		t.Errorf("serious summary: %q", got)
	}
	f := FindingsIn(vi, r)
	if f[0].Title != "Hình hero chưa có văn bản thay thế · Tiếng Anh, Tiếng Tây Ban Nha, Tiếng Đức" ||
		f[len(f)-1].Title != "role=\"presentation\" trên "+i18n.For(i18n.Vietnamese).Region.Int(int64(r.Counts.LayoutTables))+" bảng bố cục" {
		t.Errorf("findings: %q · %q", f[0].Title, f[len(f)-1].Title)
	}
	if en := Findings(r); en[0].Title != "The hero image has no alt text · English, Spanish, German" {
		t.Errorf("findings in English: %q", en[0].Title)
	}

	// A kept title reads the same after going through JSON, as Send
	// Readiness stores it.
	b, _ := json.Marshal(r.Issues[5].Texts.Title)
	var back i18n.Text
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if got := capitalize(TextIn(vi, back)); got != v.Issues[5].Title {
		t.Errorf("kept title: %q", got)
	}
}

// A fix from Review & send keeps its messages: its summary reads in each
// reader's language, and a record from before reads in English.
func TestFixesInVietnamese(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	r := check(t, []Version{{Lang: "en", MJML: digest(t), Source: true}}, Options{})
	_, change, err := applyOneClick(digest(t), r)
	if err != nil {
		t.Fatal(err)
	}
	if change.Applied != "Changed 2 headings to Heading 2 and set footer text to 12 px" || change.Summary != "Card titles are Heading 2, and footer text is 12 px" {
		t.Fatalf("English: %q / %q", change.Applied, change.Summary)
	}
	if got := change.AppliedIn(vi); got != "Đã đổi 2 đề mục thành Đề mục 2 và đã đặt cỡ chữ ở phần chân thư thành 12 px" {
		t.Errorf("applied: %q", got)
	}
	if got := change.SummaryIn(vi); got != "Các tiêu đề thẻ là Đề mục 2 và chữ ở phần chân thư cỡ 12 px" {
		t.Errorf("summary: %q", got)
	}
	if got := change.LabelIn(vi); got != "đề mục và cỡ chữ" {
		t.Errorf("label: %q", got)
	}
	rec := FixRecord{Summary: change.Summary, SummaryTexts: [][]i18n.Text{change.SummaryTexts}}
	b, _ := json.Marshal(rec.SummaryTexts)
	rec.SummaryTexts = nil
	if err := json.Unmarshal(b, &rec.SummaryTexts); err != nil {
		t.Fatal(err)
	}
	if got := rec.SummaryIn(vi); got != "Các tiêu đề thẻ là Đề mục 2 và chữ ở phần chân thư cỡ 12 px" {
		t.Errorf("stored summary: %q", got)
	}
	if got := rec.SummaryIn(i18n.For(i18n.English)); got != change.Summary {
		t.Errorf("English readers: %q", got)
	}
	if old := (FixRecord{Summary: "Card titles are Heading 2"}); old.SummaryIn(vi) != "Card titles are Heading 2" {
		t.Error("a fix recorded before its messages were kept reads in English")
	}

	// Problems with a fix read in the reader's language, in lower case as
	// errors are written.
	if _, err := ApplyFix(digest(t), FixRequest{Kind: FixAltText, Path: r.Issues[0].Path}); err == nil || err.Error() != "write the alt text first" || ErrorIn(vi, err) != "hãy viết văn bản thay thế trước" {
		t.Errorf("alt text error: %v", err)
	}
	if got := ErrorIn(vi, fixError(ErrFixTarget)); got != "phần đó của email đã thay đổi kể từ lần kiểm tra. Hãy chạy lại kiểm tra" {
		t.Errorf("changed: %q", got)
	}
	if _, err := ApplyFix(digest(t), FixRequest{Kind: FixColor, Path: r.Issues[1].Path, Value: "teal-ish"}); err == nil || err.Error() != `"teal-ish" isn't a color` || ErrorIn(vi, err) != `"teal-ish" không phải là màu` {
		t.Errorf("color error: %v", err)
	}
	if _, err := ApplyFix(digest(t), FixRequest{Kind: FixAltText, Path: []int{99}}); !errors.Is(err, ErrFixTarget) {
		t.Errorf("ErrFixTarget: %v", err)
	}
}
