package authz

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The permission catalogue keeps its English, and reads in Vietnamese
// through its messages: every permission, group and role is translated.
func TestCatalogueInVietnamese(t *testing.T) {
	if missing := msgs.Missing(i18n.Vietnamese); len(missing) > 0 {
		t.Fatalf("not translated: %v", missing)
	}
	vi := i18n.For(i18n.Vietnamese)
	d, _ := Lookup(CampaignsSend)
	if d.Label != "Send or schedule campaigns" || d.Group != "Campaigns" || d.Description != "Start or schedule a send, and resume a paused one." {
		t.Fatalf("English: %+v", d)
	}
	if got := LabelIn(vi, CampaignsSend); got != "Gửi hoặc lên lịch chiến dịch" {
		t.Errorf("label: %q", got)
	}
	if got := GroupIn(vi, g("lists")); got != "Danh sách và phân khúc" {
		t.Errorf("group: %q", got)
	}
	if got := GroupIn(i18n.Default, "Lists and segments"); got != "Lists and segments" {
		t.Errorf("English group: %q", got)
	}
	for _, d := range append(append([]Def{}, Defs...), AgencyDefs...) {
		if LabelIn(vi, d.Key) == d.Label || DescriptionIn(vi, d.Key) == d.Description || GroupIn(vi, d.Group) == "" {
			t.Errorf("%s isn't in Vietnamese", d.Key)
		}
	}
	if got := RoleDescriptionIn(vi, RoleViewer); got != "Chỉ đọc. Không bao giờ thấy thông tin bí mật" {
		t.Errorf("viewer: %q", got)
	}
	r, _ := Builtin(RoleViewer)
	if r.Description != "Read-only. Never sees secrets" {
		t.Errorf("English viewer: %q", r.Description)
	}
}

// "Who can help" reads in the reader's language, never the permission key
// (review X-1): API clients read the key from the problem's permission field.
func TestDeniedText(t *testing.T) {
	helpers := []string{"Marcus Chen", "Linh Nguyen"}
	if got := DeniedMessage(CampaignsSend, helpers); got != "Sending needs a permission you don't have. Ask Marcus Chen or Linh Nguyen." {
		t.Errorf("English: %q", got)
	}
	if got := DeniedMessage(CampaignsView, nil); got != "View campaigns needs a permission you don't have." {
		t.Errorf("English, no helpers: %q", got)
	}
	if got := DeniedMessage(AgencyView, nil); got != "This needs a permission you don't have." {
		t.Errorf("English, agency: %q", got)
	}
	if got := DeniedText(i18n.For(i18n.Vietnamese), CampaignsSend, helpers); got != "Bạn chưa có quyền gửi chiến dịch. Hãy hỏi Marcus Chen hoặc Linh Nguyen." {
		t.Errorf("Vietnamese: %q", got)
	}
}
