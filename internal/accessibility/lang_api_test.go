package accessibility_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/hostrepute/reputecast/internal/accessibility"
	"github.com/hostrepute/reputecast/internal/apptest"
	"github.com/hostrepute/reputecast/internal/i18n"
	"github.com/hostrepute/reputecast/internal/store"
)

// The composer's Accessibility tab and its problems read in the member's
// language; API clients get English. A fix recorded from Review & send
// while its author read English reads in Vietnamese for a Vietnamese
// reader, and a fix recorded before its messages were kept in English.
func TestAccessibilityInVietnamese(t *testing.T) {
	env := apptest.New(t)
	env.SeedDemo()
	linh := env.Client()
	linh.SignInDemo("linh@acme.io")
	vi := env.Client()
	vi.SignInDemo("linh@acme.io")
	vi.Header = http.Header{"Accept-Language": {"vi"}}
	id := campaignID(t, linh, "Weekly Digest #143")

	var en, inVi accessibility.AccessibilityReport
	linh.Do("GET", ws+"/campaigns/"+id+"/accessibility", nil).MustStatus(t, 200).JSON(t, &en)
	vi.Do("GET", ws+"/campaigns/"+id+"/accessibility", nil).MustStatus(t, 200).JSON(t, &inVi)
	if en.Issues[0].Title != "The hero image has no alt text" || en.Issues[0].Name != "The hero image" || en.Languages[1].Name != "Spanish" {
		t.Fatalf("English: %q · %q · %+v", en.Issues[0].Title, en.Issues[0].Name, en.Languages)
	}
	if inVi.Issues[0].Title != "Hình hero chưa có văn bản thay thế" || inVi.Issues[0].Name != "Hình hero" || inVi.Languages[1].Name != "Tiếng Tây Ban Nha" {
		t.Fatalf("Vietnamese: %q · %q · %+v", inVi.Issues[0].Title, inVi.Issues[0].Name, inVi.Languages)
	}

	p := vi.Do("POST", ws+"/campaigns/"+id+"/accessibility", map[string]any{"mjml": "<mjml><mj-body><mj-section></mj-body></mjml>"}).MustStatus(t, 422).Problem(t)
	if p["detail"] != "Không kiểm tra được email: MJML không biên dịch được. Các chỉnh sửa của bạn đã được lưu." {
		t.Fatalf("uncheckable: %v", p["detail"])
	}
	var c struct{ MJML string }
	linh.Do("GET", ws+"/campaigns/"+id, nil).MustStatus(t, 200).JSON(t, &c)
	p = vi.Do("POST", ws+"/accessibility/fix", map[string]any{"mjml": c.MJML, "kind": "decorative", "path": []int{40, 0, 0}}).MustStatus(t, 422).Problem(t)
	if p["detail"] != "Phần đó của email đã thay đổi kể từ lần kiểm tra. Hãy chạy lại kiểm tra." {
		t.Fatalf("fix problem: %v", p["detail"])
	}

	// One click from Review & send, in English.
	linh.Do("POST", ws+"/campaigns/"+id+"/readiness/fixes", map[string]any{"fixes": []map[string]any{{"action": "accessibility_one_click"}}}).MustStatus(t, 200)
	ctx := context.Background()
	wsRow, err := env.App.Q.GetWorkspaceBySlug(ctx, "acme-newsletter")
	if err != nil {
		t.Fatal(err)
	}
	facts := func() accessibility.ReadinessFacts {
		row, err := env.App.Q.AccessibilityCampaign(ctx, store.AccessibilityCampaignParams{WorkspaceID: wsRow.ID, ID: uuid.MustParse(id)})
		if err != nil {
			t.Fatal(err)
		}
		s := &accessibility.Service{Pool: env.App.Pool, Q: env.App.Q}
		return s.ReadinessFacts(ctx, &wsRow, accessibility.CampaignContent{ID: row.ID, Name: row.Name, Status: row.Status, MJML: row.Mjml, HTML: row.Html, Hash: row.ContentHash})
	}
	f := facts()
	if f.Fixed == nil || f.Fixed.Summary != "Card titles are Heading 2, and footer text is 12 px" ||
		f.Fixed.SummaryIn(i18n.For(i18n.Vietnamese)) != "Các tựa đề thẻ là Đề mục 2 và chữ ở phần chân thư cỡ 12 px" {
		t.Fatalf("fixed: %+v", f.Fixed)
	}
	// A fix recorded before its messages were kept reads in English.
	if _, err := env.App.Pool.Exec(ctx, `UPDATE accessibility_fixes SET summary_texts = '[]' WHERE campaign_id = $1`, uuid.MustParse(id)); err != nil {
		t.Fatal(err)
	}
	if f := facts(); f.Fixed == nil || f.Fixed.SummaryIn(i18n.For(i18n.Vietnamese)) != "Card titles are Heading 2, and footer text is 12 px" {
		t.Fatalf("old fix: %+v", f.Fixed)
	}
}
