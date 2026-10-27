package agency

import (
	"testing"

	"github.com/hostrepute/reputecast/internal/authz"
	"github.com/hostrepute/reputecast/internal/i18n"
)

// A refused agency action says what the reader can't do, in plain words
// and in their language; it never names a permission key (review X-1).
func TestRefusalsInPlainWords(t *testing.T) {
	en, vi := i18n.Default, i18n.For(i18n.Vietnamese)
	for _, c := range []struct{ got, want string }{
		{mDeniedAsk.Text(en, i18n.Args{"what": deniedWhat(en, authz.AgencyLibrary), "helpers": "Tomás Herrera"}), "You can't change the library or workspace templates. Ask Tomás Herrera."},
		{mDenied.Text(en, i18n.Args{"what": deniedWhat(en, authz.AgencyUsage)}), "You can't see costs and rebill rates or export usage."},
		{mDenied.Text(vi, i18n.Args{"what": deniedWhat(vi, authz.AgencyUsage)}), "Bạn không thể xem chi phí và mức tính phí lại hay xuất mức sử dụng."},
		{mCopyNeeds.Text(en, i18n.Args{"kind": KindSection, "workspace": "Contoso Coffee"}), "Your role in Contoso Coffee can't edit templates."},
		{mCopyNeeds.Text(en, i18n.Args{"kind": KindAutomation, "workspace": "Contoso Coffee"}), "Your role in Contoso Coffee can't edit automations."},
		{mCopyNeeds.Text(vi, i18n.Args{"kind": KindTransactional, "workspace": "Contoso Coffee"}), "Vai trò của bạn trong Contoso Coffee không thể sửa mẫu giao dịch."},
		{mCopyNeedsHere.Text(en, i18n.Args{"kind": KindTemplate}), "Your role can't edit templates here."},
		{mCopyNeedsHere.Text(vi, i18n.Args{"kind": KindSMS}), "Vai trò của bạn ở đây không thể sửa chiến dịch."},
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
	if missing := msgs.Missing(i18n.Vietnamese); len(missing) > 0 {
		t.Errorf("not translated: %v", missing)
	}
}
