package growth

import (
	"testing"
	"time"

	"github.com/hostrepute/reputecast/internal/i18n"
)

// The segment builder names growth sources in the reader's language;
// exports and agents keep the English.
func TestSourceLabelsInVietnamese(t *testing.T) {
	vi := i18n.For(i18n.Vietnamese)
	day := time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		src    GrowthSource
		en, vi string
	}{
		{GrowthSource{Kind: KindPerson}, "Added by a person", "Thêm thủ công"},
		{GrowthSource{Kind: KindSubscribePage}, "Subscribe page", "Trang đăng ký"},
		{GrowthSource{Kind: KindPreferenceCenter}, "Preference center", "Trung tâm tùy chọn"},
		{GrowthSource{Kind: KindAPIKey, Name: "Website signup form"}, "API key “Website signup form”", "Khóa API “Website signup form”"},
		{GrowthSource{Kind: KindImport, Name: "Brevo", ImportedAt: &day}, "Brevo import (18 Sep)", "Nhập từ Brevo (18 thg 9)"},
		{GrowthSource{Kind: KindForm, Name: "Newsletter signup"}, "Newsletter signup", "Newsletter signup"},
		{GrowthSource{Kind: "other"}, "Other", "Khác"},
		{GrowthSource{Kind: KindUnknown}, "Unknown", "Không rõ"},
	} {
		if got := EnglishLabel(c.src); got != c.en {
			t.Errorf("%s in English: %q, want %q", c.src.Kind, got, c.en)
		}
		if got := Label(vi, c.src); got != c.vi {
			t.Errorf("%s in Vietnamese: %q, want %q", c.src.Kind, got, c.vi)
		}
	}
	if missing := msgs.Missing(i18n.Vietnamese); len(missing) > 0 {
		t.Errorf("no Vietnamese for %v", missing)
	}
}
