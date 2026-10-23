package forms

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/pagelabels"
)

var vi = i18n.For(i18n.Vietnamese)

// The English the forms write reads as it did before its words moved to
// messages/en.json, and Vietnamese has every message.
func TestFormsEnglishUnchanged(t *testing.T) {
	if missing := msgs.Missing(i18n.Vietnamese); len(missing) > 0 {
		t.Errorf("Vietnamese is missing %v", missing)
	}
	_, err := DerivePalette("blue", "#FFFFFF")
	if err == nil || err.Error() != `"blue" isn't a colour. Use a hex value like #4338CA` {
		t.Errorf("not a colour: %v", err)
	}
	_, err = DerivePalette("#4338CA", "#333333")
	if err == nil || err.Error() != "#333333 is too dark for the page background: the footer text needs 4.5:1 against it. Pick a lighter colour" {
		t.Errorf("too dark: %v", err)
	}
	var pe *paletteError
	if !errors.As(err, &pe) || !pe.background || textIn(vi, pe.text) != "#333333 quá tối cho nền trang: chữ ở chân trang cần độ tương phản 4,5:1 với nền. Hãy chọn màu sáng hơn" {
		t.Errorf("too dark, in Vietnamese: %+v", pe)
	}
	p, err := DerivePalette("#FDE047", "#FFFFFF")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Notes) != 2 || !strings.HasPrefix(p.Notes[0], "White text on #FDE047 is 1.") || !strings.HasSuffix(p.Notes[0], ":1, under the 4.5:1 buttons need, so buttons use dark text.") ||
		!strings.HasPrefix(p.Notes[1], "Links use #") {
		t.Errorf("notes: %q", p.Notes)
	}
	if got := p.notesIn(i18n.Default); strings.Join(got, "|") != strings.Join(p.Notes, "|") {
		t.Errorf("English notes %q, want %q", got, p.Notes)
	}
	if got := p.notesIn(vi); len(got) != 2 || !strings.HasPrefix(got[0], "Chữ trắng trên #FDE047 có độ tương phản 1,") {
		t.Errorf("Vietnamese notes: %q", got)
	}
	for code, want := range map[string]string{
		"110200": "Cloudflare rejected the site key (error 110200)",
		"110100": "Cloudflare doesn't know this site key (error 110100)",
		"110500": "The Turnstile widget couldn't load (error 110500)",
		"110999": "Turnstile reported error 110999",
	} {
		if got := english(turnstileErrorWords(code)); got != want {
			t.Errorf("turnstile %s: %q", code, got)
		}
	}
	if got := english(secretRejected([]string{"invalid-input-secret", "missing-input-secret"})); got != "Cloudflare rejected the secret key (invalid-input-secret, missing-input-secret)" {
		t.Errorf("secret: %q", got)
	}
	if got := DefaultConsentText("Acme"); got != "I agree to receive emails from Acme. Unsubscribe any time with one click. See the privacy policy." {
		t.Errorf("consent: %q", got)
	}
	if c := withContentDefaults(contentDoc{}, pagelabels.English()); c.Button != "Subscribe" || c.ListsLabel != "What would you like?" ||
		c.EmailPlaceholder != "you@company.com" || c.NamePlaceholder != "Sam" {
		t.Errorf("English content defaults: %+v", c)
	}
	if got := labelProblem(i18n.Default, "form.button", errors.New("unclosed")); got != "Check the braces { } and tags < > of this text: one isn't closed." {
		t.Errorf("label problem: %q", got)
	}
}

// The custom domain's finding is stored in English beside its message;
// a member reads it in their language, the "Last check failed" wrapper and
// all.
func TestDomainFindingInVietnamese(t *testing.T) {
	inner := mDomainNoCNAME.Keep(i18n.Args{"domain": "news.acme.io", "target": "app.example.com"})
	detail := mDomainLastFailed.Keep(i18n.Args{"detail": inner})
	stored := english(detail)
	if stored != "Last check failed: No CNAME from news.acme.io to app.example.com yet. DNS changes can take up to an hour to show. Checking again soon." {
		t.Fatalf("English: %q", stored)
	}
	if !failedBefore(stored) || failedBefore(english(inner)) {
		t.Error("failedBefore reads the stored English")
	}
	texts := keptTexts("domain_detail", detail)
	var round map[string]json.RawMessage
	if json.Unmarshal(texts, &round) != nil || len(round) != 1 {
		t.Fatalf("texts: %s", texts)
	}
	want := "Lần kiểm tra gần nhất thất bại: Chưa có CNAME từ news.acme.io đến app.example.com. Thay đổi DNS có thể mất tới một giờ mới hiện. Sẽ sớm kiểm tra lại."
	if got := keptIn(vi, texts, "domain_detail", stored); got != want {
		t.Errorf("Vietnamese:\ngot  %q\nwant %q", got, want)
	}
	if got := keptIn(i18n.Default, texts, "domain_detail", stored); got != stored {
		t.Errorf("English: %q", got)
	}
	if got := keptIn(vi, nil, "domain_detail", "Old finding."); got != "Old finding." {
		t.Errorf("a finding from before: %q", got)
	}
}

// ReputeCast's default text of the pages reads in the page's language while
// the workspace keeps it; text the publisher wrote stays as written.
func TestPageDefaultsInThePageLanguage(t *testing.T) {
	p := defaultPages()
	if p.Subscribe.Footnote != "We'll send a confirmation link first. No spam, ever. You can change lists or pause emails from any message." {
		t.Fatalf("English default: %q", p.Subscribe.Footnote)
	}
	if got := p.withDefaultsIn("en"); got.Subscribe.Footnote != p.Subscribe.Footnote || got.Unsubscribe.FewerButton != "Send me the monthly best-of only" {
		t.Errorf("English page: %+v", got.Unsubscribe)
	}
	got := p.withDefaultsIn("vi")
	if got.Subscribe.Footnote != "Chúng tôi sẽ gửi liên kết xác nhận trước. Không bao giờ có thư rác. Bạn có thể đổi danh sách hoặc tạm dừng email từ bất kỳ thư nào." ||
		got.Unsubscribe.FewerTitle != "Quá nhiều email? Hãy nhận ít hơn." || got.Unsubscribe.FewerButton != "Chỉ gửi cho tôi bản tuyển chọn hằng tháng" {
		t.Errorf("Vietnamese page: %q / %+v", got.Subscribe.Footnote, got.Unsubscribe)
	}
	if got := p.Subscribe.FootnoteIn("es"); got != p.Subscribe.Footnote {
		t.Errorf("a language ReputeCast doesn't ship keeps the text: %q", got)
	}
	own := p
	own.Subscribe.Footnote = "Weekly, every Tuesday."
	if got := own.withDefaultsIn("vi").Subscribe.Footnote; got != "Weekly, every Tuesday." {
		t.Errorf("the publisher's own footnote: %q", got)
	}
	if c := withContentDefaults(contentDoc{}, pagelabels.For("vi", nil)); c.Button != "Đăng ký" || c.ListsLabel != "Bạn muốn nhận gì?" {
		t.Errorf("a Vietnamese form's defaults: %+v", c)
	}
	if ctype, problem := sniffLogo(vi, nil); ctype != "" || problem != "Tệp trống." {
		t.Errorf("empty logo: %q", problem)
	}
}
