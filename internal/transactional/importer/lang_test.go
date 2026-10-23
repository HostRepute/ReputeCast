package importer

import (
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The report reads in each reader's language: a note keeps its English and
// its message, the Notes cell follows them, and the stored report keeps the
// messages beside the notes.
func TestNotesInVietnamese(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	c := body(DialectPostmark, "<p>Hi</p>\n{{> footer}}\n")
	n := hasNote(c, NoteManual, "the Mustachio partial `{{> footer}}` has no Liquid equivalent. Use a saved section.")
	if n == nil {
		t.Fatalf("notes: %+v", c.Notes)
	}
	if got := n.In(vi).Message; got != "partial `{{> footer}}` của Mustachio không có cách viết tương ứng trong Liquid. Hãy dùng một phần đã lưu." {
		t.Fatalf("note (vi): %q", got)
	}
	if got := c.SummaryIn(vi); !strings.HasPrefix(got, "partial `{{> footer}}` của Mustachio") {
		t.Fatalf("summary (vi): %q", got)
	}
	if got := c.Summary(); !strings.HasPrefix(got, "the Mustachio partial") {
		t.Fatalf("summary (en): %q", got)
	}

	// Code in braces stays as it is written, in every language.
	if got := say("note.rawBlock", i18n.Args{"src": "{{{{raw}}}}"}).In(i18n.Default); got != "the raw block `{{{{raw}}}}` has no Liquid equivalent. Use `{% raw %}`." {
		t.Fatalf("raw block (en): %q", got)
	}
	if got := say("note.unsubscribe", i18n.Args{"provider": "Postmark", "src": "{{pm:unsubscribe}}"}).In(vi); got != "`{{pm:unsubscribe}}` của Postmark đã thành `{{ unsubscribe_url }}`." {
		t.Fatalf("unsubscribe (vi): %q", got)
	}

	// A clean conversion's cell names its variables.
	vs := []Variable{{Name: "name"}, {Name: "action_url"}}
	if got := VariablesLine(vs); got != "2 variables declared: name, action_url" {
		t.Fatalf("variables (en): %q", got)
	}
	if got := VariablesLineIn(vi, vs); got != "Đã khai báo 2 biến: name, action_url" {
		t.Fatalf("variables (vi): %q", got)
	}
	if got := VariablesLineIn(vi, append(vs, Variable{Name: "plan"})); got != "Đã khai báo 3 biến" {
		t.Fatalf("variables (vi): %q", got)
	}

	// The stored report: an ID that was taken, read back in Vietnamese.
	it := item(c, Source{Ref: "t", Name: "T"}, map[string]bool{"t": true})
	if it.Key != "t-2" || len(it.Texts) != len(it.Notes) {
		t.Fatalf("stored: %+v", it)
	}
	v := it.localize(vi)
	found := false
	for _, n := range v.Notes {
		if n.Message == "ID `t` đã tồn tại, nên mẫu được nhập với ID `t-2`." {
			found = true
		}
	}
	if !found || !strings.HasPrefix(v.Summary, "partial `{{> footer}}`") {
		t.Fatalf("localized: %+v", v)
	}
	// English readers get the report as it was stored.
	if en := it.localize(i18n.Default); en.Summary != it.Summary || en.Notes[0].Message != it.Notes[0].Message {
		t.Fatalf("English: %+v", en)
	}

	// A provider's failure keeps its English and reads in other languages.
	se := sourceErr(429, CodeRateLimited, "source.rateLimited", i18n.Args{"name": "SendGrid"})
	if se.Error() != "SendGrid is rate-limiting requests (429). Try again in a minute." || errorText(vi, se) != "SendGrid đang giới hạn số yêu cầu (429). Hãy thử lại sau một phút." {
		t.Fatalf("source error: %q, %q", se.Error(), errorText(vi, se))
	}
}
