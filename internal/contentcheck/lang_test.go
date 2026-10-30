package contentcheck

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// Check writes English, as Send Readiness and API clients read it; the
// composer's checks read in the member's language through In.
func TestIssuesInVietnamese(t *testing.T) {
	big := strings.Repeat(`<span style="font-family:Georgia;color:#374151;font-size:13px;line-height:19px">x</span>`, 1100)
	src := `<mjml><mj-body>
<mj-section><mj-column><mj-image src="/media/1/logo.png" alt="Acme" width="96px"></mj-image><mj-text>Pricing</mj-text></mj-column></mj-section>
<mj-section><mj-column><mj-image src="/media/1/hero.png" width="552px" height="220px" padding="0px"></mj-image><mj-text>48 hours. 30% off.</mj-text></mj-column></mj-section>
<mj-section><mj-column><mj-button href="https://go.acme-promo.link/sale">Claim your discount</mj-button><mj-button href="https://bit.ly/abc">Short</mj-button><mj-button href="#">Empty</mj-button></mj-column></mj-section>
<mj-section><mj-column><mj-text>` + big + `</mj-text></mj-column><mj-column><mj-text>{{ frist_name }}</mj-text></mj-column></mj-section>
</mj-body></mjml>`
	c := &Checker{Blocklist: StaticBlocklist{"go.acme-promo.link": {"SURBL", "Spamhaus DBL"}}}
	r, err := c.Check(context.Background(), Input{Subject: "FLASH SALE!!! 48 HOURS ONLY", MJML: src, Fields: []string{"first_name", "email"}})
	if err != nil {
		t.Fatal(err)
	}
	if en := r.In(i18n.For(i18n.English)); !reflect.DeepEqual(en, r) {
		t.Fatal("English readers get the report as Check wrote it")
	}
	vi := r.In(i18n.For(i18n.Vietnamese))
	byID := func(rep ContentCheckReport, id string) ContentCheckIssue {
		for _, is := range rep.Issues {
			if is.ID == id {
				return is
			}
		}
		t.Fatalf("no %s", id)
		return ContentCheckIssue{}
	}
	want := map[string][3]string{
		"content.alt_text":              {"Hình hero chưa có văn bản thay thế", "Hình chưa có văn bản thay thế", "Lưu"},
		"links.blocklist":               {"Nút trỏ đến go.acme-promo.link, tên miền bị liệt kê trên SURBL và Spamhaus DBL", "Tên miền liên kết bị liệt kê trên SURBL và Spamhaus DBL", "Sửa liên kết"},
		"links.shortener":               {"Nút chuyển hướng qua bit.ly, một dịch vụ rút gọn liên kết", "Liên kết rút gọn", "Sửa liên kết"},
		"links.empty":                   {"Nút “Empty” chưa có liên kết", "Nút chưa có liên kết", "Sửa liên kết"},
		"compliance.footer":             {"Nội dung thư không có liên kết hủy đăng ký hay địa chỉ bưu chính", "Không có liên kết hủy đăng ký hay địa chỉ bưu chính", "Chèn khối chân thư"},
		"personalization.unknown_field": {"{{ frist_name }} không phải là trường liên hệ; thẻ này sẽ không hiện gì", "", ""},
		"subject.spam":                  {"Tiêu đề vướng 3 dấu hiệu spam", "", "Dùng gợi ý"},
	}
	for id, w := range want {
		is := byID(vi, id)
		fix := ""
		if len(is.Fixes) > 0 {
			fix = is.Fixes[0].Label
		}
		if is.Title != w[0] || is.Tag != w[1] || fix != w[2] {
			t.Errorf("%s: %q · %q · %q", id, is.Title, is.Tag, fix)
		}
	}
	if d := byID(vi, "subject.spam").Detail; d != `Gợi ý: "48-hour flash sale"` {
		t.Errorf("suggestion: %q", d)
	}
	if d := byID(vi, "content.size").Detail; !strings.HasPrefix(d, "Gmail cắt bớt thư và ẩn chân thư, kể cả liên kết hủy đăng ký.") {
		t.Errorf("size detail: %q", d)
	}
	// The English stays as it was, for Send Readiness and API clients.
	if is := byID(r, "links.blocklist"); is.Title != "Button links to go.acme-promo.link, listed on SURBL and Spamhaus DBL" || is.Tag != "Link domain on SURBL and Spamhaus DBL" ||
		is.Fixes[0].Label != "Edit link" {
		t.Errorf("English blocklist: %+v", is)
	}
	if is := byID(r, "content.size"); is.Title != "105 KB is over Gmail's 102 KB clipping limit" {
		t.Errorf("English size: %q", is.Title)
	}

	// Kept texts survive JSON, as another package stores them.
	b, err := json.Marshal(byID(r, "content.alt_text").TitleText)
	if err != nil {
		t.Fatal(err)
	}
	var back i18n.Text
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if got := TextIn(i18n.For(i18n.Vietnamese), back); got != "Hình hero chưa có văn bản thay thế" {
		t.Errorf("kept title: %q", got)
	}
}

// Section names and lists inside an issue read in the reader's language.
func TestSectionsAndListsInVietnamese(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	cases := []struct {
		t        i18n.Text
		en, inVi string
	}{
		{mAltTitle.Keep(i18n.Args{"image": "section", "section": "2 columns"}), "Image in the 2 columns section has no alt text", "Hình trong phần 2 cột chưa có văn bản thay thế"},
		{mAltTitle.Keep(i18n.Args{"image": "logo"}), "Logo has no alt text", "Logo chưa có văn bản thay thế"},
		{mSizeBulkMany.Keep(i18n.Args{"count": 3, "section": "text", "share": percentArg(62)}), "The three text sections are the bulk of it (62%).", "3 phần văn bản chiếm phần lớn dung lượng (62%)."},
		{mSizeBulkCards.Keep(i18n.Args{"count": 4, "section": "product cards", "css": "yes"}), "Four product cards with inline CSS are the bulk of it.", "4 thẻ sản phẩm có CSS nội tuyến chiếm phần lớn dung lượng."},
		{mSizeRemoveRow.Keep(i18n.Args{"n": 3}), "Remove the third row", "Xóa hàng thứ ba"},
		{mSizeRemoveSection.Keep(i18n.Args{"section": "footer"}), "Remove the footer section", "Xóa phần chân thư"},
		{mRatioTitle.Keep(i18n.Args{"share": percentArg(92)}), "The email is 92% image", "Hình ảnh chiếm 92% email"},
		{mLinkBrokenTitle.Keep(i18n.Args{"kind": "link", "url": "https://acme.io/gone", "status": 0}), "Link to https://acme.io/gone doesn't respond", "Liên kết đến https://acme.io/gone không phản hồi"},
		{mLinkBrokenTitle.Keep(i18n.Args{"kind": "image", "url": "https://acme.io/x", "status": 404}), "Image to https://acme.io/x returns 404", "Hình đến https://acme.io/x trả về 404"},
		{mLiquidTitle.Keep(i18n.Args{"where": "preheader", "problem": "Line 1: Unterminated tag."}), "Preheader: Line 1: Unterminated tag.", "Dòng xem trước: Line 1: Unterminated tag."},
	}
	for _, c := range cases {
		if got := TextIn(i18n.Default, c.t); got != c.en {
			t.Errorf("%s in English: %q", c.t.ID, got)
		}
		if got := TextIn(vi, c.t); got != c.inVi {
			t.Errorf("%s in Vietnamese: %q", c.t.ID, got)
		}
	}
}
